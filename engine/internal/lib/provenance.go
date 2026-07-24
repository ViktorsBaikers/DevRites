package lib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/rootfacts"
	"github.com/devrites/devrites/internal/safepath"
	"github.com/devrites/devrites/internal/state"
)

const EventSchemaV1 = "devrites-event/v1"

type EventBoundary string
type RootSource string
type ExecutionMode string
type GuardStrength string
type EventOutcome string
type EventHost string

const (
	BoundaryRootSelection   EventBoundary = "root-selection"
	BoundaryLifecycleGate   EventBoundary = "lifecycle-gate"
	BoundaryHookGuard       EventBoundary = "hook-guard"
	BoundaryAgentDispatch   EventBoundary = "agent-dispatch"
	BoundaryResultReconcile EventBoundary = "result-reconciliation"
	BoundaryGitPolicy       EventBoundary = "git-policy"

	RootSourceExplicit   RootSource = "DEVRITES_ROOT"
	RootSourceGit        RootSource = "git-ancestor"
	RootSourceGitMarker  RootSource = "git-marker-ancestor"
	RootSourceFilesystem RootSource = "filesystem-ancestor"
	RootSourceNone       RootSource = "none"

	ExecutionNamed   ExecutionMode = "named"
	ExecutionGeneric ExecutionMode = "generic"
	ExecutionInline  ExecutionMode = "inline"
	ExecutionNone    ExecutionMode = "none"

	GuardEnforced    GuardStrength = "enforced"
	GuardObserved    GuardStrength = "observed"
	GuardUnavailable GuardStrength = "unavailable"
	GuardBypassed    GuardStrength = "bypassed"
	GuardNA          GuardStrength = "n/a"

	OutcomePassed      EventOutcome = "passed"
	OutcomeBlocked     EventOutcome = "blocked"
	OutcomeDenied      EventOutcome = "denied"
	OutcomeAllowed     EventOutcome = "allowed"
	OutcomeObserved    EventOutcome = "observed"
	OutcomeBypassed    EventOutcome = "bypassed"
	OutcomeUnavailable EventOutcome = "unavailable"
	OutcomeWarning     EventOutcome = "warning"
	OutcomeRecorded    EventOutcome = "recorded"

	HostEngine EventHost = "engine"
	HostClaude EventHost = "claude"
	HostCodex  EventHost = "codex"
)

// EventV1 is the privacy-bounded execution/provenance record shared by the
// existing root timeline and per-workspace event logs.
type EventV1 struct {
	Schema        string        `json:"schema"`
	TS            string        `json:"ts"`
	RunID         string        `json:"run_id"`
	Boundary      EventBoundary `json:"boundary"`
	RootSource    RootSource    `json:"root_source"`
	Workspace     string        `json:"workspace,omitempty"`
	PhaseBefore   state.Phase   `json:"phase_before,omitempty"`
	Event         string        `json:"event"`
	RuleIDs       []reason.ID   `json:"rule_ids"`
	EvidencePaths []string      `json:"evidence_paths"`
	PhaseAfter    state.Phase   `json:"phase_after,omitempty"`
	ExecutionMode ExecutionMode `json:"execution_mode"`
	GuardStrength GuardStrength `json:"guard_strength"`
	ReasonID      reason.ID     `json:"reason_id"`
	Outcome       EventOutcome  `json:"outcome"`
	Host          EventHost     `json:"host"`
}

var (
	eventToken = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,63}$`)
	slugToken  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	runIDToken = regexp.MustCompile(`^drv-run-v1:[0-9a-f]{32}$`)
	processRun = makeProcessRunID()
)

func makeProcessRunID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return "drv-run-v1:" + hex.EncodeToString(raw[:])
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))))
	return "drv-run-v1:" + hex.EncodeToString(sum[:16])
}

// CurrentRunID returns one opaque ID per engine process. Tests and hermetic
// runners may inject an already-opaque canonical ID; arbitrary environment
// values are never persisted.
func CurrentRunID() string {
	if value := os.Getenv("DEVRITES_RUN_ID"); runIDToken.MatchString(value) {
		return value
	}
	return processRun
}

// NewEventV1 fills the version, timestamp, and opaque process identity.
func NewEventV1(boundary EventBoundary, event string, reasonID reason.ID) EventV1 {
	return EventV1{
		Schema:        EventSchemaV1,
		TS:            nowUTC(),
		RunID:         CurrentRunID(),
		Boundary:      boundary,
		Event:         event,
		RuleIDs:       []reason.ID{reasonID},
		EvidencePaths: []string{},
		ExecutionMode: ExecutionNone,
		GuardStrength: GuardNA,
		ReasonID:      reasonID,
		Host:          HostEngine,
	}
}

// NewRootSelectionEvent records the canonical root chosen for one command
// invocation without persisting that root's absolute path.
func NewRootSelectionEvent(root, command string) (EventV1, error) {
	ev := NewEventV1(BoundaryRootSelection, "invoke."+command, reason.RootSelected)
	ev.Outcome = OutcomePassed
	if err := BindEventWorkspace(root, activeSlug(root), &ev); err != nil {
		return EventV1{}, err
	}
	return ev, nil
}

// NewLifecycleGateEvent turns a typed gate result into a v1 provenance record.
func NewLifecycleGateEvent(root, slug, gateName string, phase state.Phase, blocked bool, reasonID reason.ID, evidenceFiles []string) (EventV1, error) {
	ev := NewEventV1(BoundaryLifecycleGate, gateName, reasonID)
	ev.PhaseBefore = phase
	ev.PhaseAfter = phase
	ev.EvidencePaths = WorkspaceEvidencePaths(root, slug, evidenceFiles...)
	ev.Outcome = OutcomePassed
	if blocked {
		ev.Outcome = OutcomeBlocked
	}
	if err := BindEventWorkspace(root, slug, &ev); err != nil {
		return EventV1{}, err
	}
	return ev, nil
}

// BindEventWorkspace adds only canonical, project-relative root/workspace
// identity and the current phase. It never copies an absolute path into ev.
func BindEventWorkspace(root, slug string, ev *EventV1) error {
	if ev == nil {
		return fmt.Errorf("nil event")
	}
	ev.RootSource = eventRootSource(root)
	if slug == "" {
		return nil
	}
	dir := featureDir(root, slug)
	rel, err := filepath.Rel(filepath.Dir(root), dir)
	if err != nil {
		return fmt.Errorf("derive workspace identity: %w", err)
	}
	ev.Workspace = filepath.ToSlash(rel)
	if feature, err := state.LoadFeature(root, slug); err == nil {
		if ev.PhaseBefore == "" {
			ev.PhaseBefore = feature.Phase
		}
		if ev.PhaseAfter == "" {
			ev.PhaseAfter = feature.Phase
		}
	}
	return nil
}

func eventRootSource(root string) RootSource {
	override := os.Getenv("DEVRITES_ROOT")
	facts, err := rootfacts.Resolve(override)
	if err == nil && sameEventPath(facts.PhysicalRoot, root) {
		switch facts.SelectionReason {
		case string(RootSourceExplicit):
			return RootSourceExplicit
		case string(RootSourceGit):
			return RootSourceGit
		case string(RootSourceGitMarker):
			return RootSourceGitMarker
		case string(RootSourceFilesystem):
			return RootSourceFilesystem
		}
	}
	if strings.TrimSpace(override) != "" {
		return RootSourceExplicit
	}
	return RootSourceNone
}

func sameEventPath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ai, aerr := os.Stat(a)
	bi, berr := os.Stat(b)
	if aerr == nil && berr == nil {
		return os.SameFile(ai, bi)
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// WorkspaceEvidencePaths converts workspace-local names to project-relative
// paths. Invalid or absolute names are omitted; final validation remains the
// append boundary.
func WorkspaceEvidencePaths(root, slug string, names ...string) []string {
	if slug == "" {
		return []string{}
	}
	dir := featureDir(root, slug)
	project := filepath.Dir(root)
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = filepath.ToSlash(name)
		if !validRelativePath(name) {
			continue
		}
		rel, err := filepath.Rel(project, filepath.Join(dir, filepath.FromSlash(name)))
		if err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
	}
	return out
}

// ValidateEventV1 enforces the privacy and compatibility boundary before disk.
func ValidateEventV1(ev EventV1) error {
	if ev.Schema != EventSchemaV1 {
		return fmt.Errorf("schema must be %q", EventSchemaV1)
	}
	if len(ev.TS) > 64 {
		return fmt.Errorf("timestamp is too long")
	}
	if _, err := time.Parse(time.RFC3339, ev.TS); err != nil {
		return fmt.Errorf("timestamp must be RFC3339: %w", err)
	}
	if !runIDToken.MatchString(ev.RunID) {
		return fmt.Errorf("run_id must be an opaque drv-run-v1 token")
	}
	if !knownBoundary(ev.Boundary) {
		return fmt.Errorf("unknown boundary %q", ev.Boundary)
	}
	if !knownRootSource(ev.RootSource) {
		return fmt.Errorf("unknown root_source %q", ev.RootSource)
	}
	if ev.Workspace != "" && !validWorkspaceIdentity(ev.Workspace) {
		return fmt.Errorf("workspace must be a canonical project-relative DevRites workspace")
	}
	if !validPhase(ev.PhaseBefore) || !validPhase(ev.PhaseAfter) {
		return fmt.Errorf("phase_before and phase_after must be canonical phases")
	}
	if !eventToken.MatchString(ev.Event) {
		return fmt.Errorf("event must be a bounded lowercase identifier")
	}
	if len(ev.RuleIDs) == 0 || len(ev.RuleIDs) > 16 {
		return fmt.Errorf("rule_ids must contain 1..16 entries")
	}
	if err := validateReasonIDs(ev.RuleIDs); err != nil {
		return err
	}
	if len(ev.EvidencePaths) > 32 {
		return fmt.Errorf("evidence_paths exceeds 32 entries")
	}
	if err := validateRelativePaths(ev.EvidencePaths); err != nil {
		return err
	}
	if !knownExecutionMode(ev.ExecutionMode) {
		return fmt.Errorf("unknown execution_mode %q", ev.ExecutionMode)
	}
	if !knownGuardStrength(ev.GuardStrength) {
		return fmt.Errorf("unknown guard_strength %q", ev.GuardStrength)
	}
	if !reason.Known(ev.ReasonID) {
		return fmt.Errorf("unknown reason_id %q", ev.ReasonID)
	}
	if !knownOutcome(ev.Outcome) {
		return fmt.Errorf("unknown outcome %q", ev.Outcome)
	}
	if ev.Host != HostEngine && ev.Host != HostClaude && ev.Host != HostCodex {
		return fmt.Errorf("unknown host %q", ev.Host)
	}
	return nil
}

func validateReasonIDs(ids []reason.ID) error {
	seen := make(map[reason.ID]bool, len(ids))
	for _, id := range ids {
		if !reason.Known(id) {
			return fmt.Errorf("unknown rule id %q", id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate rule id %q", id)
		}
		seen[id] = true
	}
	return nil
}

func validateRelativePaths(paths []string) error {
	seen := make(map[string]bool, len(paths))
	for _, value := range paths {
		if !validRelativePath(value) {
			return fmt.Errorf("evidence path %q is not canonical project-relative", value)
		}
		if seen[value] {
			return fmt.Errorf("duplicate evidence path %q", value)
		}
		seen[value] = true
	}
	return nil
}

func validRelativePath(value string) bool {
	if value == "" || len(value) > 512 || strings.ContainsAny(value, "\\\x00") ||
		path.IsAbs(value) || filepath.IsAbs(value) {
		return false
	}
	clean := path.Clean(value)
	return clean == value && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func validWorkspaceIdentity(value string) bool {
	if !validRelativePath(value) {
		return false
	}
	parts := strings.Split(value, "/")
	return len(parts) == 3 && parts[0] == ".devrites" &&
		(parts[1] == "work" || parts[1] == "features") &&
		slugToken.MatchString(parts[2])
}

func validPhase(phase state.Phase) bool {
	return phase == "" || state.KnownPhase(phase)
}

func knownBoundary(value EventBoundary) bool {
	switch value {
	case BoundaryRootSelection, BoundaryLifecycleGate, BoundaryHookGuard,
		BoundaryAgentDispatch, BoundaryResultReconcile, BoundaryGitPolicy:
		return true
	}
	return false
}

func knownRootSource(value RootSource) bool {
	switch value {
	case RootSourceExplicit, RootSourceGit, RootSourceGitMarker,
		RootSourceFilesystem, RootSourceNone:
		return true
	}
	return false
}

func knownExecutionMode(value ExecutionMode) bool {
	switch value {
	case ExecutionNamed, ExecutionGeneric, ExecutionInline, ExecutionNone:
		return true
	}
	return false
}

func knownGuardStrength(value GuardStrength) bool {
	switch value {
	case GuardEnforced, GuardObserved, GuardUnavailable, GuardBypassed, GuardNA:
		return true
	}
	return false
}

func knownOutcome(value EventOutcome) bool {
	switch value {
	case OutcomePassed, OutcomeBlocked, OutcomeDenied, OutcomeAllowed,
		OutcomeObserved, OutcomeBypassed, OutcomeUnavailable, OutcomeWarning,
		OutcomeRecorded:
		return true
	}
	return false
}

// AppendEventV1 appends one validated record to the existing root timeline and,
// when present, the existing per-workspace event log. Callers intentionally
// treat errors as telemetry degradation rather than a policy decision.
func AppendEventV1(root string, ev EventV1) error {
	if err := ValidateEventV1(ev); err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a directory")
		}
		return fmt.Errorf("event root unavailable: %w", err)
	}
	rootLog := filepath.Join(root, "timeline.jsonl")
	if err := validateEventLineTarget(rootLog, root); err != nil {
		return err
	}
	workspaceLog := ""
	if ev.Workspace != "" {
		workspace := filepath.Join(filepath.Dir(root), filepath.FromSlash(ev.Workspace))
		workspaceInfo, err := os.Stat(workspace)
		if err != nil || !workspaceInfo.IsDir() || !safepath.WithinResolved(workspace, root) {
			if err == nil {
				err = fmt.Errorf("not a workspace directory inside root")
			}
			return fmt.Errorf("event workspace unavailable: %w", err)
		}
		workspaceLog = filepath.Join(workspace, "events.jsonl")
		if err := validateEventLineTarget(workspaceLog, root); err != nil {
			return err
		}
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := appendEventLine(rootLog, root, line); err != nil {
		return err
	}
	if workspaceLog == "" {
		return nil
	}
	return appendEventLine(workspaceLog, root, line)
}

func appendEventLine(file, root string, line []byte) error {
	if err := validateEventLineTarget(file, root); err != nil {
		return err
	}
	return appendJSONLine(file, json.RawMessage(line))
}

func validateEventLineTarget(file, root string) error {
	if info, err := os.Lstat(file); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("event log is a symlink: %s", filepath.Base(file))
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect event log: %w", err)
	}
	if !safepath.WithinResolved(filepath.Dir(file), root) {
		return fmt.Errorf("event log escapes root")
	}
	return nil
}

package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

// regression.go: progress-monotonicity check. A workspace that once recorded a
// fact — a built slice, a checked acceptance criterion, a met or abandoned
// gate, a resolved question, a present artifact, a later phase — must not
// quietly lose it. Sessions overlap, contexts compact, tools rewrite files;
// the regression baseline is the durable high-water mark each durable
// checkpoint ratchets forward and each review/seal compares against.
//
//	devrites-engine check regression <slug>            compare vs baseline
//	devrites-engine check regression <slug> --update   ratchet baseline to now
//
//	0  clean / no baseline yet
//	2  usage error or workspace unreadable
//	3  blocked: regressions listed

const regressionBaselineFile = "regression-baseline.json"

// doneSliceStates are the tasks.md slice Status values that count as closed.
var doneSliceStates = map[string]bool{
	"built":     true,
	"proven":    true,
	"sealed":    true,
	"done":      true,
	"complete":  true,
	"completed": true,
}

var (
	regressACRe      = regexp.MustCompile(`(?im)^[ \t]*- \[([xX ])\] (AC-\d+)`)
	regressSliceRe   = regexp.MustCompile(`(?m)^## (SLICE-\d+)\b[^\n]*`)
	regressStatusRe  = regexp.MustCompile(`(?im)^Status:[ \t]*([^\s]+)`)
	regressQHeaderRe = regexp.MustCompile(`(?im)^## (q-\d{4}-\d{2}-\d{2}-\d+)\b`)
	regressQStatusRe = regexp.MustCompile(`(?im)^status:[ \t]*([^\s]+)`)
	regressH2Re      = regexp.MustCompile(`(?m)^## `)
)

// progressFingerprint is the structural state one baseline records. Sets are
// sorted for deterministic output and diffing.
type progressFingerprint struct {
	Phase             string   `json:"phase,omitempty"`
	PhaseOrdinal      int      `json:"phase_ordinal"`
	CheckedAC         []string `json:"checked_ac,omitempty"`
	MetGates          []string `json:"met_gates,omitempty"`
	AbandonedGates    []string `json:"abandoned_gates,omitempty"`
	DoneSlices        []string `json:"done_slices,omitempty"`
	ResolvedQuestions []string `json:"resolved_questions,omitempty"`
	PresentArtifacts  []string `json:"present_artifacts,omitempty"`
}

// regressionBaseline is the durable high-water mark for one workspace.
type regressionBaseline struct {
	Version   int    `json:"version"`
	UpdatedAt string `json:"updated_at"`
	progressFingerprint
}

// RunCheckRegression compares the workspace against its recorded progress
// baseline, or ratchets that baseline forward with --update.
func RunCheckRegression(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(stderr, regressionUsage)
		return 2
	}
	slug := args[0]
	update := false
	if len(args) == 2 {
		if args[1] != "--update" {
			fmt.Fprintln(stderr, regressionUsage)
			return 2
		}
		update = true
	}
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "check regression: %v\n", err)
		return 2
	}
	current, err := fingerprintWorkspace(root, slug, workspace)
	if err != nil {
		fmt.Fprintf(stderr, "check regression: %v\n", err)
		return 2
	}
	baselinePath := filepath.Join(workspace, regressionBaselineFile)
	baseline, hasBaseline, err := readRegressionBaseline(baselinePath)
	if err != nil {
		fmt.Fprintf(stdout, "REGRESSION: baseline %s unreadable: %v\ncheck regression: BLOCKED\n", regressionBaselineFile, err)
		return 3
	}
	if !update {
		if !hasBaseline {
			fmt.Fprintf(stdout, "check regression: unproven — no baseline recorded; run `devrites-engine check regression %s --update` at the next durable checkpoint\n", slug)
			return 0
		}
		regressions := compareProgress(baseline.progressFingerprint, current)
		if len(regressions) == 0 {
			fmt.Fprintf(stdout, "check regression: PASS (baseline %s; %s)\n", baseline.UpdatedAt, fingerprintSummary(current))
			return 0
		}
		for _, line := range regressions {
			fmt.Fprintf(stdout, "REGRESSION: %s\n", line)
		}
		fmt.Fprintf(stdout, "check regression: BLOCKED: %d regression(s) vs baseline %s\n", len(regressions), baseline.UpdatedAt)
		return 3
	}
	return updateRegressionBaseline(root, slug, baselinePath, baseline, hasBaseline, current, stdout, stderr)
}

// fingerprintWorkspace computes the current structural progress facts.
func fingerprintWorkspace(root, slug, workspace string) (progressFingerprint, error) {
	fp := progressFingerprint{PhaseOrdinal: -1}

	if report, err := state.Status(root, slug); err == nil && report != nil {
		fp.Phase = string(report.Phase)
		for i, policy := range state.PhasePolicies() {
			if policy.Target == report.Phase {
				fp.PhaseOrdinal = i
				break
			}
		}
	}

	if masked, err := maskedArtifact(workspace, "spec.md"); err == nil {
		for _, m := range regressACRe.FindAllStringSubmatch(masked, -1) {
			if strings.EqualFold(m[1], "x") {
				fp.CheckedAC = append(fp.CheckedAC, m[2])
			}
		}
	}

	if raw, err := os.ReadFile(filepath.Join(workspace, "tasks.md")); err == nil { // #nosec G304 -- workspace artifact
		if masked, maskErr := markdowntext.Structural(raw); maskErr == nil {
			fp.DoneSlices = doneSlices(string(masked))
		}
	}

	if raw, err := os.ReadFile(filepath.Join(workspace, acceptance.GatesFile)); err == nil { // #nosec G304 -- workspace artifact
		doc := acceptance.ParseLedger(string(raw))
		for _, g := range doc.Gates {
			if g.Checked {
				fp.MetGates = append(fp.MetGates, g.ID)
			}
		}
		for id := range doc.Abandoned {
			fp.AbandonedGates = append(fp.AbandonedGates, id)
		}
	}

	if masked, err := maskedArtifact(workspace, "questions.md"); err == nil {
		fp.ResolvedQuestions = resolvedQuestions(masked)
	}

	fp.PresentArtifacts = presentArtifacts(workspace)
	sortSets(&fp)
	return fp, nil
}

// maskedArtifact returns the artifact text with fenced code blocks masked out,
// so fenced examples never count as real checkboxes, statuses, or question rows.
func maskedArtifact(workspace, name string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(workspace, name)) // #nosec G304 -- workspace artifact
	if err != nil {
		return "", err
	}
	masked, err := markdowntext.Structural(raw)
	if err != nil {
		return "", err
	}
	return string(masked), nil
}

// doneSlices returns SLICE ids whose block carries a done-state Status line.
func doneSlices(masked string) []string {
	var done []string
	headers := regressSliceRe.FindAllStringSubmatchIndex(masked, -1)
	for i, m := range headers {
		end := len(masked)
		if i+1 < len(headers) {
			end = headers[i+1][0]
		}
		block := masked[m[0]:end]
		if sm := regressStatusRe.FindStringSubmatch(block); sm != nil &&
			doneSliceStates[strings.ToLower(sm[1])] {
			done = append(done, masked[m[2]:m[3]])
		}
	}
	return done
}

// resolvedQuestions returns question ids whose block status is anything but open.
func resolvedQuestions(masked string) []string {
	var resolved []string
	headers := regressQHeaderRe.FindAllStringSubmatchIndex(masked, -1)
	for i, m := range headers {
		end := len(masked)
		if next := regressH2Re.FindStringIndex(masked[m[1]:]); next != nil {
			end = m[1] + next[0]
		}
		if i+1 < len(headers) && headers[i+1][0] < end {
			end = headers[i+1][0]
		}
		block := masked[m[0]:end]
		if sm := regressQStatusRe.FindStringSubmatch(block); sm != nil &&
			!strings.EqualFold(sm[1], "open") {
			resolved = append(resolved, masked[m[2]:m[3]])
		}
	}
	return resolved
}

// presentArtifacts lists the canonical workspace artifacts that exist and are
// non-empty, derived from every phase policy's required set so the check tracks
// the schema instead of duplicating it.
func presentArtifacts(workspace string) []string {
	seen := map[string]bool{}
	var names []string
	for _, policy := range state.PhasePolicies() {
		for _, artifact := range policy.RequiredArtifacts {
			name := string(artifact)
			if seen[name] {
				continue
			}
			seen[name] = true
			info, err := os.Stat(filepath.Join(workspace, name))
			if err != nil || info.IsDir() || info.Size() == 0 {
				continue
			}
			names = append(names, name)
		}
	}
	return names
}

// compareProgress lists every baseline fact the current fingerprint lost.
func compareProgress(base, current progressFingerprint) []string {
	var out []string
	curAC := toSet(current.CheckedAC)
	for _, id := range base.CheckedAC {
		if !curAC[id] {
			out = append(out, fmt.Sprintf("acceptance criterion %s was checked, now unchecked or removed", id))
		}
	}
	curMet := toSet(current.MetGates)
	for _, id := range base.MetGates {
		if !curMet[id] {
			out = append(out, fmt.Sprintf("gate %s was met, now unmet or removed", id))
		}
	}
	curAbandoned := toSet(current.AbandonedGates)
	for _, id := range base.AbandonedGates {
		if !curAbandoned[id] && !curMet[id] {
			out = append(out, fmt.Sprintf("abandoned gate %s disappeared (its handoff record is gone)", id))
		}
	}
	curSlices := toSet(current.DoneSlices)
	for _, id := range base.DoneSlices {
		if !curSlices[id] {
			out = append(out, fmt.Sprintf("slice %s was in a done state, now reopened or removed", id))
		}
	}
	curResolved := toSet(current.ResolvedQuestions)
	for _, id := range base.ResolvedQuestions {
		if !curResolved[id] {
			out = append(out, fmt.Sprintf("question %s was resolved, now open or removed", id))
		}
	}
	curArtifacts := toSet(current.PresentArtifacts)
	for _, name := range base.PresentArtifacts {
		if !curArtifacts[name] {
			out = append(out, fmt.Sprintf("artifact %s was present and non-empty, now missing or empty", name))
		}
	}
	if base.PhaseOrdinal >= 0 && current.PhaseOrdinal >= 0 && current.PhaseOrdinal < base.PhaseOrdinal {
		out = append(out, fmt.Sprintf("phase moved backward: %s -> %s", base.Phase, current.Phase))
	}
	sort.Strings(out)
	return out
}

// updateRegressionBaseline replaces the baseline with the current fingerprint.
// Regressions being blessed are printed, never hidden — a ratchet is the
// recorded acknowledgment that the current state is the new floor.
func updateRegressionBaseline(root, slug, baselinePath string, base regressionBaseline, hasBaseline bool, current progressFingerprint, stdout, stderr io.Writer) int {
	var blessed []string
	if hasBaseline {
		blessed = compareProgress(base.progressFingerprint, current)
	}
	// Keep the phase high-water mark: a baseline never forgets the furthest phase.
	if hasBaseline && base.PhaseOrdinal > current.PhaseOrdinal {
		current.PhaseOrdinal = base.PhaseOrdinal
		current.Phase = base.Phase
	}
	next := regressionBaseline{
		Version:             1,
		UpdatedAt:           time.Now().UTC().Format(time.RFC3339),
		progressFingerprint: current,
	}
	payload, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "check regression: marshal baseline: %v\n", err)
		return 2
	}
	writeErr := state.WithFeatureLock(root, slug, func() error {
		return state.AtomicWrite(baselinePath, append(payload, '\n'), 0o644)
	})
	if writeErr != nil {
		fmt.Fprintf(stderr, "check regression: write baseline: %v\n", writeErr)
		return 2
	}
	for _, line := range blessed {
		fmt.Fprintf(stdout, "REGRESSION-BLESSED: %s\n", line)
	}
	fmt.Fprintf(stdout, "check regression: baseline updated (%s; %s)\n", next.UpdatedAt, fingerprintSummary(current))
	recordMetric(root, slug, current.Phase, "regression-baseline", "", 0)
	return 0
}

// readRegressionBaseline loads the baseline file; hasBaseline is false when it
// does not exist.
func readRegressionBaseline(path string) (regressionBaseline, bool, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- workspace artifact
	if err != nil {
		if os.IsNotExist(err) {
			return regressionBaseline{}, false, nil
		}
		return regressionBaseline{}, false, err
	}
	var baseline regressionBaseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return regressionBaseline{}, false, fmt.Errorf("corrupt baseline: %w", err)
	}
	if baseline.Version != 1 {
		return regressionBaseline{}, false, fmt.Errorf("unsupported baseline version %d", baseline.Version)
	}
	sortSets(&baseline.progressFingerprint)
	return baseline, true, nil
}

func fingerprintSummary(fp progressFingerprint) string {
	return fmt.Sprintf("phase=%s ac=%d gates-met=%d slices-done=%d questions-resolved=%d artifacts=%d",
		orDash(fp.Phase), len(fp.CheckedAC), len(fp.MetGates), len(fp.DoneSlices), len(fp.ResolvedQuestions), len(fp.PresentArtifacts))
}

func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

func sortSets(fp *progressFingerprint) {
	sort.Strings(fp.CheckedAC)
	sort.Strings(fp.MetGates)
	sort.Strings(fp.AbandonedGates)
	sort.Strings(fp.DoneSlices)
	sort.Strings(fp.ResolvedQuestions)
	sort.Strings(fp.PresentArtifacts)
}

const regressionUsage = "usage: devrites-engine check regression <slug> [--update]"

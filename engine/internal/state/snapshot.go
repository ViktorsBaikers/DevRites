package state

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/workflow"
)

const WorkspaceSnapshotSchema = "devrites.workspace.v1"

var (
	snapshotSliceCheckRe   = regexp.MustCompile(`^- \[[^\]]*\][[:space:]]*`)
	snapshotSliceNameRe    = regexp.MustCompile(`^Slice[[:space:]]*[0-9]+:[[:space:]]*`)
	snapshotSliceStateRe   = regexp.MustCompile(`[[:space:]]+(built|pending)[[:space:]]*$`)
	snapshotSliceSepRe     = regexp.MustCompile(`[[:space:]]+[^[:alnum:][:space:]]+[[:space:]]*$`)
	snapshotSliceCheckedRe = regexp.MustCompile(`\[[xX]\]`)
	snapshotTouchedPathRe  = regexp.MustCompile("`([^`]+)`")
)

// WorkspaceSnapshot is the machine-readable DevRites workspace/status contract.
// It models DevRites' native unit of work — a feature workspace at a phase with
// section completeness — rather than a generic harness session/worker shape.
type WorkspaceSnapshot struct {
	SchemaVersion   string               `json:"schemaVersion"`
	Slug            string               `json:"slug"`
	Phase           string               `json:"phase"`
	RunMode         string               `json:"runMode"`
	Complete        bool                 `json:"complete"`
	Sections        []SectionSnapshot    `json:"sections"`
	MissingSections []string             `json:"missingSections,omitempty"`
	CurrentSlice    *SliceSnapshot       `json:"currentSlice,omitempty"`
	Evidence        EvidenceSnapshot     `json:"evidence"`
	Drift           DriftSnapshot        `json:"drift"`
	Review          ReviewSnapshot       `json:"review"`
	Harness         HarnessSnapshot      `json:"harness"`
	Extensions      ExtensionSnapshot    `json:"extensions"`
	DirtyWorkspace  DirtyWorkspace       `json:"dirtyWorkspace"`
	Capabilities    []CapabilitySnapshot `json:"capabilities"`
	NextCommand     string               `json:"nextCommand,omitempty"` // legacy: Claude Code form; prefer nextCommands.
	NextCommands    CommandSnapshot      `json:"nextCommands"`
	Warnings        []string             `json:"warnings,omitempty"`
}

type CommandSnapshot struct {
	Verb   string `json:"verb,omitempty"`
	Claude string `json:"claude,omitempty"`
	Codex  string `json:"codex,omitempty"`
}

type SectionSnapshot struct {
	Name     string `json:"name"`
	Present  bool   `json:"present"`
	Required bool   `json:"required"`
}

type SliceSnapshot struct {
	Index int    `json:"index"`
	Total int    `json:"total"`
	Name  string `json:"name,omitempty"`
	State string `json:"state"`
}

type EvidenceSnapshot struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type DriftSnapshot struct {
	Status string `json:"status"`
	Open   int    `json:"open,omitempty"`
}

type ReviewSnapshot struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type HarnessSnapshot struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type ExtensionSnapshot struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type DirtyWorkspace struct {
	Status  string   `json:"status"`
	Changed int      `json:"changed"`
	Sample  []string `json:"sample,omitempty"`
}

type CapabilitySnapshot struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	UsedBy   string `json:"usedBy"`
	Fallback string `json:"fallback"`
	Risk     string `json:"risk"`
}

// Snapshot computes the stable JSON-facing status snapshot for slug. Empty slug
// falls back to .devrites/ACTIVE so future status surfaces can inspect the live
// workspace without chat context.
func Snapshot(root, slug string) (*WorkspaceSnapshot, error) {
	if strings.TrimSpace(slug) == "" {
		slug = readActiveSlug(root)
	}
	report, err := Status(root, slug)
	if err != nil {
		return nil, fmt.Errorf("snapshot workspace: %w", err)
	}

	workDir := featureDir(root, report.Slug)
	next := workflow.ForPhase(string(report.Phase))
	snap := &WorkspaceSnapshot{
		SchemaVersion:  WorkspaceSnapshotSchema,
		Slug:           report.Slug,
		Phase:          string(report.Phase),
		RunMode:        runMode(root),
		Complete:       report.Complete(),
		CurrentSlice:   currentSlice(workDir),
		Evidence:       evidenceSummary(root, workDir),
		Drift:          driftSummary(workDir),
		Review:         reviewSummary(workDir),
		Harness:        harnessSummary(filepath.Dir(root)),
		Extensions:     extensionSummary(root),
		DirtyWorkspace: dirtyWorkspace(filepath.Dir(root)),
		Capabilities:   capabilityInventory(filepath.Dir(root)),
		NextCommand:    next.Claude,
		NextCommands:   CommandSnapshot{Verb: next.Verb, Claude: next.Claude, Codex: next.Codex},
	}
	for _, section := range Sections {
		snap.Sections = append(snap.Sections, SectionSnapshot{
			Name:     string(section),
			Present:  report.Present[section],
			Required: report.Required[section],
		})
	}
	for _, section := range report.Missing {
		snap.MissingSections = append(snap.MissingSections, string(section))
	}
	if snap.RunMode == "AFK" && len(snap.MissingSections) > 0 {
		snap.Warnings = append(snap.Warnings, "AFK workspace has missing required sections")
	}
	if snap.Evidence.Status == "missing" && (report.Phase == PhaseProve || report.Phase == PhaseSeal || report.Phase == PhaseShip) {
		snap.Warnings = append(snap.Warnings, "proof phase requires fresh evidence")
	}
	if snap.Drift.Open > 0 {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("%d open drift item(s)", snap.Drift.Open))
	}
	return snap, nil
}

func readActiveSlug(root string) string {
	if ws := devritespaths.WorkspaceOverride(root, ""); ws != "" {
		return filepath.Base(ws)
	}
	raw, err := os.ReadFile(filepath.Join(root, "ACTIVE"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func runMode(root string) string {
	if info, err := os.Stat(filepath.Join(root, "AFK")); err == nil && !info.IsDir() {
		return "AFK"
	}
	return "HITL"
}

func currentSlice(workDir string) *SliceSnapshot {
	for _, name := range []string{"state.md", "tasks.md"} {
		raw, err := os.ReadFile(filepath.Join(workDir, name))
		if err != nil {
			continue
		}
		slices := parseSliceLines(string(raw))
		if len(slices) == 0 {
			continue
		}
		for i, s := range slices {
			if s.State != "built" {
				s.Index, s.Total = i+1, len(slices)
				return &s
			}
		}
		last := slices[len(slices)-1]
		last.Index, last.Total = len(slices), len(slices)
		return &last
	}
	return nil
}

func parseSliceLines(text string) []SliceSnapshot {
	var out []SliceSnapshot
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- [") {
			continue
		}
		name := snapshotSliceCheckRe.ReplaceAllString(trimmed, "")
		name = snapshotSliceNameRe.ReplaceAllString(name, "")
		name = snapshotSliceStateRe.ReplaceAllString(name, "")
		name = snapshotSliceSepRe.ReplaceAllString(name, "")
		state := "pending"
		if snapshotSliceCheckedRe.MatchString(trimmed) {
			state = "built"
		}
		out = append(out, SliceSnapshot{Name: strings.TrimSpace(name), State: state})
	}
	return out
}

func evidenceSummary(root, workDir string) EvidenceSnapshot {
	newestProof, haveProof := fsutil.NewestModTime(
		filepath.Join(workDir, "evidence.md"),
		filepath.Join(workDir, "proof.md"),
		filepath.Join(workDir, "browser-evidence.md"),
	)
	if !haveProof {
		return EvidenceSnapshot{Status: "missing", Detail: "no evidence.md / proof.md / browser-evidence.md"}
	}
	touched, err := os.ReadFile(filepath.Join(workDir, "touched-files.md"))
	if err != nil {
		return EvidenceSnapshot{Status: "fresh", Detail: "no touched-files.md; nothing to compare"}
	}
	var newestCode int64
	var newestPath string
	for _, match := range snapshotTouchedPathRe.FindAllStringSubmatch(string(touched), -1) {
		listed := match[1]
		path := listed
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(root), path)
		}
		if m, ok := fsutil.FileModTime(path); ok && m > newestCode {
			newestCode, newestPath = m, listed
		}
	}
	if newestCode > newestProof {
		return EvidenceSnapshot{Status: "stale", Detail: newestPath + " is newer than proof"}
	}
	return EvidenceSnapshot{Status: "fresh", Detail: "proof post-dates touched files"}
}

func driftSummary(workDir string) DriftSnapshot {
	for _, name := range []string{"drift.md", "questions.md"} {
		raw, err := os.ReadFile(filepath.Join(workDir, name))
		if err != nil {
			continue
		}
		open := countOpenMarkers(string(raw))
		if open > 0 {
			return DriftSnapshot{Status: "open", Open: open}
		}
	}
	return DriftSnapshot{Status: "clear"}
}

func countOpenMarkers(text string) int {
	open := 0
	for _, line := range strings.Split(text, "\n") {
		l := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(l, "status: open") || strings.Contains(l, "- [ ]") || strings.Contains(l, "gate: blocking") {
			open++
		}
	}
	return open
}

func reviewSummary(workDir string) ReviewSnapshot {
	for _, name := range []string{"review.md", "eng-review.md"} {
		raw, err := os.ReadFile(filepath.Join(workDir, name))
		if err != nil {
			continue
		}
		text := string(raw)
		if strings.Contains(text, "**Critical**") || strings.Contains(text, "**Important**") {
			return ReviewSnapshot{Status: "findings", Detail: name}
		}
		if strings.Contains(strings.ToLower(text), "no-findings") {
			return ReviewSnapshot{Status: "clean", Detail: name}
		}
		return ReviewSnapshot{Status: "present", Detail: name}
	}
	return ReviewSnapshot{Status: "missing"}
}

func harnessSummary(projectDir string) HarnessSnapshot {
	claude := regularFileExists(filepath.Join(projectDir, ".claude", "settings.json"))
	codex := regularFileExists(filepath.Join(projectDir, ".codex", "hooks.json")) || regularFileExists(filepath.Join(projectDir, "AGENTS.md"))
	switch {
	case claude && codex:
		return HarnessSnapshot{Status: "project-local", Detail: "Claude and Codex artifacts present"}
	case claude:
		return HarnessSnapshot{Status: "project-local", Detail: "Claude artifacts present"}
	case codex:
		return HarnessSnapshot{Status: "project-local", Detail: "Codex artifacts present"}
	default:
		return HarnessSnapshot{Status: "missing", Detail: "no project-local host artifacts detected"}
	}
}

func extensionSummary(root string) ExtensionSnapshot {
	entries, err := os.ReadDir(filepath.Join(root, "extensions"))
	if err != nil {
		return ExtensionSnapshot{Status: "none", Count: 0}
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			count++
		}
	}
	if count == 0 {
		return ExtensionSnapshot{Status: "none", Count: 0}
	}
	return ExtensionSnapshot{Status: "present", Count: count}
}

func dirtyWorkspace(projectDir string) DirtyWorkspace {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return DirtyWorkspace{Status: "unknown"}
	}
	var sample []string
	changed := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		changed++
		if len(sample) < 5 {
			sample = append(sample, strings.TrimSpace(line))
		}
	}
	if changed == 0 {
		return DirtyWorkspace{Status: "clean"}
	}
	return DirtyWorkspace{Status: "dirty", Changed: changed, Sample: sample}
}

func capabilityInventory(projectDir string) []CapabilitySnapshot {
	return []CapabilitySnapshot{
		capability("codegraph", "spec/define/plan", "file reads + search", "lower placement confidence"),
		capability("graphify", "zoom-out/spec", "repo search", "stale topology or no graph context"),
		capability("playwright", "prove/polish/browser-proof", "manual/browser proof ladder", "weaker browser evidence"),
		capability("chrome-devtools", "performance/browser-proof", "Playwright or manual traces", "no Lighthouse/perf trace"),
		mcpConfigCapability(projectDir),
	}
}

func capability(binary, usedBy, fallback, risk string) CapabilitySnapshot {
	status := "missing"
	if _, err := exec.LookPath(binary); err == nil {
		status = "available"
	}
	return CapabilitySnapshot{Name: binary, Status: status, UsedBy: usedBy, Fallback: fallback, Risk: risk}
}

func mcpConfigCapability(projectDir string) CapabilitySnapshot {
	status := "missing"
	for _, rel := range []string{".mcp.json", ".claude/mcp.json", ".codex/config.toml"} {
		if regularFileExists(filepath.Join(projectDir, rel)) {
			status = "configured"
			break
		}
	}
	return CapabilitySnapshot{Name: "mcp-config", Status: status, UsedBy: "optional external tools", Fallback: "engine/file-system gates", Risk: "optional capabilities unavailable"}
}

package state

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/markdowntext"
)

// Artifact budgets mirror workspace-artifact-schema.md "What each file owns".
// A line budget also bounds bytes at BudgetBytesPerLine per budgeted line so a
// few very long lines cannot evade it.
const (
	BudgetBytesPerLine = 400
	// BudgetBlockFactor is the material-overshoot threshold. At or above this
	// multiple of a line or byte budget a gate blocks unless the artifact
	// records a structural `Budget override: <=N lines; <reason>` line that
	// covers it; smaller overshoots are reported by `orient` and stay advisory.
	BudgetBlockFactor = 1.5
	// BudgetOverrideFactor caps an override's N at this multiple of the line
	// budget; past it only lossless relocation to history/ or a split is legal.
	BudgetOverrideFactor = 2
	// MaxArtifactBytes is the per-artifact read cap. A larger artifact cannot be
	// observed and blocks as file_too_large, so callers that explain a size
	// problem quote the same number the observer enforced.
	MaxArtifactBytes = maxArtifactBytes
	// PacketMaxBytes is the serialized-packet ceiling from context-hygiene.md:
	// "A serialized packet stays <= 64 KiB; larger means the scope is wrong".
	// A non-canonical workspace-root file over it is unsanctioned placement.
	PacketMaxBytes = 64 << 10
)

var artifactLineBudgets = map[string]int{
	"README.md":            120,
	"brief.md":             80,
	"spec.md":              260,
	"decision-coverage.md": 200,
	"strategy.md":          180,
	"architecture.md":      180,
	"flows.md":             160,
	"decisions.md":         200,
	"assumptions.md":       160,
	"questions.md":         180,
	"plan.md":              220,
	"tasks.md":             280,
	"traceability.md":      220,
	"eng-review.md":        240,
	"test-plan.md":         260,
	"gates.md":             200,
	"state.md":             120,
	"evidence.md":          280,
	"browser-evidence.md":  220,
	"drift.md":             160,
	"touched-files.md":     160,
	"design-brief.md":      160,
	"handoff.md":           120,
	"polish-report.md":     200,
	"ship.md":              120,
	"review.md":            240,
	"seal.md":              200,
	"ai-spec.md":           160,
	"windows.md":           120,
	"notes.md":             160,
}

// advisoryOnlyArtifacts are append-only proof ledgers: they scale with proof
// volume, not with planning surface, so their budget stays advisory in `orient`
// and never blocks a gate. touched-files.md keeps its own deterministic
// manifest limits.
var advisoryOnlyArtifacts = map[string]bool{
	"evidence.md":         true,
	"browser-evidence.md": true,
	"touched-files.md":    true,
}

// ArtifactLineBudget returns the documented line budget for a canonical artifact.
func ArtifactLineBudget(name string) (int, bool) {
	budget, ok := artifactLineBudgets[name]
	return budget, ok
}

// sanctionedRootFiles are workspace-root files the schema places there without a
// line budget: conditional artifacts (`references.md`, `investigation-map.md`,
// `dogfood.md`, `devex.md`) and engine-owned machine files (the parallel lease,
// the metrics ledger, the regression baseline, the dispatch ledger, the drift
// baseline, the diff-scope snapshot, the tree baseline). Anything else in the root is unsanctioned placement — `packets/`
// and `history/` own by-reference packets and relocated narrative
// (workspace-artifact-schema.md "Required by phase").
var sanctionedRootFiles = map[string]bool{
	"references.md":            true,
	"investigation-map.md":     true,
	"dogfood.md":               true,
	"devex.md":                 true,
	"parallel-lease.md":        true,
	"metrics.jsonl":            true,
	"regression-baseline.json": true,
	"dispatch.json":            true,
	"readiness-inputs.json":    true,
	"diff-scope-baseline.json": true,
	"tree-baseline.json":       true,
}

// SanctionedRootFile reports whether name is a legitimate workspace-root file:
// a budgeted canonical artifact or a documented unbudgeted one.
func SanctionedRootFile(name string) bool {
	if _, ok := artifactLineBudgets[name]; ok {
		return true
	}
	return sanctionedRootFiles[name]
}

// ArtifactBudgetGateApplies reports whether the deterministic material-overshoot
// gate covers this artifact (canonical and not an advisory-only proof ledger).
func ArtifactBudgetGateApplies(name string) bool {
	_, ok := artifactLineBudgets[name]
	return ok && !advisoryOnlyArtifacts[name]
}

// BudgetGatedArtifacts lists, sorted, every artifact the material-overshoot
// gate covers, so a gate checks each one present, not only the phase-required.
func BudgetGatedArtifacts() []ArtifactPath {
	var names []ArtifactPath
	for name := range artifactLineBudgets {
		if ArtifactBudgetGateApplies(name) {
			names = append(names, ArtifactPath(name))
		}
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

// ArtifactBudgetStatus is one canonical artifact measured against its budget.
type ArtifactBudgetStatus struct {
	File       string
	Lines      int
	Bytes      int64
	LineBudget int
	ByteBudget int64
	Over       bool
	Material   bool
	// OverrideLines is the N of a well-formed override within the cap, else 0.
	OverrideLines int
	// Override is true when that override covers the artifact: at most N lines
	// and N*BudgetBytesPerLine bytes.
	Override bool
}

// ArtifactBudget measures raw against the named artifact's budget. ok is false
// for a non-canonical file name.
func ArtifactBudget(name string, raw []byte) (ArtifactBudgetStatus, bool) {
	lineBudget, ok := artifactLineBudgets[name]
	if !ok {
		return ArtifactBudgetStatus{}, false
	}
	status := ArtifactBudgetStatus{
		File:       name,
		Lines:      CountLines(raw),
		Bytes:      int64(len(raw)),
		LineBudget: lineBudget,
		ByteBudget: int64(lineBudget) * BudgetBytesPerLine,
	}
	status.Over = status.Lines > lineBudget || status.Bytes > status.ByteBudget
	status.Material = float64(status.Lines) >= float64(lineBudget)*BudgetBlockFactor ||
		float64(status.Bytes) >= float64(status.ByteBudget)*BudgetBlockFactor
	if n := BudgetOverrideLines(raw); n > 0 && n <= lineBudget*BudgetOverrideFactor {
		status.OverrideLines = n
		status.Override = status.Lines <= n && status.Bytes <= int64(n)*BudgetBytesPerLine
	}
	return status, true
}

// CountLines counts newline-terminated lines plus a final unterminated line.
func CountLines(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	lines := bytes.Count(raw, []byte{'\n'})
	if raw[len(raw)-1] != '\n' {
		lines++
	}
	return lines
}

// budgetOverrideValue matches the text after `Budget override:`: `<=N` (optionally
// `lines`), `;`, then a non-empty reason.
var budgetOverrideValue = regexp.MustCompile(`^\s*<=\s*([0-9]{1,6})(?:\s*lines?)?\s*;\s*\S`)

// BudgetOverrideLines returns N from the first structural
// `Budget override: <=N lines; <reason>` line outside fenced code blocks, or 0
// when there is none. A free-text override without N, a fenced example, and
// malformed Markdown carry no override.
func BudgetOverrideLines(raw []byte) int {
	structural, err := markdowntext.Structural(raw)
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(structural), "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), "Budget override:")
		if !ok {
			continue
		}
		if m := budgetOverrideValue.FindStringSubmatch(value); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// BudgetRemedy names the legal ways out of a material overshoot for a
// lineBudget-line artifact.
func BudgetRemedy(lineBudget int) string {
	return fmt.Sprintf("relocate closed sections losslessly to history/<file>-<YYYYMMDD>.md, split the feature via /rite-plan revise|course-correct, or record a structural `Budget override: <=N lines; <reason>` line with N <= %d (%dx the budget; never for slice or acceptance count)",
		lineBudget*BudgetOverrideFactor, BudgetOverrideFactor)
}

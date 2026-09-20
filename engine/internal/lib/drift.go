package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/state"
)

// drift.go: readiness-input drift diagnosis. Vet records one aggregate
// "Readiness inputs SHA-256" line in eng-review.md; readiness/seal refuse when
// it no longer matches. That answer is binary — the agent learns the inputs
// changed but not which artifact did, so repair starts by re-reading eleven
// files. This check keeps a per-input digest baseline and attributes drift to
// the exact artifact, so the Spec Drift Guard or a re-vet starts at the right
// file.
//
//	devrites-engine check drift <slug>            compare vs recorded digests
//	devrites-engine check drift <slug> --record   write the digest baseline
//
//	0  always — drift is advisory: a dated delta through the owning rite is
//	   legal, so this command reports rather than blocks
//	2  usage error or workspace unreadable
//	3  baseline exists but cannot be parsed (fail closed on corrupt state)
//
// Without a baseline the check falls back to the aggregate binding recorded in
// eng-review.md; attribution then lists inputs whose mtime is newer than
// eng-review.md as candidates.

const driftBaselineFile = "readiness-inputs.json"

// driftBaseline is the recorded per-input digest set for one workspace.
type driftBaseline struct {
	Version    int                         `json:"version"`
	RecordedAt string                      `json:"recorded_at"`
	Binding    string                      `json:"binding"`
	Inputs     []gate.ReadinessInputDigest `json:"inputs"`
}

// driftDiff is one input whose state differs from the baseline.
type driftDiff struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // changed | missing | added
}

// RunCheckDrift implements `devrites-engine check drift`.
func RunCheckDrift(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(stderr, driftUsage)
		return 2
	}
	slug := args[0]
	record := false
	if len(args) == 2 {
		if args[1] != "--record" {
			fmt.Fprintln(stderr, driftUsage)
			return 2
		}
		record = true
	}
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "check drift: %v\n", err)
		return 2
	}
	digests, binding, err := gate.ReadinessInputDigests(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "check drift: %v\n", err)
		return 2
	}
	baselinePath := filepath.Join(workspace, driftBaselineFile)
	if record {
		return recordDriftBaseline(root, slug, baselinePath, digests, binding, stdout, stderr)
	}
	return compareDrift(workspace, baselinePath, digests, binding, stdout, stderr)
}

// recordDriftBaseline writes the current per-input digests as the baseline.
func recordDriftBaseline(root, slug, baselinePath string, digests []gate.ReadinessInputDigest, binding string, stdout, stderr io.Writer) int {
	baseline := driftBaseline{
		Version:    1,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		Binding:    binding,
		Inputs:     digests,
	}
	payload, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "check drift: marshal baseline: %v\n", err)
		return 2
	}
	writeErr := state.WithFeatureLock(root, slug, func() error {
		return state.AtomicWrite(baselinePath, append(payload, '\n'), 0o644)
	})
	if writeErr != nil {
		fmt.Fprintf(stderr, "check drift: write baseline: %v\n", writeErr)
		return 2
	}
	fmt.Fprintf(stdout, "check drift: baseline recorded (%d inputs, %s)\n", len(digests), binding)
	recordMetric(root, slug, "", "drift-baseline", "", 0)
	return 0
}

// compareDrift diffs current inputs against the baseline, or falls back to the
// aggregate binding recorded in eng-review.md when no baseline exists.
func compareDrift(workspace, baselinePath string, digests []gate.ReadinessInputDigest, binding string, stdout, stderr io.Writer) int {
	baseline, hasBaseline, err := readDriftBaseline(baselinePath)
	if err != nil {
		fmt.Fprintf(stdout, "DRIFT: baseline %s unreadable: %v\ncheck drift: BLOCKED\n", driftBaselineFile, err)
		return 3
	}
	if !hasBaseline {
		return compareDriftFallback(workspace, binding, stdout)
	}
	diffs := diffInputs(baseline.Inputs, digests)
	if len(diffs) == 0 && baseline.Binding == binding {
		fmt.Fprintf(stdout, "check drift: none (baseline %s)\n", baseline.RecordedAt)
		return 0
	}
	for _, d := range diffs {
		fmt.Fprintf(stdout, "DRIFT: %s %s since baseline %s\n", d.Path, d.Kind, baseline.RecordedAt)
	}
	if baseline.Binding != binding && len(diffs) == 0 {
		fmt.Fprintln(stdout, "DRIFT: aggregate binding changed without a recorded input diff; re-record the baseline")
	}
	fmt.Fprintf(stdout, "check drift: %d input(s) differ — a change is legal only as a dated delta through the owning rite; otherwise restore the input or rerun /rite-vet and --record\n", len(diffs))
	return 0
}

// compareDriftFallback compares the aggregate binding recorded in eng-review.md
// and attributes candidates by modification time.
func compareDriftFallback(workspace, binding string, stdout io.Writer) int {
	engReview := filepath.Join(workspace, "eng-review.md")
	raw, err := os.ReadFile(engReview) // #nosec G304 -- workspace artifact
	if err != nil {
		fmt.Fprintf(stdout, "check drift: unproven — no %s baseline and no readable eng-review.md; run `check drift <slug> --record` after Vet writes the binding\n", driftBaselineFile)
		return 0
	}
	recorded := ""
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "Readiness inputs SHA-256: ") {
			recorded = strings.TrimSpace(line)
		}
	}
	if recorded == "" {
		fmt.Fprintf(stdout, "check drift: unproven — eng-review.md carries no readiness binding; run `check drift <slug> --record` after Vet writes it\n")
		return 0
	}
	if recorded == binding {
		fmt.Fprintln(stdout, "check drift: none (eng-review.md binding current)")
		return 0
	}
	fmt.Fprintf(stdout, "DRIFT: readiness inputs changed since the eng-review.md binding\nrecorded: %s\ncurrent:  %s\n", recorded, binding)
	reviewInfo, err := os.Stat(engReview)
	if err == nil {
		for _, name := range []string{"spec.md", "decision-coverage.md", "architecture.md", "plan.md", "tasks.md", "traceability.md", "test-plan.md", "strategy.md", "design-brief.md", "ai-spec.md"} {
			if info, err := os.Stat(filepath.Join(workspace, name)); err == nil && info.ModTime().After(reviewInfo.ModTime()) {
				fmt.Fprintf(stdout, "candidate: %s modified after eng-review.md\n", name)
			}
		}
	}
	fmt.Fprintln(stdout, "check drift: route through the Spec Drift Guard or rerun /rite-vet")
	return 0
}

// diffInputs compares baseline digests against current digests.
func diffInputs(base, current []gate.ReadinessInputDigest) []driftDiff {
	cur := make(map[string]gate.ReadinessInputDigest, len(current))
	for _, d := range current {
		cur[d.Path] = d
	}
	var diffs []driftDiff
	for _, b := range base {
		c, ok := cur[b.Path]
		switch {
		case !ok:
			diffs = append(diffs, driftDiff{Path: b.Path, Kind: "missing"})
		case b.Present != c.Present:
			kind := "missing"
			if c.Present {
				kind = "added"
			}
			diffs = append(diffs, driftDiff{Path: b.Path, Kind: kind})
		case b.Present && b.SHA256 != c.SHA256:
			diffs = append(diffs, driftDiff{Path: b.Path, Kind: "changed"})
		}
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Path < diffs[j].Path })
	return diffs
}

// readDriftBaseline loads the digest baseline; hasBaseline is false when it
// does not exist.
func readDriftBaseline(path string) (driftBaseline, bool, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- workspace artifact
	if err != nil {
		if os.IsNotExist(err) {
			return driftBaseline{}, false, nil
		}
		return driftBaseline{}, false, err
	}
	var baseline driftBaseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return driftBaseline{}, false, fmt.Errorf("corrupt baseline: %w", err)
	}
	if baseline.Version != 1 {
		return driftBaseline{}, false, fmt.Errorf("unsupported baseline version %d", baseline.Version)
	}
	return baseline, true, nil
}

const driftUsage = "usage: devrites-engine check drift <slug> [--record]"

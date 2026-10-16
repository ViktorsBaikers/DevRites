package lib

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// proportion.go: proof proportionality advisories. A slice whose proof keeps
// growing after Vet, or that carries far more proof per acceptance criterion
// than its siblings, is usually absorbing hardening no spec line asked for —
// the cause of long vet↔plan and review↔repair loops. The advisory names the
// slice so the root moves uncited rows into an optional slice instead of
// repairing them; it never blocks.

const (
	// proofGrowthLimit is the growth factor since the Vet baseline that warns.
	proofGrowthLimit = 2
	// proofRatioLimit is how many times the median per-AC proof weight warns.
	proofRatioLimit = 3
	// proofRatioMinSlices is the sibling count below which a median means little.
	proofRatioMinSlices = 3
)

// sliceWeight is one slice's proof size: its tasks.md block plus every
// test-plan.md line that names it, and the distinct AC IDs its block cites.
type sliceWeight struct {
	Bytes int
	ACs   int
}

// proofWeights measures every SLICE-### in tasks.md; a missing test-plan.md
// counts as empty.
func proofWeights(tasks, testPlan []byte) map[string]sliceWeight {
	graph := ParseTaskGraph(tasks)
	planLines := strings.Split(string(testPlan), "\n")
	weights := make(map[string]sliceWeight, len(graph.Slices))
	for _, s := range graph.Slices {
		block, _ := state.ExtractTaskSlice(tasks, s.ID)
		w := sliceWeight{Bytes: len(block)}
		seen := map[string]bool{}
		for _, ac := range sliceACRe.FindAllString(block, -1) {
			seen[ac] = true
		}
		w.ACs = len(seen)
		idRE := regexp.MustCompile(`\b` + regexp.QuoteMeta(s.ID) + `\b`)
		for _, line := range planLines {
			if idRE.MatchString(line) {
				w.Bytes += len(line) + 1
			}
		}
		weights[s.ID] = w
	}
	return weights
}

// proportionAdvisories compares current weights against the Vet baseline
// (growth) and against sibling slices (per-AC ratio). only limits output to
// one slice; empty means all.
func proportionAdvisories(weights map[string]sliceWeight, baseline map[string]int, only string) []string {
	var perAC []float64
	for _, w := range weights {
		if w.ACs > 0 {
			perAC = append(perAC, float64(w.Bytes)/float64(w.ACs))
		}
	}
	sort.Float64s(perAC)
	median := 0.0
	if len(perAC) >= proofRatioMinSlices {
		median = perAC[len(perAC)/2]
	}
	ids := make([]string, 0, len(weights))
	for id := range weights {
		if only == "" || id == only {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	const fix = "move proof rows that quote no AC/REQ/INV/PROH/DEC/EDGE line into an optional slice instead of repairing them"
	var lines []string
	for _, id := range ids {
		w := weights[id]
		if base := baseline[id]; base > 0 && w.Bytes > proofGrowthLimit*base {
			lines = append(lines, fmt.Sprintf("ADVISORY proportion: %s proof grew %.1fx since the Vet baseline (%d -> %d bytes); %s",
				id, float64(w.Bytes)/float64(base), base, w.Bytes, fix))
		}
		if median > 0 && w.ACs > 0 {
			if ratio := float64(w.Bytes) / float64(w.ACs) / median; ratio > proofRatioLimit {
				lines = append(lines, fmt.Sprintf("ADVISORY proportion: %s carries %d bytes of proof for %d AC(s), %.1fx the median slice; %s",
					id, w.Bytes, w.ACs, ratio, fix))
			}
		}
	}
	return lines
}

// workspaceProofWeights reads tasks.md and test-plan.md for slug; nil when
// tasks.md is unreadable, so the advisory stays silent rather than wrong.
func workspaceProofWeights(root, slug string) map[string]sliceWeight {
	tasks, err := readWorkspaceArtifact(root, slug, "tasks.md")
	if err != nil {
		return nil
	}
	testPlan, _ := readWorkspaceArtifact(root, slug, "test-plan.md")
	return proofWeights(tasks, testPlan)
}

// printProportion writes the advisories for slug (one slice when only is set)
// against the drift baseline's recorded proof sizes, when one exists.
func printProportion(root, slug, only string, stdout io.Writer) {
	weights := workspaceProofWeights(root, slug)
	if weights == nil {
		return
	}
	var recorded map[string]int
	if workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug); err == nil {
		if base, ok, err := readDriftBaseline(filepath.Join(workspace, driftBaselineFile)); err == nil && ok {
			recorded = base.ProofBytes
		}
	}
	for _, line := range proportionAdvisories(weights, recorded, only) {
		fmt.Fprintln(stdout, line)
	}
}

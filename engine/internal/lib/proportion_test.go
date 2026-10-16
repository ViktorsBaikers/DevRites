package lib

import (
	"fmt"
	"strings"
	"testing"
)

func proportionTasks(firstBody string) []byte {
	var b strings.Builder
	b.WriteString("# Tasks\n\n")
	for i := 1; i <= 4; i++ {
		body := "Small proof."
		if i == 1 {
			body = firstBody
		}
		fmt.Fprintf(&b, "## SLICE-00%d Slice\nDependencies: none\nSatisfies: AC-00%d\n%s\n\n", i, i, body)
	}
	return []byte(b.String())
}

func TestProofWeightsCountsBlockAndTestPlanLines(t *testing.T) {
	tasks := proportionTasks("Small proof.")
	plan := []byte("| T1 | x | SLICE-001 |\n| T2 | y | SLICE-001/SLICE-002 |\n| T3 | z | SLICE-0010 |\n")
	w := proofWeights(tasks, plan)
	if w["SLICE-001"].ACs != 1 {
		t.Fatalf("ACs=%d", w["SLICE-001"].ACs)
	}
	if got, want := w["SLICE-001"].Bytes-w["SLICE-003"].Bytes, len("| T1 | x | SLICE-001 |\n| T2 | y | SLICE-001/SLICE-002 |\n"); got != want {
		t.Fatalf("SLICE-001 test-plan bytes=%d want %d (SLICE-0010 must not count)", got, want)
	}
}

func TestProportionAdvisories(t *testing.T) {
	small := proofWeights(proportionTasks("Small proof."), nil)
	big := proofWeights(proportionTasks(strings.Repeat("hardening row\n", 40)), nil)
	baseline := map[string]int{}
	for id, w := range small {
		baseline[id] = w.Bytes
	}

	if got := proportionAdvisories(small, baseline, ""); len(got) != 0 {
		t.Fatalf("proportional workspace warned: %v", got)
	}
	got := strings.Join(proportionAdvisories(big, baseline, ""), "\n")
	for _, want := range []string{"SLICE-001 proof grew", "SLICE-001 carries", "optional slice"} {
		if !strings.Contains(got, want) {
			t.Fatalf("advisories %q lack %q", got, want)
		}
	}
	if strings.Contains(got, "SLICE-002") {
		t.Fatalf("unchanged sibling warned: %q", got)
	}
	if got := proportionAdvisories(big, baseline, "SLICE-002"); len(got) != 0 {
		t.Fatalf("only=SLICE-002 printed another slice: %v", got)
	}
	if got := proportionAdvisories(big, nil, ""); len(got) != 1 || !strings.Contains(got[0], "carries") {
		t.Fatalf("no baseline must still report the ratio only: %v", got)
	}
}

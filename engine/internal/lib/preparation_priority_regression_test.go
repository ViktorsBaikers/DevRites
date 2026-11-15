package lib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalPreparationPoolReservedRepairCanUseHost(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	t.Setenv("DEVRITES_HOST_CAPACITY", "1")
	controllerLease(t, root, 2, controllerLeaseRow("", "SLICE-001", "src/current.go"))
	frontierFixture(t, root, "work", "current16", "", "plan", "running", "/rite-plan repair current16", 1, "3")
	if code, out := controllerDispatch(t, root, "current16", "open", "--phase", "plan", "--wave", "repair", "--role", "plan-drafter"); code != 0 {
		t.Fatalf("unlaunchable reserved writer blocked prerequisite repair: %d %s", code, out)
	}
}

func TestCanonicalPreparationPoolSerialWriterDoesNotConsumeHostPriority(t *testing.T) {
	root, dir := canonicalPreparationFixture(t, false)
	t.Setenv("DEVRITES_HOST_CAPACITY", "1")
	child := poolWorkspace(t, root, "future17", "current16", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	frontierFixture(t, root, "work", "future17", "current16", "build", "running", "/rite-build SLICE-001", 2, "2")
	writeFile(t, filepath.Join(child, "plan.md"), "# Plan\n\nConsumed planning inputs: none\n")
	poolVet(t, root, "future17")
	review := filepath.Join(child, "eng-review.md")
	writeFile(t, review, readTestFile(t, review)+"\nPredecessor readiness: none\n")
	if n, err := canonicalPool(t, root, "current16").writerPriority(root, "current16"); err != nil || n != 1 {
		t.Fatalf("ready writer fixture: priority=%d error=%v", n, err)
	}
	writeFile(t, filepath.Join(dir, "evidence.md"), "serial: future17/SLICE-001 — host cannot isolate workers; ordered handoff required\n")
	if text := frontierText(t, root, "current16"); strings.Contains(text, "frontier-picks:") {
		t.Fatalf("serial deferral still picked writer: %s", text)
	}
	if code, out := controllerDispatch(t, root, "current16", "open", "--phase", "plan", "--wave", "independent-plan", "--role", "plan-drafter"); code != 0 {
		t.Fatalf("serial-deferred writer consumed preparation host: %d %s", code, out)
	}
}

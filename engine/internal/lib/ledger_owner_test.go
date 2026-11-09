package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A checked state.md item owns only the slice it names first. Later references
// on the same line (a blocked dependent, an INV note, a NOT_RUN export) are
// context, not proof that those slices are built.

const poisonedCheckpoint = "- [x] SLICE-003 built — checkpoint abc1234; real publication/support export NOT_RUN (SLICE-004, INV-002)"

func TestBuiltSlicesOwnsOnlyCheckedItemSlice(t *testing.T) {
	state := "# State\n\n## Slice checkpoints\n" + poisonedCheckpoint + "\n- [ ] SLICE-004 pending\n"
	built, err := builtSlicesFrom(func(name string) ([]byte, error) {
		if name != "state.md" {
			return nil, os.ErrNotExist
		}
		return []byte(state), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(built) != 1 || !built["3"] || built["4"] {
		t.Fatalf("a later reference must not mark SLICE-004 built: %v", built)
	}
}

func TestRegressionOwnerOnlyKeepsFloorExact(t *testing.T) {
	root, dir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n\n## SLICE-003 Export\nGoal: export\nDone condition: csv\n\n## SLICE-004 Support\nGoal: support\n")
	state := "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n\n## Slice checkpoints\n" + poisonedCheckpoint + "\n- [ ] SLICE-004 pending\n"
	writeFile(t, filepath.Join(dir, "state.md"), state)
	code, out := checkRegression(t, root, "feat", "--update")
	if code != 0 || !strings.Contains(out, "slices-done=1") {
		t.Fatalf("a reference must not inflate the floor: code=%d out=%s", code, out)
	}
	writeFile(t, filepath.Join(dir, "state.md"), strings.Replace(state, poisonedCheckpoint, "- [x] SLICE-003 built — abc1234", 1))
	code, out = checkRegression(t, root, "feat")
	if code != 0 || !strings.Contains(out, "PASS") || !strings.Contains(out, "slices-done=1") {
		t.Fatalf("no phantom SLICE-004 regression: code=%d out=%s", code, out)
	}
}

func TestNextKeepsPendingSliceEligibleAfterReferencedCheckpoint(t *testing.T) {
	root, featureDir := planWorkspace(t)
	writeFile(t, filepath.Join(featureDir, "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n\n## Slice checkpoints\n- [x] SLICE-001 built — checkpoint abc1234; real publication/support export NOT_RUN (SLICE-004, INV-002)\n- [ ] SLICE-002 pending\n- [ ] SLICE-004 pending\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunNext(root, []string{"feat"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr)
	}
	if out := stdout.String(); !strings.Contains(out, "eligible_now: SLICE-002,SLICE-003,SLICE-004\n") {
		t.Fatalf("pending SLICE-004 dropped from next:\n%s", out)
	}
}

func TestControllerAdmitsPendingSliceAfterReferencedCheckpoint(t *testing.T) {
	root, dir := controllerFixture(t)
	writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n"+
		poolSlice("SLICE-001", "src/current.go", "Workspace dependencies: none")+
		poolSlice("SLICE-002", "src/read.go", "Workspace dependencies: none")+
		poolSlice("SLICE-003", "src/fetch.go", "Workspace dependencies: none")+
		"\n## SLICE-004 Pin\nDependencies: SLICE-002, SLICE-003\nWriter allowlist: src/pin.go\nWorkspace dependencies: none\n")
	writeFile(t, filepath.Join(dir, "state.md"), cursor("build", "running", "/rite-build SLICE-004")+
		"| sequence_position | 1 |\n| sequence_workspaces_remaining | 2 |\n\n## Slice checkpoints\n"+
		"- [x] SLICE-001 built — abc1234\n"+
		"- [x] SLICE-002 built — abc1234\n"+
		poisonedCheckpoint+"\n"+
		"- [ ] SLICE-004 pending\n")
	writeFile(t, filepath.Join(dir, "evidence.md"), "# Evidence\n\n- frontier-blocked: future17 — upstream API contract awaits owner decision DEC-017\n")
	poolVet(t, root, "current16")
	code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-004")
	if code != 0 || !strings.Contains(out, "wave writer16 open") {
		t.Fatalf("pending SLICE-004 not admitted: code=%d\n%s", code, out)
	}
}

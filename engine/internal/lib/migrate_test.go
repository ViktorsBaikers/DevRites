package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedPreV5Workspace builds a pre-v5 workspace with a bullet cursor, prose,
// and no required artifacts beyond state.md.
func seedPreV5Workspace(t *testing.T, phase string) (root, workspace string) {
	t.Helper()
	root = t.TempDir()
	workspace = filepath.Join(root, "work", "feature")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "- Phase: " + phase + "\n- Status: running\n\n# Status\n\nMinting middleware landed.\n"
	if err := os.WriteFile(filepath.Join(workspace, "state.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, workspace
}

func TestMigrateDryRunPlansWithoutWriting(t *testing.T) {
	root, workspace := seedPreV5Workspace(t, "build")
	var stdout, stderr strings.Builder
	if code := Migrate(root, []string{"feature", "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "migrate: plan for feature") ||
		!strings.Contains(out, "[cursor] convert legacy cursor bullets") ||
		!strings.Contains(out, "[schema] record schema") ||
		!strings.Contains(out, "[stub] create missing spec.md") {
		t.Fatalf("plan incomplete:\n%s", out)
	}
	// Nothing written: state.md bytes unchanged and no stub exists.
	body, err := os.ReadFile(filepath.Join(workspace, "state.md"))
	if err != nil || !strings.HasPrefix(string(body), "- Phase: build\n") {
		t.Fatalf("state.md changed during dry run: %v %q", err, body)
	}
	if _, err := os.Stat(filepath.Join(workspace, "spec.md")); !os.IsNotExist(err) {
		t.Fatalf("stub created during dry run: %v", err)
	}
}

func TestMigrateExecutesCursorConversionAndStubs(t *testing.T) {
	root, workspace := seedPreV5Workspace(t, "build")
	var stdout, stderr strings.Builder
	if code := Migrate(root, []string{"feature"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	body, err := os.ReadFile(filepath.Join(workspace, "state.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "| phase | build |") || !strings.Contains(text, "| status | running |") {
		t.Fatalf("cursor bullets not converted:\n%s", text)
	}
	if !strings.Contains(text, "| schema | 3 |") {
		t.Fatalf("schema row missing:\n%s", text)
	}
	if !strings.Contains(text, "# Status\n\nMinting middleware landed.") {
		t.Fatalf("prose not preserved:\n%s", text)
	}
	// Required artifacts for build exist as empty stubs.
	for _, name := range []string{"spec.md", "brief.md", "eng-review.md", "test-plan.md"} {
		info, err := os.Stat(filepath.Join(workspace, name))
		if err != nil {
			t.Fatalf("stub %s: %v", name, err)
		}
		if info.Size() != 0 {
			t.Fatalf("stub %s synthesized content", name)
		}
	}
	// Idempotent: a second run is a no-op.
	stdout.Reset()
	if code := Migrate(root, []string{"feature"}, &stdout, &stderr); code != 0 {
		t.Fatalf("second run code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nothing to do") {
		t.Fatalf("second run should be a no-op, got %q", stdout.String())
	}
}

func TestMigrateUnknownPhaseAsksAndWritesNothing(t *testing.T) {
	root, workspace := seedPreV5Workspace(t, "research")
	var stdout, stderr strings.Builder
	if code := Migrate(root, []string{"feature"}, &stdout, &stderr); code != 3 {
		t.Fatalf("code=%d want 3", code)
	}
	if !strings.Contains(stderr.String(), "question phase:") || !strings.Contains(stderr.String(), "nothing written") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	body, _ := os.ReadFile(filepath.Join(workspace, "state.md"))
	if !strings.HasPrefix(string(body), "- Phase: research\n") {
		t.Fatalf("state.md changed on refusal: %q", body)
	}

	// Answering the question on rerun proceeds.
	if code := Migrate(root, []string{"feature", "--answer", "phase=build"}, &stdout, &stderr); code != 0 {
		t.Fatalf("answer rerun code=%d stderr=%q", code, stderr.String())
	}
	body, _ = os.ReadFile(filepath.Join(workspace, "state.md"))
	if !strings.Contains(string(body), "| phase | build |") || !strings.Contains(string(body), "| schema | 3 |") {
		t.Fatalf("answer rerun did not migrate:\n%s", body)
	}
}

func TestMigrateDoneWorkspaceIsNoOp(t *testing.T) {
	root, workspace := seedPreV5Workspace(t, "done")
	var stdout, stderr strings.Builder
	if code := Migrate(root, []string{"feature"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout.String(), "no-ops") {
		t.Fatalf("stdout=%q", stdout.String())
	}
	body, _ := os.ReadFile(filepath.Join(workspace, "state.md"))
	if strings.Contains(string(body), "| schema |") {
		t.Fatalf("done workspace was rewritten: %q", body)
	}
}

func TestMigrateCurrentAndNewerSchemaAreRefused(t *testing.T) {
	root, workspace := seedPreV5Workspace(t, "build")
	if err := os.WriteFile(filepath.Join(workspace, "state.md"), []byte("| phase | build |\n| schema | 3 |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr strings.Builder
	if code := Migrate(root, []string{"feature"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "nothing to do") {
		t.Fatalf("current schema: code=%d stdout=%q", code, stdout.String())
	}

	if err := os.WriteFile(filepath.Join(workspace, "state.md"), []byte("| phase | build |\n| schema | 9 |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := Migrate(root, []string{"feature"}, &stdout, &stderr); code != 3 || !strings.Contains(stderr.String(), "upgrade devrites") {
		t.Fatalf("newer schema: code=%d stderr=%q", code, stderr.String())
	}
}

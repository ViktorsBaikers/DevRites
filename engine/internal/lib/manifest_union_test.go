package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const unionManifest = `# Touched files: SLUG

## Touched files
SLUG scope.

## Candidate manifest
| State | File | Slice | Reason |
| --- | --- | --- | --- |
ROWS
## Review trail
SLUG trail.
`

func manifestBody(rows ...string) string {
	return strings.Join(rows, "\n") + "\n"
}

func unionWorkspace(t *testing.T, root, slug, rows string, archived bool) {
	t.Helper()
	parent := "work"
	if archived {
		parent = "archive"
	}
	dir := filepath.Join(root, parent, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stateBody := "| phase | seal |\n| schema | 4 |\n"
	if err := os.WriteFile(filepath.Join(dir, "state.md"), []byte(stateBody), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := strings.ReplaceAll(unionManifest, "SLUG", slug)
	manifest = strings.Replace(manifest, "ROWS\n", rows, 1)
	if err := os.WriteFile(filepath.Join(dir, "touched-files.md"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func unionRoot(t *testing.T) string {
	root := filepath.Join(t.TempDir(), ".devrites")
	if err := os.MkdirAll(filepath.Join(root, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestMergeManifestUnionsOrderedPredecessors(t *testing.T) {
	root := unionRoot(t)
	unionWorkspace(t, root, "feat-1", manifestBody(
		"| present | `src/a.ts` | SLICE-001 | Added. |",
		"| present | `src/shared.ts` | SLICE-001 | First write. |",
	), false)
	unionWorkspace(t, root, "feat-2", manifestBody(
		"| present | `src/b.ts` | SLICE-002 | Added. |",
		"| present | `src/shared.ts` | SLICE-002 | Rewritten. |",
	), true) // archived predecessor still resolves
	unionWorkspace(t, root, "release", manifestBody(
		"| present | `src/r.ts` | SLICE-001 | Release wiring. |",
	), false)

	var stdout, stderr bytes.Buffer
	code := RunMergeManifest(root, []string{"release", "feat-1", "feat-2"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "work", "release", "touched-files.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	want := []string{
		"| present | `src/a.ts` | SLICE-001 | Added. |",
		"| present | `src/b.ts` | SLICE-002 | Added. |",
		"| present | `src/r.ts` | SLICE-001 | Release wiring. |",
		"| present | `src/shared.ts` | SLICE-002 | Rewritten. |", // later position wins
	}
	for _, row := range want {
		if !strings.Contains(text, row) {
			t.Fatalf("merged manifest missing %q:\n%s", row, text)
		}
	}
	if strings.Contains(text, "First write.") {
		t.Fatalf("losing collision row survived:\n%s", text)
	}
	if !strings.Contains(text, "## Touched files\nrelease scope.") || !strings.Contains(text, "## Review trail\nrelease trail.") {
		t.Fatalf("non-manifest sections were not preserved:\n%s", text)
	}
	if !strings.Contains(stdout.String(), "collision") {
		t.Fatalf("stdout should report the resolved collision: %q", stdout.String())
	}
}

func TestMergeManifestWalksSequenceChain(t *testing.T) {
	root := unionRoot(t)
	unionWorkspace(t, root, "feat-1", manifestBody("| present | `src/a.ts` | SLICE-001 | Added. |"), false)
	unionWorkspace(t, root, "feat-2", manifestBody(
		"| present | `src/a.ts` | SLICE-002 | Touched again. |",
		"| present | `src/b.ts` | SLICE-002 | Added. |",
	), false)
	unionWorkspace(t, root, "release", manifestBody("| present | `src/r.ts` | SLICE-001 | Release. |"), false)
	for slug, parent := range map[string]string{"feat-2": "feat-1", "release": "feat-2"} {
		statePath := filepath.Join(root, "work", slug, "state.md")
		raw, _ := os.ReadFile(statePath)
		raw = append(raw, []byte("| sequence_parent | "+parent+" |\n")...)
		if err := os.WriteFile(statePath, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var stdout, stderr bytes.Buffer
	if code := RunMergeManifest(root, []string{"release"}, &stdout, &stderr); code != 0 {
		t.Fatalf("chain walk: code=%d stderr=%s", code, stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "work", "release", "touched-files.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, row := range []string{
		"| present | `src/a.ts` | SLICE-002 | Touched again. |",
		"| present | `src/b.ts` | SLICE-002 | Added. |",
		"| present | `src/r.ts` | SLICE-001 | Release. |",
	} {
		if !strings.Contains(text, row) {
			t.Fatalf("chain-merged manifest missing %q:\n%s", row, text)
		}
	}

	// A workspace outside the chain has nothing to walk.
	stderr.Reset()
	unionWorkspace(t, root, "lone", manifestBody("| present | `src/x.ts` | SLICE-001 | Lone. |"), false)
	if code := RunMergeManifest(root, []string{"lone"}, &stdout, &stderr); code != 3 {
		t.Fatalf("no chain: code=%d stderr=%s", code, stderr.String())
	}
}

func TestVerifyReleaseUnionEnforcesChainCoverage(t *testing.T) {
	root := unionRoot(t)
	unionWorkspace(t, root, "feat-1", manifestBody("| present | `src/a.ts` | SLICE-001 | Added. |"), false)
	unionWorkspace(t, root, "release", manifestBody("| present | `src/r.ts` | SLICE-001 | Release. |"), false)
	for slug, extra := range map[string]string{
		"release": "| sequence_parent | feat-1 |\n| sequence_role | release |\n",
		"feat-1":  "",
	} {
		if extra == "" {
			continue
		}
		statePath := filepath.Join(root, "work", slug, "state.md")
		raw, _ := os.ReadFile(statePath)
		if err := os.WriteFile(statePath, append(raw, []byte(extra)...), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A release workspace whose manifest misses a predecessor path is blocked.
	if err := VerifyReleaseUnion(root, "release"); err == nil || !strings.Contains(err.Error(), "merge-manifest") {
		t.Fatalf("expected union gap, got %v", err)
	}
	// Workspaces without the role are not checked.
	if err := VerifyReleaseUnion(root, "feat-1"); err != nil {
		t.Fatalf("non-release workspace: %v", err)
	}

	// After the merge, coverage holds.
	var stdout, stderr bytes.Buffer
	if code := RunMergeManifest(root, []string{"release"}, &stdout, &stderr); code != 0 {
		t.Fatalf("merge: code=%d stderr=%s", code, stderr.String())
	}
	if err := VerifyReleaseUnion(root, "release"); err != nil {
		t.Fatalf("post-merge: %v", err)
	}
}

func TestMergeManifestRejectsParentMismatchAndMissingPredecessor(t *testing.T) {
	root := unionRoot(t)
	unionWorkspace(t, root, "feat-1", manifestBody("| present | `src/a.ts` | SLICE-001 | Added. |"), false)
	unionWorkspace(t, root, "release", manifestBody("| present | `src/r.ts` | SLICE-001 | Release. |"), false)
	statePath := filepath.Join(root, "work", "release", "state.md")
	raw, _ := os.ReadFile(statePath)
	raw = append(raw, []byte("| sequence_parent | feat-1 |\n")...)
	if err := os.WriteFile(statePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	// The recorded parent must be the last predecessor; omitting it blocks.
	if code := RunMergeManifest(root, []string{"release", "other"}, &stdout, &stderr); code != 3 {
		t.Fatalf("parent mismatch: code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "sequence_parent") {
		t.Fatalf("stderr=%q", stderr.String())
	}
	stderr.Reset()
	if code := RunMergeManifest(root, []string{"release", "feat-1", "ghost"}, &stdout, &stderr); code != 3 {
		t.Fatalf("missing predecessor: code=%d stderr=%s", code, stderr.String())
	}
	stderr.Reset()
	if code := RunMergeManifest(root, []string{"release", "release", "feat-1"}, &stdout, &stderr); code != 3 {
		t.Fatalf("self predecessor: code=%d", code)
	}
	if code := RunMergeManifest(root, []string{}, &stdout, &stderr); code != 2 {
		t.Fatalf("missing args: code=%d", code)
	}
}

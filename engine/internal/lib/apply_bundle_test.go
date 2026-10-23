package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const bundleTasks = `# Tasks

## SLICE-001 First
Dependencies: none
Covers AC-001.

## SLICE-002 Second
Dependencies: SLICE-001
Covers AC-002.
`

func bundleWorkspace(t *testing.T) (root, dir string) {
	t.Helper()
	root = unionRoot(t)
	dir = filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"state.md": "| phase | plan |\n| schema | 4 |\n",
		"tasks.md": bundleTasks,
		"plan.md":  "# Plan\n\nAC-001 and AC-002 ship in order.\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, dir
}

func writeBundle(t *testing.T, dir string, files []candidateFile, pins map[string]string, retired ...string) string {
	t.Helper()
	raw, err := json.Marshal(candidateBundle{BaseHashes: pins, CandidateFiles: files, RetiredIDs: retired})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "bundle.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func pinOf(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return sha256Hex(raw)
}

func patch(path string, hunks ...candidateHunk) candidateFile {
	return candidateFile{Path: path, Operation: "patch", Hunks: hunks}
}

func TestApplyBundleCombinesDisjointClusters(t *testing.T) {
	root, dir := bundleWorkspace(t)
	pins := map[string]string{
		".devrites/work/feat/tasks.md": pinOf(t, dir, "tasks.md"),
		"plan.md":                      pinOf(t, dir, "plan.md"),
	}
	first := writeBundle(t, dir, []candidateFile{
		patch(".devrites/work/feat/tasks.md", candidateHunk{ID: "T1", OldText: "Covers AC-001.", NewText: "Covers AC-001 with a seam."}),
	}, pins)
	second := writeBundle(t, dir, []candidateFile{
		patch("tasks.md", candidateHunk{ID: "T2", OldText: "Covers AC-002.", NewText: "Covers AC-002 end to end."}),
		patch("plan.md", candidateHunk{ID: "P1", OldText: "ship in order", NewText: "ship in dependency order"}),
	}, pins)

	var stdout, stderr bytes.Buffer
	if code := RunApplyBundle(root, []string{"feat", first, second, "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("dry-run code=%d stderr=%s", code, stderr.String())
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "tasks.md")); string(got) != bundleTasks {
		t.Fatal("dry-run wrote tasks.md")
	}
	stdout.Reset()
	if code := RunApplyBundle(root, []string{"feat", first, second}, &stdout, &stderr); code != 0 {
		t.Fatalf("apply code=%d stderr=%s", code, stderr.String())
	}
	tasks, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
	plan, _ := os.ReadFile(filepath.Join(dir, "plan.md"))
	if !strings.Contains(string(tasks), "with a seam") || !strings.Contains(string(tasks), "end to end") || !strings.Contains(string(plan), "dependency order") {
		t.Fatalf("hunks not applied:\n%s\n%s", tasks, plan)
	}
	if !strings.Contains(stdout.String(), "applied 2 file(s), 3 hunk(s)") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func TestApplyBundleRefusesAndWritesNothing(t *testing.T) {
	cases := []struct {
		name  string
		files func(dir string) ([]candidateFile, map[string]string, []string)
		want  string
	}{
		{"stale base", func(dir string) ([]candidateFile, map[string]string, []string) {
			return []candidateFile{patch("plan.md", candidateHunk{OldText: "ship", NewText: "land"})},
				map[string]string{"plan.md": strings.Repeat("0", 64)}, nil
		}, "stale base"},
		{"non-unique anchor", func(dir string) ([]candidateFile, map[string]string, []string) {
			return []candidateFile{patch("tasks.md", candidateHunk{ID: "T1", OldText: "Dependencies:", NewText: "Deps:"})},
				map[string]string{"tasks.md": pinOf(t, dir, "tasks.md")}, nil
		}, "non-unique anchor"},
		{"overlap", func(dir string) ([]candidateFile, map[string]string, []string) {
			return []candidateFile{patch("plan.md",
					candidateHunk{ID: "A", OldText: "AC-001 and", NewText: "AC-001 plus"},
					candidateHunk{ID: "B", OldText: "and AC-002", NewText: "with AC-002"})},
				map[string]string{"plan.md": pinOf(t, dir, "plan.md")}, nil
		}, "overlap"},
		{"lost id", func(dir string) ([]candidateFile, map[string]string, []string) {
			return []candidateFile{patch("plan.md", candidateHunk{OldText: "AC-001 and AC-002", NewText: "AC-001"})},
				map[string]string{"plan.md": pinOf(t, dir, "plan.md")}, nil
		}, "AC-002"},
		{"new cycle", func(dir string) ([]candidateFile, map[string]string, []string) {
			return []candidateFile{patch("tasks.md", candidateHunk{OldText: "Dependencies: none", NewText: "Dependencies: SLICE-002"})},
				map[string]string{"tasks.md": pinOf(t, dir, "tasks.md")}, nil
		}, "dependency cycle"},
		{"escape", func(dir string) ([]candidateFile, map[string]string, []string) {
			content := "x"
			return []candidateFile{{Path: "../other/plan.md", Operation: "create", Content: &content}}, nil, nil
		}, "inside the feat workspace"},
		{"dated delta section", func(dir string) ([]candidateFile, map[string]string, []string) {
			return []candidateFile{patch("plan.md", candidateHunk{OldText: "ship in order.\n", NewText: "ship in order.\n\n## Repair delta, 2026-10-16\nAC-001 ships first.\n"})},
				map[string]string{"plan.md": pinOf(t, dir, "plan.md")}, nil
		}, "dated delta section"},
		{"unescaped preflight pipe", func(dir string) ([]candidateFile, map[string]string, []string) {
			content := "# Test plan\n\n## Build-entry preflight\n| Gate | Command | Cwd | Expected | Run at |\n|---|---|---|---|---|\n" +
				"| probe | `test -f a || exit 1` | . | exit 0 | vet |\n"
			return []candidateFile{{Path: "test-plan.md", Operation: "create", Content: &content}}, nil, nil
		}, "escape each `|`"},
		{"budget growth", func(dir string) ([]candidateFile, map[string]string, []string) {
			return []candidateFile{patch("plan.md", candidateHunk{OldText: "# Plan\n", NewText: "# Plan\n" + strings.Repeat("line\n", 400)})},
				map[string]string{"plan.md": pinOf(t, dir, "plan.md")}, nil
		}, "material overshoot"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := bundleWorkspace(t)
			beforeTasks, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
			beforePlan, _ := os.ReadFile(filepath.Join(dir, "plan.md"))
			files, pins, retired := tc.files(dir)
			bundle := writeBundle(t, dir, files, pins, retired...)
			var stdout, stderr bytes.Buffer
			if code := RunApplyBundle(root, []string{"feat", bundle}, &stdout, &stderr); code != 3 {
				t.Fatalf("code=%d stderr=%s", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr %q lacks %q", stderr.String(), tc.want)
			}
			afterTasks, _ := os.ReadFile(filepath.Join(dir, "tasks.md"))
			afterPlan, _ := os.ReadFile(filepath.Join(dir, "plan.md"))
			if !bytes.Equal(beforeTasks, afterTasks) || !bytes.Equal(beforePlan, afterPlan) {
				t.Fatal("a refused bundle wrote a file")
			}
		})
	}
}

func TestApplyBundleRetiredIDAndCreate(t *testing.T) {
	root, dir := bundleWorkspace(t)
	content := "# Traceability\n\nAC-001 -> SLICE-001\n"
	bundle := writeBundle(t, dir, []candidateFile{
		patch("plan.md", candidateHunk{OldText: "AC-001 and AC-002", NewText: "AC-001"}),
		{Path: "traceability.md", Operation: "create", Content: &content},
	}, map[string]string{"plan.md": pinOf(t, dir, "plan.md")}, "AC-002")
	var stdout, stderr bytes.Buffer
	if code := RunApplyBundle(root, []string{"feat", bundle}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "traceability.md")); string(got) != content {
		t.Fatalf("create wrote %q", got)
	}
}

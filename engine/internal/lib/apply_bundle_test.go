package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/fsutil"
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

func TestApplyBundleCombinesPathAliases(t *testing.T) {
	for _, alias := range []string{"./plan.md", "refs/../plan.md", ".devrites/work/feat/./plan.md"} {
		t.Run(alias, func(t *testing.T) {
			root, dir := bundleWorkspace(t)
			if err := os.Mkdir(filepath.Join(dir, "refs"), 0o755); err != nil {
				t.Fatal(err)
			}
			pin := pinOf(t, dir, "plan.md")
			first := writeBundle(t, dir, []candidateFile{
				patch("plan.md", candidateHunk{OldText: "AC-001", NewText: "AC-001 first"}),
			}, map[string]string{"plan.md": pin})
			second := writeBundle(t, dir, []candidateFile{
				patch(alias, candidateHunk{OldText: "AC-002", NewText: "AC-002 next"}),
			}, map[string]string{alias: pin})
			var stdout, stderr bytes.Buffer
			if code := RunApplyBundle(root, []string{"feat", first, second}, &stdout, &stderr); code != 0 {
				t.Fatalf("code=%d stderr=%s", code, stderr.String())
			}
			got, err := os.ReadFile(filepath.Join(dir, "plan.md"))
			if err != nil {
				t.Fatal(err)
			}
			if want := "# Plan\n\nAC-001 first and AC-002 next ship in order.\n"; string(got) != want {
				t.Fatalf("combined bundle lost an edit: %q", got)
			}
		})
	}
}

func TestApplyBundleDrafterJSONEnvelope(t *testing.T) {
	const beforePlan = "# Plan\n\nAC-001 and AC-002 ship in \"draft\" order.\nRollback: keep C:\\cache\\plan.\n"
	const payload = `{
  "mode": "repair",
  "base_hashes": {"plan.md": "<plan-pin>", "tasks.md": "<tasks-pin>"},
  "candidate_files": [
    {
      "path": "plan.md",
      "operation": "patch",
      "hunks": [
        {
          "id": "P1",
          "old_text": "AC-001 and AC-002 ship in \"draft\" order.\nRollback: keep C:\\cache\\plan.\n",
          "new_text": "AC-001 and AC-002 ship in \"reviewed\" order.\nRollback:\n\tKeep C:\\cache\\plan; preserve café.\n"
        }
      ]
    },
    {
      "path": "tasks.md",
      "operation": "patch",
      "hunks": [
        {
          "id": "T1",
          "old_text": "Dependencies: SLICE-001\nCovers AC-002.",
          "new_text": "Dependencies: SLICE-001\nCovers AC-002 with \"quoted\" proof.\nRollback: restore C:\\cache\\plan."
        }
      ]
    }
  ],
  "retired_ids": [],
  "acceptance_map": ["AC-001 -> SLICE-001", "AC-002 -> SLICE-002"],
  "dependency_order": ["SLICE-001", "SLICE-002"],
  "technical_decisions": ["Reuse existing rollback; rationale stays in decisions.md."],
  "human_owned_choices": ["Operator chooses \"release\" or \"hold\".\nNo approval implied."],
  "validation_commands": ["go test ./..."],
  "applicability_disposition": [],
  "self_attack_exposures": [],
  "remaining_findings": ["F2: repair remaining traceability sites together"],
  "blocked_findings": ["F3: release policy needs operator input"]
}`
	const afterPlan = "# Plan\n\nAC-001 and AC-002 ship in \"reviewed\" order.\nRollback:\n\tKeep C:\\cache\\plan; preserve café.\n"
	const afterTasks = `# Tasks

## SLICE-001 First
Dependencies: none
Covers AC-001.

## SLICE-002 Second
Dependencies: SLICE-001
Covers AC-002 with "quoted" proof.
Rollback: restore C:\cache\plan.
`
	for _, tc := range []struct {
		name string
		old  string
		new  string
		want string
	}{
		{name: "accepted partial envelope"},
		{name: "stale second target pin", old: "<tasks-pin>", new: strings.Repeat("0", 64), want: "stale base"},
		{name: "malformed JSON escape", old: `\"quoted\"`, new: `\q`, want: "not a JSON candidate bundle"},
		{name: "trailing JSON object", old: "\n}", new: "\n} {}", want: "not a JSON candidate bundle"},
		{
			name: "overlapping hunks",
			old:  `"hunks": [`,
			new:  `"hunks": [{"id": "P2", "old_text": "AC-001 and AC-002", "new_text": "AC-001 plus AC-002"},`,
			want: "overlap",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := bundleWorkspace(t)
			if err := os.WriteFile(filepath.Join(dir, "plan.md"), []byte(beforePlan), 0o644); err != nil {
				t.Fatal(err)
			}
			raw := payload
			if tc.old != "" {
				raw = strings.Replace(raw, tc.old, tc.new, 1)
			}
			raw = strings.NewReplacer(
				"<plan-pin>", pinOf(t, dir, "plan.md"),
				"<tasks-pin>", pinOf(t, dir, "tasks.md"),
			).Replace(raw)
			bundle := filepath.Join(t.TempDir(), "drafter.json")
			if err := os.WriteFile(bundle, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			wantCode := 0
			wantPlan, wantTasks := afterPlan, afterTasks
			if tc.want != "" {
				wantCode = 3
				wantPlan, wantTasks = beforePlan, bundleTasks
			}
			var stdout, stderr bytes.Buffer
			if code := RunApplyBundle(root, []string{"feat", bundle}, &stdout, &stderr); code != wantCode {
				t.Fatalf("code=%d want=%d stderr=%s", code, wantCode, stderr.String())
			}
			if tc.want != "" && !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr %q lacks %q", stderr.String(), tc.want)
			}
			if tc.want == "" && !strings.Contains(stdout.String(), "applied 2 file(s), 2 hunk(s)") {
				t.Fatalf("stdout=%s", stdout.String())
			}
			for name, want := range map[string]string{"plan.md": wantPlan, "tasks.md": wantTasks} {
				got, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != want {
					t.Errorf("%s bytes = %q, want %q", name, got, want)
				}
			}
		})
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
		{"aliased replacements", func(dir string) ([]candidateFile, map[string]string, []string) {
			content := "# Plan\n\nAC-001 and AC-002 land.\n"
			pin := pinOf(t, dir, "plan.md")
			return []candidateFile{
				{Path: "plan.md", Operation: "replace", Content: &content},
				{Path: "./plan.md", Operation: "replace", Content: &content},
			}, map[string]string{"plan.md": pin, "./plan.md": pin}, nil
		}, "conflicts with another candidate"},
		{"conflicting aliased pins", func(dir string) ([]candidateFile, map[string]string, []string) {
			return []candidateFile{patch("plan.md", candidateHunk{OldText: "ship", NewText: "land"})},
				map[string]string{"plan.md": pinOf(t, dir, "plan.md"), "./plan.md": strings.Repeat("0", 64)}, nil
		}, "conflicting base hashes"},
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

func TestWriteBundleTargetsRollsBackAttemptedTargets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		existed bool
		publish bool
	}{
		{"existing after publication", true, true},
		{"created after publication", false, true},
		{"existing before publication", true, false},
		{"created before publication", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ws, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = ws.Close() }()
			targets := []*bundleTarget{
				{rel: "earlier.md", existed: true, before: []byte("earlier before"), after: []byte("earlier after")},
				{rel: "created.md", after: []byte("created after")},
				{rel: "failing.md", existed: tc.existed, after: []byte("failing after")},
				{rel: "untouched.md", existed: true, before: []byte("untouched before"), after: []byte("untouched after")},
			}
			if tc.existed {
				targets[2].before = []byte("failing before")
			}
			for _, target := range targets {
				if target.existed {
					if err := ws.WriteFile(target.rel, target.before, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			failure := errors.New("injected write failure")
			calls := 0
			err = writeBundleTargetsWith(ws, targets, func(root *os.Root, rel string, data []byte, perm fs.FileMode) error {
				calls++
				failing := calls == 3
				if !failing || tc.publish {
					if err := fsutil.WriteFileAtomicIn(root, rel, data, perm); err != nil {
						t.Fatal(err)
					}
				}
				if failing {
					return failure
				}
				return nil
			})
			if !errors.Is(err, failure) || !strings.Contains(err.Error(), "write failing.md") ||
				!strings.Contains(err.Error(), "attempted files restored") {
				t.Fatalf("unexpected write error: %v", err)
			}
			for _, target := range targets {
				got, readErr := ws.ReadFile(target.rel)
				if target.existed {
					if readErr != nil || !bytes.Equal(got, target.before) {
						t.Errorf("%s not restored: %q, %v", target.rel, got, readErr)
					}
				} else if !errors.Is(readErr, fs.ErrNotExist) {
					t.Errorf("%s not removed: %q, %v", target.rel, got, readErr)
				}
			}
		})
	}
}

func TestWriteBundleTargetsReportsRollbackWriteFailures(t *testing.T) {
	ws, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ws.Close() }()
	targets := []*bundleTarget{
		{rel: "first.md", existed: true, before: []byte("first before"), after: []byte("first after")},
		{rel: "middle.md", existed: true, before: []byte("middle before"), after: []byte("middle after")},
		{rel: "last.md", existed: true, before: []byte("last before"), after: []byte("last after")},
	}
	for _, target := range targets {
		if err := ws.WriteFile(target.rel, target.before, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFailure := errors.New("publication sync failed")
	firstFailure := errors.New("first restore failed")
	lastFailure := errors.New("last restore failed")
	calls := 0
	err = writeBundleTargetsWith(ws, targets, func(root *os.Root, rel string, data []byte, perm fs.FileMode) error {
		calls++
		if calls > len(targets) {
			switch rel {
			case "first.md":
				return firstFailure
			case "last.md":
				return lastFailure
			}
		}
		if err := fsutil.WriteFileAtomicIn(root, rel, data, perm); err != nil {
			t.Fatal(err)
		}
		if calls == len(targets) {
			return writeFailure
		}
		return nil
	})
	for _, cause := range []error{writeFailure, firstFailure, lastFailure} {
		if !errors.Is(err, cause) {
			t.Errorf("error %v does not wrap %v", err, cause)
		}
	}
	if err == nil {
		t.Fatal("missing write error")
	}
	for _, diagnostic := range []string{"write last.md", "rollback incomplete", "rollback first.md", "rollback last.md"} {
		if !strings.Contains(err.Error(), diagnostic) {
			t.Errorf("error %q lacks %q", err, diagnostic)
		}
	}
	if strings.Contains(err.Error(), "files restored") {
		t.Fatalf("error falsely claims restoration: %v", err)
	}
	for _, target := range targets {
		want := target.after
		if target.rel == "middle.md" {
			want = target.before
		}
		got, readErr := ws.ReadFile(target.rel)
		if readErr != nil || !bytes.Equal(got, want) {
			t.Errorf("%s = %q, %v; want %q", target.rel, got, readErr, want)
		}
	}
}

func TestWriteBundleTargetsReportsRollbackRemoveFailure(t *testing.T) {
	ws, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ws.Close() }()
	targets := []*bundleTarget{
		{rel: "earlier.md", after: []byte("earlier after")},
		{rel: "failing.md", after: []byte("failing after")},
	}
	failure := errors.New("publication sync failed")
	err = writeBundleTargetsWith(ws, targets, func(root *os.Root, rel string, data []byte, perm fs.FileMode) error {
		if err := fsutil.WriteFileAtomicIn(root, rel, data, perm); err != nil {
			t.Fatal(err)
		}
		if rel == "failing.md" {
			// A nonempty directory makes rollback removal fail without permission assumptions.
			if err := root.Remove(rel); err != nil {
				t.Fatal(err)
			}
			if err := root.Mkdir(rel, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := root.WriteFile(filepath.Join(rel, "blocker"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			return failure
		}
		return nil
	})
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "rollback incomplete") ||
		!strings.Contains(err.Error(), "rollback failing.md") || strings.Contains(err.Error(), "files restored") {
		t.Fatalf("unexpected write error: %v", err)
	}
	var removeErr *os.PathError
	if !errors.As(err, &removeErr) || filepath.Base(removeErr.Path) != "failing.md" {
		t.Fatalf("missing removal failure: %v", err)
	}
	if _, err := ws.Lstat("earlier.md"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("earlier target not removed: %v", err)
	}
}

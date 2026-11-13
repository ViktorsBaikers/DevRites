package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckAnchorsBoundCandidate(t *testing.T) {
	for _, scenario := range []string{"worker", "original", "explicit account", "unstored account", "control quote", "changed bytes", "changed head", "wrong aggregate", "wrong file hash", "wrong scope", "missing account", "ambiguous account", "missing packet", "wrong packet", "missing manifest", "missing worktree", "outside scope", "same tree"} {
		t.Run(scenario, func(t *testing.T) {
			root, workspace := anchorsProject(t)
			project := filepath.Dir(root)
			initDupRepo(t, project)
			runDupGit(t, project, "add", "src/a.go")
			runDupGit(t, project, "commit", "-qm", "baseline")
			base := strings.TrimSpace(string(mustGitOutput(t, project, "rev-parse", "HEAD")))
			worker := filepath.Join(t.TempDir(), "worker")
			runDupGit(t, project, "worktree", "add", "-qb", "worker", worker)
			worker, err := filepath.EvalSymlinks(worker)
			if err != nil {
				t.Fatal(err)
			}
			source := worker
			flags := []string{"--base", base, "--worktree", worker}
			if scenario == "same tree" {
				source = project
				flags = []string{"--base", base}
			}
			writeFile(t, filepath.Join(source, "src/a.go"), "package a\nfunc Handler() { return workerIdentity }\n")
			if source == worker {
				runDupGit(t, worker, "add", "src/a.go")
				runDupGit(t, worker, "commit", "-qm", "worker candidate")
			}
			out, errs := &bytes.Buffer{}, &bytes.Buffer{}
			args := append([]string{"feat", "--wave", "r1", "--paths", "src/a.go"}, flags...)
			if code := RunCandidateFreeze(root, args, out, errs); code != 0 {
				t.Fatalf("freeze %d: %s %s", code, out, errs)
			}
			sums := readTestFile(t, filepath.Join(workspace, "packets/r1.sha256"))
			// A different control candidate must not influence the review account.
			writeFile(t, filepath.Join(workspace, "touched-files.md"), candidateTestManifest(candidateTestRow{state: "present", path: "src/a.go", slice: "S-1", reason: "control"}))
			packet := func(manifest string) string {
				return "# Packet r1: code-reviewer\n\nFeature: feat\nPhase: build\n\nCandidate:\n```text\n" + manifest + "```\n"
			}
			writeFile(t, filepath.Join(workspace, "packets/r1.code-reviewer.md"), packet(sums))
			ledger := `{"waves":{"r1":{"phase":"build","state":"complete","roles":{"code-reviewer":{"handle":"review-1","started":"2026-11-13T00:00:00Z","returned":true,"returned_at":"2026-11-13T00:01:00Z"}}}}}`
			quote, location := "`return workerIdentity`", "src/a.go:2"
			if scenario == "control quote" {
				quote = "`return r.User`"
				location = "src/a.go:4"
			}
			if scenario == "outside scope" {
				location = "src/other.go:1"
				writeFile(t, filepath.Join(project, "src/other.go"), "return workerIdentity\n")
			}
			body := "Outcome: findings\nCounts: important=1\n" + anchorsRow("Important", location, quote, anchorsGoodBreaks) + "\n"
			stored := filepath.Join(workspace, "packets/r1-code-reviewer.return.md")
			writeFile(t, stored, body)
			switch scenario {
			case "changed bytes":
				writeFile(t, filepath.Join(worker, "src/a.go"), "package a\nfunc Handler() { return changedIdentity }\n")
			case "changed head":
				writeFile(t, filepath.Join(worker, "unrelated.txt"), "another commit\n")
				runDupGit(t, worker, "add", "unrelated.txt")
				runDupGit(t, worker, "commit", "-qm", "different head")
			case "wrong aggregate", "wrong file hash", "wrong scope":
				old := "aggregate "
				if scenario == "wrong file hash" {
					old = "file "
				}
				if scenario == "wrong scope" {
					old = "src/a.go"
				}
				changed := strings.Replace(sums, old, old+"0", 1)
				writeFile(t, filepath.Join(workspace, "packets/r1.sha256"), changed)
				writeFile(t, filepath.Join(workspace, "packets/r1.code-reviewer.md"), packet(changed))
			case "missing account":
				ledger = `{"waves":{}}`
			case "ambiguous account":
				ledger = `{"waves":{"r1":{"phase":"build","state":"complete","roles":{"code-reviewer":{"handle":"review-1","started":"2026-11-13T00:00:00Z","returned":true,"returned_at":"2026-11-13T00:01:00Z"},"test-analyst":{"handle":"review-2","started":"2026-11-13T00:00:00Z","returned":true,"returned_at":"2026-11-13T00:01:00Z"}}}}}`
				writeFile(t, filepath.Join(workspace, "packets/r1-test-analyst.return.md"), body)
			case "missing packet":
				if err := os.Remove(filepath.Join(workspace, "packets/r1.code-reviewer.md")); err != nil {
					t.Fatal(err)
				}
			case "wrong packet":
				writeFile(t, filepath.Join(workspace, "packets/r1.code-reviewer.md"), "Candidate:\naggregate wrong\n")
			case "original":
				stored = writeReturn(t, workspace, body)
			case "unstored account":
				ledger = `{"waves":{"r1":{"phase":"build","state":"sealed","roles":{"code-reviewer":{"handle":"review-1","started":"2026-11-13T00:00:00Z"}}}}}`
				if err := os.Remove(stored); err != nil {
					t.Fatal(err)
				}
				stored = writeReturn(t, workspace, body)
			case "missing manifest":
				if err := os.Remove(filepath.Join(workspace, "packets/r1.sha256")); err != nil {
					t.Fatal(err)
				}
			}
			writeFile(t, filepath.Join(workspace, "dispatch.json"), ledger)
			args = []string{"feat", stored}
			if scenario != "same tree" && scenario != "missing worktree" {
				args = append(args, "--worktree", worker)
			}
			if scenario == "explicit account" || scenario == "unstored account" {
				args = append(args, "--wave", "r1", "--role", "code-reviewer")
			}
			code, output := checkAnchors(t, root, args...)
			want := 3
			if scenario == "worker" || scenario == "original" || scenario == "explicit account" || scenario == "unstored account" || scenario == "same tree" {
				want = 0
			}
			if code != want {
				t.Fatalf("want %d got %d: %s", want, code, output)
			}
			if want == 0 {
				digest := strings.TrimSpace(strings.Split(sums, "aggregate ")[1])
				if !strings.Contains(output, fmt.Sprintf("candidate %s (1 files)", digest)) || !strings.Contains(output, "anchored=1") {
					t.Fatal(output)
				}
			}
		})
	}
}

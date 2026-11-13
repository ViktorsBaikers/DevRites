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
	for _, scenario := range []string{"worker", "original", "explicit account", "unstored account", "control quote", "changed bytes", "changed head", "wrong aggregate", "wrong file hash", "wrong scope", "missing account", "missing ledger and freeze", "ambiguous account", "missing packet", "wrong packet", "missing manifest", "missing worktree", "outside scope", "workspace document shadow", "canonical document collision", "deleted document shadow", "explicit metadata document", "frozen product document", "same tree"} {
		t.Run(scenario, func(t *testing.T) {
			root, workspace := anchorsProject(t)
			project := filepath.Dir(root)
			initDupRepo(t, project)
			runDupGit(t, project, "add", "src/a.go")
			if scenario == "deleted document shadow" {
				writeFile(t, filepath.Join(project, "README.md"), "Original product documentation.\n")
				runDupGit(t, project, "add", "README.md")
			}
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
				if scenario == "deleted document shadow" {
					runDupGit(t, worker, "rm", "README.md")
				}
				if scenario == "frozen product document" {
					writeFile(t, filepath.Join(worker, "README.md"), "return workerIdentity\n")
					runDupGit(t, worker, "add", "README.md")
				}
				runDupGit(t, worker, "commit", "-qm", "worker candidate")
			}
			out, errs := &bytes.Buffer{}, &bytes.Buffer{}
			args := []string{"feat", "--wave", "r1", "--paths", "src/a.go"}
			if scenario == "deleted document shadow" || scenario == "frozen product document" {
				args = append(args, "README.md")
			}
			args = append(args, flags...)
			if code := RunCandidateFreeze(root, args, out, errs); code != 0 {
				t.Fatalf("freeze %d: %s %s", code, out, errs)
			}
			sums := readTestFile(t, filepath.Join(workspace, "packets/r1.sha256"))
			// A different control candidate must not influence the review account.
			writeFile(t, filepath.Join(project, "src/control-only.go"), "package a // control-only scope\n")
			writeFile(t, filepath.Join(workspace, "touched-files.md"), candidateTestManifest(
				candidateTestRow{state: "present", path: "src/a.go", slice: "S-1", reason: "control"},
				candidateTestRow{state: "present", path: "src/control-only.go", slice: "S-1", reason: "control-only"},
			))
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
			if scenario == "workspace document shadow" || scenario == "canonical document collision" {
				location = "docs/guide.md:1"
				if scenario == "canonical document collision" {
					location = "README.md:1"
				}
				name := strings.TrimSuffix(location, ":1")
				writeFile(t, filepath.Join(project, name), "Control-only documentation.\n")
				writeFile(t, filepath.Join(workspace, name), "return workerIdentity\n")
			}
			if scenario == "deleted document shadow" || scenario == "frozen product document" {
				location = "README.md:1"
				writeFile(t, filepath.Join(workspace, "README.md"), "return workerIdentity\n")
				if scenario == "frozen product document" {
					writeFile(t, filepath.Join(workspace, "README.md"), "Different workspace documentation.\n")
				} else if err := os.Remove(filepath.Join(project, "README.md")); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "explicit metadata document" {
				writeFile(t, filepath.Join(workspace, "docs/guide.md"), "return workerIdentity\n")
				location = workspaceRel(root, workspace) + "/docs/guide.md:1"
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
			if scenario == "missing ledger and freeze" {
				ledger = `{"waves":{}}`
				if err := os.Remove(filepath.Join(workspace, "packets/r1.sha256")); err != nil {
					t.Fatal(err)
				}
				body = strings.ReplaceAll(body, "src/a.go:2", "src/a.go:4")
				body = strings.ReplaceAll(body, "return workerIdentity", "return r.User")
				writeFile(t, stored, body)
			}
			writeFile(t, filepath.Join(workspace, "dispatch.json"), ledger)
			args = []string{"feat", stored}
			if scenario != "same tree" && scenario != "missing worktree" && scenario != "missing account" && scenario != "missing ledger and freeze" {
				args = append(args, "--worktree", worker)
			}
			if scenario == "explicit account" || scenario == "unstored account" {
				args = append(args, "--wave", "r1", "--role", "code-reviewer")
			}
			code, output := checkAnchors(t, root, args...)
			want := 3
			if scenario == "worker" || scenario == "original" || scenario == "explicit account" || scenario == "unstored account" || scenario == "same tree" || scenario == "explicit metadata document" || scenario == "frozen product document" {
				want = 0
			}
			if code != want {
				t.Fatalf("want %d got %d: %s", want, code, output)
			}
			if want == 0 {
				digest := strings.TrimSpace(strings.Split(sums, "aggregate ")[1])
				files := 1
				if scenario == "frozen product document" {
					files = 2
				}
				if !strings.Contains(output, fmt.Sprintf("candidate %s (%d files)", digest, files)) || !strings.Contains(output, "anchored=1") {
					t.Fatal(output)
				}
			}
		})
	}
}

func TestCheckAnchorsPrebuildMetadata(t *testing.T) {
	for _, scenario := range []string{"metadata", "product quote", "missing context", "stale candidate declaration", "worker selection"} {
		t.Run(scenario, func(t *testing.T) {
			root, workspace := anchorsProject(t)
			writeFile(t, filepath.Join(workspace, "plan.md"), "Every handler must validate caller permissions.\n")
			writeFile(t, filepath.Join(workspace, "ctx/vet-plan-reviewer.bundle.md"), "Read the plan and spec.\n")
			var stdout, stderr bytes.Buffer
			if code := RunPacket(root, []string{"feat", "--wave", "v1", "--phase", "vet", "--role", "plan-reviewer"}, &stdout, &stderr); code != 0 {
				t.Fatalf("packet: %d %s %s", code, &stdout, &stderr)
			}
			writeFile(t, filepath.Join(workspace, "dispatch.json"), `{"waves":{"v1":{"phase":"vet","state":"complete","roles":{"plan-reviewer":{"handle":"plan-1","started":"2026-11-13T00:00:00Z","returned":true,"returned_at":"2026-11-13T00:01:00Z"}}}}}`)
			location, quote := "plan.md:1", "`Every handler must validate caller permissions`"
			if scenario == "product quote" {
				location, quote = "src/a.go:4", "`return r.User`"
			}
			if scenario == "missing context" {
				if err := os.Remove(filepath.Join(workspace, "packets/v1.plan-reviewer.context.bundle.md")); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "stale candidate declaration" {
				packet := filepath.Join(workspace, "packets/v1.plan-reviewer.md")
				writeFile(t, packet, readTestFile(t, packet)+"\nCandidate:\ndevrites-candidate-sha256 v1\n")
			}
			ret := filepath.Join(workspace, "packets/v1-plan-reviewer.return.md")
			writeFile(t, ret, "Outcome: findings\nCounts: important=1\n"+anchorsRow("Important", location, quote, anchorsGoodBreaks)+"\n")
			args := []string{"feat", ret}
			if scenario == "worker selection" {
				args = append(args, "--worktree", filepath.Dir(root))
			}
			code, out := checkAnchors(t, root, args...)
			want := 3
			if scenario == "metadata" {
				want = 0
			}
			if code != want {
				t.Fatalf("want %d got %d: %s", want, code, out)
			}
			if want == 0 && (!strings.Contains(out, "anchored=1") || strings.Contains(out, "quotes read the live tree")) {
				t.Fatal(out)
			}
		})
	}
}

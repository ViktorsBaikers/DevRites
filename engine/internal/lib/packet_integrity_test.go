package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPacketAndDispatchRejectTamperedFrozenDiff(t *testing.T) {
	for _, phase := range []string{"build", "review", "seal"} {
		t.Run(phase, func(t *testing.T) {
			root, workspace, _ := packetTestWorkspace(t)
			writeFile(t, filepath.Join(workspace, "ctx", phase+"-test-analyst.bundle.md"), "bundle\n")
			if code, out := runPacketT(t, root, "--wave", "r1", "--phase", phase, "--role", "test-analyst"); code != 0 {
				t.Fatalf("initial packet: code=%d %s", code, out)
			}
			packet := filepath.Join(workspace, "packets", "r1.test-analyst.md")
			prior := readTestFile(t, packet)
			writeFile(t, filepath.Join(workspace, "packets", "r1.diff"), "malformed nonempty replacement\n")
			if code, out := runPacketT(t, root, "--wave", "r1", "--phase", phase, "--role", "test-analyst"); code != 3 || !strings.Contains(out, "SHA256 mismatch") {
				t.Fatalf("tampered packet: code=%d %s", code, out)
			}
			if readTestFile(t, packet) != prior {
				t.Fatal("refused packet replaced prior valid packet")
			}
			var stdout, stderr bytes.Buffer
			code := RunDispatch(root, []string{"feature", "open", "--phase", phase, "--wave", "r1", "--role", "test-analyst", "--packet", packet}, &stdout, &stderr)
			if code != 3 || !strings.Contains(stderr.String(), "SHA256 mismatch") {
				t.Fatalf("direct dispatch: code=%d %s%s", code, &stdout, &stderr)
			}
			if _, err := os.Stat(filepath.Join(workspace, "dispatch.json")); !os.IsNotExist(err) {
				t.Fatalf("refusal published dispatch state: %v", err)
			}
		})
	}
}

func TestPacketLargeHeaderStillVerifiesDiff(t *testing.T) {
	root, workspace, sums := packetTestWorkspace(t)
	large := strings.Replace(sums, "aggregate ", strings.Repeat("file absent absent.txt\n", 4000)+"aggregate ", 1)
	writeFile(t, filepath.Join(workspace, "packets", "r1.sha256"), large)
	if _, partial, err := frozenCandidate(workspace, "r1"); err != nil || !partial {
		t.Fatalf("large valid candidate: partial=%t err=%v", partial, err)
	}
	if code, out := runPacketT(t, root, "--wave", "r1", "--phase", "review", "--role", "code-reviewer"); code != 0 {
		t.Fatalf("large valid packet: code=%d %s", code, out)
	}
	packet := filepath.Join(workspace, "packets", "r1.code-reviewer.md")
	writeFile(t, filepath.Join(workspace, "packets", "r1.diff"), "tampered\n")
	if code, out := runPacketT(t, root, "--wave", "r1", "--phase", "review", "--role", "code-reviewer"); code != 3 || !strings.Contains(out, "SHA256 mismatch") {
		t.Fatalf("large tampered candidate: code=%d %s", code, out)
	}
	var stdout, stderr bytes.Buffer
	if code := RunDispatch(root, []string{"feature", "open", "--phase", "review", "--wave", "r1", "--role", "code-reviewer", "--packet", packet}, &stdout, &stderr); code != 3 || !strings.Contains(stderr.String(), "SHA256 mismatch") {
		t.Fatalf("large-header direct dispatch: code=%d %s%s", code, &stdout, &stderr)
	}
}

func TestReviewDiffRequiresExplicitMatchingChecksum(t *testing.T) {
	_, root := writeCandidateTestWorkspace(t, candidateTestManifest())
	workspace := filepath.Join(root, "work", "feature")
	patch := "diff --git a/a b/a\n"
	writeFile(t, filepath.Join(workspace, "packets", "manual.patch"), patch)
	for _, body := range []string{
		"Read packets/manual.patch\n",
		"diff not-a-sha packets/manual.patch\n",
		fmt.Sprintf("diff %064d packets/manual.patch\n", 0),
	} {
		if err := lintReviewCandidate(root, "feature", writePacket(t, body)); err == nil {
			t.Fatalf("unbound/malformed/mismatched patch accepted: %s", body)
		}
	}
	packet := writePacket(t, fmt.Sprintf("Read packets/manual.patch\ndiff %x packets/manual.patch\n", sha256.Sum256([]byte(patch))))
	if err := lintReviewCandidate(root, "feature", packet); err != nil {
		t.Fatalf("explicit manual patch checksum refused: %v", err)
	}
	writeFile(t, filepath.Join(workspace, "packets", "manual.patch"), "diff --git a/b b/b\n")
	if err := lintReviewCandidate(root, "feature", packet); err == nil {
		t.Fatal("same-size patch mutation accepted")
	}
}

func TestFrozenDiffStreamingBoundsAndOrigin(t *testing.T) {
	_, root := writeCandidateTestWorkspace(t, candidateTestManifest())
	workspace := filepath.Join(root, "work", "feature")
	name := filepath.Join(workspace, "packets", "large.diff")
	content := strings.Repeat("x", (17<<20)+1)
	writeFile(t, name, content)
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	packet := writePacket(t, fmt.Sprintf("diff %s packets/large.diff\n", digest))
	if err := lintReviewCandidate(root, "feature", packet); err != nil {
		t.Fatalf("diff above workspace 16 MiB allowance refused: %v", err)
	}
	if err := os.Truncate(name, maxFrozenDiffBytes+1); err != nil {
		t.Fatal(err)
	}
	if err := lintReviewCandidate(root, "feature", packet); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("oversized diff accepted: %v", err)
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.diff")
	writeFile(t, outside, content[:10])
	if err := os.Symlink(outside, name); err != nil {
		t.Fatal(err)
	}
	if err := lintReviewCandidate(root, "feature", packet); err == nil {
		t.Fatal("symlink diff accepted")
	}
}

func TestCandidateFreezeConcurrentWavesKeepOneConsistentPair(t *testing.T) {
	project, root := writeCandidateTestWorkspace(t, "")
	initDupRepo(t, project)
	writeCandidateTestFile(t, project, "base.txt", "base\n")
	runDupGit(t, project, "add", "base.txt")
	runDupGit(t, project, "commit", "-q", "-m", "base")
	writeCandidateTestFile(t, project, "a.txt", "a\n")
	writeCandidateTestFile(t, project, "b.txt", "b\n")
	type result struct {
		code int
		out  string
	}
	start, done := make(chan struct{}), make(chan result, 2)
	for _, name := range []string{"a.txt", "b.txt"} {
		go func(name string) {
			<-start
			var stdout, stderr bytes.Buffer
			code := RunCandidateFreeze(root, []string{"feature", "--wave", "same", "--paths", name}, &stdout, &stderr)
			done <- result{code, stdout.String() + stderr.String()}
		}(name)
	}
	close(start)
	wins, refusals := 0, 0
	for range 2 {
		got := <-done
		switch {
		case got.code == 0:
			wins++
		case got.code == 3 && strings.Contains(got.out, "already freezes a different candidate"):
			refusals++
		default:
			t.Fatalf("unexpected freeze result: %+v", got)
		}
	}
	if wins != 1 || refusals != 1 {
		t.Fatalf("wins=%d refusals=%d", wins, refusals)
	}
	workspace := filepath.Join(root, "work", "feature")
	if _, _, err := frozenCandidate(workspace, "same"); err != nil {
		t.Fatalf("concurrent publication mixed diff and sums: %v", err)
	}
}

func TestCandidateFreezeRevalidationRefusesObservedDrift(t *testing.T) {
	for _, mutation := range []string{"same-size", "absent-appears", "linked-head"} {
		t.Run(mutation, func(t *testing.T) {
			_, _, base, project := linkedWorktreeRepo(t)
			writeCandidateTestFile(t, project, "src/a.go", "package b\n")
			runDupGit(t, project, "commit", "-q", "-am", "candidate")
			head := strings.TrimSpace(string(mustGitOutput(t, project, "rev-parse", "HEAD")))
			rows, err := freezeRows(project, []string{"src/a.go", "absent.txt"})
			if err != nil {
				t.Fatal(err)
			}
			if err := validateCandidateRows(project, rows); err != nil {
				t.Fatal(err)
			}
			digest, err := hashCandidate(project, rows, true)
			if err != nil {
				t.Fatal(err)
			}
			diff, commit, err := freezeDiff(project, base, rows)
			if err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "same-size":
				writeCandidateTestFile(t, project, "src/a.go", "package c\n")
				info, _, err := candidatePathInfo(project, "src/a.go")
				if err != nil {
					t.Fatal(err)
				}
				for _, row := range rows {
					if row.state == "present" {
						if err := os.Chtimes(filepath.Join(project, row.path), info.ModTime(), row.info.ModTime()); err != nil {
							t.Fatal(err)
						}
					}
				}
			case "absent-appears":
				writeCandidateTestFile(t, project, "absent.txt", "appeared\n")
			case "linked-head":
				runDupGit(t, project, "commit", "-q", "--allow-empty", "-m", "head moved")
			}
			if err := revalidateFrozenCandidate(project, commit, rows, digest, diff, head); err == nil {
				t.Fatalf("%s drift accepted", mutation)
			}
		})
	}
}

func TestConcurrentPacketRendersKeepForeignDraftAndPriorPacket(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	foreign := filepath.Join(workspace, "packets", ".r1.code-reviewer.md.draft-foreign")
	writeFile(t, foreign, "another invocation owns this\n")
	writeFile(t, filepath.Join(workspace, "bad-focus.md"), "- F-2 src/a.go:3 tests already passed\n")
	start, done := make(chan struct{}), make(chan int, 2)
	for _, focus := range []string{"focus.md", "bad-focus.md"} {
		go func(focus string) {
			<-start
			var stdout, stderr bytes.Buffer
			done <- RunPacket(root, []string{"feature", "--wave", "r1", "--phase", "review", "--role", "code-reviewer", "--focus", focus}, &stdout, &stderr)
		}(focus)
	}
	close(start)
	a, b := <-done, <-done
	if !((a == 0 && b == 3) || (a == 3 && b == 0)) {
		t.Fatalf("render codes %d, %d", a, b)
	}
	if readTestFile(t, foreign) != "another invocation owns this\n" {
		t.Fatal("render touched another invocation's draft")
	}
	packet := filepath.Join(workspace, "packets", "r1.code-reviewer.md")
	if err := lintRenderedPacket(root, "feature", packet, "code-reviewer", false, true); err != nil {
		t.Fatalf("refused concurrent render displaced valid packet: %v", err)
	}
	if drafts, err := filepath.Glob(filepath.Join(workspace, "packets", ".r1.code-reviewer.md.draft-*")); err != nil || len(drafts) != 1 || drafts[0] != foreign {
		t.Fatalf("draft ownership/cleanup: %v (%v)", drafts, err)
	}
}

func renderAndDispatchReviewT(t *testing.T, root, wave string) string {
	t.Helper()
	if code, out := runPacketT(t, root, "--wave", wave, "--phase", "review", "--role", "code-reviewer"); code != 0 {
		t.Fatalf("packet: code=%d %s", code, out)
	}
	packet := filepath.Join(root, "work", "feature", "packets", wave+".code-reviewer.md")
	var stdout, stderr bytes.Buffer
	if code := RunDispatch(root, []string{"feature", "open", "--phase", "review", "--wave", wave, "--role", "code-reviewer", "--packet", packet}, &stdout, &stderr); code != 0 {
		t.Fatalf("dispatch: code=%d %s%s", code, &stdout, &stderr)
	}
	return packet
}

func TestPacketCandidatePatchAndDiffMetadataReachDispatch(t *testing.T) {
	for _, state := range []string{"present", "deleted"} {
		for _, suffix := range []string{".patch", ".diff", " set.patch", " set.diff"} {
			t.Run(state+suffix, func(t *testing.T) {
				name := "fixtures/change" + suffix
				manifest := candidateTestManifest(candidateTestRow{state: state, path: name, slice: "S-1", reason: "Fixture change."})
				project, root := writeCandidateTestWorkspace(t, manifest)
				initDupRepo(t, project)
				writeCandidateTestFile(t, project, name, "old fixture\n")
				runDupGit(t, project, "add", name)
				runDupGit(t, project, "commit", "-q", "-m", "baseline")
				if state == "present" {
					writeCandidateTestFile(t, project, name, "new fixture\n")
				} else if err := os.Remove(filepath.Join(project, name)); err != nil {
					t.Fatal(err)
				}
				if code, out := runFreezeT(t, root, "--wave", "r1", "--paths", name); code != 0 {
					t.Fatalf("freeze: code=%d %s", code, out)
				}
				writeFile(t, filepath.Join(root, "work", "feature", "ctx", "review-code-reviewer.bundle.md"), "bundle\n")
				packet := renderAndDispatchReviewT(t, root, "r1")
				// A real reviewer-input reference still needs its own binding even when also listed as a candidate file.
				unbound := writePacket(t, readTestFile(t, packet)+"\nRead first:\n- "+name+"\n")
				if err := lintReviewCandidate(root, "feature", unbound); err == nil {
					t.Fatal("candidate metadata hid an unbound reviewer-input reference")
				}
			})
		}
	}
}

func TestPacketWorkspaceDiffNamespaceSurvivesProductCollision(t *testing.T) {
	for _, sameBytes := range []bool{false, true} {
		t.Run(fmt.Sprintf("same-bytes=%t", sameBytes), func(t *testing.T) {
			root, workspace, _ := packetTestWorkspace(t)
			productDiff := filepath.Join(projectDir(root), "packets", "r1.diff")
			productBytes := "different product diff\n"
			if sameBytes {
				productBytes = readTestFile(t, filepath.Join(workspace, "packets", "r1.diff"))
			}
			writeFile(t, productDiff, productBytes)
			packet := renderAndDispatchReviewT(t, root, "r1")
			// An absolute project-qualified input remains independently bindable.
			manual := writePacket(t, fmt.Sprintf("Read first:\n- %s\ndiff %x %s\n", productDiff, sha256.Sum256([]byte(productBytes)), productDiff))
			var stdout, stderr bytes.Buffer
			if code := RunDispatch(root, []string{"feature", "open", "--phase", "review", "--wave", "manual", "--role", "code-reviewer", "--packet", manual}, &stdout, &stderr); code != 0 {
				t.Fatalf("project-qualified manual packet: code=%d %s%s", code, &stdout, &stderr)
			}
			writeFile(t, filepath.Join(workspace, "packets", "r1.diff"), "tampered workspace diff\n")
			stdout.Reset()
			stderr.Reset()
			if code := RunDispatch(root, []string{"feature", "open", "--phase", "review", "--wave", "tampered", "--role", "code-reviewer", "--packet", packet}, &stdout, &stderr); code != 3 || !strings.Contains(stderr.String(), "SHA256 mismatch") {
				t.Fatalf("product collision hid workspace tampering: code=%d %s%s", code, &stdout, &stderr)
			}
		})
	}
}

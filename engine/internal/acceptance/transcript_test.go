package acceptance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/state"
)

func TestWriteTranscriptsBindsRunToCandidateAndRedacts(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	work := filepath.Dir(ledgerPath)
	token := "ghp_" + strings.Repeat("a", 36)
	check := "echo TRANSCRIPT-MARKER " + token
	if err := os.Mkdir(filepath.Join(project, token), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLedger(t, ledgerPath, "- [ ] G1: probe "+token+"\n  CHECK: "+check+"\n  CWD: "+token+"\n  EXPECT: TRANSCRIPT-MARKER "+token+"\n  EVIDENCE: pending\n")
	approved := []ApprovedCommand{{Command: check, Cwd: token}}
	before := strings.Repeat("b", 64)
	var first string
	for attempt := 1; attempt <= 2; attempt++ {
		doc := ParseLedger(mustRead(t, ledgerPath))
		results := RunGates(context.Background(), project, ledgerPath, doc, approved, true, 10*time.Second, nil)
		paths, err := writeTranscripts(work, doc, results, before, before)
		if err != nil || len(paths) != 1 || !results[0].Passed {
			t.Fatalf("attempt %d: paths=%v results=%+v err=%v", attempt, paths, results, err)
		}
		body := mustRead(t, paths[0])
		if strings.Contains(body, token) {
			t.Fatal("transcript metadata and output must not keep credentials")
		}
		for _, want := range []string{"EXIT: 0", "RESULT: pass", "CANDIDATE_BEFORE: " + before,
			"OUTPUT_SHA256: " + OutputDigest(results[0].Outcome.Output), "TRANSCRIPT-MARKER [REDACTED:github-token]"} {
			if !strings.Contains(body, want) {
				t.Fatalf("attempt %d transcript lacks %q", attempt, want)
			}
		}
		if info, err := os.Stat(paths[0]); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
			t.Fatalf("transcript must be private: info=%v err=%v", info, err)
		}
		if first == paths[0] {
			t.Fatal("executed attempts must not overwrite historical proof")
		}
		first = paths[0]
	}
	doc := ParseLedger(mustRead(t, ledgerPath))
	results := RunGates(context.Background(), project, ledgerPath, doc, approved, false, 10*time.Second, &RunOptions{TranscriptWork: work})
	if results[0].Ran || !results[0].Reused || !results[0].Passed {
		t.Fatalf("redacted definition and output must retain reusable identity: %+v", results)
	}
	original := mustRead(t, results[0].Origin)
	paths, err := writeTranscripts(work, doc, results, strings.Repeat("c", 64), strings.Repeat("d", 64))
	if err != nil || len(paths) != 1 || paths[0] != results[0].Origin || mustRead(t, paths[0]) != original {
		t.Fatalf("reuse must retain original candidate identity: paths=%v err=%v", paths, err)
	}
}

func runTranscriptCLI(t *testing.T, root, verb string, transcripts bool, want int) string {
	t.Helper()
	args := []string{verb, "feat"}
	if transcripts {
		args = append(args, "--transcripts")
	}
	var stdout, stderr strings.Builder
	if code := Run(root, args, &stdout, &stderr); code != want {
		t.Fatalf("%v: exit=%d want=%d stdout=%q stderr=%q", args, code, want, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func writeTranscriptPlan(t *testing.T, work string, checks ...string) {
	t.Helper()
	plan := "## Build-entry preflight\n\n| Command | Cwd | Purpose |\n| --- | --- | --- |\n"
	for _, check := range checks {
		plan += "| " + check + " | . | probe |\n"
	}
	writeLedger(t, filepath.Join(work, "test-plan.md"), plan)
}

func transcriptPath(t *testing.T, output string, index int) string {
	t.Helper()
	var paths []string
	for _, line := range strings.Split(output, "\n") {
		if path, ok := strings.CutPrefix(line, "transcript: "); ok {
			paths = append(paths, path)
		}
	}
	if index >= len(paths) {
		t.Fatalf("missing transcript %d in %q", index, output)
	}
	return paths[index]
}

func TestGateTranscriptsRecoverOnlyMissingOrigins(t *testing.T) {
	for _, inputs := range []bool{false, true} {
		t.Run(fmt.Sprintf("inputs=%t", inputs), func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
			check := "echo x>>runs.txt && echo OK"
			attr := ""
			if inputs {
				writeLedger(t, filepath.Join(project, "src.txt"), "source")
				attr = "  INPUTS: src.txt\n"
			}
			body := ""
			for _, id := range []string{"G1", "G2"} {
				body += "- [ ] " + id + ": probe\n  CHECK: " + check + "\n  EXPECT: OK\n" + attr + "  EVIDENCE: pending\n"
			}
			writeLedger(t, ledgerPath, body)
			writeTranscriptPlan(t, work, check)
			runTranscriptCLI(t, root, "run", false, ExitOK)
			output := runTranscriptCLI(t, root, "run", true, ExitOK)
			first, second := transcriptPath(t, output, 0), transcriptPath(t, output, 1)
			if got := strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x"); got != 2 {
				t.Fatalf("missing origins must execute shared CHECK once: %d", got)
			}
			// Candidate-wide drift must not retire scoped INPUTS proof.
			writeLedger(t, filepath.Join(project, "unrelated.txt"), "changed")
			verb := "run"
			if inputs {
				verb = "reverify"
			}
			output = runTranscriptCLI(t, root, verb, true, ExitOK)
			if !strings.Contains(output, "gate G1: skipped") || !strings.Contains(output, "gate G2: skipped") {
				t.Fatalf("valid origins must reuse: %s", output)
			}
			if err := os.Remove(first); err != nil {
				t.Fatal(err)
			}
			output = runTranscriptCLI(t, root, verb, true, ExitOK)
			if !strings.Contains(output, "gate G1: pass") || !strings.Contains(output, "gate G2: skipped") || transcriptPath(t, output, 1) != second {
				t.Fatalf("only missing origin may execute: %s", output)
			}
			if got := strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x"); got != 3 {
				t.Fatalf("valid proof must not execute again: %d", got)
			}
			// A still-checked ledger cannot authorize a revoked command, even
			// when origin recovery would otherwise execute it.
			if err := os.Remove(transcriptPath(t, output, 0)); err != nil {
				t.Fatal(err)
			}
			writeTranscriptPlan(t, work)
			output = runTranscriptCLI(t, root, verb, true, ExitBlocked)
			if strings.Contains(output, "result: all-met") || strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x") != 3 {
				t.Fatalf("revoked approval must block without execution: %s", output)
			}
		})
	}
}

// Editing a record and recomputing its storage checksum still cannot change
// the definition, output or INPUTS identity bound by the ledger.
func rewriteTranscriptField(body, key, value string) string {
	end := strings.LastIndex(body, "TRANSCRIPT_SHA256: ")
	payload := body[:end]
	lines := strings.Split(payload, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, key+": ") {
			if value == "" {
				lines = append(lines[:i], lines[i+1:]...)
			} else {
				lines[i] = key + ": " + value
			}
			break
		}
	}
	payload = strings.Join(lines, "\n")
	return payload + "TRANSCRIPT_SHA256: " + OutputDigest(payload) + "\n"
}

func TestGateTranscriptsRejectUnusableOrigins(t *testing.T) {
	mutations := map[string]func(string) string{
		"incomplete": func(s string) string { return s[:len(s)/2] },
		"reused-only": func(string) string {
			return "DEVRITES-GATE-TRANSCRIPT v1\nRESULT: reused (inputs unchanged; not executed)\n"
		},
		"corrupt-output": func(s string) string { return strings.Replace(s, "\nOK\n", "\nNO\n", 1) },
	}
	for key, value := range map[string]string{
		"DEFINITION_SHA256": strings.Repeat("0", 64), "COMMAND": "echo OTHER", "CWD": "elsewhere",
		"GATE": "OTHER", "TITLE": "OTHER",
		"EXPECT": "OTHER", "INPUTS": "elsewhere.txt", "INPUTS_SHA256": strings.Repeat("0", 64),
		"OUTPUT_SHA256": strings.Repeat("0", 64), "OUTPUT_BYTES": "999", "EXIT": "1", "RESULT": "fail",
		"STARTED": "not-a-time", "ENDED": "not-a-time", "ATTEMPT": "0", "REDACTIONS": "-1", "TRUNCATED": "invalid",
	} {
		mutations["wrong-"+key] = func(s string) string { return rewriteTranscriptField(s, key, value) }
		mutations["missing-"+key] = func(s string) string { return rewriteTranscriptField(s, key, "") }
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
			check := "echo x>>runs.txt && echo OK"
			writeLedger(t, filepath.Join(project, "src.txt"), "source")
			writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  INPUTS: src.txt\n  EVIDENCE: pending\n")
			writeTranscriptPlan(t, work, check)
			path := transcriptPath(t, runTranscriptCLI(t, root, "run", true, ExitOK), 0)
			writeLedger(t, path, mutate(mustRead(t, path)))
			output := runTranscriptCLI(t, root, "reverify", true, ExitOK)
			if !strings.Contains(output, "gate G1: pass") || strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x") != 2 {
				t.Fatalf("unusable origin must execute, not certify old checkbox: %s", output)
			}
		})
	}
}

func TestGateTranscriptsRejectSymlinkOrigins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs elevated privileges on Windows")
	}
	for _, parent := range []bool{false, true} {
		t.Run(fmt.Sprintf("parent=%t", parent), func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
			check := "echo x>>runs.txt && echo OK"
			writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  EVIDENCE: pending\n")
			writeTranscriptPlan(t, work, check)
			path := transcriptPath(t, runTranscriptCLI(t, root, "run", true, ExitOK), 0)
			original := mustRead(t, path)
			link, target := path, filepath.Join(t.TempDir(), "origin.log")
			if parent {
				link, target = filepath.Dir(path), filepath.Join(t.TempDir(), "origins")
				if err := os.Rename(link, target); err != nil {
					t.Fatal(err)
				}
			} else {
				writeLedger(t, target, original)
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			want := ExitOK
			if parent {
				want = ExitBlocked // execution succeeds, unsafe publication refuses
			}
			output := runTranscriptCLI(t, root, "run", true, want)
			if !strings.Contains(output, "gate G1: pass") || strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x") != 2 {
				t.Fatalf("symlink origin must never certify a skip: %s", output)
			}
			if parent {
				target = filepath.Join(target, filepath.Base(path))
			}
			if mustRead(t, target) != original {
				t.Fatal("origin recovery must not overwrite symlink targets")
			}
		})
	}
}

func TestGateTranscriptsInterruptedPublicationRecovers(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
	check := "echo x>>runs.txt && echo OK"
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  EVIDENCE: pending\n")
	writeTranscriptPlan(t, work, check)
	runTranscriptCLI(t, root, "run", false, ExitOK)
	// Execution committed evidence, but an interrupted publication left no
	// complete origin. Neither a private draft nor an incomplete log counts.
	dir := filepath.Join(work, "references", "gates", "G1")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeLedger(t, filepath.Join(dir, ".nodigest-a1.log.tmp-interrupted"), "partial")
	writeLedger(t, filepath.Join(dir, "nodigest-a1.log"), "DEVRITES-GATE-TRANSCRIPT v2\nRESULT: pass\n")
	output := runTranscriptCLI(t, root, "run", true, ExitOK)
	path := transcriptPath(t, output, 0)
	if filepath.Base(path) == "nodigest-a1.log" || !strings.Contains(output, "gate G1: pass") {
		t.Fatalf("incomplete history must survive while new origin publishes: %s", output)
	}
	output = runTranscriptCLI(t, root, "run", true, ExitOK)
	if transcriptPath(t, output, 0) != path || strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x") != 2 {
		t.Fatalf("complete recovery must reuse next time: %s", output)
	}
}

func TestGateTranscriptsRedactedTruncatedOriginReuses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX cat")
	}
	project, ledgerPath := writeWorkspace(t)
	work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
	// Each recognized short credential expands under redaction. The executed
	// output stays within capture bounds while its redacted form is truncated.
	chunk := "token='1234567890123456' "
	output := "OK\n" + strings.Repeat(chunk, (MaxOutputBytes-3)/len(chunk))
	writeLedger(t, filepath.Join(project, "output.txt"), output)
	check := "echo x>>runs.txt && cat output.txt"
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  INPUTS: output.txt\n  EVIDENCE: pending\n")
	writeTranscriptPlan(t, work, check)
	path := transcriptPath(t, runTranscriptCLI(t, root, "run", true, ExitOK), 0)
	body := mustRead(t, path)
	if !strings.Contains(body, "TRUNCATED: true") || strings.Contains(body, "1234567890123456") {
		t.Fatal("fixture must publish truncated, fully redacted output")
	}
	out := runTranscriptCLI(t, root, "reverify", true, ExitOK)
	if !strings.Contains(out, "gate G1: skipped") || transcriptPath(t, out, 0) != path || strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x") != 1 {
		t.Fatalf("stored redacted body must not be mistaken for original output: %s", out)
	}
}

func TestGateTranscriptsCommandHeaderDoesNotLeak(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX no-op")
	}
	project, ledgerPath := writeWorkspace(t)
	work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
	token := "ghp_" + strings.Repeat("z", 36)
	check := ": " + token + "; echo x>>runs.txt; echo OK"
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  EVIDENCE: pending\n")
	writeTranscriptPlan(t, work, check)
	out := runTranscriptCLI(t, root, "run", true, ExitOK)
	path := transcriptPath(t, out, 0)
	if strings.Contains(out+mustRead(t, path), token) {
		t.Fatal("recognized credential in COMMAND must not leak even with safe command output")
	}
	out = runTranscriptCLI(t, root, "run", true, ExitOK)
	if !strings.Contains(out, "gate G1: skipped") || strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x") != 1 {
		t.Fatalf("redacted COMMAND must retain safe reusable definition identity: %s", out)
	}
}

func TestGateTranscriptsPublicationHoldsExecutionLockNotFeatureLock(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
	check := "echo x>>runs.txt && echo OK"
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  EVIDENCE: pending\n")
	writeTranscriptPlan(t, work, check)
	prior := CandidateDigest
	var calls atomic.Int32
	publishing, release, secondCandidate := make(chan struct{}), make(chan struct{}), make(chan struct{})
	CandidateDigest = func(string, string) (string, error) {
		switch calls.Add(1) {
		case 2:
			close(publishing)
			<-release
		case 3:
			close(secondCandidate)
		}
		return strings.Repeat("b", 64), nil
	}
	defer func() { CandidateDigest = prior }()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	type outcome struct {
		code     int
		out, err string
	}
	run := func(done chan<- outcome) {
		var stdout, stderr strings.Builder
		code := Run(root, []string{"run", "feat", "--transcripts"}, &stdout, &stderr)
		done <- outcome{code, stdout.String(), stderr.String()}
	}
	first, second := make(chan outcome, 1), make(chan outcome, 1)
	go run(first)
	select {
	case <-publishing:
	case <-time.After(5 * time.Second):
		t.Fatal("first run did not reach publication")
	}
	featureAvailable := make(chan error, 1)
	go func() {
		featureAvailable <- state.WithFeatureLock(root, "feat", func() error { return nil })
	}()
	select {
	case err := <-featureAvailable:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("publication must not retain feature lock")
	}
	go run(second)
	select {
	case <-secondCandidate:
		t.Error("second invocation entered before first published its origin")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	for _, done := range []chan outcome{first, second} {
		select {
		case result := <-done:
			if result.code != ExitOK {
				t.Fatalf("concurrent run failed: %+v", result)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent transcript run did not finish")
		}
	}
	if strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x") != 1 {
		t.Fatal("concurrent transcript requests must execute once, then reuse complete origin")
	}
}

func TestGateTranscriptsPublicationFailureRetriesCheckedGate(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
	check := "echo x>>runs.txt && echo OK"
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  EVIDENCE: pending\n")
	writeTranscriptPlan(t, work, check)
	blocker := filepath.Join(work, "references")
	writeLedger(t, blocker, "not a directory")
	runTranscriptCLI(t, root, "run", true, ExitBlocked)
	doc := ParseLedger(mustRead(t, ledgerPath))
	if GateState(doc.Gates[0], doc) != StateMet {
		t.Fatal("fixture must leave executed evidence before publication failed")
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	output := runTranscriptCLI(t, root, "run", true, ExitOK)
	path := transcriptPath(t, output, 0)
	if !strings.Contains(output, "gate G1: pass") {
		t.Fatalf("missing origin after failed publication must re-execute: %s", output)
	}
	output = runTranscriptCLI(t, root, "run", true, ExitOK)
	if transcriptPath(t, output, 0) != path || strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x") != 2 {
		t.Fatalf("published recovery must reuse next time: %s", output)
	}
}

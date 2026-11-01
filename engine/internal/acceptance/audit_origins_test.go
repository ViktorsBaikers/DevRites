package acceptance

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func auditRunPreflight(t *testing.T, root string, slice bool) string {
	t.Helper()
	args := []string{"feat", "--timeout", "5"}
	if slice {
		args = append(args, "--slice", "SLICE-002")
	}
	var stdout, stderr strings.Builder
	if code := RunPreconditions(root, args, &stdout, &stderr); code != ExitOK {
		t.Fatalf("preflight execution: exit=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func auditRefreshTranscriptChecksum(t *testing.T, raw string) string {
	t.Helper()
	end := strings.LastIndex(raw, "TRANSCRIPT_SHA256: ")
	if end < 0 {
		t.Fatal("executed transcript has no checksum footer")
	}
	payload := raw[:end]
	return payload + "TRANSCRIPT_SHA256: " + OutputDigest(payload) + "\n"
}

func TestAuditPreflightRejectsInvalidExecutedOrigins(t *testing.T) {
	for _, mutation := range []string{"corrupt", "truncated", "different-row", "forged-command", "forged-cwd", "forged-exit", "forged-body", "pass-header-only"} {
		t.Run(mutation, func(t *testing.T) {
			root, work, rows := writePreconditionsPlan(t, preconditionsPlan)
			path := transcriptPath(t, auditRunPreflight(t, root, true), 0)
			if !PreflightPassed(work, rows[1]) {
				t.Fatal("real successful preflight must pass before corruption")
			}
			original := mustRead(t, path)
			var forged string
			switch mutation {
			case "corrupt":
				forged = strings.Replace(original, "--- OUTPUT ---\n", "--- OUTPUT ---\nCORRUPTED", 1)
			case "truncated":
				header, _, ok := strings.Cut(original, "--- OUTPUT ---")
				if !ok {
					t.Fatal("executed transcript has no output boundary")
				}
				forged = header
			case "different-row":
				other := transcriptPath(t, auditRunPreflight(t, root, false), 0)
				forged = mustRead(t, other)
			case "forged-command":
				forged = rewriteTranscriptField(original, "COMMAND", "echo OTHER")
			case "forged-cwd":
				forged = rewriteTranscriptField(original, "CWD", "elsewhere")
			case "forged-exit":
				forged = rewriteTranscriptField(original, "EXIT", "1")
			case "forged-body":
				forged = auditRefreshTranscriptChecksum(t, strings.Replace(original, "--- OUTPUT ---\nPRE-OK\n", "--- OUTPUT ---\nFORGED\n", 1))
			case "pass-header-only":
				forged = "DEVRITES-GATE-TRANSCRIPT v2\nRESULT: pass\n"
			}
			if forged == original {
				t.Fatal("mutation must change real executed transcript")
			}
			writeLedger(t, path, forged)
			if PreflightPassed(work, rows[1]) {
				t.Fatal("invalid retained origin must not satisfy preflight readiness")
			}
		})
	}
}

func TestAuditPreflightRejectsSymlinkParents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs elevated privileges on Windows")
	}
	for _, parent := range []string{"references", "gates", "row"} {
		t.Run(parent, func(t *testing.T) {
			root, work, rows := writePreconditionsPlan(t, preconditionsPlan)
			path := transcriptPath(t, auditRunPreflight(t, root, true), 0)
			original := mustRead(t, path)
			if !PreflightPassed(work, rows[1]) {
				t.Fatal("real successful preflight must pass before replacing parent")
			}
			link := filepath.Dir(path)
			if parent == "gates" {
				link = filepath.Dir(link)
			} else if parent == "references" {
				link = filepath.Dir(filepath.Dir(link))
			}
			target := filepath.Join(t.TempDir(), "origins")
			if err := os.Rename(link, target); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			if PreflightPassed(work, rows[1]) {
				t.Error("symlink parent must not supply preflight proof")
			}
			if mustRead(t, path) != original {
				t.Fatal("preflight lookup must not alter symlink target")
			}
		})
	}
}

func TestAuditPreflightRetainsHistoricalCandidateForUnchangedRow(t *testing.T) {
	plan := strings.Replace(preconditionsPlan, "echo PRE-OK", "echo x >> preflight-runs.txt && echo PRE-OK", 1)
	root, work, rows := writePreconditionsPlan(t, plan)
	prior := CandidateDigest
	candidate := strings.Repeat("b", 64)
	CandidateDigest = func(string, string) (string, error) { return candidate, nil }
	t.Cleanup(func() { CandidateDigest = prior })
	path := transcriptPath(t, auditRunPreflight(t, root, true), 0)
	original := mustRead(t, path)
	if !strings.Contains(original, "CANDIDATE_BEFORE: "+candidate+"\n") {
		t.Fatal("fixture must retain original candidate identity")
	}
	candidate = strings.Repeat("c", 64)
	writeLedger(t, filepath.Join(filepath.Dir(root), "unrelated.txt"), "new candidate")
	if !PreflightPassed(work, rows[1]) {
		t.Fatal("candidate drift alone must not retire unchanged preflight row")
	}
	auditRunPreflight(t, root, true)
	if got := mustRead(t, filepath.Join(filepath.Dir(root), "preflight-runs.txt")); got != "x\n" {
		t.Fatalf("unchanged row must retain pre-edit execution, got counter %q", got)
	}
	if mustRead(t, path) != original {
		t.Fatal("reuse must not relabel historical candidate")
	}
	changed := rows[1]
	changed.Command = "echo OTHER"
	if PreflightPassed(work, changed) {
		t.Error("changed command must retire old preflight pass")
	}
	changed = rows[1]
	changed.Cwd = "elsewhere"
	if PreflightPassed(work, changed) {
		t.Error("changed cwd must retire old preflight pass")
	}
}

func TestAuditTranscriptBodyForgeryForcesReexecution(t *testing.T) {
	for _, replacement := range []string{"NO\n", "FORGED BODY\n"} {
		t.Run(strings.TrimSpace(replacement), func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
			check := "echo x>>runs.txt && echo OK"
			writeLedger(t, filepath.Join(project, "src.txt"), "source")
			writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  INPUTS: src.txt\n  EVIDENCE: pending\n")
			writeTranscriptPlan(t, work, check)
			path := transcriptPath(t, runTranscriptCLI(t, root, "run", true, ExitOK), 0)
			original := mustRead(t, path)
			if !strings.Contains(original, "REDACTIONS: 0\nTRUNCATED: false\n--- OUTPUT ---\nOK\n\n--- END OUTPUT ---\n") {
				t.Fatal("fixture must contain complete unredacted successful output")
			}
			forged := strings.Replace(original, "\n--- OUTPUT ---\nOK\n", "\n--- OUTPUT ---\n"+replacement, 1)
			writeLedger(t, path, auditRefreshTranscriptChecksum(t, forged))
			out := runTranscriptCLI(t, root, "reverify", true, ExitOK)
			if got := mustRead(t, filepath.Join(project, "runs.txt")); got != "x\nx\n" {
				t.Fatalf("forged body must force real reexecution; counter=%q output=%s", got, out)
			}
			if transcriptPath(t, out, 0) == path {
				t.Fatal("reverification must publish new origin, not reuse forged body")
			}
		})
	}
}

func TestAuditTranscriptRedactedOriginsRemainReusable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX cat")
	}
	for _, truncated := range []bool{false, true} {
		name := "redacted"
		if truncated {
			name = "redacted-and-truncated"
		}
		t.Run(name, func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
			chunk := "token='1234567890123456' "
			output := "OK\n" + chunk
			if truncated {
				output = "OK\n" + strings.Repeat(chunk, (MaxOutputBytes-3)/len(chunk))
			}
			writeLedger(t, filepath.Join(project, "output.txt"), output)
			check := "echo x>>runs.txt && cat output.txt"
			writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  INPUTS: output.txt\n  EVIDENCE: pending\n")
			writeTranscriptPlan(t, work, check)
			path := transcriptPath(t, runTranscriptCLI(t, root, "run", true, ExitOK), 0)
			body := mustRead(t, path)
			wantTruncated := "TRUNCATED: false\n"
			if truncated {
				wantTruncated = "TRUNCATED: true\n"
			}
			if strings.Contains(body, "1234567890123456") || strings.Contains(body, "REDACTIONS: 0\n") || !strings.Contains(body, wantTruncated) {
				t.Fatal("fixture must retain redacted output with requested truncation state")
			}
			out := runTranscriptCLI(t, root, "reverify", true, ExitOK)
			if got := mustRead(t, filepath.Join(project, "runs.txt")); got != "x\n" {
				t.Fatalf("valid redacted origin must not rerun; counter=%q output=%s", got, out)
			}
			if transcriptPath(t, out, 0) != path || mustRead(t, path) != body {
				t.Fatal("reuse must retain original redacted transcript")
			}
		})
	}
}

func TestAuditFinalRetainedOriginsNeverExecutes(t *testing.T) {
	for _, mutation := range []string{"none", "missing", "body", "inputs", "approval"} {
		t.Run(mutation, func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			work, root := filepath.Dir(ledgerPath), filepath.Join(project, ".devrites")
			check := "echo x>>runs.txt && echo OK"
			writeLedger(t, filepath.Join(project, "src.txt"), "source")
			writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: OK\n  INPUTS: src.txt\n  EVIDENCE: pending\n")
			writeTranscriptPlan(t, work, check)
			path := transcriptPath(t, runTranscriptCLI(t, root, "run", true, ExitOK), 0)
			switch mutation {
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "body":
				raw := strings.Replace(mustRead(t, path), "\n--- OUTPUT ---\nOK\n", "\n--- OUTPUT ---\nNO\n", 1)
				writeLedger(t, path, auditRefreshTranscriptChecksum(t, raw))
			case "inputs":
				writeLedger(t, filepath.Join(project, "src.txt"), "changed")
			case "approval":
				writeTranscriptPlan(t, work, "echo OTHER")
			}
			err := CheckRetainedOrigins(root, "feat")
			if (err == nil) != (mutation == "none") {
				t.Fatalf("mutation %s: retained origin check returned %v", mutation, err)
			}
			if got := mustRead(t, filepath.Join(project, "runs.txt")); got != "x\n" {
				t.Fatalf("final check executed a command: %q", got)
			}
		})
	}
}

func TestAuditFinalRetainedOriginsLeavesManualGatesUnchanged(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	writeLedger(t, ledgerPath, "- [x] G1: manual review\n  EVIDENCE: reviewed by owner\n")
	if err := CheckRetainedOrigins(filepath.Join(project, ".devrites"), "feat"); err != nil {
		t.Fatalf("manual gate must not require an executed origin: %v", err)
	}
}

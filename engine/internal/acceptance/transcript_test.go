package acceptance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteTranscriptsBindsRunToCandidateAndRedacts(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	work := filepath.Dir(ledgerPath)
	token := "ghp_" + strings.Repeat("a", 36)
	check := "echo TRANSCRIPT-MARKER " + token
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: TRANSCRIPT-MARKER\n  EVIDENCE: pending\n")
	approved := []ApprovedCommand{{Command: check}}
	before := strings.Repeat("b", 64)

	for attempt := 1; attempt <= 2; attempt++ {
		doc := ParseLedger(mustRead(t, ledgerPath))
		results := RunGates(context.Background(), project, ledgerPath, doc, approved, true, 10*time.Second, nil)
		paths, err := writeTranscripts(work, doc, results, before, before)
		if err != nil || len(paths) != 1 {
			t.Fatalf("attempt %d: paths=%v err=%v", attempt, paths, err)
		}
		if want := filepath.Join(work, "references", "gates", "G1", "bbbbbbbbbbbb-a"+string(rune('0'+attempt))+".log"); paths[0] != want {
			t.Fatalf("attempt %d path = %s, want %s", attempt, paths[0], want)
		}
		body := mustRead(t, paths[0])
		for _, want := range []string{
			"COMMAND: " + check, "CWD: .", "EXIT: 0", "RESULT: pass", "CANDIDATE_BEFORE: " + before,
			"OUTPUT_SHA256: " + OutputDigest(results[0].Outcome.Output), "REDACTIONS: 1", "TRANSCRIPT-MARKER [REDACTED:github-token]",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("attempt %d transcript lacks %q:\n%s", attempt, want, body)
			}
		}
		if strings.Contains(strings.SplitN(body, "--- OUTPUT ---", 2)[1], token) {
			t.Fatal("the transcript output must not keep the credential")
		}
	}
}

func TestWriteTranscriptsRecordsReuse(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	work := filepath.Dir(ledgerPath)
	if err := os.WriteFile(filepath.Join(project, "src.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: echo OK\n  EXPECT: OK\n  INPUTS: src.txt\n  EVIDENCE: pending\n")
	approved := []ApprovedCommand{{Command: "echo OK"}}
	RunGates(context.Background(), project, ledgerPath, ParseLedger(mustRead(t, ledgerPath)), approved, false, 10*time.Second, nil)

	doc := ParseLedger(mustRead(t, ledgerPath))
	results := RunGates(context.Background(), project, ledgerPath, doc, approved, true, 10*time.Second, nil)
	paths, err := writeTranscripts(work, doc, results, "unavailable: test", "unavailable: test")
	if err != nil || len(paths) != 1 || !strings.HasSuffix(paths[0], "nodigest-reused.log") {
		t.Fatalf("paths=%v err=%v", paths, err)
	}
	if body := mustRead(t, paths[0]); !strings.Contains(body, "RESULT: reused") || !strings.Contains(body, "PRIOR_EVIDENCE: "+EvidenceMarker) {
		t.Fatalf("reuse transcript must name the prior evidence:\n%s", body)
	}
}

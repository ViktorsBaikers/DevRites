package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoveryAttemptsPersistLimitAcrossCalls(t *testing.T) {
	root := t.TempDir()
	slug := "recovery"
	writeReadinessFile(t, root, slug, "state.md", "- Phase: build\n")
	if err := os.WriteFile(filepath.Join(root, "ACTIVE"), []byte(slug+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cause := "CSS extractor mishandles same-line imports"

	for attempt := 1; attempt <= recoveryAttemptLimit; attempt++ {
		var stdout, stderr bytes.Buffer
		code := RecoveryAttempts(root, []string{"record", cause, "scanner exit 3"}, &stdout, &stderr)
		if attempt < recoveryAttemptLimit && code != 0 {
			t.Fatalf("attempt %d code=%d stderr=%q", attempt, code, stderr.String())
		}
		if attempt == recoveryAttemptLimit && code != 3 {
			t.Fatalf("third attempt code=%d stdout=%q stderr=%q, want exhausted", code, stdout.String(), stderr.String())
		}
	}

	// A fresh invocation reads the durable ledger and cannot receive three more
	// attempts merely because chat/agent context changed.
	var stderr bytes.Buffer
	if code := RecoveryAttempts(root, []string{"check", cause}, &bytes.Buffer{}, &stderr); code != 3 {
		t.Fatalf("restart check code=%d stderr=%q, want exhausted", code, stderr.String())
	}
	if code := RecoveryAttempts(root, []string{"record", cause, "fourth failure"}, &bytes.Buffer{}, &stderr); code != 3 {
		t.Fatalf("fourth record code=%d, want refused", code)
	}

	if code := RecoveryAttempts(root, []string{"clear", cause}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("clear code=%d", code)
	}
	if code := RecoveryAttempts(root, []string{"check", cause}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("check after green clear code=%d", code)
	}
}

func TestRecoveryAttemptsKeyByNormalizedRootCause(t *testing.T) {
	root := t.TempDir()
	slug := "recovery"
	writeReadinessFile(t, root, slug, "state.md", "- Phase: build\n")
	if got, want := recoveryFingerprint(" CSS   GAP "), recoveryFingerprint("css gap"); got != want {
		t.Fatalf("normalized fingerprints differ: %s != %s", got, want)
	}
	if recoveryFingerprint("css gap") == recoveryFingerprint("browser timeout") {
		t.Fatal("different root causes share a fingerprint")
	}

	if code := RecoveryAttempts(root, []string{"record", "css gap", "exit 3", slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("record code=%d", code)
	}
	var stdout bytes.Buffer
	if code := RecoveryAttempts(root, []string{"check", "browser timeout", slug}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("different cause check code=%d", code)
	}
	if !strings.Contains(stdout.String(), "0/3") {
		t.Fatalf("different cause inherited attempts: %q", stdout.String())
	}
}

func TestRecoveryAttemptsFailsClosedOnCorruptLedger(t *testing.T) {
	root := t.TempDir()
	slug := "recovery"
	writeReadinessFile(t, root, slug, "state.md", "- Phase: build\n")
	writeReadinessFile(t, root, slug, recoveryAttemptsFile, "{not-json}\n")
	var stderr bytes.Buffer
	if code := RecoveryAttempts(root, []string{"check", "css gap", slug}, &bytes.Buffer{}, &stderr); code != 2 {
		t.Fatalf("code=%d stderr=%q, want fail closed", code, stderr.String())
	}
}

func TestRecoveryRoutesCoverEveryClass(t *testing.T) {
	tests := []struct {
		scenario   string
		class      recoveryClass
		owner      recoveryOwner
		action     recoveryAction
		humanPause bool
	}{
		{"desired behavior is undecided", recoveryIntentGap, "human_clarify", "clarify_intent", true},
		{"acceptance omits an error outcome", recoverySpecGap, "clarify", "clarify_missing_decision", true},
		{"proof plan omits required provenance", recoveryPlanGap, "plan", "rite_plan_repair", false},
		{"stable product assertion is red", recoveryImplementationDefect, "wright", "repair_implementation_and_rerun_proof", false},
		{"declared CSS import defeats the scanner", recoveryProofToolDefect, "debug_recovery", "repair_proof_tool_and_rerun_original_proof", false},
		{"browser runner exhausts resources", recoveryEnvironmentDefect, "debug_recovery", "normalize_environment_and_run_discriminator", false},
		{"unrelated baseline lint already fails", recoveryPreexisting, "baseline", "record_baseline_and_fix_if_acceptance_blocked", false},
		{"observed behavior matches accepted authority", recoveryNotADefect, "caller", "record_authority_and_continue", false},
	}

	if len(recoveryRoutes) != len(tests) {
		t.Fatalf("route count=%d, want %d", len(recoveryRoutes), len(tests))
	}
	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			route, ok := recoveryRouteFor(test.class)
			if !ok {
				t.Fatalf("missing route for %q", test.class)
			}
			if route.Schema != recoveryRouteSchema || route.Owner != test.owner ||
				route.Action != test.action || route.HumanPause != test.humanPause {
				t.Fatalf("route=%+v", route)
			}
		})
	}
}

func TestRecoveryRouteOutputIsStableJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := RecoveryAttempts("", []string{"route", "proof_tool_defect"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	const want = `{"schema":"recovery-route/v1","class":"proof_tool_defect","owner":"debug_recovery","action":"repair_proof_tool_and_rerun_original_proof","humanPause":false}` + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout=%q, want %q", stdout.String(), want)
	}
}

func TestRecoveryReplayDEC031RoutesWithoutAuthorization(t *testing.T) {
	proofRoute, ok := recoveryRouteFor(recoveryProofToolDefect)
	if !ok || proofRoute.HumanPause || proofRoute.Action != "repair_proof_tool_and_rerun_original_proof" {
		t.Fatalf("DEC-031 proof route=%+v, found=%v", proofRoute, ok)
	}
	planRoute, ok := recoveryRouteFor(recoveryPlanGap)
	if !ok || planRoute.HumanPause || planRoute.Action != "rite_plan_repair" {
		t.Fatalf("DEC-031 provenance route=%+v, found=%v", planRoute, ok)
	}
}

func TestRecoveryReplayDEC032SeparatesScannerAndEnvironment(t *testing.T) {
	scannerCause := "CSS extractor mishandles same-line imports"
	environmentCause := "browser target timeout ERR_INSUFFICIENT_RESOURCES"
	if recoveryFingerprint(scannerCause) == recoveryFingerprint(environmentCause) {
		t.Fatal("DEC-032 causal fingerprints must not share an attempt budget")
	}
	for _, class := range []recoveryClass{recoveryProofToolDefect, recoveryEnvironmentDefect} {
		route, ok := recoveryRouteFor(class)
		if !ok || route.HumanPause || route.Owner != "debug_recovery" {
			t.Fatalf("DEC-032 route for %s=%+v, found=%v", class, route, ok)
		}
	}
}

func TestRecoveryAttemptsPersistsClassAndReadsLegacyJSONL(t *testing.T) {
	root := t.TempDir()
	slug := "recovery"
	cause := "declared CSS import exits 3"
	writeReadinessFile(t, root, slug, "state.md", "- Phase: build\n")
	legacy := recoveryAttempt{
		Fingerprint: recoveryFingerprint(cause),
		RootCause:   normalizeRecoveryCause(cause),
		Attempt:     1,
		Status:      "failed",
		Failure:     "legacy scanner failure",
		At:          "2026-07-23T00:00:00Z",
	}
	legacyJSON, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "work", slug, recoveryAttemptsFile)
	writeReadinessFile(t, root, slug, recoveryAttemptsFile, string(legacyJSON)+"\n")

	var stdout, stderr bytes.Buffer
	args := []string{"record", "--class", "proof_tool_defect", cause, "scanner still exits 3", slug}
	if code := RecoveryAttempts(root, args, &stdout, &stderr); code != 0 {
		t.Fatalf("record code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	entries, err := readRecoveryAttempts(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Class != "" || entries[1].Class != recoveryProofToolDefect {
		t.Fatalf("entries=%+v", entries)
	}
	if entries[1].Attempt != 2 {
		t.Fatalf("typed attempt=%d, want legacy budget continuation at 2", entries[1].Attempt)
	}

	stdout.Reset()
	stderr.Reset()
	clearArgs := []string{"clear", "--class", "proof_tool_defect", cause, slug}
	if code := RecoveryAttempts(root, clearArgs, &stdout, &stderr); code != 0 {
		t.Fatalf("clear code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	entries, err = readRecoveryAttempts(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || entries[2].Class != recoveryProofToolDefect || entries[2].Status != "cleared" {
		t.Fatalf("typed clear=%+v", entries)
	}
}

func TestRecoveryAttemptsRejectsUnknownClass(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"record", "--class", "retry_authorization", "cause", "failure", "slug"}
	if code := RecoveryAttempts(t.TempDir(), args, &bytes.Buffer{}, &stderr); code != 2 {
		t.Fatalf("code=%d stderr=%q, want usage failure", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown class") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

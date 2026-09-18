package acceptance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- parsing -------------------------------------------------------------

func TestParseLedgerRunnableAndManualGates(t *testing.T) {
	doc := ParseLedger(`# Gates

- [ ] AC-001: binary builds clean
  CHECK: go build ./...
  EXPECT: ok
  CWD: engine
  EVIDENCE: pending

- [x] AC-002: spec reviewed by a human
  EVIDENCE: reviewed 2024-06-01 by ops
`)
	if len(doc.Errors) != 0 {
		t.Fatalf("errors=%v", doc.Errors)
	}
	if len(doc.Gates) != 2 {
		t.Fatalf("gates=%d", len(doc.Gates))
	}
	runnable, manual := doc.Gates[0], doc.Gates[1]
	if runnable.ID != "AC-001" || runnable.Check != "go build ./..." || runnable.Expect != "ok" || runnable.Cwd != "engine" {
		t.Fatalf("runnable gate misparsed: %+v", runnable)
	}
	if runnable.Checked || runnable.Evidence != "pending" {
		t.Fatalf("runnable state misparsed: %+v", runnable)
	}
	if manual.ID != "AC-002" || !manual.Checked || manual.Evidence != "reviewed 2024-06-01 by ops" {
		t.Fatalf("manual gate misparsed: %+v", manual)
	}
}

func TestParseLedgerIgnoresFencedExamples(t *testing.T) {
	doc := ParseLedger(`# Gates

Example ledger, do not parse:

` + "```markdown" + `
- [ ] FAKE-1: fenced gate
  EVIDENCE: fenced evidence
` + "```" + `

- [x] REAL-1: real gate
  EVIDENCE: attested

` + "```" + `
  EVIDENCE: fenced attribute decoy
` + "```" + `
`)
	if len(doc.Errors) != 0 {
		t.Fatalf("errors=%v", doc.Errors)
	}
	if len(doc.Gates) != 1 || doc.Gates[0].ID != "REAL-1" {
		t.Fatalf("fenced gate leaked into parse: %+v", doc.Gates)
	}
	if doc.Gates[0].Evidence != "attested" {
		t.Fatalf("fenced attribute decoy overwrote evidence: %q", doc.Gates[0].Evidence)
	}
}

func TestParseLedgerErrors(t *testing.T) {
	cases := map[string]string{
		"empty":              "# Gates\n",
		"partial-check":      "- [ ] G1: x\n  CHECK: cmd\n",
		"partial-expect":     "- [ ] G1: x\n  EXPECT: out\n",
		"unindented-attr":    "- [ ] G1: x\nCHECK: cmd\n  EXPECT: out\n",
		"orphan-attr":        "  EVIDENCE: nowhere\n- [ ] G1: x\n",
		"duplicate-id":       "- [ ] G1: a\n- [ ] G1: b\n",
		"long-id":            "- [ ] " + strings.Repeat("G", 65) + ": x\n",
		"indented-abandon":   "- [ ] G1: x\n  ABANDON: G1 nope\n",
		"abandon-unknown":    "- [ ] G1: x\nABANDON: G2 gone\n",
		"abandon-no-reason":  "- [ ] G1: x\nABANDON: G1\n",
		"absolute-cwd":       "- [ ] G1: x\n  CHECK: c\n  EXPECT: o\n  CWD: /etc\n",
		"windows-abs-cwd":    "- [ ] G1: x\n  CHECK: c\n  EXPECT: o\n  CWD: C:\\tmp\n",
		"traversal-cwd":      "- [ ] G1: x\n  CHECK: c\n  EXPECT: o\n  CWD: a/../b\n",
		"bad-expect-pattern": "- [ ] G1: x\n  CHECK: c\n  EXPECT: /[/\n",
	}
	for name, text := range cases {
		doc := ParseLedger(text)
		if len(doc.Errors) == 0 {
			t.Errorf("%s: expected parse error, got none", name)
		}
	}
}

func TestParseLedgerAbandonRecordsReason(t *testing.T) {
	doc := ParseLedger("- [ ] G1: x\n- [ ] G2: y\n\nABANDON: G1 upstream API removed\n")
	if len(doc.Errors) != 0 {
		t.Fatalf("errors=%v", doc.Errors)
	}
	if doc.Abandoned["G1"] != "upstream API removed" {
		t.Fatalf("abandoned=%v", doc.Abandoned)
	}
}

func TestParseLedgerPreservesCRLF(t *testing.T) {
	doc := ParseLedger("- [ ] G1: x\r\n  EVIDENCE: pending\r\n")
	if doc.Newline() != "\r\n" {
		t.Fatalf("newline=%q", doc.Newline())
	}
}

func TestCompileExpectForms(t *testing.T) {
	sub, err := CompileExpect("all tests pass")
	if err != nil || sub.Kind != "substring" || !sub.Matches("ok all tests pass ok") {
		t.Fatalf("substring: %+v %v", sub, err)
	}
	re, err := CompileExpect(`/PASS \d+ files/`)
	if err != nil || re.Kind != "regex" || !re.Matches("PASS 3 files") {
		t.Fatalf("regex: %+v %v", re, err)
	}
	if re.PathLike {
		t.Fatalf("non-path regex misflagged: %+v", re)
	}
	pathRe, err := CompileExpect(`/tmp/out\.txt/`)
	if err != nil || !pathRe.PathLike {
		t.Fatalf("path-like regex not flagged: %+v %v", pathRe, err)
	}
	ci, err := CompileExpect(`/success/i`)
	if err != nil || !ci.Matches("SUCCESS") {
		t.Fatalf("case-insensitive: %v", err)
	}
	if _, err := CompileExpect(""); err == nil {
		t.Fatal("empty EXPECT must error")
	}
	// A /…/ form with non-flag trailing characters is a substring, not a
	// regex — `/a/b/c` stays a usable literal path expectation.
	subLiteral, err := CompileExpect(`/a/b/c`)
	if err != nil || subLiteral.Kind != "substring" {
		t.Fatalf("slash literal: %+v %v", subLiteral, err)
	}
}

// --- state reduction ------------------------------------------------------

func runnableGate(id string, checked bool, evidence string) *Gate {
	return &Gate{ID: id, Title: "outcome", Checked: checked, Check: "probe", Expect: "ok", Evidence: evidence}
}

func TestGateStateManual(t *testing.T) {
	doc := &Document{Abandoned: map[string]string{}}
	manual := &Gate{ID: "M1", Evidence: "pending"}
	if got := GateState(manual, doc); got != StateUnmet {
		t.Fatalf("unchecked manual=%s", got)
	}
	manual.Checked = true
	if got := GateState(manual, doc); got != StateUnmet {
		t.Fatalf("pending-evidence manual=%s", got)
	}
	manual.Evidence = "human verified on staging"
	if got := GateState(manual, doc); got != StateMet {
		t.Fatalf("attested manual=%s", got)
	}
}

func TestGateStateRunnable(t *testing.T) {
	doc := &Document{Abandoned: map[string]string{}}
	gate := runnableGate("R1", false, "pending")
	if got := GateState(gate, doc); got != StateUnmet {
		t.Fatalf("unchecked=%s", got)
	}
	gate.Checked = true
	gate.Evidence = "looks good to me"
	if got := GateState(gate, doc); got != StateStaleUnmet {
		t.Fatalf("prose evidence on runnable gate must be stale, got %s", got)
	}
	gate.Evidence = FormatEvidence(gate, OutputDigest("out"), 3, "2024-01-01T00:00:00Z")
	if got := GateState(gate, doc); got != StateMet {
		t.Fatalf("bound evidence=%s", got)
	}
}

func TestGateStateDefinitionDriftStale(t *testing.T) {
	doc := &Document{Abandoned: map[string]string{}}
	gate := runnableGate("R1", true, "")
	gate.Evidence = FormatEvidence(gate, OutputDigest("out"), 3, "t")
	for _, mutate := range []func(*Gate){
		func(g *Gate) { g.Check = "probe --verbose" },
		func(g *Gate) { g.Expect = "ok2" },
		func(g *Gate) { g.Cwd = "sub" },
	} {
		edited := *gate
		mutate(&edited)
		if got := GateState(&edited, doc); got != StateStaleUnmet {
			t.Fatalf("edited definition must be stale, got %s", got)
		}
	}
}

func TestGateStateAbandonedWins(t *testing.T) {
	doc := &Document{Abandoned: map[string]string{"R1": "vendor removed"}}
	gate := runnableGate("R1", true, "")
	gate.Evidence = FormatEvidence(gate, OutputDigest("o"), 1, "t")
	if got := GateState(gate, doc); got != StateAbandoned {
		t.Fatalf("abandoned=%s", got)
	}
}

func TestReduceCountsAndIds(t *testing.T) {
	doc := &Document{Abandoned: map[string]string{"G3": "handoff"}}
	met := runnableGate("G1", true, "")
	met.Evidence = FormatEvidence(met, OutputDigest("o"), 1, "t")
	unmet := runnableGate("G2", false, "pending")
	stale := runnableGate("G4", true, "stale prose")
	doc.Gates = []*Gate{met, unmet, {ID: "G3", Title: "gone"}, stale}

	red := doc.Reduce([]ApprovedCommand{{Command: "probe", Cwd: ""}})
	if red.Total != 4 || red.Met != 1 || red.Unmet != 2 || red.Abandoned != 1 || red.Stale != 1 {
		t.Fatalf("reduction=%+v", red)
	}
	if red.AllMet() || !red.HandoffRequired() {
		t.Fatalf("verdict wrong: %+v", red)
	}
	// G2 and G4 are unmet runnable gates whose CHECK "probe" IS approved.
	if len(red.UnapprIDs) != 0 {
		t.Fatalf("unapproved=%v", red.UnapprIDs)
	}
	red = doc.Reduce([]ApprovedCommand{{Command: "other", Cwd: ""}})
	if len(red.UnapprIDs) != 2 {
		t.Fatalf("unapproved ids=%v", red.UnapprIDs)
	}
	// A nil approved surface (unreadable plan) reports no unapproved diagnostic.
	red = doc.Reduce(nil)
	if len(red.UnapprIDs) != 0 {
		t.Fatalf("nil approved must not flag: %v", red.UnapprIDs)
	}
}

func TestReduceAllMet(t *testing.T) {
	doc := &Document{Abandoned: map[string]string{}}
	gate := runnableGate("G1", true, "")
	gate.Evidence = FormatEvidence(gate, OutputDigest("o"), 1, "t")
	doc.Gates = []*Gate{gate}
	if !doc.Reduce(nil).AllMet() {
		t.Fatal("single met gate must reduce all-met")
	}
}

// --- approval -------------------------------------------------------------

const testPlanPreflight = `# Test plan

## Build-entry preflight

| Command | Cwd | Purpose |
| --- | --- | --- |
| go test ./internal/acceptance | engine | package tests |
| ./scripts/scan.sh \| tail -1 | repository | escaped pipe survives |
| make build | . | root convention |

## Consumptive actions

| Command | Cwd |
| --- | --- |
| rm -rf dist | . |
`

func TestApprovedCommandsPreflightOnly(t *testing.T) {
	approved := ApprovedCommands([]byte(testPlanPreflight))
	if len(approved) != 3 {
		t.Fatalf("approved=%+v", approved)
	}
	if !Approved("go test ./internal/acceptance", "engine", approved) {
		t.Fatal("exact pair must approve")
	}
	if Approved("go test ./internal/acceptance", ".", approved) {
		t.Fatal("same command, different cwd must not approve")
	}
	if !Approved(" go test ./internal/acceptance ", "engine/", approved) {
		t.Fatal("whitespace/slash normalization must approve the same pair")
	}
	if Approved("rm -rf dist", ".", approved) {
		t.Fatal("non-preflight section must not approve")
	}
	if !Approved("./scripts/scan.sh | tail -1", "root", approved) {
		t.Fatal("escaped-pipe command and root cwd must approve")
	}
}

func TestApprovedCommandsNoSection(t *testing.T) {
	if approved := ApprovedCommands([]byte("# Plan\n\nno table\n")); len(approved) != 0 {
		t.Fatalf("approved=%+v", approved)
	}
	if approved := ApprovedCommands(nil); approved != nil {
		t.Fatalf("nil plan must approve nothing: %+v", approved)
	}
}

func TestNormalizeApprovedCwd(t *testing.T) {
	for _, root := range []string{"", ".", "./", "repository", "root", "repo", "n/a", "-", "`.`"} {
		if got := normalizeApprovedCwd(root); got != "" {
			t.Fatalf("root convention %q -> %q", root, got)
		}
	}
	if got := normalizeApprovedCwd("engine/"); got != "engine" {
		t.Fatalf("engine/ -> %q", got)
	}
}

// --- execution ------------------------------------------------------------

func writeWorkspace(t *testing.T) (project, ledgerPath string) {
	t.Helper()
	project = t.TempDir()
	root := filepath.Join(project, ".devrites")
	work := filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	return project, filepath.Join(work, GatesFile)
}

func writeLedger(t *testing.T, ledgerPath, body string) {
	t.Helper()
	if err := os.WriteFile(ledgerPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunGatesExecutesAndBindsEvidence(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	writeLedger(t, ledgerPath, "- [ ] G1: probe prints marker\n  CHECK: echo RUN-MARKER\n  EXPECT: RUN-MARKER\n  EVIDENCE: pending\n")
	doc := ParseLedger(mustRead(t, ledgerPath))
	approved := []ApprovedCommand{{Command: "echo RUN-MARKER", Cwd: ""}}

	results := RunGates(context.Background(), project, ledgerPath, doc, approved, false, 10*time.Second)
	if len(results) != 1 || !results[0].Ran || !results[0].Passed || results[0].WriteErr != nil {
		t.Fatalf("results=%+v", results)
	}
	doc = ParseLedger(mustRead(t, ledgerPath))
	if len(doc.Errors) != 0 {
		t.Fatalf("rewritten ledger malformed: %v", doc.Errors)
	}
	gate := doc.Gates[0]
	if !gate.Checked || !strings.HasPrefix(gate.Evidence, EvidenceMarker) {
		t.Fatalf("ledger not updated: %+v", gate)
	}
	if GateState(gate, doc) != StateMet {
		t.Fatalf("gate not met after run: %s", GateState(gate, doc))
	}
	// Already met: skipped without reverify.
	results = RunGates(context.Background(), project, ledgerPath, doc, approved, false, 10*time.Second)
	if !results[0].Passed || results[0].Ran {
		t.Fatalf("met gate must skip: %+v", results[0])
	}
	// Reverify re-executes it.
	results = RunGates(context.Background(), project, ledgerPath, doc, approved, true, 10*time.Second)
	if !results[0].Ran || !results[0].Passed {
		t.Fatalf("reverify must re-run: %+v", results[0])
	}
}

func TestRunGatesFailureClearsBox(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	writeLedger(t, ledgerPath, "- [x] G1: prior pass\n  CHECK: echo nothing\n  EXPECT: NEVER-PRESENT\n  EVIDENCE: bogus\n")
	doc := ParseLedger(mustRead(t, ledgerPath))
	approved := []ApprovedCommand{{Command: "echo nothing", Cwd: ""}}

	results := RunGates(context.Background(), project, ledgerPath, doc, approved, true, 10*time.Second)
	if !results[0].Ran || results[0].Passed {
		t.Fatalf("results=%+v", results)
	}
	doc = ParseLedger(mustRead(t, ledgerPath))
	if doc.Gates[0].Checked || doc.Gates[0].Evidence != "pending" {
		t.Fatalf("failed gate must uncheck and reset evidence: %+v", doc.Gates[0])
	}
}

func TestRunGatesSkipsUnapprovedManualAbandoned(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	writeLedger(t, ledgerPath,
		"- [ ] G1: unvetted\n  CHECK: echo hi\n  EXPECT: hi\n  EVIDENCE: pending\n"+
			"- [ ] G2: judged\n  EVIDENCE: pending\n"+
			"- [ ] G3: gone\n  CHECK: echo hi\n  EXPECT: hi\n  EVIDENCE: pending\n"+
			"ABANDON: G3 removed upstream\n")
	doc := ParseLedger(mustRead(t, ledgerPath))
	results := RunGates(context.Background(), project, ledgerPath, doc, []ApprovedCommand{}, false, 10*time.Second)
	if len(results) != 3 {
		t.Fatalf("results=%+v", results)
	}
	for _, r := range results {
		if r.Ran {
			t.Fatalf("gate %s must not execute: %+v", r.ID, r)
		}
	}
	// Ledger untouched: no evidence was written for skipped gates.
	doc = ParseLedger(mustRead(t, ledgerPath))
	for _, gate := range doc.Gates {
		if gate.Checked {
			t.Fatalf("skipped gate %s mutated", gate.ID)
		}
	}
}

func TestWriteResultDiscardsMovedOracle(t *testing.T) {
	_, ledgerPath := writeWorkspace(t)
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: echo hi\n  EXPECT: hi\n  EVIDENCE: pending\n")
	doc := ParseLedger(mustRead(t, ledgerPath))
	ran := doc.Gates[0]
	// The oracle moves between execution and writeback.
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: echo hi --verbose\n  EXPECT: hi\n  EVIDENCE: pending\n")
	if err := writeResult(ledgerPath, ran, "evidence", true); !errors.Is(err, ErrOracleMoved) {
		t.Fatalf("err=%v", err)
	}
	// The ledger still shows the new, unchecked definition.
	doc = ParseLedger(mustRead(t, ledgerPath))
	if doc.Gates[0].Checked || doc.Gates[0].Check != "echo hi --verbose" {
		t.Fatalf("ledger clobbered: %+v", doc.Gates[0])
	}
}

func TestWriteResultIgnoresFencedEvidenceDecoy(t *testing.T) {
	_, ledgerPath := writeWorkspace(t)
	writeLedger(t, ledgerPath,
		"- [ ] G1: probe\n  CHECK: echo hi\n  EXPECT: hi\n\n"+
			"```\n  EVIDENCE: fenced decoy\n```\n")
	doc := ParseLedger(mustRead(t, ledgerPath))
	if err := writeResult(ledgerPath, doc.Gates[0], "automatic-evidence=v1; def=x; exit=0; expect=matched; output-sha256=y; output-bytes=1; at=t", true); err != nil {
		t.Fatal(err)
	}
	text := mustRead(t, ledgerPath)
	if strings.Contains(text, "fenced decoy") == false {
		t.Fatalf("fenced decoy must survive:\n%s", text)
	}
	doc = ParseLedger(text)
	if doc.Gates[0].Evidence == "fenced decoy" || !strings.HasPrefix(doc.Gates[0].Evidence, EvidenceMarker) {
		t.Fatalf("evidence misrouted: %q", doc.Gates[0].Evidence)
	}
}

func TestReadLedgerFileBounds(t *testing.T) {
	_, ledgerPath := writeWorkspace(t)
	if _, err := ReadLedgerFile(ledgerPath); err == nil {
		t.Fatal("missing ledger must error")
	}
	dir := filepath.Join(filepath.Dir(ledgerPath), "notafile")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadLedgerFile(dir); err == nil {
		t.Fatal("non-regular file must error")
	}
}

// --- lint -----------------------------------------------------------------

func TestLintFindings(t *testing.T) {
	doc := ParseLedger(`- [ ] G1: echo done
  CHECK: echo done
  EXPECT: pass

- [ ] G2: Improve coverage by 40 percent
  EVIDENCE: pending

- [ ] G3: real probe
  CHECK: ./probe.sh
  EXPECT: /tmp/out\.txt/

- [ ] G4: unvetted
  CHECK: ./other.sh
  EXPECT: marker
`)
	if len(doc.Errors) != 0 {
		t.Fatalf("errors=%v", doc.Errors)
	}
	findings := LintLedger(doc, []ApprovedCommand{{Command: "./probe.sh", Cwd: ""}})
	rules := map[string]string{}
	for _, f := range findings {
		rules[f.Gate+":"+f.Rule] = f.Level
	}
	for _, want := range []string{
		"G1:tautological-check", "G1:weak-expect",
		"G2:manual-gate", "G2:unmeasured-number", "G2:activity-not-outcome",
		"G3:path-read-as-regex",
		"G4:unapproved-check",
	} {
		if _, ok := rules[want]; !ok {
			t.Errorf("missing finding %s in %+v", want, rules)
		}
	}
	if rules["G4:unapproved-check"] != "error" {
		t.Fatalf("unapproved must be an error: %+v", rules)
	}
	// 3 of 4 live gates runnable → no mostly-manual warning.
	if _, ok := rules[":mostly-manual"]; ok {
		t.Fatalf("unexpected mostly-manual: %+v", rules)
	}
}

func TestLintMostlyManualAndCounts(t *testing.T) {
	doc := ParseLedger("- [ ] G1: a\n  CHECK: c\n  EXPECT: o\n- [ ] G2: b\n- [ ] G3: c\n")
	findings := LintLedger(doc, nil)
	var mostlyManual bool
	for _, f := range findings {
		if f.Rule == "mostly-manual" {
			mostlyManual = true
		}
	}
	if !mostlyManual {
		t.Fatalf("findings=%+v", findings)
	}
	errs, warns := LintCounts(findings)
	if errs != 0 || warns == 0 {
		t.Fatalf("counts=%d/%d", errs, warns)
	}
}

// --- scaffold -------------------------------------------------------------

func TestScaffoldLedger(t *testing.T) {
	empty := ScaffoldLedger("feat", nil)
	doc := ParseLedger(empty)
	if len(doc.Errors) != 0 || len(doc.Gates) != 1 || doc.Gates[0].ID != "G1" {
		t.Fatalf("placeholder scaffold malformed: %v %+v", doc.Errors, doc.Gates)
	}
	spec := []byte("# Spec\n\n- AC-002 first\n- AC-001 second\n- AC-002 repeated\n")
	scaffolded := ScaffoldLedger("feat", spec)
	doc = ParseLedger(scaffolded)
	if len(doc.Errors) != 0 || len(doc.Gates) != 2 || doc.Gates[0].ID != "AC-002" || doc.Gates[1].ID != "AC-001" {
		t.Fatalf("scaffolded gates: %v %+v", doc.Errors, doc.Gates)
	}
}

// --- CLI ------------------------------------------------------------------

func TestRunCLIStatusAndScaffold(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	work := filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr strings.Builder

	if code := Run(root, []string{"bogus"}, &stdout, &stderr); code != ExitUsage {
		t.Fatalf("unknown sub=%d", code)
	}
	if code := Run(root, []string{"status", "feat"}, &stdout, &stderr); code != ExitBlocked {
		t.Fatalf("missing ledger status=%d", code)
	}
	if code := Run(root, []string{"scaffold", "feat"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("scaffold=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "created:") {
		t.Fatalf("scaffold stdout=%q", stdout.String())
	}
	// Scaffold never clobbers.
	if code := Run(root, []string{"scaffold", "feat"}, &stdout, &stderr); code != ExitBlocked {
		t.Fatalf("second scaffold=%d", code)
	}
	// The scaffolded ledger parses and reports all-unmet.
	stdout.Reset()
	if code := Run(root, []string{"status", "feat"}, &stdout, &stderr); code != ExitBlocked {
		t.Fatalf("status=%d out=%q", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "result: not-met") {
		t.Fatalf("status stdout=%q", stdout.String())
	}
}

func TestRunCLIAttestAbandon(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	work := filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLedger(t, filepath.Join(work, GatesFile),
		"- [ ] M1: human judged\n  EVIDENCE: pending\n- [ ] M2: other\n  EVIDENCE: pending\n")
	var stdout, stderr strings.Builder

	if code := Run(root, []string{"attest", "feat", "M1", "verified", "on staging"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("attest=%d stderr=%q", code, stderr.String())
	}
	if code := Run(root, []string{"attest", "feat", "M9", "nope"}, &stdout, &stderr); code != ExitBlocked {
		t.Fatalf("unknown gate attest=%d", code)
	}
	if code := Run(root, []string{"abandon", "feat", "M2", "vendor gone"}, &stdout, &stderr); code != ExitOK {
		t.Fatalf("abandon=%d stderr=%q", code, stderr.String())
	}
	// Abandoned gates refuse attestation.
	if code := Run(root, []string{"attest", "feat", "M2", "too late"}, &stdout, &stderr); code != ExitBlocked {
		t.Fatalf("abandoned attest=%d", code)
	}
	stdout.Reset()
	if code := Run(root, []string{"status", "feat"}, &stdout, &stderr); code != ExitBlocked {
		t.Fatalf("status=%d", code)
	}
	if !strings.Contains(stdout.String(), "result: handoff") || !strings.Contains(stdout.String(), "handoff-id: M2") {
		t.Fatalf("status stdout=%q", stdout.String())
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

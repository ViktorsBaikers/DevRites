package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const editState = `# State

## Cursor
| Key | Value |
| --- | --- |
| phase | plan |
| status | running |
| next_action | /rite-vet |
| return_phase | build |
| schema | 4 |
`

func editWorkspace(t *testing.T) (root, dir string) {
	t.Helper()
	root = unionRoot(t)
	dir = filepath.Join(root, "work", "feat")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"state.md": editState,
		"drift.md": "# Drift\n\n## DRIFT-001 first\n\nKept.\n\n```markdown\n## DRIFT-002 example only\n```\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, dir
}

func readEdit(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestStateSetWritesFieldsAllOrNothing(t *testing.T) {
	root, dir := editWorkspace(t)
	var out, errOut bytes.Buffer
	if code := RunStateSet(root, []string{"feat", "phase=vet", "next_action=/rite-build SLICE-001", "active_slice=SLICE-001"}, &out, &errOut); code != 0 {
		t.Fatalf("set exit %d: %s", code, errOut.String())
	}
	got := readEdit(t, dir, "state.md")
	for _, want := range []string{"| phase | vet |", "| next_action | /rite-build SLICE-001 |", "| active_slice | SLICE-001 |", "| schema | 4 |"} {
		if !strings.Contains(got, want) {
			t.Fatalf("state.md lacks %q:\n%s", want, got)
		}
	}

	before := readEdit(t, dir, "state.md")
	for _, args := range [][]string{
		{"feat", "status=blocked", "phase=nowhere"},
		{"feat", "schema=5"},
		{"feat", "next_action=a | b"},
		{"feat", "--unset", "phase"},
	} {
		errOut.Reset()
		if code := RunStateSet(root, args, &out, &errOut); code != 3 {
			t.Fatalf("%v: exit %d, want 3 (%s)", args, code, errOut.String())
		}
		if readEdit(t, dir, "state.md") != before {
			t.Fatalf("%v changed state.md despite refusal", args)
		}
	}
	if code := RunStateSet(root, []string{"feat", "Phase=vet"}, &out, &errOut); code != 2 {
		t.Fatalf("non-snake key: exit %d, want 2", code)
	}
}

func TestStateSetNeverOverwritesReturnCursor(t *testing.T) {
	root, dir := editWorkspace(t)
	var out, errOut bytes.Buffer
	if code := RunStateSet(root, []string{"feat", "return_phase=prove"}, &out, &errOut); code != 3 {
		t.Fatalf("overwrite exit %d, want 3", code)
	}
	if code := RunStateSet(root, []string{"feat", "return_phase=build"}, &out, &errOut); code != 0 {
		t.Fatalf("same-value exit %d: %s", code, errOut.String())
	}
	if code := RunStateSet(root, []string{"feat", "--unset", "return_phase", "phase=build"}, &out, &errOut); code != 0 {
		t.Fatalf("restore exit %d: %s", code, errOut.String())
	}
	if got := readEdit(t, dir, "state.md"); strings.Contains(got, "return_phase") || !strings.Contains(got, "| phase | build |") {
		t.Fatalf("restore did not remove the return row:\n%s", got)
	}
}

func TestStateSetAwaitingHumanNamesAFKAutoPicks(t *testing.T) {
	root, dir := editWorkspace(t)
	questions := "# Questions\n\n## q-2026-10-19-001\nstatus: open\ngate: validating\nproposed: Approve saved plan\n\n" +
		"## q-2026-10-19-002\nstatus: open\ngate: blocking\nproposed: Pin the driver\n\n" +
		"## q-2026-10-19-003\nstatus: answered\ngate: validating\nproposed: Done\n"
	if err := os.WriteFile(filepath.Join(dir, "questions.md"), []byte(questions), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := RunStateSet(root, []string{"feat", "status=awaiting_human"}, &out, &errOut); code != 0 || strings.Contains(out.String(), "advisory") {
		t.Fatalf("no sentinel: exit %d, out %q", code, out.String())
	}
	if err := os.WriteFile(filepath.Join(root, "AFK"), []byte("max_slices: 3\nallow_gates: [advisory, validating]  # auto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := RunStateSet(root, []string{"feat", "status=awaiting_human"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "auto-picks q-2026-10-19-001") || strings.Contains(got, "Approve saved plan") || strings.Contains(got, "q-2026-10-19-002") || strings.Contains(got, "q-2026-10-19-003") {
		t.Fatalf("advisory must name only the open allowed-gate question:\n%s", got)
	}
}

func TestStateAppendAddsSectionAndRefusesDuplicates(t *testing.T) {
	root, dir := editWorkspace(t)
	var out, errOut bytes.Buffer
	body := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(body, []byte("Fingerprint: x\n### Detail\nok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A heading that only appears inside a fenced example is not a duplicate.
	if code := RunStateAppend(root, []string{"feat", "drift.md", "--heading", "DRIFT-002 example only", "--body-file", body}, nil, &out, &errOut); code != 0 {
		t.Fatalf("append exit %d: %s", code, errOut.String())
	}
	got := readEdit(t, dir, "drift.md")
	if !strings.HasPrefix(got, "# Drift\n\n## DRIFT-001 first\n\nKept.\n") || !strings.HasSuffix(got, "\n\n## DRIFT-002 example only\n\nFingerprint: x\n### Detail\nok\n") {
		t.Fatalf("unexpected drift.md:\n%s", got)
	}

	before := got
	for _, args := range [][]string{
		{"feat", "drift.md", "--heading", "DRIFT-001 first", "--body-file", body},
		{"feat", "drift.md", "--heading", "DRIFT-001 — reused id, new title", "--body-file", body},
		{"feat", "drift.md", "--heading", "## DRIFT-009", "--body-file", body},
		{"feat", "state.md", "--heading", "x", "--body-file", body},
		{"feat", "spec.md", "--heading", "x", "--body-file", body},
	} {
		errOut.Reset()
		if code := RunStateAppend(root, args, nil, &out, &errOut); code != 3 {
			t.Fatalf("%v: exit %d, want 3 (%s)", args, code, errOut.String())
		}
	}
	for _, smuggled := range []string{"text\n## smuggled\n", "text\n##\tsmuggled\n", "text\n##\n", "open fence\n```\n"} {
		if code := RunStateAppend(root, []string{"feat", "drift.md", "--heading", "DRIFT-003", "--body-file", "-"}, strings.NewReader(smuggled), &out, &errOut); code != 3 {
			t.Fatalf("body %q: exit %d, want 3", smuggled, code)
		}
	}
	if readEdit(t, dir, "drift.md") != before {
		t.Fatal("a refused append changed drift.md")
	}

	if code := RunStateAppend(root, []string{"feat", "evidence.md", "--heading", "EV-001", "--body-file", "-"}, strings.NewReader("ran it\n"), &out, &errOut); code != 0 {
		t.Fatalf("new ledger exit %d: %s", code, errOut.String())
	}
	if got := readEdit(t, dir, "evidence.md"); got != "## EV-001\n\nran it\n" {
		t.Fatalf("unexpected evidence.md: %q", got)
	}
}

func TestStateAppendBlocksCrossingTheMaterialBudgetBand(t *testing.T) {
	root, dir := editWorkspace(t)
	var out, errOut bytes.Buffer
	// drift.md has a 160-line budget, so 240 lines is the material band.
	filler := strings.Repeat("round note\n", 230)
	if code := RunStateAppend(root, []string{"feat", "drift.md", "--heading", "DRIFT-003 rounds", "--body-file", "-"}, strings.NewReader(filler), &out, &errOut); code != 3 {
		t.Fatalf("crossing append exit %d, want 3 (%s)", code, errOut.String())
	}
	if msg := errOut.String(); !strings.Contains(msg, "history/") || !strings.Contains(msg, "Budget override: <=N lines; <reason>") {
		t.Fatalf("block must name the remedies: %s", msg)
	}
	if strings.Contains(readEdit(t, dir, "drift.md"), "DRIFT-003") {
		t.Fatal("a blocked append changed drift.md")
	}

	// A reason-only override is not a bound; one with N within 2x is.
	for _, c := range []struct {
		override string
		want     int
	}{
		{"Budget override: reslice reviews\n", 3},
		{"Budget override: <=300 lines; reslice reviews\n", 0},
	} {
		errOut.Reset()
		if code := RunStateAppend(root, []string{"feat", "drift.md", "--heading", "DRIFT-003 rounds", "--body-file", "-"}, strings.NewReader(c.override+filler), &out, &errOut); code != c.want {
			t.Fatalf("%q: exit %d, want %d (%s)", c.override, code, c.want, errOut.String())
		}
	}

	// Proof ledgers scale with proof volume and stay advisory.
	if code := RunStateAppend(root, []string{"feat", "evidence.md", "--heading", "EV-001", "--body-file", "-"}, strings.NewReader(strings.Repeat("ran it\n", 500)), &out, &errOut); code != 0 {
		t.Fatalf("evidence.md exit %d: %s", code, errOut.String())
	}
}

func TestStateAppendLintsNewQuestionEntries(t *testing.T) {
	const recommended = "status: open\nslice: SLICE-002\ngate: validating\nquestion: Which cache key wins?\noptions: |\n  1. Per user (Recommended) — logic: isolates tenants\n  2. Global — infra: one entry\n  3. Something else — describe it\nproposed: Per user\nraised_at: 2026-10-19T08:17:08Z\n"
	const ownerDecides = "status: open\nslice: spec\ngate: blocking\nquestion: Ship the export behind a flag?\noptions: |\n  1. Behind a flag — business: staged rollout\n  2. On for everyone — UX: no setup\n  3. Something else — describe it\nraised_at: 2026-10-19T03:25:00+02:00\n"
	root, dir := editWorkspace(t)
	existing := "# Questions\n\n## q-2026-10-16-001\nstatus: answered\nquestion: legacy entry without gate or raised_at\n"
	if err := os.WriteFile(filepath.Join(dir, "questions.md"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, heading, body, want string
	}{
		{"missing gate", "q-2026-10-19-001", strings.Replace(recommended, "gate: validating\n", "", 1), "exactly one gate:"},
		{"unknown gate", "q-2026-10-19-001", strings.Replace(recommended, "gate: validating", "gate: urgent", 1), "gate must be"},
		{"date-only raised_at", "q-2026-10-19-001", strings.Replace(recommended, "2026-10-19T08:17:08Z", "2026-10-19", 1), "RFC 3339"},
		{"recommended not first", "q-2026-10-19-001", strings.NewReplacer("Per user (Recommended)", "Per user", "Global —", "Global (Recommended) —").Replace(recommended), "only option 1"},
		{"recommended without proposed", "q-2026-10-19-001", strings.Replace(recommended, "proposed: Per user\n", "", 1), "add one proposed:"},
		{"proposed without recommended", "q-2026-10-19-001", ownerDecides + "proposed: owner decides\n", "omit it"},
		{"five concrete options", "q-2026-10-19-001", strings.Replace(ownerDecides, "  3. Something", "  3. A\n  4. B\n  5. C\n  6. Something", 1), "want 2-4"},
		{"one concrete option", "q-2026-10-19-001", strings.Replace(ownerDecides, "  2. On for everyone — UX: no setup\n", "", 1), "want 2-4"},
		{"no escape hatch", "q-2026-10-19-001", strings.Replace(ownerDecides, "Something else — describe it", "Later", 1), "escape hatch"},
		{"not open", "q-2026-10-19-001", strings.Replace(ownerDecides, "status: open", "status: answered", 1), "status must be open"},
		{"two questions", "q-2026-10-19-001", ownerDecides + "question: And another?\n", "exactly one question:"},
		{"bad id", "q-2026-10-19-1", ownerDecides, "q-YYYY-MM-DD-NNN"},
	} {
		var out, errOut bytes.Buffer
		if code := RunStateAppend(root, []string{"feat", "questions.md", "--heading", c.heading, "--body-file", "-"}, strings.NewReader(c.body), &out, &errOut); code != 3 || !strings.Contains(errOut.String(), c.want) {
			t.Fatalf("%s: exit %d, want 3 naming %q: %s", c.name, code, c.want, errOut.String())
		}
	}
	if got := readEdit(t, dir, "questions.md"); got != existing {
		t.Fatalf("a refused entry changed questions.md:\n%s", got)
	}

	// The canonical recommended and owner-decides shapes pass; length only warns.
	long := strings.Replace(ownerDecides, "raised_at:", "impact: "+strings.Repeat("evidence ", 300)+"\nraised_at:", 1)
	for i, body := range []string{recommended, ownerDecides, long} {
		var out, errOut bytes.Buffer
		heading := fmt.Sprintf("q-2026-10-19-00%d", i+1)
		if code := RunStateAppend(root, []string{"feat", "questions.md", "--heading", heading, "--body-file", "-"}, strings.NewReader(body), &out, &errOut); code != 0 {
			t.Fatalf("%s: exit %d: %s", heading, code, errOut.String())
		}
		if warned := strings.Contains(out.String(), "advisory: "+heading+": entry is"); warned != (body == long) {
			t.Fatalf("%s: length advisory %v: %s", heading, warned, out.String())
		}
	}
	if got := readEdit(t, dir, "questions.md"); !strings.HasPrefix(got, existing) || !strings.Contains(got, "## q-2026-10-19-003") {
		t.Fatalf("questions.md after appends:\n%s", got)
	}
}

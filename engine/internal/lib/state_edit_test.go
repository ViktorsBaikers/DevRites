package lib

import (
	"bytes"
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

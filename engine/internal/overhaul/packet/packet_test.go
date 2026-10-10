package packet

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runArea(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "full-1")
}

func writeArgs(run string, extra ...string) []string {
	base := []string{"write", run, "TASK-1", "a1", "implementer",
		"--phase", "REMEDIATING", "--lane", "backend", "--snapshot", "abc",
		"--authorized-by", "APPROVAL-1", "--brief", "packets/BRIEF.md"}
	return append(base, extra...)
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestWriteThenLoadRoundTrip(t *testing.T) {
	run := runArea(t)
	mustMkdir(t, run)
	var out, errb bytes.Buffer
	if code := Run(writeArgs(run, "--write-path", "a.go", "--write-path", "b.go"), &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	path := filepath.Join(run, "packets", "full-1__TASK-1__a1__implementer.json")
	if got := strings.TrimSpace(out.String()); got != path {
		t.Fatalf("printed %q, want %q", got, path)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if p["receipt_path"] != filepath.Join(run, "receipts", "full-1__TASK-1__a1__implementer.json") {
		t.Errorf("receipt_path = %v", p["receipt_path"])
	}
	if wp, _ := p["write_paths"].([]any); len(wp) != 2 || wp[0] != "a.go" || wp[1] != "b.go" {
		t.Errorf("write_paths = %v", p["write_paths"])
	}
	if p["run_id"] != "full-1" || p["lane"] != "backend" || p["authorized_by"] != "APPROVAL-1" {
		t.Errorf("fields not carried: %v", p)
	}
}

func TestWriteRefusesToOverwrite(t *testing.T) {
	run := runArea(t)
	mustMkdir(t, run)
	if code := Run(writeArgs(run), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("first write exit %d", code)
	}
	path := filepath.Join(run, "packets", "full-1__TASK-1__a1__implementer.json")
	before, _ := os.ReadFile(path)
	var errb bytes.Buffer
	if code := Run(writeArgs(run, "--lane", "frontend"), &bytes.Buffer{}, &errb); code != 1 {
		t.Fatalf("second write exit %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "already exists") {
		t.Errorf("stderr = %q", errb.String())
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Error("existing packet was modified")
	}
	if ents, _ := os.ReadDir(filepath.Dir(path)); len(ents) != 1 {
		t.Errorf("packets dir has %d entries, want 1", len(ents))
	}
}

func TestWriteRejectsBadIDs(t *testing.T) {
	for _, bad := range []string{"../x", "a/b", `a\b`, "..", "a__b"} {
		run := runArea(t)
		mustMkdir(t, run)
		args := writeArgs(run)
		args[2] = bad
		var errb bytes.Buffer
		if code := Run(args, &bytes.Buffer{}, &errb); code != 1 {
			t.Errorf("task %q: exit %d, want 1", bad, code)
		}
		if _, err := os.Stat(filepath.Join(run, "packets")); err == nil {
			t.Errorf("task %q: packets dir created", bad)
		}
	}
}

func TestWriteRejectsMissingFieldAndMissingRun(t *testing.T) {
	run := runArea(t)
	mustMkdir(t, run)
	args := []string{"write", run, "TASK-1", "a1", "implementer", "--phase", "P"}
	var errb bytes.Buffer
	if code := Run(args, &bytes.Buffer{}, &errb); code != 1 || !strings.Contains(errb.String(), "missing or empty lane") {
		t.Errorf("exit/stderr = %q", errb.String())
	}
	if code := Run(writeArgs(filepath.Join(t.TempDir(), "nope")), &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Errorf("missing run area exit %d, want 2", code)
	}
}

func TestFromFileSuppliesPlanAndFlagsOverride(t *testing.T) {
	run := runArea(t)
	mustMkdir(t, run)
	from := filepath.Join(t.TempDir(), "extra.json")
	if err := os.WriteFile(from, []byte(`{"lane":"other","plan":{"file":"revisions/plan-r1.json","rev":1},"write_paths":["x.go"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := Run(writeArgs(run, "--from", from), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	p, err := Load(filepath.Join(run, "packets", "full-1__TASK-1__a1__implementer.json"))
	if err != nil {
		t.Fatal(err)
	}
	if p["lane"] != "backend" {
		t.Errorf("lane = %v, flag should override file", p["lane"])
	}
	if wp, _ := p["write_paths"].([]any); len(wp) != 1 || wp[0] != "x.go" {
		t.Errorf("write_paths = %v, want file value kept", p["write_paths"])
	}
	if p["plan"] == nil {
		t.Error("plan from file lost")
	}
}

func TestLoadRejectsMismatchedNameAndBadPlan(t *testing.T) {
	run := runArea(t)
	mustMkdir(t, run)
	Run(writeArgs(run), &bytes.Buffer{}, &bytes.Buffer{})
	src := filepath.Join(run, "packets", "full-1__TASK-1__a1__implementer.json")
	b, _ := os.ReadFile(src)
	moved := filepath.Join(run, "packets", "full-1__TASK-2__a1__implementer.json")
	if err := os.WriteFile(moved, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(moved); err == nil || !strings.Contains(err.Error(), "file name does not match") {
		t.Errorf("Load of renamed packet: %v", err)
	}
	p := map[string]any{"plan": map[string]any{}}
	if why := Validate("", p); !strings.Contains(strings.Join(why, ";"), "plan must be an object with a file") {
		t.Errorf("Validate plan: %v", why)
	}
}

func TestUsage(t *testing.T) {
	var errb bytes.Buffer
	if code := Run([]string{"write", "run"}, &bytes.Buffer{}, &errb); code != 2 || !strings.Contains(errb.String(), "usage:") {
		t.Errorf("exit/stderr = %q", errb.String())
	}
}

func TestWriteRejectsReceiptPathOutsideRunArea(t *testing.T) {
	for i, bad := range []string{"/etc/evil.json", "../../outside/evil.json", "receipts/other.json", ""} {
		run := runArea(t)
		mustMkdir(t, run)
		from := filepath.Join(t.TempDir(), "extra.json")
		body := `{"receipt_path":"` + bad + `"}`
		if err := os.WriteFile(from, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		var errb bytes.Buffer
		code := Run(writeArgs(run, "--from", from), &bytes.Buffer{}, &errb)
		if bad == "" {
			// An empty receipt_path falls back to the derived default.
			if code != 0 {
				t.Errorf("case %d: exit %d, want 0", i, code)
			}
			continue
		}
		if code != 1 || !strings.Contains(errb.String(), "receipt_path") {
			t.Errorf("receipt_path %q: exit %d stderr %q, want 1 naming receipt_path", bad, code, errb.String())
		}
		if _, err := os.Stat(filepath.Join(run, "packets")); err == nil {
			t.Errorf("receipt_path %q: packet written", bad)
		}
	}
}

func TestLoadRejectsReceiptPathOutsideRunArea(t *testing.T) {
	run := runArea(t)
	mustMkdir(t, run)
	if code := Run(writeArgs(run), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	path := filepath.Join(run, "packets", "full-1__TASK-1__a1__implementer.json")
	if _, err := Load(path); err != nil {
		t.Fatalf("derived receipt_path rejected: %v", err)
	}
	b, _ := os.ReadFile(path)
	good := filepath.Join(run, "receipts", "full-1__TASK-1__a1__implementer.json")
	for _, bad := range []string{"/etc/evil.json", "../../outside/evil.json", filepath.Join(run, "receipts", "..", "x.json")} {
		edited := strings.Replace(string(b), good, bad, 1)
		if edited == string(b) {
			t.Fatal("receipt_path not found in packet")
		}
		if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "receipt_path") {
			t.Errorf("Load accepted receipt_path %q: %v", bad, err)
		}
	}
}

func TestWriteRefusesSymlinkedRunOrPacketsDir(t *testing.T) {
	outside := t.TempDir()
	base := t.TempDir()

	linkedRun := filepath.Join(base, "full-1")
	if err := os.Symlink(outside, linkedRun); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if code := Run(writeArgs(linkedRun), &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Errorf("symlinked run dir: exit %d, want 2", code)
	}
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("symlinked run dir: wrote %d entries outside", len(ents))
	}

	run := filepath.Join(base, "full-2")
	mustMkdir(t, run)
	if err := os.Symlink(outside, filepath.Join(run, "packets")); err != nil {
		t.Fatal(err)
	}
	if code := Run(writeArgs(run), &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Errorf("symlinked packets dir: exit %d, want 2", code)
	}
	if ents, _ := os.ReadDir(outside); len(ents) != 0 {
		t.Errorf("symlinked packets dir: wrote %d entries outside", len(ents))
	}
}

func TestWriteRejectsExtraPositionals(t *testing.T) {
	for _, tc := range [][]string{
		append(writeArgs("RUN"), "stray"),
		append([]string{"write", "RUN", "TASK-1", "a1", "implementer", "stray"}, writeArgs("RUN")[5:]...),
	} {
		run := runArea(t)
		mustMkdir(t, run)
		tc[1] = run
		var errb bytes.Buffer
		if code := Run(tc, &bytes.Buffer{}, &errb); code != 2 || !strings.Contains(errb.String(), "unexpected argument") {
			t.Errorf("%v: exit %d stderr %q, want 2 unexpected argument", tc[5:], code, errb.String())
		}
		if _, err := os.Stat(filepath.Join(run, "packets")); err == nil {
			t.Errorf("%v: packet written", tc[5:])
		}
	}
}

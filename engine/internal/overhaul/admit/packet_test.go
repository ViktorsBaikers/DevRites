package admit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const packetName = "full-1__T-1__A1__implementer.json"

// packetRun builds a run area named full-1 with one dispatched attempt and
// returns the run path and the path of its valid packet.
func packetRun(t *testing.T) (string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "work", "full-1")
	write(t, filepath.Join(dir, "CURRENT"), "g0001\n")
	write(t, filepath.Join(dir, "g0001/run.json"), J{"run_id": "full-1"})
	write(t, filepath.Join(dir, "g0001/dispatch.json"), J{"attempts": L{
		J{"task_id": "T-1", "attempt_id": "A1", "role": "implementer", "snapshot": "s1", "status": "dispatched",
			"receipt": "receipts/full-1__T-1__A1__implementer.json", "packet": "packets/" + packetName},
		J{"task_id": "T-1", "attempt_id": "A0", "role": "implementer", "snapshot": "s1", "status": "admitted"},
	}})
	pk := write(t, filepath.Join(dir, "packets", packetName), validPacket(dir))
	return dir, pk
}

func validPacket(dir string) J {
	return J{"run_id": "full-1", "phase": "REMEDIATING", "task_id": "T-1", "attempt_id": "A1",
		"role": "implementer", "lane": "backend", "snapshot": "s1", "authorized_by": "APPROVAL-1",
		"brief": "packets/BRIEF.md", "write_paths": L{"a.go"},
		"receipt_path": filepath.Join(dir, "receipts", "full-1__T-1__A1__implementer.json")}
}

func TestPacketAdmit(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(J)
		code   int
		want   string
	}{
		{"valid", func(J) {}, 0, `{"verdict": "ADMISSIBLE", "reasons": []}`},
		{"missing snapshot", func(p J) { delete(p, "snapshot") }, 1, "missing or empty snapshot"},
		{"missing attempt_id", func(p J) { delete(p, "attempt_id") }, 1, "missing or empty attempt_id"},
		{"empty role", func(p J) { p["role"] = "" }, 1, "missing or empty role"},
		{"wrong run", func(p J) { p["run_id"] = "full-2" }, 1, "run_id differs from the run"},
		{"receipt elsewhere", func(p J) { p["receipt_path"] = "/elsewhere/r.json" }, 1, "receipt_path must be <run>/receipts/"},
		{"snapshot mismatch", func(p J) { p["snapshot"] = "s2" }, 1, "snapshot differs from the dispatched attempt"},
		{"role mismatch", func(p J) { p["role"] = "reviewer" }, 1, "role differs from the dispatched attempt"},
		{"unknown attempt", func(p J) { p["attempt_id"] = "A9" }, 1, "no such dispatched attempt"},
		{"already admitted", func(p J) { p["attempt_id"] = "A0" }, 3, `{"verdict": "DUPLICATE_OR_LATE", "attempt_status": "admitted"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, _ := packetRun(t)
			p := validPacket(dir)
			c.mutate(p)
			// the file keeps its name so only the field under test is wrong
			pk := write(t, filepath.Join(dir, "packets", packetName), p)
			code, out, errs := run("packet", dir, pk)
			if code != c.code || !strings.Contains(out, c.want) {
				t.Fatalf("want %d with %q, got %d:\n%s%s", c.code, c.want, code, out, errs)
			}
		})
	}
}

// The dispatch.json attempt records the receipt location; a packet naming
// another run-relative receipt is not the packet that was dispatched.
func TestPacketReceiptDiffersFromAttempt(t *testing.T) {
	dir, pk := packetRun(t)
	write(t, filepath.Join(dir, "g0001/dispatch.json"), J{"attempts": L{
		J{"task_id": "T-1", "attempt_id": "A1", "role": "implementer", "snapshot": "s1", "status": "dispatched",
			"receipt": "receipts/other.json"}}})
	code, out, _ := run("packet", dir, pk)
	if code != 1 || !strings.Contains(out, "receipt_path differs from the dispatched attempt's receipt") {
		t.Fatalf("got %d: %s", code, out)
	}
}

// The attempt also records the packet location; an attempt that recorded one
// packet cannot admit a different file whose identifiers happen to match.
func TestPacketPathDiffersFromAttempt(t *testing.T) {
	dir, pk := packetRun(t)
	write(t, filepath.Join(dir, "g0001/dispatch.json"), J{"attempts": L{
		J{"task_id": "T-1", "attempt_id": "A1", "role": "implementer", "snapshot": "s1", "status": "dispatched",
			"packet": "packets/other.json"}}})
	code, out, _ := run("packet", dir, pk)
	if code != 1 || !strings.Contains(out, "packet path differs from the dispatched attempt's packet") {
		t.Fatalf("got %d: %s", code, out)
	}
}

// An attempt that records no packet key predates the field and is judged on
// the other agreements alone.
func TestPacketAttemptWithoutPacketKey(t *testing.T) {
	dir, pk := packetRun(t)
	write(t, filepath.Join(dir, "g0001/dispatch.json"), J{"attempts": L{
		J{"task_id": "T-1", "attempt_id": "A1", "role": "implementer", "snapshot": "s1", "status": "dispatched"}}})
	code, out, errs := run("packet", dir, pk)
	if code != 0 || !strings.Contains(out, `"verdict": "ADMISSIBLE"`) {
		t.Fatalf("got %d: %s%s", code, out, errs)
	}
}

// A recorded packet path that names the same file by an unclean spelling is
// the packet that was dispatched.
func TestPacketPathUncleanSpelling(t *testing.T) {
	for _, rec := range []string{"packets/./" + packetName, "./packets/../packets/" + packetName} {
		t.Run(rec, func(t *testing.T) {
			dir, pk := packetRun(t)
			write(t, filepath.Join(dir, "g0001/dispatch.json"), J{"attempts": L{
				J{"task_id": "T-1", "attempt_id": "A1", "role": "implementer", "snapshot": "s1", "status": "dispatched",
					"packet": rec}}})
			code, out, errs := run("packet", dir, pk)
			if code != 0 || !strings.Contains(out, `"verdict": "ADMISSIBLE"`) {
				t.Fatalf("got %d: %s%s", code, out, errs)
			}
		})
	}
}

// admit packet accepts exactly the receipt_path spellings packet.Validate
// accepts: the run's receipts file, absolute or spelled from the working
// directory, and nothing else.
func TestPacketReceiptPathSpellings(t *testing.T) {
	dir, pk := packetRun(t)
	name := "full-1__T-1__A1__implementer.json"
	for _, c := range []struct {
		label, path string
		code        int
	}{
		{"bare file name", name, 1},
		{"receipts/<name>", filepath.Join("receipts", name), 1},
		{"another run", filepath.Join("work", "full-2", "receipts", name), 1},
		{"relative to the working directory", filepath.Join("full-1", "receipts", name), 0},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Chdir(filepath.Dir(dir))
			p := validPacket(dir)
			p["receipt_path"] = c.path
			write(t, pk, p)
			code, out, errs := run("packet", dir, pk)
			if code != c.code || (c.code == 1 && !strings.Contains(out, "receipt_path must be")) {
				t.Fatalf("want %d, got %d: %s%s", c.code, code, out, errs)
			}
		})
	}
}

// A valid packet with an absolute receipt_path is admitted from an unrelated
// working directory with an absolute run path, and from the run's parent
// directory with a relative run path.
func TestPacketAdmitFromAnyCwd(t *testing.T) {
	dir, pk := packetRun(t)
	t.Chdir(t.TempDir())
	if code, out, errs := run("packet", dir, pk); code != 0 {
		t.Fatalf("absolute run: %d %s%s", code, out, errs)
	}
	t.Chdir(filepath.Dir(dir))
	if code, out, errs := run("packet", "full-1", filepath.Join("full-1", "packets", packetName)); code != 0 {
		t.Fatalf("relative run: %d %s%s", code, out, errs)
	}
}

func TestPacketUsesWorkingGeneration(t *testing.T) {
	dir, pk := packetRun(t)
	stage := filepath.Join(dir, "g0002.tmp")
	write(t, filepath.Join(stage, ".base"), "g0001")
	write(t, filepath.Join(stage, "run.json"), J{"run_id": "full-1"})
	write(t, filepath.Join(stage, "dispatch.json"), J{"attempts": L{}})
	if code, out, _ := run("packet", dir, pk); code != 1 || !strings.Contains(out, "no such dispatched attempt") {
		t.Fatalf("open stage ignored: %d %s", code, out)
	}
}

func TestPacketSymlinkedPath(t *testing.T) {
	dir, pk := packetRun(t)
	link := filepath.Join(dir, "packets", "link.json")
	if err := os.Symlink(pk, link); err != nil {
		t.Skip(err)
	}
	if code, out, _ := run("packet", dir, link); code != 1 || !strings.Contains(out, "must not be a symlink") {
		t.Fatalf("symlinked file: %d %s", code, out)
	}
	// a symlinked packets directory is not the run's packets directory
	real := filepath.Join(t.TempDir(), "packets")
	if err := os.Rename(filepath.Join(dir, "packets"), real); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, "packets")); err != nil {
		t.Skip(err)
	}
	if code, out, _ := run("packet", dir, filepath.Join(dir, "packets", packetName)); code != 1 || !strings.Contains(out, "not under the run's packets directory") {
		t.Fatalf("symlinked packets dir: %d %s", code, out)
	}
}

func TestPacketMisplaced(t *testing.T) {
	dir, pk := packetRun(t)
	moved := filepath.Join(dir, "stray.json")
	if err := os.Rename(pk, moved); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run("packet", dir, moved)
	if code != 1 || !strings.Contains(out, "file name does not match") || !strings.Contains(out, "not under the run's packets directory") {
		t.Fatalf("got %d: %s", code, out)
	}
}

func TestPacketIOAndUsage(t *testing.T) {
	dir, _ := packetRun(t)
	if code, _, errs := run("packet", dir, filepath.Join(dir, "packets", "missing.json")); code != 2 || !strings.HasPrefix(errs, "ERROR: ") {
		t.Fatalf("missing packet: %d %q", code, errs)
	}
	write(t, filepath.Join(dir, "packets", "bad.json"), "{not json")
	if code, _, errs := run("packet", dir, filepath.Join(dir, "packets", "bad.json")); code != 2 || !strings.HasPrefix(errs, "ERROR: ") {
		t.Fatalf("unparsable packet: %d %q", code, errs)
	}
	if code, _, errs := run("packet", dir); code != 2 || !strings.Contains(errs, "packet RUN PACKET") {
		t.Fatalf("usage: %d %q", code, errs)
	}
}

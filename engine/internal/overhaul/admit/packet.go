package admit

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/overhaul/ovio"
	"github.com/devrites/devrites/internal/overhaul/packet"
	"github.com/devrites/devrites/internal/overhaul/records"
)

// packetAdmit checks a dispatch packet file against the run area and the
// dispatch.json attempt it names. The packet must sit, unlinked, at
// RUN/packets/<name>.json and pass packet.Validate for that run, which reads
// a relative receipt_path from the working directory, as packet write does.
func packetAdmit(run, path string, stdout io.Writer) (int, error) {
	cur, err := os.ReadFile(join(run, "CURRENT"))
	if err != nil {
		return 0, err
	}
	gen := workingGeneration(run, strings.TrimSpace(string(cur)))
	rj, err := ovio.LoadObject(join(gen, "run.json"))
	if err != nil {
		return 0, err
	}
	disp, err := ovio.LoadObject(join(gen, "dispatch.json"))
	if err != nil {
		return 0, err
	}
	p, err := ovio.LoadObject(path)
	if err != nil {
		return 0, err
	}
	eq := func(a, b any) bool { return records.Repr(a) == records.Repr(b) }

	// Resolved once, so every check below sees the same run location whatever
	// the working directory. receipt_path is judged exactly as packet write
	// judges it: unchanged, by packet.Validate.
	absRun, err := filepath.Abs(run)
	if err != nil {
		return 0, err
	}
	canon := filepath.Join(absRun, "receipts", packet.Name(p)+".json")
	why := packet.Validate(absRun, p)

	if st, err := os.Lstat(path); err != nil || st.Mode()&os.ModeSymlink != 0 {
		why = append(why, "packet file must not be a symlink")
	}
	if filepath.Base(path) != packet.Name(p)+".json" {
		why = append(why, "file name does not match run_id, task_id, attempt_id and role")
	}
	if realpath(filepath.Dir(path)) != filepath.Join(realpath(run), "packets") {
		why = append(why, "packet is not under the run's packets directory")
	}
	if !eq(p["run_id"], rj["run_id"]) {
		why = append(why, "run_id differs from the run")
	}

	var x map[string]any
	for _, it := range ovio.List(disp["attempts"]) {
		if a := ovio.Obj(it); eq(a["task_id"], p["task_id"]) && eq(a["attempt_id"], p["attempt_id"]) {
			x = a
			break
		}
	}
	if x == nil {
		why = append(why, "no such dispatched attempt")
		return result(stdout, why, "ADMISSIBLE"), nil
	}
	if s := ovio.Str(x["status"]); s == "admitted" || s == "rejected" || s == "cancelled" {
		verdict(stdout, "DUPLICATE_OR_LATE", "attempt_status", x["status"])
		return 3, nil
	}
	if !eq(p["role"], x["role"]) {
		why = append(why, "role differs from the dispatched attempt")
	}
	if ovio.Truthy(x["snapshot"]) && !eq(p["snapshot"], x["snapshot"]) {
		why = append(why, "snapshot differs from the dispatched attempt")
	}
	if r := ovio.Str(x["receipt"]); r != "" && filepath.Clean(join(absRun, r)) != canon {
		why = append(why, "receipt_path differs from the dispatched attempt's receipt")
	}
	if pp := ovio.Str(x["packet"]); pp != "" && filepath.Clean(join(absRun, pp)) != filepath.Join(absRun, "packets", packet.Name(p)+".json") {
		why = append(why, "packet path differs from the dispatched attempt's packet")
	}
	return result(stdout, why, "ADMISSIBLE"), nil
}

// Package packet implements the /overhaul packet tool: it writes the dispatch
// packet for one worker attempt and validates packets on load.
//
//	write RUN TASK ATTEMPT ROLE [flags]   create RUN/packets/<run>__<task>__<attempt>__<role>.json
//
// A packet is a JSON object. Flags fill the common fields; --from FILE supplies
// any further fields (plan, task, read paths, budgets) and flags override it.
// Exit: 0 written, 1 invalid packet or packet already exists, 2 usage/IO error.
package packet

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

const usage = `usage: overhaul packet <command> ...

  write RUN TASK ATTEMPT ROLE [--from FILE] [--phase P] [--wave W] [--lane L]
        [--snapshot S] [--authorized-by A] [--brief PATH] [--write-path PATH]...
Exit: 0 written, 1 invalid packet or packet already exists, 2 usage/IO error.
`

// required are the string fields every packet carries.
var required = []string{"run_id", "phase", "task_id", "attempt_id", "role", "lane", "snapshot", "authorized_by", "brief", "receipt_path"}

// idFields name the path components of a packet file and must be plain names.
var idFields = []string{"run_id", "task_id", "attempt_id", "role"}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(s string) error { *m = append(*m, s); return nil }

// Run executes the tool with its arguments (tool name excluded).
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 5 || args[0] != "write" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	fs := flag.NewFlagSet("packet write", flag.ContinueOnError)
	fs.SetOutput(stderr)
	from := fs.String("from", "", "JSON object with further packet fields")
	set := map[string]*string{}
	for flagName, field := range map[string]string{
		"phase": "phase", "wave": "wave", "lane": "lane", "snapshot": "snapshot",
		"authorized-by": "authorized_by", "brief": "brief",
	} {
		set[field] = fs.String(flagName, "", field)
	}
	var paths multi
	fs.Var(&paths, "write-path", "exact path the worker may write (repeatable)")
	if err := fs.Parse(args[5:]); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "ERROR: unexpected argument %q\n", fs.Arg(0))
		return 2
	}
	run := args[1]
	p := map[string]any{}
	if *from != "" {
		m, err := ovio.LoadObject(*from)
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: %v\n", err)
			return 2
		}
		p = m
	}
	p["run_id"], p["task_id"], p["attempt_id"], p["role"] = filepath.Base(filepath.Clean(run)), args[2], args[3], args[4]
	for field, v := range set {
		if *v != "" {
			p[field] = *v
		}
	}
	if _, ok := p["write_paths"]; !ok || len(paths) > 0 {
		wp := []any{}
		for _, w := range paths {
			wp = append(wp, w)
		}
		p["write_paths"] = wp
	}
	if ovio.Str(p["receipt_path"]) == "" {
		p["receipt_path"] = filepath.Join(run, "receipts", Name(p)+".json")
	}
	if why := Validate(run, p); len(why) > 0 {
		for _, w := range why {
			fmt.Fprintf(stderr, "REJECT: %s\n", w)
		}
		return 1
	}
	out, code, err := write(run, p)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return code
	}
	fmt.Fprintln(stdout, out)
	return 0
}

// Name is the packet file stem: <run>__<task>__<attempt>__<role>.
func Name(p map[string]any) string {
	return strings.Join([]string{ovio.Str(p["run_id"]), ovio.Str(p["task_id"]), ovio.Str(p["attempt_id"]), ovio.Str(p["role"])}, "__")
}

// Validate returns every reason the packet is malformed for the run area run;
// none means valid. receipt_path must be a clean path that resolves to <run>/receipts/<name>.json.
func Validate(run string, p map[string]any) []string {
	var why []string
	for _, k := range required {
		if ovio.Str(p[k]) == "" {
			why = append(why, "missing or empty "+k)
		}
	}
	for _, k := range idFields {
		if s := ovio.Str(p[k]); s == "." || s == ".." || strings.ContainsAny(s, `/\`+"\x00") || strings.Contains(s, "__") {
			why = append(why, k+" must be a plain name without path separators or \"__\"")
		}
	}
	if rp := ovio.Str(p["receipt_path"]); rp != "" && !samePath(rp, filepath.Join(run, "receipts", Name(p)+".json")) {
		why = append(why, "receipt_path must be <run>/receipts/<run_id>__<task_id>__<attempt_id>__<role>.json")
	}
	if w, ok := p["write_paths"].([]any); !ok {
		why = append(why, "write_paths must be an array (empty for read-only roles)")
	} else if len(ovio.Strings(w)) != len(w) {
		why = append(why, "write_paths must hold only strings")
	}
	if pl, ok := p["plan"]; ok && ovio.Str(ovio.Obj(pl)["file"]) == "" {
		why = append(why, "plan must be an object with a file")
	}
	return why
}

// samePath reports whether a is the clean spelling of the location b names.
func samePath(a, b string) bool {
	x, err1 := filepath.Abs(a)
	y, err2 := filepath.Abs(b)
	return err1 == nil && err2 == nil && a == filepath.Clean(a) && x == y
}

// Load reads a packet file and validates it against the run area that holds
// its packets/ directory, including that the file name matches the packet's
// own identifiers.
func Load(path string) (map[string]any, error) {
	p, err := ovio.LoadObject(path)
	if err != nil {
		return nil, err
	}
	why := Validate(filepath.Dir(filepath.Dir(path)), p)
	if len(why) == 0 && filepath.Base(path) != Name(p)+".json" {
		why = append(why, "file name does not match run_id, task_id, attempt_id and role")
	}
	if len(why) > 0 {
		return nil, fmt.Errorf("%s: %s", path, strings.Join(why, "; "))
	}
	return p, nil
}

// write publishes the packet atomically and never replaces an existing one:
// the content is written to a temporary file that is then hard-linked into place.
func write(run string, p map[string]any) (string, int, error) {
	if !realDir(run) {
		return "", 2, fmt.Errorf("run area %s is not a directory", run)
	}
	dir := filepath.Join(run, "packets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", 2, err
	}
	if !realDir(dir) {
		return "", 2, fmt.Errorf("%s is not a plain directory", dir)
	}
	b, err := ovio.MarshalIndent(p)
	if err != nil {
		return "", 2, err
	}
	tmp, err := os.CreateTemp(dir, ".packet-*.tmp")
	if err != nil {
		return "", 2, err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return "", 2, err
	}
	if err := tmp.Close(); err != nil {
		return "", 2, err
	}
	dest := filepath.Join(dir, Name(p)+".json")
	if err := os.Link(tmp.Name(), dest); err != nil {
		if os.IsExist(err) {
			return "", 1, fmt.Errorf("packet %s already exists", dest)
		}
		return "", 2, err
	}
	return dest, 0, nil
}

// realDir reports whether path is a directory and not a symlink to one, so a
// packet is never written through a link out of the run area.
func realDir(path string) bool {
	st, err := os.Lstat(filepath.Clean(path))
	return err == nil && st.IsDir()
}

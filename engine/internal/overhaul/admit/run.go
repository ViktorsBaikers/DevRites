// Package admit implements the /overhaul admit tool: read-only admission
// checks for worker output.
//
//	receipt RUN RECEIPT [--observed FILE]   receipt vs its dispatch packet; writers vs observed paths
//	anchor ROOT PROPOSALS                   every quoted location matches its file lines (ROOT = snapshot tree or repo)
//
// Exit: 0 admissible, 1 reject (reasons listed), 2 usage/IO error, 3 duplicate
// or late receipt. Nothing here edits canonical records; the coordinator
// records the verdict in a staged generation.
package admit

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/devrites/devrites/internal/overhaul/ovio"
	"github.com/devrites/devrites/internal/overhaul/records"
)

const usage = `usage: overhaul admit <command> ...

  receipt RUN RECEIPT [--observed FILE]   receipt vs its dispatch packet; writers vs observed paths
  anchor ROOT PROPOSALS                   every quoted location matches its file lines (ROOT = snapshot tree or repo)
Exit: 0 admissible, 1 reject (reasons listed), 2 usage/IO error, 3 duplicate or late receipt.
`

var (
	writers  = map[string]bool{"implementer": true}
	outcomes = []string{"findings", "gap", "no-findings", "patch", "verdict"} // sorted
)

// Run executes the tool with its arguments (tool name excluded).
func Run(args []string, stdout, stderr io.Writer) int {
	var code int
	var err error
	switch {
	case len(args) >= 3 && args[0] == "receipt":
		observed, hasObserved := "", false
		for i, a := range args {
			if a == "--observed" {
				if i+1 >= len(args) {
					fmt.Fprint(stderr, usage)
					return 2
				}
				observed, hasObserved = args[i+1], true
				break
			}
		}
		code, err = receipt(args[1], args[2], observed, hasObserved, stdout)
	case len(args) >= 3 && args[0] == "anchor":
		code, err = anchor(args[1], args[2], stdout, stderr)
	default:
		fmt.Fprint(stderr, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	return code
}

// join mirrors os.path.join: an absolute part replaces everything before it.
func join(base, rel string) string {
	switch {
	case strings.HasPrefix(rel, "/"):
		return rel
	case base == "" || strings.HasSuffix(base, "/"):
		return base + rel
	}
	return base + "/" + rel
}

func realpath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	return ovio.Resolve(abs)
}

// pyJSON renders s like Python's json.dumps (ensure_ascii=True).
func pyJSON(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			switch {
			case r > 0xffff:
				r1, r2 := utf16.EncodeRune(r)
				fmt.Fprintf(&b, `\u%04x\u%04x`, r1, r2)
			case r < 0x20 || r > 0x7e:
				fmt.Fprintf(&b, `\u%04x`, r)
			default:
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func pyValue(v any) string {
	switch x := v.(type) {
	case string:
		return pyJSON(x)
	case []string:
		parts := make([]string, len(x))
		for i, s := range x {
			parts[i] = pyJSON(s)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// verdict prints {"verdict": v, <key>: val} exactly as Python's json.dumps does.
func verdict(w io.Writer, v, key string, val any) {
	fmt.Fprintf(w, "{\"verdict\": %s, %s: %s}\n", pyJSON(v), pyJSON(key), pyValue(val))
}

func keyed(m any, k string) (any, error) {
	o, ok := m.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("'%s': parent is not an object", k)
	}
	v, ok := o[k]
	if !ok {
		return nil, fmt.Errorf("'%s'", k)
	}
	return v, nil
}

// strSet collects string members keyed by records.Repr (non-strings never match paths).
func strSet(vals []any) map[string]any {
	s := map[string]any{}
	for _, v := range vals {
		s[records.Repr(v)] = v
	}
	return s
}

func sortedVals(s map[string]any) []any {
	out := make([]any, 0, len(s))
	for _, v := range s {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return records.PyStr(out[i]) < records.PyStr(out[j]) })
	return out
}

func receipt(run, path, observed string, hasObserved bool, stdout io.Writer) (int, error) {
	cur, err := os.ReadFile(join(run, "CURRENT"))
	if err != nil {
		return 0, err
	}
	gen := join(run, strings.TrimSpace(string(cur)))
	rj, err := ovio.LoadObject(join(gen, "run.json"))
	if err != nil {
		return 0, err
	}
	disp, err := ovio.LoadObject(join(gen, "dispatch.json"))
	if err != nil {
		return 0, err
	}
	rc, err := ovio.LoadObject(path)
	if err != nil {
		return 0, err
	}
	eq := func(a, b any) bool { return records.Repr(a) == records.Repr(b) }
	var why []string
	var x map[string]any
	for _, it := range ovio.List(disp["attempts"]) {
		if a := ovio.Obj(it); eq(a["task_id"], rc["task_id"]) && eq(a["attempt_id"], rc["attempt_id"]) {
			x = a
			break
		}
	}
	if ovio.Str(rc["schema"]) != "overhaul.receipt/1" || !eq(rc["run_id"], rj["run_id"]) {
		why = append(why, "wrong schema or run")
	}
	if x == nil {
		why = append(why, "no such dispatched attempt")
	} else {
		if s := ovio.Str(x["status"]); s == "admitted" || s == "rejected" || s == "cancelled" {
			verdict(stdout, "DUPLICATE_OR_LATE", "attempt_status", x["status"])
			return 3, nil
		}
		for _, k := range []string{"role", "lane", "snapshot"} {
			if !eq(rc[k], x[k]) {
				why = append(why, k+" differs from the dispatch packet")
			}
		}
		for _, k := range []string{"started_at", "finished_at"} {
			if !ovio.Truthy(rc[k]) {
				why = append(why, "missing "+k)
			}
		}
		if !slices.Contains(outcomes, ovio.Str(rc["outcome"])) {
			why = append(why, "outcome must be one of "+records.Repr(outcomes))
		}
		if ovio.Str(rc["outcome"]) == "no-findings" && !ovio.Truthy(rc["inspected"]) {
			why = append(why, "no-findings without inspected ranges is malformed")
		}
		if writers[ovio.Str(x["role"])] {
			v, err := keyed(rj, "revisions")
			if err == nil {
				v, err = keyed(v, "plan")
			}
			if err == nil {
				v, err = keyed(v, "file")
			}
			if err != nil {
				return 0, err
			}
			plan, err := ovio.LoadObject(join(run, ovio.Str(v)))
			if err != nil {
				return 0, err
			}
			allowed := map[string]any{}
			for _, t := range ovio.List(plan["tasks"]) {
				if tm := ovio.Obj(t); eq(tm["id"], x["task_id"]) {
					for k, p := range strSet(ovio.List(tm["allowed_paths"])) {
						allowed[k] = p
					}
				}
			}
			claimed := strSet(ovio.List(rc["changed_paths"]))
			outside := map[string]any{}
			for k, p := range claimed {
				if _, ok := allowed[k]; !ok {
					outside[k] = p
				}
			}
			if len(outside) > 0 {
				why = append(why, "changed paths outside contract: "+records.Repr(sortedVals(outside)))
			}
			if !hasObserved {
				why = append(why, "writer receipt needs observed changed paths (--observed)")
			} else {
				b, err := os.ReadFile(observed) // #nosec G304 -- observed-paths file is an explicit operator argument
				if err != nil {
					return 0, err
				}
				text := strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(string(b)) // universal newlines
				seen := map[string]any{}
				for _, ln := range strings.Split(text, "\n") {
					if ln = strings.TrimSpace(ln); ln != "" {
						seen[records.Repr(ln)] = ln
					}
				}
				same := len(seen) == len(claimed)
				for k := range seen {
					if _, ok := claimed[k]; !ok {
						same = false
					}
				}
				if !same {
					why = append(why, fmt.Sprintf("observed paths %s differ from claimed %s",
						records.Repr(sortedVals(seen)), records.Repr(sortedVals(claimed))))
				}
			}
		}
	}
	return result(stdout, why, "ADMISSIBLE"), nil
}

func result(stdout io.Writer, why []string, ok string) int {
	if why == nil {
		why = []string{}
	}
	if len(why) > 0 {
		verdict(stdout, "REJECT", "reasons", why)
		return 1
	}
	verdict(stdout, ok, "reasons", why)
	return 0
}

// splitlines is Python's str.splitlines().
func splitlines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		switch r {
		case '\n', '\r', '\v', '\f', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
			out = append(out, s[start:i-size])
			if r == '\r' && i < len(s) && s[i] == '\n' {
				i++
			}
			start = i
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// pyInt is Python's int() on a decoded JSON value.
func pyInt(v any) (int, error) {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return int(i), nil
		}
		f, err := x.Float64()
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return 0, fmt.Errorf("invalid line number %s", x)
		}
		return int(f), nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(x))
		if err != nil {
			return 0, fmt.Errorf("invalid literal for int() with base 10: %s", records.Repr(x))
		}
		return i, nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	}
	return 0, fmt.Errorf("int() argument must be a string or a number, not %s", records.Repr(v))
}

func anchor(root, path string, stdout, stderr io.Writer) (int, error) {
	doc, err := ovio.LoadJSON(path)
	if err != nil {
		return 0, err
	}
	items, ok := ovio.Obj(doc)["findings"].([]any)
	if !ok || len(items) == 0 {
		fmt.Fprintln(stderr, `ERROR: expected {"findings": [...]} with at least one finding`)
		return 2, nil
	}
	why := []string{}
	base := realpath(root)
	for _, it := range items {
		f := ovio.Obj(it)
		id := records.PyStr(f["id"])
		if !ovio.Truthy(f["locations"]) {
			why = append(why, id+": no locations to anchor")
		}
		for _, l := range ovio.List(f["locations"]) {
			loc := ovio.Obj(l)
			rel, quote := ovio.Str(loc["path"]), ovio.Str(loc["quote"])
			full := realpath(join(base, rel))
			if fi, err := os.Stat(full); !strings.HasPrefix(full, base+string(os.PathSeparator)) || err != nil || !fi.Mode().IsRegular() {
				why = append(why, fmt.Sprintf("%s: %s is not a file under the reviewed tree", id, rel))
				continue
			}
			b, err := os.ReadFile(full) // #nosec G304 -- anchor path checked to stay inside the reviewed tree
			if err != nil {
				return 0, err
			}
			lines := splitlines(strings.ToValidUTF8(string(b), "�"))
			start, ok := loc["start"]
			if !ok {
				start = json.Number("0")
			}
			end, ok := loc["end"]
			if !ok {
				end = start
			}
			s, err := pyInt(start)
			if err != nil {
				return 0, err
			}
			e, err := pyInt(end)
			if err != nil {
				return 0, err
			}
			if !(1 <= s && s <= e && e <= len(lines)) {
				why = append(why, fmt.Sprintf("%s: %s lines %d-%d outside file (%d lines)", id, rel, s, e, len(lines)))
				continue
			}
			q := strings.Join(strings.Fields(quote), " ")
			w := strings.Join(strings.Fields(strings.Join(lines[s-1:e], " ")), " ")
			if q == "" || !strings.Contains(w, q) {
				why = append(why, fmt.Sprintf("%s: quote not found at %s:%d-%d", id, rel, s, e))
			} else if utf8.RuneCountInString(strings.ReplaceAll(q, " ", "")) < 8 && q != w {
				why = append(why, fmt.Sprintf("%s: quote at %s:%d is too short to anchor; quote the whole line", id, rel, s))
			}
		}
	}
	return result(stdout, why, "ANCHORED"), nil
}

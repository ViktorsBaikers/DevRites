package admit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type J = map[string]any
type L = []any

func write(t *testing.T, path string, v any) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b, ok := v.(string)
	if !ok {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		b = string(raw)
	}
	if err := os.WriteFile(path, []byte(b), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func newRun(t *testing.T) string {
	dir := filepath.Join(t.TempDir(), "run")
	write(t, filepath.Join(dir, "CURRENT"), "g0001\n")
	write(t, filepath.Join(dir, "g0001/run.json"), J{"run_id": "run-1",
		"revisions": J{"plan": J{"rev": 1, "file": "revisions/plan-r1.json"}}})
	write(t, filepath.Join(dir, "g0001/dispatch.json"), J{"attempts": L{
		J{"task_id": "T-1", "attempt_id": "A0", "role": "implementer", "lane": "fix", "snapshot": "s1", "status": "admitted"},
		J{"task_id": "T-1", "attempt_id": "A1", "role": "implementer", "lane": "fix", "snapshot": "s1", "status": "running"},
		J{"task_id": "R-1", "attempt_id": "A1", "role": "reviewer", "lane": "security", "snapshot": "s1", "status": "running"},
	}})
	write(t, filepath.Join(dir, "revisions/plan-r1.json"), J{"rev": 1, "tasks": L{
		J{"id": "T-1", "allowed_paths": L{"src/a.py"}}, J{"id": "T-2", "allowed_paths": L{"src/b.py"}}}})
	return dir
}

func writerReceipt() J {
	return J{"schema": "overhaul.receipt/1", "run_id": "run-1", "task_id": "T-1", "attempt_id": "A1",
		"role": "implementer", "lane": "fix", "snapshot": "s1", "started_at": "t0", "finished_at": "t1",
		"outcome": "patch", "changed_paths": L{"src/a.py"}}
}

func TestReceipt(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(J)
		observed string // "-" means no --observed flag
		code     int
		want     string
	}{
		{"writer admissible", func(J) {}, "src/a.py\n\n", 0, `{"verdict": "ADMISSIBLE", "reasons": []}`},
		{"observed differs", func(J) {}, "src/b.py\n", 1, "observed paths ['src/b.py'] differ from claimed ['src/a.py']"},
		{"claimed outside contract", func(r J) { r["changed_paths"] = L{"src/a.py", "src/c.py"} }, "src/a.py\nsrc/c.py\n", 1, "changed paths outside contract: ['src/c.py']"},
		{"writer needs observed", func(J) {}, "-", 1, "writer receipt needs observed changed paths (--observed)"},
		{"snapshot mismatch", func(r J) { r["snapshot"] = "s2" }, "src/a.py\n", 1, "snapshot differs from the dispatch packet"},
		{"bad outcome", func(r J) { r["outcome"] = "done" }, "src/a.py\n", 1, "outcome must be one of ['findings', 'gap', 'no-findings', 'patch', 'verdict']"},
		{"missing finished_at", func(r J) { delete(r, "finished_at") }, "src/a.py\n", 1, "missing finished_at"},
		{"wrong run", func(r J) { r["run_id"] = "run-2" }, "src/a.py\n", 1, "wrong schema or run"},
		{"unknown attempt", func(r J) { r["attempt_id"] = "A9" }, "-", 1, "no such dispatched attempt"},
		{"duplicate or late", func(r J) { r["attempt_id"] = "A0" }, "src/a.py\n", 3, `{"verdict": "DUPLICATE_OR_LATE", "attempt_status": "admitted"}`},
		{"reviewer no-findings without inspected", func(r J) {
			r["task_id"], r["role"], r["lane"], r["outcome"] = "R-1", "reviewer", "security", "no-findings"
			delete(r, "changed_paths")
		}, "-", 1, "no-findings without inspected ranges is malformed"},
		{"reviewer admissible", func(r J) {
			r["task_id"], r["role"], r["lane"], r["outcome"], r["inspected"] = "R-1", "reviewer", "security", "no-findings", L{"src/a.py:1-9"}
		}, "-", 0, "ADMISSIBLE"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := newRun(t)
			rc := writerReceipt()
			c.mutate(rc)
			args := []string{"receipt", dir, write(t, filepath.Join(dir, "receipts/r.json"), rc)}
			if c.observed != "-" {
				args = append(args, "--observed", write(t, filepath.Join(dir, "observed.txt"), c.observed))
			}
			code, out, errs := run(args...)
			if code != c.code || !strings.Contains(out, c.want) {
				t.Fatalf("want %d with %q, got %d:\n%s%s", c.code, c.want, code, out, errs)
			}
		})
	}
}

func TestReceiptIOError(t *testing.T) {
	dir := newRun(t)
	if code, _, errs := run("receipt", dir, filepath.Join(dir, "missing.json")); code != 2 || !strings.HasPrefix(errs, "ERROR: ") {
		t.Fatalf("got %d %s", code, errs)
	}
	if code, _, _ := run("receipt", dir, "x.json", "--observed"); code != 2 {
		t.Fatalf("dangling --observed exit %d", code)
	}
	if code, _, _ := run("anchor"); code != 2 {
		t.Fatalf("usage exit %d", code)
	}
}

func TestAnchor(t *testing.T) {
	loc := func(path string, start, end int, quote string) J {
		return J{"path": path, "start": start, "end": end, "quote": quote}
	}
	cases := []struct {
		name     string
		findings L
		code     int
		want     string
	}{
		{"match", L{J{"id": "F-1", "locations": L{loc("src/a.py", 2, 2, "  return   request.user ")}}}, 0, `{"verdict": "ANCHORED", "reasons": []}`},
		{"multi-line match", L{J{"id": "F-1", "locations": L{loc("src/a.py", 1, 2, "(request): return request")}}}, 0, "ANCHORED"},
		{"wrong quote", L{J{"id": "F-1", "locations": L{loc("src/a.py", 2, 2, "return request.admin")}}}, 1, "F-1: quote not found at src/a.py:2-2"},
		{"lines outside file", L{J{"id": "F-1", "locations": L{loc("src/a.py", 2, 9, "return request.user")}}}, 1, "F-1: src/a.py lines 2-9 outside file (3 lines)"},
		{"path escape", L{J{"id": "F-1", "locations": L{loc("../secret.py", 1, 1, "top secret value")}}}, 1, "F-1: ../secret.py is not a file under the reviewed tree"},
		{"directory", L{J{"id": "F-1", "locations": L{loc("src", 1, 1, "whatever long")}}}, 1, "is not a file under"},
		{"too short", L{J{"id": "F-1", "locations": L{loc("src/a.py", 2, 2, "return")}}}, 1, "F-1: quote at src/a.py:2 is too short to anchor; quote the whole line"},
		{"short quote equal to window", L{J{"id": "F-1", "locations": L{loc("src/a.py", 3, 3, "x = 1")}}}, 0, "ANCHORED"},
		{"no locations", L{J{"id": "F-2"}}, 1, "F-2: no locations to anchor"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			tree := filepath.Join(dir, "tree")
			write(t, filepath.Join(tree, "src/a.py"), "def handler(request):\n    return request.user\nx = 1\n")
			write(t, filepath.Join(dir, "secret.py"), "top secret value\n")
			code, out, errs := run("anchor", tree, write(t, filepath.Join(dir, "p.json"), J{"findings": c.findings}))
			if code != c.code || !strings.Contains(out, c.want) {
				t.Fatalf("want %d with %q, got %d:\n%s%s", c.code, c.want, code, out, errs)
			}
		})
	}
}

func TestAnchorBadInput(t *testing.T) {
	dir := t.TempDir()
	for _, doc := range []any{J{"findings": L{}}, L{}, J{"items": L{J{"id": "F-1"}}}} {
		code, _, errs := run("anchor", dir, write(t, filepath.Join(dir, "p.json"), doc))
		if code != 2 || !strings.Contains(errs, `expected {"findings": [...]} with at least one finding`) {
			t.Fatalf("%v: got %d %s", doc, code, errs)
		}
	}
}

func TestSplitlines(t *testing.T) {
	got := splitlines("a\r\nb\rc\n\nd\u2028e")
	want := []string{"a", "b", "c", "", "d", "e"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("splitlines = %q", got)
	}
	if pyJSON("é\"<\x01") != `"\u00e9\"<\u0001"` {
		t.Fatalf("pyJSON = %s", pyJSON("é\"<\x01"))
	}
}

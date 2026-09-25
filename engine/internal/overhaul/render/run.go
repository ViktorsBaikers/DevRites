// Package render implements the /overhaul render tool: offline review and
// report views (HTML + Markdown) projected from one generation's canonical
// JSON. Every repository-derived string is escaped; pages carry no scripts, no
// remote assets and a hash-pinned stylesheet under a restrictive CSP.
package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

const usage = "usage: devrites-engine overhaul render <run> <gen-dir> [review|report]\n"

// css is byte-identical to the Python recipe so the CSP hash matches.
const css = `:root{color-scheme:light dark;--fg:#1b1f24;--bg:#fff;--muted:#57606a;--line:#d0d7de;--bad:#a40e26;--ok:#1a7f37}
@media (prefers-color-scheme:dark){:root{--fg:#e6edf3;--bg:#0d1117;--muted:#9da7b3;--line:#3d444d;--bad:#ff7b72;--ok:#56d364}}
body{margin:0 auto;max-width:72rem;padding:1rem;font:16px/1.5 system-ui,sans-serif;color:var(--fg);background:var(--bg)}
*,*::before,*::after{box-sizing:border-box}main *{min-width:0}img,svg{max-width:100%}
a{color:inherit}a:focus-visible,textarea:focus-visible,.scroll:focus-visible{outline:3px solid #0969da;outline-offset:2px}
.skip{position:absolute;left:-9999px}.skip:focus{left:1rem;top:1rem;background:var(--bg);padding:.5rem}
nav ul{display:flex;flex-wrap:wrap;gap:.25rem 1rem;padding:0;list-style:none}
.scroll{overflow-x:auto}table{border-collapse:collapse;width:100%;margin:.5rem 0 1rem}
th,td{border:1px solid var(--line);padding:.3rem .5rem;text-align:left;vertical-align:top;overflow-wrap:anywhere}
caption{text-align:left;font-weight:600}code,pre,textarea{font:13px/1.4 ui-monospace,monospace}code{overflow-wrap:anywhere}
.muted{color:var(--muted)}.FAIL,.NOT_READY,.UNKNOWN,.NOT_ASSESSABLE,.BLOCKED{color:var(--bad);font-weight:600}
.PASS,.MEETS_TARGETS{color:var(--ok);font-weight:600}textarea{width:100%;min-height:5rem}
@media print{nav,.skip{display:none}}`

var (
	rowID    = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[A-Za-z0-9._-]+$`)
	unsafeID = regexp.MustCompile(`[^A-Za-z0-9_-]`)
	flags    = set("PASS", "FAIL", "UNKNOWN", "NOT_READY", "MEETS_TARGETS", "NOT_ASSESSABLE", "BLOCKED")
	leads    = set("candidate", "needs-validation")
	reviewed = set("semantically-reviewed", "cross-boundary-reviewed", "verified")
	// esc matches Python html.escape(quote=True) byte for byte.
	esc   = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#x27;").Replace
	mdEsc = strings.NewReplacer(`\`, `\\`, "|", `\|`, "\n", " ", "<", "&lt;").Replace
)

func set(xs ...string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

type row []any

type section struct {
	anchor, heading string
	cols            []string
	body            []row
}

// Run executes `render <run> <gen-dir> [review|report]` and prints the HTML path.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 || len(args) > 3 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	kind := "review"
	if len(args) == 3 {
		kind = args[2]
	}
	if kind != "review" && kind != "report" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	home, _ := os.UserHomeDir()
	path, err := render(args[0], args[1], kind, home)
	if err != nil {
		fmt.Fprintf(stderr, "overhaul render: %v\n", err)
		return 2
	}
	fmt.Fprintln(stdout, path)
	return 0
}

type view struct{ home string }

func (v view) redact(s string) string {
	if v.home == "" || v.home == "/" {
		return s
	}
	return strings.ReplaceAll(s, v.home, "~")
}

func empty(x any) bool {
	switch t := x.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	}
	return false
}

// e renders one HTML cell value.
func (v view) e(x any) string {
	if empty(x) {
		return `<span class="muted">not recorded</span>`
	}
	switch t := x.(type) {
	case []any:
		parts := make([]string, len(t))
		for i, y := range t {
			parts[i] = v.e(y)
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		return esc(v.redact(pyJSON(t)))
	}
	s := esc(v.redact(pyStr(x)))
	if flags[s] {
		return `<span class="` + s + `">` + s + `</span>`
	}
	return s
}

// m renders one Markdown table cell value.
func (v view) m(x any) string {
	if empty(x) {
		return "not recorded"
	}
	if t, ok := x.([]any); ok {
		parts := make([]string, len(t))
		for i, y := range t {
			parts[i] = v.m(y)
		}
		return strings.Join(parts, ", ")
	}
	return mdEsc(v.redact(pyStr(x)))
}

// pyStr mirrors Python str() of a decoded JSON value. Numbers keep their JSON
// source text; lists and objects use json.dumps(sort_keys=True) form.
func pyStr(x any) string {
	switch t := x.(type) {
	case nil:
		return "None"
	case bool:
		if t {
			return "True"
		}
		return "False"
	case string:
		return t
	case json.Number:
		return t.String()
	}
	return pyJSON(x)
}

// pyJSON mirrors Python json.dumps(v, sort_keys=True) (ASCII-only output).
func pyJSON(x any) string {
	var b strings.Builder
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case nil:
			b.WriteString("null")
		case bool:
			fmt.Fprint(&b, t)
		case json.Number:
			b.WriteString(t.String())
		case string:
			b.WriteByte('"')
			for _, r := range t {
				switch {
				case r == '"' || r == '\\':
					b.WriteByte('\\')
					b.WriteRune(r)
				case r == '\n':
					b.WriteString(`\n`)
				case r == '\r':
					b.WriteString(`\r`)
				case r == '\t':
					b.WriteString(`\t`)
				case r == '\b':
					b.WriteString(`\b`)
				case r == '\f':
					b.WriteString(`\f`)
				case r >= 0x20 && r <= 0x7e:
					b.WriteRune(r)
				case r > 0xffff:
					r1, r2 := utf16.EncodeRune(r)
					fmt.Fprintf(&b, `\u%04x\u%04x`, r1, r2)
				default:
					fmt.Fprintf(&b, `\u%04x`, r)
				}
			}
			b.WriteByte('"')
		case []any:
			b.WriteByte('[')
			for i, y := range t {
				if i > 0 {
					b.WriteString(", ")
				}
				walk(y)
			}
			b.WriteByte(']')
		case map[string]any:
			b.WriteByte('{')
			for i, k := range sortedKeys(t) {
				if i > 0 {
					b.WriteString(", ")
				}
				walk(k)
				b.WriteString(": ")
				walk(t[k])
			}
			b.WriteByte('}')
		default:
			fmt.Fprint(&b, t)
		}
	}
	walk(x)
	return b.String()
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// load reads a JSON object; a missing file is an empty object.
func load(path string) (map[string]any, error) {
	m, err := ovio.LoadObject(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	return m, err
}

// objectKeys returns the keys of the top-level object field name in document
// order: Python iterates rubric domains in JSON order, Go maps do not.
func objectKeys(path, name string) []string {
	b, err := os.ReadFile(path) // #nosec G304 -- record file inside the operator-selected generation
	if err != nil {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err := dec.Token(); err != nil {
		return nil
	}
	var skip json.RawMessage
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return nil
		}
		if k != name {
			if dec.Decode(&skip) != nil {
				return nil
			}
			continue
		}
		if d, _ := dec.Token(); d != json.Delim('{') {
			return nil
		}
		var keys []string
		seen := map[string]bool{}
		for dec.More() {
			k, err := dec.Token()
			if err != nil || dec.Decode(&skip) != nil {
				return keys
			}
			if s := k.(string); !seen[s] {
				seen[s] = true
				keys = append(keys, s)
			}
		}
		return keys
	}
	return nil
}

func objs(v any) []map[string]any {
	l := ovio.List(v)
	out := make([]map[string]any, len(l))
	for i, x := range l {
		out[i] = ovio.Obj(x)
	}
	return out
}

func strs(xs []string) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}

// sf formats with every argument converted through pyStr, like Python "%s".
func sf(format string, xs ...any) string {
	for i, x := range xs {
		xs[i] = pyStr(x)
	}
	return fmt.Sprintf(format, xs...)
}

func has(list any, s string) bool {
	for _, x := range ovio.List(list) {
		if y, ok := x.(string); ok && y == s {
			return true
		}
	}
	return false
}

func rowsOf(items []map[string]any, fields ...string) []row {
	out := make([]row, 0, len(items))
	for _, x := range items {
		r := make(row, len(fields))
		for i, f := range fields {
			r[i] = ovio.Get(x, f)
		}
		out = append(out, r)
	}
	return out
}

func filter(xs []map[string]any, keep func(map[string]any) bool) []map[string]any {
	var out []map[string]any
	for _, x := range xs {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}

func pick(card map[string]any, scope string) any {
	for _, r := range objs(card["thresholds"]) {
		if s, ok := r["scope"].(string); ok && s == scope {
			return ovio.Get(r, "value.exact")
		}
	}
	return nil
}

func scorecardRows(rubric map[string]any, domains []string, base, sc map[string]any) ([]row, error) {
	live := filter(objs(rubric["controls"]), func(c map[string]any) bool { return c != nil && c["na"] == nil })
	dw, lanes, st := ovio.Obj(rubric["domains"]), ovio.Strings(rubric["lanes"]), ovio.Obj(sc["status_by_control"])
	type scope struct{ name, lane, dom string }
	scopes := []scope{{"global", "", ""}}
	for _, d := range domains {
		scopes = append(scopes, scope{"domain " + d, "", d})
	}
	for _, l := range lanes {
		scopes = append(scopes, scope{"lane " + l, l, ""})
	}
	for _, l := range lanes {
		for _, d := range domains {
			scopes = append(scopes, scope{"lane " + l + " / " + d, l, d})
		}
	}
	var out []row
	for _, s := range scopes {
		cs := filter(live, func(c map[string]any) bool {
			return (s.dom == "" || c["domain"] == any(s.dom)) && (s.lane == "" || has(c["lanes"], s.lane))
		})
		b, c := pick(base, s.name), pick(sc, s.name)
		if b == nil && c == nil {
			continue
		}
		sum := new(big.Rat)
		gaps, blockers := []any{}, []any{}
		for _, ctl := range cs {
			w, ok := ctl["weight"]
			if !ok {
				w = json.Number("0")
			}
			r, ok := new(big.Rat).SetString(pyStr(w))
			if !ok {
				return nil, fmt.Errorf("control %s: invalid weight %s", pyStr(ctl["id"]), pyStr(w))
			}
			sum.Add(sum, r)
			status := ovio.Str(st[ovio.Str(ctl["id"])])
			if status != "PASS" && status != "FAIL" {
				gaps = append(gaps, ctl["id"])
			}
			if ovio.Truthy(ctl["hard_gate"]) && status != "PASS" {
				blockers = append(blockers, ctl["id"])
			}
		}
		var weight any
		if s.dom != "" {
			weight = dw[s.dom]
		}
		out = append(out, row{s.name, weight, sum.RatString(), b, c, gaps, blockers})
	}
	return out, nil
}

type built struct {
	rj             map[string]any
	secs           []section
	approve, stamp string
}

func build(run, gen string) (*built, error) {
	R := map[string]map[string]any{}
	for _, n := range []string{"run", "coverage", "stack-profiles", "dispatch", "contracts", "findings", "evidence",
		"approval", "scorecard", "scorecard-baseline", "results-baseline", "results-candidate", "cycles"} {
		m, err := load(filepath.Join(gen, n+".json"))
		if err != nil {
			return nil, err
		}
		R[n] = m
	}
	rj := R["run"]
	pr := ovio.Obj(ovio.Get(rj, "revisions.plan"))
	plan, digest := map[string]any{}, "none"
	if len(pr) > 0 {
		p := filepath.Join(run, ovio.Str(pr["file"]))
		var err error
		if plan, err = load(p); err != nil {
			return nil, err
		}
		if digest, err = ovio.SHA256File(p); err != nil {
			return nil, err
		}
	}
	rfile := ovio.Get(plan, "rubric.file")
	if !ovio.Truthy(rfile) {
		rfile = ovio.Get(rj, "revisions.rubric.file")
	}
	rubric, domains := map[string]any{}, []string(nil)
	if ovio.Truthy(rfile) {
		p := filepath.Join(run, pyStr(rfile))
		var err error
		if rubric, err = load(p); err != nil {
			return nil, err
		}
		domains = objectKeys(p, "domains")
	}
	finds, att := objs(R["findings"]["findings"]), objs(R["dispatch"]["attempts"])
	sc, base := R["scorecard"], R["scorecard-baseline"]
	cov := objs(R["coverage"]["files"])
	tasks := objs(plan["tasks"])

	laneSet := set("frontend", "backend", "boundary")
	for _, x := range att {
		if l := ovio.Str(x["lane"]); l != "" {
			laneSet[l] = true
		}
	}
	lanes := make([]string, 0, len(laneSet))
	for l := range laneSet {
		lanes = append(lanes, l)
	}
	sort.Strings(lanes)
	admitted := func(x map[string]any) bool { return ovio.Str(x["status"]) == "admitted" }
	changed := map[string]any{}
	for _, x := range att {
		if ovio.Str(x["role"]) == "implementer" && admitted(x) {
			changed[pyStr(x["task_id"])] = x["changed_paths"]
		}
	}
	byF := map[string][]map[string]any{}
	for _, t := range tasks {
		for _, fid := range ovio.List(t["findings"]) {
			byF[pyStr(fid)] = append(byF[pyStr(fid)], t)
		}
	}

	var laneRows []row
	for _, lane := range lanes {
		xs := filter(att, func(x map[string]any) bool { return ovio.Str(x["lane"]) == lane && admitted(x) })
		var windows, open []string
		profSet := map[string]bool{}
		for _, x := range xs {
			windows = append(windows, sf("%s %s to %s", x["role"], x["started_at"], x["finished_at"]))
			for _, p := range ovio.List(x["profiles"]) {
				profSet[pyStr(p)] = true
			}
		}
		var profs []string
		for p := range profSet {
			profs = append(profs, p)
		}
		sort.Strings(profs)
		done, total := 0, 0
		for _, c := range cov {
			if !ovio.Truthy(c["eligible"]) {
				continue
			}
			for _, r := range objs(c["ranges"]) {
				ls := r["lanes"]
				if !ovio.Truthy(ls) {
					ls = c["lanes"]
				}
				if !has(ls, lane) {
					continue
				}
				total++
				if reviewed[ovio.Str(r["state"])] {
					done++
				} else {
					open = append(open, sf("%s:%s-%s %s", c["path"], r["start"], r["end"], r["state"]))
				}
			}
		}
		laneScore := ovio.Get(ovio.Obj(ovio.Obj(sc["lanes"])[lane]), "Q.exact")
		laneRows = append(laneRows, row{lane, strs(windows), strs(profs), fmt.Sprintf("%d / %d", done, total), strs(open), laneScore})
	}

	var result []row
	for _, k := range []string{"run_id", "mode", "assessment_only", "phase", "readiness_verdict", "execution_outcome",
		"identities.comparison", "identities.baseline", "identities.candidate", "capabilities", "degradation", "budget", "stop"} {
		result = append(result, row{k, ovio.Get(rj, k)})
	}
	S := []section{
		{"result", "Scope, identities and result", []string{"field", "value"}, result},
		{"lane-coverage", "Lane, stack and boundary coverage", []string{"lane", "admitted reviewers and windows", "profiles",
			"reviewed / eligible ranges", "open ranges", "lane score"}, laneRows},
	}
	for _, lane := range lanes {
		var body []row
		for _, x := range att {
			if ovio.Str(x["lane"]) == lane {
				body = append(body, row{sf("%s#%s", x["task_id"], x["attempt_id"]), x["role"], x["status"], x["profiles"],
					x["started_at"], x["finished_at"], x["receipt"]})
			}
		}
		S = append(S, section{"lane-" + unsafeID.ReplaceAllString(lane, "-"), "Lane: " + lane,
			[]string{"attempt", "role", "status", "profiles", "started", "finished", "receipt"}, body})
	}

	var covRows []row
	for _, c := range cov {
		var rs []string
		for _, r := range objs(c["ranges"]) {
			rs = append(rs, sf("%s-%s %s", r["start"], r["end"], r["state"]))
		}
		covRows = append(covRows, row{c["path"], c["eligible"], c["lanes"], strs(rs), ovio.Get(c, "exclusion.reason")})
	}
	notLead := func(f map[string]any) bool { s := ovio.Str(f["status"]); return !leads[s] && s != "rejected" }
	opp := func(f map[string]any) bool { return ovio.Str(f["kind"]) == "opportunity" }
	var repairs, rejected, planRows, controls, changes []row
	for _, f := range finds {
		for _, t := range byF[pyStr(f["id"])] {
			repairs = append(repairs, row{sf("%s / %s", f["id"], t["id"]), []any{f["severity"], f["confidence"]},
				f["failing_scenario"], changed[pyStr(t["id"])], t["oracle"], []any{f["status"], f["verified_by"]}, t["risk"]})
		}
		if ovio.Str(f["status"]) == "rejected" {
			var reason any
			if h := ovio.List(f["history"]); len(h) > 0 {
				reason = ovio.Get(h[len(h)-1], "reason")
			}
			rejected = append(rejected, row{f["id"], f["title"], reason})
		}
	}
	bench := filter(objs(R["evidence"]["items"]), func(x map[string]any) bool { return ovio.Str(x["kind"]) == "benchmark" })
	var perfFields []string
	for _, k := range []string{"view", "scenario", "metric", "baseline_median", "candidate_median", "absolute_delta", "ratio",
		"ci95", "n", "target", "guardrails", "verdict"} {
		perfFields = append(perfFields, "measurement."+k)
	}
	for _, t := range tasks {
		planRows = append(planRows, row{t["id"], t["lane"], t["findings"], t["deps"], t["allowed_paths"], t["oracle"],
			sf("%s / %s", t["benefit"], t["cost"]), t["rollback"], t["risk"], sf("%s / %s", t["writer_role"], t["verifier_role"])})
	}
	for _, c := range objs(rubric["controls"]) {
		id := ovio.Str(c["id"])
		controls = append(controls, row{c["id"], c["domain"], c["lanes"], c["weight"], c["hard_gate"],
			ovio.Obj(base["status_by_control"])[id], ovio.Obj(sc["status_by_control"])[id]})
	}
	var scoreRows []row
	for _, r := range objs(sc["thresholds"]) {
		scoreRows = append(scoreRows, row{r["scope"], ovio.Get(r, "value.exact"), ovio.Get(r, "value.decimal"), r["min"], r["result"]})
	}
	gates := ovio.Obj(sc["gates"])
	for _, g := range sortedKeys(gates) {
		scoreRows = append(scoreRows, row{g, nil, nil, nil, gates[g]})
	}
	card, err := scorecardRows(rubric, domains, base, sc)
	if err != nil {
		return nil, err
	}
	for _, x := range filter(att, admitted) {
		changes = append(changes, row{sf("%s#%s", x["task_id"], x["attempt_id"]), x["task_id"], x["changed_paths"]})
	}
	orNone := func(v any) string {
		if ovio.Truthy(v) {
			return pyStr(v)
		}
		return "none"
	}
	verdict := "not computed"
	if v, ok := sc["readiness_verdict"]; ok {
		verdict = pyStr(v)
	}
	S = append(S,
		section{"coverage", "Coverage ledger", []string{"path", "eligible", "lanes", "ranges (state)", "exclusion"}, covRows},
		section{"profiles", "Stack profiles", []string{"id", "revision", "status", "file", "sha256"},
			rowsOf(objs(R["stack-profiles"]["profiles"]), "id", "rev", "status", "file", "sha256")},
		section{"contracts", "Contracts and boundaries", []string{"id", "producer", "consumers", "version", "evidence", "open questions"},
			rowsOf(objs(R["contracts"]["contracts"]), "id", "producer", "consumers", "version", "evidence", "questions")},
		section{"findings", "Confirmed findings", []string{"id", "title", "kind", "severity", "status", "lanes", "locations", "failing case", "impact", "evidence"},
			rowsOf(filter(finds, func(f map[string]any) bool { return notLead(f) && !opp(f) }), "id", "title", "kind",
				"severity", "status", "lanes", "locations", "failing_scenario", "impact", "evidence")},
		section{"opportunities", "Opportunities (not defects)", []string{"id", "title", "status", "locations", "benefit"},
			rowsOf(filter(finds, func(f map[string]any) bool { return opp(f) && notLead(f) }), "id", "title", "status", "locations", "impact")},
		section{"repairs", "Findings and repairs", []string{"finding / task", "severity and confidence", "previous failure", "change made",
			"regression proof", "independent result", "remaining risk"}, repairs},
		section{"leads", "Unvalidated leads (no severity)", []string{"id", "title", "status", "potential impact", "missing fact"},
			rowsOf(filter(finds, func(f map[string]any) bool { return leads[ovio.Str(f["status"])] }), "id", "title", "status", "potential_impact", "blockers")},
		section{"rejected", "Rejected candidates", []string{"id", "title", "reason"}, rejected},
		section{"performance", "Performance before/after", []string{"view", "scenario", "metric", "before", "after", "delta", "ratio",
			"CI 95%", "n", "target", "guardrails", "verdict"}, rowsOf(bench, perfFields...)},
		section{"plan", "Repair plan revision " + orNone(plan["rev"]), []string{"task", "lane", "findings", "deps", "allowed paths",
			"oracle", "benefit / cost", "rollback", "risk", "writer / verifier"}, planRows},
		section{"controls", "Rubric controls (revision " + orNone(rubric["rev"]) + ")", []string{"control", "domain", "lanes", "weight",
			"hard gate", "baseline", "candidate"}, controls},
		section{"score", "Scorecard (verdict " + verdict + ")", []string{"scope", "exact", "decimal", "minimum", "result"}, scoreRows},
		section{"scorecard", "Transparent scorecard (baseline vs candidate)", []string{"scope", "domain weight",
			"applicable control weight", "baseline", "candidate", "evidence gaps", "hard blockers"}, card},
		section{"approvals", "Approvals and decisions", []string{"id", "kind", "status", "plan revision", "tasks", "channel", "quote"},
			rowsOf(objs(R["approval"]["events"]), "id", "kind", "status", "plan_rev", "tasks", "channel", "quote")},
		section{"cycles", "Iteration history", []string{"cycle", "deficits", "hypothesis", "agents", "changes", "outcome", "scores", "next / stop"},
			rowsOf(objs(R["cycles"]["cycles"]), "id", "deficits", "hypothesis", "agents", "changes", "outcome", "scores", "next")},
		section{"dependencies", "Dependency decisions", []string{"dependency", "before", "after", "decision", "evidence date", "reason", "proof", "risk"},
			rowsOf(objs(R["stack-profiles"]["dependencies"]), "name", "before", "after", "decision", "evidence_date", "reason", "proof", "risk")},
		section{"changes", "Changed paths", []string{"attempt", "task", "paths"}, changes},
		section{"questions", "Open questions and remaining work", []string{"id", "question", "blocks", "status"},
			rowsOf(objs(rj["questions"]), "id", "text", "blocks", "status")},
	)

	approve := ""
	if len(pr) > 0 && !ovio.Truthy(rj["assessment_only"]) {
		ids := make([]string, len(tasks))
		for i, t := range tasks {
			if id, ok := t["id"]; ok {
				ids[i] = pyStr(id)
			}
		}
		approve = sf("Approve overhaul %s plan revision %s, digest %s, tasks %s. Keep all listed constraints and exclusions.",
			rj["run_id"], pr["rev"], digest, strings.Join(ids, " "))
	}
	planStamp := "none"
	if len(pr) > 0 {
		planStamp = sf("r%s:%s", pr["rev"], digest)
	}
	stamp := sf("overhaul-stamp run=%s generation=%s plan=%s", rj["run_id"],
		strings.ReplaceAll(filepath.Base(strings.TrimRight(gen, "/")), ".tmp", ""), planStamp)
	return &built{rj, S, approve, stamp}, nil
}

func mdTable(v view, cols []string, body []row) []string {
	out := []string{"| " + strings.Join(cols, " | ") + " |", "|" + strings.Repeat("---|", len(cols))}
	for _, r := range body {
		cells := make([]string, len(r))
		for i, x := range r {
			cells[i] = v.m(x)
		}
		out = append(out, "| "+strings.Join(cells, " | ")+" |")
	}
	return out
}

func render(run, gen, kind, home string) (string, error) {
	b, err := build(run, gen)
	if err != nil {
		return "", err
	}
	v := view{home}
	vd := filepath.Join(gen, "views")
	if err := os.MkdirAll(vd, 0o755); err != nil {
		return "", err
	}
	runID := pyStr(b.rj["run_id"])
	if kind == "review" {
		for _, s := range b.secs {
			if s.anchor != "plan" || len(s.body) == 0 {
				continue
			}
			lines := append([]string{"<!-- " + b.stamp + " -->", "# Overhaul plan " + runID, ""}, mdTable(v, s.cols, s.body)...)
			if b.approve != "" {
				lines = append(lines, "", "```text", b.approve, "```")
			}
			if err := os.WriteFile(filepath.Join(vd, "plan.md"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
				return "", err
			}
		}
	}
	sum := sha256.Sum256([]byte(css))
	csp := "default-src 'none'; style-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) +
		"'; img-src data:; base-uri 'none'; form-action 'none'"
	title := "Overhaul " + kind + " " + runID
	var nav strings.Builder
	for _, s := range b.secs {
		fmt.Fprintf(&nav, `<li><a href="#%s">%s</a></li>`, esc(s.anchor), esc(s.heading))
	}
	out := []string{`<!doctype html><html lang="en"><head><meta charset="utf-8">`,
		`<meta http-equiv="Content-Security-Policy" content="` + csp + `">`,
		`<meta name="viewport" content="width=device-width,initial-scale=1">`,
		"<title>" + esc(title) + "</title><style>" + css + "</style></head><body>",
		// Escaped (unlike the Python recipe) so a hostile run id cannot close the comment.
		"<!-- " + esc(b.stamp) + " -->", `<a class="skip" href="#main">Skip to content</a>`,
		`<header><h1>` + esc(title) + `</h1><p class="muted"><code>` + esc(b.stamp) + `</code></p></header>`,
		`<nav aria-label="Sections"><ul>` + nav.String() + "</ul></nav>",
		`<main id="main">`}
	md := []string{"<!-- " + b.stamp + " -->", "# " + title, "", "`" + b.stamp + "`", ""}
	summary := sf("Outcome %s · verdict %s · phase %s", b.rj["execution_outcome"], b.rj["readiness_verdict"], b.rj["phase"])
	out = append(out, `<p role="status"><strong>`+esc(summary)+`</strong></p>`)
	md = append(md, "**"+v.m(summary)+"**", "")
	if kind == "review" && b.approve != "" {
		out = append(out, `<section id="decide"><h2>Decision</h2><p>Nothing on this page approves anything. Copy a message into the `+
			`conversation with the agent; only your message there counts. Edit the task list to approve a subset, or `+
			`reply with rejections, deferrals or questions per task ID.</p><label for="ap">Approval message</label>`+
			`<textarea id="ap" readonly>`+esc(b.approve)+`</textarea></section>`)
		md = append(md, "## Decision", "", "Only a message you send in the conversation counts. Approval text:", "", "```text", b.approve, "```", "")
	}
	for _, s := range b.secs {
		out = append(out, `<section id="`+esc(s.anchor)+`"><h2>`+esc(s.heading)+`</h2>`)
		md = append(md, "## "+v.m(s.heading), "")
		if len(s.body) == 0 {
			out = append(out, `<p class="muted">No records.</p></section>`)
			md = append(md, "No records.", "")
			continue
		}
		var th strings.Builder
		for _, c := range s.cols {
			th.WriteString(`<th scope="col">` + esc(c) + `</th>`)
		}
		out = append(out, `<div class="scroll" role="region" tabindex="0" aria-label="`+esc(s.heading)+`"><table><caption>`+
			esc(s.heading)+`</caption><thead><tr>`+th.String()+`</tr></thead><tbody>`)
		for _, r := range s.body {
			id := ""
			if first := pyStr(r[0]); rowID.MatchString(first) {
				id = ` id="` + esc(first) + `"`
			}
			var td strings.Builder
			for _, x := range r {
				td.WriteString("<td>" + v.e(x) + "</td>")
			}
			out = append(out, "<tr"+id+">"+td.String()+"</tr>")
		}
		out = append(out, "</tbody></table></div></section>")
		md = append(append(md, mdTable(v, s.cols, s.body)...), "")
	}
	out = append(out, "</main></body></html>")
	for ext, text := range map[string][]string{"html": out, "md": md} {
		if err := os.WriteFile(filepath.Join(vd, kind+"."+ext), []byte(strings.Join(text, "\n")+"\n"), 0o644); err != nil {
			return "", err
		}
	}
	return filepath.Join(vd, kind+".html"), nil
}

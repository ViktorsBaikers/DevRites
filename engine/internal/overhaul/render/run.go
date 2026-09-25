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

// css is the page's only stylesheet; the CSP pins it by sha256. Light is the
// base (and print) theme; screens that prefer dark get their own validated
// steps. Every colour is a token on :root and the dark block redefines each.
// Chart colours: severity is one red ramp (ordinal, validated light and dark);
// controls are blue pass / red fail / grey no-evidence.
const css = `:root{color-scheme:light;--bg:#f3f4f6;--surface:#ffffff;--raised:#eceef2;--fg:#15171c;--muted:#565e6b;--line:#d5d9e0;--line-2:#e9ebef;--accent:#1f5fbf;--ok:#11703a;--bad:#b3261e;--warn:#8a5a00;--track:#e6e8ec;--sev1:#8e1b1b;--sev2:#c2402f;--sev3:#e0704a;--sev4:#eea07c;--info:#a3aab5;--pass:#2a78d6;--fail:#c2402f;--unk:#b7bdc6;--s-critical:#8e1b1b;--s-high:#c2402f;--s-medium:#e0704a;--s-low:#eea07c;--s-informational:#a3aab5;--sans:system-ui,-apple-system,'Segoe UI',Roboto,sans-serif;--mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
@media screen and (prefers-color-scheme:dark){:root{color-scheme:dark;--bg:#111216;--surface:#1b1c20;--raised:#24262c;--fg:#eceef2;--muted:#a3a9b4;--line:#34373e;--line-2:#27292f;--accent:#7fb0ff;--ok:#5fd38a;--bad:#ff8a7a;--warn:#e9b44c;--track:#2b2e35;--sev1:#ff9a82;--sev2:#f26a4f;--sev3:#cc4a33;--sev4:#a33e2d;--info:#6b7280;--pass:#3987e5;--fail:#d9473a;--unk:#5a606b;--s-critical:#ff9a82;--s-high:#f26a4f;--s-medium:#cc4a33;--s-low:#a33e2d;--s-informational:#6b7280}}
*,*::before,*::after{box-sizing:border-box}main *{min-width:0}svg{max-width:100%}
html{scroll-padding-top:4.5rem;-webkit-text-size-adjust:100%}
@media (prefers-reduced-motion:no-preference){html{scroll-behavior:smooth}}
body{margin:0;background:var(--bg);color:var(--fg);font:15px/1.5 var(--sans)}
h1,h2,h3,h4{font-weight:600;line-height:1.25;margin:0;text-wrap:balance}
h2{font-size:1.25rem;margin:0 0 .25rem}h4{font-size:.8125rem;color:var(--muted);margin:0 0 .25rem}
p{margin:0 0 1rem}a{color:var(--accent);text-underline-offset:2px}
code,pre,textarea,.m{font-family:var(--mono);font-size:.8125rem}code{overflow-wrap:anywhere}
:focus-visible{outline:2px solid var(--accent);outline-offset:2px}
.muted{color:var(--muted)}.count{font:500 .75rem/1.5 var(--mono);color:var(--muted)}
.lede{color:var(--muted);margin:0 0 1rem;max-width:70ch}
.skip{position:absolute;left:-9999px;z-index:30}.skip:focus{left:1rem;top:.5rem;background:var(--surface);padding:.5rem 1rem;border-radius:8px}
.top{position:sticky;top:0;z-index:20;display:flex;align-items:center;gap:1rem;padding:.5rem 1rem;background:var(--surface);border-bottom:1px solid var(--line)}
.top .brand{flex:0 1 auto;min-width:0;overflow:hidden;text-overflow:ellipsis;font-weight:600;white-space:nowrap}
.top code{flex:1 1 0;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--muted);font-size:.75rem}
.shell{max-width:84rem;margin:0 auto;padding:1.5rem 1rem 4rem}
.toc ul{list-style:none;margin:0 0 1.5rem;padding:0 0 .5rem;display:flex;gap:.5rem;overflow-x:auto}
.toc a{display:block;white-space:nowrap;padding:.25rem .75rem;border:1px solid var(--line);border-radius:999px;text-decoration:none;color:var(--fg);font-size:.875rem}
.toc a:hover{background:var(--raised)}
@media (min-width:64rem){.shell{display:grid;grid-template-columns:12rem minmax(0,1fr);gap:2.5rem}.toc{position:sticky;top:4.5rem;align-self:start}.toc ul{flex-direction:column;gap:0;overflow:visible;margin:0}.toc a{white-space:normal;border:0;border-radius:8px;padding:.3rem .5rem}}
main>section,.records>section{margin:0 0 1.5rem;padding:1.5rem;background:var(--surface);border:1px solid var(--line);border-radius:12px}
.hero{margin:0 0 1.5rem;padding:1.5rem;background:var(--surface);border:1px solid var(--line);border-radius:12px}
.hero>p[role=status]{font-size:.8125rem;color:var(--muted);margin:0 0 1rem}.hero>p[role=status] strong{font-weight:500}
.answer{display:grid;gap:2rem;grid-template-columns:minmax(0,1fr)}
@media (min-width:56rem){.answer{grid-template-columns:minmax(0,3fr) minmax(0,2fr)}}
.verdict{font-size:1.5rem}.verdict.t-bad{color:var(--bad)}.verdict.t-ok{color:var(--ok)}.verdict.t-warn{color:var(--warn)}
.hero-num{font-size:3.5rem;font-weight:650;line-height:1.05;margin:.25rem 0}.hero-num span{font-size:1.25rem;font-weight:500;color:var(--muted)}
.hero .score>.bar{height:18px;margin:.5rem 0 1.25rem}
svg.bar{display:block;width:100%;height:12px;overflow:visible}
.trk{fill:var(--track)}.tgt{fill:var(--fg)}.was{fill:var(--muted)}
.g0,.c-pass,.c-part{fill:var(--pass)}.g4{fill:var(--sev4)}.g3{fill:var(--sev3)}.g2{fill:var(--sev2)}.g1{fill:var(--sev1)}.c-fail{fill:var(--fail)}.c-unk{fill:var(--unk)}
.s-critical{fill:var(--sev1)}.s-high{fill:var(--sev2)}.s-medium{fill:var(--sev3)}.s-low{fill:var(--sev4)}.s-informational,.s-none{fill:var(--info)}
.rows{list-style:none;margin:0;padding:0;display:grid;gap:.75rem}
.rows li{display:grid;grid-template-columns:minmax(0,1fr) auto;grid-template-areas:"l v" "b b";gap:.25rem .75rem;align-items:center}
.rows li:has(.wide){grid-template-columns:minmax(0,1fr);grid-template-areas:"l" "b" "v"}.rows li:has(.wide) .val{text-align:left}
.rows .lbl{grid-area:l;overflow-wrap:anywhere}.rows .bar{grid-area:b}.rows .val{grid-area:v;text-align:right;font-variant-numeric:tabular-nums;color:var(--muted);font-size:.875rem}
@media (min-width:48rem){.rows li{grid-template-columns:minmax(0,13rem) minmax(0,1fr) 3.5rem;grid-template-areas:"l b v"}.rows li:has(.wide){grid-template-columns:minmax(0,11rem) minmax(0,1fr) 17rem;grid-template-areas:"l b v"}.rows .val{text-align:left}}
.lanes{margin:0 0 .5rem}
.legend{list-style:none;display:flex;flex-wrap:wrap;gap:.25rem 1.25rem;margin:1rem 0 0;padding:0;font-size:.8125rem;color:var(--muted)}
.legend li{display:flex;align-items:center;gap:.4rem}.legend strong{color:var(--fg)}
.key{display:inline-block;width:.7rem;height:.7rem;border-radius:2px;background:var(--c,var(--muted));flex:none}
.key.g0,.key.c-pass,.key.c-part{--c:var(--pass)}.key.g4,.key.s-low{--c:var(--sev4)}.key.g3,.key.s-medium{--c:var(--sev3)}.key.g2,.key.s-high{--c:var(--sev2)}.key.g1,.key.s-critical{--c:var(--sev1)}.key.c-fail{--c:var(--fail)}.key.c-unk{--c:var(--unk)}.key.s-informational,.key.s-none{--c:var(--info)}.key.was{--c:var(--muted);width:3px}.key.tgt{--c:var(--fg);width:2px;height:.9rem;border-radius:0}
.blockers h2{font-size:1rem;margin:0 0 .75rem}
.gates{list-style:none;margin:0 0 .75rem;padding:0;display:grid;gap:.6rem}
.gates li{display:flex;gap:.6rem;align-items:flex-start}.gates code{color:var(--muted);font-size:.75rem}
.mark{flex:none;display:inline-grid;place-items:center;width:1.4rem;height:1.4rem;border-radius:999px;font-size:.75rem;font-weight:700;color:var(--surface);background:var(--bad)}
.t-warn .mark{background:var(--warn)}
.strip{margin:1.75rem 0 0;padding-top:1.25rem;border-top:1px solid var(--line-2)}.strip h2{font-size:1rem;margin:0 0 .5rem}.strip .bar{height:16px}
.next{margin:1.25rem 0 0;font-weight:600}
#gaps table{min-width:36rem}#gaps th[scope=row]{font-weight:500;white-space:nowrap;background:var(--surface)}
#gaps td.cell{min-width:8rem}#gaps .val{display:block;font-variant-numeric:tabular-nums;font-weight:600;margin:0 0 .25rem}
#gaps .cell.g1 .val,#gaps .cell.g2 .val{color:var(--bad)}#gaps .cell.g0 .val{color:var(--ok)}#gaps .cell.g0 .val::after{content:" ✓"}
.pill{display:inline-block;padding:0 .5rem;border:1px solid var(--line);border-radius:999px;font:600 .75rem/1.6 var(--sans);white-space:nowrap;color:var(--muted)}
.pill.t-bad{color:var(--bad);border-color:currentColor}.pill.t-ok{color:var(--ok);border-color:currentColor}.pill.t-warn{color:var(--warn);border-color:currentColor}
.filter{display:flex;flex-wrap:wrap;align-items:center;gap:.5rem;border:0;margin:0 0 1rem;padding:0}
.filter legend{float:left;margin-right:.5rem;padding:0;font-size:.875rem;color:var(--muted)}
.filter input{position:absolute;opacity:0;width:1px;height:1px;margin:0}
.filter label{cursor:pointer;padding:.25rem .75rem;border:1px solid var(--line);border-radius:999px;font-size:.875rem}
.filter input:checked+label{background:var(--fg);color:var(--bg);border-color:var(--fg)}.filter input:checked+label .count{color:inherit}
.filter input:focus-visible+label{outline:2px solid var(--accent);outline-offset:2px}
@supports not selector(:has(a)){.filter{display:none}}
#findings:has(#sev-critical:checked) .finding:not(.s-critical),#findings:has(#sev-high:checked) .finding:not(.s-high),#findings:has(#sev-medium:checked) .finding:not(.s-medium),#findings:has(#sev-low:checked) .finding:not(.s-low),#findings:has(#sev-informational:checked) .finding:not(.s-informational){display:none}
.cards{border:1px solid var(--line);border-radius:10px;overflow:hidden}
.finding,.task{border-bottom:1px solid var(--line-2)}.finding:last-child,.task:last-child{border-bottom:0}.sub{font-size:1rem;margin:1.5rem 0 .5rem}
.finding>summary,.task>summary{list-style:none;cursor:pointer;display:grid;grid-template-columns:5.5rem 4.5rem minmax(0,1fr);gap:.25rem .75rem;align-items:baseline;padding:.6rem .9rem}.task>summary{grid-template-columns:4.5rem 6.5rem minmax(0,1fr) auto}.task>summary>.pill:nth-child(2){justify-self:start}
.finding>summary::-webkit-details-marker,.task>summary::-webkit-details-marker{display:none}.finding>summary:hover,.task>summary:hover{background:var(--raised)}
.finding .where{grid-column:3;color:var(--muted);font-size:.75rem}
@media (min-width:60rem){.finding>summary{grid-template-columns:5.5rem 4.5rem minmax(0,1fr) minmax(0,16rem)}.finding .where{grid-column:auto;text-align:right;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}}
.sev{display:inline-flex;align-items:center;gap:.4rem;font-size:.8125rem;font-weight:600}
.fid{font-weight:600}.ftitle{overflow-wrap:anywhere}
.finding[open]>summary,.task[open]>summary{background:var(--raised)}.fbody{padding:.75rem .9rem 1rem;display:grid;gap:.9rem}
.kv{display:grid;grid-template-columns:minmax(0,9rem) minmax(0,1fr);gap:.25rem 1rem;margin:0}
.kv dt{color:var(--muted);font-size:.8125rem}.kv dd{margin:0;overflow-wrap:anywhere}
@media (max-width:40rem){.kv{grid-template-columns:minmax(0,1fr);gap:0}.kv dd{margin-bottom:.5rem}}
.hist{margin:0;padding-left:1.25rem}.hist li{margin:0 0 .25rem;overflow-wrap:anywhere}
.records{margin:2rem 0 0}.records>summary{cursor:pointer;display:inline-block;margin:0 0 1rem;padding:.5rem 1rem;border:1px solid var(--line);border-radius:999px;background:var(--surface);font-weight:600}
.scroll{overflow:auto;max-height:75vh;border:1px solid var(--line);border-radius:10px;background:var(--surface)}
table{border-collapse:separate;border-spacing:0;width:100%;font-size:.875rem}
.scroll:has(th:nth-child(6)) table{min-width:60rem}
caption{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%);white-space:nowrap}
th,td{padding:.5rem .75rem;text-align:left;vertical-align:top;border-bottom:1px solid var(--line-2);overflow-wrap:anywhere}
thead th{position:sticky;top:0;z-index:1;background:var(--raised);color:var(--muted);font-weight:600;font-size:.75rem;border-bottom:1px solid var(--line)}
tbody tr:last-child>*{border-bottom:0}
td.n{text-align:right;font-variant-numeric:tabular-nums;white-space:nowrap}td.m{font-size:.8125rem}
tr:target,.finding:target,.task:target{outline:2px solid var(--accent);outline-offset:-2px}
#decide{border:2px solid var(--accent)}
#decide label{display:block;font-weight:600;margin:0 0 .5rem}
textarea{display:block;width:100%;min-height:7rem;padding:.75rem;border:1px solid var(--line);border-radius:8px;background:var(--bg);color:var(--fg);resize:vertical}
@media print{.top,.toc,.skip,.filter{display:none}.shell{display:block;padding:0}.scroll{max-height:none;overflow:visible}thead th{position:static}.scroll:has(th:nth-child(6)) table{min-width:0}.finding,tr,.rows li{break-inside:avoid}body{font-size:10pt}}
@media print{details::details-content{content-visibility:visible}}`

var (
	rowID    = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[A-Za-z0-9._-]+$`)
	unsafeID = regexp.MustCompile(`[^A-Za-z0-9_-]`)
	flags    = set("PASS", "FAIL", "UNKNOWN", "NOT_READY", "MEETS_TARGETS", "NOT_ASSESSABLE", "BLOCKED")
	leads    = set("candidate", "needs-validation")
	reviewed = set("semantically-reviewed", "cross-boundary-reviewed", "verified")
	// esc matches Python html.escape(quote=True) byte for byte.
	esc   = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#x27;").Replace
	mdEsc = strings.NewReplacer(`\`, `\\`, "|", `\|`, "\n", " ", "<", "&lt;").Replace

	// HTML-only presentation: severity scale, status order, typed table cells.
	sevOrder = []string{"critical", "high", "medium", "low", "informational"}
	sevLabel = map[string]string{"critical": "Critical", "high": "High", "medium": "Medium", "low": "Low", "informational": "Info", "none": "Unrated"}
	monoCols = set("field", "id", "path", "paths", "attempt", "task", "control", "file", "sha256", "receipt", "allowed paths",
		"finding / task", "oracle", "regression proof", "change made", "open ranges", "ranges (state)")
	numCell = regexp.MustCompile(`^-?\d+(\.\d+)?( ?/ ?\d+(\.\d+)?)?$`)
	goodTok = set("PASS", "MEETS_TARGETS", "MEETS_TARGETS_WITH_APPROVED_DEGRADATION")
	warnTok = set("UNKNOWN", "AWAITING_APPROVAL", "WAIVED_OPERATIONAL")
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

// txt renders a scalar as plain escaped text (no flag span), for pills and KPIs.
func (v view) txt(x any) string {
	if empty(x) {
		return "not recorded"
	}
	return esc(v.redact(pyStr(x)))
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
	rj, sc, base   map[string]any
	rubric         map[string]any
	domains        []string
	tasks          []map[string]any
	laneCov        map[string][2]int // lane -> reviewed, eligible ranges
	secs           []section
	approve, stamp string
	finds          []map[string]any
	lanes          []string
	cards          map[string][]map[string]any // section anchor -> findings drawn as cards in HTML
	byF            map[string][]map[string]any // finding id -> plan tasks
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
	laneCov := map[string][2]int{}
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
		laneCov[lane] = [2]int{done, total}
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
	confirmed := filter(finds, func(f map[string]any) bool { return notLead(f) && !opp(f) })
	opps := filter(finds, func(f map[string]any) bool { return opp(f) && notLead(f) })
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
			rowsOf(confirmed, "id", "title", "kind", "severity", "status", "lanes", "locations", "failing_scenario", "impact", "evidence")},
		section{"opportunities", "Opportunities (not defects)", []string{"id", "title", "status", "locations", "benefit"},
			rowsOf(opps, "id", "title", "status", "locations", "impact")},
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
	return &built{rj: rj, sc: sc, base: base, rubric: rubric, domains: domains, tasks: tasks, laneCov: laneCov, secs: S, approve: approve, stamp: stamp, finds: finds, lanes: lanes,
		cards: map[string][]map[string]any{"findings": confirmed, "opportunities": opps}, byF: byF}, nil
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
	summary := sf("Outcome %s · verdict %s · phase %s", b.rj["execution_outcome"], b.rj["readiness_verdict"], b.rj["phase"])
	decide := kind == "review" && b.approve != ""
	md := []string{"<!-- " + b.stamp + " -->", "# " + title, "", "`" + b.stamp + "`", ""}
	md = append(md, "**"+v.m(summary)+"**", "")
	if decide {
		md = append(md, "## Decision", "", "Only a message you send in the conversation counts. Approval text:", "", "```text", b.approve, "```", "")
	}
	for _, s := range b.secs {
		md = append(md, "## "+v.m(s.heading), "")
		if len(s.body) == 0 {
			md = append(md, "No records.", "")
			continue
		}
		md = append(append(md, mdTable(v, s.cols, s.body)...), "")
	}
	out := v.page(b, kind, title, summary, csp, decide)
	for ext, text := range map[string][]string{"html": out, "md": md} {
		if err := os.WriteFile(filepath.Join(vd, kind+"."+ext), []byte(strings.Join(text, "\n")+"\n"), 0o644); err != nil {
			return "", err
		}
	}
	return filepath.Join(vd, kind+".html"), nil
}

// page renders the HTML view: the answer and gap charts first, then the plan
// and decision (review) or repairs (report), the findings, and every other
// record table in a collapsed appendix.
func (v view) page(b *built, kind, title, summary, csp string, decide bool) []string {
	verdict := pyStr(b.rj["readiness_verdict"])
	lead := []string{"findings", "opportunities", "leads"}
	if kind == "review" {
		lead = append([]string{"plan"}, lead...)
	} else {
		lead = append([]string{"repairs", "performance", "cycles"}, lead...)
	}
	byAnchor := map[string]section{}
	for _, s := range b.secs {
		byAnchor[s.anchor] = s
	}
	charts := []struct{ id, name, html string }{
		{"gaps", "Gap map", v.gapMap(b)}, {"losses", "Score lost", v.controlLoss(b)},
		{"hotspots", "Hotspots", v.hotspots(b)}, {"reach", "Coverage", v.reach(b)},
	}
	var nav strings.Builder
	link := func(id, name, extra string) {
		fmt.Fprintf(&nav, `<li><a href="#%s">%s%s</a></li>`, id, esc(name), extra)
	}
	link("summary", "Summary", "")
	for _, c := range charts {
		if c.html != "" {
			link(c.id, c.name, "")
		}
	}
	shown := map[string]bool{}
	for _, a := range lead {
		s := byAnchor[a]
		if s.anchor == "" || (len(s.body) == 0 && a != "findings" && a != "plan") {
			continue
		}
		shown[a] = true
		if a == "findings" && decide {
			link("decide", "Decision", "")
		}
		name := s.heading
		if short := navName[a]; short != "" {
			name = short
		}
		link(a, name, count(s))
	}
	if decide && !shown["findings"] {
		link("decide", "Decision", "")
	}
	link("records", "All records", "")

	out := []string{`<!doctype html><html lang="en"><head><meta charset="utf-8">`,
		`<meta http-equiv="Content-Security-Policy" content="` + csp + `">`,
		`<meta name="viewport" content="width=device-width,initial-scale=1">`,
		"<title>" + esc(title) + "</title><style>" + css + "</style></head><body>",
		// Escaped (unlike the Python recipe) so a hostile run id cannot close the comment.
		"<!-- " + esc(b.stamp) + " -->", `<a class="skip" href="#main">Skip to content</a>`,
		`<header class="top"><span class="brand">` + esc(title) + `</span><code>` + esc(b.stamp) + `</code>` +
			`<span class="pill ` + tone(verdict) + `">` + v.txt(b.rj["readiness_verdict"]) + `</span></header>`,
		`<div class="shell"><nav class="toc" aria-label="Sections"><ul>` + nav.String() + "</ul></nav>",
		`<main id="main">`, v.summaryBlock(b, summary, decide)}
	for _, c := range charts {
		if c.html != "" {
			out = append(out, c.html)
		}
	}
	decision := `<section id="decide"><h2>Decision</h2><p><strong>Nothing on this page approves anything.</strong> Copy a message into the ` +
		`conversation with the agent; only your message there counts. Edit the task list to approve a subset, or ` +
		`reply with rejections, deferrals or questions per task ID.</p><label for="ap">Approval message</label>` +
		`<textarea id="ap" readonly>` + esc(b.approve) + `</textarea></section>`
	placed := false
	for _, a := range lead {
		if !shown[a] {
			continue
		}
		if a == "findings" && decide {
			out = append(out, decision)
			placed = true
		}
		out = append(out, v.section(b, byAnchor[a]))
	}
	if decide && !placed {
		out = append(out, decision)
	}
	rest := 0
	for _, s := range b.secs {
		if !shown[s.anchor] {
			rest++
		}
	}
	out = append(out, fmt.Sprintf(`<details id="records" class="records"><summary>All records <span class="count">%d tables</span></summary>`+
		`<p class="muted">Every canonical record behind the charts above, as tables.</p>`, rest))
	for _, s := range b.secs {
		if !shown[s.anchor] {
			out = append(out, v.section(b, s))
		}
	}
	return append(out, "</details></main></div></body></html>")
}

func (v view) section(b *built, s section) string {
	var w strings.Builder
	w.WriteString(`<section id="` + esc(s.anchor) + `"><h2>` + esc(s.heading) + count(s) + `</h2>`)
	if s.anchor == "plan" && len(b.tasks) > 0 {
		return v.planView(b, s)
	}
	switch items, isCards := b.cards[s.anchor]; {
	case len(s.body) == 0:
		w.WriteString(`<p class="muted">No records.</p>`)
	case isCards:
		w.WriteString(v.cardList(items, b.byF, s.anchor == "findings"))
	default:
		w.WriteString(v.table(s))
	}
	return w.String() + "</section>"
}

func count(s section) string {
	if s.anchor == "result" {
		return ""
	}
	return fmt.Sprintf(` <span class="count">%d</span>`, len(s.body))
}

// tone maps a verdict, outcome or gate token to its semantic colour role.
func tone(s string) string {
	switch {
	case goodTok[s]:
		return "t-ok"
	case s == "FAIL" || s == "NOT_READY" || s == "NOT_ASSESSABLE" || strings.HasPrefix(s, "BLOCKED"):
		return "t-bad"
	case warnTok[s]:
		return "t-warn"
	}
	return "t-none"
}

func (v view) table(s section) string {
	var w strings.Builder
	w.WriteString(`<div class="scroll" role="region" tabindex="0" aria-label="` + esc(s.heading) + `"><table><caption>` +
		esc(s.heading) + `</caption><thead><tr>`)
	for _, c := range s.cols {
		w.WriteString(`<th scope="col">` + esc(c) + `</th>`)
	}
	w.WriteString("</tr></thead><tbody>")
	for _, r := range s.body {
		w.WriteString("\n<tr")
		if first := pyStr(r[0]); rowID.MatchString(first) {
			w.WriteString(` id="` + esc(first) + `"`)
		}
		w.WriteString(">")
		for i, x := range r {
			cls := ""
			if t, ok := x.(string); ok && numCell.MatchString(t) {
				cls = ` class="n"`
			} else if t, ok := x.(json.Number); ok && numCell.MatchString(t.String()) {
				cls = ` class="n"`
			} else if i < len(s.cols) && monoCols[s.cols[i]] && !empty(x) {
				cls = ` class="m"`
			}
			w.WriteString("<td" + cls + ">" + v.e(x) + "</td>")
		}
		w.WriteString("</tr>")
	}
	w.WriteString("</tbody></table></div>")
	return w.String()
}

// sevKey is the finding's severity if it is on the known scale, else "none";
// only these fixed keys ever reach a class attribute.
func sevKey(f map[string]any) string {
	if s := ovio.Str(f["severity"]); s != "none" && sevLabel[s] != "" {
		return s
	}
	return "none"
}

func sevRank(f map[string]any) int {
	for i, s := range sevOrder {
		if sevKey(f) == s {
			return i
		}
	}
	return len(sevOrder)
}

// cardList draws findings most severe first; the confirmed-findings list
// carries a CSS-only severity filter (without :has() support all cards show).
func (v view) cardList(items []map[string]any, byF map[string][]map[string]any, filter bool) string {
	sorted := append([]map[string]any(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool { return sevRank(sorted[i]) < sevRank(sorted[j]) })
	var w strings.Builder
	if filter {
		n := map[string]int{}
		for _, f := range items {
			n[sevKey(f)]++
		}
		fmt.Fprintf(&w, `<fieldset class="filter"><legend>Show</legend><input type="radio" name="sev" id="sev-all" checked>`+
			`<label for="sev-all">All <span class="count">%d</span></label>`, len(items))
		for _, s := range sevOrder {
			if n[s] > 0 {
				fmt.Fprintf(&w, `<input type="radio" name="sev" id="sev-%s"><label for="sev-%s">%s <span class="count">%d</span></label>`,
					s, s, sevLabel[s], n[s])
			}
		}
		w.WriteString("</fieldset>")
	}
	w.WriteString(`<div class="cards">`)
	for _, f := range sorted {
		w.WriteString("\n" + v.card(f, byF[pyStr(f["id"])], !filter))
	}
	w.WriteString("</div>")
	return w.String()
}

// card renders one finding as a one-line row that expands: severity, id,
// title and first location in the summary; failing case, impact, evidence,
// fix, verification and history inside.
func (v view) card(f map[string]any, tasks []map[string]any, opp bool) string {
	sev := sevKey(f)
	var w strings.Builder
	w.WriteString("<details")
	if id := pyStr(f["id"]); rowID.MatchString(id) {
		w.WriteString(` id="` + esc(id) + `"`)
	}
	w.WriteString(` class="finding s-` + sev + `"><summary>`)
	if !opp {
		label := sevLabel[sev]
		if s := ovio.Str(f["severity"]); sev == "none" && s != "" {
			label = esc(v.redact(s))
		}
		w.WriteString(`<span class="sev s-` + sev + `"><span class="key s-` + sev + `"></span>` + label + `</span>`)
	}
	w.WriteString(`<code class="fid">` + v.e(f["id"]) + `</code><span class="ftitle">` + v.e(f["title"]))
	if st := ovio.Str(f["status"]); st != "" && st != "confirmed" {
		w.WriteString(` <span class="pill">` + v.e(f["status"]) + `</span>`)
	}
	w.WriteString(`</span>`)
	if l := ovio.List(f["locations"]); len(l) > 0 {
		more := ""
		if len(l) > 1 {
			more = fmt.Sprintf(" +%d", len(l)-1)
		}
		w.WriteString(`<code class="where">` + v.loc(l[0]) + more + `</code>`)
	}
	w.WriteString(`</summary><div class="fbody">`)
	if opp {
		w.WriteString(v.kv(f, "benefit:impact"))
	} else {
		w.WriteString(v.kv(f, "failing case:failing_scenario", "impact", "lanes", "kind"))
	}
	w.WriteString(v.details("Evidence", v.kv(f, "evidence", "confidence", "expected", "actual", "prerequisites",
		"safeguards checked:safeguards_checked", "limitations", "locations", "related", "domain", "scope origin:scope_origin",
		"rules:rule_ids", "profiles:profile_ids", "components:component_ids")))
	w.WriteString(v.details("Proposed fix", v.kv(f, "smallest fix:smallest_fix", "fix options:fix_options",
		"write scope:write_scope", "test plan:test_plan", "performance hypothesis:perf_hypothesis")))
	var ver strings.Builder
	if !empty(f["verified_by"]) {
		ver.WriteString(`<dt>verified by</dt><dd>` + v.e(f["verified_by"]) + `</dd>`)
	}
	if len(tasks) > 0 {
		ver.WriteString(`<dt>repair tasks</dt><dd>`)
		for i, t := range tasks {
			if i > 0 {
				ver.WriteString(", ")
			}
			if id := pyStr(t["id"]); rowID.MatchString(id) {
				ver.WriteString(`<a href="#` + esc(id) + `">` + esc(id) + `</a>`)
			} else {
				ver.WriteString(v.e(t["id"]))
			}
		}
		ver.WriteString("</dd>")
	}
	if ver.Len() > 0 {
		w.WriteString(v.details("Verification", `<dl class="kv">`+ver.String()+`</dl>`))
	}
	if h := ovio.List(f["history"]); len(h) > 0 {
		var hist strings.Builder
		hist.WriteString(`<ol class="hist">`)
		for _, e := range h {
			hist.WriteString("<li>")
			if m, ok := e.(map[string]any); ok {
				hist.WriteString(`<strong>` + v.e(m["status"]) + `</strong>`)
				for _, k := range sortedKeys(m) {
					if k != "status" && !empty(m[k]) {
						hist.WriteString(` · <span class="muted">` + esc(k) + `</span> ` + v.e(m[k]))
					}
				}
			} else {
				hist.WriteString(v.e(e))
			}
			hist.WriteString("</li>")
		}
		hist.WriteString("</ol>")
		w.WriteString(v.details(fmt.Sprintf("History (%d)", len(h)), hist.String()))
	}
	w.WriteString("</div></details>")
	return w.String()
}

// loc renders a location object as path:start-end (symbol).
func (v view) loc(x any) string {
	l, ok := x.(map[string]any)
	if !ok || empty(l["path"]) {
		return v.e(x)
	}
	s := pyStr(l["path"])
	if !empty(l["start"]) {
		s += ":" + pyStr(l["start"])
		if e := l["end"]; !empty(e) && pyStr(e) != pyStr(l["start"]) {
			s += "-" + pyStr(e)
		}
	}
	if !empty(l["symbol"]) {
		s += " (" + pyStr(l["symbol"]) + ")"
	}
	return esc(v.redact(s))
}

// kv renders the recorded fields as a definition list; "label:key" renames.
func (v view) kv(f map[string]any, fields ...string) string {
	var w strings.Builder
	for _, fl := range fields {
		label, key, ok := strings.Cut(fl, ":")
		if !ok {
			key = label
		}
		x := f[key]
		if empty(x) {
			continue
		}
		val := v.e(x)
		if key == "locations" {
			var ls []string
			for _, l := range ovio.List(x) {
				ls = append(ls, `<code>`+v.loc(l)+`</code>`)
			}
			val = strings.Join(ls, "<br>")
		}
		w.WriteString(`<dt>` + esc(label) + `</dt><dd>` + val + `</dd>`)
	}
	if w.Len() == 0 {
		return ""
	}
	return `<dl class="kv">` + w.String() + `</dl>`
}

// details renders one labelled block inside an expanded finding.
func (v view) details(summary, body string) string {
	if body == "" {
		return ""
	}
	return `<div class="blk"><h4>` + esc(summary) + `</h4>` + body + `</div>`
}

// threshold returns one scorecard threshold row: value, minimum and result.
func threshold(sc map[string]any, scope string) (val, least *big.Rat, result string) {
	for _, r := range objs(sc["thresholds"]) {
		if s, ok := r["scope"].(string); !ok || s != scope {
			continue
		}
		if x := ovio.Get(r, "value.exact"); !empty(x) {
			val, _ = new(big.Rat).SetString(pyStr(x))
		}
		if !empty(r["min"]) {
			least, _ = new(big.Rat).SetString(pyStr(r["min"]))
		}
		return val, least, ovio.Str(r["result"])
	}
	return nil, nil, ""
}

// floor2 shows a score to two decimals, rounded down so it never overstates.
func floor2(r *big.Rat) string {
	if r.Sign() < 0 {
		return r.FloatString(2)
	}
	n := new(big.Int).Mul(r.Num(), big.NewInt(100))
	n.Div(n, r.Denom())
	q, m := new(big.Int).DivMod(n, big.NewInt(100), new(big.Int))
	return fmt.Sprintf("%s.%02d", q, m.Int64())
}

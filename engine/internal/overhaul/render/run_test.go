package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/overhaul/score"
)

const (
	hostile    = "\"><img src=x onerror=alert(1)><script>alert(2)</script>|`rm -rf /`"
	hostileEsc = "&quot;&gt;&lt;img src=x onerror=alert(1)&gt;&lt;script&gt;alert(2)&lt;/script&gt;|`rm -rf /`"
	fakeHome   = "/Users/tester"
	badLane    = `x"><a href="#p">y</a>`
)

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

type obj = map[string]any

// fixture builds a run dir and generation; withPlan=false makes an
// assessment-only audit that has a rubric revision but no plan.
func fixture(t *testing.T, withPlan bool) (run, gen string) {
	t.Helper()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome) // os.UserHomeDir on Windows
	run = t.TempDir()
	gen = filepath.Join(run, "generations", "g0003.tmp")
	writeJSON(t, filepath.Join(run, "revisions", "rubric-r1.json"), obj{
		"rev":     1,
		"domains": obj{"correctness": 1, "security": 1},
		"lanes":   []any{"backend"},
		"controls": []any{
			obj{"id": "C-1", "domain": "correctness", "lanes": []any{"backend"}, "weight": 0.1, "hard_gate": true},
			obj{"id": "C-2", "domain": "security", "lanes": []any{"backend"}, "weight": 0.2},
		},
	})
	revisions := obj{"rubric": obj{"rev": 1, "file": "revisions/rubric-r1.json"}}
	if withPlan {
		writeJSON(t, filepath.Join(run, "revisions", "plan-r1.json"), obj{
			"rev":    1,
			"rubric": obj{"file": "revisions/rubric-r1.json"},
			"tasks": []any{
				obj{"id": "T-1", "lane": "backend", "findings": []any{"F-1"}, "oracle": "go test ./x", "risk": "low"},
				obj{"id": "T-2", "lane": "frontend"},
			},
		})
		revisions["plan"] = obj{"rev": 1, "file": "revisions/plan-r1.json"}
	}
	writeJSON(t, filepath.Join(gen, "run.json"), obj{
		"run_id": "R1", "mode": "repair", "assessment_only": !withPlan, "phase": "review",
		"readiness_verdict": "NOT_READY", "execution_outcome": "AWAITING_APPROVAL", "revisions": revisions,
	})
	writeJSON(t, filepath.Join(gen, "coverage.json"), obj{"files": []any{
		obj{"path": fakeHome + "/proj/a.go", "eligible": true, "lanes": []any{"backend"},
			"ranges": []any{obj{"start": 1, "end": 9, "state": "verified"}}},
		obj{"path": "vendor/x.js", "eligible": false, "exclusion": obj{"reason": hostile}},
	}, "applicability": []any{obj{"domain": "operations", "component": "api", "lane": "backend", "state": "blocked",
		"reason": "no container runtime"}}})
	writeJSON(t, filepath.Join(gen, "findings.json"), obj{"findings": []any{
		obj{"id": "F-1", "title": hostile, "kind": "defect", "severity": "high", "status": "confirmed",
			"failing_scenario": "nil deref"},
		obj{"id": "F-2", "title": "OPP-TITLE", "kind": "opportunity", "status": "confirmed"},
	}})
	writeJSON(t, filepath.Join(gen, "evidence.json"), obj{"items": []any{}})
	writeJSON(t, filepath.Join(gen, "dispatch.json"), obj{"attempts": []any{
		obj{"task_id": "T-1", "attempt_id": 1, "lane": "backend", "role": "implementer", "status": "admitted",
			"changed_paths": []any{"x/a.go"}},
		obj{"task_id": "T-9", "attempt_id": 1, "lane": badLane, "role": "reviewer", "status": "admitted"},
	}})
	writeJSON(t, filepath.Join(gen, "approval.json"), obj{"events": []any{
		obj{"id": "A-1", "kind": "plan", "status": "recorded", "quote": "QUOTE-approve r1"},
	}})
	writeJSON(t, filepath.Join(gen, "cycles.json"), obj{"cycles": []any{}})
	writeJSON(t, filepath.Join(gen, "scorecard.json"), obj{
		"readiness_verdict": "NOT_READY",
		"status_by_control": obj{"C-1": "FAIL"},
		"global":            obj{"Q": obj{"exact": "1/2"}, "domains": obj{"correctness": obj{"exact": "5/2"}, "security": obj{"exact": "19/2"}}},
		"lanes":             obj{"backend": obj{"Q": obj{"exact": "487/50"}, "domains": obj{"correctness": obj{"exact": "1"}, "security": obj{"exact": "10"}}}},
		"thresholds": []any{
			obj{"scope": "global", "value": obj{"exact": "1/2", "decimal": "0.5"}, "min": "97/10", "result": "FAIL"},
			obj{"scope": "lane backend", "value": obj{"exact": "487/50", "decimal": "9.74"}, "min": "97/10", "result": "PASS"},
			obj{"scope": "domain correctness", "value": obj{"exact": "5/2"}, "min": "9", "result": "FAIL"},
			obj{"scope": "domain security", "value": obj{"exact": "19/2"}, "min": "9", "result": "PASS"},
		},
		"gates": obj{"G-THRESHOLDS": "FAIL", "G-MANDATORY": "FAIL", "G-COVERAGE": "PASS", "G-ORACLES": "UNKNOWN"},
	})
	writeJSON(t, filepath.Join(gen, "scorecard-baseline.json"), obj{
		"thresholds": []any{obj{"scope": "global", "value": obj{"exact": "1/4"}}},
	})
	return run, gen
}

func renderKind(t *testing.T, run, gen, kind string) (string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	if code := Run([]string{run, gen, kind}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	want := filepath.Join(gen, "views", kind+".html")
	if strings.TrimSpace(out.String()) != want {
		t.Fatalf("printed %q, want %q", out.String(), want)
	}
	h, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	m, err := os.ReadFile(filepath.Join(gen, "views", kind+".md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(h), string(m)
}

// sectionHTML returns the HTML of one <section id="...">.
func sectionHTML(t *testing.T, page, id string) string {
	t.Helper()
	i := strings.Index(page, `<section id="`+id+`">`)
	if i < 0 {
		t.Fatalf("section %q missing", id)
	}
	s := page[i:]
	return s[:strings.Index(s, "</section>")]
}

// tags returns every markup tag; escaped text never contains a raw '<'.
func tags(page string) []string {
	var out []string
	for _, part := range strings.Split(page, "<")[1:] {
		out = append(out, part[:strings.IndexByte(part, '>')])
	}
	return out
}

func TestHostileTextIsInert(t *testing.T) {
	run, gen := fixture(t, true)
	h, _ := renderKind(t, run, gen, "review")
	for _, tag := range tags(h) {
		low := strings.ToLower(tag)
		if strings.HasPrefix(low, "script") || strings.HasPrefix(low, "img") || strings.Contains(low, "onerror") {
			t.Fatalf("live markup injected: <%s>", tag)
		}
	}
	for _, id := range []string{"findings", "coverage"} {
		if !strings.Contains(sectionHTML(t, h, id), hostileEsc) {
			t.Errorf("escaped hostile text missing in %s", id)
		}
	}
}

func TestCSPPinsStylesheetAndNoRemoteURLs(t *testing.T) {
	run, gen := fixture(t, true)
	h, _ := renderKind(t, run, gen, "review")
	style := h[strings.Index(h, "<style>")+len("<style>") : strings.Index(h, "</style>")]
	sum := sha256.Sum256([]byte(style))
	want := `content="default-src 'none'; style-src 'sha256-` + base64.StdEncoding.EncodeToString(sum[:]) +
		`'; img-src data:; base-uri 'none'; form-action 'none'"`
	if !strings.Contains(h, want) {
		t.Fatal("CSP missing or hash does not match the emitted stylesheet")
	}
	if strings.Contains(h, "http://") || strings.Contains(h, "https://") || strings.Contains(h, "<script") {
		t.Fatal("remote URL or script present")
	}
	if nt, nc := strings.Count(h, "<table>"), strings.Count(h, "<caption>"); nt == 0 || nt != nc {
		t.Fatalf("tables %d captions %d", nt, nc)
	}
	if strings.Count(h, `<div class="scroll" role="region" tabindex="0"`) != strings.Count(h, "<table>") {
		t.Fatal("table outside a focusable scroll region")
	}
}

func TestMarkdownEscapesAndHomeRedacted(t *testing.T) {
	run, gen := fixture(t, true)
	h, md := renderKind(t, run, gen, "review")
	if strings.Contains(md, "<script>") || !strings.Contains(md, "&lt;/script>\\|`rm -rf /`") {
		t.Fatal("markdown cell not escaped")
	}
	for _, page := range []string{h, md} {
		if strings.Contains(page, fakeHome) || !strings.Contains(page, "~/proj/a.go") {
			t.Fatal("home path not redacted")
		}
	}
}

func TestMarkdownCellDropsCarriageReturns(t *testing.T) {
	run, gen := fixture(t, true)
	var f obj
	b, err := os.ReadFile(filepath.Join(gen, "findings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	f["findings"].([]any)[0].(obj)["failing_scenario"] = "line one\r\n# HEADING\r\n```\rlone"
	writeJSON(t, filepath.Join(gen, "findings.json"), f)
	_, md := renderKind(t, run, gen, "report")
	if strings.Contains(md, "\r") {
		t.Fatal("markdown cell carries a carriage return")
	}
	if !strings.Contains(md, "line one # HEADING ``` lone") {
		t.Fatal("line breaks not collapsed to single spaces")
	}
}

func TestRowIDsDecisionAndStamp(t *testing.T) {
	run, gen := fixture(t, true)
	h, md := renderKind(t, run, gen, "review")
	// Findings render as expandable rows: the row element carries the finding's anchor.
	if !strings.Contains(sectionHTML(t, h, "findings"), `<details id="F-1" class="finding s-high">`) || strings.Count(h, `id="F-1"`) != 1 {
		t.Fatal("finding anchor F-1 missing or not unique")
	}
	if !strings.Contains(sectionHTML(t, h, "plan"), `<details id="T-1" class="task">`) || strings.Count(h, `id="T-1"`) != 1 {
		t.Fatal("task anchor T-1 missing or not unique")
	}
	if strings.Contains(h, `id="nil deref"`) || strings.Contains(h, `id="run_id"`) {
		t.Fatal("id on a non-ID first cell")
	}
	d, f := strings.Index(h, `<section id="decide">`), strings.Index(h, `<section id="findings">`)
	if d < 0 || d > f {
		t.Fatal("decision section missing or after findings")
	}
	if !strings.Contains(sectionHTML(t, h, "decide"), `<textarea id="ap" readonly>Approve overhaul R1 plan revision 1, digest `) {
		t.Fatal("approval textarea missing")
	}
	if !strings.HasPrefix(md, "<!-- overhaul-stamp run=R1 generation=g0003 plan=r1:") ||
		!strings.Contains(h, "<!-- overhaul-stamp run=R1 generation=g0003 plan=r1:") ||
		!strings.Contains(h, `<header class="top"><span class="brand">Overhaul review R1</span><code>overhaul-stamp run=R1 generation=g0003 plan=r1:`) {
		t.Fatal("stamp missing")
	}
	if !strings.Contains(h, `<p role="status"><strong>Outcome AWAITING_APPROVAL · verdict NOT_READY · phase review</strong></p>`) {
		t.Fatal("status summary missing")
	}
}

func TestHostileLaneCannotInject(t *testing.T) {
	run, gen := fixture(t, true)
	h, md := renderKind(t, run, gen, "review")
	if strings.Contains(h, `<a href="#p">`) || strings.Contains(md, `<a href="#p">`) {
		t.Fatal("lane name injected an anchor")
	}
	if !strings.Contains(h, `<section id="lane-x---a-href---p--y--a-">`) {
		t.Fatal("lane anchor not sanitized")
	}
}

func TestSectionContents(t *testing.T) {
	run, gen := fixture(t, true)
	h, md := renderKind(t, run, gen, "review")
	for _, id := range []string{"findings", "leads", "rejected", "repairs"} {
		if strings.Contains(sectionHTML(t, h, id), "OPP-TITLE") {
			t.Fatalf("opportunity leaked into %s", id)
		}
	}
	if !strings.Contains(sectionHTML(t, h, "opportunities"), "OPP-TITLE") {
		t.Fatal("opportunity missing")
	}
	if !strings.Contains(md, "| F-1 / T-1 | high, not recorded | nil deref | x/a.go | go test ./x |") {
		t.Fatal("repairs join missing")
	}
	card := sectionHTML(t, h, "scorecard")
	if !strings.Contains(card, `<td>global</td><td><span class="muted">not recorded</span></td><td class="n">3/10</td><td class="n">1/4</td><td class="n">1/2</td><td>C-2</td><td>C-1</td>`) {
		t.Fatalf("transparent scorecard global row wrong: %s", card)
	}
	if !strings.Contains(sectionHTML(t, h, "approvals"), "QUOTE-approve r1") {
		t.Fatal("approval quote missing")
	}
	if !strings.Contains(sectionHTML(t, h, "applicability"), "no container runtime") {
		t.Fatal("blocked applicability cell missing from the report")
	}
}

func TestPlanMarkdown(t *testing.T) {
	run, gen := fixture(t, true)
	renderKind(t, run, gen, "review")
	b, err := os.ReadFile(filepath.Join(gen, "views", "plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	p := string(b)
	if !strings.HasPrefix(p, "<!-- overhaul-stamp run=R1 generation=g0003 plan=r1:") ||
		!strings.Contains(p, "| T-1 | backend | F-1 |") ||
		!strings.Contains(p, "```text\nApprove overhaul R1 plan revision 1, digest ") {
		t.Fatalf("plan.md incomplete:\n%s", p)
	}
}

func TestAssessmentOnlyAudit(t *testing.T) {
	run, gen := fixture(t, false)
	for _, kind := range []string{"review", "report"} {
		h, md := renderKind(t, run, gen, kind)
		ctl := sectionHTML(t, h, "controls")
		if !strings.Contains(ctl, `<tr id="C-1">`) || !strings.Contains(ctl, `<tr id="C-2">`) {
			t.Fatal("rubric controls not listed from run.json revisions")
		}
		if strings.Contains(h+md, "Approve overhaul") || strings.Contains(h, `id="decide"`) {
			t.Fatal("assessment-only run shows an approval message")
		}
		if !strings.Contains(h, "plan=none") {
			t.Fatal("stamp should record plan=none")
		}
	}
	if _, err := os.Stat(filepath.Join(gen, "views", "plan.md")); !os.IsNotExist(err) {
		t.Fatal("plan.md written without a plan")
	}
}

func TestUsage(t *testing.T) {
	var out, errb bytes.Buffer
	for _, args := range [][]string{{"run"}, {"run", "gen", "bogus"}} {
		if Run(args, &out, &errb) != 2 {
			t.Fatalf("%v: want exit 2", args)
		}
	}
}

func TestSummaryAnswersFirst(t *testing.T) {
	run, gen := fixture(t, true)
	h, _ := renderKind(t, run, gen, "review")
	i := strings.Index(h, `<header id="summary" class="hero">`)
	if i < 0 || i > strings.Index(h, `<section id="gaps">`) || strings.Index(h, `<section id="gaps">`) > strings.Index(h, `<section id="decide">`) {
		t.Fatal("summary, then charts, then the decision")
	}
	hero := h[i : i+strings.Index(h[i:], "</header>")]
	for _, want := range []string{
		`<h1 class="verdict t-bad">Not ready</h1>`,
		`<p class="hero-num">0.50<span> / 10</span></p>`,
		`Target 9.70 · 9.20 to go`,
		`aria-label="Global score 0.50 of 10, target 9.70"`,
		`<span class="lbl">backend</span>`, `<span class="val">9.74</span>`,
		// open gates in plain words with their id and status; passing gates counted
		`Scores are below their minimums<br><code>G-THRESHOLDS · FAIL</code>`,
		`<li class="t-warn"><span class="mark">?</span><span>Regression tests are not proven red then green<br><code>G-ORACLES · UNKNOWN</code>`,
		`1 other gate passes.`,
		// confirmed findings by severity (the opportunity F-2 is not a finding)
		`<h2>1 open finding</h2>`, `<a href="#findings">High</a> <strong>1</strong>`,
		`role="img" aria-label="Open findings by severity"><title>Open findings by severity</title>`,
		`<a href="#decide">Decide on the repair plan</a> <span class="muted">2 tasks proposed</span>`,
	} {
		if !strings.Contains(hero, want) {
			t.Errorf("summary lacks %s", want)
		}
	}
	if strings.Contains(hero, "Confirmed findings by severity") {
		t.Error("severity bar label still says confirmed findings")
	}
	if !strings.Contains(h, `<span class="pill t-bad">NOT_READY</span></header>`) {
		t.Error("top bar verdict pill missing")
	}
	f := sectionHTML(t, h, "findings")
	if !strings.Contains(f, `id="sev-all" checked>`) || !strings.Contains(f, `id="sev-high"`) || strings.Contains(f, `id="sev-critical"`) {
		t.Error("severity filter should offer All plus only the severities present")
	}
	if strings.Contains(sectionHTML(t, h, "opportunities"), `class="filter"`) {
		t.Error("opportunities carry no severity filter")
	}
	if !strings.Contains(h, `<details id="records" class="records">`) || strings.Index(h, `<details id="records"`) > strings.Index(h, `<section id="coverage">`) {
		t.Error("record tables belong in the collapsed appendix")
	}
	// The appendix count names what it counts: sections, not tables, since a
	// card-list section emits no <table>. It must equal the sections emitted.
	ap := h[strings.Index(h, `<details id="records"`):]
	m := regexp.MustCompile(`<summary>All records <span class="count">(\d+) sections</span></summary>`).FindStringSubmatch(ap)
	if m == nil {
		t.Fatal("appendix summary must count sections")
	}
	if n := strconv.Itoa(strings.Count(ap, "<section ")); m[1] != n {
		t.Errorf("appendix states %s sections but emits %s", m[1], n)
	}
}

// A run scored only at baseline must still show its scores (a real run did not).
func TestSummaryFallsBackToBaselineScorecard(t *testing.T) {
	run, gen := fixture(t, true)
	if err := os.Remove(filepath.Join(gen, "scorecard.json")); err != nil {
		t.Fatal(err)
	}
	h, _ := renderKind(t, run, gen, "review")
	if !strings.Contains(h, `<p class="hero-num">0.25<span> / 10</span></p>`) {
		t.Fatal("baseline scorecard not used when no candidate scorecard exists")
	}
}

func TestGapCharts(t *testing.T) {
	run, gen := fixture(t, true)
	h, _ := renderKind(t, run, gen, "review")
	gaps := sectionHTML(t, h, "gaps")
	c, s := strings.Index(gaps, `#controls">correctness`), strings.Index(gaps, `#controls">security`)
	if c < 0 || s < 0 || c > s {
		t.Fatal("gap map lists the worst domain first")
	}
	for _, want := range []string{
		`<td class="cell g2"><span class="val">2.50</span>`, // 6.5 below the 9.0 floor
		`<td class="cell g0"><span class="val">9.50</span>`, // meets the floor
		`<td class="cell g2"><span class="val">1.00</span>`, // backend correctness: exactly 8 below
		`<th scope="col">backend</th>`,
	} {
		if !strings.Contains(gaps, want) {
			t.Errorf("gap map lacks %s", want)
		}
	}
	// A cell with no score is drawn gx and the legend names that bucket.
	if strings.Contains(gaps, `class="cell gx"`) {
		t.Fatal("control: every fixture cell is scored")
	}
	b, err := os.ReadFile(filepath.Join(gen, "scorecard.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sc obj
	if err := json.Unmarshal(b, &sc); err != nil {
		t.Fatal(err)
	}
	delete(sc["lanes"].(obj)["backend"].(obj)["domains"].(obj), "security")
	writeJSON(t, filepath.Join(gen, "scorecard.json"), sc)
	h, _ = renderKind(t, run, gen, "review")
	gaps = sectionHTML(t, h, "gaps")
	if !strings.Contains(gaps, `class="cell gx"`) {
		t.Fatal("a lane domain without a score should be drawn gx")
	}
	if !strings.Contains(gaps, `<li><span class="key gx"></span>not scored</li>`) {
		t.Error("gap map legend lacks a row for unscored cells")
	}
	loss := sectionHTML(t, h, "losses")
	if !strings.Contains(loss, `1 of 1 failing</span>`) || !strings.Contains(loss, `0 of 1 failing · 1 without evidence</span>`) {
		t.Errorf("control split wrong: %s", loss)
	}
	if !strings.Contains(sectionHTML(t, h, "hotspots"), `<code>(no location)</code>`) {
		t.Error("finding without a location missing from hotspots")
	}
	if !strings.Contains(sectionHTML(t, h, "plan"), `1 of 1</span>`) {
		t.Error("plan coverage of the high finding missing")
	}
	if !strings.Contains(sectionHTML(t, h, "reach"), `all reviewed`) {
		t.Error("backend lane coverage missing")
	}
	// Every chart is an image with a text alternative, and its values are printed too.
	if n, m := strings.Count(h, `<svg class="bar"`), strings.Count(h, `role="img" aria-label="`); n == 0 || n != m {
		t.Fatalf("charts %d, labelled %d", n, m)
	}
}

func TestGapClassBoundaries(t *testing.T) {
	floor := big.NewRat(9, 1)
	for val, want := range map[string]string{"9": "g0", "10": "g0", "8.99": "g4", "7": "g4", "6.99": "g3", "4": "g3", "3.99": "g2", "1": "g2", "0.99": "g1", "0": "g1"} {
		r, _ := new(big.Rat).SetString(val)
		if got := gapClass(r, floor); got != want {
			t.Errorf("gapClass(%s) = %s, want %s", val, got, want)
		}
	}
	if gapClass(nil, floor) != "gx" {
		t.Error("missing score must not look like a result")
	}
}

func TestArea(t *testing.T) {
	for p, want := range map[string]string{"crates/central/src/x.rs": "crates/central", "frontend/x.ts": "frontend",
		"README.md": "(repository root)", "/a/b/c": "a/b"} {
		if got := area(p); got != want {
			t.Errorf("area(%q) = %q, want %q", p, got, want)
		}
	}
}

func TestFloor2NeverOverstates(t *testing.T) {
	for r, want := range map[*big.Rat]string{big.NewRat(9699, 1000): "9.69", big.NewRat(97, 10): "9.70",
		big.NewRat(10, 1): "10.00", big.NewRat(0, 1): "0.00", big.NewRat(1, 3): "0.33"} {
		if got := floor2(r); got != want {
			t.Errorf("floor2(%s) = %s, want %s", r, got, want)
		}
	}
}

// Light is the base (and print) theme; dark applies on screens that prefer it
// and must redefine every colour token.
func TestThemeTokens(t *testing.T) {
	tok := regexp.MustCompile(`--[a-z0-9-]+:`)
	block := func(start string) string {
		i := strings.Index(css, start)
		if i < 0 {
			t.Fatalf("%q missing", start)
		}
		s := css[i+len(start):]
		return s[:strings.IndexByte(s, '}')]
	}
	light, dark := block(":root{"), block("@media screen and (prefers-color-scheme:dark){:root{")
	if !strings.Contains(light, "color-scheme:light") || !strings.Contains(dark, "color-scheme:dark") {
		t.Fatal("color-scheme not declared per theme")
	}
	darkTok := map[string]bool{}
	for _, k := range tok.FindAllString(dark, -1) {
		darkTok[k] = true
	}
	for _, k := range tok.FindAllString(light, -1) {
		if k != "--serif:" && k != "--sans:" && k != "--mono:" && !darkTok[k] {
			t.Errorf("dark theme does not redefine %s", k)
		}
	}
	for _, k := range []string{"--bg:", "--fg:", "--accent:", "--s-critical:", "--s-high:", "--s-medium:", "--s-low:", "--s-informational:"} {
		if !strings.Contains(light, k) || !darkTok[k] {
			t.Errorf("token %s missing from a theme", k)
		}
	}
	if strings.Contains(css, "url(") || strings.Contains(css, "@import") || strings.Contains(css, "@font-face") {
		t.Error("stylesheet references an external resource")
	}
}

// A threshold minimum that is recorded but cannot be read must be
// reported as unknown. Substituting the renderer's own policy constant prints a
// number the record never held, and the hero then states "met" against it.
func TestUnreadableMinimumIsNotSubstituted(t *testing.T) {
	run, gen := fixture(t, true)

	// Control: this run's parseable minimum still drives the hero. Without this
	// the assertions below could pass because the page printed nothing.
	h, _ := renderKind(t, run, gen, "review")
	hero := heroOf(t, h)
	if !strings.Contains(hero, `Target 9.70 · 9.20 to go`) {
		t.Fatalf("control: a parseable minimum must still print a numeric target, got %s", hero)
	}

	// Same run, same score, recorded minimum replaced by one this renderer
	// cannot parse ("nine point seven").
	b, err := os.ReadFile(filepath.Join(gen, "scorecard.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sc obj
	if err := json.Unmarshal(b, &sc); err != nil {
		t.Fatal(err)
	}
	sc["thresholds"].([]any)[0].(obj)["min"] = "nine point seven"
	writeJSON(t, filepath.Join(gen, "scorecard.json"), sc)

	h, _ = renderKind(t, run, gen, "review")
	hero = heroOf(t, h)

	// The hero still reports the score: this is not a page that printed nothing.
	if !strings.Contains(hero, `<p class="hero-num">0.50<span> / 10</span></p>`) {
		t.Errorf("hero lost its score line: %s", hero)
	}
	// The unreadable minimum is reported as unknown, in the text and in the
	// bar's text alternative.
	for _, want := range []string{
		`Target not recorded`,
		`aria-label="Global score 0.50 of 10, target not recorded"`,
	} {
		if !strings.Contains(hero, want) {
			t.Errorf("hero must report an unreadable minimum as unknown, lacks %s: %s", want, hero)
		}
	}
	// No substituted number is printed as the target.
	if m := regexp.MustCompile(`Target\s+[0-9]`).FindString(hero); m != "" {
		t.Errorf("hero printed a numeric target %q for an unreadable minimum", m)
	}
	if m := regexp.MustCompile(`aria-label="Global score [^"]*target [0-9]`).FindString(hero); m != "" {
		t.Errorf("global bar labelled with a numeric target %q for an unreadable minimum", m)
	}
	// The record still shows the raw minimum, so the reader sees what was unread.
	if !strings.Contains(h, `nine point seven`) {
		t.Error("the scorecard record should still print the raw minimum")
	}
}

func heroOf(t *testing.T, h string) string {
	t.Helper()
	i := strings.Index(h, `<header id="summary" class="hero">`)
	if i < 0 {
		t.Fatal("hero missing")
	}
	return h[i : i+strings.Index(h[i:], "</header>")]
}

// A scope with no recorded minimum must not borrow the renderer's policy
// constants: the view states the target is unknown and prints no number for it.
func TestAbsentMinimumIsNotSubstituted(t *testing.T) {
	run, gen := fixture(t, true)
	b, err := os.ReadFile(filepath.Join(gen, "scorecard.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sc obj
	if err := json.Unmarshal(b, &sc); err != nil {
		t.Fatal(err)
	}
	// The global row loses its min, the lane row is dropped, and the domain
	// rows are dropped, so no scope records a minimum.
	rows := sc["thresholds"].([]any)
	delete(rows[0].(obj), "min")
	sc["thresholds"] = rows[:1]
	writeJSON(t, filepath.Join(gen, "scorecard.json"), sc)

	h, _ := renderKind(t, run, gen, "review")
	hero := heroOf(t, h)
	for _, want := range []string{
		`<p class="hero-num">0.50<span> / 10</span></p>`,
		`Target not recorded`,
		`aria-label="Global score 0.50 of 10, target not recorded"`,
	} {
		if !strings.Contains(hero, want) {
			t.Errorf("hero lacks %s: %s", want, hero)
		}
	}
	if m := regexp.MustCompile(`Target\s+[0-9]|target [0-9]`).FindString(hero); m != "" {
		t.Errorf("hero printed a substituted target %q: %s", m, hero)
	}
	gaps := sectionHTML(t, h, "gaps")
	if m := regexp.MustCompile(`floor [0-9]`).FindString(gaps); m != "" {
		t.Errorf("gap map printed a substituted floor %q", m)
	}
	if !strings.Contains(gaps, "floor not recorded") {
		t.Error("gap map should say the floor is not recorded")
	}
	// A cell with no floor has no gap to colour: it must not land in a gap bucket.
	if m := regexp.MustCompile(`<td class="cell (g[0-4])"`).FindString(gaps); m != "" {
		t.Errorf("gap map placed a no-floor cell in a gap bucket: %s", m)
	}
	if !strings.Contains(gaps, `<td class="cell c-unk">`) {
		t.Error("a no-floor cell should render with the neutral unknown class")
	}
}

func renderWithGates(t *testing.T, withPlan bool, gates obj) string {
	t.Helper()
	run, gen := fixture(t, withPlan)
	b, err := os.ReadFile(filepath.Join(gen, "scorecard.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sc obj
	if err := json.Unmarshal(b, &sc); err != nil {
		t.Fatal(err)
	}
	sc["gates"] = gates
	writeJSON(t, filepath.Join(gen, "scorecard.json"), sc)
	h, _ := renderKind(t, run, gen, "review")
	return heroOf(t, h)
}

func passing(ids []string) obj {
	g := obj{}
	for _, id := range ids {
		g[id] = "PASS"
	}
	return g
}

// A gate record that omits gates must never read as "nothing blocks": every
// gate the run's mode requires and the record lacks is named as not recorded.
func TestIncompleteGateRecordDoesNotClearReadiness(t *testing.T) {
	audit, repair := score.GateIDs(false), score.GateIDs(true)
	if len(audit) != 10 || len(repair) != 19 {
		t.Fatalf("expected 10 audit and 19 remediation gates, got %d and %d", len(audit), len(repair))
	}

	// One gate recorded on a remediation run: the other eighteen are named.
	hero := renderWithGates(t, true, obj{"G-THRESHOLDS": "PASS"})
	if strings.Contains(hero, "Nothing blocks readiness") {
		t.Errorf("a one-gate record cleared readiness: %s", hero)
	}
	for _, id := range repair[1:] {
		if !strings.Contains(hero, "<code>"+id+" · NOT_RECORDED</code>") {
			t.Errorf("hero does not name the absent gate %s: %s", id, hero)
		}
	}
	if strings.Contains(hero, "<code>G-THRESHOLDS ·") {
		t.Errorf("a recorded passing gate was listed as blocking: %s", hero)
	}

	// A complete audit scorecard on a remediation run lacks the repair gates.
	hero = renderWithGates(t, true, passing(audit))
	if strings.Contains(hero, "Nothing blocks readiness") {
		t.Errorf("an audit scorecard cleared a remediation run: %s", hero)
	}
	for _, id := range repair[len(audit):] {
		if !strings.Contains(hero, "<code>"+id+" · NOT_RECORDED</code>") {
			t.Errorf("hero does not name the absent repair gate %s: %s", id, hero)
		}
	}
	for _, id := range audit {
		if strings.Contains(hero, "<code>"+id+" ·") {
			t.Errorf("recorded audit gate %s listed as blocking: %s", id, hero)
		}
	}

	// The same audit scorecard is complete for an assessment-only run, and a
	// complete remediation record is complete for a remediation run.
	for name, c := range map[string]struct {
		plan bool
		ids  []string
	}{"assessment-only audit": {false, audit}, "remediation full": {true, repair}} {
		if hero := renderWithGates(t, c.plan, passing(c.ids)); !strings.Contains(hero, "Nothing blocks readiness") {
			t.Errorf("%s: a complete passing record should clear readiness: %s", name, hero)
		}
	}
}

func readObj(t *testing.T, path string) obj {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m obj
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// setGates records every gate as PASS except the overrides, plus a verdict.
func setGates(t *testing.T, gen string, over map[string]string, verdict string) {
	t.Helper()
	g := obj{}
	for _, id := range score.GateIDs(true) {
		g[id] = "PASS"
	}
	for k, v := range over {
		g[k] = v
	}
	sc := readObj(t, filepath.Join(gen, "scorecard.json"))
	sc["gates"], sc["readiness_verdict"] = g, verdict
	writeJSON(t, filepath.Join(gen, "scorecard.json"), sc)
	rj := readObj(t, filepath.Join(gen, "run.json"))
	rj["readiness_verdict"] = verdict
	writeJSON(t, filepath.Join(gen, "run.json"), rj)
}

func TestExcludedRangeCountsAsReviewed(t *testing.T) {
	run, gen := fixture(t, true)
	c := readObj(t, filepath.Join(gen, "coverage.json"))
	c["files"].([]any)[0].(obj)["ranges"] = []any{obj{"start": 1, "end": 4, "state": "verified"}, obj{"start": 5, "end": 9, "state": "excluded"}}
	writeJSON(t, filepath.Join(gen, "coverage.json"), c)
	h, md := renderKind(t, run, gen, "report")
	if reach := sectionHTML(t, h, "reach"); !strings.Contains(reach, "all reviewed") {
		t.Errorf("excluded range shown as open: %s", reach)
	}
	if strings.Contains(md, "1 / 2") {
		t.Errorf("lane coverage lists the excluded range as open")
	}
}

func TestWaivedGateDoesNotBlock(t *testing.T) {
	run, gen := fixture(t, true)
	setGates(t, gen, map[string]string{"G-CONCURRENCY": "WAIVED_OPERATIONAL"}, "MEETS_TARGETS_WITH_APPROVED_DEGRADATION")
	h, _ := renderKind(t, run, gen, "report")
	hero := heroOf(t, h)
	if strings.Contains(hero, "What blocks readiness") {
		t.Errorf("approved waiver listed as blocking: %s", hero)
	}
	if !strings.Contains(hero, "waived with approval: G-CONCURRENCY") {
		t.Errorf("waiver not named: %s", hero)
	}
}

func TestNotApplicableGatesAreNotCountedAsPassing(t *testing.T) {
	run, gen := fixture(t, true)
	setGates(t, gen, map[string]string{"G-CONCURRENCY": "NOT_APPLICABLE", "G-PERF-TARGETS": "NOT_APPLICABLE", "G-ORACLES": "UNKNOWN"}, "NOT_READY")
	h, _ := renderKind(t, run, gen, "report")
	hero := heroOf(t, h)
	if !strings.Contains(hero, "16 other gates pass · 2 not applicable") {
		t.Errorf("blocked hero miscounts not-applicable gates: %s", hero)
	}
	run, gen = fixture(t, true)
	setGates(t, gen, map[string]string{"G-CONCURRENCY": "NOT_APPLICABLE"}, "MEETS_TARGETS")
	h, _ = renderKind(t, run, gen, "report")
	hero = heroOf(t, h)
	if strings.Contains(hero, "All 19 gates pass") || !strings.Contains(hero, "18 gates pass · 1 not applicable") {
		t.Errorf("clear hero miscounts not-applicable gates: %s", hero)
	}
}

func TestVerifiedFindingIsNotOpen(t *testing.T) {
	run, gen := fixture(t, true)
	f := readObj(t, filepath.Join(gen, "findings.json"))
	f1 := f["findings"].([]any)[0].(obj)
	f1["status"], f1["severity"] = "verified", "critical"
	writeJSON(t, filepath.Join(gen, "findings.json"), f)
	setGates(t, gen, nil, "MEETS_TARGETS")
	h, _ := renderKind(t, run, gen, "report")
	i := strings.Index(h, `<div class="strip">`)
	strip := h[i : i+strings.Index(h[i:], "</div>")]
	if strings.Contains(strip, "confirmed finding") || strings.Contains(strip, "Critical 1") {
		t.Errorf("verified finding counted as open: %s", strip)
	}
	if !strings.Contains(strip, "0 open findings · 1 verified repaired") {
		t.Errorf("verified repair not summarised: %s", strip)
	}
	if strings.Contains(h, `id="hotspots"`) {
		t.Errorf("verified finding listed as a hotspot")
	}
}

func TestHotspotAreaRedactsHome(t *testing.T) {
	run, gen := fixture(t, true)
	f := readObj(t, filepath.Join(gen, "findings.json"))
	f["findings"].([]any)[0].(obj)["locations"] = []any{obj{"path": fakeHome + "/proj/a.go"}}
	writeJSON(t, filepath.Join(gen, "findings.json"), f)
	h, _ := renderKind(t, run, gen, "report")
	if hs := sectionHTML(t, h, "hotspots"); strings.Contains(hs, "tester") {
		t.Errorf("hotspots leak the home directory: %s", hs)
	}
}

func TestHotspotSplitIsPrinted(t *testing.T) {
	run, gen := fixture(t, true)
	h, _ := renderKind(t, run, gen, "review")
	if hs := sectionHTML(t, h, "hotspots"); !strings.Contains(hs, `High 1</span></li>`) {
		t.Errorf("hotspot severity split not printed: %s", hs)
	}
}

func TestLossChartSkipsApprovedNotApplicable(t *testing.T) {
	run, gen := fixture(t, true)
	p := filepath.Join(run, "revisions", "rubric-r1.json")
	r := readObj(t, p)
	r["controls"].([]any)[1].(obj)["na"] = obj{"reason": "r", "evidence": "e", "approved": true}
	writeJSON(t, p, r)
	h, _ := renderKind(t, run, gen, "review")
	loss := sectionHTML(t, h, "losses")
	if strings.Contains(loss, "without evidence") || strings.Contains(loss, `#controls">security`) {
		t.Errorf("approved not-applicable control drawn as missing evidence: %s", loss)
	}
	if !strings.Contains(loss, `1 of 1 failing</span>`) {
		t.Errorf("live control split lost: %s", loss)
	}
}

func TestStylesheetRules(t *testing.T) {
	for _, c := range []struct {
		name string
		want []string
		not  []string
	}{
		{"narrow summary reflow", []string{
			"@media (max-width:40rem){.finding>summary,.task>summary{grid-template-columns:auto auto minmax(0,1fr)}.ftitle,.finding .where{grid-column:1/-1}}"}, nil},
		{"forced colors keep the checked filter and legend swatches", []string{
			"@media (forced-colors:active){.filter input:checked+label{forced-color-adjust:none;background:Highlight;color:HighlightText;border-color:Highlight}.key{forced-color-adjust:none}}"}, nil},
		{"filter never hides a targeted finding", []string{
			".finding:not(.s-critical):not(:target)", ".finding:not(.s-high):not(:target)", ".finding:not(.s-medium):not(:target)",
			".finding:not(.s-low):not(:target)", ".finding:not(.s-informational):not(:target)"}, nil},
		{"filter applies on screen only", []string{"@media screen{#findings:has(#sev-critical:checked)"}, nil},
		{"gap legend key is not the unknown colour", []string{
			".key.c-unk{--c:var(--unk)}.key.gx{--c:var(--track);box-shadow:inset 0 0 0 1px var(--muted)}"}, []string{".key.c-unk,.key.gx", ".key.gx{--c:var(--unk)}"}},
		{"print keeps a closed appendix closed", []string{
			"details::details-content{content-visibility:visible}", "details.records:not([open])::details-content{content-visibility:hidden}"}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			for _, s := range c.want {
				if !strings.Contains(css, s) {
					t.Errorf("stylesheet lacks %q", s)
				}
			}
			for _, s := range c.not {
				if strings.Contains(css, s) {
					t.Errorf("stylesheet still has %q", s)
				}
			}
		})
	}
}

func TestDisclosureCue(t *testing.T) {
	run, gen := fixture(t, true)
	h, _ := renderKind(t, run, gen, "review")
	for _, s := range []string{".finding[open]>summary .fid::before", ".task[open]>summary .fid::before", ".records[open]>summary::before"} {
		if !strings.Contains(h, s) {
			t.Errorf("no disclosure cue %q", s)
		}
	}
}

func TestHeadingsNest(t *testing.T) {
	run, gen := fixture(t, true)
	p := filepath.Join(gen, "findings.json")
	f := readObj(t, p)
	f["findings"].([]any)[0].(obj)["evidence"] = "trace"
	f["findings"].([]any)[0].(obj)["actual"] = "panic"
	f["findings"].([]any)[0].(obj)["smallest_fix"] = "guard"
	writeJSON(t, p, f)
	for _, kind := range []string{"review", "report"} {
		h, _ := renderKind(t, run, gen, kind)
		prev := 0
		for _, m := range regexp.MustCompile(`<h([1-6])[ >]`).FindAllStringSubmatch(h, -1) {
			lvl, _ := strconv.Atoi(m[1])
			if prev > 0 && lvl > prev+1 {
				t.Fatalf("%s: heading level jumps from h%d to h%d", kind, prev, lvl)
			}
			prev = lvl
		}
	}
}

func TestAppendixSectionsAreDisclosures(t *testing.T) {
	run, gen := fixture(t, true)
	h, _ := renderKind(t, run, gen, "report")
	i := strings.Index(h, `<details id="records"`)
	if i < 0 {
		t.Fatal("appendix missing")
	}
	ap := h[i:]
	secs, recs := strings.Count(ap, "<section "), strings.Count(ap, `<details class="rec"><summary>`)
	if secs == 0 || secs != recs {
		t.Errorf("%d sections, %d rec details", secs, recs)
	}
}

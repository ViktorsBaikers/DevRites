package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	}})
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
		"thresholds":        []any{obj{"scope": "global", "value": obj{"exact": "1/2", "decimal": "0.5"}}},
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

func TestRowIDsDecisionAndStamp(t *testing.T) {
	run, gen := fixture(t, true)
	h, md := renderKind(t, run, gen, "review")
	if !strings.Contains(h, `<tr id="F-1">`) {
		t.Fatal("row id F-1 missing")
	}
	if strings.Contains(h, `<tr id="nil deref"`) || strings.Contains(h, `<tr id="run_id"`) {
		t.Fatal("row id on a non-ID first cell")
	}
	d, f := strings.Index(h, `<section id="decide">`), strings.Index(h, `<section id="findings">`)
	if d < 0 || d > f {
		t.Fatal("decision section missing or after findings")
	}
	if !strings.Contains(sectionHTML(t, h, "decide"), `<textarea id="ap" readonly>Approve overhaul R1 plan revision 1, digest `) {
		t.Fatal("approval textarea missing")
	}
	if !strings.HasPrefix(md, "<!-- overhaul-stamp run=R1 generation=g0003 plan=r1:") ||
		!strings.Contains(h, "<!-- overhaul-stamp run=R1 generation=g0003 plan=r1:") {
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
	if !strings.Contains(card, "<td>global</td><td><span class=\"muted\">not recorded</span></td><td>3/10</td><td>1/4</td><td>1/2</td><td>C-2</td><td>C-1</td>") {
		t.Fatalf("transparent scorecard global row wrong: %s", card)
	}
	if !strings.Contains(sectionHTML(t, h, "approvals"), "QUOTE-approve r1") {
		t.Fatal("approval quote missing")
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

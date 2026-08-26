package lib

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOutlineInventoryIDs(t *testing.T) {
	outline := `# Title

## Purpose
demo

## ID inventory

| HTML ` + "`id`" + ` | Meaning |
| --- | --- |
| ` + "`viz-title`" + ` | Header |
| ` + "`diagram-overview`" + ` | Overview |
| viz-title | duplicate ignored |

## Relationships
| From | To | Note |
| --- | --- | --- |
| a | b | c |
`
	got := ParseOutlineInventoryIDs(outline)
	want := []string{"viz-title", "diagram-overview"}
	if len(got) != len(want) {
		t.Fatalf("ids = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %#v, want %#v", got, want)
		}
	}
}

func TestScanHTMLIDs(t *testing.T) {
	html := `<!doctype html><div id="viz-title"></div><svg><marker id='arrow'></marker><g id="viz-title"></g></svg>`
	got := ScanHTMLIDs(html)
	want := []string{"viz-title", "arrow"}
	if len(got) != len(want) {
		t.Fatalf("ids = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %#v, want %#v", got, want)
		}
	}
}

func TestCheckVisualIDConsistencyMatch(t *testing.T) {
	html := `<section id="viz-title"></section><section id="diagram-overview"></section><marker id="arrow"></marker>`
	outline := `## ID inventory
| HTML id | Meaning |
| --- | --- |
| viz-title | Header |
| diagram-overview | Overview |
`
	report := CheckVisualIDConsistency(html, outline)
	if len(report.MissingInHTML) != 0 {
		t.Fatalf("MissingInHTML = %#v, want empty", report.MissingInHTML)
	}
	if len(report.Inventory) != 2 {
		t.Fatalf("Inventory = %#v, want 2 ids", report.Inventory)
	}
	// HTML-only decorative ids are not reported.
	if len(report.HTML) != 3 {
		t.Fatalf("HTML = %#v, want viz-title, diagram-overview, arrow", report.HTML)
	}
}

func TestCheckVisualIDConsistencyMissingInHTML(t *testing.T) {
	html := `<section id="viz-title"></section>`
	outline := `## ID inventory
| HTML id | Meaning |
| --- | --- |
| viz-title | Header |
| viz-open-questions | Open questions |
`
	report := CheckVisualIDConsistency(html, outline)
	if len(report.MissingInHTML) != 1 || report.MissingInHTML[0] != "viz-open-questions" {
		t.Fatalf("MissingInHTML = %#v, want [viz-open-questions]", report.MissingInHTML)
	}
}

func TestCheckVisualIDConsistencyMissingOutlineSection(t *testing.T) {
	report := CheckVisualIDConsistency(`<div id="x"></div>`, "# Title\n\ndemo\n")
	if len(report.Inventory) != 0 || len(report.MissingInHTML) != 0 {
		t.Fatalf("report = %#v, want empty inventory/missing", report)
	}
}

func TestOpenVisualWarnsIDMismatch(t *testing.T) {
	root, html := writeOpenVisualFixture(t, true)
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(root, "work", "feature"))

	outline := strings.TrimSuffix(html, ".html") + ".outline.md"
	writeBasenameFile(t, filepath.Dir(outline), filepath.Base(outline), `# demo

## ID inventory
| HTML id | Meaning |
| --- | --- |
| viz-title | Header |
| missing-node | Absent from HTML |
`)
	writeBasenameFile(t, filepath.Dir(html), filepath.Base(html), `<!doctype html><title>demo</title><section id="viz-title"></section>
`)

	restore := swapOpenVisualOpener(func(string) error { return nil })
	defer restore()

	var stdout, stderr bytes.Buffer
	code := OpenVisual(root, []string{"demo", "--no-open"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("OpenVisual() = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "outline inventory id(s) missing from HTML") {
		t.Fatalf("stderr = %q, want missing-id warning", stderr.String())
	}
	if !strings.Contains(stderr.String(), "missing-node") {
		t.Fatalf("stderr = %q, want missing-node listed", stderr.String())
	}
	if !strings.Contains(stdout.String(), "open-visual: ids=mismatch") {
		t.Fatalf("stdout = %q, want ids=mismatch tip", stdout.String())
	}
}

func TestOpenVisualIDsOkTip(t *testing.T) {
	root, html := writeOpenVisualFixture(t, true)
	t.Setenv("DEVRITES_WORKSPACE", filepath.Join(root, "work", "feature"))

	outline := strings.TrimSuffix(html, ".html") + ".outline.md"
	writeBasenameFile(t, filepath.Dir(outline), filepath.Base(outline), `# demo

## ID inventory
| HTML id | Meaning |
| --- | --- |
| viz-title | Header |
`)
	writeBasenameFile(t, filepath.Dir(html), filepath.Base(html), `<!doctype html><title>demo</title><section id="viz-title"></section><marker id="arrow"></marker>
`)

	restore := swapOpenVisualOpener(func(string) error { return nil })
	defer restore()

	var stdout, stderr bytes.Buffer
	code := OpenVisual(root, []string{"demo", "--no-open"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("OpenVisual() = %d stderr=%q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty when consistent", stderr.String())
	}
	if !strings.Contains(stdout.String(), "open-visual: ids=ok (1 inventory)") {
		t.Fatalf("stdout = %q, want ids=ok tip", stdout.String())
	}
}

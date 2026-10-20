package state

import (
	"strings"

	"slices"
	"strconv"
	"testing"
)

func TestArtifactBudgetMeasuresLinesBytesAndMateriality(t *testing.T) {
	if _, ok := ArtifactBudget("notes.json", []byte("x")); ok {
		t.Fatal("non-canonical files have no budget")
	}
	within, ok := ArtifactBudget("tasks.md", []byte("# Tasks\n## SLICE-001 A\nDependencies: none\n"))
	if !ok || within.Over || within.Material {
		t.Fatalf("within=%+v", within)
	}
	over, _ := ArtifactBudget("tasks.md", []byte(strings.Repeat("x\n", 300)))
	if !over.Over || over.Material {
		t.Fatalf("300 lines against a 280-line budget must be over but not material: %+v", over)
	}
	material, _ := ArtifactBudget("tasks.md", []byte(strings.Repeat("x\n", 500)))
	if !material.Material {
		t.Fatalf("500 lines must be material: %+v", material)
	}
	longLine, _ := ArtifactBudget("state.md", []byte(strings.Repeat("y", 100_000)+"\n"))
	if !longLine.Material || longLine.Lines != 1 {
		t.Fatalf("one very long line must trip the byte budget: %+v", longLine)
	}
}

func hasOverride(raw string) bool { return BudgetOverrideLines([]byte(raw)) > 0 }

func TestBudgetOverrideLinesRequiresBoundedNumberAndReason(t *testing.T) {
	for _, c := range []struct {
		line string
		want int
	}{
		{"Budget override: <=420 lines; vendored requirement matrix", 420},
		{"Budget override: <=420; vendored requirement matrix", 420},
		{"Budget override: <= 12 line ; table form", 12},
		{"Budget override: vendored requirement matrix, relocation pending", 0},
		{"Budget override: <=420 lines", 0},
		{"Budget override: <=0 lines; nothing", 0},
		{"Budget override: 420 lines; no comparator", 0},
	} {
		if got := BudgetOverrideLines([]byte("# Tasks\n" + c.line + "\n")); got != c.want {
			t.Errorf("%q: got %d want %d", c.line, got, c.want)
		}
	}
}

// An override covers the artifact only up to its own N lines and N*400 bytes,
// and N above BudgetOverrideFactor times the line budget never counts.
func TestArtifactBudgetOverrideIsBoundedAtTwiceTheBudget(t *testing.T) {
	withOverride := func(n, lines int, pad string) []byte {
		head := "Budget override: <=" + strconv.Itoa(n) + " lines; vendored requirement matrix\n" + pad
		return []byte(head + strings.Repeat("x\n", lines-1))
	}
	for _, c := range []struct {
		name         string
		raw          []byte
		wantN        int
		wantOverride bool
	}{
		{"material within N", withOverride(500, 500, ""), 500, true},
		{"one line past N", withOverride(500, 501, ""), 500, false},
		{"N at the 2x cap", withOverride(560, 560, ""), 560, true},
		{"N above the 2x cap", withOverride(561, 450, ""), 0, false},
		{"bytes past N*400", withOverride(500, 10, strings.Repeat("y", 200_001)), 500, false},
		{"legacy free-text override", []byte("Budget override: relocation pending\n" + strings.Repeat("x\n", 450)), 0, false},
	} {
		status, _ := ArtifactBudget("tasks.md", c.raw)
		if !status.Material || status.OverrideLines != c.wantN || status.Override != c.wantOverride {
			t.Errorf("%s: %+v, want material, N=%d, override=%v", c.name, status, c.wantN, c.wantOverride)
		}
	}
}

func TestBudgetGatedArtifactsCoversUnrequiredLedgersOnly(t *testing.T) {
	gated := BudgetGatedArtifacts()
	if !slices.IsSorted(gated) || !slices.Contains(gated, "drift.md") || !slices.Contains(gated, "tasks.md") {
		t.Fatalf("gated=%v", gated)
	}
	for _, advisory := range []ArtifactPath{"evidence.md", "browser-evidence.md", "touched-files.md"} {
		if slices.Contains(gated, advisory) {
			t.Errorf("proof ledger %s must stay advisory", advisory)
		}
	}
}

func TestBudgetOverrideIgnoresFencedExamples(t *testing.T) {
	if !hasOverride("plain text\nBudget override: <=300 lines; relocation pending\n") {
		t.Fatal("structural override line must be recognized")
	}
	fenced := "See the schema:\n\n```markdown\nBudget override: <reason>\n```\n\nNothing recorded.\n"
	if hasOverride(fenced) {
		t.Fatal("a fenced example is not an override")
	}
	indented := "  Budget override: <=300; table form is required\n"
	if !hasOverride(indented) {
		t.Fatal("leading whitespace must not hide a real override line")
	}
	for _, bare := range []string{"x\nBudget override:\n", "x\nBudget override:   \n", "x\n  Budget override:\t\n", "x\nBudget override: <=300 lines;\n"} {
		if hasOverride(bare) {
			t.Fatalf("an override line with no reason must not count: %q", bare)
		}
	}
}

// Every fence style that opens a CommonMark fenced block must reach the same
// fenced-code guard, so a quoted example can never record an override.
func TestBudgetOverrideMasksEveryFenceStyle(t *testing.T) {
	for _, fence := range []string{"```", "~~~", "````", "~~~markdown", "   ~~~"} {
		t.Run(fence, func(t *testing.T) {
			closer := strings.TrimRight(fence, "abcdefghijklmnopqrstuvwxyz")
			closed := "Intro paragraph.\n\n" + fence + "\nBudget override: <reason>\n" + closer + "\n\nClosing prose.\n"
			if hasOverride(closed) {
				t.Fatalf("fence %q: a closed fenced example is not a structural override", fence)
			}
			unterminated := "Intro paragraph.\n\n" + fence + "\nBudget override: <reason>\n\nStill open.\n"
			if hasOverride(unterminated) {
				t.Fatalf("fence %q: an unterminated fence masks to end of artifact", fence)
			}
			after := "Intro paragraph.\n\n" + fence + "\nSchema reference.\n" + closer + "\n\nBudget override: <=300 lines; relocation pending\n"
			if !hasOverride(after) {
				t.Fatalf("fence %q: the guard must close and a later real override line still counts", fence)
			}
		})
	}
}

func TestBudgetOverrideFenceStyleIsNotOutcomeBearing(t *testing.T) {
	bodies := []string{
		"Budget override: <=300 lines; relocation pending\n",
		"text\nBudget override: <=300; <reason>\n```\nmore\n```\nBudget override: <=300; second\n",
	}
	for _, body := range bodies {
		backtick := "```\n" + body + "```\n"
		tilde := "~~~\n" + body + "~~~\n"
		if got, want := hasOverride(tilde), hasOverride(backtick); got != want {
			t.Fatalf("fence style changed the outcome for %q: tilde=%v backtick=%v", body, got, want)
		}
	}
}

func TestBudgetOverrideNestedAndIndentedFences(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"backtick block quoted inside tilde fence", "~~~markdown\n```\nBudget override: <reason>\n```\n~~~\n", false},
		{"tilde block quoted inside backtick fence", "```markdown\n~~~\nBudget override: <reason>\n~~~\n```\n", false},
		{"four-space indent is not a fence", "    ~~~\nBudget override: <=300 lines; real\n", true},
		{"malformed markdown has no structural override", "Budget override: <=300 lines; real\n\x00", false},
	}
	for _, c := range cases {
		if got := hasOverride(c.raw); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestAcceptanceSectionIgnoresFencedExamples(t *testing.T) {
	spec := []byte("# Spec\n\nHere is the shape we want:\n\n```markdown\n## Acceptance criteria\n- AC-900: phantom\n```\n\nNo real section exists.\n")
	got := ParseAcceptanceMap(spec, []byte("no ids here"), []byte("no ids here"), true, true)
	if len(got.SpecIDs) != 0 {
		t.Fatalf("a fenced example must not fabricate acceptance ids: ids=%v", got.SpecIDs)
	}
	if len(got.Problems) != 0 {
		t.Fatalf("no real section means no required ids: problems=%v", got.Problems)
	}
}

func TestAcceptanceSectionKeepsRealSectionPastFencedHeadings(t *testing.T) {
	spec := []byte("# Spec\n\n## Acceptance criteria\n\n```markdown\n## Non-goals\n- AC-900: phantom\n```\n\n- AC-001: export CSV\n\n## Non-goals\n- AC-998: outside\n")
	tasks := []byte("Satisfies: AC-001\n")
	plan := []byte("AC-001\n")
	got := ParseAcceptanceMap(spec, tasks, plan, true, true)
	if !slices.Equal(got.SpecIDs, []string{"AC-001"}) {
		t.Fatalf("a fenced heading must not truncate the real section: ids=%v", got.SpecIDs)
	}
	if len(got.Problems) != 0 {
		t.Fatalf("problems=%v", got.Problems)
	}
}

func TestAcceptanceSectionIgnoresMalformedMarkdown(t *testing.T) {
	spec := []byte("## Acceptance criteria\n- AC-001: export CSV\n\x00")
	got := ParseAcceptanceMap(spec, []byte("no ids here"), []byte("no ids here"), true, true)
	if len(got.SpecIDs) != 0 || len(got.Problems) != 0 {
		t.Fatalf("malformed input carries no usable section: ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}

// Engine-owned machine files and documented conditional artifacts grow in the
// workspace root by design; they must never trip the placement gate.
func TestSanctionedRootFileCoversEngineAndConditionalFiles(t *testing.T) {
	for _, name := range []string{"dispatch.json", "readiness-inputs.json", "tree-baseline.json", "devex.md"} {
		if !SanctionedRootFile(name) {
			t.Errorf("%s is not sanctioned in the workspace root", name)
		}
	}
	if SanctionedRootFile("stray-transcript.md") {
		t.Error("an undocumented root file is sanctioned")
	}
}

package state

import (
	"strings"

	"slices"
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

func TestHasBudgetOverrideIgnoresFencedExamples(t *testing.T) {
	if HasBudgetOverride([]byte("plain text\nBudget override: relocation pending\n")) != true {
		t.Fatal("structural override line must be recognized")
	}
	fenced := "See the schema:\n\n```markdown\nBudget override: <reason>\n```\n\nNothing recorded.\n"
	if HasBudgetOverride([]byte(fenced)) {
		t.Fatal("a fenced example is not an override")
	}
	indented := "  Budget override: table form is required\n"
	if !HasBudgetOverride([]byte(indented)) {
		t.Fatal("leading whitespace must not hide a real override line")
	}
	for _, bare := range []string{"x\nBudget override:\n", "x\nBudget override:   \n", "x\n  Budget override:\t\n"} {
		if HasBudgetOverride([]byte(bare)) {
			t.Fatalf("an override line with no reason must not count: %q", bare)
		}
	}
}

// Every fence style that opens a CommonMark fenced block must reach the same
// fenced-code guard, so a quoted example can never record an override.
func TestHasBudgetOverrideMasksEveryFenceStyle(t *testing.T) {
	for _, fence := range []string{"```", "~~~", "````", "~~~markdown", "   ~~~"} {
		t.Run(fence, func(t *testing.T) {
			closer := strings.TrimRight(fence, "abcdefghijklmnopqrstuvwxyz")
			closed := "Intro paragraph.\n\n" + fence + "\nBudget override: <reason>\n" + closer + "\n\nClosing prose.\n"
			if HasBudgetOverride([]byte(closed)) {
				t.Fatalf("fence %q: a closed fenced example is not a structural override", fence)
			}
			unterminated := "Intro paragraph.\n\n" + fence + "\nBudget override: <reason>\n\nStill open.\n"
			if HasBudgetOverride([]byte(unterminated)) {
				t.Fatalf("fence %q: an unterminated fence masks to end of artifact", fence)
			}
			after := "Intro paragraph.\n\n" + fence + "\nSchema reference.\n" + closer + "\n\nBudget override: relocation pending\n"
			if !HasBudgetOverride([]byte(after)) {
				t.Fatalf("fence %q: the guard must close and a later real override line still counts", fence)
			}
		})
	}
}

func TestHasBudgetOverrideFenceStyleIsNotOutcomeBearing(t *testing.T) {
	bodies := []string{
		"Budget override: relocation pending\n",
		"text\nBudget override: <reason>\n```\nmore\n```\nBudget override: second\n",
	}
	for _, body := range bodies {
		backtick := "```\n" + body + "```\n"
		tilde := "~~~\n" + body + "~~~\n"
		if got, want := HasBudgetOverride([]byte(tilde)), HasBudgetOverride([]byte(backtick)); got != want {
			t.Fatalf("fence style changed the outcome for %q: tilde=%v backtick=%v", body, got, want)
		}
	}
}

func TestHasBudgetOverrideNestedAndIndentedFences(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"backtick block quoted inside tilde fence", "~~~markdown\n```\nBudget override: <reason>\n```\n~~~\n", false},
		{"tilde block quoted inside backtick fence", "```markdown\n~~~\nBudget override: <reason>\n~~~\n```\n", false},
		{"four-space indent is not a fence", "    ~~~\nBudget override: real\n", true},
		{"malformed markdown has no structural override", "Budget override: real\n\x00", false},
	}
	for _, c := range cases {
		if got := HasBudgetOverride([]byte(c.raw)); got != c.want {
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

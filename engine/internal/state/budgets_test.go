package state

import (
	"strings"
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
}

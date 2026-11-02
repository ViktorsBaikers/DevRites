package state

import (
	"bytes"
	"slices"
	"testing"

	"github.com/devrites/devrites/internal/markdowntext"
)

func TestParseAcceptanceMapEmptySpecPasses(t *testing.T) {
	got := ParseAcceptanceMap([]byte("# Spec\n\nReady.\n"), []byte("tasks"), []byte("plan"), true, true)
	if len(got.SpecIDs) != 0 || len(got.Problems) != 0 {
		t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}

func TestParseAcceptanceMapRequiresTasksAndTestPlan(t *testing.T) {
	spec := []byte("# Spec\n\n## Acceptance criteria\n- [ ] AC-001: export CSV\n- [ ] AC-002: reject cross-user\n")
	tasks := []byte("# Tasks\nSatisfies: AC-001\n")
	plan := []byte("# Test plan\nAC-001 → T1\n")
	got := ParseAcceptanceMap(spec, tasks, plan, true, true)
	if !slices.Equal(got.SpecIDs, []string{"AC-001", "AC-002"}) {
		t.Fatalf("ids=%v", got.SpecIDs)
	}
	if !slices.Equal(got.Problems, []string{
		"acceptance AC-002 is not referenced in tasks.md",
		"acceptance AC-002 is not referenced in test-plan.md",
	}) {
		t.Fatalf("problems=%v", got.Problems)
	}
}

func TestParseAcceptanceMapSkipsUnrequiredArtifacts(t *testing.T) {
	spec := []byte("## Acceptance Criteria\n- AC-001: bind readiness inputs.\n")
	got := ParseAcceptanceMap(spec, nil, nil, false, false)
	if !slices.Equal(got.SpecIDs, []string{"AC-001"}) || len(got.Problems) != 0 {
		t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}

func TestParseAcceptanceMapIgnoresNoncanonicalAndDuplicates(t *testing.T) {
	spec := []byte("## Acceptance criteria\n- AC-001: one\n- AC1: ignored\n- AC-001: again\n")
	tasks := []byte("Satisfies: AC-001\n")
	plan := []byte("AC-001\n")
	got := ParseAcceptanceMap(spec, tasks, plan, true, true)
	if !slices.Equal(got.SpecIDs, []string{"AC-001"}) || len(got.Problems) != 0 {
		t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}

func TestParseAcceptanceMapStopsAtNextH2(t *testing.T) {
	spec := []byte("## Acceptance criteria\n- AC-001: in section\n\n## Non-goals\n- AC-999: outside\n")
	tasks := []byte("AC-001\n")
	plan := []byte("AC-001\n")
	got := ParseAcceptanceMap(spec, tasks, plan, true, true)
	if !slices.Equal(got.SpecIDs, []string{"AC-001"}) || len(got.Problems) != 0 {
		t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
	}
}

func TestAcceptanceDeclarationsOutsideSection(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
	}{
		{"checkbox", "- [ ] AC-002: required outcome"},
		{"checked checkbox", "- [x] AC-002: required outcome"},
		{"bullet", "- AC-002: required outcome"},
		{"numbered", "1. AC-002: required outcome"},
		{"heading", "### AC-002: required outcome"},
		{"bare", "AC-002: required outcome"},
		{"bold", "- **AC-002**: required outcome"},
		{"code", "- `AC-002`: required outcome"},
		{"table", "| AC-002 | required outcome |"},
		{"dash", "- AC-002 — required outcome"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := []byte(tc.line + "\n\n## Acceptance criteria\n" + tc.line + "\n\n## Notes\n" + tc.line + "\n")
			if got := AcceptanceDeclarationsOutsideSection(spec); !slices.Equal(got, []string{"AC-002", "AC-002"}) {
				t.Fatalf("outside declarations=%v", got)
			}
		})
	}
}

func TestAcceptanceIgnoresCommentedAndFencedExamples(t *testing.T) {
	for _, tc := range []struct {
		name    string
		example string
	}{
		{"comment", "<!--\n## Acceptance criteria\n- [ ] AC-099: example only\n-->\n"},
		{"comment containing fence", "<!--\n```markdown\n## Acceptance criteria\n- [ ] AC-099: example only\n-->\n"},
		{"fence containing comment opener", "```markdown\n<!--\n## Acceptance criteria\n- [ ] AC-099: example only\n```\n"},
		{"inline code comment opener", "`<!--`\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := []byte(tc.example + "\n- AC-002: actual preheading obligation\n\n## Acceptance criteria\n- AC-001: real criterion\n")
			got := ParseAcceptanceMap(spec, []byte("AC-001"), []byte("AC-001"), true, true)
			if !slices.Equal(got.SpecIDs, []string{"AC-001"}) || len(got.Problems) != 0 {
				t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
			}
			if outside := AcceptanceDeclarationsOutsideSection(spec); !slices.Equal(outside, []string{"AC-002"}) {
				t.Fatalf("outside declarations=%v", outside)
			}
		})
	}
}

func TestAcceptanceVisibleCommentSuffixNeedsMapping(t *testing.T) {
	for _, line := range []string{
		"<!-- editorial AC-099 --> AC-002: required",
		"<!-- editorial AC-099\n--> AC-002: required",
		"<!-- first --> <!-- second --> AC-002: required",
		"AC-002: required <!-- editorial AC-099 -->",
	} {
		t.Run(line, func(t *testing.T) {
			spec := []byte("## Acceptance criteria\nAC-001: mapped\n" + line + "\n")
			got := ParseAcceptanceMap(spec, []byte("AC-001"), []byte("AC-001"), true, true)
			if !slices.Equal(got.SpecIDs, []string{"AC-001", "AC-002"}) || !slices.Equal(got.Problems, []string{
				"acceptance AC-002 is not referenced in tasks.md",
				"acceptance AC-002 is not referenced in test-plan.md",
			}) {
				t.Fatalf("ids=%v problems=%v", got.SpecIDs, got.Problems)
			}
		})
	}
}

func TestAcceptanceProseInlineCommentAlignment(t *testing.T) {
	input := []byte("prefix <!-- hidden --> suffix <!-- second --> end\r\n<!-- multi\r\nhidden --> tail\r\n")
	want := []byte("prefix                 suffix                 end\r\n          \r\n           tail\r\n")
	got, err := markdowntext.Prose(input)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("prose=%q, want %q; error=%v", got, want, err)
	}
	again, err := markdowntext.Prose(got)
	if err != nil || !bytes.Equal(again, got) {
		t.Fatalf("prose mask is not idempotent: %q, %v", again, err)
	}
	structural, err := markdowntext.Structural(input)
	if err != nil || !bytes.Equal(structural, input) {
		t.Fatalf("structural comment markers changed: %q, %v", structural, err)
	}
}

func TestAcceptanceProseMaskPreservesStructuralContract(t *testing.T) {
	input := []byte("<!--\r\n- AC-099: example\r\n-->\r\n## Acceptance criteria\r\n- AC-001: real\r\n")
	got, err := markdowntext.Prose(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("    \r\n                 \r\n   \r\n## Acceptance criteria\r\n- AC-001: real\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("prose=%q, want %q", got, want)
	}
	again, err := markdowntext.Prose(got)
	if err != nil || !bytes.Equal(again, got) {
		t.Fatalf("prose mask is not idempotent: %q, %v", again, err)
	}
	structural, err := markdowntext.Structural(input)
	if err != nil || !bytes.Equal(structural, input) {
		t.Fatalf("structural comment markers changed: %q, %v", structural, err)
	}
	for _, malformed := range [][]byte{{0xff}, {0}} {
		if _, err := markdowntext.Prose(malformed); err == nil {
			t.Fatalf("malformed prose %q accepted", malformed)
		}
	}
}

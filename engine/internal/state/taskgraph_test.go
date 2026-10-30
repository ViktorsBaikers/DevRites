package state

import (
	"strings"
	"testing"
)

const fenceLive = "## SLICE-001\nDependencies: SLICE-002\n\n## SLICE-002\nDependencies: none\n"

const fenceDoc = "## SLICE-001\nDependencies: SLICE-002\n\nExample layout:\n\n```markdown\n## SLICE-002\nDependencies: none\n```\n"

func TestParseTaskGraphUnfencedControlYieldsTwoSlices(t *testing.T) {
	got := ParseTaskGraph([]byte(fenceLive))
	if len(got.Slices) != 2 || len(got.Unknown) != 0 {
		t.Fatalf("slices=%v unknown=%v, want two slices and no unknown", got.Slices, got.Unknown)
	}
}

func TestParseTaskGraphIgnoresFencedSliceHeader(t *testing.T) {
	got := ParseTaskGraph([]byte(fenceDoc))
	if len(got.Slices) != 1 || got.Slices[0].ID != "SLICE-001" {
		t.Fatalf("slices=%v, want only SLICE-001", got.Slices)
	}
	if len(got.Unknown) != 1 || got.Unknown[0] != "SLICE-002" {
		t.Fatalf("unknown=%v, want the dangling SLICE-002 reported", got.Unknown)
	}
}

func TestExtractTaskSliceIgnoresFencedSliceHeader(t *testing.T) {
	if _, ok := ExtractTaskSlice([]byte(fenceDoc), "SLICE-002"); ok {
		t.Fatal("fenced SLICE-002 reported as a live slice")
	}
	block, ok := ExtractTaskSlice([]byte(fenceDoc), "SLICE-001")
	if !ok {
		t.Fatal("live SLICE-001 not found")
	}
	if !strings.Contains(block, "```markdown") || !strings.Contains(block, "## SLICE-002") {
		t.Fatalf("live slice must keep its fenced example verbatim, got %q", block)
	}
	if got, ok := ExtractTaskSlice([]byte(fenceLive), "SLICE-002"); !ok || !strings.HasPrefix(got, "## SLICE-002") {
		t.Fatalf("unfenced control: got %q ok=%v", got, ok)
	}
}

func TestParseTaskGraphEmptyDependenciesDoesNotReadNextLine(t *testing.T) {
	got := ParseTaskGraph([]byte("## SLICE-001\nDependencies:\nConsumes / Produces: api\n"))
	for _, problem := range got.Problems {
		if strings.Contains(problem, "malformed") {
			t.Fatalf("problems=%v, the next line was read as dependencies", got.Problems)
		}
	}
	if len(got.Slices) != 1 || len(got.Slices[0].Dependencies) != 0 {
		t.Fatalf("slices=%v, want SLICE-001 with no dependencies", got.Slices)
	}
}

func TestParseTaskGraphKeepsViaPerEdge(t *testing.T) {
	doc := "## SLICE-001\nDependencies: none\nConsumes / Produces: produces Store.Get\n\n" +
		"## SLICE-003\nDependencies: none\n\n" +
		"## SLICE-002\nDependencies: SLICE-001 (via Store.Get), `SLICE-003` (via /api/items)\ndepends_on: [SLICE-001, SLICE-003]\n"
	got := ParseTaskGraph([]byte(doc))
	if len(got.Problems) != 0 {
		t.Fatalf("problems=%v, via suffixes and a bare depends_on mirror must agree", got.Problems)
	}
	s := got.Slices[2]
	if strings.Join(s.Dependencies, ",") != "SLICE-001,SLICE-003" {
		t.Fatalf("deps=%v, want bare ids", s.Dependencies)
	}
	if s.Via["SLICE-001"] != "Store.Get" || s.Via["SLICE-003"] != "/api/items" {
		t.Fatalf("via=%v", s.Via)
	}
	if got.Slices[0].ConsumesProduces != "produces Store.Get" || got.Slices[1].ConsumesProduces != "" {
		t.Fatalf("consumes/produces=%q,%q", got.Slices[0].ConsumesProduces, got.Slices[1].ConsumesProduces)
	}
}

func TestParseTaskGraphRejectsMalformedVia(t *testing.T) {
	for line, token := range map[string]string{
		"SLICE-001 (via )":            "(via )",
		"SLICE-001 (via api":          "(via api",
		"SLICE-001, (via api)":        "(via api)",
		"SLICE-001 (via a) (via b)":   "(via b)",
		"SLICE-001 (via ), SLICE-001": "(via )",
	} {
		got := ParseTaskGraph([]byte("## SLICE-001\nDependencies: none\n\n## SLICE-002\nDependencies: " + line + "\n"))
		want := `SLICE-002 has malformed dependency "` + token + `"`
		if strings.Join(got.Problems, "\n") != want {
			t.Errorf("Dependencies: %s -> problems=%q, want only %q", line, got.Problems, want)
		}
		if len(got.Slices) != 2 || strings.Join(got.Slices[1].Dependencies, ",") != "SLICE-001" {
			t.Errorf("Dependencies: %s -> slices=%+v, want SLICE-001 kept", line, got.Slices)
		}
	}
}

func TestParseTaskGraphWorkflowOnlyRequiresExplicitUnmixedScope(t *testing.T) {
	for _, tc := range []struct {
		name, body        string
		workflow, blocked bool
	}{
		{"explicit", "Work scope: workflow-only\n", true, false},
		{"CRLF", "Work scope: workflow-only\r\n", true, false},
		{"empty", "# Tasks\n", false, true},
		{"fenced", "```\nWork scope: workflow-only\n```\n", false, true},
		{"mixed", "Work scope: workflow-only\n## SLICE-001\nDependencies: none\n", false, true},
		{"duplicate", "Work scope: workflow-only\nWork scope: workflow-only\n", false, true},
		{"product", "## SLICE-001\nDependencies: none\n", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseTaskGraph([]byte(tc.body))
			if got.WorkflowOnly != tc.workflow || (len(got.Problems) > 0) != tc.blocked {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

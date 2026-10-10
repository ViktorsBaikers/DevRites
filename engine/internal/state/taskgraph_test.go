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

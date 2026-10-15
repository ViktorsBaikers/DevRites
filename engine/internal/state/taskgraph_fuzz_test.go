package state

import (
	"slices"
	"testing"
)

func FuzzTaskGraphParsing(f *testing.F) {
	f.Add([]byte("## SLICE-1\n\nDependencies: none\n"))
	f.Add([]byte("## SLICE-1\nDependencies: [SLICE-2]\n\n## SLICE-2\nDependencies: [SLICE-1]\n"))
	f.Add([]byte("## SLICE-1\nDependencies: none\ndepends_on: SLICE-9\n"))
	f.Add([]byte("## SLICE-1\n## SLICE-1\nDependencies: none\n"))
	f.Add([]byte("## SLICE-1\nDependencies: [oops, SLICE-2]\n\n## SLICE-2\ndepends_on: [SLICE-1]\n"))
	f.Add([]byte("## SLICE-42\nDependencies:\n"))
	f.Add([]byte("## SLICE-1\nDependencies: none\nConsumes / Produces: x\n\n## SLICE-2\nDependencies: SLICE-1 (via x), SLICE-3 (via \n"))
	f.Add([]byte("## SLICE-1\nDependencies: (via y) SLICE-1 (via ) (via z)\ndepends_on: [SLICE-1 (via q)]\n"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, doc []byte) {
		result := ParseTaskGraph(doc)

		known := make(map[string]bool, len(result.Slices))
		for _, slice := range result.Slices {
			if !sliceIDValidRE.MatchString(slice.ID) {
				t.Fatalf("slice id %q does not satisfy the id grammar", slice.ID)
			}
			if known[slice.ID] {
				t.Fatalf("slice id %q emitted twice", slice.ID)
			}
			known[slice.ID] = true
		}
		unknown := make(map[string]bool, len(result.Unknown))
		for _, dep := range result.Unknown {
			unknown[dep] = true
		}
		for _, slice := range result.Slices {
			for dep, via := range slice.Via {
				if via == "" || !slices.Contains(slice.Dependencies, dep) {
					t.Fatalf("slice %s via %q names %q outside its dependencies %v", slice.ID, via, dep, slice.Dependencies)
				}
			}
		}
		for _, slice := range result.Slices {
			for _, dep := range slice.Dependencies {
				if !known[dep] && !unknown[dep] {
					t.Fatalf("slice %s depends on %q which is neither defined nor reported unknown", slice.ID, dep)
				}
			}
		}
	})
}

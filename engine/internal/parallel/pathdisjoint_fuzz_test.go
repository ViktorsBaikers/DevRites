package parallel

import (
	"strings"
	"testing"
)

func FuzzPathDisjoint(f *testing.F) {
	f.Add("src/a.go", []byte(`{"slices":[{"id":"a","paths":["src/a.go"]},{"id":"b","paths":["src/b.go"]}]}`))
	f.Add("../escape", []byte(`[{"id":"a","paths":["/abs"]},{"id":"b","paths":[".devrites/state.md"]}]`))
	f.Add("src\\windows.go", []byte(`{"slices":[{"id":"a","paths":["src", "src"]}]}`))
	f.Add("src//double/", []byte(`not json`))
	f.Add(".devrites", []byte(`{"slices":[]}`))
	f.Fuzz(func(t *testing.T, raw string, doc []byte) {
		normalized, err := NormalizePath(raw)
		if err == nil {
			if normalized == "" || strings.HasPrefix(normalized, "/") || strings.Contains(normalized, "\\") {
				t.Fatalf("NormalizePath accepted %q into %q", raw, normalized)
			}
			for _, part := range strings.Split(normalized, "/") {
				if part == "" || part == "." || part == ".." {
					t.Fatalf("NormalizePath accepted %q into %q with bad segment %q", raw, normalized, part)
				}
			}
			again, err := NormalizePath(normalized)
			if err != nil || again != normalized {
				t.Fatalf("NormalizePath not idempotent: %q -> %q (err %v)", normalized, again, err)
			}
		}

		slices, err := ParseSlicesJSON(doc)
		if err != nil {
			return
		}
		ids, err := CheckPathDisjoint(slices, "")
		if err != nil {
			return
		}
		if len(ids) != len(slices) {
			t.Fatalf("CheckPathDisjoint returned %d ids for %d slices", len(ids), len(slices))
		}
	})
}

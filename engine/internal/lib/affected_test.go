package lib

import "testing"

func TestSelectAffectedCounts(t *testing.T) {
	cases := []struct {
		name     string
		changed  []string
		pkgs     int
		widen    bool
		reasonIn string
	}{
		{
			name:    "one go file selects exactly one package",
			changed: []string{"engine/internal/lib/affected.go"},
			pkgs:    1,
		},
		{
			name:    "three files in one directory collapse to one package",
			changed: []string{"engine/internal/lib/a.go", "engine/internal/lib/b.go", "engine/internal/lib/c_test.go"},
			pkgs:    1,
		},
		{
			name:    "two directories select two packages",
			changed: []string{"engine/internal/lib/a.go", "engine/internal/parallel/b.go"},
			pkgs:    2,
		},
		{
			name:     "go.mod widens because it can change every package",
			changed:  []string{"engine/internal/lib/a.go", "go.mod"},
			widen:    true,
			reasonIn: "go.mod",
		},
		{
			name:     "ci config widens because it is not inside a package",
			changed:  []string{".github/workflows/test.yml"},
			widen:    true,
			reasonIn: ".github",
		},
		{
			name:     "non-go change widens and names the path",
			changed:  []string{"docs/adr/0013-parallel.md"},
			widen:    true,
			reasonIn: "docs/adr/0013-parallel.md",
		},
		{
			name:     "nested manifest widens because the module is not only the root",
			changed:  []string{"engine/go.mod"},
			widen:    true,
			reasonIn: "is a module manifest",
		},
		{
			name:     "nested go.sum widens",
			changed:  []string{"engine/go.sum"},
			widen:    true,
			reasonIn: "is a module manifest",
		},
		{
			name:     "unclassified path is not dropped beside a go file",
			changed:  []string{"engine/internal/lib/a.go", "engine/templates/role.tmpl"},
			widen:    true,
			reasonIn: "role.tmpl",
		},
		{
			name:     "empty input widens",
			changed:  nil,
			widen:    true,
			reasonIn: "no Go package",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SelectAffected(tc.changed)
			if got.Widen != tc.widen {
				t.Fatalf("widen = %v, want %v (reason %q)", got.Widen, tc.widen, got.Reason)
			}
			if tc.widen {
				if tc.reasonIn == "" || !contains(got.Reason, tc.reasonIn) {
					t.Fatalf("reason = %q, want it to mention %q", got.Reason, tc.reasonIn)
				}
				return
			}
			if len(got.Packages) != tc.pkgs {
				t.Fatalf("selected %d packages (%v), want %d", len(got.Packages), got.Packages, tc.pkgs)
			}
		})
	}
}

func TestExpandDependents(t *testing.T) {
	rows := []pkgDeps{
		{Import: "m/parallel", Dir: "/r/parallel", Deps: []string{"m/util"}},
		{Import: "m/lib", Dir: "/r/lib", Deps: []string{"m/parallel", "m/util"}},
		{Import: "m/cmd", Dir: "/r/cmd", Deps: []string{"m/lib"}},
		{Import: "m/unrelated", Dir: "/r/unrelated"},
	}
	got := expandDependents([]string{"m/parallel"}, rows)
	want := []string{"m/cmd", "m/lib", "m/parallel"}
	if len(got) != len(want) {
		t.Fatalf("expandDependents = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expandDependents = %v, want %v", got, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

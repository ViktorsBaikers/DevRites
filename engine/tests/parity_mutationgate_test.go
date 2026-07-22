package main_test

import (
	"path/filepath"
	"testing"
)

// TestParityMutationGate runs mutation-gate in a NON-git temp dir with no
// manifests so it resolves gitroot="" and detects no runner: the deterministic
// advisory-skip path, and checks stdout + exit against the golden snapshot. (If
// the host happens to have a runner on PATH, it is detected via the same lookup,
// so the golden check still holds.)
func TestParityMutationGate(t *testing.T) {
	work := t.TempDir()

	// The workspace directory must exist for the gate to proceed past its exit-2.
	makeFeatureDir(t, work, "feat")
	writeFile(t, work, filepath.Join(".devrites", "ACTIVE"), "feat\n")

	cases := []struct {
		name string
		slug []string // extra args after the subcommand
	}{
		{"explicit-slug", []string{"feat"}},
		{"active-slug", nil},         // resolves slug from ACTIVE
		{"ghost", []string{"ghost"}}, // no workspace -> exit 2
	}
	for _, tc := range cases {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  append([]string{"mutation-gate"}, tc.slug...),
		}
		t.Run(tc.name, func(t *testing.T) { c.assertEqual(t) })
	}
}

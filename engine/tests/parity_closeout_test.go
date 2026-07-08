package main_test

// Golden snapshot for the ported close-out command (lib.CloseOut).
//
// close-out MUTATES the tree — it moves .devrites/features/<slug> to
// .devrites/archive/<slug> and, when it is the active slug, clears
// .devrites/ACTIVE. Each case snapshots the command's stdout + exit code (stderr
// is diagnostic, not contract) and then asserts the post-move filesystem shape
// with plain Go checks.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParityCloseOut(t *testing.T) {
	dirExists := func(path string) bool {
		fi, err := os.Stat(path)
		return err == nil && fi.IsDir()
	}
	gone := func(path string) bool {
		_, err := os.Stat(path)
		return os.IsNotExist(err)
	}

	// run executes close-out in gw and snapshots its stdout + exit code under the
	// current subtest's golden.
	run := func(t *testing.T, gw string, args ...string) {
		t.Helper()
		out, code := runArgv(t, gw, libEnv, "", binPath, append([]string{"close-out"}, args...)...)
		assertGolden(t, out, code)
	}

	t.Run("active_slug_cleared", func(t *testing.T) {
		slug := "feat-x"
		gw := t.TempDir()
		writeFile(t, gw, filepath.Join(".devrites/features", slug, "spec.md"), "# spec\n")
		writeFile(t, gw, ".devrites/ACTIVE", slug+"\n")

		run(t, gw, slug, ".devrites")

		// archive created, features/ source gone, ACTIVE truncated.
		if !dirExists(filepath.Join(gw, ".devrites/archive", slug)) {
			t.Error("archive dir not created")
		}
		if !gone(filepath.Join(gw, ".devrites/features", slug)) {
			t.Error("source workspace not moved")
		}
		if b, err := os.ReadFile(filepath.Join(gw, ".devrites/ACTIVE")); err != nil {
			t.Fatalf("reading ACTIVE: %v", err)
		} else if len(b) != 0 {
			t.Errorf("ACTIVE not cleared: %q", string(b))
		}
	})

	t.Run("active_pointed_elsewhere", func(t *testing.T) {
		slug := "feat-y"
		other := "someone-else\n"
		gw := t.TempDir()
		writeFile(t, gw, filepath.Join(".devrites/features", slug, "spec.md"), "# spec\n")
		writeFile(t, gw, ".devrites/ACTIVE", other)

		run(t, gw, slug, ".devrites")

		// The archive moved, but ACTIVE (pointed elsewhere) is left untouched.
		if !dirExists(filepath.Join(gw, ".devrites/archive", slug)) {
			t.Error("archive dir not created")
		}
		if b, err := os.ReadFile(filepath.Join(gw, ".devrites/ACTIVE")); err != nil {
			t.Fatalf("reading ACTIVE: %v", err)
		} else if string(b) != other {
			t.Errorf("ACTIVE changed: %q", string(b))
		}
	})

	t.Run("missing_workspace", func(t *testing.T) {
		// No workspace seeded -> non-zero exit, empty stdout.
		gw := t.TempDir()
		run(t, gw, "ghost", ".devrites")
	})

	t.Run("archive_clobber", func(t *testing.T) {
		slug := "feat-z"
		gw := t.TempDir()
		writeFile(t, gw, filepath.Join(".devrites/features", slug, "spec.md"), "# spec\n")
		writeFile(t, gw, filepath.Join(".devrites/archive", slug, "old.md"), "old\n")

		run(t, gw, slug, ".devrites")

		// Refused: source stays in place, the pre-existing archive is not clobbered.
		if !dirExists(filepath.Join(gw, ".devrites/features", slug)) {
			t.Error("source workspace was moved despite clobber refusal")
		}
	})

	t.Run("missing_slug", func(t *testing.T) {
		// No slug arg -> non-zero exit.
		gw := t.TempDir()
		run(t, gw)
	})
}

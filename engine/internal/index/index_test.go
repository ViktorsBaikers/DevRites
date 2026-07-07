package index

// Focused unit tests on the index core: roundtrip parity with files-only status,
// staleness detection, and drop-and-rebuild on a corrupt/mismatched DB. These
// reach unexported state (isStale, schema) that the CLI black-box can't.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/testutil"
)

// workspace copies the committed fixture into a temp dir and returns the copied
// root.
func workspace(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "fixtures", "basic", "devrites-root")
	root := filepath.Join(t.TempDir(), ".devrites")
	testutil.CopyTree(t, src, root)
	return root
}

func TestReindexStatusMatchesFilesOnly(t *testing.T) {
	root := workspace(t)
	if _, err := Reindex(root); err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"auth-tokens", "search-ranking"} {
		fromIndex, err := Status(root, slug)
		if err != nil {
			t.Fatalf("index Status(%q): %v", slug, err)
		}
		fromFiles, err := state.Status(root, slug)
		if err != nil {
			t.Fatalf("files Status(%q): %v", slug, err)
		}
		if fromIndex.Render() != fromFiles.Render() {
			t.Errorf("%s: index render != files render\nindex:\n%s\nfiles:\n%s",
				slug, fromIndex.Render(), fromFiles.Render())
		}
	}
}

// TestFingerprintTracksEdits proves the staleness primitive: a feature's
// fingerprint is stable across a re-read but changes after a hand-edit, which is
// what lets Status heal a stale cache so a human edit always wins.
func TestFingerprintTracksEdits(t *testing.T) {
	root := workspace(t)

	before, err := fingerprint(root, "auth-tokens")
	if err != nil {
		t.Fatal(err)
	}
	// A second read with no change must be identical (deterministic fingerprint).
	if again, err := fingerprint(root, "auth-tokens"); err != nil || again != before {
		t.Fatalf("fingerprint not stable across a re-read: %q vs %q (err %v)", before, again, err)
	}

	// Give the file real content; the fingerprint (content-hashed) must change.
	tasks := filepath.Join(root, "features", "auth-tokens", "tasks.md")
	if err := os.WriteFile(tasks, []byte("# Tasks\n\n- [x] mint\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := fingerprint(root, "auth-tokens")
	if err != nil {
		t.Fatal(err)
	}
	if after == before {
		t.Error("fingerprint unchanged after a hand-edit, want it to change")
	}
}

func TestOpenRebuildsUnreadableDB(t *testing.T) {
	root := workspace(t)
	if _, err := Reindex(root); err != nil {
		t.Fatal(err)
	}
	// Clobber the DB with garbage, then Open must drop and rebuild rather than
	// error or serve stale data.
	if err := os.WriteFile(filepath.Join(root, DBName), []byte("garbage, not sqlite"), 0o644); err != nil {
		t.Fatal(err)
	}
	ix, err := Open(root)
	if err != nil {
		t.Fatalf("Open on corrupt DB: %v, want a clean rebuild", err)
	}
	defer ix.Close()
	if !ix.schemaCurrent() {
		t.Error("schema not current after rebuild")
	}
}

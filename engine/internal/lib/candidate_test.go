package lib

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type candidateTestRow struct {
	state  string
	path   string
	slice  string
	reason string
}

func candidateTestManifest(rows ...candidateTestRow) string {
	var body strings.Builder
	body.WriteString("# Touched files\n\n## Touched files\nCandidate paths are declared only in the manifest below.\n\n## Candidate manifest\n")
	if len(rows) == 0 {
		body.WriteString("No project files.\n")
		return body.String()
	}
	body.WriteString("| State | File | Slice | Reason |\n| --- | --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(&body, "| %s | `%s` | %s | %s |\n", row.state, row.path, row.slice, row.reason)
	}
	return body.String()
}

func writeCandidateTestWorkspace(t *testing.T, manifest string) (project, root string) {
	t.Helper()
	project = t.TempDir()
	root = filepath.Join(project, ".devrites")
	workspace := filepath.Join(root, "work", "feature")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(workspace, "touched-files.md"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, "state.md"), []byte("| schema | 3 |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return project, root
}

func writeCandidateTestFile(t *testing.T, project, name, content string) {
	t.Helper()
	path := filepath.Join(project, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateIdentityIsContentBoundAndMtimeIndependent(t *testing.T) {
	rows := []candidateTestRow{
		{state: "present", path: "a.txt", slice: "S-1", reason: "First file."},
		{state: "present", path: "b.txt", slice: "S-2", reason: "Second file."},
		{state: "deleted", path: "gone.txt", slice: "S-1", reason: "Removed file."},
	}
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(rows...))
	writeCandidateTestFile(t, project, "a.txt", "alpha\n")
	writeCandidateTestFile(t, project, "b.txt", "beta\n")

	digest, count, err := CandidateIdentity(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || len(digest) != 64 {
		t.Fatalf("digest=%q count=%d", digest, count)
	}

	for _, name := range []string{"a.txt", "b.txt"} {
		path := filepath.Join(project, name)
		if err := os.Chtimes(path, time.Unix(2_000_000_000, 0), time.Unix(2_000_000_000, 0)); err != nil {
			t.Fatal(err)
		}
	}
	writeCandidateTestFile(t, project, "unmanifested.txt", "ignored\n")
	got, _, err := CandidateIdentity(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if got != digest {
		t.Fatalf("mtime or unmanifested file changed digest: %s != %s", got, digest)
	}

	path := filepath.Join(project, "a.txt")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	writeCandidateTestFile(t, project, "a.txt", "ALPHA\n")
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	changed, _, err := CandidateIdentity(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if changed == digest {
		t.Fatal("content change with restored mtime did not change digest")
	}
}

func TestCandidateIdentityGoldenDigest(t *testing.T) {
	rows := []candidateTestRow{
		{state: "present", path: "alpha.sh", slice: "S-1", reason: "Present."},
		{state: "deleted", path: "gone.txt", slice: "S-1", reason: "Removed."},
	}
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(rows...))
	writeCandidateTestFile(t, project, "alpha.sh", "echo hi\n")

	digest, count, err := CandidateIdentity(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	const golden = "72b4d1643d24981baa9745e01776dc4819732f1b612165596e72c6629189b666"
	if digest != golden || count != 2 {
		t.Fatalf("digest=%q count=%d, want %q and 2", digest, count, golden)
	}

	mutations := []struct {
		name    string
		content string
		mode    os.FileMode
	}{
		{name: "content", content: "echo HO\n", mode: 0o644},
		{name: "length", content: "echo hello\n", mode: 0o644},
		{name: "mode", content: "echo hi\n", mode: 0o755},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			if runtime.GOOS == "windows" && mutation.name == "mode" {
				t.Skip("Windows does not expose a portable executable mode bit")
			}
			if err := os.WriteFile(filepath.Join(project, "alpha.sh"), []byte(mutation.content), mutation.mode); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Join(project, "alpha.sh"), mutation.mode); err != nil {
				t.Fatal(err)
			}
			got, _, err := CandidateIdentity(root, "feature")
			if err != nil {
				t.Fatal(err)
			}
			if got == golden {
				t.Fatalf("%s mutation did not change digest", mutation.name)
			}
		})
	}

	t.Run("path", func(t *testing.T) {
		project, root := writeCandidateTestWorkspace(t, candidateTestManifest(
			candidateTestRow{state: "present", path: "beta.sh", slice: "S-1", reason: "Present."},
			candidateTestRow{state: "deleted", path: "gone.txt", slice: "S-1", reason: "Removed."},
		))
		writeCandidateTestFile(t, project, "beta.sh", "echo hi\n")
		got, _, err := CandidateIdentity(root, "feature")
		if err != nil {
			t.Fatal(err)
		}
		if got == golden {
			t.Fatal("path mutation did not change digest")
		}
	})

	t.Run("state and type", func(t *testing.T) {
		project, root := writeCandidateTestWorkspace(t, candidateTestManifest(
			candidateTestRow{state: "present", path: "alpha.sh", slice: "S-1", reason: "Present."},
			candidateTestRow{state: "present", path: "gone.txt", slice: "S-1", reason: "Restored."},
		))
		writeCandidateTestFile(t, project, "alpha.sh", "echo hi\n")
		writeCandidateTestFile(t, project, "gone.txt", "")
		got, _, err := CandidateIdentity(root, "feature")
		if err != nil {
			t.Fatal(err)
		}
		if got == golden {
			t.Fatal("state/type mutation did not change digest")
		}
	})
}

func TestCandidateIdentityBindsExecutableBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose a portable executable mode bit")
	}
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(candidateTestRow{state: "present", path: "tool", slice: "S-1", reason: "Executable."}))
	writeCandidateTestFile(t, project, "tool", "#!/bin/sh\n")
	plain, _, err := CandidateIdentity(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(project, "tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	executable, _, err := CandidateIdentity(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if executable == plain {
		t.Fatal("executable-bit change did not change digest")
	}
}

func TestCandidateIdentityRejectsDistinctPathsToSameFile(t *testing.T) {
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(
		candidateTestRow{state: "present", path: "alias.txt", slice: "S-1", reason: "Alias."},
		candidateTestRow{state: "present", path: "original.txt", slice: "S-1", reason: "Original."},
	))
	writeCandidateTestFile(t, project, "original.txt", "shared\n")
	if err := os.Link(filepath.Join(project, "original.txt"), filepath.Join(project, "alias.txt")); err != nil {
		t.Fatal(err)
	}

	if _, _, err := CandidateIdentity(root, "feature"); err == nil || !strings.Contains(err.Error(), "same filesystem object") {
		t.Fatalf("error=%v, want same-filesystem-object rejection", err)
	}
}

func TestCandidateIdentityHandlesUnicodeNormalizationAliases(t *testing.T) {
	decomposed := "cafe\u0301.txt"
	composed := "caf\u00e9.txt"
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(
		candidateTestRow{state: "present", path: decomposed, slice: "S-1", reason: "Decomposed spelling."},
		candidateTestRow{state: "present", path: composed, slice: "S-1", reason: "Composed spelling."},
	))
	writeCandidateTestFile(t, project, composed, "composed\n")
	composedInfo, err := os.Lstat(filepath.Join(project, composed))
	if err != nil {
		t.Fatal(err)
	}
	decomposedInfo, err := os.Lstat(filepath.Join(project, decomposed))
	aliases := err == nil && os.SameFile(composedInfo, decomposedInfo)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if !aliases {
		writeCandidateTestFile(t, project, decomposed, "decomposed\n")
	}

	_, count, err := CandidateIdentity(root, "feature")
	if aliases {
		if err == nil || !strings.Contains(err.Error(), "same filesystem object") {
			t.Fatalf("error=%v, want normalization-alias rejection", err)
		}
		return
	}
	if err != nil || count != 2 {
		t.Fatalf("distinct Unicode paths: count=%d err=%v", count, err)
	}
}

func TestCandidateIdentityAcceptsEmptyAndDurableDevritesPaths(t *testing.T) {
	_, root := writeCandidateTestWorkspace(t, candidateTestManifest())
	if digest, count, err := CandidateIdentity(root, "feature"); err != nil || count != 0 || len(digest) != 64 {
		t.Fatalf("empty candidate: digest=%q count=%d err=%v", digest, count, err)
	}

	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(
		candidateTestRow{state: "present", path: ".devrites/principles.md", slice: "S-1", reason: "Project invariant."},
		candidateTestRow{state: "present", path: ".devrites/specs/api.md", slice: "S-1", reason: "Durable contract."},
	))
	writeCandidateTestFile(t, project, ".devrites/specs/api.md", "# API\n")
	writeCandidateTestFile(t, project, ".devrites/principles.md", "# Principles\n")
	if _, count, err := CandidateIdentity(root, "feature"); err != nil || count != 2 {
		t.Fatalf("durable .devrites paths: count=%d err=%v", count, err)
	}
}

func TestCandidateIdentityRejectsUnsafeWorkspaceSelection(t *testing.T) {
	t.Setenv("DEVRITES_WORKSPACE", "")
	manifest := []byte(candidateTestManifest())

	t.Run("invalid slug", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		escaped := filepath.Join(root, "escaped")
		if err := os.MkdirAll(escaped, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(escaped, "touched-files.md"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := CandidateIdentity(root, "../escaped"); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-INVALID") {
			t.Fatalf("error=%v, want invalid-slug rejection", err)
		}
	})

	t.Run("symlinked canonical workspace", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		escaped := filepath.Join(root, "escaped")
		if err := os.MkdirAll(filepath.Join(root, "work"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(escaped, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(escaped, "touched-files.md"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../escaped", filepath.Join(root, "work", "feature")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, _, err := CandidateIdentity(root, "feature"); err == nil || !strings.Contains(err.Error(), "DRV-WORKSPACE-SYMLINK") {
			t.Fatalf("error=%v, want symlinked-workspace rejection", err)
		}
	})
}

func TestCandidateIdentityRejectsMalformedManifestsAndPaths(t *testing.T) {
	validPrefix := "# Touched files\n\n## Touched files\nManifest below.\n\n## Candidate manifest\n"
	tests := []struct {
		name     string
		manifest string
		want     string
	}{
		{name: "missing artifact", want: "touched-files.md"},
		{name: "legacy list", manifest: "# Touched files\n\n## Touched files\n- `a.txt`\n", want: "Candidate manifest"},
		{name: "missing touched section", manifest: "## Candidate manifest\nNo project files.\n", want: "Touched files"},
		{name: "duplicate section", manifest: validPrefix + "No project files.\n\n## Candidate manifest\nNo project files.\n", want: "exactly one"},
		{name: "wrong marker", manifest: validPrefix + "No source files.\n", want: "strict table"},
		{name: "inexact marker", manifest: validPrefix + " No project files.\n", want: "strict table"},
		{name: "wrong header", manifest: validPrefix + "| File | Slice | Reason |\n| --- | --- | --- |\n| `a.txt` | S-1 | Added. |\n", want: "header"},
		{name: "bad state", manifest: candidateTestManifest(candidateTestRow{state: "modified", path: "a.txt", slice: "S-1", reason: "Changed."}), want: "state"},
		{name: "empty slice", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a.txt", reason: "Removed."}), want: "Slice"},
		{name: "empty reason", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a.txt", slice: "S-1"}), want: "Reason"},
		{name: "not backticks", manifest: validPrefix + "| State | File | Slice | Reason |\n| --- | --- | --- | --- |\n| deleted | a.txt | S-1 | Removed. |\n", want: "backticks"},
		{name: "duplicate", manifest: candidateTestManifest(
			candidateTestRow{state: "deleted", path: "a.txt", slice: "S-1", reason: "One."},
			candidateTestRow{state: "deleted", path: "a.txt", slice: "S-2", reason: "Two."},
		), want: "duplicate"},
		{name: "unsorted", manifest: candidateTestManifest(
			candidateTestRow{state: "deleted", path: "b.txt", slice: "S-1", reason: "Removed."},
			candidateTestRow{state: "deleted", path: "a.txt", slice: "S-1", reason: "Removed."},
		), want: "strictly sorted"},
		{name: "case folded collision", manifest: candidateTestManifest(
			candidateTestRow{state: "deleted", path: "A.txt", slice: "S-1", reason: "Removed."},
			candidateTestRow{state: "deleted", path: "a.txt", slice: "S-1", reason: "Removed."},
		), want: "case-fold collision"},
		{name: "unicode case folded collision", manifest: candidateTestManifest(
			candidateTestRow{state: "deleted", path: "Σ.txt", slice: "S-1", reason: "Removed."},
			candidateTestRow{state: "deleted", path: "ς.txt", slice: "S-1", reason: "Removed."},
		), want: "case-fold collision"},
		{name: "absolute", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "/tmp/a", slice: "S-1", reason: "Removed."}), want: "absolute"},
		{name: "drive", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "C:/tmp/a", slice: "S-1", reason: "Removed."}), want: "drive"},
		{name: "traversal", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a/../b", slice: "S-1", reason: "Removed."}), want: "traversal"},
		{name: "dot", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a/./b", slice: "S-1", reason: "Removed."}), want: "dot"},
		{name: "empty component", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a//b", slice: "S-1", reason: "Removed."}), want: "empty"},
		{name: "backslash", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: `a\b`, slice: "S-1", reason: "Removed."}), want: "backslash"},
		{name: "control", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a\tb", slice: "S-1", reason: "Removed."}), want: "control"},
		{name: "reserved character", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a:b", slice: "S-1", reason: "Removed."}), want: "reserved character"},
		{name: "reserved name", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "docs/AUX.txt", slice: "S-1", reason: "Removed."}), want: "reserved name"},
		{name: "reserved numbered name", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "COM9.go", slice: "S-1", reason: "Removed."}), want: "reserved name"},
		{name: "trailing dot", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a.", slice: "S-1", reason: "Removed."}), want: "trailing dot or space"},
		{name: "trailing space", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: "a ", slice: "S-1", reason: "Removed."}), want: "trailing dot or space"},
		{name: "git", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".git/config", slice: "S-1", reason: "Removed."}), want: "state path"},
		{name: "case folded git", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".GIT/config", slice: "S-1", reason: "Removed."}), want: "state path"},
		{name: "work state", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".devrites/work/x/state.md", slice: "S-1", reason: "Removed."}), want: "state path"},
		{name: "active state", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".devrites/ACTIVE", slice: "S-1", reason: "Removed."}), want: "state path"},
		{name: "checkpoint state", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".devrites/CHECKPOINT", slice: "S-1", reason: "Removed."}), want: "state path"},
		{name: "afk state", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".devrites/AFK", slice: "S-1", reason: "Removed."}), want: "state path"},
		{name: "archive state", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".devrites/archive/x", slice: "S-1", reason: "Removed."}), want: "state path"},
		{name: "unknown devrites sibling", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".devrites/notes.md", slice: "S-1", reason: "Removed."}), want: "state path"},
		{name: "case variant durable owner", manifest: candidateTestManifest(candidateTestRow{state: "deleted", path: ".DEVRITES/specs/a.md", slice: "S-1", reason: "Removed."}), want: "state path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, root := writeCandidateTestWorkspace(t, test.manifest)
			_, _, err := CandidateIdentity(root, "feature")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestCandidateIdentityRejectsStateAndFileTypeMismatches(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string)
		row   candidateTestRow
		want  string
	}{
		{name: "missing present", row: candidateTestRow{state: "present", path: "missing", slice: "S-1", reason: "Added."}, want: "missing"},
		{name: "existing deleted", setup: func(t *testing.T, project string) { writeCandidateTestFile(t, project, "still-here", "x") }, row: candidateTestRow{state: "deleted", path: "still-here", slice: "S-1", reason: "Removed."}, want: "still exists"},
		{name: "directory", setup: func(t *testing.T, project string) {
			if err := os.Mkdir(filepath.Join(project, "dir"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, row: candidateTestRow{state: "present", path: "dir", slice: "S-1", reason: "Added."}, want: "regular file"},
		{name: "final symlink", setup: func(t *testing.T, project string) {
			writeCandidateTestFile(t, project, "target", "x")
			if err := os.Symlink("target", filepath.Join(project, "link")); err != nil {
				t.Fatal(err)
			}
		}, row: candidateTestRow{state: "present", path: "link", slice: "S-1", reason: "Added."}, want: "symlink"},
		{name: "symlink parent", setup: func(t *testing.T, project string) {
			writeCandidateTestFile(t, project, "real/file", "x")
			if err := os.Symlink("real", filepath.Join(project, "alias")); err != nil {
				t.Fatal(err)
			}
		}, row: candidateTestRow{state: "present", path: "alias/file", slice: "S-1", reason: "Added."}, want: "symlink"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project, root := writeCandidateTestWorkspace(t, candidateTestManifest(test.row))
			if test.setup != nil {
				test.setup(t, project)
			}
			_, _, err := CandidateIdentity(root, "feature")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want substring %q", err, test.want)
			}
		})
	}
}

func TestCandidateIdentityRejectsSpecialFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-domain socket fixture is unavailable on Windows")
	}
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(candidateTestRow{state: "present", path: "socket", slice: "S-1", reason: "Added."}))
	listener, err := net.Listen("unix", filepath.Join(project, "socket"))
	if err != nil {
		t.Skipf("Unix-domain sockets unavailable: %v", err)
	}
	defer listener.Close()
	if _, _, err := CandidateIdentity(root, "feature"); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("error=%v, want special-file rejection", err)
	}
}

func TestCandidateIdentityEnforcesBounds(t *testing.T) {
	t.Run("manifest", func(t *testing.T) {
		_, root := writeCandidateTestWorkspace(t, candidateTestManifest()+strings.Repeat("x", (1<<20)+1))
		if _, _, err := CandidateIdentity(root, "feature"); err == nil || !strings.Contains(err.Error(), "1 MiB") {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("rows", func(t *testing.T) {
		rows := make([]candidateTestRow, 4097)
		for i := range rows {
			rows[i] = candidateTestRow{state: "deleted", path: fmt.Sprintf("gone/%04d", i), slice: "S", reason: "Removed."}
		}
		_, root := writeCandidateTestWorkspace(t, candidateTestManifest(rows...))
		if _, _, err := CandidateIdentity(root, "feature"); err == nil || !strings.Contains(err.Error(), "4096 rows") {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("path", func(t *testing.T) {
		_, root := writeCandidateTestWorkspace(t, candidateTestManifest(candidateTestRow{state: "deleted", path: strings.Repeat("a", 4097), slice: "S", reason: "Removed."}))
		if _, _, err := CandidateIdentity(root, "feature"); err == nil || !strings.Contains(err.Error(), "4096 bytes") {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("file", func(t *testing.T) {
		project, root := writeCandidateTestWorkspace(t, candidateTestManifest(candidateTestRow{state: "present", path: "large", slice: "S", reason: "Added."}))
		if err := os.Truncate(filepath.Join(project, "large"), (64<<20)+1); err != nil {
			if err := os.WriteFile(filepath.Join(project, "large"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Truncate(filepath.Join(project, "large"), (64<<20)+1); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := CandidateIdentity(root, "feature"); err == nil || !strings.Contains(err.Error(), "64 MiB") {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("aggregate", func(t *testing.T) {
		rows := make([]candidateTestRow, 5)
		project, root := writeCandidateTestWorkspace(t, "")
		for i := range rows {
			name := fmt.Sprintf("large-%d", i)
			rows[i] = candidateTestRow{state: "present", path: name, slice: "S", reason: "Added."}
			path := filepath.Join(project, name)
			if err := os.WriteFile(path, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			size := int64(64 << 20)
			if i == len(rows)-1 {
				size = 1
			}
			if err := os.Truncate(path, size); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(root, "work", "feature", "touched-files.md"), []byte(candidateTestManifest(rows...)), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := CandidateIdentity(root, "feature"); err == nil || !strings.Contains(err.Error(), "256 MiB") {
			t.Fatalf("error=%v", err)
		}
	})
}

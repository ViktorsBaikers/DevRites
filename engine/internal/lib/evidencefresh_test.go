package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeBoundCandidateWorkspace(t *testing.T, browser bool) (project, root, digest string) {
	t.Helper()
	project, root = writeCandidateTestWorkspace(t, candidateTestManifest(candidateTestRow{state: "present", path: "source.go", slice: "S-1", reason: "Implementation."}))
	writeCandidateTestFile(t, project, "source.go", "package source\n")
	digest, _, err := CandidateIdentity(root, "feature")
	if err != nil {
		t.Fatal(err)
	}
	line := "Candidate SHA-256: " + digest + "\n"
	workspace := filepath.Join(root, "work", "feature")
	for _, name := range []string{"evidence.md", "review.md", "seal.md"} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte("# Artifact\n\n"+line), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if browser {
		if err := os.WriteFile(filepath.Join(workspace, "browser-evidence.md"), []byte("# Browser evidence\n\n"+line), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return project, root, digest
}

func runEvidenceFresh(t *testing.T, root string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := EvidenceFresh(root, []string{"feature"}, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestEvidenceFreshUsesCandidateContentNotMtime(t *testing.T) {
	project, root, _ := writeBoundCandidateWorkspace(t, false)
	path := filepath.Join(project, "source.go")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, time.Unix(2_000_000_000, 0), time.Unix(2_000_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := runEvidenceFresh(t, root); code != 0 || !strings.Contains(stdout, "evidence-fresh: OK") {
		t.Fatalf("unchanged touch: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	writeCandidateTestFile(t, project, "source.go", "package changed\n")
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := runEvidenceFresh(t, root); code == 0 || !strings.Contains(stderr, "candidate digest") {
		t.Fatalf("restored-mtime change: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestEvidenceFreshRequiresOneExactMatchingDigestLine(t *testing.T) {
	tests := []struct {
		name string
		body func(string) string
		want string
	}{
		{name: "missing", body: func(string) string { return "# Evidence\n\nNo binding.\n" }, want: "missing"},
		{name: "duplicate", body: func(digest string) string {
			return "Candidate SHA-256: " + digest + "\nCandidate SHA-256: " + digest + "\n"
		}, want: "duplicate"},
		{name: "malformed", body: func(string) string { return "Candidate SHA-256: ABCD\n" }, want: "malformed"},
		{name: "mismatch", body: func(string) string { return "Candidate SHA-256: " + strings.Repeat("0", 64) + "\n" }, want: "does not match"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, root, digest := writeBoundCandidateWorkspace(t, false)
			path := filepath.Join(root, "work", "feature", "evidence.md")
			if err := os.WriteFile(path, []byte(test.body(digest)), 0o644); err != nil {
				t.Fatal(err)
			}
			code, stdout, stderr := runEvidenceFresh(t, root)
			if code == 0 || stdout != "" || !strings.Contains(stderr, test.want) {
				t.Fatalf("code=%d stdout=%q stderr=%q, want %q", code, stdout, stderr, test.want)
			}
		})
	}
}

func TestEvidenceFreshRequiresOptionalBrowserBinding(t *testing.T) {
	_, root, digest := writeBoundCandidateWorkspace(t, true)
	path := filepath.Join(root, "work", "feature", "browser-evidence.md")
	if err := os.WriteFile(path, []byte("# Browser evidence\n\nNo binding.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := runEvidenceFresh(t, root); code == 0 || !strings.Contains(stderr, "browser-evidence.md") {
		t.Fatalf("missing browser binding: code=%d stderr=%q", code, stderr)
	}
	if err := os.WriteFile(path, []byte("Candidate SHA-256: "+digest+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := runEvidenceFresh(t, root); code != 0 {
		t.Fatalf("browser binding: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestEvidenceFreshBoundsDigestArtifacts(t *testing.T) {
	_, root, digest := writeBoundCandidateWorkspace(t, false)
	path := filepath.Join(root, "work", "feature", "review.md")
	body := "Candidate SHA-256: " + digest + "\n" + strings.Repeat("x", 1<<20)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := runEvidenceFresh(t, root); code == 0 || stdout != "" || !strings.Contains(stderr, "1 MiB") {
		t.Fatalf("oversized proof: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

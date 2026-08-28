package lib

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/devrites/devrites/internal/testutil"
)

func runSecretScanGit(t *testing.T, project string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", project}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func runSecretScanGitInput(t *testing.T, project string, input io.Reader, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", project}, args...)...)
	cmd.Stdin = input
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func runSecretScanGitRedacted(t *testing.T, project string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", project}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal("redacted Git fixture command failed")
	}
	return string(out)
}

type secretScanZeroReader struct{}

func (secretScanZeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

type secretScanCountingReader struct {
	remaining int64
	consumed  int64
	maxRead   int
}

func (r *secretScanCountingReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if len(p) > r.maxRead {
		p = p[:r.maxRead]
	}
	if int64(len(p)) > r.remaining {
		p = p[:int(r.remaining)]
	}
	n := len(p)
	r.remaining -= int64(n)
	r.consumed += int64(n)
	return n, nil
}

type secretScanCappedWriter struct {
	remaining int
}

func (w *secretScanCappedWriter) Write(p []byte) (int, error) {
	if len(p) <= w.remaining {
		w.remaining -= len(p)
		return len(p), nil
	}
	n := w.remaining
	w.remaining = 0
	return n, nil
}

func TestBoundedSecretScanBufferStopsSourceConsumptionAtLimit(t *testing.T) {
	const (
		totalBytes = int64(1 << 20)
		limitBytes = int64(4500)
		readBytes  = 1024
	)
	source := &secretScanCountingReader{remaining: totalBytes, maxRead: readBytes}
	output := &boundedSecretScanBuffer{limit: limitBytes}
	copied, err := io.Copy(output, source)
	if !errors.Is(err, errSecretScanLimit) {
		t.Fatalf("copy error = %v, want scan limit; copied=%d consumed=%d", err, copied, source.consumed)
	}
	if copied != limitBytes {
		t.Fatalf("copied = %d, want %d", copied, limitBytes)
	}
	if source.consumed != 5*readBytes {
		t.Fatalf("source consumed = %d, want %d", source.consumed, 5*readBytes)
	}
	if output.Len() != int(limitBytes) {
		t.Fatalf("buffered = %d, want %d", output.Len(), limitBytes)
	}
	t.Logf("source consumed=%d copied=%d buffered=%d total=%d", source.consumed, copied, output.Len(), totalBytes)
}

func TestSecretScanScansExactStagedBlobWithoutDisclosure(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	path := "staged.txt"
	secret := "ghp_" + strings.Repeat("a", 32)
	testutil.WriteFile(t, filepath.Join(project, path), secret+"\n")
	runSecretScanGit(t, project, "add", "--", path)
	stagedOID := strings.TrimSpace(runSecretScanGit(t, project, "rev-parse", ":"+path))
	if stagedOID == "" {
		t.Fatal("staged blob ID is empty")
	}
	testutil.WriteFile(t, filepath.Join(project, path), "safe worktree bytes\n")

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 3 {
		t.Fatalf("want staged blob block rc=3 got %d", code)
	}
	combined := out.String() + err.String()
	if strings.Contains(combined, secret) {
		t.Fatal("secret material disclosed")
	}
}

func TestSecretScanSkipsDeletedWorktreePaths(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	runSecretScanGit(t, project, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, project, "config", "user.name", "DevRites Tests")
	path := "deleted.txt"
	testutil.WriteFile(t, filepath.Join(project, path), "safe baseline\n")
	runSecretScanGit(t, project, "add", "--", path)
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	if err := os.Remove(filepath.Join(project, path)); err != nil {
		t.Fatal(err)
	}

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), nil, strings.NewReader(""), &out, &err); code != 0 {
		t.Fatalf("deleted worktree path returned %d, stdout=%q stderr=%q", code, out.String(), err.String())
	}
}

func TestSecretScanIgnoresReplacementOfHEADWhenEnumeratingStagedBlobs(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	runSecretScanGit(t, project, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, project, "config", "user.name", "DevRites Tests")
	path := "payload.txt"
	testutil.WriteFile(t, filepath.Join(project, path), "safe baseline\n")
	runSecretScanGit(t, project, "add", "--", path)
	runSecretScanGit(t, project, "commit", "-qm", "baseline")

	secret := "ghp_" + strings.Repeat("h", 32)
	testutil.WriteFile(t, filepath.Join(project, path), secret+"\n")
	runSecretScanGit(t, project, "add", "--", path)
	stagedTree := strings.TrimSpace(runSecretScanGit(t, project, "write-tree"))
	replacement := strings.TrimSpace(runSecretScanGitRedacted(t, project, "commit-tree", stagedTree, "-m", "replacement"))
	head := strings.TrimSpace(runSecretScanGit(t, project, "rev-parse", "HEAD"))
	runSecretScanGitRedacted(t, project, "replace", head, replacement)
	if control := runSecretScanGit(t, project, "diff", "--cached", "--raw", "--"); control != "" {
		t.Fatal("replacement ref did not suppress the unprotected control diff")
	}

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 3 {
		t.Fatalf("want replacement-resistant staged block rc=3 got %d", code)
	}
	if strings.Contains(out.String()+err.String(), secret) {
		t.Fatal("secret material disclosed")
	}
}

func TestSecretScanIgnoresReplacementOfStagedBlob(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	runSecretScanGit(t, project, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, project, "config", "user.name", "DevRites Tests")
	testutil.WriteFile(t, filepath.Join(project, "baseline.txt"), "baseline\n")
	runSecretScanGit(t, project, "add", "--", "baseline.txt")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")

	path := "payload.txt"
	secret := "ghp_" + strings.Repeat("i", 32)
	testutil.WriteFile(t, filepath.Join(project, path), secret+"\n")
	runSecretScanGit(t, project, "add", "--", path)
	stagedOID := strings.TrimSpace(runSecretScanGit(t, project, "rev-parse", ":"+path))
	safePath := filepath.Join(project, "safe-object.txt")
	testutil.WriteFile(t, safePath, "safe replacement bytes\n")
	safeOID := strings.TrimSpace(runSecretScanGit(t, project, "hash-object", "-w", "--no-filters", "--", safePath))
	if err := os.Remove(safePath); err != nil {
		t.Fatal(err)
	}
	runSecretScanGitRedacted(t, project, "replace", stagedOID, safeOID)
	if control := runSecretScanGitRedacted(t, project, "cat-file", "-p", stagedOID); control != "safe replacement bytes\n" {
		t.Fatal("replacement ref did not substitute the unprotected control blob")
	}

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 3 {
		t.Fatalf("want replacement-resistant staged block rc=3 got %d", code)
	}
	if strings.Contains(out.String()+err.String(), secret) {
		t.Fatal("secret material disclosed")
	}
}

func TestSecretScanRedactsSecretShapedStagedPath(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	pathSecret := "ghp_" + strings.Repeat("j", 32)
	contentSecret := "ghp_" + strings.Repeat("k", 32)
	path := "report-" + pathSecret + ".txt"
	if err := os.WriteFile(filepath.Join(project, path), []byte(contentSecret+"\n"), 0o644); err != nil {
		t.Fatal("cannot create secret-shaped path fixture")
	}
	runSecretScanGitRedacted(t, project, "add", "--", path)

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 3 {
		t.Fatalf("want secret-shaped path block rc=3 got %d", code)
	}
	combined := out.String() + err.String()
	if !strings.Contains(out.String(), `source="<redacted-path>"`) {
		t.Fatal("stdout lacks redacted source metadata")
	}
	if strings.Contains(combined, pathSecret) || strings.Contains(combined, contentSecret) {
		t.Fatal("secret-shaped path or content material disclosed")
	}
}

func TestSecretScanRejectsOversizedStdin(t *testing.T) {
	stdin := io.LimitReader(secretScanZeroReader{}, (64<<20)+1)
	var out, err bytes.Buffer
	if code := SecretScan(t.TempDir(), []string{"--stdin"}, stdin, &out, &err); code != 2 {
		t.Fatalf("want oversized stdin refusal rc=2 got %d", code)
	}
	if out.Len() != 0 {
		t.Fatal("oversized stdin produced stdout")
	}
	if got, want := err.String(), "secret-scan: cannot inspect stdin\n"; got != want {
		t.Fatal("stderr lacks the redacted fail-closed stdin diagnostic")
	}
}

func TestSecretScanRejectsOversizedStagedBlob(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	runSecretScanGit(t, project, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, project, "config", "user.name", "DevRites Tests")
	runSecretScanGit(t, project, "commit", "--allow-empty", "-qm", "baseline")
	oid := strings.TrimSpace(runSecretScanGitInput(t, project, io.LimitReader(secretScanZeroReader{}, (64<<20)+1), "hash-object", "-w", "--stdin"))
	runSecretScanGit(t, project, "update-index", "--add", "--cacheinfo", "100644,"+oid+",large.bin")

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 2 {
		t.Fatalf("want oversized staged blob refusal rc=2 got %d", code)
	}
	if out.Len() != 0 {
		t.Fatal("oversized staged blob produced stdout")
	}
	if got, want := err.String(), "secret-scan: cannot inspect staged blobs\n"; got != want {
		t.Fatal("stderr lacks the redacted fail-closed staged blob diagnostic")
	}
}

func TestSecretScanRejectsTooManyStagedEntries(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	runSecretScanGit(t, project, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, project, "config", "user.name", "DevRites Tests")
	runSecretScanGit(t, project, "commit", "--allow-empty", "-qm", "baseline")
	oid := strings.TrimSpace(runSecretScanGitInput(t, project, strings.NewReader("safe\n"), "hash-object", "-w", "--stdin"))
	var entries strings.Builder
	for i := 0; i < 4097; i++ {
		entries.WriteString("100644 blob ")
		entries.WriteString(oid)
		entries.WriteByte('\t')
		entries.WriteString("entry-")
		entries.WriteString(strconv.Itoa(i))
		entries.WriteByte('\n')
	}
	runSecretScanGitInput(t, project, strings.NewReader(entries.String()), "update-index", "--index-info")

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 2 {
		t.Fatalf("want staged entry limit refusal rc=2 got %d", code)
	}
	if out.Len() != 0 {
		t.Fatal("excessive staged entries produced stdout")
	}
	if got, want := err.String(), "secret-scan: cannot inspect staged index\n"; got != want {
		t.Fatal("stderr lacks the redacted fail-closed staged index diagnostic")
	}
}

func TestSecretScanRejectsExcessiveFindings(t *testing.T) {
	secret := "ghp_" + strings.Repeat("l", 32)
	var body strings.Builder
	for i := 0; i < 4097; i++ {
		body.WriteString(secret)
		body.WriteByte('\n')
	}
	var out, err bytes.Buffer
	if code := SecretScan(t.TempDir(), []string{"--stdin"}, strings.NewReader(body.String()), &out, &err); code != 2 {
		t.Fatalf("want finding limit refusal rc=2 got %d", code)
	}
	if out.Len() != 0 {
		t.Fatal("excessive findings produced stdout")
	}
	if got, want := err.String(), "secret-scan: cannot inspect stdin\n"; got != want {
		t.Fatal("stderr lacks the redacted fail-closed stdin diagnostic")
	}
	if strings.Contains(err.String(), secret) {
		t.Fatal("finding limit diagnostic disclosed secret material")
	}
}

func TestSecretScanFailsClosedOnOutputWriterOverflow(t *testing.T) {
	stdout := &secretScanCappedWriter{remaining: 1}
	var err bytes.Buffer
	if code := SecretScan(t.TempDir(), []string{"--stdin"}, strings.NewReader("safe"), stdout, &err); code != 2 {
		t.Fatalf("want output writer overflow rc=2 got %d", code)
	}
	if got, want := err.String(), "secret-scan: cannot report result\n"; got != want {
		t.Fatal("stderr lacks the redacted fail-closed output diagnostic")
	}
}

func TestSecretScanBlocksHighSeverityTouchedFile(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "leak\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "leak", "touched-files.md"), candidateTestManifest(candidateTestRow{state: "present", path: "secrets.txt", slice: "S-1", reason: "Credential check."}))
	testutil.WriteFile(t, filepath.Join(root, "work", "leak", "state.md"), "| schema | 3 |\n")
	secret := "ghp_" + strings.Repeat("f", 32)
	testutil.WriteFile(t, filepath.Join(project, "secrets.txt"), secret+"\n")
	var out, err bytes.Buffer
	if code := SecretScan(root, nil, strings.NewReader(""), &out, &err); code != 3 {
		t.Fatalf("want block rc=3 got %d", code)
	}
	if strings.Contains(out.String()+err.String(), secret) {
		t.Fatal("secret material disclosed")
	}
}

func TestSecretScanSlugUsesCandidateWithoutGitFallback(t *testing.T) {
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest())
	secret := "ghp_" + strings.Repeat("m", 32)
	writeCandidateTestFile(t, project, "unmanifested.txt", secret+"\n")

	var out, err bytes.Buffer
	if code := SecretScan(root, []string{"feature"}, strings.NewReader(""), &out, &err); code != 0 {
		t.Fatalf("empty candidate returned %d, stdout=%q stderr=%q", code, out.String(), err.String())
	}
	if got := out.String(); got != "secret-scan: PASS\n" {
		t.Fatalf("stdout=%q", got)
	}
}

func TestSecretScanSlugSkipsDeletedCandidateRows(t *testing.T) {
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(candidateTestRow{state: "deleted", path: "removed.txt", slice: "S-1", reason: "Removed."}))
	secret := "ghp_" + strings.Repeat("n", 32)
	writeCandidateTestFile(t, project, "unmanifested.txt", secret+"\n")

	var out, err bytes.Buffer
	if code := SecretScan(root, []string{"feature"}, strings.NewReader(""), &out, &err); code != 0 {
		t.Fatalf("deleted candidate returned %d, stdout=%q stderr=%q", code, out.String(), err.String())
	}
}

func TestSecretScanSlugFailsClosedOnInvalidCandidate(t *testing.T) {
	for _, test := range []struct {
		name     string
		manifest string
	}{
		{name: "missing"},
		{name: "legacy", manifest: "# Touched files\n\n## Touched files\n- `source.go`\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, root := writeCandidateTestWorkspace(t, test.manifest)
			var out, err bytes.Buffer
			if code := SecretScan(root, []string{"feature"}, strings.NewReader(""), &out, &err); code != 2 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), err.String())
			}
			if out.Len() != 0 || !strings.Contains(err.String(), "candidate manifest") {
				t.Fatalf("stdout=%q stderr=%q", out.String(), err.String())
			}
		})
	}
}

func TestSecretScanFailsClosedWhenGitIsUnavailable(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	t.Chdir(project)
	t.Setenv("PATH", t.TempDir())

	var out, err bytes.Buffer
	if code := SecretScan(root, nil, strings.NewReader(""), &out, &err); code != 2 {
		t.Fatalf("want rc=2 got %d stdout=%s stderr=%s", code, out.String(), err.String())
	}
	if got, want := err.String(), "secret-scan: cannot inspect changed paths\n"; got != want {
		t.Fatalf("stderr = %q, want redacted fail-closed diagnostic %q", got, want)
	}
}

func TestSecretScanScansPRBodyFromStdinWithoutDisclosure(t *testing.T) {
	secret := "ghp_" + strings.Repeat("b", 32)
	var out, err bytes.Buffer
	if code := SecretScan(t.TempDir(), []string{"--stdin"}, strings.NewReader("review "+secret), &out, &err); code != 3 {
		t.Fatalf("want stdin block rc=3 got %d", code)
	}
	if got := out.String(); !strings.Contains(got, `source="<stdin>" kind=github-token offset=7`) {
		t.Fatal("stdout lacks safe stdin metadata")
	}
	if strings.Contains(out.String()+err.String(), secret) {
		t.Fatal("secret material disclosed")
	}
}

func TestSecretScanFailsClosedOnStdinReadErrorWithoutDisclosure(t *testing.T) {
	secret := "ghp_" + strings.Repeat("c", 32)
	var out, err bytes.Buffer
	if code := SecretScan(t.TempDir(), []string{"--stdin"}, iotest.ErrReader(errors.New(secret)), &out, &err); code != 2 {
		t.Fatalf("want stdin error rc=2 got %d", code)
	}
	if got, want := err.String(), "secret-scan: cannot inspect stdin\n"; got != want {
		t.Fatal("stderr lacks the redacted fail-closed stdin diagnostic")
	}
	if strings.Contains(out.String()+err.String(), secret) {
		t.Fatal("stdin error disclosed secret material")
	}
}

func TestSecretScanRejectsReviewTextInArgvWithoutDisclosure(t *testing.T) {
	secret := "ghp_" + strings.Repeat("g", 32)
	var out, err bytes.Buffer
	if code := SecretScan(t.TempDir(), []string{"--text", secret}, strings.NewReader(""), &out, &err); code != 2 {
		t.Fatalf("want argv refusal rc=2 got %d", code)
	}
	if got, want := err.String(), "secret-scan: review text must be supplied with --stdin\n"; got != want {
		t.Fatal("stderr lacks the safe stdin remediation")
	}
	if strings.Contains(out.String()+err.String(), secret) {
		t.Fatal("argv refusal disclosed secret material")
	}
}

func TestSecretScanScansUnusualStagedEntriesWithoutReadingWorktree(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	runSecretScanGit(t, project, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, project, "config", "user.name", "DevRites Tests")
	for _, path := range []string{"deleted.txt", "old-name.txt"} {
		testutil.WriteFile(t, filepath.Join(project, path), "baseline\n")
	}
	runSecretScanGit(t, project, "add", "--", "deleted.txt", "old-name.txt")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")

	secret := "ghp_" + strings.Repeat("d", 32)
	if err := os.Remove(filepath.Join(project, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(project, "old-name.txt"), filepath.Join(project, "new-name.txt")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"new-name.txt", "space name.txt"} {
		testutil.WriteFile(t, filepath.Join(project, path), secret+"\n")
	}
	if err := os.WriteFile(filepath.Join(project, "binary.dat"), append([]byte{0, 1, 2}, []byte(secret)...), 0o644); err != nil {
		t.Fatal(err)
	}
	runSecretScanGit(t, project, "add", "-A")
	// Git for Windows otherwise rejects the fixture's index-only LF path.
	runSecretScanGit(t, project, "config", "core.protectNTFS", "false")
	secretOID := strings.TrimSpace(runSecretScanGitInput(t, project, strings.NewReader(secret), "hash-object", "-w", "--stdin"))
	indexEntries := "100644 blob " + secretOID + "\tline\nbreak.txt\x00" +
		"120000 blob " + secretOID + "\tlinked-secret\x00"
	runSecretScanGitInput(t, project, strings.NewReader(indexEntries), "update-index", "-z", "--index-info")

	for _, path := range []string{"new-name.txt", "space name.txt", "binary.dat"} {
		testutil.WriteFile(t, filepath.Join(project, path), "safe worktree bytes\n")
	}

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 3 {
		t.Fatalf("want unusual staged entries to block rc=3 got %d", code)
	}
	for _, path := range []string{"new-name.txt", "space name.txt", "line\nbreak.txt", "binary.dat", "linked-secret"} {
		if !strings.Contains(out.String(), "source="+strconv.Quote(path)) {
			t.Errorf("stdout lacks finding for %q", path)
		}
	}
	if strings.Contains(out.String(), "deleted.txt") {
		t.Fatal("staged deletion was treated as readable content")
	}
	if strings.Contains(out.String()+err.String(), secret) {
		t.Fatal("secret material disclosed")
	}
}

func TestSecretScanFailsClosedOnNonBlobIndexEntry(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	runSecretScanGit(t, project, "config", "user.email", "tests@example.invalid")
	runSecretScanGit(t, project, "config", "user.name", "DevRites Tests")
	testutil.WriteFile(t, filepath.Join(project, "baseline.txt"), "baseline\n")
	runSecretScanGit(t, project, "add", "--", "baseline.txt")
	runSecretScanGit(t, project, "commit", "-qm", "baseline")
	commitOID := strings.TrimSpace(runSecretScanGit(t, project, "rev-parse", "HEAD"))
	runSecretScanGit(t, project, "update-index", "--add", "--cacheinfo", "160000,"+commitOID+",nested-repository")

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 2 {
		t.Fatalf("want non-blob source rc=2 got %d stdout=%q stderr=%q", code, out.String(), err.String())
	}
	if got, want := err.String(), "secret-scan: cannot inspect staged index\n"; got != want {
		t.Fatalf("stderr = %q, want redacted fail-closed diagnostic %q", got, want)
	}
}

func TestSecretScanFailsClosedOnUnreadableStagedBlob(t *testing.T) {
	project := t.TempDir()
	runSecretScanGit(t, project, "init", "-q")
	testutil.WriteFile(t, filepath.Join(project, "missing-object.txt"), "staged bytes\n")
	runSecretScanGit(t, project, "add", "--", "missing-object.txt")
	oid := strings.TrimSpace(runSecretScanGit(t, project, "rev-parse", ":missing-object.txt"))
	if err := os.Remove(filepath.Join(project, ".git", "objects", oid[:2], oid[2:])); err != nil {
		t.Fatal(err)
	}

	var out, err bytes.Buffer
	if code := SecretScan(filepath.Join(project, ".devrites"), []string{"--staged"}, strings.NewReader(""), &out, &err); code != 2 {
		t.Fatalf("want unreadable blob rc=2 got %d stdout=%q stderr=%q", code, out.String(), err.String())
	}
	if got, want := err.String(), "secret-scan: cannot inspect staged blobs\n"; got != want {
		t.Fatalf("stderr = %q, want redacted fail-closed diagnostic %q", got, want)
	}
}

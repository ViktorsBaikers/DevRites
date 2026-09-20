package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func claimTestRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".devrites")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func runClaim(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := RunClaim(root, args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestClaimAddAndList(t *testing.T) {
	root := claimTestRoot(t)
	code, out, errOut := runClaim(t, root, "add", "--session", "s1", "--reason", "slice work", "pkg/a.go", "pkg/b.go")
	if code != 0 {
		t.Fatalf("add: code=%d stderr=%s", code, errOut)
	}
	if !strings.Contains(out, "cl-") || !strings.Contains(out, "2 path(s)") {
		t.Fatalf("add output missing claim id/paths: %q", out)
	}
	code, out, _ = runClaim(t, root, "list")
	if code != 0 || !strings.Contains(out, "session=s1") || !strings.Contains(out, "pkg/a.go") {
		t.Fatalf("list: code=%d out=%q", code, out)
	}
}

func TestClaimConflictBlocks(t *testing.T) {
	root := claimTestRoot(t)
	if code, _, e := runClaim(t, root, "add", "--session", "s1", "pkg/a.go"); code != 0 {
		t.Fatalf("first add: %d %s", code, e)
	}
	code, _, errOut := runClaim(t, root, "add", "--session", "s2", "pkg/a.go")
	if code != 3 {
		t.Fatalf("conflicting add should exit 3, got %d", code)
	}
	if !strings.Contains(errOut, "BLOCKED") || !strings.Contains(errOut, "s1") {
		t.Fatalf("conflict output should name holder: %q", errOut)
	}
	// Same session re-claim is allowed.
	if code, _, e := runClaim(t, root, "add", "--session", "s1", "pkg/a.go"); code != 0 {
		t.Fatalf("same-session re-claim: %d %s", code, e)
	}
	// Disjoint path is fine for another session.
	if code, _, e := runClaim(t, root, "add", "--session", "s2", "pkg/c.go"); code != 0 {
		t.Fatalf("disjoint add: %d %s", code, e)
	}
}

func TestClaimCheck(t *testing.T) {
	root := claimTestRoot(t)
	code, out, _ := runClaim(t, root, "check", "pkg/a.go")
	if code != 0 || !strings.Contains(out, "free") {
		t.Fatalf("check free: %d %q", code, out)
	}
	if code, _, e := runClaim(t, root, "add", "--session", "s1", "pkg/a.go"); code != 0 {
		t.Fatalf("add: %d %s", code, e)
	}
	code, out, _ = runClaim(t, root, "check", "--session", "s2", "pkg/a.go", "pkg/z.go")
	if code != 3 || !strings.Contains(out, "s1") {
		t.Fatalf("check conflict: %d %q", code, out)
	}
	// Holder's own check is free.
	code, out, _ = runClaim(t, root, "check", "--session", "s1", "pkg/a.go")
	if code != 0 {
		t.Fatalf("check own: %d %q", code, out)
	}
}

func TestClaimRelease(t *testing.T) {
	root := claimTestRoot(t)
	code, out, _ := runClaim(t, root, "add", "--session", "s1", "pkg/a.go")
	if code != 0 {
		t.Fatalf("add: %d", code)
	}
	id := strings.Fields(out)[1]
	// Wrong session cannot release.
	if code, _, e := runClaim(t, root, "release", "--session", "s2", "--id", id); code != 2 {
		t.Fatalf("release by wrong session: %d %s", code, e)
	}
	if code, _, e := runClaim(t, root, "release", "--session", "s1", "--id", id); code != 0 {
		t.Fatalf("release: %d %s", code, e)
	}
	// Released claim no longer conflicts.
	code, _, _ = runClaim(t, root, "check", "--session", "s2", "pkg/a.go")
	if code != 0 {
		t.Fatalf("check after release: %d", code)
	}
	// Double release is a usage error.
	if code, _, _ := runClaim(t, root, "release", "--session", "s1", "--id", id); code != 2 {
		t.Fatalf("double release: %d", code)
	}
	// Unknown id.
	if code, _, _ := runClaim(t, root, "release", "--session", "s1", "--id", "cl-nope"); code != 2 {
		t.Fatalf("unknown release: %d", code)
	}
}

func TestClaimExpiry(t *testing.T) {
	root := claimTestRoot(t)
	path := filepath.Join(root, ClaimsFile)
	old := claimRecord{
		ID:        "cl-old",
		Session:   "s1",
		Paths:     []string{"pkg/a.go"},
		ClaimedAt: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
		ExpiresAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}
	line, _ := json.Marshal(old)
	if err := os.WriteFile(path, append(line, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	// Expired claims do not conflict and do not list by default.
	code, _, errOut := runClaim(t, root, "add", "--session", "s2", "pkg/a.go")
	if code != 0 {
		t.Fatalf("add over expired: %d %s", code, errOut)
	}
	code, out, _ := runClaim(t, root, "list")
	if code != 0 || strings.Contains(out, "cl-old") {
		t.Fatalf("list should hide expired: %d %q", code, out)
	}
	code, out, _ = runClaim(t, root, "list", "--all")
	if code != 0 || !strings.Contains(out, "cl-old") || !strings.Contains(out, "expired") {
		t.Fatalf("list --all should show expired: %d %q", code, out)
	}
}

func TestClaimRejectsBadInput(t *testing.T) {
	root := claimTestRoot(t)
	cases := [][]string{
		{"add", "pkg/a.go"},                                 // no session
		{"add", "--session", "s1"},                          // no paths
		{"add", "--session", "s1", "/abs/path.go"},          // absolute
		{"add", "--session", "s1", "../escape.go"},          // traversal
		{"add", "--session", "s1", ".devrites/work/x/f.md"}, // control tree
		{"add", "--session", "s1", "a.go", "a.go"},          // duplicate
		{"add", "--session", "bad session!", "a.go"},        // bad session
		{"add", "--session", "s1", "--ttl", "x", "a.go"},    // bad ttl
	}
	for _, c := range cases {
		if code, _, _ := runClaim(t, root, c...); code != 2 {
			t.Fatalf("args %v: expected exit 2, got %d", c, code)
		}
	}
	// TTL clamps rather than fails.
	code, _, _ := runClaim(t, root, "add", "--session", "s1", "--ttl", "0", "pkg/a.go")
	if code != 0 {
		t.Fatalf("ttl clamp low: %d", code)
	}
	code, _, _ = runClaim(t, root, "add", "--session", "s1", "--ttl", "9999", "pkg/b.go")
	if code != 0 {
		t.Fatalf("ttl clamp high: %d", code)
	}
}

func TestClaimLedgerToleratesMalformedLines(t *testing.T) {
	root := claimTestRoot(t)
	path := filepath.Join(root, ClaimsFile)
	if err := os.WriteFile(path, []byte("not json\n{\"id\":\"x\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runClaim(t, root, "list")
	if code != 0 || !strings.Contains(out, "unparsed") {
		t.Fatalf("malformed ledger: %d %q", code, out)
	}
}

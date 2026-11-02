package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	// Expired claims do not conflict but still list by default: the hold
	// protocol reads them.
	code, _, errOut := runClaim(t, root, "add", "--session", "s2", "pkg/a.go")
	if code != 0 {
		t.Fatalf("add over expired: %d %s", code, errOut)
	}
	code, out, _ := runClaim(t, root, "list")
	if code != 0 || !strings.Contains(out, "cl-old") || !strings.Contains(out, "expired") {
		t.Fatalf("list should show expired: %d %q", code, out)
	}
	code, out, _ = runClaim(t, root, "list", "--all")
	if code != 0 || !strings.Contains(out, "cl-old") || !strings.Contains(out, "expired") {
		t.Fatalf("list --all should show expired: %d %q", code, out)
	}
}

// Bare list hides released history behind one count line; --all and a
// --session filter still reach the released rows.
func TestClaimListFoldsReleased(t *testing.T) {
	root := claimTestRoot(t)
	ids := map[string]string{}
	for _, s := range []string{"s1", "s2", "s3"} {
		code, out, errOut := runClaim(t, root, "add", "--session", s, "pkg/"+s+".go")
		if code != 0 {
			t.Fatalf("add %s: %d %s", s, code, errOut)
		}
		ids[s] = strings.Fields(strings.TrimPrefix(out, "claim: "))[0]
	}
	for _, s := range []string{"s1", "s2"} {
		if code, _, errOut := runClaim(t, root, "release", "--session", s, "--id", ids[s]); code != 0 {
			t.Fatalf("release %s: %d %s", s, code, errOut)
		}
	}
	code, out, _ := runClaim(t, root, "list")
	if code != 0 || strings.Contains(out, "released session=") || !strings.Contains(out, ids["s3"]) || !strings.Contains(out, "claim: 2 released (use --all)") {
		t.Fatalf("bare list: %d %q", code, out)
	}
	code, out, _ = runClaim(t, root, "list", "--all")
	if code != 0 || strings.Count(out, "released session=") != 2 || strings.Contains(out, "use --all") {
		t.Fatalf("list --all: %d %q", code, out)
	}
	code, out, _ = runClaim(t, root, "list", "--all", "--session", "s1")
	if code != 0 || !strings.Contains(out, ids["s1"]+" released") || strings.Contains(out, ids["s2"]) {
		t.Fatalf("list --all --session: %d %q", code, out)
	}
	code, out, _ = runClaim(t, root, "list", "--session", "s1")
	if code != 0 || !strings.Contains(out, "claim: no claims") || !strings.Contains(out, "claim: 1 released (use --all)") {
		t.Fatalf("list --session of a released holder: %d %q", code, out)
	}
}

func TestClaimWorkspaceLeaseBlocksSecondSession(t *testing.T) {
	root := claimTestRoot(t)
	if code, _, e := runClaim(t, root, "add", "--session", "lookahead", "--workspace", "feat-b"); code != 0 {
		t.Fatalf("workspace lease: %d %s", code, e)
	}
	code, out, _ := runClaim(t, root, "check", "--session", "main", "--workspace", "feat-b")
	if code != 3 || !strings.Contains(out, ".devrites/work/feat-b") || !strings.Contains(out, "lookahead") {
		t.Fatalf("main check on leased workspace: %d %q", code, out)
	}
	if code, _, _ := runClaim(t, root, "check", "--session", "main", "--workspace", "feat-c"); code != 0 {
		t.Fatalf("another workspace must stay free, got %d", code)
	}
	if code, _, _ := runClaim(t, root, "check", "--session", "lookahead", "--workspace", "feat-b"); code != 0 {
		t.Fatalf("the holder's own check must be free, got %d", code)
	}
	// A positional control-tree path stays refused; only --workspace reaches it.
	if code, _, _ := runClaim(t, root, "check", ".devrites/work/feat-b"); code != 2 {
		t.Fatalf("positional .devrites path: want exit 2, got %d", code)
	}
	for _, bad := range []string{"../x", "a/b", ".."} {
		if code, _, _ := runClaim(t, root, "add", "--session", "s", "--workspace", bad); code != 2 {
			t.Fatalf("--workspace %q: want exit 2, got %d", bad, code)
		}
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

func TestClaimCheckFailsClosedOnUnparsedLine(t *testing.T) {
	root := claimTestRoot(t)
	path := filepath.Join(root, ClaimsFile)
	full, err := json.Marshal(&claimRecord{
		ID: "cl-1", Session: "s-1", Paths: []string{"b.txt"},
		ClaimedAt: "2026-01-01T00:00:00Z",
		ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	truncated := string(full[:len(full)-12]) + "\n"
	if err := os.WriteFile(path, []byte(truncated), 0o600); err != nil {
		t.Fatal(err)
	}
	latest, order, bad, err := readClaims(path)
	if err != nil || bad != 1 || len(liveClaims(latest, order, time.Now())) != 0 {
		t.Fatalf("premise: want 1 unparsed line and no live claims, got bad=%d live=%d err=%v", bad, len(liveClaims(latest, order, time.Now())), err)
	}
	code, out, errOut := runClaim(t, root, "check", "--session", "s-2", "b.txt")
	if code == 0 || !strings.Contains(out+errOut, "unparsed") {
		t.Fatalf("check over unreadable ledger must fail closed: code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, errOut = runClaim(t, root, "add", "--session", "s-2", "b.txt")
	if code == 0 || !strings.Contains(out+errOut, "unparsed") {
		t.Fatalf("add over unreadable ledger must fail closed: code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestClaimAddConcurrentOverlapOneWins(t *testing.T) {
	root := claimTestRoot(t)
	const n = 8
	codes := make(chan int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out, errOut bytes.Buffer
			codes <- RunClaim(root, []string{"add", "--session", fmt.Sprintf("s%d", i), "pkg/a.go"}, &out, &errOut)
		}()
	}
	wg.Wait()
	close(codes)
	wins := 0
	for c := range codes {
		if c == 0 {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("overlapping concurrent claims: %d succeeded, want exactly 1", wins)
	}
}

func TestClaimAddRejectsOversizedRecord(t *testing.T) {
	root := claimTestRoot(t)
	ledger := filepath.Join(root, ClaimsFile)
	reason := strings.Repeat("<", 190000)
	code, _, errOut := runClaim(t, root, "add", "--session", "s1", "--reason", reason, "a.txt")
	if code != 2 {
		t.Fatalf("oversized add should exit 2, got %d stderr=%.200s", code, errOut)
	}
	if info, err := os.Stat(ledger); err == nil && info.Size() != 0 {
		t.Fatalf("oversized add wrote %d bytes to the ledger", info.Size())
	}
	if code, _, e := runClaim(t, root, "list"); code != 0 {
		t.Fatalf("list after refused add: %d %s", code, e)
	}
}

func TestClaimReadNamesLedgerOnOversizedLine(t *testing.T) {
	root := claimTestRoot(t)
	ledger := filepath.Join(root, ClaimsFile)
	if err := os.WriteFile(ledger, []byte(strings.Repeat("x", (1<<20)+10)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := runClaim(t, root, "list")
	if code != 2 || !strings.Contains(errOut, ledger) || !strings.Contains(errOut, "repair") {
		t.Fatalf("list: code=%d stderr=%q", code, errOut)
	}
}

func TestClaimConflictCoversContainedPaths(t *testing.T) {
	root := claimTestRoot(t)
	if code, _, e := runClaim(t, root, "add", "--session", "A", "src/auth"); code != 0 {
		t.Fatalf("first add: %d %s", code, e)
	}
	for _, path := range []string{"src/auth/login.go", "src"} {
		if code, out, _ := runClaim(t, root, "check", "--session", "B", path); code != 3 || !strings.Contains(out, "BLOCKED") {
			t.Fatalf("check %s: code=%d out=%q, want overlap blocked", path, code, out)
		}
		if code, _, _ := runClaim(t, root, "add", "--session", "B", path); code != 3 {
			t.Fatalf("add %s: code=%d, want overlap blocked", path, code)
		}
	}
	if code, _, e := runClaim(t, root, "add", "--session", "B", "src/authz.go"); code != 0 {
		t.Fatalf("sibling with shared prefix: %d %s", code, e)
	}
}

func TestClaimRejectsUnknownFlags(t *testing.T) {
	root := claimTestRoot(t)
	for _, args := range [][]string{
		{"add", "--session", "C", "--ttll", "5", "x.go"},
		{"check", "--sesion", "C", "x.go"},
	} {
		if code, _, errOut := runClaim(t, root, args...); code != 2 || !strings.Contains(errOut, "unknown flag") {
			t.Fatalf("%v: code=%d stderr=%q, want unknown flag rejected", args, code, errOut)
		}
	}
	if code, out, _ := runClaim(t, root, "list", "--all"); code != 0 || !strings.Contains(out, "no claims") {
		t.Fatalf("a claim was recorded: %q", out)
	}
}

// Claims fold case: on a case-insensitive filesystem SRC/Auth and src/auth are
// one tree, and .DEVRITES is the control tree.
func TestClaimFoldsCase(t *testing.T) {
	root := claimTestRoot(t)
	if code, _, e := runClaim(t, root, "add", "--session", "A", "src/auth"); code != 0 {
		t.Fatalf("first add: %d %s", code, e)
	}
	for _, path := range []string{"SRC/Auth/login.go", "Src"} {
		if code, out, _ := runClaim(t, root, "check", "--session", "B", path); code != 3 || !strings.Contains(out, "BLOCKED") {
			t.Fatalf("check %s: code=%d out=%q, want case-folded overlap blocked", path, code, out)
		}
	}
	for _, path := range []string{".DEVRITES/work/x/state.md", ".GIT/hooks/pre-commit"} {
		if code, _, _ := runClaim(t, root, "add", "--session", "B", path); code != 2 {
			t.Fatalf("add %s: code=%d, want control-tree refusal", path, code)
		}
	}
}

// A shell or multiplexer that launched the session outlives it; once the
// recorded holder session exits, the claim is dead even while they run.
func TestClaimHolderSessionExitFreesPathsDespiteLiveShell(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Skip("no hostname")
	}
	old := "2026-10-15T13:10:20Z"
	session := claimProc{PID: 62926, Start: "2026-10-16T01:28:12Z"}
	held := claimRecord{
		ID: "cl-native", Session: "s1", Paths: []string{"pkg/a.go"},
		ClaimedAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		ExpiresAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		Host:      host,
		Anchors:   []claimProc{session, {PID: 4954, Start: old}, {PID: 4062, Start: old}},
		Holder:    &session,
	}
	shellAndMux := map[int]procInfo{4954: {ppid: 4062, start: old, comm: "fish"}, 4062: {ppid: 1, start: old, comm: "herdr"}}
	cases := []struct {
		name   string
		table  map[int]procInfo
		state  string
		addOut int
	}{
		{"session exited", shellAndMux, "dead", 0},
		{"session running", map[int]procInfo{62926: {ppid: 4954, start: session.Start, comm: "omp"}, 4954: shellAndMux[4954], 4062: shellAndMux[4062]}, "live", 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saved := processTable
			processTable = func() map[int]procInfo { return tc.table }
			t.Cleanup(func() { processTable = saved })
			root := claimTestRoot(t)
			line, _ := json.Marshal(held)
			if err := os.WriteFile(filepath.Join(root, ClaimsFile), append(line, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, out, _ := runClaim(t, root, "list"); !strings.Contains(out, "cl-native "+tc.state) {
				t.Fatalf("list state want %s: %q", tc.state, out)
			}
			if code, _, errOut := runClaim(t, root, "add", "--session", "s2", "pkg/a.go"); code != tc.addOut {
				t.Fatalf("add: code=%d want %d stderr=%s", code, tc.addOut, errOut)
			}
		})
	}
}

func TestClaimAddRecordsSupportedSessionIdentity(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Skip("no hostname")
	}
	old := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	parent := os.Getppid()
	saved := processTable
	processTable = func() map[int]procInfo {
		return map[int]procInfo{parent: {ppid: 777, start: old, comm: "/usr/local/bin/omp"}, 777: {ppid: 1, start: old, comm: "-zsh"}}
	}
	t.Cleanup(func() { processTable = saved })
	root := claimTestRoot(t)
	if code, _, e := runClaim(t, root, "add", "--session", "s1", "pkg/a.go"); code != 0 {
		t.Fatalf("add: %d %s", code, e)
	}
	raw, err := os.ReadFile(filepath.Join(root, ClaimsFile))
	if err != nil {
		t.Fatal(err)
	}
	var rec claimRecord
	want := claimProc{PID: parent, Start: old}
	if err := json.Unmarshal(raw, &rec); err != nil || rec.Holder == nil || *rec.Holder != want || rec.Host != host {
		t.Fatalf("host = %q, holder = %+v, want host %q and holder %+v (err %v)", rec.Host, rec.Holder, host, want, err)
	}
}

func TestClaimHolderExitFreesPaths(t *testing.T) {
	host, err := os.Hostname()
	if err != nil {
		t.Skip("no hostname")
	}
	anchor := claimProc{PID: 4242, Start: "2026-10-15T20:00:00Z"}
	held := claimRecord{
		ID:        "cl-held",
		Session:   "s1",
		Paths:     []string{"pkg/a.go"},
		ClaimedAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		ExpiresAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		Host:      host,
		Anchors:   []claimProc{anchor},
	}
	cases := []struct {
		name   string
		table  map[int]procInfo
		host   string
		state  string
		addOut int
	}{
		{"holder running", map[int]procInfo{4242: {ppid: 1, start: anchor.Start}}, host, "live", 3},
		{"holder exited", map[int]procInfo{1: {}}, host, "dead", 0},
		{"pid reused", map[int]procInfo{4242: {ppid: 1, start: "2026-10-16T01:00:00Z"}}, host, "dead", 0},
		{"process table unavailable", nil, host, "live", 3},
		{"other host", map[int]procInfo{1: {}}, "elsewhere", "live", 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saved := processTable
			processTable = func() map[int]procInfo { return tc.table }
			t.Cleanup(func() { processTable = saved })
			root := claimTestRoot(t)
			rec := held
			rec.Host = tc.host
			line, _ := json.Marshal(rec)
			if err := os.WriteFile(filepath.Join(root, ClaimsFile), append(line, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, out, _ := runClaim(t, root, "list"); !strings.Contains(out, "cl-held "+tc.state) {
				t.Fatalf("list state want %s: %q", tc.state, out)
			}
			if code, _, _ := runClaim(t, root, "check", "--session", "s2", "pkg/a.go"); code != tc.addOut {
				t.Fatalf("check: code=%d want %d", code, tc.addOut)
			}
			if code, _, errOut := runClaim(t, root, "add", "--session", "s2", "pkg/a.go"); code != tc.addOut {
				t.Fatalf("add: code=%d want %d stderr=%s", code, tc.addOut, errOut)
			}
		})
	}
}

func TestClaimRetainedWriterRepositoryAuthority(t *testing.T) {
	saved := processTable
	processTable = func() map[int]procInfo { return nil }
	t.Cleanup(func() { processTable = saved })
	root := claimTestRoot(t)
	rec := &claimRecord{
		ID: "cl-retained", Session: "owner", Paths: []string{"pkg/a.go"},
		ClaimedAt: "2020-01-01T00:00:00Z", ExpiresAt: "2020-01-01T00:01:00Z",
	}
	if err := appendClaim(claimsPath(root), rec); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "archive", "feat", "state.md"), "# State\n")
	writeFile(t, filepath.Join(root, "archive", "feat", "dispatch.json"), `{"waves":{"old":{"phase":"build","state":"abandoned","roles":{"slice-wright":{}},"claim":"cl-retained","claim_session":"owner"}}}`)
	for _, sub := range []string{"check", "add"} {
		for _, session := range []string{"owner", "other"} {
			if code, out, errOut := runClaim(t, root, sub, "--session", session, "pkg/a.go"); code != 3 {
				t.Errorf("%s same/foreign expired retained claim: code=%d\n%s%s", sub, code, out, errOut)
			}
		}
	}
	if code, _, errOut := runClaim(t, root, "add", "--session", "owner", "pkg/b.go"); code != 0 {
		t.Fatalf("disjoint: code=%d %s", code, errOut)
	}
	otherRoot := claimTestRoot(t)
	if code, _, errOut := runClaim(t, otherRoot, "add", "--session", "owner", "pkg/a.go"); code != 0 {
		t.Fatalf("other physical repo: code=%d %s", code, errOut)
	}
	if code, _, errOut := runClaim(t, root, "release", "--session", "owner", "--id", rec.ID); code != 0 {
		t.Fatalf("release: code=%d %s", code, errOut)
	}
	if code, _, errOut := runClaim(t, root, "add", "--session", "owner", "pkg/a.go"); code != 0 {
		t.Fatalf("released: code=%d %s", code, errOut)
	}
}

func TestClaimDotFeatureRetainedWriter(t *testing.T) {
	saved := processTable
	processTable = func() map[int]procInfo { return nil }
	t.Cleanup(func() { processTable = saved })
	for _, parent := range []string{"work", "archive"} {
		for _, expired := range []bool{false, true} {
			for _, session := range []string{"owner", "other"} {
				for _, sub := range []string{"check", "add"} {
					t.Run(fmt.Sprintf("%s/expired=%t/%s/%s", parent, expired, session, sub), func(t *testing.T) {
						root := claimTestRoot(t)
						expires := time.Now().UTC().Add(time.Hour)
						if expired {
							expires = time.Now().UTC().Add(-time.Hour)
						}
						rec := &claimRecord{ID: "cl-retained", Session: "owner", Paths: []string{"pkg/a.go"}, ClaimedAt: "2020-01-01T00:00:00Z", ExpiresAt: expires.Format(time.RFC3339)}
						if err := appendClaim(claimsPath(root), rec); err != nil {
							t.Fatal(err)
						}
						dir := filepath.Join(root, parent, ".feature")
						writeFile(t, filepath.Join(dir, "state.md"), "# State\n")
						path := filepath.Join(dir, "dispatch.json")
						writeFile(t, path, `{"waves":{"old":{"phase":"build","state":"abandoned","roles":{"slice-wright":{}},"claim":"cl-retained","claim_session":"owner"}}}`)
						claimsBefore, err := os.ReadFile(claimsPath(root))
						if err != nil {
							t.Fatal(err)
						}
						dispatchBefore, err := os.ReadFile(path)
						if err != nil {
							t.Fatal(err)
						}
						if code, out, stderr := runClaim(t, root, sub, "--session", session, "pkg/a.go"); code != 3 {
							t.Errorf("hidden retained writer admitted: code=%d\n%s%s", code, out, stderr)
						}
						for file, before := range map[string][]byte{claimsPath(root): claimsBefore, path: dispatchBefore} {
							after, err := os.ReadFile(file)
							if err != nil {
								t.Fatal(err)
							}
							if !bytes.Equal(after, before) {
								t.Errorf("refused %s changed %s", sub, file)
							}
						}
					})
				}
			}
		}
	}
}

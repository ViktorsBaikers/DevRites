package lib

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/parallel"
)

// ClaimsFile is the repo-level append-only claim ledger under .devrites.
// Advisory, session-scoped file claims let concurrent agent sessions sharing
// one working tree see each other's write intent before overlapping edits
// collide. A claim never blocks a writer by force — it surfaces the conflict
// so the second session can route around it or stop for the human.
const ClaimsFile = "claims.jsonl"

const (
	claimDefaultTTL = 30
	claimMinTTL     = 1
	claimMaxTTL     = 240
)

const claimUsage = `usage: devrites-engine claim <add|release|list|check> ...

  claim add --session <id> [--ttl <minutes>] [--reason <s>] <path>...
  claim release --session <id> --id <claim-id>
  claim list [--session <id>] [--all]
  claim check [--session <id>] <path>...
`

type claimRecord struct {
	ID        string   `json:"id"`
	Session   string   `json:"session"`
	Paths     []string `json:"paths"`
	Reason    string   `json:"reason,omitempty"`
	ClaimedAt string   `json:"claimed_at"`
	ExpiresAt string   `json:"expires_at"`
	Released  bool     `json:"released,omitempty"`
}

// RunClaim manages the advisory file-claim ledger.
func RunClaim(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, claimUsage)
		return 2
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		return claimAdd(root, rest, stdout, stderr)
	case "release":
		return claimRelease(root, rest, stdout, stderr)
	case "list":
		return claimList(root, rest, stdout, stderr)
	case "check":
		return claimCheck(root, rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "devrites: unknown claim command %q\n", sub)
		return 2
	}
}

func claimsPath(root string) string {
	return filepath.Join(root, ClaimsFile)
}

func readClaims(path string) (latest map[string]*claimRecord, order []string, bad int, err error) {
	latest = map[string]*claimRecord{}
	f, err := os.Open(path) // #nosec G304 -- path is the validated project claims ledger
	if err != nil {
		if os.IsNotExist(err) {
			return latest, nil, 0, nil
		}
		return nil, nil, 0, err
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec claimRecord
		if json.Unmarshal([]byte(line), &rec) != nil || rec.ID == "" || rec.Session == "" {
			bad++
			continue
		}
		if _, seen := latest[rec.ID]; !seen {
			order = append(order, rec.ID)
		}
		r := rec
		latest[rec.ID] = &r
	}
	return latest, order, bad, scanner.Err()
}

func liveClaims(latest map[string]*claimRecord, order []string, now time.Time) []*claimRecord {
	var out []*claimRecord
	for _, id := range order {
		rec := latest[id]
		if rec == nil || rec.Released {
			continue
		}
		expires, err := time.Parse(time.RFC3339, rec.ExpiresAt)
		if err != nil || !expires.After(now) {
			continue
		}
		out = append(out, rec)
	}
	return out
}

func appendClaim(path string, rec *claimRecord) error {
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- path is the validated project claims ledger
	if err != nil {
		return err
	}
	_, werr := f.Write(append(line, '\n'))
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}

func normalizeClaimPaths(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("at least one path is required")
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		norm, err := parallel.NormalizePath(p)
		if err != nil {
			return nil, err
		}
		if norm == ".devrites" || strings.HasPrefix(norm, ".devrites/") ||
			norm == ".git" || strings.HasPrefix(norm, ".git/") {
			return nil, fmt.Errorf("path must not claim control-tree or git internals: %q", norm)
		}
		if _, dup := seen[norm]; dup {
			return nil, fmt.Errorf("duplicate path %q", norm)
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	sort.Strings(out)
	return out, nil
}

func validSession(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '-' || r == '_' || r == '.' || r == ':' || r == '/') {
			return false
		}
	}
	return true
}

// claimConflicts returns live claims by other sessions covering any path.
func claimConflicts(live []*claimRecord, session string, paths []string) []*claimRecord {
	want := map[string]struct{}{}
	for _, p := range paths {
		want[p] = struct{}{}
	}
	var out []*claimRecord
	for _, rec := range live {
		if rec.Session == session {
			continue
		}
		for _, p := range rec.Paths {
			if _, hit := want[p]; hit {
				out = append(out, rec)
				break
			}
		}
	}
	return out
}

func printConflicts(w io.Writer, conflicts []*claimRecord) {
	for _, rec := range conflicts {
		fmt.Fprintf(w, "claim: BLOCKED: %s held by session %q until %s", strings.Join(rec.Paths, ","), rec.Session, rec.ExpiresAt)
		if rec.Reason != "" {
			fmt.Fprintf(w, " (%s)", rec.Reason)
		}
		fmt.Fprintln(w)
	}
}

func newClaimID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("cl-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("cl-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b[:]))
}

func claimAdd(root string, args []string, stdout, stderr io.Writer) int {
	session, reason := "", ""
	ttl := claimDefaultTTL
	var raw []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			session = argAt(args, i+1)
			i++
		case "--reason":
			reason = argAt(args, i+1)
			i++
		case "--ttl":
			n, err := strconv.Atoi(argAt(args, i+1))
			if err != nil {
				fmt.Fprintf(stderr, "claim: --ttl expects minutes, got %q\n", argAt(args, i+1))
				return 2
			}
			ttl = min(max(n, claimMinTTL), claimMaxTTL)
			i++
		default:
			raw = append(raw, args[i])
		}
	}
	if !validSession(session) {
		fmt.Fprintln(stderr, "claim: --session is required (1-128 chars: letters, digits, - _ . : /)")
		return 2
	}
	paths, err := normalizeClaimPaths(raw)
	if err != nil {
		fmt.Fprintf(stderr, "claim: %v\n", err)
		return 2
	}
	path := claimsPath(root)
	latest, order, _, err := readClaims(path)
	if err != nil {
		fmt.Fprintf(stderr, "claim: %v\n", err)
		return 2
	}
	now := time.Now().UTC()
	conflicts := claimConflicts(liveClaims(latest, order, now), session, paths)
	if len(conflicts) > 0 {
		printConflicts(stderr, conflicts)
		return 3
	}
	rec := &claimRecord{
		ID:        newClaimID(),
		Session:   session,
		Paths:     paths,
		Reason:    reason,
		ClaimedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(time.Duration(ttl) * time.Minute).Format(time.RFC3339),
	}
	if err := appendClaim(path, rec); err != nil {
		fmt.Fprintf(stderr, "claim: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "claim: %s claimed %d path(s) until %s\n", rec.ID, len(paths), rec.ExpiresAt)
	return 0
}

func claimRelease(root string, args []string, stdout, stderr io.Writer) int {
	session, id := "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			session = argAt(args, i+1)
			i++
		case "--id":
			id = argAt(args, i+1)
			i++
		default:
			fmt.Fprintf(stderr, "claim: unknown flag %q\n", args[i])
			return 2
		}
	}
	if session == "" || id == "" {
		fmt.Fprintln(stderr, "claim: release requires --session and --id")
		return 2
	}
	path := claimsPath(root)
	latest, _, _, err := readClaims(path)
	if err != nil {
		fmt.Fprintf(stderr, "claim: %v\n", err)
		return 2
	}
	rec, ok := latest[id]
	if !ok {
		fmt.Fprintf(stderr, "claim: unknown claim %q\n", id)
		return 2
	}
	if rec.Session != session {
		fmt.Fprintf(stderr, "claim: %q belongs to session %q, not %q\n", id, rec.Session, session)
		return 2
	}
	if rec.Released {
		fmt.Fprintf(stderr, "claim: %q already released\n", id)
		return 2
	}
	released := *rec
	released.Released = true
	if err := appendClaim(path, &released); err != nil {
		fmt.Fprintf(stderr, "claim: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "claim: %s released %d path(s)\n", id, len(rec.Paths))
	return 0
}

func claimList(root string, args []string, stdout, stderr io.Writer) int {
	session := ""
	all := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			session = argAt(args, i+1)
			i++
		case "--all":
			all = true
		default:
			fmt.Fprintf(stderr, "claim: unknown flag %q\n", args[i])
			return 2
		}
	}
	latest, order, bad, err := readClaims(claimsPath(root))
	if err != nil {
		fmt.Fprintf(stderr, "claim: %v\n", err)
		return 2
	}
	now := time.Now().UTC()
	rows := liveClaims(latest, order, now)
	if all {
		rows = rows[:0]
		for _, id := range order {
			rows = append(rows, latest[id])
		}
	}
	printed := 0
	for _, rec := range rows {
		if rec == nil || (session != "" && rec.Session != session) {
			continue
		}
		state := "live"
		if rec.Released {
			state = "released"
		} else if exp, err := time.Parse(time.RFC3339, rec.ExpiresAt); err != nil || !exp.After(now) {
			state = "expired"
		}
		fmt.Fprintf(stdout, "%s %-8s session=%s paths=%s until=%s", rec.ID, state, rec.Session, strings.Join(rec.Paths, ","), rec.ExpiresAt)
		if rec.Reason != "" {
			fmt.Fprintf(stdout, " reason=%q", rec.Reason)
		}
		fmt.Fprintln(stdout)
		printed++
	}
	if printed == 0 {
		fmt.Fprintln(stdout, "claim: no claims")
	}
	if bad > 0 {
		fmt.Fprintf(stdout, "claim: %d unparsed ledger line(s)\n", bad)
	}
	return 0
}

func claimCheck(root string, args []string, stdout, stderr io.Writer) int {
	session := ""
	var raw []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			session = argAt(args, i+1)
			i++
		default:
			raw = append(raw, args[i])
		}
	}
	paths, err := normalizeClaimPaths(raw)
	if err != nil {
		fmt.Fprintf(stderr, "claim: %v\n", err)
		return 2
	}
	latest, order, _, err := readClaims(claimsPath(root))
	if err != nil {
		fmt.Fprintf(stderr, "claim: %v\n", err)
		return 2
	}
	conflicts := claimConflicts(liveClaims(latest, order, time.Now().UTC()), session, paths)
	if len(conflicts) > 0 {
		printConflicts(stdout, conflicts)
		return 3
	}
	fmt.Fprintf(stdout, "claim: %d path(s) free\n", len(paths))
	return 0
}

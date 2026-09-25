// Package snapshot implements the /overhaul snapshot tool (snapshot overhaul/1):
// baseline capture and drift check without touching the repository.
//
// Git runs read-only: GIT_OPTIONAL_LOCKS=0, no index refresh, no external diff
// drivers; no refs, stash or commit. Submodules and nested repositories are
// recorded as boundaries, never read. Files matching sensitive name patterns or
// holding private key material are hashed, never copied.
package snapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/gitenv"
	"github.com/devrites/devrites/internal/overhaul/ovio"
)

const usage = `usage: devrites-engine overhaul snapshot <command> ...

  capture <repo> <out>        copy tracked + non-ignored untracked files to <out>/tree, record index/HEAD state
  fingerprint <repo>          content fingerprint of the working tree (tracked + non-ignored untracked)
  verify <repo> <out> [--agent-paths <file>]   report index changes, user drift, and agent-owned changes
  state <repo> <out.json>     write per-file hashes (the before-image of one attempt)
  delta <before.json> <repo>  print paths whose content or mode changed since <before.json>, one per line
`

var sensitive = []string{".env", ".env.*", "*.key", "*.p12", "*.pfx", "*.keystore", "*.jks", ".npmrc", ".pypirc", ".netrc",
	"*credentials*", "*secret*", "*.tfstate", "*.tfvars"}

// keyMarker flags private key material in any file, whatever its name.
var keyMarker = []byte("PRIVATE KEY-----")

var diffArgs = []string{"-c", "diff.autoRefreshIndex=false", "diff", "--no-ext-diff", "--no-textconv", "--no-color",
	"--src-prefix=a/", "--dst-prefix=b/", "--binary"}

// Fields are declared in sorted order so the manifest matches sort_keys output.
type entry struct {
	Copied    bool   `json:"copied"`
	Mode      int    `json:"mode"`
	Sensitive bool   `json:"sensitive"`
	SHA256    string `json:"sha256"`
	Source    string `json:"source"`
}

type manifest struct {
	Fingerprint string            `json:"fingerprint"`
	Files       map[string]*entry `json:"files"`
	Head        *string           `json:"head"`
	IndexSHA256 string            `json:"index_sha256"`
	Root        string            `json:"root"`
	Schema      string            `json:"schema"`
	Submodules  []string          `json:"submodules_not_captured"`
}

// Run executes the tool with its arguments (tool name excluded).
func Run(args []string, stdout, stderr io.Writer) int {
	code, err := dispatch(args, stdout)
	if err != nil {
		fmt.Fprintln(stderr, err)
	}
	return code
}

var errUsage = errors.New(strings.TrimRight(usage, "\n"))

func dispatch(a []string, stdout io.Writer) (int, error) {
	if len(a) == 0 {
		return 2, errUsage
	}
	switch {
	case a[0] == "capture" && len(a) == 3:
		return exit(capture(a[1], a[2], stdout))
	case a[0] == "fingerprint" && len(a) == 2:
		st, err := state(a[1])
		if err != nil {
			return 2, err
		}
		fmt.Fprintln(stdout, fingerprint(st))
		return 0, nil
	case a[0] == "verify" && (len(a) == 3 || len(a) == 5 && a[3] == "--agent-paths"):
		agent := map[string]bool{}
		if len(a) == 5 {
			b, err := os.ReadFile(a[4])
			if err != nil {
				return 2, err
			}
			for _, ln := range strings.Split(string(b), "\n") {
				if ln = strings.TrimSpace(ln); ln != "" {
					agent[ln] = true
				}
			}
		}
		return verify(a[1], a[2], agent, stdout)
	case a[0] == "state" && len(a) == 3:
		st, err := state(a[1])
		if err != nil {
			return 2, err
		}
		out := map[string][]any{}
		for p, v := range st {
			out[p] = []any{v.SHA256, v.Mode}
		}
		return exit(ovio.WriteJSON(a[2], out, 0o644))
	case a[0] == "delta" && len(a) == 3:
		return exit(delta(a[1], a[2], stdout))
	}
	return 2, errUsage
}

func exit(err error) (int, error) {
	if err != nil {
		return 2, err
	}
	return 0, nil
}

func git(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(gitenv.Sanitize(os.Environ()), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, bytes.TrimSpace(ee.Stderr))
		}
		return nil, fmt.Errorf("git %s: %v", strings.Join(args, " "), err)
	}
	return out, nil
}

func split0(b []byte) []string {
	var out []string
	for _, s := range strings.Split(string(b), "\x00") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// files lists tracked paths and non-ignored untracked paths, each sorted and deduplicated.
func files(root string) (tracked, untracked []string, err error) {
	t, err := git(root, "ls-files", "-z", "--cached")
	if err != nil {
		return nil, nil, err
	}
	u, err := git(root, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, nil, err
	}
	tracked = split0(t)
	sort.Strings(tracked)
	tracked = compact(tracked)
	seen := map[string]bool{}
	for _, p := range tracked {
		seen[p] = true
	}
	for _, p := range split0(u) {
		if !seen[p] {
			seen[p] = true
			untracked = append(untracked, p)
		}
	}
	sort.Strings(untracked)
	return tracked, untracked, nil
}

func compact(s []string) []string {
	out := s[:0]
	for i, p := range s {
		if i == 0 || p != s[i-1] {
			out = append(out, p)
		}
	}
	return out
}

// permBits mirrors Python's stat.S_IMODE: permission plus setuid/setgid/sticky bits.
func permBits(m fs.FileMode) int {
	b := int(m.Perm())
	if m&fs.ModeSetuid != 0 {
		b |= 0o4000
	}
	if m&fs.ModeSetgid != 0 {
		b |= 0o2000
	}
	if m&fs.ModeSticky != 0 {
		b |= 0o1000
	}
	return b
}

func digest(p string) (string, int, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		return "deleted", 0, nil
	}
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		target, err := os.Readlink(p)
		if err != nil {
			return "", 0, err
		}
		return "link:" + ovio.SHA256Bytes([]byte(target)), 0o120000, nil
	case fi.IsDir():
		return "directory", 0, nil
	}
	sum, err := ovio.SHA256File(p)
	return sum, permBits(fi.Mode()), err
}

func holdsKey(p string) (bool, error) {
	fi, err := os.Lstat(p)
	if err != nil || !fi.Mode().IsRegular() {
		return false, nil
	}
	f, err := os.Open(p)
	if err != nil {
		return false, err
	}
	defer f.Close()
	head, err := io.ReadAll(io.LimitReader(f, 65536))
	return bytes.Contains(head, keyMarker), err
}

func isSensitiveName(p string) bool {
	base := p[strings.LastIndex(p, "/")+1:]
	for _, pat := range sensitive {
		if ok, _ := path.Match(pat, base); ok {
			return true
		}
	}
	return false
}

func state(root string) (map[string]*entry, error) {
	tracked, untracked, err := files(root)
	if err != nil {
		return nil, err
	}
	out := map[string]*entry{}
	for _, g := range []struct {
		src   string
		paths []string
	}{{"tracked", tracked}, {"untracked", untracked}} {
		for _, p := range g.paths {
			full := filepath.Join(root, filepath.FromSlash(p))
			sum, mode, err := digest(full)
			if err != nil {
				return nil, err
			}
			sens := isSensitiveName(p)
			if !sens {
				if sens, err = holdsKey(full); err != nil {
					return nil, err
				}
			}
			out[p] = &entry{SHA256: sum, Mode: mode, Source: g.src, Sensitive: sens}
		}
	}
	return out, nil
}

func indexDigest(root string) (string, error) {
	b, err := git(root, "ls-files", "-s", "-z")
	if err != nil {
		return "", err
	}
	return ovio.SHA256Bytes(b), nil
}

// fingerprint is sha256 over sorted "path\0sha\0octal-mode\n" lines.
func fingerprint(st map[string]*entry) string {
	paths := make([]string, 0, len(st))
	for p := range st {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var buf bytes.Buffer
	for _, p := range paths {
		fmt.Fprintf(&buf, "%s\x00%s\x00%o\n", p, st[p].SHA256, st[p].Mode)
	}
	return ovio.SHA256Bytes(buf.Bytes())
}

func capture(root, out string, stdout io.Writer) error {
	if _, err := os.Lstat(out); err == nil {
		return fmt.Errorf("refusing to overwrite existing snapshot %s", out)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	root = ovio.Resolve(absRoot)
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	if strings.HasPrefix(ovio.Resolve(absOut), root+string(os.PathSeparator)) {
		return errors.New("snapshot must live outside the target repository")
	}
	idx, err := git(root, "ls-files", "-s", "-z")
	if err != nil {
		return err
	}
	entries := split0(idx)
	subs := map[string]bool{}
	for _, e := range entries {
		meta, p, _ := strings.Cut(e, "\t")
		if f := strings.Fields(meta); len(f) < 3 || f[2] != "0" {
			return errors.New("index has unmerged entries; ask the user which baseline to capture")
		}
		if strings.HasPrefix(e, "160000 ") {
			subs[p] = true
		}
	}
	before := ovio.SHA256Bytes(idx)
	st, err := state(root)
	if err != nil {
		return err
	}
	for p, v := range st {
		if v.SHA256 == "directory" {
			subs[p] = true
		}
	}
	gitlinks := []string{}
	for p := range subs {
		gitlinks = append(gitlinks, p)
	}
	sort.Strings(gitlinks)

	if err := os.MkdirAll(filepath.Join(out, "tree"), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(out, 0o700); err != nil {
		return err
	}
	for p, v := range st {
		v.Copied = !v.Sensitive && v.SHA256 != "deleted" && v.SHA256 != "directory"
		if v.Copied {
			if err := copyEntry(filepath.Join(root, filepath.FromSlash(p)), filepath.Join(out, "tree", filepath.FromSlash(p))); err != nil {
				return err
			}
		}
	}
	// Sensitive paths stay out of the patches too: hash only, never content.
	safe := []string{"--", "."}
	for p, v := range st {
		if v.Sensitive {
			safe = append(safe, ":(top,exclude,literal)"+p)
		}
	}
	for _, o := range []struct {
		name string
		args []string
	}{
		{"status.bin", []string{"status", "--porcelain=v2", "-z", "--untracked-files=all"}},
		{"index.bin", []string{"ls-files", "-s", "-z"}},
		{"staged.patch", slices.Concat(diffArgs, []string{"--cached"}, safe)},
		{"unstaged.patch", slices.Concat(diffArgs, safe)},
	} {
		b, err := git(root, o.args...)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, o.name), b, 0o600); err != nil {
			return err
		}
	}
	var head *string
	if b, err := git(root, "rev-parse", "--verify", "-q", "HEAD"); err == nil {
		if h := strings.TrimSpace(string(b)); h != "" {
			head = &h
		}
	}
	after, err := indexDigest(root)
	if err != nil {
		return err
	}
	if after != before {
		return errors.New("index changed during capture; stop and reconcile with the user")
	}
	now, err := state(root)
	if err != nil {
		return err
	}
	man := manifest{Schema: "overhaul.snapshot/1", Root: root, Head: head, IndexSHA256: before,
		Fingerprint: fingerprint(now), Files: st, Submodules: gitlinks}
	if err := ovio.WriteJSON(filepath.Join(out, "manifest.json"), man, 0o600); err != nil {
		return err
	}
	withheld := []string{}
	for p, v := range st {
		if v.Sensitive {
			withheld = append(withheld, p)
		}
	}
	sort.Strings(withheld)
	b, err := json.Marshal(struct {
		Snapshot    string   `json:"snapshot"`
		Fingerprint string   `json:"fingerprint"`
		Files       int      `json:"files"`
		Withheld    []string `json:"sensitive_not_copied"`
	}{out, man.Fingerprint, len(st), withheld})
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

// copyEntry copies like shutil.copy2(follow_symlinks=False): symlinks stay
// symlinks; files keep their mode and modification time.
func copyEntry(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o777); err != nil {
		return err
	}
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	w, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(dst, fi.Mode())
	}
	if err == nil {
		err = os.Chtimes(dst, time.Time{}, fi.ModTime())
	}
	return err
}

func verify(root, out string, agent map[string]bool, stdout io.Writer) (int, error) {
	b, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		return 2, err
	}
	var man manifest
	if err := json.Unmarshal(b, &man); err != nil {
		return 2, fmt.Errorf("%s: invalid manifest (%v)", out, err)
	}
	now, err := state(root)
	if err != nil {
		return 2, err
	}
	union := map[string]bool{}
	for p := range man.Files {
		union[p] = true
	}
	for p := range now {
		union[p] = true
	}
	report := struct {
		IndexUnchanged bool     `json:"index_unchanged"`
		Agent          []string `json:"agent_owned_changes"`
		User           []string `json:"changes_outside_agent_paths"`
		Fingerprint    string   `json:"fingerprint"`
	}{Agent: []string{}, User: []string{}, Fingerprint: fingerprint(now)}
	for p := range union {
		was, cur := man.Files[p], now[p]
		if cur == nil {
			cur = &entry{SHA256: "deleted"}
		}
		if was != nil && was.SHA256 == cur.SHA256 && was.Mode == cur.Mode {
			continue
		}
		if agent[p] {
			report.Agent = append(report.Agent, p)
		} else {
			report.User = append(report.User, p)
		}
	}
	sort.Strings(report.Agent)
	sort.Strings(report.User)
	idx, err := indexDigest(root)
	if err != nil {
		return 2, err
	}
	report.IndexUnchanged = idx == man.IndexSHA256
	j, err := ovio.MarshalIndent(report)
	if err != nil {
		return 2, err
	}
	stdout.Write(j)
	if report.IndexUnchanged && len(report.User) == 0 {
		return 0, nil
	}
	return 1, nil
}

func delta(beforePath, root string, stdout io.Writer) error {
	b, err := os.ReadFile(beforePath)
	if err != nil {
		return err
	}
	var old map[string]any
	if err := json.Unmarshal(b, &old); err != nil {
		return fmt.Errorf("%s: invalid state (%v)", beforePath, err)
	}
	st, err := state(root)
	if err != nil {
		return err
	}
	union := map[string]bool{}
	for p := range old {
		union[p] = true
	}
	for p := range st {
		union[p] = true
	}
	paths := make([]string, 0, len(union))
	for p := range union {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	missing := []any{"deleted", float64(0)}
	for _, p := range paths {
		was, ok := old[p]
		if !ok {
			was = missing
		}
		cur := missing
		if v := st[p]; v != nil {
			cur = []any{v.SHA256, float64(v.Mode)}
		}
		if !reflect.DeepEqual(was, cur) {
			fmt.Fprintln(stdout, p)
		}
	}
	return nil
}

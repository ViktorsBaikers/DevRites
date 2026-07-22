package lib

import (
	"bufio"
	"context"
	"crypto/sha1" // #nosec G505 -- non-cryptographic decision-index id; persisted hashes depend on sha1
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
)

type codeHealthEntry struct {
	TS         string            `json:"ts"`
	Status     string            `json:"status"`
	Categories map[string]string `json:"categories"`
	Notes      []string          `json:"notes,omitempty"`
}

// FirstTask classifies the current repo into one deterministic next-action token.
func FirstTask(root string, _ []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, FirstTaskToken(root))
	return 0
}

// FirstTaskToken is the classification behind the first-task command, exposed so
// the SessionStart orientation can render a first-run nudge from the same enum.
// The token set is load-bearing: callers map tokens to prose, so a new token
// needs a new mapping, never a reworded one.
func FirstTaskToken(root string) string {
	project := projectDir(root)
	if isGitDirty(project) {
		return "dirty-worktree"
	}
	if activeSlug(root) != "" {
		return "active-feature"
	}
	if gitBranchAhead(project) {
		return "branch-ahead"
	}
	if !isDir(root) || !isDir(filepath.Join(root, "work")) && !isDir(filepath.Join(root, "specs")) && !isDir(filepath.Join(root, "archive")) {
		if hasProjectFiles(project) {
			return "brownfield-unadopted"
		}
		return "greenfield"
	}
	if hasProjectFiles(project) {
		return "clean-default"
	}
	return "greenfield"
}

func projectDir(root string) string {
	if filepath.Base(root) == ".devrites" {
		return filepath.Dir(root)
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func hasProjectFiles(project string) bool {
	for _, name := range []string{"package.json", "go.mod", "pyproject.toml", "Cargo.toml", "pom.xml", "Gemfile", "Makefile", "README.md", ".git"} {
		if _, err := os.Stat(filepath.Join(project, name)); err == nil {
			return true
		}
	}
	return false
}

func isGitDirty(project string) bool {
	out, err := boundedCommandOutput(15*time.Second, "", "git", "-C", project, "status", "--porcelain")
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func gitBranchAhead(project string) bool {
	out, err := boundedCommandOutput(15*time.Second, "", "git", "-C", project, "status", "-sb", "--porcelain")
	return err == nil && strings.Contains(string(out), "ahead ")
}

// Config reads the tiny project-local config surface. Missing outside_voice defaults to auto.
func Config(root string, args []string, stdout, stderr io.Writer) int {
	if argAt(args, 0) != "get" || argAt(args, 1) == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine config get <key>")
		return 2
	}
	key := argAt(args, 1)
	vals := readDevritesConfig(filepath.Join(root, "config"))
	if v := vals[key]; v != "" {
		fmt.Fprintln(stdout, v)
		return 0
	}
	if key == "outside_voice" {
		fmt.Fprintln(stdout, "auto")
		return 0
	}
	fmt.Fprintln(stdout)
	return 0
}

func readDevritesConfig(path string) map[string]string {
	out := map[string]string{}
	for _, ext := range []string{".toml", ".env", ".md", ""} {
		b, err := os.ReadFile(path + ext)
		if err != nil {
			continue
		}
		for _, line := range splitLinesNoTrailing(b) {
			line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if k, v, ok := strings.Cut(line, "="); ok {
				out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
				continue
			}
			if k, v, ok := strings.Cut(line, ":"); ok {
				out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
			}
		}
		break
	}
	return out
}

// OutsideVoice reports whether a Codex second opinion can run under outside_voice=auto.
func OutsideVoice(root string, args []string, stdout, stderr io.Writer) int {
	modeBuf := &strings.Builder{}
	_ = Config(root, []string{"get", "outside_voice"}, modeBuf, stderr)
	mode := strings.TrimSpace(modeBuf.String())
	if mode == "disabled" || mode == "off" || mode == "false" {
		fmt.Fprintln(stdout, "outside-voice: disabled")
		return 0
	}
	if _, err := exec.LookPath("codex"); err != nil {
		fmt.Fprintln(stdout, "outside-voice: skipped-unavailable")
		return 0
	}
	fmt.Fprintln(stdout, "outside-voice: available")
	return 0
}

// CodeHealth runs a read-only, best-effort project quality dashboard and records the trend.
func CodeHealth(root string, args []string, stdout, stderr io.Writer) int {
	project := projectDir(root)
	cats := map[string]string{}
	notes := []string{}
	run := func(name string, argv ...string) {
		out, err := boundedCommandOutput(10*time.Minute, project, argv[0], argv[1:]...)
		if err == nil {
			cats[name] = "PASS"
			return
		}
		cats[name] = "FAIL"
		line := firstLine(strings.TrimSpace(string(out)))
		if line != "" {
			notes = append(notes, name+": "+line)
		}
	}
	if isFile(filepath.Join(project, "package.json")) {
		if npmScript(project, "test") {
			run("tests", "npm", "test")
		} else {
			cats["tests"] = "WARN no npm test script"
		}
		for _, s := range []string{"lint", "typecheck", "build"} {
			if npmScript(project, s) {
				run(s, "npm", "run", s)
			}
		}
	}
	if isFile(filepath.Join(project, "go.mod")) {
		run("go-test", "go", "test", "./...")
	}
	if isFile(filepath.Join(project, "pyproject.toml")) && commandExists("pytest") {
		run("pytest", "pytest")
	}
	if isFile(filepath.Join(project, "scripts", "devrites-detect.sh")) {
		run("secret-slop-scan", "bash", "scripts/devrites-detect.sh")
	}
	if isFile(filepath.Join(project, "package.json")) && npmScript(project, "validate") {
		run("pack-validity", "npm", "run", "validate")
	}
	status := "PASS"
	for _, v := range cats {
		if strings.HasPrefix(v, "FAIL") {
			status = "FAIL"
			break
		}
		if strings.HasPrefix(v, "WARN") && status == "PASS" {
			status = "WARN"
		}
	}
	if len(cats) == 0 {
		status = "WARN"
		cats["detect"] = "WARN no known checks"
	}
	entry := codeHealthEntry{TS: nowUTC(), Status: status, Categories: cats, Notes: notes}
	_ = os.MkdirAll(root, 0o755)
	if err := appendJSONLine(filepath.Join(root, "health.jsonl"), entry); err != nil {
		fmt.Fprintf(stderr, "health: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Health: %s\n", status)
	keys := make([]string, 0, len(cats))
	for k := range cats {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(stdout, "%-18s %s\n", k, cats[k])
	}
	if len(notes) > 0 {
		fmt.Fprintln(stdout, "Notes:")
		for _, n := range notes {
			fmt.Fprintf(stdout, "- %s\n", n)
		}
	}
	return 0
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func commandExists(name string) bool { _, err := exec.LookPath(name); return err == nil }

func boundedCommandOutput(timeout time.Duration, dir, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	configureBoundedCommand(cmd)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func npmScript(project, name string) bool {
	b, err := os.ReadFile(filepath.Join(project, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]any `json:"scripts"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return false
	}
	_, ok := pkg.Scripts[name]
	return ok
}

type decisionIndexEntry struct {
	Slug string `json:"slug"`
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
	Hash string `json:"hash"`
}

// Decisions indexes/searches archived per-feature decisions.md files.
func Decisions(root string, args []string, stdout, stderr io.Writer) int {
	switch argAt(args, 0) {
	case "index":
		entries, err := collectDecisions(root)
		if err != nil {
			fmt.Fprintf(stderr, "decisions: %v\n", err)
			return 1
		}
		var b strings.Builder
		for _, e := range entries {
			j, _ := json.Marshal(e)
			b.Write(j)
			b.WriteByte('\n')
		}
		_ = os.MkdirAll(root, 0o755)
		if err := fsutil.WriteFileAtomic(filepath.Join(root, "decisions-index.jsonl"), []byte(b.String()), 0o644); err != nil {
			fmt.Fprintf(stderr, "decisions: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "decisions: indexed %d entr%s.\n", len(entries), pluralY(len(entries)))
		return 0
	case "search":
		q := strings.ToLower(strings.Join(args[1:], " "))
		if strings.TrimSpace(q) == "" {
			fmt.Fprintln(stderr, "usage: devrites-engine decisions search <query>")
			return 2
		}
		entries, err := collectDecisions(root)
		if err != nil {
			fmt.Fprintf(stderr, "decisions: %v\n", err)
			return 1
		}
		terms := strings.Fields(q)
		matches := []decisionIndexEntry{}
		for _, e := range entries {
			low := strings.ToLower(e.Text)
			ok := true
			for _, term := range terms {
				if !strings.Contains(low, term) {
					ok = false
					break
				}
			}
			if ok {
				matches = append(matches, e)
			}
		}
		for _, e := range matches {
			fmt.Fprintf(stdout, "%s:%d %s: %s\n", e.Path, e.Line, e.Slug, e.Text)
		}
		if len(matches) == 0 {
			fmt.Fprintln(stdout, "decisions: no matches.")
		}
		return 0
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine decisions index|search <query>")
		return 2
	}
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func collectDecisions(root string) ([]decisionIndexEntry, error) {
	paths := []string{}
	for _, base := range []string{filepath.Join(root, "archive"), filepath.Join(root, "work")} {
		err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && filepath.Base(p) == "decisions.md" {
				paths = append(paths, p)
			}
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("walk decisions in %s: %w", base, err)
		}
	}
	sort.Strings(paths)
	var out []decisionIndexEntry
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("open decisions %s: %w", p, err)
		}
		slug := filepath.Base(filepath.Dir(p))
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 64*1024), 1024*1024)
		lineNo := 0
		for s.Scan() {
			lineNo++
			text := strings.TrimSpace(s.Text())
			if !strings.HasPrefix(text, "-") && !strings.HasPrefix(text, "*") && !strings.HasPrefix(text, "ADR") {
				continue
			}
			text = strings.TrimSpace(strings.TrimLeft(text, "-* "))
			if text == "" {
				continue
			}
			h := sha1.Sum([]byte(slug + "\n" + text)) // #nosec G401 -- stable 12-hex content id, not a security boundary
			rel, _ := filepath.Rel(projectDir(root), p)
			out = append(out, decisionIndexEntry{Slug: slug, Path: rel, Line: lineNo, Text: text, Hash: hex.EncodeToString(h[:])[:12]})
		}
		if err := s.Err(); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("scan decisions %s: %w", p, err)
		}
		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("close decisions %s: %w", p, err)
		}
	}
	return out, nil
}

type secretFinding struct{ Severity, Path, Kind, Match string }

var secretPatterns = []struct {
	sev, kind string
	re        *regexp.Regexp
}{
	{"HIGH", "private-key", regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY-----`)},
	{"HIGH", "aws-secret", regexp.MustCompile(`(?i)aws(.{0,20})?(secret|access).{0,20}[=:]\s*['\"]?[A-Za-z0-9/+=]{30,}`)},
	{"HIGH", "github-token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{30,}`)},
	{"HIGH", "slack-token", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{20,}`)},
	{"MEDIUM", "generic-secret", regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)\s*[:=]\s*['\"][^'\"\n]{16,}['\"]`)},
	{"LOW", "placeholder-secret", regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password).{0,20}(example|changeme|placeholder|dummy)`)},
}

// SecretScan scans staged/touched text for credentials. HIGH exits 3.
func SecretScan(root string, args []string, stdout, stderr io.Writer) int {
	project := projectDir(root)
	paths := []string{}
	texts := map[string]string{}
	slug := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--staged":
			paths = append(paths, gitStagedNames(project)...)
		case "--text":
			i++
			if i < len(args) {
				texts["<text>"] = args[i]
			}
		default:
			if slug == "" {
				slug = args[i]
			}
		}
	}
	if slug == "" {
		slug = activeSlug(root)
	}
	if slug != "" {
		paths = append(paths, touchedFiles(root, slug)...)
	}
	if len(paths) == 0 {
		paths = append(paths, gitDiffNames(project)...)
	}
	findings := []secretFinding{}
	seen := map[string]bool{}
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		b, err := os.ReadFile(filepath.Join(project, p))
		if err != nil {
			continue
		}
		findings = append(findings, scanSecrets(p, string(b))...)
	}
	for p, t := range texts {
		findings = append(findings, scanSecrets(p, t)...)
	}
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "secret-scan: PASS")
		return 0
	}
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Severity+findings[i].Path < findings[j].Severity+findings[j].Path
	})
	high := false
	for _, f := range findings {
		fmt.Fprintf(stdout, "%s %s %s %s\n", f.Severity, f.Path, f.Kind, f.Match)
		if f.Severity == "HIGH" {
			high = true
		}
	}
	if high {
		return 3
	}
	return 0
}

func scanSecrets(path, text string) []secretFinding {
	var out []secretFinding
	for _, p := range secretPatterns {
		for _, m := range p.re.FindAllString(text, -1) {
			if len(m) > 80 {
				m = m[:80] + "…"
			}
			out = append(out, secretFinding{Severity: p.sev, Path: path, Kind: p.kind, Match: m})
		}
	}
	return out
}

func gitStagedNames(project string) []string {
	out, err := boundedCommandOutput(15*time.Second, "", "git", "-C", project, "diff", "--cached", "--name-only")
	if err != nil {
		return nil
	}
	return splitLinesNoTrailing(out)
}

func touchedFiles(root, slug string) []string {
	data, ok := readFileOK(filepath.Join(featureDir(root, slug), "touched-files.md"))
	if !ok {
		return nil
	}
	var out []string
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "-* `"))
		line = strings.TrimRight(line, "`")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, strings.Fields(line)[0])
	}
	return out
}

// DocsStale warns when public surfaces changed without docs changes.
func DocsStale(root string, args []string, stdout, stderr io.Writer) int {
	project := projectDir(root)
	slug := slugOrActive(root, args)
	changed := touchedFiles(root, slug)
	if len(changed) == 0 {
		changed = gitDiffNames(project)
	}
	docsTouched, surfaceTouched := false, []string{}
	for _, p := range changed {
		if isDocPath(p) {
			docsTouched = true
		}
		if isPublicSurfacePath(p) {
			surfaceTouched = append(surfaceTouched, p)
		}
	}
	if len(surfaceTouched) == 0 || docsTouched {
		fmt.Fprintln(stdout, "docs-stale: PASS")
		return 0
	}
	fmt.Fprintf(stdout, "docs-stale: WARN public surface changed without docs: %s\n", strings.Join(surfaceTouched, ", "))
	return 0
}

func isDocPath(p string) bool {
	base := strings.ToLower(filepath.Base(p))
	return strings.HasPrefix(p, "docs/") || base == "readme.md" || base == "security.md" || strings.HasSuffix(base, ".md")
}

func isPublicSurfacePath(p string) bool {
	low := strings.ToLower(p)
	if strings.HasPrefix(low, "docs/") || strings.HasSuffix(low, ".md") {
		return false
	}
	needles := []string{"cli", "cmd", "api", "route", "handler", "public", "export", "install", "readme", "package.json", "openapi", "schema"}
	for _, n := range needles {
		if strings.Contains(low, n) {
			return true
		}
	}
	return false
}

// SpecDedupe searches local PRDs/issues and archived specs for likely prior art.
func SpecDedupe(root string, args []string, stdout, stderr io.Writer) int {
	query := strings.Join(args, " ")
	terms := dedupeTerms(query)
	if len(terms) == 0 {
		fmt.Fprintln(stderr, "usage: devrites-engine spec-dedupe <feature words>")
		return 2
	}
	project := projectDir(root)
	targets := []string{filepath.Join(project, ".scratch"), filepath.Join(root, "archive")}
	type hit struct {
		score      int
		path, line string
	}
	var hits []hit
	for _, base := range targets {
		_ = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".md") {
				return nil
			}
			lowName := strings.ToLower(filepath.Base(p))
			if !strings.Contains(p, string(filepath.Separator)+"archive"+string(filepath.Separator)) && lowName != "issue.md" && lowName != "prd.md" && !strings.Contains(lowName, "prd") {
				return nil
			}
			b, err := os.ReadFile(p) // #nosec G122 -- scoring walk over the project's own .scratch tree; a symlink race requires an attacker already writing to the checkout
			if err != nil {
				return nil
			}
			text := strings.ToLower(string(b))
			score := 0
			for _, term := range terms {
				if strings.Contains(text, term) {
					score++
				}
			}
			if score >= 2 || score == len(terms) {
				rel, _ := filepath.Rel(project, p)
				hits = append(hits, hit{score: score, path: filepath.ToSlash(rel), line: firstLine(string(b))})
			}
			return nil
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score == hits[j].score {
			return hits[i].path < hits[j].path
		}
		return hits[i].score > hits[j].score
	})
	if len(hits) == 0 {
		fmt.Fprintln(stdout, "spec-dedupe: no local matches.")
		return 0
	}
	limit := len(hits)
	if limit > 5 {
		limit = 5
	}
	fmt.Fprintln(stdout, "spec-dedupe: possible local matches:")
	for _, h := range hits[:limit] {
		fmt.Fprintf(stdout, "- %s (%d/%d): %s\n", h.path, h.score, len(terms), strings.TrimSpace(h.line))
	}
	return 0
}

func dedupeTerms(q string) []string {
	stop := map[string]bool{"the": true, "and": true, "for": true, "with": true, "that": true, "this": true, "from": true, "into": true, "feature": true, "add": true, "make": true}
	seen := map[string]bool{}
	var out []string
	for _, raw := range regexp.MustCompile(`[A-Za-z0-9_-]+`).FindAllString(strings.ToLower(q), -1) {
		if len(raw) < 4 || stop[raw] || seen[raw] {
			continue
		}
		seen[raw] = true
		out = append(out, raw)
		if len(out) == 4 {
			break
		}
	}
	return out
}

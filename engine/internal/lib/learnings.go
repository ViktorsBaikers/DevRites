package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Learnings is the cross-feature learning ledger for /rite-learn. Its state is
// project-wide, not per-feature: the ledger, the archive it mines, and the review
// marker all live directly under root (the .devrites directory). Subcommands:
//
//	add <slug> "<text>" [tag] [conf]  append a dated entry (conf = 0..1 confidence)
//	list                        print the ledger, or note that it is empty
//	top                         print the high-confidence entries eligible for injection
//	mine [archive-dir]          cluster finding/decision phrases that recur (>= 2)
//	nudge [archive-dir]         surface a phrase recurring >= 3x, if new since review
//
// A ledger entry may carry a confidence token (`c=0.8`) in its metadata. Only
// entries at or above DEVRITES_LEARNING_CONFIDENCE_THRESHOLD (default 0.7) are
// injected into the SessionStart orientation, capped at
// DEVRITES_MAX_INJECTED_LEARNINGS (default 5). An unmarked entry defaults to 0.5
// — below the floor — so nothing auto-injects until a human promotes it via
// /rite-learn. The human gate is preserved; only the bounded surfacing is new.
//
// Exit codes: 0 ok · 2 bad args.
func Learnings(root string, args []string, stdout, stderr io.Writer) int {
	ledger := filepath.Join(root, "learnings.md")
	cmd := argAt(args, 0)

	switch cmd {
	case "add":
		slug := argAt(args, 1)
		text := argAt(args, 2)
		tag := argAt(args, 3)
		confArg := argAt(args, 4)
		if tag == "" {
			tag = "note"
		}
		if text == "" {
			fmt.Fprintln(stderr, `usage: devrites-engine learnings add <slug> "<text>" [tag] [confidence 0..1]`)
			return 2
		}
		meta := tag
		if confArg != "" {
			c, ok := parseConfidence(confArg)
			if !ok {
				fmt.Fprintln(stderr, "learnings: confidence must be a number in [0,1]")
				return 2
			}
			meta = fmt.Sprintf("%s · c=%s", tag, strconv.FormatFloat(c, 'f', -1, 64))
		}
		if !isFile(ledger) {
			_ = os.MkdirAll(root, 0o755)
			_ = os.WriteFile(ledger, []byte(learningsHeader), 0o644)
		}
		if slug == "" {
			slug = "?"
		}
		entry := fmt.Sprintf("- [%s] (%s · %s) %s\n", time.Now().Format("2006-01-02"), meta, slug, text)
		if f, err := os.OpenFile(ledger, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			_, _ = f.WriteString(entry)
			_ = f.Close()
		}
		fmt.Fprintln(stdout, "learnings: recorded.")
		return 0

	case "list":
		if isFile(ledger) {
			if data, err := os.ReadFile(ledger); err == nil {
				_, _ = stdout.Write(data)
			}
		} else {
			fmt.Fprintf(stdout, "learnings: ledger empty (%s not present).\n", ledger)
		}
		return 0

	case "top":
		lines := TopLearnings(root, InjectMax(), InjectThreshold())
		if len(lines) == 0 {
			fmt.Fprintln(stdout, "learnings: no entries at or above the injection confidence floor.")
			return 0
		}
		for _, l := range lines {
			fmt.Fprintln(stdout, l)
		}
		return 0

	case "mine":
		arch := archiveDir(root, args)
		if !isDir(arch) {
			fmt.Fprintf(stdout, "learnings: no archive at %s — nothing to mine.\n", arch)
			return 0
		}
		fmt.Fprintln(stdout, "learnings: repeated finding/decision phrases across archived features (count >= 2):")
		clusters := clusterPhrases(minedPhrases(arch, 80, true))
		printed := 0
		for _, c := range clusters {
			if c.count < 2 {
				continue
			}
			if printed >= 25 {
				break
			}
			fmt.Fprintf(stdout, "  %2d×  %s\n", c.count, c.phrase)
			printed++
		}
		fmt.Fprintln(stdout, "(review these — a stable recurring class is a candidate for a project rule or a ledger entry.)")
		return 0

	case "nudge":
		arch := archiveDir(root, args)
		if !isDir(arch) {
			return 0
		}
		// A cross-feature nudge needs at least two shipped features, and stays quiet
		// once reviewed until something in the archive changes again.
		if countSubdirs(arch) < 2 {
			return 0
		}
		marker := filepath.Join(root, ".learnings-reviewed")
		if fi, err := os.Stat(marker); err == nil && fi.Mode().IsRegular() && !anyFileNewerThan(arch, fi.ModTime()) {
			return 0
		}
		clusters := clusterPhrases(minedPhrases(arch, 60, false))
		var top phraseCount
		for _, c := range clusters {
			if c.count >= 3 {
				top = c
				break
			}
		}
		if top.count == 0 {
			return 0
		}
		phrase := top.phrase
		if len(phrase) > 48 {
			phrase = phrase[:48]
		}
		fmt.Fprintf(stdout, "learnings: a pattern recurs %sx across shipped features (\"%s…\") — review + maybe promote it to a rule with /rite-learn.\n",
			strconv.Itoa(top.count), phrase)
		return 0

	default:
		fmt.Fprintln(stderr, "usage: devrites-engine learnings add|list|top|mine|nudge [...]")
		return 2
	}
}

// Injection defaults — the threshold + cap that bound how much of the ledger
// reaches the SessionStart orientation, so a growing ledger can never bloat the
// context window.
const (
	defaultInjectThreshold = 0.7
	defaultInjectMax       = 5
)

// confInLine pulls a confidence token (c=0.8 or c0.8, case-insensitive) out of a
// ledger line. Anchored on a word boundary so it does not match inside a word.
var confInLine = regexp.MustCompile(`(?i)\bc=?([01](?:\.[0-9]+)?)\b`)

// TopLearnings returns up to max ledger entries whose confidence is at or above
// threshold, highest-confidence first, each rendered as a single injection-ready
// bullet. An unmarked entry is treated as 0.5 (below the default floor). max <= 0
// disables injection entirely (returns nil).
func TopLearnings(root string, max int, threshold float64) []string {
	if max <= 0 {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(root, "learnings.md"))
	if err != nil {
		return nil
	}
	type scored struct {
		line string
		conf float64
	}
	var kept []scored
	for _, line := range splitLinesNoTrailing(data) {
		if !strings.HasPrefix(strings.TrimSpace(line), "- [") {
			continue // only dated ledger entries, never the header prose
		}
		conf := 0.5
		if m := confInLine.FindStringSubmatch(line); m != nil {
			if c, err := strconv.ParseFloat(m[1], 64); err == nil {
				conf = c
			}
		}
		if conf+1e-9 < threshold {
			continue
		}
		kept = append(kept, scored{line: strings.TrimSpace(line), conf: conf})
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].conf > kept[j].conf })
	if len(kept) > max {
		kept = kept[:max]
	}
	out := make([]string, len(kept))
	for i, k := range kept {
		out[i] = k.line
	}
	return out
}

// parseConfidence parses a 0..1 confidence, rejecting out-of-range or garbage.
func parseConfidence(s string) (float64, bool) {
	c, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || c < 0 || c > 1 {
		return 0, false
	}
	return c, true
}

// InjectThreshold / InjectMax read the injection-bound env knobs, falling back to
// the defaults on empty or unparseable input.
func InjectThreshold() float64 {
	if v := os.Getenv("DEVRITES_LEARNING_CONFIDENCE_THRESHOLD"); v != "" {
		if c, ok := parseConfidence(v); ok {
			return c
		}
	}
	return defaultInjectThreshold
}

func InjectMax() int {
	if v := os.Getenv("DEVRITES_MAX_INJECTED_LEARNINGS"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			return n
		}
	}
	return defaultInjectMax
}

// archiveDir is the archive to mine — an explicit second argument, or archive/
// under root by default.
func archiveDir(root string, args []string) string {
	if arch := argAt(args, 1); arch != "" {
		return arch
	}
	return filepath.Join(root, "archive")
}

// learningsHeader is the ledger preamble written when `add` first creates it.
const learningsHeader = "# DevRites learnings ledger\n\n" +
	"Project-local lessons mined from shipped features — recurring corrections,\n" +
	"dismissed-finding classes, and dead-ends. Loaded by the review skills before a fan-out so\n" +
	"the same false positive or the same mistake does not recur. Untrusted prior: live code\n" +
	"always overrides a ledger entry (see standards/security.md).\n\n"

var (
	// findingLine matches a bullet, or a line mentioning a finding/dead-end/drift/
	// dismissal — the candidate lines worth clustering.
	findingLine = regexp.MustCompile(`(?i)^[-*] |finding|dead end|drift|dismiss`)
	// Normalisers applied to each candidate line so wording clusters despite
	// incidental differences.
	leadingBullet = regexp.MustCompile(`^[-*[:space:]]+`) // the leading bullet/indent
	codeSpan      = regexp.MustCompile("`[^`]*`")         // inline `code`
	digits        = regexp.MustCompile(`[0-9]+`)          // ids and counts
	whitespaceRun = regexp.MustCompile(`[[:space:]]+`)    // runs of whitespace
)

// phraseCount is a normalised phrase and how many lines reduced to it.
type phraseCount struct {
	phrase string
	count  int
}

// minedPhrases reads the decisions/drift/review notes of every archived feature and
// returns the normalised candidate phrases. Normalisation strips the leading
// bullet, inline code, and digits, collapses whitespace, and lowercases; phrases of
// 12 characters or fewer are dropped as too short to be meaningful, and the rest are
// truncated to trunc bytes. When collapse is set, leading/trailing whitespace is
// trimmed and internal runs squeezed to single spaces before the length check.
func minedPhrases(arch string, trunc int, collapse bool) []string {
	var phrases []string
	for _, leaf := range []string{"decisions.md", "drift.md", "review.md"} {
		files, _ := filepath.Glob(filepath.Join(arch, "*", leaf))
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			for _, line := range splitLinesNoTrailing(data) {
				if !findingLine.MatchString(line) {
					continue
				}
				s := leadingBullet.ReplaceAllString(line, "")
				s = codeSpan.ReplaceAllString(s, "")
				s = digits.ReplaceAllString(s, "")
				s = whitespaceRun.ReplaceAllString(s, " ")
				s = asciiLower(s)
				if collapse {
					s = strings.Join(strings.Fields(s), " ")
				}
				if len(s) > 12 {
					if len(s) > trunc {
						s = s[:trunc]
					}
					phrases = append(phrases, s)
				}
			}
		}
	}
	return phrases
}

// clusterPhrases groups identical phrases and orders them by frequency, most
// frequent first. Equal counts keep ascending phrase order.
func clusterPhrases(phrases []string) []phraseCount {
	sort.Strings(phrases)
	var clusters []phraseCount
	for _, p := range phrases {
		if n := len(clusters); n > 0 && clusters[n-1].phrase == p {
			clusters[n-1].count++
		} else {
			clusters = append(clusters, phraseCount{phrase: p, count: 1})
		}
	}
	sort.SliceStable(clusters, func(i, j int) bool { return clusters[i].count > clusters[j].count })
	return clusters
}

// countSubdirs counts arch's immediate non-hidden subdirectories (its features).
func countSubdirs(arch string) int {
	entries, err := os.ReadDir(arch)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			n++
		}
	}
	return n
}

// anyFileNewerThan reports whether any regular file under arch was modified after t.
func anyFileNewerThan(arch string, t time.Time) bool {
	newer := false
	_ = filepath.WalkDir(arch, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() && info.ModTime().After(t) {
			newer = true
			return filepath.SkipAll
		}
		return nil
	})
	return newer
}

// asciiLower lowercases ASCII A–Z, leaving every other byte untouched.
func asciiLower(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		return r
	}, s)
}

package lib

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/parallel"
)

// RecallUsage is the help and usage-error text for `recall`.
const RecallUsage = `usage: devrites-engine recall --paths <p1,p2,...> [--paths <p>]... [--limit n]

Lists archived features (.devrites/archive/<slug>/) whose touched-files.md
candidate manifest covers a queried project-relative path: exact or as a
directory prefix, case-folded. Ranked by overlap count, then newest archive
directory mtime, then slug; --limit caps the features shown (default 10).
Each feature prints up to 8 pointers (then "+N more"): decisions.md DEC-###
entries and ## Carry-forward constraints bullets whose Files: overlap (or a
constraint naming no files), and review.md/seal.md/drift.md lines that mention
a queried path. Fenced examples are ignored. Pointer files are relative to
.devrites. Read-only: computed on every call; nothing is written, cached, or
indexed. An unreadable archive entry prints "unreadable:" and the scan goes on.
Exit 0 ok (including no archive or no match), 1 invalid archive directory, 2 usage.`

const (
	recallDefaultLimit = 10
	recallPointerCap   = 8
	recallClipRunes    = 120
)

// recallPointerFiles are scanned in this order; decisions.md is structural,
// the rest are searched for path mentions.
var recallPointerFiles = []string{"decisions.md", "review.md", "seal.md", "drift.md"}

var recallDecisionHeading = regexp.MustCompile(`^#{1,6}[ \t]+DEC-[0-9]+`)

type recallFeature struct {
	slug    string
	dir     string
	mtime   time.Time
	matched []string // queried paths the manifest covers, in query order
}

// RunRecall answers "what did earlier features decide or find about these
// paths" by reading the archive on every call (ADR-0014/ADR-0022 forbid an
// engine index or ledger). It never writes, not even a metric.
func RunRecall(root string, args []string, stdout, stderr io.Writer) int {
	queries, limit, ok := parseRecallArgs(args, stderr)
	if !ok {
		return 2
	}
	archiveDir, err := devritespaths.ArchiveDirChecked(root)
	if err != nil {
		fmt.Fprintf(stderr, "recall: invalid archive directory: %v\n", err)
		return 1
	}
	entries, err := os.ReadDir(archiveDir)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintln(stdout, "recall: no archive")
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "recall: read archive: %v\n", err)
		return 1
	}
	var features []recallFeature
	var unreadable []string
	for _, entry := range entries {
		name := entry.Name()
		// Hidden scratch dirs and stray files are not archived features.
		if strings.HasPrefix(name, ".") || entry.Type().IsRegular() {
			continue
		}
		label := "archive/" + name
		dir, err := devritespaths.ExistingArchivedFeatureDirChecked(root, name)
		if err != nil {
			unreadable = append(unreadable, fmt.Sprintf("%s (%v)", label, err))
			continue
		}
		raw, err := readBoundedRegularFile(filepath.Join(dir, "touched-files.md"), maxCandidateManifestBytes)
		if errors.Is(err, fs.ErrNotExist) {
			continue // never built: touched nothing
		}
		var rows []candidateRow
		if err == nil {
			rows, err = parseCandidateManifest(raw)
		}
		if err != nil {
			unreadable = append(unreadable, fmt.Sprintf("%s/touched-files.md (%v)", label, err))
			continue
		}
		matched := recallMatched(queries, rows)
		if len(matched) == 0 {
			continue
		}
		feature := recallFeature{slug: name, dir: dir, matched: matched}
		if info, err := entry.Info(); err == nil {
			feature.mtime = info.ModTime()
		}
		features = append(features, feature)
	}
	sort.SliceStable(features, func(i, j int) bool {
		a, b := features[i], features[j]
		if len(a.matched) != len(b.matched) {
			return len(a.matched) > len(b.matched)
		}
		if !a.mtime.Equal(b.mtime) {
			return a.mtime.After(b.mtime)
		}
		return a.slug < b.slug
	})
	if len(features) == 0 {
		fmt.Fprintln(stdout, "recall: no archived feature touched these paths")
	} else {
		fmt.Fprintf(stdout, "recall: %d archived feature(s) touched these paths\n", len(features))
		if len(features) > limit {
			fmt.Fprintf(stdout, "recall: truncated to %d feature(s) (--limit)\n", limit)
			features = features[:limit]
		}
		for _, f := range features {
			fmt.Fprintf(stdout, "feature %s overlap=%d paths=%s\n", f.slug, len(f.matched), strings.Join(f.matched, ","))
			writeRecallPointers(stdout, f.dir, "archive/"+f.slug, queries)
		}
	}
	for _, line := range unreadable {
		fmt.Fprintf(stdout, "unreadable: %s\n", line)
	}
	return 0
}

func parseRecallArgs(args []string, stderr io.Writer) ([]string, int, bool) {
	limit := recallDefaultLimit
	var queries []string
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paths":
			value := argAt(args, i+1)
			i++
			if strings.TrimSpace(value) == "" || strings.HasPrefix(value, "-") {
				fmt.Fprintln(stderr, "recall: --paths requires a comma-separated path list (not a flag)")
				return nil, 0, false
			}
			for _, raw := range strings.Split(value, ",") {
				if strings.TrimSpace(raw) == "" {
					continue
				}
				path, err := parallel.NormalizePath(raw)
				if err == nil && strings.HasPrefix(path, "-") {
					err = fmt.Errorf("path must not look like a flag: %q", raw)
				}
				if err != nil {
					fmt.Fprintf(stderr, "recall: --paths: %v\n", err)
					return nil, 0, false
				}
				if key := strings.ToLower(path); !seen[key] {
					seen[key] = true
					queries = append(queries, path)
				}
			}
		case "--limit":
			n, err := strconv.Atoi(argAt(args, i+1))
			if err != nil || n < 1 {
				fmt.Fprintf(stderr, "recall: --limit expects a positive integer, got %q\n", argAt(args, i+1))
				return nil, 0, false
			}
			limit = n
			i++
		default:
			fmt.Fprintf(stderr, "recall: unexpected argument %q\n%s\n", args[i], RecallUsage)
			return nil, 0, false
		}
	}
	if len(queries) == 0 {
		fmt.Fprintln(stderr, RecallUsage)
		return nil, 0, false
	}
	return queries, limit, true
}

// recallMatched returns the queries covered by a manifest row, folding case
// and matching directory prefixes with the claim-overlap rule.
func recallMatched(queries []string, rows []candidateRow) []string {
	var matched []string
	for _, q := range queries {
		for _, row := range rows {
			if claimPathsOverlap(row.path, q) {
				matched = append(matched, q)
				break
			}
		}
	}
	return matched
}

func writeRecallPointers(w io.Writer, dir, label string, queries []string) {
	var pointers, unreadable []string
	for _, name := range recallPointerFiles {
		raw, err := readBoundedRegularFile(filepath.Join(dir, name), maxCandidateArtifactBytes)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err == nil {
			raw, err = markdowntext.Structural(raw)
		}
		if err != nil {
			unreadable = append(unreadable, fmt.Sprintf("%s/%s (%v)", label, name, err))
			continue
		}
		lines := strings.Split(string(raw), "\n")
		var hits []int
		if name == "decisions.md" {
			hits = recallDecisionHits(lines, queries)
		} else {
			for i, line := range lines {
				if recallMentions(line, queries) {
					hits = append(hits, i)
				}
			}
		}
		for _, i := range hits {
			pointers = append(pointers, fmt.Sprintf("%s/%s:%d  %s", label, name, i+1, clipRunes(strings.TrimSpace(lines[i]), recallClipRunes)))
		}
	}
	for i, p := range pointers {
		if i == recallPointerCap {
			fmt.Fprintf(w, "  +%d more\n", len(pointers)-recallPointerCap)
			break
		}
		fmt.Fprintf(w, "  %s\n", p)
	}
	for _, u := range unreadable {
		fmt.Fprintf(w, "  unreadable: %s\n", u)
	}
}

// recallDecisionHits returns the line indexes of DEC-### headings whose
// Files: field overlaps the queries, and of carry-forward constraint bullets
// that overlap or name no files. Fenced lines arrive already blanked.
func recallDecisionHits(lines []string, queries []string) []int {
	var hits []int
	dec, decDone, carry := -1, false, false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		rest := strings.TrimLeft(trimmed, "#")
		if level := len(trimmed) - len(rest); level > 0 && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
			dec, decDone = -1, false
			if recallDecisionHeading.MatchString(trimmed) {
				dec = i
			}
			// As in handoff's dead ends, a ### subheading stays in the section.
			if level <= 2 {
				carry = strings.EqualFold(trimmed, "## Carry-forward constraints")
			}
			continue
		}
		if dec >= 0 {
			if files, ok := recallFilesField(trimmed); ok && !decDone && recallOverlaps(files, queries) {
				hits = append(hits, dec)
				decDone = true
			}
			continue
		}
		entry, ok := strings.CutPrefix(trimmed, "- ")
		if !carry || !ok || strings.TrimSpace(entry) == "" {
			continue
		}
		var files []string
		if at := strings.LastIndex(entry, "Files:"); at >= 0 {
			files = recallPathList(entry[at+len("Files:"):])
		}
		if len(files) == 0 || recallOverlaps(files, queries) {
			hits = append(hits, i)
		}
	}
	return hits
}

// recallFilesField parses a `Files:` field line such as "- Files: a, b" or
// "**Files:** `a` `b`".
func recallFilesField(trimmed string) ([]string, bool) {
	s := strings.TrimLeft(trimmed, "-*+ \t")
	if !strings.HasPrefix(strings.ToLower(s), "files:") {
		return nil, false
	}
	return recallPathList(strings.TrimLeft(s[len("files:"):], "* \t")), true
}

func recallPathList(s string) []string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "none", "n/a", "-", "—":
		return nil
	}
	var out []string
	for _, field := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
		field = strings.TrimSuffix(strings.Trim(field, "`'\"()[]"), ".")
		if path, err := parallel.NormalizePath(field); err == nil {
			out = append(out, path)
		}
	}
	return out
}

func recallOverlaps(files, queries []string) bool {
	for _, f := range files {
		for _, q := range queries {
			if claimPathsOverlap(f, q) {
				return true
			}
		}
	}
	return false
}

// recallMentions reports a case-folded occurrence of a queried path that is
// not part of a longer path: "src/a.go" matches "`src/a.go:12`" but not
// "mysrc/a.go", "lib/src/a.go", or "src/a.gox".
func recallMentions(line string, queries []string) bool {
	lower := strings.ToLower(line)
	for _, q := range queries {
		q = strings.ToLower(q)
		for from := 0; from < len(lower); {
			at := strings.Index(lower[from:], q)
			if at < 0 {
				break
			}
			start, end := from+at, from+at+len(q)
			before := start == 0 || !recallPathByte(lower[start-1])
			after := end == len(lower) || lower[end] == '/' || !recallPathByte(lower[end]) ||
				lower[end] == '.' && (end+1 == len(lower) || !recallPathByte(lower[end+1]))
			if before && after {
				return true
			}
			from = start + 1
		}
	}
	return false
}

func recallPathByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		c == '_' || c == '-' || c == '.' || c == '/' || c >= 0x80
}

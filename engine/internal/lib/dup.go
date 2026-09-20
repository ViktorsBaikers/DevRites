package lib

// dup.go — deterministic near-duplicate detection.
//
// `devrites-engine check dup` finds structurally identical code regions —
// copies that survive renaming and reformatting — with no model and no
// network. Reviewers catch verbatim paste; this catches the same block
// re-typed with different identifiers, which is the class that review
// memory misses in large trees.
//
// Method: strip comments and strings, tokenize each line (keywords and
// punctuation kept, identifiers/numbers/string literals erased), then
// match k-line shingles across files and chain co-linear anchor hits into
// segment pairs. Pairs merge into transitive clusters ranked by similarity
// boosted by distance — a twin two modules away matters more than a twin
// in the same file. Each cluster gets a stable content hash so a dismissed
// cluster stays dismissed in .devrites/dup-ignore until the code changes.
//
//	devrites-engine check dup [slug] [--all|--worktree|--staged|--base <ref>]
//	    [--min-lines n] [--threshold f] [--ignore-file <path>] [--limit n]
//	    [--exclude <csv>] [--cwd <dir>]
//
// Successful scans exit 0; usage or incomplete scans exit 2. Clusters are
// leads for reviewer adjudication, not a gate. A reported cluster is an unverified claim until a reader
// confirms the regions are genuinely interchangeable.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
)

const dupUsage = `usage: devrites-engine check dup [slug] [--all|--worktree|--staged|--base <ref>] [--min-lines n] [--threshold f] [--ignore-file <path>] [--limit n] [--exclude <csv>] [--phase p] [--cwd <dir>]

Finds near-duplicate code regions that survive renaming: comment/string-stripped
token shingles are matched across files, chained into segment pairs, clustered,
and ranked by similarity boosted by distance. In diff modes only clusters
touching changed lines print, with changed units marked '*'.
Ignore file: one cluster hash per line, '# comment' allowed; default
.devrites/dup-ignore. Advisory: exit 0 on success, 2 on usage or scan errors; clusters are leads for review.`

const (
	dupMaxFiles        = 8192
	dupMaxBytes        = 64 << 20
	dupMaxFileBytes    = 1 << 20
	dupShingleLines    = 3
	dupMaxPostings     = 32
	dupMaxPairAnchors  = 8192
	dupMergeGap        = 4
	dupMaxClusters     = 200
	dupHashLength      = 12
	dupCrossDirMaxHops = 8
	dupCrossDirBoost   = 0.15
	dupSameFileMaxStep = 8
	dupSameFileBoost   = 0.10
	dupSameFileStepLn  = 250
)

// dupCommentStyle names the comment syntax used to strip noise before
// tokenizing. Languages sharing a style share one table entry.
type dupCommentStyle struct {
	line   []string
	blocks [][2]string
}

var (
	dupCStyle    = dupCommentStyle{line: []string{"//"}, blocks: [][2]string{{"/*", "*/"}}}
	dupHashStyle = dupCommentStyle{line: []string{"#"}, blocks: [][2]string{{"<#", "#>"}}}
	dupDashStyle = dupCommentStyle{line: []string{"--"}, blocks: [][2]string{{"{-", "-}"}}}
	dupSemiStyle = dupCommentStyle{line: []string{";"}}
	dupPctStyle  = dupCommentStyle{line: []string{"%"}}
	dupXMLStyle  = dupCommentStyle{blocks: [][2]string{{"<!--", "-->"}}}
	dupCSSStyle  = dupCommentStyle{blocks: [][2]string{{"/*", "*/"}}}
	dupMixedWeb  = dupCommentStyle{line: []string{"//"}, blocks: [][2]string{{"/*", "*/"}, {"<!--", "-->"}}}
	dupShellCfg  = dupCommentStyle{line: []string{"#", "//"}, blocks: [][2]string{{"/*", "*/"}}}
)

// dupStyleByExt maps a lowercase file extension (with dot) to its comment
// style. Extensions absent here are not scanned: prose, data, and lockfile
// formats produce shingle noise, not reviewable duplication.
var dupStyleByExt = map[string]dupCommentStyle{
	".go": dupCStyle, ".c": dupCStyle, ".h": dupCStyle,
	".cpp": dupCStyle, ".cc": dupCStyle, ".cxx": dupCStyle,
	".hpp": dupCStyle, ".hh": dupCStyle, ".m": dupCStyle, ".mm": dupCStyle,
	".java": dupCStyle, ".kt": dupCStyle, ".kts": dupCStyle,
	".scala": dupCStyle, ".groovy": dupCStyle, ".gradle": dupCStyle,
	".js": dupCStyle, ".jsx": dupCStyle, ".mjs": dupCStyle, ".cjs": dupCStyle,
	".ts": dupCStyle, ".tsx": dupCStyle, ".cts": dupCStyle, ".mts": dupCStyle,
	".rs": dupCStyle, ".swift": dupCStyle, ".cs": dupCStyle,
	".php": dupCStyle, ".phtml": dupCStyle, ".dart": dupCStyle,
	".zig": dupCStyle, ".proto": dupCStyle, ".prisma": dupCStyle,
	".gd": dupCStyle, ".nim": dupCStyle, ".vala": dupCStyle,
	".py": dupHashStyle, ".pyi": dupHashStyle, ".rb": dupHashStyle,
	".sh": dupHashStyle, ".bash": dupHashStyle, ".zsh": dupHashStyle,
	".fish": dupHashStyle, ".pl": dupHashStyle, ".pm": dupHashStyle,
	".r": dupHashStyle, ".jl": dupHashStyle, ".ex": dupHashStyle,
	".exs": dupHashStyle, ".yaml": dupHashStyle, ".yml": dupHashStyle,
	".toml": dupHashStyle, ".tf": dupShellCfg, ".hcl": dupShellCfg,
	".graphql": dupHashStyle, ".gql": dupHashStyle, ".ps1": dupHashStyle,
	".psm1": dupHashStyle, ".coffee": dupHashStyle, ".cr": dupHashStyle,
	".sql": dupDashStyle, ".lua": dupDashStyle, ".hs": dupDashStyle,
	".lhs": dupDashStyle, ".elm": dupDashStyle, ".ada": dupDashStyle,
	".adb": dupDashStyle, ".ads": dupDashStyle,
	".clj": dupSemiStyle, ".cljs": dupSemiStyle, ".lisp": dupSemiStyle,
	".scm": dupSemiStyle, ".rkt": dupSemiStyle,
	".erl": dupPctStyle, ".hrl": dupPctStyle, ".tex": dupPctStyle,
	".html": dupXMLStyle, ".htm": dupXMLStyle, ".xml": dupXMLStyle,
	".xhtml": dupXMLStyle, ".csproj": dupXMLStyle, ".fsproj": dupXMLStyle,
	".css":  dupCSSStyle,
	".scss": dupCStyle, ".less": dupCStyle, ".sass": dupHashStyle,
	".vue": dupMixedWeb, ".svelte": dupMixedWeb,
}

// dupStyleByName covers extensionless code files worth scanning.
var dupStyleByName = map[string]dupCommentStyle{
	"dockerfile": dupHashStyle, "containerfile": dupHashStyle,
	"makefile": dupHashStyle, "gnumakefile": dupHashStyle,
	"rakefile": dupHashStyle, "gemfile": dupHashStyle,
	"vagrantfile": dupHashStyle, "brewfile": dupHashStyle,
	"cmakelists.txt": dupHashStyle, "justfile": dupHashStyle,
	"jenkinsfile": dupCStyle, "podfile": dupHashStyle,
}

// dupSkipPrefixes are vendored or fixture trees where duplication is
// deliberate or out of the author's control.
var dupSkipPrefixes = []string{
	"vendor/", "node_modules/", "third_party/", "external/", "extern/",
	"testdata/", ".devrites/", ".git/",
}

// dupKeywords is one union keyword set across the scanned language families.
// Keeping control/declaration words verbatim preserves the skeleton
// difference between `if` and `while`; every other word collapses to `id`,
// which is what lets renamed copies match.
var dupKeywords = map[string]bool{
	"if": true, "else": true, "elif": true, "for": true, "while": true,
	"do": true, "switch": true, "case": true, "default": true,
	"break": true, "continue": true, "return": true, "goto": true,
	"func": true, "function": true, "fn": true, "def": true,
	"lambda": true, "class": true, "struct": true, "interface": true,
	"impl": true, "enum": true, "trait": true, "type": true,
	"package": true, "import": true, "from": true, "use": true,
	"mod": true, "pub": true, "let": true, "const": true, "var": true,
	"val": true, "mut": true, "ref": true, "static": true,
	"try": true, "catch": true, "except": true, "finally": true,
	"throw": true, "throws": true, "raise": true, "rescue": true,
	"ensure": true, "async": true, "await": true, "yield": true,
	"go": true, "defer": true, "select": true, "chan": true,
	"public": true, "private": true, "protected": true, "final": true,
	"abstract": true, "override": true, "virtual": true, "internal": true,
	"sealed": true, "new": true, "delete": true, "this": true,
	"self": true, "super": true, "extends": true, "implements": true,
	"nil": true, "null": true, "none": true, "undefined": true,
	"true": true, "false": true, "and": true, "or": true, "not": true,
	"in": true, "is": true, "of": true, "as": true, "with": true,
	"where": true, "when": true, "match": true, "then": true,
	"begin": true, "end": true, "elsif": true, "until": true,
	"unless": true, "repeat": true, "void": true, "int": true,
	"long": true, "short": true, "float": true, "double": true,
	"bool": true, "char": true, "string": true, "byte": true,
	"rune": true, "uint": true, "export": true, "declare": true,
	"namespace": true, "module": true, "fi": true, "esac": true,
	"done": true, "local": true, "readonly": true, "eval": true,
	"echo": true, "sizeof": true, "typeof": true, "instanceof": true,
	"macro": true, "unsafe": true, "extern": true, "inline": true,
	"typedef": true, "union": true, "volatile": true, "const_cast": true,
}

type dupFile struct {
	path  string
	lines []dupLine
}

type dupLine struct {
	tokens  string // canonical token stream; "" when the line carried nothing
	trivial bool   // single punctuation/keyword line that still pads spans
}

type dupPos struct {
	file int
	line int // index into dupFile.lines (0-based)
}

type dupAnchor struct {
	aLine int
	bLine int
}

type dupPairKey struct {
	a int
	b int
}

type dupSegment struct {
	aFile, bFile   int
	aStart, aEnd   int // 0-based inclusive line indexes into dupFile.lines
	bStart, bEnd   int
	anchors        int
	score, boosted float64
}

type dupUnit struct {
	file        int
	start, end  int // 0-based inclusive
	contentHash string
	changed     bool
}

type dupEdge struct {
	a, b    int // unit indexes
	boosted float64
}

type dupCluster struct {
	units   []int
	score   float64 // max boosted edge inside the cluster
	hash    string
	changed bool
}

// normalizeDupSource strips comments and strings then canonicalizes each
// line into a token stream. The result keeps line indexes aligned with the
// source file so reported ranges stay true line numbers.
func normalizeDupSource(raw string, style dupCommentStyle) []dupLine {
	srcLines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]dupLine, len(srcLines))
	var blockEnd string
	for i, src := range srcLines {
		var b strings.Builder
		for j := 0; j < len(src); {
			if blockEnd != "" {
				if k := strings.Index(src[j:], blockEnd); k >= 0 {
					j += k + len(blockEnd)
					blockEnd = ""
				} else {
					break
				}
				continue
			}
			matched := false
			for _, marker := range style.line {
				if strings.HasPrefix(src[j:], marker) {
					j = len(src)
					matched = true
					break
				}
			}
			if matched || j >= len(src) {
				break
			}
			for _, pair := range style.blocks {
				if strings.HasPrefix(src[j:], pair[0]) {
					blockEnd = pair[1]
					j += len(pair[0])
					matched = true
					break
				}
			}
			if matched {
				continue
			}
			c := src[j]
			if c == '\'' || c == '"' || c == '`' {
				b.WriteString(" \x01")
				j++
				for j < len(src) && src[j] != c {
					if src[j] == '\\' && c != '`' {
						j++
					}
					j++
				}
				if j < len(src) {
					j++
				}
				continue
			}
			b.WriteByte(c)
			j++
		}
		out[i] = canonicalizeDupLine(b.String())
	}
	return out
}

// canonicalizeDupLine reduces one comment-free line to a space-joined token
// string: keywords and punctuation verbatim, identifiers → id, numbers → 0,
// strings already → s. Fewer than two tokens is a trivial line; it still
// participates as the wildcard `_` so a copied block stays contiguous.
func canonicalizeDupLine(text string) dupLine {
	var tokens []string
	for i := 0; i < len(text); {
		c := text[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '\x01':
			tokens = append(tokens, "s")
			i++
		case isDupIdentStart(c):
			j := i + 1
			for j < len(text) && isDupIdentPart(text[j]) {
				j++
			}
			word := strings.ToLower(text[i:j])
			if dupKeywords[word] {
				tokens = append(tokens, word)
			} else {
				tokens = append(tokens, "id")
			}
			i = j
		case c >= '0' && c <= '9':
			j := i + 1
			for j < len(text) && (isDupIdentPart(text[j]) || text[j] == '.') {
				j++
			}
			tokens = append(tokens, "0")
			i = j
		default:
			tokens = append(tokens, string(c))
			i++
		}
	}
	if len(tokens) < 2 {
		return dupLine{tokens: "_", trivial: true}
	}
	return dupLine{tokens: strings.Join(tokens, " ")}
}

func isDupIdentStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func isDupIdentPart(c byte) bool {
	return isDupIdentStart(c) || c >= '0' && c <= '9'
}

// dupShingle is the fnv64a fingerprint of dupShingleLines consecutive
// canonical lines.
func dupShingleHash(lines []dupLine, start int) uint64 {
	h := fnv.New64a()
	for i := start; i < start+dupShingleLines; i++ {
		_, _ = h.Write([]byte(lines[i].tokens))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// dupAnchorSegments chains shared-shingle hits between one file pair into
// maximal similar segments. Hits merge when both sides advance by at most
// dupMergeGap lines, so copies light edits stay one segment.
func dupAnchorSegments(key dupPairKey, anchors []dupAnchor, files []dupFile, minLines int, threshold float64) []dupSegment {
	if len(anchors) == 0 {
		return nil
	}
	sort.Slice(anchors, func(i, j int) bool {
		if anchors[i].aLine != anchors[j].aLine {
			return anchors[i].aLine < anchors[j].aLine
		}
		return anchors[i].bLine < anchors[j].bLine
	})
	var segs []dupSegment
	start := 0
	for i := 1; i <= len(anchors); i++ {
		split := i == len(anchors) ||
			anchors[i].aLine-anchors[i-1].aLine > dupMergeGap ||
			abs(anchors[i].bLine-anchors[i-1].bLine) > dupMergeGap
		if !split {
			continue
		}
		run := anchors[start:i]
		start = i
		seg := dupSegment{
			aFile: key.a, bFile: key.b,
			aStart: run[0].aLine, aEnd: run[len(run)-1].aLine + dupShingleLines - 1,
			bStart: run[0].bLine, bEnd: run[len(run)-1].bLine + dupShingleLines - 1,
			anchors: len(run),
		}
		spanA := seg.aEnd - seg.aStart + 1
		spanB := seg.bEnd - seg.bStart + 1
		if spanA < minLines || spanB < minLines {
			continue
		}
		if key.a == key.b && seg.aEnd >= seg.bStart {
			continue // overlapping same-file windows are one region, not a pair
		}
		possible := max(spanA, spanB) - dupShingleLines + 1
		seg.score = min(float64(seg.anchors)/float64(possible), 1.0)
		if seg.score < threshold {
			continue
		}
		seg.boosted = seg.score * (1 + dupDistanceBoost(files[key.a].path, files[key.b].path, seg))
		segs = append(segs, seg)
	}
	return segs
}

// dupDistanceBoost lifts pairs that sit far apart: a twin across modules
// hides from review where a twin one screen down does not. Same-file
// distance counts in 250-line steps; cross-file distance counts directory
// hops from the deepest common ancestor. Both boosts are log-scaled caps
// ported from the embedding-detector playbook.
func dupDistanceBoost(pathA, pathB string, seg dupSegment) float64 {
	if pathA == pathB {
		gap := seg.bStart - seg.aEnd
		if gap < 0 {
			gap = 0
		}
		steps := min(gap/dupSameFileStepLn, dupSameFileMaxStep)
		if steps <= 0 {
			return 0
		}
		return math.Log2(1+float64(steps)) / math.Log2(1+dupSameFileMaxStep) * dupSameFileBoost
	}
	da := strings.Split(path.Dir(pathA), "/")
	db := strings.Split(path.Dir(pathB), "/")
	common := 0
	for common < len(da) && common < len(db) && da[common] == db[common] {
		common++
	}
	hops := (len(da) - common) + (len(db) - common)
	if hops <= 0 {
		return 0
	}
	hops = min(hops, dupCrossDirMaxHops)
	return math.Log2(1+float64(hops)) / math.Log2(1+dupCrossDirMaxHops) * dupCrossDirBoost
}

// dupClusterUnits merges per-file ranges into units, unions segments into
// transitive clusters, and orders clusters by their strongest boosted pair.
func dupClusterUnits(segs []dupSegment, files []dupFile) ([]dupCluster, []dupUnit, []dupEdge) {
	// Merge each file's overlapping/adjacent ranges into units.
	perFile := map[int][]dupUnit{}
	for _, s := range segs {
		perFile[s.aFile] = append(perFile[s.aFile], dupUnit{file: s.aFile, start: s.aStart, end: s.aEnd})
		perFile[s.bFile] = append(perFile[s.bFile], dupUnit{file: s.bFile, start: s.bStart, end: s.bEnd})
	}
	var units []dupUnit
	for f, list := range perFile {
		sort.Slice(list, func(i, j int) bool { return list[i].start < list[j].start })
		var merged []dupUnit
		for _, u := range list {
			if n := len(merged); n > 0 && u.start <= merged[n-1].end+2 {
				if u.end > merged[n-1].end {
					merged[n-1].end = u.end
				}
				continue
			}
			merged = append(merged, u)
		}
		for _, u := range merged {
			u.contentHash = dupUnitHash(files[f].lines[u.start : u.end+1])
			units = append(units, u)
		}
	}
	// Locate the unit covering a segment endpoint: same file, maximal overlap.
	unitIndex := func(file, start, end int) int {
		best, bestOverlap := -1, 0
		for i, u := range units {
			if u.file != file {
				continue
			}
			ov := min(u.end, end) - max(u.start, start) + 1
			if ov > bestOverlap {
				best, bestOverlap = i, ov
			}
		}
		return best
	}
	var edges []dupEdge
	parent := make([]int, len(units))
	for i := range parent {
		parent[i] = i
	}
	find := func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}
		return x
	}
	for _, s := range segs {
		ua := unitIndex(s.aFile, s.aStart, s.aEnd)
		ub := unitIndex(s.bFile, s.bStart, s.bEnd)
		if ua < 0 || ub < 0 || ua == ub {
			continue
		}
		edges = append(edges, dupEdge{a: ua, b: ub, boosted: s.boosted})
		ra, rb := find(ua), find(ub)
		if ra != rb {
			parent[rb] = ra
		}
	}
	groups := map[int][]int{}
	for _, e := range edges {
		r := find(e.a)
		if !containsInt(groups[r], e.a) {
			groups[r] = append(groups[r], e.a)
		}
		if !containsInt(groups[r], e.b) {
			groups[r] = append(groups[r], e.b)
		}
	}
	var clusters []dupCluster
	for _, members := range groups {
		sort.Ints(members)
		c := dupCluster{units: members}
		var sb strings.Builder
		var hashes []string
		for _, u := range members {
			hashes = append(hashes, files[units[u].file].path+"\x00"+units[u].contentHash)
		}
		sort.Strings(hashes)
		for _, h := range hashes {
			sb.WriteString(h)
			sb.WriteByte('\n')
		}
		sum := sha256.Sum256([]byte(sb.String()))
		c.hash = hex.EncodeToString(sum[:])[:dupHashLength]
		clusters = append(clusters, c)
	}
	for i := range clusters {
		best := 0.0
		member := map[int]bool{}
		for _, u := range clusters[i].units {
			member[u] = true
		}
		for _, e := range edges {
			if member[e.a] && member[e.b] && e.boosted > best {
				best = e.boosted
			}
		}
		clusters[i].score = best
	}
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].score != clusters[j].score {
			return clusters[i].score > clusters[j].score
		}
		return clusters[i].hash < clusters[j].hash
	})
	return clusters, units, edges
}

func dupUnitHash(lines []dupLine) string {
	// Trivial `_` lines (lone braces, blank padding) carry no identity: a block
	// that moves past padding keeps its hash so the ignore ledger holds.
	var sb strings.Builder
	for _, l := range lines {
		if l.trivial || dupHeaderLine(l.tokens) {
			continue
		}
		sb.WriteString(l.tokens)
		sb.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(sum[:])[:16]
}

// dupHeaderLine reports whether a normalized line is a file-scope declaration
// (package/module/namespace). Header lines are file context, not copied code:
// including them would make unit hashes flip when a header enters or leaves a
// unit's matched range.
func dupHeaderLine(tokens string) bool {
	fields := strings.Fields(tokens)
	if len(fields) == 0 || len(fields) > 2 {
		return false
	}
	switch fields[0] {
	case "package", "module", "namespace":
		return true
	}
	return false
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// dupChangedRanges parses `git diff -U0` output into per-file added-line
// ranges on the new side. Deleted files contribute nothing.
func dupChangedRanges(diff string) map[string][][2]int {
	ranges := map[string][][2]int{}
	var cur string
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			p := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			if strings.HasPrefix(p, `"`) {
				decoded, err := strconv.Unquote(p)
				if err != nil {
					cur = ""
					continue
				}
				p = decoded
			}
			if p == "/dev/null" {
				cur = ""
				continue
			}
			cur = strings.TrimPrefix(p, "b/")
		case strings.HasPrefix(line, "@@") && cur != "":
			i := strings.Index(line, "+")
			if i < 0 {
				continue
			}
			rest := line[i+1:]
			end := strings.IndexAny(rest, " @")
			if end < 0 {
				end = len(rest)
			}
			spec := rest[:end] // "start,count" — the comma stays inside spec
			parts := strings.SplitN(spec, ",", 2)
			startN, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}
			count := 1
			if len(parts) == 2 {
				if n, err := strconv.Atoi(parts[1]); err == nil {
					count = n
				}
			}
			if count == 0 {
				ranges[cur] = append(ranges[cur], [2]int{startN, startN})
			} else {
				ranges[cur] = append(ranges[cur], [2]int{startN, startN + count - 1})
			}
		}
	}
	return ranges
}

func dupLoadIgnore(path string) map[string]bool {
	ignored := map[string]bool{}
	// #nosec G304 -- path is the documented --ignore-file/default ledger under the project root
	data, err := os.ReadFile(path)
	if err != nil {
		return ignored
	}
	for _, line := range strings.Split(string(data), "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		if h := strings.TrimSpace(line); h != "" {
			ignored[h] = true
		}
	}
	return ignored
}

type dupOptions struct {
	slug       string
	phase      string
	mode       string // all | worktree | staged | base
	base       string
	minLines   int
	threshold  float64
	ignoreFile string
	limit      int
	exclude    []string
	cwd        string
	root       string
}

func parseDupArgs(root string, args []string, stderr io.Writer) (dupOptions, int) {
	opts := dupOptions{mode: "worktree", phase: "review", minLines: 8, threshold: 0.6, limit: 50, root: root}
	var positional []string
	seen := map[string]bool{}
	modeSeen := false
	for i := 0; i < len(args); i++ {
		flag := args[i]
		if strings.HasPrefix(flag, "--") {
			if seen[flag] {
				fmt.Fprintf(stderr, "dup: repeated option %s\n", flag)
				return opts, 2
			}
			seen[flag] = true
			switch flag {
			case "--all", "--worktree", "--staged", "--base":
				if modeSeen {
					fmt.Fprintln(stderr, "dup: conflicting scan modes")
					return opts, 2
				}
				modeSeen = true
			}
			switch flag {
			case "--base", "--phase", "--min-lines", "--threshold", "--ignore-file", "--limit", "--exclude", "--cwd":
				value := argAt(args, i+1)
				if strings.TrimSpace(value) == "" || strings.HasPrefix(value, "-") {
					fmt.Fprintf(stderr, "dup: %s requires a value (not a flag)\n", flag)
					return opts, 2
				}
			}
		}

		switch args[i] {
		case "--phase":
			opts.phase = argAt(args, i+1)
			i++
		case "--all":
			opts.mode = "all"
		case "--worktree":
			opts.mode = "worktree"
		case "--staged":
			opts.mode = "staged"
		case "--base":
			opts.mode = "base"
			opts.base = argAt(args, i+1)
			i++
		case "--min-lines":
			n, err := strconv.Atoi(argAt(args, i+1))
			if err != nil || n < dupShingleLines {
				fmt.Fprintf(stderr, "dup: --min-lines must be an integer >= %d\n", dupShingleLines)
				return opts, 2
			}
			opts.minLines = n
			i++
		case "--threshold":
			f, err := strconv.ParseFloat(argAt(args, i+1), 64)
			if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f <= 0 || f > 1 {
				fmt.Fprintln(stderr, "dup: --threshold must be in (0, 1]")
				return opts, 2
			}
			opts.threshold = f
			i++
		case "--ignore-file":
			opts.ignoreFile = argAt(args, i+1)
			i++
		case "--limit":
			n, err := strconv.Atoi(argAt(args, i+1))
			if err != nil || n < 1 {
				fmt.Fprintln(stderr, "dup: --limit must be a positive integer")
				return opts, 2
			}
			opts.limit = n
			i++
		case "--exclude":
			for _, entry := range strings.Split(argAt(args, i+1), ",") {
				if entry = strings.TrimSpace(entry); entry != "" {
					opts.exclude = append(opts.exclude, strings.TrimSuffix(entry, "/")+"/")
				}
			}
			i++
		case "--cwd":
			opts.cwd = argAt(args, i+1)
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "dup: unknown flag %q\n", args[i])
				return opts, 2
			}
			positional = append(positional, args[i])
		}
	}
	if len(positional) > 1 || opts.mode == "base" && opts.base == "" {
		fmt.Fprint(stderr, dupUsage)
		return opts, 2
	}
	if len(positional) == 1 {
		opts.slug = positional[0]
	}
	return opts, 0
}

// RunCheckDup scans tracked source for near-duplicate regions and prints
// ranked clusters. Advisory only: it never gates — a cluster is a lead that
// a reviewer confirms or dismisses into the ignore file.
func RunCheckDup(root string, args []string, stdout, stderr io.Writer) int {
	opts, code := parseDupArgs(root, args, stderr)
	if code != 0 {
		return code
	}
	dir := opts.cwd
	if dir == "" {
		dir = projectDir(root)
	}
	repoOut, err := runGitCommand(dir, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		fmt.Fprintf(stderr, "dup: %v\n", err)
		return 2
	}
	repo := strings.TrimSpace(string(repoOut))
	listed, err := dupGitOutput(repo, nil, gitOutputLimit, "ls-files", "-z")
	if err != nil {
		fmt.Fprintf(stderr, "dup: %v\n", err)
		return 2
	}
	var paths []string
	for _, p := range strings.Split(strings.TrimRight(string(listed), "\x00"), "\x00") {
		if p != "" && dupScannable(p, opts.exclude) {
			paths = append(paths, p)
		}
	}
	var untracked []string

	if opts.mode != "staged" {
		out, uerr := dupGitOutput(repo, nil, gitOutputLimit, "ls-files", "--others", "--exclude-standard", "-z")
		if uerr != nil {
			fmt.Fprintf(stderr, "dup: %v\n", uerr)
			return 2
		}
		for _, p := range strings.Split(string(out), "\x00") {
			if p != "" && dupScannable(p, opts.exclude) {
				untracked = append(untracked, p)
				paths = append(paths, p)
			}
		}
	}

	if len(paths) > dupMaxFiles {
		fmt.Fprintf(stderr, "dup: too many files (%d > %d)\n", len(paths), dupMaxFiles)
		return 2
	}

	var files []dupFile
	var skipped, failed int
	if opts.mode == "staged" {
		files, skipped, failed, err = dupReadIndex(repo, opts.exclude)
		if err != nil {
			fmt.Fprintf(stderr, "dup: incomplete scan: %v\n", err)
			return 2
		}
	} else {
		files, skipped, failed = dupReadFiles(repo, paths)
	}
	if failed > 0 {
		fmt.Fprintf(stderr, "dup: incomplete scan: %d eligible file(s) unavailable or over resource limits; %d scanned\n", failed, len(files))
	}
	if len(files) == 0 {
		if failed > 0 {
			return 2
		}
		if skipped > 0 {
			fmt.Fprintf(stdout, "dup: no source scanned (%d generated files excluded)\n", skipped)
		} else {
			fmt.Fprintln(stdout, "dup: no eligible source files")
		}
		return 0
	}

	clusters, units := dupDetect(files, opts)

	// Changed-range marking and filtering.
	allChanged := false
	changedRanges := map[string][][2]int{}
	switch opts.mode {
	case "all":
		// no changed marking
	case "staged":
		diff, derr := dupGitOutput(repo, nil, gitOutputLimit, "--no-replace-objects", "diff", "--no-ext-diff", "--no-textconv", "--no-color", "--src-prefix=a/", "--dst-prefix=b/", "--cached", "-U0", "--")
		if derr != nil {
			fmt.Fprintf(stderr, "dup: %v\n", derr)
			return 2
		}
		changedRanges = dupChangedRanges(string(diff))
	case "base":
		diff, derr := dupGitOutput(repo, nil, gitOutputLimit, "--no-replace-objects", "diff", "--no-ext-diff", "--no-textconv", "--no-color", "--src-prefix=a/", "--dst-prefix=b/", opts.base, "-U0", "--")
		if derr != nil {
			fmt.Fprintf(stderr, "dup: %v\n", derr)
			return 2
		}
		changedRanges = dupChangedRanges(string(diff))
	default: // worktree
		if isUnbornHEAD(repo) {
			allChanged = true
		} else {
			diff, derr := dupGitOutput(repo, nil, gitOutputLimit, "--no-replace-objects", "diff", "--no-ext-diff", "--no-textconv", "--no-color", "--src-prefix=a/", "--dst-prefix=b/", "HEAD", "-U0", "--")
			if derr != nil {
				fmt.Fprintf(stderr, "dup: %v\n", derr)
				return 2
			}
			changedRanges = dupChangedRanges(string(diff))
		}
		for _, p := range untracked {
			changedRanges[p] = [][2]int{{1, 1 << 30}}
		}
	}
	if allChanged {
		for i := range units {
			units[i].changed = true
		}
	} else {
		for i := range units {
			for _, r := range changedRanges[files[units[i].file].path] {
				// unit lines are 0-based inclusive; ranges are 1-based.
				if units[i].start+1 <= r[1] && r[0] <= units[i].end+1 {
					units[i].changed = true
					break
				}
			}
		}
	}
	for i := range clusters {
		for _, u := range clusters[i].units {
			if units[u].changed {
				clusters[i].changed = true
				break
			}
		}
	}
	totalClusters := len(clusters)
	if opts.mode != "all" {
		var kept []dupCluster
		for _, c := range clusters {
			if c.changed {
				kept = append(kept, c)
			}
		}
		clusters = kept
	}

	ignoreFile := opts.ignoreFile
	if ignoreFile == "" {
		ignoreFile = filepath.Join(projectDir(root), devritespaths.DevritesRootName, "dup-ignore")
	}
	if !filepath.IsAbs(ignoreFile) {
		ignoreFile = filepath.Join(projectDir(root), ignoreFile)
	}
	ignored := dupLoadIgnore(ignoreFile)
	var shown []dupCluster
	ignoredCount := 0
	for _, c := range clusters {
		if ignored[c.hash] {
			ignoredCount++
			continue
		}
		shown = append(shown, c)
	}

	relIgnore, rerr := filepath.Rel(projectDir(root), ignoreFile)
	if rerr != nil {
		relIgnore = ignoreFile
	}
	fmt.Fprintf(stdout, "dup: scanned %d files (%d skipped), %d units; ignore-file %s\n",
		len(files), skipped, len(units), filepath.ToSlash(relIgnore))
	if len(shown) == 0 {
		if failed > 0 {
			return 2
		}
		if ignoredCount > 0 {
			fmt.Fprintf(stdout, "dup: ok (all %d cluster(s) ignored via %s)\n", ignoredCount, filepath.ToSlash(relIgnore))
		} else if opts.mode != "all" && totalClusters > 0 {
			fmt.Fprintln(stdout, "dup: ok (no clusters touch the diff)")
		} else {
			fmt.Fprintln(stdout, "dup: ok (no similar regions)")
		}
		if opts.slug != "" {
			recordMetric(root, opts.slug, opts.phase, "dup", "", int64(len(clusters)))
		}
		return 0
	}
	fmt.Fprintf(stdout, "dup: %d cluster(s)", len(shown))
	if ignoredCount > 0 {
		fmt.Fprintf(stdout, " (%d ignored)", ignoredCount)
	}
	fmt.Fprintln(stdout)
	if len(shown) > opts.limit {
		shown = shown[:opts.limit]
		fmt.Fprintf(stdout, "dup: truncated to %d clusters (--limit)\n", opts.limit)
	}
	for i, c := range shown {
		fmt.Fprintf(stdout, "cluster %d (%s) score %.2f:\n", i+1, c.hash, c.score)
		members := append([]int(nil), c.units...)
		sort.Slice(members, func(a, b int) bool {
			ua, ub := units[members[a]], units[members[b]]
			if ua.changed != ub.changed {
				return ua.changed
			}
			if files[ua.file].path != files[ub.file].path {
				return files[ua.file].path < files[ub.file].path
			}
			return ua.start < ub.start
		})
		for _, u := range members {
			unit := units[u]
			marker := "  "
			if unit.changed {
				marker = "* "
			}
			fmt.Fprintf(stdout, "  %s%s:%d-%d\n", marker, files[unit.file].path, unit.start+1, unit.end+1)
		}
	}
	if failed > 0 {
		return 2
	}
	if opts.slug != "" {
		recordMetric(root, opts.slug, opts.phase, "dup", "", int64(len(shown)))
	}
	return 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// dupDetect runs the shingle/segment/cluster pipeline over scanned files.
func dupDetect(files []dupFile, opts dupOptions) ([]dupCluster, []dupUnit) {
	postings := map[uint64][]dupPos{}
	for fi := range files {
		lines := files[fi].lines
		for i := 0; i+dupShingleLines <= len(lines); i++ {
			allTrivial := true
			for j := i; j < i+dupShingleLines; j++ {
				if !lines[j].trivial {
					allTrivial = false
					break
				}
			}
			if allTrivial {
				continue // padding runs anchor everywhere and add no signal
			}
			h := dupShingleHash(lines, i)
			postings[h] = append(postings[h], dupPos{file: fi, line: i})
		}
	}
	anchorMap := map[dupPairKey][]dupAnchor{}
	for _, list := range postings {
		if len(list) > dupMaxPostings {
			continue // boilerplate shingle (e.g. brace runs) — no signal
		}
		for i := 0; i < len(list); i++ {
			for j := i + 1; j < len(list); j++ {
				a, b := list[i], list[j]
				if a.file > b.file || a.file == b.file && a.line > b.line {
					a, b = b, a
				}
				if a.file == b.file && b.line-a.line < dupShingleLines {
					continue // overlapping windows in one file match trivially
				}
				key := dupPairKey{a: a.file, b: b.file}
				anchors := anchorMap[key]
				if len(anchors) >= dupMaxPairAnchors {
					continue
				}
				anchorMap[key] = append(anchors, dupAnchor{aLine: a.line, bLine: b.line})
			}
		}
	}
	var segs []dupSegment
	for key, anchors := range anchorMap {
		segs = append(segs, dupAnchorSegments(key, anchors, files, opts.minLines, opts.threshold)...)
	}
	clusters, units, _ := dupClusterUnits(segs, files)
	return clusters, units
}

// dupReadFiles loads and normalizes every scannable path under budgets.
func dupReadFiles(repo string, paths []string) ([]dupFile, int, int) {
	var files []dupFile
	var skipped, failed int
	var remaining int64 = dupMaxBytes
	for _, p := range paths {
		full := filepath.Join(repo, filepath.FromSlash(p))
		info, err := os.Lstat(full)
		if err != nil || !info.Mode().IsRegular() || info.Size() > dupMaxFileBytes || info.Size() > remaining {
			failed++
			continue
		}
		// #nosec G304 -- repo-relative path from git ls-files, lstat'd regular file
		f, err := os.Open(full)
		if err != nil {
			failed++
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(f, min(dupMaxFileBytes, remaining)+1))
		closeErr := f.Close()
		if err != nil || closeErr != nil || int64(len(raw)) > min(dupMaxFileBytes, remaining) {
			failed++
			continue
		}
		remaining -= int64(len(raw))
		if isDupGenerated(raw) {
			skipped++
			continue
		}
		style := dupStyleFor(p)
		files = append(files, dupFile{path: p, lines: normalizeDupSource(string(raw), style)})
	}
	return files, skipped, failed
}

func isDupGenerated(raw []byte) bool {
	head := string(raw[:min(len(raw), 2048)])
	return strings.Contains(head, "Code generated") ||
		strings.Contains(head, "DO NOT EDIT") ||
		strings.Contains(head, "auto-generated") ||
		strings.Contains(head, "Automatically generated")
}

// dupScannable reports whether a repo-relative slash path is in scope: a
// known code extension or basename, outside vendored/generated trees, and
// not excluded by --exclude prefixes.
func dupScannable(p string, exclude []string) bool {
	for _, prefix := range dupSkipPrefixes {
		if dupPathPrefixMatch(p, prefix) {
			return false
		}
	}
	for _, prefix := range exclude {
		if dupPathPrefixMatch(p, prefix) {
			return false
		}
	}
	base := strings.ToLower(path.Base(p))
	if strings.Contains(base, ".min.") || strings.Contains(base, ".pb.") ||
		strings.Contains(base, ".gen.") || strings.HasSuffix(base, ".designer.cs") {
		return false
	}
	_, ok := dupStyleForOK(p)
	return ok
}

// dupPathPrefixMatch reports whether path p sits under directory prefix —
// gitignore-style: at the root or below any intermediate directory, so a
// nested `services/x/vendor/` tree is skipped the same as a root `vendor/`.
func dupPathPrefixMatch(p, prefix string) bool {
	return strings.HasPrefix(p, prefix) || strings.Contains(p, "/"+prefix)
}

func dupStyleFor(p string) dupCommentStyle {
	style, _ := dupStyleForOK(p)
	return style
}

func dupStyleForOK(p string) (dupCommentStyle, bool) {
	if style, ok := dupStyleByName[strings.ToLower(path.Base(p))]; ok {
		return style, true
	}
	style, ok := dupStyleByExt[strings.ToLower(path.Ext(p))]
	return style, ok
}

// Git listings and blob batches must fail on truncation, not scan a prefix.
type dupOutput struct {
	buffer bytes.Buffer
	limit  int
}

func (w *dupOutput) Write(p []byte) (int, error) {
	if len(p) > w.limit-w.buffer.Len() {
		return 0, fmt.Errorf("scan output exceeds resource limit")
	}
	return w.buffer.Write(p)
}

func dupGitOutput(repo string, input []byte, limit int, args ...string) ([]byte, error) {
	out := dupOutput{limit: limit}
	_, err := runGitCommandIO(repo, nil, input, &out, args...)
	return out.buffer.Bytes(), err
}

func dupReadIndex(repo string, exclude []string) ([]dupFile, int, int, error) {
	listing, err := dupGitOutput(repo, nil, gitOutputLimit, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, 0, 0, err
	}
	var paths, oids []string
	var input strings.Builder
	for _, entry := range strings.Split(string(listing), "\x00") {
		if entry == "" {
			continue
		}
		header, p, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 3 || fields[2] != "0" {
			return nil, 0, 0, fmt.Errorf("unresolved or malformed index")
		}
		if !dupScannable(p, exclude) {
			continue
		}
		if fields[0] != "100644" && fields[0] != "100755" {
			return nil, 0, 0, fmt.Errorf("non-regular indexed source %q", p)
		}
		if !isObjectID(fields[1]) || isZeroObjectID(fields[1]) {
			return nil, 0, 0, fmt.Errorf("invalid indexed blob")
		}
		paths = append(paths, p)
		oids = append(oids, fields[1])
		input.WriteString(fields[1] + "\n")
		if len(paths) > dupMaxFiles {
			return nil, 0, 0, fmt.Errorf("too many eligible files")
		}
	}
	if len(paths) == 0 {
		return nil, 0, 0, nil
	}
	data, err := dupGitOutput(repo, []byte(input.String()), dupMaxBytes+dupMaxFiles*128, "--no-replace-objects", "cat-file", "--batch")
	if err != nil {
		return nil, 0, 0, err
	}
	var files []dupFile
	skipped, failed, total := 0, 0, 0
	for i, p := range paths {
		end := bytes.IndexByte(data, '\n')
		if end < 0 {
			return nil, 0, 0, fmt.Errorf("missing blob header")
		}
		fields := strings.Fields(string(data[:end]))
		data = data[end+1:]
		if len(fields) != 3 || fields[0] != oids[i] || fields[1] != "blob" {
			return nil, 0, 0, fmt.Errorf("unavailable indexed blob")
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size >= len(data) || data[size] != '\n' {
			return nil, 0, 0, fmt.Errorf("invalid blob size")
		}
		raw := data[:size]
		data = data[size+1:]
		total += size
		if size > dupMaxFileBytes || total > dupMaxBytes {
			failed++
			continue
		}
		if isDupGenerated(raw) {
			skipped++
			continue
		}
		files = append(files, dupFile{path: p, lines: normalizeDupSource(string(raw), dupStyleFor(p))})
	}
	if len(data) != 0 {
		return nil, 0, 0, fmt.Errorf("unexpected blob output")
	}
	return files, skipped, failed, nil
}

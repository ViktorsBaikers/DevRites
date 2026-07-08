package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ArchiveSearch surfaces shipped features whose spec.md overlaps a query, so
// /rite-spec can spot prior art before writing a new spec — an extension, a
// conflict, or a re-spec of solved work. It is advisory, not a gate: it always
// exits 0 when it runs (empty output means no overlap), and reserves exit 2 for
// a missing query. The query is the feature's key nouns, one or many args; a
// quoted phrase is split on whitespace. Ranking is by the count of distinct query
// terms a spec contains, highest first, ties broken by slug.
//
// args is `<term>...`. Terms shorter than three characters are dropped as noise.
func ArchiveSearch(root string, args []string, stdout, stderr io.Writer) int {
	var terms []string
	seen := map[string]bool{}
	for _, a := range args {
		for _, t := range strings.Fields(strings.ToLower(a)) {
			if len(t) >= 3 && !seen[t] {
				seen[t] = true
				terms = append(terms, t)
			}
		}
	}
	if len(terms) == 0 {
		fmt.Fprintln(stderr, `usage: devrites-engine archive-search "<key nouns>"`)
		return 2 // usage
	}

	entries, err := os.ReadDir(filepath.Join(root, "archive"))
	if err != nil {
		// No archive yet — no prior art to surface. Silent, successful no-op.
		return 0
	}

	type hit struct {
		slug  string
		n     int
		title string
	}
	var hits []hit
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		spec := filepath.Join(root, "archive", e.Name(), "spec.md")
		b, err := os.ReadFile(spec)
		if err != nil {
			continue
		}
		body := strings.ToLower(string(b))
		n := 0
		for _, t := range terms {
			if strings.Contains(body, t) {
				n++
			}
		}
		if n > 0 {
			hits = append(hits, hit{slug: e.Name(), n: n, title: specTitle(b)})
		}
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].n != hits[j].n {
			return hits[i].n > hits[j].n
		}
		return hits[i].slug < hits[j].slug
	})
	for _, h := range hits {
		fmt.Fprintf(stdout, "%s\t%d/%d terms\t%s\n", h.slug, h.n, len(terms), h.title)
	}
	return 0
}

// specTitle returns the spec's first Markdown heading, or its first non-empty
// line, as a one-line label — empty when the spec has neither.
func specTitle(b []byte) string {
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return strings.TrimSpace(strings.TrimLeft(line, "#"))
	}
	return ""
}

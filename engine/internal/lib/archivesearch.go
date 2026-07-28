package lib

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ArchiveSearch surfaces shipped features whose spec.md overlaps a query, so
// /rite-spec can spot prior art before writing a new spec: an extension, a
// conflict, or a re-spec of solved work. It is advisory, not a gate: empty
// output means no overlap. Exit 2 means a missing query; exit 3 means the
// archive could not be read. The query is the feature's key nouns, one or many
// args; a quoted phrase is split on whitespace. Ranking is by the count of
// distinct query terms a spec contains, highest first, ties broken by slug.
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

	archive := filepath.Join(root, "archive")
	info, err := os.Stat(archive)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No archive yet: no prior art to surface. Silent, successful no-op.
			return 0
		}
		fmt.Fprintf(stderr, "archive-search: read archive: %v\n", err)
		return 3
	}
	if !info.IsDir() {
		fmt.Fprintf(stderr, "archive-search: read archive: %s is not a directory\n", archive)
		return 3
	}
	entries, err := os.ReadDir(archive)
	if err != nil {
		fmt.Fprintf(stderr, "archive-search: read archive: %v\n", err)
		return 3
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
		info, err := os.Stat(spec)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			fmt.Fprintf(stderr, "archive-search: read %s: %v\n", spec, err)
			return 3
		}
		if !info.Mode().IsRegular() {
			fmt.Fprintf(stderr, "archive-search: read %s: not a regular file\n", spec)
			return 3
		}
		b, err := os.ReadFile(spec)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			fmt.Fprintf(stderr, "archive-search: read %s: %v\n", spec, err)
			return 3
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
// line, as a one-line label: empty when the spec has neither.
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

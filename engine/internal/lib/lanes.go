package lib

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var lanePathRe = regexp.MustCompile("`([^`]+)`")

// Lanes plans conservative parallelism for a feature without changing state. It
// never authorizes parallel production writes; it only labels read-only lanes,
// Forge candidates, and obviously independent slices so the human/agent can keep
// DevRites' one-verified-slice default intact.
func Lanes(root string, args []string, stdout, stderr io.Writer) int {
	if argAt(args, 0) != "plan" {
		fmt.Fprintln(stderr, "usage: devrites-engine lanes plan [slug]")
		return 2
	}
	slug := argAt(args, 1)
	if slug == "" {
		slug = activeSlug(root)
	}
	if slug == "" {
		fmt.Fprintln(stderr, "lanes: no active feature")
		return 2
	}
	tasks, ok := readFileOK(filepath.Join(featureDir(root, slug), "tasks.md"))
	if !ok {
		fmt.Fprintf(stdout, "lanes: no tasks.md for %s — no parallel write lanes.\n", slug)
		return 0
	}
	slices := parseLaneSlices(tasks)
	fmt.Fprintf(stdout, "# Lane plan: %s\n\n", slug)
	if len(slices) == 0 {
		fmt.Fprintln(stdout, "No slice headings found. Keep the default single-slice build loop.")
		return 0
	}
	groups := map[string][]string{}
	for _, s := range slices {
		key := "read-only/review"
		if s.forge {
			key = "forge-candidate"
		} else if len(s.paths) > 0 {
			key = strings.Join(s.paths, ",")
		}
		groups[key] = append(groups[key], s.name)
	}
	fmt.Fprintln(stdout, "Default: one production write slice at a time. Safe parallelism is advisory only.")
	fmt.Fprintln(stdout)
	for _, s := range slices {
		kind := "serial-write"
		if s.forge {
			kind = "forge-candidate"
		} else if len(s.paths) == 0 {
			kind = "read-only/spike"
		}
		conflict := "unknown write surface"
		if len(s.paths) > 0 {
			conflict = strings.Join(s.paths, ", ")
		}
		fmt.Fprintf(stdout, "- %s: %s — %s\n", s.name, kind, conflict)
	}
	return 0
}

type laneSlice struct {
	name  string
	paths []string
	forge bool
}

func parseLaneSlices(md string) []laneSlice {
	var out []laneSlice
	var cur *laneSlice
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
		}
	}
	for _, line := range strings.Split(md, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "## Slice") {
			flush()
			cur = &laneSlice{name: strings.TrimPrefix(trim, "## ")}
			continue
		}
		if cur == nil {
			continue
		}
		lower := strings.ToLower(trim)
		if strings.Contains(lower, "forge: yes") || strings.Contains(lower, "forge yes") {
			cur.forge = true
		}
		for _, m := range lanePathRe.FindAllStringSubmatch(trim, -1) {
			if !slices.Contains(cur.paths, m[1]) {
				cur.paths = append(cur.paths, m[1])
			}
		}
	}
	flush()
	return out
}

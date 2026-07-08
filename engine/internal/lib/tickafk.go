package lib

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// budgetField matches an "AFK slices remaining:" line and captures the value that
// follows. The optional leading "- " tolerates state.md's markdown-bullet form.
var budgetField = regexp.MustCompile(`^[[:space:]]*-?[[:space:]]*AFK slices remaining:[[:space:]]*`)

// TickAfk spends one slice from a run's AFK budget: it reads the "AFK slices
// remaining" count from the state.md at args[0], decrements it, and writes the
// file back. /rite-build calls it after building a slice unattended.
//
// It operates on a state.md path (not a feature slug) so a caller can point it at
// any workspace. Exit codes:
//
//	0  a slice was spent and budget remains, or no budget is set (nothing to do)
//	3  the budget is now exhausted — the caller must stop and hand back to a human
//	5  the file is missing, or the field holds a non-numeric value
func TickAfk(args []string, stdout, stderr io.Writer) int {
	path := argAt(args, 0)
	if path == "" || !isFile(path) {
		where := path
		if where == "" {
			where = "<unset>"
		}
		fmt.Fprintf(stderr, "tick-afk: state.md not found at %s\n", where)
		return 5
	}

	data, _ := os.ReadFile(path)
	lines := splitLinesNoTrailing(data)

	remaining, found := readBudget(lines)
	if !found || remaining == "none" {
		fmt.Fprintf(stdout, "tick-afk: no \"AFK slices remaining\" budget set in %s — no-op\n", path)
		return 0
	}
	if !isAllDigits(remaining) {
		fmt.Fprintf(stderr, "tick-afk: \"AFK slices remaining\" is not a number (%s) in %s\n", remaining, path)
		return 5
	}

	n, _ := strconv.Atoi(remaining)
	if n--; n < 0 {
		n = 0
	}

	if err := writeFileAtomic(path, setBudget(lines, n), 0o644); err != nil {
		fmt.Fprintf(stderr, "tick-afk: %v\n", err)
		return 5
	}
	fmt.Fprintln(stdout, n)

	if n <= 0 {
		return 3
	}
	return 0
}

// readBudget returns the current "AFK slices remaining" value — the first blank-
// delimited token after the field — and whether the field is present at all.
func readBudget(lines []string) (value string, found bool) {
	for _, line := range lines {
		loc := budgetField.FindStringIndex(line)
		if loc == nil {
			continue
		}
		value = line[loc[1]:]
		if i := strings.IndexAny(value, spaceChars); i >= 0 {
			value = value[:i]
		}
		return value, true
	}
	return "", false
}

// setBudget rewrites every budget line to the canonical bullet form with the new
// count, passing all other lines through. The result is newline-terminated.
func setBudget(lines []string, n int) []byte {
	var b strings.Builder
	for _, line := range lines {
		if budgetField.MatchString(line) {
			fmt.Fprintf(&b, "- AFK slices remaining: %d", n)
		} else {
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

package lib

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/state"
)

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

	if err := fsutil.WriteFileAtomic(path, setBudget(lines, n), 0o644); err != nil {
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
	value, found = state.CursorField(lines, "afk_slices_remaining")
	if !found {
		return "", false
	}
	if i := strings.IndexAny(value, spaceChars); i >= 0 {
		value = value[:i]
	}
	return value, true
}

// setBudget preserves the cursor's canonical-table or legacy-bullet format.
func setBudget(lines []string, n int) []byte {
	updated, _ := state.SetCursorField(lines, "afk_slices_remaining", strconv.Itoa(n))
	return []byte(strings.Join(updated, "\n") + "\n")
}

package acceptance

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/markdowntext"
)

// ApprovedCommand is one vetted runtime command row: the exact command text
// and its normalized working directory. A gate CHECK may execute only when
// (command, cwd) matches a row of test-plan.md's Build-entry preflight table —
// approval is review of an exact pair, never a shell-equivalence judgment.
type ApprovedCommand struct {
	Command string
	Cwd     string // normalized; "" means the repository root
}

// PreflightRow is one row of test-plan.md's Build-entry preflight table: the
// approved (command, cwd) pair plus the cells check preconditions reads.
type PreflightRow struct {
	ApprovedCommand
	Gate     string
	Expected string
	// RunAt is the optional `Run at` cell (`vet`, `pre-dispatch SLICE-###`,
	// `action`), whitespace-collapsed; "" when the row or table has none.
	RunAt string
	// Width and HeaderWidth are the row's and its table header's cell counts;
	// they differ when a bare `|` in a command splits the row.
	Width, HeaderWidth int
}

// ApprovedCommands extracts the approved command surface of test-plan.md: the
// Command and Cwd cells of every row under `## Build-entry preflight`. No
// other section approves execution — fenced examples, prose mentions, and
// consumptive-action rows are not re-runnable oracles. A test plan without
// the section approves nothing, so every runnable gate stays unapproved.
//
// The result is never nil: an absent, unreadable, or unparsable plan yields an
// empty surface rather than an absent one, because "approvals could not be
// determined" must be a value the caller can refuse on, never a nil that reads
// as "no approval is applicable".
func ApprovedCommands(testPlan []byte) []ApprovedCommand {
	approved := []ApprovedCommand{}
	rows, _ := PreflightRows(testPlan)
	for _, row := range rows {
		approved = append(approved, row.ApprovedCommand)
	}
	return approved
}

// PreflightRows parses every command row under `## Build-entry preflight`, the
// one parser behind the approved surface. hasRunAt reports whether a table in
// the section declares the optional `Run at` column; plans without it keep
// the pre-column behavior everywhere.
func PreflightRows(testPlan []byte) (rows []PreflightRow, hasRunAt bool) {
	structural, err := markdowntext.Structural(testPlan)
	if err != nil {
		return nil, false
	}
	inSection := false
	commandCol, cwdCol, gateCol, expectedCol, runAtCol, headerWidth := -1, -1, -1, -1, -1, 0
	for _, line := range strings.Split(string(structural), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			inSection = strings.EqualFold(trimmed, "## Build-entry preflight")
			commandCol = -1
			continue
		}
		if !inSection {
			continue
		}
		cells, ok := preflightCells(line)
		if !ok {
			continue
		}
		if commandCol < 0 {
			commandCol = columnIndex(cells, "command")
			cwdCol = columnIndex(cells, "cwd")
			gateCol = columnIndex(cells, "gate")
			expectedCol = columnIndex(cells, "expected")
			runAtCol = columnIndex(cells, "run at")
			headerWidth = len(cells)
			hasRunAt = hasRunAt || (commandCol >= 0 && runAtCol >= 0)
			continue
		}
		if isDividerRow(cells) || commandCol >= len(cells) {
			continue
		}
		command := unquoteCode(strings.TrimSpace(cells[commandCol]))
		if command == "" {
			continue
		}
		row := PreflightRow{ApprovedCommand: ApprovedCommand{Command: command}, Width: len(cells), HeaderWidth: headerWidth}
		if cwdCol >= 0 && cwdCol < len(cells) {
			row.Cwd = normalizeApprovedCwd(cells[cwdCol])
		}
		row.Gate = cellText(cells, gateCol)
		row.Expected = cellText(cells, expectedCol)
		row.RunAt = strings.Join(strings.Fields(cellText(cells, runAtCol)), " ")
		rows = append(rows, row)
	}
	return rows, hasRunAt
}

// PreflightShapeProblems names each Build-entry preflight row that cannot run
// as written: a cell count that differs from its header (an unescaped `|`
// in the command, which must be written `\|`) or an invalid `Run at` value.
// `check preconditions` would otherwise find it only when Vet runs the row.
func PreflightShapeProblems(testPlan []byte) []string {
	rows, _ := PreflightRows(testPlan)
	var problems []string
	for _, row := range rows {
		name := valueOr(row.Gate, row.Command)
		if row.Width != row.HeaderWidth {
			problems = append(problems, fmt.Sprintf("preflight row %q has %d cells but its header has %d; escape each `|` in the command as `\\|`", name, row.Width, row.HeaderWidth))
		} else if !ValidRunAt(row.RunAt) {
			problems = append(problems, fmt.Sprintf("preflight row %q has invalid Run at %q (want vet, pre-dispatch SLICE-###, or action)", name, row.RunAt))
		}
	}
	return problems
}

// ValidRunAt reports whether a `Run at` cell is empty or a known value.
func ValidRunAt(runAt string) bool {
	lower := strings.ToLower(runAt)
	return lower == "" || lower == "vet" || lower == "action" || preDispatchRunAtRE.MatchString(runAt)
}

// cellText returns a cell's trimmed text without code backticks, "" when the
// column is absent or the row is short.
func cellText(cells []string, col int) string {
	if col < 0 || col >= len(cells) {
		return ""
	}
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(cells[col]), "`"))
}

// Approved reports whether (check, cwd) is a vetted pair. Approval is
// exact-text equality on the command after outer-whitespace normalization,
// plus exact equality on the normalized working directory: a differently
// spelled command or a different directory is a different oracle that needs
// its own preflight row.
func Approved(check, cwd string, approved []ApprovedCommand) bool {
	check = normalizeCommand(check)
	cwd = normalizeApprovedCwd(cwd)
	for _, row := range approved {
		if row.Command == check && row.Cwd == cwd {
			return true
		}
	}
	return false
}

// normalizeApprovedCwd maps the preflight table's root conventions ("",
// "repository", "root", ".") onto the empty root marker and cleans relative
// paths so `engine/` and `engine` name the same directory.
func normalizeApprovedCwd(cwd string) string {
	cwd = strings.TrimSpace(strings.Trim(cwd, "`"))
	switch strings.ToLower(cwd) {
	case "", ".", "./", "repository", "root", "repo", "n/a", "-":
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(cwd)))
}

func columnIndex(cells []string, name string) int {
	for i, cell := range cells {
		if strings.EqualFold(strings.TrimSpace(cell), name) {
			return i
		}
	}
	return -1
}

// preflightCells splits a table row on unescaped pipes and unescapes `\|`, so
// a command containing a shell pipe survives as one cell.
func preflightCells(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return nil, false
	}
	body := trimmed[1 : len(trimmed)-1]
	var cells []string
	var cell strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] == '\\' && i+1 < len(body) && body[i+1] == '|' {
			cell.WriteByte('|')
			i++
			continue
		}
		if body[i] == '|' {
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
			continue
		}
		cell.WriteByte(body[i])
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	return cells, true
}

func isDividerRow(cells []string) bool {
	for _, cell := range cells {
		c := strings.TrimSpace(cell)
		if c == "" {
			continue
		}
		if strings.Trim(c, ":-") != "" {
			return false
		}
	}
	return true
}

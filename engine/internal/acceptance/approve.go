package acceptance

import (
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

// ApprovedCommands extracts the approved command surface of test-plan.md: the
// Command and Cwd cells of every row under `## Build-entry preflight`. No
// other section approves execution — fenced examples, prose mentions, and
// consumptive-action rows are not re-runnable oracles. A test plan without
// the section approves nothing, so every runnable gate stays unapproved.
func ApprovedCommands(testPlan []byte) []ApprovedCommand {
	structural, err := markdowntext.Structural(testPlan)
	if err != nil {
		return nil
	}
	var approved []ApprovedCommand
	inSection := false
	commandCol, cwdCol := -1, -1
	for _, line := range strings.Split(string(structural), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			inSection = strings.EqualFold(trimmed, "## Build-entry preflight")
			commandCol, cwdCol = -1, -1
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
			continue
		}
		if isDividerRow(cells) || commandCol >= len(cells) {
			continue
		}
		command := unquoteCode(strings.TrimSpace(cells[commandCol]))
		if command == "" {
			continue
		}
		cwd := ""
		if cwdCol >= 0 && cwdCol < len(cells) {
			cwd = normalizeApprovedCwd(cells[cwdCol])
		}
		approved = append(approved, ApprovedCommand{Command: command, Cwd: cwd})
	}
	return approved
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

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/devrites/devrites/internal/harness"
)

// cmdHarnessMatrix renders the harness adapter-compliance matrix, or (with
// --check FILE) verifies the generated block committed in FILE still matches the
// engine's frozen matrix. The check is the drift guard: an edit to the matrix
// data without regenerating the doc — or vice versa — fails CI.
//
// Exit: 0 rendered / in-sync · 2 usage · 3 drift (regenerate + commit).
func cmdHarnessMatrix(args []string, stdout, stderr io.Writer) int {
	block := harness.RenderMatrix()

	if len(args) == 0 {
		fmt.Fprintln(stdout, block)
		return exitOK
	}
	if len(args) != 2 || args[0] != "--check" {
		fmt.Fprintln(stderr, "usage: devrites-engine harness-matrix [--check <file>]")
		return exitUsage
	}
	path := args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "devrites: harness-matrix: %v\n", err)
		return exitUsage
	}
	got, ok := extractBlock(string(data), harness.MatrixBeginMarker, harness.MatrixEndMarker)
	if !ok {
		fmt.Fprintf(stderr, "harness-matrix: no generated block found in %s (expected the BEGIN/END markers)\n", path)
		return exitBlocked
	}
	if strings.TrimSpace(got) != strings.TrimSpace(block) {
		fmt.Fprintf(stderr, "harness-matrix: %s is out of date — run `devrites-engine harness-matrix` and commit the regenerated block.\n", path)
		return exitBlocked
	}
	fmt.Fprintf(stdout, "harness-matrix: %s is in sync.\n", path)
	return exitOK
}

// extractBlock returns the text from begin to end inclusive, or ok=false if
// either marker is absent or out of order.
func extractBlock(s, begin, end string) (string, bool) {
	i := strings.Index(s, begin)
	if i < 0 {
		return "", false
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		return "", false
	}
	return s[i : i+j+len(end)], true
}

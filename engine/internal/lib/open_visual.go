package lib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/safepath"
)

const openVisualPlaybookHint = "pack/.claude/skills/devrites-lib/reference/visual-playbooks/index.md"

// openVisualOpener launches a local file in the OS default browser.
// Tests replace this to assert --no-open and avoid spawning a browser.
var openVisualOpener = openLocalFile

// OpenVisual resolves a workspace visual HTML file, optionally opens it in the
// OS browser, warns when the sibling outline is missing or inventory ids are
// absent from HTML, and prints an agent tip. It never performs network I/O.
//
// Usage: open-visual <path-or-name> [--slug <slug>] [--no-open]
func OpenVisual(root string, args []string, stdout, stderr io.Writer) int {
	operand, slug, noOpen, err := parseOpenVisualArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "open-visual: %v\n", err)
		fmt.Fprintln(stderr, "usage: devrites-engine open-visual <path-or-name> [--slug <slug>] [--no-open]")
		return 2
	}

	htmlPath, nameMode, err := resolveOpenVisualHTML(root, slug, operand)
	if err != nil {
		fmt.Fprintf(stderr, "open-visual: %v\n", err)
		return 2
	}

	info, err := os.Stat(htmlPath)
	if err != nil {
		fmt.Fprintf(stderr, "open-visual: cannot open %s: %v\n", htmlPath, err)
		return 2
	}
	if info.IsDir() {
		fmt.Fprintf(stderr, "open-visual: %s is a directory, not an HTML file\n", htmlPath)
		return 2
	}
	if resolved, err := filepath.EvalSymlinks(htmlPath); err == nil {
		htmlPath = resolved
	}
	if !strings.EqualFold(filepath.Ext(htmlPath), ".html") {
		fmt.Fprintf(stderr, "open-visual: require a .html file, got %s\n", htmlPath)
		return 2
	}
	if nameMode {
		feature, err := resolveOpenVisualFeatureDir(root, slug)
		if err != nil {
			fmt.Fprintf(stderr, "open-visual: %v\n", err)
			return 2
		}
		visualDir := filepath.Join(feature, "visual")
		if !safepath.WithinResolved(htmlPath, visualDir) {
			fmt.Fprintf(stderr, "open-visual: refused: resolved path escapes workspace visual/\n")
			return 2
		}
	}

	outlinePath := strings.TrimSuffix(htmlPath, filepath.Ext(htmlPath)) + ".outline.md"
	outlineMissing := false
	if st, err := os.Stat(outlinePath); err != nil || st.IsDir() {
		outlineMissing = true
		fmt.Fprintf(stderr, "open-visual: warning: missing outline companion %s\n", outlinePath)
	}

	var idReport VisualIDConsistency
	idsChecked := false
	if !outlineMissing {
		// #nosec G304 -- visual evidence files inside the workspace
		htmlBody, herr := os.ReadFile(htmlPath)
		// #nosec G304 -- visual evidence files inside the workspace
		outlineBody, oerr := os.ReadFile(outlinePath)
		switch {
		case herr != nil:
			fmt.Fprintf(stderr, "open-visual: warning: cannot read HTML for id check: %v\n", herr)
		case oerr != nil:
			fmt.Fprintf(stderr, "open-visual: warning: cannot read outline for id check: %v\n", oerr)
		default:
			idReport = CheckVisualIDConsistency(string(htmlBody), string(outlineBody))
			idsChecked = true
			if len(idReport.MissingInHTML) > 0 {
				fmt.Fprintf(stderr, "open-visual: warning: %d outline inventory id(s) missing from HTML: %s\n",
					len(idReport.MissingInHTML), strings.Join(idReport.MissingInHTML, ", "))
			}
		}
	}

	// Print tips before OS open so agents still get paths if the opener fails.
	fmt.Fprintf(stdout, "open-visual: html=%s\n", htmlPath)
	if outlineMissing {
		fmt.Fprintf(stdout, "open-visual: outline=(missing) %s\n", outlinePath)
	} else {
		fmt.Fprintf(stdout, "open-visual: outline=%s\n", outlinePath)
	}
	fmt.Fprintf(stdout, "open-visual: playbooks=%s\n", openVisualPlaybookHint)
	if idsChecked {
		switch {
		case len(idReport.MissingInHTML) > 0:
			fmt.Fprintf(stdout, "open-visual: ids=mismatch missing=%d inventory=%d\n",
				len(idReport.MissingInHTML), len(idReport.Inventory))
		case len(idReport.Inventory) > 0:
			fmt.Fprintf(stdout, "open-visual: ids=ok (%d inventory)\n", len(idReport.Inventory))
		}
	}

	if !noOpen {
		if err := openVisualOpener(htmlPath); err != nil {
			fmt.Fprintf(stderr, "open-visual: warning: failed to open browser: %v\n", err)
			// Tips already printed; HTML is local and resolved — warn-and-continue.
		}
	}
	return 0
}

func parseOpenVisualArgs(args []string) (operand, slug string, noOpen bool, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--no-open":
			noOpen = true
		case arg == "--slug":
			if i+1 >= len(args) {
				return "", "", false, fmt.Errorf("--slug requires a value")
			}
			i++
			slug = strings.TrimSpace(args[i])
			if slug == "" {
				return "", "", false, fmt.Errorf("--slug requires a value")
			}
		case strings.HasPrefix(arg, "--slug="):
			slug = strings.TrimSpace(strings.TrimPrefix(arg, "--slug="))
			if slug == "" {
				return "", "", false, fmt.Errorf("--slug requires a value")
			}
		case strings.HasPrefix(arg, "-"):
			return "", "", false, fmt.Errorf("unknown flag %q", arg)
		default:
			if operand != "" {
				return "", "", false, fmt.Errorf("unexpected argument %q", arg)
			}
			operand = arg
		}
	}
	if strings.TrimSpace(operand) == "" {
		return "", "", false, fmt.Errorf("path or visual name required")
	}
	return operand, slug, noOpen, nil
}

// resolveOpenVisualHTML returns the HTML path and whether the operand was a
// workspace visual name (nameMode). Absolute/relative path operands open any
// local .html; name operands resolve under the feature visual/ tree.
func resolveOpenVisualHTML(root, slug, operand string) (string, bool, error) {
	if isOpenVisualPathOperand(operand) {
		abs, err := filepath.Abs(operand)
		if err != nil {
			retErr := fmt.Errorf("resolve path: %w", err)
			return "", false, retErr
		}
		cleaned := filepath.Clean(abs)
		return cleaned, false, nil
	}

	name := operand
	if !strings.EqualFold(filepath.Ext(name), ".html") {
		name += ".html"
	}
	if filepath.Base(name) != name {
		retErr := fmt.Errorf("visual name must not contain path separators")
		return "", true, retErr
	}

	feature, err := resolveOpenVisualFeatureDir(root, slug)
	if err != nil {
		return "", true, err
	}
	htmlPath := filepath.Join(feature, "visual", name)
	return htmlPath, true, nil
}

// isOpenVisualPathOperand reports whether operand is a filesystem path rather
// than a workspace visual name. Absolute paths and operands with separators are
// paths. Leading "./" or "../" (and bare "." / "..") are relative paths; a
// leading-dot basename alone (e.g. ".draft") is still a visual name.
func isOpenVisualPathOperand(operand string) bool {
	if filepath.IsAbs(operand) {
		return true
	}
	if operand == "." || operand == ".." {
		return true
	}
	if strings.HasPrefix(operand, "./") || strings.HasPrefix(operand, "../") {
		return true
	}
	if strings.HasPrefix(operand, `.\`) || strings.HasPrefix(operand, `..\`) {
		return true
	}
	return strings.ContainsAny(operand, `/\`)
}

func resolveOpenVisualFeatureDir(root, slug string) (string, error) {
	if root == "" {
		retErr := fmt.Errorf("DevRites root required to resolve a visual name")
		return "", retErr
	}
	if slug == "" {
		active, err := devritespaths.ActiveSlug(root)
		if err != nil {
			return "", err
		}
		if active == "" {
			retErr := fmt.Errorf("no slug: pass --slug or set ACTIVE / DEVRITES_WORKSPACE")
			return "", retErr
		}
		slug = active
	}
	dir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		if os.IsNotExist(err) {
			retErr := fmt.Errorf("no workspace for slug %q", slug)
			return "", retErr
		}
		return "", err
	}
	return dir, nil
}

func openLocalFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// argv only — no shell.
		// #nosec G204 -- fixed platform opener; path passed as argv, no shell
		cmd = exec.Command("open", path)
	case "windows":
		// Avoid cmd.exe /c shell composition; FileProtocolHandler takes one path argv.
		// #nosec G204 -- fixed platform opener; path passed as argv, no shell
		cmd = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", path)
	default:
		// argv only — no shell.
		// #nosec G204 -- fixed platform opener; path passed as argv, no shell
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	return nil
}

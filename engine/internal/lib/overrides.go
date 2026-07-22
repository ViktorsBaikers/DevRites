package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// subversionPhrase matches an override trying to talk a reviewer OUT of a gate
// rather than adding emphasis to it. Overrides are advisory reviewer input; they
// can raise the bar, never lower it. This is a heuristic tripwire, not a parser:
// it exists so a human notices an override that reads like "waive the gate".
var subversionPhrase = regexp.MustCompile(`(?i)\b(disable|skip|waive|bypass|ignore|suppress|turn off|don'?t (?:block|flag|enforce))\b[^.\n]{0,40}\b(gate|critical|no-?go|review|principle|check|severity|finding)\b|` +
	`\b(downgrade|lower|relax|treat)\b[^.\n]{0,40}\b(critical|severity|no-?go|blocker)\b`)

// Overrides is the linter for the project's reviewer-override layer:
// .devrites/overrides/<agent-name>.md, advisory text a reviewer reads AFTER its
// governing standards to pick up house rules. The layer lets a project tune a
// shipped reviewer without forking the pack; this command keeps it honest: an
// override may add checks, never relax a gate.
//
//	list       enumerate override files and the agent each targets
//	validate   flag empty overrides and any that read like gate-subversion; 1 on a subversion hit
func Overrides(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "usage: devrites-engine overrides list|validate"
	sub := argAt(args, 0)
	dir := filepath.Join(root, "overrides")
	switch sub {
	case "list":
		return overridesList(dir, stdout)
	case "validate":
		return overridesValidate(dir, filepath.Dir(root), stdout, stderr)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
}

// overrideFiles returns the *.md override files under dir, sorted. A missing dir
// means the project declares no overrides: not an error.
func markdownFiles(dir, kind string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list %s: %w", kind, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		out = append(out, e.Name())
	}
	return out, nil
}

func overrideFiles(dir string) ([]string, error) { return markdownFiles(dir, "overrides") }

func overridesList(dir string, stdout io.Writer) int {
	files, err := overrideFiles(dir)
	if err != nil {
		fmt.Fprintf(stdout, "overrides: cannot read %s: %v\n", dir, err)
		return 0
	}
	if len(files) == 0 {
		fmt.Fprintln(stdout, "overrides: none (.devrites/overrides/ empty or absent)")
		return 0
	}
	for _, f := range files {
		fmt.Fprintf(stdout, "  %s → agent %s\n", f, strings.TrimSuffix(f, ".md"))
	}
	return 0
}

// overridesValidate checks each override is a non-empty file that reads as added
// emphasis, not a gate waiver. An orphan (no matching agent in the installed pack)
// is a warning, not a failure: the pack layout differs across harnesses. A
// subversion phrase is a hard finding, since an override that waives a gate is the
// one thing this layer must never do.
//
//	0  every override well-formed (or none)
//	1  an override reads like it waives a gate
func overridesValidate(dir, projectDir string, stdout, stderr io.Writer) int {
	files, err := overrideFiles(dir)
	if err != nil {
		fmt.Fprintf(stderr, "overrides: cannot read %s: %v\n", dir, err)
		return 1
	}
	templates, err := markdownFiles(filepath.Join(dir, "templates"), "template overrides")
	if err != nil {
		fmt.Fprintf(stderr, "overrides: cannot read %s: %v\n", filepath.Join(dir, "templates"), err)
		return 1
	}
	if len(files) == 0 && len(templates) == 0 {
		fmt.Fprintln(stdout, "overrides: none to validate")
		return 0
	}

	subversions := 0
	for _, f := range files {
		body, _ := readFileOK(filepath.Join(dir, f))
		if strings.TrimSpace(body) == "" {
			fmt.Fprintf(stdout, "  warning: %s is empty: no override applied\n", f)
			continue
		}
		agent := strings.TrimSuffix(f, ".md")
		if !overrideTargetExists(projectDir, agent) {
			fmt.Fprintf(stdout, "  warning: %s targets %q, which is not an installed agent (orphan override)\n", f, agent)
		}
		if subversionPhrase.MatchString(body) {
			fmt.Fprintf(stderr, "  VIOLATION: %s reads like it waives a gate: an override may add checks, never relax one.\n", f)
			subversions++
		}
	}
	for _, f := range templates {
		rel := filepath.ToSlash(filepath.Join("templates", f))
		body, _ := readFileOK(filepath.Join(dir, "templates", f))
		if strings.TrimSpace(body) == "" {
			fmt.Fprintf(stdout, "  warning: %s is empty: no template override applied\n", rel)
			continue
		}
		if subversionPhrase.MatchString(body) {
			fmt.Fprintf(stderr, "  VIOLATION: %s reads like it waives a gate: a template override may add structure, never relax one.\n", rel)
			subversions++
		}
		for _, term := range requiredTemplateGateTerms(f) {
			if !strings.Contains(strings.ToLower(body), term) {
				fmt.Fprintf(stderr, "  VIOLATION: %s missing required gate term %q\n", rel, term)
				subversions++
			}
		}
	}

	if subversions > 0 {
		fmt.Fprintf(stderr, "overrides: %d override(s) attempt to relax a gate. A gate stays authoritative; remove the waiver.\n", subversions)
		return 1
	}
	fmt.Fprintf(stdout, "overrides: OK: %d reviewer override(s), %d template override(s), none relaxing a gate\n", len(files), len(templates))
	return 0
}

func requiredTemplateGateTerms(name string) []string {
	switch name {
	case "seal.md":
		return []string{"type-go", "no-go"}
	case "ship.md":
		return []string{"type-go"}
	default:
		return nil
	}
}

// overrideTargetExists reports whether an installed agent named agent is present
// under either harness's agent dir. Best-effort: a false negative only downgrades
// to a warning.
func overrideTargetExists(projectDir, agent string) bool {
	return isFile(filepath.Join(projectDir, ".claude", "agents", agent+".md")) ||
		isFile(filepath.Join(projectDir, ".codex", "agents", agent+".toml"))
}

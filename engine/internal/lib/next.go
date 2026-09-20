package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/workflow"
)

// RunNext prints the minimal remaining lifecycle path: the first phase whose
// required artifacts are not all present, plus advisory skips for phases whose
// remaining work is vacuous (e.g. clarify with zero open questions). Advisory
// only — the rite still owns the decision to advance.
func RunNext(root string, args []string, stdout, stderr io.Writer) int {
	slug, code, err := ActiveSlug(root, args)
	if err != nil {
		fmt.Fprintf(stderr, "next: %v\n", err)
		if code == 0 {
			code = 2
		}
		return code
	}
	featureDir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "next: %v\n", err)
		return 2
	}
	report, err := state.Status(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "next: %v\n", err)
		return 2
	}
	openQuestions := countOpenQuestions(filepath.Join(featureDir, "questions.md"))

	policies := state.PhasePolicies()
	currentIdx := -1
	for i, policy := range policies {
		if policy.Target == report.Phase {
			currentIdx = i
			break
		}
	}
	if currentIdx < 0 {
		fmt.Fprintf(stderr, "next: unknown phase %q in state.md\n", report.Phase)
		return 2
	}

	fmt.Fprintf(stdout, "current: %s\n", report.Phase)
	if report.Complete() {
		fmt.Fprintln(stdout, "current-status: complete")
	} else {
		fmt.Fprintln(stdout, "current-status: incomplete")
	}
	fmt.Fprintf(stdout, "open-questions: %d\n", openQuestions)

	if report.Phase == state.PhaseDone {
		fmt.Fprintln(stdout, "next-phase: done")
		return 0
	}

	// Walk from the current phase to the first incomplete one. Vacuous phases
	// are reported as advisory skips, not silently dropped.
	var skips []string
	for i := currentIdx; i < len(policies); i++ {
		policy := policies[i]
		if policy.Target == state.PhaseDone {
			fmt.Fprintln(stdout, "next-phase: done")
			printSkips(stdout, skips)
			return 0
		}
		var missing []string
		if i == currentIdx {
			if report.Complete() {
				continue
			}
			missing = report.MissingFiles
		} else {
			if policy.Target == state.PhaseClarify && openQuestions == 0 {
				skips = append(skips, "clarify (0 open questions)")
				continue
			}
			missing = missingArtifacts(featureDir, policy.RequiredArtifacts)
			if len(missing) == 0 {
				skips = append(skips, fmt.Sprintf("%s (artifacts complete)", policy.Target))
				continue
			}
		}
		fmt.Fprintf(stdout, "next-phase: %s\n", policy.Target)
		fmt.Fprintf(stdout, "next-command: %s\n", renderNextCommand(policy))
		if policy.BlocksOpenQuestions && openQuestions > 0 && policy.Target != state.PhaseClarify {
			fmt.Fprintf(stdout, "blocked: %d open question(s) gate this phase\n", openQuestions)
		}
		if len(missing) > 0 {
			fmt.Fprintf(stdout, "missing: %s\n", strings.Join(missing, ","))
		}
		printSkips(stdout, skips)
		return 0
	}
	fmt.Fprintln(stdout, "next-phase: done")
	printSkips(stdout, skips)
	return 0
}

func renderNextCommand(policy state.PhasePolicy) string {
	verb := policy.ResumeVerb
	if verb == "" {
		return "(no resume command)"
	}
	return workflow.ForVerb(verb).Claude
}

func printSkips(stdout io.Writer, skips []string) {
	for _, skip := range skips {
		fmt.Fprintf(stdout, "skip: %s\n", skip)
	}
}

func missingArtifacts(featureDir string, required []state.ArtifactPath) []string {
	var missing []string
	for _, artifact := range required {
		path := filepath.Join(featureDir, filepath.FromSlash(string(artifact)))
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			missing = append(missing, string(artifact))
		}
	}
	return missing
}

var (
	openStatusLine = regexp.MustCompile(`(?i)^\s*status:\s*open\s*$`)
	openTableCell  = regexp.MustCompile(`(?i)\|\s*open\s*\|`)
)

// countOpenQuestions tolerates both the block form ("status: open") and the
// register table form ("| Q-001 | open | ... |").
func countOpenQuestions(path string) int {
	data, err := os.ReadFile(path) // #nosec G304 -- path is a validated workspace artifact
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if openStatusLine.MatchString(line) || openTableCell.MatchString(line) {
			count++
		}
	}
	return count
}

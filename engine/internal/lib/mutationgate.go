package lib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// MutationGate advises whether the slice's tests are strong enough to back "done".
// Coverage says a line ran; mutation testing says it is checked. Running a
// mutation suite is slow, so this does not run one: it detects which runner the
// project has, prints the command to run it scoped to the changed files, and leaves
// the resulting score for /rite-seal to band. With no runner installed it prints a
// hand-mutation note instead: a project without mutation tooling is never blocked.
//
// Exit codes: 0 advisory printed · 2 no active workspace. (Turning a low score into
// a NO-GO is /rite-seal's job against the recorded score, so this always exits 0
// once it has a workspace.)
func MutationGate(root string, args []string, stdout, stderr io.Writer) int {
	slug := slugOrActive(root, args)
	if slug == "" {
		fmt.Fprintln(stderr, "mutation-gate: no active workspace.")
		return 2
	}
	d := featureDir(root, slug)
	if !isDir(d) {
		fmt.Fprintf(stderr, "mutation-gate: no workspace at %s.\n", d)
		return 2
	}

	// Config files are looked up relative to the repo root, falling back to the
	// working directory when this is not a git repo.
	cwd, _ := os.Getwd()
	base, err := gitToplevel(cwd)
	if errors.Is(err, errNotGitRepository) {
		base = "."
	} else if err != nil {
		fmt.Fprintf(stderr, "mutation-gate: cannot resolve git worktree; using current directory: %v\n", err)
		base = "."
	}

	enforce := os.Getenv("DEVRITES_MUTATION")
	if enforce == "" {
		enforce = "advisory"
	}
	minScore := os.Getenv("DEVRITES_MUTATION_MIN")
	if minScore == "" {
		minScore = "60"
	}

	// Detect the mutation runner. A stryker config commits the project to stryker:
	// if its config is present but npx is not, the tool stays unset (no fall-through
	// to the other runners): the project chose stryker, it just is not installed.
	tool := ""
	if isFile(base+"/stryker.conf.json") || isFile(base+"/stryker.config.js") || isFile(base+"/stryker.config.mjs") {
		if _, err := exec.LookPath("npx"); err == nil {
			tool = "stryker"
		}
	} else if _, err := exec.LookPath("mutmut"); err == nil {
		tool = "mutmut"
	} else if _, err := exec.LookPath("cosmic-ray"); err == nil {
		tool = "cosmic-ray"
	} else if _, err := exec.LookPath("pitest"); err == nil || isFile(base+"/pom.xml") {
		tool = "pitest"
	}

	if tool == "" {
		fmt.Fprintln(stdout, "mutation-gate: no mutation runner detected (stryker / mutmut / cosmic-ray / pitest): advisory skipped.")
		fmt.Fprintln(stdout, "  Strengthen the criticals by hand-mutating (flip a comparison, drop a guard) and confirming a test goes red (standards/testing.md).")
		return 0
	}

	fmt.Fprintf(stdout, "mutation-gate: detected '%s'. Run it scoped to the slice's changed files and read the mutation score:\n", tool)
	switch tool {
	case "stryker":
		fmt.Fprintln(stdout, `  npx stryker run --mutate "$(git diff --name-only HEAD | paste -sd, -)"`)
	case "mutmut":
		fmt.Fprintln(stdout, `  mutmut run --paths-to-mutate "$(git diff --name-only HEAD | tr '\n' ' ')" && mutmut results`)
	case "cosmic-ray":
		fmt.Fprintln(stdout, "  (configure session over the changed files, then: cosmic-ray exec ... && cr-report ...)")
	case "pitest":
		fmt.Fprintln(stdout, "  mvn org.pitest:pitest-maven:mutationCoverage -DtargetClasses=<changed>")
	}
	fmt.Fprintln(stdout, "  Record the score + any survived mutants in evidence.md; a survivor is a behaviour no test checks (a finding for the test reviewer).")
	fmt.Fprintf(stdout, "  Mode: %s (DEVRITES_MUTATION=enforce makes a score < %s%% a NO-GO).\n", enforce, minScore)
	return 0
}

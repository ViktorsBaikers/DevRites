package gate

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/testutil"
)

const manifestTasks = "# Tasks\n\n## SLICE-001 Pin driver\nSatisfies: AC-001\nDependencies: none\nFiles likely touched: `go.mod`; `go.sum`; internal/store/db.go\n"

func preconditionProblems(t *testing.T, root, slug string) string {
	t.Helper()
	res, err := Check(Readiness, root, slug)
	if err != nil {
		t.Fatal(err)
	}
	var problems []string
	for _, problem := range res.StateProblems {
		if strings.HasPrefix(problem, "preconditions: ") {
			problems = append(problems, problem)
		}
	}
	if len(problems) > 0 && !res.Blocked {
		t.Fatalf("precondition problems must block: %v", problems)
	}
	return strings.Join(problems, "\n")
}

func TestCheckRequiresDependencyDryRunForManifestSliceAtVet(t *testing.T) {
	root := t.TempDir()
	writeCompleteGateFeature(t, root, "dep", state.PhaseVet, state.PhaseVet, "none\n")
	workspace := filepath.Join(root, "work", "dep")
	testutil.WriteFile(t, filepath.Join(workspace, "tasks.md"), manifestTasks)
	plan := "# Test plan\n\n## Build-entry preflight\n| Gate | Command | Cwd | Expected | Run at |\n|---|---|---|---|---|\n| unit | echo UNIT | . | exit 0 | action |\n"
	testutil.WriteFile(t, filepath.Join(workspace, "test-plan.md"), plan)

	if got := preconditionProblems(t, root, "dep"); !strings.Contains(got, "SLICE-001 touches go.mod") || !strings.Contains(got, "`dependency-dry-run SLICE-001`") {
		t.Fatalf("missing dry-run row must block, got %q", got)
	}

	plan += "| dependency-dry-run SLICE-001 | echo DRY-OK | . | exit 0 | vet |\n"
	testutil.WriteFile(t, filepath.Join(workspace, "test-plan.md"), plan)
	if got := preconditionProblems(t, root, "dep"); !strings.Contains(got, "without a passing transcript") {
		t.Fatalf("an unrun dry-run row must block, got %q", got)
	}

	if code := acceptance.RunPreconditions(root, []string{"dep"}, io.Discard, io.Discard); code != acceptance.ExitOK {
		t.Fatalf("check preconditions = %d", code)
	}
	if got := preconditionProblems(t, root, "dep"); got != "" {
		t.Fatalf("a passing dry-run transcript must clear the requirement, got %q", got)
	}
}

func TestCheckDependencyDryRunKeepsLegacyAndOtherPhases(t *testing.T) {
	root := t.TempDir()
	// A test plan without the Run at column predates the requirement.
	writeCompleteGateFeature(t, root, "legacy", state.PhaseVet, state.PhaseVet, "none\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "legacy", "tasks.md"), manifestTasks)
	testutil.WriteFile(t, filepath.Join(root, "work", "legacy", "test-plan.md"),
		"# Test plan\n\n## Build-entry preflight\n| Gate | Command | Cwd | Expected |\n|---|---|---|---|\n| unit | echo UNIT | . | exit 0 |\n")
	if got := preconditionProblems(t, root, "legacy"); got != "" {
		t.Fatalf("legacy test plan must keep today's behavior, got %q", got)
	}

	// Past Vet the requirement no longer applies: it gates leaving Vet only.
	writeCompleteGateFeature(t, root, "built", state.PhaseBuild, state.PhaseBuild, "none\n")
	testutil.WriteFile(t, filepath.Join(root, "work", "built", "tasks.md"), manifestTasks)
	testutil.WriteFile(t, filepath.Join(root, "work", "built", "test-plan.md"),
		"# Test plan\n\n## Build-entry preflight\n| Gate | Command | Cwd | Expected | Run at |\n|---|---|---|---|---|\n| unit | echo UNIT | . | exit 0 | action |\n")
	if got := preconditionProblems(t, root, "built"); got != "" {
		t.Fatalf("build-phase readiness must not re-require the Vet dry run, got %q", got)
	}
}

func TestTouchedManifestReadsPathFieldsOnly(t *testing.T) {
	cases := map[string]string{
		"Files likely touched: web/package.json, web/src/a.ts\n": "package.json",
		"Writer allowlist: `Cargo.lock`\n":                       "Cargo.lock",
		"Files likely touched: src/gomod.go; docs/go.mod.md\n":   "",
		"Notes: bump go.mod later\n":                             "",
	}
	for block, want := range cases {
		if got := touchedManifest(block); got != want {
			t.Fatalf("touchedManifest(%q) = %q, want %q", block, got, want)
		}
	}
}

package gate

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// DependencyManifests pairs each package manifest with its lockfiles; an edit
// to any of them changes dependency resolution.
var DependencyManifests = []struct {
	Manifest string
	Locks    []string
}{
	{"go.mod", []string{"go.sum"}},
	{"Cargo.toml", []string{"Cargo.lock"}},
	{"package.json", []string{"package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb"}},
	{"pyproject.toml", []string{"uv.lock", "poetry.lock"}},
	{"Pipfile", []string{"Pipfile.lock"}},
}

// dependencyManifests are the manifest and lockfile basenames whose edit
// changes dependency resolution, which only a dry run observes before Build.
var dependencyManifests = func() map[string]bool {
	names := map[string]bool{}
	for _, m := range DependencyManifests {
		names[m.Manifest] = true
		for _, lock := range m.Locks {
			names[lock] = true
		}
	}
	return names
}()

var slicePathFieldRE = regexp.MustCompile(`(?im)^\s*(?:[-*]\s+)?(?:files likely touched|writer allowlist):\s*(.*)$`)

// dependencyDryRunProblems blocks leaving Vet while a slice that edits a
// dependency manifest lacks a `dependency-dry-run <SLICE>` Build-entry
// preflight row with `Run at` `vet` whose newest transcript passed. Test plans
// without the `Run at` column predate the requirement and keep today's
// behavior.
func dependencyDryRunProblems(kind Kind, root string, observation *state.WorkspaceObservation) []string {
	if kind != Readiness {
		return nil
	}
	if phase, err := observation.DeclaredPhase(); err != nil || phase != state.PhaseVet {
		return nil
	}
	tasks, ok := observation.Fact("tasks.md")
	plan, planOK := observation.Fact("test-plan.md")
	if !ok || !planOK || tasks.State() != state.ArtifactPresent || plan.State() != state.ArtifactPresent {
		return nil // the missing-artifact path already blocks it
	}
	rows, hasRunAt := acceptance.PreflightRows(plan.Bytes())
	if !hasRunAt {
		return nil
	}
	slug := observation.Slug()
	work, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return []string{fmt.Sprintf("dependency dry-run transcripts cannot be read: %v", err)}
	}
	var problems []string
	for _, slice := range state.ParseTaskGraph(tasks.Bytes()).Slices {
		block, _ := state.ExtractTaskSlice(tasks.Bytes(), slice.ID)
		manifest := touchedManifest(block)
		if manifest == "" {
			continue
		}
		matching := dryRunRows(rows, slice.ID)
		switch {
		case len(matching) == 0:
			problems = append(problems, fmt.Sprintf(
				"%s touches %s but test-plan.md has no `dependency-dry-run %s` Build-entry preflight row with Run at `vet`; add one that applies the planned change in an isolated copy, runs the tidy/lock step, then the project's admission checks",
				slice.ID, manifest, slice.ID))
		case !allPassed(work, matching):
			problems = append(problems, fmt.Sprintf(
				"%s dependency dry run has a row without a passing transcript; run `devrites-engine check preconditions %s` and route a failure to Plan repair",
				slice.ID, slug))
		}
	}
	return problems
}

func touchedManifest(block string) string {
	for _, match := range slicePathFieldRE.FindAllStringSubmatch(block, -1) {
		for _, part := range strings.FieldsFunc(match[1], func(r rune) bool { return r == ';' || r == ',' }) {
			if name := path.Base(strings.Trim(strings.TrimSpace(part), "`'\"")); dependencyManifests[name] {
				return name
			}
		}
	}
	return ""
}

func dryRunRows(rows []acceptance.PreflightRow, sliceID string) []acceptance.PreflightRow {
	var matching []acceptance.PreflightRow
	for _, row := range rows {
		if strings.EqualFold(row.RunAt, "vet") && strings.Contains(row.Gate, "dependency-dry-run") &&
			containsToken(row.Gate, sliceID) {
			matching = append(matching, row)
		}
	}
	return matching
}

func allPassed(work string, rows []acceptance.PreflightRow) bool {
	for _, row := range rows {
		if !acceptance.PreflightPassed(work, row) {
			return false
		}
	}
	return true
}

func containsToken(text, token string) bool {
	for _, field := range strings.FieldsFunc(text, func(r rune) bool { return r == ' ' || r == ',' || r == ';' }) {
		if field == token {
			return true
		}
	}
	return false
}

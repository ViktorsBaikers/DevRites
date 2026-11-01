package gate

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func TestAuditAcceptanceReadinessChecksDeclaredCriteria(t *testing.T) {
	const criteria = `### Requirement: bounded acceptance

#### Scenario: first outcome
- [ ] AC-001: WHEN first input arrives THEN first outcome is recorded.

#### Scenario: middle outcome
- [ ] AC-002: WHEN middle input arrives THEN middle outcome is recorded.

#### Scenario: last outcome
- [ ] AC-003: WHEN last input arrives THEN last outcome is recorded.
`
	const endpoints = "AC-001 AC-003"
	const all = "AC-001 AC-002 AC-003"
	for _, tc := range []struct {
		name    string
		spec    string
		mapped  string
		blocked bool
	}{
		{
			name:    "declarations before trailing summary cannot hide middle criterion",
			spec:    "# Spec\n\n" + criteria + "\n## Acceptance criteria\nAC-001 through AC-003\n",
			mapped:  endpoints,
			blocked: true,
		},
		{
			name:    "fully mapped declarations still require canonical placement",
			spec:    "# Spec\n\n" + criteria + "\n## Acceptance criteria\nAC-001 through AC-003\n",
			mapped:  all,
			blocked: true,
		},
		{
			name:    "declarations after next H2 cannot hide middle criterion",
			spec:    "# Spec\n\n## Acceptance criteria\nAC-001 through AC-003\n\n## Notes\n" + criteria,
			mapped:  endpoints,
			blocked: true,
		},
		{
			name:    "indented H2 ends canonical section",
			spec:    "# Spec\n\n## Acceptance criteria\nAC-001 through AC-003\n\n  ## Notes\n" + criteria,
			mapped:  all,
			blocked: true,
		},
		{
			name:    "H1 ends canonical section",
			spec:    "# Spec\n\n## Acceptance criteria\nAC-001 through AC-003\n\n# Appendix\n" + criteria,
			mapped:  all,
			blocked: true,
		},
		{
			name:   "canonical heading before all criterion bodies",
			spec:   "# Spec\n\n## Acceptance criteria\n" + criteria,
			mapped: all,
		},
		{
			name:    "canonical middle criterion needs planning references",
			spec:    "# Spec\n\n## Acceptance criteria\n" + criteria,
			mapped:  endpoints,
			blocked: true,
		},
		{
			name:   "ordinary cross references outside canonical section",
			spec:   "# Spec\n\n## Context\nPrior AC-099 informs AC-002; see AC-001.\n\n## Acceptance criteria\n" + criteria + "\n## Notes\nSee AC-098 for related work.\n",
			mapped: all,
		},
		{
			name:   "fenced example definitions outside canonical section",
			spec:   "# Spec\n\n## Examples\n```markdown\n### Requirement: example only\n#### Scenario: illustrative\n- [ ] AC-099: WHEN example runs THEN no real obligation is declared.\n```\n\n## Acceptance criteria\n" + criteria,
			mapped: all,
		},
		{
			name:   "commented definitions outside canonical section",
			spec:   "# Spec\n\n<!--\n" + criteria + "-->\n\n## Acceptance criteria\n" + criteria,
			mapped: all,
		},
		{
			name:   "commented canonical heading and invalid ids are not obligations",
			spec:   "# Spec\n\n<!--\n## Acceptance criteria\n- [ ] AC-99: example only\n-->\n\n## Acceptance criteria\n" + criteria,
			mapped: all,
		},
		{
			name:   "unclosed example fence inside comment cannot mask real obligations",
			spec:   "# Spec\n\n<!--\n```markdown\n" + criteria + "-->\n\n## Acceptance criteria\n" + criteria,
			mapped: all,
		},
		{
			name:    "real criterion after commented fence still needs mapping",
			spec:    "# Spec\n\n<!--\n```markdown\n" + criteria + "-->\n\n## Acceptance criteria\n" + criteria,
			mapped:  endpoints,
			blocked: true,
		},
		{
			name:    "visible criterion after inline comment needs mapping",
			spec:    "# Spec\n\n## Acceptance criteria\nAC-001: mapped\n<!-- editorial --> AC-002: required\n",
			mapped:  "AC-001",
			blocked: true,
		},
		{
			name:    "comment opener inside fence cannot hide actual preheading declarations",
			spec:    "# Spec\n\n```markdown\n<!--\n```\n\n" + criteria + "\n## Acceptance criteria\nAC-001 through AC-003\n",
			mapped:  all,
			blocked: true,
		},
		{
			name:    "tilde fence info comment cannot hide required mapping",
			spec:    "# Spec\n\n~~~html <!--\n## Acceptance criteria\nAC-099: example only\n~~~\n\n## Acceptance criteria\n" + criteria,
			mapped:  endpoints,
			blocked: true,
		},
		{
			name:   "tilde fence info comment keeps examples hidden",
			spec:   "# Spec\n\n~~~html <!--\n## Acceptance criteria\nAC-099: example only\n~~~\n\n## Acceptance criteria\n" + criteria,
			mapped: all,
		},
		{
			name:   "bullet cross references outside canonical section",
			spec:   "# Spec\n\n## Context\n- See AC-099: earlier related work.\n- AC-098 informs current scope.\n\n## Acceptance criteria\n" + criteria,
			mapped: all,
		},
		{
			name:   "range prose does not invent middle criteria",
			spec:   "# Spec\n\n## Acceptance criteria\nAC-001 through AC-003\n",
			mapped: endpoints,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			const slug = "audit-acceptance"
			workspace := writeReadinessFixture(t, root, slug, "build")
			testutil.WriteFile(t, filepath.Join(workspace, "spec.md"), tc.spec)
			testutil.AppendFile(t, filepath.Join(workspace, "tasks.md"), "\n"+tc.mapped+"\n")
			testutil.AppendFile(t, filepath.Join(workspace, "test-plan.md"), "\n"+tc.mapped+"\n")
			testutil.AppendFile(t, filepath.Join(workspace, "eng-review.md"), "\n"+mustReadinessBinding(t, root, slug)+"\n")

			result, err := Check(Readiness, root, slug)
			if err != nil {
				t.Fatal(err)
			}
			if result.Blocked != tc.blocked {
				t.Fatalf("readiness blocked=%v, want %v; result=%+v", result.Blocked, tc.blocked, result)
			}
			if tc.blocked && !strings.Contains(strings.Join(result.StateProblems, "\n"), "acceptance-map:") {
				t.Fatalf("readiness must block on acceptance map, got %+v", result)
			}
		})
	}
}

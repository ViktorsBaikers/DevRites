package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/testutil"
)

func workflowAdmissionFixture(slug string) string {
	text := "## Workflow Artifact admission\nDevRites contract: devrites.workflow-artifact-admission.v1\n\n| Field | Value |\n| --- | --- |\n"
	values := []string{slug, "devrites-engine check readiness --emit-binding " + slug, "vet", "/rite-vet " + slug, "utf8-bytewise-path-ascending", "1", "4096", "4096", "9", "256", "34", "3", "30", "60", "2"}
	for i, key := range workflowAdmissionFields {
		text += fmt.Sprintf("| %s | `%s` |\n", key, values[i])
	}
	text += "\n| " + strings.Join(workflowTargetColumns, " | ") + " |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n"
	text += "| `00000000` | `proof/check.sh` | `0700` | `WA-BEH-CHECK` | `WA-IF-CHECK` | `WA-FIX-PCHECK` | `WA-FIX-FCHECK` | `sh proof/check.sh` | `active-workspace` | `PASS` | `restore-preimage-or-absence` | `status,reason` |\n"
	for _, ref := range []struct{ id, kind, keys string }{
		{"WA-BEH-CHECK", "behavior", "success observable_effect"},
		{"WA-IF-CHECK", "interface", "inputs invariants ordering errors configuration performance"},
		{"WA-FIX-PCHECK", "positive-fixture", "setup action expected"},
		{"WA-FIX-FCHECK", "failure-fixture", "setup fault expected"},
	} {
		text += "\n## " + ref.id + "\nDevRites workflow reference: " + ref.kind + "\n\n| Field | Value |\n| --- | --- |\n"
		for _, key := range strings.Fields(ref.keys) {
			text += "| " + key + " | `Concrete fixture assertion` |\n"
		}
	}
	return text
}

func TestWorkflowAdmissionStructure(t *testing.T) {
	good := workflowAdmissionFixture("diagnostic")
	if _, err := workflowAdmission([]byte(good), "diagnostic"); err != nil {
		t.Fatal(err)
	}
	tighterAggregate := strings.Replace(good, "| aggregate_bytes_limit | `4096` |", "| aggregate_bytes_limit | `2048` |", 1)
	if _, err := workflowAdmission([]byte(tighterAggregate), "diagnostic"); err != nil {
		t.Fatalf("independent tighter aggregate cap rejected: %v", err)
	}
	reviewed := strings.Replace(good, "\n## WA-BEH-CHECK", "\nWorkflow Artifact reviewed identity: "+strings.Repeat("a", 64)+"\n\n## WA-BEH-CHECK", 1)
	if _, err := workflowAdmission([]byte(reviewed), "diagnostic"); err != nil {
		t.Fatalf("reviewed identity rejected: %v", err)
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"comparison", "`Concrete fixture assertion`", "`count > 0`"},
		{"less than comparison", "`Concrete fixture assertion`", "`count < 10`"},
		{"single quoted pipe", "`sh proof/check.sh`", "`printf 'PASS\\|CHECK\\n'`"},
		{"double quoted pipe", "`sh proof/check.sh`", "`printf \"PASS\\|CHECK\\n\"`"},
		{"escaped list data", "`sh proof/check.sh`", "`printf PASS\\;\\&CHECK`"},
		{"quoted list data", "`sh proof/check.sh`", "`printf '; && \\|\\|'`"},
		{"simple substitution", "`sh proof/check.sh`", "`printf \"$(pwd)\"`"},
		{"literal substitution", "`sh proof/check.sh`", "`printf '$(printf x \\| cat)'`"},
		{"return action", "`/rite-vet diagnostic`", "`/rite-plan diagnostic`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := workflowAdmission([]byte(strings.Replace(good, tc.old, tc.replacement, 1)), "diagnostic"); err != nil {
				t.Fatalf("valid admission rejected: %v", err)
			}
		})
	}
	_, cells, ok := workflowTableCells(`| one\|two | three\\| four |`)
	if !ok || len(cells) != 3 || cells[0] != " one|two " || cells[1] != ` three\\` {
		t.Fatalf("admission escaping: %q %v", cells, ok)
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"wrong workspace", "`diagnostic`", "`other`"},
		{"traversal", "`proof/check.sh`", "`../other/check.sh`"},
		{"workspace ledger", "`proof/check.sh`", "`evidence.md`"},
		{"workspace absolute", "`proof/check.sh`", "`.devrites/work/other/check.sh`"},
		{"bounds", "`4096`", "`0`"},
		{"overflow", "`4096`", "`18446744073709551616`"},
		{"file minimum", "`9`", "`8`"},
		{"journal minimum", "`34`", "`33`"},
		{"missing reference", "## WA-IF-CHECK", "## WA-IF-OTHER"},
		{"todo", "`Concrete fixture assertion`", "`TODO`"},
		{"tbd", "`Concrete fixture assertion`", "`tBd`"},
		{"placeholder", "`Concrete fixture assertion`", "`PlAcEhOlDeR`"},
		{"angle placeholder", "`Concrete fixture assertion`", "`<assertion>`"},
		{"empty cell", "`Concrete fixture assertion`", "` `"},
		{"nested backticks", "`Concrete fixture assertion`", "``assertion``"},
		{"multiline cell", "`Concrete fixture assertion`", "`first\nsecond`"},
		{"shell list", "`sh proof/check.sh`", "`sh proof/check.sh; true`"},
		{"pipeline", "`sh proof/check.sh`", "`sh proof/check.sh \\| cat`"},
		{"escaped backslash then pipeline", "`sh proof/check.sh`", "`printf PASS\\\\\\| cat`"},
		{"or list", "`sh proof/check.sh`", "`sh proof/check.sh \\|\\| true`"},
		{"and list", "`sh proof/check.sh`", "`sh proof/check.sh && true`"},
		{"background list", "`sh proof/check.sh`", "`sh proof/check.sh &`"},
		{"substitution pipeline", "`sh proof/check.sh`", "`printf \"$(printf x \\| cat)\"`"},
		{"substitution literal operator", "`sh proof/check.sh`", "`printf \"$(pwd)\\|CHECK\"`"},
		{"quoted argument then pipeline", "`sh proof/check.sh`", "`printf 'PASS\\|CHECK' \\| cat`"},
		{"unclosed quote", "`sh proof/check.sh`", "`printf 'PASS\\|CHECK`"},
		{"trailing escape", "`sh proof/check.sh`", "`printf PASS\\`"},
		{"missing action verb", "`/rite-vet diagnostic`", "`/ anything diagnostic`"},
		{"unsupported action shape", "`/rite-vet diagnostic`", "`/other diagnostic`"},
		{"extra action argument", "`/rite-vet diagnostic`", "`/rite-vet extra diagnostic`"},
		{"wrong action slug", "`/rite-vet diagnostic`", "`/rite-vet other`"},
		{"action extra space", "`/rite-vet diagnostic`", "`/rite-vet  diagnostic`"},
		{"cwd", "`active-workspace`", "`../other`"},
		{"mode", "`0700`", "`777`"},
		{"index", "`00000000`", "`00000001`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := workflowAdmission([]byte(strings.Replace(good, tc.old, tc.replacement, 1)), "diagnostic"); err == nil {
				t.Fatal("malformed admission accepted")
			}
		})
	}
	if _, err := workflowAdmission(nil, "diagnostic"); err == nil {
		t.Fatal("missing admission accepted")
	}
}

func TestWorkflowOnlyReadinessNonActive(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	slug := "diagnostic"
	workspace := writeReadinessFixture(t, root, slug, "vet")
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "other\n")
	testutil.WriteFile(t, filepath.Join(workspace, "spec.md"), "# Spec\n\n## Acceptance criteria\n- AC-001: check works.\n")
	mapping := "\n| AC | Workflow Artifact behavior |\n| --- | --- |\n| AC-001 | WA-BEH-CHECK |\n"
	testutil.WriteFile(t, filepath.Join(workspace, "tasks.md"), "# Tasks\n\nWork scope: workflow-only\n"+mapping)
	testutil.WriteFile(t, filepath.Join(workspace, "test-plan.md"), "# Test plan\n"+mapping+workflowAdmissionFixture(slug))
	binding := mustReadinessBinding(t, root, slug)
	testutil.WriteFile(t, filepath.Join(workspace, "eng-review.md"), "# Engineering review\n\nImplementation readiness: READY\n"+binding+"\n")
	before, err := os.ReadFile(filepath.Join(workspace, "state.md"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Check(Readiness, root, slug)
	if err != nil || result.Blocked {
		t.Fatalf("valid workflow blocked: %+v %v", result, err)
	}
	testutil.WriteFile(t, filepath.Join(workspace, "test-plan.md"), "# Test plan\n"+mapping+strings.Replace(workflowAdmissionFixture(slug), "Concrete fixture assertion", "Changed assertion", 1))
	result, err = Check(Readiness, root, slug)
	if err != nil || !result.Blocked || result.ReasonID != reason.GateReadinessStale {
		t.Fatalf("stale admission: %+v %v", result, err)
	}
	testutil.WriteFile(t, filepath.Join(workspace, "test-plan.md"), "# Test plan\n"+mapping)
	binding = mustReadinessBinding(t, root, slug)
	testutil.WriteFile(t, filepath.Join(workspace, "eng-review.md"), "# Engineering review\n\nImplementation readiness: READY\n"+binding+"\n")
	result, err = Check(Readiness, root, slug)
	if err != nil || !result.Blocked || !strings.Contains(strings.Join(result.StateProblems, " "), "admission") {
		t.Fatalf("missing admission: %+v %v", result, err)
	}
	after, _ := os.ReadFile(filepath.Join(workspace, "state.md"))
	active, _ := os.ReadFile(filepath.Join(root, "ACTIVE"))
	if string(before) != string(after) || string(active) != "other\n" {
		t.Fatal("readiness modified state/built counts or ACTIVE")
	}
}

func TestWorkflowOnlyReadinessRejectsEveryUnadmittedMapping(t *testing.T) {
	for _, name := range []string{"tasks.md", "test-plan.md"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			slug := "diagnostic"
			workspace := writeReadinessFixture(t, root, slug, "vet")
			testutil.WriteFile(t, filepath.Join(workspace, "spec.md"), "# Spec\n\n## Acceptance criteria\n- AC-001: check works.\n")
			mapping := "\n| AC | Workflow Artifact behavior |\n| --- | --- |\n| AC-001 | WA-BEH-CHECK |\n"
			for artifact, body := range map[string]string{
				"tasks.md":     "# Tasks\n\nWork scope: workflow-only\n" + mapping,
				"test-plan.md": "# Test plan\n" + mapping + workflowAdmissionFixture(slug),
			} {
				if artifact == name {
					body = strings.Replace(body, "| AC-001 | WA-BEH-CHECK |", "| AC-001 | WA-BEH-CHECK |\n| AC-001 | WA-BEH-NOT-ADMITTED |", 1)
				}
				testutil.WriteFile(t, filepath.Join(workspace, artifact), body)
			}
			binding := mustReadinessBinding(t, root, slug)
			testutil.WriteFile(t, filepath.Join(workspace, "eng-review.md"), "# Engineering review\n\nImplementation readiness: READY\n"+binding+"\n")
			result, err := Check(Readiness, root, slug)
			if err != nil {
				t.Fatal(err)
			}
			diagnostic := name + " maps AC-001 to invalid or unadmitted Workflow Artifact behavior \"WA-BEH-NOT-ADMITTED\""
			if !result.Blocked || result.ReasonID == reason.GateReadinessStale || !strings.Contains(strings.Join(result.StateProblems, "\n"), diagnostic) {
				t.Fatalf("extra unadmitted mapping accepted with fresh binding: %+v", result)
			}
		})
	}
}

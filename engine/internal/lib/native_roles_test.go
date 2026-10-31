package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextNativeRoleLabels(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	feature := filepath.Join(root, "work", "feat")
	skills := filepath.Join(project, ".omp", "skills")
	writeFile(t, filepath.Join(feature, "state.md"), "# State\n")
	writeFile(t, filepath.Join(feature, "spec.md"), "# Spec\nBilling export.\n")
	writeFile(t, filepath.Join(feature, "plan.md"), "# Plan\nEXCLUDED_PHASE_DEFAULT\n")
	writeFile(t, filepath.Join(feature, "tasks.md"), "## SLICE-001 Export\nGoal: export billing rows\n")
	writeFile(t, filepath.Join(skills, "devrites-lib/reference/standards/core.md"), "# Core\n\n## Models\n<!-- model-surface -->\nMODEL_SAFETY_RULE\n<!-- /model-surface -->\n")
	writeFile(t, filepath.Join(skills, "devrites-lib/reference/standards/coding-style.md"), "# Coding\n")
	writeFile(t, filepath.Join(skills, "rite-build/SKILL.md"), `<!-- loads: {"always":["devrites-lib/reference/standards/core.md"],"triggers":{"coding":["devrites-lib/reference/standards/coding-style.md"]},"workspace":["plan.md"],"workspaceByRole":{"code-reviewer":["spec.md"],"security-auditor":["spec.md"],"slice-wright":["tasks.md"]}} -->`+"\n# Build\n")
	for _, role := range []string{"code-reviewer", "security-auditor", "slice-wright"} {
		writeFile(t, filepath.Join(project, ".omp/agents/devrites-"+role+".md"), "# Specialist\n")
	}
	for _, tc := range []struct {
		label, base, artifact string
	}{
		{"code-reviewer.frontend", "code-reviewer", "spec.md"},
		{"devrites-code-reviewer.backend", "code-reviewer", "spec.md"},
		{"devrites-security-auditor.backend", "security-auditor", "spec.md"},
		{"devrites-slice-wright", "slice-wright", "tasks.md"},
		{"slice-wright.SLICE-001", "slice-wright", "tasks.md"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			var out, stderr bytes.Buffer
			args := []string{"feat", "--phase", "build", "--role", tc.label, "--skills-root", skills}
			name := "build-" + tc.label
			if tc.base != "slice-wright" {
				args = append(args, "--slice", "SLICE-001")
				name += ".SLICE-001"
			}
			if code := RunContext(root, args, &out, &stderr); code != 0 {
				t.Fatalf("context code=%d: %s%s", code, &out, &stderr)
			}
			data, err := os.ReadFile(filepath.Join(feature, "ctx", name+".bundle.md"))
			if err != nil {
				t.Fatalf("label-specific bundle missing: %v", err)
			}
			text := string(data)
			if !strings.Contains(text, "agents/devrites-"+tc.base+".md") || !strings.Contains(text, "workspace/"+tc.artifact) || strings.Contains(text, "EXCLUDED_PHASE_DEFAULT") {
				t.Fatalf("wrong specialist or workspace scope:\n%s", text)
			}
			if got := strings.Contains(text, "MODEL_SAFETY_RULE"); got != (tc.base == "security-auditor") {
				t.Fatalf("model safety inclusion=%v for %s", got, tc.label)
			}
			if tc.base == "slice-wright" && !strings.Contains(out.String(), "coding") {
				t.Fatalf("writer lost coding trigger suggestion: %s", &out)
			}
		})
	}
}

func TestDispatchReviewerRequiresClassification(t *testing.T) {
	for _, role := range []string{"code-reviewer", "devrites-code-reviewer.backend", "test-analyst", "security-auditor"} {
		t.Run(role, func(t *testing.T) {
			root, feature := newWorkspace(t, "build")
			for _, args := range [][]string{
				{"open", "--phase", "build", "--wave", "recheck", "--role", role, "--packet", reviewerPacketPath(t)},
				{"start", "--wave", "recheck", "--role", role, "--handle", "native-child"},
			} {
				if code, out := runDispatchT(t, root, args...); code != 0 {
					t.Fatalf("%v: code=%d %s", args, code, out)
				}
			}
			journal := filepath.Join(feature, "dispatch.json")
			before, err := os.ReadFile(journal)
			if err != nil {
				t.Fatal(err)
			}
			if code, out := runDispatchT(t, root, "return", "--wave", "recheck", "--role", role); code != 3 {
				t.Fatalf("unclassified reviewer return accepted: code=%d %s", code, out)
			}
			after, err := os.ReadFile(journal)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("refused return changed journal: %v", err)
			}
			if code, out := runDispatchT(t, root, "status", "--wave", "recheck"); code != 3 || !strings.Contains(out, role+"=started") {
				t.Fatalf("reviewer no longer pending: code=%d %s", code, out)
			}
			if code, out := runDispatchT(t, root, "return", "--wave", "recheck", "--role", role, "--outcome", "gap"); code != 0 {
				t.Fatalf("explicit gap refused: code=%d %s", code, out)
			}
			if code, out := runDispatchT(t, root, "status", "--wave", "recheck"); code != 0 || !strings.Contains(out, "complete") {
				t.Fatalf("classified wave not complete: code=%d %s", code, out)
			}
		})
	}
}

package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func consumedPlanningFixture(t *testing.T) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	dirs := map[string]string{}
	parent := ""
	for _, slug := range []string{"a", "b", "c"} {
		dirs[slug] = writeReadinessFixture(t, root, slug, "build")
		testutil.AppendFile(t, filepath.Join(dirs[slug], "state.md"), "| sequence_parent | "+parent+" |\n")
		parent = slug
	}
	dirs["other"] = writeReadinessFixture(t, root, "other", "build")
	return root, dirs
}

func consumedPlanningVet(t *testing.T, root, declaration, bindings string) {
	t.Helper()
	dir := filepath.Join(root, "work", "c")
	testutil.WriteFile(t, filepath.Join(dir, "plan.md"), "# Plan\n\n"+declaration+"\n")
	testutil.WriteFile(t, filepath.Join(dir, "eng-review.md"), "# Engineering review\n\nREADY\n\n"+bindings+"\n"+mustReadinessBinding(t, root, "c")+"\n")
}

func consumedBinding(t *testing.T, root, slug string) string {
	t.Helper()
	return predecessorLabel + slug + " " + strings.TrimPrefix(mustReadinessBinding(t, root, slug), readinessBindingLabel)
}

func TestConsumedPlanningInputsExactSetReadiness(t *testing.T) {
	for _, tc := range []struct {
		name        string
		declaration string
		bindings    []string
		want        string
	}{
		{"nonadjacent", "Consumed planning inputs: a", []string{"a"}, ""},
		{"multiple", "Consumed planning inputs: a, b", []string{"b", "a"}, ""},
		{"none", "Consumed planning inputs: none", []string{"none"}, ""},
		{"missing binding", "Consumed planning inputs: a, b", []string{"a"}, "missing Predecessor readiness"},
		{"extra binding", "Consumed planning inputs: a", []string{"a", "b"}, "extra Predecessor readiness"},
		{"duplicate binding", "Consumed planning inputs: a", []string{"a", "a"}, "duplicate Predecessor readiness"},
		{"duplicate producer", "Consumed planning inputs: a, a", []string{"a"}, "duplicate Consumed planning inputs"},
		{"duplicate declaration", "Consumed planning inputs: a\nConsumed planning inputs: b", []string{"a"}, "more than one"},
		{"self", "Consumed planning inputs: c", nil, "workspace itself"},
		{"unrelated", "Consumed planning inputs: other", []string{"other"}, "unrelated"},
		{"unsafe", "Consumed planning inputs: ../a", nil, "malformed"},
		{"absolute", "Consumed planning inputs: /a", nil, "malformed"},
		{"missing producer", "Consumed planning inputs: missing", nil, "unavailable"},
		{"blank", "Consumed planning inputs: ", nil, "malformed"},
		{"space separated", "Consumed planning inputs: a b", nil, "malformed"},
		{"empty member", "Consumed planning inputs: a,", nil, "malformed"},
		{"mixed none", "Consumed planning inputs: none, a", nil, "malformed"},
		{"none missing", "Consumed planning inputs: none", nil, "exactly"},
		{"none extra", "Consumed planning inputs: none", []string{"none", "a"}, "exactly"},
		{"none for producer", "Consumed planning inputs: a", []string{"none"}, "malformed"},
		{"bad digest", "Consumed planning inputs: a", []string{"bad"}, "malformed"},
		{"missing colon", "Consumed planning inputs a", nil, "malformed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := consumedPlanningFixture(t)
			var bindings []string
			for _, slug := range tc.bindings {
				switch slug {
				case "none":
					bindings = append(bindings, predecessorLabel+"none")
				case "bad":
					bindings = append(bindings, predecessorLabel+"a not-a-digest")
				default:
					bindings = append(bindings, consumedBinding(t, root, slug))
				}
			}
			consumedPlanningVet(t, root, tc.declaration, strings.Join(bindings, "\n"))
			result, err := Check(Readiness, root, "c")
			if err != nil {
				t.Fatal(err)
			}
			if result.Blocked != (tc.want != "") || tc.want != "" && !strings.Contains(result.Render(), tc.want) {
				t.Fatalf("blocked=%v want=%q: %s", result.Blocked, tc.want, result.Render())
			}
		})
	}
}

func TestConsumedPlanningInputsOnlyConsumedDriftInvalidates(t *testing.T) {
	root, dirs := consumedPlanningFixture(t)
	consumedPlanningVet(t, root, "Consumed planning inputs: a", consumedBinding(t, root, "a"))
	testutil.AppendFile(t, filepath.Join(dirs["b"], "plan.md"), "\nUnconsumed adjacent producer changed.\n")
	result, err := Check(Readiness, root, "c")
	if err != nil || result.Blocked {
		t.Fatalf("unconsumed B drift blocked C: result=%v err=%v", result, err)
	}
	archive := filepath.Join(root, "archive", "a")
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(dirs["a"], archive); err != nil {
		t.Fatal(err)
	}
	result, err = Check(Readiness, root, "c")
	if err != nil || result.Blocked {
		t.Fatalf("unchanged archived A blocked C: result=%v err=%v", result, err)
	}
	testutil.AppendFile(t, filepath.Join(archive, "plan.md"), "\nConsumed A changed.\n")
	result, err = Check(Readiness, root, "c")
	if err != nil || !result.Blocked || !strings.Contains(result.Render(), "predecessor a changed") {
		t.Fatalf("consumed A drift not reported: result=%v err=%v", result, err)
	}
}

func TestConsumedPlanningInputsRetainConsumerObservation(t *testing.T) {
	root, dirs := consumedPlanningFixture(t)
	consumedPlanningVet(t, root, "Consumed planning inputs: a", consumedBinding(t, root, "a"))
	observation := mustObserve(t, root, "c")
	testutil.WriteFile(t, filepath.Join(dirs["c"], "plan.md"), "# Plan\n\nConsumed planning inputs: none\n")
	testutil.AppendFile(t, filepath.Join(dirs["c"], "state.md"), "| sequence_parent | other |\n")
	if problems := ConsumedPlanningInputProblems(root, observation, true); len(problems) != 0 {
		t.Fatalf("retained consumer observation reread changed artifacts: %v", problems)
	}
	if problems := ConsumedPlanningInputProblems(root, mustObserve(t, root, "c"), true); len(problems) == 0 {
		t.Fatal("fresh consumer observation ignored changed planning declaration")
	}
}

func TestConsumedPlanningInputsRequireStructuralDeclaration(t *testing.T) {
	root, _ := consumedPlanningFixture(t)
	for _, declaration := range []string{"Ready.", "```text\nConsumed planning inputs: none\n```", "<!-- Consumed planning inputs: none -->"} {
		consumedPlanningVet(t, root, declaration, "")
		observation := mustObserve(t, root, "c")
		if problems := ConsumedPlanningInputProblems(root, observation, false); len(problems) != 0 {
			t.Fatalf("legacy absence rejected: %v", problems)
		}
		if problems := ConsumedPlanningInputProblems(root, observation, true); len(problems) == 0 || !strings.Contains(problems[0], "missing Consumed planning inputs") {
			t.Fatalf("canonical absence accepted: %v", problems)
		}
	}
}

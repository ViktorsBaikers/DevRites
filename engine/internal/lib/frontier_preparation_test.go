package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func preparationContext(t *testing.T, root, slug, phase, role string, alias bool, extra ...string) (int, string) {
	t.Helper()
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-"+phase, "SKILL.md"), "---\nname: rite-"+phase+"\n---\n<!-- loads: {\"workspace\":[\"state.md\"]} -->\n# Native phase\n")
	if role != "" {
		writeFile(t, filepath.Join(skills, "..", "agents", "devrites-"+role+".md"), "# Native role\n")
	}
	flag, value := "--phase", phase
	if alias {
		flag, value = "--skill", "rite-"+phase
	}
	args := []string{slug, flag, value, "--skills-root", skills}
	if role != "" {
		args = append(args, "--role", role)
	}
	var out, errOut bytes.Buffer
	code := RunContext(root, append(args, extra...), &out, &errOut)
	return code, out.String() + errOut.String()
}

func preparationOpen(t *testing.T, root, slug, phase, role, wave string, extra ...string) (int, string) {
	t.Helper()
	args := []string{"open", "--phase", phase, "--role", role, "--wave", wave}
	if isReviewerRole(role) {
		args = append(args, "--packet", reviewerPacketPath(t))
	}
	return controllerDispatch(t, root, slug, append(args, extra...)...)
}

func TestAutocompletePreparationEarlyPhasesRefuseBeforeWrites(t *testing.T) {
	for _, phase := range []struct{ phase, role string }{
		{"temper", "strategy-reviewer"}, {"define", "plan-drafter"},
		{"plan", "plan-drafter"}, {"vet", "plan-reviewer"},
	} {
		for _, successor := range []string{"missing", "unstarted"} {
			for _, surface := range []string{"dispatch", "context phase", "context skill"} {
				t.Run(phase.phase+"/"+successor+"/"+surface, func(t *testing.T) {
					root, _ := controllerFixture(t)
					if successor == "unstarted" {
						frontierFixture(t, root, "work", "future17", "current16", "spec", "running", "/rite-spec future17", 2, "1")
					}
					before := controllerSnapshot(t, root)
					var code int
					var out string
					if surface == "dispatch" {
						code, out = preparationOpen(t, root, "current16", phase.phase, phase.role, "early")
					} else {
						code, out = preparationContext(t, root, "current16", phase.phase, phase.role, surface == "context skill")
					}
					if code != 3 || !strings.Contains(out, "future17") || !strings.Contains(out, "BLOCKED") {
						t.Fatalf("ordinary forward phase skipped successor: %d\n%s", code, out)
					}
					if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
						t.Fatal("refusal wrote bundle, account, metrics, claims, rounds or history")
					}
				})
			}
		}
	}
}

func TestAutocompletePreparationMarkedSuccessorBoundsPipeline(t *testing.T) {
	root, _ := controllerFixture(t)
	child := frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "1")
	if code, out := preparationContext(t, root, "future17", "plan", "plan-drafter", false, "--prepare-for", "current16"); code != 0 {
		t.Fatalf("preparation context recursively demanded future18: %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(child, "dispatch.json")); !os.IsNotExist(err) {
		t.Fatalf("context created dispatch account: %v", err)
	}
	if code, out := preparationOpen(t, root, "future17", "plan", "plan-drafter", "prepare", "--prepare-for", "current16"); code != 0 {
		t.Fatalf("marked successor demanded its successor: %d\n%s", code, out)
	}
	var account struct {
		Waves map[string]map[string]any `json:"waves"`
	}
	if err := json.Unmarshal([]byte(readTestFile(t, filepath.Join(child, "dispatch.json"))), &account); err != nil {
		t.Fatal(err)
	}
	if account.Waves["prepare"]["preparation_for"] != "current16" {
		t.Fatalf("missing durable parent binding: %#v", account.Waves["prepare"])
	}
	if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "prepare", "--role", "plan-drafter", "--handle", "native-preparation"); code != 0 {
		t.Fatalf("start preparation: %d\n%s", code, out)
	}
	for _, alias := range []bool{false, true} {
		if code, out := preparationContext(t, root, "current16", "temper", "strategy-reviewer", alias); code != 0 {
			t.Fatalf("pending immediate preparation did not admit parent context: %d %s", code, out)
		}
	}
	if code, out := preparationOpen(t, root, "current16", "temper", "strategy-reviewer", "forward"); code != 0 {
		t.Fatalf("pending immediate preparation did not admit parent: %d\n%s", code, out)
	}
	for _, observe := range []string{"frontier", "next", "continuation"} {
		t.Run(observe, func(t *testing.T) {
			before := controllerSnapshot(t, root)
			var out, errOut bytes.Buffer
			if observe == "frontier" {
				lines, err := FrontierLines(root, "current16")
				if err != nil {
					t.Fatal(err)
				}
				out.WriteString(strings.Join(lines, "\n"))
			} else {
				run := RunNext
				if observe == "continuation" {
					run = RunCheckContinuation
				}
				if code := run(root, []string{"current16"}, &out, &errOut); code != 0 {
					t.Fatalf("observe: %d %s", code, &errOut)
				}
			}
			if strings.Contains(out.String(), "frontier-prepare: /rite-spec 'future18'") || strings.Contains(out.String(), "future18 location=missing action=prepare") {
				t.Fatalf("preparatory child recursively demanded future18:\n%s", &out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("observation changed workflow")
			}
		})
	}
	before := controllerSnapshot(t, root)
	if code, out := preparationOpen(t, root, "future17", "plan", "plan-drafter", "ordinary-child"); code != 3 || !strings.Contains(out, "future18") {
		t.Fatalf("marker history exempted ordinary child admission: %d\n%s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("ordinary child refusal wrote workflow")
	}
	if _, err := os.Stat(filepath.Join(root, "work", "future18")); !os.IsNotExist(err) {
		t.Fatalf("created recursive child: %v", err)
	}
}

func TestAutocompletePreparationRejectsInvalidMarkersBeforeWrites(t *testing.T) {
	for _, tc := range []struct {
		name, phase, role, parent string
		extra                     []string
		setup                     func(*testing.T, string)
		code                      int
	}{
		{"missing parent", "plan", "plan-drafter", "missing", nil, nil, 3},
		{"self parent", "plan", "plan-drafter", "future17", nil, nil, 3},
		{"wrong binding", "plan", "plan-drafter", "current16", nil, func(t *testing.T, root string) {
			frontierFixture(t, root, "work", "future17", "other", "plan", "running", "/rite-vet future17", 2, "1")
		}, 3},
		{"nonadjacent", "plan", "plan-drafter", "current16", nil, func(t *testing.T, root string) {
			frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 3, "1")
		}, 3},
		{"cycle", "plan", "plan-drafter", "current16", nil, func(t *testing.T, root string) {
			frontierFixture(t, root, "work", "current16", "future17", "plan", "running", "/rite-plan", 1, "2")
		}, 3},
		{"wrong role", "plan", "proof-runner", "current16", nil, nil, 2},
		{"Build", "build", "slice-wright", "current16", nil, nil, 2},
		{"combined flags", "plan", "evidence-scout", "current16", []string{"--sequence-research"}, nil, 2},
	} {
		for _, surface := range []string{"dispatch", "context"} {
			t.Run(tc.name+"/"+surface, func(t *testing.T) {
				root, _ := controllerFixture(t)
				frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "1")
				if tc.setup != nil {
					tc.setup(t, root)
				}
				before := controllerSnapshot(t, root)
				extra := append([]string{"--prepare-for", tc.parent}, tc.extra...)
				var code int
				var out string
				if surface == "dispatch" {
					code, out = preparationOpen(t, root, "future17", tc.phase, tc.role, "invalid", extra...)
				} else {
					code, out = preparationContext(t, root, "future17", tc.phase, tc.role, false, extra...)
				}
				if code != tc.code {
					t.Fatalf("invalid preparation admitted: %d want %d\n%s", code, tc.code, out)
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("invalid marker wrote workflow")
				}
			})
		}
	}
}

func TestAutocompleteSequenceResearchBypassesOnlyNormalization(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		t.Run(map[bool]string{false: "conflicting history", true: "invalid canonical"}[malformed], func(t *testing.T) {
			root, dir := controllerFixture(t)
			if malformed {
				writeFile(t, filepath.Join(dir, "roadmap.md"), "Sequence root: current16\nDecision: DEC-017\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 2 | duplicate17 |\n")
			} else {
				writeFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | renamed17 |\n")
			}
			for _, surface := range []string{"dispatch", "context"} {
				before := controllerSnapshot(t, root)
				var code int
				var out string
				if surface == "dispatch" {
					code, out = preparationOpen(t, root, "current16", "plan", "plan-drafter", "normal")
				} else {
					code, out = preparationContext(t, root, "current16", "plan", "plan-drafter", false)
				}
				if code != 3 || !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatalf("normal admission bypassed normalization: %d\n%s", code, out)
				}
				if surface == "dispatch" {
					code, out = preparationOpen(t, root, "current16", "plan", "evidence-scout", "research", "--sequence-research")
				} else {
					code, out = preparationContext(t, root, "current16", "plan", "evidence-scout", true, "--sequence-research")
				}
				if code != 0 {
					t.Fatalf("exact-source research could not inspect authority: %d\n%s", code, out)
				}
			}
			var account struct {
				Waves map[string]map[string]any `json:"waves"`
			}
			if err := json.Unmarshal([]byte(readTestFile(t, filepath.Join(dir, "dispatch.json"))), &account); err != nil {
				t.Fatal(err)
			}
			if account.Waves["research"]["sequence_research"] != true {
				t.Fatal("research account lost durable restriction")
			}
			before := controllerSnapshot(t, root)
			if code, out := preparationOpen(t, root, "current16", "plan", "plan-drafter", "not-research", "--sequence-research"); code != 2 {
				t.Fatalf("research admitted writer role: %d %s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("invalid research changed workflow")
			}
		})
	}
}

func TestAutocompletePreparationFiniteTailAndExistingChildLimits(t *testing.T) {
	for _, tc := range []struct {
		name, remaining, policy string
		child, allowed          bool
	}{
		{"finite tail", "0", "", false, true},
		{"zero budget mapped missing child", "0", "", false, true},
		{"opt-out mapped missing child", "2", "continue_sequence: false\n", false, true},
		{"zero budget existing child", "0", "", true, false},
		{"opt-out existing child", "2", "continue_sequence: false\n", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := controllerFixture(t)
			frontierFixture(t, root, "work", "current16", "", "plan", "running", "/rite-plan", 1, tc.remaining)
			if tc.child {
				frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "1")
			} else if tc.name == "finite tail" {
				writeFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n")
			}
			writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n"+tc.policy)
			contextBefore := controllerSnapshot(t, root)
			contextCode, contextOut := preparationContext(t, root, "current16", "plan", "plan-drafter", false)
			if tc.allowed {
				if contextCode != 0 {
					t.Fatalf("finite-tail context invented successor: %d %s", contextCode, contextOut)
				}
			} else if contextCode != 3 || !strings.Contains(contextOut, "future17") || !reflect.DeepEqual(contextBefore, controllerSnapshot(t, root)) {
				t.Fatalf("context opening limit erased existing child or wrote bundle: %d %s", contextCode, contextOut)
			}
			before := controllerSnapshot(t, root)
			code, out := preparationOpen(t, root, "current16", "plan", "plan-drafter", "limited")
			if tc.allowed {
				if code != 0 {
					t.Fatalf("finite tail invented successor: %d %s", code, out)
				}
			} else {
				if code != 3 || !strings.Contains(out, "future17") {
					t.Fatalf("new-opening limit erased existing child: %d %s", code, out)
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("limited refusal wrote workflow")
				}
			}
		})
	}
}

func TestAutocompleteSequenceResearchDoesNotPrepareSuccessor(t *testing.T) {
	root, _ := controllerFixture(t)
	frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "1")
	if code, out := preparationOpen(t, root, "future17", "plan", "evidence-scout", "research", "--sequence-research"); code != 0 {
		t.Fatalf("research open: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "research", "--role", "evidence-scout", "--handle", "native-research"); code != 0 {
		t.Fatalf("research start: %d %s", code, out)
	}
	before := controllerSnapshot(t, root)
	if code, out := preparationOpen(t, root, "current16", "plan", "plan-drafter", "forward"); code != 3 || !strings.Contains(out, "future17") {
		t.Fatalf("research counted as successor preparation: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("research-only refusal wrote workflow")
	}
}

func TestAutocompletePreparationPreservesNativeCursorReadinessAndRetainedActors(t *testing.T) {
	for _, transition := range []string{"native cursor", "validated readiness", "pending actor", "unintegrated writer"} {
		t.Run(transition, func(t *testing.T) {
			root, _ := controllerFixture(t)
			child := frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "1")
			if code, out := preparationOpen(t, root, "future17", "plan", "plan-drafter", "prepare", "--prepare-for", "current16"); code != 0 {
				t.Fatalf("prepare open: %d %s", code, out)
			}
			if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "prepare", "--role", "plan-drafter", "--handle", "native-plan"); code != 0 {
				t.Fatalf("prepare start: %d %s", code, out)
			}
			if transition != "pending actor" {
				if code, out := controllerDispatch(t, root, "future17", "return", "--wave", "prepare", "--role", "plan-drafter", "--handle", "native-plan"); code != 0 {
					t.Fatalf("prepare return: %d %s", code, out)
				}
			}
			history := readTestFile(t, filepath.Join(child, "dispatch.json"))
			switch transition {
			case "native cursor":
				frontierFixture(t, root, "work", "future17", "current16", "vet", "running", "/rite-vet future17", 2, "1")
			case "validated readiness", "pending actor", "unintegrated writer":
				controllerReadyFuture(t, root, "src/future.go")
			}
			if transition == "unintegrated writer" {
				writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
				if code, out := controllerOpen(t, root, "future17", "writer", "SLICE-001"); code != 0 {
					t.Fatalf("writer fixture: %d %s", code, out)
				}
				for _, args := range [][]string{
					{"start", "--wave", "writer", "--role", "slice-wright", "--handle", "native-writer"},
					{"return", "--wave", "writer", "--role", "slice-wright", "--handle", "native-writer"},
				} {
					if code, out := controllerDispatch(t, root, "future17", args...); code != 0 {
						t.Fatalf("unintegrated fixture: %d %s", code, out)
					}
				}
				writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
				history = readTestFile(t, filepath.Join(child, "dispatch.json"))
			}
			before := controllerSnapshot(t, root)
			lines, err := FrontierLines(root, "current16")
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Join(lines, "\n")
			switch transition {
			case "native cursor":
				if !strings.Contains(text, "future17 location=live action=prepare reason=native internal phase cursor: /rite-vet future17") {
					t.Fatalf("completed preparation masked advanced native cursor:\n%s", text)
				}
			case "validated readiness":
				if !strings.Contains(text, "future17/SLICE-001") || strings.Contains(text, "future17 location=live action=inspect reason=retained dispatch") {
					t.Fatalf("completed preparation masked validated readiness:\n%s", text)
				}
			case "pending actor":
				if !strings.Contains(text, "future17 location=live action=wait reason=retained dispatch work") {
					t.Fatalf("readiness hid pending preparation actor:\n%s", text)
				}
			case "unintegrated writer":
				if !strings.Contains(text, "future17 location=live action=inspect reason=retained dispatch returns") {
					t.Fatalf("unfinished native reconciliation disappeared:\n%s", text)
				}
			}
			if transition == "pending actor" && strings.Contains(text, "frontier-prepare: /rite-spec 'future18'") {
				t.Fatalf("pending preparatory actor recursively opened next workspace:\n%s", text)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) || readTestFile(t, filepath.Join(child, "dispatch.json")) != history {
				t.Fatal("routing rewrote immutable preparation history")
			}
		})
	}
}

func TestAutocompletePreparationMarkerDoesNotExemptChildBuild(t *testing.T) {
	root, _ := controllerFixture(t)
	controllerReadyFuture(t, root, "src/future.go")
	frontierFixture(t, root, "work", "future17", "current16", "vet", "ready", "/rite-build SLICE-001", 2, "1")
	poolVet(t, root, "future17")
	if code, out := preparationOpen(t, root, "future17", "vet", "plan-reviewer", "prepare", "--prepare-for", "current16"); code != 0 {
		t.Fatalf("marked Vet: %d %s", code, out)
	}
	before := controllerSnapshot(t, root)
	if code, out := controllerOpen(t, root, "future17", "forward-build", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/future.go"); code != 3 || !strings.Contains(out, "future18") {
		t.Fatalf("child Build inherited preparation exemption: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("child Build refusal wrote claim, round or history")
	}
}

func TestAutocompleteSequenceResearchRejectsRolePhaseAndSourceMisuse(t *testing.T) {
	for _, tc := range []struct{ name, slug, phase, role string }{
		{"noncanonical role", "current16", "plan", "evidence-scout.extra"},
		{"review role", "current16", "temper", "strategy-reviewer"},
		{"Build phase", "current16", "build", "evidence-scout"},
		{"proof phase", "current16", "prove", "evidence-scout"},
		{"missing source", "future17", "plan", "evidence-scout"},
	} {
		for _, surface := range []string{"dispatch", "context"} {
			t.Run(tc.name+"/"+surface, func(t *testing.T) {
				root, _ := controllerFixture(t)
				before := controllerSnapshot(t, root)
				var code int
				var out string
				if surface == "dispatch" {
					code, out = preparationOpen(t, root, tc.slug, tc.phase, tc.role, "research", "--sequence-research")
				} else {
					code, out = preparationContext(t, root, tc.slug, tc.phase, tc.role, false, "--sequence-research")
				}
				want := 2
				if code != want {
					t.Fatalf("invalid research admitted: %d want %d\n%s", code, want, out)
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("invalid research published workflow")
				}
			})
		}
	}
}

func TestAutocompletePreparationBoundaryPrecedesMarkedHistory(t *testing.T) {
	for _, history := range []string{"new workspace", "spec", "clarify"} {
		t.Run(history, func(t *testing.T) {
			root, _ := controllerFixture(t)
			phase := "spec"
			if history == "clarify" {
				phase = "clarify"
			}
			child := frontierFixture(t, root, "work", "future17", "current16", phase, "running", "/rite-"+phase+" future17", 2, "1")
			if history != "new workspace" {
				if code, out := preparationOpen(t, root, "future17", phase, "evidence-scout", "initial"); code != 0 {
					t.Fatalf("unmarked initial preparation: %d %s", code, out)
				}
				if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "initial", "--role", "evidence-scout", "--handle", "native-initial"); code != 0 {
					t.Fatalf("initial preparation start: %d %s", code, out)
				}
			}
			if code, out := preparationContext(t, root, "future17", phase, "", false); code != 0 {
				t.Fatalf("initial root context: %d %s", code, out)
			}
			before := controllerSnapshot(t, root)
			for _, observe := range []string{"frontier", "next", "continuation"} {
				var out, errOut bytes.Buffer
				if observe == "frontier" {
					lines, err := FrontierLines(root, "current16")
					if err != nil {
						t.Fatal(err)
					}
					out.WriteString(strings.Join(lines, "\n"))
				} else {
					run := RunNext
					if observe == "continuation" {
						run = RunCheckContinuation
					}
					if code := run(root, []string{"current16"}, &out, &errOut); code != 0 {
						t.Fatalf("%s: %d %s", observe, code, &errOut)
					}
				}
				if strings.Contains(out.String(), "frontier-prepare: /rite-spec 'future18'") || strings.Contains(out.String(), "future18 location=missing action=prepare") {
					t.Fatalf("%s recursively demanded C before marked history existed:\n%s", observe, &out)
				}
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("initial preparation observation changed workflow")
			}
			if history == "new workspace" {
				if _, err := os.Stat(filepath.Join(child, "dispatch.json")); !os.IsNotExist(err) {
					t.Fatalf("root context created preparation account: %v", err)
				}
			}
		})
	}
}

func TestAutocompletePreparationRolelessRootContextDoesNotGrantDispatch(t *testing.T) {
	root, _ := controllerFixture(t)
	child := frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-plan", 2, "1")
	for _, alias := range []bool{false, true} {
		if code, out := preparationContext(t, root, "future17", "plan", "", alias, "--prepare-for", "current16"); code != 0 {
			t.Fatalf("roleless controlling-root preparation context: %d %s", code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(child, "dispatch.json")); !os.IsNotExist(err) {
		t.Fatalf("root preparation context created account: %v", err)
	}
	before := controllerSnapshot(t, root)
	var out, errOut bytes.Buffer
	code := RunDispatch(root, []string{"future17", "open", "--phase", "plan", "--wave", "roleless", "--prepare-for", "current16"}, &out, &errOut)
	if code != 2 || !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatalf("roleless root context granted roleless dispatch: %d %s%s", code, &out, &errOut)
	}
	if code, out := preparationOpen(t, root, "current16", "plan", "plan-drafter", "parent-forward"); code != 3 || !strings.Contains(out, "future17") {
		t.Fatalf("root context alone counted as started successor: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("context-only preparation refusal wrote workflow")
	}
}

func TestAutocompletePreparationBoundaryReleasesAtNativeHandoff(t *testing.T) {
	for _, release := range []string{"terminal predecessor", "ACTIVE child", "ordinary forward admission"} {
		t.Run(release, func(t *testing.T) {
			root, _ := controllerFixture(t)
			child := frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-plan", 2, "1")
			switch release {
			case "terminal predecessor":
				frontierFixture(t, root, "work", "current16", "", "done", "shipped", "/rite-spec future17", 1, "2")
			case "ACTIVE child":
				writeFile(t, filepath.Join(root, "ACTIVE"), "future17\n")
			case "ordinary forward admission":
				evidence := filepath.Join(child, "evidence.md")
				writeFile(t, evidence, "frontier-blocked: future18 — DEC-018 needs owner scope decision\n")
				if code, out := preparationOpen(t, root, "future17", "plan", "plan-drafter", "ordinary"); code != 0 {
					t.Fatalf("ordinary child admission with concrete deferral: %d %s", code, out)
				}
				writeFile(t, evidence, "# Evidence\n\nDEC-018 scheduling constraint resolved.\n")
			}
			before := controllerSnapshot(t, root)
			lines, err := FrontierLines(root, "future17")
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Join(lines, "\n")
			if !strings.Contains(text, "frontier-prepare: /rite-spec 'future18'") {
				t.Fatalf("%s retained preparatory-only boundary after native release:\n%s", release, text)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("handoff observation changed workflow")
			}
		})
	}
}

func TestAutocompletePreparationPurposeAppliesOnlyToDispatchOpen(t *testing.T) {
	for _, sub := range []string{"start", "return", "status"} {
		for _, purpose := range [][]string{{"--prepare-for", "current16"}, {"--sequence-research"}} {
			t.Run(sub+"/"+purpose[0], func(t *testing.T) {
				root, _ := controllerFixture(t)
				before := controllerSnapshot(t, root)
				args := append([]string{sub, "--wave", "unknown", "--role", "evidence-scout"}, purpose...)
				if code, out := controllerDispatch(t, root, "current16", args...); code != 2 {
					t.Fatalf("purpose accepted outside open: %d %s", code, out)
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("misplaced purpose changed workflow")
				}
			})
		}
	}
}

func TestAutocompletePreparationRejectsRepeatedParentBeforeWrites(t *testing.T) {
	for _, parents := range [][]string{{"current16", "current16"}, {"missing", "current16"}} {
		for _, surface := range []string{"context", "dispatch"} {
			t.Run(surface+"/"+strings.Join(parents, "-"), func(t *testing.T) {
				root, _ := controllerFixture(t)
				frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-plan", 2, "1")
				before := controllerSnapshot(t, root)
				flags := []string{"--prepare-for", parents[0], "--prepare-for", parents[1]}
				var code int
				var out string
				if surface == "context" {
					code, out = preparationContext(t, root, "future17", "plan", "plan-drafter", false, flags...)
				} else {
					code, out = preparationOpen(t, root, "future17", "plan", "plan-drafter", "duplicate", flags...)
				}
				if code != 2 {
					t.Fatalf("repeated singleton parent accepted: %d %s", code, out)
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("duplicate parent refusal wrote workflow")
				}
			})
		}
	}
}

func TestAutocompletePreparationExistingDescendantDoesNotRecursivelyReroute(t *testing.T) {
	for _, descendant := range []string{"initial frame cursor", "initial cursor", "ready candidate", "retained actor"} {
		t.Run(descendant, func(t *testing.T) {
			root, _ := controllerFixture(t)
			controllerPlanning(t, root, "plan", "plan-drafter", true)
			frontierFixture(t, root, "work", "future18", "future17", "spec", "running", "/rite-spec future18", 3, "0")
			switch descendant {
			case "initial frame cursor":
				frontierFixture(t, root, "work", "future18", "future17", "frame", "running", "/rite-spec future18", 3, "0")
			case "ready candidate":
				poolWorkspace(t, root, "future18", "future17", poolSlice("SLICE-001", "src/independent.go", "Workspace dependencies: none"))
				frontierFixture(t, root, "work", "future18", "future17", "vet", "ready", "/rite-build SLICE-001", 3, "0")
				poolVet(t, root, "future18")
			case "retained actor":
				if code, out := preparationOpen(t, root, "future18", "spec", "evidence-scout", "independent"); code != 0 {
					t.Fatalf("independent descendant open: %d %s", code, out)
				}
				if code, out := controllerDispatch(t, root, "future18", "start", "--wave", "independent", "--role", "evidence-scout", "--handle", "native-independent"); code != 0 {
					t.Fatalf("independent descendant start: %d %s", code, out)
				}
			}
			for _, surface := range []string{"frontier", "next", "continuation"} {
				before := controllerSnapshot(t, root)
				var out, errOut bytes.Buffer
				if surface == "frontier" {
					lines, err := FrontierLines(root, "current16")
					if err != nil {
						t.Fatal(err)
					}
					out.WriteString(strings.Join(lines, "\n"))
				} else {
					run := RunNext
					if surface == "continuation" {
						run = RunCheckContinuation
					}
					if code := run(root, []string{"current16"}, &out, &errOut); code != 0 {
						t.Fatalf("%s: %d %s", surface, code, &errOut)
					}
				}
				text := out.String()
				switch descendant {
				case "initial frame cursor", "initial cursor":
					if !strings.Contains(text, "future18 location=live action=wait") || strings.Contains(text, "future18 location=live action=prepare") || strings.Contains(text, "next-command: /rite-autocomplete") || strings.Contains(text, "invoke: /rite-autocomplete") {
						t.Fatalf("%s recursively rerouted eligible A for unadmitted C:\n%s", surface, text)
					}
				case "ready candidate":
					if !strings.Contains(text, "future18/SLICE-001") {
						t.Fatalf("%s hid independently ready descendant:\n%s", surface, text)
					}
				case "retained actor":
					if !strings.Contains(text, "future18 location=live action=wait reason=retained dispatch work") {
						t.Fatalf("%s hid actual descendant actor reconciliation:\n%s", surface, text)
					}
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("descendant observation changed workflow")
				}
			}
		})
	}
}

func TestAutocompletePreparationReplannedParentPreservesSettledHistory(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(map[bool]string{false: "settled history", true: "pending actor"}[pending], func(t *testing.T) {
			root, source := controllerFixture(t)
			controllerPlanning(t, root, "plan", "plan-drafter", true)
			if !pending {
				if code, out := controllerDispatch(t, root, "future17", "return", "--wave", "prepare17", "--role", "plan-drafter", "--handle", "native-17"); code != 0 {
					t.Fatalf("settle historical preparation: %d %s", code, out)
				}
			}
			child := filepath.Join(root, "work", "future17")
			history := readTestFile(t, filepath.Join(child, "dispatch.json"))
			frontierFixture(t, root, "work", "middle", "current16", "plan", "running", "/rite-plan", 2, "2")
			frontierFixture(t, root, "work", "future17", "middle", "plan", "running", "/rite-plan", 3, "1")
			writeFile(t, filepath.Join(source, "decisions.md"), "# Decisions\n\n## DEC-019\nInsert middle before future17.\n")
			writeFile(t, filepath.Join(source, "roadmap.md"), "Sequence root: current16\nDecision: DEC-019\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | middle |\n| 3 | future17 |\n| 4 | future18 |\n")
			poolVet(t, root, "current16")
			before := controllerSnapshot(t, root)
			lines, err := FrontierLines(root, "current16")
			if pending {
				if err == nil && !strings.Contains(strings.Join(lines, "\n"), "preparation account does not match") {
					t.Fatalf("pending actor was silently rebound to repaired parent:\n%s", strings.Join(lines, "\n"))
				}
			} else {
				if err != nil {
					t.Fatalf("settled historical marker froze legitimate metadata repair: %v", err)
				}
				if code, out := preparationOpen(t, root, "middle", "plan", "plan-drafter", "ordinary-middle"); code != 3 || !strings.Contains(out, "future17") {
					t.Fatalf("stale settled history supplied new-parent preparation credit: %d %s", code, out)
				}
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) || readTestFile(t, filepath.Join(child, "dispatch.json")) != history {
				t.Fatal("replan observation/admission rewrote immutable history")
			}
		})
	}
}

func TestAutocompletePreparationSingleCapRetainsLegacyDescendantRouting(t *testing.T) {
	root, _ := controllerFixture(t)
	frontierFixture(t, root, "work", "future17", "current16", "spec", "running", "/rite-spec future17", 2, "1")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 1\n")
	before := controllerSnapshot(t, root)
	lines, err := FrontierLines(root, "current16")
	if err != nil {
		t.Fatal(err)
	}
	if text := strings.Join(lines, "\n"); !strings.Contains(text, "frontier-prepare: /rite-spec 'future18' source=future17") {
		t.Fatalf("cap1 acquired new preparatory-only routing policy:\n%s", text)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("cap1 observation changed workflow")
	}
}

func preparationReparentedActorFixture(t *testing.T) (string, string) {
	t.Helper()
	root, source := controllerFixture(t)
	controllerPlanning(t, root, "plan", "plan-drafter", true)
	frontierFixture(t, root, "work", "current16", "", "build", "running", "/rite-build SLICE-001", 1, "3")
	frontierFixture(t, root, "work", "middle", "current16", "plan", "running", "/rite-plan", 2, "2")
	child := frontierFixture(t, root, "work", "future17", "middle", "plan", "running", "/rite-plan", 3, "1")
	writeFile(t, filepath.Join(source, "decisions.md"), "# Decisions\n\n## DEC-019\nInsert middle before future17.\n")
	writeFile(t, filepath.Join(source, "roadmap.md"), "Sequence root: current16\nDecision: DEC-019\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | middle |\n| 3 | future17 |\n| 4 | future18 |\n")
	writeFile(t, filepath.Join(child, "evidence.md"), "frontier-blocked: future18 — DEC-018 needs owner scope decision\n")
	poolVet(t, root, "current16")
	return root, child
}

func preparationAddLedgerWave(t *testing.T, child, name, waveJSON string) {
	t.Helper()
	path := filepath.Join(child, "dispatch.json")
	var account struct {
		Waves map[string]json.RawMessage `json:"waves"`
	}
	if err := json.Unmarshal([]byte(readTestFile(t, path)), &account); err != nil {
		t.Fatal(err)
	}
	account.Waves[name] = json.RawMessage(waveJSON)
	raw, err := json.Marshal(account)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(raw))
}

func TestAutocompletePreparationReparentedActorBlocksOwnForwardAdmissionBeforeWrites(t *testing.T) {
	for _, surface := range []string{"marked context", "marked dispatch", "ordinary context", "ordinary dispatch"} {
		t.Run(surface, func(t *testing.T) {
			root, _ := preparationReparentedActorFixture(t)
			before := controllerSnapshot(t, root)
			var purpose []string
			if strings.HasPrefix(surface, "marked") {
				purpose = []string{"--prepare-for", "middle"}
			}
			var code int
			var out string
			if strings.HasSuffix(surface, "context") {
				code, out = preparationContext(t, root, "future17", "plan", "plan-drafter", false, purpose...)
			} else {
				code, out = preparationOpen(t, root, "future17", "plan", "plan-drafter", "new-scope", purpose...)
			}
			if code != 3 || !strings.Contains(out, "preparation account does not match") {
				t.Fatalf("unsettled old scope allowed new forward admission: %d %s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("old-scope actor refusal published bundle, wave or history")
			}
		})
	}
}

func TestAutocompletePreparationMixedBindingsNeverSupplyParentCredit(t *testing.T) {
	root, child := preparationReparentedActorFixture(t)
	preparationAddLedgerWave(t, child, "matching-current-parent", `{"phase":"plan","state":"sealed","preparation_for":"middle","roles":{"plan-drafter":{"handle":"native-current-parent","started":"2026-11-05T00:00:00Z"}}}`)
	before := controllerSnapshot(t, root)
	if code, out := preparationOpen(t, root, "middle", "plan", "plan-drafter", "parent-forward"); code != 3 || !strings.Contains(out, "preparation account does not match") {
		t.Fatalf("one matching actor hid another mismatched actor: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("mixed-ledger refusal wrote parent workflow")
	}
}

func TestAutocompletePreparationRoutingCannotHideOldActorBehindHandoffOrOrdinaryWave(t *testing.T) {
	for _, release := range []string{"ACTIVE child", "ordinary wave"} {
		t.Run(release, func(t *testing.T) {
			root, child := preparationReparentedActorFixture(t)
			if release == "ACTIVE child" {
				writeFile(t, filepath.Join(root, "ACTIVE"), "future17\n")
			} else {
				preparationAddLedgerWave(t, child, "ordinary-forward", `{"phase":"plan","state":"open","roles":{"plan-drafter":{}}}`)
			}
			before := controllerSnapshot(t, root)
			lines, err := FrontierLines(root, "future17")
			if err == nil && !strings.Contains(strings.Join(lines, "\n"), "preparation account does not match") {
				t.Fatalf("%s hid old-scope actor binding:\n%s", release, strings.Join(lines, "\n"))
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("binding observation rewrote ledger")
			}
		})
	}
}

func TestAutocompletePreparationOldActorCanBeInvestigatedAndSettledAfterReparent(t *testing.T) {
	for _, settlement := range []string{"return", "abandon"} {
		t.Run(settlement, func(t *testing.T) {
			root, child := preparationReparentedActorFixture(t)
			if code, out := preparationContext(t, root, "future17", "plan", "evidence-scout", false, "--sequence-research"); code != 0 {
				t.Fatalf("old binding prevented investigative context: %d %s", code, out)
			}
			args := []string{"return", "--wave", "prepare17", "--role", "plan-drafter", "--handle", "native-17"}
			want := 0
			if settlement == "abandon" {
				args = []string{"abandon", "--wave", "prepare17", "--reason", "host confirmed old actor terminal", "--release"}
				want = 3
			}
			if code, out := controllerDispatch(t, root, "future17", args...); code != want || strings.Contains(out, "preparation account does not match") {
				t.Fatalf("old actor could not settle through %s: %d %s", settlement, code, out)
			}
			settled := readTestFile(t, filepath.Join(child, "dispatch.json"))
			var account struct {
				Waves map[string]map[string]any `json:"waves"`
			}
			if err := json.Unmarshal([]byte(settled), &account); err != nil {
				t.Fatal(err)
			}
			wave := account.Waves["prepare17"]
			if settlement == "return" && wave["state"] != "complete" || settlement == "abandon" && (wave["state"] != "abandoned" || wave["terminal_at"] == nil) {
				t.Fatalf("settlement was not retained: %#v", wave)
			}
			before := controllerSnapshot(t, root)
			if _, err := FrontierLines(root, "current16"); err != nil {
				t.Fatalf("settled history still blocks routing: %v", err)
			}
			if code, out := preparationOpen(t, root, "middle", "plan", "plan-drafter", "no-history-credit"); code != 3 || !strings.Contains(out, "future17") {
				t.Fatalf("settled old actor supplied current preparation credit: %d %s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) || readTestFile(t, filepath.Join(child, "dispatch.json")) != settled {
				t.Fatal("settled immutable history was rewritten during observation/admission")
			}
		})
	}
}

func TestAutocompletePreparationReparentedActorBlocksOwnBuildBeforeWrites(t *testing.T) {
	root, _ := preparationReparentedActorFixture(t)
	poolWorkspace(t, root, "future17", "middle", poolSlice("SLICE-001", "src/future.go", "Workspace dependencies: none"))
	frontierFixture(t, root, "work", "future17", "middle", "vet", "ready", "/rite-build SLICE-001", 3, "1")
	poolVet(t, root, "future17")
	before := controllerSnapshot(t, root)
	if code, out := controllerOpen(t, root, "future17", "new-build-scope", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/future.go"); code != 3 || !strings.Contains(out, "preparation account does not match") {
		t.Fatalf("Build did not retain old-scope actor binding: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("old-scope Build refusal wrote claims, rounds or history")
	}
}

func TestAutocompletePreparationOrdinaryBuildSpellingReleasesBoundary(t *testing.T) {
	for _, phase := range []string{"build", "BUILD"} {
		t.Run(phase, func(t *testing.T) {
			root, _ := controllerFixture(t)
			child := controllerReadyFuture(t, root, "src/future.go")
			frontierFixture(t, root, "work", "future17", "current16", "vet", "ready", "/rite-build SLICE-001", 2, "1")
			poolVet(t, root, "future17")
			writeFile(t, filepath.Join(child, "evidence.md"), "frontier-blocked: future18 — DEC-018 needs owner scope decision\n")
			controllerLease(t, root, 2,
				controllerLeaseRow("", "SLICE-001", "src/current.go"),
				controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
			if code, out := controllerDispatch(t, root, "future17", "open", "--phase", phase, "--wave", "ordinary-build", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t)); code != 0 {
				t.Fatalf("ordinary %s admission: %d %s", phase, code, out)
			}
			writeFile(t, filepath.Join(child, "evidence.md"), "# Evidence\n\nDEC-018 scheduling constraint resolved.\n")
			before := controllerSnapshot(t, root)
			lines, err := FrontierLines(root, "current16")
			if err != nil {
				t.Fatal(err)
			}
			if text := strings.Join(lines, "\n"); !strings.Contains(text, "frontier-prepare: /rite-spec 'future18' source=future17") {
				t.Fatalf("ordinary %s admission did not release preparation boundary:\n%s", phase, text)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("ordinary Build routing rewrote admitted account")
			}
		})
	}
}

func TestAutocompletePreparationForeignReadOnlyBindingGuardsReservedWriter(t *testing.T) {
	for _, settled := range []bool{false, true} {
		t.Run(map[bool]string{false: "unsettled foreign binding", true: "settled foreign history"}[settled], func(t *testing.T) {
			root, child := preparationReparentedActorFixture(t)
			if code, out := preparationOpen(t, root, "middle", "plan", "plan-drafter", "prepare-middle", "--prepare-for", "current16"); code != 0 {
				t.Fatalf("immediate preparation open: %d %s", code, out)
			}
			if code, out := controllerDispatch(t, root, "middle", "start", "--wave", "prepare-middle", "--role", "plan-drafter", "--handle", "native-middle"); code != 0 {
				t.Fatalf("immediate preparation start: %d %s", code, out)
			}
			if settled {
				if code, out := controllerDispatch(t, root, "future17", "return", "--wave", "prepare17", "--role", "plan-drafter", "--handle", "native-17"); code != 0 {
					t.Fatalf("settle foreign old-scope actor: %d %s", code, out)
				}
			}
			controllerLease(t, root, 2, controllerLeaseRow("", "SLICE-001", "src/current.go"))
			history := readTestFile(t, filepath.Join(child, "dispatch.json"))
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "reserved-writer", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/current.go")
			if settled {
				if code != 0 || !strings.Contains(out, "wave reserved-writer open") || !strings.Contains(out, "wright round 1") {
					t.Fatalf("settled foreign history blocked valid reserved writer: %d %s", code, out)
				}
			} else {
				if code != 3 || !strings.Contains(out, "preparation account does not match") {
					t.Fatalf("foreign non-wright actor escaped writer-retention binding checks: %d %s", code, out)
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("foreign binding refusal wrote claims, rounds, wave or history")
				}
			}
			if readTestFile(t, filepath.Join(child, "dispatch.json")) != history {
				t.Fatal("writer admission rewrote foreign immutable history")
			}
		})
	}
}

func TestAutocompletePreparationReadOnlyBuildDoesNotReleaseBoundary(t *testing.T) {
	root, _ := controllerFixture(t)
	controllerPlanning(t, root, "plan", "plan-drafter", true)
	if code, out := preparationOpen(t, root, "future17", "build", "evidence-scout", "readonly-build"); code != 0 || !strings.Contains(out, "wave readonly-build open") {
		t.Fatalf("compatible read-only Build opening was blocked: %d %s", code, out)
	}
	for _, surface := range []string{"frontier", "next", "continuation"} {
		before := controllerSnapshot(t, root)
		var out, errOut bytes.Buffer
		if surface == "frontier" {
			lines, err := FrontierLines(root, "current16")
			if err != nil {
				t.Fatal(err)
			}
			out.WriteString(strings.Join(lines, "\n"))
		} else {
			run := RunNext
			if surface == "continuation" {
				run = RunCheckContinuation
			}
			if code := run(root, []string{"current16"}, &out, &errOut); code != 0 {
				t.Fatalf("%s: %d %s", surface, code, &errOut)
			}
		}
		text := out.String()
		if strings.Contains(text, "frontier-prepare: /rite-spec 'future18'") || strings.Contains(text, "future18 location=missing action=prepare") || strings.Contains(text, "next-command: /rite-autocomplete") || strings.Contains(text, "invoke: /rite-autocomplete") {
			t.Fatalf("%s treated read-only Build declaration as admitted writer:\n%s", surface, text)
		}
		if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
			t.Fatal("read-only Build routing changed workflow")
		}
	}
}

func TestAutocompletePreparationExplicitDescendantWaitControlsLocalFallbacks(t *testing.T) {
	for _, work := range []string{"unadmitted Spec", "ready descendant", "protected proof", "independent global work"} {
		t.Run(work, func(t *testing.T) {
			root, source := controllerFixture(t)
			controllerPlanning(t, root, "plan", "plan-drafter", true)
			frontierFixture(t, root, "work", "current16", "", "temper", "running", "/rite-define current16", 1, "2")
			if code, out := preparationOpen(t, root, "current16", "temper", "strategy-reviewer", "held-temper"); code != 0 {
				t.Fatalf("held predecessor review: %d %s", code, out)
			}
			if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "held-temper", "--role", "strategy-reviewer", "--handle", "native-temper"); code != 0 {
				t.Fatalf("held predecessor start: %d %s", code, out)
			}
			frontierFixture(t, root, "work", "future18", "future17", "spec", "running", "/rite-spec future18", 3, "0")
			switch work {
			case "ready descendant":
				poolWorkspace(t, root, "future18", "future17", poolSlice("SLICE-001", "src/independent.go", "Workspace dependencies: none"))
				frontierFixture(t, root, "work", "future18", "future17", "vet", "ready", "/rite-build SLICE-001", 3, "0")
				poolVet(t, root, "future18")
			case "protected proof":
				frontierFixture(t, root, "work", "future18", "future17", "prove", "running", "/rite-prove future18", 3, "0")
				if code, out := preparationOpen(t, root, "future18", "prove", "proof-runner", "protected-proof"); code != 0 {
					t.Fatalf("protected descendant proof: %d %s", code, out)
				}
				if code, out := controllerDispatch(t, root, "future18", "start", "--wave", "protected-proof", "--role", "proof-runner", "--handle", "native-proof"); code != 0 {
					t.Fatalf("protected descendant proof start: %d %s", code, out)
				}
			case "independent global work":
				frontierFixture(t, root, "work", "current16", "", "temper", "running", "/rite-define current16", 1, "3")
				frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "2")
				writeFile(t, filepath.Join(source, "decisions.md"), "# Decisions\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 3 | future18 |\n| 4 | future19 |\n")
				frontierFixture(t, root, "work", "future18", "future17", "spec", "running", "/rite-spec future18", 3, "1")
				poolWorkspace(t, root, "future19", "future18", poolSlice("SLICE-001", "src/independent.go", "Workspace dependencies: none"))
				frontierFixture(t, root, "work", "future19", "future18", "vet", "ready", "/rite-build SLICE-001", 4, "0")
				poolVet(t, root, "future19")
			}
			for _, surface := range []string{"frontier", "next", "continuation"} {
				before := controllerSnapshot(t, root)
				var out, errOut bytes.Buffer
				if surface == "frontier" {
					lines, err := FrontierLines(root, "future18")
					if err != nil {
						t.Fatal(err)
					}
					out.WriteString(strings.Join(lines, "\n"))
				} else {
					run := RunNext
					if surface == "continuation" {
						run = RunCheckContinuation
					}
					if code := run(root, []string{"future18"}, &out, &errOut); code != 0 {
						t.Fatalf("%s: %d %s", surface, code, &errOut)
					}
				}
				text := out.String()
				switch work {
				case "unadmitted Spec":
					if !strings.Contains(text, "future18 location=live action=wait") || strings.Contains(text, "invoke:") || strings.Contains(text, "next-command:") || strings.Contains(text, "next-phase:") {
						t.Fatalf("%s overrode inherited wait with local phase invocation:\n%s", surface, text)
					}
					if surface == "continuation" && !strings.HasPrefix(text, "continuation: clear\n") {
						t.Fatalf("inherited wait did not clear local continuation:\n%s", text)
					}
				case "ready descendant":
					if !strings.Contains(text, "future18/SLICE-001") {
						t.Fatalf("%s hid independently ready descendant:\n%s", surface, text)
					}
				case "protected proof":
					if !strings.Contains(text, "future18 location=live action=wait reason=retained dispatch work") {
						t.Fatalf("%s hid protected descendant proof:\n%s", surface, text)
					}
				case "independent global work":
					if !strings.Contains(text, "future19/SLICE-001") || surface == "next" && !strings.Contains(text, "next-command: /rite-autocomplete") || surface == "continuation" && !strings.Contains(text, "invoke: /rite-autocomplete") {
						t.Fatalf("%s let inherited wait hide global actionable work:\n%s", surface, text)
					}
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("explicit descendant observation changed ACTIVE or workflow")
				}
			}
		})
	}
}

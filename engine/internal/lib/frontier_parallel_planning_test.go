package lib

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutocompleteSchedulesAgentOwnedPlanningTogether(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
	frontierFixture(t, root, "work", "alpha", "", "plan", "running", "/rite-plan repair alpha", 1, "0")
	frontierFixture(t, root, "work", "beta", "alpha", "plan", "blocked", "/rite-plan repair beta", 2, "0")
	frontierFixture(t, root, "work", "gamma", "beta", "plan", "awaiting_human", "/rite-vet gamma", 3, "0")
	frontierFixture(t, root, "work", "delta", "gamma", "define", "blocked", "/rite-define delta", 4, "0")

	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"alpha"}, &out, &errOut); code != 0 {
		t.Fatalf("code=%d: %s", code, &errOut)
	}
	text := out.String()
	for _, want := range []string{
		"continuation: required\n",
		"invoke: /rite-autocomplete\n",
		"source: alpha\n",
		"frontier-workspace: beta location=live action=prepare reason=native internal phase cursor: /rite-plan repair beta",
		"frontier-workspace: gamma location=live action=prepare reason=native internal phase cursor: /rite-vet gamma",
		"frontier-workspace: delta location=live action=prepare reason=native internal phase cursor: /rite-define delta",
		"frontier-concurrent: /rite-plan repair alpha\n",
		"frontier-concurrent: /rite-plan repair beta\n",
		"launch: /rite-plan repair alpha\n",
		"launch: /rite-plan repair beta\n",
		"reason: a source GO does not end autocomplete; run every launch command and every picks writer this turn; do not ask; Git stays unauthorized\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q:\n%s", want, text)
		}
	}
	if strings.Count(text, "\nfrontier-concurrent: ") != 2 {
		t.Errorf("cap did not limit the launch list to 2:\n%s", text)
	}
	for _, later := range []string{"frontier-concurrent: /rite-vet gamma", "frontier-concurrent: /rite-define delta", "frontier-concurrent: /rite-build"} {
		if strings.Contains(text, later) {
			t.Errorf("launch list included %s:\n%s", later, text)
		}
	}
}

func TestAutocompleteBlockedPlanningCursorContinuesWithoutSibling(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 5\n")
	frontierFixture(t, root, "work", "beta", "", "plan", "blocked", "/rite-plan repair beta", 1, "0")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"beta"}, &out, &errOut); code != 0 {
		t.Fatalf("code=%d: %s", code, &errOut)
	}
	text := out.String()
	for _, want := range []string{
		"continuation: required\n",
		"invoke: /rite-plan repair beta\n",
		"source: beta\n",
		"frontier-concurrent: /rite-plan repair beta\n",
		"launch: /rite-plan repair beta\n",
		"reason: agent-owned planning cursor; launch every frontier-concurrent command before replying\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "continuation: clear") || strings.Contains(text, "invoke: /rite-autocomplete") {
		t.Fatalf("sole planning cursor was treated as a stop:\n%s", text)
	}
}

func TestAutocompletePlanningRecoveryKeepsBuildAndQuestionGates(t *testing.T) {
	t.Run("build", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 5\n")
		frontierFixture(t, root, "work", "ship-hold", "", "plan", "awaiting_human", "/rite-build SLICE-001", 1, "0")
		text := frontierText(t, root, "ship-hold")
		if !strings.Contains(text, "ship-hold location=live action=wait reason=human/blocked cursor; retain native gate") || strings.Contains(text, "frontier-concurrent:") {
			t.Fatalf("Build cursor was scheduled from awaiting_human:\n%s", text)
		}
	})
	t.Run("question", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 5\n")
		dir := frontierFixture(t, root, "work", "questioned", "", "plan", "blocked", "/rite-vet questioned", 1, "0")
		writeFile(t, filepath.Join(dir, "questions.md"), "## q-owned\nstatus: open\ngate: blocking\n")
		var out, errOut bytes.Buffer
		if code := RunCheckContinuation(root, []string{"questioned"}, &out, &errOut); code != 0 {
			t.Fatalf("code=%d: %s", code, &errOut)
		}
		text := out.String()
		if !strings.Contains(text, "continuation: clear\n") || !strings.Contains(text, "questioned location=live action=wait reason=open blocking question; retain native gate") || strings.Contains(text, "frontier-concurrent:") {
			t.Fatalf("open blocking question was scheduled:\n%s", text)
		}
	})
}

func TestStandaloneBlockedPlanningCursorStaysHeld(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
	frontierFixture(t, root, "work", "beta", "", "plan", "blocked", "/rite-plan repair beta", 1, "0")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"beta"}, &out, &errOut); code != 0 {
		t.Fatalf("code=%d: %s", code, &errOut)
	}
	text := out.String()
	if !strings.Contains(text, "continuation: clear\n") || !strings.Contains(text, "reason: status=blocked\n") || strings.Contains(text, "frontier-concurrent:") || strings.Contains(text, "action=prepare") {
		t.Fatalf("standalone mode scheduled a blocked cursor:\n%s", text)
	}
}

func TestAutocompleteSpecChainStillWaitsForUnadmittedDescendant(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 5\n")
	frontierFixture(t, root, "work", "chain-a", "", "vet", "running", "/rite-vet chain-a", 1, "0")
	frontierFixture(t, root, "work", "chain-b", "chain-a", "spec", "running", "/rite-spec chain-b", 2, "0")
	frontierFixture(t, root, "work", "chain-c", "chain-b", "spec", "running", "/rite-spec chain-c", 3, "0")
	text := frontierText(t, root, "chain-a")
	for _, want := range []string{
		"frontier-workspace: chain-b location=live action=prepare reason=native internal phase cursor: /rite-spec chain-b",
		"frontier-workspace: chain-c location=live action=wait reason=" + frontierInheritedPreparationWait,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "frontier-concurrent: /rite-spec chain-c") {
		t.Fatalf("unadmitted grandchild was launched:\n%s", text)
	}
}

func TestAutocompleteBarePhaseCommandNamesWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 5\n")
	frontierFixture(t, root, "work", "bare", "", "vet", "running", "/rite-vet", 1, "0")
	text := frontierText(t, root, "bare")
	for _, want := range []string{
		"reason=native internal phase cursor: /rite-vet bare",
		"frontier-concurrent: /rite-vet bare",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q:\n%s", want, text)
		}
	}
}

func TestAutocompleteSourceGODoesNotStopOtherWorkspaces(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 5\n")
	dir := frontierFixture(t, root, "work", "hold", "", "seal", "awaiting_human", "/rite-seal hold", 1, "1")
	writeFile(t, filepath.Join(dir, "questions.md"), "## q-go\nstatus: open\ngate: blocking\n")
	frontierFixture(t, root, "work", "other", "hold", "plan", "blocked", "/rite-plan repair other", 2, "0")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"hold"}, &out, &errOut); code != 0 {
		t.Fatalf("code=%d: %s", code, &errOut)
	}
	text := out.String()
	for _, want := range []string{
		"continuation: required\n",
		"launch: /rite-plan repair other\n",
		"a source GO does not end autocomplete",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "continuation: clear") {
		t.Fatalf("source GO ended the run:\n%s", text)
	}
}

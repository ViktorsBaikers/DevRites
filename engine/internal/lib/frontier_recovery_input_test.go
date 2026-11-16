package lib

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLegacyPreparationDeferralSurvivesSequenceConflict(t *testing.T) {
	for _, owner := range []string{"alpha", "beta", "gamma"} {
		t.Run(owner, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			t.Setenv("DEVRITES_HOST_CAPACITY", "10")
			writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 10\n")
			frontierFixture(t, root, "work", "alpha", "", "plan", "blocked", "/rite-vet alpha", 1, "0")
			frontierFixture(t, root, "work", "beta", "alpha", "plan", "blocked", "/rite-vet beta", 2, "0")
			frontierFixture(t, root, "work", "gamma", "beta", "plan", "blocked", "/rite-vet gamma", 3, "0")
			writeFile(t, filepath.Join(root, "work", "alpha", "decisions.md"), "| Position | Workspace |\n| --- | --- |\n| 1 | alpha |\n| 2 | beta |\n| 3 | gamma |\n")
			writeFile(t, filepath.Join(root, "work", "gamma", "decisions.md"), "| Position | Workspace |\n| --- | --- |\n| 1 | alpha |\n| 2 | gamma |\n")
			for _, source := range []string{"alpha", "beta", "gamma"} {
				writeFile(t, filepath.Join(root, "work", source, "evidence.md"), "frontier-blocked: sequence — human holds map normalization\n")
			}
			ownerPath := filepath.Join(root, "work", owner, "evidence.md")
			original := readTestFile(t, ownerPath)
			writeFile(t, ownerPath, original+"frontier-blocked: beta — producer contract held; owner must publish before re-evaluation\n")
			before := controllerSnapshot(t, root)
			for _, source := range []string{"alpha", "beta"} {
				var out, errOut bytes.Buffer
				if code := RunCheckContinuation(root, []string{source}, &out, &errOut); code != 0 {
					t.Fatalf("continuation: %d %s", code, &errOut)
				}
				text := out.String()
				if !strings.Contains(text, "frontier-blocked: beta — owner="+owner) || !strings.Contains(text, "launch: /rite-vet gamma\n") || strings.Contains(text, "launch: /rite-vet beta\n") {
					t.Fatalf("held source repeated or independent work lost:\n%s", text)
				}
			}
			packet := reviewerPacketPath(t)
			if code, out := controllerDispatch(t, root, "beta", "open", "--phase", "vet", "--wave", "repeat", "--role", "plan-reviewer", "--packet", packet); code != 3 || !strings.Contains(out, "frontier-blocked: beta") {
				t.Fatalf("ordinary admission bypassed deferral: %d %s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("observation or refused admission changed workflow state")
			}
			writeFile(t, ownerPath, original)
			if text := frontierText(t, root, "alpha"); !strings.Contains(text, "frontier-concurrent: /rite-vet beta") {
				t.Fatalf("resolved deferral did not re-admit preparation:\n%s", text)
			}
		})
	}
}

func TestAutocompleteMissingRepairInputRoutesToRoot(t *testing.T) {
	for _, input := range []string{"missing", "empty"} {
		t.Run(input, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 10\n")
			dir := frontierFixture(t, root, "work", "repair-source", "", "plan", "blocked", "/rite-plan repair repair-source", 1, "0")
			if input == "empty" {
				writeFile(t, filepath.Join(dir, "drift.md"), " \n\t")
			}
			before := controllerSnapshot(t, root)
			for _, run := range []func(string, []string, *bytes.Buffer, *bytes.Buffer) int{
				func(root string, args []string, out, errOut *bytes.Buffer) int {
					return RunNext(root, args, out, errOut)
				},
				func(root string, args []string, out, errOut *bytes.Buffer) int {
					return RunCheckContinuation(root, args, out, errOut)
				},
			} {
				var out, errOut bytes.Buffer
				if code := run(root, []string{"repair-source"}, &out, &errOut); code != 0 {
					t.Fatalf("observe: %d %s", code, &errOut)
				}
				if !strings.Contains(out.String(), "repair-input: /rite-plan repair repair-source\n") || strings.Contains(out.String(), "launch: /rite-plan repair") || strings.Contains(out.String(), "continuation: clear") {
					t.Fatalf("missing input became a worker launch or unexplained stop:\n%s", &out)
				}
			}
			if code, out := controllerDispatch(t, root, "repair-source", "open", "--phase", "plan", "--wave", "repeat", "--role", "plan-drafter"); code != 3 || !strings.Contains(out, "repair-input:") {
				t.Fatalf("missing input admitted another drafter: %d %s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("preflight changed workflow state or invented drift")
			}
			writeFile(t, filepath.Join(dir, "drift.md"), "## DRIFT-001\nRecorded review found an unowned proof; repair its existing slice binding.\n")
			if code, out := controllerDispatch(t, root, "repair-source", "open", "--phase", "plan", "--wave", "repaired-input", "--role", "plan-drafter"); code != 0 {
				t.Fatalf("evidenced repair input not admitted: %d %s", code, out)
			}
		})
	}
}

func TestAutocompleteExplicitDeferralStopsWithoutRecursiveNext(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 10\n")
	dir := frontierFixture(t, root, "work", "held", "", "plan", "blocked", "/rite-vet held", 1, "0")
	writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: held — owner must publish producer contract before Vet\n")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"held"}, &out, &errOut); code != 0 {
		t.Fatalf("continuation: %d %s", code, &errOut)
	}
	if !strings.Contains(out.String(), "continuation: clear\n") || !strings.Contains(out.String(), "owner must publish producer contract") || strings.Contains(out.String(), "invoke:") || strings.Contains(out.String(), "launch:") {
		t.Fatalf("explicit hold still required recursive continuation:\n%s", &out)
	}
	out.Reset()
	errOut.Reset()
	if code := RunNext(root, []string{"held"}, &out, &errOut); code != 0 {
		t.Fatalf("next: %d %s", code, &errOut)
	}
	if strings.Contains(out.String(), "next-command:") || strings.Contains(out.String(), "launch:") {
		t.Fatalf("local next fallback escaped explicit hold:\n%s", &out)
	}
}

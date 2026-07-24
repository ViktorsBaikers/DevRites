package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/toolpolicy"
)

func TestGitClassifierReasonIDsBelongToDiagnosticCatalog(t *testing.T) {
	ids := []toolpolicy.ReasonID{
		toolpolicy.ReasonInputTooLarge,
		toolpolicy.ReasonAmbiguousSyntax,
		toolpolicy.ReasonAmbiguousDynamic,
		toolpolicy.ReasonAmbiguousAlias,
		toolpolicy.ReasonAmbiguousGlobalOption,
		toolpolicy.ReasonAmbiguousStdin,
		toolpolicy.ReasonDestructiveReset,
		toolpolicy.ReasonDestructiveClean,
		toolpolicy.ReasonDestructiveCheckout,
		toolpolicy.ReasonDestructiveRestore,
		toolpolicy.ReasonDestructiveSwitch,
		toolpolicy.ReasonDestructiveRemove,
		toolpolicy.ReasonDestructiveBranch,
		toolpolicy.ReasonDestructiveTag,
		toolpolicy.ReasonDestructiveUpdateRef,
		toolpolicy.ReasonDestructiveStash,
		toolpolicy.ReasonDestructiveReflog,
		toolpolicy.ReasonDestructivePrune,
		toolpolicy.ReasonDestructiveHistory,
		toolpolicy.ReasonDestructivePushForce,
		toolpolicy.ReasonDestructivePushDelete,
		toolpolicy.ReasonDestructiveWorktree,
		toolpolicy.ReasonDestructiveRefDeletion,
	}
	for _, id := range ids {
		if _, err := reason.Parse(string(id)); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestGitGuardOrdinaryGitPassesSilently(t *testing.T) {
	for _, command := range []string{
		"git status --short",
		"git diff -- reset",
		"git add file && git commit -m safe",
		"printf '%s\\n' 'git reset --hard HEAD'",
	} {
		var stdout, stderr bytes.Buffer
		if code := hookGitGuard(harness.Claude, gitGuardInput("Bash", command), &stdout, &stderr); code != 0 {
			t.Fatalf("%q code=%d", command, code)
		}
		if stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("%q emitted stdout=%q stderr=%q", command, stdout.String(), stderr.String())
		}
	}
}

func TestGitGuardAmbiguousHighImpactAlwaysDenies(t *testing.T) {
	for _, host := range []harness.Harness{harness.Claude, harness.Codex} {
		t.Run(string(host), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := hookGitGuard(host, gitGuardInput("Bash", "eval 'git reset --hard HEAD'"), &stdout, &stderr); code != 0 {
				t.Fatalf("code=%d", code)
			}
			if !strings.Contains(stdout.String(), `"permissionDecision":"deny"`) ||
				!strings.Contains(stdout.String(), string(reason.GitAmbiguousDynamic)) ||
				!strings.Contains(stdout.String(), "direct, literal git command") {
				t.Fatalf("stdout=%q", stdout.String())
			}
			if strings.Contains(stdout.String(), "reset --hard") || stderr.Len() != 0 {
				t.Fatalf("guard leaked command or stderr: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestGitGuardOpensResolvesAndConsumesExactQuestionForBothHosts(t *testing.T) {
	var hostReasons []string
	for _, host := range []harness.Harness{harness.Claude, harness.Codex} {
		t.Run(string(host), func(t *testing.T) {
			root := makeHookWorkspace(t)
			work := filepath.Join(root, "work", "demo")
			if err := os.WriteFile(filepath.Join(work, "questions.md"), []byte("# Questions\n\nNone.\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
			const command = "git reset --hard SECRET-REF"

			var denied, deniedErr bytes.Buffer
			if code := hookGitGuard(host, gitGuardInput("Bash", command), &denied, &deniedErr); code != 0 {
				t.Fatalf("pending code=%d", code)
			}
			if !strings.Contains(denied.String(), `"permissionDecision":"deny"`) ||
				!strings.Contains(denied.String(), string(reason.GitAuthorityPending)) {
				t.Fatalf("pending stdout=%q", denied.String())
			}
			questionData, err := os.ReadFile(filepath.Join(work, "questions.md"))
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(questionData, []byte("SECRET-REF")) || bytes.Contains(questionData, []byte("reset --hard")) {
				t.Fatalf("question leaked command: %s", questionData)
			}
			qid := gitGuardQuestionID(t, questionData)

			t.Setenv("DEVRITES_NOW", "2026-07-23T10:01:00Z")
			var resolveOut, resolveErr bytes.Buffer
			if code := lib.Resolve(root, []string{qid, lib.GitAuthorityAnswer}, &resolveOut, &resolveErr); code != 0 {
				t.Fatalf("resolve code=%d stderr=%q", code, resolveErr.String())
			}

			var allowed, allowedErr bytes.Buffer
			if code := hookGitGuard(host, gitGuardInput("Bash", command), &allowed, &allowedErr); code != 0 {
				t.Fatalf("allow code=%d", code)
			}
			if !strings.Contains(allowed.String(), `"permissionDecision":"allow"`) ||
				!strings.Contains(allowed.String(), string(reason.GitAuthorityGranted)) {
				t.Fatalf("allow stdout=%q", allowed.String())
			}
			ledger, err := os.ReadFile(filepath.Join(work, lib.GitAuthorityLedgerFile))
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(ledger, []byte("SECRET-REF")) || bytes.Contains(ledger, []byte("reset --hard")) {
				t.Fatalf("ledger leaked command: %s", ledger)
			}
			timeline, err := os.ReadFile(filepath.Join(root, "timeline.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(timeline, []byte("SECRET-REF")) || bytes.Contains(timeline, []byte("reset --hard")) {
				t.Fatalf("timeline leaked command: %s", timeline)
			}
			events := readV1HookEvents(t, root)
			if len(events) != 2 {
				t.Fatalf("events=%+v", events)
			}
			for _, event := range events {
				if event.Boundary != lib.BoundaryGitPolicy || event.Event != "git-guard" ||
					event.Host != lib.EventHost(host) || event.GuardStrength != lib.GuardEnforced {
					t.Fatalf("event=%+v", event)
				}
			}
			hostReasons = append(hostReasons, string(events[0].ReasonID)+"/"+string(events[0].Outcome))
		})
	}
	if len(hostReasons) != 2 || hostReasons[0] != hostReasons[1] {
		t.Fatalf("host policy drift: %v", hostReasons)
	}
}

func TestGitGuardWrongDigestAndNoWorkspaceDeny(t *testing.T) {
	root := makeHookWorkspace(t)
	work := filepath.Join(root, "work", "demo")
	if err := os.WriteFile(filepath.Join(work, "questions.md"), []byte("# Questions\n\nNone.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")

	var first bytes.Buffer
	hookGitGuard(harness.Claude, gitGuardInput("Bash", "git reset --hard HEAD"), &first, &bytes.Buffer{})
	qdata, err := os.ReadFile(filepath.Join(work, "questions.md"))
	if err != nil {
		t.Fatal(err)
	}
	qid := gitGuardQuestionID(t, qdata)
	var resolveOut, resolveErr bytes.Buffer
	if code := lib.Resolve(root, []string{qid, lib.GitAuthorityAnswer}, &resolveOut, &resolveErr); code != 0 {
		t.Fatalf("resolve code=%d stderr=%q", code, resolveErr.String())
	}
	var mismatch bytes.Buffer
	hookGitGuard(harness.Claude, gitGuardInput("Bash", "git clean -fd"), &mismatch, &bytes.Buffer{})
	if !strings.Contains(mismatch.String(), `"permissionDecision":"deny"`) ||
		strings.Contains(mismatch.String(), `"permissionDecision":"allow"`) {
		t.Fatalf("wrong digest stdout=%q", mismatch.String())
	}

	t.Setenv("DEVRITES_ROOT", filepath.Join(t.TempDir(), ".devrites"))
	if err := os.MkdirAll(os.Getenv("DEVRITES_ROOT"), 0o755); err != nil {
		t.Fatal(err)
	}
	var absent bytes.Buffer
	hookGitGuard(harness.Claude, gitGuardInput("Bash", "git reset --hard HEAD"), &absent, &bytes.Buffer{})
	if !strings.Contains(absent.String(), string(reason.GitWorkspaceUnavailable)) ||
		!strings.Contains(absent.String(), `"permissionDecision":"deny"`) {
		t.Fatalf("no-workspace stdout=%q", absent.String())
	}
}

func TestGitGuardEventFailureNeverChangesConsumedAllow(t *testing.T) {
	root := makeHookWorkspace(t)
	work := filepath.Join(root, "work", "demo")
	if err := os.WriteFile(filepath.Join(work, "questions.md"), []byte("# Questions\n\nNone.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
	const command = "git reset --hard HEAD"
	var pending bytes.Buffer
	hookGitGuard(harness.Claude, gitGuardInput("Bash", command), &pending, &bytes.Buffer{})
	qdata, err := os.ReadFile(filepath.Join(work, "questions.md"))
	if err != nil {
		t.Fatal(err)
	}
	qid := gitGuardQuestionID(t, qdata)
	var resolveOut, resolveErr bytes.Buffer
	if code := lib.Resolve(root, []string{qid, lib.GitAuthorityAnswer}, &resolveOut, &resolveErr); code != 0 {
		t.Fatalf("resolve code=%d stderr=%q", code, resolveErr.String())
	}
	timeline := filepath.Join(root, "timeline.jsonl")
	if err := os.Remove(timeline); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(timeline, 0o755); err != nil {
		t.Fatal(err)
	}

	var allowed bytes.Buffer
	hookGitGuard(harness.Claude, gitGuardInput("Bash", command), &allowed, &bytes.Buffer{})
	if !strings.Contains(allowed.String(), `"permissionDecision":"allow"`) {
		t.Fatalf("telemetry failure changed allow: %q", allowed.String())
	}
	if _, err := os.Stat(filepath.Join(work, lib.GitAuthorityLedgerFile)); err != nil {
		t.Fatalf("grant was not consumed before telemetry: %v", err)
	}
}

func TestGitGuardDispatchAndKillPath(t *testing.T) {
	root := makeHookWorkspace(t)
	if err := os.WriteFile(filepath.Join(root, "work", "demo", "questions.md"), []byte("# Questions\n\nNone.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{
		"tool_name":  "Bash",
		"tool_input": map[string]string{"command": "git reset --hard HEAD"},
	})

	t.Setenv("DEVRITES_HOOK_PROFILE", "standard")
	t.Setenv("DEVRITES_DISABLED_HOOKS", "git-guard")
	var disabled bytes.Buffer
	if code := run([]string{"hook", "git-guard", "--harness=claude"}, bytes.NewReader(payload), &disabled, &bytes.Buffer{}); code != 0 {
		t.Fatalf("disabled code=%d", code)
	}
	if disabled.Len() != 0 {
		t.Fatalf("disabled hook blocked: %q", disabled.String())
	}

	t.Setenv("DEVRITES_DISABLED_HOOKS", "")
	var enabled bytes.Buffer
	if code := run([]string{"hook", "git-guard", "--harness=claude"}, bytes.NewReader(payload), &enabled, &bytes.Buffer{}); code != 0 {
		t.Fatalf("enabled code=%d", code)
	}
	if !strings.Contains(enabled.String(), `"permissionDecision":"deny"`) {
		t.Fatalf("enabled hook did not deny: %q", enabled.String())
	}
}

func gitGuardInput(tool, command string) *bytes.Reader {
	data, _ := json.Marshal(map[string]any{
		"tool_name":  tool,
		"tool_input": map[string]string{"command": command},
	})
	return bytes.NewReader(data)
}

func gitGuardQuestionID(t *testing.T, data []byte) string {
	t.Helper()
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "## q-") {
			return strings.TrimPrefix(line, "## ")
		}
	}
	t.Fatalf("question id missing:\n%s", data)
	return ""
}

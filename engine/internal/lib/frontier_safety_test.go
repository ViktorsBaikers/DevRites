package lib

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/parallel"
)

func frontierSelectText(t *testing.T, root, workspaces, capN string) string {
	t.Helper()
	saved := parallel.WorkspaceReady
	parallel.WorkspaceReady = WorkspaceReady
	t.Cleanup(func() { parallel.WorkspaceReady = saved })
	var out, errOut bytes.Buffer
	code := parallel.Run("parallel", []string{"select", "--root", filepath.Dir(root), "--slug", "parent", "--from-plan", "--workspaces", workspaces, "--cap", capN}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("select code=%d: %s%s", code, &out, &errOut)
	}
	return out.String()
}

func frontierDispatch(t *testing.T, root string, wantCode int, args ...string) string {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := RunDispatch(root, append([]string{"parent"}, args...), &out, &errOut); code != wantCode {
		t.Fatalf("dispatch %v code=%d, want %d: %s%s", args, code, wantCode, &out, &errOut)
	}
	return out.String()
}

func TestFrontierGlobalCapacityIgnoresLegacySliceBudgets(t *testing.T) {
	for _, budget := range []string{"1", "0"} {
		t.Run("remaining="+budget, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
			var tasks strings.Builder
			for i := 1; i <= 5; i++ {
				tasks.WriteString(poolSlice(fmt.Sprintf("SLICE-%03d", i), fmt.Sprintf("src/child-%d.go", i), "Workspace dependencies: none"))
			}
			child := poolWorkspace(t, root, "child", "parent", tasks.String())
			writeFile(t, filepath.Join(parent, "state.md"), cursor("build", "running", "/rite-build SLICE-001")+"| afk_slices_remaining | "+budget+" |\n")
			writeFile(t, filepath.Join(child, "state.md"), cursor("build", "running", "/rite-build SLICE-001")+"| sequence_parent | parent |\n| afk_slices_remaining | "+budget+" |\n")
			writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
			const picks = "SLICE-001,child/SLICE-001,child/SLICE-002,child/SLICE-003,child/SLICE-004"
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"parent"}, &out, &errOut); code != 0 {
				t.Fatalf("continuation code=%d: %s%s", code, &out, &errOut)
			}
			for _, want := range []string{"continuation: required\n", "frontier: actionable\n", "--workspaces 'parent,child' --cap 5\n", "frontier-picks: " + picks + "\n"} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("global capacity throttled by legacy budget; missing %q:\n%s", want, &out)
				}
			}
			if got := frontierSelectText(t, root, "parent,child", "5"); got != "select: n_eff=5 mode=parallel slices="+picks+"\n" {
				t.Fatalf("global five-slot selection changed: %s", got)
			}
		})
	}
}

func TestFrontierCompletedOwnerWithZeroBudgetKeepsPreparedChild(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	parent := filepath.Join(root, "work", "parent")
	writeFile(t, filepath.Join(parent, "state.md"), cursor("done", "shipped", "")+"| sequence_position | 1 |\n| sequence_workspaces_remaining | 0 |\n| afk_slices_remaining | 0 |\n")
	child := poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	writeFile(t, filepath.Join(child, "state.md"), cursor("build", "running", "/rite-build SLICE-001")+"| sequence_parent | parent |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 0 |\n| afk_slices_remaining | 0 |\n")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\ncontinue_sequence: true\n")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"parent"}, &out, &errOut); code != 0 {
		t.Fatalf("continuation code=%d: %s%s", code, &out, &errOut)
	}
	for _, want := range []string{"continuation: required\n", "invoke: /rite-autocomplete\n", "parent location=live action=complete ", "--slug 'parent' --from-plan --workspaces 'child' --cap 5\n", "frontier-picks: child/SLICE-001\n"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("completed owner hid prepared child; missing %q:\n%s", want, &out)
		}
	}
	if strings.Contains(out.String(), "frontier-prepare:") || strings.Contains(out.String(), "invoke: /rite-build") {
		t.Errorf("exhausted sequence opened work or bypassed global controller:\n%s", &out)
	}
	if got := frontierSelectText(t, root, "child", "5"); got != "select: n_eff=1 mode=serial slices=child/SLICE-001\n" {
		t.Fatalf("prepared child not selected independently: %s", got)
	}
}

func TestFrontierContradictoryStatusCannotInvokeBuild(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
	writeFile(t, filepath.Join(parent, "state.md"), cursor("vet", "ready", "/rite-build SLICE-001")+"| status | awaiting_human |\n")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"parent"}, &out, &errOut); code != 0 {
		t.Fatalf("continuation code=%d: %s%s", code, &out, &errOut)
	}
	text := out.String()
	if !strings.Contains(text, "duplicate status") || !strings.Contains(text, "frontier: blocked\n") {
		t.Errorf("contradictory native authority not surfaced:\n%s", text)
	}
	for _, forbidden := range []string{"invoke: /rite-build", "frontier-picks:", "frontier-select-command:"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("first ready status overrode awaiting_human; emitted %q:\n%s", forbidden, text)
		}
	}
}

func TestFrontierPendingDispatchRequiresHostTerminal(t *testing.T) {
	for _, reason := range []string{"", "writer termination unknown", "host confirmed terminal"} {
		name := reason
		if name == "" {
			name = "sealed writer with slice IDs only"
		}
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
			poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
			writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
			frontierDispatch(t, root, 0, "open", "--phase", "build", "--wave", "writer", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t))
			frontierDispatch(t, root, 0, "start", "--wave", "writer", "--role", "slice-wright", "--handle", "host-writer")
			if reason != "" {
				frontierDispatch(t, root, 3, "abandon", "--wave", "writer", "--reason", reason)
			}
			text := frontierText(t, root, "parent") + "\n"
			for _, want := range []string{"parent location=live action=wait ", "--workspaces 'child' --cap 5\n", "frontier-picks: child/SLICE-001\n"} {
				if !strings.Contains(text, want) {
					t.Errorf("unconfirmed writer not withheld; missing %q:\n%s", want, text)
				}
			}
			if got := frontierSelectText(t, root, "child", "5"); got != "select: n_eff=1 mode=serial slices=child/SLICE-001\n" {
				t.Fatalf("child-only selection relaunched unknown source: %s", got)
			}
			frontierDispatch(t, root, 3, "abandon", "--wave", "writer", "--reason", "host confirmed terminal", "--release")
			text = frontierText(t, root, "parent") + "\n"
			for _, want := range []string{"--workspaces 'parent,child' --cap 5\n", "frontier-picks: SLICE-001,child/SLICE-001\n"} {
				if !strings.Contains(text, want) {
					t.Errorf("host-confirmed terminal did not release source; missing %q:\n%s", want, text)
				}
			}
			if got := frontierSelectText(t, root, "parent,child", "5"); got != "select: n_eff=2 mode=parallel slices=SLICE-001,child/SLICE-001\n" {
				t.Fatalf("terminal source not released: %s", got)
			}
		})
	}
}

func TestFrontierMalformedDispatchCannotReleaseSource(t *testing.T) {
	for _, tc := range []struct{ name, dispatch, diagnostic string }{
		{
			"abandoned with invalid terminal timestamp",
			`{"waves":{"writer":{"phase":"build","state":"abandoned","slices":["SLICE-001"],"abandoned_at":"2026-11-03T10:01:00Z","terminal_at":"host says stopped","roles":{"slice-wright":{"handle":"host-writer","started":"2026-11-03T10:00:00Z","returned":false}}}}}`,
			"invalid terminal timestamp",
		},
		{
			"complete with unstarted returned account",
			`{"waves":{"writer":{"phase":"build","state":"complete","slices":["SLICE-001"],"roles":{"slice-wright":{"returned":true,"returned_at":"2026-11-03T10:01:00Z"}}}}}`,
			"inconsistent start or return",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", "")+"\n## SLICE-002 Next\nDependencies: SLICE-001\nWriter allowlist: src/next.go\n")
			writeFile(t, filepath.Join(parent, "state.md"), cursor("build", "running", "/rite-build SLICE-002")+"\n- [x] SLICE-001 built — abc1234\n")
			poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
			writeFile(t, filepath.Join(parent, "dispatch.json"), tc.dispatch)
			writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
			var out, errOut bytes.Buffer
			if code := RunDispatch(root, []string{"parent", "status"}, &out, &errOut); code != 3 || !strings.Contains(errOut.String(), tc.diagnostic) {
				t.Fatalf("native dispatch accepted malformed fixture: code=%d stdout=%s stderr=%s", code, &out, &errOut)
			}
			text := frontierText(t, root, "parent") + "\n"
			for _, want := range []string{"parent location=live action=repair ", tc.diagnostic, "--workspaces 'child' --cap 5\n", "frontier-picks: child/SLICE-001\n"} {
				if !strings.Contains(text, want) {
					t.Errorf("malformed dispatch bypassed native authority; missing %q:\n%s", want, text)
				}
			}
			if got := frontierSelectText(t, root, "child", "5"); got != "select: n_eff=1 mode=serial slices=child/SLICE-001\n" {
				t.Fatalf("malformed source leaked into child-only selection: %s", got)
			}
		})
	}
}

func TestFrontierCompletedWriterNeedsNativeBuiltAcceptance(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".devrites")
	parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", "")+"\n## SLICE-002 Next\nDependencies: SLICE-001\nWriter allowlist: src/next.go\n")
	poolWorkspace(t, root, "child", "parent", poolSlice("SLICE-001", "src/child.go", "Workspace dependencies: none"))
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
	frontierDispatch(t, root, 0, "open", "--phase", "build", "--wave", "writer", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t))
	frontierDispatch(t, root, 0, "start", "--wave", "writer", "--role", "slice-wright", "--handle", "host-writer")
	returned := frontierDispatch(t, root, 0, "return", "--wave", "writer", "--role", "slice-wright", "--handle", "host-writer")
	if !strings.Contains(returned, "wave writer complete") {
		t.Fatalf("native final return did not complete dispatch: %s", returned)
	}
	text := frontierText(t, root, "parent") + "\n"
	for _, want := range []string{"parent location=live action=inspect ", "--workspaces 'child' --cap 5\n", "frontier-picks: child/SLICE-001\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("completed return bypassed native acceptance; missing %q:\n%s", want, text)
		}
	}
	if got := frontierSelectText(t, root, "child", "5"); got != "select: n_eff=1 mode=serial slices=child/SLICE-001\n" {
		t.Fatalf("unaccepted writer selected again: %s", got)
	}
	writeFile(t, filepath.Join(parent, "state.md"), cursor("build", "running", "/rite-build SLICE-002")+"\n- [x] SLICE-001 built — abc1234\n")
	text = frontierText(t, root, "parent") + "\n"
	for _, want := range []string{"--workspaces 'parent,child' --cap 5\n", "frontier-picks: SLICE-002,child/SLICE-001\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("historical accepted dispatch blocked new work; missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "parent location=live action=inspect ") {
		t.Errorf("native BUILT acceptance left obsolete dispatch hold:\n%s", text)
	}
	if got := frontierSelectText(t, root, "parent,child", "5"); got != "select: n_eff=2 mode=parallel slices=SLICE-002,child/SLICE-001\n" {
		t.Fatalf("accepted historical slice blocked next candidate: %s", got)
	}
}

func TestFrontierCanonicalAFKCommentsPreserveSequenceBudget(t *testing.T) {
	for _, tc := range []struct {
		name, policy, remaining string
		prepare                 bool
	}{
		{"canonical template", "continue_sequence: true              # after Seal GO, open the next recorded continuation\n", "1", true},
		{"commented opt out", "continue_sequence: false              # after Seal GO, open the next recorded continuation\n", "1", false},
		{"opening budget exhausted", "continue_sequence: true              # after Seal GO, open the next recorded continuation\n", "0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ".devrites")
			parent := poolWorkspace(t, root, "parent", "", poolSlice("SLICE-001", "src/parent.go", ""))
			writeFile(t, filepath.Join(parent, "state.md"), cursor("build", "running", "/rite-spec absent")+"| sequence_position | 1 |\n| sequence_workspaces_remaining | "+tc.remaining+" |\n")
			writeFile(t, filepath.Join(root, "AFK"), tc.policy+"max_parallel: 10 # unattended default cap; `--parallel N` on this invocation wins and may rewrite only this field\n")
			text := frontierText(t, root, "parent") + "\n"
			for _, want := range []string{"frontier: actionable\n", "--workspaces 'parent' --cap 10\n", "frontier-picks: SLICE-001\n"} {
				if !strings.Contains(text, want) {
					t.Errorf("documented AFK comments rejected; missing %q:\n%s", want, text)
				}
			}
			if got := strings.Contains(text, "frontier-prepare: /rite-spec 'absent' source=parent reason=read recorded sequence authority before preparation; no workspace created"); got != tc.prepare {
				t.Errorf("sequence preparation=%t, want %t:\n%s", got, tc.prepare, text)
			}
			if !tc.prepare && strings.Contains(text, "frontier-prepare:") {
				t.Errorf("sequence opening prohibition bypassed:\n%s", text)
			}
		})
	}
}

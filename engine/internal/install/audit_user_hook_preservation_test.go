package install

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func TestAuditUserHookPreservationLifecycle(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	payload := testPayload(t)
	source := testSource(t, "4.0.0")
	userCommands := []string{
		`printf '%s\n' 'DevRites: custom user hook'`,
		`printf '%s\n' 'devrites-engine hook stop-gate --harness=claude'`,
		`echo 'command -v devrites-engine >/dev/null 2>&1 && exec devrites-engine hook stop-gate --harness=claude || exit 0'`,
		`printf '%s\n' 'bash "$CLAUDE_PROJECT_DIR/.claude/hooks/devrites-allow.sh"'`,
		`echo '.codex/hooks/devrites-stop-gate.sh'`,
		`devrites-engine hook stop-gate; echo user-owned-tail`,
		`devrites-engine hook user-custom`,
		`bash "$CLAUDE_PROJECT_DIR/.claude/hooks/devrites-custom-user.sh"`,
	}
	// Historical Claude/Codex wiring plus the existing legacy old-gate fixture.
	legacyCommands := []string{
		`devrites-engine hook old-gate --harness=claude`,
		`command -v devrites-engine >/dev/null 2>&1 && exec devrites-engine hook stop-gate --harness=claude || exit 0`,
		`command -v devrites-engine >/dev/null 2>&1 && exec devrites-engine hook event subagent-stop || exit 0`,
		`command -v devrites-engine >/dev/null 2>&1 || exit 0; cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)" 2>/dev/null || exit 0; exec devrites-engine hook stop-gate --harness=codex`,
		`command -v devrites-engine >/dev/null 2>&1 || exit 0; cd "$(git rev-parse --show-toplevel 2>/dev/null || pwd)" 2>/dev/null || exit 0; exec env DEVRITES_REVIEWER_AGENT_REQUIRED=1 devrites-engine hook reviewer-readonly --harness=codex`,
		`bash "$CLAUDE_PROJECT_DIR/.claude/hooks/devrites-allow.sh"`,
		`bash "${CLAUDE_PLUGIN_ROOT}/pack/.claude/hooks/devrites-stop-gate.sh"`,
		`root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"; cd "$root" && bash "$root/.codex/hooks/devrites-stop-gate.sh"`,
		`root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"; cd "$root" && DEVRITES_WRIGHT_AGENT_REQUIRED=1 bash "$root/.codex/hooks/devrites-wright-scope.sh"`,
		`printf '%s\n' 'DevRites: this subagent is the single write-capable slice-wright. Stay within the current slice contract and touched-files.md; write no .devrites bookkeeping.'`,
		`printf '%s\n' 'DevRites: this subagent is read-only. Inspect with read/search/test commands and return findings only; do not mutate the tree.'`,
		`printf '%s\n' 'DevRites: reconcile this subagent result against the active DevRites skill contract before claiming completion.'`,
	}
	for _, tc := range []struct {
		name       string
		statusLine string
		owned      bool
	}{
		{name: "custom status", statusLine: `printf '%s\n' 'DevRites: custom user status'`},
		{name: "quoted engine status", statusLine: `printf '%s\n' 'devrites-engine hook statusline --harness=claude'`},
		{name: "legacy status", statusLine: `command -v devrites-engine >/dev/null 2>&1 && exec devrites-engine hook statusline --harness=claude || exit 0`, owned: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := t.TempDir()
			path := filepath.Join(target, ".claude", "settings.json")
			var commands, wantCommands []any
			for _, command := range userCommands {
				hook := map[string]any{"type": "command", "command": command}
				commands = append(commands, hook)
				wantCommands = append(wantCommands, hook)
			}
			for _, command := range legacyCommands {
				commands = append(commands, map[string]any{"type": "command", "command": command})
			}
			status := map[string]any{"type": "command", "command": tc.statusLine}
			wantHooks := map[string]any{"Stop": []any{map[string]any{"matcher": "user-matcher", "hooks": wantCommands}}}
			settings := map[string]any{
				"statusLine": status,
				"hooks":      map[string]any{"Stop": []any{map[string]any{"matcher": "user-matcher", "hooks": commands}}},
			}
			data, err := json.Marshal(settings)
			if err != nil {
				t.Fatal(err)
			}
			testutil.WriteFile(t, path, string(data)+"\n")
			for _, stage := range []string{"install", "refresh", "uninstall"} {
				switch stage {
				case "install":
					runInstall(t, target, payload, func(*Options) {})
				case "refresh":
					runLocalUpdate(t, target, source, payload)
				case "uninstall":
					runUninstall(t, target)
				}
				var got map[string]any
				if err := json.Unmarshal([]byte(testutil.ReadFile(t, path)), &got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got["hooks"], wantHooks) {
					t.Errorf("%s hooks = %#v, want preserved user hooks %#v and no legacy hooks", stage, got["hooks"], wantHooks)
				}
				if tc.owned {
					if _, exists := got["statusLine"]; exists {
						t.Errorf("%s retained legacy statusLine: %#v", stage, got["statusLine"])
					}
				} else if !reflect.DeepEqual(got["statusLine"], status) {
					t.Errorf("%s statusLine = %#v, want preserved user status %#v", stage, got["statusLine"], status)
				}
			}
		})
	}
}

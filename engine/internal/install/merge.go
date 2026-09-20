package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/hostpack"
)

func (r *runner) installClaudeSettingsMarker(ownsDefaultMode bool, managed map[string]bool) error {
	ownership := "preexisting"
	if ownsDefaultMode {
		ownership = "added"
	}
	var b strings.Builder
	b.WriteString(hostpack.ClaudeSettingsMerge.MarkerText + "\ndefault-mode=" + ownership)
	rules := make([]string, 0, len(managed))
	for rule := range managed {
		rules = append(rules, rule)
	}
	sort.Strings(rules)
	for _, rule := range rules {
		b.WriteString("\nrule:" + rule)
	}
	return r.installMarker(hostpack.ClaudeSettingsMerge.MarkerRel, b.String())
}

func (r *runner) claudeDefaultModeOwned() bool {
	data, err := os.ReadFile(filepath.Join(r.target, filepath.FromSlash(hostpack.ClaudeSettingsMerge.MarkerRel)))
	return err == nil && strings.Contains(string(data), "\ndefault-mode=added\n")
}

// managedClaudePermissionRules returns the exact non-engine allow rules
// DevRites manages in .claude/settings.json: rules recorded in the merge
// marker by past installs, unioned with the current payload's non-engine
// rules so newly added and retired grants both resolve correctly. Engine
// command rules are matched by prefix and never need listing here.
func (r *runner) managedClaudePermissionRules() map[string]bool {
	managed := map[string]bool{}
	data, err := os.ReadFile(filepath.Join(r.target, filepath.FromSlash(hostpack.ClaudeSettingsMerge.MarkerRel)))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if rule, ok := strings.CutPrefix(line, "rule:"); ok {
				managed[strings.TrimSpace(rule)] = true
			}
		}
	}
	payload, err := readJSONFS(r.payloadFS, hostpack.ClaudeSettingsMerge.PayloadRel)
	if err == nil {
		for rule := range managedRulesFromPayload(payload) {
			managed[rule] = true
		}
	}
	return managed
}

// managedRulesFromPayload extracts the payload's non-engine allow rules —
// the MCP/graph/git/.devrites grants a bare prefix match cannot identify.
func managedRulesFromPayload(payload map[string]any) map[string]bool {
	managed := map[string]bool{}
	permissions, _ := payload["permissions"].(map[string]any)
	allow, _ := permissions["allow"].([]any)
	for _, rule := range allow {
		if s, ok := rule.(string); ok {
			s = strings.TrimSpace(s)
			if !strings.HasPrefix(s, "Bash(devrites-engine ") {
				managed[s] = true
			}
		}
	}
	return managed
}

func (r *runner) mergeMarkerFile(merge hostpack.MarkerMerge) error {
	block, err := fs.ReadFile(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return fmt.Errorf("read payload %s: %w", merge.PayloadRel, err)
	}
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	if r.opts.DryRun {
		verb := "create DevRites block"
		// #nosec G304 -- dest joins the operator target with a fixed manifest-relative record
		current, readErr := os.ReadFile(dest)
		if readErr == nil {
			if bytes.Contains(current, []byte(merge.Begin)) {
				verb = "refresh DevRites block"
			} else {
				verb = "append DevRites block"
			}
		} else if !errors.Is(readErr, fs.ErrNotExist) {
			return fmt.Errorf("cannot read %s: %w", merge.TargetRel, readErr)
		}
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s (%s)\n", merge.TargetRel, verb)
	} else {
		if err := r.recheckPath(merge.TargetRel); err != nil {
			return err
		}
		next := block
		// #nosec G304 -- dest joins the operator target with a fixed manifest-relative record
		current, readErr := os.ReadFile(dest)
		if readErr == nil {
			next = hostpack.MergeMarkerBlock(current, block, merge.Begin, merge.End)
		} else if !errors.Is(readErr, fs.ErrNotExist) {
			return fmt.Errorf("cannot read %s: %w", merge.TargetRel, readErr)
		}
		if err := fsutil.WriteFileAtomic(dest, next, 0o644); err != nil {
			return fmt.Errorf("cannot write %s: %w", merge.TargetRel, err)
		}
		// Multiple marker blocks can share one target (AGENTS.md carries both
		// the Codex and pi blocks).
		if err := r.refreshPreflight(merge.TargetRel); err != nil {
			return err
		}
	}
	return r.installMarker(merge.MarkerRel, merge.MarkerText)
}

func (r *runner) mergeCodexConfig() error {
	merge := hostpack.CodexConfigMerge
	block, err := fs.ReadFile(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return fmt.Errorf("read payload %s: %w", merge.PayloadRel, err)
	}
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s (prepend DevRites permission block)\n", merge.TargetRel)
		return r.installMarker(merge.MarkerRel, merge.MarkerText)
	}
	if err := r.recheckPath(merge.TargetRel); err != nil {
		return err
	}
	var current []byte
	// #nosec G304 -- dest joins the operator target with a fixed manifest-relative record
	if data, readErr := os.ReadFile(dest); readErr == nil {
		current = stripMarkerBlock(data, merge.Begin, merge.End)
		current = stripMarkerBlock(current, "# BEGIN DEVRITES CODEX MCP", "# END DEVRITES CODEX MCP")
		current = stripMarkerBlock(current, "### BEGIN DEVRITES CODEX MCP", "### END DEVRITES CODEX MCP")
		if hasTopLevelTOMLKey(current, "default_permissions") {
			return fmt.Errorf("%s already sets top-level default_permissions; remove that project override before installing the DevRites read-only-root profile", merge.TargetRel)
		}
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return fmt.Errorf("cannot read %s: %w", merge.TargetRel, readErr)
	}
	next := append([]byte(nil), block...)
	if len(next) == 0 || next[len(next)-1] != '\n' {
		next = append(next, '\n')
	}
	if len(bytes.TrimSpace(current)) > 0 {
		next = append(next, '\n')
		next = append(next, current...)
	}
	if err := fsutil.WriteFileAtomic(dest, next, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", merge.TargetRel, err)
	}
	return r.installMarker(merge.MarkerRel, merge.MarkerText)
}

func (r *runner) mergeClaudeSettings(merge hostpack.JSONMerge) error {
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	devrites, err := readJSONFS(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return fmt.Errorf("load Claude settings payload: %w", err)
	}
	current := map[string]any{}
	if data, readErr := readJSON(dest); readErr == nil {
		current = data
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return fmt.Errorf("load existing Claude settings: %w", readErr)
	}
	next, ownsDefaultMode, err := mergeClaudeSettingsConfig(current, devrites, r.claudeDefaultModeOwned(), r.managedClaudePermissionRules())
	if err != nil {
		return err
	}
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s\n", merge.DryRunText)
		return r.installClaudeSettingsMarker(ownsDefaultMode, managedRulesFromPayload(devrites))
	}
	if err := r.recheckPath(merge.TargetRel); err != nil {
		return err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode merged Claude settings: %w", err)
	}
	data = append(data, '\n')
	if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", merge.TargetRel, err)
	}
	return r.installClaudeSettingsMarker(ownsDefaultMode, managedRulesFromPayload(devrites))
}

func (r *runner) seedClaudeSettings() error {
	return r.mergeClaudeSettings(hostpack.ClaudeSettingsMerge)
}

func stripHooksPath(path string) error {
	current, err := readJSON(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load hooks config: %w", err)
	}
	next := stripDevritesHooks(current)
	if len(next) == 0 {
		return os.Remove(path)
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode hooks config: %w", err)
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func (r *runner) stripClaudeSettings(path string, preserveEmpty bool) error {
	current, err := readJSON(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load Claude settings: %w", err)
	}
	next := stripDevritesSettings(current, r.claudeDefaultModeOwned(), r.managedClaudePermissionRules())
	if len(next) == 0 && !preserveEmpty {
		return os.Remove(path)
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Claude settings: %w", err)
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func stripMarkerPath(path, begin, end string) error {
	// #nosec G304 -- managed file path from the install manifest
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	next := stripMarkerBlock(data, begin, end)
	if strings.TrimSpace(string(next)) == "" {
		return os.Remove(path)
	}
	return fsutil.WriteFileAtomic(path, next, 0o644)
}

func stripMarkerBlock(data []byte, begin, end string) []byte {
	var out strings.Builder
	inBlock := false
	for _, line := range strings.SplitAfter(string(data), "\n") {
		trim := strings.TrimSuffix(line, "\n")
		trim = strings.TrimSuffix(trim, "\r")
		switch trim {
		case begin:
			inBlock = true
			continue
		case end:
			inBlock = false
			continue
		}
		if !inBlock {
			out.WriteString(line)
		}
	}
	return []byte(out.String())
}

func hasTopLevelTOMLKey(data []byte, key string) bool {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			return false
		}
		name, _, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(name) == key {
			return true
		}
	}
	return false
}

func readJSON(path string) (map[string]any, error) {
	// #nosec G304 -- engine-written workspace or metadata JSON
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return decodeJSON(data)
}

func readJSONFS(src fs.FS, path string) (map[string]any, error) {
	data, err := fs.ReadFile(src, path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return decodeJSON(data)
}

func decodeJSON(data []byte) (map[string]any, error) {
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	return out, nil
}

func isDevritesHookCommand(v any) bool {
	hook, ok := v.(map[string]any)
	if !ok {
		return false
	}
	command, _ := hook["command"].(string)
	return strings.Contains(command, "devrites-engine hook ") ||
		strings.Contains(command, ".claude/hooks/devrites-") ||
		strings.Contains(command, ".codex/hooks/devrites-") ||
		(strings.Contains(command, "printf ") && strings.Contains(command, "DevRites:"))
}

func isDevritesHooksComment(comment string) bool {
	return comment == "DevRites hooks" ||
		strings.HasPrefix(comment, "DevRites hooks: every event invokes the global `devrites-engine` engine binary") ||
		strings.HasPrefix(comment, "DevRites hooks: auto-approve the read-only orientation/gate scripts") ||
		strings.HasPrefix(comment, "DevRites hooks — every event invokes the global `devrites-engine` engine binary") ||
		strings.HasPrefix(comment, "DevRites hooks — auto-approve the read-only orientation/gate scripts") ||
		strings.HasPrefix(comment, "DevRites hooks for Codex. Project hooks load only after")
}

func isDevritesSettingsComment(comment string) bool {
	return isDevritesHooksComment(comment) ||
		strings.HasPrefix(comment, "DevRites keeps the root orchestration context source-read-only.")
}

func stripDevritesHooks(config map[string]any) map[string]any {
	next := map[string]any{}
	for k, v := range config {
		if k == "$comment" {
			if s, ok := v.(string); ok && isDevritesHooksComment(s) {
				continue
			}
		}
		if k == "statusLine" && isDevritesHookCommand(v) {
			continue
		}
		next[k] = v
	}
	rawHooksValue, exists := next["hooks"]
	if !exists {
		return next
	}
	rawHooks, ok := rawHooksValue.(map[string]any)
	if !ok {
		return next
	}
	hooks := map[string]any{}
	for event, entries := range rawHooks {
		arr, ok := entries.([]any)
		if !ok {
			hooks[event] = entries
			continue
		}
		var kept []any
		for _, entry := range arr {
			entryMap, ok := entry.(map[string]any)
			if !ok {
				kept = append(kept, entry)
				continue
			}
			commands, ok := entryMap["hooks"].([]any)
			if !ok {
				kept = append(kept, entry)
				continue
			}
			var keptCommands []any
			for _, command := range commands {
				if !isDevritesHookCommand(command) {
					keptCommands = append(keptCommands, command)
				}
			}
			if len(keptCommands) > 0 {
				nextEntry := make(map[string]any, len(entryMap))
				for k, v := range entryMap {
					nextEntry[k] = v
				}
				nextEntry["hooks"] = keptCommands
				kept = append(kept, nextEntry)
			}
		}
		if len(kept) > 0 {
			hooks[event] = kept
		}
	}
	if len(hooks) == 0 {
		delete(next, "hooks")
	} else {
		next["hooks"] = hooks
	}
	return next
}

func stripDevritesSettings(config map[string]any, removeOwnedDefaultMode bool, managed map[string]bool) map[string]any {
	next := stripDevritesHooks(config)
	if comment, ok := next["$comment"].(string); ok && isDevritesSettingsComment(comment) {
		delete(next, "$comment")
	}

	rawPermissions, exists := next["permissions"]
	if !exists {
		return next
	}
	permissions, ok := rawPermissions.(map[string]any)
	if !ok {
		return next
	}
	clean := make(map[string]any, len(permissions))
	for key, value := range permissions {
		switch key {
		case "allow":
			rules, ok := value.([]any)
			if !ok {
				clean[key] = value
				continue
			}
			kept := make([]any, 0, len(rules))
			for _, rule := range rules {
				if !isDevritesPermissionRule(rule, managed) {
					kept = append(kept, rule)
				}
			}
			if len(kept) > 0 {
				clean[key] = kept
			}
		case "defaultMode":
			if removeOwnedDefaultMode && value == "plan" {
				continue
			}
			clean[key] = value
		default:
			clean[key] = value
		}
	}
	if len(clean) == 0 {
		delete(next, "permissions")
	} else {
		next["permissions"] = clean
	}
	return next
}

func mergeClaudeSettingsConfig(current, desired map[string]any, defaultModeOwned bool, managed map[string]bool) (map[string]any, bool, error) {
	desiredPermissions, ok := desired["permissions"].(map[string]any)
	if !ok || desiredPermissions["defaultMode"] != "plan" {
		return nil, false, fmt.Errorf("DevRites Claude settings payload must declare permissions.defaultMode=plan")
	}
	desiredAllow, ok := desiredPermissions["allow"].([]any)
	if !ok {
		return nil, false, fmt.Errorf("DevRites Claude settings payload permissions.allow must be an array")
	}

	next := stripDevritesSettings(current, false, managed)
	permissions := map[string]any{}
	if raw, exists := next["permissions"]; exists {
		var ok bool
		permissions, ok = raw.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("existing Claude permissions must be a JSON object")
		}
	}

	switch mode, exists := permissions["defaultMode"]; {
	case !exists:
		defaultModeOwned = true
	case mode == "plan":
	case mode != "plan":
		return nil, false, fmt.Errorf("existing Claude permissions.defaultMode is %q; DevRites requires plan mode for the read-only root orchestrator", mode)
	}
	permissions["defaultMode"] = "plan"
	var allow []any
	if existingAllow, exists := permissions["allow"]; exists {
		allow, ok = existingAllow.([]any)
		if !ok {
			return nil, false, fmt.Errorf("existing Claude permissions.allow must be an array")
		}
	}
	seen := map[string]bool{}
	for _, rule := range allow {
		if value, ok := rule.(string); ok {
			seen[value] = true
		}
	}
	for _, rule := range desiredAllow {
		value, ok := rule.(string)
		if !ok {
			return nil, false, fmt.Errorf("DevRites Claude permission rules must be strings")
		}
		if !seen[value] {
			allow = append(allow, value)
			seen[value] = true
		}
	}
	permissions["allow"] = allow
	next["permissions"] = permissions
	if _, exists := next["$comment"]; !exists {
		if comment, ok := desired["$comment"].(string); ok {
			next["$comment"] = comment
		}
	}
	return next, defaultModeOwned, nil
}

// isDevritesPermissionRule reports whether a permissions.allow rule is
// DevRites-managed: every devrites-engine command rule, plus an exact match
// against a rule the shipped payload owns (MCP nav tools, read-only git,
// .devrites-scoped writes). User-owned lookalikes (e.g. a user's own
// "Bash(git checkout *)") never match the managed set and survive uninstall.
func isDevritesPermissionRule(value any, managed map[string]bool) bool {
	rule, ok := value.(string)
	if !ok {
		return false
	}
	rule = strings.TrimSpace(rule)
	return strings.HasPrefix(rule, "Bash(devrites-engine ") || managed[rule]
}

package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/hostpack"
)

func (r *runner) installClaudeSettingsMarker(state claudeSettingsMerge) error {
	var b strings.Builder
	b.WriteString(hostpack.ClaudeSettingsMerge.MarkerText + "\n" + claudeRulesAddedLine)
	if state.created {
		b.WriteString("\n" + claudeSettingsCreatedLine)
	}
	rules := make([]string, 0, len(state.added))
	for rule := range state.added {
		rules = append(rules, rule)
	}
	sort.Strings(rules)
	for _, rule := range rules {
		b.WriteString("\nrule:" + rule)
	}
	return r.installMarker(hostpack.ClaudeSettingsMerge.MarkerRel, b.String())
}

const (
	// claudeRulesAddedLine marks a settings marker whose rule: lines list only
	// the rules DevRites added; older markers listed every payload rule.
	claudeRulesAddedLine = "rules=added"
	// claudeSettingsCreatedLine records that DevRites created settings.json,
	// so uninstall removes it once nothing else is left.
	claudeSettingsCreatedLine = "settings=created"
)

func (r *runner) claudeSettingsMarkerLines() []string {
	data, err := os.ReadFile(filepath.Join(r.target, filepath.FromSlash(hostpack.ClaudeSettingsMerge.MarkerRel)))
	if err != nil {
		return nil
	}
	return strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
}

// claudeDefaultModeOwned reports a marker from an install that set
// permissions.defaultMode=plan itself; DevRites no longer sets a mode, so
// update and uninstall remove that plan mode.
func (r *runner) claudeDefaultModeOwned() bool {
	return slices.Contains(r.claudeSettingsMarkerLines(), "default-mode=added")
}

// claudeSettingsCreated reports whether DevRites created settings.json. A
// legacy marker (no rules=added line) predates the settings=created record,
// so its file is treated as created: uninstall still removes it only when
// stripping leaves nothing.
func (r *runner) claudeSettingsCreated() bool {
	lines := r.claudeSettingsMarkerLines()
	return slices.Contains(lines, claudeSettingsCreatedLine) || (lines != nil && !slices.Contains(lines, claudeRulesAddedLine))
}

// managedClaudePermissionRules returns the exact non-engine allow rules
// DevRites manages in .claude/settings.json: the rules the merge marker
// records as DevRites-added (none before the first install). A legacy marker
// that predates added-only recording is unioned with the current payload's non-engine rules, as
// before. Engine command rules are matched by prefix and never need listing.
func (r *runner) managedClaudePermissionRules() map[string]bool {
	managed := map[string]bool{}
	lines := r.claudeSettingsMarkerLines()
	for _, line := range lines {
		if rule, ok := strings.CutPrefix(line, "rule:"); ok {
			managed[strings.TrimSpace(rule)] = true
		}
	}
	if lines == nil || slices.Contains(lines, claudeRulesAddedLine) {
		return managed
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
	if !r.opts.DryRun {
		if err := r.recheckPath(merge.TargetRel); err != nil {
			return err
		}
	}
	next, current, err := r.markerMergeNext(merge)
	if err != nil {
		return err
	}
	if r.opts.DryRun {
		verb := "create DevRites block"
		if current != nil {
			verb = "append DevRites block"
			if bytes.Contains(current, []byte(merge.Begin)) {
				verb = "refresh DevRites block"
			}
		}
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s (%s)\n", merge.TargetRel, verb)
	} else {
		dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
		if err := r.writeConfined(dest, next, 0o644); err != nil {
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

// markerMergeNext returns the merged target bytes and the current bytes
// (nil when the target is absent) without writing anything.
func (r *runner) markerMergeNext(merge hostpack.MarkerMerge) ([]byte, []byte, error) {
	block, err := fs.ReadFile(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return nil, nil, fmt.Errorf("read payload %s: %w", merge.PayloadRel, err)
	}
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	// #nosec G304 -- dest joins the operator target with a fixed manifest-relative record
	current, readErr := os.ReadFile(dest)
	if errors.Is(readErr, fs.ErrNotExist) {
		return block, nil, nil
	}
	if readErr != nil {
		return nil, nil, fmt.Errorf("cannot read %s: %w", merge.TargetRel, readErr)
	}
	if err := hostpack.ValidateMarkerBlocks(current, r.sharedMarkerMerges(merge)...); err != nil {
		return nil, nil, fmt.Errorf("%w in %s; fix it and rerun", err, merge.TargetRel)
	}
	next, err := hostpack.MergeMarkerBlock(current, block, merge.Begin, merge.End)
	if err != nil {
		return nil, nil, fmt.Errorf("%w in %s; fix it and rerun", err, merge.TargetRel)
	}
	return next, current, nil
}

func (r *runner) sharedMarkerMerges(merge hostpack.MarkerMerge) []hostpack.MarkerMerge {
	merges := []hostpack.MarkerMerge{merge}
	for _, other := range []struct {
		enabled bool
		merge   hostpack.MarkerMerge
	}{
		{r.opts.WithCodex, hostpack.CodexAgentsMerge},
		{r.opts.WithPi, hostpack.PiAgentsMerge},
		{r.opts.WithDevin, hostpack.DevinAgentsMerge},
	} {
		if other.enabled && other.merge.TargetRel == merge.TargetRel && other.merge.Begin != merge.Begin {
			merges = append(merges, other.merge)
		}
	}
	return merges
}

func (r *runner) mergeCodexConfig() error {
	merge := hostpack.CodexConfigMerge
	if r.opts.DryRun {
		if _, err := r.codexConfigNext(); err != nil {
			return err
		}
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s (insert DevRites permission block)\n", merge.TargetRel)
		return r.installMarker(merge.MarkerRel, merge.MarkerText)
	}
	if err := r.recheckPath(merge.TargetRel); err != nil {
		return err
	}
	next, err := r.codexConfigNext()
	if err != nil {
		return err
	}
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	if err := r.writeConfined(dest, next, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", merge.TargetRel, err)
	}
	return r.installMarker(merge.MarkerRel, merge.MarkerText)
}

// codexConfigNext returns the merged .codex/config.toml without writing it.
// The DevRites block ends inside its [permissions.*] table, so it goes after
// the user's top-level keys and before their first table header; anywhere
// earlier would re-scope those keys into the DevRites table.
func (r *runner) codexConfigNext() ([]byte, error) {
	merge := hostpack.CodexConfigMerge
	block, err := fs.ReadFile(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return nil, fmt.Errorf("read payload %s: %w", merge.PayloadRel, err)
	}
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	var current []byte
	// #nosec G304 -- dest joins the operator target with a fixed manifest-relative record
	if data, readErr := os.ReadFile(dest); readErr == nil {
		// Drop a UTF-8 BOM: the block may go at offset 0, and a BOM after
		// it is invalid TOML.
		data = bytes.TrimPrefix(data, utf8BOM)
		if current, err = hostpack.StripCodexConfigBlocks(data); err != nil {
			return nil, fmt.Errorf("%w in %s; fix it and rerun", err, merge.TargetRel)
		}
		if hasTopLevelTOMLKey(current, "default_permissions") {
			return nil, fmt.Errorf("%s already sets top-level default_permissions; remove that project override before installing the DevRites read-only-root profile", merge.TargetRel)
		}
	} else if !errors.Is(readErr, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot read %s: %w", merge.TargetRel, readErr)
	}
	if len(bytes.TrimSpace(current)) == 0 {
		current = nil
	}
	split := firstTOMLTableHeader(current)
	var next []byte
	if split > 0 {
		next = append(next, current[:split]...)
		if next[len(next)-1] != '\n' {
			next = append(next, '\n')
		}
		// Blank separator; StripMarkerBlock removes it with the block.
		next = append(next, '\n')
	}
	next = append(next, block...)
	if len(next) == 0 || next[len(next)-1] != '\n' {
		next = append(next, '\n')
	}
	return append(next, current[split:]...), nil
}

var utf8BOM = []byte("\xef\xbb\xbf")

// firstTOMLTableHeader returns the byte offset of the first [table] or
// [[array]] header line, or len(data) when there is none.
func firstTOMLTableHeader(data []byte) int {
	split := len(data)
	scanTOMLTopLevel(data, func(offset int, line string) bool {
		if tomlTableHeader.MatchString(line) {
			split = offset
			return false
		}
		return true
	})
	return split
}

// scanTOMLTopLevel calls visit with the byte offset and trimmed text of each
// line that starts outside any multi-line string or array/inline-table value
// (a leading UTF-8 BOM is ignored), until visit returns false. Only such lines
// can hold a table header or a key.
func scanTOMLTopLevel(data []byte, visit func(offset int, line string) bool) {
	depth, inString, offset := 0, "", 0
	for i, raw := range strings.SplitAfter(string(data), "\n") {
		line := raw
		if i == 0 {
			line = strings.TrimPrefix(line, string(utf8BOM))
		}
		lineOffset := offset
		offset += len(raw)
		if depth == 0 && inString == "" {
			trimmed := strings.TrimSpace(line)
			if !visit(lineOffset, trimmed) {
				return
			}
			if tomlTableHeader.MatchString(trimmed) {
				continue
			}
		}
		depth, inString = scanTOMLLine(line, depth, inString)
	}
}

// scanTOMLLine carries the open [ / { depth and any open multi-line string
// delimiter (""" or ”') across one line, skipping strings and comments.
func scanTOMLLine(line string, depth int, inString string) (int, string) {
	for i := 0; i < len(line); i++ {
		if inString != "" {
			if inString == `"""` && line[i] == '\\' {
				i++
			} else if strings.HasPrefix(line[i:], inString) {
				i += 2
				// Up to two extra quotes still belong to the content.
				for i+1 < len(line) && line[i+1] == inString[0] {
					i++
				}
				inString = ""
			}
			continue
		}
		switch c := line[i]; c {
		case '#':
			return depth, ""
		case '[', '{':
			depth++
		case ']', '}':
			if depth > 0 {
				depth--
			}
		case '"', '\'':
			if delim := strings.Repeat(string(c), 3); strings.HasPrefix(line[i:], delim) {
				inString = delim
				i += 2
				continue
			}
			for i++; i < len(line) && line[i] != c; i++ {
				if c == '"' && line[i] == '\\' {
					i++
				}
			}
		}
	}
	return depth, inString
}

// tomlTableHeader matches a trimmed [table] or [[array]] header line. Quoted
// key parts may hold any character; a bare part stops at brackets, quotes,
// '#', '=' and ',', so array and inline-table value lines never match.
var tomlTableHeader = regexp.MustCompile(`^\[\[?(?:[^\[\]"'#=,]|"(?:[^"\\]|\\.)*"|'[^']*')+\]\]?\s*(#.*)?$`)

func (r *runner) mergeClaudeSettings(merge hostpack.JSONMerge) error {
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	next, state, err := r.claudeSettingsNext()
	if err != nil {
		return err
	}
	if permissions, _ := next["permissions"].(map[string]any); permissions["defaultMode"] == "plan" {
		fmt.Fprintf(r.opts.Stderr, "warning: %s: permissions.defaultMode is plan, which blocks the root's .devrites writes every rite needs (ADR-0035); remove it to run the lifecycle\n", merge.TargetRel)
	}
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [merge] %s\n", merge.DryRunText)
		return r.installClaudeSettingsMarker(state)
	}
	if err := r.recheckPath(merge.TargetRel); err != nil {
		return err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode merged Claude settings: %w", err)
	}
	data = append(data, '\n')
	if err := r.writeConfined(dest, data, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", merge.TargetRel, err)
	}
	return r.installClaudeSettingsMarker(state)
}

// claudeSettingsMerge is what the settings marker records about a merge.
type claudeSettingsMerge struct {
	created bool
	added   map[string]bool
}

// claudeSettingsNext returns the merged .claude/settings.json without
// writing it.
func (r *runner) claudeSettingsNext() (map[string]any, claudeSettingsMerge, error) {
	merge := hostpack.ClaudeSettingsMerge
	dest := filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))
	devrites, err := readJSONFS(r.payloadFS, merge.PayloadRel)
	if err != nil {
		return nil, claudeSettingsMerge{}, fmt.Errorf("load Claude settings payload: %w", err)
	}
	current := map[string]any{}
	state := claudeSettingsMerge{created: r.claudeSettingsCreated()}
	if data, readErr := readJSON(dest); readErr == nil {
		current = data
	} else if errors.Is(readErr, fs.ErrNotExist) {
		state.created = true
	} else {
		return nil, claudeSettingsMerge{}, fmt.Errorf("load existing Claude settings: %w", readErr)
	}
	next, added, err := mergeClaudeSettingsConfig(current, devrites, r.claudeDefaultModeOwned(), r.managedClaudePermissionRules())
	if err != nil {
		return nil, claudeSettingsMerge{}, err
	}
	state.added = added
	return next, state, nil
}

func (r *runner) seedClaudeSettings() error {
	return r.mergeClaudeSettings(hostpack.ClaudeSettingsMerge)
}

func (r *runner) stripHooksConfined(path string) error {
	return stripHooksWith(path, func(data []byte) error {
		return r.writeConfined(path, data, 0o644)
	}, func() error { return r.removeConfined(path) })
}

func stripHooksWith(path string, write func([]byte) error, remove func() error) error {
	current, err := readJSON(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load hooks config: %w", err)
	}
	next := stripDevritesHooks(current)
	if len(next) == 0 {
		return remove()
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode hooks config: %w", err)
	}
	data = append(data, '\n')
	return write(data)
}

func (r *runner) stripClaudeSettings(path string) error {
	current, err := readJSON(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("load Claude settings: %w", err)
	}
	next := stripDevritesSettings(current, r.claudeDefaultModeOwned(), r.managedClaudePermissionRules())
	if len(next) == 0 && r.claudeSettingsCreated() {
		return r.removeConfined(path)
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Claude settings: %w", err)
	}
	data = append(data, '\n')
	return r.writeConfined(path, data, 0o644)
}

func (r *runner) stripMarkerConfined(path, begin, end string) error {
	return stripMarkerWith(path, begin, end, func(data []byte) error {
		return r.writeConfined(path, data, 0o644)
	}, func() error { return r.removeConfined(path) })
}

func stripMarkerWith(path, begin, end string, write func([]byte) error, remove func() error) error {
	// #nosec G304 -- managed file path from the install manifest
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	next, err := hostpack.StripMarkerBlock(data, begin, end)
	if err != nil {
		return fmt.Errorf("%w in %s; fix it and rerun", err, path)
	}
	if strings.TrimSpace(string(next)) == "" {
		return remove()
	}
	return write(next)
}

func hasTopLevelTOMLKey(data []byte, key string) bool {
	found := false
	scanTOMLTopLevel(data, func(_ int, line string) bool {
		if tomlTableHeader.MatchString(line) {
			return false
		}
		name, _, ok := strings.Cut(line, "=")
		name = strings.TrimSpace(name)
		if len(name) >= 2 && (name[0] == '"' || name[0] == '\'') && name[len(name)-1] == name[0] {
			name = name[1 : len(name)-1]
		}
		found = ok && name == key
		return !found
	})
	return found
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
		strings.HasPrefix(comment, "DevRites keeps the root orchestration context source-read-only")
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

// mergeClaudeSettingsConfig also returns the non-engine payload rules the
// merge added, i.e. those the user did not already grant themselves. It never
// sets permissions.defaultMode; removeOwnedPlan drops a plan mode an older
// DevRites install wrote.
func mergeClaudeSettingsConfig(current, desired map[string]any, removeOwnedPlan bool, managed map[string]bool) (map[string]any, map[string]bool, error) {
	desiredPermissions, _ := desired["permissions"].(map[string]any)
	desiredAllow, ok := desiredPermissions["allow"].([]any)
	if !ok {
		return nil, nil, fmt.Errorf("DevRites Claude settings payload permissions.allow must be an array")
	}

	next := stripDevritesSettings(current, removeOwnedPlan, managed)
	permissions := map[string]any{}
	if raw, exists := next["permissions"]; exists {
		var ok bool
		permissions, ok = raw.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("existing Claude permissions must be a JSON object")
		}
	}

	var allow []any
	if existingAllow, exists := permissions["allow"]; exists {
		allow, ok = existingAllow.([]any)
		if !ok {
			return nil, nil, fmt.Errorf("existing Claude permissions.allow must be an array")
		}
	}
	added := map[string]bool{}
	seen := map[string]bool{}
	for _, rule := range allow {
		if value, ok := rule.(string); ok {
			seen[value] = true
		}
	}
	for _, rule := range desiredAllow {
		value, ok := rule.(string)
		if !ok {
			return nil, nil, fmt.Errorf("DevRites Claude permission rules must be strings")
		}
		if !seen[value] {
			allow = append(allow, value)
			seen[value] = true
			if !strings.HasPrefix(value, "Bash(devrites-engine ") {
				added[value] = true
			}
		}
	}
	permissions["allow"] = allow
	next["permissions"] = permissions
	if _, exists := next["$comment"]; !exists {
		if comment, ok := desired["$comment"].(string); ok {
			next["$comment"] = comment
		}
	}
	return next, added, nil
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

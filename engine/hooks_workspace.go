package main

// Workspace hooks read a feature's runtime state (state.md fields, questions.md,
// .red) as sidecar files in the feature directory. work/ is canonical; features/
// remains readable as a compatibility alias. Every hook is read-only and
// fail-open, staying silent + exit 0 outside a DevRites workspace.

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/orient"
	"github.com/devrites/devrites/internal/state"
)

// State-field patterns. Each hook reads a field the way the scripts do:
// `grep -iE <find>` (case-INSENSITIVE) selects the first matching line, then
// `sed -E s/<strip>//` (case-SENSITIVE, as sed is) removes the key prefix — a
// no-op when the case differs, so the mismatched-case behavior is preserved.
var (
	findNextStep  = regexp.MustCompile(`(?i)^[-[:space:]]*Next step:`)
	stripNextStep = regexp.MustCompile(`^[-[:space:]]*Next step:[[:space:]]*`)
	findStatus    = regexp.MustCompile(`(?i)^[-[:space:]]*Status:`)
	stripStatus   = regexp.MustCompile(`^[-[:space:]]*Status:[[:space:]]*`)
	findPhase     = regexp.MustCompile(`(?i)^[-[:space:]]*Phase:`)
	stripPhase    = regexp.MustCompile(`^[-[:space:]]*Phase:[[:space:]]*`)
	findAFKRem    = regexp.MustCompile(`(?i)AFK slices remaining:`)
	stripAFKRem   = regexp.MustCompile(`.*remaining:[[:space:]]*`)
	openStatusRe  = regexp.MustCompile(`(?i)^[[:space:]]*status:[[:space:]]*open`)
)

// resolveWorkspaceDir resolves the active feature's directory. New writes default
// to work/; existing features/ workspaces remain readable.
func resolveWorkspaceDir(root, slug string) string {
	work := filepath.Join(root, "work", slug)
	if wsIsDir(work) {
		return work
	}
	features := filepath.Join(root, "features", slug)
	if wsIsDir(features) {
		return features
	}
	return work
}

// resolveWorkspace resolves the active feature's directory.
// ok is false — a silent no-op — when there is no .devrites root or no active
// feature, keeping the hooks fail-open outside a DevRites workspace.
func resolveWorkspace() (root, slug, dir string, ok bool) {
	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return "", "", "", false
	}
	slug, err = orient.ActiveSlug(root)
	if err != nil || slug == "" {
		return "", "", "", false
	}
	return root, slug, resolveWorkspaceDir(root, slug), true
}

// hookCursor re-injects the active feature's cursor each turn (UserPromptSubmit),
// so the load-bearing next-action sits at the high-attention end of context. Ported
// from devrites-cursor.sh onto the new schema: plain-text output, silent + exit 0
// on a non-DevRites project.
func hookCursor(stdin io.Reader, stdout, stderr io.Writer) int {
	_, _ = io.Copy(io.Discard, stdin)
	root, slug, dir, ok := resolveWorkspace()
	if !ok || !wsIsDir(dir) {
		return exitOK
	}
	stateLines := wsReadLines(filepath.Join(dir, "state.md"))

	next := wsField(stateLines, findNextStep, stripNextStep)
	status := wsField(stateLines, findStatus, stripStatus)
	gates := wsGateCount(filepath.Join(dir, "questions.md"))
	afk := ""
	if wsIsFile(filepath.Join(root, "AFK")) {
		afk = wsField(stateLines, findAFKRem, stripAFKRem)
	}

	fmt.Fprintf(stdout, "DevRites cursor — active feature: %s\n", slug)
	if status != "" {
		fmt.Fprintf(stdout, "  status: %s\n", status)
	}
	if next != "" {
		fmt.Fprintf(stdout, "  next: %s\n", next)
	}
	fmt.Fprintf(stdout, "  open questions: %s\n", wsOrZero(gates))
	if afk != "" {
		fmt.Fprintf(stdout, "  AFK slices remaining: %s\n", afk)
	}
	if wsIsFile(filepath.Join(dir, ".red")) {
		fmt.Fprintln(stdout, "  ⚠ tests/build RED — resolve before stopping")
	}
	return exitOK
}

// hookStatusline is the one-line workspace HUD (settings.json statusLine). Ported
// from devrites-statusline.sh onto the new schema: reads the session JSON on stdin
// (ignored), silent + exit 0 when there is no active workspace.
func hookStatusline(stdin io.Reader, stdout, stderr io.Writer) int {
	_, _ = io.Copy(io.Discard, stdin)
	root, slug, dir, ok := resolveWorkspace()
	if !ok {
		return exitOK
	}
	phase := wsField(wsReadLines(filepath.Join(dir, "state.md")), findPhase, stripPhase)
	if phase == "" {
		phase = "?"
	}
	gates := wsOrZero(wsGateCount(filepath.Join(dir, "questions.md")))
	mode := "HITL"
	if wsIsFile(filepath.Join(root, "AFK")) {
		mode = "AFK"
	}
	red := ""
	if wsIsFile(filepath.Join(dir, ".red")) {
		red = " · RED"
	}
	fmt.Fprintf(stdout, "DevRites: %s · %s · gates:%s · %s%s\n", slug, phase, gates, mode, red)
	return exitOK
}

// redwatch detection patterns, ported from devrites-redwatch.sh (grep -qiE, so
// case-insensitive; the FAIL/PASS scans are line-oriented, hence (?m)).
// pack-scan-ignore: these are the hook's own red/green heuristics, not a payload.
var (
	redTestCmdRe = regexp.MustCompile(`(?i)(npm|pnpm|yarn|bun)([[:space:]]+run)?[[:space:]]+(test|build|lint|typecheck|check)|jest|vitest|pytest|go[[:space:]]+test|cargo[[:space:]]+(test|build|clippy)|\bmvn\b|gradle|eslint|ruff|mypy|\btsc\b|\bmake[[:space:]]+(test|build|check)`)
	redFailRe    = regexp.MustCompile(`(?im)(^|[^A-Za-z])FAIL([^A-Za-z]|$)|[1-9][0-9]*[[:space:]]+(failed|failing|failures?|errors?)|not ok|panic:|error TS[0-9]|Traceback|AssertionError|✗|✘|BUILD FAILED|exit code [1-9]`)
	redPassRe    = regexp.MustCompile(`(?im)PASS(ED)?|0[[:space:]]+(failed|failing|errors?)|all tests passed|BUILD SUCC(ESS|EEDED)|succeeded|no (errors|problems|issues)|(^|[^a-z])ok([^a-z]|$)|✓`)
)

const redReasonFmt = "DevRites: tests/build are RED (%s). Fix to green or record the failure + next step in state.md before stopping — the Stop gate blocks an end-of-turn while .red is set."

// hookRedwatch is the fail-on-red sentinel (PostToolUse Bash). After a test / build
// / lint command it sets or clears <featureDir>/.red so the Stop gate can refuse to
// end a turn while the suite is red. Ported from devrites-redwatch.sh onto the new
// schema; heuristic on the command + its output, fail-open and silent on any doubt.
func hookRedwatch(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	if !bytes.Contains(data, []byte(`"tool_name"`)) {
		return exitOK
	}
	_, _, dir, ok := resolveWorkspace()
	if !ok || !wsIsDir(dir) {
		return exitOK
	}
	in := h.ParseGuardInput(bytes.NewReader(data))
	if in.ToolName != "Bash" {
		return exitOK
	}
	// Only react to test/build/lint/typecheck commands.
	if !redTestCmdRe.MatchString(in.Command) {
		return exitOK
	}

	red := filepath.Join(dir, ".red")
	combo := in.Command + "\n" + in.ToolResponse
	if redFailRe.MatchString(combo) {
		_ = os.WriteFile(red, []byte(in.Command+"\n"), 0o644)
		out, err := h.PostToolContext(fmt.Sprintf(redReasonFmt, redwatchSafe(in.Command)))
		if err != nil {
			debugf(stderr, "redwatch: %v", err)
			return exitOK
		}
		fmt.Fprintln(stdout, out)
		return exitOK
	}
	if redPassRe.MatchString(combo) {
		_ = os.Remove(red)
	}
	return exitOK
}

// redwatchSafe truncates a command to 80 bytes and drops " and \, matching
// redwatch.sh's `head -c 80 | tr -d '"\\'`.
func redwatchSafe(cmd string) string {
	if len(cmd) > 80 {
		cmd = cmd[:80]
	}
	return strings.NewReplacer(`"`, "", `\`, "").Replace(cmd)
}

// patchPathRe captures each file path an apply_patch command touches, matching the
// node regex the shell guards use.
var patchPathRe = regexp.MustCompile(`(?m)^\*\*\* (?:(?:Add|Update|Delete) File|Move to): (.+)$`)

// isEditTool reports whether a tool can write source — the Edit family plus
// Codex's apply_patch.
func isEditTool(tool string) bool {
	switch tool {
	case "Edit", "Write", "MultiEdit", "NotebookEdit", "apply_patch":
		return true
	}
	return false
}

// patchPaths returns the file paths an apply_patch command touches.
func patchPaths(command string) []string {
	var paths []string
	for _, m := range patchPathRe.FindAllStringSubmatch(command, -1) {
		paths = append(paths, m[1])
	}
	return paths
}

// underDevrites reports whether an absolute path is inside the .devrites root
// (the orchestrator's own bookkeeping, always allowed to be edited).
func underDevrites(abs, root string) bool {
	return strings.HasPrefix(abs, root+string(filepath.Separator))
}

const a1DenyReason = "DevRites A1: the orchestrator must not edit source mid-build — the slice-wright is the only writer. Re-dispatch the wright (continue it once) or stop + escalate; do not patch the code yourself. (devrites-a1-guard)"

// hookA1Guard keeps the /rite-build orchestrator from editing source while a slice
// is mid-build — the slice-wright (a subagent) is the only sanctioned writer of
// code. Ported from devrites-a1-guard.sh onto the new schema: FAIL-OPEN, OBSERVE by
// default (logs to .a1-guard.log), blocking only when the build window is open, the
// edit is source from the MAIN thread, and enforce mode is on.
func hookA1Guard(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	if !bytes.Contains(data, []byte(`"tool_name"`)) {
		return exitOK
	}
	root, _, dir, ok := resolveWorkspace()
	if !ok {
		return exitOK
	}
	// Only armed between a reconcile snapshot and its check; the inline fallback
	// means the orchestrator is the legitimate writer.
	if !wsIsFile(filepath.Join(dir, ".reconcile-base")) {
		return exitOK
	}
	if wsIsFile(filepath.Join(dir, ".reconcile-inline")) {
		return exitOK
	}

	in := h.ParseGuardInput(bytes.NewReader(data))
	if !isEditTool(in.ToolName) {
		return exitOK
	}
	// A subagent caller (the wright) is allowed. Claude sends agent_id; Codex agent_type.
	if in.AgentID != "" || in.AgentType != "" {
		return exitOK
	}

	projectDir := filepath.Dir(root)
	target := ""
	if in.ToolName == "apply_patch" {
		sourcePatch := false
		for _, p := range patchPaths(in.Command) {
			abs := p
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(projectDir, abs)
			}
			if !underDevrites(abs, root) {
				sourcePatch = true
				break
			}
		}
		if !sourcePatch {
			return exitOK
		}
		target = "apply_patch"
	} else {
		if in.FilePath == "" {
			return exitOK
		}
		abs := in.FilePath
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(projectDir, abs)
		}
		if underDevrites(abs, root) {
			return exitOK // orchestrator editing its own .devrites bookkeeping
		}
		target = abs
	}

	mode := os.Getenv("DEVRITES_A1_HOOK")
	if wsIsFile(filepath.Join(dir, ".a1-enforce")) {
		mode = "enforce"
	}
	// The sentinel file forces enforce; hookEnforce also honors the strict profile.
	if mode == "enforce" || hookEnforce("DEVRITES_A1_HOOK") {
		out, err := h.PreToolDeny(a1DenyReason)
		if err != nil {
			debugf(stderr, "a1-guard: %v", err)
			return exitOK
		}
		fmt.Fprintln(stdout, out)
		return exitOK
	}
	_ = state.AppendLog(filepath.Join(dir, ".a1-guard.log"), "WOULD-BLOCK\t"+in.ToolName+"\t"+target)
	return exitOK
}

const wrightDenyReason = "DevRites scope: this path is not in touched-files.md. Build only the slice contract; if the slice genuinely needs this file, return an Escalation so the orchestrator routes it through the Spec Drift Guard — do not widen scope yourself. (devrites-wright-scope)"

// hookWrightScope fences the slice-wright to its slice: it denies an Edit/Write to a
// path not listed in the feature's touched-files.md. Ported from
// devrites-wright-scope.sh onto the new schema: FAIL-OPEN, OBSERVE by default (logs
// to .wright-scope.log), enforced with DEVRITES_WRIGHT_SCOPE=enforce.
func hookWrightScope(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	if !bytes.Contains(data, []byte(`"tool_name"`)) {
		return exitOK
	}
	root, _, dir, ok := resolveWorkspace()
	if !ok {
		return exitOK
	}
	tfData, err := os.ReadFile(filepath.Join(dir, "touched-files.md"))
	if err != nil {
		return exitOK // no touched-files.md → not fencing
	}
	touched := string(tfData)

	in := h.ParseGuardInput(bytes.NewReader(data))
	if os.Getenv("DEVRITES_WRIGHT_AGENT_REQUIRED") == "1" && in.AgentType != "devrites-slice-wright" {
		return exitOK
	}
	if !isEditTool(in.ToolName) {
		return exitOK
	}

	projectDir := filepath.Dir(root)
	var bad []string
	if in.ToolName == "apply_patch" {
		for _, p := range patchPaths(in.Command) {
			if strings.HasPrefix(p, ".devrites/") || strings.Contains(p, "/.devrites/") {
				continue
			}
			rel := strings.TrimPrefix(p, projectDir+string(filepath.Separator))
			if !strings.Contains(touched, rel) && !strings.Contains(touched, p) {
				bad = append(bad, p)
			}
		}
	} else {
		if in.FilePath == "" {
			return exitOK
		}
		if strings.Contains(in.FilePath, ".devrites/") {
			return exitOK
		}
		rel := strings.TrimPrefix(in.FilePath, projectDir+string(filepath.Separator))
		if !strings.Contains(touched, rel) && !strings.Contains(touched, in.FilePath) {
			bad = append(bad, in.FilePath)
		}
	}
	if len(bad) == 0 {
		return exitOK
	}

	if hookEnforce("DEVRITES_WRIGHT_SCOPE") {
		out, err := h.PreToolDeny(wrightDenyReason)
		if err != nil {
			debugf(stderr, "wright-scope: %v", err)
			return exitOK
		}
		fmt.Fprintln(stdout, out)
		return exitOK
	}
	_ = state.AppendLog(filepath.Join(dir, ".wright-scope.log"), "WOULD-BLOCK\t"+strings.Join(bad, ", "))
	return exitOK
}

// wsField mirrors `grep -iE <find> | head -1 | sed -E s/<strip>//`: the first line
// matching find (case-insensitive) with strip removed (case-sensitive).
func wsField(lines []string, find, strip *regexp.Regexp) string {
	for _, line := range lines {
		if find.MatchString(line) {
			return strip.ReplaceAllString(line, "")
		}
	}
	return ""
}

// wsGateCount mirrors `grep -ciE '^\s*status:\s*open' questions.md || echo 0`,
// INCLUDING grep's quirk: on a present file with zero matches grep prints "0" and
// exits 1, so `|| echo 0` appends a second "0" (the captured value is "0\n0").
func wsGateCount(qPath string) string {
	data, err := os.ReadFile(qPath)
	if err != nil {
		return "0"
	}
	n := 0
	for _, line := range wsSplitLines(data) {
		if openStatusRe.MatchString(line) {
			n++
		}
	}
	if n == 0 {
		return "0\n0"
	}
	return strconv.Itoa(n)
}

func wsOrZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func wsReadLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return wsSplitLines(data)
}

// wsSplitLines splits on '\n', dropping a single trailing newline's empty final
// element so a "\n"-terminated file yields N records, like grep/awk/sed.
func wsSplitLines(data []byte) []string {
	s := strings.TrimSuffix(string(data), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func wsIsDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func wsIsFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

package main

// Workspace hooks read state.md, questions.md, and other runtime files from the
// active feature directory. work is canonical, while features remains readable
// for compatibility. These hooks are read only and stay silent outside a
// DevRites workspace.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/forge"
	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/orient"
	drvreason "github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/rootfacts"
	"github.com/devrites/devrites/internal/safepath"
	"github.com/devrites/devrites/internal/state"
)

var (
	openStatusRe = regexp.MustCompile(`(?i)^[[:space:]]*status:[[:space:]]*open`)
)

// resolveWorkspaceDir finds the active feature directory. New workspaces use
// work, while existing features workspaces remain readable.
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
// ok is false when no .devrites root or active feature exists. Callers use that
// result to return silently.
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

func recordHookGuard(h harness.Harness, hook string, reasonID drvreason.ID, strength lib.GuardStrength, outcome lib.EventOutcome, evidenceFiles ...string) {
	root, slug, _, ok := resolveWorkspace()
	if !ok {
		return
	}
	ev := lib.NewEventV1(lib.BoundaryHookGuard, hook, reasonID)
	ev.GuardStrength = strength
	ev.Outcome = outcome
	ev.Host = lib.EventHost(h)
	if ev.Host != lib.HostClaude && ev.Host != lib.HostCodex {
		ev.Host = lib.HostEngine
	}
	ev.EvidencePaths = lib.WorkspaceEvidencePaths(root, slug, evidenceFiles...)
	if err := lib.BindEventWorkspace(root, slug, &ev); err != nil {
		return
	}
	_ = lib.AppendEventV1(root, ev)
}

// hookAUQ records one metadata-only v1 event for each AskUserQuestion item after
// the tool returns. Prompts and answers remain in questions.md and are not
// copied into telemetry.
func hookAUQ(stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(io.LimitReader(stdin, 1<<20))
	if err != nil {
		return exitOK
	}
	var payload struct {
		ToolInput struct {
			Questions []struct {
				Question string `json:"question"`
			} `json:"questions"`
		} `json:"tool_input"`
	}
	if json.Unmarshal(data, &payload) != nil || len(payload.ToolInput.Questions) == 0 {
		return exitOK
	}
	root, slug, _, ok := resolveWorkspace()
	if !ok {
		return exitOK
	}
	const maxObservedQuestions = 64
	for i, q := range payload.ToolInput.Questions {
		if i == maxObservedQuestions {
			break
		}
		if strings.TrimSpace(q.Question) == "" {
			continue
		}
		event := lib.NewEventV1(
			lib.BoundaryHookGuard,
			"human-wait-resumed",
			drvreason.HookStopUnsurfacedHumanGate,
		)
		event.GuardStrength = lib.GuardObserved
		event.Outcome = lib.OutcomeObserved
		event.Host = lib.HostClaude
		if err := lib.BindEventWorkspace(root, slug, &event); err != nil {
			continue
		}
		_ = lib.AppendEventV1(root, event)
	}
	return exitOK
}

// hookEvent records host lifecycle notifications as metadata-only v1 events. It
// does not change workflow state.
func hookEvent(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	_, _ = io.Copy(io.Discard, stdin)
	event := ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			continue
		}
		event = arg
		break
	}
	switch event {
	case "session-end":
		event = "session-ended"
	case "subagent-stop":
		event = "agent-finished"
	default:
		return exitOK
	}
	root, slug, _, ok := resolveWorkspace()
	if !ok {
		return exitOK
	}
	observation := lib.NewEventV1(lib.BoundaryAgentDispatch, event, drvreason.RootSelected)
	observation.GuardStrength = lib.GuardObserved
	observation.Outcome = lib.OutcomeObserved
	observation.Host = lib.HostClaude
	if err := lib.BindEventWorkspace(root, slug, &observation); err != nil {
		return exitOK
	}
	_ = lib.AppendEventV1(root, observation)
	return exitOK
}

// hookHandoffSnapshot appends a short resume note before context compaction. It
// does not rewrite canonical workspace artifacts.
func hookHandoffSnapshot(stdin io.Reader, stdout, stderr io.Writer) int {
	_, _ = io.Copy(io.Discard, stdin)
	_, slug, dir, ok := resolveWorkspace()
	if !ok {
		return exitOK
	}
	stateLines := wsReadLines(filepath.Join(dir, "state.md"))
	phase, _ := state.CursorField(stateLines, state.CursorPhase)
	status, _ := state.CursorField(stateLines, state.CursorStatus)
	next, _ := state.CursorField(stateLines, state.CursorNextAction)
	stamp := time.Now().UTC().Format(time.RFC3339)
	var b strings.Builder
	fmt.Fprintf(&b, "\n## Handoff snapshot: %s\n", stamp)
	fmt.Fprintf(&b, "- Feature: %s\n", slug)
	if phase != "" {
		fmt.Fprintf(&b, "- Phase: %s\n", phase)
	}
	if status != "" {
		fmt.Fprintf(&b, "- Status: %s\n", status)
	}
	if next != "" {
		fmt.Fprintf(&b, "- Next: %s\n", next)
	}
	fmt.Fprintf(&b, "- Open questions: %s\n", wsOrZero(wsGateCount(filepath.Join(dir, "questions.md"))))
	_ = state.AppendLog(filepath.Join(dir, "handoff.md"), strings.TrimRight(b.String(), "\n"))
	fmt.Fprintf(stdout, "DevRites: compaction handoff saved for %s; resume with the status command or read handoff.md.\n", slug)
	return exitOK
}

// hookCursor adds the active feature cursor to each UserPromptSubmit event so the
// next action appears near the end of the current context. It keeps the plain
// text output and silent nonproject behavior of devrites-cursor.sh.
func hookCursor(stdin io.Reader, stdout, stderr io.Writer) int {
	_, _ = io.Copy(io.Discard, stdin)
	root, slug, dir, ok := resolveWorkspace()
	if !ok || !wsIsDir(dir) {
		return exitOK
	}
	stateLines := wsReadLines(filepath.Join(dir, "state.md"))

	next, _ := state.CursorField(stateLines, state.CursorNextAction)
	status, _ := state.CursorField(stateLines, state.CursorStatus)
	gates := wsGateCount(filepath.Join(dir, "questions.md"))
	afk := ""
	if wsIsFile(filepath.Join(root, "AFK")) {
		afk, _ = state.CursorField(stateLines, state.CursorAFKSlicesRemaining)
	}

	fmt.Fprintf(stdout, "DevRites cursor: active feature: %s\n", slug)
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
		fmt.Fprintln(stdout, "  ⚠ tests/build RED: resolve before stopping")
	}
	return exitOK
}

// hookStatusline prints the one-line workspace status configured by settings.json.
// It reads and ignores the session JSON on stdin, matching
// devrites-statusline.sh, and stays silent without an active workspace.
func hookStatusline(stdin io.Reader, stdout, stderr io.Writer) int {
	_, _ = io.Copy(io.Discard, stdin)
	root, slug, dir, ok := resolveWorkspace()
	if !ok {
		return exitOK
	}
	phase, _ := state.CursorField(wsReadLines(filepath.Join(dir, "state.md")), state.CursorPhase)
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
	obs := lib.ObservabilityStatus(root, slug)
	if obs != "" {
		obs = " · " + obs
	}
	fmt.Fprintf(stdout, "DevRites: %s · %s · gates:%s · %s%s%s\n", slug, phase, gates, mode, red, obs)
	return exitOK
}

// redwatch detection patterns match devrites-redwatch.sh. They are case
// insensitive, and FAIL/PASS scans are line oriented.
// pack-scan-ignore: these are the hook's own red/green heuristics, not a payload.
var (
	redTestCmdRe = regexp.MustCompile(`(?i)(npm|pnpm|yarn|bun)([[:space:]]+run)?[[:space:]]+(test|build|lint|typecheck|check)|jest|vitest|pytest|go[[:space:]]+test|cargo[[:space:]]+(test|build|clippy)|\bmvn\b|gradle|eslint|ruff|mypy|\btsc\b|\bmake[[:space:]]+(test|build|check)`)
	redFailRe    = regexp.MustCompile(`(?im)(^|[^A-Za-z])FAIL([^A-Za-z]|$)|[1-9][0-9]*[[:space:]]+(failed|failing|failures?|errors?)|not ok|panic:|error TS[0-9]|Traceback|AssertionError|✗|✘|BUILD FAILED|exit code [1-9]`)
	redPassRe    = regexp.MustCompile(`(?im)PASS(ED)?|0[[:space:]]+(failed|failing|errors?)|all tests passed|BUILD SUCC(ESS|EEDED)|succeeded|no (errors|problems|issues)|(^|[^a-z])ok([^a-z]|$)|✓`)
)

const redReasonFmt = "DevRites: tests/build are RED (%s). Fix to green or record the failure + next step in state.md before stopping: the Stop gate blocks an end-of-turn while .red is set."

// hookRedwatch handles PostToolUse Bash events for test, build, lint, and type
// checking commands. It sets or clears <featureDir>/.red so the Stop gate can
// catch a known failing suite. Its command and output checks match
// devrites-redwatch.sh, and uncertain input leaves state unchanged.
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
	// Ignore commands unrelated to tests, builds, linting, or type checking.
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

// isEditTool reports whether a tool can write source. toolBaseName also handles
// host-qualified names such as functions.apply_patch and MCP symbol editors.
func isEditTool(tool string) bool {
	switch toolBaseName(tool) {
	case "edit", "write", "multiedit", "multi_edit", "notebookedit", "notebook_edit",
		"apply_patch", "applypatch", "edit_file", "write_file", "create_file", "delete_file",
		"replace_symbol_body", "insert_before_symbol", "insert_after_symbol", "safe_delete_symbol":
		return true
	}
	return false
}

// patchPaths returns the file paths an apply_patch command touches.
func patchPaths(command string) []string {
	var paths []string
	for _, m := range patchPathRe.FindAllStringSubmatch(command, -1) {
		paths = append(paths, strings.TrimSpace(m[1]))
	}
	return paths
}

// shellOutputRedirections removes harmless fd redirects and returns every file
// target of >, >>, or >|. Dynamic targets are opaque and therefore unsafe.
func shellOutputRedirections(command string) (sanitized string, paths []string, opaque bool) {
	buf := []byte(command)
	sanitizedBytes := append([]byte(nil), buf...)
	var quote byte
	for i := 0; i < len(buf); i++ {
		switch {
		case quote != 0:
			if buf[i] == '\\' && quote == '"' {
				i++
			} else if buf[i] == quote {
				quote = 0
			}
			continue
		case buf[i] == '\'' || buf[i] == '"':
			quote = buf[i]
			continue
		case buf[i] != '>':
			continue
		}

		start := i
		for start > 0 && buf[start-1] >= '0' && buf[start-1] <= '9' {
			start--
		}
		i++
		if i < len(buf) && (buf[i] == '>' || buf[i] == '|') {
			i++
		}
		for i < len(buf) && (buf[i] == ' ' || buf[i] == '\t') {
			i++
		}
		if i < len(buf) && buf[i] == '&' {
			i++
			for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
				i++
			}
			for j := start; j < i; j++ {
				sanitizedBytes[j] = ' '
			}
			i--
			continue
		}
		if i >= len(buf) || buf[i] == '(' {
			return "", paths, true
		}

		wordStart := i
		var wordQuote byte
		for i < len(buf) {
			c := buf[i]
			if wordQuote != 0 {
				if c == '\\' && wordQuote == '"' {
					i += 2
					continue
				}
				if c == wordQuote {
					wordQuote = 0
				}
				i++
				continue
			}
			if c == '\'' || c == '"' {
				wordQuote = c
				i++
				continue
			}
			if c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == ';' || c == '|' || c == '&' {
				break
			}
			i++
		}
		if wordQuote != 0 {
			return "", paths, true
		}
		target := strings.Trim(string(buf[wordStart:i]), `"'`)
		if target == "" || strings.ContainsAny(target, "$`*?[]{}()") {
			return "", paths, true
		}
		if target != "/dev/null" {
			paths = append(paths, target)
		}
		for j := start; j < i; j++ {
			sanitizedBytes[j] = ' '
		}
		i--
	}
	return string(sanitizedBytes), paths, false
}

// underDevrites reports whether an absolute path is inside the .devrites root.
func underDevrites(abs, root string) bool {
	return pathWithin(abs, root)
}

func pathWithin(path, parent string) bool {
	path = filepath.Clean(path)
	parent = filepath.Clean(parent)
	return path == parent || strings.HasPrefix(path, parent+string(filepath.Separator))
}

const a1DenyReason = "DevRites A1: the orchestrator must not edit source mid-build: the slice-wright is the only writer. Re-dispatch the wright (continue it once) or stop + escalate; do not patch the code yourself. (devrites-a1-guard)"

// hookA1Guard prevents /rite-build from editing source while slice-wright owns an
// open build window. It matches devrites-a1-guard.sh by logging observations to
// .a1-guard.log unless enforcement is enabled.
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
	// The guard is active between a reconcile snapshot and its check. Inline
	// fallback makes the orchestrator the expected writer.
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
	// The wright may edit. Claude sends agent_id; Codex sends agent_type.
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
		recordHookGuard(h, "a1-guard", drvreason.HookA1Denied, lib.GuardEnforced, lib.OutcomeDenied, ".reconcile-base")
		return exitOK
	}
	_ = state.AppendLog(filepath.Join(dir, ".a1-guard.log"), "WOULD-BLOCK\t"+in.ToolName+"\t"+target)
	recordHookGuard(h, "a1-guard", drvreason.HookA1Observed, lib.GuardObserved, lib.OutcomeObserved, ".reconcile-base")
	return exitOK
}

const (
	wrightDenyReason       = "DevRites scope: this write is not in the orchestrator-provided exact wright allowlist, or it targets .devrites. Build only the slice contract; return the blocked path to the orchestrator instead of widening scope. (devrites-wright-scope)"
	wrightForbiddenReason  = "DevRites scope: slice-wrights cannot dispatch nested agents, commit/push, install dependencies, run live migrations/deployments, or execute an uninspectable write path. (devrites-wright-scope)"
	forgeWrightDenyReason  = "DevRites Forge scope: this candidate does not match its root-owned manifest, physical worktree, Git identity, branch, worker identity, or live process token. Return to the orchestrator; never widen or guess Forge ownership. (devrites-forge-wright-scope)"
	wrightAllowlistEnvName = "DEVRITES_WRIGHT_ALLOWLIST_FILE"
	forgeRunEnvName        = "DEVRITES_FORGE_RUN_ID"
	forgeCandidateEnvName  = "DEVRITES_FORGE_CANDIDATE"
	forgeWorkerEnvName     = "DEVRITES_FORGE_WORKER_ID"
	forgePIDEnvName        = "DEVRITES_FORGE_WORKER_PID"
	forgeProcessEnvName    = "DEVRITES_FORGE_PROCESS_START"
)

var wrightForbiddenShellRe = regexp.MustCompile(`(?im)\bgit[[:space:]]+(add|commit|push|reset|checkout|restore|clean|rm|mv|stash|tag|apply|merge|rebase|cherry-pick|am)\b|\b(npm|pnpm|yarn)[[:space:]]+(install|i|ci|add|publish)\b|\b(pip|pip3)[[:space:]]+install\b|\b(poetry|uv)[[:space:]]+(add|install)\b|\b(bundle|composer)[[:space:]]+(install|update)\b|\bgo[[:space:]]+(get|install|generate)\b|\bcargo[[:space:]]+(add|install|publish)\b|\b(devrites-engine[[:space:]]+migrate|alembic[[:space:]]+(upgrade|downgrade)|prisma[[:space:]]+migrate|flyway[[:space:]]+migrate|liquibase[[:space:]]+update|manage\.py[[:space:]]+migrate|(rails|rake)[[:space:]]+db:(migrate|rollback|seed)|sequelize[[:space:]]+db:migrate|knex[[:space:]]+migrate:latest)\b|\b(terraform[[:space:]]+(apply|destroy)|kubectl[[:space:]]+(apply|create|delete|patch|replace)|helm[[:space:]]+(install|upgrade|uninstall)|docker[[:space:]]+push)\b|\b(curl|wget|scp|ssh|nc|socat|telnet)\b|\b(codex|claude)[[:space:]]+(exec|agent)\b`)

var opaqueInterpreterRe = regexp.MustCompile(`(?im)(^|[;&|[:space:]])(python[0-9.]*[[:space:]]+(-c|-)|node[[:space:]]+(-e|--)|ruby[[:space:]]+-e|perl[[:space:]]+(-e|-p|-i)|[.][/][^[:space:]]+\.(sh|py|js|rb))([;&|[:space:]]|$)`)

// hookWrightScope restricts the write-capable leaf to the orchestrator's exact
// path allowlist. It always enforces active DevRites runs.
func hookWrightScope(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	if !bytes.Contains(data, []byte(`"tool_name"`)) {
		return exitOK
	}
	in := h.ParseGuardInput(bytes.NewReader(data))
	kind := devritesAgent(in.AgentType)
	forgeDeclared := forgeWrightDeclared()
	active := kind != devritesAgentNone
	if forgeDeclared && kind != devritesAgentWright {
		return denyWright(h, stdout, stderr, forgeWrightDenyReason, drvreason.HookForgeBindingDenied)
	}
	if kind == devritesAgentNone && (in.AgentType != "" || os.Getenv("DEVRITES_WRIGHT_AGENT_REQUIRED") == "1") {
		return exitOK
	}
	if kind == devritesAgentReadonly {
		return exitOK // reviewer-readonly owns every non-wright leaf
	}
	if kind == devritesAgentInvalid {
		if !isEditTool(in.ToolName) && !isShellTool(in.ToolName) &&
			!isOpaqueExecutionTool(in.ToolName) && !isAgentDispatchTool(in.ToolName) {
			return exitOK
		}
		return denyWright(h, stdout, stderr, reviewerReadonlyDenyReason, drvreason.HookReviewerReadonlyDenied)
	}

	var targets []string
	forbidden := isAgentDispatchTool(in.ToolName) || isOpaqueExecutionTool(in.ToolName)
	switch {
	case forbidden:
	case isEditTool(in.ToolName):
		targets = mutationToolPaths(data, in)
		if len(targets) == 0 {
			forbidden = true
		}
	case isShellTool(in.ToolName):
		targets, forbidden = wrightShellWritePaths(in.Command)
		if !forbidden && len(targets) == 0 {
			return exitOK
		}
	default:
		return exitOK
	}

	root, slug, dir, ok := resolveWorkspace()
	if !ok {
		if kind == devritesAgentWright {
			return denyWright(h, stdout, stderr, wrightForbiddenReason, drvreason.HookWrightForbiddenDenied)
		}
		return exitOK
	}

	if wrightForbiddenShellRe.MatchString(in.Command) ||
		(opaqueInterpreterRe.MatchString(in.Command) && !safeReadonlyShellCommand(in.Command)) {
		forbidden = true
	}
	if forbidden {
		return denyOrObserveWright(h, stdout, stderr, dir, active, wrightForbiddenReason, drvreason.HookWrightForbiddenDenied, []string{"opaque/forbidden operation"})
	}

	projectDir := filepath.Dir(root)
	if !forgeDeclared && kind == devritesAgentWright && insideForgeStaging(projectDir) {
		return denyOrObserveWright(h, stdout, stderr, dir, true, forgeWrightDenyReason, drvreason.HookForgeBindingDenied, []string{"missing Forge binding"})
	}
	if forgeDeclared {
		projectDir, err = forgeWrightProjectDir(projectDir, slug, in)
		if err != nil {
			debugf(stderr, "wright-scope Forge binding: %v", err)
			return denyOrObserveWright(h, stdout, stderr, dir, true, forgeWrightDenyReason, drvreason.HookForgeBindingDenied, []string{"invalid Forge binding"})
		}
	}
	allow, err := readWrightAllowlist(projectDir, root, dir)
	if err != nil {
		debugf(stderr, "wright-scope allowlist: %v", err)
		return denyOrObserveWright(h, stdout, stderr, dir, active, wrightDenyReason, drvreason.HookWrightScopeDenied, []string{"invalid/missing allowlist"})
	}
	var bad []string
	for _, target := range targets {
		rel, err := normalizeProjectPath(projectDir, root, target, false)
		if err != nil {
			bad = append(bad, target)
			continue
		}
		if _, ok := allow[rel]; !ok {
			bad = append(bad, target)
		}
	}
	if len(bad) == 0 {
		return exitOK
	}
	return denyOrObserveWright(h, stdout, stderr, dir, active, wrightDenyReason, drvreason.HookWrightScopeDenied, bad)
}

func forgeWrightDeclared() bool {
	for _, name := range []string{
		forgeRunEnvName,
		forgeCandidateEnvName,
		forgeWorkerEnvName,
		forgePIDEnvName,
		forgeProcessEnvName,
	} {
		if os.Getenv(name) != "" {
			return true
		}
	}
	return false
}

func insideForgeStaging(primaryProject string) bool {
	cwd, err := physicalWorkingDir()
	if err != nil {
		return false
	}
	primary, err := filepath.Abs(primaryProject)
	if err != nil {
		return false
	}
	primary, err = filepath.EvalSymlinks(primary)
	if err != nil {
		return false
	}
	staging := filepath.Join(filepath.Dir(primary), "."+filepath.Base(primary)+".devrites-forge")
	rel, err := filepath.Rel(staging, cwd)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func forgeWrightProjectDir(primaryProject, activeSlug string, in harness.GuardInput) (string, error) {
	values := map[string]string{
		forgeRunEnvName:       os.Getenv(forgeRunEnvName),
		forgeCandidateEnvName: os.Getenv(forgeCandidateEnvName),
		forgeWorkerEnvName:    os.Getenv(forgeWorkerEnvName),
		forgePIDEnvName:       os.Getenv(forgePIDEnvName),
		forgeProcessEnvName:   os.Getenv(forgeProcessEnvName),
	}
	for name, value := range values {
		if value == "" || strings.TrimSpace(value) != value {
			return "", fmt.Errorf("%s must be present and exact", name)
		}
	}

	manifest, _, err := forge.Load(primaryProject, values[forgeRunEnvName])
	if err != nil {
		return "", err
	}
	if manifest.FeatureSlug != activeSlug {
		return "", fmt.Errorf("Forge run belongs to feature %q, not active feature %q", manifest.FeatureSlug, activeSlug)
	}
	candidate, err := manifest.Candidate(forge.CandidateID(values[forgeCandidateEnvName]))
	if err != nil {
		return "", err
	}
	if candidate.State != forge.StateRunning {
		return "", fmt.Errorf("Forge candidate %s is %s, not running", candidate.ID, candidate.State)
	}
	pid, err := strconv.Atoi(values[forgePIDEnvName])
	if err != nil || strconv.Itoa(pid) != values[forgePIDEnvName] {
		return "", fmt.Errorf("%s must be a canonical positive PID", forgePIDEnvName)
	}
	if candidate.Worker.ID != values[forgeWorkerEnvName] ||
		candidate.Worker.PID != pid ||
		candidate.Worker.ProcessStart != values[forgeProcessEnvName] {
		return "", fmt.Errorf("Forge worker identity does not match the manifest")
	}
	if in.AgentID != "" && in.AgentID != candidate.Worker.ID {
		return "", fmt.Errorf("hook agent identity does not match Forge worker %q", candidate.Worker.ID)
	}
	liveToken, err := forge.ProcessStartToken(candidate.Worker.PID)
	if err != nil || liveToken != candidate.Worker.ProcessStart {
		return "", fmt.Errorf("Forge worker liveness token is no longer valid")
	}

	cwd, err := physicalWorkingDir()
	if err != nil {
		return "", err
	}
	if filepath.Clean(cwd) != filepath.Clean(candidate.Worktree) {
		return "", fmt.Errorf("physical cwd %q is not candidate %s worktree %q", cwd, candidate.ID, candidate.Worktree)
	}
	facts, _ := rootfacts.ResolveFrom(cwd, "")
	if facts.Git.TopLevel != candidate.Worktree || facts.Git.CommonDir != manifest.GitCommonDir {
		return "", fmt.Errorf("Forge candidate Git identity does not match the manifest")
	}
	cmd := exec.Command("git", "-C", cwd, "symbolic-ref", "--quiet", "--short", "HEAD")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	branch, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(branch)) != candidate.Branch {
		return "", fmt.Errorf("Forge candidate branch does not match the manifest")
	}
	return candidate.Worktree, nil
}

func physicalWorkingDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(cwd)
}

func mutationToolPaths(data []byte, in harness.GuardInput) []string {
	var envelope struct {
		ToolInput any `json:"tool_input"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return nil
	}
	var paths []string
	if toolBaseName(in.ToolName) == "apply_patch" || toolBaseName(in.ToolName) == "applypatch" {
		collectPatchMutationPaths(envelope.ToolInput, &paths)
		paths = append(paths, patchPaths(in.Command)...)
		return uniqueStrings(paths)
	}
	collectMutationPaths(envelope.ToolInput, &paths)
	if len(paths) == 0 && in.FilePath != "" {
		paths = append(paths, in.FilePath)
	}
	return uniqueStrings(paths)
}

func collectPatchMutationPaths(value any, paths *[]string) {
	switch value := value.(type) {
	case string:
		*paths = append(*paths, patchPaths(value)...)
	case map[string]any:
		for _, child := range value {
			collectPatchMutationPaths(child, paths)
		}
	case []any:
		for _, child := range value {
			collectPatchMutationPaths(child, paths)
		}
	}
}

func collectMutationPaths(value any, paths *[]string) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			normalizedKey := strings.ReplaceAll(strings.ToLower(key), "-", "_")
			switch normalizedKey {
			case "file_path", "filepath", "path", "relative_path", "notebook_path", "new_path", "target_path":
				if path, ok := child.(string); ok && strings.TrimSpace(path) != "" {
					*paths = append(*paths, strings.TrimSpace(path))
					continue
				}
			}
			collectMutationPaths(child, paths)
		}
	case []any:
		for _, child := range value {
			collectMutationPaths(child, paths)
		}
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func wrightShellWritePaths(command string) ([]string, bool) {
	if strings.TrimSpace(command) == "" || wrightForbiddenShellRe.MatchString(command) {
		return nil, true
	}
	sanitized, paths, opaque := shellOutputRedirections(command)
	if opaque {
		return nil, true
	}

	patches := patchPaths(command)
	if strings.Contains(command, "apply_patch") {
		if len(patches) == 0 {
			return nil, true
		}
		paths = append(paths, patches...)
		sanitized = strings.ReplaceAll(sanitized, "apply_patch", "true")
	}

	for _, segment := range shellSegmentRe.Split(sanitized, -1) {
		if safeReadonlyShellSegment(segment) {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(segment))
		if len(fields) == 0 {
			continue
		}
		for len(fields) > 0 && filepath.Base(fields[0]) == "rtk" {
			fields = fields[1:]
		}
		if len(fields) == 0 {
			return nil, true
		}
		base := strings.ToLower(filepath.Base(strings.Trim(fields[0], `"'`)))
		args := fields[1:]
		var found []string
		switch base {
		case "gofmt", "goimports", "shfmt":
			if !hasAnyArg(args, "-w") {
				return nil, true
			}
			found = nonFlagArgs(args)
		case "prettier":
			if !hasAnyArg(args, "--write") {
				return nil, true
			}
			found = nonFlagArgs(args)
		case "ruff":
			if len(args) == 0 || args[0] != "format" || hasAnyArg(args, "--check") {
				return nil, true
			}
			found = nonFlagArgs(args[1:])
		case "black", "rustfmt":
			found = nonFlagArgs(args)
		case "rm", "unlink", "touch", "mkdir", "rmdir", "truncate":
			found = nonFlagArgs(args)
		default:
			return nil, true
		}
		if len(found) == 0 {
			return nil, true
		}
		for _, path := range found {
			if strings.ContainsAny(path, "$`*?[]{}()") {
				return nil, true
			}
		}
		paths = append(paths, found...)
	}

	if len(paths) == 0 {
		if safeReadonlyShellCommand(command) {
			return nil, false
		}
		return nil, true
	}
	return uniqueStrings(paths), false
}

func nonFlagArgs(args []string) []string {
	var out []string
	for _, arg := range args {
		arg = strings.Trim(arg, `"'`)
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		out = append(out, arg)
	}
	return out
}

func readWrightAllowlist(projectDir, devritesRoot, workspaceDir string) (map[string]struct{}, error) {
	path := strings.TrimSpace(os.Getenv(wrightAllowlistEnvName))
	if path == "" {
		path = filepath.Join(workspaceDir, ".wright-allowlist")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(devritesRoot), path)
	}
	path = filepath.Clean(path)
	if !safepath.WithinResolved(path, workspaceDir) {
		return nil, fmt.Errorf("allowlist file is outside the active workspace")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	allow := make(map[string]struct{})
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if line == "" {
			continue
		}
		if strings.TrimSpace(line) != line || strings.HasPrefix(line, "#") {
			return nil, fmt.Errorf("allowlist entry is not an exact normalized path: %q", line)
		}
		rel, err := normalizeProjectPath(projectDir, devritesRoot, line, true)
		if err != nil {
			return nil, err
		}
		if _, duplicate := allow[rel]; duplicate {
			return nil, fmt.Errorf("duplicate allowlist path: %q", line)
		}
		if info, statErr := os.Stat(filepath.Join(projectDir, filepath.FromSlash(rel))); statErr == nil && info.IsDir() {
			return nil, fmt.Errorf("allowlist path names a directory: %q", line)
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return nil, statErr
		}
		allow[rel] = struct{}{}
	}
	return allow, nil
}

func normalizeProjectPath(projectDir, devritesRoot, raw string, requireNormalized bool) (string, error) {
	if raw == "" || strings.ContainsRune(raw, 0) || strings.ContainsRune(raw, '\\') {
		return "", fmt.Errorf("invalid project path %q", raw)
	}
	var abs string
	if filepath.IsAbs(raw) {
		if requireNormalized {
			return "", fmt.Errorf("allowlist path must be project-relative: %q", raw)
		}
		abs = filepath.Clean(raw)
	} else {
		clean := filepath.Clean(filepath.FromSlash(raw))
		if requireNormalized && filepath.ToSlash(clean) != raw {
			return "", fmt.Errorf("allowlist path is not normalized: %q", raw)
		}
		abs = filepath.Join(projectDir, clean)
	}
	rel, err := filepath.Rel(projectDir, abs)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path is outside the project: %q", raw)
	}
	if !safepath.WithinResolved(abs, projectDir) || safepath.WithinResolved(abs, devritesRoot) {
		return "", fmt.Errorf("path escapes the project or targets .devrites: %q", raw)
	}
	return filepath.ToSlash(rel), nil
}

func denyOrObserveWright(h harness.Harness, stdout, stderr io.Writer, dir string, active bool, message string, reasonID drvreason.ID, bad []string) int {
	if active || hookEnforce("DEVRITES_WRIGHT_SCOPE") {
		return denyWright(h, stdout, stderr, message, reasonID)
	}
	_ = state.AppendLog(filepath.Join(dir, ".wright-scope.log"), "WOULD-BLOCK\t"+strings.Join(bad, ", "))
	recordHookGuard(h, "wright-scope", drvreason.HookWrightScopeObserved, lib.GuardObserved, lib.OutcomeObserved)
	return exitOK
}

func denyWright(h harness.Harness, stdout, stderr io.Writer, message string, reasonID drvreason.ID) int {
	out, err := h.PreToolDeny(message)
	if err != nil {
		debugf(stderr, "wright-scope: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	recordHookGuard(h, "wright-scope", reasonID, lib.GuardEnforced, lib.OutcomeDenied)
	return exitOK
}

// wsGateCount mirrors `grep -ciE '^\s*status:\s*open' questions.md || echo 0`.
// On a present file with no matches, grep prints "0" and exits 1, so the fallback
// appends another "0". The captured value is therefore "0\n0".
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

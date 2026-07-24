package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/orient"
	drvreason "github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
)

// hookOrient emits SessionStart context for the active feature. It is read only
// and returns silently when the workspace is missing, unreadable, or has nothing
// to report. An inline shell guard handles a missing binary; this function
// handles a present binary outside a workspace.
func hookOrient(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	// Orientation depends only on workspace files, so read and discard the
	// SessionStart payload.
	_, _ = io.Copy(io.Discard, stdin)

	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return exitOK // Stay silent outside a DevRites workspace.
	}
	text, has, err := orient.Digest(root)
	if err != nil {
		debugf(stderr, "orient: %v", err)
		return exitOK
	}
	if !has {
		// On the first session without an active feature, print a starting hint
		// based on the repository.
		if text, has = orient.FirstRunDigest(root); !has {
			return exitOK
		}
	}
	out, err := h.SessionStartContext(text)
	if err != nil {
		debugf(stderr, "orient: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	return exitOK
}

// hookStopGate catches inconsistent state when a turn ends. It mirrors
// devrites-stop-gate.sh, observes by default, and blocks only when
// DEVRITES_STOP_GATE=enforce. stop_hook_active prevents a blocking loop, and
// parsing or workspace errors do not block.
func hookStopGate(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	in := h.ParseStopInput(stdin)
	if !in.StopHookActivePresent {
		recordHookGuard(h, "stop-gate", drvreason.HookStopInputInvalid, lib.GuardUnavailable, lib.OutcomeUnavailable)
		return exitOK // Invalid input does not block.
	}
	if in.StopHookActive {
		recordHookGuard(h, "stop-gate", drvreason.HookStopReentry, lib.GuardBypassed, lib.OutcomeBypassed)
		return exitOK // A reentered Stop must finish.
	}
	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return exitOK
	}
	res, err := gate.StopGate(root)
	if err != nil {
		recordHookGuard(h, "stop-gate", drvreason.HookStopWorkspaceUnavailable, lib.GuardUnavailable, lib.OutcomeUnavailable)
		return exitOK
	}
	if res.ReasonID == "" {
		recordHookGuard(h, "stop-gate", drvreason.HookStopWorkspaceUnavailable, lib.GuardUnavailable, lib.OutcomeUnavailable)
		return exitOK
	}
	if !res.Blocked {
		strength := lib.GuardObserved
		if hookEnforce("DEVRITES_STOP_GATE") {
			strength = lib.GuardEnforced
		}
		recordHookGuard(h, "stop-gate", res.ReasonID, strength, lib.OutcomePassed)
		return exitOK
	}
	if !hookEnforce("DEVRITES_STOP_GATE") {
		// Observe mode records the finding without blocking. O_APPEND keeps each
		// log entry intact when DevRites runs concurrent processes.
		logPath := filepath.Join(resolveWorkspaceDir(root, res.Slug), ".stop-gate.log")
		if err := state.AppendLog(logPath, "WOULD-BLOCK\t"+res.Reason); err != nil {
			debugf(stderr, "stop-gate log: %v", err)
		}
		recordHookGuard(h, "stop-gate", res.ReasonID, lib.GuardObserved, lib.OutcomeObserved, res.EvidenceFiles...)
		return exitOK
	}
	out, err := h.StopBlock(res.Reason)
	if err != nil {
		debugf(stderr, "stop-gate: %v", err)
		recordHookGuard(h, "stop-gate", res.ReasonID, lib.GuardUnavailable, lib.OutcomeUnavailable, res.EvidenceFiles...)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	recordHookGuard(h, "stop-gate", res.ReasonID, lib.GuardEnforced, lib.OutcomeBlocked, res.EvidenceFiles...)
	return exitOK
}

// allowReason is the auto-approval message the allow hook emits.
const allowReason = "DevRites read-only orientation/gate command: auto-approved by the devrites-allow hook"

var allowReadonlyCommands = map[string]bool{
	"check-acceptance": true,
	"doubt-coverage":   true,
	"evidence-fresh":   true,
	"preamble":         true,
	"progress":         true,
	"readiness":        true,
	"review-integrity": true,
}

var allowReadonlySubcommands = map[string]map[string]bool{
	"footprint":      {"render": true, "roster": true},
	"ledger":         {"diff": true, "validate": true, "list": true, "show": true},
	"reviewer-stats": {"report": true},
}

// hookAllow approves exact read-only DevRites orientation and gate commands.
// Any other command stays in the harness's normal permission flow.
func hookAllow(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	// Ignore commands that do not mention DevRites.
	if !bytes.Contains(data, []byte("devrites")) {
		return exitOK
	}
	cmd := h.ParsePreToolInput(bytes.NewReader(data)).Command
	if cmd == "" {
		return exitOK
	}
	if !safeAllowCommand(cmd) {
		return exitOK
	}
	out, err := h.PreToolAllow(allowReason)
	if err != nil {
		debugf(stderr, "allow: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	recordHookGuard(h, "allow", drvreason.HookAllowApproved, lib.GuardEnforced, lib.OutcomeAllowed)
	return exitOK
}

func safeAllowCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}
	for _, prefix := range []string{
		"command -v devrites-engine >/dev/null 2>&1 && ",
		"command -v devrites-engine && ",
	} {
		if strings.HasPrefix(cmd, prefix) {
			cmd = strings.TrimSpace(strings.TrimPrefix(cmd, prefix))
			break
		}
	}
	if strings.ContainsAny(cmd, "\n\r;|&<>$`\\\"'(){}[]*?!") {
		return false
	}
	fields := strings.Fields(cmd)
	if len(fields) < 2 || fields[0] != "devrites-engine" {
		return false
	}
	return safeAllowShape(fields[1], fields[2:])
}

func safeAllowShape(cmd string, args []string) bool {
	if subs, ok := allowReadonlySubcommands[cmd]; ok {
		if len(args) == 0 || !subs[args[0]] {
			return false
		}
		args = args[1:]
	} else if !allowReadonlyCommands[cmd] {
		return false
	}
	return safeAllowArgs(cmd, args)
}

func safeAllowArgs(cmd string, args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "--json":
				continue
			default:
				return false
			}
		}
		if !safeAllowArg(arg) {
			return false
		}
	}
	return true
}

func safeAllowArg(arg string) bool {
	if arg == "" || strings.HasPrefix(arg, "-") {
		return false
	}
	if arg == "." || arg == ".." || strings.HasPrefix(arg, "/") || strings.Contains(arg, "..") || strings.ContainsRune(arg, '\\') {
		return false
	}
	for _, r := range arg {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case strings.ContainsRune("._@:+,-/", r):
		default:
			return false
		}
	}
	return true
}

// reviewerReadonlyDenyReason is the exact denial message
// devrites-reviewer-readonly.sh emits. It must stay byte-identical to keep the
// parity oracle green.
const reviewerReadonlyDenyReason = "DevRites: reviewers are read-only. This Bash command can mutate or exfiltrate; inspect with Read/Grep/Glob and return findings: do not modify the tree or reach the network. (devrites-reviewer-readonly)"

// reviewerMutateRe is the legacy observe-mode deny-list. Active DevRites leaves
// use the fail-closed safe-command check below instead.
var reviewerMutateRe = regexp.MustCompile(`(?im)>>|[^0-9 ]>[^>&]|[[:space:]]-i([[:space:]]|$)|sed[[:space:]]+-i|\b(rm|mv|cp|touch|mkdir|rmdir|tee|truncate|dd|chmod|chown|ln|unlink|install|patch)\b|git[[:space:]]+(add|commit|push|reset|checkout|restore|clean|rm|mv|stash|tag|apply|merge|rebase)|npm[[:space:]]+(install|i|ci|publish)|pnpm[[:space:]]+(install|add|publish)|yarn[[:space:]]+(add|install|publish)|pip[[:space:]]+install|go[[:space:]]+(get|install|generate)|cargo[[:space:]]+(add|install|publish)|\b(curl|wget|scp|ssh|nc|socat|telnet)\b`)

var shellSegmentRe = regexp.MustCompile(`\|\||&&|[;|\n]`)

type devritesAgentKind uint8

const (
	devritesAgentNone devritesAgentKind = iota
	devritesAgentReadonly
	devritesAgentWright
	devritesAgentInvalid
)

// devritesAgent identifies the one write-capable leaf without copying the agent
// roster. DEVRITES_AGENT_RUN supplies identity when a hook payload cannot carry
// agent_type.
func devritesAgent(payloadAgent string) devritesAgentKind {
	payloadAgent = strings.TrimSpace(payloadAgent)
	envAgent := strings.TrimSpace(os.Getenv("DEVRITES_ACTIVE_AGENT"))
	declared := os.Getenv("DEVRITES_AGENT_RUN") == "1" || strings.HasPrefix(payloadAgent, "devrites-")
	if !declared {
		return devritesAgentNone
	}
	if payloadAgent != "" && envAgent != "" && payloadAgent != envAgent {
		return devritesAgentInvalid
	}
	agent := payloadAgent
	if agent == "" {
		agent = envAgent
	}
	switch {
	case agent == "devrites-slice-wright":
		return devritesAgentWright
	case strings.HasPrefix(agent, "devrites-") && len(agent) > len("devrites-"):
		return devritesAgentReadonly
	default:
		return devritesAgentInvalid
	}
}

func toolBaseName(tool string) string {
	tool = strings.ToLower(strings.TrimSpace(tool))
	for _, sep := range []string{"::", "__", "/", "."} {
		if i := strings.LastIndex(tool, sep); i >= 0 {
			tool = tool[i+len(sep):]
		}
	}
	return strings.ReplaceAll(tool, "-", "_")
}

func isShellTool(tool string) bool {
	switch toolBaseName(tool) {
	case "bash", "shell", "sh", "exec_command", "run_command":
		return true
	}
	return false
}

func isOpaqueExecutionTool(tool string) bool {
	switch toolBaseName(tool) {
	case "exec", "js", "python", "computer", "computer_use", "write_stdin", "run_code":
		return true
	}
	return false
}

func isAgentDispatchTool(tool string) bool {
	switch toolBaseName(tool) {
	case "agent", "task", "spawn_agent", "delegate", "dispatch_agent", "create_agent":
		return true
	}
	return false
}

// safeReadonlyShellCommand accepts common inspection and proof commands. A
// read-only leaf can use Read or Grep, or ask the orchestrator to run anything
// outside this conservative set.
func safeReadonlyShellCommand(command string) bool {
	command, writeTargets, opaqueRedirect := shellOutputRedirections(command)
	if opaqueRedirect || len(writeTargets) != 0 ||
		strings.Contains(command, "$(") || strings.ContainsRune(command, '`') ||
		reviewerMutateRe.MatchString(command) {
		return false
	}
	segments := shellSegmentRe.Split(command, -1)
	if len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if !safeReadonlyShellSegment(segment) {
			return false
		}
	}
	return true
}

func safeReadonlyShellSegment(segment string) bool {
	fields := strings.Fields(strings.TrimSpace(segment))
	for len(fields) > 0 && strings.Contains(fields[0], "=") && !strings.HasPrefix(fields[0], "=") {
		fields = fields[1:]
	}
	for len(fields) > 0 {
		switch filepath.Base(fields[0]) {
		case "rtk", "timeout", "nice":
			fields = fields[1:]
			continue
		case "command":
			if len(fields) == 2 && fields[1] == "-v" {
				return true
			}
			fields = fields[1:]
			continue
		case "env":
			fields = fields[1:]
			for len(fields) > 0 && (strings.HasPrefix(fields[0], "-") || strings.Contains(fields[0], "=")) {
				fields = fields[1:]
			}
			continue
		}
		break
	}
	if len(fields) == 0 {
		return false
	}

	base := strings.ToLower(filepath.Base(strings.Trim(fields[0], `"'`)))
	args := fields[1:]
	switch base {
	case "true", "false", "pwd", "ls", "cat", "head", "tail", "less", "more",
		"grep", "rg", "wc", "sort", "uniq", "cut", "tr", "stat", "file",
		"readlink", "realpath", "basename", "dirname", "cmp", "diff", "jq", "yq",
		"tree", "du", "df", "printenv", "which", "date", "uname", "id", "whoami",
		"ps", "echo", "printf", "test", "[", "cd":
		return true
	case "find":
		return !hasAnyArg(args, "-delete", "-exec", "-execdir", "-ok", "-okdir", "-fprint", "-fprintf")
	case "sed":
		return !hasArgPrefix(args, "-i")
	case "git":
		return len(args) > 0 && hasAnyArg(args[:1], "diff", "status", "show", "log", "rev-parse", "ls-files", "grep", "blame")
	case "go":
		if len(args) == 0 || !hasAnyArg(args[:1], "test", "vet", "list", "version", "env") {
			return false
		}
		return args[0] != "env" || !hasAnyArg(args[1:], "-w", "-u")
	case "npm", "pnpm", "yarn":
		return safePackageProof(args)
	case "pytest", "rspec":
		return true
	case "python", "python3":
		return len(args) >= 2 && args[0] == "-m" && hasAnyArg(args[1:2], "pytest", "unittest", "compileall")
	case "cargo":
		if len(args) == 0 {
			return false
		}
		if hasAnyArg(args[:1], "test", "check", "clippy") {
			return true
		}
		return args[0] == "fmt" && hasAnyArg(args[1:], "--check")
	case "ruff":
		return len(args) > 0 && args[0] == "check" && !hasAnyArg(args[1:], "--fix")
	case "prettier":
		return hasAnyArg(args, "--check", "--list-different")
	case "eslint":
		return !hasAnyArg(args, "--fix", "--fix-dry-run")
	case "tsc":
		return hasAnyArg(args, "--noEmit")
	case "node":
		return hasAnyArg(args, "--check", "--test", "--version", "-v")
	case "bash", "sh", "zsh":
		return len(args) > 0 && args[0] == "-n"
	case "make":
		return len(args) > 0 && hasAnyArg(args[:1], "test", "check", "lint", "vet")
	case "devrites-engine":
		return len(args) > 0 && safeAllowShape(args[0], args[1:])
	}
	return false
}

func safePackageProof(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "test" {
		return true
	}
	return len(args) > 1 && args[0] == "run" &&
		hasAnyArg(args[1:2], "test", "lint", "check", "typecheck", "type-check")
}

func hasAnyArg(args []string, wants ...string) bool {
	for _, arg := range args {
		for _, want := range wants {
			if arg == want {
				return true
			}
		}
	}
	return false
}

func hasArgPrefix(args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}

// hookReviewerReadonly keeps every DevRites leaf except slice-wright read only.
// It always enforces declared DevRites runs. Undeclared runs keep the legacy
// observe mode.
func hookReviewerReadonly(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	if !bytes.Contains(data, []byte(`"tool_name"`)) {
		return exitOK
	}
	in := h.ParseGuardInput(bytes.NewReader(data))
	kind := devritesAgent(in.AgentType)

	// Do not let a global hook interfere with the main thread or another
	// product's agent when no DevRites run is declared.
	if kind == devritesAgentNone && (in.AgentType != "" || os.Getenv("DEVRITES_REVIEWER_AGENT_REQUIRED") == "1") {
		return exitOK
	}
	if kind == devritesAgentWright {
		if !isAgentDispatchTool(in.ToolName) {
			return exitOK
		}
	} else {
		active := kind != devritesAgentNone
		unsafe := isEditTool(in.ToolName) || isOpaqueExecutionTool(in.ToolName) || isAgentDispatchTool(in.ToolName)
		if isShellTool(in.ToolName) {
			if in.Command == "" {
				unsafe = active
			} else if active {
				unsafe = !safeReadonlyShellCommand(in.Command)
			} else {
				unsafe = reviewerMutateRe.MatchString(in.Command)
			}
		}
		if !unsafe {
			return exitOK
		}
	}

	active := kind != devritesAgentNone
	if active || hookEnforce("DEVRITES_REVIEWER_RO") {
		out, err := h.PreToolDeny(reviewerReadonlyDenyReason)
		if err != nil {
			debugf(stderr, "reviewer-readonly: %v", err)
			return exitOK
		}
		fmt.Fprintln(stdout, out)
		recordHookGuard(h, "reviewer-readonly", drvreason.HookReviewerReadonlyDenied, lib.GuardEnforced, lib.OutcomeDenied)
		return exitOK
	}

	if root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT")); err == nil {
		if slug, _ := orient.ActiveSlug(root); slug != "" {
			logPath := filepath.Join(resolveWorkspaceDir(root, slug), ".reviewer-ro.log")
			action := in.Command
			if action == "" {
				action = in.ToolName + "\t" + in.FilePath
			}
			if err := state.AppendLog(logPath, "WOULD-BLOCK\t"+head80(action)); err != nil {
				debugf(stderr, "reviewer-readonly log: %v", err)
			}
		}
	}
	recordHookGuard(h, "reviewer-readonly", drvreason.HookReviewerReadonlyObserved, lib.GuardObserved, lib.OutcomeObserved)
	return exitOK
}

// subagentOrientContext is the fixed discipline injected into a spawned
// devrites-* subagent. It is embedded verbatim from the same text
// devrites-subagent-orient.sh emits, so the two stay byte-identical for the
// parity oracle (the file is the single source both read).
//
//go:embed subagent_orient_context.txt
var subagentOrientContext string

// hookSubagentOrient adds DevRites instructions when a devrites-* agent starts.
// Spawned agents have fresh context and do not load the rite-* framework the
// same way as the main thread. The output matches devrites-subagent-orient.sh.
func hookSubagentOrient(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	agentType := h.SubagentAgentType(stdin)
	if !strings.HasPrefix(agentType, "devrites-") {
		return exitOK // Stay silent without a DevRites agent identity.
	}
	out, err := h.SubagentStartContext(strings.ReplaceAll(subagentOrientContext, "\r\n", "\n"))
	if err != nil {
		debugf(stderr, "subagent-orient: %v", err)
		return exitOK
	}
	// No trailing newline: matches the script's `node ... process.stdout.write`.
	fmt.Fprint(stdout, out)
	return exitOK
}

// head80 returns at most the first 80 bytes of s, mirroring the shell guards'
// `head -c 80` truncation of a logged command.
func head80(s string) string {
	if len(s) > 80 {
		return s[:80]
	}
	return s
}

// debugf writes diagnostics only when DEVRITES_DEBUG is set.
func debugf(stderr io.Writer, format string, args ...any) {
	if os.Getenv("DEVRITES_DEBUG") != "" {
		fmt.Fprintf(stderr, "devrites: "+format+"\n", args...)
	}
}

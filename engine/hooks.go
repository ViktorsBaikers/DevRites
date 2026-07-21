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
	"github.com/devrites/devrites/internal/orient"
	"github.com/devrites/devrites/internal/state"
)

// hookOrient emits the SessionStart orientation for the active feature. It is
// FAIL-OPEN and read-only: on a non-DevRites directory, an unreadable workspace,
// or nothing to say, it stays silent and exits 0 so a session is never wedged.
// The engine's inline shell guard handles a missing binary; this handles a
// present binary run outside a workspace.
func hookOrient(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	// The SessionStart payload is drained but unused — orientation is a pure read
	// of the workspace files, not a function of the harness's stdin.
	_, _ = io.Copy(io.Discard, stdin)

	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return exitOK // not a DevRites workspace — silent no-op
	}
	text, has, err := orient.Digest(root)
	if err != nil {
		debugf(stderr, "orient: %v", err)
		return exitOK
	}
	if !has {
		// No active feature: on the first-ever such session, orient once with a
		// repo-classified starting nudge instead of staying silent.
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

// hookStopGate refuses to end a turn at a provably inconsistent rest point. It
// mirrors devrites-stop-gate.sh: OBSERVE by default, blocking only when
// DEVRITES_STOP_GATE=enforce; loop-guarded by the harness's stop_hook_active so
// it can never wedge the session; fully fail-open.
func hookStopGate(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	in := h.ParseStopInput(stdin)
	if in.StopHookActive {
		return exitOK // already re-entered this stop cycle — let it stop
	}
	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return exitOK
	}
	res, err := gate.StopGate(root)
	if err != nil || !res.Blocked {
		return exitOK
	}
	if !hookEnforce("DEVRITES_STOP_GATE") {
		// Observe mode: never block. Record a would-block to the feature's
		// append-only log (mirroring devrites-stop-gate.sh) so the invariant is
		// visible without gating; the append is O_APPEND-atomic across the
		// concurrent processes DevRites spawns.
		logPath := filepath.Join(resolveWorkspaceDir(root, res.Slug), ".stop-gate.log")
		if err := state.AppendLog(logPath, "WOULD-BLOCK\t"+res.Reason); err != nil {
			debugf(stderr, "stop-gate log: %v", err)
		}
		return exitOK
	}
	out, err := h.StopBlock(res.Reason)
	if err != nil {
		debugf(stderr, "stop-gate: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	return exitOK
}

// allowReason is the auto-approval message the allow hook emits.
const allowReason = "DevRites read-only orientation/gate command — auto-approved by the devrites-allow hook"

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

// hookAllow auto-approves the read-only devrites orientation/gate subcommands so
// they stop triggering a permission prompt on every skill run. FAIL-OPEN: it only
// ever EMITS an allow for a structurally exact command shape; on any doubt it
// stays silent, leaving the normal permission flow untouched.
func hookAllow(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	// Fast path: a command not mentioning devrites at all is never ours.
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
const reviewerReadonlyDenyReason = "DevRites: reviewers are read-only. This Bash command can mutate or exfiltrate; inspect with Read/Grep/Glob and return findings — do not modify the tree or reach the network. (devrites-reviewer-readonly)"

// reviewerMutateRe matches a Bash command that could mutate the tree or reach the
// network — the deny-list ported verbatim from devrites-reviewer-readonly.sh. A
// fresh-context reviewer reads untrusted source, so a silent write/exfil path is
// a prompt-injection surface (standards/security.md). pack-scan-ignore: this is
// the hook's own defensive deny-list, not a payload.
//
// The pattern is byte-identical to the bash `grep -E`. ONE deliberate divergence:
// this matches against the WHOLE command, while the bash hook's
// `read -r tool cmd agent_type` truncates the command at its first newline and so
// only ever scans line 1. A reviewer running a multi-line Bash whose later lines
// mutate (e.g. `cat x\nsed -i …`) slips past the bash hook but is denied here —
// intentional hardening of a latent bash bug, not a parity regression. (RE2's
// `[[:space:]]` already matches `\n`, so no `(?m)` flag is needed for the `-i$`
// alternative to fire at an interior line break.)
var reviewerMutateRe = regexp.MustCompile(`>>|[^0-9 ]>[^>&]|[[:space:]]-i([[:space:]]|$)|sed[[:space:]]+-i|\brm\b|\bmv\b|\btee\b|\btruncate\b|\bdd\b|chmod|chown|git[[:space:]]+(add|commit|push|reset|checkout|rm|mv|stash|tag|apply)|npm[[:space:]]+(install|i|publish|run)|pnpm[[:space:]]+(install|add)|yarn[[:space:]]+add|pip[[:space:]]+install|curl[[:space:]].*[[:space:]]-o\b|\bwget\b|\bscp\b|\bssh\b|\bnc\b`)

// hookReviewerReadonly keeps the read-only reviewer subagents read-only: it denies
// a Bash command that could mutate the tree or exfiltrate. It mirrors
// devrites-reviewer-readonly.sh: OBSERVE by default (logs a would-block, allows),
// blocking only when DEVRITES_REVIEWER_RO=enforce; fully fail-open.
func hookReviewerReadonly(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return exitOK
	}
	// Fast path: a payload with no tool call is not ours — mirrors the script's
	// `case "$input" in *'"tool_name"'*`.
	if !bytes.Contains(data, []byte(`"tool_name"`)) {
		return exitOK
	}
	in := h.ParsePreToolInput(bytes.NewReader(data))

	// When the surface guarantees subagent identity, scope to devrites reviewers:
	// the slice-wright is a sanctioned writer and the main thread / other agents
	// are not our concern.
	if os.Getenv("DEVRITES_REVIEWER_AGENT_REQUIRED") == "1" {
		switch {
		case in.AgentType == "" || in.AgentType == "devrites-slice-wright":
			return exitOK
		case !strings.HasPrefix(in.AgentType, "devrites-"):
			return exitOK
		}
	}
	if in.ToolName != "Bash" || in.Command == "" {
		return exitOK
	}
	if !reviewerMutateRe.MatchString(in.Command) {
		return exitOK
	}

	if hookEnforce("DEVRITES_REVIEWER_RO") {
		out, err := h.PreToolDeny(reviewerReadonlyDenyReason)
		if err != nil {
			debugf(stderr, "reviewer-readonly: %v", err)
			return exitOK
		}
		fmt.Fprintln(stdout, out)
		return exitOK
	}

	// Observe mode: record the would-block to the active feature's append-only log,
	// mirroring the stop-gate's log convention, then allow. The append is
	// O_APPEND-atomic across the concurrent processes DevRites spawns.
	if root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT")); err == nil {
		if slug, _ := orient.ActiveSlug(root); slug != "" {
			logPath := filepath.Join(resolveWorkspaceDir(root, slug), ".reviewer-ro.log")
			if err := state.AppendLog(logPath, "WOULD-BLOCK\t"+head80(in.Command)); err != nil {
				debugf(stderr, "reviewer-readonly log: %v", err)
			}
		}
	}
	return exitOK
}

// subagentOrientContext is the fixed discipline injected into a spawned
// devrites-* subagent. It is embedded verbatim from the same text
// devrites-subagent-orient.sh emits, so the two stay byte-identical for the
// parity oracle (the file is the single source both read).
//
//go:embed subagent_orient_context.txt
var subagentOrientContext string

// hookSubagentOrient injects the DevRites discipline into every devrites-* subagent
// at spawn: a spawned subagent starts in a fresh context and neither auto-loads the
// rite-* framework nor discovers skills the way the main thread does. It mirrors
// devrites-subagent-orient.sh — fire only for devrites-* agents, emit a fixed
// context envelope, stay silent + fail-open otherwise.
func hookSubagentOrient(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	agentType := h.SubagentAgentType(stdin)
	if !strings.HasPrefix(agentType, "devrites-") {
		return exitOK // not a DevRites subagent (or no identity) — silent no-op
	}
	out, err := h.SubagentStartContext(strings.ReplaceAll(subagentOrientContext, "\r\n", "\n"))
	if err != nil {
		debugf(stderr, "subagent-orient: %v", err)
		return exitOK
	}
	// No trailing newline — matches the script's `node ... process.stdout.write`.
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

// debugf writes an operational diagnostic only when DEVRITES_DEBUG is set, so a
// fail-open hook can surface WHY it stayed silent without ever polluting normal
// hook stdout/stderr.
func debugf(stderr io.Writer, format string, args ...any) {
	if os.Getenv("DEVRITES_DEBUG") != "" {
		fmt.Fprintf(stderr, "devrites: "+format+"\n", args...)
	}
}

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/notes"
	"github.com/devrites/devrites/internal/overhaul"
	"github.com/devrites/devrites/internal/parallel"
)

const (
	checkUsage = `usage: devrites-engine check <candidate|readiness|preconditions|seal|path-disjoint|task-graph|affected|slice|diff-scope|skill-trust|indexes|regression|continuation|drift|windows|dup|anchors|perturb> ...

  check candidate <slug>
  check readiness <slug>
  check readiness --emit-binding <slug>
  check preconditions <slug> [--slice SLICE-###] [--timeout N]
  check seal <slug>
  check path-disjoint [--root <dir>] [<json-file>|-]
  check task-graph <slug> [--strict]
  check affected <path...>
  check slice <slug> <SLICE-ID>
  check diff-scope <slug> --allow <csv> | --allow-file <path> [--worktree|--staged|--base <ref>] [--allow-empty] [--baseline] [--verbose] [--cwd <dir>]
  check diff-scope <slug> --snapshot [--worktree|--staged|--base <ref>] [--cwd <dir>]
  check perturb <slug> --patch <file> --expect <test-name> --gate <G-ID> [--timeout N] [--worktree <path> --slice <SLICE-ID>]
  check skill-trust <path>
  check indexes [--root <dir>]
  check regression <slug> [--update]
  check continuation [slug]
  check drift <slug> [--record]
  check windows <slug> [--worktree|--staged|--base <ref>|--since-dispatch <wave>] [--cwd <dir>]
  check dup [slug] [--all|--worktree|--staged|--base <ref>]
  check anchors <slug> <return-file> [--skills-root <dir>]
`

	stateUsage = `usage: devrites-engine state <resolve|merge-manifest|apply-bundle|set|append|close> ...

  state resolve <qid> "<answer>"
  state resolve --drop <qid> ["<reason>"]
  state resolve --batch <file>
  state resolve --human <qid> "<answer>"
  state resolve --human --drop <qid> ["<reason>"]
  state resolve --human --batch <file>
  state merge-manifest <slug> [<predecessor>...]
  state apply-bundle <slug> <bundle.json>... [--dry-run]
  state set <slug> <key>=<value>... [--unset <key>]...
  state append <slug> <ledger.md> --heading "<text>" --body-file <path|->
  state close <slug>
`

	observeUsage = `usage: devrites-engine observe summary <slug> | observe slice <slug> <SLICE-ID>

  observe summary [slug]
  observe slice <slug> <SLICE-ID>
`

	orientUsage  = "usage: devrites-engine orient [slug]"
	nextUsage    = "usage: devrites-engine next [slug]"
	handoffUsage = "usage: devrites-engine handoff [slug] [--resume]"
	contextUsage = `usage: devrites-engine context [slug] (--phase <p> | --skill <name>) [--role <r>] [--slice <SLICE-###>] [--recheck <inventory>] [--trigger a,b] [--inline-contract] [--whole] [--skills-root <dir>] [--out <path>]`
	metricsUsage = `usage: devrites-engine metrics <record|summary> ...

  metrics record <slug> --phase <p> --event <e> [--role r] [--bytes n] [--note s]
  metrics summary [slug] [--by-slice]
`
	dispatchUsage  = `usage: devrites-engine dispatch <slug> <open|start|seal|return|status|abandon|attribute> [--phase p] [--wave w] [--role r] [--handle h] [--reason s] [--slice SLICE-###] [--reslice-review <DRIFT-ID>] [--contract <file>] [--packet <file>] [--outcome findings|no-findings|gap] [--result <file>] [--all] [--claim-session <id> --claim <path>... [--claim-ttl <minutes>]] [--release] [--path <path>...]`
	claimUsageText = `usage: devrites-engine claim <add|release|list|check> ...

  claim add --session <id> [--ttl <minutes>] [--reason <s>] <path>...
  claim release --session <id> --id <claim-id>
  claim list [--session <id>] [--all]
  claim check [--session <id>] <path>...
`
	diffScopeUsage = `usage: devrites-engine check diff-scope <slug> --allow <csv> | --allow-file <path> [--worktree|--staged|--base <ref>] [--allow-empty] [--baseline] [--verbose] [--cwd <dir>]
       devrites-engine check diff-scope <slug> --snapshot [--worktree|--staged|--base <ref>] [--cwd <dir>]`
	migrateUsage = "usage: devrites-engine migrate <slug> [--dry-run] [--answer id=choice]"
	// scanCmdUsage is help text for `secret-scan`. The identifier must not
	// contain "secret": gosec G101 treats secret* string consts as credentials.
	scanCmdUsage        = "usage: devrites-engine secret-scan [--staged] [--stdin] [slug]"
	openVisualUsage     = "usage: devrites-engine open-visual <path-or-name> [--slug <slug>] [--no-open]"
	versionUsage        = "usage: devrites-engine version"
	candidateUsage      = "usage: devrites-engine check candidate <slug>"
	readinessUsage      = "usage: devrites-engine check readiness <slug>\n       devrites-engine check readiness --emit-binding <slug>"
	preconditionsUsage  = "usage: devrites-engine check preconditions <slug> [--slice SLICE-###] [--timeout N]"
	sealUsage           = "usage: devrites-engine check seal <slug>"
	taskGraphUsage      = "usage: devrites-engine check task-graph <slug> [--strict]"
	sliceCheckUsage     = "usage: devrites-engine check slice <slug> <SLICE-ID>"
	skillTrustUsage     = "usage: devrites-engine check skill-trust <path>"
	indexesUsage        = "usage: devrites-engine check indexes [--root <dir>]"
	regressionUsageLine = "usage: devrites-engine check regression <slug> [--update]"
	continuationUsage   = "usage: devrites-engine check continuation [slug]"
	driftUsageLine      = "usage: devrites-engine check drift <slug> [--record]"
	detectUsageLine     = "usage: devrites-engine detect commands [--root <dir>] [--json]"
	windowsUsageLine    = "usage: devrites-engine check windows <slug> [--worktree|--staged|--base <ref>|--since-dispatch <wave>] [--cwd <dir>]"
	dupUsageLine        = "usage: devrites-engine check dup [slug] [--all|--worktree|--staged|--base <ref>] [--min-lines n] [--threshold f] [--ignore-file <path>] [--limit n] [--exclude <csv>] [--phase p] [--cwd <dir>]"
	resolveUsage        = `usage: devrites-engine state resolve <qid> "<answer>"  |  state resolve --drop <qid> ["<reason>"]  |  state resolve --batch <file>  |  state resolve --human <qid> "<answer>"  |  state resolve --human --drop <qid> ["<reason>"]  |  state resolve --human --batch <file>`
	mergeManifestUsage  = "usage: devrites-engine state merge-manifest <slug> [<predecessor>...]"
	applyBundleUsage    = "usage: devrites-engine state apply-bundle <slug> <bundle.json>... [--dry-run]"
	closeUsage          = "usage: devrites-engine state close <slug>"
	stateSetUsage       = "usage: devrites-engine state set <slug> <key>=<value>... [--unset <key>]..."
	stateAppendUsage    = `usage: devrites-engine state append <slug> <ledger.md> --heading "<text>" --body-file <path|->`
	observeSliceUsage   = "usage: devrites-engine observe slice <slug> <SLICE-ID>"
	observeSummaryUsage = "usage: devrites-engine observe summary [slug]"
	gatesFamilyUsage    = `usage: devrites-engine gates <scaffold|status|run|reverify|lint|precheck|attest|abandon> <slug> ...

  gates scaffold <slug>            Create gates.md from spec.md AC ids (only when absent)
  gates status <slug>              Reduce the ledger without executing; exit 3 unless ALL MET
  gates run <slug> [--timeout N] [--transcripts]  Execute unmet runnable gates and record evidence
  gates reverify <slug> [--timeout N] [--transcripts]  Re-execute every runnable gate, demoting stale passes
                                   (--transcripts: one redacted, candidate-bound file per gate in references/gates/)
  gates lint <slug> [--strict]     Audit oracle quality without executing
  gates precheck <slug> --worktree <path> [--slice <SLICE-ID>]  Run RUN: writer gates + check windows in <path>; --slice skips gates owned only by other unbuilt slices; no ledger, EVIDENCE or transcript write (check windows logs its metric); never evidence
  gates attest <slug> <id> <note>  Record human evidence on a manual gate
  gates abandon <slug> <id> <why>  Record a terminal ABANDON handoff on a gate
`
)

func isHelpFlag(arg string) bool {
	switch arg {
	case "-h", "-help", "--help":
		return true
	default:
		return false
	}
}

func isHelpToken(arg string) bool {
	return isHelpFlag(arg) || arg == "help"
}

// takesSubcommand reports whether command's first argument names a subcommand.
// A subcommand is consumed by dispatch, so a help flag directly after it is
// still in flag position rather than in a value position.
func takesSubcommand(command string) bool {
	switch command {
	case "check", "state", "observe", "parallel", "gates", "claim", "note", "metrics", "detect", "overhaul", "candidate":
		return true
	default:
		return false
	}
}

// valueFlags lists, per command, the flags that consume the next argument as a
// value. A command listed here has known flags, so its help flag is honoured
// anywhere except as the value of one of these flags.
var valueFlags = map[string]map[string]bool{
	"claim":     {"--session": true, "--reason": true, "--ttl": true, "--id": true},
	"dispatch":  {"--phase": true, "--wave": true, "--role": true, "--handle": true, "--reason": true, "--claim-session": true, "--claim": true, "--claim-ttl": true, "--path": true},
	"packet":    {"--wave": true, "--phase": true, "--role": true, "--slice": true, "--checks": true, "--focus": true},
	"candidate": {"--wave": true, "--slice": true, "--base": true},
	"recall":    {"--paths": true, "--limit": true},
}

// hasHelpFlag reports whether args (the arguments after the command word) are a
// help request. The command path is consumed by dispatch, so a help flag is
// honoured among the leading flags that follow it. The scan stops at the first
// argument that does not start with a dash, because from there the engine
// cannot tell a flag from a value or operand. Commands in valueFlags honour a
// help flag later, and never one that is the value of such a flag.
func hasHelpFlag(command string, args []string) bool {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") && takesSubcommand(command) {
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		if isHelpFlag(args[i]) {
			return true
		}
		known, ok := valueFlags[command]
		if ok && known[args[i]] {
			i++
		} else if !ok && !strings.HasPrefix(args[i], "-") {
			return false
		}
	}
	return false
}

func shouldPrintHelp(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if isHelpToken(args[0]) {
		return true
	}
	if hasHelpFlag(args[0], args[1:]) {
		return true
	}
	switch args[0] {
	case "check", "state", "observe", "parallel", "gates", "claim", "detect", "note", "metrics", "handoff", "context", "orient", "migrate", "overhaul", "next", "dispatch", "secret-scan", "open-visual", "recall", "packet", "candidate":
		return len(args) >= 2 && args[1] == "help"
	default:
		return false
	}
}

func firstNonHelp(args []string) string {
	for _, arg := range args {
		if isHelpToken(arg) || strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
}

func writeHelp(stdout io.Writer, text string) int {
	fmt.Fprintln(stdout, strings.TrimRight(text, "\n"))
	return exitOK
}

// commandHelp returns the usage text for a help request on a known command.
// ok is false when args are not a help request, or the command is unknown so
// the existing unknown-command path should run.
func commandHelp(args []string) (string, bool) {
	if !shouldPrintHelp(args) {
		return "", false
	}
	switch args[0] {
	case "-h", "-help", "--help", "help":
		return usage, true
	case "check":
		return helpCheck(args[1:]), true
	case "state":
		return helpState(args[1:]), true
	case "observe":
		return helpObserve(args[1:]), true
	case "orient":
		return orientUsage, true
	case "next":
		return nextUsage, true
	case "handoff":
		return handoffUsage, true
	case "recall":
		return lib.RecallUsage, true
	case "context":
		return contextUsage, true
	case "metrics":
		return metricsUsage, true
	case "dispatch":
		return dispatchUsage, true
	case "packet":
		return lib.PacketUsage, true
	case "candidate":
		return lib.CandidateFreezeUsage, true
	case "claim":
		return claimUsageText, true
	case "parallel":
		return parallel.CommandUsage(firstNonHelp(args[1:])), true
	case "gates":
		return gatesFamilyUsage, true
	case "note":
		return notes.Usage, true
	case "overhaul":
		return overhaul.Usage, true
	case "migrate":
		return migrateUsage, true
	case "secret-scan":
		return scanCmdUsage, true
	case "open-visual":
		return openVisualUsage, true
	case "detect":
		return detectUsageLine, true
	case "version", "--version":
		return versionUsage, true
	default:
		return "", false
	}
}

func helpCheck(rest []string) string {
	switch firstNonHelp(rest) {
	case "candidate":
		return candidateUsage
	case "readiness":
		return readinessUsage
	case "preconditions":
		return preconditionsUsage
	case "seal":
		return sealUsage
	case "path-disjoint":
		return parallel.CommandUsage("path-disjoint")
	case "task-graph":
		return taskGraphUsage
	case "slice":
		return sliceCheckUsage
	case "diff-scope":
		return diffScopeUsage
	case "skill-trust":
		return skillTrustUsage
	case "indexes":
		return indexesUsage
	case "regression":
		return regressionUsageLine
	case "continuation":
		return continuationUsage
	case "drift":
		return driftUsageLine
	case "windows":
		return windowsUsageLine
	case "dup":
		return dupUsageLine
	case "anchors":
		return lib.AnchorsUsage
	case "perturb":
		return lib.PerturbUsage
	default:
		return checkUsage
	}
}

func helpState(rest []string) string {
	switch firstNonHelp(rest) {
	case "resolve":
		return resolveUsage
	case "merge-manifest":
		return mergeManifestUsage
	case "apply-bundle":
		return applyBundleUsage
	case "set":
		return stateSetUsage
	case "append":
		return stateAppendUsage
	case "close":
		return closeUsage
	default:
		return stateUsage
	}
}

func helpObserve(rest []string) string {
	switch firstNonHelp(rest) {
	case "summary":
		return observeSummaryUsage
	case "slice":
		return observeSliceUsage
	default:
		return observeUsage
	}
}

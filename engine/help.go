package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/devrites/devrites/internal/overhaul"
	"github.com/devrites/devrites/internal/parallel"
)

const (
	checkUsage = `usage: devrites-engine check <candidate|readiness|seal|path-disjoint|task-graph|slice|diff-scope|skill-trust|indexes|regression|drift|windows|dup> ...

  check candidate <slug>
  check readiness <slug>
  check readiness --emit-binding <slug>
  check seal <slug>
  check path-disjoint [--root <dir>] [<json-file>|-]
  check task-graph <slug>
  check slice <slug> <SLICE-ID>
  check diff-scope <slug> --allow <csv> | --allow-file <path> [--worktree|--staged|--base <ref>]
  check skill-trust <path>
  check indexes [--root <dir>]
  check regression <slug> [--update]
  check drift <slug> [--record]
  check windows <slug> [--worktree|--staged|--base <ref>]
  check dup [slug] [--all|--worktree|--staged|--base <ref>]
`

	stateUsage = `usage: devrites-engine state <resolve|merge-manifest|close> ...

  state resolve <qid> "<answer>"
  state resolve --drop <qid> ["<reason>"]
  state resolve --batch <file>
  state merge-manifest <slug> [<predecessor>...]
  state close <slug>
`

	observeUsage = `usage: devrites-engine observe summary <slug> | observe slice <slug> <SLICE-ID>

  observe summary [slug]
  observe slice <slug> <SLICE-ID>
`

	orientUsage  = "usage: devrites-engine orient [slug]"
	nextUsage    = "usage: devrites-engine next [slug]"
	handoffUsage = "usage: devrites-engine handoff [slug]"
	contextUsage = `usage: devrites-engine context [slug] (--phase <p> | --skill <name>) [--role <r>] [--trigger a,b] [--skills-root <dir>] [--out <path>]`
	metricsUsage = `usage: devrites-engine metrics <record|summary> ...

  metrics record <slug> --phase <p> --event <e> [--role r] [--bytes n] [--note s]
  metrics summary [slug]
`
	dispatchUsage  = `usage: devrites-engine dispatch <slug> <open|start|seal|return|status|abandon> [--phase p] [--wave w] [--role r] [--handle h] [--reason s]`
	claimUsageText = `usage: devrites-engine claim <add|release|list|check> ...

  claim add --session <id> [--ttl <minutes>] [--reason <s>] <path>...
  claim release --session <id> --id <claim-id>
  claim list [--session <id>] [--all]
  claim check [--session <id>] <path>...
`
	diffScopeUsage = `usage: devrites-engine check diff-scope <slug> --allow <csv> | --allow-file <path> [--worktree|--staged|--base <ref>] [--allow-empty] [--cwd <dir>]`
	migrateUsage   = "usage: devrites-engine migrate <slug> [--dry-run] [--answer id=choice]"
	// scanCmdUsage is help text for `secret-scan`. The identifier must not
	// contain "secret": gosec G101 treats secret* string consts as credentials.
	scanCmdUsage        = "usage: devrites-engine secret-scan [--staged] [--stdin] [slug]"
	openVisualUsage     = "usage: devrites-engine open-visual <path-or-name> [--slug <slug>] [--no-open]"
	versionUsage        = "usage: devrites-engine version"
	candidateUsage      = "usage: devrites-engine check candidate <slug>"
	readinessUsage      = "usage: devrites-engine check readiness <slug>\n       devrites-engine check readiness --emit-binding <slug>"
	sealUsage           = "usage: devrites-engine check seal <slug>"
	taskGraphUsage      = "usage: devrites-engine check task-graph <slug>"
	sliceCheckUsage     = "usage: devrites-engine check slice <slug> <SLICE-ID>"
	skillTrustUsage     = "usage: devrites-engine check skill-trust <path>"
	indexesUsage        = "usage: devrites-engine check indexes [--root <dir>]"
	regressionUsageLine = "usage: devrites-engine check regression <slug> [--update]"
	driftUsageLine      = "usage: devrites-engine check drift <slug> [--record]"
	detectUsageLine     = "usage: devrites-engine detect commands [--root <dir>] [--json]"
	windowsUsageLine    = "usage: devrites-engine check windows <slug> [--worktree|--staged|--base <ref>] [--cwd <dir>]"
	dupUsageLine        = "usage: devrites-engine check dup [slug] [--all|--worktree|--staged|--base <ref>] [--min-lines n] [--threshold f] [--ignore-file <path>] [--limit n] [--exclude <csv>] [--phase p] [--cwd <dir>]"
	resolveUsage        = `usage: devrites-engine state resolve <qid> "<answer>"  |  state resolve --drop <qid> ["<reason>"]  |  state resolve --batch <file>`
	mergeManifestUsage  = "usage: devrites-engine state merge-manifest <slug> [<predecessor>...]"
	closeUsage          = "usage: devrites-engine state close <slug>"
	observeSliceUsage   = "usage: devrites-engine observe slice <slug> <SLICE-ID>"
	observeSummaryUsage = "usage: devrites-engine observe summary [slug]"
	gatesFamilyUsage    = `usage: devrites-engine gates <scaffold|status|run|reverify|lint|attest|abandon> <slug> ...

  gates scaffold <slug>            Create gates.md from spec.md AC ids (only when absent)
  gates status <slug>              Reduce the ledger without executing; exit 3 unless ALL MET
  gates run <slug> [--timeout N]   Execute unmet runnable gates and record evidence
  gates reverify <slug> [--timeout N]  Re-execute every runnable gate, demoting stale passes
  gates lint <slug> [--strict]     Audit oracle quality without executing
  gates attest <slug> <id> <note>  Record human evidence on a manual gate
  gates abandon <slug> <id> <why>  Record a terminal ABANDON handoff on a gate
`
	noteFamilyUsage = `usage: devrites-engine note <add|list|check|rm> <slug> ...

  note add <slug> <subject> <quote> <title> [body]
                                  Anchor a workspace note to verbatim code text
  note list <slug>                List notes.md entries with their current grade
  note check <slug> [--repair]    Regrade every anchor; --repair rewrites moved subjects
  note rm <slug> <NOTE-id>        Remove a note
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

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if isHelpFlag(arg) {
			return true
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
	if hasHelpFlag(args) {
		return true
	}
	switch args[0] {
	case "check", "state", "observe", "parallel", "gates", "claim", "detect", "note", "metrics", "handoff", "context", "orient", "migrate", "overhaul":
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
	case "context":
		return contextUsage, true
	case "metrics":
		return metricsUsage, true
	case "dispatch":
		return dispatchUsage, true
	case "claim":
		return claimUsageText, true
	case "parallel":
		return parallel.CommandUsage(firstNonHelp(args[1:])), true
	case "gates":
		return gatesFamilyUsage, true
	case "note":
		return noteFamilyUsage, true
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
	case "drift":
		return driftUsageLine
	case "windows":
		return windowsUsageLine
	case "dup":
		return dupUsageLine
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

// Command devrites is the DevRites control-plane engine: a single static,
// pure-Go binary that owns the deterministic workflow state (phases, gates,
// completeness) over a project's .devrites/ directory. It makes zero model or
// network calls — the in-session LLM remains the judgment data plane.
//
// Command dispatch is intentionally a small stdlib switch (no third-party deps)
// so the binary cross-compiles trivially with CGO_ENABLED=0; it can be swapped
// for Cobra later without changing the CLI contract.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/index"
	"github.com/devrites/devrites/internal/iohooks"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/version"
)

const usage = `devrites — DevRites control-plane engine

Usage:
  devrites-engine status <slug>            Print a feature's phase and per-section completeness
  devrites-engine reindex                  Rebuild the SQLite index from the .devrites files
  devrites-engine readiness <slug>         Gate: are the sections required to leave this phase complete?
  devrites-engine seal <slug>              Gate: is the feature complete enough to seal?
  devrites-engine spec-validate <dir|file> [--against <ledger-root>]  Lint the structured Requirement/Scenario grammar in a spec.md (--against cross-checks delta sections vs the capability ledger)
  devrites-engine spec-skeleton <dir|file> Check spec.md declares the six top-level spec sections
  devrites-engine check-acceptance <dir>   Grade spec.md's [ACn] criteria against seal.md
  devrites-engine footprint <sub> <slug>   Fan-out footprint: log|render|roster the dispatch log
  devrites-engine evidence-fresh [slug]    Gate: does the proof post-date every touched file?
  devrites-engine coverage [slug]          Render the AC → slice(s) → proven traceability matrix
  devrites-engine doubt-coverage <slug>    Did the build doubt the decisions it stood?
  devrites-engine budget [slug]            Lint each workspace file against its context-size ceiling
  devrites-engine preamble [slug]          Print the active feature's workspace-state orientation
  devrites-engine progress [slug]          Render the active feature's phase/slice/flow footer
  devrites-engine stuck <sub> <slug>       Unattended-build loop detector: log|check
  devrites-engine tick-afk <state.md>      Decrement the AFK slice budget in a state.md
  devrites-engine build-readiness [slug]   Gate: is the plan approved and the state build-ready?
  devrites-engine analyze [slug]           Cross-artifact spec↔tasks coverage/consistency gate
  devrites-engine mutation-gate [slug]     Advisory: detect the mutation runner and scope it
  devrites-engine test-integrity [slug]    Gate: no test deleted, skipped, or de-asserted
  devrites-engine review-integrity [slug]  Gate: no adversarial review axis is silent (zero findings, no justification)
  devrites-engine package-existence [slug] Gate: every new import is declared in a manifest
  devrites-engine reconcile <sub> [slug]   A1 gate: snapshot|check the wright's source writes
  devrites-engine resolve <qid> "<ans>"    Resolve an open question; keep state.md consistent
  devrites-engine close-out <slug>         Archive a shipped feature and clear ACTIVE
  devrites-engine archive-search "<nouns>" Surface shipped features whose spec.md overlaps the query (prior-art probe)
  devrites-engine ledger <sub> [...]       Capability ledger (living specs/): sync|diff|validate|list|show
  devrites-engine learnings <sub> [...]    Cross-feature learning ledger: add|list|top|mine|nudge
  devrites-engine conventions <sub> [...]  Project convention ledger: band|read|orient|promote|contradict
  devrites-engine extensions <sub>         Project extensions (.devrites/extensions/): list|validate|sync
  devrites-engine overrides <sub>          Reviewer-override linter (.devrites/overrides/): list|validate
  devrites-engine doctor                   Report the binary / pack / state-schema version triangle
  devrites-engine migrate                  Normalize .devrites workspaces to the current schema
  devrites-engine validate-pack [dir]      Lint the shipped pack's hook wiring + skill/agent frontmatter contracts + docs/command-map.md links
  devrites-engine harness-matrix [--check F] Render (or drift-check) the harness adapter-compliance matrix
  devrites-engine version                  Print the engine binary's version
  devrites-engine hook <name> --harness=H  Run hook <name> for harness H (claude|codex)

Hooks:
  hook allow              Auto-approve the read-only devrites orientation/gate subcommands
  hook orient             Emit SessionStart orientation for the active feature
  hook stop-gate          Refuse to end a turn at a provably inconsistent rest point
  hook reviewer-readonly  Deny a reviewer subagent's mutating/exfiltrating Bash command
  hook subagent-orient    Inject the DevRites discipline into a spawned devrites-* subagent
  hook cursor             Re-inject the active feature's cursor (UserPromptSubmit)
  hook statusline         One-line workspace HUD (settings.json statusLine)
  hook redwatch           Fail-on-red sentinel: set/clear .red after a test/build command
  hook a1-guard           Block the orchestrator editing source mid-build
  hook wright-scope       Fence the slice-wright to its touched-files.md
  hook source-cache-pre   Serve a cached page reading on a 304 revalidation (WebFetch)
  hook source-cache-post  Store a WebFetch result keyed on its origin validators
  hook refresh-indexes    Refresh the code-intelligence indexes (detached) on Stop

Exit codes:
  0  ok / gate passed
  2  usage error
  3  blocked — a gate pause or a version-skew refuse; resolve the reported
     gap and retry (HITL, never a crash)

Environment:
  DEVRITES_ROOT   Path to the .devrites directory. Defaults to the nearest
                  .devrites found walking up from the working directory.

Hook control plane:
  DEVRITES_HOOK_PROFILE   minimal | standard (default) | strict. minimal runs
                          orientation/approval hooks only; standard adds the
                          gates, guards and caches; strict additionally flips
                          every OBSERVE-default guard (a1-guard, stop-gate,
                          reviewer-readonly, wright-scope) to enforce at once.
  DEVRITES_DISABLED_HOOKS Comma-separated hook ids to force off, e.g.
                          "stop-gate,redwatch". A disabled hook is a clean
                          fail-open no-op.
`

// Exit codes shared across commands.
const (
	exitOK      = 0
	exitUsage   = 2
	exitBlocked = 3 // a gate blocked; HITL-resolvable, never a crash
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the testable entry point. It returns the process exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitUsage
	}
	switch args[0] {
	case "status":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("status", j, stdout, stderr, func(o, e io.Writer) int { return cmdStatus(rest, o, e) })
	case "reindex":
		return cmdReindex(args[1:], stdout, stderr)
	case "readiness":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("readiness", j, stdout, stderr, func(o, e io.Writer) int { return cmdGate(gate.Readiness, rest, o, e) })
	case "seal":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("seal", j, stdout, stderr, func(o, e io.Writer) int { return cmdGate(gate.Seal, rest, o, e) })
	case "spec-validate":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("spec-validate", j, stdout, stderr, func(o, e io.Writer) int { return cmdSpecValidate(rest, o, e) })
	case "spec-skeleton":
		return cmdSpecSkeleton(args[1:], stdout, stderr)
	case "check-acceptance":
		return cmdCheckAcceptance(args[1:], stdout, stderr)
	case "footprint":
		return cmdFootprint(args[1:], stdout, stderr)
	case "evidence-fresh":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("evidence-fresh", j, stdout, stderr, func(o, e io.Writer) int { return cmdEvidenceFresh(rest, o, e) })
	case "coverage":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("coverage", j, stdout, stderr, func(o, e io.Writer) int { return cmdCoverage(rest, o, e) })
	case "doubt-coverage":
		return cmdDoubtCoverage(args[1:], stdout, stderr)
	case "budget":
		return lib.Budget(resolveRootLenient(), args[1:], stdout, stderr)
	case "preamble":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("preamble", j, stdout, stderr, func(o, e io.Writer) int { return cmdPreamble(rest, o, e) })
	case "progress":
		return cmdProgress(args[1:], stdout, stderr)
	case "stuck":
		return cmdStuck(args[1:], stdout, stderr)
	case "tick-afk":
		return lib.TickAfk(args[1:], stdout, stderr)
	case "build-readiness":
		return lib.BuildReadiness(resolveRootLenient(), args[1:], stdout, stderr)
	case "analyze":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("analyze", j, stdout, stderr, func(o, e io.Writer) int { return lib.Analyze(resolveRootLenient(), rest, o, e) })
	case "mutation-gate":
		return lib.MutationGate(resolveRootLenient(), args[1:], stdout, stderr)
	case "test-integrity":
		return lib.TestIntegrity(resolveRootLenient(), args[1:], stdout, stderr)
	case "review-integrity":
		return lib.ReviewIntegrity(resolveRootLenient(), args[1:], stdout, stderr)
	case "package-existence":
		return lib.PackageExistence(resolveRootLenient(), args[1:], stdout, stderr)
	case "reconcile":
		return lib.Reconcile(resolveRootLenient(), args[1:], stdout, stderr)
	case "resolve":
		return lib.Resolve(resolveRootLenient(), args[1:], stdout, stderr)
	case "close-out":
		return lib.CloseOut(resolveRootLenient(), args[1:], stdout, stderr)
	case "archive-search":
		return lib.ArchiveSearch(resolveRootLenient(), args[1:], stdout, stderr)
	case "ledger":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("ledger", j, stdout, stderr, func(o, e io.Writer) int { return lib.Ledger(resolveRootLenient(), rest, o, e) })
	case "learnings":
		return lib.Learnings(resolveRootLenient(), args[1:], stdout, stderr)
	case "conventions":
		return lib.Conventions(args[1:], stdout, stderr)
	case "extensions":
		return lib.Extensions(resolveRootLenient(), args[1:], stdout, stderr)
	case "overrides":
		return lib.Overrides(resolveRootLenient(), args[1:], stdout, stderr)
	case "doctor":
		rest, j := extractFlag(args[1:], "--json")
		return jsonWrap("doctor", j, stdout, stderr, func(o, e io.Writer) int { return cmdDoctor(rest, o, e) })
	case "migrate":
		return cmdMigrate(args[1:], stdout, stderr)
	case "validate-pack":
		return cmdValidatePack(args[1:], stdout, stderr)
	case "harness-matrix":
		return cmdHarnessMatrix(args[1:], stdout, stderr)
	case "hook":
		return cmdHook(args[1:], stdin, stdout, stderr)
	case "version", "--version":
		fmt.Fprintln(stdout, version.Version)
		return exitOK
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return exitOK
	default:
		fmt.Fprintf(stderr, "devrites: unknown command %q\n\n%s", args[0], usage)
		return exitUsage
	}
}

func cmdStatus(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: devrites-engine status <slug>")
		return exitUsage
	}
	root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	report, err := index.Status(root, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	fmt.Fprint(stdout, report.Render())
	return exitOK
}

func cmdReindex(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: devrites-engine reindex")
		return exitUsage
	}
	root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	n, err := index.Reindex(root)
	if err != nil {
		fmt.Fprintf(stderr, "devrites: reindex failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "reindexed %d feature(s)\n", n)
	return exitOK
}

// cmdGate runs a completeness gate (readiness or seal). A complete phase passes
// (exit 0); an incomplete one prints a structured, actionable "missing X" and
// exits with the HITL pause code — never a crash.
func cmdGate(kind gate.Kind, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintf(stderr, "usage: devrites-engine %s <slug>\n", kind)
		return exitUsage
	}
	root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	result, err := gate.Check(kind, root, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	fmt.Fprint(stdout, result.Render())
	if result.Blocked {
		return exitBlocked
	}
	return exitOK
}

// cmdSpecValidate lints the structured Requirement/Scenario grammar in a spec.md.
// It takes the workspace/spec positional plus an optional --against <ledger-root>
// that turns on the delta cross-check. Exit codes: 0 valid/flat · 1 violation ·
// 2 usage · 5 missing spec.md.
func cmdSpecValidate(args []string, stdout, stderr io.Writer) int {
	arg, against := "", ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--against" && i+1 < len(args) {
			against = args[i+1]
			i++
			continue
		}
		if v, ok := cutFlag(a, "--against"); ok {
			against = v
			continue
		}
		if arg == "" {
			arg = a
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	return lib.SpecValidate(arg, against, cwd, stdout, stderr)
}

func cmdSpecSkeleton(args []string, stdout, stderr io.Writer) int {
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	return lib.SpecSkeleton(arg, cwd, stdout, stderr)
}

// cmdCheckAcceptance grades a workspace's [ACn] acceptance criteria against its
// seal.md (0 all proven · 1 gap · 2 usage · 5 missing spec.md/seal.md).
func cmdCheckAcceptance(args []string, stdout, stderr io.Writer) int {
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	return lib.CheckAcceptance(arg, stdout, stderr)
}

// cmdFootprint renders / logs / rosters the fan-out dispatch footprint for a
// feature. Workspace state lives under <root>/work/<slug>, with features/ as an
// alias for older workspaces.
func cmdFootprint(args []string, stdout, stderr io.Writer) int {
	return lib.Footprint(resolveRootLenient(), args, stdout, stderr)
}

// cmdEvidenceFresh runs the evidence-freshness gate (0 fresh · 3 stale · 5 no
// workspace/evidence).
func cmdEvidenceFresh(args []string, stdout, stderr io.Writer) int {
	return lib.EvidenceFresh(resolveRootLenient(), args, stdout, stderr)
}

// cmdCoverage renders the AC → slice → proven traceability matrix (0 rendered ·
// 2 no workspace/spec).
func cmdCoverage(args []string, stdout, stderr io.Writer) int {
	return lib.Coverage(resolveRootLenient(), args, stdout, stderr)
}

// cmdDoubtCoverage runs the doubt-coverage assessment (0 covered/n-a · 1 no doubt
// ran · 2 usage · 3 doubt: MISSING).
func cmdDoubtCoverage(args []string, stdout, stderr io.Writer) int {
	return lib.DoubtCoverage(resolveRootLenient(), args, stdout, stderr)
}

// cmdPreamble prints the active feature's workspace-state orientation (always
// exit 0).
func cmdPreamble(args []string, stdout, stderr io.Writer) int {
	return lib.Preamble(resolveRootLenient(), args, stdout, stderr)
}

// cmdProgress renders the active feature's phase/slice/flow footer (always exit 0,
// silent when no workspace).
func cmdProgress(args []string, stdout, stderr io.Writer) int {
	return lib.Progress(resolveRootLenient(), args, stdout, stderr)
}

// cmdStuck is the unattended-build loop detector: `stuck log|check <slug>` (0 not
// stuck · 2 bad args · 3 STUCK).
func cmdStuck(args []string, stdout, stderr io.Writer) int {
	return lib.Stuck(resolveRootLenient(), args, stdout, stderr)
}

func cmdHook(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: devrites-engine hook <name> --harness=claude|codex")
		return exitUsage
	}
	name := args[0]
	// Control plane: a hook stripped by the active DEVRITES_HOOK_PROFILE or named
	// in DEVRITES_DISABLED_HOOKS is a clean fail-open no-op, indistinguishable from
	// an absent binary. Consulted before any work, including refresh-indexes.
	if !hookActive(name) {
		return exitOK
	}
	// refresh-indexes carries its own positional flags (--worker/--force ROOT) and
	// needs no harness, so it is dispatched before harness parsing.
	if name == "refresh-indexes" {
		return iohooks.RefreshIndexes(args[1:], stdin, stdout, stderr)
	}
	h, err := harness.Parse(harnessFlag(args[1:]))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	switch name {
	case "allow":
		return hookAllow(h, stdin, stdout, stderr)
	case "orient":
		return hookOrient(h, stdin, stdout, stderr)
	case "stop-gate":
		return hookStopGate(h, stdin, stdout, stderr)
	case "reviewer-readonly":
		return hookReviewerReadonly(h, stdin, stdout, stderr)
	case "subagent-orient":
		return hookSubagentOrient(h, stdin, stdout, stderr)
	case "cursor":
		return hookCursor(stdin, stdout, stderr)
	case "statusline":
		return hookStatusline(stdin, stdout, stderr)
	case "redwatch":
		return hookRedwatch(h, stdin, stdout, stderr)
	case "a1-guard":
		return hookA1Guard(h, stdin, stdout, stderr)
	case "wright-scope":
		return hookWrightScope(h, stdin, stdout, stderr)
	case "source-cache-pre":
		return iohooks.SourceCachePre(stdin, stdout, stderr)
	case "source-cache-post":
		return iohooks.SourceCachePost(stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "devrites: unknown hook %q\n", name)
		return exitUsage
	}
}

// harnessFlag extracts --harness=VALUE or --harness VALUE from args, returning
// "" if absent (which Parse rejects with a clear message).
func harnessFlag(args []string) string {
	const pfx = "--harness"
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == pfx && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := cutFlag(a, pfx); ok {
			return v
		}
	}
	return ""
}

func cutFlag(arg, name string) (string, bool) {
	if len(arg) > len(name)+1 && arg[:len(name)] == name && arg[len(name)] == '=' {
		return arg[len(name)+1:], true
	}
	return "", false
}

// extractFlag removes a bare boolean flag (e.g. --json) from args, returning the
// remaining args and whether the flag was present.
func extractFlag(args []string, flag string) ([]string, bool) {
	out := make([]string, 0, len(args))
	found := false
	for _, a := range args {
		if a == flag {
			found = true
			continue
		}
		out = append(out, a)
	}
	return out, found
}

// jsonWrap runs a command thunk, wrapping its captured output in the machine-
// readable envelope when jsonMode is set and passing the real writers through
// otherwise. It is applied only to the AFK-parsed read commands (see the
// --json set in run()); other commands never accept --json.
func jsonWrap(command string, jsonMode bool, stdout, stderr io.Writer, thunk func(o, e io.Writer) int) int {
	if !jsonMode {
		return thunk(stdout, stderr)
	}
	var out, errb bytes.Buffer
	code := thunk(&out, &errb)
	lib.WriteEnvelope(stdout, lib.NewEnvelope(command, code, out.String(), errb.String()))
	return code
}

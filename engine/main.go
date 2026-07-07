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
	"fmt"
	"io"
	"os"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/index"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/state"
)

const usage = `devrites — DevRites control-plane engine

Usage:
  devrites status <slug>            Print a feature's phase and per-section completeness
  devrites reindex                  Rebuild the SQLite index from the .devrites files
  devrites readiness <slug>         Gate: are the sections required to leave this phase complete?
  devrites seal <slug>              Gate: is the feature complete enough to seal?
  devrites spec-validate <dir|file> Lint the structured Requirement/Scenario grammar in a spec.md
  devrites check-acceptance <dir>   Grade spec.md's [ACn] criteria against seal.md
  devrites footprint <sub> <slug>   Fan-out footprint: log|render|roster the dispatch log
  devrites evidence-fresh [slug]    Gate: does the proof post-date every touched file?
  devrites coverage [slug]          Render the AC → slice(s) → proven traceability matrix
  devrites doubt-coverage <slug>    Did the build doubt the decisions it stood?
  devrites doctor                   Report the binary / pack / state-schema version triangle
  devrites migrate                  Upgrade an old-layout .devrites workspace to the current schema
  devrites hook <name> --harness=H  Run hook <name> for harness H (claude|codex)

Hooks:
  hook orient             Emit SessionStart orientation for the active feature
  hook stop-gate          Refuse to end a turn at a provably inconsistent rest point
  hook reviewer-readonly  Deny a reviewer subagent's mutating/exfiltrating Bash command
  hook subagent-orient    Inject the DevRites discipline into a spawned devrites-* subagent

Exit codes:
  0  ok / gate passed
  2  usage error
  3  blocked — a gate pause or a version-skew refuse; resolve the reported
     gap and retry (HITL, never a crash)

Environment:
  DEVRITES_ROOT   Path to the .devrites directory. Defaults to the nearest
                  .devrites found walking up from the working directory.
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
		return cmdStatus(args[1:], stdout, stderr)
	case "reindex":
		return cmdReindex(args[1:], stdout, stderr)
	case "readiness":
		return cmdGate(gate.Readiness, args[1:], stdout, stderr)
	case "seal":
		return cmdGate(gate.Seal, args[1:], stdout, stderr)
	case "spec-validate":
		return cmdSpecValidate(args[1:], stdout, stderr)
	case "check-acceptance":
		return cmdCheckAcceptance(args[1:], stdout, stderr)
	case "footprint":
		return cmdFootprint(args[1:], stdout, stderr)
	case "evidence-fresh":
		return cmdEvidenceFresh(args[1:], stdout, stderr)
	case "coverage":
		return cmdCoverage(args[1:], stdout, stderr)
	case "doubt-coverage":
		return cmdDoubtCoverage(args[1:], stdout, stderr)
	case "doctor":
		return cmdDoctor(args[1:], stdout, stderr)
	case "migrate":
		return cmdMigrate(args[1:], stdout, stderr)
	case "hook":
		return cmdHook(args[1:], stdin, stdout, stderr)
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
		fmt.Fprintln(stderr, "usage: devrites status <slug>")
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
		fmt.Fprintln(stderr, "usage: devrites reindex")
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
		fmt.Fprintf(stderr, "usage: devrites %s <slug>\n", kind)
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
// Like the script, it reads only the first positional arg and returns the port's
// own exit codes (0 valid/flat · 1 violation · 2 usage · 5 missing spec.md).
func cmdSpecValidate(args []string, stdout, stderr io.Writer) int {
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	return lib.SpecValidate(arg, cwd, stdout, stderr)
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
// feature. Like the script it is strictly CWD-relative (its state lives at
// .devrites/work/<slug>/footprint.log), so it passes the working directory as
// the base rather than resolving a DEVRITES_ROOT.
func cmdFootprint(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	return lib.Footprint(cwd, args, stdout, stderr)
}

// cmdEvidenceFresh runs the evidence-freshness gate (0 fresh · 3 stale · 5 no
// workspace/evidence). CWD-relative, matching the script.
func cmdEvidenceFresh(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	return lib.EvidenceFresh(cwd, args, stdout, stderr)
}

// cmdCoverage renders the AC → slice → proven traceability matrix (0 rendered ·
// 2 no workspace/spec). CWD-relative, matching the script.
func cmdCoverage(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	return lib.Coverage(cwd, args, stdout, stderr)
}

// cmdDoubtCoverage runs the doubt-coverage assessment (0 covered/n-a · 1 no doubt
// ran · 2 usage · 3 doubt: MISSING). CWD-relative, matching the script.
func cmdDoubtCoverage(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	return lib.DoubtCoverage(cwd, args, stdout, stderr)
}

func cmdHook(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: devrites hook <name> --harness=claude|codex")
		return exitUsage
	}
	name := args[0]
	h, err := harness.Parse(harnessFlag(args[1:]))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	switch name {
	case "orient":
		return hookOrient(h, stdin, stdout, stderr)
	case "stop-gate":
		return hookStopGate(h, stdin, stdout, stderr)
	case "reviewer-readonly":
		return hookReviewerReadonly(h, stdin, stdout, stderr)
	case "subagent-orient":
		return hookSubagentOrient(h, stdin, stdout, stderr)
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

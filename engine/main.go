// Command devrites-engine runs the deterministic DevRites control plane without
// calling a model or the network. Native host agents own semantic judgment.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/install"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/notes"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/version"
)

const usage = `devrites: DevRites control-plane engine

Usage:
  devrites-engine install [flags]          Install DevRites skills, agents, and host configuration
  devrites-engine update [flags]           Update an existing DevRites install in place
  devrites-engine uninstall [flags]        Remove a DevRites install, preserving runtime state
  devrites-engine check candidate <slug>   Validate and hash the closed project candidate
  devrites-engine check readiness <slug>   Check required files, tasks.md graph, and Build-input binding
  devrites-engine check readiness --emit-binding <slug>  Emit the stable Build-input binding for Vet
  devrites-engine check seal <slug>        Recheck files, tasks.md graph, Build-input binding, and evidence freshness
  devrites-engine check path-disjoint [--root <dir>] [<json-file>|-]
                                         Verify slice path sets are pairwise disjoint
  devrites-engine check task-graph <slug>  Validate tasks.md slice dependency graph
  devrites-engine check slice <slug> <SLICE-ID>  Pre-dispatch lint of one slice's wright contract
  devrites-engine check diff-scope <slug> --allow <csv>|--allow-file <path> [--worktree|--staged|--base <ref>]
                                         Verify the changed-path set stays inside the declared allowlist
  devrites-engine check skill-trust <path> Scan one skill/agent Markdown for trust violations
  devrites-engine check regression <slug> [--update]
                                         Compare workspace progress against the recorded baseline; --update ratchets it
  devrites-engine check drift <slug> [--record]
                                         Attribute readiness-input changes to the exact artifact since the Vet baseline
  devrites-engine check windows <slug> [--worktree|--staged|--base <ref>]
                                         Fail on deferral markers the change adds without a windows.md waiver
  devrites-engine check dup [slug] [--all|--worktree|--staged|--base <ref>]
                                         Report near-duplicate code clusters that survive renaming (advisory)
  devrites-engine observe summary <slug>   Emit sanitized JSON workspace summary
  devrites-engine orient <slug>            Alias for observe summary
  devrites-engine next [slug]              Print the minimal remaining lifecycle path with advisory skips
  devrites-engine handoff [slug]           Emit the deterministic resume record (cursor, blocking gates, ledger, dead ends)
  devrites-engine context <slug> --phase <p> [--role <r>] [--trigger a,b]
                                         Emit one deduplicated read-set bundle for a phase or dispatch role
  devrites-engine metrics record <slug> --phase <p> --event <e> [--role r] [--bytes n] [--note s]
  devrites-engine metrics summary [slug]   Summarize the per-feature metrics.jsonl event ledger
  devrites-engine dispatch <slug> <sub>    Launch-wave barrier for parallel dispatch: open/start/seal/return/status/abandon
  devrites-engine claim <add|release|list|check>   Advisory session-scoped file claims for same-tree concurrent sessions (.devrites/claims.jsonl)
  devrites-engine note <subcommand> <slug>         Anchored workspace notes bound to verbatim code quotes; drift regrades them (notes.md)
  devrites-engine observe slice <slug> <SLICE-ID>  Print one SLICE-### section of tasks.md
  devrites-engine check indexes [--root <dir>]  Report manifest and code-index presence as JSON
  devrites-engine detect commands [--root <dir>] [--json]
                                         Resolve test/lint/vet/build commands from the repository's own wiring
  devrites-engine parallel <subcommand>   Deterministic parallel worktree lease/create/integrate/cleanup
  devrites-engine gates <subcommand> <slug>  Machine-checked gates.md ledger: scaffold/status/run/reverify/lint/attest/abandon
  devrites-engine state resolve <qid> "<ans>"  Resolve an open question and update state atomically
  devrites-engine state merge-manifest <slug> [pred...]  Fold the recorded predecessor chain's manifests into the release candidate manifest
  devrites-engine state close <slug>       Archive a shipped feature and clear ACTIVE
  devrites-engine migrate <slug> [--dry-run]  Normalize a pre-v5 workspace to the current schema
  devrites-engine secret-scan [--staged] [--stdin] [slug]  Scan exact staged blobs, stdin, or touched files; HIGH blocks
  devrites-engine open-visual <path-or-name> [--slug <slug>] [--no-open]
                                         Resolve a visual HTML file, optionally open it locally, print agent paths
  devrites-engine version                  Print the engine binary's version
Exit codes:
  0  ok / gate passed
  2  usage error
  3  blocked: a deterministic gate paused; resolve the reported gap and retry
     (HITL, never a crash)

Environment:
  DEVRITES_ROOT   Path to the project root or .devrites directory. Defaults to
                  the nearest .devrites found walking up from the working directory.
  DEVRITES_WORKSPACE  Explicit feature workspace path for CI/agents; overrides
                  .devrites/ACTIVE when a command defaults to the active feature.
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
	if text, ok := commandHelp(args); ok {
		return writeHelp(stdout, text)
	}
	root, rootExit, err := resolveRootFor(args[0], args[1:])
	if err != nil {
		fmt.Fprintf(stderr, "devrites: root selection: %v\n", err)
		return rootExit
	}
	switch args[0] {
	case "install":
		return install.Run(args[1:], stdout, stderr, install.ModeInstall)
	case "update":
		return install.Run(args[1:], stdout, stderr, install.ModeUpdate)
	case "uninstall":
		return install.Run(args[1:], stdout, stderr, install.ModeUninstall)
	case "check":
		return cmdCheck(root, args[1:], stdin, stdout, stderr)
	case "parallel":
		return parallel.Run("parallel", args[1:], stdin, stdout, stderr)
	case "gates":
		return acceptance.Run(root, args[1:], stdout, stderr)
	case "observe":
		return cmdObserve(root, args[1:], stdout, stderr)
	case "orient":
		return cmdOrient(root, args[1:], stdout, stderr)
	case "next":
		return lib.RunNext(root, args[1:], stdout, stderr)
	case "handoff":
		return lib.RunHandoff(root, args[1:], stdout, stderr)
	case "context":
		return lib.RunContext(root, args[1:], stdout, stderr)
	case "metrics":
		return lib.RunMetrics(root, args[1:], stdout, stderr)
	case "dispatch":
		return lib.RunDispatch(root, args[1:], stdout, stderr)
	case "claim":
		return lib.RunClaim(root, args[1:], stdout, stderr)
	case "note":
		return notes.Run(root, args[1:], stdout, stderr)
	case "state":
		return cmdState(root, args[1:], stdout, stderr)
	case "migrate":
		return lib.Migrate(root, args[1:], stdout, stderr)
	case "secret-scan":
		return lib.SecretScan(root, args[1:], stdin, stdout, stderr)
	case "open-visual":
		return lib.OpenVisual(root, args[1:], stdout, stderr)
	case "detect":
		return lib.RunDetectCommands(root, args[1:], stdout, stderr)
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

func cmdCheck(root string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, checkUsage)
		return exitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "candidate":
		return cmdCandidate(root, rest, stdout, stderr)
	case "readiness", "seal":
		return cmdGate(root, gate.Kind(sub), rest, stdout, stderr)
	case "path-disjoint":
		return parallel.Run("path-disjoint", rest, stdin, stdout, stderr)
	case "task-graph":
		return cmdTaskGraph(root, rest, stdout, stderr)
	case "slice":
		return lib.RunCheckSlice(root, rest, stdout, stderr)
	case "diff-scope":
		return lib.RunCheckDiffScope(root, rest, stdout, stderr)
	case "skill-trust":
		return cmdSkillTrust(rest, stdout, stderr)
	case "indexes":
		return lib.RunEnvironmentCheck(root, rest, stdout, stderr)
	case "regression":
		return lib.RunCheckRegression(root, rest, stdout, stderr)
	case "windows":
		return lib.RunCheckWindows(root, rest, stdout, stderr)
	case "dup":
		return lib.RunCheckDup(root, rest, stdout, stderr)
	case "drift":
		return lib.RunCheckDrift(root, rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "devrites: unknown check %q\n", sub)
		return exitUsage
	}
}

func cmdCandidate(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, candidateUsage)
		return exitUsage
	}
	digest, files, err := lib.CandidateIdentity(root, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "candidate: BLOCKED: %v\n", err)
		return exitBlocked
	}
	if err := lib.VerifyReleaseUnion(root, args[0]); err != nil {
		fmt.Fprintf(stderr, "candidate: BLOCKED: %v\n", err)
		return exitBlocked
	}
	fmt.Fprintf(stdout, "candidate-sha256: %s\ncandidate-files: %d\n", digest, files)
	return exitOK
}

func cmdState(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, stateUsage)
		return exitUsage
	}
	switch args[0] {
	case "resolve":
		return lib.Resolve(root, args[1:], stdout, stderr)
	case "merge-manifest":
		return lib.RunMergeManifest(root, args[1:], stdout, stderr)
	case "close":
		return lib.CloseOut(root, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "devrites: unknown state command %q\n", args[0])
		return exitUsage
	}
}

// cmdGate runs readiness completeness or the final seal aggregate. Missing or
// failed requirements return the HITL pause code; invalid gate state returns
// the usage/internal code.
func cmdGate(root string, kind gate.Kind, args []string, stdout, stderr io.Writer) int {
	emitBinding := kind == gate.Readiness && len(args) == 2 && args[0] == "--emit-binding"
	if !emitBinding && len(args) != 1 {
		if kind == gate.Readiness {
			fmt.Fprintln(stderr, readinessUsage)
		} else {
			fmt.Fprintln(stderr, sealUsage)
		}
		return exitUsage
	}
	if emitBinding {
		binding, err := gate.ReadinessBinding(root, args[1])
		if err != nil {
			fmt.Fprintf(stderr, "readiness-binding: BLOCKED: %v\n", err)
			return exitBlocked
		}
		fmt.Fprintln(stdout, binding)
		return exitOK
	}
	result, err := gate.Check(kind, root, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	if result.Blocked {
		fmt.Fprint(stdout, result.Render())
		return exitBlocked
	}
	if kind == gate.Seal {
		if err := lib.VerifyReleaseUnion(root, args[0]); err != nil {
			fmt.Fprintf(stdout, "release-union: BLOCKED: %v\nreason: %s\n", err, gate.ResultReasonID(kind, true))
			return exitBlocked
		}
		code := lib.EvidenceFresh(root, []string{args[0]}, stdout, stderr)
		if code == exitUsage {
			return exitUsage
		}
		if code != exitOK {
			fmt.Fprintf(stdout, "reason: %s\n", gate.ResultReasonID(kind, true))
			return exitBlocked
		}
		if !notes.SealCheck(root, args[0], stdout) {
			fmt.Fprintf(stdout, "reason: %s\n", gate.ResultReasonID(kind, true))
			return exitBlocked
		}
	}
	fmt.Fprint(stdout, result.Render())
	return exitOK
}

func cmdTaskGraph(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, taskGraphUsage)
		return exitUsage
	}
	return lib.RunTaskGraphCheck(root, args[0], stdout, stderr)
}

func cmdSkillTrust(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, skillTrustUsage)
		return exitUsage
	}
	return lib.RunSkillTrustCheck(args[0], stdout, stderr)
}

func cmdObserve(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, observeUsage)
		return exitUsage
	}
	switch args[0] {
	case "summary":
		slug, code, err := lib.ActiveSlug(root, args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "observe: %v\n", err)
			if code == 0 {
				code = exitUsage
			}
			return code
		}
		return lib.RunObserveSummary(root, slug, stdout, stderr)
	case "slice":
		if len(args) != 3 {
			fmt.Fprintln(stderr, observeSliceUsage)
			return exitUsage
		}
		return lib.RunObserveSlice(root, args[1], args[2], stdout, stderr)
	}
	fmt.Fprintf(stderr, "devrites: unknown observe command %q\n", args[0])
	return exitUsage
}

func cmdOrient(root string, args []string, stdout, stderr io.Writer) int {
	slug, code, err := lib.ActiveSlug(root, args)
	if err != nil {
		fmt.Fprintf(stderr, "orient: %v\n", err)
		if code == 0 {
			code = exitUsage
		}
		return code
	}
	return lib.RunObserveSummary(root, slug, stdout, stderr)
}

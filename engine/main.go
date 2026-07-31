// Command devrites-engine runs the deterministic DevRites control plane without
// calling a model or the network. Native host agents own semantic judgment.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/install"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/version"
)

const usage = `devrites: DevRites control-plane engine

Usage:
  devrites-engine install [flags]          Install DevRites skills, agents, and host configuration
  devrites-engine update [flags]           Update an existing DevRites install in place
  devrites-engine uninstall [flags]        Remove a DevRites install, preserving runtime state
  devrites-engine check candidate <slug>   Validate and hash the closed project candidate
  devrites-engine check readiness <slug>   Check required files and the stable Build-input binding
  devrites-engine check readiness --emit-binding <slug>  Emit the stable Build-input binding for Vet
  devrites-engine check seal <slug>        Recheck the Build-input binding, final files, and evidence freshness
  devrites-engine state resolve <qid> "<ans>"  Resolve an open question and update state atomically
  devrites-engine state close <slug>       Archive a shipped feature and clear ACTIVE
  devrites-engine secret-scan [--staged] [--stdin] [slug]  Scan exact staged blobs, stdin, or touched files; HIGH blocks
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
		return cmdCheck(args[1:], stdout, stderr)
	case "state":
		return cmdState(root, args[1:], stdout, stderr)
	case "secret-scan":
		return lib.SecretScan(root, args[1:], stdin, stdout, stderr)
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

func cmdCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: devrites-engine check <candidate|readiness|seal> ...")
		return exitUsage
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "candidate":
		return cmdCandidate(rest, stdout, stderr)
	case "readiness", "seal":
		return cmdGate(gate.Kind(sub), rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "devrites: unknown check %q\n", sub)
		return exitUsage
	}
}

func cmdCandidate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: devrites-engine check candidate <slug>")
		return exitUsage
	}
	root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
		return exitUsage
	}
	digest, files, err := lib.CandidateIdentity(root, args[0])
	if err != nil {
		fmt.Fprintf(stderr, "candidate: BLOCKED: %v\n", err)
		return exitBlocked
	}
	fmt.Fprintf(stdout, "candidate-sha256: %s\ncandidate-files: %d\n", digest, files)
	return exitOK
}

func cmdState(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: devrites-engine state <resolve|close> ...")
		return exitUsage
	}
	switch args[0] {
	case "resolve":
		return lib.Resolve(root, args[1:], stdout, stderr)
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
func cmdGate(kind gate.Kind, args []string, stdout, stderr io.Writer) int {
	emitBinding := kind == gate.Readiness && len(args) == 2 && args[0] == "--emit-binding"
	if !emitBinding && len(args) != 1 {
		fmt.Fprintf(stderr, "usage: devrites-engine check %s <slug>\n", kind)
		return exitUsage
	}
	root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		fmt.Fprintf(stderr, "devrites: %v\n", err)
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
		code := lib.EvidenceFresh(root, []string{args[0]}, stdout, stderr)
		if code == exitUsage {
			return exitUsage
		}
		if code != exitOK {
			fmt.Fprintf(stdout, "reason: %s\n", gate.ResultReasonID(kind, true))
			return exitBlocked
		}
	}
	fmt.Fprint(stdout, result.Render())
	return exitOK
}

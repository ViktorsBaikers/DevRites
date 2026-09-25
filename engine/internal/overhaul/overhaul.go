// Package overhaul dispatches the deterministic tools behind the standalone
// /overhaul skill: run-record generations and validation, worker-output
// admission, baseline snapshots, readiness scoring, benchmark statistics and
// escaped offline views. It never calls a model or the network.
package overhaul

import (
	"fmt"
	"io"

	"github.com/devrites/devrites/internal/overhaul/admit"
	"github.com/devrites/devrites/internal/overhaul/bench"
	"github.com/devrites/devrites/internal/overhaul/records"
	"github.com/devrites/devrites/internal/overhaul/render"
	"github.com/devrites/devrites/internal/overhaul/score"
	"github.com/devrites/devrites/internal/overhaul/snapshot"
)

// Usage is the command-family help text.
const Usage = `usage: devrites-engine overhaul <tool> ...

  overhaul records init <repo> <run-id>
  overhaul records digest <file>
  overhaul records stage <run>
  overhaul records publish <run>
  overhaul records validate <run>
  overhaul admit receipt <run> <receipt.json> [--observed <paths.txt>]
  overhaul admit anchor <tree> <proposals.json>
  overhaul snapshot capture <repo> <out>
  overhaul snapshot fingerprint <repo>
  overhaul snapshot verify <repo> <out> [--agent-paths <file>]
  overhaul snapshot state <repo> <out.json>
  overhaul snapshot delta <before.json> <repo>
  overhaul score --rubric <r.json> --results <s.json> --gates <g.json> [--out <scorecard.json>]
  overhaul score compare --rubric <r.json> --baseline <a.json> --candidate <b.json>
  overhaul bench <result.json>
  overhaul render <run> <staged-generation> <review|report>
`

// Run executes one overhaul tool. Exit codes follow each tool: 0 ok, 1 rule
// violation or rejection, 2 usage or I/O error, 3 duplicate or late receipt.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, Usage)
		return 2
	}
	rest := args[1:]
	switch args[0] {
	case "records":
		return records.Run(rest, stdout, stderr)
	case "admit":
		return admit.Run(rest, stdout, stderr)
	case "snapshot":
		return snapshot.Run(rest, stdout, stderr)
	case "score":
		return score.Run(rest, stdout, stderr)
	case "bench":
		return bench.Run(rest, stdout, stderr)
	case "render":
		return render.Run(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "devrites: unknown overhaul tool %q\n\n%s", args[0], Usage)
		return 2
	}
}

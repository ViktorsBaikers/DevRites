package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/state"
)

// ContinuationUsage is the help text for `check continuation`.
const ContinuationUsage = "usage: devrites-engine check continuation [slug]"

// internalPhaseVerbs are autocomplete phase edges. /rite-ship and
// /rite-autocomplete stay user-facing (fresh GO, parked reopen).
var internalPhaseVerbs = map[string]bool{
	"/rite-spec":     true,
	"/rite-clarify":  true,
	"/rite-temper":   true,
	"/rite-define":   true,
	"/rite-plan":     true,
	"/rite-vet":      true,
	"/rite-build":    true,
	"/rite-prove":    true,
	"/rite-polish":   true,
	"/rite-review":   true,
	"/rite-converge": true,
	"/rite-seal":     true,
}

// continuationVerdict is the cursor-shaped answer to "may autocomplete reply?".
// Required is an internal phase edge, not a human pause (exit stays 0).
// shortcut: a host that ends the turn without running this check still stops; no Stop hook (ADR-0015). Upgrade if a host can block stop without a DevRites hook payload.
type continuationVerdict struct {
	Required bool
	Invoke   string
	Reason   string
}

// RunCheckContinuation prints whether an armed AFK run must invoke its
// next_action instead of replying. Exit 0 either way; exit 2 on usage or
// an unreadable workspace.
func RunCheckContinuation(root string, args []string, stdout, stderr io.Writer) int {
	slug, code, err := ActiveSlug(root, args)
	if err != nil {
		fmt.Fprintf(stderr, "check continuation: %v\n", err)
		if code == 0 {
			code = 2
		}
		return code
	}
	verdict, err := classifyContinuation(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "check continuation: %v\n", err)
		return 2
	}
	writeContinuation(stdout, verdict)
	return 0
}

func classifyContinuation(root, slug string) (continuationVerdict, error) {
	featureDir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return continuationVerdict{}, err
	}
	report, err := state.Status(root, slug)
	if err != nil {
		return continuationVerdict{}, err
	}
	if !afkPresent(root) {
		return continuationVerdict{Reason: "no AFK sentinel"}, nil
	}
	if report.Status != "running" {
		if report.Status == "" {
			return continuationVerdict{Reason: "status unset"}, nil
		}
		return continuationVerdict{Reason: "status=" + report.Status}, nil
	}
	// #nosec G304 -- state.md inside the checked feature workspace
	data, err := os.ReadFile(filepath.Join(featureDir, "state.md"))
	if err != nil {
		return continuationVerdict{}, err
	}
	if qid, ok := state.CursorField(strings.Split(string(data), "\n"), state.CursorQuestionID); ok && strings.TrimSpace(qid) != "" {
		return continuationVerdict{Reason: "question cursor set"}, nil
	}
	// #nosec G304 -- questions.md inside the checked feature workspace
	qdata, err := os.ReadFile(filepath.Join(featureDir, "questions.md"))
	if err != nil && !os.IsNotExist(err) {
		return continuationVerdict{}, err
	}
	if err == nil {
		for _, kind := range gate.OpenBlockingQuestionGates(qdata) {
			if kind == "blocking" || kind == "escalating" {
				return continuationVerdict{Reason: "open " + kind + " question"}, nil
			}
		}
	}
	invoke, ok := internalPhaseEdge(report.NextAction)
	if !ok {
		return continuationVerdict{Reason: "next_action is not an internal phase edge"}, nil
	}
	return continuationVerdict{Required: true, Invoke: invoke, Reason: "AFK status=running internal phase edge"}, nil
}

func afkPresent(root string) bool {
	info, err := os.Stat(filepath.Join(root, "AFK")) // #nosec G304 -- .devrites/AFK sentinel
	return err == nil && info.Mode().IsRegular()
}

func internalPhaseEdge(action string) (string, bool) {
	fields := strings.Fields(action)
	if len(fields) == 0 || !internalPhaseVerbs[strings.ToLower(fields[0])] {
		return "", false
	}
	return strings.Join(fields, " "), true
}

func writeContinuation(stdout io.Writer, verdict continuationVerdict) {
	if verdict.Required {
		fmt.Fprintln(stdout, "continuation: required")
		fmt.Fprintf(stdout, "invoke: %s\n", verdict.Invoke)
	} else {
		fmt.Fprintln(stdout, "continuation: clear")
	}
	fmt.Fprintf(stdout, "reason: %s\n", verdict.Reason)
}

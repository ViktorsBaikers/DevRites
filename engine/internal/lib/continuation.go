package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/workflow"
)

// ContinuationUsage is the help text for `check continuation`.
const ContinuationUsage = "usage: devrites-engine check continuation [slug]"

// continuationVerdict is the cursor-shaped answer to "may autocomplete reply?".
// Required is an internal phase edge, not a human pause (exit stays 0).
// shortcut: a host that ends the turn without running this check still stops; no Stop hook (ADR-0015). Upgrade if a host can block stop without a DevRites hook payload.
type continuationVerdict struct {
	Required bool
	Invoke   string
	Source   string
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
	mode, err := autocompleteMode(root)
	if err != nil {
		fmt.Fprintf(stderr, "check continuation: %v\n", err)
		return 2
	}
	verdict, err := classifyContinuation(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "check continuation: %v\n", err)
		return 2
	}
	var frontier []string
	if afkPresent(root) {
		frontier, err = FrontierLines(root, slug)
		if err != nil {
			fmt.Fprintf(stderr, "check continuation: frontier: %v\n", err)
			return 2
		}
		if mode && frontierGlobalPriority(slug, frontier) || !mode && !verdict.Required && len(frontier) > 0 && frontier[0] == "frontier: actionable" {
			reason := "global frontier requires native preparation, authority review, or selection; Git stays unauthorized"
			if mode && frontierHasBatch(frontier) {
				reason = "a source GO does not end autocomplete; run every launch command and every picks writer this turn; do not ask; Git stays unauthorized"
			}
			verdict = continuationVerdict{Required: true, Invoke: "/rite-autocomplete", Source: slug, Reason: reason}
		} else if mode && frontierSourcePreparationWait(slug, frontier) {
			verdict = continuationVerdict{Reason: frontierInheritedPreparationWait}
		} else if mode && !verdict.Required {
			if invoke, ok := frontierSourcePlanningCursor(slug, frontier); ok {
				verdict = continuationVerdict{Required: true, Invoke: invoke, Source: slug, Reason: "agent-owned planning cursor; launch every frontier-concurrent command before replying"}
			}
		}
	}
	writeContinuation(stdout, verdict, frontier)
	for _, line := range frontier {
		fmt.Fprintln(stdout, line)
	}
	return 0
}

// frontierHasBatch reports a capped same-turn batch: planning commands and/or
// admitted slice writers. The controller invoke stays /rite-autocomplete.
func frontierHasBatch(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, "frontier-concurrent: ") || strings.HasPrefix(line, "frontier-picks: ") {
			return true
		}
	}
	return false
}

func writeFrontierBatch(stdout io.Writer, lines []string) {
	for _, line := range lines {
		if cmd, ok := strings.CutPrefix(line, "frontier-concurrent: "); ok && cmd != "" {
			fmt.Fprintf(stdout, "launch: %s\n", cmd)
		}
	}
	for _, line := range lines {
		if picks, ok := strings.CutPrefix(line, "frontier-picks: "); ok && picks != "" {
			fmt.Fprintf(stdout, "picks: %s\n", picks)
		}
	}
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
	invoke, internal, err := internalPhaseEdge(report.NextAction, slug)
	if err != nil {
		if !frontierBoundSpec(root, slug, report.NextAction) {
			return continuationVerdict{}, err
		}
		// A successor Spec belongs to the global controller, never this source's phase invocation.
		invoke, internal = "", false
	}
	if !afkPresent(root) {
		return continuationVerdict{Reason: "no AFK sentinel"}, nil
	}
	if report.Status != "running" && !(report.Phase == state.PhaseVet && report.Status == "ready") {
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
	structural, err := markdowntext.Structural(data)
	if err != nil {
		return continuationVerdict{Reason: "malformed source cursor; native normalization required"}, nil
	}
	cursorLines := strings.Split(string(structural), "\n")
	for _, key := range []string{state.CursorPhase, state.CursorStatus, state.CursorNextAction, state.CursorQuestionID} {
		if _, _, err := frontierField(cursorLines, key); err != nil {
			return continuationVerdict{Reason: err.Error() + "; native normalization required"}, nil
		}
	}
	if qid, ok := state.CursorField(cursorLines, state.CursorQuestionID); ok && strings.TrimSpace(qid) != "" {
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
	if !internal {
		return continuationVerdict{Reason: "next_action is not an internal phase edge"}, nil
	}
	return continuationVerdict{Required: true, Invoke: invoke, Source: slug, Reason: "AFK status=" + report.Status + " internal phase edge"}, nil
}

func afkPresent(root string) bool {
	info, err := os.Stat(filepath.Join(root, "AFK")) // #nosec G304 -- .devrites/AFK sentinel
	return err == nil && info.Mode().IsRegular()
}

func internalPhaseEdge(action, slug string) (string, bool, error) {
	command := workflow.ForAction(action)
	policy, known := state.PolicyFor(state.Phase(command.Verb))
	if !known || policy.Target == state.PhaseFrame || policy.Target == state.PhaseShip || policy.ResumeVerb == "" {
		return "", false, nil
	}
	fields := strings.Fields(action)
	if len(fields) == 0 || fields[0] != command.Claude && fields[0] != command.Codex && fields[0] != command.Skill {
		return "", false, fmt.Errorf("next_action must start with one native phase invocation")
	}
	if workflow.ForAction(strings.Join(fields[1:], " ")).Verb != "" {
		return "", false, fmt.Errorf("next_action must contain only one phase invocation")
	}
	for i, field := range fields[1:] {
		if field == "—" {
			fields = fields[:i+1]
			break
		}
	}
	args, err := continuationArguments(command, fields[1:], slug)
	if err != nil {
		return "", false, err
	}
	return strings.Join(append(fields[:1], args...), " "), true, nil
}

func continuationArguments(command workflow.Command, args []string, slug string) ([]string, error) {
	badArgs := func() ([]string, error) {
		return nil, fmt.Errorf("invalid next_action arguments for %s: %q", command.Claude, strings.Join(args, " "))
	}
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--full" && (command.Verb == "vet" || command.Verb == "review" || command.Verb == "seal"):
		case arg == "--cross-model" && command.Verb == "vet":
		case arg == "--lookahead" && command.Verb == "spec":
		case arg == "--mode" && command.Verb == "temper":
			i++
			if i >= len(args) {
				return badArgs()
			}
			switch args[i] {
			case "expand", "selective", "hold-rigor", "reduce-to-MVP":
			default:
				return badArgs()
			}
		case arg == "--parallel" && command.Verb == "build":
			i++
			if i >= len(args) {
				return badArgs()
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 || n > 10 {
				return badArgs()
			}
		case strings.HasPrefix(arg, "-"):
			return badArgs()
		default:
			positional = append(positional, arg)
		}
	}
	if command.SourceIsFirstArgument() {
		if len(positional) > 1 {
			return badArgs()
		}
		if len(positional) == 1 && positional[0] != slug {
			return nil, fmt.Errorf("next_action targets workspace %q, not selected source %q", positional[0], slug)
		}
		if len(positional) == 0 {
			args = append(args, slug)
		}
		return args, nil
	}
	switch command.Verb {
	case "build", "polish":
		// Build's positional argument is a slice number/name, not a workspace.
		return args, nil
	case "plan":
		if len(positional) == 0 {
			return args, nil
		}
		if len(positional) <= 2 {
			switch positional[0] {
			case "decompose", "reslice", "repair", "reorder", "split", "unblock", "course-correct", "revise":
				if len(positional) == 2 && positional[1] != slug {
					return nil, fmt.Errorf("next_action targets workspace %q, not selected source %q", positional[1], slug)
				}
				return args, nil
			}
		}
	case "review":
		if len(positional) == 0 || len(positional) == 1 && positional[0] == "feature" {
			return args, nil
		}
		if len(positional) == 2 && positional[0] == "slice" {
			n, err := strconv.Atoi(positional[1])
			if err == nil && n > 0 || state.ValidSliceID(positional[1]) {
				return args, nil
			}
		}
	}
	return badArgs()
}

func writeContinuation(stdout io.Writer, verdict continuationVerdict, frontier []string) {
	if verdict.Required {
		fmt.Fprintln(stdout, "continuation: required")
		fmt.Fprintf(stdout, "invoke: %s\n", verdict.Invoke)
		fmt.Fprintf(stdout, "source: %s\n", verdict.Source)
		writeFrontierBatch(stdout, frontier)
	} else {
		fmt.Fprintln(stdout, "continuation: clear")
	}
	fmt.Fprintf(stdout, "reason: %s\n", verdict.Reason)
}

package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/state"
)

// deadEndCap bounds how many decisions.md "## Dead ends" bullets the resume
// record replays; older entries stay in the artifact itself.
const deadEndCap = 10

// readNextOrder is the canonical resume read order inside a feature
// workspace; only files that exist with content are emitted.
var readNextOrder = []string{
	"README.md",
	"state.md",
	"decisions.md",
	"questions.md",
	"gates.md",
	"test-plan.md",
	"evidence.md",
	"touched-files.md",
}

// RunHandoff emits the deterministic resume record for one feature workspace:
// cursor fields, blocking question gates, the acceptance-ledger reduction,
// recorded dead ends, in-flight dispatch waves, and the canonical read-next
// order. The record is a faithful reduction of durable artifacts so a fresh
// agent (or a handoff writer) does not depend on transcript memory to resume.
// It writes nothing unless --resume is given: then it appends one resume
// metric so `metrics summary` can time each resume to the next dispatch.
func RunHandoff(root string, args []string, stdout, stderr io.Writer) int {
	resume := slices.Contains(args, "--resume")
	args = slices.DeleteFunc(slices.Clone(args), func(a string) bool { return a == "--resume" })
	slug, code, err := ActiveSlug(root, args)
	if err != nil {
		fmt.Fprintf(stderr, "handoff: %v\n", err)
		if code == 0 {
			code = 2
		}
		return code
	}
	featureDir, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "handoff: %v\n", err)
		return 2
	}
	report, err := state.Status(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "handoff: %v\n", err)
		return 2
	}

	fmt.Fprintf(stdout, "feature: %s\n", slug)
	fmt.Fprintf(stdout, "phase: %s\n", report.Phase)
	if report.Status == "" {
		fmt.Fprintln(stdout, "status: unset")
	} else {
		fmt.Fprintf(stdout, "status: %s\n", report.Status)
	}
	if report.NextAction == "" {
		fmt.Fprintln(stdout, "next_action: unset")
	} else {
		fmt.Fprintf(stdout, "next_action: %s\n", report.NextAction)
	}
	if report.SequenceParent != "" {
		fmt.Fprintf(stdout, "sequence: parent=%s position=%s remaining=%s role=%s\n",
			report.SequenceParent, report.SequencePosition, report.SequenceWorkspacesRemaining, report.SequenceRole)
	}
	fmt.Fprintf(stdout, "awaiting_human: %s\n", yesNo(report.Status == "awaiting_human"))

	writeHandoffQuestions(featureDir, stdout)
	writeHandoffLedger(featureDir, stdout)
	writeHandoffDeadEnds(featureDir, stdout)
	writeHandoffWaves(root, slug, stdout)

	if report.Complete() {
		fmt.Fprintln(stdout, "missing_required: none")
	} else {
		fmt.Fprintf(stdout, "missing_required: %s\n", strings.Join(report.MissingFiles, ", "))
	}
	fmt.Fprintf(stdout, "read_next: %s\n", strings.Join(readNext(featureDir), ", "))
	if resume {
		recordMetric(root, slug, string(report.Phase), "resume", "root", 0)
	}
	return 0
}

func yesNo(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

func readWorkspaceFile(featureDir, name string) []byte {
	data, err := os.ReadFile(filepath.Join(featureDir, filepath.FromSlash(name))) // #nosec G304 -- name is a workspace artifact selected by engine code
	if err != nil {
		return nil
	}
	return data
}

// writeHandoffQuestions reports open question count and which gate kinds block
// progress, reusing the same parser the lifecycle gate applies.
func writeHandoffQuestions(featureDir string, stdout io.Writer) {
	questionsPath := filepath.Join(featureDir, "questions.md")
	fmt.Fprintf(stdout, "open_questions: %d\n", countOpenQuestions(questionsPath))
	gates := gate.OpenBlockingQuestionGates(readWorkspaceFile(featureDir, "questions.md"))
	if len(gates) == 0 {
		fmt.Fprintln(stdout, "blocking_question_gates: none")
		return
	}
	fmt.Fprintf(stdout, "blocking_question_gates: %s\n", strings.Join(gates, ", "))
}

// writeHandoffLedger reduces gates.md against the vetted test-plan surface and
// names the exact non-met ids so the resuming agent sees the compliance gap,
// not just a count.
func writeHandoffLedger(featureDir string, stdout io.Writer) {
	data := readWorkspaceFile(featureDir, acceptance.GatesFile)
	if data == nil {
		fmt.Fprintln(stdout, "gates_ledger: absent")
		return
	}
	doc := acceptance.ParseLedger(string(data))
	approved := acceptance.ApprovedCommands(readWorkspaceFile(featureDir, "test-plan.md"))
	r := doc.Reduce(approved)
	fmt.Fprintf(stdout, "gates_total: %d\ngates_met: %d\ngates_unmet: %d\ngates_stale: %d\ngates_abandoned: %d\n",
		r.Total, r.Met, r.Unmet, r.Stale, r.Abandoned)
	for _, id := range r.UnmetIDs {
		fmt.Fprintf(stdout, "unmet_id: %s\n", id)
	}
	for _, id := range r.StaleIDs {
		fmt.Fprintf(stdout, "stale_id: %s\n", id)
	}
	for _, id := range r.UnapprIDs {
		fmt.Fprintf(stdout, "unapproved_id: %s\n", id)
	}
	for _, id := range r.Handoffs {
		fmt.Fprintf(stdout, "abandoned_id: %s\n", id)
	}
	for _, parseErr := range doc.Errors {
		fmt.Fprintf(stdout, "ledger_error: %s\n", parseErr)
	}
}

// writeHandoffDeadEnds replays the decisions.md "## Dead ends" bullets so a
// resuming agent does not retry approaches already ruled out.
func writeHandoffDeadEnds(featureDir string, stdout io.Writer) {
	// Lstat-checked: a symlinked decisions.md could replay lines from any
	// file outside the workspace.
	data, _ := readBoundedRegularFile(filepath.Join(featureDir, "decisions.md"), workspaceArtifactLimit)
	ends := deadEnds(data)
	if len(ends) == 0 {
		fmt.Fprintln(stdout, "dead_ends: none")
		return
	}
	for _, end := range ends {
		fmt.Fprintf(stdout, "dead_end: %s\n", end)
	}
}

// writeHandoffWaves lists dispatch waves whose agents have not all returned,
// with the reduction observe uses, so a resume reconciles a running cohort
// instead of re-dispatching it.
func writeHandoffWaves(root, slug string, stdout io.Writer) {
	f, _, err := loadDispatch(root, slug)
	if err != nil {
		fmt.Fprintf(stdout, "inflight_waves: unreadable (%v)\n", err)
		return
	}
	waves := inflightWaves(f)
	if len(waves) == 0 {
		fmt.Fprintln(stdout, "inflight_waves: none")
		return
	}
	fmt.Fprintf(stdout, "inflight_waves: %d\n", len(waves))
	for _, w := range waves {
		fmt.Fprintf(stdout, "inflight_wave: %s phase=%s state=%s pending=%s slices=%s\n",
			w.Wave, w.Phase, w.State, orDash(strings.Join(w.Pending, ",")), orDash(strings.Join(w.Slices, ",")))
	}
}

func deadEnds(data []byte) []string {
	var ends []string
	inSection := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		rest := strings.TrimLeft(trimmed, "#")
		if level := len(trimmed) - len(rest); level > 0 && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
			// Only a level-one or level-two heading leaves the section; a
			// ### subheading inside it does not.
			if level <= 2 {
				inSection = strings.EqualFold(trimmed, "## Dead ends")
			}
			continue
		}
		// A bullet needs "- "; a "---" rule is not an entry.
		entry, ok := strings.CutPrefix(trimmed, "- ")
		if !inSection || !ok || strings.TrimSpace(entry) == "" {
			continue
		}
		ends = append(ends, strings.TrimSpace(entry))
	}
	if len(ends) > deadEndCap {
		ends = ends[len(ends)-deadEndCap:]
	}
	return ends
}

func readNext(featureDir string) []string {
	var next []string
	for _, name := range readNextOrder {
		path := filepath.Join(featureDir, filepath.FromSlash(name))
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			next = append(next, name)
		}
	}
	if len(next) == 0 {
		return []string{"README.md"}
	}
	return next
}

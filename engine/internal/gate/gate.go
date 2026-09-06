// Package gate implements deterministic completeness checks. Each gate checks
// only the files required for its transition when the command runs. Missing
// content returns a human-resolvable block instead of an error. Judgment gates
// remain advisory; only deterministic checks can block.
package gate

import (
	"fmt"
	"strings"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
)

// Kind names a gate. Its string form is also the CLI subcommand.
type Kind string

const (
	// Readiness checks the sections required to LEAVE the feature's current
	// phase: the gate fired when advancing to the next phase.
	Readiness Kind = "readiness"
	// Seal checks the full seal-phase requirement set regardless of the
	// feature's current phase: the final completeness gate before shipping.
	Seal Kind = "seal"
)

// Result is a gate outcome. Blocked is true iff a deterministic requirement
// failed; missing files and diagnostics retain their canonical policy order.
type Result struct {
	Kind            Kind
	Slug            string
	Phase           state.Phase
	Target          state.Phase
	Missing         []state.Section
	MissingFiles    []string
	Diagnostics     []state.ArtifactDiagnostic
	AddContentFiles []string
	StateProblems   []string
	Blocked         bool
	ReasonID        reason.ID
}

// Check runs a gate against feature <slug> under root. One workspace observation
// supplies every lifecycle, question, and readiness fact used by the result.
func Check(kind Kind, root, slug string) (*Result, error) {
	observation, err := state.ObserveWorkspace(root, slug)
	if err != nil {
		return nil, fmt.Errorf("gate %s: %w", kind, err)
	}
	result, err := checkObservation(kind, observation)
	if err != nil {
		return nil, fmt.Errorf("gate %s: %w", kind, err)
	}
	return result, nil
}

func checkObservation(kind Kind, observation *state.WorkspaceObservation) (*Result, error) {
	phase, err := observation.DeclaredPhase()
	if err != nil {
		return nil, err
	}
	target := phase
	if kind == Seal {
		target = state.PhaseSeal
	}
	policy, ok := state.PolicyFor(target)
	if !ok {
		return nil, fmt.Errorf("unknown target phase %q", target)
	}
	target = policy.Target

	missing := missingSections(observation, policy.RequiredSections)
	missingArtifacts, diagnostics := observation.Missing(policy.RequiredArtifacts)
	missingFiles := make([]string, len(missingArtifacts))
	var addContentFiles []string
	for i, artifact := range missingArtifacts {
		missingFiles[i] = string(artifact)
		fact, ok := observation.Fact(artifact)
		if ok && (fact.State() == state.ArtifactAbsent || fact.State() == state.ArtifactEmpty) {
			addContentFiles = append(addContentFiles, string(artifact))
		}
	}
	blocked := len(missingFiles) > 0
	readinessStale := false
	var stateProblems []string

	if policy.BlocksOpenQuestions {
		if gates, awaitingHuman := retainedHumanGates(observation); len(gates) > 0 {
			problem := fmt.Sprintf("open %s human question(s) remain in questions.md", strings.Join(gates, "/"))
			if !awaitingHuman {
				problem += " but state.md is not awaiting_human"
			}
			stateProblems = append(stateProblems, problem)
			blocked = true
		}
	}
	if len(missingFiles) == 0 && phaseRequiresTasks(policy) {
		if fact, ok := observation.Fact("tasks.md"); ok && fact.State() == state.ArtifactPresent {
			graph := state.ParseTaskGraph(fact.Bytes())
			for _, problem := range graph.Problems {
				stateProblems = append(stateProblems, "task-graph: "+problem)
				blocked = true
			}
		}
	}
	if len(missingFiles) == 0 {
		for _, problem := range acceptanceMapProblems(observation, policy) {
			stateProblems = append(stateProblems, "acceptance-map: "+problem)
			blocked = true
		}
	}
	if len(missingFiles) == 0 && phaseRequiresReadinessBinding(policy) {
		expected, bindingErr := verifyReadinessBinding(observation)
		if bindingErr != nil {
			readinessStale = true
			blocked = true
			diagnostics = append(diagnostics, readinessDiagnostics(observation)...)
			if expected == "" {
				stateProblems = append(stateProblems, bindingErr.Error()+"; repair the input and rerun /rite-vet")
			} else {
				stateProblems = append(stateProblems, "readiness inputs are stale or the binding is invalid; rerun /rite-vet and record exactly one standalone line: "+expected)
			}
		}
	}

	result := &Result{
		Kind:            kind,
		Slug:            observation.Slug(),
		Phase:           phase,
		Target:          target,
		Missing:         missing,
		MissingFiles:    missingFiles,
		Diagnostics:     diagnostics,
		AddContentFiles: addContentFiles,
		StateProblems:   stateProblems,
		Blocked:         blocked,
	}
	result.ReasonID = ResultReasonID(kind, blocked)
	if readinessStale {
		result.ReasonID = reason.GateReadinessStale
	}
	return result, nil
}

func acceptanceMapProblems(observation *state.WorkspaceObservation, policy state.PhasePolicy) []string {
	requireTasks := phaseRequiresTasks(policy)
	requireTestPlan := phaseRequiresTestPlan(policy)
	if !requireTasks && !requireTestPlan {
		return nil
	}
	spec, ok := observation.Fact("spec.md")
	if !ok || spec.State() != state.ArtifactPresent {
		return nil
	}
	var tasks, testPlan []byte
	if requireTasks {
		if fact, ok := observation.Fact("tasks.md"); ok && fact.State() == state.ArtifactPresent {
			tasks = fact.Bytes()
		}
	}
	if requireTestPlan {
		if fact, ok := observation.Fact("test-plan.md"); ok && fact.State() == state.ArtifactPresent {
			testPlan = fact.Bytes()
		}
	}
	return state.ParseAcceptanceMap(spec.Bytes(), tasks, testPlan, requireTasks, requireTestPlan).Problems
}

func missingSections(observation *state.WorkspaceObservation, required []state.Section) []state.Section {
	var missing []state.Section
	for _, section := range required {
		artifact := state.ArtifactPath(string(section) + ".md")
		switch section {
		case state.SectionProof:
			artifact = state.EvidenceFile
		case state.SectionStatus:
			artifact = state.LedgerFile
		}
		fact, ok := observation.Fact(artifact)
		if !ok || fact.State() != state.ArtifactPresent {
			missing = append(missing, section)
		}
	}
	return missing
}

// ResultReasonID returns the typed outcome owned by the lifecycle gate.
func ResultReasonID(kind Kind, blocked bool) reason.ID {
	switch kind {
	case Readiness:
		if blocked {
			return reason.GateReadinessMissing
		}
		return reason.GateReadinessPassed
	case Seal:
		if blocked {
			return reason.GateSealMissing
		}
		return reason.GateSealPassed
	default:
		return ""
	}
}

// Render returns stable, greppable output with a trailing newline. A blocked
// result names missing files, selected diagnostics, recovery, and the retry.
func (r *Result) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "gate: %s\n", r.Kind)
	fmt.Fprintf(&b, "feature: %s\n", r.Slug)
	fmt.Fprintf(&b, "phase: %s\n", r.Phase)
	if !r.Blocked {
		b.WriteString("result: pass\n")
		fmt.Fprintf(&b, "reason: %s\n", r.ReasonID)
		return b.String()
	}
	missing := r.MissingFiles
	switch {
	case len(missing) == 0:
		b.WriteString("result: blocked (state invariant)\n")
	case r.Kind == Seal:
		fmt.Fprintf(&b, "result: blocked (missing to seal: %s)\n", strings.Join(missing, ", "))
	default:
		fmt.Fprintf(&b, "result: blocked (missing to leave %q: %s)\n", r.Phase, strings.Join(missing, ", "))
	}
	fmt.Fprintf(&b, "reason: %s\n", r.ReasonID)
	for _, diagnostic := range r.Diagnostics {
		fmt.Fprintf(&b, "artifact: %s: %s (%s)\n", diagnostic.Path, diagnostic.State, diagnostic.Code)
	}
	if len(r.AddContentFiles) > 0 {
		fmt.Fprintf(&b, "next: add real content to %s\n", strings.Join(r.AddContentFiles, ", "))
	}
	for _, diagnostic := range r.Diagnostics {
		if recovery := diagnosticRecovery(diagnostic, targetRequiresArtifact(r.Target, diagnostic.Path)); recovery != "" {
			b.WriteString(recovery)
			b.WriteByte('\n')
		}
	}
	for _, problem := range r.StateProblems {
		fmt.Fprintf(&b, "invariant: %s\n", problem)
	}
	fmt.Fprintf(&b, "retry: devrites-engine check %s %s\n", r.Kind, r.Slug)
	return b.String()
}

func diagnosticRecovery(diagnostic state.ArtifactDiagnostic, required bool) string {
	prefix := fmt.Sprintf("next: repair %s: ", diagnostic.Path)
	repair := diagnosticRepair(diagnostic.Code)
	if repair == "" {
		return ""
	}
	if required && diagnostic.Code == state.DiagnosticMalformedMarkdown {
		repair += "; required artifacts need substantive content"
	}
	if !required {
		repair += "; optional readiness input may instead be removed"
	}
	return prefix + repair
}

func targetRequiresArtifact(target state.Phase, path state.ArtifactPath) bool {
	policy, ok := state.PolicyFor(target)
	if !ok {
		return false
	}
	for _, required := range policy.RequiredArtifacts {
		if required == path {
			return true
		}
	}
	return false
}

func diagnosticRepair(code state.DiagnosticCode) string {
	switch code {
	case state.DiagnosticMalformedMarkdown:
		return "replace invalid Markdown with valid Markdown"
	case state.DiagnosticParentSymlink:
		return "replace the symlinked parent with a real directory"
	case state.DiagnosticFinalSymlink:
		return "replace the symlink with a regular file"
	case state.DiagnosticNonRegular:
		return "replace the non-regular entry with a regular file"
	case state.DiagnosticFileTooLarge:
		return "reduce the file to at most 1 MiB"
	case state.DiagnosticPermissionDenied:
		return "grant read permission"
	case state.DiagnosticReadFailure:
		return "restore a readable regular file"
	default:
		return ""
	}
}

const gateSpaceChars = " \t\n\v\f\r"

func retainedHumanGates(observation *state.WorkspaceObservation) ([]string, bool) {
	questions, ok := observation.Fact("questions.md")
	if !ok || (questions.State() != state.ArtifactPresent && questions.State() != state.ArtifactEmpty) {
		return nil, false
	}
	gates := openBlockingQuestionGates(questions.Bytes())
	if len(gates) == 0 {
		return nil, false
	}
	ledger, ok := observation.Fact(state.LedgerFile)
	return gates, ok && ledger.State() == state.ArtifactPresent && stateAwaitingHuman(ledger.Bytes())
}

func openBlockingQuestionGates(data []byte) []string {
	lines := splitLinesNoTrailing(data)
	seen := map[string]bool{}
	var gates []string
	inQ := false
	status, gate := "", ""
	tableOpen := false
	statusColumn, gateColumn := -1, -1
	add := func(questionStatus, questionGate string) {
		questionStatus = strings.ToLower(strings.TrimSpace(questionStatus))
		questionGate = strings.ToLower(strings.TrimSpace(questionGate))
		if questionStatus == "open" && (questionGate == "blocking" || questionGate == "validating" || questionGate == "escalating") && !seen[questionGate] {
			seen[questionGate] = true
			gates = append(gates, questionGate)
		}
	}
	finalize := func() {
		if inQ {
			add(status, gate)
		}
	}
	for _, line := range lines {
		if cells, ok := questionTableCells(line); ok {
			if !tableOpen {
				tableOpen = true
				statusColumn = tableColumn(cells, "status")
				gateColumn = tableColumn(cells, "gate")
			}
			if statusColumn >= 0 && statusColumn < len(cells) && gateColumn >= 0 && gateColumn < len(cells) {
				add(cells[statusColumn], cells[gateColumn])
			}
		} else {
			tableOpen = false
			statusColumn, gateColumn = -1, -1
		}
		switch {
		case strings.HasPrefix(strings.ToLower(line), "## q-"):
			finalize()
			inQ, status, gate = true, "", ""
		case inQ && strings.HasPrefix(line, "status:"):
			status = strings.TrimLeft(strings.TrimPrefix(line, "status:"), gateSpaceChars)
		case inQ && strings.HasPrefix(line, "gate:"):
			gate = strings.TrimLeft(strings.TrimPrefix(line, "gate:"), gateSpaceChars)
		case inQ && isHeadingLine(line):
			finalize()
			inQ = false
		}
	}
	finalize()
	return gates
}

func questionTableCells(line string) ([]string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil, false
	}
	parts := strings.Split(strings.Trim(line, "|"), "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts, true
}

func tableColumn(cells []string, name string) int {
	for i, cell := range cells {
		if strings.EqualFold(cell, name) {
			return i
		}
	}
	return -1
}

func stateAwaitingHuman(data []byte) bool {
	status, _ := state.CursorField(splitLinesNoTrailing(data), state.CursorStatus)
	return status == "awaiting_human"
}

func isHeadingLine(line string) bool {
	if !strings.HasPrefix(line, "##") {
		return false
	}
	rest := line[2:]
	return rest != "" && strings.ContainsRune(gateSpaceChars, rune(rest[0]))
}

func splitLinesNoTrailing(data []byte) []string {
	s := string(data)
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

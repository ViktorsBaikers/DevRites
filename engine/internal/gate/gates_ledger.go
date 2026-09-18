package gate

import (
	"fmt"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/state"
)

// maxGateLedgerProblems bounds one gate's invariant lines so a malformed
// ledger reports a usable diagnostic instead of flooding the render.
const maxGateLedgerProblems = 8

// gateLedgerProblems reduces the workspace's gates.md acceptance ledger into
// deterministic problems: a malformed ledger, an abandoned gate's required
// handoff, an unapproved CHECK (the runner can never execute it), and — once
// the target phase requires proof — every unmet or stale outcome. The engine
// reports structure only; whether a manual gate's evidence is honest stays
// with the Prove/Seal reviewers.
func gateLedgerProblems(observation *state.WorkspaceObservation, policy state.PhasePolicy) []string {
	required := false
	for _, artifact := range policy.RequiredArtifacts {
		if artifact == state.ArtifactPath(acceptance.GatesFile) {
			required = true
			break
		}
	}
	if !required {
		return nil
	}
	fact, ok := observation.Fact(state.ArtifactPath(acceptance.GatesFile))
	if !ok || fact.State() != state.ArtifactPresent {
		return nil // the missing-artifact path already blocks it
	}
	doc := acceptance.ParseLedger(string(fact.Bytes()))
	var problems []string
	if len(doc.Errors) > 0 {
		for i, parseErr := range doc.Errors {
			if i >= maxGateLedgerProblems {
				problems = append(problems, fmt.Sprintf("... and %d more ledger errors", len(doc.Errors)-maxGateLedgerProblems))
				break
			}
			problems = append(problems, "malformed ledger: "+parseErr)
		}
		return problems
	}

	var testPlan []byte
	if plan, ok := observation.Fact("test-plan.md"); ok && plan.State() == state.ArtifactPresent {
		testPlan = plan.Bytes()
	}
	approved := acceptance.ApprovedCommands(testPlan)
	slug := observation.Slug()
	for _, gate := range doc.Gates {
		gateState := acceptance.GateState(gate, doc)
		switch {
		case gateState == acceptance.StateAbandoned:
			problems = append(problems, fmt.Sprintf(
				"gate %s abandoned: %s; hand off the outcome instead of reporting completion",
				gate.ID, doc.Abandoned[gate.ID]))
		case gate.Check != "" && !acceptance.Approved(gate.Check, gate.Cwd, approved):
			problems = append(problems, fmt.Sprintf(
				"gate %s CHECK/CWD matches no test-plan.md Build-entry preflight row; the gate can never execute",
				gate.ID))
		case gateState == acceptance.StateStaleUnmet:
			problems = append(problems, fmt.Sprintf(
				"gate %s evidence is stale or malformed; rerun `devrites-engine gates reverify %s`",
				gate.ID, slug))
		case gateState != acceptance.StateMet && policy.ProofRequired:
			problems = append(problems, fmt.Sprintf(
				"gate %s is unmet; run `devrites-engine gates run %s` or record manual evidence via `gates attest`",
				gate.ID, slug))
		}
	}
	return problems
}

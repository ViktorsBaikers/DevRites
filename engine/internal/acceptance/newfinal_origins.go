package acceptance

import "fmt"

// CheckRetainedOrigins checks current runnable met gates without executing commands.
// Historical candidate labels remain valid when the definition and INPUTS match.
func CheckRetainedOrigins(root, slug string) error {
	work, ledgerPath, testPlanPath, project, err := featurePaths(root, slug)
	if err != nil {
		return err
	}
	text, err := ReadLedgerFile(ledgerPath)
	if err != nil {
		return err
	}
	doc := ParseLedger(text)
	if len(doc.Errors) != 0 {
		return fmt.Errorf("retained gate origins: invalid %s: %v", GatesFile, doc.Errors)
	}
	approved := loadTestPlan(testPlanPath)
	for _, gate := range doc.Gates {
		if gate.Check == "" || GateState(gate, doc) != StateMet {
			continue
		}
		if !Approved(gate.Check, gate.Cwd, approved) {
			return fmt.Errorf("gate %s: CHECK/CWD is not approved by current test-plan.md; restore the approved definition and record its passing origin", gate.ID)
		}
		if gate.Inputs != "" {
			inputs, err := InputsDigest(project, gate.Inputs)
			if err != nil || inputs == "" || inputs != recordedInputs(gate) {
				return fmt.Errorf("gate %s: retained origin does not match current INPUTS; record a passing approved check against current inputs", gate.ID)
			}
		}
		if passingOrigin(work, gate) == "" {
			return fmt.Errorf("gate %s: no usable retained passing origin; record the approved check with gates run --transcripts", gate.ID)
		}
	}
	return nil
}

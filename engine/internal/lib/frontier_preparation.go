package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/workflow"
)

type preparationPurpose struct {
	parent   string
	research bool
}

func preparationPhase(phase string) bool {
	switch phase {
	case "spec", "clarify", "temper", "define", "plan", "vet":
		return true
	}
	return false
}

func preparationRole(phase, role string) bool {
	switch phase {
	case "spec", "clarify", "temper":
		return role == "evidence-scout" || role == "strategy-reviewer"
	case "define", "plan":
		return role == "plan-drafter" || role == "evidence-scout"
	case "vet":
		return role == "plan-reviewer" || role == "devex-reviewer" || role == "strategy-reviewer" || role == "code-reviewer"
	}
	return false
}

func (p preparationPurpose) validate(phase string, roles []string, context bool) error {
	phase = strings.ToLower(phase)
	if p.parent == "" && !p.research {
		return nil
	}
	if p.parent != "" && p.research {
		return fmt.Errorf("--prepare-for and --sequence-research are mutually exclusive")
	}
	if !preparationPhase(phase) {
		return fmt.Errorf("preparation purpose requires a native preparation phase")
	}
	if len(roles) == 0 && (!context || p.research) {
		return fmt.Errorf("preparation dispatch/research requires an allowed named role")
	}
	for _, role := range roles {
		if p.research && role != "evidence-scout" || !p.research && !preparationRole(phase, roleBase(role)) {
			return fmt.Errorf("role %q is not permitted for this preparation purpose", role)
		}
	}
	return nil
}

func ordinaryAdmittedPhase(phase string) bool {
	switch phase {
	case "temper", "define", "plan", "vet", "build", "prove", "polish", "review", "seal", "ship", "done":
		return true
	default:
		return false
	}
}

func earlyPreparationPhase(phase string) bool {
	phase = strings.ToLower(phase)
	return phase == "temper" || phase == "define" || phase == "plan" || phase == "vet"
}

// Context and dispatch share admission; neither a bundle nor a purpose flag proves a native launch.
func preparationAdmission(root, source, phase string, purpose preparationPurpose, current *dispatchFile) ([]string, error) {
	phase = strings.ToLower(phase)
	if purpose.research {
		if current != nil {
			return nil, nil
		}
		dir, err := devritespaths.ExistingFeatureDirChecked(root, source)
		if err != nil {
			return nil, err
		}
		raw, err := readBoundedRegularFile(filepath.Join(dir, "dispatch.json"), workspaceArtifactLimit)
		if os.IsNotExist(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		_, err = decodeDispatch(raw)
		return nil, err
	}
	if preparationPhase(phase) {
		pool, err := evaluatePreparationPool(root, source, current)
		if err != nil {
			return nil, err
		}
		if pool.enabled {
			if err := pool.admit(source, purpose); err != nil {
				return nil, err
			}
			return nil, preparationInputAdmission(pool.nodes[source])
		}
	}
	mode, err := autocompleteMode(root)
	if err != nil {
		return nil, err
	}
	var nodes map[string]*frontierWorkspace
	if mode && preparationPhase(phase) {
		nodes, err = frontierScan(root)
		if err != nil {
			return nil, err
		}
		chain, err := frontierChain(nodes, source)
		if err != nil {
			return nil, err
		}
		ordered, _ := frontierRelated(nodes, chain)
		parked, err := preparationParking(root, source, ordered, nodes)
		if err != nil {
			return nil, err
		}
		if parked != "" {
			return nil, fmt.Errorf("frontier-blocked: %s — %s", source, parked)
		}
		if err := preparationInputAdmission(nodes[source]); err != nil {
			return nil, err
		}
	}
	if purpose.parent == "" && !earlyPreparationPhase(phase) {
		return nil, nil
	}
	if purpose.parent == "" && !mode {
		return nil, nil
	}
	allow, capacity, err := frontierPolicy(root)
	if err != nil {
		return nil, err
	}
	if mode {
		_, lease, leaseErr := parallel.ControllerLease(filepath.Dir(root))
		if leaseErr != nil {
			return nil, leaseErr
		}
		if lease != nil && lease.Capacity > 0 && lease.Capacity < capacity {
			capacity = lease.Capacity
		}
	}
	if purpose.parent == "" && capacity <= 1 {
		return nil, nil
	}
	if nodes == nil {
		nodes, err = frontierScan(root)
		if err != nil {
			return nil, err
		}
	}
	chain, err := frontierChain(nodes, source)
	if err != nil {
		return nil, err
	}
	if mode && capacity > 1 {
		if _, err := frontierPreparationBindings(nodes[source], current); err != nil {
			return nil, err
		}
	}
	ordered, _ := frontierRelated(nodes, chain)
	mapping, _, problem, invalid := frontierRoadmap(nodes, ordered)
	if invalid {
		return nil, fmt.Errorf("invalid canonical roadmap: %s", problem)
	}
	if purpose.parent != "" {
		parent, child := nodes[purpose.parent], nodes[source]
		if parent == nil || child == nil || parent == child || child.parent != parent.slug || !parent.hasPosition || !child.hasPosition || child.position != parent.position+1 {
			return nil, fmt.Errorf("--prepare-for requires an existing bound immediate parent with adjacent sequence positions")
		}
		if _, err := frontierChain(nodes, parent.slug); err != nil {
			return nil, err
		}
		target, err := frontierSuccessor(parent, nodes, mapping)
		if problem != "" || err != nil || target != source {
			return nil, fmt.Errorf("--prepare-for parent/successor authority does not agree: %s %v", problem, err)
		}
		return nil, nil
	}
	records, err := frontierEvidence(root, source)
	if err != nil {
		return nil, err
	}
	return frontierPreparationAdmission(root, source, nodes, mapping, problem, allow && capacity > 1, records)
}

const preparationInputReason = "Plan repair requires non-empty drift.md; controller must reconcile recorded findings and conflicting briefs before drafting; preserve scope and human holds"

func preparationInputRepair(n *frontierWorkspace) (string, error) {
	fields := strings.Fields(n.next)
	if len(fields) < 2 || fields[1] != "repair" || workflow.ForAction(n.next).Verb != "plan" {
		return "", nil
	}
	invoke, internal, err := internalPhaseEdge(n.next, n.slug)
	if err != nil || !internal {
		return "", err
	}
	raw, err := readBoundedRegularFile(filepath.Join(n.dir, "drift.md"), workspaceArtifactLimit)
	if os.IsNotExist(err) || err == nil && len(bytes.TrimSpace(raw)) == 0 {
		return commandWithWorkspace(invoke, n.slug), nil
	}
	return "", err
}

func preparationInputAdmission(n *frontierWorkspace) error {
	command, err := preparationInputRepair(n)
	if err != nil {
		return err
	}
	if command != "" {
		return fmt.Errorf("%s: %s; root must consume repair-input: %s before dispatch open", n.slug, preparationInputReason, command)
	}
	return nil
}

// Validate the whole retained ledger before any current-scope classification or credit.
// A decoded account may be supplied by dispatch admission; recovery uses decode alone.
func frontierPreparationBindings(n *frontierWorkspace, account *dispatchFile) (*dispatchFile, error) {
	if n == nil {
		return nil, fmt.Errorf("preparation binding source is not an existing sequence workspace")
	}
	if account == nil {
		raw, err := readBoundedRegularFile(filepath.Join(n.dir, "dispatch.json"), workspaceArtifactLimit)
		if os.IsNotExist(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		account, err = decodeDispatch(raw)
		if err != nil {
			return nil, err
		}
	}
	for _, wave := range account.Waves {
		if wave.PreparationFor != "" && wave.PreparationFor != n.parent && wave.State != "complete" && (wave.State != "abandoned" || wave.TerminalAt == "") {
			return nil, fmt.Errorf("workspace %s preparation account does not match its immediate parent", n.slug)
		}
	}
	return account, nil
}

func frontierPreparatoryOnly(root string, n *frontierWorkspace, nodes map[string]*frontierWorkspace, mapping map[int]string, problem string) (bool, *dispatchFile, error) {
	account, err := frontierPreparationBindings(n, nil)
	if err != nil {
		return false, nil, err
	}
	parent := nodes[n.parent]
	if parent == nil || frontierTerminal(parent) || frontierTerminal(n) || problem != "" || !parent.hasPosition || !n.hasPosition || n.position != parent.position+1 {
		return false, account, nil
	}
	if _, err := frontierChain(nodes, n.slug); err != nil {
		return false, account, nil
	}
	target, err := frontierSuccessor(parent, nodes, mapping)
	if err != nil || target != n.slug {
		return false, account, nil
	}
	active, err := readBoundedRegularFile(filepath.Join(root, "ACTIVE"), workspaceArtifactLimit)
	if err != nil && !os.IsNotExist(err) {
		return false, account, err
	}
	if strings.TrimSpace(string(active)) == n.slug {
		return false, account, nil
	}
	if account == nil {
		// A recorded Temper-or-later phase is ordinary work. Missing dispatch
		// waves do not keep every later sibling preparatory-only.
		if ordinaryAdmittedPhase(n.phase) {
			return false, nil, nil
		}
		return true, nil, nil
	}
	for _, wave := range account.Waves {
		if wave.PreparationFor == "" && !wave.SequenceResearch && (earlyPreparationPhase(wave.Phase) || isBuildWrightWave(wave)) {
			return false, account, nil
		}
	}
	return true, account, nil
}

package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/parallel"
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
	if purpose.parent == "" && !earlyPreparationPhase(phase) {
		return nil, nil
	}
	mode, err := autocompleteMode(root)
	if err != nil {
		return nil, err
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
	nodes, err := frontierScan(root)
	if err != nil {
		return nil, err
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
		return true, nil, nil
	}
	for _, wave := range account.Waves {
		if wave.PreparationFor == "" && !wave.SequenceResearch && (earlyPreparationPhase(wave.Phase) || isBuildWrightWave(wave)) {
			return false, account, nil
		}
	}
	return true, account, nil
}

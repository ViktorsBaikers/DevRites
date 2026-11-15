package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/parallel"
)

type poolParkingRecord struct {
	Root     string `json:"root"`
	Source   string `json:"source"`
	Parent   string `json:"parent,omitempty"`
	Position int    `json:"position"`
	Reason   string `json:"reason"`
	ParkedAt string `json:"parked_at"`
}

type preparationPoolEntry struct {
	target, parent                  string
	position, remaining             int
	missing, selected               bool
	parked                          string
	occupied, started, buildStarted bool
	retained                        bool
}

type preparationPool struct {
	enabled    bool
	controller string
	capacity   int
	entries    []preparationPoolEntry
	nodes      map[string]*frontierWorkspace
	accounts   map[string]*dispatchFile
	mapping    map[int]string
	occupied   int
}

func (p *preparationPool) entry(target string) *preparationPoolEntry {
	for i := range p.entries {
		if p.entries[i].target == target {
			return &p.entries[i]
		}
	}
	return nil
}

func unresolvedWave(w *dispatchWave) bool {
	return w.State != "complete" && !(w.State == "abandoned" && w.TerminalAt != "")
}

func preparationAccount(root, source string, n *frontierWorkspace, current *dispatchFile) (*dispatchFile, error) {
	if n.slug == source && current != nil {
		return current, nil
	}
	raw, err := readBoundedRegularFile(filepath.Join(n.dir, "dispatch.json"), workspaceArtifactLimit)
	if os.IsNotExist(err) {
		return &dispatchFile{Waves: map[string]*dispatchWave{}}, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeDispatch(raw)
}

// Holds have one lookup regardless of the observing source or admission marker.
func preparationParking(root, target string, ordered []string, nodes map[string]*frontierWorkspace) (string, error) {
	var reasons []string
	for _, owner := range ordered {
		records, err := frontierEvidence(root, owner)
		if err != nil {
			return "", err
		}
		if reason := records.blocked[target]; reason != "" {
			reasons = append(reasons, "owner="+owner+": "+reason+"; re-evaluate after owner resolves frontier-blocked record")
		}
	}
	if n := nodes[target]; n != nil && n.location == "live" {
		hold, err := frontierSchedulingHold(n, true)
		if err != nil {
			return "", err
		}
		if hold != "" {
			reasons = append(reasons, hold+"; re-evaluate after native hold resolution")
		}
	}
	return strings.Join(reasons, "; "), nil
}

// Selection is derived from canonical identity, state and retained dispatch evidence.
// Reads reserve nothing; callers publish under the repository controller lock.
func evaluatePreparationPool(root, source string, current *dispatchFile) (*preparationPool, error) {
	p := &preparationPool{}
	mode, err := autocompleteMode(root)
	if err != nil || !mode {
		return p, err
	}
	allow, capacity, err := frontierPolicy(root)
	if err != nil {
		return nil, err
	}
	controller, lease, err := parallel.ControllerLease(filepath.Dir(root))
	if err != nil {
		return nil, err
	}
	if lease != nil && lease.Capacity > 0 && lease.Capacity < capacity {
		capacity = lease.Capacity
	}
	p.capacity = capacity
	if capacity <= 1 {
		return p, nil
	}
	nodes, err := frontierScan(root)
	if err != nil {
		return nil, err
	}
	chain, err := frontierChain(nodes, source)
	if err != nil {
		return nil, err
	}
	ordered, related := frontierRelated(nodes, chain)
	// A nonnil canonical result is provenance, not an inferred equality with legacy maps.
	mapping, _, err := frontierCanonicalRoadmap(nodes, ordered)
	if err != nil {
		return nil, err
	}
	if mapping == nil {
		return p, nil
	}
	if controller == "" {
		for _, slug := range ordered {
			if nodes[slug].location == "live" {
				controller = slug
				break
			}
		}
	} else if !related[controller] {
		return nil, fmt.Errorf("repository controller %s is outside source sequence", controller)
	}
	p.enabled, p.controller, p.nodes, p.mapping = true, controller, nodes, mapping
	p.accounts = make(map[string]*dispatchFile, len(nodes))
	// Decode all accounts before classifying eligibility, including detached/archived actors.
	for slug, n := range nodes {
		if n.dir == "" {
			return nil, fmt.Errorf("workspace %s: %v", slug, n.problem)
		}
		account, err := preparationAccount(root, source, n, current)
		if err != nil {
			return nil, fmt.Errorf("workspace %s dispatch: %w", slug, err)
		}
		p.accounts[slug] = account
		if parked := account.PoolParking; parked != nil && (parked.Root == mapping[1] || related[slug]) {
			if parked.Source != slug || parked.Root != mapping[1] || mapping[parked.Position] != slug || parked.Parent != mapping[parked.Position-1] || n.parent != parked.Parent {
				return nil, fmt.Errorf("parked source %s has changed canonical binding; normalize source-owned parking before resume", slug)
			}
		}
		for _, wave := range account.Waves {
			if !unresolvedWave(wave) || wave.PoolSource == "" {
				continue
			}
			if wave.PoolRoot == mapping[1] || related[slug] {
				if wave.PoolSource != slug || wave.PoolRoot != mapping[1] || mapping[wave.PoolPosition] != slug || wave.PoolParent != mapping[wave.PoolPosition-1] || n.parent != wave.PoolParent {
					return nil, fmt.Errorf("retained preparation %s has changed canonical binding; settle/invalidate native actors before new admission", slug)
				}
			}
		}
	}
	override, err := devritespaths.WorkspaceOverrideChecked(root, source)
	if err != nil {
		return nil, err
	}
	if override != "" {
		p.enabled = false
		return p, nil
	}
	remaining, knownBudget := 0, false
	for position := 1; position <= len(mapping); position++ {
		target := mapping[position]
		if target == "" {
			return nil, fmt.Errorf("canonical roadmap positions must be contiguous")
		}
		e := preparationPoolEntry{target: target, parent: mapping[position-1], position: position}
		n := nodes[target]
		if n == nil {
			e.missing = true
			if knownBudget && remaining > 0 {
				remaining--
			} else {
				knownBudget = false
			}
			e.remaining = remaining
			if !allow {
				e.parked = "new workspace openings disabled"
			} else if !knownBudget {
				e.parked = "durable sequence opening budget exhausted or missing"
			}
		} else {
			if n.problem != nil {
				return nil, n.problem
			}
			if n.hasRemaining {
				if position > 1 && knownBudget {
					remaining = min(max(0, remaining-1), n.remaining)
				} else {
					remaining = n.remaining
				}
				knownBudget = true
			} else {
				knownBudget = false
			}
			e.remaining = remaining
			account := p.accounts[target]
			for _, wave := range account.Waves {
				if strings.EqualFold(wave.Phase, "build") {
					for role, actor := range wave.Roles {
						if strings.Contains(role, "wright") && actor != nil && actor.StartedAt != "" {
							e.buildStarted = true
						}
					}
				}
				if !unresolvedWave(wave) {
					continue
				}
				for _, role := range wave.Roles {
					if !role.Returned {
						e.retained = true
					}
				}
				if wave.PoolSource != "" && (wave.PoolSource != target || wave.PoolRoot != mapping[1] || wave.PoolPosition != position || wave.PoolParent != e.parent) {
					return nil, fmt.Errorf("retained preparation %s has changed canonical binding; settle/invalidate native actors before new admission", target)
				}
				if wave.PreparationFor != "" && wave.PreparationFor != e.parent {
					return nil, fmt.Errorf("workspace %s preparation account does not match its immediate parent", target)
				}
			}
			e.started = frontierPendingPreparation(account, n.phase)
			if !e.buildStarted && (e.retained || n.location == "live" && !frontierTerminal(n)) {
				e.occupied = true
			}
		}
		parked, err := preparationParking(root, target, ordered, nodes)
		if err != nil {
			return nil, err
		}
		if parked != "" {
			e.parked = parked
		}
		if account := p.accounts[target]; account != nil && account.PoolParking != nil {
			e.parked = "durably parked: " + account.PoolParking.Reason + "; re-admit with devrites-engine dispatch " + target + " resume after hold resolution"
		}
		if e.parked != "" && !e.retained {
			e.occupied = false
		}
		if e.occupied {
			p.occupied++
		}
		p.entries = append(p.entries, e)
	}
	free := capacity - p.occupied
	for i := range p.entries {
		e := &p.entries[i]
		if e.parked != "" {
			continue
		}
		if e.buildStarted {
			e.selected = true
			continue
		}
		if e.occupied {
			e.selected = e.retained || p.occupied <= capacity
			continue
		}
		if !e.missing {
			continue
		}
		// Creation publishes only an immediate identity connector, then refreshes.
		if free > 0 && nodes[e.parent] != nil {
			e.selected = true
			free--
		}
	}
	return p, nil
}

func (p *preparationPool) admit(source string, purpose preparationPurpose) error {
	e := p.entry(source)
	if e == nil || e.missing {
		return fmt.Errorf("source %s is outside canonical preparation pool", source)
	}
	if purpose.parent != "" && purpose.parent != e.parent {
		return fmt.Errorf("--prepare-for requires exact immediate parent %s", e.parent)
	}
	if e.parked != "" {
		return fmt.Errorf("frontier-blocked: %s — %s", source, e.parked)
	}
	if !e.selected {
		return fmt.Errorf("resource deferral: preparation pool full (%d/%d); re-evaluate after first Build start, terminal settlement or durable parking", p.occupied, p.capacity)
	}
	return nil
}

func nativeHostCapacity() (int, error) {
	raw, present := os.LookupEnv("DEVRITES_HOST_CAPACITY")
	if !present {
		return 1, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 || strconv.Itoa(n) != raw {
		return 0, fmt.Errorf("invalid DEVRITES_HOST_CAPACITY; native host must supply a positive total actor budget")
	}
	return n, nil
}

type hostReservation struct {
	source, wave, role string
}

func hostReservations(root, source string, current *dispatchFile) (int, hostReservation, error) {
	nodes, err := frontierScan(root)
	if err != nil {
		return 0, hostReservation{}, err
	}
	used := 0
	var first hostReservation
	for _, n := range nodes {
		if n.dir == "" {
			return 0, hostReservation{}, fmt.Errorf("cannot reconcile retained host accounts for %s: %v", n.slug, n.problem)
		}
		account, err := preparationAccount(root, source, n, current)
		if err != nil {
			return 0, hostReservation{}, err
		}
		for name, wave := range account.Waves {
			// Host slots end at abandonment; fulfillment and claim settlement remain separate.
			if wave.State == "abandoned" || !unresolvedWave(wave) {
				continue
			}
			for nameRole, role := range wave.Roles {
				if !role.Returned {
					if used == 0 {
						first = hostReservation{source: n.slug, wave: name, role: nameRole}
					}
					used++
				}
			}
		}
	}
	return used, first, nil
}

func hostWaveAdmission(root, source string, current *dispatchFile, roles int) error {
	capacity, err := nativeHostCapacity()
	if err != nil {
		return err
	}
	used, first, err := hostReservations(root, source, current)
	if err != nil {
		return err
	}
	if roles > capacity-used {
		reservation := ""
		if used > 0 {
			reservation = fmt.Sprintf("; live or unreconciled reservation %s/%s/%s", first.source, first.wave, first.role)
		}
		return fmt.Errorf("resource deferral: native host capacity %d, retained reservations %d, complete wave requires %d%s; split required roster into sealable subwaves before open, retain full review barrier, and re-evaluate after actor settlement", capacity, used, roles, reservation)
	}
	return nil
}

func (p *preparationPool) writerPriority(root, source string) (int, error) {
	records, err := frontierEvidence(root, source)
	if err != nil {
		return 0, err
	}
	var candidates []parallel.SlicePaths
	readyKeys := map[string]bool{}
	for _, e := range p.entries {
		n := p.nodes[e.target]
		if n == nil || n.location != "live" || e.parked != "" || e.retained || !(n.phase == "build" || n.phase == "vet" && n.status == "ready") {
			continue
		}
		ready, _, err := WorkspaceReady(filepath.Dir(root), p.controller, e.target)
		if err != nil {
			return 0, err
		}
		for _, candidate := range ready {
			workspace := candidate.Workspace
			if workspace == "" {
				workspace = p.controller
			}
			if workspace != source && frontierSerialReason(records, source, workspace, candidate.ID) != "" {
				continue
			}
			candidates = append(candidates, candidate)
			readyKeys[parallel.SliceKey(workspace, candidate.ID)] = true
		}
	}
	pending := map[string]bool{}
	controller, lease, err := parallel.ControllerLease(filepath.Dir(root))
	if err != nil {
		return 0, err
	}
	if lease != nil {
		for _, row := range lease.Slices {
			if row.WrightStatus == parallel.WrightIntegrated || row.WrightStatus == parallel.WrightGreen {
				continue
			}
			workspace := row.Workspace
			if workspace == "" {
				workspace = controller
			}
			key := parallel.SliceKey(workspace, row.ID)
			if !readyKeys[key] {
				continue
			}
			dispatched := false
			if account := p.accounts[workspace]; account != nil {
				for _, wave := range account.Waves {
					if unresolvedWave(wave) && isBuildWrightWave(wave) && containsSlice(wave.Slices, row.ID) {
						dispatched = true
						break
					}
				}
			}
			if !dispatched {
				pending[key] = true
			}
		}
	}
	if len(candidates) != 0 {
		_, picks, _, err := parallel.SelectForSlug(filepath.Dir(root), p.controller, p.capacity, candidates)
		if err != nil {
			return 0, err
		}
		for _, pick := range picks {
			workspace := pick.Workspace
			if workspace == "" {
				workspace = p.controller
			}
			pending[parallel.SliceKey(workspace, pick.ID)] = true
		}
	}
	return len(pending), nil
}

func (p *preparationPool) hostPreparationAdmission(root, source string, current *dispatchFile, roles int) error {
	if err := hostWaveAdmission(root, source, current, roles); err != nil {
		return err
	}
	priority, err := p.writerPriority(root, source)
	if err != nil {
		return err
	}
	capacity, err := nativeHostCapacity()
	if err != nil {
		return err
	}
	used, _, err := hostReservations(root, source, current)
	if err != nil {
		return err
	}
	if roles > capacity-used-priority {
		return fmt.Errorf("resource deferral: %d eligible writer launches have native host priority; re-evaluate preparation after writer launch or actor settlement", priority)
	}
	return nil
}

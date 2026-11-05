package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/state"
)

func autocompleteMode(root string) (bool, error) {
	raw, err := readBoundedRegularFile(filepath.Join(root, "AFK"), workspaceArtifactLimit)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	seen := false
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok || strings.TrimSpace(key) != "mode" {
			continue
		}
		if seen {
			return false, fmt.Errorf("duplicate AFK mode; normalize controller identity")
		}
		seen = true
		value = strings.TrimSpace(value)
		if i := strings.Index(value, " #"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		if value != "autocomplete" {
			return false, fmt.Errorf("invalid AFK mode; expected autocomplete")
		}
	}
	return seen, nil
}

func frontierRelated(nodes map[string]*frontierWorkspace, chain []string) ([]string, map[string]bool) {
	related := map[string]bool{chain[len(chain)-1]: true}
	for range maxSequenceChain {
		changed := false
		for name, n := range nodes {
			if related[n.parent] && !related[name] {
				related[name], changed = true, true
			}
		}
		if !changed {
			break
		}
	}
	ordered := make([]string, 0, len(related))
	depth := make(map[string]int, len(related))
	for name := range related {
		ordered = append(ordered, name)
		ancestors, _ := frontierChain(nodes, name)
		depth[name] = len(ancestors)
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := nodes[ordered[i]], nodes[ordered[j]]
		if depth[a.slug] != depth[b.slug] {
			return depth[a.slug] < depth[b.slug]
		}
		if a.position != b.position {
			return a.position < b.position
		}
		return a.slug < b.slug
	})
	return ordered, related
}

type frontierRecords struct {
	blocked map[string]string
	serial  map[string]string
}

var frontierBlockedRe = regexp.MustCompile("(?m)^[ \\t]*(?:[-*][ \\t]+)?frontier-blocked:[ \\t]+([a-zA-Z0-9][a-zA-Z0-9._-]*)[ \\t]+—[ \\t]+([^\\s`][^\\n]*|`[^\\s`][^\\n]*)")

func frontierEvidence(root, slug string) (frontierRecords, error) {
	records := frontierRecords{}
	raw, err := readSequenceArtifact(root, slug, "evidence.md")
	if os.IsNotExist(err) {
		return records, nil
	}
	if err != nil {
		return records, err
	}
	raw, err = markdowntext.Structural(raw)
	if err != nil {
		return records, err
	}
	text := string(raw)
	for _, match := range frontierBlockedRe.FindAllStringSubmatch(text, -1) {
		if devritespaths.ValidSlug(match[1]) {
			if records.blocked == nil {
				records.blocked = map[string]string{}
			}
			records.blocked[match[1]] = strings.TrimSpace(match[2])
		}
	}
	for _, match := range serialRecordRe.FindAllStringSubmatch(text, -1) {
		if !serialReasonWaives(match[2]) {
			continue
		}
		for _, token := range strings.FieldsFunc(match[1], func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '`' }) {
			workspace, id, qualified := strings.Cut(token, "/")
			if !qualified {
				id, workspace = workspace, ""
			}
			id = strings.ToUpper(id)
			if !state.ValidSliceID(id) || qualified && !devritespaths.ValidSlug(workspace) {
				continue
			}
			if records.serial == nil {
				records.serial = map[string]string{}
			}
			records.serial[parallel.SliceKey(workspace, "SLICE-"+sliceNumber(id))] = strings.TrimSpace(match[2])
		}
	}
	return records, nil
}

func frontierSerialReason(records frontierRecords, source, workspace, id string) string {
	if source == workspace {
		workspace = ""
	}
	return records.serial[parallel.SliceKey(workspace, "SLICE-"+sliceNumber(id))]
}

func frontierPendingPreparation(n *frontierWorkspace) (bool, error) {
	raw, err := readBoundedRegularFile(filepath.Join(n.dir, "dispatch.json"), workspaceArtifactLimit)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	dispatch, err := decodeDispatch(raw)
	if err != nil {
		return false, err
	}
	for _, wave := range dispatch.Waves {
		if wave.State != "open" && wave.State != "sealed" {
			continue
		}
		for role, actor := range wave.Roles {
			if actor.Handle == "" || actor.StartedAt == "" || actor.Returned {
				continue
			}
			base := roleBase(role)
			switch strings.ToLower(wave.Phase) {
			case "spec", "clarify", "temper":
				if base == "evidence-scout" || base == "strategy-reviewer" {
					return true, nil
				}
			case "define", "plan":
				if base == "plan-drafter" || base == "evidence-scout" {
					return true, nil
				}
			case "vet":
				if base == "plan-reviewer" || base == "devex-reviewer" || base == "strategy-reviewer" || base == "code-reviewer" {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func frontierSuccessor(n *frontierWorkspace, nodes map[string]*frontierWorkspace, mapping map[int]string) (string, error) {
	target := frontierSpecTarget(n)
	if mapped := mapping[n.position+1]; n.hasPosition && mapped != "" {
		if target != "" && mapped != target {
			return "", fmt.Errorf("next Spec conflicts with recorded successor; native authority-normalization required")
		}
		target = mapped
	}
	for _, child := range nodes {
		if child.parent != n.slug || !n.hasPosition || !child.hasPosition || child.position != n.position+1 {
			continue
		}
		if target != "" && target != child.slug {
			return "", fmt.Errorf("conflicting successor identities; native authority-normalization required")
		}
		target = child.slug
	}
	if child := nodes[target]; child != nil && child.parent != n.slug {
		return "", fmt.Errorf("successor %s is not bound to source %s; native authority-normalization required", target, n.slug)
	}
	return target, nil
}

func frontierPreparationAdmission(root, source string, nodes map[string]*frontierWorkspace, mapping map[int]string, problem string, allow bool, records frontierRecords) ([]string, error) {
	n := nodes[source]
	if !allow || n.hasRemaining && n.remaining == 0 {
		return nil, nil
	}
	if hold, err := frontierQuestionHold(n); err != nil || hold != "" {
		return nil, err
	}
	target, err := frontierSuccessor(n, nodes, mapping)
	for _, identity := range [...]string{frontierSpecTarget(n), mapping[n.position+1], target} {
		if child := nodes[identity]; child != nil && child.problem != nil {
			return nil, child.problem
		}
	}
	if problem != "" || err != nil || target == "" && n.hasRemaining && n.remaining > 0 {
		if reason := records.blocked["sequence"]; reason != "" {
			return []string{"frontier-blocked: sequence — " + reason + "; scheduling deferred, not normalized"}, nil
		}
		reason := problem
		if err != nil {
			reason = err.Error()
		}
		if reason == "" {
			reason = "positive remaining budget has no exact recorded successor"
		}
		return nil, fmt.Errorf("source %s native authority-normalization required before writer admission: %s; read sequence artifacts, record current roadmap through its owning decision, or retain frontier-blocked: sequence — <specific constraint>", source, reason)
	}
	if target == "" {
		return nil, nil
	}
	if child := nodes[target]; child != nil {
		if _, err := frontierChain(nodes, target); err != nil {
			return nil, err
		}
	}
	if reason := records.blocked[target]; reason != "" {
		return []string{"frontier-blocked: " + target + " — " + reason + "; preparation deferred, not approved"}, nil
	}
	if child := nodes[target]; child != nil {
		if hold, err := frontierQuestionHold(child); err != nil || hold != "" {
			return nil, err
		}
		if frontierTerminal(child) {
			return nil, nil
		}
		if pending, err := frontierPendingPreparation(child); err != nil || pending {
			return nil, err
		}
		ready, notices, err := WorkspaceReady(filepath.Dir(root), source, target)
		if err != nil {
			return nil, err
		}
		if len(ready) > 0 || len(notices) == 0 {
			var diagnostics []string
			for _, notice := range notices {
				diagnostics = append(diagnostics, "frontier-readiness: "+notice)
			}
			return diagnostics, nil
		}
		dependenciesOnly := true
		for _, notice := range notices {
			dependenciesOnly = dependenciesOnly && strings.Contains(notice, ": waiting for durable BUILT/integrated producer ")
		}
		if dependenciesOnly {
			return nil, nil
		}
	}
	return nil, fmt.Errorf("successor %s preparation has not actually started; invoke /rite-spec %s or its native Plan/Vet cursor and retain launched planning/review handles before /rite-build %s; alternatively retain frontier-blocked: %s — <specific constraint> in evidence.md", target, frontierQuote(target), source, target)
}

func frontierNormalizePaths(paths []string) error {
	for i, path := range paths {
		normalized, err := parallel.NormalizePath(path)
		if err != nil {
			return err
		}
		paths[i] = normalized
	}
	return nil
}

func frontierSamePaths(admitted, reserved []string) bool {
	if len(admitted) == 0 || len(admitted) != len(reserved) {
		return false
	}
	for _, reservedPath := range reserved {
		found := false
		for _, path := range admitted {
			found = found || strings.EqualFold(path, reservedPath)
		}
		if !found {
			return false
		}
	}
	return true
}

func frontierLeaseReserved(lease *parallel.Lease, controller, source string, ids []string, ready []parallel.SlicePaths) bool {
	if lease == nil || lease.Status != parallel.StatusRunning && lease.Status != parallel.StatusIntegrateFailed || lease.PendingLanding != nil {
		return false
	}
	for _, id := range ids {
		found := false
		for _, row := range lease.Slices {
			workspace := row.Workspace
			if workspace == "" {
				workspace = controller
			}
			if workspace != source || sliceNumber(row.ID) != sliceNumber(id) || row.WrightStatus == parallel.WrightIntegrated {
				continue
			}
			for _, candidate := range ready {
				if sliceNumber(candidate.ID) == sliceNumber(id) && frontierSamePaths(candidate.Paths, row.Paths) {
					found = true
				}
			}
		}
		if !found {
			return false
		}
	}
	return len(ids) > 0
}

func frontierWriterRetention(root, source, controller, binding string, lease *parallel.Lease, nodes map[string]*frontierWorkspace, ordered, ids []string, current *dispatchFile) error {
	var claims map[string]*claimRecord
	for _, slug := range ordered {
		if _, err := frontierChain(nodes, slug); err != nil {
			return err
		}
		dispatch := current
		if slug != source {
			raw, err := readBoundedRegularFile(filepath.Join(nodes[slug].dir, "dispatch.json"), workspaceArtifactLimit)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			dispatch, err = decodeDispatch(raw)
			if err != nil {
				return err
			}
		}
		for name, wave := range dispatch.Waves {
			if wave.State == "complete" || wave.TerminalAt != "" {
				continue
			}
			writerChecked := false
			for role, actor := range wave.Roles {
				wright := strings.Contains(role, "wright")
				if actor.Returned || actor.Handle == "" && (!wright || wave.State == "abandoned") {
					continue
				}
				if !wright {
					if slug == source && (isReviewerRole(role) || roleBase(role) == "proof-runner") {
						return fmt.Errorf("source %s retains a protected reviewer/proof account; reconcile it before writer admission", source)
					}
					continue
				}
				if !writerChecked {
					if binding == "" || wave.ControllerBatch != binding {
						return fmt.Errorf("retained writer %s/%s has unknown/same-tree isolation; reconcile its native handle before another writer; later lease IDs cannot retrofit actor checkout", slug, name)
					}
					if wave.Claim != "" {
						if claims == nil {
							latest, _, bad, err := readClaims(claimsPath(root))
							if err != nil {
								return err
							}
							if bad > 0 {
								return fmt.Errorf("%d unparsed ledger line(s); retained writer claim identity cannot be validated", bad)
							}
							claims = latest
						}
						claim := claims[wave.Claim]
						if claim == nil || claim.Session != wave.ClaimSession {
							return fmt.Errorf("retained writer %s/%s claim %s has missing or mismatched ledger session", slug, name, wave.Claim)
						}
						// Release/expiry changes allocation, never the pending actor's frozen authority.
					} else if wave.ClaimSession != "" {
						return fmt.Errorf("retained writer %s/%s has a claim session without its exact claim identity", slug, name)
					}
					writerChecked = true
				}
				retainedID := ""
				if lane, ok := wrightLane(role); ok {
					for _, id := range wave.Slices {
						if sliceNumber(id) == sliceNumber(lane) {
							retainedID = id
							break
						}
					}
				} else if len(wave.Slices) == 1 {
					retainedID = wave.Slices[0]
				}
				admitted := wave.AdmittedPaths[retainedID]
				if retainedID == "" || len(admitted) == 0 {
					return fmt.Errorf("retained writer %s/%s role %s lacks immutable admitted paths; reconcile the native actor, not mutable tasks or lease IDs", slug, name, role)
				}
				for _, path := range admitted {
					normalized, err := parallel.NormalizePath(path)
					if err != nil || normalized != path {
						return fmt.Errorf("retained writer %s/%s has invalid admitted paths for %s", slug, name, retainedID)
					}
				}
				counted, matchingPaths := false, false
				for _, row := range lease.Slices {
					workspace := row.Workspace
					if workspace == "" {
						workspace = controller
					}
					if workspace == slug && sliceNumber(row.ID) == sliceNumber(retainedID) && row.WrightStatus != parallel.WrightIntegrated {
						counted = true
						matchingPaths = frontierSamePaths(admitted, row.Paths)
						break
					}
				}
				if !counted {
					return fmt.Errorf("retained writer %s/%s lost its active lease reservation for %s; reconcile the native actor before admitting more work", slug, name, retainedID)
				}
				if !matchingPaths {
					return fmt.Errorf("retained writer %s/%s admitted paths for %s no longer match its active lease reservation; mutable tasks cannot move actor authority", slug, name, retainedID)
				}
				if slug == source && containsSlice(ids, retainedID) {
					return fmt.Errorf("retained writer %s/%s still owns %s", slug, name, retainedID)
				}
			}
		}
	}
	return nil
}

// The binding records admission into an existing cohort, never an actor's cwd/profile.
func autocompleteWriterAdmission(root, source string, wave *dispatchWave, current *dispatchFile) (string, []string, error) {
	mode, err := autocompleteMode(root)
	if err != nil || !mode {
		return "", nil, err
	}
	allow, capN, err := frontierPolicy(root)
	if err != nil {
		return "", nil, err
	}
	controller, lease, err := parallel.ControllerLease(filepath.Dir(root))
	if err != nil {
		return "", nil, err
	}
	if lease != nil {
		effectiveCap, occupied := capN, 0
		if lease.Capacity > 0 && lease.Capacity < effectiveCap {
			effectiveCap = lease.Capacity
		}
		if lease.Status == parallel.StatusRunning || lease.Status == parallel.StatusIntegrateFailed {
			for _, row := range lease.Slices {
				if row.WrightStatus != parallel.WrightIntegrated {
					occupied++
				}
			}
		}
		if occupied > effectiveCap {
			return "", nil, fmt.Errorf("retained cohort has %d occupied reservations above effective cap %d; reconcile capacity and drain existing work before any new writer launch; do not cancel workers", occupied, effectiveCap)
		}
		for i := range lease.Slices {
			if lease.Slices[i].WrightStatus != parallel.WrightIntegrated {
				if err := frontierNormalizePaths(lease.Slices[i].Paths); err != nil {
					return "", nil, err
				}
			}
		}
	}
	if capN <= 1 && lease == nil {
		return "", nil, nil
	}
	ids := wave.Slices
	nodes, err := frontierScan(root)
	if err != nil {
		return "", nil, err
	}
	chain, err := frontierChain(nodes, source)
	if err != nil {
		return "", nil, err
	}
	ordered, related := frontierRelated(nodes, chain)
	mapping, _, problem, invalidCanonical := frontierRoadmap(nodes, ordered)
	if invalidCanonical {
		return "", nil, fmt.Errorf("invalid canonical roadmap: %s", problem)
	}
	if controller == "" {
		for _, slug := range ordered {
			if nodes[slug].location == "live" {
				controller = slug
				break
			}
		}
	} else if !related[controller] {
		return "", nil, fmt.Errorf("retained repository controller %s is outside source sequence", controller)
	}
	records, err := frontierEvidence(root, source)
	if err != nil {
		return "", nil, err
	}
	notices, err := frontierPreparationAdmission(root, source, nodes, mapping, problem, allow && capN > 1, records)
	if err != nil {
		return "", nil, err
	}
	local, withheld, err := WorkspaceReady(filepath.Dir(root), controller, source)
	if err != nil {
		return "", nil, err
	}
	if len(local) == 0 && len(withheld) > 0 {
		return "", nil, fmt.Errorf("source readiness: %s", strings.Join(withheld, "; "))
	}
	for _, notice := range withheld {
		notices = append(notices, "frontier-readiness: "+notice)
	}
	for _, id := range ids {
		eligible := false
		for _, candidate := range local {
			eligible = eligible || sliceNumber(candidate.ID) == sliceNumber(id)
		}
		if !eligible {
			return "", nil, fmt.Errorf("source readiness: %s/%s is not eligible; %s", source, id, strings.Join(withheld, "; "))
		}
	}
	for i := range local {
		if err := frontierNormalizePaths(local[i].Paths); err != nil {
			return "", nil, err
		}
	}
	binding := ""
	reserved := frontierLeaseReserved(lease, controller, source, ids, local)
	if reserved {
		binding = controller + "/" + lease.BatchID + "@" + lease.CreatedAt
	}
	if err := frontierWriterRetention(root, source, controller, binding, lease, nodes, ordered, ids, current); err != nil {
		return "", nil, err
	}
	var candidates []parallel.SlicePaths
	for _, slug := range ordered {
		if slug == source {
			candidates = append(candidates, local...)
			continue
		}
		if nodes[slug].location != "live" {
			continue
		}
		n := nodes[slug]
		if hold, err := frontierQuestionHold(n); err != nil {
			return "", nil, err
		} else if hold != "" {
			continue
		}
		if _, _, err := frontierLocal(root, controller, n, &candidates); err != nil {
			return "", nil, err
		}
	}
	filtered := candidates[:0]
	for _, candidate := range candidates {
		workspace := candidate.Workspace
		if workspace == "" {
			workspace = controller
		}
		if workspace != source {
			if reason := frontierSerialReason(records, source, workspace, candidate.ID); reason != "" {
				notices = append(notices, "frontier-serial: "+workspace+"/"+candidate.ID+" — "+reason+"; ordered handoff retained")
				continue
			}
		}
		filtered = append(filtered, candidate)
	}
	_, picks, _, err := parallel.SelectForSlug(filepath.Dir(root), controller, capN, filtered)
	if err != nil {
		return "", nil, err
	}
	for _, id := range ids {
		selected := reserved
		for _, pick := range picks {
			workspace := pick.Workspace
			if workspace == "" {
				workspace = controller
			}
			selected = selected || workspace == source && sliceNumber(pick.ID) == sliceNumber(id)
		}
		if !selected || source != controller && !reserved {
			return "", nil, fmt.Errorf("%s/%s is not admitted by global lease capacity/path/readiness selection; run frontier and reserve the mixed cohort with parallel create/add before foreign dispatch", source, id)
		}
	}
	var missing []string
	for _, pick := range picks {
		workspace := pick.Workspace
		if workspace == "" {
			workspace = controller
		}
		if workspace == source && containsSlice(ids, pick.ID) {
			continue
		}
		if frontierSerialReason(records, source, workspace, pick.ID) != "" {
			continue
		}
		missing = append(missing, parallel.SliceKey(pick.Workspace, pick.ID))
	}
	if len(missing) != 0 {
		return "", nil, fmt.Errorf("global picks %s omitted; run devrites-engine parallel select --root %s --slug %s --from-plan --workspaces %s --cap %d and reserve picks with parallel create/add, or record exact serial: <source/SLICE-ID> — <constraint>", strings.Join(missing, ","), frontierQuote(filepath.Dir(root)), frontierQuote(controller), frontierQuote(strings.Join(ordered, ",")), capN)
	}
	wave.AdmittedPaths = make(map[string][]string, len(ids))
	for _, id := range ids {
		for _, candidate := range local {
			if sliceNumber(candidate.ID) == sliceNumber(id) {
				wave.AdmittedPaths[id] = candidate.Paths
				break
			}
		}
	}
	return binding, notices, nil
}

func frontierGlobalPriority(source string, lines []string) bool {
	controller := source
	for _, line := range lines {
		if strings.HasPrefix(line, "frontier-select-command:") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "--slug" && i+1 < len(fields) {
					controller = strings.Trim(fields[i+1], "'")
				}
			}
		}
		if strings.HasPrefix(line, "frontier-prepare:") || strings.HasPrefix(line, "frontier-review-sequence:") {
			return true
		}
		if strings.HasPrefix(line, "frontier-workspace:") {
			fields := strings.Fields(line)
			if len(fields) > 3 && fields[1] != source && (fields[3] == "action=repair" || fields[3] == "action=inspect" || fields[3] == "action=prepare") {
				return true
			}
		}
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "frontier-picks: ") {
			continue
		}
		for _, key := range strings.Split(strings.TrimPrefix(line, "frontier-picks: "), ",") {
			workspace, _, qualified := strings.Cut(key, "/")
			if !qualified {
				workspace = controller
			}
			if workspace != source {
				return true
			}
		}
	}
	return false
}

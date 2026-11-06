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
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/workflow"
)

const frontierWorkspaceLimit = 512

type frontierWorkspace struct {
	slug, dir, location, parent, phase, status, next, question string
	position, remaining                                        int
	hasPosition, hasRemaining                                  bool
	selectionHeld                                              bool
	problem                                                    error
}

// RunFrontier observes the recorded sequence without creating or selecting work.
func RunFrontier(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 || len(args) == 1 && strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(stderr, "usage: devrites-engine frontier [slug]")
		return 2
	}
	slug := ""
	if len(args) == 1 {
		slug = args[0]
	}
	lines, err := FrontierLines(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "frontier: %v\n", err)
		return 2
	}
	for _, line := range lines {
		fmt.Fprintln(stdout, line)
	}
	return 0
}

// FrontierNotices preserves source-local selection; callers must explicitly opt into the pool.
func FrontierNotices(repoRoot, slug string) ([]string, error) {
	return FrontierLines(filepath.Join(repoRoot, devritespaths.DevritesRootName), slug)
}

func frontierField(lines []string, key string) (string, bool, error) {
	value, present := "", false
	for _, line := range lines {
		if v, ok := state.CursorField([]string{line}, key); ok {
			if present {
				return value, true, fmt.Errorf("duplicate %s cursor", key)
			}
			value, present = v, true
		}
	}
	return value, present, nil
}

func frontierRead(root, slug, location string) *frontierWorkspace {
	n := &frontierWorkspace{slug: slug, location: location}
	if location == "live" {
		n.dir, n.problem = devritespaths.ExistingCanonicalFeatureDirChecked(root, slug)
	} else {
		n.dir, n.problem = devritespaths.ExistingArchivedFeatureDirChecked(root, slug)
	}
	if n.problem != nil {
		return n
	}
	raw, err := readBoundedRegularFile(filepath.Join(n.dir, "state.md"), workspaceArtifactLimit)
	if err != nil {
		n.problem = err
		return n
	}
	raw, err = markdowntext.Structural(raw)
	if err != nil {
		n.problem = err
		return n
	}
	lines := strings.Split(string(raw), "\n")
	for _, field := range []struct {
		key string
		dst *string
	}{
		{state.CursorSequenceParent, &n.parent}, {state.CursorPhase, &n.phase}, {state.CursorStatus, &n.status}, {state.CursorNextAction, &n.next}, {state.CursorQuestionID, &n.question},
	} {
		value, _, err := frontierField(lines, field.key)
		*field.dst = value
		if err != nil {
			n.problem = err
		}
	}
	for _, field := range []struct {
		key      string
		dst      *int
		present  *bool
		positive bool
	}{
		{state.CursorSequencePosition, &n.position, &n.hasPosition, true},
		{state.CursorSequenceWorkspacesRemain, &n.remaining, &n.hasRemaining, false},
	} {
		value, present, err := frontierField(lines, field.key)
		*field.present = present
		if err != nil {
			n.problem = err
			continue
		}
		if !present {
			continue
		}
		number, err := strconv.Atoi(value)
		if err != nil || number < 0 || field.positive && number == 0 || strconv.Itoa(number) != value {
			n.problem = fmt.Errorf("%s must be a canonical %sdecimal", field.key, map[bool]string{true: "positive ", false: "nonnegative "}[field.positive])
			continue
		}
		*field.dst = number
	}
	if n.parent != "" && !devritespaths.ValidSlug(n.parent) {
		n.problem = fmt.Errorf("invalid sequence_parent")
	}
	if _, known := state.PolicyFor(state.Phase(n.phase)); !known && n.problem == nil {
		n.problem = fmt.Errorf("unknown or missing native phase %q", n.phase)
	}
	return n
}

func frontierScan(root string) (map[string]*frontierWorkspace, error) {
	nodes := map[string]*frontierWorkspace{}
	for _, location := range []string{"work", "archive"} {
		path := filepath.Join(root, location)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s must be a canonical directory", location)
		}
		f, err := os.Open(path) // #nosec G304 -- canonical directory checked above
		if err != nil {
			return nil, err
		}
		entries, err := f.ReadDir(frontierWorkspaceLimit + 1)
		_ = f.Close()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if len(entries)+len(nodes) > frontierWorkspaceLimit {
			return nil, fmt.Errorf("frontier exceeds %d workspace entries; native sequence review required", frontierWorkspaceLimit)
		}
		for _, entry := range entries {
			if !entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			slug := entry.Name()
			where := "live"
			if location == "archive" {
				where = "archive"
			}
			n := frontierRead(root, slug, where)
			if old := nodes[slug]; old != nil {
				old.problem = fmt.Errorf("duplicate live/archive identity")
				continue
			}
			nodes[slug] = n
		}
	}
	return nodes, nil
}

func frontierChain(nodes map[string]*frontierWorkspace, slug string) ([]string, error) {
	var chain []string
	seen := map[string]bool{}
	for len(chain) < maxSequenceChain {
		if seen[slug] {
			return chain, fmt.Errorf("sequence_parent cycle at %s", slug)
		}
		seen[slug] = true
		n := nodes[slug]
		if n == nil {
			return chain, fmt.Errorf("sequence_parent %s missing", slug)
		}
		chain = append(chain, slug)
		if n.problem != nil {
			return chain, fmt.Errorf("%s: %w", slug, n.problem)
		}
		if n.parent == "" {
			if n.hasPosition && n.position != 1 {
				return chain, fmt.Errorf("%s position %d has no sequence_parent", slug, n.position)
			}
			return chain, nil
		}
		if p := nodes[n.parent]; p != nil && (n.hasPosition || p.hasPosition) && (!n.hasPosition || !p.hasPosition || n.position != p.position+1) {
			return chain, fmt.Errorf("%s sequence_position does not follow %s", slug, n.parent)
		}
		slug = n.parent
	}
	return chain, fmt.Errorf("sequence_parent chain exceeds %d workspaces", maxSequenceChain)
}

func frontierSpecTarget(n *frontierWorkspace) string {
	command := workflow.ForAction(n.next)
	if command.Verb != "spec" {
		return ""
	}
	fields := strings.Fields(n.next)
	if len(fields) < 2 || fields[0] != command.Claude && fields[0] != command.Codex && fields[0] != command.Skill {
		return ""
	}
	if workflow.ForAction(strings.Join(fields[1:], " ")).Verb != "" {
		return ""
	}
	for i, field := range fields {
		if field == "—" {
			fields = fields[:i]
			break
		}
	}
	if len(fields) != 2 || !devritespaths.ValidSlug(fields[1]) || strings.HasPrefix(fields[1], "-") || fields[1] == n.slug {
		return ""
	}
	return fields[1]
}

func frontierTerminal(n *frontierWorkspace) bool {
	return n.phase == "done" || n.status == "shipped" || n.status == "done" || n.status == "archived"
}

func frontierQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// FrontierLines reports structural obligations, never semantic authorization or Ship approval.
func FrontierLines(root, slug string) ([]string, error) {
	mode, err := autocompleteMode(root)
	if err != nil {
		return nil, err
	}
	if slug == "" {
		var err error
		slug, err = devritespaths.ActiveSlug(root)
		if err != nil {
			return nil, err
		}
	}
	workspaceOverride := ""
	if slug != "" {
		if !devritespaths.ValidSlug(slug) {
			return nil, fmt.Errorf("invalid source slug")
		}
		workspaceOverride, err = devritespaths.WorkspaceOverrideChecked(root, slug)
		if err != nil {
			return nil, err
		}
	}
	nodes, err := frontierScan(root)
	if err != nil {
		return nil, err
	}
	if slug == "" {
		var tips []string
		for name, n := range nodes {
			if n.location != "archive" || !frontierTerminal(n) || frontierSpecTarget(n) == "" {
				continue
			}
			if _, err := frontierChain(nodes, name); err != nil {
				continue
			}
			hasChild := false
			for _, child := range nodes {
				hasChild = hasChild || child.parent == name
			}
			if !hasChild {
				tips = append(tips, name)
			}
		}
		if len(tips) != 1 {
			return nil, fmt.Errorf("empty ACTIVE: need a unique archived terminal sequence tip with exact next Spec edge; found %d", len(tips))
		}
		slug = tips[0]
	}
	source := nodes[slug]
	if source == nil {
		return nil, fmt.Errorf("source %s missing", slug)
	}
	chain, chainErr := frontierChain(nodes, slug)
	if chainErr != nil {
		location := "work"
		if source.location == "archive" {
			location = "archive"
		}
		artifact := filepath.Join(root, location, slug, "state.md")
		return []string{
			"frontier: blocked", "frontier-source: " + slug,
			fmt.Sprintf("frontier-workspace: %s location=%s action=repair reason=%s", slug, source.location, chainErr),
			"frontier-review-sequence: reason=native authority-normalization required: " + chainErr.Error() + " artifacts=" + artifact,
		}, nil
	}
	ordered, related := frontierRelated(nodes, chain)
	if len(related) == 1 && source.parent == "" && !source.hasPosition && !source.hasRemaining && frontierSpecTarget(source) == "" {
		return []string{"frontier: none"}, nil
	}
	controller := ""
	for _, name := range ordered {
		if nodes[name].location == "live" {
			if _, err := frontierChain(nodes, name); err == nil {
				controller = name
				break
			}
		}
	}
	retainedController, lease, err := parallel.ControllerLease(filepath.Dir(root))
	if err != nil {
		return []string{"frontier: blocked", "frontier-source: " + slug, "frontier-selection: blocked reason=" + err.Error()}, nil
	}
	controllerProblem := ""
	if retainedController != "" {
		controller = retainedController
		if !related[controller] || nodes[controller].location != "live" {
			controllerProblem = "repository controller " + controller + " is outside the source sequence; recover retained controller before admission"
		}
	}
	allowFuture, capN, err := frontierPolicy(root)
	if err != nil {
		return nil, err
	}
	if workspaceOverride != "" {
		allowFuture = false
	}
	if lease != nil && lease.Capacity > 0 && capN > lease.Capacity {
		capN = lease.Capacity
	}
	lines := []string{"", "frontier-source: " + slug}
	actionable, waiting, blocked := false, false, false
	if controllerProblem != "" {
		lines = append(lines, "frontier-selection: blocked reason="+controllerProblem)
		blocked = true
	}
	var live []string
	var candidates []parallel.SlicePaths
	if source.hasRemaining && source.remaining == 0 {
		allowFuture = false
	}
	maps, artifacts, mapProblem, invalidCanonical := frontierRoadmap(nodes, ordered)
	if invalidCanonical {
		return []string{"frontier: blocked", "frontier-source: " + slug, "frontier-review-sequence: reason=native authority-normalization required: invalid canonical authority: " + mapProblem + " artifacts=" + strings.Join(artifacts, ",")}, nil
	}
	if len(maps) > 0 && mapProblem == "" {
		lines = append(lines, "frontier-authority: "+strings.Join(artifacts, ","))
	}
	records := frontierRecords{}
	if mode {
		records, err = frontierEvidence(root, slug)
		if err != nil {
			return nil, err
		}
	}
	sequenceDeferred := mode && records.blocked["sequence"] != ""
	if mapProblem != "" {
		if sequenceDeferred {
			lines = append(lines, "frontier-blocked: sequence — "+records.blocked["sequence"]+"; scheduling deferred, not normalized")
		} else {
			lines = append(lines, "frontier-review-sequence: reason=native authority-normalization required: "+mapProblem+" artifacts="+strings.Join(artifacts, ","))
			blocked = true
		}
	}
	deferred := map[string]string{}
	prepared := map[string]bool{}
	var preparatory map[string]bool
	var preparationAccounts map[string]*dispatchFile
	if mode && capN > 1 {
		preparatory = make(map[string]bool, len(ordered))
		preparationAccounts = make(map[string]*dispatchFile, len(ordered))
		for _, name := range ordered {
			preparatory[name], preparationAccounts[name], err = frontierPreparatoryOnly(root, nodes[name], nodes, maps, mapProblem)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, name := range ordered {
		n := nodes[name]
		action, reason := "complete", "archived predecessor; no writer admission"
		held := false
		_, invalid := frontierChain(nodes, name)
		if invalid != nil {
			action, reason, blocked = "repair", invalid.Error(), true
		} else if gateReason, gateErr := frontierQuestionHold(n); gateErr != nil {
			action, reason, held, blocked = "repair", gateErr.Error(), true, true
		} else if gateReason != "" {
			action, reason, held, blocked = "wait", gateReason, true, true
		} else if n.location == "live" {
			action, reason, err = frontierLocal(root, controller, n, &candidates, preparationAccounts[name])
			if err != nil {
				action, reason, n.selectionHeld = "repair", err.Error(), true
			}
			if mode && action == "prepare" && deferred[name] != "" {
				action, reason, n.selectionHeld = "wait", "frontier-blocked: "+name+" — "+deferred[name], true
			}
			inheritedPreparation := preparatory[name] && preparatory[n.parent] && (n.phase == "frame" || preparationPhase(n.phase)) && action == "prepare"
			if inheritedPreparation {
				action, reason, n.selectionHeld = "wait", frontierInheritedPreparationWait, true
			}
			if !n.selectionHeld {
				live = append(live, name)
			}
			held = inheritedPreparation || action == "blocked" || err != nil || mode && action == "wait" && deferred[name] != ""
			switch action {
			case "inspect", "prepare", "repair":
				actionable = true
			case "wait":
				waiting = true
			case "no-cursor":
				blocked = true
				action = "wait"
			case "blocked":
				blocked = true
				action = "wait"
			}
		}
		lines = append(lines, fmt.Sprintf("frontier-workspace: %s location=%s action=%s reason=%s", name, n.location, action, reason))
		if preparatory[name] {
			continue
		}
		if invalid != nil || held || !allowFuture || n.hasRemaining && n.remaining == 0 || n.status == "awaiting_human" || n.status == "blocked" || n.question != "" {
			continue
		}
		target := frontierSpecTarget(n)
		if target == "" && n.hasRemaining && n.remaining > 0 && n.hasPosition && mapProblem == "" {
			target = maps[n.position+1]
		}
		if mapProblem != "" {
			if !sequenceDeferred && (target != "" || n.hasRemaining && n.remaining > 0) {
				actionable = true
			}
			continue
		}
		if target != "" && frontierConflictingSuccessor(nodes, n, target) {
			lines = append(lines, "frontier-review-sequence: reason=conflicting successor identities; native authority-normalization required source="+name+" artifacts="+strings.Join(artifacts, ","))
			actionable, blocked = true, true
			continue
		}
		if target != "" && n.hasPosition && maps[n.position+1] != "" && maps[n.position+1] != target {
			lines = append(lines, "frontier-review-sequence: reason=next Spec conflicts with recorded successor; native authority-normalization required source="+name+" artifacts="+strings.Join(artifacts, ","))
			actionable, blocked = true, true
			continue
		}
		if target != "" && nodes[target] != nil && nodes[target].parent != name {
			lines = append(lines, fmt.Sprintf("frontier-workspace: %s location=%s action=repair reason=recorded successor is not bound to source %s; native authority-normalization required", target, nodes[target].location, name))
			actionable, blocked = true, true
			continue
		}
		if mode && target != "" {
			sourceRecords := records
			if name != slug {
				sourceRecords, err = frontierEvidence(root, name)
				if err != nil {
					return nil, err
				}
			}
			if reason := sourceRecords.blocked[target]; reason != "" {
				deferred[target] = reason
				lines = append(lines, "frontier-blocked: "+target+" source="+name+" — "+reason+"; preparation deferred, not approved")
				continue
			}
		}
		if target != "" && nodes[target] == nil && !prepared[target] {
			prepared[target] = true
			// A table identifies a preparation obligation, not approval to implement it.
			lines = append(lines, fmt.Sprintf("frontier-workspace: %s location=missing action=prepare reason=recorded next workspace; native scope and independence review required", target), fmt.Sprintf("frontier-prepare: /rite-spec %s source=%s reason=read recorded sequence authority before preparation; no workspace created", frontierQuote(target), name))
			actionable = true
		} else if target == "" && n.hasRemaining && n.remaining > 0 {
			lines = append(lines, "frontier-review-sequence: reason=remaining work unresolved; native authority-normalization required source="+name+" artifacts="+strings.Join(artifacts, ","))
			actionable, blocked = true, true
		}
	}
	if mode {
		filtered := candidates[:0]
		for _, candidate := range candidates {
			workspace := candidate.Workspace
			if workspace == "" {
				workspace = controller
			}
			if workspace != slug {
				if reason := frontierSerialReason(records, slug, workspace, candidate.ID); reason != "" {
					lines = append(lines, "frontier-serial: "+workspace+"/"+candidate.ID+" — "+reason+"; ordered handoff retained")
					continue
				}
			}
			filtered = append(filtered, candidate)
		}
		candidates = filtered
	}
	if controller != "" && controllerProblem == "" && len(live) > 0 {
		lines = append(lines, fmt.Sprintf("frontier-select-command: devrites-engine parallel select --root %s --slug %s --from-plan --workspaces %s --cap %d", frontierQuote(filepath.Dir(root)), frontierQuote(controller), frontierQuote(strings.Join(live, ",")), capN))
		if len(candidates) > 0 {
			_, picks, running, err := parallel.SelectForSlug(filepath.Dir(root), controller, capN, candidates)
			if err != nil {
				lines = append(lines, "frontier-selection: blocked reason="+err.Error())
				blocked = true
			} else if len(picks) > 0 {
				actionable = true
				var ids []string
				for _, pick := range picks {
					ids = append(ids, parallel.SliceKey(pick.Workspace, pick.ID))
				}
				lines = append(lines, "frontier-picks: "+strings.Join(ids, ","))
			} else if running > 0 {
				waiting = true
			}
		}
	}
	verdict := "clear"
	if blocked {
		verdict = "blocked"
	}
	if waiting {
		verdict = "waiting"
	}
	if actionable {
		verdict = "actionable"
	}
	lines[0] = "frontier: " + verdict
	return lines, nil
}

func frontierPolicy(root string) (bool, int, error) {
	capN := parallel.MaxParallelSlices
	raw, err := readBoundedRegularFile(filepath.Join(root, "AFK"), workspaceArtifactLimit)
	if os.IsNotExist(err) {
		return true, capN, nil
	}
	if err != nil {
		return false, 0, err
	}
	seen := map[string]bool{}
	allow := true
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !ok || key != "continue_sequence" && key != "max_parallel" {
			continue
		}
		for i := 1; i < len(value); i++ {
			if value[i] == '#' && (value[i-1] == ' ' || value[i-1] == '\t') {
				value = strings.TrimSpace(value[:i])
				break
			}
		}
		if seen[key] {
			return false, 0, fmt.Errorf("duplicate AFK %s", key)
		}
		seen[key] = true
		if key == "continue_sequence" {
			switch value {
			case "true":
			case "false":
				allow = false
			default:
				return false, 0, fmt.Errorf("invalid AFK continue_sequence")
			}
		} else {
			capN, err = strconv.Atoi(value)
			if err != nil || capN < 1 || capN > parallel.MaxParallelSlices || strconv.Itoa(capN) != value {
				return false, 0, fmt.Errorf("invalid AFK max_parallel")
			}
		}
	}
	return allow, capN, nil
}

func frontierQuestionHold(n *frontierWorkspace) (string, error) {
	if n.status == "awaiting_human" || n.status == "blocked" || n.question != "" {
		return "human/blocked cursor; retain native gate", nil
	}
	questions, err := readBoundedRegularFile(filepath.Join(n.dir, "questions.md"), workspaceArtifactLimit)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	for _, kind := range gate.OpenBlockingQuestionGates(questions) {
		if kind == "blocking" || kind == "escalating" {
			return "open " + kind + " question; retain native gate", nil
		}
	}
	return "", nil
}

func frontierLocal(root, controller string, n *frontierWorkspace, candidates *[]parallel.SlicePaths, dispatch *dispatchFile) (string, string, error) {
	n.selectionHeld = true
	if frontierTerminal(n) {
		return "complete", "terminal workspace", nil
	}
	if _, err := devritespaths.WorkspaceOverrideChecked(root, n.slug); err != nil {
		return "blocked", "DEVRITES_WORKSPACE scope retained; unset override before global admission", nil
	}
	if n.status != "running" && n.status != "ready" {
		return "blocked", "status lacks an actionable native cursor; normalize state without inventing authority", nil
	}
	if dispatch == nil {
		raw, err := readBoundedRegularFile(filepath.Join(n.dir, "dispatch.json"), workspaceArtifactLimit)
		if err != nil && !os.IsNotExist(err) {
			return "", "", err
		}
		if err == nil {
			dispatch, err = decodeDispatch(raw)
			if err != nil {
				return "", "", err
			}
		}
	}
	if dispatch != nil {
		var err error
		retained, pending := false, false
		var built map[string]bool
		for _, wave := range dispatch.Waves {
			if wave.State == "complete" {
				if !isBuildWrightWave(wave) {
					continue
				}
				if built == nil {
					built, err = builtSlicesFrom(func(name string) ([]byte, error) {
						return readBoundedRegularFile(filepath.Join(n.dir, name), workspaceArtifactLimit)
					})
					if err != nil {
						return "", "", err
					}
				}
				if len(wave.Slices) > 0 && !slicesUnbuilt(wave.Slices, built) {
					continue
				}
			}
			if wave.State == "abandoned" && wave.TerminalAt != "" {
				continue
			}
			wavePending := false
			for _, role := range wave.Roles {
				if wave.State != "abandoned" || role.Handle != "" || role.StartedAt != "" {
					wavePending = wavePending || !role.Returned
				}
			}
			if wave.State == "abandoned" && !wavePending {
				continue
			}
			retained = true
			pending = pending || wavePending
		}
		if pending {
			return "wait", "retained dispatch work; reconcile handles before another launch; isolated lease IDs do not bind actor checkout", nil
		}
		if retained {
			return "inspect", "retained dispatch returns need native reconciliation before another launch", nil
		}
	}
	n.selectionHeld = false
	if n.phase == "vet" && n.status == "ready" || n.phase == "build" {
		ready, notices, err := WorkspaceReady(filepath.Dir(root), controller, n.slug)
		if err != nil {
			return "", "", err
		}
		*candidates = append(*candidates, ready...)
		if len(notices) > 0 {
			action := "wait"
			for _, notice := range notices {
				if !strings.Contains(notice, ": waiting for durable BUILT/integrated producer ") {
					action = "repair"
				}
			}
			return action, "native Plan/Vet readiness: " + strings.Join(notices, "; "), nil
		}
		return "wait", "writer candidates require mixed selection; retained leases and capacity remain authoritative", nil
	}
	if frontierSpecTarget(n) != "" {
		return "complete", "native next Spec cursor recorded", nil
	}
	if _, internal, err := internalPhaseEdge(n.next, n.slug); err != nil {
		return "", "", err
	} else if internal && n.status == "running" {
		return "prepare", "native internal phase cursor: " + n.next, nil
	}
	if command := workflow.ForAction(n.next); command.Verb == "ship" {
		fields := strings.Fields(n.next)
		valid := len(fields) > 0 && (fields[0] == command.Claude || fields[0] == command.Codex || fields[0] == command.Skill)
		valid = valid && workflow.ForAction(strings.Join(fields[1:], " ")).Verb == ""
		for i, field := range fields {
			if field == "—" {
				fields = fields[:i]
				break
			}
		}
		valid = valid && (len(fields) == 1 || len(fields) == 2 && fields[1] == n.slug)
		if !valid {
			return "blocked", "invalid native Ship cursor; retain native gate", nil
		}
		// Ship approval holds this writer, not independently recorded future preparation.
		n.selectionHeld = true
		return "wait", "Ship requires existing native authority; controller invocation is not approval", nil
	}
	if n.next == "" {
		// Missing local work is not a hold on independently recorded preparation.
		n.selectionHeld = true
		return "no-cursor", "no actionable native internal cursor; review state without inventing authority", nil
	}
	return "blocked", "no actionable native internal cursor; review state without inventing authority", nil
}

// Current canonical authority supersedes historical tables without rewriting them.
// Without it, differing exact legacy maps require native normalization.
func frontierRoadmap(nodes map[string]*frontierWorkspace, ordered []string) (map[int]string, []string, string, bool) {
	if mapping, artifacts, err := frontierCanonicalRoadmap(nodes, ordered); err != nil {
		return nil, artifacts, err.Error(), true
	} else if mapping != nil {
		return mapping, artifacts, "", false
	}
	result := map[int]string{}
	var artifacts []string
	problem := ""
	var first map[int]string
	for _, slug := range ordered {
		for _, name := range []string{"decisions.md", "roadmap.md", "spec.md"} {
			path := filepath.Join(nodes[slug].dir, name)
			if nodes[slug].dir == "" {
				continue
			}
			raw, err := readBoundedRegularFile(path, workspaceArtifactLimit)
			if os.IsNotExist(err) {
				continue
			}
			artifacts = append(artifacts, path)
			if err != nil {
				problem = "recorded artifact unreadable"
				continue
			}
			raw, err = markdowntext.Structural(raw)
			if err != nil {
				problem = "recorded artifact malformed"
				continue
			}
			lines := strings.Split(string(raw), "\n")
			for i := 0; i < len(lines); i++ {
				cells := frontierCells(lines[i])
				pos, workspace, headings := -1, -1, 0
				for j, cell := range cells {
					switch strings.ToLower(cell) {
					case "position":
						pos = j
						headings++
					case "slug", "workspace":
						workspace = j
						headings++
					}
				}
				if pos < 0 || workspace < 0 {
					continue
				}
				if headings != 2 {
					problem = "ambiguous position/workspace headers"
					continue
				}
				table := map[int]string{}
				seen := map[string]bool{}
				for i++; i < len(lines); i++ {
					row := frontierCells(lines[i])
					if len(row) != len(cells) {
						break
					}
					if strings.Trim(row[pos], "-: ") == "" {
						continue
					}
					n, err := strconv.Atoi(row[pos])
					target := strings.Trim(row[workspace], "`")
					if err != nil || n < 1 || strconv.Itoa(n) != row[pos] || !devritespaths.ValidSlug(target) || strings.ContainsAny(target, " \t") || strings.HasPrefix(target, "-") {
						problem = "ambiguous position/workspace table"
						continue
					}
					if table[n] != "" || seen[target] {
						problem = "duplicate sequence position or workspace"
					}
					table[n], seen[target] = target, true
				}
				if len(table) == 0 {
					problem = "empty position/workspace table"
					continue
				}
				if first == nil {
					first = table
				} else {
					if len(first) != len(table) {
						problem = "conflicting or amended sequence maps"
					}
					for n, target := range table {
						if first[n] != target {
							problem = "conflicting or amended sequence maps"
						}
					}
				}
				for n, target := range table {
					result[n] = target
				}
			}
		}
	}
	for _, slug := range ordered {
		n := nodes[slug]
		if n.hasPosition && result[n.position] != "" && result[n.position] != slug {
			problem = "live cursor conflicts with recorded sequence map"
		}
	}
	if len(artifacts) == 0 {
		for _, slug := range ordered {
			artifacts = append(artifacts, filepath.Join(nodes[slug].dir, "state.md"))
		}
	}
	return result, artifacts, problem, false
}

func frontierCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	cells := strings.Split(strings.Trim(line, "|"), "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func frontierBoundSpec(root, slug, action string) bool {
	nodes, err := frontierScan(root)
	if err != nil {
		return false
	}
	n := nodes[slug]
	if n == nil || n.next != action || n.parent == "" && !n.hasPosition && !n.hasRemaining {
		return false
	}
	target := frontierSpecTarget(n)
	if target == "" {
		return false
	}
	chain, err := frontierChain(nodes, slug)
	if frontierConflictingSuccessor(nodes, n, target) {
		return false
	}
	if err != nil {
		return false
	}
	allow, _, err := frontierPolicy(root)
	if err != nil {
		return false
	}
	ordered, _ := frontierRelated(nodes, chain)
	mapping, _, problem, invalidCanonical := frontierRoadmap(nodes, ordered)
	if invalidCanonical {
		return false
	}
	// Existing checked children remain resumable under ambiguous historical maps.
	if child := nodes[target]; child != nil {
		if child.parent != slug {
			return false
		}
		if _, err := frontierChain(nodes, target); err != nil {
			return false
		}
		return true
	}
	if problem != "" || n.hasPosition && mapping[n.position+1] != "" && mapping[n.position+1] != target {
		return false
	}
	return allow && (!n.hasRemaining || n.remaining > 0)
}

func frontierConflictingSuccessor(nodes map[string]*frontierWorkspace, source *frontierWorkspace, target string) bool {
	for _, child := range nodes {
		if child.parent == source.slug && child.slug != target && source.hasPosition && child.hasPosition && child.position == source.position+1 {
			return true
		}
	}
	return false
}

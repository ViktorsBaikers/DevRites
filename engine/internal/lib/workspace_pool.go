package lib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/state"
)

// WorkspaceReady admits only the controller or its recorded forward continuation.
// Notices describe withheld work without removing other workspaces from the pool.
func WorkspaceReady(repoRoot, controller, target string) ([]parallel.SlicePaths, []string, error) {
	root := filepath.Join(repoRoot, devritespaths.DevritesRootName)
	if !devritespaths.ValidSlug(controller) || !devritespaths.ValidSlug(target) {
		return nil, nil, fmt.Errorf("invalid controller or target workspace slug")
	}
	if _, err := devritespaths.WorkspaceOverrideChecked(root, target); err != nil {
		return nil, []string{fmt.Sprintf("%s: global workspace admission preserves DEVRITES_WORKSPACE scope: %v; unset DEVRITES_WORKSPACE to query the global pool", target, err)}, nil
	}
	if _, err := devritespaths.ExistingFeatureDirChecked(root, controller); err != nil {
		return nil, nil, fmt.Errorf("controller %s: %w", controller, err)
	}
	observation, err := state.ObserveWorkspace(root, target)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: state unavailable: %v; repair workspace and rerun /rite-vet", target, err)}, nil
	}
	cursor, _ := observation.Fact(state.LedgerFile)
	cursorBytes := cursor.Bytes()
	if target != controller {
		if err := forwardWorkspace(root, controller, target, cursorBytes); err != nil {
			return nil, []string{fmt.Sprintf("%s: %v", target, err)}, nil
		}
	}
	result, err := gate.CheckObserved(gate.Readiness, root, observation)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: readiness unavailable: %v; repair workspace and rerun /rite-vet", target, err)}, nil
	}
	structuralCursor, _ := markdowntext.Structural(cursorBytes)
	status, _ := state.CursorField(strings.Split(string(structuralCursor), "\n"), state.CursorStatus)
	var notices []string
	if result.Phase != state.PhaseVet && result.Phase != state.PhaseBuild {
		notices = append(notices, fmt.Sprintf("%s: phase %s is not Build entry; native Plan/Vet must prepare this workspace before admission", target, result.Phase))
	}
	if result.Phase == state.PhaseVet && status != "ready" {
		notices = append(notices, fmt.Sprintf("%s: Vet is not ready; native Plan/Vet must complete before admission", target))
	}
	switch status {
	case "awaiting_human", "blocked", "shipped", "done", "archived":
		notices = append(notices, fmt.Sprintf("%s: status %s withholds writer admission; native controller must resolve this state", target, status))
	}
	if result.Blocked {
		for _, line := range strings.Split(strings.TrimSpace(result.Render()), "\n") {
			notices = append(notices, target+": "+line)
		}
	}
	if len(notices) > 0 {
		return nil, notices, nil
	}
	ready, notices, err := workspacePlanReady(root, target, target != controller, func(name string) ([]byte, error) {
		fact, ok := observation.Fact(state.ArtifactPath(name))
		if !ok || fact.State() == state.ArtifactAbsent {
			return nil, os.ErrNotExist
		}
		if fact.State() != state.ArtifactPresent && fact.State() != state.ArtifactEmpty {
			return nil, fmt.Errorf("%s: retained artifact is %s", name, fact.State())
		}
		return fact.Bytes(), nil
	})
	if err != nil {
		return nil, notices, err
	}
	if target != controller {
		for i := range ready {
			ready[i].Workspace = target
		}
	}
	return ready, notices, nil
}

// readSequenceArtifact permits archived predecessors, never an archived writer.
func readSequenceArtifact(root, slug, name string) ([]byte, error) {
	dir, err := devritespaths.ExistingCanonicalFeatureDirChecked(root, slug)
	if errors.Is(err, os.ErrNotExist) {
		dir, err = devritespaths.ExistingArchivedFeatureDirChecked(root, slug)
	}
	if err != nil {
		return nil, err
	}
	return readBoundedRegularFile(filepath.Join(dir, name), workspaceArtifactLimit)
}

func forwardWorkspace(root, controller, target string, targetState []byte) error {
	seen := map[string]bool{}
	current := target
	found := false
	for range maxSequenceChain {
		if seen[current] {
			return fmt.Errorf("sequence_parent cycle at %s; repair native sequence cursor", current)
		}
		seen[current] = true
		found = found || current == controller
		// Target is already retained from its live workspace observation.
		raw := targetState
		if current != target {
			var err error
			raw, err = readSequenceArtifact(root, current, "state.md")
			if err != nil {
				return fmt.Errorf("sequence_parent %s unavailable: %w", current, err)
			}
		}
		structural, err := markdowntext.Structural(raw)
		if err != nil {
			return fmt.Errorf("sequence_parent %s state.md is invalid Markdown", current)
		}
		parent, _ := state.CursorField(strings.Split(string(structural), "\n"), state.CursorSequenceParent)
		if parent == "" {
			if found {
				return nil
			}
			return fmt.Errorf("not a recorded forward continuation of %s; native sequence authorization required", controller)
		}
		if !devritespaths.ValidSlug(parent) {
			return fmt.Errorf("sequence_parent %q is not a workspace slug", parent)
		}
		current = parent
	}
	return fmt.Errorf("sequence_parent chain exceeds %d workspaces", maxSequenceChain)
}

type workspaceProducer struct {
	workspace string
	id        string
}

type workspacePlan struct {
	raw   []byte
	graph TaskGraphResult
	built map[string]bool
}

type workspaceDependencies struct {
	root  string
	plans map[string]*workspacePlan
}

func (d *workspaceDependencies) load(slug string) (*workspacePlan, error) {
	if plan := d.plans[slug]; plan != nil {
		return plan, nil
	}
	raw, err := readSequenceArtifact(d.root, slug, "tasks.md")
	if err != nil {
		return nil, fmt.Errorf("producer %s tasks.md: %w", slug, err)
	}
	graph := ParseTaskGraph(raw)
	if len(graph.Problems) > 0 {
		return nil, fmt.Errorf("producer %s tasks.md: %s", slug, strings.Join(graph.Problems, "; "))
	}
	built, err := builtSlicesFrom(func(name string) ([]byte, error) {
		return readSequenceArtifact(d.root, slug, name)
	})
	if err != nil {
		return nil, fmt.Errorf("producer %s checkpoints: %w", slug, err)
	}
	plan := &workspacePlan{raw: raw, graph: graph, built: built}
	d.plans[slug] = plan
	return plan, nil
}

func declaredWorkspaceDependencies(block string) ([]workspaceProducer, bool, error) {
	structural, err := markdowntext.Structural([]byte(block))
	if err != nil {
		return nil, false, fmt.Errorf("workspace dependencies cannot be read from invalid Markdown")
	}
	block = string(structural)
	fields := sliceFields(block)
	value, declared := fieldValue(fields, "workspace dependencies")
	count := 0
	for _, field := range sliceFieldRe.FindAllStringSubmatch(block, -1) {
		if strings.EqualFold(strings.TrimSpace(field[1]), "workspace dependencies") {
			count++
		}
	}
	if count > 1 {
		return nil, true, fmt.Errorf("duplicate Workspace dependencies; rerun native Plan/Vet")
	}
	if !declared || value == "none" {
		return nil, declared, nil
	}
	var refs []workspaceProducer
	seen := map[string]bool{}
	for token := range strings.SplitSeq(value, ",") {
		token = strings.TrimSpace(token)
		slug, id, ok := strings.Cut(token, "/")
		if !ok || !devritespaths.ValidSlug(slug) || !state.ValidSliceID(id) {
			return nil, true, fmt.Errorf("malformed Workspace dependencies reference %q; use none or <slug>/SLICE-### and rerun native Plan/Vet", token)
		}
		if seen[token] {
			return nil, true, fmt.Errorf("duplicate Workspace dependencies reference %q", token)
		}
		seen[token] = true
		refs = append(refs, workspaceProducer{workspace: slug, id: id})
	}
	return refs, true, nil
}

// validate follows local and workspace edges even across built nodes: a recorded
// checkpoint cannot make an unknown reference or a cyclic plan graph valid.
func (d *workspaceDependencies) validate(ref workspaceProducer, visiting, valid map[string]bool) error {
	key := parallel.SliceKey(ref.workspace, ref.id)
	if visiting[key] {
		return fmt.Errorf("workspace dependencies cycle at %s; repair native Plan/Vet", key)
	}
	if valid[key] {
		return nil
	}
	if len(visiting)+len(valid) >= 4096 {
		return fmt.Errorf("workspace dependencies graph exceeds 4096 slices")
	}
	plan, err := d.load(ref.workspace)
	if err != nil {
		return err
	}
	var node *TaskSlice
	for i := range plan.graph.Slices {
		if plan.graph.Slices[i].ID == ref.id {
			node = &plan.graph.Slices[i]
			break
		}
	}
	if node == nil {
		return fmt.Errorf("unknown Workspace dependencies producer %s", key)
	}
	block, _ := state.ExtractTaskSlice(plan.raw, ref.id)
	refs, _, err := declaredWorkspaceDependencies(block)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	for _, id := range node.Dependencies {
		refs = append(refs, workspaceProducer{workspace: ref.workspace, id: id})
	}
	visiting[key] = true
	defer delete(visiting, key)
	for _, dependency := range refs {
		if err := d.validate(dependency, visiting, valid); err != nil {
			return err
		}
	}
	valid[key] = true
	return nil
}

func workspacePlanReady(root, slug string, requireDeclaration bool, read func(string) ([]byte, error)) ([]parallel.SlicePaths, []string, error) {
	raw, err := read("tasks.md")
	if err != nil {
		return nil, nil, err
	}
	graph := ParseTaskGraph(raw)
	if len(graph.Problems) > 0 {
		return nil, nil, fmt.Errorf("tasks.md: %s", strings.Join(graph.Problems, "; "))
	}
	built, err := builtSlicesFrom(read)
	if err != nil {
		return nil, nil, err
	}
	deps := workspaceDependencies{root: root, plans: map[string]*workspacePlan{
		slug: {raw: raw, graph: graph, built: built},
	}}
	paths, _ := sliceWritePaths(raw, graph)
	var ready []parallel.SlicePaths
	var notices []string
	valid := map[string]bool{}
	for _, node := range graph.Slices {
		if built[sliceNumber(node.ID)] || slicesUnbuilt(node.Dependencies, built) {
			continue
		}
		key := parallel.SliceKey(slug, node.ID)
		block, _ := state.ExtractTaskSlice(raw, node.ID)
		refs, declared, problem := declaredWorkspaceDependencies(block)
		if problem == nil && requireDeclaration && !declared {
			problem = fmt.Errorf("missing Workspace dependencies; native Plan/Vet must bind none or explicit producers before foreign admission")
		}
		if problem == nil {
			problem = deps.validate(workspaceProducer{workspace: slug, id: node.ID}, map[string]bool{}, valid)
		}
		if problem != nil {
			notices = append(notices, key+": "+problem.Error())
			continue
		}
		waiting := false
		for _, ref := range refs {
			producer := deps.plans[ref.workspace] // validate loaded every referenced graph.
			if !producer.built[sliceNumber(ref.id)] {
				notices = append(notices, fmt.Sprintf("%s: waiting for durable BUILT/integrated producer %s", key, parallel.SliceKey(ref.workspace, ref.id)))
				waiting = true
			}
		}
		if waiting {
			continue
		}
		if len(paths[node.ID]) == 0 {
			if requireDeclaration {
				notices = append(notices, key+": no Writer allowlist or Files likely touched; repair native Plan/Vet")
				continue
			}
			return nil, notices, fmt.Errorf("%s has no Writer allowlist or Files likely touched", node.ID)
		}
		external, _ := fieldValue(sliceFields(block), "external dependencies")
		ready = append(ready, parallel.SlicePaths{ID: node.ID, Paths: withManifests(filepath.Dir(root), paths[node.ID], external)})
	}
	return ready, notices, nil
}

func slicesUnbuilt(ids []string, built map[string]bool) bool {
	for _, id := range ids {
		if !built[sliceNumber(id)] {
			return true
		}
	}
	return false
}

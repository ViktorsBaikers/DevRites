package state

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
)

const (
	maxArtifactBytes    = 1 << 20
	maxObservationBytes = 8 << 20
)

type ArtifactState string

const (
	ArtifactAbsent     ArtifactState = "absent"
	ArtifactEmpty      ArtifactState = "empty"
	ArtifactMalformed  ArtifactState = "malformed"
	ArtifactUnsafe     ArtifactState = "unsafe"
	ArtifactUnreadable ArtifactState = "unreadable"
	ArtifactPresent    ArtifactState = "present"
)

type DiagnosticCode string

const (
	DiagnosticMalformedMarkdown DiagnosticCode = "malformed_markdown"
	DiagnosticParentSymlink     DiagnosticCode = "parent_symlink"
	DiagnosticFinalSymlink      DiagnosticCode = "final_symlink"
	DiagnosticNonRegular        DiagnosticCode = "non_regular"
	DiagnosticFileTooLarge      DiagnosticCode = "file_too_large"
	DiagnosticPermissionDenied  DiagnosticCode = "permission_denied"
	DiagnosticReadFailure       DiagnosticCode = "read_failure"
)

type ObservationFailure string

const (
	ObservationWorkspaceInvalid  ObservationFailure = "workspace_invalid"
	ObservationAggregateTooLarge ObservationFailure = "aggregate_too_large"
	ObservationConcurrentChange  ObservationFailure = "concurrent_change"
)

func (f ObservationFailure) Error() string {
	switch f {
	case ObservationWorkspaceInvalid:
		return "workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry"
	case ObservationAggregateTooLarge:
		return "workspace observation: aggregate_too_large: retained content exceeds the 8 MiB aggregate limit; reduce retained Markdown below 8 MiB, then retry"
	case ObservationConcurrentChange:
		return "workspace observation: concurrent_change: workspace changed during acquisition; retry"
	default:
		return "workspace observation: " + string(f)
	}
}

type ArtifactDiagnostic struct {
	Path  ArtifactPath
	State ArtifactState
	Code  DiagnosticCode
}

type ArtifactFact struct {
	path  ArtifactPath
	state ArtifactState
	bytes []byte
	code  DiagnosticCode
}

func (f ArtifactFact) Path() ArtifactPath { return f.path }

func (f ArtifactFact) State() ArtifactState { return f.state }

func (f ArtifactFact) Bytes() []byte { return copyArtifactBytes(f.bytes) }

func (f ArtifactFact) Diagnostic() (ArtifactDiagnostic, bool) {
	if f.code == "" {
		return ArtifactDiagnostic{}, false
	}
	return ArtifactDiagnostic{Path: f.path, State: f.state, Code: f.code}, true
}

type WorkspaceObservation struct {
	slug  string
	facts []ArtifactFact
	index map[ArtifactPath]int
}

func (o *WorkspaceObservation) Slug() string { return o.slug }

func (o *WorkspaceObservation) Facts() []ArtifactFact {
	facts := make([]ArtifactFact, len(o.facts))
	for i, fact := range o.facts {
		facts[i] = cloneArtifactFact(fact)
	}
	return facts
}

func (o *WorkspaceObservation) Fact(path ArtifactPath) (ArtifactFact, bool) {
	index, ok := o.index[path]
	if !ok {
		return ArtifactFact{}, false
	}
	return cloneArtifactFact(o.facts[index]), true
}

func (o *WorkspaceObservation) DeclaredPhase() (Phase, error) {
	fact, _ := o.Fact(LedgerFile)
	if fact.state != ArtifactPresent {
		recovery := "repair state.md and retry"
		if fact.state == ArtifactAbsent || fact.state == ArtifactEmpty {
			recovery = "add real content to state.md and retry"
		}
		if diagnostic, ok := fact.Diagnostic(); ok {
			return "", fmt.Errorf("feature %q: %s is %s (%s); %s", o.slug, LedgerFile, fact.state, diagnostic.Code, recovery)
		}
		return "", fmt.Errorf("feature %q: %s is %s; %s", o.slug, LedgerFile, fact.state, recovery)
	}
	structural, err := markdowntext.Structural(fact.bytes)
	if err != nil {
		return "", fmt.Errorf("feature %q: %s is %s (%s); repair state.md and retry", o.slug, LedgerFile, ArtifactMalformed, DiagnosticMalformedMarkdown)
	}
	value, ok := CursorField(strings.Split(string(structural), "\n"), CursorPhase)
	if !ok {
		return "", fmt.Errorf("feature %q: no phase in %s ledger; record phase in state.md and retry", o.slug, LedgerFile)
	}
	phase := Phase(firstPhaseWord(value))
	policy, ok := PolicyFor(phase)
	if !ok {
		return "", fmt.Errorf("feature %q: unknown phase %q; record a known phase in state.md and retry", o.slug, phase)
	}
	return policy.Target, nil
}

func (o *WorkspaceObservation) Missing(required []ArtifactPath) ([]ArtifactPath, []ArtifactDiagnostic) {
	var missing []ArtifactPath
	var diagnostics []ArtifactDiagnostic
	for _, path := range required {
		fact, ok := o.Fact(path)
		if ok && fact.state == ArtifactPresent {
			continue
		}
		missing = append(missing, path)
		if ok {
			if diagnostic, hasDiagnostic := fact.Diagnostic(); hasDiagnostic {
				diagnostics = append(diagnostics, diagnostic)
			}
		}
	}
	return missing, diagnostics
}

func cloneArtifactFact(fact ArtifactFact) ArtifactFact {
	fact.bytes = copyArtifactBytes(fact.bytes)
	return fact
}

func copyArtifactBytes(raw []byte) []byte {
	if raw == nil {
		return nil
	}
	copied := make([]byte, len(raw))
	copy(copied, raw)
	return copied
}

type observationStage uint8

const (
	observationAfterResolve observationStage = iota + 1
	observationAfterInspect
	observationBeforeOpen
	observationBeforeRead
	observationAfterRead
	observationBeforeFinalSweep
)

type observationCallback func(observationStage, ArtifactPath) error

type artifactLocation struct {
	path ArtifactPath
	root *os.Root
	name string
}

type entryState uint8

const (
	entryAbsent entryState = iota
	entryPresent
	entryUnreadable
)

type observedEntry struct {
	state entryState
	info  os.FileInfo
	code  DiagnosticCode
}

type namedEntry struct {
	name  string
	entry observedEntry
}

type artifactSnapshot struct {
	parents []namedEntry
	final   *observedEntry
}

type observationRoots struct {
	devrites         *os.Root
	workspace        *os.Root
	rootInfo         os.FileInfo
	workspaceInfo    os.FileInfo
	workspaceEntry   namedEntry
	workspaceParents []namedEntry
}

func ObserveWorkspace(root, slug string) (*WorkspaceObservation, error) {
	return observeWorkspace(root, slug, nil)
}

func observeWorkspace(root, slug string, callback observationCallback) (*WorkspaceObservation, error) {
	if !observationWorkspaceOverrideValid(root, slug) {
		return nil, ObservationWorkspaceInvalid
	}
	workspacePath, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return nil, ObservationWorkspaceInvalid
	}
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() {
		return nil, ObservationWorkspaceInvalid
	}
	workspaceInfo, err := os.Stat(workspacePath)
	if err != nil || !workspaceInfo.IsDir() {
		return nil, ObservationWorkspaceInvalid
	}
	if err := runObservationStage(callback, observationAfterResolve, ""); err != nil {
		return nil, ObservationWorkspaceInvalid
	}

	roots, err := openObservationRoots(root, workspacePath, rootInfo, workspaceInfo)
	if err != nil {
		return nil, ObservationWorkspaceInvalid
	}
	defer roots.devrites.Close()
	defer roots.workspace.Close()

	inventory := observationInventory()
	facts := make([]ArtifactFact, 0, len(inventory))
	snapshots := make([]artifactSnapshot, 0, len(inventory))
	locations := make([]artifactLocation, 0, len(inventory))
	retainedBytes := 0
	for _, artifactPath := range inventory {
		location := artifactLocation{path: artifactPath, root: roots.workspace, name: string(artifactPath)}
		if artifactPath == ".devrites/principles.md" {
			location.root = roots.devrites
			location.name = "principles.md"
		}
		fact, snapshot, observeErr := observeArtifact(location, callback)
		if observeErr != nil {
			return nil, observeErr
		}
		retainedBytes += len(fact.bytes)
		if retainedBytes > maxObservationBytes {
			return nil, ObservationAggregateTooLarge
		}
		facts = append(facts, fact)
		snapshots = append(snapshots, snapshot)
		locations = append(locations, location)
	}

	if err := runObservationStage(callback, observationBeforeFinalSweep, ""); err != nil {
		return nil, ObservationConcurrentChange
	}
	if !observationRootsUnchanged(roots) {
		return nil, ObservationConcurrentChange
	}
	for i, location := range locations {
		if !artifactUnchanged(location, snapshots[i]) {
			return nil, ObservationConcurrentChange
		}
	}

	index := make(map[ArtifactPath]int, len(facts))
	for i, fact := range facts {
		index[fact.path] = i
	}
	return &WorkspaceObservation{slug: slug, facts: facts, index: index}, nil
}

func observationWorkspaceOverrideValid(root, slug string) bool {
	raw := os.Getenv("DEVRITES_WORKSPACE")
	if raw == "" {
		return true
	}
	if strings.TrimSpace(raw) != raw || filepath.Clean(raw) != raw {
		return false
	}
	if !filepath.IsAbs(raw) && !filepath.IsLocal(raw) {
		return false
	}
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	physicalRoot, err = filepath.Abs(physicalRoot)
	if err != nil {
		return false
	}
	override := raw
	if !filepath.IsAbs(override) {
		override = filepath.Join(filepath.Dir(physicalRoot), override)
	}
	override, err = filepath.Abs(override)
	if err != nil {
		return false
	}
	want := filepath.Join(physicalRoot, "work", slug)
	return filepath.Clean(override) == filepath.Clean(want)
}

func observationInventory() []ArtifactPath {
	seen := make(map[ArtifactPath]bool)
	inventory := make([]ArtifactPath, 0, 21)
	for _, policy := range PhasePolicies() {
		for _, artifact := range policy.RequiredArtifacts {
			if seen[artifact] {
				continue
			}
			seen[artifact] = true
			inventory = append(inventory, artifact)
		}
	}
	for _, artifact := range []ArtifactPath{"strategy.md", "design-brief.md", "ai-spec.md", ".devrites/principles.md"} {
		if !seen[artifact] {
			seen[artifact] = true
			inventory = append(inventory, artifact)
		}
	}
	return inventory
}

func openObservationRoots(rootPath, workspacePath string, rootInfo, workspaceInfo os.FileInfo) (*observationRoots, error) {
	devritesRoot, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	workspaceRoot, err := os.OpenRoot(workspacePath)
	if err != nil {
		_ = devritesRoot.Close()
		return nil, err
	}
	roots := &observationRoots{
		devrites:      devritesRoot,
		workspace:     workspaceRoot,
		rootInfo:      rootInfo,
		workspaceInfo: workspaceInfo,
	}
	rootHandleInfo, rootErr := devritesRoot.Stat(".")
	workspaceHandleInfo, workspaceErr := workspaceRoot.Stat(".")
	if rootErr != nil || workspaceErr != nil || !sameEntry(rootInfo, rootHandleInfo) || !sameEntry(workspaceInfo, workspaceHandleInfo) {
		_ = devritesRoot.Close()
		_ = workspaceRoot.Close()
		return nil, errors.New("observation root identity changed")
	}

	relativeWorkspace, err := filepath.Rel(rootPath, workspacePath)
	if err != nil || relativeWorkspace == "." || filepath.IsAbs(relativeWorkspace) || strings.HasPrefix(relativeWorkspace, ".."+string(filepath.Separator)) {
		_ = devritesRoot.Close()
		_ = workspaceRoot.Close()
		return nil, errors.New("observation workspace is outside root")
	}
	for _, parentName := range parentPaths(filepath.ToSlash(relativeWorkspace)) {
		entry := inspectEntry(devritesRoot, filepath.FromSlash(parentName))
		if entry.state != entryPresent || entry.info.Mode()&os.ModeSymlink != 0 || !entry.info.IsDir() {
			_ = devritesRoot.Close()
			_ = workspaceRoot.Close()
			return nil, errors.New("observation workspace parent is invalid")
		}
		roots.workspaceParents = append(roots.workspaceParents, namedEntry{name: filepath.FromSlash(parentName), entry: entry})
	}
	workspaceEntry := inspectEntry(devritesRoot, relativeWorkspace)
	if workspaceEntry.state != entryPresent || workspaceEntry.info.Mode()&os.ModeSymlink != 0 || !workspaceEntry.info.IsDir() || !sameEntry(workspaceInfo, workspaceEntry.info) {
		_ = devritesRoot.Close()
		_ = workspaceRoot.Close()
		return nil, errors.New("observation workspace identity changed")
	}
	roots.workspaceEntry = namedEntry{name: relativeWorkspace, entry: workspaceEntry}
	return roots, nil
}

func observationRootsUnchanged(roots *observationRoots) bool {
	rootInfo, rootErr := roots.devrites.Stat(".")
	workspaceInfo, workspaceErr := roots.workspace.Stat(".")
	if rootErr != nil || workspaceErr != nil || !sameEntry(roots.rootInfo, rootInfo) || !sameEntry(roots.workspaceInfo, workspaceInfo) {
		return false
	}
	for _, parent := range roots.workspaceParents {
		if !sameObservedEntry(parent.entry, inspectEntry(roots.devrites, parent.name)) {
			return false
		}
	}
	return sameObservedEntry(roots.workspaceEntry.entry, inspectEntry(roots.devrites, roots.workspaceEntry.name))
}

func observeArtifact(location artifactLocation, callback observationCallback) (ArtifactFact, artifactSnapshot, error) {
	snapshot := artifactSnapshot{}
	for _, parentName := range parentPaths(filepath.ToSlash(location.name)) {
		entry := inspectEntry(location.root, filepath.FromSlash(parentName))
		snapshot.parents = append(snapshot.parents, namedEntry{name: filepath.FromSlash(parentName), entry: entry})
		switch {
		case entry.state == entryUnreadable:
			return unreadableFact(location.path, entry.code), snapshot, nil
		case entry.state == entryAbsent:
			return ArtifactFact{path: location.path, state: ArtifactAbsent}, snapshot, nil
		case entry.info.Mode()&os.ModeSymlink != 0:
			return diagnosticFact(location.path, ArtifactUnsafe, DiagnosticParentSymlink), snapshot, nil
		case !entry.info.IsDir():
			return diagnosticFact(location.path, ArtifactUnsafe, DiagnosticNonRegular), snapshot, nil
		}
	}

	entry := inspectEntry(location.root, location.name)
	snapshot.final = &entry
	if err := runObservationStage(callback, observationAfterInspect, location.path); err != nil {
		return artifactFailureResult(location.path, snapshot, artifactUnchanged(location, snapshot), err)
	}
	switch {
	case entry.state == entryAbsent:
		return ArtifactFact{path: location.path, state: ArtifactAbsent}, snapshot, nil
	case entry.state == entryUnreadable:
		return unreadableFact(location.path, entry.code), snapshot, nil
	case entry.info.Mode()&os.ModeSymlink != 0:
		return diagnosticFact(location.path, ArtifactUnsafe, DiagnosticFinalSymlink), snapshot, nil
	case !entry.info.Mode().IsRegular():
		return diagnosticFact(location.path, ArtifactUnsafe, DiagnosticNonRegular), snapshot, nil
	case entry.info.Size() > maxArtifactBytes:
		return diagnosticFact(location.path, ArtifactUnsafe, DiagnosticFileTooLarge), snapshot, nil
	}

	if err := runObservationStage(callback, observationBeforeOpen, location.path); err != nil {
		return artifactFailureResult(location.path, snapshot, artifactUnchanged(location, snapshot), err)
	}
	file, err := openObservedFile(location.root, location.name)
	if err != nil {
		return artifactFailureResult(location.path, snapshot, artifactUnchanged(location, snapshot), err)
	}
	defer file.Close()

	openedInfo, err := file.Stat()
	if err != nil {
		return artifactFailureResult(location.path, snapshot, artifactUnchanged(location, snapshot), err)
	}
	if !sameEntry(entry.info, openedInfo) || !openedInfo.Mode().IsRegular() {
		return ArtifactFact{}, snapshot, ObservationConcurrentChange
	}
	if err := runObservationStage(callback, observationBeforeRead, location.path); err != nil {
		return artifactFailureResult(location.path, snapshot, openedFileAndPathUnchanged(file, openedInfo, location, snapshot), err)
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxArtifactBytes+1))
	if err != nil {
		return artifactFailureResult(location.path, snapshot, openedFileAndPathUnchanged(file, openedInfo, location, snapshot), err)
	}
	if err := runObservationStage(callback, observationAfterRead, location.path); err != nil {
		return artifactFailureResult(location.path, snapshot, openedFileAndPathUnchanged(file, openedInfo, location, snapshot), err)
	}
	postReadInfo, err := file.Stat()
	if err != nil || !sameEntry(openedInfo, postReadInfo) || int64(len(raw)) != postReadInfo.Size() || !artifactUnchanged(location, snapshot) {
		return ArtifactFact{}, snapshot, ObservationConcurrentChange
	}
	if len(raw) > maxArtifactBytes {
		return diagnosticFact(location.path, ArtifactUnsafe, DiagnosticFileTooLarge), snapshot, nil
	}
	state, code := classifyArtifact(raw)
	return ArtifactFact{path: location.path, state: state, bytes: copyArtifactBytes(raw), code: code}, snapshot, nil
}

func artifactFailureResult(path ArtifactPath, snapshot artifactSnapshot, unchanged bool, cause error) (ArtifactFact, artifactSnapshot, error) {
	if !unchanged {
		return ArtifactFact{}, snapshot, ObservationConcurrentChange
	}
	return unreadableFact(path, unreadableDiagnostic(cause)), snapshot, nil
}

func openedFileAndPathUnchanged(file *os.File, openedInfo os.FileInfo, location artifactLocation, snapshot artifactSnapshot) bool {
	currentInfo, err := file.Stat()
	return err == nil && sameEntry(openedInfo, currentInfo) && artifactUnchanged(location, snapshot)
}

func artifactUnchanged(location artifactLocation, snapshot artifactSnapshot) bool {
	for _, parent := range snapshot.parents {
		if !sameObservedEntry(parent.entry, inspectEntry(location.root, parent.name)) {
			return false
		}
	}
	if snapshot.final == nil {
		return true
	}
	return sameObservedEntry(*snapshot.final, inspectEntry(location.root, location.name))
}

func inspectEntry(root *os.Root, name string) observedEntry {
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return observedEntry{state: entryAbsent}
	}
	if err != nil {
		return observedEntry{state: entryUnreadable, code: unreadableDiagnostic(err)}
	}
	return observedEntry{state: entryPresent, info: info}
}

func sameObservedEntry(left, right observedEntry) bool {
	if left.state != right.state || left.code != right.code {
		return false
	}
	if left.state != entryPresent {
		return true
	}
	return sameEntry(left.info, right.info)
}

func sameEntry(left, right os.FileInfo) bool {
	return left != nil && right != nil && left.Mode().Type() == right.Mode().Type() && left.Size() == right.Size() && os.SameFile(left, right)
}

func parentPaths(name string) []string {
	clean := path.Clean(name)
	if clean == "." || clean == "/" {
		return nil
	}
	parts := strings.Split(clean, "/")
	parents := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		parents = append(parents, strings.Join(parts[:i], "/"))
	}
	return parents
}

func classifyArtifact(raw []byte) (ArtifactState, DiagnosticCode) {
	if _, err := markdowntext.Structural(raw); err != nil {
		return ArtifactMalformed, DiagnosticMalformedMarkdown
	}
	body := stripFrontmatter(raw)
	for _, line := range strings.Split(string(body), "\n") {
		if !blankOrHash(strings.TrimSpace(line)) {
			return ArtifactPresent, ""
		}
	}
	return ArtifactEmpty, ""
}

func diagnosticFact(path ArtifactPath, state ArtifactState, code DiagnosticCode) ArtifactFact {
	return ArtifactFact{path: path, state: state, code: code}
}

func unreadableFact(path ArtifactPath, code DiagnosticCode) ArtifactFact {
	return diagnosticFact(path, ArtifactUnreadable, code)
}

func unreadableDiagnostic(err error) DiagnosticCode {
	if errors.Is(err, fs.ErrPermission) {
		return DiagnosticPermissionDenied
	}
	return DiagnosticReadFailure
}

func runObservationStage(callback observationCallback, stage observationStage, path ArtifactPath) error {
	if callback == nil {
		return nil
	}
	return callback(stage, path)
}

// Package rootfacts resolves one read-only DevRites project identity.
package rootfacts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/safepath"
)

var (
	ErrNoRoot     = errors.New("no DevRites root")
	ErrUnsafeRoot = errors.New("unsafe DevRites root")
)

// Hazard is a stable diagnostic plus one command the user can paste.
type Hazard struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
	Refuse      bool   `json:"refuse,omitempty"`
}

// GitFacts are physical identities reported by Git itself.
type GitFacts struct {
	TopLevel       string `json:"topLevel,omitempty"`
	Dir            string `json:"dir,omitempty"`
	CommonDir      string `json:"commonDir,omitempty"`
	Superproject   string `json:"superproject,omitempty"`
	LinkedWorktree bool   `json:"linkedWorktree"`
	Submodule      bool   `json:"submodule"`
}

// Facts are the canonical project/root selection and its repository topology.
type Facts struct {
	LexicalProject  string   `json:"lexicalProject,omitempty"`
	PhysicalProject string   `json:"physicalProject,omitempty"`
	LexicalRoot     string   `json:"lexicalRoot,omitempty"`
	PhysicalRoot    string   `json:"physicalRoot,omitempty"`
	SelectionReason string   `json:"selectionReason"`
	Git             GitFacts `json:"git"`
	Hazards         []Hazard `json:"hazards"`
}

// Resolve discovers facts from the process working directory.
func Resolve(override string) (Facts, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Facts{}, fmt.Errorf("resolve working directory: %w", err)
	}
	return ResolveFrom(cwd, override)
}

// ResolveFrom is Resolve with an explicit starting directory for deterministic
// callers and fixtures.
func ResolveFrom(cwd, override string) (Facts, error) {
	var facts Facts
	lexicalCWD, err := absolute(cwd)
	if err != nil {
		return facts, fmt.Errorf("resolve working directory %q: %w", cwd, err)
	}
	physicalCWD, err := physical(lexicalCWD)
	if err != nil {
		return facts, fmt.Errorf("resolve working directory %q: %w", cwd, err)
	}
	if info, statErr := os.Stat(physicalCWD); statErr != nil || !info.IsDir() {
		if statErr == nil {
			statErr = errors.New("path is not a directory")
		}
		return facts, fmt.Errorf("resolve working directory %q: %w", cwd, statErr)
	}

	if strings.TrimSpace(override) != "" {
		return resolveExplicit(physicalCWD, override)
	}

	facts.Git = probeGit(physicalCWD)
	boundary := filesystemRoot(physicalCWD)
	reason := "filesystem-ancestor"
	repositoryBoundary := ""
	if facts.Git.TopLevel != "" && within(physicalCWD, facts.Git.TopLevel) {
		boundary = facts.Git.TopLevel
		repositoryBoundary = facts.Git.TopLevel
		reason = "git-ancestor"
	} else if marker := nearestGitMarker(physicalCWD); marker != "" {
		boundary = marker
		repositoryBoundary = marker
		reason = "git-marker-ancestor"
		facts.Hazards = append(facts.Hazards, Hazard{
			ID:          "DRV-GIT-FACTS-UNAVAILABLE",
			Severity:    "warning",
			Message:     fmt.Sprintf("Git topology probes failed; root search is conservatively bounded by %q", marker),
			Remediation: fmt.Sprintf("git -C %s rev-parse --show-toplevel", shellQuote(marker)),
		})
	}

	for dir := physicalCWD; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, devritespaths.DevritesRootName)
		if isDir(candidate) {
			resolvedRoot, err := physical(candidate)
			if err != nil {
				return facts, fmt.Errorf("resolve .devrites root %q: %w", candidate, err)
			}
			facts.LexicalProject = dir
			facts.PhysicalProject = dir
			facts.LexicalRoot = candidate
			facts.PhysicalRoot = resolvedRoot
			facts.SelectionReason = reason
			if !within(facts.PhysicalRoot, facts.PhysicalProject) {
				facts.Hazards = append(facts.Hazards, Hazard{
					ID:          "DRV-ROOT-SYMLINK-ESCAPE",
					Severity:    "error",
					Message:     fmt.Sprintf(".devrites resolves outside project %q", facts.PhysicalProject),
					Remediation: fmt.Sprintf("rm %s  # then create project-local state with /rite-spec", shellQuote(facts.LexicalRoot)),
					Refuse:      true,
				})
				return facts, fmt.Errorf("%w: .devrites resolves outside the project", ErrUnsafeRoot)
			}
			inspectWorkspaceState(&facts)
			if facts.Refuses() {
				return facts, fmt.Errorf("%w: implicit root has unsafe workspace state", ErrUnsafeRoot)
			}
			return facts, nil
		}
		if samePath(dir, boundary) {
			break
		}
	}

	facts.SelectionReason = "none"
	if repositoryBoundary != "" {
		facts.LexicalProject = repositoryBoundary
		facts.PhysicalProject = repositoryBoundary
		if outside := ancestorRoot(filepath.Dir(repositoryBoundary)); outside != "" {
			facts.Hazards = append(facts.Hazards, Hazard{
				ID:       "DRV-ROOT-OUTSIDE-GIT",
				Severity: "error",
				Message:  fmt.Sprintf("parent DevRites root %q is outside the current Git repository %q", outside, repositoryBoundary),
				Remediation: fmt.Sprintf("cd %s  # use the parent workspace; otherwise run /rite-spec in %s",
					shellQuote(filepath.Dir(outside)), shellQuote(repositoryBoundary)),
				Refuse: true,
			})
			return facts, fmt.Errorf("%w: parent .devrites is outside the current Git repository", ErrUnsafeRoot)
		}
	} else {
		facts.LexicalProject = physicalCWD
		facts.PhysicalProject = physicalCWD
	}
	facts.Hazards = append(facts.Hazards, Hazard{
		ID:          "DRV-ROOT-NOT-FOUND",
		Severity:    "warning",
		Message:     fmt.Sprintf("no .devrites directory exists within the search boundary %q", boundary),
		Remediation: fmt.Sprintf("cd %s  # then run /rite-spec", shellQuote(boundary)),
	})
	if strings.TrimSpace(os.Getenv("DEVRITES_WORKSPACE")) != "" {
		facts.Hazards = append(facts.Hazards, Hazard{
			ID:          "DRV-WORKSPACE-WITHOUT-ROOT",
			Severity:    "error",
			Message:     "DEVRITES_WORKSPACE is set but no containing DevRites root was selected",
			Remediation: "unset DEVRITES_WORKSPACE",
			Refuse:      true,
		})
		return facts, fmt.Errorf("%w: DEVRITES_WORKSPACE requires a selected DevRites root", ErrUnsafeRoot)
	}
	return facts, fmt.Errorf("%w: no .devrites directory found within %q", ErrNoRoot, boundary)
}

func resolveExplicit(cwd, override string) (Facts, error) {
	facts := Facts{
		LexicalProject:  cwd,
		PhysicalProject: cwd,
		SelectionReason: "DEVRITES_ROOT",
	}
	lexical, err := absoluteFrom(cwd, override)
	if err != nil {
		return invalidExplicit(facts, override, err)
	}
	info, err := os.Stat(lexical)
	if err != nil {
		return invalidExplicit(facts, override, err)
	}
	if !info.IsDir() {
		return invalidExplicit(facts, override, errors.New("path is not a directory"))
	}

	if filepath.Base(filepath.Clean(lexical)) == devritespaths.DevritesRootName {
		facts.LexicalRoot = lexical
		facts.LexicalProject = filepath.Dir(lexical)
	} else {
		child := filepath.Join(lexical, devritespaths.DevritesRootName)
		if !isDir(child) {
			return invalidExplicit(facts, override, errors.New("path is not a project root or .devrites directory"))
		}
		facts.LexicalProject = lexical
		facts.LexicalRoot = child
	}

	facts.PhysicalRoot, err = physical(facts.LexicalRoot)
	if err != nil {
		return invalidExplicit(facts, override, err)
	}
	facts.PhysicalProject, err = physical(facts.LexicalProject)
	if err != nil {
		return invalidExplicit(facts, override, err)
	}
	if !within(facts.PhysicalRoot, facts.PhysicalProject) {
		facts.Hazards = append(facts.Hazards, Hazard{
			ID:          "DRV-ROOT-SYMLINK-ESCAPE",
			Severity:    "error",
			Message:     fmt.Sprintf(".devrites resolves outside explicit project %q", facts.PhysicalProject),
			Remediation: "unset DEVRITES_ROOT",
			Refuse:      true,
		})
		return facts, fmt.Errorf("%w: explicit .devrites resolves outside the project", ErrUnsafeRoot)
	}
	facts.Git = probeGit(facts.PhysicalProject)
	inspectWorkspaceState(&facts)
	if facts.Refuses() {
		return facts, fmt.Errorf("%w: explicit root has unsafe workspace state", ErrUnsafeRoot)
	}
	return facts, nil
}

func invalidExplicit(facts Facts, override string, cause error) (Facts, error) {
	facts.Hazards = append(facts.Hazards, Hazard{
		ID:          "DRV-ROOT-INVALID",
		Severity:    "error",
		Message:     fmt.Sprintf("DEVRITES_ROOT %q is invalid: %v", override, cause),
		Remediation: "unset DEVRITES_ROOT",
		Refuse:      true,
	})
	return facts, fmt.Errorf("%w: DEVRITES_ROOT %q: %v", ErrUnsafeRoot, override, cause)
}

// Refuses reports whether any topology hazard makes an action unsafe.
func (f Facts) Refuses() bool {
	for _, hazard := range f.Hazards {
		if hazard.Refuse {
			return true
		}
	}
	return false
}

func inspectWorkspaceState(facts *Facts) {
	root := facts.PhysicalRoot
	if root == "" {
		return
	}
	for _, layout := range []string{"work", "features"} {
		entries, err := os.ReadDir(filepath.Join(root, layout))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			path := filepath.Join(root, layout, entry.Name())
			if safepath.WithinResolved(path, root) {
				continue
			}
			facts.Hazards = append(facts.Hazards, Hazard{
				ID:          "DRV-WORKSPACE-SYMLINK-ESCAPE",
				Severity:    "error",
				Message:     fmt.Sprintf("workspace path %q resolves outside the selected root", path),
				Remediation: fmt.Sprintf("rm %s", shellQuote(path)),
				Refuse:      true,
			})
		}
	}
	if strings.TrimSpace(os.Getenv("DEVRITES_WORKSPACE")) != "" {
		if _, err := devritespaths.WorkspaceOverrideChecked(root, ""); err != nil {
			id := diagnosticID(err.Error(), "DRV-WORKSPACE-INVALID")
			facts.Hazards = append(facts.Hazards, Hazard{
				ID:          id,
				Severity:    "error",
				Message:     err.Error(),
				Remediation: "unset DEVRITES_WORKSPACE",
				Refuse:      true,
			})
		}
	}

	activePath := filepath.Join(root, "ACTIVE")
	if info, err := os.Lstat(activePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		facts.Hazards = append(facts.Hazards, Hazard{
			ID:          "DRV-ACTIVE-SYMLINK",
			Severity:    "error",
			Message:     "ACTIVE is a symlink rather than root-owned state",
			Remediation: fmt.Sprintf("rm -f %s", shellQuote(activePath)),
			Refuse:      true,
		})
		return
	}
	raw, err := os.ReadFile(activePath)
	if err != nil {
		return
	}
	slug := strings.TrimSpace(string(raw))
	if slug == "" {
		return
	}
	remediation := fmt.Sprintf("rm -f %s", shellQuote(activePath))
	if !validSlug(slug) {
		facts.Hazards = append(facts.Hazards, Hazard{
			ID:          "DRV-ACTIVE-INVALID",
			Severity:    "error",
			Message:     fmt.Sprintf("ACTIVE value %q is not a feature slug", slug),
			Remediation: remediation,
			Refuse:      true,
		})
		return
	}
	if !isDir(filepath.Join(root, "work", slug)) && !isDir(filepath.Join(root, "features", slug)) {
		facts.Hazards = append(facts.Hazards, Hazard{
			ID:          "DRV-ACTIVE-STALE",
			Severity:    "warning",
			Message:     fmt.Sprintf("ACTIVE points to missing workspace %q", slug),
			Remediation: remediation,
		})
	}
}

func probeGit(dir string) GitFacts {
	top, ok := gitPath(dir, "--show-toplevel")
	if !ok {
		return GitFacts{}
	}
	gitDir, _ := gitPath(dir, "--git-dir")
	commonDir, _ := gitPath(dir, "--git-common-dir")
	superproject, _ := gitPath(dir, "--show-superproject-working-tree")
	return GitFacts{
		TopLevel:       top,
		Dir:            gitDir,
		CommonDir:      commonDir,
		Superproject:   superproject,
		LinkedWorktree: gitDir != "" && commonDir != "" && !samePath(gitDir, commonDir),
		Submodule:      superproject != "",
	}
}

func gitPath(dir, flag string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--path-format=absolute", flag)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil || ctx.Err() != nil {
		return "", false
	}
	value := strings.TrimSpace(string(out))
	if value == "" {
		return "", true
	}
	value = filepath.FromSlash(value)
	if !filepath.IsAbs(value) {
		value = filepath.Join(dir, value)
	}
	if resolved, err := physical(value); err == nil {
		return resolved, true
	}
	return filepath.Clean(value), true
}

func ancestorRoot(start string) string {
	for dir := start; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, devritespaths.DevritesRootName)
		if isDir(candidate) {
			if resolved, err := physical(candidate); err == nil {
				return resolved
			}
			return filepath.Clean(candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
	}
}

func nearestGitMarker(start string) string {
	for dir := start; ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(filepath.Join(dir, ".git"))
		if err == nil && info.Mode()&os.ModeSymlink == 0 && (info.IsDir() || info.Mode().IsRegular()) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
	}
}

func physical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return safepath.ResolveExisting(abs)
}

func absolute(path string) (string, error) {
	return filepath.Abs(filepath.Clean(path))
}

func absoluteFrom(cwd, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	return absolute(path)
}

func filesystemRoot(path string) string {
	for {
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		path = parent
	}
}

func samePath(a, b string) bool {
	ai, aerr := os.Stat(a)
	bi, berr := os.Stat(b)
	if aerr == nil && berr == nil {
		return os.SameFile(ai, bi)
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func within(candidate, parent string) bool {
	return safepath.WithinResolved(candidate, parent)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func validSlug(slug string) bool {
	return slug != "." && slug != ".." &&
		filepath.Base(slug) == slug &&
		!strings.ContainsAny(slug, `/\`)
}

func diagnosticID(message, fallback string) string {
	id, _, ok := strings.Cut(message, ":")
	if ok && strings.HasPrefix(id, "DRV-") {
		return id
	}
	return fallback
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

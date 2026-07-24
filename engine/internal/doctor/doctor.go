// Package doctor compares the engine, installed pack, and workspace schema
// versions. It also reports root and installation problems without changing
// workspace state.
package doctor

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/rootfacts"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/version"
)

// Unknown is reported when the installed pack version cannot be found. Doctor
// does not report version skew against this value.
const Unknown = "unknown"

// Report is the resolved version triangle plus its verdict.
type Report struct {
	Binary       string   // the engine binary version
	Pack         string   // the installed pack version, or Unknown
	StateSchema  int      // the highest schemaVersion declared on disk
	BinarySchema int      // the schema version this binary understands
	Verdict      string   // one-line human summary
	Checks       []string // advisory install/host-artifact drift checks
	Refuse       bool     // true when topology or state schema makes action unsafe
	Root         rootfacts.Facts
}

// Diagnose compares versions for projectDir and its .devrites root.
//
//   - A state schema newer than the binary causes a refusal because the binary
//     cannot safely parse it.
//   - A binary older than the pack produces a warning but does not block.
//   - All other comparable versions are compatible. A newer binary can read
//     older state because schema changes are additive.
func Diagnose(projectDir, root string) (*Report, error) {
	return DiagnoseFacts(rootfacts.Facts{
		LexicalProject:  projectDir,
		PhysicalProject: projectDir,
		LexicalRoot:     root,
		PhysicalRoot:    root,
		SelectionReason: "provided",
		Hazards:         []rootfacts.Hazard{},
	})
}

// DiagnoseFacts reports the same canonical root facts used by state selection
// and context show.
func DiagnoseFacts(facts rootfacts.Facts) (*Report, error) {
	stateSchema := state.SchemaVersion
	if facts.PhysicalRoot != "" {
		var err error
		stateSchema, err = state.MaxDeclaredSchemaVersion(facts.PhysicalRoot)
		if err != nil {
			return nil, fmt.Errorf("read state schema: %w", err)
		}
		if stateSchema == 0 {
			// Nothing on disk declares a version: treat it as the current schema;
			// undeclared state is read as the engine's own version.
			stateSchema = state.SchemaVersion
		}
	}
	projectDir := facts.PhysicalProject

	r := &Report{
		Binary:       version.Version,
		Pack:         packVersion(projectDir),
		StateSchema:  stateSchema,
		BinarySchema: state.SchemaVersion,
		Checks:       driftChecks(projectDir),
		Root:         facts,
	}

	switch {
	case facts.Refuses():
		r.Refuse = true
		r.Verdict = "REFUSE: root or workspace selection is unsafe; apply the reported remediation"
	case r.StateSchema > r.BinarySchema:
		r.Refuse = true
		r.Verdict = fmt.Sprintf("REFUSE: state schema v%d is newer than this binary supports (v%d): update devrites",
			r.StateSchema, r.BinarySchema)
	case r.packSkew():
		r.Verdict = fmt.Sprintf("WARN: binary %s is older than the installed pack %s: update the devrites-engine binary",
			r.Binary, r.Pack)
	case len(facts.Hazards) > 0:
		r.Verdict = "WARN: workspace or repository hazards need attention"
	default:
		r.Verdict = "ok: binary, pack, and state schema are compatible"
	}
	return r, nil
}

// packSkew reports whether the binary is comparably older than the pack.
func (r *Report) packSkew() bool {
	if r.Pack == Unknown {
		return false
	}
	cmp, ok := version.Compare(r.Binary, r.Pack)
	return ok && cmp < 0
}

// Render produces the deterministic doctor report with a trailing newline.
func (r *Report) Render() string {
	var b strings.Builder
	physicalRoot := r.Root.PhysicalRoot
	if physicalRoot == "" {
		physicalRoot = "none"
	}
	physicalProject := r.Root.PhysicalProject
	if physicalProject == "" {
		physicalProject = "unknown"
	}
	fmt.Fprintf(&b, "project: %s\n", physicalProject)
	fmt.Fprintf(&b, "root: %s\n", physicalRoot)
	fmt.Fprintf(&b, "root-selection: %s\n", r.Root.SelectionReason)
	if r.Root.LexicalProject != "" && r.Root.LexicalProject != r.Root.PhysicalProject {
		fmt.Fprintf(&b, "project-lexical: %s\n", r.Root.LexicalProject)
	}
	if r.Root.LexicalRoot != "" && r.Root.LexicalRoot != r.Root.PhysicalRoot {
		fmt.Fprintf(&b, "root-lexical: %s\n", r.Root.LexicalRoot)
	}
	if r.Root.Git.TopLevel == "" {
		fmt.Fprintln(&b, "git: none")
	} else {
		fmt.Fprintf(&b, "git: top=%s dir=%s common=%s linked-worktree=%t submodule=%t\n",
			r.Root.Git.TopLevel, r.Root.Git.Dir, r.Root.Git.CommonDir,
			r.Root.Git.LinkedWorktree, r.Root.Git.Submodule)
		if r.Root.Git.Superproject != "" {
			fmt.Fprintf(&b, "git-superproject: %s\n", r.Root.Git.Superproject)
		}
	}
	fmt.Fprintf(&b, "binary: %s\n", r.Binary)
	fmt.Fprintf(&b, "pack: %s\n", r.Pack)
	fmt.Fprintf(&b, "state-schema: v%d (binary supports v%d)\n", r.StateSchema, r.BinarySchema)
	fmt.Fprintf(&b, "verdict: %s\n", r.Verdict)
	if len(r.Root.Hazards) == 0 {
		fmt.Fprintln(&b, "hazards: ok")
	} else {
		fmt.Fprintln(&b, "hazards:")
		for _, hazard := range r.Root.Hazards {
			fmt.Fprintf(&b, "  - [%s] %s: %s\n", hazard.ID, strings.ToUpper(hazard.Severity), hazard.Message)
			fmt.Fprintf(&b, "    fix: %s\n", hazard.Remediation)
		}
	}
	if len(r.Checks) == 0 {
		fmt.Fprintln(&b, "checks: ok")
	} else {
		fmt.Fprintln(&b, "checks:")
		for _, check := range r.Checks {
			fmt.Fprintf(&b, "  - %s\n", check)
		}
	}
	return b.String()
}

// packVersion discovers the installed DevRites pack version. It prefers an
// explicit marker (.claude/devrites.version) and falls back to the npm
// package.json at the project root; either absent leaves the pack Unknown so no
// false skew is asserted.
func packVersion(projectDir string) string {
	if v := firstLine(filepath.Join(projectDir, ".claude", "devrites.version")); v != "" {
		return v
	}
	if v := packageJSONVersion(filepath.Join(projectDir, "package.json")); v != "" {
		return v
	}
	return Unknown
}

func firstLine(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(raw), "\n")
	return strings.TrimSpace(line)
}

func driftChecks(projectDir string) []string {
	var checks []string
	settings := filepath.Join(projectDir, ".claude", "settings.json")
	if isFile(settings) {
		body := readString(settings)
		if !strings.Contains(body, "devrites-engine hook") {
			checks = append(checks, "WARN: .claude/settings.json exists but does not reference devrites-engine hooks")
		}
	} else if isDir(filepath.Join(projectDir, ".claude")) {
		checks = append(checks, "WARN: .claude/settings.json missing: Claude hooks/statusline may be absent")
	}
	if isDir(filepath.Join(projectDir, ".claude", "skills")) && !isDir(filepath.Join(projectDir, ".agents", "skills")) {
		checks = append(checks, "WARN: Claude skills installed but Codex skill mirror .agents/skills is missing")
	}
	if isDir(filepath.Join(projectDir, ".claude", "agents")) && !isDir(filepath.Join(projectDir, ".codex", "agents")) {
		checks = append(checks, "WARN: Claude agents installed but Codex agent mirror .codex/agents is missing")
	}
	if firstLine(filepath.Join(projectDir, ".claude", "devrites.version")) == "" && isDir(filepath.Join(projectDir, ".claude", "skills")) {
		checks = append(checks, "WARN: installed skills have no .claude/devrites.version marker")
	}
	if generatedClaudeDrift(projectDir) {
		checks = append(checks, "[DRV-GENERATED-DRIFT] WARN: pack/generated/claude differs from canonical pack/.claude; run `bash scripts/build-host-artifacts.sh`")
	}
	return checks
}

func generatedClaudeDrift(projectDir string) bool {
	canonical := filepath.Join(projectDir, "pack", ".claude")
	generated := filepath.Join(projectDir, "pack", "generated", "claude")
	if !isDir(canonical) {
		return false
	}
	want, err := fileDigests(canonical, func(rel string) bool {
		return rel == filepath.Join("agents", ".impeccable") ||
			strings.HasPrefix(rel, filepath.Join("agents", ".impeccable")+string(filepath.Separator))
	})
	if err != nil {
		return true
	}
	got, err := fileDigests(generated, nil)
	if err != nil || len(want) != len(got) {
		return true
	}
	for path, digest := range want {
		if got[path] != digest {
			return true
		}
	}
	return false
}

func fileDigests(root string, skip func(string) bool) (map[string][sha256.Size]byte, error) {
	files := map[string][sha256.Size]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skip != nil && skip(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("generated tree contains non-regular file %s", rel)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = sha256.Sum256(body)
		return nil
	})
	return files, err
}

func readString(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func packageJSONVersion(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return ""
	}
	return strings.TrimSpace(pkg.Version)
}

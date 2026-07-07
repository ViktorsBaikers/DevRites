// Package doctor reports the version triangle — the engine binary, the installed
// DevRites pack, and the on-disk .devrites state schema — and a clear skew
// verdict. Version skew is a leading cause of silent breakage; doctor turns it
// into one legible line. Everything here is a read; doctor never mutates state.
package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/version"
)

// Unknown is the pack value reported when the installed pack version can't be
// discovered — skew against an unknown pack is not asserted.
const Unknown = "unknown"

// Report is the resolved version triangle plus its verdict.
type Report struct {
	Binary       string // the engine binary version
	Pack         string // the installed pack version, or Unknown
	StateSchema  int    // the highest schemaVersion declared on disk
	BinarySchema int    // the schema version this binary understands
	Verdict      string // one-line human summary
	Refuse       bool   // true iff state schema is a newer major than the binary supports
}

// Diagnose resolves the triangle for the project rooted at projectDir with the
// .devrites directory at root, and computes the skew verdict.
//
//   - State schema newer (higher major) than the binary supports → Refuse: the
//     binary would not safely parse it, so it reports that rather than guessing.
//   - Binary older than the installed pack → a warning, never a block: an older
//     binary can still run; the user is just told to update.
//   - Otherwise → ok. Additive changes (older state read by a newer binary) are
//     always fine.
func Diagnose(projectDir, root string) (*Report, error) {
	stateSchema, err := state.MaxDeclaredSchemaVersion(root)
	if err != nil {
		return nil, err
	}
	if stateSchema == 0 {
		// Nothing on disk declares a version — treat it as the current schema;
		// undeclared state is read as the engine's own version.
		stateSchema = state.SchemaVersion
	}

	r := &Report{
		Binary:       version.Version,
		Pack:         packVersion(projectDir),
		StateSchema:  stateSchema,
		BinarySchema: state.SchemaVersion,
	}

	switch {
	case r.StateSchema > r.BinarySchema:
		r.Refuse = true
		r.Verdict = fmt.Sprintf("REFUSE: state schema v%d is newer than this binary supports (v%d) — update devrites",
			r.StateSchema, r.BinarySchema)
	case r.packSkew():
		r.Verdict = fmt.Sprintf("WARN: binary %s is older than the installed pack %s — update the devrites binary",
			r.Binary, r.Pack)
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
	fmt.Fprintf(&b, "binary: %s\n", r.Binary)
	fmt.Fprintf(&b, "pack: %s\n", r.Pack)
	fmt.Fprintf(&b, "state-schema: v%d (binary supports v%d)\n", r.StateSchema, r.BinarySchema)
	fmt.Fprintf(&b, "verdict: %s\n", r.Verdict)
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

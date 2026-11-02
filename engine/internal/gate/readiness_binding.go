package gate

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

const (
	readinessBindingLabel   = "Readiness inputs SHA-256: "
	readinessBindingDomain  = "devrites-readiness-inputs"
	readinessBindingVersion = "2"
)

type readinessInput struct {
	logical  state.ArtifactPath
	required bool
}

var readinessInputs = [...]readinessInput{
	{logical: "spec.md", required: true},
	{logical: "decision-coverage.md", required: true},
	{logical: "architecture.md", required: true},
	{logical: "plan.md", required: true},
	{logical: "tasks.md", required: true},
	{logical: "traceability.md", required: true},
	{logical: "test-plan.md", required: true},
	{logical: "strategy.md"},
	{logical: "design-brief.md"},
	{logical: "ai-spec.md"},
	{logical: ".devrites/principles.md"},
	{logical: acceptance.GatesFile},
}

// ReadinessBinding returns the exact line Vet records in eng-review.md.
// Explicit sources resolve live first, then safely archived planning bytes.
// Gate definitions are bound independently of proof progress. Older bindings
// omitted those definitions and require normal re-Vet, never automatic refresh.
func ReadinessBinding(root, slug string) (string, error) {
	observation, err := state.ObserveReferencedWorkspace(root, slug)
	if err != nil {
		return "", err
	}
	return readinessBindingFromObservation(observation)
}

func readinessBindingFromObservation(observation *state.WorkspaceObservation) (string, error) {
	digest := sha256.New()
	writeReadinessField(digest, readinessBindingDomain)
	writeReadinessField(digest, readinessBindingVersion)
	writeReadinessField(digest, fmt.Sprintf("%d", len(readinessInputs)))
	for _, input := range readinessInputs {
		content, present, err := retainedReadinessInput(observation, input)
		if err != nil {
			return "", err
		}
		if input.logical == "traceability.md" {
			content = withoutProgressCells(content)
		}
		writeReadinessField(digest, string(input.logical))
		if present {
			writeReadinessField(digest, "present")
		} else {
			writeReadinessField(digest, "absent")
		}
		writeReadinessLength(digest, uint64(len(content)))
		_, _ = digest.Write(content)
	}
	return readinessBindingLabel + hex.EncodeToString(digest.Sum(nil)), nil
}

func retainedReadinessInput(observation *state.WorkspaceObservation, input readinessInput) ([]byte, bool, error) {
	fact, ok := observation.Fact(input.logical)
	if !ok {
		return nil, false, fmt.Errorf("readiness input %s cannot be inspected", input.logical)
	}
	switch fact.State() {
	case state.ArtifactAbsent:
		required := input.required
		if input.logical == acceptance.GatesFile {
			phase, err := observation.DeclaredPhase()
			if err != nil {
				return nil, false, err
			}
			policy, _ := state.PolicyFor(phase)
			required = policyRequires(policy, input.logical)
		}
		if required {
			return nil, false, fmt.Errorf("readiness input %s is missing", input.logical)
		}
		return nil, false, nil
	case state.ArtifactEmpty, state.ArtifactPresent:
		content := fact.Bytes()
		if input.logical == acceptance.GatesFile {
			content, err := normalizedReadinessGates(content)
			return content, true, err
		}
		return content, true, nil
	case state.ArtifactMalformed, state.ArtifactUnsafe, state.ArtifactUnreadable:
		diagnostic, _ := fact.Diagnostic()
		return nil, false, readinessDiagnosticError(diagnostic)
	default:
		return nil, false, fmt.Errorf("readiness input %s cannot be inspected", input.logical)
	}
}

// Vet approves obligations, not their checkbox, evidence, or progress prose.
func normalizedReadinessGates(content []byte) ([]byte, error) {
	doc := acceptance.ParseLedger(string(content))
	if len(doc.Errors) > 0 {
		return nil, fmt.Errorf("readiness input gates.md is malformed: %s", strings.Join(doc.Errors, "; "))
	}
	slices.SortFunc(doc.Gates, func(a, b *acceptance.Gate) int {
		return strings.Compare(a.ID, b.ID)
	})
	var normalized bytes.Buffer
	writeReadinessLength(&normalized, uint64(len(doc.Gates)))
	for _, gate := range doc.Gates {
		for _, field := range [...]string{gate.ID, gate.Title, gate.Check, gate.Expect, gate.Cwd, gate.Inputs, gate.Run} {
			writeReadinessField(&normalized, field)
		}
	}
	return normalized.Bytes(), nil
}

func readinessDiagnostics(observation *state.WorkspaceObservation) []state.ArtifactDiagnostic {
	var diagnostics []state.ArtifactDiagnostic
	for _, input := range readinessInputs {
		fact, ok := observation.Fact(input.logical)
		if !ok {
			continue
		}
		if diagnostic, hasDiagnostic := fact.Diagnostic(); hasDiagnostic {
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return diagnostics
}

func verifyReadinessBinding(observation *state.WorkspaceObservation) (string, error) {
	expected, err := readinessBindingFromObservation(observation)
	if err != nil {
		return "", err
	}
	fact, ok := observation.Fact("eng-review.md")
	if !ok || fact.State() == state.ArtifactAbsent {
		return expected, errors.New("readiness input eng-review.md is missing")
	}
	if diagnostic, hasDiagnostic := fact.Diagnostic(); hasDiagnostic {
		return expected, readinessDiagnosticError(diagnostic)
	}
	visible, err := markdowntext.Structural(fact.Bytes())
	if err != nil {
		return expected, errors.New("readiness input eng-review.md is invalid Markdown")
	}

	attempts := 0
	matches := 0
	for _, raw := range strings.Split(string(visible), "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if !strings.Contains(line, readinessBindingLabel) {
			continue
		}
		attempts++
		if line == expected {
			matches++
		}
	}
	if attempts != 1 || matches != 1 {
		return expected, errors.New("eng-review.md does not contain exactly one current standalone readiness binding")
	}
	return expected, nil
}

// ReadinessInputDigest is one readiness input's content identity at a moment
// in time (canonical definitions only for gates.md). The aggregate binding
// answers "did anything change"; the per-input digest answers "which artifact
// changed", which is what a drift diagnosis and a re-vet route need.
type ReadinessInputDigest struct {
	Path    string `json:"path"`
	Present bool   `json:"present"`
	SHA256  string `json:"sha256,omitempty"`
	// Normalized is test-plan.md's digest without its reviewed-identity lines,
	// so drift can prove a Workflow Artifact byte correction changed nothing
	// else Vet approved, or traceability.md's digest without progress cells.
	Normalized string `json:"normalized,omitempty"`
}

// reviewedIdentityRE is the exact test-plan.md line binding the Workflow
// Artifact bytes an independent code review approved. Only this exact shape
// is ignored, so the exclusion cannot hide any other edit.
var reviewedIdentityRE = regexp.MustCompile(`(?m)^Workflow Artifact reviewed identity: [0-9a-f]{64}\r?\n?`)

// withoutReviewedIdentity drops every exact reviewed-identity line.
func withoutReviewedIdentity(content []byte) []byte {
	return reviewedIdentityRE.ReplaceAll(content, nil)
}

// ReadinessInputDigests returns each readiness input's content digest plus the
// aggregate binding line Vet records in eng-review.md. A required input that
// is missing, or any unreadable input, is an error — the same refusal the
// readiness gate applies. Source selection matches ReadinessBinding.
func ReadinessInputDigests(root, slug string) ([]ReadinessInputDigest, string, error) {
	observation, err := state.ObserveReferencedWorkspace(root, slug)
	if err != nil {
		return nil, "", err
	}
	binding, err := readinessBindingFromObservation(observation)
	if err != nil {
		return nil, "", err
	}
	digests := make([]ReadinessInputDigest, 0, len(readinessInputs))
	for _, input := range readinessInputs {
		content, present, err := retainedReadinessInput(observation, input)
		if err != nil {
			return nil, "", err
		}
		entry := ReadinessInputDigest{Path: string(input.logical), Present: present}
		if present {
			sum := sha256.Sum256(content)
			entry.SHA256 = hex.EncodeToString(sum[:])
			switch input.logical {
			case "test-plan.md":
				normalized := sha256.Sum256(withoutReviewedIdentity(content))
				entry.Normalized = hex.EncodeToString(normalized[:])
			case "traceability.md":
				normalized := sha256.Sum256(withoutProgressCells(content))
				entry.Normalized = hex.EncodeToString(normalized[:])
			}
		}
		digests = append(digests, entry)
	}
	return digests, binding, nil
}

// progressColumns name the traceability.md columns Build and Prove fill in as
// evidence lands. Vet approves the rows, not their progress, so the binding
// blanks these cells and keeps every other cell byte-exact.
var progressColumns = map[string]bool{"evidence": true, "status": true}

var delimiterCellRE = regexp.MustCompile(`^\s*:?-+:?\s*$`)

// withoutProgressCells blanks every Evidence and Status cell in the Markdown
// tables of content. A table is the header row, its delimiter row, and the
// table rows that follow.
func withoutProgressCells(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	var blank []int
	for i, line := range lines {
		prefix, cells, ok := tableCells(line)
		switch {
		case !ok:
			blank = nil
		case i+1 < len(lines) && delimiterRow(lines[i+1]):
			blank = nil
			for c, cell := range cells {
				if progressColumns[strings.ToLower(strings.TrimSpace(cell))] {
					blank = append(blank, c)
				}
			}
		case len(blank) > 0 && !delimiterRow(line):
			for _, c := range blank {
				if c < len(cells) {
					cells[c] = ""
				}
			}
			lines[i] = prefix + "|" + strings.Join(cells, "|") + "|"
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// tableCells splits a Markdown table row on its unescaped pipes. prefix is the
// indentation before the first pipe.
func tableCells(line string) (prefix string, cells []string, ok bool) {
	trimmed := strings.TrimRight(strings.TrimSuffix(line, "\r"), " \t")
	start := strings.Index(trimmed, "|")
	if start < 0 || strings.TrimSpace(trimmed[:start]) != "" {
		return "", nil, false
	}
	var cell strings.Builder
	for i := start + 1; i < len(trimmed); i++ {
		switch {
		case trimmed[i] == '\\' && i+1 < len(trimmed) && trimmed[i+1] == '|':
			cell.WriteString(`\|`)
			i++
		case trimmed[i] == '|':
			cells = append(cells, cell.String())
			cell.Reset()
		default:
			cell.WriteByte(trimmed[i])
		}
	}
	if cell.Len() > 0 {
		cells = append(cells, cell.String())
	}
	return trimmed[:start], cells, true
}

func delimiterRow(line string) bool {
	_, cells, ok := tableCells(line)
	if !ok || len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if !delimiterCellRE.MatchString(cell) {
			return false
		}
	}
	return true
}

func readinessDiagnosticError(diagnostic state.ArtifactDiagnostic) error {
	prefix := fmt.Sprintf("readiness input %s is %s (%s); ", diagnostic.Path, diagnostic.State, diagnostic.Code)
	repair := diagnosticRepair(diagnostic.Code)
	if repair == "" {
		repair = "restore a readable regular file"
	}
	return errors.New(prefix + repair)
}

func phaseRequiresTasks(policy state.PhasePolicy) bool {
	return policyRequires(policy, "tasks.md")
}

func phaseRequiresTestPlan(policy state.PhasePolicy) bool {
	return policyRequires(policy, "test-plan.md")
}

func phaseRequiresReadinessBinding(policy state.PhasePolicy) bool {
	return policyRequires(policy, "eng-review.md")
}

func policyRequires(policy state.PhasePolicy, name state.ArtifactPath) bool {
	for _, artifact := range policy.RequiredArtifacts {
		if artifact == name {
			return true
		}
	}
	return false
}

func writeReadinessField(digest io.Writer, value string) {
	writeReadinessLength(digest, uint64(len(value)))
	_, _ = digest.Write([]byte(value))
}

func writeReadinessLength(digest io.Writer, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = digest.Write(encoded[:])
}

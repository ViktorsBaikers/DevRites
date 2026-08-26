package gate

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strings"

	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

const (
	readinessBindingLabel   = "Readiness inputs SHA-256: "
	readinessBindingDomain  = "devrites-readiness-inputs"
	readinessBindingVersion = "1"
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
}

// ReadinessBinding returns the exact line Vet records in eng-review.md.
func ReadinessBinding(root, slug string) (string, error) {
	observation, err := state.ObserveWorkspace(root, slug)
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
		if input.required {
			return nil, false, fmt.Errorf("readiness input %s is missing", input.logical)
		}
		return nil, false, nil
	case state.ArtifactEmpty, state.ArtifactPresent:
		return fact.Bytes(), true, nil
	case state.ArtifactMalformed, state.ArtifactUnsafe, state.ArtifactUnreadable:
		diagnostic, _ := fact.Diagnostic()
		return nil, false, readinessDiagnosticError(diagnostic)
	default:
		return nil, false, fmt.Errorf("readiness input %s cannot be inspected", input.logical)
	}
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

func readinessDiagnosticError(diagnostic state.ArtifactDiagnostic) error {
	prefix := fmt.Sprintf("readiness input %s is %s (%s); ", diagnostic.Path, diagnostic.State, diagnostic.Code)
	repair := diagnosticRepair(diagnostic.Code)
	if repair == "" {
		repair = "restore a readable regular file"
	}
	return errors.New(prefix + repair)
}

func phaseRequiresTasks(policy state.PhasePolicy) bool {
	for _, artifact := range policy.RequiredArtifacts {
		if artifact == "tasks.md" {
			return true
		}
	}
	return false
}

func phaseRequiresReadinessBinding(policy state.PhasePolicy) bool {
	for _, artifact := range policy.RequiredArtifacts {
		if artifact == "eng-review.md" {
			return true
		}
	}
	return false
}

func writeReadinessField(digest hash.Hash, value string) {
	writeReadinessLength(digest, uint64(len(value)))
	_, _ = digest.Write([]byte(value))
}

func writeReadinessLength(digest hash.Hash, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = digest.Write(encoded[:])
}

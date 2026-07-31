package gate

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

const (
	readinessBindingLabel         = "Readiness inputs SHA-256: "
	readinessBindingDomain        = "devrites-readiness-inputs"
	readinessBindingVersion       = "1"
	maxReadinessInputBytes  int64 = 1 << 20
	maxReadinessTotalBytes  int64 = 8 << 20
)

type readinessInput struct {
	logical  string
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
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return "", errors.New("readiness inputs: feature workspace is unavailable")
	}

	digest := sha256.New()
	writeReadinessField(digest, readinessBindingDomain)
	writeReadinessField(digest, readinessBindingVersion)
	writeReadinessField(digest, fmt.Sprintf("%d", len(readinessInputs)))
	var total int64
	for _, input := range readinessInputs {
		path := filepath.Join(workspace, filepath.FromSlash(input.logical))
		if input.logical == ".devrites/principles.md" {
			path = filepath.Join(root, "principles.md")
		}
		content, present, err := readReadinessInput(path, input.logical, input.required)
		if err != nil {
			return "", err
		}
		total += int64(len(content))
		if total > maxReadinessTotalBytes {
			return "", fmt.Errorf("readiness input %s exceeds the 8 MiB aggregate limit", input.logical)
		}

		writeReadinessField(digest, input.logical)
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

func verifyReadinessBinding(root, slug string) (string, error) {
	expected, err := ReadinessBinding(root, slug)
	if err != nil {
		return "", err
	}
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return "", errors.New("readiness input eng-review.md cannot be inspected")
	}
	content, _, err := readReadinessInput(filepath.Join(workspace, "eng-review.md"), "eng-review.md", true)
	if err != nil {
		return expected, err
	}
	structural, err := markdowntext.Structural(content)
	if err != nil {
		return expected, errors.New("readiness input eng-review.md is invalid Markdown")
	}

	attempts := 0
	matches := 0
	for _, raw := range strings.Split(string(structural), "\n") {
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

func phaseRequiresReadinessBinding(phase state.Phase) bool {
	for _, name := range state.RequiredWorkspaceFiles(phase) {
		if name == "eng-review.md" {
			return true
		}
	}
	return false
}

func readReadinessInput(path, logical string, required bool) ([]byte, bool, error) {
	before, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if required {
			return nil, false, fmt.Errorf("readiness input %s is missing", logical)
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("readiness input %s cannot be inspected", logical)
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return nil, false, fmt.Errorf("readiness input %s is a symlink", logical)
	}
	if !before.Mode().IsRegular() {
		return nil, false, fmt.Errorf("readiness input %s is not a regular file", logical)
	}
	if before.Size() < 0 || before.Size() > maxReadinessInputBytes {
		return nil, false, fmt.Errorf("readiness input %s exceeds the 1 MiB limit", logical)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, false, fmt.Errorf("readiness input %s cannot be opened", logical)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("readiness input %s cannot be inspected", logical)
	}
	if !opened.Mode().IsRegular() || opened.Size() != before.Size() || !os.SameFile(before, opened) {
		return nil, false, fmt.Errorf("readiness input %s changed type while opening", logical)
	}
	content, err := io.ReadAll(io.LimitReader(file, maxReadinessInputBytes+1))
	if err != nil {
		return nil, false, fmt.Errorf("readiness input %s cannot be read", logical)
	}
	if int64(len(content)) > maxReadinessInputBytes {
		return nil, false, fmt.Errorf("readiness input %s exceeds the 1 MiB limit", logical)
	}
	after, err := file.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("readiness input %s cannot be inspected", logical)
	}
	current, err := os.Lstat(path)
	if err != nil {
		return nil, false, fmt.Errorf("readiness input %s cannot be inspected", logical)
	}
	if current.Mode()&os.ModeSymlink != 0 ||
		!current.Mode().IsRegular() ||
		!after.Mode().IsRegular() ||
		after.Size() != opened.Size() ||
		after.Size() != int64(len(content)) ||
		!os.SameFile(opened, after) ||
		!os.SameFile(opened, current) {
		return nil, false, fmt.Errorf("readiness input %s changed size or type while reading", logical)
	}
	if _, err := markdowntext.Structural(content); err != nil {
		return nil, false, fmt.Errorf("readiness input %s is invalid Markdown", logical)
	}
	return content, true, nil
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

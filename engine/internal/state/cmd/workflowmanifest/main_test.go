package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func TestRunWritesDeterministicManifest(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "workflow_manifest.json")

	if code := run([]string{"--out", out}, os.Stderr); code != 0 {
		t.Fatalf("run exit=%d", code)
	}
	first, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		GeneratedBy   string `json:"generatedBy"`
		SchemaVersion int    `json:"schemaVersion"`
		Phases        []struct {
			ID              state.Phase `json:"id"`
			TransitionRight string      `json:"transitionRight"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(first, &document); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	if document.SchemaVersion != state.SchemaVersion {
		t.Fatalf("schemaVersion=%d want %d", document.SchemaVersion, state.SchemaVersion)
	}
	if !strings.HasPrefix(document.GeneratedBy, "go generate ./internal/state") {
		t.Fatalf("generatedBy %q", document.GeneratedBy)
	}
	if len(document.Phases) == 0 {
		t.Fatal("manifest must list phases")
	}

	// Deterministic: a second run produces identical bytes.
	if code := run([]string{"--out", out}, os.Stderr); code != 0 {
		t.Fatalf("second run exit=%d", code)
	}
	second, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("manifest output is not deterministic")
	}
}

func TestRunCheckPassesOnCurrentAndFailsOnStale(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "workflow_manifest.json")

	if code := run([]string{"--out", out}, os.Stderr); code != 0 {
		t.Fatalf("generate exit=%d", code)
	}
	if code := run([]string{"--out", out, "--check"}, os.Stderr); code != 0 {
		t.Fatalf("check on current manifest exit=%d", code)
	}

	if err := os.WriteFile(out, []byte("{\"stale\": true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stderr := &strings.Builder{}
	if code := run([]string{"--out", out, "--check"}, stderr); code != 1 {
		t.Fatalf("check on stale manifest exit=%d want 1", code)
	}
	if !strings.Contains(stderr.String(), "is stale") {
		t.Fatalf("stderr should name the stale manifest, got %q", stderr.String())
	}
}

func TestRunCheckFailsOnMissingManifest(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent.json")
	stderr := &strings.Builder{}
	if code := run([]string{"--out", missing, "--check"}, stderr); code != 1 {
		t.Fatalf("check on missing manifest exit=%d want 1", code)
	}
}

func TestRunRejectsUnknownFlags(t *testing.T) {
	stderr := &strings.Builder{}
	if code := run([]string{"--bogus"}, stderr); code != 2 {
		t.Fatalf("unknown flag exit=%d want 2", code)
	}
}

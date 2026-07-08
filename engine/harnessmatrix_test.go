package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/harness"
)

func TestHarnessMatrixRender(t *testing.T) {
	var out, errb bytes.Buffer
	if code := cmdHarnessMatrix(nil, &out, &errb); code != exitOK {
		t.Fatalf("render exit %d", code)
	}
	if !strings.Contains(out.String(), harness.MatrixBeginMarker) {
		t.Error("rendered block missing begin marker")
	}
}

func TestHarnessMatrixCheckInSync(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "m.md")
	body := "# heading\n\n" + harness.RenderMatrix() + "\n\ntrailing prose\n"
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := cmdHarnessMatrix([]string{"--check", doc}, &out, &errb); code != exitOK {
		t.Fatalf("in-sync doc should pass, got %d: %s", code, errb.String())
	}
}

func TestHarnessMatrixCheckDrift(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "m.md")
	drifted := strings.Replace(harness.RenderMatrix(), "Native", "MADE-UP", 1)
	if err := os.WriteFile(doc, []byte(drifted), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := cmdHarnessMatrix([]string{"--check", doc}, &out, &errb); code != exitBlocked {
		t.Fatalf("drifted doc should block (exit 3), got %d", code)
	}
}

func TestHarnessMatrixCheckMissingMarkers(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "m.md")
	if err := os.WriteFile(doc, []byte("no markers here"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := cmdHarnessMatrix([]string{"--check", doc}, &out, &errb); code != exitBlocked {
		t.Fatalf("missing markers should block, got %d", code)
	}
}

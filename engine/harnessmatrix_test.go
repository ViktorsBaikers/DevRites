package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHarnessMatrixCommandIsRemoved(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"harness-matrix"}, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
		t.Fatalf("run(harness-matrix) = %d, want %d", code, exitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no success output", stdout.String())
	}
	if !strings.Contains(stderr.String(), `unknown command "harness-matrix"`) {
		t.Fatalf("stderr = %q, want unknown-command diagnostic", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("stderr = %q, want help after unknown-command diagnostic", stderr.String())
	}
	if strings.Contains(stderr.String(), "\n  devrites-engine harness-matrix ") {
		t.Fatalf("usage still advertises harness-matrix:\n%s", stderr.String())
	}
}

func TestHarnessMatrixCheckCommandIsRemoved(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"harness-matrix", "--check", "docs/harness-compliance.md"}
	if code := run(args, strings.NewReader(""), &stdout, &stderr); code != exitUsage {
		t.Fatalf("run(%q) = %d, want %d", args, code, exitUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no success output", stdout.String())
	}
	if !strings.Contains(stderr.String(), `unknown command "harness-matrix"`) {
		t.Fatalf("stderr = %q, want unknown-command diagnostic", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("stderr = %q, want help after unknown-command diagnostic", stderr.String())
	}
	if strings.Contains(stderr.String(), "\n  devrites-engine harness-matrix ") {
		t.Fatalf("usage still advertises harness-matrix:\n%s", stderr.String())
	}
}

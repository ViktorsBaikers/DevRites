package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

const goodSettings = `{
  "hooks": {
    "SessionStart": [
      { "hooks": [ { "type": "command", "command": "command -v devrites-engine >/dev/null 2>&1 && exec devrites-engine hook orient --harness=claude || exit 0" } ] }
    ],
    "PreToolUse": [
      { "matcher": "Bash", "hooks": [ { "type": "command", "command": "exec devrites-engine hook allow --harness=claude" } ] }
    ]
  }
}`

func TestValidatePackGood(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "settings.json"), goodSettings)
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{dir}, &out, &errb); code != exitOK {
		t.Fatalf("expected OK, got %d; stderr=%s", code, errb.String())
	}
}

func TestValidatePackUnknownHookID(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "settings.json"), `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"devrites-engine hook stopgate --harness=claude"}]}]}}`)
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{dir}, &out, &errb); code != 1 {
		t.Fatalf("expected exit 1 for typo'd hook id, got %d", code)
	}
	if !bytes.Contains(errb.Bytes(), []byte("stopgate")) {
		t.Errorf("expected the unknown id in the report, got: %s", errb.String())
	}
}

func TestValidatePackStructural(t *testing.T) {
	dir := t.TempDir()
	// type is not "command", and an empty command.
	testutil.WriteFile(t, filepath.Join(dir, "hooks.json"), `{"hooks":{"Stop":[{"hooks":[{"type":"prompt","command":""}]}]}}`)
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{dir}, &out, &errb); code != 1 {
		t.Fatalf("expected exit 1 for structural violations, got %d", code)
	}
}

func TestValidatePackInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "settings.json"), `{"hooks": `)
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{dir}, &out, &errb); code != 1 {
		t.Fatalf("expected exit 1 for invalid JSON, got %d", code)
	}
}

func TestValidatePackReportsUnreadableJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture requires Unix semantics")
	}
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "missing-target"), filepath.Join(dir, "unreadable.json")); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{dir}, &out, &errb); code != 1 {
		t.Fatalf("expected exit 1 for unreadable JSON, got %d", code)
	}
	if !bytes.Contains(errb.Bytes(), []byte("unreadable.json: cannot read")) {
		t.Fatalf("expected unreadable path in report, got: %s", errb.String())
	}
}

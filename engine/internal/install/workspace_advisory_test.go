package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func writeLedger(t *testing.T, target, slug, body string) {
	t.Helper()
	dir := filepath.Join(target, ".devrites", "work", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, state.LedgerFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReportStaleWorkspaces(t *testing.T) {
	target := t.TempDir()
	writeLedger(t, target, "current-feat", "# State\n\n| Key | Value |\n| --- | --- |\n| schema | "+strconv.Itoa(state.SchemaVersion)+" |\n")
	writeLedger(t, target, "legacy-feat", "# State\n\n- Phase: build\n- Next step: keep going\n")
	writeLedger(t, target, "bad-schema", "# State\n\n| Key | Value |\n| --- | --- |\n| schema | nah |\n")
	if err := os.MkdirAll(filepath.Join(target, ".devrites", "work", "remnant"), 0o755); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	reportStaleWorkspaces(out, target)
	text := out.String()
	if !strings.Contains(text, "legacy-feat") || !strings.Contains(text, "bad-schema") {
		t.Fatalf("stale workspaces not reported:\n%s", text)
	}
	if strings.Contains(text, "current-feat") || strings.Contains(text, "remnant") {
		t.Fatalf("current/remnant workspace falsely flagged:\n%s", text)
	}
	if !strings.Contains(text, "migrate <slug>") {
		t.Fatalf("remediation missing:\n%s", text)
	}
}

func TestReportStaleWorkspacesNoWorkspaces(t *testing.T) {
	out := &bytes.Buffer{}
	reportStaleWorkspaces(out, t.TempDir())
	if out.Len() != 0 {
		t.Fatalf("advisory printed with no workspaces: %q", out.String())
	}
}

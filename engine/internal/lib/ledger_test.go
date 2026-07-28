package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLedgerSyncFoldLifecycle(t *testing.T) {
	root := t.TempDir()     // the .devrites root
	wsParent := t.TempDir() // holds feature workspaces
	capSpec := ledgerCapabilityPath(root, "theming")

	sync := func(t *testing.T, slug, content string) (int, string, string) {
		t.Helper()
		writeSpec(t, wsParent, slug, content)
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := Ledger(root, []string{"sync", filepath.Join(wsParent, slug)}, stdout, stderr)
		return code, stdout.String(), stderr.String()
	}
	read := func(t *testing.T) string {
		t.Helper()
		b, err := os.ReadFile(capSpec)
		if err != nil {
			t.Fatalf("read ledger spec: %v", err)
		}
		return string(b)
	}

	// 1. ADD a new requirement: seeds the capability.
	code, out, errs := sync(t, "add-dark-mode", `## ADDED Requirements — capability: theming
### Requirement: Dark mode honors the system preference
The system SHALL default to the OS colour-scheme.
#### Scenario: first load
- **WHEN** a visitor loads the app **THEN** the theme matches the OS
`)
	if code != 0 {
		t.Fatalf("sync add code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, "+1 added") {
		t.Fatalf("want +1 added, got %q", out)
	}
	if !strings.Contains(read(t), "Dark mode honors the system preference") {
		t.Fatal("ledger missing the added requirement")
	}

	// 2. Re-sync the SAME feature: idempotent (ADDED becomes an in-place upsert).
	code, out, _ = sync(t, "add-dark-mode-again", `## ADDED Requirements — capability: theming
### Requirement: Dark mode honors the system preference
The system SHALL default to the OS colour-scheme.
#### Scenario: first load
- **WHEN** a visitor loads the app **THEN** the theme matches the OS
`)
	if code != 0 || strings.Contains(out, "+1 added") {
		t.Fatalf("re-sync should upsert, not re-add: %q", out)
	}

	// 3. MODIFY the requirement: replaces in place.
	_, out, _ = sync(t, "persist-choice", `## MODIFIED Requirements — capability: theming
### Requirement: Dark mode honors the system preference
The system SHALL default to the OS colour-scheme and persist an override.
#### Scenario: stored override
- **WHEN** a visitor has toggled the theme **THEN** the stored choice wins
`)
	if !strings.Contains(out, "~1 modified") {
		t.Fatalf("want ~1 modified, got %q", out)
	}
	if body := read(t); !strings.Contains(body, "persist an override") || strings.Contains(body, "matches the OS") {
		t.Fatalf("modify did not replace the block: %q", body)
	}

	// 4. REMOVE it: the empty capability spec is dropped.
	_, out, _ = sync(t, "drop-dark-mode", `## REMOVED Requirements — capability: theming
### Requirement: Dark mode honors the system preference
Removed: theming moved to a plugin.
`)
	if !strings.Contains(out, "-1 removed") {
		t.Fatalf("want -1 removed, got %q", out)
	}
	if _, err := os.Stat(capSpec); !os.IsNotExist(err) {
		t.Fatal("emptied capability spec should be removed")
	}
}

func TestLedgerDiffDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	wsParent := t.TempDir()
	writeSpec(t, wsParent, "feat", `## ADDED Requirements — capability: billing
### Requirement: Invoices export as PDF
The system SHALL export an invoice as a PDF.
#### Scenario: export
- **WHEN** export is clicked **THEN** a PDF is produced
`)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Ledger(root, []string{"diff", filepath.Join(wsParent, "feat")}, stdout, stderr)
	if code != 0 {
		t.Fatalf("diff code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "+1 added") {
		t.Fatalf("diff should preview the add: %q", stdout.String())
	}
	if _, err := os.Stat(ledgerCapabilityPath(root, "billing")); !os.IsNotExist(err) {
		t.Fatal("diff must not write the ledger")
	}
}

func TestLedgerListAndShow(t *testing.T) {
	root := t.TempDir()
	wsParent := t.TempDir()
	writeSpec(t, wsParent, "feat", `## ADDED Requirements — capability: search
### Requirement: Results rank by relevance
The system SHALL rank results by relevance.
#### Scenario: ranking
- **WHEN** a query runs **THEN** the top hit is the most relevant
`)
	Ledger(root, []string{"sync", filepath.Join(wsParent, "feat")}, &bytes.Buffer{}, &bytes.Buffer{})

	stdout := &bytes.Buffer{}
	if code := Ledger(root, []string{"list"}, stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("list code=%d", code)
	}
	if !strings.Contains(stdout.String(), "search") || !strings.Contains(stdout.String(), "1 requirement") {
		t.Fatalf("list output=%q", stdout.String())
	}

	stdout.Reset()
	if code := Ledger(root, []string{"show", "search"}, stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("show code=%d", code)
	}
	if !strings.Contains(stdout.String(), "Results rank by relevance") {
		t.Fatalf("show output=%q", stdout.String())
	}
}

func TestLedgerMultiCapabilityFold(t *testing.T) {
	root := t.TempDir()
	wsParent := t.TempDir()
	// One feature touches two capabilities in a single sync.
	writeSpec(t, wsParent, "feat", `## ADDED Requirements — capability: theming
### Requirement: Theme toggle exists
The system SHALL expose a theme toggle.
#### Scenario: toggle
- **WHEN** settings opens **THEN** a toggle is shown

## ADDED Requirements — capability: settings-ui
### Requirement: Settings groups appearance controls
The system SHALL group appearance controls together.
#### Scenario: grouping
- **WHEN** settings opens **THEN** appearance controls are grouped
`)
	code := Ledger(root, []string{"sync", filepath.Join(wsParent, "feat")}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("multi-cap sync code=%d", code)
	}
	for _, capability := range []string{"theming", "settings-ui"} {
		if _, err := os.Stat(ledgerCapabilityPath(root, capability)); err != nil {
			t.Fatalf("capability %q not folded: %v", capability, err)
		}
	}
}

func TestLedgerSyncDoesNotOverwriteUnreadableCapability(t *testing.T) {
	root := t.TempDir()
	capability := ledgerCapabilityPath(root, "billing")
	if err := os.MkdirAll(capability, 0o755); err != nil {
		t.Fatal(err)
	}
	wsParent := t.TempDir()
	writeSpec(t, wsParent, "feat", `## ADDED Requirements — capability: billing
### Requirement: Invoice export
The system SHALL export invoices.
`)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := Ledger(root, []string{"sync", filepath.Join(wsParent, "feat")}, stdout, stderr)
	if code != 2 {
		t.Fatalf("sync code=%d, want 2; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "read capability billing") {
		t.Fatalf("stderr=%q, want capability read error", stderr.String())
	}
	if info, err := os.Stat(capability); err != nil || !info.IsDir() {
		t.Fatalf("capability was overwritten: info=%v err=%v", info, err)
	}
}

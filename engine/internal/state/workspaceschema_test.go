package state

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func writeSchemaWorkspace(t *testing.T, stateBody string) (root, slug string) {
	t.Helper()
	root = t.TempDir()
	slug = "demo"
	if err := os.MkdirAll(featureDir(root, slug), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featureDir(root, slug), LedgerFile), []byte(stateBody), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, slug
}

func TestWorkspaceSchemaDefaultsToPreV5WhenRowIsAbsent(t *testing.T) {
	root, slug := writeSchemaWorkspace(t, "- Phase: build\n- Status: running\n")
	version, err := WorkspaceSchema(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Fatalf("version=%d want 2 (pre-v5 default)", version)
	}
	if err := RequireWorkspaceSchema(root, slug); err == nil {
		t.Fatal("pre-v5 workspace must be refused")
	} else if !strings.Contains(err.Error(), "devrites-engine migrate demo") {
		t.Fatalf("refusal must name migrate, got %v", err)
	}
}

func TestWorkspaceSchemaAcceptsCurrentAndRefusesNewer(t *testing.T) {
	root, slug := writeSchemaWorkspace(t, "| phase | build |\n| schema | 4 |\n")
	if err := RequireWorkspaceSchema(root, slug); err != nil {
		t.Fatalf("current schema must pass, got %v", err)
	}

	root, slug = writeSchemaWorkspace(t, "| phase | build |\n| schema | 9 |\n")
	err := RequireWorkspaceSchema(root, slug)
	if err == nil {
		t.Fatal("newer schema must be refused")
	}
	if !strings.Contains(err.Error(), "upgrade devrites") {
		t.Fatalf("newer-schema refusal must name the upgrade path, got %v", err)
	}
}

func TestWorkspaceSchemaRefusesMalformedRow(t *testing.T) {
	root, slug := writeSchemaWorkspace(t, "| phase | build |\n| schema | banana |\n")
	if err := RequireWorkspaceSchema(root, slug); err == nil {
		t.Fatal("malformed schema row must be refused")
	}
}

func TestWorkspaceSchemaMissingLedgerIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := RequireWorkspaceSchema(root, "absent"); err == nil {
		t.Fatal("missing workspace must be refused")
	}
}

func TestRefusalMessagesFormatSchemaVersionWithoutVersionLabel(t *testing.T) {
	root, slug := writeSchemaWorkspace(t, "| phase | build |\n| schema | 2 |\n")
	older := RequireWorkspaceSchema(root, slug)
	root, slug = writeSchemaWorkspace(t, fmt.Sprintf("| phase | build |\n| schema | %d |\n", SchemaVersion+1))
	newer := RequireWorkspaceSchema(root, slug)
	for name, err := range map[string]error{"older": older, "newer": newer} {
		if err == nil {
			t.Fatalf("%s schema must be refused", name)
		}
		if strings.Contains(err.Error(), "v5") {
			t.Fatalf("%s refusal hardcodes a version label: %v", name, err)
		}
		if want := fmt.Sprintf("%d", SchemaVersion); !strings.Contains(err.Error(), want) {
			t.Fatalf("%s refusal must format SchemaVersion %s: %v", name, want, err)
		}
	}
}

func TestStateTemplateSchemaCellMatchesEngine(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "pack", ".claude", "skills", "rite-spec", "reference", "state-workspace.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^\|\s*schema\s*\|\s*(\d+)\s*\|`).FindSubmatch(raw)
	if m == nil {
		t.Fatal("template has no schema cell")
	}
	if want := fmt.Sprintf("%d", SchemaVersion); string(m[1]) != want {
		t.Fatalf("template schema cell=%s, engine SchemaVersion=%s", m[1], want)
	}
}

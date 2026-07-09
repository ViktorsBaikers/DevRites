package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeExtSkill(t *testing.T, root, ext, frontmatter string) {
	t.Helper()
	dir := filepath.Join(root, "extensions", ext, "skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(frontmatter), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeExtAgent(t *testing.T, root, ext, frontmatter string) {
	t.Helper()
	dir := filepath.Join(root, "extensions", ext)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent.md"), []byte(frontmatter), 0o644); err != nil {
		t.Fatal(err)
	}
}

const goodSkill = "---\nname: rite-audit-lite\ndescription: A house audit rite.\n---\n\n# body\n"
const goodAgent = "---\nname: house-reviewer\ndescription: A house reviewer.\n---\n\n# body\n"

func TestExtensionsValidateGood(t *testing.T) {
	root := t.TempDir()
	writeExtSkill(t, root, "audit-lite", goodSkill)
	writeExtAgent(t, root, "audit-lite", goodAgent)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Extensions(root, []string{"validate"}, stdout, stderr); code != 0 {
		t.Fatalf("valid extension should pass, got %d\n%s%s", code, stdout, stderr)
	}
}

func TestExtensionsValidateComponentManifestPasses(t *testing.T) {
	root := t.TempDir()
	writeExtSkill(t, root, "audit-lite", goodSkill)
	writeExtManifest(t, root, "audit-lite", `schema_version: "1.0"
component:
  id: audit-lite
  kind: extension
  version: 0.1.0
  scope: project-local
  distribution: npx-managed
permissions:
  writes:
    - .devrites/**
    - .claude/**
safety:
  may_weaken_gates: false
  requires_type_go_bypass: false
`)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Extensions(root, []string{"validate"}, stdout, stderr); code != 0 {
		t.Fatalf("safe component manifest should pass, got %d\n%s%s", code, stdout, stderr)
	}
	if !contains(stdout.String(), "manifest") {
		t.Fatalf("validation should report manifest coverage, got:\n%s", stdout.String())
	}
}

func TestExtensionsValidateComponentManifestRejectsPluginDistribution(t *testing.T) {
	root := t.TempDir()
	writeExtSkill(t, root, "audit-lite", goodSkill)
	writeExtManifest(t, root, "audit-lite", `component:
  id: audit-lite
  kind: extension
  scope: project-local
  distribution: claude-plugin
`)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Extensions(root, []string{"validate"}, stdout, stderr); code != 1 {
		t.Fatalf("plugin distribution should fail, got %d\n%s%s", code, stdout, stderr)
	}
	if !contains(stderr.String(), "distribution must be npx-managed") {
		t.Fatalf("want distribution complaint, got:\n%s", stderr.String())
	}
}

func TestExtensionsValidateComponentManifestRejectsUnsafeClaims(t *testing.T) {
	root := t.TempDir()
	writeExtSkill(t, root, "audit-lite", goodSkill)
	writeExtManifest(t, root, "audit-lite", `component:
  id: audit-lite
  kind: extension
  scope: global
  distribution: npx-managed
permissions:
  writes:
    - ~/.claude/**
safety:
  may_weaken_gates: true
  requires_type_go_bypass: true
`)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Extensions(root, []string{"validate"}, stdout, stderr); code != 1 {
		t.Fatalf("unsafe component manifest should fail, got %d\n%s%s", code, stdout, stderr)
	}
	for _, want := range []string{"scope must be project-local", "write root", "may_weaken_gates must be false", "requires_type_go_bypass must be false"} {
		if !contains(stderr.String(), want) {
			t.Fatalf("want %q in stderr, got:\n%s", want, stderr.String())
		}
	}
}

func writeExtManifest(t *testing.T, root, ext, body string) {
	t.Helper()
	dir := filepath.Join(root, "extensions", ext)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "component.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExtensionsValidateMissingFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeExtSkill(t, root, "broken", "# no frontmatter here\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Extensions(root, []string{"validate"}, stdout, stderr); code != 1 {
		t.Fatalf("missing frontmatter should fail with 1, got %d\n%s%s", code, stdout, stderr)
	}
	if !contains(stderr.String(), "missing YAML frontmatter") {
		t.Fatalf("want frontmatter complaint, got:\n%s", stderr.String())
	}
}

func TestExtensionsValidateEmptyExtension(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "extensions", "hollow"), 0o755); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Extensions(root, []string{"validate"}, stdout, stderr); code != 1 {
		t.Fatalf("empty extension should fail, got %d", code)
	}
}

func TestExtensionsValidateDuplicateName(t *testing.T) {
	root := t.TempDir()
	writeExtSkill(t, root, "one", goodSkill)
	writeExtSkill(t, root, "two", goodSkill) // same skill name in both
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Extensions(root, []string{"validate"}, stdout, stderr); code != 1 {
		t.Fatalf("duplicate skill name should fail, got %d\n%s", code, stderr)
	}
}

func TestExtensionsNoneIsClean(t *testing.T) {
	root := t.TempDir()
	if code := Extensions(root, []string{"validate"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("no extensions dir should pass, got %d", code)
	}
}

func TestExtensionsSyncMirrorsIntoClaude(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	writeExtSkill(t, root, "audit-lite", goodSkill)
	writeExtAgent(t, root, "audit-lite", goodAgent)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := Extensions(root, []string{"sync"}, stdout, stderr); code != 0 {
		t.Fatalf("sync should succeed, got %d\n%s%s", code, stdout, stderr)
	}
	if !isFile(filepath.Join(project, ".claude", "skills", "audit-lite", "SKILL.md")) {
		t.Fatal("skill not mirrored into .claude/skills/")
	}
	if !isFile(filepath.Join(project, ".claude", "agents", "audit-lite.md")) {
		t.Fatal("agent not mirrored into .claude/agents/")
	}
	if isFile(filepath.Join(project, ".codex", "agents", "audit-lite.toml")) {
		t.Fatal("extension sync must not generate Codex agent mirrors")
	}
	if isFile(filepath.Join(project, ".agents", "skills", "audit-lite", "SKILL.md")) {
		t.Fatal("extension sync must not generate Codex skill mirrors")
	}
	// Idempotent second sync.
	if code := Extensions(root, []string{"sync"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("re-sync should be idempotent, got %d", code)
	}
}

func TestExtensionsSyncRefusesBroken(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	writeExtSkill(t, root, "broken", "# no frontmatter\n")
	if code := Extensions(root, []string{"sync"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("sync should refuse a broken set, got %d", code)
	}
	if isFile(filepath.Join(project, ".claude", "skills", "broken", "SKILL.md")) {
		t.Fatal("broken extension must not be synced")
	}
}

func TestExtensionsAliases(t *testing.T) {
	root := t.TempDir()
	writeExtSkill(t, root, "renamed", goodSkill)
	meta := "aliases: [old-audit, legacy-audit]\n"
	if err := os.WriteFile(filepath.Join(root, "extensions", "renamed", "extension.yaml"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	exts, err := discoverExtensions(filepath.Join(root, "extensions"))
	if err != nil {
		t.Fatal(err)
	}
	if len(exts) != 1 || len(exts[0].aliases) != 2 || exts[0].aliases[0] != "old-audit" {
		t.Fatalf("aliases not parsed: %+v", exts)
	}
}

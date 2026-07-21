package hostpack

import (
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

func TestValidatePayloadRequiresClaudeAndCodexFiles(t *testing.T) {
	payload := fstest.MapFS{
		"claude/skills/rite/SKILL.md":              {},
		"claude/agents/devrites-code-reviewer.md":  {},
		"claude/settings.json":                     {},
		"codex/skills/rite/SKILL.md":               {},
		"codex/agents/devrites-code-reviewer.toml": {},
		"codex/AGENTS.md":                          {},
		"codex/hooks.json":                         {},
	}
	if err := ValidatePayload(payload, true); err != nil {
		t.Fatal(err)
	}
	delete(payload, "codex/hooks.json")
	err := ValidatePayload(payload, true)
	if err == nil || !strings.Contains(err.Error(), "missing codex/hooks.json") {
		t.Fatalf("ValidatePayload error = %v, want missing codex/hooks.json", err)
	}
	if err := ValidatePayload(payload, false); err != nil {
		t.Fatalf("claude-only payload should not require codex files: %v", err)
	}
}

func TestInstallTreesMapPayloadsToTargets(t *testing.T) {
	got := InstallTrees(true, true, true)
	want := []Tree{
		{PayloadPrefix: "claude/skills", TargetPrefix: ".claude/skills"},
		{PayloadPrefix: "codex/skills", TargetPrefix: ".agents/skills"},
		{PayloadPrefix: "claude/agents", TargetPrefix: ".claude/agents"},
		{PayloadPrefix: "codex/agents", TargetPrefix: ".codex/agents"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("InstallTrees() = %#v, want %#v", got, want)
	}
}

func TestAliasTemplateOutput(t *testing.T) {
	data, err := RenderAliasSkill(Alias{Name: "define", To: "rite-define"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		"name: define",
		"Delegates to /rite-define.",
		"# /define - alias of /rite-define",
		"run `rite-define/SKILL.md`",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("alias template missing %q:\n%s", want, got)
		}
	}
}

func TestMergeMarkerBlockIsIdempotent(t *testing.T) {
	block := []byte("<!-- BEGIN DEVRITES CODEX -->\nDevRites\n<!-- END DEVRITES CODEX -->\n")
	current := []byte("user\n\n<!-- BEGIN DEVRITES CODEX -->\nold\n<!-- END DEVRITES CODEX -->\n")
	once := MergeMarkerBlock(current, block, "<!-- BEGIN DEVRITES CODEX -->", "<!-- END DEVRITES CODEX -->")
	twice := MergeMarkerBlock(once, block, "<!-- BEGIN DEVRITES CODEX -->", "<!-- END DEVRITES CODEX -->")
	if string(once) != string(twice) {
		t.Fatalf("merge not idempotent:\nonce:\n%s\ntwice:\n%s", once, twice)
	}
	if strings.Count(string(twice), "<!-- BEGIN DEVRITES CODEX -->") != 1 {
		t.Fatalf("merge duplicated marker:\n%s", twice)
	}
	if !strings.Contains(string(twice), "user") || strings.Contains(string(twice), "old") {
		t.Fatalf("merge did not preserve user content and replace block:\n%s", twice)
	}
}

func TestUninstallAndPruneClassification(t *testing.T) {
	entries := []string{".claude/devrites.agents-merge", ".claude/devrites.codex-hooks-merge"}
	for _, rel := range []string{"AGENTS.md", ".codex/hooks.json"} {
		if ShouldRemoveOnUninstall(rel, entries) {
			t.Fatalf("%s should be preserved because its merge marker is manifest-managed", rel)
		}
		if !PreserveOnPrune(rel) {
			t.Fatalf("%s should be preserved during prune", rel)
		}
	}
	for _, rel := range []string{"", "# comment", "/abs", "../escape", ".devrites/ACTIVE"} {
		if rel != ".devrites/ACTIVE" && ShouldRemoveOnUninstall(rel, entries) {
			t.Fatalf("%s should be rejected for uninstall removal", rel)
		}
	}
	if !PreserveOnPrune(".devrites/README.md") {
		t.Fatal(".devrites paths should be preserved during prune")
	}
	if !ShouldRemoveOnUninstall(".claude/skills/rite/SKILL.md", entries) {
		t.Fatal("managed skill should be removed during uninstall")
	}
}

func TestTemplateFSIsEmbedded(t *testing.T) {
	if _, err := fs.ReadFile(templatesFS, "templates/alias_skill.md.tmpl"); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderDevritesReadme(); err != nil {
		t.Fatal(err)
	}
}

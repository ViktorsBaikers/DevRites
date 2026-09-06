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
		"claude/skills/rite/SKILL.md":                  {},
		"claude/agents/devrites-code-reviewer.md":      {},
		"claude/workflows/devrites-readonly-review.js": {},
		"claude/settings.json":                         {},
		"codex/skills/rite/SKILL.md":                   {},
		"codex/agents/devrites-code-reviewer.toml":     {},
		"codex/AGENTS.md":                              {},
		"codex/config.toml":                            {},
		"omp/skills/rite/SKILL.md":                     {},
		"omp/agents/devrites-code-reviewer.md":         {},
		"omp/.omp-plugin/plugin.json":                  {},
	}
	if err := ValidatePayload(payload, true, true); err != nil {
		t.Fatal(err)
	}
	delete(payload, "codex/config.toml")
	err := ValidatePayload(payload, true, true)
	if err == nil || !strings.Contains(err.Error(), "missing codex/config.toml") {
		t.Fatalf("ValidatePayload error = %v, want missing codex/config.toml", err)
	}
	if err := ValidatePayload(payload, false, true); err != nil {
		t.Fatalf("claude-only payload should not require codex files: %v", err)
	}
	delete(payload, "omp/.omp-plugin/plugin.json")
	err = ValidatePayload(payload, false, true)
	if err == nil || !strings.Contains(err.Error(), "missing omp/.omp-plugin/plugin.json") {
		t.Fatalf("ValidatePayload error = %v, want missing omp/.omp-plugin/plugin.json", err)
	}
	if err := ValidatePayload(payload, false, false); err != nil {
		t.Fatalf("claude-only payload should not require omp files: %v", err)
	}
}

func TestInstallTreesMapPayloadsToTargets(t *testing.T) {
	got := InstallTrees(true, true, true, true)
	want := []Tree{
		{PayloadPrefix: "claude/skills", TargetPrefix: ".claude/skills"},
		{PayloadPrefix: "codex/skills", TargetPrefix: ".agents/skills"},
		{PayloadPrefix: "omp/skills", TargetPrefix: ".omp/skills"},
		{PayloadPrefix: "claude/agents", TargetPrefix: ".claude/agents"},
		{PayloadPrefix: "codex/agents", TargetPrefix: ".codex/agents"},
		{PayloadPrefix: "omp/agents", TargetPrefix: ".omp/agents"},
		{PayloadPrefix: "claude/workflows", TargetPrefix: ".claude/workflows"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("InstallTrees() = %#v, want %#v", got, want)
	}
	for _, trees := range [][]Tree{InstallTrees(true, false, true, true), InstallTrees(false, true, true, true)} {
		for _, tree := range trees {
			if tree.PayloadPrefix == "claude/workflows" {
				t.Fatal("Claude workflow installed without both skills and agents")
			}
		}
	}
	for _, tree := range InstallTrees(true, true, true, false) {
		if strings.HasPrefix(tree.TargetPrefix, ".omp/") {
			t.Fatalf("omp tree installed with withOmp=false: %#v", tree)
		}
	}
}

func TestAliasTargetsIncludeOmpWhenEnabled(t *testing.T) {
	got := AliasTargets(Alias{Name: "define"}, true, true)
	want := []string{
		".claude/skills/define/SKILL.md",
		".agents/skills/define/SKILL.md",
		".omp/skills/define/SKILL.md",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AliasTargets() = %#v, want %#v", got, want)
	}
	got = AliasTargets(Alias{Name: "define"}, true, false)
	for _, rel := range got {
		if strings.HasPrefix(rel, ".omp/") {
			t.Fatalf("omp alias installed with withOmp=false: %s", rel)
		}
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
	entries := []string{CodexAgentsMerge.MarkerRel, CodexConfigMerge.MarkerRel}
	for _, rel := range []string{CodexAgentsMerge.TargetRel, CodexConfigMerge.TargetRel} {
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

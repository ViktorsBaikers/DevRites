package hostpack

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/devrites/devrites/internal/devritespaths"
)

const (
	ClaudeSkillsTarget    = devritespaths.ClaudeSkillsTarget
	CodexSkillsTarget     = devritespaths.CodexSkillsTarget
	OmpSkillsTarget       = devritespaths.OmpSkillsTarget
	ClaudeAgentsTarget    = devritespaths.ClaudeAgentsTarget
	CodexAgentsTarget     = devritespaths.CodexAgentsTarget
	OmpAgentsTarget       = devritespaths.OmpAgentsTarget
	ClaudeWorkflowsTarget = devritespaths.ClaudeWorkflowsTarget
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

var templates = template.Must(template.ParseFS(templatesFS, "templates/*.tmpl"))

type Tree struct {
	PayloadPrefix string
	TargetPrefix  string
}

type Alias struct {
	Name string
	To   string
}

type MarkerMerge struct {
	TargetRel  string
	PayloadRel string
	Begin      string
	End        string
	MarkerRel  string
	MarkerText string
}

type JSONMerge struct {
	PayloadRel string
	TargetRel  string
	MarkerRel  string
	MarkerText string
	DryRunText string
}

type ManagedMerge struct {
	MarkerRel string
	TargetRel string
	Begin     string
	End       string
	DryRun    string
}

var Aliases = []Alias{
	{Name: "define", To: "rite-define"},
	{Name: "build", To: "rite-build"},
	{Name: "prove", To: "rite-prove"},
	{Name: "seal", To: "rite-seal"},
}

var CodexAgentsMerge = MarkerMerge{
	TargetRel:  "AGENTS.md",
	PayloadRel: "codex/AGENTS.md",
	Begin:      "<!-- BEGIN DEVRITES CODEX -->",
	End:        "<!-- END DEVRITES CODEX -->",
	MarkerRel:  ".claude/devrites.agents-merge",
	MarkerText: "AGENTS.md contains a DevRites managed block between BEGIN/END DEVRITES CODEX markers.",
}

var ClaudeSettingsMerge = JSONMerge{
	PayloadRel: "claude/settings.json",
	TargetRel:  ".claude/settings.json",
	MarkerRel:  ".claude/devrites.claude-hooks-merge",
	MarkerText: ".claude/settings.json contains DevRites managed permissions.",
	DryRunText: ".claude/settings.json (DevRites permissions)",
}

var CodexConfigMerge = MarkerMerge{
	TargetRel:  ".codex/config.toml",
	PayloadRel: "codex/config.toml",
	Begin:      "# BEGIN DEVRITES CODEX PERMISSIONS",
	End:        "# END DEVRITES CODEX PERMISSIONS",
	MarkerRel:  ".claude/devrites.codex-config-merge",
	MarkerText: ".codex/config.toml contains the DevRites read-only-root permission profile.",
}

var LegacyCodexHooksMerge = ManagedMerge{
	MarkerRel: ".claude/devrites.codex-hooks-merge",
	TargetRel: ".codex/hooks.json",
	DryRun:    ".codex/hooks.json legacy DevRites hooks",
}

var managedMerges = []ManagedMerge{
	{
		MarkerRel: CodexAgentsMerge.MarkerRel,
		TargetRel: CodexAgentsMerge.TargetRel,
		Begin:     CodexAgentsMerge.Begin,
		End:       CodexAgentsMerge.End,
		DryRun:    "AGENTS.md DevRites block",
	},
	{
		MarkerRel: CodexConfigMerge.MarkerRel,
		TargetRel: CodexConfigMerge.TargetRel,
		Begin:     CodexConfigMerge.Begin,
		End:       CodexConfigMerge.End,
		DryRun:    ".codex/config.toml DevRites permission block",
	},
	{
		MarkerRel: ClaudeSettingsMerge.MarkerRel,
		TargetRel: ClaudeSettingsMerge.TargetRel,
		DryRun:    ".claude/settings.json DevRites permissions",
	},
	LegacyCodexHooksMerge,
}

func RequiredPayload(withCodex, withOmp bool) []string {
	required := []string{
		"claude/skills",
		"claude/agents",
		"claude/workflows",
		"claude/settings.json",
	}
	if withCodex {
		required = append(required,
			"codex/skills",
			"codex/agents",
			"codex/AGENTS.md",
			"codex/config.toml",
		)
	}
	if withOmp {
		required = append(required,
			"omp/skills",
			"omp/agents",
			"omp/.omp-plugin/plugin.json",
		)
	}
	return required
}

func ValidatePayload(payload fs.FS, withCodex, withOmp bool) error {
	for _, rel := range RequiredPayload(withCodex, withOmp) {
		if _, err := fs.Stat(payload, rel); err != nil {
			return fmt.Errorf("missing %s", rel)
		}
	}
	return nil
}

func InstallTrees(withSkills, withAgents, withCodex, withOmp bool) []Tree {
	var trees []Tree
	if withSkills {
		trees = append(trees, Tree{PayloadPrefix: "claude/skills", TargetPrefix: ClaudeSkillsTarget})
		if withCodex {
			trees = append(trees, Tree{PayloadPrefix: "codex/skills", TargetPrefix: CodexSkillsTarget})
		}
		if withOmp {
			trees = append(trees, Tree{PayloadPrefix: "omp/skills", TargetPrefix: OmpSkillsTarget})
		}
	}
	if withAgents {
		trees = append(trees, Tree{PayloadPrefix: "claude/agents", TargetPrefix: ClaudeAgentsTarget})
		if withCodex {
			trees = append(trees, Tree{PayloadPrefix: "codex/agents", TargetPrefix: CodexAgentsTarget})
		}
		if withOmp {
			trees = append(trees, Tree{PayloadPrefix: "omp/agents", TargetPrefix: OmpAgentsTarget})
		}
	}
	if withSkills && withAgents {
		trees = append(trees, Tree{PayloadPrefix: "claude/workflows", TargetPrefix: ClaudeWorkflowsTarget})
	}
	return trees
}

func AliasTargets(alias Alias, withCodex, withOmp bool) []string {
	targets := []string{filepath.ToSlash(filepath.Join(ClaudeSkillsTarget, alias.Name, "SKILL.md"))}
	if withCodex {
		targets = append(targets, filepath.ToSlash(filepath.Join(CodexSkillsTarget, alias.Name, "SKILL.md")))
	}
	if withOmp {
		targets = append(targets, filepath.ToSlash(filepath.Join(OmpSkillsTarget, alias.Name, "SKILL.md")))
	}
	return targets
}

func RenderAliasSkill(alias Alias) ([]byte, error) {
	return render("alias_skill.md.tmpl", alias)
}

func RenderDevritesReadme() ([]byte, error) {
	return render("devrites_readme.md.tmpl", nil)
}

func ManagedMergeForMarker(markerRel string) (ManagedMerge, bool) {
	for _, merge := range managedMerges {
		if merge.MarkerRel == markerRel {
			return merge, true
		}
	}
	return ManagedMerge{}, false
}

func PreserveOnPrune(rel string) bool {
	if strings.HasPrefix(rel, ".devrites/") {
		return true
	}
	for _, merge := range managedMerges {
		if rel == merge.TargetRel {
			return true
		}
	}
	return false
}

func ShouldRemoveOnUninstall(rel string, entries []string) bool {
	if rel == "" || strings.HasPrefix(rel, "#") || strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
		return false
	}
	for _, merge := range managedMerges {
		if rel == merge.TargetRel && slices.Contains(entries, merge.MarkerRel) {
			return false
		}
	}
	return true
}

func MergeMarkerBlock(current, block []byte, begin, end string) []byte {
	lines := strings.SplitAfter(string(current), "\n")
	var out strings.Builder
	inBlock := false
	found := false
	for _, line := range lines {
		trim := strings.TrimSuffix(line, "\n")
		trim = strings.TrimSuffix(trim, "\r")
		switch trim {
		case begin:
			out.Write(block)
			if len(block) == 0 || block[len(block)-1] != '\n' {
				out.WriteByte('\n')
			}
			inBlock = true
			found = true
			continue
		case end:
			inBlock = false
			continue
		}
		if inBlock {
			continue
		}
		out.WriteString(line)
	}
	if !found {
		if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
			out.WriteByte('\n')
		}
		out.WriteByte('\n')
		out.Write(block)
		if len(block) == 0 || block[len(block)-1] != '\n' {
			out.WriteByte('\n')
		}
	}
	return []byte(out.String())
}

func render(name string, data any) ([]byte, error) {
	var out bytes.Buffer
	if err := templates.ExecuteTemplate(&out, name, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return out.Bytes(), nil
}

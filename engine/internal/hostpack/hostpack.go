package hostpack

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/devrites/devrites/internal/devritespaths"
)

const (
	ClaudeSkillsTarget = devritespaths.ClaudeSkillsTarget
	CodexSkillsTarget  = devritespaths.CodexSkillsTarget
	ClaudeAgentsTarget = devritespaths.ClaudeAgentsTarget
	CodexAgentsTarget  = devritespaths.CodexAgentsTarget
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

var CodexHooksMerge = JSONMerge{
	PayloadRel: "codex/hooks.json",
	TargetRel:  ".codex/hooks.json",
	MarkerRel:  ".claude/devrites.codex-hooks-merge",
	MarkerText: ".codex/hooks.json contains DevRites managed hook entries.",
	DryRunText: ".codex/hooks.json (DevRites hooks)",
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
		MarkerRel: ".claude/devrites.codex-config-merge",
		TargetRel: ".codex/config.toml",
		Begin:     "# BEGIN DEVRITES CODEX MCP",
		End:       "# END DEVRITES CODEX MCP",
		DryRun:    ".codex/config.toml DevRites MCP block",
	},
	{
		MarkerRel: CodexHooksMerge.MarkerRel,
		TargetRel: CodexHooksMerge.TargetRel,
		DryRun:    ".codex/hooks.json DevRites hooks",
	},
}

func RequiredPayload(withCodex bool) []string {
	required := []string{
		"claude/skills",
		"claude/agents",
		"claude/settings.json",
	}
	if withCodex {
		required = append(required,
			"codex/skills",
			"codex/agents",
			"codex/AGENTS.md",
			"codex/hooks.json",
		)
	}
	return required
}

func ValidatePayload(payload fs.FS, withCodex bool) error {
	for _, rel := range RequiredPayload(withCodex) {
		if _, err := fs.Stat(payload, rel); err != nil {
			return fmt.Errorf("missing %s", rel)
		}
	}
	return nil
}

func ValidPayload(payload fs.FS) bool {
	return ValidatePayload(payload, true) == nil
}

func InstallTrees(withSkills, withAgents, withCodex bool) []Tree {
	var trees []Tree
	if withSkills {
		trees = append(trees, Tree{PayloadPrefix: "claude/skills", TargetPrefix: ClaudeSkillsTarget})
		if withCodex {
			trees = append(trees, Tree{PayloadPrefix: "codex/skills", TargetPrefix: CodexSkillsTarget})
		}
	}
	if withAgents {
		trees = append(trees, Tree{PayloadPrefix: "claude/agents", TargetPrefix: ClaudeAgentsTarget})
		if withCodex {
			trees = append(trees, Tree{PayloadPrefix: "codex/agents", TargetPrefix: CodexAgentsTarget})
		}
	}
	return trees
}

func AliasTargets(alias Alias, withCodex bool) []string {
	targets := []string{filepath.ToSlash(filepath.Join(ClaudeSkillsTarget, alias.Name, "SKILL.md"))}
	if withCodex {
		targets = append(targets, filepath.ToSlash(filepath.Join(CodexSkillsTarget, alias.Name, "SKILL.md")))
	}
	return targets
}

func RenderAliasSkill(alias Alias) ([]byte, error) {
	return render("alias_skill.md.tmpl", alias)
}

func RenderDevritesReadme() ([]byte, error) {
	return render("devrites_readme.md.tmpl", nil)
}

func MarkerMerges(withSkills, withCodex bool) []MarkerMerge {
	if !withSkills || !withCodex {
		return nil
	}
	return []MarkerMerge{CodexAgentsMerge}
}

func JSONMerges(withSkills, withCodex bool) []JSONMerge {
	if !withSkills || !withCodex {
		return nil
	}
	return []JSONMerge{CodexHooksMerge}
}

func ManagedMergeForMarker(markerRel string) (ManagedMerge, bool) {
	for _, merge := range managedMerges {
		if merge.MarkerRel == markerRel {
			return merge, true
		}
	}
	return ManagedMerge{}, false
}

func HasManagedMarker(entries []string, markerRel string) bool {
	for _, entry := range entries {
		if entry == markerRel {
			return true
		}
	}
	return false
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
		if rel == merge.TargetRel && HasManagedMarker(entries, merge.MarkerRel) {
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
		return nil, err
	}
	return out.Bytes(), nil
}

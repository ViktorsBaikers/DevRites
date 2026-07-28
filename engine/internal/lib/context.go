package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/rootfacts"
	"github.com/devrites/devrites/internal/state"
)

const (
	contextStart = "<!-- DEVRITES START -->"
	contextEnd   = "<!-- DEVRITES END -->"
)

// Context owns only a small delimited block in project context files. It leaves
// the surrounding AGENTS.md or CLAUDE.md content unchanged.
func Context(root string, args []string, stdout, stderr io.Writer) int {
	switch argAt(args, 0) {
	case "sync":
		facts, err := rootfacts.Resolve(root)
		if err != nil {
			fmt.Fprintf(stderr, "context: unsafe root selection: %v\n", err)
			return 3
		}
		return contextSync(facts.PhysicalRoot, args[1:], stdout, stderr)
	case "show":
		return contextShow(root, args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine context sync [file ...] | context show [--json]")
		return 2
	}
}

func contextSync(root string, args []string, stdout, stderr io.Writer) int {
	targets, err := contextTargets(root, args)
	if err != nil {
		fmt.Fprintf(stderr, "context: %v\n", err)
		return 2
	}
	block := managedContextBlock(root)
	project := filepath.Dir(root)
	for _, rel := range targets {
		path := filepath.Join(project, rel)
		if err := upsertContextBlock(path, block); err != nil {
			fmt.Fprintf(stderr, "context: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "context: synced %s\n", rel)
	}
	return 0
}

type contextShowDocument struct {
	Root            string             `json:"root"`
	Project         string             `json:"project"`
	LexicalRoot     string             `json:"lexicalRoot,omitempty"`
	LexicalProject  string             `json:"lexicalProject,omitempty"`
	RootSelection   string             `json:"rootSelection"`
	Git             rootfacts.GitFacts `json:"git"`
	ActiveWorkspace string             `json:"activeWorkspace,omitempty"`
	Source          string             `json:"source"`
	HostCommands    contextCommands    `json:"hostCommands"`
	Status          []rootfacts.Hazard `json:"status"`
}

type contextCommands struct {
	Claude string `json:"claude"`
	Codex  string `json:"codex"`
}

func contextShow(root string, args []string, stdout, stderr io.Writer) int {
	jsonMode := false
	for _, a := range args {
		if a == "--json" {
			jsonMode = true
			continue
		}
		fmt.Fprintln(stderr, "usage: devrites-engine context show [--json]")
		return 2
	}
	doc := contextDocument(root)
	if jsonMode {
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "context: %v\n", err)
			return 1
		}
		_, _ = stdout.Write(append(b, '\n'))
		return 0
	}
	fmt.Fprintln(stdout, "DevRites context")
	fmt.Fprintf(stdout, "Project: %s\n", doc.Project)
	rootDisplay := doc.Root
	if rootDisplay == "" {
		rootDisplay = "none"
	}
	fmt.Fprintf(stdout, "Root: %s\n", rootDisplay)
	fmt.Fprintf(stdout, "Root selection: %s\n", doc.RootSelection)
	if doc.LexicalProject != "" && doc.LexicalProject != doc.Project {
		fmt.Fprintf(stdout, "Project (lexical): %s\n", doc.LexicalProject)
	}
	if doc.LexicalRoot != "" && doc.LexicalRoot != doc.Root {
		fmt.Fprintf(stdout, "Root (lexical): %s\n", doc.LexicalRoot)
	}
	if doc.Git.TopLevel != "" {
		fmt.Fprintf(stdout, "Git: %s (dir: %s, common: %s, linked worktree: %t, submodule: %t)\n",
			doc.Git.TopLevel, doc.Git.Dir, doc.Git.CommonDir, doc.Git.LinkedWorktree, doc.Git.Submodule)
	}
	if doc.ActiveWorkspace != "" {
		fmt.Fprintf(stdout, "Active workspace: %s (source: %s)\n", doc.ActiveWorkspace, doc.Source)
	} else {
		fmt.Fprintf(stdout, "Active workspace: none (source: %s)\n", doc.Source)
	}
	fmt.Fprintf(stdout, "Commands: Claude %s · Codex %s\n", doc.HostCommands.Claude, doc.HostCommands.Codex)
	for _, hazard := range doc.Status {
		fmt.Fprintf(stdout, "Status [%s] %s: %s\n", hazard.ID, hazard.Severity, hazard.Message)
		fmt.Fprintf(stdout, "Fix: %s\n", hazard.Remediation)
	}
	return 0
}

func contextDocument(root string) contextShowDocument {
	facts, err := rootfacts.Resolve(os.Getenv("DEVRITES_ROOT"))
	if err != nil && len(facts.Hazards) == 0 && root != "" {
		facts, _ = rootfacts.Resolve(root)
	}
	root = facts.PhysicalRoot
	project := facts.PhysicalProject
	slug := ""
	if root != "" {
		slug = activeSlug(root)
	}
	source := "none"
	active := ""
	if strings.TrimSpace(os.Getenv("DEVRITES_WORKSPACE")) != "" {
		source = "DEVRITES_WORKSPACE"
	} else if slug != "" {
		source = "ACTIVE"
	}
	if slug != "" {
		active = filepath.Join(".devrites", "work", slug)
		if rel, err := filepath.Rel(project, featureDir(root, slug)); err == nil {
			active = rel
		}
	}
	return contextShowDocument{
		Root:            root,
		Project:         project,
		LexicalRoot:     facts.LexicalRoot,
		LexicalProject:  facts.LexicalProject,
		RootSelection:   facts.SelectionReason,
		Git:             facts.Git,
		ActiveWorkspace: active,
		Source:          source,
		HostCommands:    contextCommands{Claude: "/rite", Codex: "$rite"},
		Status:          append([]rootfacts.Hazard{}, facts.Hazards...),
	}
}

func contextTargets(root string, args []string) ([]string, error) {
	if len(args) > 0 {
		return cleanContextTargets(args)
	}
	if configured := parseContextConfig(filepath.Join(root, "context.yaml")); len(configured) > 0 {
		return cleanContextTargets(configured)
	}
	project := filepath.Dir(root)
	var existing []string
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md"} {
		if isFile(filepath.Join(project, rel)) {
			existing = append(existing, rel)
		}
	}
	if len(existing) > 0 {
		return existing, nil
	}
	return []string{"AGENTS.md"}, nil
}

func parseContextConfig(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	inList := false
	for _, line := range splitLinesNoTrailing(data) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "context_files:") {
			inList = true
			continue
		}
		if inList && strings.HasPrefix(trimmed, "-") {
			out = append(out, strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")), `"'`))
			continue
		}
		inList = false
		if k, v, ok := strings.Cut(trimmed, ":"); ok && strings.TrimSpace(k) == "context_file" {
			out = append(out, strings.Trim(strings.TrimSpace(v), `"'`))
		}
	}
	return out
}

func cleanContextTargets(raw []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, rel := range raw {
		rel = filepath.Clean(strings.TrimSpace(rel))
		if rel == "." || rel == "" {
			continue
		}
		if filepath.IsAbs(rel) || strings.Contains(rel, "..") || strings.Contains(rel, `\`) {
			return nil, fmt.Errorf("unsafe context path %q", rel)
		}
		if !seen[rel] {
			seen[rel] = true
			out = append(out, rel)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no context files configured")
	}
	return out, nil
}

func managedContextBlock(root string) string {
	slug := activeSlug(root)
	lines := []string{
		contextStart,
		"DevRites project guidance:",
		"- Run `devrites-engine preamble` before DevRites workflow work.",
		"- Use `/rite` (Claude) / `$rite` (Codex) for the DevRites menu.",
	}
	if slug != "" {
		shown := filepath.Join(".devrites", "work", slug)
		if rel, err := filepath.Rel(filepath.Dir(root), featureDir(root, slug)); err == nil {
			shown = filepath.ToSlash(rel)
		}
		lines = append(lines, "- Active workspace: `"+shown+"/` (selected by `.devrites/ACTIVE` or `DEVRITES_WORKSPACE`).")
	}
	if isFile(filepath.Join(root, "principles.md")) {
		lines = append(lines, "- Project principles: `.devrites/principles.md` are binding gates.")
	}
	lines = append(lines, contextEnd, "")
	return strings.Join(lines, "\n")
}

func upsertContextBlock(path, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create context dir: %w", err)
	}
	existingBytes, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read existing context file: %w", err)
	}
	existing := string(existingBytes)
	var next string
	start := strings.Index(existing, contextStart)
	end := strings.Index(existing, contextEnd)
	if start >= 0 && end > start {
		end += len(contextEnd)
		next = strings.TrimRight(existing[:start], "\n") + "\n\n" + strings.TrimRight(block, "\n") + "\n" + strings.TrimLeft(existing[end:], "\n")
	} else if strings.TrimSpace(existing) == "" {
		next = block
	} else {
		next = strings.TrimRight(existing, "\n") + "\n\n" + block
	}
	return state.AtomicWrite(path, []byte(next), 0o644)
}

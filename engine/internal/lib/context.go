package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/state"
)

const (
	contextStart = "<!-- DEVRITES START -->"
	contextEnd   = "<!-- DEVRITES END -->"
)

// Context owns only a small delimited block in project context files. It never
// rewrites the surrounding AGENTS.md / CLAUDE.md content.
func Context(root string, args []string, stdout, stderr io.Writer) int {
	if argAt(args, 0) != "sync" {
		fmt.Fprintln(stderr, "usage: devrites-engine context sync [file ...]")
		return 2
	}
	targets, err := contextTargets(root, args[1:])
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
			shown = rel
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
		return err
	}
	existingBytes, _ := os.ReadFile(path)
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

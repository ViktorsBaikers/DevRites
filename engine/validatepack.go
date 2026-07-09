package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// validate-pack lints the shipped pack's machine-readable surfaces so a
// malformed hook wiring fails at author time, not silently at runtime. It is the
// deterministic zero-token counterpart to spec-validate, but for DevRites' OWN
// config rather than a user's spec. Two checks:
//
//  1. Structural — every settings.json / hooks/hooks.json parses, and each hook
//     entry is a well-formed {type:"command", command:"…"} (mirrors
//     schemas/hooks.schema.json, hand-checked to keep the engine dependency-free).
//  2. Referential — every `devrites-engine hook <id>` in any JSON file names a
//     hook the binary actually dispatches. A typo'd id (`hook stopgate`) would
//     otherwise no-op forever; this is the highest-value catch.
//  3. Markdown contracts (see validatepackskills.go) — SKILL.md / agent
//     frontmatter is well-formed and the naming invariant holds, and every
//     docs/command-map.md link into the pack resolves on disk.
//
// Exit: 0 clean · 1 violations found · 2 usage / unreadable dir.

// hookCommandRe pulls the hook id out of a wired command, tolerating either
// binary name and any surrounding shell guard.
var hookCommandRe = regexp.MustCompile(`devrites(?:-engine)?\s+hook\s+([a-zA-Z0-9-]+)`)

// cmdValidatePack validates the pack config tree rooted at the given dir
// (default: ./pack/.claude, else ./.claude).
func cmdValidatePack(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: devrites-engine validate-pack [pack-dir]")
		return exitUsage
	}
	dir := ""
	if len(args) == 1 {
		dir = args[0]
	} else {
		dir = defaultPackDir()
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "devrites: validate-pack: not a directory: %s\n", dir)
		return exitUsage
	}

	var issues []string
	files := 0
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				rel = path
			}
			issues = append(issues, fmt.Sprintf("%s: cannot inspect: %v", rel, err))
			return nil
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		data, err := os.ReadFile(path) // #nosec G122 -- validation walk over the pack dir the operator passed in; a symlink race requires an attacker already writing to that checkout
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: cannot read: %v", rel, err))
			return nil
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			issues = append(issues, fmt.Sprintf("%s: invalid JSON: %v", rel, err))
			return nil
		}
		files++
		// Referential: every wired hook id must be one the binary dispatches.
		for _, id := range hookIDsInCommands(data) {
			if _, ok := hookRegistry[id]; !ok {
				issues = append(issues, fmt.Sprintf("%s: unknown hook id %q — not a command the engine dispatches", rel, id))
			}
		}
		// Structural: only files that carry a `hooks` object in the .claude shape.
		if base := filepath.Base(path); base == "settings.json" || base == "hooks.json" {
			issues = append(issues, structuralIssues(rel, doc)...)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "devrites: validate-pack: %v\n", err)
		return exitUsage
	}

	// Markdown contracts: SKILL.md / agent frontmatter + naming invariant, then
	// the docs/command-map.md → pack link claims. Same issues slice/format.
	contractIssues, contracts := packContractIssues(dir)
	issues = append(issues, contractIssues...)
	issues = append(issues, docClaimIssues(dir)...)

	if len(issues) > 0 {
		sort.Strings(issues)
		fmt.Fprintf(stderr, "validate-pack: %d issue(s):\n", len(issues))
		for _, s := range issues {
			fmt.Fprintf(stderr, "  - %s\n", s)
		}
		return 1
	}
	fmt.Fprintf(stdout, "validate-pack: OK (%d JSON file(s), %d known hook ids, %d skill/agent contract(s))\n", files, len(hookRegistry), contracts)
	return exitOK
}

// defaultPackDir picks the conventional pack root relative to the working dir.
func defaultPackDir() string {
	for _, c := range []string{filepath.Join("pack", ".claude"), ".claude"} {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return filepath.Join("pack", ".claude")
}

// hookIDsInCommands returns every hook id referenced by a `devrites-engine hook
// <id>` command anywhere in the raw JSON bytes. Scanning the bytes (not the
// decoded tree) keeps it robust to harness-specific wiring shapes (Claude vs
// Codex) that place commands at different nesting depths.
func hookIDsInCommands(data []byte) []string {
	var ids []string
	for _, m := range hookCommandRe.FindAllSubmatch(data, -1) {
		ids = append(ids, string(m[1]))
	}
	return ids
}

// structuralIssues checks the .claude hooks shape: hooks -> event -> [ {matcher?,
// hooks:[ {type:"command", command:"…"} ]} ].
func structuralIssues(rel string, doc any) []string {
	var issues []string
	top, ok := doc.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: top level is not a JSON object", rel)}
	}
	hooksAny, present := top["hooks"]
	if !present {
		return nil // not a hooks file (e.g. a settings.json with only statusLine)
	}
	events, ok := hooksAny.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s: \"hooks\" is not an object", rel)}
	}
	for event, groupsAny := range events {
		groups, ok := groupsAny.([]any)
		if !ok {
			issues = append(issues, fmt.Sprintf("%s: hooks.%s is not an array", rel, event))
			continue
		}
		for i, gAny := range groups {
			g, ok := gAny.(map[string]any)
			if !ok {
				issues = append(issues, fmt.Sprintf("%s: hooks.%s[%d] is not an object", rel, event, i))
				continue
			}
			entriesAny, ok := g["hooks"]
			if !ok {
				issues = append(issues, fmt.Sprintf("%s: hooks.%s[%d] missing \"hooks\" array", rel, event, i))
				continue
			}
			entries, ok := entriesAny.([]any)
			if !ok || len(entries) == 0 {
				issues = append(issues, fmt.Sprintf("%s: hooks.%s[%d].hooks is empty or not an array", rel, event, i))
				continue
			}
			for j, eAny := range entries {
				e, ok := eAny.(map[string]any)
				if !ok {
					issues = append(issues, fmt.Sprintf("%s: hooks.%s[%d].hooks[%d] is not an object", rel, event, i, j))
					continue
				}
				if t, _ := e["type"].(string); t != "command" {
					issues = append(issues, fmt.Sprintf("%s: hooks.%s[%d].hooks[%d] type must be \"command\", got %q", rel, event, i, j, t))
				}
				if c, _ := e["command"].(string); strings.TrimSpace(c) == "" {
					issues = append(issues, fmt.Sprintf("%s: hooks.%s[%d].hooks[%d] has an empty command", rel, event, i, j))
				}
			}
		}
	}
	return issues
}

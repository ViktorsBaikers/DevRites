package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// This file extends validate-pack (see validatepack.go) from linting the pack's
// JSON hook wiring to linting its markdown *contracts* — the SKILL.md / agent
// frontmatter that Claude and Codex parse, and the docs → pack links that drift
// silently when a skill is renamed. Same output shape as the JSON pass
// (`rel: message`, appended to the shared issues slice, sorted, exit 1 if any).
//
// It is deliberately stdlib-only: the frontmatter parser is a minimal
// top-level-key scanner (no nesting, no YAML dependency), which is all the
// contract needs — name/description/user-invocable are always scalar top-level
// keys.
//
// Checks:
//
//	A. Frontmatter contract — every skills/*/SKILL.md and agents/*.md opens with
//	   a terminated `---` block carrying non-empty name + description (skills also
//	   user-invocable), and its name matches the directory (skills) or filename
//	   (agents).
//	B. Naming invariant (docs/command-map.md "Naming convention") — a devrites-*
//	   skill is model-invoked only (never user-invocable:true); a user-invocable
//	   skill lives under a rite-* directory (the bare `rite` router included).
//	C. Doc-claims — every ../pack/.claude/... link in docs/command-map.md must
//	   resolve to a file on disk, catching stale docs after a rename/move.

// docLinkRe pulls the target out of any markdown link `](target)`. We filter to
// pack/.claude targets afterwards rather than in the pattern so nested reference
// links (e.g. devrites-lib/reference/…) are covered too.
var docLinkRe = regexp.MustCompile(`\]\(([^)\s]+)\)`)

// packContractIssues runs the frontmatter + naming contract over every skill and
// agent markdown under the pack root. It returns the issues (already prefixed
// with a dir-relative path) and the number of contract files inspected.
func packContractIssues(dir string) (issues []string, checked int) {
	skillFiles, _ := filepath.Glob(filepath.Join(dir, "skills", "*", "SKILL.md"))
	for _, p := range skillFiles {
		skillDir := filepath.Base(filepath.Dir(p))
		if strings.HasPrefix(skillDir, ".") {
			continue // tooling dirs like .impeccable are not shipped skills
		}
		rel, _ := filepath.Rel(dir, p)
		checked++
		issues = append(issues, frontmatterIssues(rel, p, skillDir, true)...)
	}
	agentFiles, _ := filepath.Glob(filepath.Join(dir, "agents", "*.md"))
	for _, p := range agentFiles {
		base := filepath.Base(p)
		if strings.HasPrefix(base, ".") {
			continue
		}
		name := strings.TrimSuffix(base, ".md")
		rel, _ := filepath.Rel(dir, p)
		checked++
		issues = append(issues, frontmatterIssues(rel, p, name, false)...)
	}
	return issues, checked
}

// frontmatterIssues validates one markdown contract file. expectName is the
// directory name (skills) or the filename sans .md (agents) that `name:` must
// equal; isSkill toggles the skill-only checks (user-invocable key + the
// rite-/devrites- naming invariant).
func frontmatterIssues(rel, path, expectName string, isSkill bool) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: unreadable: %v", rel, err)}
	}
	keys, opened, terminated := parseFrontmatter(data)
	if !opened {
		return []string{fmt.Sprintf("%s: missing YAML frontmatter (file must open with a --- block)", rel)}
	}
	if !terminated {
		return []string{fmt.Sprintf("%s: unterminated frontmatter (no closing ---)", rel)}
	}

	var issues []string
	required := []string{"name", "description"}
	if isSkill {
		required = append(required, "user-invocable")
	}
	for _, k := range required {
		if strings.TrimSpace(keys[k]) == "" {
			issues = append(issues, fmt.Sprintf("%s: missing or empty required frontmatter key %q", rel, k))
		}
	}

	if got := keys["name"]; got != "" && got != expectName {
		kind := "directory"
		if !isSkill {
			kind = "filename"
		}
		issues = append(issues, fmt.Sprintf("%s: name %q must equal the %s name %q", rel, got, kind, expectName))
	}

	if isSkill {
		userInvocable := keys["user-invocable"] == "true"
		// B4: a devrites-* skill is model-invoked only.
		if strings.HasPrefix(expectName, "devrites-") && userInvocable {
			issues = append(issues, fmt.Sprintf("%s: devrites-* skills are model-invoked only — must not set user-invocable: true", rel))
		}
		// B5: a user-invocable skill lives under rite-* (the bare `rite` router included).
		if userInvocable && expectName != "rite" && !strings.HasPrefix(expectName, "rite-") {
			issues = append(issues, fmt.Sprintf("%s: user-invocable skills must use the rite-* directory prefix (per docs/command-map.md naming convention)", rel))
		}
	}
	return issues
}

// parseFrontmatter reads a leading `---` … `---` block and returns its top-level
// scalar keys. opened is true when the file starts with a `---` line; terminated
// is true when a closing `---` was found. Indented (nested), blank, and comment
// lines are skipped, so an agent's `hooks:`/`tools:` sub-tree does not pollute
// the top-level key map. Surrounding quotes on values are stripped.
func parseFrontmatter(data []byte) (keys map[string]string, opened, terminated bool) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\uFEFF") // tolerate a UTF-8 BOM
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false, false
	}
	keys = map[string]string{}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return keys, true, true
		}
		if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
			continue // blank / nested / comment — not a top-level key
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		v = strings.Trim(v, `"'`)
		keys[k] = v
	}
	return keys, true, false // opened but never closed
}

// docClaimIssues verifies every markdown link into ../pack/.claude in
// docs/command-map.md resolves to a real file. The doc is located relative to
// the repo root (the pack dir's grandparent: <root>/pack/.claude → <root>);
// if it is absent the pass is a silent no-op.
func docClaimIssues(dir string) []string {
	repoRoot := filepath.Dir(filepath.Dir(dir))
	docPath := filepath.Join(repoRoot, "docs", "command-map.md")
	data, err := os.ReadFile(docPath)
	if err != nil {
		return nil // no command-map to cross-check — skip silently
	}
	docDir := filepath.Dir(docPath)
	seen := map[string]bool{}
	var issues []string
	for _, m := range docLinkRe.FindAllStringSubmatch(string(data), -1) {
		link := m[1]
		if i := strings.IndexByte(link, '#'); i >= 0 {
			link = link[:i] // drop any anchor fragment
		}
		if link == "" || !strings.Contains(link, "pack/.claude") {
			continue // only cross-check links that point into the pack
		}
		if seen[link] {
			continue
		}
		seen[link] = true
		resolved := filepath.Clean(filepath.Join(docDir, link))
		if _, err := os.Stat(resolved); err != nil {
			issues = append(issues, fmt.Sprintf("docs/command-map.md: stale link — %q does not exist on disk", link))
		}
	}
	return issues
}

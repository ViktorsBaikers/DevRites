package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

// goodSkill / goodAgent are minimal well-formed contracts for the happy path and
// as a base to mutate in the table tests.
const goodSkill = `---
name: rite-demo
description: A demo rite skill.
user-invocable: true
---

# /rite-demo
body
`

const goodAgent = `---
name: devrites-demo-reviewer
description: A demo reviewer agent.
tools: Read, Grep
---

body
`

// TestPackContractGood: a well-formed skill + agent lint clean.
func TestPackContractGood(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "skills/rite-demo/SKILL.md"), goodSkill)
	testutil.WriteFile(t, filepath.Join(dir, "agents/devrites-demo-reviewer.md"), goodAgent)
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{dir}, &out, &errb); code != exitOK {
		t.Fatalf("expected OK, got %d; stderr=%s", code, errb.String())
	}
}

func TestPackContractFrontmatter(t *testing.T) {
	cases := []struct {
		name    string
		relPath string
		body    string
		want    string // substring expected in the violation report
	}{
		{
			name:    "missing frontmatter",
			relPath: "skills/rite-demo/SKILL.md",
			body:    "# no frontmatter here\n",
			want:    "missing YAML frontmatter",
		},
		{
			name:    "unterminated frontmatter",
			relPath: "skills/rite-demo/SKILL.md",
			body:    "---\nname: rite-demo\ndescription: x\nuser-invocable: true\n",
			want:    "unterminated frontmatter",
		},
		{
			name:    "missing description key",
			relPath: "skills/rite-demo/SKILL.md",
			body:    "---\nname: rite-demo\nuser-invocable: true\n---\n",
			want:    `required frontmatter key "description"`,
		},
		{
			name:    "skill missing user-invocable key",
			relPath: "skills/rite-demo/SKILL.md",
			body:    "---\nname: rite-demo\ndescription: x\n---\n",
			want:    `required frontmatter key "user-invocable"`,
		},
		{
			name:    "skill name/dir mismatch",
			relPath: "skills/rite-demo/SKILL.md",
			body:    "---\nname: rite-wrong\ndescription: x\nuser-invocable: true\n---\n",
			want:    "must equal the directory name",
		},
		{
			name:    "agent name/filename mismatch",
			relPath: "agents/devrites-demo-reviewer.md",
			body:    "---\nname: wrong-name\ndescription: x\n---\n",
			want:    "must equal the filename name",
		},
		{
			name:    "devrites-* skill user-invocable true",
			relPath: "skills/devrites-demo/SKILL.md",
			body:    "---\nname: devrites-demo\ndescription: x\nuser-invocable: true\n---\n",
			want:    "model-invoked only",
		},
		{
			name:    "user-invocable non-rite dir",
			relPath: "skills/oddball/SKILL.md",
			body:    "---\nname: oddball\ndescription: x\nuser-invocable: true\n---\n",
			want:    "rite-* directory prefix",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			testutil.WriteFile(t, filepath.Join(dir, tc.relPath), tc.body)
			var out, errb bytes.Buffer
			if code := cmdValidatePack([]string{dir}, &out, &errb); code != 1 {
				t.Fatalf("expected exit 1, got %d; stderr=%s", code, errb.String())
			}
			if !strings.Contains(errb.String(), tc.want) {
				t.Errorf("expected report to contain %q, got: %s", tc.want, errb.String())
			}
		})
	}
}

// The bare `rite` router is user-invocable but its directory is `rite` (not
// rite-*); the naming rule must accept it.
func TestPackContractRiteRouterAllowed(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "skills/rite/SKILL.md"), "---\nname: rite\ndescription: router\nuser-invocable: true\n---\n")
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{dir}, &out, &errb); code != exitOK {
		t.Fatalf("expected OK for bare rite router, got %d; stderr=%s", code, errb.String())
	}
}

// A devrites-* skill with user-invocable:false is legitimate and must lint clean.
func TestPackContractDevritesModelInvokedOK(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "skills/devrites-demo/SKILL.md"), "---\nname: devrites-demo\ndescription: x\nuser-invocable: false\n---\n")
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{dir}, &out, &errb); code != exitOK {
		t.Fatalf("expected OK, got %d; stderr=%s", code, errb.String())
	}
}

// TestDocClaimBrokenLink: a command-map.md link into the pack that points at a
// nonexistent file is a stale-doc violation. The doc is located at
// <repo-root>/docs/command-map.md where repo-root is the pack dir's grandparent.
func TestDocClaimBrokenLink(t *testing.T) {
	root := t.TempDir()
	packDir := root + "/pack/.claude"
	// A real skill so the linked-good case exists, plus a dangling reference.
	testutil.WriteFile(t, filepath.Join(packDir, "skills/rite-demo/SKILL.md"), goodSkill)
	testutil.WriteFile(t, filepath.Join(root, "docs/command-map.md"), strings.Join([]string{
		"[rite-demo](../pack/.claude/skills/rite-demo/SKILL.md)", // exists
		"[gone](../pack/.claude/skills/rite-ghost/SKILL.md)",     // missing
		"[external](https://example.com)",                        // ignored
	}, "\n")+"\n")
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{packDir}, &out, &errb); code != 1 {
		t.Fatalf("expected exit 1 for broken doc link, got %d; stderr=%s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "rite-ghost") {
		t.Errorf("expected the broken link in the report, got: %s", errb.String())
	}
	if strings.Contains(errb.String(), "example.com") {
		t.Errorf("external links must be ignored, got: %s", errb.String())
	}
}

// A pack with a command-map whose links all resolve lints clean.
func TestDocClaimAllResolve(t *testing.T) {
	root := t.TempDir()
	packDir := root + "/pack/.claude"
	testutil.WriteFile(t, filepath.Join(packDir, "skills/rite-demo/SKILL.md"), goodSkill)
	testutil.WriteFile(t, filepath.Join(root, "docs/command-map.md"), "[rite-demo](../pack/.claude/skills/rite-demo/SKILL.md)\n")
	var out, errb bytes.Buffer
	if code := cmdValidatePack([]string{packDir}, &out, &errb); code != exitOK {
		t.Fatalf("expected OK, got %d; stderr=%s", code, errb.String())
	}
}

package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestCanonicalSectionMarkers guards the <!-- root-only --> and
// <!-- model-surface --> markers in the shipped standards: every block
// balances and names a heading, leaf-binding sections stay unmarked, and no
// section an agent contract cites is stripped from its bundle as root-only.
func TestCanonicalSectionMarkers(t *testing.T) {
	const pack = "../../../pack/.claude"
	paths, err := filepath.Glob(filepath.Join(pack, "skills", "devrites-lib", "reference", "standards", "*.md"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no canonical standards found: %v", err)
	}
	rootOnly := map[string][]string{} // file -> headings a --role bundle drops as root-only
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(path)
		keepAll, err := stripSections(data, name, "probe", true)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		dropAll, err := stripSections(data, name, "probe", false)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		rootOnly[name] = droppedHeadings(data, keepAll)
		dropped := droppedHeadings(data, dropAll)
		for _, leaf := range map[string][]string{
			"core.md":     {"Operating rules (every phase)", "Gate contract", "Precedence"},
			"agents.md":   {"Source-writing boundary", "Independence", "Result admission", "Return budget"},
			"security.md": {"Prompt-injection resistance (agents reading untrusted input)"},
		}[name] {
			if slices.Contains(dropped, leaf) {
				t.Errorf("%s: leaf-binding section %q must not be marked", name, leaf)
			}
		}
	}

	sectionRef := regexp.MustCompile("standards/([a-z0-9-]+\\.md)[`)]?\\s*§\\s*\\**([^,.;)*\\n]+)")
	anchorRef := regexp.MustCompile(`standards/([a-z0-9-]+\.md)#([a-z0-9-]+)`)
	err = filepath.WalkDir(filepath.Join(pack, "agents"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range sectionRef.FindAllStringSubmatch(string(data), -1) {
			cited := strings.TrimSpace(m[2])
			for _, h := range rootOnly[m[1]] {
				if cited != "" && strings.HasPrefix(h, cited) {
					t.Errorf("%s cites %s § %s, a root-only section", path, m[1], cited)
				}
			}
		}
		for _, m := range anchorRef.FindAllStringSubmatch(string(data), -1) {
			for _, h := range rootOnly[m[1]] {
				if headingSlug(h) == m[2] {
					t.Errorf("%s cites %s#%s, a root-only section", path, m[1], m[2])
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestContextBundleTOCMatchesSections checks the engine-written TOC lists
// every ===== section in order with its exact line range in the final file.
func TestContextBundleTOCMatchesSections(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\nno trailing newline")
	skills := newSkillsRoot(t)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	args := []string{"feat", "--phase", "build", "--role", "slice-wright", "--trigger", "tdd", "--skills-root", skills}
	if code := RunContext(root, args, stdout, stderr); code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	raw, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build-slice-wright.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	var headers []int // 1-based line numbers
	for i, line := range lines {
		if sectionHeaderRE.MatchString(line) {
			headers = append(headers, i+1)
		}
	}
	// Contract, core, testing, spec, missing workspace artifact.
	if len(headers) != 5 || !strings.HasPrefix(lines[3], "<!-- sections:") || lines[4+len(headers)] != "" || headers[0] != len(headers)+6 {
		t.Fatalf("TOC must follow the header comments and list all %d sections:\n%s", len(headers), raw)
	}
	entryRE := regexp.MustCompile(`^L(\d+)-(\d+) (.+)$`)
	for i, h := range headers {
		m := entryRE.FindStringSubmatch(lines[4+i])
		if m == nil {
			t.Fatalf("TOC entry %d malformed: %q", i, lines[4+i])
		}
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		wantEnd := len(lines)
		if i+1 < len(headers) {
			wantEnd = headers[i+1] - 1
		}
		if start != h || end != wantEnd || lines[start-1] != "===== "+m[3]+" =====" {
			t.Fatalf("TOC entry %q: want L%d-%d for %q", lines[4+i], h, wantEnd, lines[h-1])
		}
	}
}

func TestStripSectionsRejectsMalformedMarkers(t *testing.T) {
	for name, src := range map[string]string{
		"nested":     "## A\n<!-- root-only -->\n<!-- model-surface -->\nx\n<!-- /model-surface -->\n<!-- /root-only -->\n",
		"mismatched": "## A\n<!-- root-only -->\nx\n<!-- /model-surface -->\n",
		"stray":      "## A\nx\n<!-- /root-only -->\n",
		"unclosed":   "## A\n<!-- model-surface -->\nx\n",
		"headless":   "<!-- root-only -->\nx\n<!-- /root-only -->\n",
	} {
		if _, err := stripSections([]byte(src), "f.md", "probe", false); err == nil {
			t.Errorf("%s: malformed markers accepted", name)
		}
	}
	// A marker inside a code fence is example text, not a section.
	fenced := "## A\n```\n<!-- root-only -->\n```\nkept\n"
	if out, err := stripSections([]byte(fenced), "f.md", "probe", false); err != nil || string(out) != fenced {
		t.Errorf("fenced marker changed output: %v\n%s", err, out)
	}
}

// droppedHeadings lists the heading texts in orig that stripped no longer has.
func droppedHeadings(orig, stripped []byte) []string {
	headings := func(data []byte) []string {
		var out []string
		for _, line := range strings.Split(string(data), "\n") {
			if anchorHeadingRE.MatchString(line) {
				out = append(out, strings.TrimSpace(strings.TrimLeft(line, "#")))
			}
		}
		return out
	}
	kept := headings(stripped)
	var dropped []string
	for _, h := range headings(orig) {
		if !slices.Contains(kept, h) {
			dropped = append(dropped, h)
		}
	}
	return dropped
}

// headingSlug approximates the Markdown anchor of a heading.
func headingSlug(h string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(h) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

// A --recheck bundle inlines the inventory, core, and only the standard
// sections the open findings cite; every other triggered pack file stays one
// anchored Read away. An unscoped Build reviewer bundle is refused.
func TestContextRecheckBundle(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "## SLICE-001 x\nCovers AC-001.\n")
	skills := newSkillsRoot(t)
	std := filepath.Join(skills, "devrites-lib", "reference", "standards")
	testingMD := "# Testing\n\n## Assertion strength: a test that can't fail\nstrength rule\n\n### Positive proof\nchild rule\n\n## Determinism: no flaky tests\nflake rule\n"
	security := "# Security\n\n## Input validation\nvalidation rule\n"
	writeFile(t, filepath.Join(std, "testing.md"), testingMD)
	writeFile(t, filepath.Join(std, "security.md"), security)
	writeFile(t, filepath.Join(std, "handoff.md"), "# handoff\n")
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\",\"devrites-lib/reference/standards/testing.md\",\"devrites-lib/reference/standards/handoff.md\"],"+
		"\"rootOnly\":[\"devrites-lib/reference/standards/handoff.md\"],\"triggers\":{\"security\":[\"devrites-lib/reference/standards/security.md\"]}} -->\n")
	writeFile(t, filepath.Join(featureDir, "recheck.md"), "Finding: Important conf 8 | a.go:3 | weak assert | when x → y | breaks [`testing.md`](testing.md) § Assertion strength \"quoted\" | fix\n"+
		"Finding: Minor conf 7 | a.go:9 | pyramid | when x → y | breaks testing.md § Testing pyramid | fix\n"+
		"See [security.md § Input validation](standards/security.md#input-validation).\nCorrection diff: a.go\n")
	run := func(args ...string) (int, string) {
		t.Helper()
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := RunContext(root, append([]string{"feat", "--phase", "build", "--skills-root", skills}, args...), stdout, stderr)
		return code, stdout.String() + stderr.String()
	}

	code, out := run("--role", "code-reviewer", "--slice", "SLICE-001", "--trigger", "security", "--recheck", "recheck.md")
	if code != 0 || !strings.Contains(out, "recheck: recheck.md (2 cited sections inlined, 2 pack files by reference)") {
		t.Fatalf("code=%d\n%s", code, out)
	}
	raw, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build-code-reviewer.SLICE-001.recheck.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"===== recheck inventory: recheck.md =====\nFinding: Important",
		"===== devrites-lib/reference/standards/core.md =====\n# core\nrules\n",
		"===== devrites-lib/reference/standards/testing.md (by reference) =====",
		fmt.Sprintf("sha256: %x", sha256.Sum256([]byte(testingMD))),
		"- L3 ## Assertion strength: a test that can't fail",
		"- L9 ## Determinism: no flaky tests",
		"===== devrites-lib/reference/standards/testing.md § Assertion strength: a test that can't fail (cited by the recheck inventory) =====\n## Assertion strength: a test that can't fail\nstrength rule\n\n### Positive proof\nchild rule\n\n=====",
		"===== devrites-lib/reference/standards/security.md (by reference) =====",
		fmt.Sprintf("sha256: %x", sha256.Sum256([]byte(security))),
		"- L3 ## Input validation",
		"===== devrites-lib/reference/standards/security.md § Input validation (cited by the recheck inventory) =====\n## Input validation\nvalidation rule\n",
		"===== root-only (not inlined for role code-reviewer) =====\ndevrites-lib/reference/standards/handoff.md",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("recheck bundle missing %q:\n%s", want, text)
		}
	}
	for _, reject := range []string{"flake rule", "# handoff"} {
		if strings.Contains(text, reject) {
			t.Fatalf("recheck bundle inlined uncited text %q:\n%s", reject, text)
		}
	}

	for _, tc := range []struct {
		name string
		args []string
		code int
		want string
	}{
		{"unscoped reviewer", []string{"--role", "code-reviewer"}, 3, "unscoped build reviewer bundle"},
		{"recheck without slice", []string{"--role", "slice-wright", "--recheck", "recheck.md"}, 2, "--recheck needs --phase build, --role, and --slice"},
		{"missing inventory", []string{"--role", "code-reviewer", "--slice", "SLICE-001", "--recheck", "nope.md"}, 2, "context: --recheck:"},
	} {
		if code, out := run(tc.args...); code != tc.code || !strings.Contains(out, tc.want) {
			t.Fatalf("%s: code=%d want %d and %q\n%s", tc.name, code, tc.code, tc.want, out)
		}
	}
	writeFile(t, filepath.Join(featureDir, "empty.md"), " \n")
	if code, out := run("--role", "code-reviewer", "--slice", "SLICE-001", "--recheck", "empty.md"); code != 2 || !strings.Contains(out, "empty.md is empty") {
		t.Fatalf("empty inventory: code=%d\n%s", code, out)
	}
	// The wright keeps its unscoped bundle.
	if code, out := run("--role", "slice-wright"); code != 0 {
		t.Fatalf("slice-wright without --slice: code=%d\n%s", code, out)
	}
}

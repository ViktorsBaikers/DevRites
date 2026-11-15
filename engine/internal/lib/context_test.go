package lib

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
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

// TestContextBundleTOCMatchesSections uses emitted entries, not source-shaped headers.
func TestContextBundleTOCMatchesSections(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	const source = "# spec\nUnicode café 世界\n===== fake source header =====\nno trailing newline"
	writeFile(t, filepath.Join(featureDir, "spec.md"), source)
	skills := newSkillsRoot(t)
	for _, role := range []string{"", "slice-wright"} {
		args := []string{"feat", "--phase", "build", "--trigger", "tdd", "--skills-root", skills}
		name := "build"
		labels := []string{coreStandard, "devrites-lib/reference/standards/testing.md", "workspace/spec.md", "workspace/missing.md"}
		if role != "" {
			args = append(args, "--role", role)
			name += "-" + role
			labels = append([]string{"agents/devrites-slice-wright.md"}, labels...)
		}
		stdout := contextIntegrityRun(t, root, args, 0)
		path := filepath.Join(featureDir, "ctx", name+".bundle.md")
		index := contextTestReadIndex(t, path, stdout)
		rows := contextIndexRowRE.FindAllStringSubmatch(index, -1)
		if len(rows) != len(labels) {
			t.Fatalf("want %d recorded sections, got %s", len(labels), index)
		}
		firstStart, _ := strconv.Atoi(rows[0][1])
		if firstStart != strings.Count(index, "\n")+2 {
			t.Fatalf("first section not immediately after index: %s", index)
		}
		previousEnd := 0
		for i, row := range rows {
			start, _ := strconv.Atoi(row[1])
			end, _ := strconv.Atoi(row[2])
			if row[6] != labels[i] || (i > 0 && start != previousEnd+1) {
				t.Fatalf("unexpected boundary or label: %v", row)
			}
			body := contextTestReadLines(t, path, start, end)
			contextTestSectionIdentity(t, row, body)
			if !strings.HasPrefix(body, "===== "+labels[i]+" =====\n") {
				t.Fatalf("wrong section start: %q", body)
			}
			wantReuse := "never"
			if role == "" && i < 2 {
				wantReuse = "rule"
			}
			if row[5] != wantReuse {
				t.Fatalf("%s: reuse=%s, want %s", row[6], row[5], wantReuse)
			}
			if row[6] == "workspace/spec.md" && body != "===== workspace/spec.md =====\n"+source+"\n\n" {
				t.Fatalf("source bytes changed: %q", body)
			}
			previousEnd = end
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		lines := 0
		for scanner.Scan() {
			lines++
		}
		err = scanner.Err()
		file.Close()
		if err != nil || lines != previousEnd {
			t.Fatalf("last range misses final bytes: lines=%d end=%d error=%v", lines, previousEnd, err)
		}
	}
}

var contextIndexRowRE = regexp.MustCompile(`(?m)^L([0-9]+)-([0-9]+) sha256=([0-9a-f]{64}) bytes=([0-9]+) reuse=(rule|never) (.+)$`)

func contextTestReadLines(t *testing.T, path string, start, end int) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var selected strings.Builder
	for line := 1; line <= end; line++ {
		text, err := reader.ReadString('\n')
		if err != nil && (err != io.EOF || line != end || text == "") {
			t.Fatalf("read L%d-%d at L%d: %v", start, end, line, err)
		}
		if line >= start {
			selected.WriteString(text)
		}
	}
	return selected.String()
}

func contextTestReadIndex(t *testing.T, path, stdout string) string {
	t.Helper()
	match := regexp.MustCompile(`(?m)^index: L1-([0-9]+)$`).FindStringSubmatch(stdout)
	if match == nil {
		t.Fatalf("no stdout index range: %s", stdout)
	}
	end, _ := strconv.Atoi(match[1])
	index := contextTestReadLines(t, path, 1, end)
	last := strings.Split(strings.TrimSuffix(index, "\n"), "\n")
	if !contextIndexRowRE.MatchString(last[len(last)-1]) || strings.Contains(index, "===== ") {
		t.Fatalf("index range includes body or lacks final row: %q", index)
	}
	return index
}

func contextTestSectionIdentity(t *testing.T, row []string, body string) {
	t.Helper()
	count, _ := strconv.Atoi(row[4])
	if count != len(body) || row[3] != fmt.Sprintf("%x", sha256.Sum256([]byte(body))) {
		t.Fatalf("rendered identity mismatch: %v for %q", row, body)
	}
}

func TestContextSectionIdentityAndEligibility(t *testing.T) {
	root, workspace := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	core := filepath.Join(skills, filepath.FromSlash(coreStandard))
	include := filepath.Join(filepath.Dir(core), "included.md")
	writeFile(t, core, "# core\n<!-- include:included.md -->\n")
	writeFile(t, include, "## Shared\ncafé 世界")
	args := []string{"feat", "--phase", "build", "--skills-root", skills}
	path := filepath.Join(workspace, "ctx", "build.bundle.md")
	readCore := func(args []string, path string) []string {
		index := contextTestReadIndex(t, path, contextIntegrityRun(t, root, args, 0))
		for _, row := range contextIndexRowRE.FindAllStringSubmatch(index, -1) {
			if row[6] == coreStandard {
				start, _ := strconv.Atoi(row[1])
				end, _ := strconv.Atoi(row[2])
				contextTestSectionIdentity(t, row, contextTestReadLines(t, path, start, end))
				return row
			}
		}
		t.Fatal("missing core section")
		return nil
	}
	first := readCore(args, path)
	if first[5] != "rule" {
		t.Fatal("complete expanded root rule not reusable")
	}
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "<!-- loads: {\"always\":[\"devrites-lib/reference/standards/testing.md\",\""+coreStandard+"\"]} -->\n")
	shifted := readCore(args, path)
	if shifted[1] == first[1] || shifted[3] != first[3] {
		t.Fatal("moving an unchanged rule changed its identity or did not move it")
	}
	writeFile(t, include, "## Shared\nchanged")
	changed := readCore(args, path)
	if changed[3] == first[3] {
		t.Fatal("included edit did not invalidate identity")
	}
	writeFile(t, core, "# core\n<!-- include:missing.md -->\n")
	if row := readCore(args, path); row[5] != "never" {
		t.Fatal("unresolved include marked reusable")
	}
	writeFile(t, core, "# core\n## Root\n<!-- root-only -->\nprivate root rule\n<!-- /root-only -->\n## Shared\nshared rule\n")
	rolePath := filepath.Join(workspace, "ctx", "build-slice-wright.bundle.md")
	row := readCore(append(args, "--role", "slice-wright"), rolePath)
	start, _ := strconv.Atoi(row[1])
	end, _ := strconv.Atoi(row[2])
	body := contextTestReadLines(t, rolePath, start, end)
	if row[5] != "never" || strings.Contains(body, "private root rule") || !strings.Contains(body, "shared rule") {
		t.Fatalf("role filtering/reuse changed: %s", body)
	}
}

func TestContextSectionIndexLabelsAndNonRules(t *testing.T) {
	var bundle contextBundle
	bundle.WriteString("header\n\n")
	label := strings.Repeat("長", 140) + "\r\nfull label"
	bundle.startSection(label, true)
	bundle.WriteString("Unicode 世界\n===== source header =====\n\n")
	seen := map[string]bool{}
	writeContextData(&bundle, "missing", "missing", nil, os.ErrNotExist, seen, false)
	writeContextData(&bundle, "inventory", "recheck inventory", []byte("finding\n"), nil, seen, false)
	writeAnchorIndex(&bundle, ".", "rule.md", "rule.md", []byte("## Rule\ntext\n"), nil)
	writeCitedSections(&bundle, "rule.md", []byte("## Rule\ntext\n"), []byte("rule.md § Rule"))
	rendered := insertSectionTOC(&bundle)
	rows := contextIndexRowRE.FindAllStringSubmatch(rendered, -1)
	if len(rows) != 5 || rows[0][6] != strings.ReplaceAll(strings.ReplaceAll(label, "\r", "\\r"), "\n", "\\n") {
		t.Fatalf("labels truncated, unescaped, or source heading indexed: %s", rendered)
	}
	lines := strings.SplitAfter(rendered, "\n")
	for i, row := range rows {
		start, _ := strconv.Atoi(row[1])
		end, _ := strconv.Atoi(row[2])
		contextTestSectionIdentity(t, row, strings.Join(lines[start-1:end], ""))
		if i > 0 && row[5] != "never" {
			t.Fatalf("partial or nonrule section reusable: %v", row)
		}
	}
}

func TestContextEmptySectionIndexFallback(t *testing.T) {
	var bundle contextBundle
	bundle.WriteString("fallback body\n")
	if got := insertSectionTOC(&bundle); got != "fallback body\n" {
		t.Fatalf("empty-section fallback changed: %q", got)
	}
	root, workspace := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "<!-- loads: {} -->\n")
	args := []string{"feat", "--phase", "build", "--skills-root", skills}
	stdout := contextIntegrityRun(t, root, args, 0)
	if strings.Contains(stdout, "index:") {
		t.Fatalf("empty bundle advertises index: %s", stdout)
	}
	body, err := os.ReadFile(filepath.Join(workspace, "ctx", "build.bundle.md"))
	if err != nil || !bytes.Contains(body, []byte("devrites context bundle")) || bytes.Contains(body, []byte("<!-- sections:")) {
		t.Fatalf("complete-read fallback unavailable: %s %v", body, err)
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

func TestContextVetRecheckBundle(t *testing.T) {
	root, featureDir := newWorkspace(t, "vet")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-vet", "SKILL.md"), `<!-- loads: {"always":["devrites-lib/reference/standards/core.md","devrites-lib/reference/standards/testing.md"],"workspace":["tasks.md","plan.md","traceability.md"]} -->`)
	writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "testing.md"), "# Testing\n\n## Closure\nclosure rule\n\n## Other\nuncited rule\n")
	for _, role := range []string{"plan-reviewer", "security-auditor", "test-analyst"} {
		writeFile(t, filepath.Join(skills, "..", "agents", "devrites-"+role+".md"), "# Independent reviewer\n")
	}
	tasks := "## SLICE-001 First\nfirst body AC-001\n\n## SLICE-002 Second\nsecond body AC-002\n\n## SLICE-003 Third\nthird body AC-003\n"
	writeFile(t, filepath.Join(featureDir, "tasks.md"), tasks)
	plan := strings.Repeat("whole plan content\n", 1100)
	writeFile(t, filepath.Join(featureDir, "plan.md"), plan)
	writeFile(t, filepath.Join(featureDir, "traceability.md"), "AC-003 dependency closure\n")
	writeFile(t, filepath.Join(featureDir, "recheck.md"), "Neutral reproduction AC-003; correction scope; testing.md § Closure\n")
	writeFile(t, filepath.Join(featureDir, "empty.md"), " \n")
	if err := os.Mkdir(filepath.Join(featureDir, "inventory-dir"), 0700); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (int, string) {
		t.Helper()
		var out bytes.Buffer
		code := RunContext(root, append([]string{"--skills-root", skills}, args...), &out, &out)
		return code, out.String()
	}
	for _, selector := range [][]string{{"--phase", "vet"}, {"--skill", "rite-vet"}} {
		for _, ids := range [][]string{nil, {"SLICE-001"}, {"SLICE-002", "SLICE-001"}, {"SLICE-001", "SLICE-001"}} {
			args := append([]string{"feat"}, selector...)
			args = append(args, "--role", "plan-reviewer", "--recheck", "recheck.md")
			name := "vet-plan-reviewer"
			for _, id := range ids {
				args = append(args, "--slice", id)
			}
			if len(ids) > 0 {
				name += "." + strings.Join(ids, "+")
			}
			code, out := run(args...)
			if code != 0 {
				t.Fatalf("%v: code=%d\n%s", args, code, out)
			}
			path := filepath.Join(featureDir, "ctx", name+".recheck.bundle.md")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(raw)
			for _, want := range []string{"phase=vet", "Neutral reproduction AC-003", "closure rule", "SLICE-003 Third", "AC-003 dependency closure"} {
				if !strings.Contains(text, want) {
					t.Fatalf("%v: missing %q", args, want)
				}
			}
			if strings.Contains(text, "uncited rule") {
				t.Fatal("uncited rule inlined")
			}
			if len(ids) == 0 {
				if !strings.Contains(text, tasks) || !strings.Contains(text, plan) || strings.Contains(out, "scope: slice") {
					t.Fatal("unsliced Vet recovery narrowed workspace")
				}
			} else {
				if strings.Contains(text, "third body") {
					t.Fatal("unselected slice body inlined")
				}
				for _, id := range ids {
					body := "first body"
					if id == "SLICE-002" {
						body = "second body"
					}
					if !strings.Contains(text, body) {
						t.Fatalf("selected slice %s missing", id)
					}
				}
			}
			index := contextTestReadIndex(t, path, out)
			for _, row := range contextIndexRowRE.FindAllStringSubmatch(index, -1) {
				if row[5] != "never" {
					t.Fatalf("recovery section reusable: %s", row[0])
				}
			}
		}
	}
	for _, role := range []string{"security-auditor", "test-analyst"} {
		if code, out := run("feat", "--phase", "vet", "--role", role, "--recheck", "recheck.md"); code != 0 {
			t.Fatalf("%s: code=%d\n%s", role, code, out)
		}
	}
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"roleless", []string{"feat", "--phase", "vet", "--recheck", "recheck.md"}, 2},
		{"drafter", []string{"feat", "--phase", "vet", "--role", "plan-drafter", "--recheck", "recheck.md"}, 2},
		{"wright", []string{"feat", "--phase", "vet", "--role", "slice-wright", "--recheck", "recheck.md"}, 2},
		{"no slug", []string{"--skill", "rite-vet", "--role", "plan-reviewer", "--out", "unused.md", "--recheck", "recheck.md"}, 2},
		{"missing value", []string{"feat", "--phase", "vet", "--role", "plan-reviewer", "--recheck"}, 2},
		{"empty value", []string{"feat", "--phase", "vet", "--role", "plan-reviewer", "--recheck", ""}, 2},
		{"flag value", []string{"feat", "--phase", "vet", "--role", "plan-reviewer", "--recheck", "--whole"}, 2},
		{"missing inventory", []string{"feat", "--phase", "vet", "--role", "plan-reviewer", "--recheck", "missing.md"}, 2},
		{"empty inventory", []string{"feat", "--phase", "vet", "--role", "plan-reviewer", "--recheck", "empty.md"}, 2},
		{"nonregular inventory", []string{"feat", "--phase", "vet", "--role", "plan-reviewer", "--recheck", "inventory-dir"}, 2},
		{"invalid slice", []string{"feat", "--phase", "vet", "--role", "plan-reviewer", "--recheck", "recheck.md", "--slice", "bad"}, 2},
		{"unknown slice", []string{"feat", "--skill", "rite-vet", "--role", "plan-reviewer", "--recheck", "recheck.md", "--slice", "SLICE-999"}, 3},
		{"review", []string{"feat", "--phase", "review", "--role", "plan-reviewer", "--recheck", "recheck.md", "--slice", "SLICE-001"}, 2},
		{"skill build", []string{"feat", "--skill", "rite-build", "--role", "code-reviewer", "--recheck", "recheck.md", "--slice", "SLICE-001"}, 2},
		{"unsliced build", []string{"feat", "--phase", "build", "--role", "code-reviewer", "--recheck", "recheck.md"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if code, out := run(tc.args...); code != tc.code {
				t.Fatalf("code=%d want %d\n%s", code, tc.code, out)
			}
		})
	}
	if code, out := run("feat", "--skill", "rite-vet", "--role", "plan-reviewer"); code != 0 {
		t.Fatalf("ordinary skill context: code=%d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(featureDir, "ctx", "rite-vet-plan-reviewer.bundle.md")); err != nil {
		t.Fatalf("ordinary skill filename changed: %v", err)
	}
}

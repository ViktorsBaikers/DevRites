package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const anchorsGoodBreaks = `breaks AC-1 "checks the caller's role before returning user data"`

// anchorsProject builds a project with source, a workspace spec/decisions and
// one shipped standard; it returns the .devrites root and the workspace.
func anchorsProject(t *testing.T) (root, featureDir string) {
	t.Helper()
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	root = filepath.Join(project, ".devrites")
	featureDir = filepath.Join(root, "work", "feat")
	writeFile(t, filepath.Join(featureDir, "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# Spec\n\n- AC-1: Every handler checks the caller's\n  role before returning user data.\n")
	writeFile(t, filepath.Join(featureDir, "decisions.md"), "DEC-1: Admins bypass the role check only through Root.\n")
	writeFile(t, filepath.Join(project, "pack/.claude/skills/devrites-lib/reference/standards/security.md"),
		"## Reachability sets severity\nSeverity follows the trust boundary an attacker can reach.\n")
	writeFile(t, filepath.Join(project, "src/a.go"), strings.Join([]string{
		"package a",
		"",
		"func Handler(r *Request) User {",
		"\treturn r.User",
		"}",
		`func Admin(r *Request) bool { return r.Role == "admin" || r.Root }`,
		`func Dup() { log.Println("duplicate line here") }`,
		`func Dup2() { log.Println("duplicate line here") }`,
	}, "\n")+"\n")
	writeFile(t, filepath.Join(dir, "secret.go"), "top secret value\n")
	return root, featureDir
}

func anchorsRow(severity, location, observed, breaks string) string {
	return fmt.Sprintf("Finding: %s conf 9 | %s | %s | when an anonymous caller hits Handler → user data leaks | %s | add the role check", severity, location, observed, breaks)
}

func checkAnchors(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunCheckAnchors(root, args, stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestCheckAnchorsRows(t *testing.T) {
	cases := []struct {
		name, location, observed, breaks string
		code                             int
		want                             string
	}{
		{"exact", "src/a.go:4", "`return r.User` returns with no role check", anchorsGoodBreaks, 0, "anchored src/a.go:4"},
		{"whitespace only differs", "src/a.go:4", "`  return    r.User `", anchorsGoodBreaks, 0, "anchored src/a.go:4"},
		{"backticked range", "`src/a.go:3-4`", "`Handler(r *Request) User { return r.User`", anchorsGoodBreaks, 0, "anchored src/a.go:3-4"},
		{"pipe inside quote", "src/a.go:6", "`r.Role == \"admin\" || r.Root`", anchorsGoodBreaks, 0, "anchored src/a.go:6"},
		{"relocated", "src/a.go:2", "`return r.User`", anchorsGoodBreaks, 0, "relocated src/a.go:2 -> src/a.go:4"},
		{"workspace artifact section", "spec.md § AC-1", "`- AC-1: Every handler checks the caller's`", anchorsGoodBreaks, 0, "anchored spec.md:3"},
		{"fabricated line", "src/a.go:99", "`return r.Admin`", anchorsGoodBreaks, 3, "— anchor lost"},
		{"two locations", "src/a.go:1", "`log.Println(\"duplicate line here\")`", anchorsGoodBreaks, 3, "— anchor ambiguous"},
		{"too short", "src/a.go:4", "`return`", anchorsGoodBreaks, 3, "anchor ambiguous: quote under 8 characters"},
		{"path outside project", "../secret.go:1", "`top secret value`", anchorsGoodBreaks, 3, "../secret.go is not a file under the workspace or project"},
		{"no quote", "src/a.go:4", "returns with no role check", anchorsGoodBreaks, 3, "— missing quote"},
		{"no file:line", "go test ./...", "`exit 1`", anchorsGoodBreaks, 3, "— missing file:line"},
		{"breaks quote absent", "src/a.go:4", "`return r.User`", `breaks AC-1 "every request is rate limited"`, 3, "— breaks anchor lost"},
		{"breaks without quote", "src/a.go:4", "`return r.User`", "breaks AC-1", 3, "Deferred: "},
		{"no breaks cell", "src/a.go:4", "`return r.User`", "no spec line", 3, "— no spec anchor"},
		{"breaks decision", "src/a.go:4", "`return r.User`", `breaks DEC-1 "Admins bypass the role check only through Root"`, 0, "anchored src/a.go:4"},
		{"breaks standard", "src/a.go:4", "`return r.User`", `breaks security.md § Reachability sets severity "Severity follows the trust boundary an attacker can reach"`, 0, "anchored src/a.go:4"},
		{"breaks qualified standard line", "src/a.go:4", "`return r.User`", `breaks pack/.claude/skills/devrites-lib/reference/standards/security.md:2 § Reachability sets severity "Severity follows the trust boundary an attacker can reach"`, 0, "anchored src/a.go:4"},
		{"standard line cannot use spec quote", "src/a.go:4", "`return r.User`", `breaks security.md:2 § Reachability sets severity "checks the caller's role before returning user data"`, 3, "breaks anchor lost"},
		{"breaks non-standard file", "src/a.go:4", "`return r.User`", `breaks contract.md § X "Severity follows the trust boundary an attacker can reach"`, 3, "breaks anchor lost: contract.md § X is unreadable"},
		{"first path with line", "src/a.go:4", "`return r.User`", `breaks contract.md:2 then security.md § X "Severity follows the trust boundary an attacker can reach"`, 3, "unverified=1"},
		{"first path unquoted", "src/a.go:4", "`return r.User`", "breaks contract.md then `security.md` § X \"Severity follows the trust boundary an attacker can reach\"", 3, "unverified=1"},
		{"first path quoted", "src/a.go:4", "`return r.User`", "breaks `security.md` then contract.md § X \"Severity follows the trust boundary an attacker can reach\"", 0, "anchored src/a.go:4"},
		{"breaks curly quotes", "src/a.go:4", "`return r.User`", "breaks AC-1 “checks the caller's role before returning user data”", 0, "anchored src/a.go:4"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root, featureDir := anchorsProject(t)
			ret := writeReturn(t, featureDir, "Outcome: findings\nCounts: critical=1\n"+anchorsRow("Critical", c.location, c.observed, c.breaks)+"\n")
			code, out := checkAnchors(t, root, "feat", ret)
			if code != c.code || !strings.Contains(out, c.want) {
				t.Fatalf("want %d with %q, got %d:\n%s", c.code, c.want, code, out)
			}
		})
	}
}

func writeReturn(t *testing.T, featureDir, body string) string {
	t.Helper()
	p := filepath.Join(featureDir, "return.md")
	writeFile(t, p, body)
	return p
}

// Only Critical/Important rows are anchored; the summary counts each class and
// lands in metrics.jsonl.
func TestCheckAnchorsSummaryAndMetric(t *testing.T) {
	root, featureDir := anchorsProject(t)
	ret := writeReturn(t, featureDir, strings.Join([]string{
		"Outcome: findings",
		"Counts: critical=1 important=1 suggestion=1",
		anchorsRow("Critical", "src/a.go:4", "`return r.User`", anchorsGoodBreaks),
		anchorsRow("Important", "src/a.go:99", "`return r.Admin`", anchorsGoodBreaks),
		"Finding: Suggestion | nowhere | prose | when never → nothing | none | none",
		"Basis: read src/a.go",
	}, "\r\n"))
	code, out := checkAnchors(t, root, "feat", ret)
	if code != 3 || !strings.Contains(out, "BLOCKED: 2 Critical/Important rows, anchored=1 relocated=0 unverified=1 deferred=0") ||
		!strings.Contains(out, "4: Unverified: Important conf 9 | src/a.go:99") {
		t.Fatalf("got %d:\n%s", code, out)
	}
	metrics, err := os.ReadFile(filepath.Join(featureDir, MetricsFile))
	if err != nil || !strings.Contains(string(metrics), `"event":"anchors"`) || !strings.Contains(string(metrics), "unverified=1") {
		t.Fatalf("metrics: %v %s", err, metrics)
	}

	ret = writeReturn(t, featureDir, "Outcome: findings\nCounts: critical=0 important=1\n\nImportant: the handler skips the role check.\n")
	if code, out := checkAnchors(t, root, "feat", ret); code != 3 || !strings.Contains(out, "BLOCKED: Counts declares 1 Critical/Important but 0 are Finding: rows") {
		t.Fatalf("prose blocker: %d %s", code, out)
	}

	ret = writeReturn(t, featureDir, "Outcome: findings\nCounts: suggestion=1\nFinding: Suggestion | nowhere | prose | x | y | z\n")
	if code, out := checkAnchors(t, root, "feat", ret); code != 0 || !strings.Contains(out, "OK: 0 Critical/Important rows") {
		t.Fatalf("suggestion-only: %d %s", code, out)
	}
}

// Format drift never skips a blocker row: bulleted, bold or colon/paren
// severities are still anchored (here, a fabricated line blocks).
func TestCheckAnchorsRowFormats(t *testing.T) {
	rest := " | src/a.go:99 | `return r.Admin` | when x → y | " + anchorsGoodBreaks + " | fix"
	for _, row := range []string{
		"- Finding: Critical conf 9" + rest,
		"* Finding: important conf 9" + rest,
		"Finding: **Critical** conf 9" + rest,
		"**Finding:** Critical conf 9" + rest,
		"Finding: Critical: conf 9" + rest,
		"Finding: Important(conf 9)" + rest,
	} {
		root, featureDir := anchorsProject(t)
		ret := writeReturn(t, featureDir, "Outcome: findings\n"+row+"\n")
		if code, out := checkAnchors(t, root, "feat", ret); code != 3 || !strings.Contains(out, "unverified=1") {
			t.Fatalf("%q: want exit 3 unverified=1, got %d:\n%s", row, code, out)
		}
	}
}

// A host install passes --skills-root; standards resolve there, not in the default tree.
func TestCheckAnchorsSkillsRoot(t *testing.T) {
	root, featureDir := anchorsProject(t)
	writeFile(t, filepath.Join(filepath.Dir(root), ".agents/skills/devrites-lib/reference/standards/host.md"),
		"Host standards ship with the host install.\n")
	ret := writeReturn(t, featureDir, anchorsRow("Critical", "src/a.go:4", "`return r.User`",
		`breaks host.md § Install "Host standards ship with the host install"`)+"\n")
	if code, out := checkAnchors(t, root, "feat", ret, "--skills-root", ".agents/skills"); code != 0 || !strings.Contains(out, "anchored src/a.go:4") {
		t.Fatalf("with --skills-root: %d %s", code, out)
	}
	if code, out := checkAnchors(t, root, "feat", ret); code != 3 || !strings.Contains(out, "host.md § Install is unreadable") {
		t.Fatalf("default skills root: %d %s", code, out)
	}
}

func TestCheckAnchorsStandardPaths(t *testing.T) {
	for _, skillsRoot := range []string{"pack/.claude/skills", ".agents/skills", "custom skills"} {
		t.Run(skillsRoot, func(t *testing.T) {
			root, featureDir := anchorsProject(t)
			project := filepath.Dir(root)
			standardsRel := skillsRoot + "/devrites-lib/reference/standards"
			standards := filepath.Join(project, standardsRel)
			const rule = "Acquired resource without a matching release on every exit path"
			writeFile(t, filepath.Join(standards, "review/default.md"), rule+"\n")
			writeFile(t, filepath.Join(standards, "default.md"), "A different flat standard must not supply the nested rule.\n")
			writeFile(t, filepath.Join(standards, "security.md"), rule+"\n")
			writeFile(t, filepath.Join(standards, "../security.md"), rule+"\n")
			writeFile(t, filepath.Join(project, "plan.md"), rule+"\n")
			writeFile(t, filepath.Join(project, "other/skills/devrites-lib/reference/standards/security.md"), rule+"\n")
			outside := filepath.Join(project, "outside")
			writeFile(t, filepath.Join(outside, "default.md"), rule+"\n")
			if err := os.Symlink(filepath.Join(outside, "default.md"), filepath.Join(standards, "linked.md")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(standards, "linked")); err != nil {
				t.Fatal(err)
			}
			cases := []struct {
				name, path string
				code       int
			}{
				{"nested", "review/default.md", 0},
				{"qualified nested", standardsRel + "/review/default.md", 0},
				{"qualified nested line", standardsRel + "/review/default.md:166", 0},
				{"qualified nested range", standardsRel + "/review/default.md:166-170", 0},
				{"native qualified nested", filepath.Join(standardsRel, "review", "default.md"), 0},
				{"absolute nested", filepath.Join(standards, "review/default.md"), 0},
				{"flat", "security.md", 0},
				{"qualified flat", standardsRel + "/security.md", 0},
				{"absolute flat", filepath.Join(standards, "security.md"), 0},
				{"absolute flat line", filepath.Join(standards, "security.md") + ":66", 0},
				{"wrong nested rule", "missing/default.md", 3},
				{"wrong authority", "other/skills/devrites-lib/reference/standards/security.md", 3},
				{"project artifact", "plan.md", 3},
				{"absolute project artifact", filepath.Join(project, "plan.md"), 3},
				{"parent traversal", "../security.md", 3},
				{"parent traversal line", "../security.md:66", 3},
				{"internal traversal", "review/../security.md", 3},
				{"backslash traversal", `review\..\security.md`, 3},
				{"qualified traversal", standardsRel + "/review/../security.md", 3},
				{"file symlink escape", "linked.md", 3},
				{"file symlink escape range", "linked.md:1-2", 3},
				{"directory symlink escape", "linked/default.md", 3},
			}
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					ret := writeReturn(t, featureDir, anchorsRow("Critical", "src/a.go:4", "`return r.User`",
						fmt.Sprintf("breaks `%s` § Resource lifecycle %q", c.path, rule))+"\n")
					code, out := checkAnchors(t, root, "feat", ret, "--skills-root", skillsRoot)
					want := "anchored src/a.go:4"
					if c.code == 3 {
						want = "unverified=1"
					}
					if code != c.code || !strings.Contains(out, want) {
						t.Fatalf("want exit %d with %q, got %d:\n%s", c.code, want, code, out)
					}
				})
			}
		})
	}
}

func TestCheckAnchorsUsage(t *testing.T) {
	root, featureDir := anchorsProject(t)
	if code, out := checkAnchors(t, root, "feat"); code != 2 || !strings.Contains(out, "usage:") {
		t.Fatalf("usage: %d %s", code, out)
	}
	if code, out := checkAnchors(t, root, "feat", filepath.Join(featureDir, "spec.md"), "--skills-root"); code != 2 || !strings.Contains(out, "usage:") {
		t.Fatalf("dangling --skills-root: %d %s", code, out)
	}
	if code, _ := checkAnchors(t, root, "feat", filepath.Join(featureDir, "missing.md")); code != 2 {
		t.Fatalf("missing return exit %d", code)
	}
	if code, _ := checkAnchors(t, root, "nope", filepath.Join(featureDir, "spec.md")); code != 2 {
		t.Fatalf("missing workspace exit %d", code)
	}
}

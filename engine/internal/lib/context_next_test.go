package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newWorkspace(t *testing.T, phase string) (root, featureDir string) {
	t.Helper()
	root = t.TempDir()
	featureDir = filepath.Join(root, "work", "feat")
	state := "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | " + phase + " |\n| status | running |\n"
	writeFile(t, filepath.Join(featureDir, "state.md"), state)
	return root, featureDir
}

func TestMetricsRecordAndSummary(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunMetrics(root, []string{"record", "feat", "--phase", "build", "--event", "context", "--role", "wright", "--bytes", "1234"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("record code=%d out=%s", code, stderr.String())
	}
	stdout.Reset()
	code = RunMetrics(root, []string{"summary", "feat"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("summary code=%d out=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "build") || !strings.Contains(out, "context=1") || !strings.Contains(out, "bytes=1234") {
		t.Fatalf("summary missing aggregates:\n%s", out)
	}
}

func newSkillsRoot(t *testing.T) string {
	t.Helper()
	skills := t.TempDir()
	writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "core.md"), "# core\nrules\n")
	writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "testing.md"), "# testing\n")
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "---\nname: rite-build\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"triggers\":{\"tdd\":[\"devrites-lib/reference/standards/testing.md\"]},\"workspace\":[\"spec.md\",\"missing.md\"]} -->\n# build\n")
	writeFile(t, filepath.Join(skills, "..", "agents", "devrites-slice-wright.md"), "# wright\n")
	writeFile(t, filepath.Join(skills, "..", "agents", "devrites-code-reviewer.md"), "# reviewer\n")
	writeFile(t, filepath.Join(skills, "..", "agents", "devrites-proof-runner.md"), "# runner\n")
	return skills
}

func TestContextEmitsDedupedBundle(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\n")
	skills := newSkillsRoot(t)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	args := []string{"feat", "--phase", "build", "--role", "slice-wright", "--trigger", "tdd", "--skills-root", skills}
	code := RunContext(root, args, stdout, stderr)
	out := stdout.String() + stderr.String()
	if code != 0 {
		t.Fatalf("code=%d want 0\n%s", code, out)
	}
	bundle, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build-slice-wright.bundle.md"))
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	text := string(bundle)
	if !strings.Contains(text, "MISSING: workspace/missing.md") {
		t.Fatalf("missing marker absent from bundle:\n%s", text)
	}
	for _, want := range []string{"===== devrites-lib/reference/standards/core.md =====", "# core", "# testing", "# wright", "# spec"} {
		if !strings.Contains(text, want) {
			t.Fatalf("bundle missing %q:\n%s", want, text)
		}
	}
	if strings.Count(text, "core.md") < 1 {
		t.Fatal("core not bundled")
	}
	// metrics auto-recorded
	stdout.Reset()
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code != 0 || !strings.Contains(stdout.String(), "context=1") {
		t.Fatalf("context metric not recorded: code=%d\n%s", code, stdout.String())
	}
}

func TestContextRejectsUnknownTrigger(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunContext(root, []string{"feat", "--phase", "build", "--trigger", "nope", "--skills-root", skills}, stdout, stderr)
	if code != 2 || !strings.Contains(stderr.String(), "unknown trigger") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestContextSkillModeAndTriggerVisibility(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "devrites-audit", "SKILL.md"),
		"---\nname: devrites-audit\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"triggers\":{\"sec\":[\"devrites-lib/reference/standards/testing.md\"]}} -->\n# audit\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunContext(root, []string{"feat", "--skill", "devrites-audit", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout.String(), "unselected=[sec]") {
		t.Fatalf("unselected triggers not reported:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(featureDir, "ctx", "devrites-audit.bundle.md")); err != nil {
		t.Fatalf("skill bundle not written: %v", err)
	}
}

func TestContextSkillWithoutSlugNeedsOut(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "devrites-audit", "SKILL.md"),
		"---\nname: devrites-audit\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"workspace\":[\"spec.md\"]} -->\n# audit\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunContext(root, []string{"--skill", "devrites-audit", "--skills-root", skills}, stdout, stderr); code != 2 {
		t.Fatalf("no-slug no-out: want 2\n%s", stderr)
	}
	out := filepath.Join(t.TempDir(), "audit.bundle.md")
	stdout.Reset()
	stderr.Reset()
	if code := RunContext(root, []string{"--skill", "devrites-audit", "--out", out, "--skills-root", skills}, stdout, stderr); code != 0 {
		t.Fatalf("no-slug with out: code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "skipped: no feature workspace") {
		t.Fatalf("workspace skip marker missing:\n%s", bundle)
	}
}

func TestDispatchWaveBarrier(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	run := func(args ...string) (int, *bytes.Buffer, *bytes.Buffer) {
		out, err := &bytes.Buffer{}, &bytes.Buffer{}
		return RunDispatch(root, args, out, err), out, err
	}
	if c, _, _ := run("feat", "open", "--phase", "review", "--wave", "w1", "--role", "a", "--role", "b"); c != 0 {
		t.Fatal("open failed")
	}
	if c, _, e := run("feat", "seal", "--wave", "w1"); c == 0 {
		t.Fatalf("seal before all starts should fail: %s", e)
	}
	if c, _, e := run("feat", "return", "--wave", "w1", "--role", "a"); c == 0 {
		t.Fatalf("return before seal should fail: %s", e)
	}
	if c, _, _ := run("feat", "start", "--wave", "w1", "--role", "a", "--handle", "h1"); c != 0 {
		t.Fatal("start a failed")
	}
	if c, _, e := run("feat", "start", "--wave", "w1", "--role", "b", "--handle", "h1"); c == 0 {
		t.Fatalf("duplicate handle accepted: %s", e)
	}
	if c, _, _ := run("feat", "start", "--wave", "w1", "--role", "b", "--handle", "h2"); c != 0 {
		t.Fatal("start b failed")
	}
	if c, _, _ := run("feat", "seal", "--wave", "w1"); c != 0 {
		t.Fatal("seal failed after all starts")
	}
	if c, _, _ := run("feat", "return", "--wave", "w1", "--role", "a"); c != 0 {
		t.Fatal("return a failed")
	}
	if c, _, e := run("feat", "status", "--wave", "w1"); c == 0 {
		t.Fatalf("status should fail while b pending: %s", e)
	}
	if c, _, _ := run("feat", "return", "--wave", "w1", "--role", "b"); c != 0 {
		t.Fatal("return b failed")
	}
	if c, o, _ := run("feat", "status", "--wave", "w1"); c != 0 || !strings.Contains(o.String(), "complete") {
		t.Fatalf("status should report complete: %d %s", c, o)
	}
	// metrics auto-recorded for start+return
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code != 0 ||
		!strings.Contains(stdout.String(), "dispatch=2") || !strings.Contains(stdout.String(), "return=2") {
		t.Fatalf("dispatch metrics missing:\n%s", stdout)
	}
}

func TestDispatchReopenAndAbandon(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	run := func(args ...string) int {
		out, err := &bytes.Buffer{}, &bytes.Buffer{}
		return RunDispatch(root, args, out, err)
	}
	if c := run("feat", "open", "--phase", "build", "--wave", "w", "--role", "x"); c != 0 {
		t.Fatal("open")
	}
	if c := run("feat", "open", "--phase", "build", "--wave", "w", "--role", "x"); c == 0 {
		t.Fatal("reopen of open wave should fail")
	}
	if c := run("feat", "abandon", "--wave", "w", "--reason", "host limit"); c == 0 {
		t.Fatal("abandon must signal handoff (nonzero)")
	}
	if c := run("feat", "return", "--wave", "w", "--role", "x"); c == 0 {
		t.Fatal("return into abandoned wave should fail")
	}
}

func TestContextWorkspaceByRoleScopes(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\n")
	writeFile(t, filepath.Join(featureDir, "plan.md"), "# plan\n")
	writeFile(t, filepath.Join(featureDir, "eng-review.md"), "# review\n")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"),
		"---\nname: rite-build\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"workspace\":[\"spec.md\",\"plan.md\",\"eng-review.md\"],\"workspaceByRole\":{\"slice-wright\":[\"spec.md\",\"tasks.md\"],\"proof-runner\":[]}} -->\n# build\n")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "## SLICE-001 x\n")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunContext(root, []string{"feat", "--phase", "build", "--role", "slice-wright", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build-slice-wright.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(bundle)
	if !strings.Contains(text, "workspace/spec.md") || !strings.Contains(text, "workspace/tasks.md") {
		t.Fatalf("scoped bundle missing role files:\n%s", text)
	}
	if strings.Contains(text, "eng-review.md") {
		t.Fatalf("scoped bundle leaked excluded artifact:\n%s", text)
	}
	if !strings.Contains(stdout.String(), "role-scoped") {
		t.Fatalf("role-scope not reported:\n%s", stdout)
	}

	// Role absent from workspaceByRole keeps the flat list.
	stdout.Reset()
	stderr.Reset()
	code = RunContext(root, []string{"feat", "--phase", "build", "--role", "code-reviewer", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err = os.ReadFile(filepath.Join(featureDir, "ctx", "build-code-reviewer.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "eng-review.md") {
		t.Fatal("unmapped role should get the flat workspace list")
	}

	// Empty role list bundles no workspace artifacts.
	stdout.Reset()
	stderr.Reset()
	code = RunContext(root, []string{"feat", "--phase", "build", "--role", "proof-runner", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err = os.ReadFile(filepath.Join(featureDir, "ctx", "build-proof-runner.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "role-scoped read-set declares no workspace artifacts") {
		t.Fatalf("empty role list should emit the skip marker:\n%s", bundle)
	}
	if strings.Contains(string(bundle), "workspace/spec.md") {
		t.Fatal("empty role list still bundled workspace files")
	}
}

func TestContextRoleAutoFiresAgentsTrigger(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "core.md"), "# core\n")
	writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "agents.md"), "# agents contract\n")
	writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "testing.md"), "# testing\n")
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"),
		"---\nname: rite-build\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"triggers\":{\"agents\":[\"devrites-lib/reference/standards/agents.md\"],\"testing\":[\"devrites-lib/reference/standards/testing.md\"]},\"workspace\":[]} -->\n# build\n")

	// A role call is a dispatch: the declared agents trigger auto-fires.
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunContext(root, []string{"feat", "--phase", "build", "--role", "slice-wright", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build-slice-wright.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "agents contract") {
		t.Fatalf("auto-fired agents trigger missing from bundle:\n%s", bundle)
	}
	out := stdout.String()
	if !strings.Contains(out, "auto=[agents]") {
		t.Fatalf("auto-fire not reported:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "triggers: applied=") {
			unsel := line[strings.Index(line, "unselected=["):]
			if strings.Contains(unsel, "agents") {
				t.Fatalf("agents listed as unselected:\n%s", line)
			}
		}
	}

	// Without --role the trigger stays unselected (no auto-fire).
	stdout.Reset()
	stderr.Reset()
	code = RunContext(root, []string{"feat", "--phase", "build", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	if strings.Contains(stdout.String(), "auto=[agents]") {
		t.Fatalf("auto-fire without --role:\n%s", stdout)
	}
	if !strings.Contains(stdout.String(), "agents") {
		t.Fatalf("agents should appear in unselected list:\n%s", stdout)
	}

	// Explicit --trigger agents with --role does not double-report.
	stdout.Reset()
	stderr.Reset()
	code = RunContext(root, []string{"feat", "--phase", "build", "--role", "slice-wright", "--trigger", "agents", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	if strings.Contains(stdout.String(), "auto=[agents]") {
		t.Fatalf("explicit trigger also reported as auto:\n%s", stdout)
	}

	// A role with no agent contract file hard-fails: a dispatch packet
	// without its contract is a bug, not a soft omission.
	stdout.Reset()
	stderr.Reset()
	code = RunContext(root, []string{"feat", "--phase", "build", "--role", "ghost", "--skills-root", skills}, stdout, stderr)
	if code != 3 {
		t.Fatalf("missing agent contract should exit 3, got %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout.String(), "MISSING: agents/devrites-ghost.md") {
		t.Fatalf("missing agent not reported:\n%s", stdout)
	}
}

func TestContextSuggestsTriggersFromFacts(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\nA React component with auth token handling.\n")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 4\n")
	writeFile(t, filepath.Join(root, "principles.md"), "# principles\n")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"),
		"---\nname: rite-build\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"triggers\":{\"afk\":[\"devrites-lib/reference/standards/testing.md\"],\"parallel\":[\"devrites-lib/reference/standards/testing.md\"],\"frontend\":[\"devrites-lib/reference/standards/testing.md\"],\"security\":[\"devrites-lib/reference/standards/testing.md\"],\"principles\":[\"devrites-lib/reference/standards/testing.md\"],\"coding\":[\"devrites-lib/reference/standards/testing.md\"],\"cross-model\":[\"devrites-lib/reference/standards/testing.md\"]},\"workspace\":[]} -->\n# build\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunContext(root, []string{"feat", "--phase", "build", "--role", "slice-wright", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	out := stdout.String()
	for _, want := range []string{"afk", "parallel", "frontend", "security", "principles", "coding"} {
		if !strings.Contains(out, want) {
			t.Fatalf("suggestion %q missing:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "suggested=[") {
		t.Fatalf("suggested line absent:\n%s", out)
	}
	if strings.Contains(out, "cross-model") && strings.Contains(out[strings.Index(out, "suggested="):], "cross-model") {
		t.Fatalf("flag-only trigger must not be suggested:\n%s", out)
	}
}

func TestContextUnchangedDigestSkipsRewrite(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\n")
	skills := newSkillsRoot(t)
	args := []string{"feat", "--phase", "build", "--role", "slice-wright", "--trigger", "tdd", "--skills-root", skills}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunContext(root, args, stdout, stderr); code != 0 {
		t.Fatalf("first run code=%d\n%s", code, stderr)
	}
	bundlePath := filepath.Join(featureDir, "ctx", "build-slice-wright.bundle.md")
	info1, err := os.Stat(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if code := RunContext(root, args, stdout, stderr); code != 0 {
		t.Fatalf("second run code=%d\n%s", code, stderr)
	}
	if !strings.Contains(stdout.String(), "unchanged") {
		t.Fatalf("unchanged marker absent:\n%s", stdout)
	}
	info2, err := os.Stat(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Fatal("unchanged bundle was rewritten")
	}
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec v2\n")
	stdout.Reset()
	if code := RunContext(root, args, stdout, stderr); code != 0 {
		t.Fatalf("third run code=%d\n%s", code, stderr)
	}
	if strings.Contains(stdout.String(), "unchanged") {
		t.Fatal("changed input still reported unchanged")
	}
}

func TestMetricsSummaryReportsRoles(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	for _, args := range [][]string{
		{"record", "feat", "--phase", "build", "--event", "context", "--role", "slice-wright", "--bytes", "100"},
		{"record", "feat", "--phase", "build", "--event", "context", "--role", "slice-wright", "--bytes", "50"},
		{"record", "feat", "--phase", "review", "--event", "context", "--role", "code-reviewer", "--bytes", "200"},
	} {
		if code := RunMetrics(root, args, stdout, stderr); code != 0 {
			t.Fatalf("record %v code=%d\n%s", args, code, stderr)
		}
	}
	stdout.Reset()
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code != 0 {
		t.Fatalf("summary code=%d\n%s", code, stderr)
	}
	out := stdout.String()
	if !strings.Contains(out, "roles:") || !strings.Contains(out, "slice-wright") || !strings.Contains(out, "bytes=150") {
		t.Fatalf("role rollup missing:\n%s", out)
	}
}

func TestContextPhaseAndSkillMutuallyExclusive(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	for _, args := range [][]string{
		{"feat", "--skills-root", "x"},
		{"feat", "--phase", "build", "--skill", "devrites-audit"},
		{"feat", "--skill", "../escape"},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := RunContext(root, args, stdout, stderr); code != 2 {
			t.Fatalf("args %v: code=%d want 2", args, code)
		}
	}
}

func TestNextReportsPhasePath(t *testing.T) {
	root, featureDir := newWorkspace(t, "spec")
	// spec requires: brief, spec, state, decisions, assumptions, questions
	for _, name := range []string{"brief.md", "spec.md", "decisions.md", "questions.md"} {
		writeFile(t, filepath.Join(featureDir, name), "x\n")
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunNext(root, []string{"feat"}, stdout, stderr)
	out := stdout.String()
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	if !strings.Contains(out, "next-phase: spec") {
		t.Fatalf("expected incomplete spec as next:\n%s", out)
	}
	if !strings.Contains(out, "missing: assumptions.md") {
		t.Fatalf("assumptions.md should be missing:\n%s", out)
	}
}

func TestNextBlocksOnOpenQuestions(t *testing.T) {
	root, featureDir := newWorkspace(t, "spec")
	for _, name := range []string{"brief.md", "spec.md", "decisions.md", "assumptions.md"} {
		writeFile(t, filepath.Join(featureDir, name), "x\n")
	}
	writeFile(t, filepath.Join(featureDir, "questions.md"), "# Questions\n\n| Q-001 | open | g | q | | i |\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunNext(root, []string{"feat"}, stdout, stderr)
	out := stdout.String()
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	if !strings.Contains(out, "next-phase: clarify") {
		t.Fatalf("expected clarify next:\n%s", out)
	}
	if !strings.Contains(out, "missing: decision-coverage.md") {
		t.Fatalf("decision-coverage.md should be missing:\n%s", out)
	}
}

func TestNextSkipsVacuousClarify(t *testing.T) {
	root, featureDir := newWorkspace(t, "spec")
	for _, name := range []string{"brief.md", "spec.md", "decisions.md", "assumptions.md"} {
		writeFile(t, filepath.Join(featureDir, name), "x\n")
	}
	writeFile(t, filepath.Join(featureDir, "questions.md"), "# Questions\n\nNo open questions remain.\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunNext(root, []string{"feat"}, stdout, stderr)
	out := stdout.String()
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	if !strings.Contains(out, "skip: clarify (0 open questions)") {
		t.Fatalf("expected clarify skip advisory:\n%s", out)
	}
	if !strings.Contains(out, "next-phase: temper") {
		t.Fatalf("expected temper next:\n%s", out)
	}
}

func TestDiffScopeAllowlist(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	repo := t.TempDir()
	initGitRepository(t, repo)
	writeFile(t, filepath.Join(repo, "src", "ok.go"), "package ok\n")
	writeFile(t, filepath.Join(repo, "src", "evil.go"), "package evil\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	args := []string{"feat", "--allow", "src/ok.go", "--cwd", repo}
	code := RunCheckDiffScope(root, args, stdout, stderr)
	out := stdout.String()
	if code != 3 || !strings.Contains(out, "src/evil.go outside allowlist") {
		t.Fatalf("code=%d want violation\n%s", code, out)
	}
	stdout.Reset()
	code = RunCheckDiffScope(root, []string{"feat", "--allow", "src", "--cwd", repo}, stdout, stderr)
	if code != 0 || !strings.Contains(stdout.String(), "2 changed") {
		t.Fatalf("code=%d\n%s", code, stdout.String())
	}
}

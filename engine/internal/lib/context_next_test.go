package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/devrites/devrites/internal/state"
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

func TestContextRelativeSkillsRootResolvesFromProject(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "# State\n")
	skills := filepath.Join(project, ".omp", "skills")
	writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "core.md"), "# omp core\n")
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"]} -->\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	// The test cwd is the package dir, so only the project-root fallback finds it.
	if code := RunContext(root, []string{"feat", "--phase", "build", "--skills-root", ".omp/skills"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err := os.ReadFile(filepath.Join(root, "work", "feat", "ctx", "build.bundle.md"))
	if err != nil || !strings.Contains(string(bundle), "# omp core") {
		t.Fatalf("bundle not built from project-relative skills root: %v\n%s", err, bundle)
	}
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
	if c, _, e := run("feat", "status"); c == 0 {
		t.Fatalf("bare status should fail while a wave is open: %s", e)
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
	if c, _, e := run("feat", "status"); c != 0 {
		t.Fatalf("bare status should pass once every wave is complete: %s", e)
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
	code = RunContext(root, []string{"feat", "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err = os.ReadFile(filepath.Join(featureDir, "ctx", "build-code-reviewer.SLICE-001.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "eng-review.md") {
		t.Fatal("unmapped role should get the flat workspace list")
	}

	// Empty role list bundles no workspace artifacts.
	stdout.Reset()
	stderr.Reset()
	code = RunContext(root, []string{"feat", "--phase", "build", "--role", "proof-runner", "--slice", "SLICE-001", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err = os.ReadFile(filepath.Join(featureDir, "ctx", "build-proof-runner.SLICE-001.bundle.md"))
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
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "## SLICE-001 x\n")
	code = RunContext(root, []string{"feat", "--phase", "build", "--role", "ghost", "--slice", "SLICE-001", "--skills-root", skills}, stdout, stderr)
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
	if !strings.Contains(out, "triggers: apply with: devrites-engine context feat --phase build --role slice-wright --trigger ") {
		t.Fatalf("apply hint absent or malformed:\n%s", out)
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
	info2, err := os.Stat(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && !info1.ModTime().Equal(info2.ModTime()) {
		t.Fatal("unchanged bundle was rewritten")
	}
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec v2\n")
	stdout.Reset()
	if code := RunContext(root, args, stdout, stderr); code != 0 {
		t.Fatalf("third run code=%d\n%s", code, stderr)
	}
	updated, err := os.ReadFile(bundlePath)
	if err != nil || !bytes.Contains(updated, []byte("# spec v2\n")) {
		t.Fatalf("changed source not reflected in bundle: %v\n%s", err, updated)
	}
}

func TestContextRoleBundleStubsMarkedSections(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	core := filepath.Join(skills, "devrites-lib", "reference", "standards", "core.md")
	writeFile(t, core, "# core\n\n## Leaf\nleaf rule\n\n<!-- root-only -->\n## Handoff\nroot rule\n<!-- /root-only -->\n\n"+
		"## Models\n<!-- model-surface -->\nmodel rule\n<!-- /model-surface -->\n")
	writeFile(t, filepath.Join(skills, "..", "agents", "devrites-security-auditor.md"), "# auditor\n")
	// Vet, not Build: Build reviewer bundles need --slice and a tasks.md,
	// which would change the model-surface evidence this test controls.
	writeFile(t, filepath.Join(skills, "rite-vet", "SKILL.md"), "<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"]} -->\n# vet\n")
	bundle := func(role string) string {
		t.Helper()
		args, name := []string{"feat", "--phase", "vet", "--skills-root", skills}, "vet.bundle.md"
		if role != "" {
			args, name = append(args, "--role", role), "vet-"+role+".bundle.md"
		}
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		if code := RunContext(root, args, stdout, stderr); code != 0 {
			t.Fatalf("role=%q code=%d\n%s%s", role, code, stdout, stderr)
		}
		data, err := os.ReadFile(filepath.Join(featureDir, "ctx", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	expect := func(label, text string, want, reject []string) {
		t.Helper()
		for _, w := range want {
			if !strings.Contains(text, w) {
				t.Fatalf("%s: bundle missing %q:\n%s", label, w, text)
			}
		}
		for _, r := range reject {
			if strings.Contains(text, r) {
				t.Fatalf("%s: bundle must not contain %q:\n%s", label, r, text)
			}
		}
	}
	rootStub := `[root-only section "Handoff" omitted for role code-reviewer`
	modelStub := `[model-surface section "Models" omitted for role code-reviewer`

	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\nA billing export.\n")
	expect("root", bundle(""), []string{"root rule", "model rule"}, []string{"omitted for role"})
	expect("non-ai leaf", bundle("code-reviewer"),
		[]string{"leaf rule", rootStub, "## Models", modelStub, "devrites-lib/reference/standards/core.md § Models"},
		[]string{"root rule", "## Handoff", "model rule"})
	expect("auditor", bundle("security-auditor"), []string{"model rule"}, []string{"root rule"})

	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\nAn LLM prompt router.\n")
	expect("ai leaf", bundle("code-reviewer"), []string{rootStub, "model rule"}, []string{"root rule", modelStub})

	// Template boilerplate (the AI-SPEC annex, the .claude/skills discovery
	// line) is not an AI signal; an ai-spec.md file is.
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\nA billing export.\n\n## AI-SPEC annex\n"+
		"- Model/RAG/agent/eval/LLM-output scope: `ai-spec.md` from `ai-spec-template.md`.\n- Otherwise: not applicable.\n\n## Success metrics\n- Export matches ledger.\n")
	writeFile(t, filepath.Join(featureDir, "plan.md"), "# plan\n## Technical context\n"+
		"- Test commands: <discovered runner commands; discovery contract: .claude/skills/rite-prove/reference/test-command-discovery.md>\n")
	expect("template annex", bundle("code-reviewer"), []string{modelStub}, []string{"model rule"})
	writeFile(t, filepath.Join(featureDir, "ai-spec.md"), "# ai spec\n")
	expect("ai-spec file", bundle("code-reviewer"), []string{"model rule"}, []string{modelStub})

	// No spec/plan/tasks evidence: unsure, so the model-surface rules stay inline.
	for _, name := range []string{"spec.md", "plan.md", "ai-spec.md"} {
		if err := os.Remove(filepath.Join(featureDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	expect("no corpus", bundle("code-reviewer"), []string{"model rule"}, []string{"root rule"})

	writeFile(t, core, "# core\n<!-- root-only -->\n## Handoff\nroot rule\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunContext(root, []string{"feat", "--phase", "vet", "--role", "code-reviewer", "--skills-root", skills}, stdout, stderr)
	if code != 2 || !strings.Contains(stderr.String(), "unclosed <!-- root-only --> section") {
		t.Fatalf("unbalanced marker: code=%d want 2\n%s%s", code, stdout, stderr)
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

// Phases that add no required artifact over the previous one (build and
// converge after vet; polish and review after prove) cannot be proven done by
// artifact presence, so they are never skipped.
func TestNextDoesNotSkipPhasesWithoutNewArtifacts(t *testing.T) {
	for _, tc := range []struct {
		phase state.Phase
		want  string
	}{
		{state.PhaseVet, "next-phase: build"},
		{state.PhaseProve, "next-phase: polish"},
	} {
		root, featureDir := newWorkspace(t, string(tc.phase))
		policy, _ := state.PolicyFor(tc.phase)
		for _, artifact := range policy.RequiredArtifacts {
			if artifact != state.LedgerFile {
				writeFile(t, filepath.Join(featureDir, string(artifact)), "x\n")
			}
		}
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		if code := RunNext(root, []string{"feat"}, stdout, stderr); code != 0 {
			t.Fatalf("%s: code=%d err=%s", tc.phase, code, stderr.String())
		}
		if out := stdout.String(); !strings.Contains(out, tc.want+"\n") || strings.Contains(out, "artifacts complete") {
			t.Fatalf("%s: want %q without artifact skips:\n%s", tc.phase, tc.want, out)
		}
	}
}

func TestNextSkipsConditionalConvergeAfterBuild(t *testing.T) {
	root, featureDir := newWorkspace(t, string(state.PhaseBuild))
	policy, _ := state.PolicyFor(state.PhaseBuild)
	for _, artifact := range policy.RequiredArtifacts {
		if artifact != state.LedgerFile {
			writeFile(t, filepath.Join(featureDir, string(artifact)), "x\n")
		}
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunNext(root, []string{"feat"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d err=%s", code, stderr.String())
	}
	// /rite-build hands off to /rite-prove; converge is a recovery phase.
	if out := stdout.String(); !strings.Contains(out, "next-phase: prove\n") || !strings.Contains(out, "converge (artifacts complete)") {
		t.Fatalf("want prove with converge skipped:\n%s", out)
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

	// A changed path whose own name contains the porcelain rename separator
	// is checked as that full path, never as a fragment of it. Windows forbids
	// '>' in file names, so the case cannot exist there.
	if runtime.GOOS != "windows" {
		sepRepo := t.TempDir()
		initGitRepository(t, sepRepo)
		runDiffScopeGit(t, sepRepo, "config", "user.email", "tests@example.invalid")
		runDiffScopeGit(t, sepRepo, "config", "user.name", "DevRites Tests")
		writeFile(t, filepath.Join(sepRepo, "docs", "secret -> notes.md"), "v1\n")
		writeFile(t, filepath.Join(sepRepo, "docs", "secret"), "v1\n")
		runDiffScopeGit(t, sepRepo, "add", "-A")
		runDiffScopeGit(t, sepRepo, "commit", "-qm", "base")
		writeFile(t, filepath.Join(sepRepo, "docs", "secret -> notes.md"), "changed out of contract\n")

		stdout.Reset()
		stderr.Reset()
		code = RunCheckDiffScope(root, []string{"feat", "--allow", "notes.md", "--cwd", sepRepo}, stdout, stderr)
		if code != 3 || !strings.Contains(stdout.String(), "BLOCKED: docs/secret -> notes.md outside allowlist") {
			t.Fatalf("separator-named path checked as its tail: code=%d want 3 with the full path\n%s", code, stdout.String())
		}
		stdout.Reset()
		stderr.Reset()
		code = RunCheckDiffScope(root, []string{"feat", "--allow", "docs/secret -> notes.md", "--cwd", sepRepo}, stdout, stderr)
		if code != 0 {
			t.Fatalf("declared path rejected: code=%d\n%s", code, stdout.String())
		}

		// A rename reports its new name and a separator-named file reports its
		// full name, even when both end in the same tail.
		runDiffScopeGit(t, sepRepo, "mv", "docs/secret", "notes.md")
		stdout.Reset()
		stderr.Reset()
		code = RunCheckDiffScope(root, []string{"feat", "--allow", "notes.md", "--cwd", sepRepo}, stdout, stderr)
		if code != 3 || !strings.Contains(stdout.String(), "BLOCKED: docs/secret -> notes.md outside allowlist") ||
			!strings.Contains(stdout.String(), "BLOCKED: docs/secret outside allowlist") {
			t.Fatalf("rename and separator-named file not distinguishable: code=%d\n%s", code, stdout.String())
		}
	}

	// An ordinary rename is reported by its new name, and its old name is
	// checked against the allowlist too.
	renameRepo := t.TempDir()
	initGitRepository(t, renameRepo)
	runDiffScopeGit(t, renameRepo, "config", "user.email", "tests@example.invalid")
	runDiffScopeGit(t, renameRepo, "config", "user.name", "DevRites Tests")
	writeFile(t, filepath.Join(renameRepo, "old.go"), "package old\n")
	runDiffScopeGit(t, renameRepo, "add", "-A")
	runDiffScopeGit(t, renameRepo, "commit", "-qm", "base")
	runDiffScopeGit(t, renameRepo, "mv", "old.go", "new.go")
	stdout.Reset()
	stderr.Reset()
	code = RunCheckDiffScope(root, []string{"feat", "--allow", "old.go", "--cwd", renameRepo}, stdout, stderr)
	if code != 3 || !strings.Contains(stdout.String(), "BLOCKED: new.go outside allowlist") {
		t.Fatalf("rename not reported by its new name: code=%d\n%s", code, stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = RunCheckDiffScope(root, []string{"feat", "--allow", "new.go", "--cwd", renameRepo}, stdout, stderr)
	if code != 3 || !strings.Contains(stdout.String(), "BLOCKED: old.go outside allowlist") {
		t.Fatalf("rename old name not checked: code=%d\n%s", code, stdout.String())
	}
}

func runDiffScopeGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	if out, err := runGitCommand(repo, nil, args...); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestDispatchConcurrentReturnsCompleteWave(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	run := func(args ...string) int {
		out, err := &bytes.Buffer{}, &bytes.Buffer{}
		return RunDispatch(root, args, out, err)
	}
	roles := []string{"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7"}
	open := []string{"feat", "open", "--phase", "review", "--wave", "w"}
	for _, r := range roles {
		open = append(open, "--role", r)
	}
	if c := run(open...); c != 0 {
		t.Fatal("open failed")
	}
	for _, r := range roles {
		if c := run("feat", "start", "--wave", "w", "--role", r, "--handle", "h-"+r); c != 0 {
			t.Fatalf("start %s failed", r)
		}
	}
	if c := run("feat", "seal", "--wave", "w"); c != 0 {
		t.Fatal("seal failed")
	}
	var wg sync.WaitGroup
	for _, r := range roles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c := run("feat", "return", "--wave", "w", "--role", r); c != 0 {
				t.Errorf("return %s failed: %d", r, c)
			}
		}()
	}
	wg.Wait()
	if c := run("feat", "status", "--wave", "w"); c != 0 {
		data, _ := os.ReadFile(filepath.Join(featureDir, "dispatch.json"))
		t.Fatalf("concurrent returns lost an update; wave not complete:\n%s", data)
	}
}

func TestDispatchAbandonRecordsReason(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	if c := RunDispatch(root, []string{"feat", "open", "--phase", "build", "--wave", "w", "--role", "x"}, out, errOut); c != 0 {
		t.Fatal("open failed")
	}
	RunDispatch(root, []string{"feat", "abandon", "--wave", "w", "--reason", "host limit"}, out, errOut)
	data, err := os.ReadFile(filepath.Join(featureDir, "dispatch.json"))
	if err != nil || !strings.Contains(string(data), `"reason": "host limit"`) {
		t.Fatalf("abandon reason not persisted: %v\n%s", err, data)
	}
}

func TestContextWorkspaceIncludeMarkerStaysLiteral(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	marker := "<!-- include:../../outside.txt -->"
	writeFile(t, filepath.Join(featureDir, "spec.md"), marker+"\n")
	writeFile(t, filepath.Join(root, "outside.txt"), "SENTINEL-OUTSIDE\n")
	skills := newSkillsRoot(t)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunContext(root, []string{"feat", "--phase", "build", "--skills-root", skills}, stdout, stderr)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(bundle)
	if strings.Contains(text, "SENTINEL-OUTSIDE") {
		t.Fatalf("workspace include marker was expanded:\n%s", text)
	}
	if !strings.Contains(text, marker) {
		t.Fatalf("workspace include marker not kept literal:\n%s", text)
	}
}

func TestContextSliceScopesWorkspaceAndRole(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-build", "reference", "root-procedure.md"), "# root procedure body\n")
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "---\nname: rite-build\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\",\"rite-build/reference/root-procedure.md\"],\"rootOnly\":[\"rite-build/reference/root-procedure.md\"],\"workspace\":[\"tasks.md\",\"decisions.md\",\"spec.md\"]} -->\n# build\n")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "# Tasks\n\n## SLICE-001 First\nGoal: first body\n\n## SLICE-002 Second\nGoal: second body\nSatisfies: AC-001\nDecisions: DEC-010\ndepends_on: [SLICE-001]\n")
	var decisions strings.Builder
	decisions.WriteString("# Decisions\n")
	for i := 0; i < 250; i++ {
		decisions.WriteString("filler decision row\n")
	}
	decisions.WriteString("## DEC-010 Use the store\nDEC-0100 is a different id\n")
	writeFile(t, filepath.Join(featureDir, "decisions.md"), decisions.String())
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\nAC-001 small spec body\n")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	args := []string{"feat", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-002", "--skills-root", skills}
	if code := RunContext(root, args, stdout, stderr); code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout.String(), "scope: slice SLICE-002 (1 whole, 2 by-reference)") {
		t.Fatalf("scope line missing:\n%s", stdout)
	}
	raw, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build-slice-wright.SLICE-002.bundle.md"))
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Goal: second body",
		"workspace/decisions.md (by reference)",
		"/work/feat/decisions.md sha256: ",
		"- L252 ## DEC-010 Use the store",
		"- DEC-010: L252\n",
		"- SLICE-001: L3 L10\n",
		"- AC-001: L8\n",
		"small spec body",
		"===== root-only (not inlined for role slice-wright) =====\nrite-build/reference/root-procedure.md",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("bundle missing %q:\n%s", want, text)
		}
	}
	for _, absent := range []string{"Goal: first body", "filler decision row", "root procedure body"} {
		if strings.Contains(text, absent) {
			t.Fatalf("bundle inlined %q:\n%s", absent, text)
		}
	}
	if strings.Index(text, "# wright") > strings.Index(text, "# core") {
		t.Fatalf("role contract must lead the bundle:\n%s", text)
	}

	// Without --role the root still gets its procedure inline.
	stdout.Reset()
	if code := RunContext(root, []string{"feat", "--phase", "build", "--skills-root", skills}, stdout, stderr); code != 0 {
		t.Fatalf("phase bundle code=%d\n%s", code, stderr)
	}
	phase, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build.bundle.md"))
	if err != nil || !strings.Contains(string(phase), "root procedure body") || !strings.Contains(string(phase), "Goal: first body") {
		t.Fatalf("phase bundle must stay whole: err=%v\n%s", err, phase)
	}
}

// The root's build/autocomplete bundle follows state.md active_slice; --whole,
// a role, an unset ("none") or stale cursor keep the whole feature view.
func TestContextRootBundleDefaultsToActiveSlice(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	manifest := "<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"workspace\":[\"tasks.md\",\"spec.md\"]} -->\n"
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), manifest+"# build\n")
	writeFile(t, filepath.Join(skills, "rite-autocomplete", "SKILL.md"), manifest+"# autocomplete\n")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "# Tasks\n\n## SLICE-001 First\nGoal: first body\n\n## SLICE-002 Second\nGoal: second body\n")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\n")
	setCursor := func(phase, id string) {
		writeFile(t, filepath.Join(featureDir, "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | "+phase+" |\n| status | running |\n| active_slice | "+id+" |\n")
	}
	setActive := func(id string) { setCursor("build", id) }
	run := func(want int, args ...string) string {
		t.Helper()
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		if code := RunContext(root, append([]string{"feat", "--skills-root", skills}, args...), stdout, stderr); code != want {
			t.Fatalf("context %v: code=%d want %d\n%s%s", args, code, want, stdout, stderr)
		}
		return stdout.String()
	}
	read := func(name string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(featureDir, "ctx", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}

	setActive("SLICE-002")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 5\n")
	for _, phase := range []string{"build", "autocomplete"} {
		if out := run(0, "--phase", phase); strings.Contains(out, "scope: slice") {
			t.Fatalf("%s: parallel batch must keep every slice visible:\n%s", phase, out)
		}
		text := read(phase + ".bundle.md")
		if !strings.Contains(text, "Goal: first body") || !strings.Contains(text, "Goal: second body") {
			t.Fatalf("%s: parallel bundle dropped a slice:\n%s", phase, text)
		}
	}
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 1\n")
	for _, phase := range []string{"build", "autocomplete"} {
		if out := run(0, "--phase", phase); !strings.Contains(out, "scope: slice SLICE-002 (auto from state.md) (") {
			t.Fatalf("%s: auto scope line missing:\n%s", phase, out)
		}
		if text := read(phase + ".SLICE-002.bundle.md"); !strings.Contains(text, "Goal: second body") || strings.Contains(text, "Goal: first body") {
			t.Fatalf("%s: bundle not scoped to SLICE-002:\n%s", phase, text)
		}
	}
	if out := run(0, "--phase", "build", "--whole"); strings.Contains(out, "scope: slice") || !strings.Contains(read("build.bundle.md"), "Goal: first body") {
		t.Fatalf("--whole must keep the whole bundle:\n%s", out)
	}
	run(2, "--phase", "build", "--whole", "--slice", "SLICE-001")
	if out := run(0, "--phase", "build", "--role", "slice-wright"); strings.Contains(out, "scope: slice") {
		t.Fatalf("role bundle must not auto-scope:\n%s", out)
	}
	for _, id := range []string{"none", "SLICE-009"} {
		setActive(id)
		if out := run(0, "--phase", "build"); strings.Contains(out, "scope: slice") {
			t.Fatalf("active_slice %s must keep the whole bundle:\n%s", id, out)
		}
	}
	// A stale active_slice after Build never scopes the root's autocomplete bundle.
	setCursor("review", "SLICE-002")
	if out := run(0, "--phase", "autocomplete"); strings.Contains(out, "scope: slice") || !strings.Contains(read("autocomplete.bundle.md"), "Goal: first body") {
		t.Fatalf("cursor phase review must keep the whole bundle:\n%s", out)
	}
}

func TestContextSliceTakesLargeUnderBudgetArtifactsByReference(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "---\nname: rite-build\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"workspace\":[\"tasks.md\",\"plan.md\",\"spec.md\"]} -->\n# build\n")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "# Tasks\n\n## SLICE-001 Only\nGoal: only body\nSatisfies: AC-001\n")
	// 60 long lines: under plan.md's 220-line budget, over the 16 KiB slice cap.
	longRow := strings.Repeat("wide plan prose ", 24) + "\n"
	writeFile(t, filepath.Join(featureDir, "plan.md"), "# Plan\n## SLICE-001 seam\n"+strings.Repeat(longRow, 60))
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# Spec\nAC-001 whole spec\n"+strings.Repeat(longRow, 60))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunContext(root, []string{"feat", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001", "--skills-root", skills}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	raw, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build-slice-wright.SLICE-001.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{"workspace/plan.md (by reference)", "- L2 ## SLICE-001 seam", "- SLICE-001: L2\n", "AC-001 whole spec"} {
		if !strings.Contains(text, want) {
			t.Fatalf("slice bundle missing %q:\n%s", want, text)
		}
	}
	if strings.Count(text, "wide plan prose") != 60*24 {
		t.Fatalf("only spec.md's rows may be inlined; got %d", strings.Count(text, "wide plan prose"))
	}

	// Without --slice the role bundle still reads plan.md whole.
	if code := RunContext(root, []string{"feat", "--phase", "build", "--role", "slice-wright", "--skills-root", skills}, stdout, stderr); code != 0 {
		t.Fatalf("role bundle code=%d\n%s", code, stderr)
	}
	whole, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build-slice-wright.bundle.md"))
	if err != nil || strings.Contains(string(whole), "(by reference)") {
		t.Fatalf("non-slice role bundle must stay whole: err=%v\n%s", err, whole)
	}
}

func TestContextSliceRejectsBadOrUnknownID(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "# Tasks\n\n## SLICE-001 First\nGoal: x\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunContext(root, []string{"feat", "--phase", "build", "--slice", "slice-1", "--skills-root", skills}, stdout, stderr); code != 2 {
		t.Fatalf("malformed id: code=%d want 2\n%s", code, stderr)
	}
	if code := RunContext(root, []string{"--skill", "rite-build", "--slice", "SLICE-001", "--out", "x.md", "--skills-root", skills}, stdout, stderr); code != 2 {
		t.Fatalf("slice without slug: code=%d want 2\n%s", code, stderr)
	}
	stdout.Reset()
	if code := RunContext(root, []string{"feat", "--phase", "build", "--slice", "SLICE-009", "--skills-root", skills}, stdout, stderr); code != 3 || !strings.Contains(stdout.String(), "BLOCKED: SLICE-009 not found") {
		t.Fatalf("unknown slice: code=%d\n%s", code, stdout)
	}
}

func TestDispatchWrightRoundCap(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	run := func(args ...string) (int, string) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := RunDispatch(root, append([]string{"feat"}, args...), stdout, stderr)
		return code, stdout.String() + stderr.String()
	}
	open := func(wave string, extra ...string) (int, string) {
		return run(append([]string{"open", "--phase", "build", "--wave", wave, "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t)}, extra...)...)
	}
	if code, out := run("open", "--phase", "build", "--wave", "w0", "--role", "slice-wright"); code != 2 || !strings.Contains(out, "needs --slice") {
		t.Fatalf("missing --slice: code=%d\n%s", code, out)
	}
	if code, out := run("open", "--phase", "vet", "--wave", "v1", "--role", "plan-reviewer", "--packet", reviewerPacketPath(t)); code != 0 {
		t.Fatalf("non-build wave must not need --slice: code=%d\n%s", code, out)
	}
	for i := 1; i <= wrightRoundsPerReview; i++ {
		code, out := open("r" + string(rune('0'+i)))
		if code != 0 || !strings.Contains(out, "SLICE-001 wright round "+string(rune('0'+i))) || !strings.Contains(out, "writer-safe checks the wright runs") {
			t.Fatalf("round %d: code=%d\n%s", i, code, out)
		}
	}
	if code, out := run("abandon", "--wave", "r5", "--reason", "orphaned"); !strings.Contains(out, "wave r5 abandoned") {
		t.Fatalf("abandon: code=%d\n%s", code, out)
	}
	if code, out := open("r5b"); code != 0 || !strings.Contains(out, "wright round 5") {
		t.Fatalf("abandoned wave must not count: code=%d\n%s", code, out)
	}
	if code, out := open("r6"); code != 3 || !strings.Contains(out, "BLOCKED: SLICE-001 reached 5 wright rounds") {
		t.Fatalf("round 6 must block: code=%d\n%s", code, out)
	}
	writeFile(t, filepath.Join(featureDir, "drift.md"), "# Drift\n\n## D-010 — SLICE-002 unrelated\n\n## D-011 — SLICE-001 reslice review: no smaller AC-preserving split\n")
	if code, out := open("r6", "--reslice-review", "D-099"); code != 3 {
		t.Fatalf("unknown drift id: code=%d\n%s", code, out)
	}
	if code, out := open("r6", "--reslice-review", "D-010"); code != 3 || !strings.Contains(out, "naming SLICE-001") {
		t.Fatalf("drift heading for another slice: code=%d\n%s", code, out)
	}
	if code, out := open("r6", "--reslice-review", "D-011"); code != 0 || !strings.Contains(out, "wright round 6") {
		t.Fatalf("valid review must admit: code=%d\n%s", code, out)
	}
	for i := 7; i <= 2*wrightRoundsPerReview; i++ {
		if code, out := open("r" + string(rune('a'+i))); code != 0 {
			t.Fatalf("round %d inside the reviewed window: code=%d\n%s", i, code, out)
		}
	}
	if code, out := open("r11", "--reslice-review", "D-011"); code != 3 || !strings.Contains(out, "already admitted") {
		t.Fatalf("reused review: code=%d\n%s", code, out)
	}
	if code, out := run("open", "--phase", "build", "--wave", "s2", "--role", "slice-wright", "--slice", "SLICE-002", "--contract", wrightContractPath(t), "--reslice-review", "D-010"); code != 2 {
		t.Fatalf("review before the cap must be refused: code=%d\n%s", code, out)
	}
}

func TestDispatchVetRoundsCountSeparately(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	run := func(args ...string) (int, string) {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := RunDispatch(root, append([]string{"feat"}, args...), stdout, stderr)
		return code, stdout.String() + stderr.String()
	}
	for i := 0; i < 7; i++ {
		if code, out := run("open", "--phase", "vet", "--wave", "full"+string(rune('a'+i)), "--role", "plan-reviewer", "--packet", reviewerPacketPath(t)); code != 0 {
			t.Fatalf("full Vet without --slice stays uncapped: code=%d\n%s", code, out)
		}
	}
	if code, out := run("open", "--phase", "build", "--wave", "w1", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t)); code != 0 || !strings.Contains(out, "wright round 1") {
		t.Fatalf("wright round must not see vet waves: code=%d\n%s", code, out)
	}
	for i := 1; i <= wrightRoundsPerReview; i++ {
		if code, out := run("open", "--phase", "vet", "--wave", "v"+string(rune('0'+i)), "--role", "plan-reviewer", "--packet", reviewerPacketPath(t), "--slice", "SLICE-001"); code != 0 || !strings.Contains(out, "vet round "+string(rune('0'+i))) {
			t.Fatalf("vet round %d: code=%d\n%s", i, code, out)
		}
	}
	if code, out := run("open", "--phase", "vet", "--wave", "v6", "--role", "plan-reviewer", "--packet", reviewerPacketPath(t), "--slice", "SLICE-001"); code != 3 || !strings.Contains(out, "reached 5 vet rounds") {
		t.Fatalf("vet round 6 must block: code=%d\n%s", code, out)
	}
	writeFile(t, filepath.Join(featureDir, "drift.md"), "## D-001 — SLICE-001 reslice review: proof seam already minimal\n")
	if code, out := run("open", "--phase", "vet", "--wave", "v6", "--role", "plan-reviewer", "--packet", reviewerPacketPath(t), "--slice", "SLICE-001", "--reslice-review", "D-001"); code != 0 {
		t.Fatalf("reviewed vet round 6: code=%d\n%s", code, out)
	}
	if code, out := run("open", "--phase", "plan", "--wave", "p1", "--role", "plan-drafter", "--slice", "SLICE-001"); code != 2 {
		t.Fatalf("--slice on a plan wave: code=%d\n%s", code, out)
	}
}

func TestContextRepeatedSliceUnion(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "---\nname: rite-build\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"workspace\":[\"tasks.md\"]} -->\n# build\n")
	writeFile(t, filepath.Join(featureDir, "tasks.md"), "# Tasks\n\n## SLICE-001 A\nGoal: alpha body\n\n## SLICE-002 B\nGoal: beta body\n\n## SLICE-003 C\nGoal: gamma body\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	args := []string{"feat", "--phase", "build", "--slice", "SLICE-001", "--slice", "SLICE-003", "--skills-root", skills}
	if code := RunContext(root, args, stdout, stderr); code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	raw, err := os.ReadFile(filepath.Join(featureDir, "ctx", "build.SLICE-001+SLICE-003.bundle.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "alpha body") || !strings.Contains(text, "gamma body") || strings.Contains(text, "beta body") {
		t.Fatalf("union of slice blocks wrong:\n%s", text)
	}
}

func TestContextRootBundleAnchorsOverBudgetArtifacts(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "---\nname: rite-build\n---\n<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"workspace\":[\"decisions.md\",\"spec.md\"]} -->\n# build\n")
	writeFile(t, filepath.Join(featureDir, "decisions.md"), "# Decisions\n"+strings.Repeat("filler decision row\n", 250)+"## DEC-010 Use the store\n")
	writeFile(t, filepath.Join(featureDir, "spec.md"), "# spec\nsmall spec body\n")
	run := func(args ...string) string {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		if code := RunContext(root, append([]string{"feat", "--phase", "build", "--skills-root", skills}, args...), stdout, stderr); code != 0 {
			t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
		}
		return stdout.String()
	}
	out := run()
	if !strings.Contains(out, "scope: root (1 whole, 1 by-reference over budget)") {
		t.Fatalf("root scope line missing:\n%s", out)
	}
	rootBundle, _ := os.ReadFile(filepath.Join(featureDir, "ctx", "build.bundle.md"))
	if text := string(rootBundle); !strings.Contains(text, "workspace/decisions.md (by reference)") || strings.Contains(text, "filler decision row") || !strings.Contains(text, "small spec body") {
		t.Fatalf("root bundle must anchor only the over-budget artifact:\n%s", text)
	}
	// Build reviewer bundles need --slice; the wright role still takes an unscoped bundle.
	run("--role", "slice-wright")
	roleBundle, _ := os.ReadFile(filepath.Join(featureDir, "ctx", "build-slice-wright.bundle.md"))
	if !strings.Contains(string(roleBundle), "filler decision row") {
		t.Fatalf("a role bundle without --slice must stay whole:\n%s", roleBundle)
	}
}

func TestMetricsSummaryBySlice(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	writeFile(t, filepath.Join(featureDir, "dispatch.json"), `{"waves":{
 "w1":{"phase":"build","state":"complete","slices":["SLICE-001"],"roles":{"slice-wright":{"handle":"a","started":"2026-10-15T10:00:00Z","returned":true,"returned_at":"2026-10-15T10:30:00Z"}}},
 "w2":{"phase":"build","state":"complete","slices":["SLICE-001","SLICE-002"],"roles":{"slice-wright":{"handle":"b","started":"2026-10-15T11:00:00Z","returned":true,"returned_at":"2026-10-15T11:20:00Z"}}},
 "w3":{"phase":"build","state":"abandoned","slices":["SLICE-001"],"roles":{"slice-wright":{"handle":"c"}}},
 "v1":{"phase":"vet","state":"complete","slices":["SLICE-002"],"roles":{"plan-reviewer":{"handle":"d","started":"2026-10-15T12:00:00Z","returned":true}}},
 "f1":{"phase":"vet","state":"complete","roles":{"plan-reviewer":{"handle":"e"}}}
}}`)
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunMetrics(root, []string{"summary", "feat", "--by-slice"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d\n%s", code, stderr)
	}
	out := stdout.String()
	for _, want := range []string{
		"metrics: SLICE-001 wright_rounds=2 vet_rechecks=0 agent_minutes=40\n",
		"metrics: SLICE-002 wright_rounds=1 vet_rechecks=1 agent_minutes=10\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

func TestMetricsSummaryLedgerErrorsAndLongLines(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	note := strings.Repeat("<", 1<<20) // JSON-escapes to ~6 MiB on one line
	if code := RunMetrics(root, []string{"record", "feat", "--phase", "build", "--event", "note", "--note", note}, stdout, stderr); code != 0 {
		t.Fatalf("record code=%d err=%s", code, stderr.String())
	}
	stdout.Reset()
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code != 0 || !strings.Contains(stdout.String(), "note=1") {
		t.Fatalf("long line: code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
	// A ledger that exists but cannot be read is an error, never "no events".
	ledger := filepath.Join(featureDir, MetricsFile)
	if err := os.Remove(ledger); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(ledger, 0o755); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := RunMetrics(root, []string{"summary", "feat"}, stdout, stderr); code == 0 || strings.Contains(stdout.String(), "no events") {
		t.Fatalf("unreadable ledger: code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}
}

// A Codex install keeps skills in .agents/skills and role agents in
// .codex/agents/<name>.toml; Codex loads that file as the agent's
// instructions, so the bundle leads with a stub naming it.
func TestContextRoleContractFallsBackToCodexAgents(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "# State\n")
	skills := filepath.Join(project, ".agents", "skills")
	writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "core.md"), "# core\n")
	writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"]} -->\n")
	writeFile(t, filepath.Join(project, ".codex", "agents", "devrites-slice-wright.toml"), "name = \"devrites-slice-wright\"\ndeveloper_instructions = \"codex wright\"\n")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunContext(root, []string{"feat", "--phase", "build", "--role", "slice-wright", "--skills-root", ".agents/skills"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d\n%s%s", code, stdout, stderr)
	}
	bundle, err := os.ReadFile(filepath.Join(root, "work", "feat", "ctx", "build-slice-wright.bundle.md"))
	if err != nil || !strings.Contains(string(bundle), "===== .codex/agents/devrites-slice-wright.toml (not inlined") ||
		!strings.Contains(string(bundle), "read .codex/agents/devrites-slice-wright.toml only if") || strings.Contains(string(bundle), "codex wright") {
		t.Fatalf("codex role contract stub missing: %v\n%s", err, bundle)
	}
}

// A role contract the host loads as the agent's system prompt (.claude,
// .omp) is stubbed with its size and SHA-256; an unverified layout (.pi) and
// --inline-contract keep the body, and a missing contract still fails.
func TestContextRoleContractStubOnNativeHosts(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "# State\n")
	writeFile(t, filepath.Join(root, "work", "feat", "tasks.md"), "## SLICE-001 x\n")
	body := "# reviewer body\nrule\n"
	bundle := func(host string, extra ...string) (int, string, string) {
		t.Helper()
		skills := filepath.Join(project, host, "skills")
		writeFile(t, filepath.Join(skills, "devrites-lib", "reference", "standards", "core.md"), "# core\n")
		writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"]} -->\n")
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := RunContext(root, append([]string{"feat", "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--skills-root", skills}, extra...), stdout, stderr)
		data, _ := os.ReadFile(filepath.Join(root, "work", "feat", "ctx", "build-code-reviewer.SLICE-001.bundle.md"))
		return code, string(data), stdout.String() + stderr.String()
	}
	stub := fmt.Sprintf("(not inlined: your host loads this role contract as your system prompt; %d bytes, sha256 %x;", len(body), sha256.Sum256([]byte(body)))
	for _, host := range []string{".claude", ".omp"} {
		agent := filepath.Join(project, host, "agents", "devrites-code-reviewer.md")
		writeFile(t, agent, body)
		code, text, out := bundle(host)
		if code != 0 || !strings.Contains(text, "===== agents/devrites-code-reviewer.md "+stub) ||
			!strings.Contains(text, "read "+host+"/agents/devrites-code-reviewer.md only if") || strings.Contains(text, "# reviewer body") {
			t.Fatalf("%s: want contract stub, code=%d\n%s\n%s", host, code, out, text)
		}
		if code, text, out := bundle(host, "--inline-contract"); code != 0 || !strings.Contains(text, "===== agents/devrites-code-reviewer.md =====\n# reviewer body") {
			t.Fatalf("%s --inline-contract: want body, code=%d\n%s\n%s", host, code, out, text)
		}
		if err := os.Remove(agent); err != nil {
			t.Fatal(err)
		}
		if code, _, out := bundle(host); code != 3 || !strings.Contains(out, "MISSING: agents/devrites-code-reviewer.md") {
			t.Fatalf("%s: missing contract must fail, code=%d\n%s", host, code, out)
		}
	}
	writeFile(t, filepath.Join(project, ".pi", "agents", "devrites-code-reviewer.md"), body)
	if code, text, out := bundle(".pi"); code != 0 || !strings.Contains(text, "# reviewer body") || strings.Contains(text, "not inlined") {
		t.Fatalf(".pi: want inlined body, code=%d\n%s\n%s", code, out, text)
	}
}

package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reviewerPacketPath(t *testing.T) string {
	t.Helper()
	return writePacket(t, "# Scope\nReview the SLICE-001 diff against AC-001.\nReturn the canonical shape: Outcome: <findings | no-findings | gap>.\n")
}

func writePacket(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "packet.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLintReviewerPacket(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"clean", "Scope: SLICE-006 diff; rubric: testing.md.\nOutcome: <findings | no-findings | gap>\n", ""},
		{"gate token", "Gate result: ROOT-NAMES-PASS.\n", "ROOT-NAMES-PASS"},
		{"sibling outcome", "Code review: no-findings.\n", "no-findings"},
		{"result line", "Outcome: findings\n", "Outcome:"},
		{"already passed", "The smoke test already passed on this candidate.\n", "already passed"},
		{"zero counts", "Prior pass: Critical 0, Important 0.\n", "Critical 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := lintReviewerPacket(writePacket(t, tc.body))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("clean packet refused: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
	if err := lintReviewerPacket(""); err == nil || !strings.Contains(err.Error(), "needs --packet") {
		t.Fatalf("missing flag: %v", err)
	}
	for role, want := range map[string]bool{"code-reviewer": true, "code-reviewer.frontend": true, "security-auditor": true, "test-analyst": true, "devrites-test-analyst": true, "devrites-code-reviewer.backend": true, "slice-wright": false, "proof-runner": false, "plan-drafter": false, "devrites-plan-drafter": false} {
		if isReviewerRole(role) != want {
			t.Fatalf("isReviewerRole(%q) = %v", role, !want)
		}
	}
	if keys := countKeys("devrites-plan-reviewer.a"); !strings.Contains(strings.Join(keys, ","), "mechanism") {
		t.Fatalf("prefixed plan-reviewer must use its own Counts vocabulary, got %v", keys)
	}
}

// TestLintReviewerPacketObservedSeeding replays seeding lines that voided real
// reviews (paraphrased) and clean lines from real reviewer packets.
func TestLintReviewerPacketObservedSeeding(t *testing.T) {
	cases := []struct {
		name, body string
		roles      []string
		want       string
	}{
		{"parent narration", "Parent observed three completed passes: reservation RED -> restore -> join makes all3green.\n", []string{"code-reviewer"}, "Parent observed"},
		{"green outcome", "The join makes all3green.\n", []string{"test-analyst"}, "makes all3green"},
		{"root check", "Root scope check passed for SLICE-007.\n", []string{"code-reviewer"}, "check passed"},
		{"returned severity", "ListingPlan returned Important F1, confidence 9.\n", []string{"doubt-reviewer"}, "returned Important"},
		{"severity fingerprint", "Recheck Important F2 against the diff.\n", []string{"code-reviewer"}, "Important F2"},
		{"drafter self-attack", "Drafter self-attack exposes no route to a false pass.\n", []string{"plan-reviewer"}, "Drafter self-attack exposes"},
		{"already fixed", "AC-002 is already fixed in the candidate.\n", []string{"code-reviewer"}, "already fixed"},
		{"closed by", "The race is closed by D-012.\n", []string{"code-reviewer"}, "closed by"},
		{"return file", "Prior pass: .devrites/work/feat/packets/r1-code.return.md\n", []string{"code-reviewer"}, "packets/r1-code.return.md"},
		{"other role fingerprint", "Doubt F1: the reservation can outlive its owner.\n", []string{"plan-reviewer"}, "Doubt F1"},
		{"mixed wave fingerprints", "Plan F1 and DevEx S1 stay open.\n", []string{"plan-reviewer", "devex-reviewer"}, "Plan F1"},
		{"fingerprint without roles", "Plan F1 stays open.\n", nil, "Plan F1"},
		{"agent file reference", "Role: code-reviewer (agents/devrites-code-reviewer.md).\n", []string{"code-reviewer"}, ""},
		{"own fingerprint recheck", "Plan F1: the rollback step is missing; check the correction diff.\n", []string{"plan-reviewer"}, ""},
		{"own fingerprint across lanes", "Plan F1 (Important): rollback step.\n", []string{"plan-reviewer.a", "plan-reviewer.b"}, ""},
		{"prefixed role fingerprint", "Code F2: unchecked error.\n", []string{"devrites-code-reviewer"}, ""},
		{"perf fingerprint", "Perf F1: N+1 query.\n", []string{"performance-reviewer"}, ""},
		{"placeholder fingerprint", "Fingerprint: <Plan F1 | Code F2>\n", []string{"code-reviewer", "test-analyst"}, ""},
		{"requirement red green", "AC-004 needs RED, restoration, normal GREEN.\n", []string{"test-analyst"}, ""},
		{"do not run commands", "Do not run commands; read the diff at evidence/r2.diff.\n", []string{"code-reviewer"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := lintReviewerWavePacket(writePacket(t, tc.body), tc.roles)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("clean packet refused: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "agents.md § Independence") {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
}

func TestLintKnownRoles(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	writeFile(t, filepath.Join(project, ".claude", "skills", "devrites-lib", "core.md"), "core\n")
	writeFile(t, filepath.Join(project, ".claude", "skills", "rite-vet", "SKILL.md"), `<!-- loads: {"agents":{"vet-scout":"devrites-evidence-scout.md"}} -->`+"\n")
	for _, name := range []string{"devrites-code-reviewer.md", "devrites-test-analyst.md", "devrites-evidence-scout.md"} {
		writeFile(t, filepath.Join(project, ".claude", "agents", name), "agent\n")
	}
	if err := lintKnownRoles(root, "build", []string{"code-reviewer.backend", "devrites-test-analyst"}); err != nil {
		t.Fatalf("canonical, lane and prefixed roles must pass: %v", err)
	}
	if err := lintKnownRoles(root, "Vet", []string{"vet-scout"}); err != nil {
		t.Fatalf("a loads-manifest agents override must resolve: %v", err)
	}
	for _, role := range []string{"test-analyst-022-r5", "slice-wright-SLICE-022", "vet-scout", "../x", "x/../../agents/devrites-code-reviewer"} {
		if err := lintKnownRoles(root, "build", []string{"code-reviewer", role}); err == nil || !strings.Contains(err.Error(), role) || !strings.Contains(err.Error(), "names no pack agent") {
			t.Fatalf("free-form role %q: %v", role, err)
		}
	}
	// Non-Claude host layouts: omp keeps agents beside skills, Codex keeps
	// them at .codex/agents/<name>.toml.
	omp := t.TempDir()
	writeFile(t, filepath.Join(omp, ".omp", "skills", "devrites-lib", "core.md"), "core\n")
	writeFile(t, filepath.Join(omp, ".omp", "agents", "devrites-code-reviewer.md"), "agent\n")
	codex := t.TempDir()
	writeFile(t, filepath.Join(codex, ".agents", "skills", "devrites-lib", "core.md"), "core\n")
	writeFile(t, filepath.Join(codex, ".codex", "agents", "devrites-code-reviewer.toml"), "agent\n")
	for _, project := range []string{omp, codex} {
		hostRoot := filepath.Join(project, ".devrites")
		if err := lintKnownRoles(hostRoot, "build", []string{"devrites-code-reviewer.backend"}); err != nil {
			t.Fatalf("%s: a canonical role must pass: %v", project, err)
		}
		if err := lintKnownRoles(hostRoot, "build", []string{"test-analyst-022-r5"}); err == nil || !strings.Contains(err.Error(), "names no pack agent") {
			t.Fatalf("%s: a free-form role must be refused: %v", project, err)
		}
	}
	// No project-level install: nothing to check against, so any role passes.
	bare, _ := newWorkspace(t, "build")
	if err := lintKnownRoles(bare, "build", []string{"a", "test-analyst-022-r5"}); err != nil {
		t.Fatalf("roles without a project install must pass: %v", err)
	}
}

// TestLintRenderedPacketUsesItsRole keeps `packet` linting the way dispatch
// open does: a recheck may name its own role's fingerprint, not another's.
func TestLintRenderedPacketUsesItsRole(t *testing.T) {
	path := writePacket(t, "Open findings:\n- F1 plan.md:12 Plan F1 rollback step missing\n")
	if err := lintRenderedPacket("", "feature", path, "plan-reviewer", false, false); err != nil {
		t.Fatalf("a plan-reviewer recheck naming Plan F1 must pass: %v", err)
	}
	if err := lintRenderedPacket("", "feature", path, "code-reviewer", false, false); err == nil || !strings.Contains(err.Error(), "Plan F1") {
		t.Fatalf("Plan F1 in a code-reviewer packet must be refused: %v", err)
	}
}

func TestDispatchOpenRefusesSeededReviewerPacket(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	seeded := writePacket(t, "Scope: SLICE-006.\nROOT-NAMES already printed ROOT-NAMES-PASS.\n")
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "r1", "--role", "code-reviewer", "--role", "test-analyst"); code != 3 || !strings.Contains(out, "needs --packet") {
		t.Fatalf("missing packet: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "r1", "--role", "code-reviewer", "--packet", seeded); code != 3 || !strings.Contains(out, "seeds a conclusion") {
		t.Fatalf("seeded packet: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "r1", "--role", "code-reviewer", "--packet", reviewerPacketPath(t)); code != 0 {
		t.Fatalf("clean packet after refusals must open: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "plan", "--wave", "p1", "--role", "plan-drafter"); code != 0 {
		t.Fatalf("non-reviewer waves need no packet: code=%d\n%s", code, out)
	}
}

// TestDispatchOpenLintsWaveRoles keeps open on the same role checks as
// packet: a free-form role is refused against a project install, and a
// fingerprint tagged with a role in the wave passes.
func TestDispatchOpenLintsWaveRoles(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	writeFile(t, filepath.Join(root, "work", "feat", "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n")
	writeFile(t, filepath.Join(project, ".claude", "skills", "devrites-lib", "core.md"), "core\n")
	for _, name := range []string{"devrites-code-reviewer.md", "devrites-plan-reviewer.md"} {
		writeFile(t, filepath.Join(project, ".claude", "agents", name), "agent\n")
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "r1", "--role", "test-analyst-022-r5", "--packet", reviewerPacketPath(t)); code != 3 || !strings.Contains(out, "names no pack agent") {
		t.Fatalf("free-form role: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "r1", "--role", "code-reviewer", "--packet", reviewerPacketPath(t)); code != 0 {
		t.Fatalf("canonical role: code=%d\n%s", code, out)
	}
	recheck := writePacket(t, "Open findings:\n- F1 plan.md:12 Plan F1 rollback step missing\n")
	if code, out := runDispatchT(t, root, "open", "--phase", "plan", "--wave", "p1", "--role", "plan-reviewer", "--packet", recheck); code != 0 {
		t.Fatalf("a plan-reviewer recheck naming Plan F1 must open: code=%d\n%s", code, out)
	}
}

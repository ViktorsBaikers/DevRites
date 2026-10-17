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
	for role, want := range map[string]bool{"code-reviewer": true, "code-reviewer.frontend": true, "security-auditor": true, "test-analyst": true, "slice-wright": false, "proof-runner": false, "plan-drafter": false} {
		if isReviewerRole(role) != want {
			t.Fatalf("isReviewerRole(%q) = %v", role, !want)
		}
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

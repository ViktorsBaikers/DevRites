package main_test

// TestParityBuildReadiness checks the build-readiness gate's stdout + exit code
// against the golden snapshot for each state.md fixture.

import "testing"

func TestParityBuildReadiness(t *testing.T) {
	work := t.TempDir()

	// Status halts (stderr-only messages; parity is stdout + exit code).
	writeFeatureFile(t, work, "await", "state.md", "- Status: awaiting_human\n") // exit 3
	writeFeatureFile(t, work, "blocked", "state.md", "- Status: blocked\n")      // exit 4

	// Ready: plan approved + status.
	writeFeatureFile(t, work, "approved", "state.md", "- Plan approved: 2024-01-01\n- Status: running\n") // exit 0

	// Plan not approved: missing line, or the literal "none".
	writeFeatureFile(t, work, "noapprove", "state.md", "- Status: running\n")                          // exit 2
	writeFeatureFile(t, work, "approvenone", "state.md", "- Plan approved: none\n- Status: running\n") // exit 2

	// field() strip: a trailing " | none" / " # note" annotation is trimmed off
	// Status before the case match, so both stay "running" and reach OK.
	writeFeatureFile(t, work, "trailpipe", "state.md", "- Plan approved: 2024-01-01\n- Status: running | none\n") // exit 0
	writeFeatureFile(t, work, "trailhash", "state.md", "- Plan approved: 2024-01-01\n- Status: running # note\n") // exit 0

	// Empty Status value + plan approved → status renders as "running".
	writeFeatureFile(t, work, "emptystatus", "state.md", "- Plan approved: 2024-01-01\n- Status: \n") // exit 0

	// "ghost" is intentionally never created → no workspace/state.md (exit 5).
	for _, arg := range []string{
		"await", "blocked", "approved", "noapprove", "approvenone",
		"trailpipe", "trailhash", "emptystatus", "ghost",
	} {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  []string{"build-readiness", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}
}

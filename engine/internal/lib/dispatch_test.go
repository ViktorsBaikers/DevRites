package lib

import (
	"bytes"
	"strings"
	"testing"
)

func runDispatchT(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunDispatch(root, append([]string{"feat"}, args...), stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestDispatchRoundCapIgnoresSliceZeroPaddingAndPhaseCase(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	ids := []string{"SLICE-001", "SLICE-1", "SLICE-01", "SLICE-0001", "SLICE-001"}
	phases := []string{"build", "Build", "BUILD", "build", "build"}
	for i := range ids {
		wave := "r" + string(rune('1'+i))
		if code, out := runDispatchT(t, root, "open", "--phase", phases[i], "--wave", wave, "--role", "slice-wright", "--slice", ids[i]); code != 0 || !strings.Contains(out, "wright round "+string(rune('1'+i))) {
			t.Fatalf("round %d (%s, %s): code=%d\n%s", i+1, ids[i], phases[i], code, out)
		}
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "Build", "--wave", "r6", "--role", "slice-wright", "--slice", "SLICE-1"); code != 3 || !strings.Contains(out, "BLOCKED") {
		t.Fatalf("round 6 must block however the slice/phase is spelled: code=%d\n%s", code, out)
	}
}

func TestDispatchAbandonRefusesCompleteWave(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	for _, args := range [][]string{
		{"open", "--phase", "review", "--wave", "w", "--role", "a"},
		{"start", "--wave", "w", "--role", "a", "--handle", "h"},
		{"seal", "--wave", "w"},
		{"return", "--wave", "w", "--role", "a"},
	} {
		if code, out := runDispatchT(t, root, args...); code != 0 {
			t.Fatalf("%v: code=%d\n%s", args, code, out)
		}
	}
	if code, out := runDispatchT(t, root, "abandon", "--wave", "w", "--reason", "redo"); code != 3 || !strings.Contains(out, "complete") {
		t.Fatalf("abandon of a complete wave must be refused: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "status", "--wave", "w"); code != 0 || !strings.Contains(out, "wave w complete") {
		t.Fatalf("refused abandon changed the wave: code=%d\n%s", code, out)
	}
}

func TestDispatchAggregateStatusListsAbandonedWithoutFailing(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	runDispatchT(t, root, "open", "--phase", "review", "--wave", "w", "--role", "a")
	runDispatchT(t, root, "abandon", "--wave", "w", "--reason", "host limit")
	if code, out := runDispatchT(t, root, "status"); code != 0 || !strings.Contains(out, "wave w state=abandoned") {
		t.Fatalf("aggregate status after abandon: code=%d\n%s", code, out)
	}
	runDispatchT(t, root, "open", "--phase", "review", "--wave", "w2", "--role", "a")
	if code, _ := runDispatchT(t, root, "status"); code == 0 {
		t.Fatal("aggregate status must still fail while a wave is open")
	}
}

func TestDispatchAbandonedStartedWaveCountsTowardRoundCap(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	for i := 1; i <= 5; i++ {
		wave := "s" + string(rune('0'+i))
		for _, args := range [][]string{
			{"open", "--phase", "build", "--wave", wave, "--role", "slice-wright", "--slice", "SLICE-001"},
			{"start", "--wave", wave, "--role", "slice-wright", "--handle", "h"},
		} {
			if code, out := runDispatchT(t, root, args...); code != 0 {
				t.Fatalf("round %d %v: code=%d\n%s", i, args, code, out)
			}
		}
		// abandon records the wave and exits 3 (handoff required).
		if code, out := runDispatchT(t, root, "abandon", "--wave", wave, "--reason", "redo"); code != 3 || !strings.Contains(out, "abandoned") {
			t.Fatalf("round %d abandon: code=%d\n%s", i, code, out)
		}
	}
	// An abandoned wave whose wright never started does not spend a round.
	runDispatchT(t, root, "open", "--phase", "build", "--wave", "u", "--role", "slice-wright", "--slice", "SLICE-001")
	runDispatchT(t, root, "abandon", "--wave", "u", "--reason", "host limit")
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "s6", "--role", "slice-wright", "--slice", "SLICE-001"); code != 3 || !strings.Contains(out, "BLOCKED") {
		t.Fatalf("five started-then-abandoned rounds must hit the cap: code=%d\n%s", code, out)
	}
}

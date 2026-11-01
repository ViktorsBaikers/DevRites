package parallel

import (
	"bytes"
	"strings"
	"testing"
)

func TestSelectGreedyOneThenMany(t *testing.T) {
	t.Parallel()
	round1, err := SelectGreedy(5, []SlicePaths{
		{ID: "SLICE-001", Paths: []string{"src/a.go"}},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(round1) != 1 || round1[0].ID != "SLICE-001" {
		t.Fatalf("round1=%#v", round1)
	}

	round2, err := SelectGreedy(5, []SlicePaths{
		{ID: "SLICE-002", Paths: []string{"src/b.go"}},
		{ID: "SLICE-003", Paths: []string{"src/c.go"}},
		{ID: "SLICE-004", Paths: []string{"src/d.go"}},
		{ID: "SLICE-005", Paths: []string{"src/e.go"}},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(round2) != 4 {
		t.Fatalf("after a one-slice round, cap 5 with 4 ready must select 4, got %d (%#v)", len(round2), round2)
	}
}

func TestSelectGreedySkipsOverlapKeepsCap(t *testing.T) {
	t.Parallel()
	got, err := SelectGreedy(5, []SlicePaths{
		{ID: "SLICE-001", Paths: []string{"src/shared.go"}},
		{ID: "SLICE-002", Paths: []string{"src/shared.go", "src/b.go"}},
		{ID: "SLICE-003", Paths: []string{"src/c.go"}},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "SLICE-001" || got[1].ID != "SLICE-003" {
		t.Fatalf("got %#v", got)
	}
}

func TestSelectGreedyRejectsBadCap(t *testing.T) {
	t.Parallel()
	if _, err := SelectGreedy(0, nil, ""); err == nil {
		t.Fatal("cap 0 should fail")
	}
	if _, err := SelectGreedy(11, nil, ""); err == nil {
		t.Fatal("cap 11 should fail")
	}
}

func TestSelectCLIOneThenMany(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	in := strings.NewReader(`{"slices":[{"id":"SLICE-001","paths":["src/a.go"]}]}`)
	code := Run("parallel", []string{"select", "--cap", "5", "-"}, in, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if got := stdout.String(); got != "select: n_eff=1 mode=serial slices=SLICE-001\n" {
		t.Fatalf("round1 got %q", got)
	}

	stdout.Reset()
	stderr.Reset()
	in = strings.NewReader(`{"slices":[
		{"id":"SLICE-002","paths":["src/b.go"]},
		{"id":"SLICE-003","paths":["src/c.go"]},
		{"id":"SLICE-004","paths":["src/d.go"]}
	]}`)
	code = Run("parallel", []string{"select", "--cap", "5", "-"}, in, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if got := stdout.String(); got != "select: n_eff=3 mode=parallel slices=SLICE-002,SLICE-003,SLICE-004\n" {
		t.Fatalf("round2 got %q; must not stay n_eff=1", got)
	}
}

func TestSelectCLIRequiresCap(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := Run("parallel", []string{"select", "-"}, strings.NewReader(`{"slices":[]}`), &stdout, &stderr)
	if code != ExitUsage {
		t.Fatalf("code=%d want %d stderr=%s", code, ExitUsage, stderr.String())
	}
}

// --from-plan takes the ready set from PlanReady instead of JSON input.
func TestSelectCLIFromPlan(t *testing.T) {
	saved := PlanReady
	t.Cleanup(func() { PlanReady = saved })
	root := t.TempDir()
	run := func(args ...string) (int, string, string) {
		var stdout, stderr bytes.Buffer
		code := Run("parallel", append([]string{"select", "--cap", "5", "--root", root}, args...), strings.NewReader(""), &stdout, &stderr)
		return code, stdout.String(), stderr.String()
	}

	PlanReady = nil
	if code, _, stderr := run("--slug", "feat", "--from-plan"); code != ExitBlocked || !strings.Contains(stderr, "unavailable") {
		t.Fatalf("unwired hook: code=%d stderr=%s", code, stderr)
	}
	PlanReady = func(root, slug string) ([]SlicePaths, error) {
		if slug != "feat" {
			t.Fatalf("slug=%q", slug)
		}
		return []SlicePaths{
			{ID: "SLICE-004", Paths: []string{"src/a.go"}},
			{ID: "SLICE-005", Paths: []string{"src/b.go"}},
			{ID: "SLICE-006", Paths: []string{"src/A.go"}},
		}, nil
	}
	code, stdout, stderr := run("--root", t.TempDir(), "--slug", "feat", "--from-plan")
	if code != ExitOK || stdout != "select: n_eff=2 mode=parallel slices=SLICE-004,SLICE-005\n" {
		t.Fatalf("code=%d stdout=%q stderr=%s", code, stdout, stderr)
	}
	for _, args := range [][]string{{"--from-plan"}, {"--slug", "feat", "--from-plan", "slices.json"}} {
		if code, _, _ := run(args...); code != ExitUsage {
			t.Fatalf("%v: code=%d want %d", args, code, ExitUsage)
		}
	}
}

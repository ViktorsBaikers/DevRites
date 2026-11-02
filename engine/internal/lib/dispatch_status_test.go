package lib

import (
	"fmt"
	"strings"
	"testing"
)

// Bare status folds complete waves into one count line but keeps every open,
// sealed, and abandoned row and the nonzero exit; --all lists every wave.
func TestDispatchBareStatusFoldsCompleteWaves(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	f := &dispatchFile{Waves: map[string]*dispatchWave{
		"open-1": {Phase: "build", State: "open", Roles: map[string]*dispatchRole{"a": {}}},
		"gone-1": {Phase: "build", State: "abandoned", Roles: map[string]*dispatchRole{"a": {}}},
	}}
	for i := range 40 {
		f.Waves[fmt.Sprintf("done-%02d", i)] = &dispatchWave{Phase: "review", State: "complete", Roles: map[string]*dispatchRole{"a": {Handle: fmt.Sprintf("h-%02d", i), StartedAt: "2026-10-15T10:00:00Z", Returned: true, ReturnedAt: "2026-10-15T10:30:00Z"}}}
	}
	if err := saveDispatch(dispatchPath(root, "feat"), f); err != nil {
		t.Fatal(err)
	}
	code, out := runDispatchT(t, root, "status")
	if code == 0 || !strings.Contains(out, "wave open-1 state=open") || !strings.Contains(out, "wave gone-1 state=abandoned") ||
		!strings.Contains(out, "40 complete wave(s) (use --all)") || strings.Contains(out, "state=complete") || !strings.Contains(out, "waves not complete: open-1") {
		t.Fatalf("bare status: code=%d\n%s", code, out)
	}
	code, out = runDispatchT(t, root, "status", "--all")
	if code == 0 || strings.Count(out, "state=complete") != 40 || strings.Count(out, "dispatch: wave ") != 42 || strings.Contains(out, "use --all") {
		t.Fatalf("status --all: code=%d\n%s", code, out)
	}
	f.Waves["open-1"].State = "complete"
	f.Waves["open-1"].Roles["a"] = &dispatchRole{Handle: "h-open", StartedAt: "2026-10-15T11:00:00Z", Returned: true, ReturnedAt: "2026-10-15T11:30:00Z"}
	if err := saveDispatch(dispatchPath(root, "feat"), f); err != nil {
		t.Fatal(err)
	}
	if code, out := runDispatchT(t, root, "status"); code != 0 || !strings.Contains(out, "41 complete wave(s)") {
		t.Fatalf("all complete: code=%d\n%s", code, out)
	}
}

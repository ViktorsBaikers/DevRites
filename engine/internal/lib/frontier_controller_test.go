package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/parallel"
)

func controllerFixture(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".devrites")
	dir := poolWorkspace(t, root, "current16", "", poolSlice("SLICE-001", "src/current.go", "Workspace dependencies: none"))
	frontierFixture(t, root, "work", "current16", "", "build", "running", "/rite-build SLICE-001", 1, "2")
	writeFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 3 | future18 |\n")
	poolVet(t, root, "current16")
	writeFile(t, filepath.Join(root, "ACTIVE"), "current16\n")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
	return root, dir
}

func controllerDispatch(t *testing.T, root, slug string, args ...string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := RunDispatch(root, append([]string{slug}, args...), &out, &errOut)
	return code, out.String() + errOut.String()
}

func controllerOpen(t *testing.T, root, slug, wave, id string, extra ...string) (int, string) {
	t.Helper()
	args := []string{"open", "--phase", "build", "--wave", wave, "--role", "slice-wright", "--slice", id, "--contract", wrightContractPath(t)}
	return controllerDispatch(t, root, slug, append(args, extra...)...)
}

func controllerSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := frontierSnapshot(t, root)
	// Locks may be created by a refused command; no workflow account may be.
	for path := range files {
		if strings.HasSuffix(path, ".lock") {
			delete(files, path)
		}
	}
	return files
}

func controllerPlanning(t *testing.T, root, phase, role string, started bool) {
	t.Helper()
	frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "1")
	args := []string{"open", "--phase", phase, "--wave", "prepare17", "--role", role}
	if isReviewerRole(role) {
		args = append(args, "--packet", reviewerPacketPath(t))
	}
	if code, out := controllerDispatch(t, root, "future17", args...); code != 0 {
		t.Fatalf("planning open: code=%d\n%s", code, out)
	}
	if started {
		if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "prepare17", "--role", role, "--handle", "native-17"); code != 0 {
			t.Fatalf("planning start: code=%d\n%s", code, out)
		}
	}
}

func controllerReadyFuture(t *testing.T, root, paths string) string {
	t.Helper()
	dir := poolWorkspace(t, root, "future17", "current16", poolSlice("SLICE-001", paths, "Workspace dependencies: none"))
	frontierFixture(t, root, "work", "future17", "current16", "vet", "ready", "/rite-build SLICE-001", 2, "0")
	poolVet(t, root, "future17")
	return dir
}

func controllerLease(t *testing.T, root string, capacity int, rows ...parallel.LeaseSlice) {
	t.Helper()
	path, err := parallel.LeasePath(filepath.Dir(root), "current16")
	if err != nil {
		t.Fatal(err)
	}
	lease := &parallel.Lease{
		BatchID: "pipeline", CreatedAt: "2026-11-05T00:00:00Z", BaseSHA: "abcdef1",
		N: len(rows), Capacity: capacity, Status: parallel.StatusRunning, ControlPIDOrSession: "native-controller",
		Slices: rows,
	}
	if err := parallel.WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
}

func controllerLeaseRow(workspace, id, path string) parallel.LeaseSlice {
	key := id
	if workspace != "" {
		key = "_workspaces/" + workspace + "/" + id
	}
	return parallel.LeaseSlice{
		Workspace: workspace, ID: id, Paths: []string{path},
		WorktreePath: "/tmp/controller-fixture/" + key,
		Branch:       "devrites/parallel/current16/pipeline/" + key, WrightStatus: parallel.WrightPending,
	}
}

func TestAutocompleteControllerRefusesUnstartedSuccessorBeforeWorkflowWrites(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*testing.T, string, string)
	}{
		{"missing", func(t *testing.T, root, dir string) {}},
		{"placeholder", func(t *testing.T, root, dir string) {
			frontierFixture(t, root, "work", "future17", "current16", "spec", "running", "/rite-spec future17", 2, "1")
			writeFile(t, filepath.Join(root, "work", "future17", "spec.md"), "# Spec\n\nPreparation not yet started.\n")
		}},
		{"prose preparation claim", func(t *testing.T, root, dir string) {
			writeFile(t, filepath.Join(dir, "evidence.md"), "# Evidence\n\nfuture17 planning is in progress; host will launch it later.\n")
		}},
		{"declared but no handle", func(t *testing.T, root, dir string) {
			controllerPlanning(t, root, "plan", "plan-drafter", false)
		}},
		{"unrelated proof handle", func(t *testing.T, root, dir string) {
			controllerPlanning(t, root, "prove", "proof-runner", true)
		}},
		{"Build review is not preparation", func(t *testing.T, root, dir string) {
			controllerPlanning(t, root, "build", "code-reviewer", true)
		}},
		{"source planning handle is not successor handle", func(t *testing.T, root, dir string) {
			for _, args := range [][]string{
				{"open", "--phase", "plan", "--wave", "source-plan", "--role", "plan-drafter"},
				{"start", "--wave", "source-plan", "--role", "plan-drafter", "--handle", "native-source"},
			} {
				if code, out := controllerDispatch(t, root, "current16", args...); code != 0 {
					t.Fatalf("source planning: %d %s", code, out)
				}
			}
		}},
		{"completed planning without readiness", func(t *testing.T, root, dir string) {
			controllerPlanning(t, root, "plan", "plan-drafter", true)
			if code, out := controllerDispatch(t, root, "future17", "return", "--wave", "prepare17", "--role", "plan-drafter", "--handle", "native-17"); code != 0 {
				t.Fatalf("planning return: %d %s", code, out)
			}
		}},
		{"abandoned planning handle", func(t *testing.T, root, dir string) {
			controllerPlanning(t, root, "plan", "plan-drafter", true)
			if code, out := controllerDispatch(t, root, "future17", "abandon", "--wave", "prepare17", "--reason", "host confirmed terminal", "--release"); code != 3 {
				t.Fatalf("planning abandon: %d %s", code, out)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := controllerFixture(t)
			tc.setup(t, root, dir)
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/current.go")
			if code != 3 || !strings.Contains(out, "future17") || !strings.Contains(out, "BLOCKED") {
				t.Fatalf("unstarted successor admitted: code=%d\n%s", code, out)
			}
			if !strings.Contains(out, "/rite-") || !strings.Contains(out, "frontier-blocked:") {
				t.Errorf("refusal lacks native remedy and explicit deferral alternative:\n%s", out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Error("refused writer changed claims, history, rounds, dispatch or other workflow artifacts")
			}
		})
	}
}

func TestAutocompleteControllerActualSuccessorPreparationAdmitsCurrentWriter(t *testing.T) {
	for _, tc := range []struct{ name, phase, role string }{
		{"planning", "plan", "plan-drafter"},
		{"native Vet review", "vet", "plan-reviewer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := controllerFixture(t)
			controllerPlanning(t, root, tc.phase, tc.role, true)
			before := controllerSnapshot(t, filepath.Join(root, "work", "future17"))
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/current.go")
			if code != 0 || !strings.Contains(out, "wave writer16 open") || !strings.Contains(out, "wright round 1") {
				t.Fatalf("actual preparation did not admit writer: code=%d\n%s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, filepath.Join(root, "work", "future17"))) {
				t.Error("source admission mutated successor's native planning account")
			}
			if _, err := os.Stat(filepath.Join(root, "work", "future18")); !os.IsNotExist(err) {
				t.Fatalf("immediate preparation required or created entire sequence: %v", err)
			}
		})
	}
}

func TestAutocompleteControllerDeferralMustNameConcreteTarget(t *testing.T) {
	for _, tc := range []struct {
		name, record string
		allowed      bool
	}{
		{"specific constraint", "- frontier-blocked: future17 — upstream API contract awaits owner decision DEC-017\n", true},
		{"different successor", "- frontier-blocked: future18 — upstream API awaits DEC-018\n", false},
		{"empty constraint", "- frontier-blocked: future17 — \n", false},
		{"generic local serial", "- serial: SLICE-001 — shared go.mod requires serial writes\n", false},
		{"fenced example", "```text\nfrontier-blocked: future17 — upstream API awaits DEC-017\n```\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := controllerFixture(t)
			writeFile(t, filepath.Join(dir, "evidence.md"), "# Evidence\n\n"+tc.record)
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001")
			if tc.allowed {
				if code != 0 || !strings.Contains(out, "future17") || !strings.Contains(out, "frontier-blocked") {
					t.Fatalf("concrete scheduling deferral not visible/admitted: %d\n%s", code, out)
				}
			} else if code != 3 || !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatalf("unrelated/empty/prose deferral admitted or mutated workflow: %d\n%s", code, out)
			}
		})
	}
}

func TestAutocompleteControllerSequenceNormalizationIsSeparateDeferral(t *testing.T) {
	for _, tc := range []struct {
		record  string
		allowed bool
	}{
		{"", false},
		{"frontier-blocked: future17 — waiting for owner\n", false},
		{"frontier-blocked: sequence — DEC-017 must reconcile renamed successor before enrollment\n", true},
	} {
		t.Run(tc.record, func(t *testing.T) {
			root, dir := controllerFixture(t)
			writeFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | renamed17 |\n")
			poolVet(t, root, "current16")
			writeFile(t, filepath.Join(dir, "evidence.md"), tc.record)
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001")
			if tc.allowed {
				if code != 0 || !strings.Contains(out, "sequence") || !strings.Contains(out, "frontier-blocked") {
					t.Fatalf("explicit sequence deferral lost: %d %s", code, out)
				}
			} else if code != 3 || !strings.Contains(out, "normaliz") || !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatalf("unresolved authority failed open or lacks normalization remedy: %d %s", code, out)
			}
		})
	}
}

func TestAutocompleteControllerPreservesMissingFutureLimitsAndStandaloneAFK(t *testing.T) {
	for _, tc := range []struct{ name, afk, remaining string }{
		{"sequence opt out", "mode: autocomplete\nmax_parallel: 2\ncontinue_sequence: false\n", "2"},
		{"zero future budget", "mode: autocomplete\nmax_parallel: 2\n", "0"},
		{"single capacity", "mode: autocomplete\nmax_parallel: 1\n", "2"},
		{"standalone AFK", "max_parallel: 2\n", "2"},
		{"human invoked without AFK", "", "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := controllerFixture(t)
			frontierFixture(t, root, "work", "current16", "", "build", "running", "/rite-build SLICE-001", 1, tc.remaining)
			if tc.afk == "" {
				if err := os.Remove(filepath.Join(root, "AFK")); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFile(t, filepath.Join(root, "AFK"), tc.afk)
			}
			if code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001"); code != 0 {
				t.Fatalf("existing admission contract changed: %d %s", code, out)
			}
		})
	}
}

func TestAutocompleteControllerInvalidIdentityCannotDisableAdmissionGate(t *testing.T) {
	for _, mode := range []string{"mode: unknown\n", "mode:\n", "mode: autocomplete\nmode: standalone\n", "mode: autocomplete\nmode: autocomplete\n"} {
		t.Run(mode, func(t *testing.T) {
			root, _ := controllerFixture(t)
			writeFile(t, filepath.Join(root, "AFK"), mode+"max_parallel: 2\n")
			before := controllerSnapshot(t, root)
			if code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001"); code == 0 || !strings.Contains(out, "mode") {
				t.Fatalf("invalid controller identity downgraded to standalone: %d %s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("invalid identity mutated workflow")
			}
		})
	}
}

func TestAutocompleteControllerReadyForeignPickCannotBeWaivedByLocalSerialRecord(t *testing.T) {
	for _, evidence := range []string{"", "serial: SLICE-001 — shared go.mod requires serial writes\n", "frontier-blocked: future17 — dependency owner unavailable\n"} {
		t.Run(evidence, func(t *testing.T) {
			root, dir := controllerFixture(t)
			controllerReadyFuture(t, root, "src/future.go")
			writeFile(t, filepath.Join(dir, "evidence.md"), evidence)
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001")
			if code != 3 || !strings.Contains(out, "future17/SLICE-001") || !strings.Contains(out, "--workspaces") {
				t.Fatalf("ready foreign work silently omitted: %d\n%s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("mixed-cohort refusal mutated workflow")
			}
		})
	}
}

func TestAutocompleteControllerMixedLeaseReservationsPermitSeparateSourceWaves(t *testing.T) {
	root, _ := controllerFixture(t)
	controllerReadyFuture(t, root, "src/future.go")
	controllerLease(t, root, 2,
		controllerLeaseRow("", "SLICE-001", "src/current.go"),
		controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
	for _, slug := range []string{"current16", "future17"} {
		if code, out := controllerOpen(t, root, slug, "writer", "SLICE-001"); code != 0 {
			t.Fatalf("reserved mixed cohort demanded atomic foreign wave: %s %d\n%s", slug, code, out)
		}
		if code, out := controllerDispatch(t, root, slug, "start", "--wave", "writer", "--role", "slice-wright", "--handle", "native-"+slug); code != 0 {
			t.Fatalf("reserved source start: %s %d %s", slug, code, out)
		}
	}
}

func TestAutocompleteControllerLeaseReservationsDoNotWaiveCapacityOrPaths(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		capacity   int
	}{
		{"global cap full", "src/extra.go", 2},
		{"foreign path held", "src/future.go", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := controllerFixture(t)
			controllerReadyFuture(t, root, "src/future.go")
			writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n"+poolSlice("SLICE-001", "src/current.go", "Workspace dependencies: none")+poolSlice("SLICE-002", tc.path, "Workspace dependencies: none"))
			poolVet(t, root, "current16")
			writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 3\n")
			controllerLease(t, root, tc.capacity,
				controllerLeaseRow("", "SLICE-001", "src/current.go"),
				controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
			before := controllerSnapshot(t, root)
			if code, out := controllerOpen(t, root, "current16", "unreserved", "SLICE-002"); code != 3 {
				t.Fatalf("unreserved writer bypassed %s: %d\n%s", tc.name, code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("unsafe writer mutated workflow")
			}
		})
	}
}

func TestAutocompleteControllerContinuationGlobalPriorityWithoutLocalRecursion(t *testing.T) {
	for _, kind := range []string{"missing future", "foreign ready", "normalization", "local only", "standalone"} {
		t.Run(kind, func(t *testing.T) {
			root, dir := controllerFixture(t)
			global := kind != "local only" && kind != "standalone"
			switch kind {
			case "foreign ready":
				controllerReadyFuture(t, root, "src/future.go")
			case "normalization":
				writeFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | renamed17 |\n")
				poolVet(t, root, "current16")
			case "local only":
				frontierFixture(t, root, "work", "current16", "", "build", "running", "/rite-build SLICE-001", 1, "0")
			case "standalone":
				writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
			}
			before := controllerSnapshot(t, root)
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"current16"}, &out, &errOut); code != 0 {
				t.Fatalf("continuation: %d %s", code, &errOut)
			}
			want := "invoke: /rite-build SLICE-001\n"
			if global {
				want = "invoke: /rite-autocomplete\n"
			}
			if !strings.Contains(out.String(), "continuation: required\n") || !strings.Contains(out.String(), want) {
				t.Fatalf("wrong continuation priority for %s:\n%s", kind, &out)
			}
			if global && strings.Contains(out.String(), "\ninvoke: /rite-build") || strings.Contains(out.String(), "\ninvoke: /rite-ship") {
				t.Fatalf("global work bypassed controller or implied Ship:\n%s", &out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("continuation changed local cursor/workflow")
			}
		})
	}
}

func TestAutocompleteControllerHumanHoldsDoNotInventMissingPreparation(t *testing.T) {
	for _, hold := range []string{"awaiting_human", "question cursor", "blocking", "escalating"} {
		t.Run(hold, func(t *testing.T) {
			root, dir := controllerFixture(t)
			status, extra := "running", ""
			if hold == "awaiting_human" {
				status = hold
			}
			if hold == "question cursor" {
				extra = "| question_id | Q-017 |\n"
			}
			frontierFixture(t, root, "work", "current16", "", "build", status, "/rite-build SLICE-001", 1, "2")
			if extra != "" {
				writeFile(t, filepath.Join(dir, "state.md"), cursor("build", status, "/rite-build SLICE-001")+"| sequence_position | 1 |\n| sequence_workspaces_remaining | 2 |\n"+extra)
			}
			if hold == "blocking" || hold == "escalating" {
				writeFile(t, filepath.Join(dir, "questions.md"), "## Q-017\nstatus: open\ngate: "+hold+"\n")
			}
			before := controllerSnapshot(t, root)
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"current16"}, &out, &errOut); code != 0 {
				t.Fatalf("held continuation: %d %s", code, &errOut)
			}
			if !strings.HasPrefix(out.String(), "continuation: clear\n") || strings.Contains(out.String(), "\ninvoke:") || strings.Contains(out.String(), "frontier-prepare:") {
				t.Fatalf("human hold invented preparation or Build approval:\n%s", &out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("human hold observation mutated workflow")
			}
		})
	}
}

func TestAutocompleteControllerDispatchReturnRefreshesFutureAfterClaimRelease(t *testing.T) {
	root, _ := controllerFixture(t)
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
	if code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/current.go"); code != 0 {
		t.Fatalf("writer open: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "writer16", "--role", "slice-wright", "--handle", "native-16"); code != 0 {
		t.Fatalf("writer start: %d %s", code, out)
	}
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
	before := controllerSnapshot(t, root)
	if code, out := controllerDispatch(t, root, "current16", "status", "--wave", "writer16"); code != 3 || strings.Contains(out, "frontier:") {
		t.Fatalf("status became controller mutation/dispatch hook: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "return", "--wave", "writer16", "--role", "slice-wright", "--handle", "wrong-host"); code != 3 || strings.Contains(out, "frontier:") {
		t.Fatalf("failed return emitted scheduling success: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("status or failed return changed workflow")
	}
	for _, replay := range []bool{false, true} {
		code, out := controllerDispatch(t, root, "current16", "return", "--wave", "writer16", "--role", "slice-wright", "--handle", "native-16")
		if code != 0 || strings.Count(out, "frontier-source: current16\n") != 1 || !strings.Contains(out, "frontier-prepare: /rite-spec 'future17' source=current16") {
			t.Fatalf("recorded return/reconciliation omitted fresh future scheduling (replay=%t): %d\n%s", replay, code, out)
		}
		released, frontier := strings.Index(out, "released claim"), strings.Index(out, "frontier:")
		if replay {
			released = strings.Index(out, "already released")
		}
		if released < 0 || frontier <= released {
			t.Fatalf("frontier emitted before claim mutation settled:\n%s", out)
		}
	}
}

func TestAutocompleteControllerExistingSuccessorHoldsPermitIndependentCurrentWriter(t *testing.T) {
	for _, hold := range []string{"awaiting_human", "blocking question", "producer dependency"} {
		t.Run(hold, func(t *testing.T) {
			root, _ := controllerFixture(t)
			dir := controllerReadyFuture(t, root, "src/future.go")
			switch hold {
			case "awaiting_human":
				frontierFixture(t, root, "work", "future17", "current16", "plan", "awaiting_human", "/rite-vet future17", 2, "1")
			case "blocking question":
				writeFile(t, filepath.Join(dir, "questions.md"), "## Q-017\nstatus: open\ngate: blocking\n")
			case "producer dependency":
				writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n"+poolSlice("SLICE-001", "src/future.go", "Workspace dependencies: current16/SLICE-001"))
				poolVet(t, root, "future17")
			}
			if code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001"); code != 0 {
				t.Fatalf("successor %s prevented independent source progress: %d\n%s", hold, code, out)
			}
		})
	}
}

func TestAutocompleteControllerQualifiedSerialRecordPreservesOrderedHandoff(t *testing.T) {
	for _, tc := range []struct {
		name, record string
		allowed      bool
	}{
		{"qualified foreign pick", "serial: future17/SLICE-001 — host cannot isolate workers; ordered handoff required\n", true},
		{"same local slice number", "serial: SLICE-001 — host cannot isolate workers; ordered handoff required\n", false},
		{"different foreign workspace", "serial: future18/SLICE-001 — host cannot isolate workers; ordered handoff required\n", false},
		{"empty foreign reason", "serial: future17/SLICE-001 — \n", false},
		{"fenced foreign example", "```text\nserial: future17/SLICE-001 — host cannot isolate workers; ordered handoff required\n```\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, dir := controllerFixture(t)
			controllerReadyFuture(t, root, "src/future.go")
			writeFile(t, filepath.Join(dir, "evidence.md"), tc.record)
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001")
			if !tc.allowed {
				if code != 3 || !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatalf("nonmatching serial record waived foreign writer: %d\n%s", code, out)
				}
				return
			}
			if code != 0 || !strings.Contains(out, "future17/SLICE-001") || !strings.Contains(out, "serial") {
				t.Fatalf("truthful ordered local handoff unavailable or undisclosed: %d\n%s", code, out)
			}
			if _, err := os.Stat(filepath.Join(root, "work", "future17", "dispatch.json")); !os.IsNotExist(err) {
				t.Fatalf("local fallback launched foreign same-tree writer: %v", err)
			}
		})
	}
}

func TestAutocompleteControllerPreparationDeferralDoesNotLoopContinuation(t *testing.T) {
	root, dir := controllerFixture(t)
	writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — API owner must resolve DEC-017 before planning\n")
	before := controllerSnapshot(t, root)
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"current16"}, &out, &errOut); code != 0 {
		t.Fatalf("deferred continuation: %d %s", code, &errOut)
	}
	for _, want := range []string{"continuation: required\n", "invoke: /rite-build SLICE-001\n", "future17", "frontier-blocked"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("disclosed preparation deferral missing %q:\n%s", want, &out)
		}
	}
	if strings.Contains(out.String(), "invoke: /rite-autocomplete\n") || strings.Contains(out.String(), "frontier-prepare:") {
		t.Fatalf("discharged preparation repeatedly reinvoked controller:\n%s", &out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("deferral observation rewrote evidence or source cursor")
	}
}

func TestAutocompleteControllerQualifiedSerialHandoffDoesNotLoopContinuation(t *testing.T) {
	root, dir := controllerFixture(t)
	controllerReadyFuture(t, root, "src/future.go")
	writeFile(t, filepath.Join(dir, "evidence.md"), "serial: future17/SLICE-001 — host cannot isolate workers; ordered handoff required\n")
	before := controllerSnapshot(t, root)
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"current16"}, &out, &errOut); code != 0 {
		t.Fatalf("ordered handoff continuation: %d %s", code, &errOut)
	}
	if !strings.Contains(out.String(), "invoke: /rite-build SLICE-001\n") || strings.Contains(out.String(), "invoke: /rite-autocomplete\n") || !strings.Contains(out.String(), "future17/SLICE-001") || !strings.Contains(out.String(), "serial") {
		t.Fatalf("foreign serial deferral hidden or recursively actionable:\n%s", &out)
	}
	if strings.Contains(out.String(), "frontier-picks: SLICE-001,future17/SLICE-001") {
		t.Fatalf("deferred foreign pick remained actionable:\n%s", &out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("serial handoff observation mutated workflow")
	}
}

func TestAutocompleteControllerDeferralsCannotWaiveStructuralOrWriterSafety(t *testing.T) {
	for _, kind := range []string{"stale source readiness", "foreign file claim", "invalid canonical roadmap", "symlink canonical roadmap"} {
		t.Run(kind, func(t *testing.T) {
			root, dir := controllerFixture(t)
			writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — API owner must resolve DEC-017\nfrontier-blocked: sequence — owner must reconcile historical map\n")
			switch kind {
			case "stale source readiness":
				writeFile(t, filepath.Join(dir, "plan.md"), "# Plan\n\nUnvetted writer interface change.\n")
			case "foreign file claim":
				if code, out, errOut := runClaim(t, root, "add", "--session", "foreign-owner", "src/current.go"); code != 0 {
					t.Fatalf("fixture claim: %d %s%s", code, out, errOut)
				}
			case "invalid canonical roadmap":
				writeFile(t, filepath.Join(dir, "roadmap.md"), "Sequence root: current16\nDecision: DEC-017\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 2 | duplicate17 |\n")
			case "symlink canonical roadmap":
				writeFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n## DEC-017\nKeep the recorded sequence.\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 3 | future18 |\n")
				poolVet(t, root, "current16")
				target := filepath.Join(t.TempDir(), "outside-roadmap")
				writeFile(t, target, "Sequence root: current16\nDecision: DEC-017\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 3 | future18 |\n")
				if err := os.Symlink(target, filepath.Join(dir, "roadmap.md")); err != nil {
					t.Fatal(err)
				}
			}
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/current.go")
			if code == 0 || !strings.Contains(out, "BLOCKED") {
				t.Fatalf("scheduling deferral waived %s: %d\n%s", kind, code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("unsafe deferred admission changed workflow")
			}
		})
	}
}

func TestAutocompleteControllerFutureOpeningLimitsRetainExistingSuccessor(t *testing.T) {
	for _, policy := range []string{"continue_sequence: false\n", "continue_sequence: true\n"} {
		t.Run(policy, func(t *testing.T) {
			root, _ := controllerFixture(t)
			frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "1")
			writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n"+policy)
			frontierFixture(t, root, "work", "current16", "", "build", "running", "/rite-build SLICE-001", 1, "0")
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"current16"}, &out, &errOut); code != 0 {
				t.Fatalf("existing preparation: %d %s", code, &errOut)
			}
			if !strings.Contains(out.String(), "invoke: /rite-autocomplete\n") || !strings.Contains(out.String(), "future17 location=live") || strings.Contains(out.String(), "frontier-prepare:") {
				t.Fatalf("future opening limit discarded existing native successor or opened another:\n%s", &out)
			}
		})
	}
}

func TestAutocompleteControllerProtectedProofDoesNotHoldIndependentPreparation(t *testing.T) {
	root, _ := controllerFixture(t)
	for _, args := range [][]string{
		{"open", "--phase", "prove", "--wave", "proof16", "--role", "proof-runner"},
		{"start", "--wave", "proof16", "--role", "proof-runner", "--handle", "native-proof16"},
	} {
		if code, out := controllerDispatch(t, root, "current16", args...); code != 0 {
			t.Fatalf("proof fixture: %d %s", code, out)
		}
	}
	before := controllerSnapshot(t, root)
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"current16"}, &out, &errOut); code != 0 {
		t.Fatalf("proof continuation: %d %s", code, &errOut)
	}
	for _, want := range []string{"invoke: /rite-autocomplete\n", "current16 location=live action=wait", "frontier-prepare: /rite-spec 'future17'"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("protected proof hid independent preparation, missing %q:\n%s", want, &out)
		}
	}
	if strings.Contains(out.String(), "\ninvoke: /rite-ship") || !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("planning observation altered protected proof or implied Ship approval")
	}
}

func TestAutocompleteControllerPreparationNeedsCanonicalNativePhaseAndRole(t *testing.T) {
	for _, tc := range []struct{ phase, role string }{
		{"not-a-native-phase", "plan-drafter"},
		{"plan", "code-reviewer"},
		{"plan", "proof-runner"},
		{"plan", "unrelated-helper"},
	} {
		t.Run(tc.phase+"/"+tc.role, func(t *testing.T) {
			root, _ := controllerFixture(t)
			controllerPlanning(t, root, tc.phase, tc.role, true)
			before := controllerSnapshot(t, root)
			if code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001"); code != 3 {
				t.Fatalf("unrelated launched handle counted as native preparation: %d\n%s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("unrelated-handle refusal changed workflow")
			}
		})
	}
}

func TestAutocompleteControllerReturnSchedulingRunsOutsideDispatchLocks(t *testing.T) {
	root, _ := controllerFixture(t)
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
	if code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001"); code != 0 {
		t.Fatalf("writer fixture: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "writer16", "--role", "slice-wright", "--handle", "native-16"); code != 0 {
		t.Fatalf("start fixture: %d %s", code, out)
	}
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
	var out, errOut bytes.Buffer
	checked := false
	var pending chan int
	writer := auditDispatchWriteFunc(func(p []byte) (int, error) {
		if !checked && bytes.Contains(p, []byte("frontier:")) {
			checked = true
			pending = make(chan int, 1)
			go func() {
				var nestedOut, nestedErr bytes.Buffer
				pending <- RunDispatch(root, []string{"current16", "status", "--wave", "writer16"}, &nestedOut, &nestedErr)
			}()
			select {
			case code := <-pending:
				if code != 0 {
					t.Errorf("fresh scheduling observed incomplete return: status=%d", code)
				}
				pending = nil
			case <-time.After(2 * time.Second):
				t.Error("fresh scheduling emitted while feature or repository claims lock remained held")
			}
		}
		return out.Write(p)
	})
	code := RunDispatch(root, []string{"current16", "return", "--wave", "writer16", "--role", "slice-wright", "--handle", "native-16"}, writer, &errOut)
	if pending != nil {
		select {
		case <-pending:
		case <-time.After(2 * time.Second):
			t.Error("nested status did not settle after dispatch returned")
		}
	}
	if code != 0 || !checked {
		t.Fatalf("return omitted post-lock scheduling: code=%d checked=%t\n%s%s", code, checked, &out, &errOut)
	}
}

func TestAutocompleteControllerRetainsUnisolatedWriterAcrossSourcesAndLaterLeases(t *testing.T) {
	for _, kind := range []string{"foreign source", "same source overlap", "retroactive matching lease"} {
		t.Run(kind, func(t *testing.T) {
			root, dir := controllerFixture(t)
			writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — owner must settle DEC-017 before preparation\n")
			if code, out := controllerOpen(t, root, "current16", "legacy-writer", "SLICE-001"); code != 0 {
				t.Fatalf("initial truthful owner dispatch: %d %s", code, out)
			}
			if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "legacy-writer", "--role", "slice-wright", "--handle", "native-unisolated16"); code != 0 {
				t.Fatalf("initial owner start: %d %s", code, out)
			}
			target := "current16"
			if kind != "same source overlap" {
				target = "future17"
				controllerReadyFuture(t, root, "src/future.go")
			}
			if kind == "retroactive matching lease" {
				controllerLease(t, root, 2,
					controllerLeaseRow("", "SLICE-001", "src/current.go"),
					controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
			}
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, target, "unsafe-second", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/future.go")
			if code != 3 || !strings.Contains(out, "BLOCKED") {
				t.Fatalf("retained unknown/same-tree writer was hidden by %s: %d\n%s", kind, code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("unsafe second writer changed claims/history/dispatch")
			}
		})
	}
}

func TestAutocompleteControllerReservationCannotBindChangedWriterPaths(t *testing.T) {
	root, _ := controllerFixture(t)
	controllerReadyFuture(t, root, "src/future.go")
	controllerLease(t, root, 2,
		controllerLeaseRow("", "SLICE-001", "src/old-current.go"),
		controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
	before := controllerSnapshot(t, root)
	if code, out := controllerOpen(t, root, "current16", "changed-contract", "SLICE-001"); code != 3 {
		t.Fatalf("same-ID lease retroactively accepted changed writer allowlist: %d\n%s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("changed-contract admission mutated workflow")
	}
}

func TestAutocompleteControllerOneReservationCannotOpenTwoUnstartedWriterWaves(t *testing.T) {
	root, _ := controllerFixture(t)
	controllerReadyFuture(t, root, "src/future.go")
	controllerLease(t, root, 2,
		controllerLeaseRow("", "SLICE-001", "src/current.go"),
		controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
	if code, out := controllerOpen(t, root, "current16", "first-reserved", "SLICE-001"); code != 0 {
		t.Fatalf("first cohort wave: %d %s", code, out)
	}
	before := controllerSnapshot(t, root)
	if code, out := controllerOpen(t, root, "current16", "duplicate-reserved", "SLICE-001"); code != 3 {
		t.Fatalf("one reservation admitted duplicate undeployed writers: %d\n%s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("duplicate reservation assignment changed workflow")
	}
}

func TestAutocompleteControllerDeferralCannotHideMalformedSuccessorWorkspace(t *testing.T) {
	for _, kind := range []string{"duplicate successor cursor", "symlink successor workspace"} {
		t.Run(kind, func(t *testing.T) {
			root, dir := controllerFixture(t)
			writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — owner decision pending\nfrontier-blocked: sequence — historical normalization pending\n")
			state := cursor("plan", "running", "/rite-vet future17") + "| sequence_parent | current16 |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 1 |\n"
			if kind == "duplicate successor cursor" {
				writeFile(t, filepath.Join(root, "work", "future17", "state.md"), state+"| status | ready |\n")
			} else {
				outside := t.TempDir()
				writeFile(t, filepath.Join(outside, "state.md"), state)
				if err := os.Symlink(outside, filepath.Join(root, "work", "future17")); err != nil {
					t.Fatal(err)
				}
			}
			before := controllerSnapshot(t, root)
			if code, out := controllerOpen(t, root, "current16", "unsafe-normalization-waiver", "SLICE-001"); code != 3 {
				t.Fatalf("scheduling deferral hid %s: %d\n%s", kind, code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("malformed successor admission changed workflow")
			}
		})
	}
}

func TestAutocompleteControllerReusedBatchLabelCannotRetrofitRetainedActor(t *testing.T) {
	root, _ := controllerFixture(t)
	controllerReadyFuture(t, root, "src/future.go")
	controllerLease(t, root, 2,
		controllerLeaseRow("", "SLICE-001", "src/current.go"),
		controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
	if code, out := controllerOpen(t, root, "current16", "original-cohort", "SLICE-001"); code != 0 {
		t.Fatalf("original cohort open: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "original-cohort", "--role", "slice-wright", "--handle", "native-original-cohort"); code != 0 {
		t.Fatalf("original actor start: %d %s", code, out)
	}
	path, err := parallel.LeasePath(filepath.Dir(root), "current16")
	if err != nil {
		t.Fatal(err)
	}
	lease, err := parallel.ReadLease(path)
	if err != nil {
		t.Fatal(err)
	}
	lease.CreatedAt = "2026-11-05T01:00:00Z"
	if err := parallel.WriteLease(path, lease); err != nil {
		t.Fatal(err)
	}
	before := controllerSnapshot(t, root)
	if code, out := controllerOpen(t, root, "future17", "reused-label", "SLICE-001"); code != 3 {
		t.Fatalf("reused batch label reclassified actor into another lease instance: %d\n%s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("reused-cohort admission changed workflow")
	}
}

func TestAutocompleteControllerRetainedActorCannotOutliveItsCountedReservation(t *testing.T) {
	root, _ := controllerFixture(t)
	controllerReadyFuture(t, root, "src/future.go")
	controllerLease(t, root, 2,
		controllerLeaseRow("", "SLICE-001", "src/current.go"),
		controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
	if code, out := controllerOpen(t, root, "current16", "retained-cohort", "SLICE-001"); code != 0 {
		t.Fatalf("cohort open: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "retained-cohort", "--role", "slice-wright", "--handle", "native-retained-cohort"); code != 0 {
		t.Fatalf("cohort actor: %d %s", code, out)
	}
	controllerLease(t, root, 2, controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
	before := controllerSnapshot(t, root)
	if code, out := controllerOpen(t, root, "future17", "lost-actor-slot", "SLICE-001"); code != 3 {
		t.Fatalf("matching cohort label hid an actor no longer counted by the lease: %d\n%s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("lost actor slot admission changed workflow")
	}
}

func TestAutocompleteControllerRecordedReturnRefreshesEvenWhenClaimReleaseIsHeld(t *testing.T) {
	root, dir := controllerFixture(t)
	writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — owner must settle DEC-017\n")
	if code, out := controllerOpen(t, root, "current16", "held-return", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/current.go"); code != 0 {
		t.Fatalf("claimed writer: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "held-return", "--role", "slice-wright", "--handle", "native-held-return"); code != 0 {
		t.Fatalf("writer start: %d %s", code, out)
	}
	writeFile(t, filepath.Join(dir, "evidence.md"), "# Evidence\n\nPreparation constraint resolved; future17 still needs native preparation.\n")
	claimPath := claimsPath(root)
	moved := claimPath + ".saved"
	interrupted := false
	var out, errOut bytes.Buffer
	writer := auditDispatchWriteFunc(func(p []byte) (int, error) {
		if !interrupted && bytes.Contains(p, []byte("returned (wave held-return complete)")) {
			if err := os.Rename(claimPath, moved); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(claimPath, 0o755); err != nil {
				t.Fatal(err)
			}
			interrupted = true
		}
		return out.Write(p)
	})
	code := RunDispatch(root, []string{"current16", "return", "--wave", "held-return", "--role", "slice-wright", "--handle", "native-held-return"}, writer, &errOut)
	if !interrupted {
		t.Fatalf("fixture never reached recorded return: %d %s%s", code, &out, &errOut)
	}
	if err := os.Remove(claimPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(moved, claimPath); err != nil {
		t.Fatal(err)
	}
	if code != 3 || !strings.Contains(errOut.String(), "claim not released") || !strings.Contains(out.String(), "frontier-prepare: /rite-spec 'future17'") {
		t.Fatalf("recorded-but-held return lost failure or fresh scheduling: %d\n%s%s", code, &out, &errOut)
	}
	if claimCode, stdout, stderr := runClaim(t, root, "check", "--session", "foreign-owner", "src/current.go"); claimCode != 3 {
		t.Fatalf("held return released claim exclusion: %d %s%s", claimCode, stdout, stderr)
	}
	if code, out := controllerDispatch(t, root, "current16", "return", "--wave", "held-return", "--role", "slice-wright", "--handle", "native-held-return"); code != 0 || !strings.Contains(out, "frontier-prepare: /rite-spec 'future17'") {
		t.Fatalf("successful claim reconciliation lost fresh scheduling: %d %s", code, out)
	}
}

func TestAutocompleteControllerCompletedSuccessorNeedsNoReopenedPreparation(t *testing.T) {
	for _, location := range []string{"work", "archive"} {
		for _, pendingActor := range []bool{false, true} {
			name := location + "/completed"
			if pendingActor {
				name = location + "/retained-writer"
			}
			t.Run(name, func(t *testing.T) {
				root, source := controllerFixture(t)
				child := controllerReadyFuture(t, root, "src/future.go")
				writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 2\n")
				if code, out := controllerOpen(t, root, "future17", "completed17", "SLICE-001"); code != 0 {
					t.Fatalf("successor writer fixture: %d %s", code, out)
				}
				for _, args := range [][]string{
					{"start", "--wave", "completed17", "--role", "slice-wright", "--handle", "native-completed17"},
					{"return", "--wave", "completed17", "--role", "slice-wright", "--handle", "native-completed17"},
				} {
					if code, out := controllerDispatch(t, root, "future17", args...); code != 0 {
						t.Fatalf("completed successor account: %d %s", code, out)
					}
				}
				if pendingActor {
					if code, out := controllerOpen(t, root, "future17", "retained17", "SLICE-001"); code != 0 {
						t.Fatalf("retained successor writer fixture: %d %s", code, out)
					}
					if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "retained17", "--role", "slice-wright", "--handle", "native-retained17"); code != 0 {
						t.Fatalf("retained successor actor: %d %s", code, out)
					}
				}
				writeFile(t, filepath.Join(child, "state.md"), cursor("done", "shipped", "")+"| schema | 4 |\n| sequence_parent | current16 |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 0 |\n\n## Slices\n- [x] SLICE-001 built — abcdef1\n")
				writeFile(t, filepath.Join(child, "evidence.md"), "# Evidence\n\n## EVID-017: SLICE-001 checkpointed\nCheckpoint: abcdef1\n")
				if location == "archive" {
					if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
						t.Fatal(err)
					}
					archived := filepath.Join(root, "archive", "future17")
					if err := os.Rename(child, archived); err != nil {
						t.Fatal(err)
					}
					child = archived
				}
				writeFile(t, filepath.Join(source, "decisions.md"), "# Decisions\n\n## DEC-017\nRetain the completed successor in the current sequence.\n")
				writeFile(t, filepath.Join(source, "roadmap.md"), "Sequence root: current16\nDecision: DEC-017\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 3 | future18 |\n")
				poolVet(t, root, "current16")
				writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
				before := controllerSnapshot(t, root)
				childBefore := controllerSnapshot(t, child)
				code, out := controllerOpen(t, root, "current16", "writer-after-completed-successor", "SLICE-001")
				if pendingActor {
					if code != 3 || !strings.Contains(out, "retained writer") || !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
						t.Fatalf("terminal child hid retained native writer: %d\n%s", code, out)
					}
				} else if code != 0 || !strings.Contains(out, "wave writer-after-completed-successor open") || strings.Contains(out, "preparation has not actually started") {
					t.Fatalf("completed successor required reopening native Spec: %d\n%s", code, out)
				}
				if !reflect.DeepEqual(childBefore, controllerSnapshot(t, child)) {
					t.Fatal("source admission rewrote completed successor artifacts")
				}
				if location == "archive" {
					if _, err := os.Stat(filepath.Join(root, "work", "future17")); !os.IsNotExist(err) {
						t.Fatalf("source admission resurrected archived successor: %v", err)
					}
				}
			})
		}
	}
}

func TestAutocompleteControllerLocalReconciliationKeepsOwningPhase(t *testing.T) {
	for _, kind := range []string{"returned Build", "local Plan repair"} {
		t.Run(kind, func(t *testing.T) {
			root, source := controllerFixture(t)
			controllerPlanning(t, root, "plan", "plan-drafter", true)
			frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-vet future17", 2, "0")
			invoke := "/rite-build SLICE-001"
			if kind == "returned Build" {
				if code, out := controllerOpen(t, root, "current16", "returned16", "SLICE-001"); code != 0 {
					t.Fatalf("writer fixture: %d %s", code, out)
				}
				for _, sub := range []string{"start", "return"} {
					if code, out := controllerDispatch(t, root, "current16", sub, "--wave", "returned16", "--role", "slice-wright", "--handle", "native-returned16"); code != 0 {
						t.Fatalf("writer %s: %d %s", sub, code, out)
					}
				}
			} else {
				invoke = "/rite-plan repair current16"
				writeFile(t, filepath.Join(source, "plan.md"), "# Plan\n\nNeeds native repair before another writer.\n")
				frontierFixture(t, root, "work", "current16", "", "build", "running", invoke, 1, "2")
			}
			before := controllerSnapshot(t, root)
			var out, errOut bytes.Buffer
			if code := RunCheckContinuation(root, []string{"current16"}, &out, &errOut); code != 0 {
				t.Fatalf("local reconciliation: %d %s", code, &errOut)
			}
			if !strings.Contains(out.String(), "invoke: "+invoke+"\n") || strings.Contains(out.String(), "invoke: /rite-autocomplete\n") || !strings.Contains(out.String(), "future17 location=live action=wait") {
				t.Fatalf("local-only reconciliation recursively invoked controller:\n%s", &out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("continuation changed local or pending planning accounts")
			}
		})
	}
}

func TestAutocompleteControllerAdmitsEligibleSliceDespiteSiblingNotices(t *testing.T) {
	for _, declaration := range []string{"Workspace dependencies: current16/SLICE-001", ""} {
		t.Run(declaration, func(t *testing.T) {
			root, _ := controllerFixture(t)
			child := controllerReadyFuture(t, root, "src/independent.go")
			writeFile(t, filepath.Join(child, "tasks.md"), "# Tasks\n"+
				poolSlice("SLICE-001", "src/independent.go", "Workspace dependencies: none")+
				poolSlice("SLICE-002", "src/dependent.go", declaration))
			poolVet(t, root, "future17")
			controllerLease(t, root, 2,
				controllerLeaseRow("", "SLICE-001", "src/current.go"),
				controllerLeaseRow("future17", "SLICE-001", "src/independent.go"))
			before := controllerSnapshot(t, root)
			if code, out := controllerOpen(t, root, "future17", "blocked-sibling", "SLICE-002"); code != 3 || !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatalf("withheld sibling was admitted or wrote workflow: %d %s", code, out)
			}
			if code, out := controllerOpen(t, root, "current16", "current-independent", "SLICE-001"); code != 0 {
				t.Fatalf("successor's candidate-local notice blocked current writer: %d %s", code, out)
			}
			if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "current-independent", "--role", "slice-wright", "--handle", "native-current-independent"); code != 0 {
				t.Fatalf("current writer start: %d %s", code, out)
			}
			code, out := controllerOpen(t, root, "future17", "future-independent", "SLICE-001")
			if code != 0 || !strings.Contains(out, "wave future-independent open") || !strings.Contains(out, "future17/SLICE-002") {
				t.Fatalf("eligible foreign slice blocked or sibling diagnostic lost: %d\n%s", code, out)
			}
		})
	}
}

func TestAutocompleteControllerMatchesNormalizedReservationWithoutRewritingPlan(t *testing.T) {
	root, _ := controllerFixture(t)
	child := controllerReadyFuture(t, root, "./src/future.go")
	controllerLease(t, root, 2,
		controllerLeaseRow("", "SLICE-001", "src/current.go"),
		controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
	tasks := readTestFile(t, filepath.Join(child, "tasks.md"))
	if code, out := controllerOpen(t, root, "current16", "normalized-current", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/"); code != 0 {
		t.Fatalf("current cohort writer: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "normalized-current", "--role", "slice-wright", "--handle", "native-normalized-current"); code != 0 {
		t.Fatalf("current actor: %d %s", code, out)
	}
	if code, out := controllerOpen(t, root, "future17", "normalized-future", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/"); code != 0 {
		t.Fatalf("normalized lease or legitimate same-session cohort claim refused: %d\n%s", code, out)
	}
	if after := readTestFile(t, filepath.Join(child, "tasks.md")); after != tasks {
		t.Fatal("reservation matching rewrote the approved plan")
	}
}

func TestAutocompleteControllerRetainedWriterPathsCannotBeMovedWithLeaseAndPlan(t *testing.T) {
	for _, releaseClaim := range []bool{false, true} {
		name := "live-claim"
		if releaseClaim {
			name = "released-claim-still-pending-actor"
		}
		t.Run(name, func(t *testing.T) {
			root, source := controllerFixture(t)
			child := controllerReadyFuture(t, root, "src/future.go")
			controllerLease(t, root, 2,
				controllerLeaseRow("", "SLICE-001", "src/current.go"),
				controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
			if code, out := controllerOpen(t, root, "current16", "frozen-actor", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/"); code != 0 {
				t.Fatalf("original actor wave: %d %s", code, out)
			}
			if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "frozen-actor", "--role", "slice-wright", "--handle", "native-frozen-actor"); code != 0 {
				t.Fatalf("original actor start: %d %s", code, out)
			}
			if releaseClaim {
				account, _, err := loadDispatch(root, "current16")
				if err != nil {
					t.Fatal(err)
				}
				if code, out, errOut := runClaim(t, root, "release", "--session", "native-controller", "--id", account.Waves["frozen-actor"].Claim); code != 0 {
					t.Fatalf("claim release fixture: %d %s%s", code, out, errOut)
				}
			}
			writeFile(t, filepath.Join(source, "tasks.md"), "# Tasks\n"+poolSlice("SLICE-001", "src/spare.go", "Workspace dependencies: none"))
			poolVet(t, root, "current16")
			writeFile(t, filepath.Join(child, "tasks.md"), "# Tasks\n"+poolSlice("SLICE-001", "src/current.go", "Workspace dependencies: none"))
			poolVet(t, root, "future17")
			controllerLease(t, root, 2,
				controllerLeaseRow("", "SLICE-001", "src/spare.go"),
				controllerLeaseRow("future17", "SLICE-001", "src/current.go"))
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "future17", "retrofitted-path", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/")
			if code != 3 || !strings.Contains(out, "retained") || !strings.Contains(out, "path") {
				t.Fatalf("mutable lease/tasks moved a pending actor's frozen write authority: %d\n%s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("retrofitted-path refusal changed claims/history/dispatch")
			}
		})
	}
}

func TestAutocompleteControllerRetainedAccountNeedsFrozenPathsAndExactClaimSession(t *testing.T) {
	for _, corrupt := range []string{"admitted_paths", "claim_session"} {
		t.Run(corrupt, func(t *testing.T) {
			root, _ := controllerFixture(t)
			controllerReadyFuture(t, root, "src/future.go")
			controllerLease(t, root, 2,
				controllerLeaseRow("", "SLICE-001", "src/current.go"),
				controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
			if code, out := controllerOpen(t, root, "current16", "retained-account", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/"); code != 0 {
				t.Fatalf("retained account fixture: %d %s", code, out)
			}
			if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "retained-account", "--role", "slice-wright", "--handle", "native-retained-account"); code != 0 {
				t.Fatalf("retained actor fixture: %d %s", code, out)
			}
			path := filepath.Join(root, "work", "current16", "dispatch.json")
			var account struct {
				Waves map[string]map[string]any `json:"waves"`
			}
			if err := json.Unmarshal([]byte(readTestFile(t, path)), &account); err != nil {
				t.Fatal(err)
			}
			if corrupt == "admitted_paths" {
				delete(account.Waves["retained-account"], corrupt)
			} else {
				account.Waves["retained-account"][corrupt] = "wrong-ledger-session"
			}
			data, err := json.Marshal(account)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, string(data))
			before := controllerSnapshot(t, root)
			if code, out := controllerOpen(t, root, "future17", "untrusted-retained-account", "SLICE-001", "--claim-session", "native-controller", "--claim", "src/"); code != 3 {
				t.Fatalf("retained account %s corruption was ignored: %d\n%s", corrupt, code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("untrusted-account refusal changed workflow")
			}
		})
	}
}

func TestAutocompleteControllerRollingRefillUsesOnlyPendingWriterLane(t *testing.T) {
	root, source := controllerFixture(t)
	writeFile(t, filepath.Join(source, "tasks.md"), "# Tasks\n"+
		poolSlice("SLICE-001", "src/first.go", "Workspace dependencies: none")+
		poolSlice("SLICE-002", "src/second.go", "Workspace dependencies: none")+
		poolSlice("SLICE-003", "src/third.go", "Workspace dependencies: none"))
	poolVet(t, root, "current16")
	controllerReadyFuture(t, root, "src/future.go")
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 3\n")
	controllerLease(t, root, 3,
		controllerLeaseRow("", "SLICE-001", "src/first.go"),
		controllerLeaseRow("", "SLICE-002", "src/second.go"),
		controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
	if code, out := controllerDispatch(t, root, "current16", "open", "--phase", "build", "--wave", "rolling",
		"--role", "slice-wright.SLICE-001", "--role", "slice-wright.SLICE-002",
		"--slice", "SLICE-001", "--slice", "SLICE-002", "--contract", wrightContractPath(t)); code != 0 {
		t.Fatalf("multi-role cohort fixture: %d %s", code, out)
	}
	for _, id := range []string{"SLICE-001", "SLICE-002"} {
		if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "rolling", "--role", "slice-wright."+id, "--handle", "native-rolling-"+id); code != 0 {
			t.Fatalf("cohort role start: %d %s", code, out)
		}
	}
	if code, out := controllerDispatch(t, root, "current16", "return", "--wave", "rolling", "--role", "slice-wright.SLICE-001", "--handle", "native-rolling-SLICE-001"); code != 0 {
		t.Fatalf("first sibling return: %d %s", code, out)
	}
	writeFile(t, filepath.Join(source, "state.md"), cursor("build", "running", "/rite-build SLICE-003")+"| sequence_position | 1 |\n| sequence_workspaces_remaining | 2 |\n\n## Slices\n- [x] SLICE-001 built — abcdef2\n")
	writeFile(t, filepath.Join(source, "evidence.md"), "# Evidence\n\n## EVID-001: SLICE-001 checkpointed\nCheckpoint: abcdef2\n")
	first := controllerLeaseRow("", "SLICE-001", "src/first.go")
	first.WrightStatus, first.TransferCommit = parallel.WrightIntegrated, "abcdef2"
	controllerLease(t, root, 3, first,
		controllerLeaseRow("", "SLICE-002", "src/second.go"),
		controllerLeaseRow("future17", "SLICE-001", "src/future.go"),
		controllerLeaseRow("", "SLICE-003", "src/third.go"))
	if code, out := controllerOpen(t, root, "current16", "rolling-refill", "SLICE-003"); code != 0 {
		t.Fatalf("returned/integrated sibling blocked disjoint rolling refill: %d\n%s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "status", "--wave", "rolling"); code != 3 || !strings.Contains(out, "slice-wright.SLICE-001=returned") || !strings.Contains(out, "slice-wright.SLICE-002=started") {
		t.Fatalf("rolling refill altered pending sibling identity: %d %s", code, out)
	}
}

func TestAutocompleteControllerLowerCapBlocksNewLaunchWithoutCancellingReservations(t *testing.T) {
	for _, capN := range []string{"1", "2"} {
		for _, target := range []struct{ slug, id string }{{"current16", "SLICE-001"}, {"current16", "SLICE-002"}, {"future17", "SLICE-001"}} {
			t.Run(capN+"/"+target.slug+"/"+target.id, func(t *testing.T) {
				root, source := controllerFixture(t)
				writeFile(t, filepath.Join(source, "tasks.md"), "# Tasks\n"+
					poolSlice("SLICE-001", "src/current.go", "Workspace dependencies: none")+
					poolSlice("SLICE-002", "src/current-two.go", "Workspace dependencies: none"))
				poolVet(t, root, "current16")
				controllerReadyFuture(t, root, "src/future.go")
				controllerLease(t, root, 3,
					controllerLeaseRow("", "SLICE-001", "src/current.go"),
					controllerLeaseRow("", "SLICE-002", "src/current-two.go"),
					controllerLeaseRow("future17", "SLICE-001", "src/future.go"))
				writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: "+capN+"\n")
				before := controllerSnapshot(t, root)
				if code, out := controllerOpen(t, root, target.slug, "over-cap", target.id); code != 3 || !strings.Contains(out, "cap") {
					t.Fatalf("three occupied reservations bypassed lower cap %s: %d\n%s", capN, code, out)
				}
				if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
					t.Fatal("lower-cap refusal cancelled/resized reservations or wrote dispatch")
				}
			})
		}
	}
}

func TestAutocompleteControllerSequenceCursorNormalizationPreemptsLocalBuild(t *testing.T) {
	root, _ := controllerFixture(t)
	frontierFixture(t, root, "work", "current16", "", "build", "running", "/rite-build SLICE-001", 2, "2")
	before := controllerSnapshot(t, root)
	var out, errOut bytes.Buffer
	if code := RunCheckContinuation(root, []string{"current16"}, &out, &errOut); code != 0 {
		t.Fatalf("sequence normalization: %d %s", code, &errOut)
	}
	if !strings.Contains(out.String(), "invoke: /rite-autocomplete\n") || !strings.Contains(out.String(), "frontier-review-sequence:") || strings.Contains(out.String(), "invoke: /rite-build") {
		t.Fatalf("invalid sequence cursor used local Build instead of authority normalization:\n%s", &out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("normalization observation rewrote source authority")
	}
}

func TestAutocompleteControllerSingleCapRetainsExistingLeaseAuthority(t *testing.T) {
	root, _ := controllerFixture(t)
	writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 1\n")
	controllerLease(t, root, 1, controllerLeaseRow("", "SLICE-001", "src/current.go"))
	if code, out := controllerOpen(t, root, "current16", "single-reserved", "SLICE-001"); code != 0 {
		t.Fatalf("single reserved writer invented missing-successor obligation: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "single-reserved", "--role", "slice-wright", "--handle", "native-single-reserved"); code != 0 {
		t.Fatalf("single reserved actor: %d %s", code, out)
	}
	controllerReadyFuture(t, root, "src/future.go")
	before := controllerSnapshot(t, root)
	if code, out := controllerOpen(t, root, "future17", "unreserved-second", "SLICE-001"); code != 3 {
		t.Fatalf("cap one bypassed retained lease/actor authority: %d\n%s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("cap-one refusal changed retained reservations or workflow")
	}
}

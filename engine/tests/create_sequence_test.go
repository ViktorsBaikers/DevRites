package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/testutil"
)

func createSequenceCLIFixture(t *testing.T, buildStarted bool) (string, map[string]string) {
	t.Helper()
	t.Setenv("DEVRITES_WORKSPACE", "")
	t.Setenv("DEVRITES_HOST_CAPACITY", "4")
	root := filepath.Join(t.TempDir(), ".devrites")
	phase, next := "spec", "/rite-spec a"
	if buildStarted {
		phase, next = "build", "/rite-build SLICE-002"
	}
	dir := frontierCLIWorkspace(t, root, "a", fmt.Sprintf("| phase | %s |\n| status | running |\n| sequence_position | 1 |\n| sequence_workspaces_remaining | 3 |\n| next_action | %s |\n", phase, next), false)
	testutil.WriteFile(t, filepath.Join(dir, "brief.md"), "# Brief\n\nBuild the approved account settings screen; SLICE-002 remains unfinished.\n")
	testutil.WriteFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n## DEC-001 Approved independent scope\n\nThe owner approved account settings in a, CSV export in b, audit search in c, and keyboard navigation in d. Prepare these scopes without adding product behavior beyond their briefs.\n")
	testutil.WriteFile(t, filepath.Join(dir, "roadmap.md"), "Sequence root: a\nDecision: DEC-001\n\n| Position | Workspace |\n| --- | --- |\n| 1 | a |\n| 2 | b |\n| 3 | c |\n| 4 | d |\n")
	if buildStarted {
		testutil.WriteFile(t, filepath.Join(dir, "dispatch.json"), `{"waves":{"build-one":{"phase":"build","state":"complete","slices":["SLICE-001"],"roles":{"slice-wright":{"handle":"engine-fixture-a-writer","started":"2026-01-01T00:00:00Z","returned":true,"returned_at":"2026-01-01T00:01:00Z"}}}}}`)
	}
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "a\n")
	testutil.WriteFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
	briefs := map[string]string{}
	for slug, scope := range map[string]string{
		"b": "Export the current filtered records to CSV with existing column names. Preserve authorization and exclude hidden fields.",
		"c": "Search the existing audit log by actor and date. Preserve existing pagination and access rules.",
		"d": "Add keyboard navigation to the current settings screen, preserving focus order and reduced-motion preferences.",
	} {
		briefs[slug] = filepath.Join(t.TempDir(), slug+".brief.md")
		testutil.WriteFile(t, briefs[slug], "# Approved scope: "+slug+"\n\nSource: a/decisions.md DEC-001.\n\n"+scope+"\n")
	}
	return root, briefs
}

func createSequenceCLICommand(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, errOut, code := runDevrites(t, root, args...)
	if code != 0 || errOut != "" {
		t.Fatalf("%v: code=%d stdout=%q stderr=%q", args, code, out, errOut)
	}
	return out
}

func createSequenceCLIRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func createSequenceCLIAssertCreated(t *testing.T, root, target, parent, brief string, position, remaining int) {
	t.Helper()
	out := createSequenceCLICommand(t, root, "state", "create-sequence", "a", target, "--brief", brief)
	want := fmt.Sprintf("sequence-workspace: created target=%s position=%d parent=%s\n", target, position, parent)
	if out != want {
		t.Fatalf("creation result=%q, want %q", out, want)
	}
	dir := filepath.Join(root, "work", target)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != "brief.md" || entries[1].Name() != "state.md" || entries[0].IsDir() || entries[1].IsDir() {
		t.Fatalf("pending bootstrap must publish only brief.md and state.md: %v", entries)
	}
	if got := createSequenceCLIRead(t, filepath.Join(dir, "brief.md")); got != createSequenceCLIRead(t, brief) {
		t.Fatalf("brief changed during publication: %q", got)
	}
	cursor := createSequenceCLIRead(t, filepath.Join(dir, "state.md"))
	for _, row := range []string{
		"| schema | 4 |", "| phase | spec |", "| status | running |",
		"| next_action | /rite-spec " + target + " |",
		"| sequence_parent | " + parent + " |",
		fmt.Sprintf("| sequence_position | %d |", position),
		fmt.Sprintf("| sequence_workspaces_remaining | %d |", remaining),
	} {
		if !strings.Contains(cursor, row) {
			t.Fatalf("pending cursor missing %q: %s", row, cursor)
		}
	}
}

func createSequenceCLIRefused(t *testing.T, root, target, brief string) {
	t.Helper()
	before := autocompleteCLISnapshot(t, root)
	out, errOut, code := runDevrites(t, root, "state", "create-sequence", "a", target, "--brief", brief)
	if code != 3 || out != "" || !strings.Contains(errOut, "refused without publishing target") || !strings.Contains(errOut, "refresh frontier") {
		t.Fatalf("creation refusal: code=%d stdout=%q stderr=%q", code, out, errOut)
	}
	if !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatal("refused creation changed workflow bytes")
	}
	if _, err := os.Lstat(filepath.Join(root, "work", target)); !os.IsNotExist(err) {
		t.Fatalf("refused target became discoverable: %v", err)
	}
}

func TestCreateSequenceCLIRollingPreparationAndSettledParkRefill(t *testing.T) {
	root, briefs := createSequenceCLIFixture(t, true)
	aState := createSequenceCLIRead(t, filepath.Join(root, "work", "a", "state.md"))
	aDispatch := createSequenceCLIRead(t, filepath.Join(root, "work", "a", "dispatch.json"))
	entries, err := os.ReadDir(filepath.Join(root, "work"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "a" {
		t.Fatalf("fixture must start with only A: %v, %v", entries, err)
	}
	before := autocompleteCLISnapshot(t, root)
	frontier := createSequenceCLICommand(t, root, "frontier", "a")
	if !strings.Contains(frontier, "frontier-create: controller=a target=b parent=a\n") || strings.Contains(frontier, "frontier-create: controller=a target=c") {
		t.Fatalf("first bootstrap must select B, not skip its missing identity: %s", frontier)
	}
	continuation := createSequenceCLICommand(t, root, "check", "continuation", "a")
	if !strings.Contains(continuation, "\ncreate: controller=a target=b parent=a\n") || strings.Contains(continuation, "\nlaunch: /rite-spec b\n") {
		t.Fatalf("continuation lost creation route: %s", continuation)
	}
	if !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatal("frontier observation created or changed workspace data")
	}
	createSequenceCLIAssertCreated(t, root, "b", "a", briefs["b"], 2, 2)
	frontier = createSequenceCLICommand(t, root, "frontier", "a")
	if !strings.Contains(frontier, "frontier-create: controller=a target=c parent=b\n") {
		t.Fatalf("B publication did not refill creation route for C: %s", frontier)
	}
	createSequenceCLIAssertCreated(t, root, "c", "b", briefs["c"], 3, 1)
	createSequenceCLIRefused(t, root, "d", briefs["d"])
	for _, slug := range []string{"b", "c"} {
		out := createSequenceCLICommand(t, root, "next", slug)
		for _, want := range []string{"source: " + slug + "\n", "current: spec\n", "current-status: incomplete\n", "/rite-spec"} {
			if !strings.Contains(out, want) {
				t.Fatalf("pending Spec next %s missing %q: %s", slug, want, out)
			}
		}
	}
	// Every invocation is a fresh binary process; reuse must survive restart with a full pool.
	before = autocompleteCLISnapshot(t, root)
	if out := createSequenceCLICommand(t, root, "state", "create-sequence", "a", "b", "--brief", briefs["b"]); out != "sequence-workspace: reused target=b position=2 parent=a\n" {
		t.Fatalf("restart did not reuse B: %s", out)
	}
	if !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatal("restart reuse rewrote cursor, brief, or budget")
	}
	createSequenceCLICommand(t, root, "state", "set", "c", "phase=clarify", "next_action=/rite-clarify c")
	before = autocompleteCLISnapshot(t, root)
	if out := createSequenceCLICommand(t, root, "state", "create-sequence", "a", "c", "--brief", briefs["c"]); out != "sequence-workspace: reused target=c position=3 parent=b\n" {
		t.Fatalf("restart did not reuse advanced C: %s", out)
	}
	if !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatal("reuse reset advanced Clarify cursor or recharged its parent")
	}

	// Evidence-scout is the native read-only role shared by Spec and Clarify.
	for _, slug := range []string{"b", "c"} {
		phase := "spec"
		if slug == "c" {
			phase = "clarify"
		}
		createSequenceCLICommand(t, root, "dispatch", slug, "open", "--phase", phase, "--wave", "prepare-"+slug, "--role", "evidence-scout")
		createSequenceCLICommand(t, root, "dispatch", slug, "check-spawn", "--wave", "prepare-"+slug, "--role", "evidence-scout")
		createSequenceCLICommand(t, root, "dispatch", slug, "start", "--wave", "prepare-"+slug, "--role", "evidence-scout", "--handle", "engine-evidence-"+slug)
	}
	hold := "frontier-blocked: b — owner must confirm CSV field contract; retry after DEC-002 records the answer\n"
	out, errOut, code := runDevritesIO(t, root, hold, nil, "state", "append", "a", "evidence.md", "--heading", "CSV contract hold", "--body-file", "-")
	if code != 0 || errOut != "" {
		t.Fatalf("record hold: code=%d stdout=%q stderr=%q", code, out, errOut)
	}
	before = autocompleteCLISnapshot(t, root)
	out, errOut, code = runDevrites(t, root, "dispatch", "b", "park")
	if code != 3 || !strings.Contains(errOut, "settle every role") || !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatalf("park discarded retained actor: code=%d stdout=%q stderr=%q", code, out, errOut)
	}
	createSequenceCLIRefused(t, root, "d", briefs["d"])
	createSequenceCLICommand(t, root, "dispatch", "b", "return", "--wave", "prepare-b", "--role", "evidence-scout", "--handle", "engine-evidence-b")
	createSequenceCLICommand(t, root, "dispatch", "b", "park")
	frontier = createSequenceCLICommand(t, root, "frontier", "a")
	if !strings.Contains(frontier, "frontier-create: controller=a target=d parent=c\n") || !strings.Contains(frontier, "frontier-blocked: b") {
		t.Fatalf("settled durable park did not refill D independently: %s", frontier)
	}
	createSequenceCLIAssertCreated(t, root, "d", "c", briefs["d"], 4, 0)
	createSequenceCLICommand(t, root, "dispatch", "c", "return", "--wave", "prepare-c", "--role", "evidence-scout", "--handle", "engine-evidence-c")

	// Recorded handles prove engine admission/accounting, not live native-host concurrency.
	handles := map[string]bool{}
	var lastStart, firstReturn time.Time
	for _, slug := range []string{"b", "c"} {
		var account struct {
			Waves map[string]struct {
				State          string `json:"state"`
				PoolSource     string `json:"pool_source"`
				PreparationFor string `json:"preparation_for"`
				Roles          map[string]struct {
					Handle     string    `json:"handle"`
					Started    time.Time `json:"started"`
					Returned   bool      `json:"returned"`
					ReturnedAt time.Time `json:"returned_at"`
				} `json:"roles"`
			} `json:"waves"`
		}
		if err := json.Unmarshal([]byte(createSequenceCLIRead(t, filepath.Join(root, "work", slug, "dispatch.json"))), &account); err != nil {
			t.Fatal(err)
		}
		wave := account.Waves["prepare-"+slug]
		role := wave.Roles["evidence-scout"]
		if wave.State != "complete" || wave.PoolSource != slug || wave.PreparationFor != "" || role.Handle != "engine-evidence-"+slug || !role.Returned || role.Started.IsZero() || role.ReturnedAt.Before(role.Started) {
			t.Fatalf("ordinary preparation did not retain source-owned return for %s: %+v", slug, wave)
		}
		handles[role.Handle] = true
		if role.Started.After(lastStart) {
			lastStart = role.Started
		}
		if firstReturn.IsZero() || role.ReturnedAt.Before(firstReturn) {
			firstReturn = role.ReturnedAt
		}
	}
	if len(handles) != 2 || lastStart.After(firstReturn) {
		t.Fatalf("need two overlapping recorded preparation accounts: handles=%v lastStart=%v firstReturn=%v", handles, lastStart, firstReturn)
	}
	if createSequenceCLIRead(t, filepath.Join(root, "ACTIVE")) != "a\n" || createSequenceCLIRead(t, filepath.Join(root, "work", "a", "state.md")) != aState || createSequenceCLIRead(t, filepath.Join(root, "work", "a", "dispatch.json")) != aDispatch {
		t.Fatal("preparation moved ACTIVE, recharged A, or completed its unfinished Build")
	}
}

func TestCreateSequenceCLIPreBuildControllerConsumesSlot(t *testing.T) {
	root, briefs := createSequenceCLIFixture(t, false)
	createSequenceCLIAssertCreated(t, root, "b", "a", briefs["b"], 2, 2)
	frontier := createSequenceCLICommand(t, root, "frontier", "a")
	if strings.Contains(frontier, "frontier-create: controller=a target=c") {
		t.Fatalf("pre-Build A escaped preparation occupancy: %s", frontier)
	}
	createSequenceCLIRefused(t, root, "c", briefs["c"])
}

func TestCreateSequenceCLIOptOutDoesNotOpenWorkspace(t *testing.T) {
	root, briefs := createSequenceCLIFixture(t, true)
	testutil.WriteFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\ncontinue_sequence: false\n")
	frontier := createSequenceCLICommand(t, root, "frontier", "a")
	if strings.Contains(frontier, "frontier-create:") {
		t.Fatalf("opt-out still schedules creation: %s", frontier)
	}
	createSequenceCLIRefused(t, root, "b", briefs["b"])
}

func TestCreateSequenceCLIHelpAndStrictRootErrors(t *testing.T) {
	unsafeRoot := filepath.Join(t.TempDir(), ".devrites")
	testutil.WriteFile(t, unsafeRoot, "not a directory\n")
	usage := "usage: devrites-engine state create-sequence <controller> <target> --brief <file>\n"
	if out := createSequenceCLICommand(t, unsafeRoot, "state", "create-sequence", "--help"); out != usage {
		t.Fatalf("help=%q, want %q", out, usage)
	}
	out, errOut, code := runDevrites(t, unsafeRoot, "state", "create-sequence", "a", "b", "--brief", "absent.md")
	if code != 3 || out != "" || !strings.Contains(errOut, "devrites: root selection:") || !strings.Contains(errOut, "DEVRITES_ROOT") {
		t.Fatalf("unsafe mutation root was not refused before command execution: code=%d stdout=%q stderr=%q", code, out, errOut)
	}
	if createSequenceCLIRead(t, unsafeRoot) != "not a directory\n" {
		t.Fatal("strict-root refusal changed selected path")
	}
	root, _ := createSequenceCLIFixture(t, true)
	before := autocompleteCLISnapshot(t, root)
	out, errOut, code = runDevrites(t, root, "state", "create-sequence", "a", "b")
	if code != 2 || out != "" || errOut != usage || !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatalf("missing brief usage changed workflow: code=%d stdout=%q stderr=%q", code, out, errOut)
	}
}

func TestCreateSequenceCLIReadyWritersKeepProducerDependencies(t *testing.T) {
	root, briefs := createSequenceCLIFixture(t, true)
	for i, slug := range []string{"b", "c"} {
		parent := "a"
		if i > 0 {
			parent = "b"
		}
		createSequenceCLIAssertCreated(t, root, slug, parent, briefs[slug], i+2, 2-i)
		// Simulate source-owned native planning outputs without replacing bootstrap identity.
		prepared := frontierCLIWorkspace(t, filepath.Join(t.TempDir(), ".devrites"), slug, "| phase | vet |\n| status | ready |\n", true)
		entries, err := os.ReadDir(prepared)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.Name() == "state.md" || entry.Name() == "brief.md" || entry.Name() == "eng-review.md" {
				continue
			}
			testutil.WriteFile(t, filepath.Join(root, "work", slug, entry.Name()), createSequenceCLIRead(t, filepath.Join(prepared, entry.Name())))
		}
		testutil.WriteFile(t, filepath.Join(root, "work", slug, "plan.md"), "# Plan\n\nConsumed planning inputs: none\n\nImplement the approved independent scope in its declared paths.\n")
		createSequenceCLICommand(t, root, "state", "set", slug, "phase=vet", "status=ready", "next_action=/rite-build SLICE-001")
		autocompleteCLIVet(t, root, slug)
		review := filepath.Join(root, "work", slug, "eng-review.md")
		testutil.WriteFile(t, review, createSequenceCLIRead(t, review)+"\nPredecessor readiness: none\n")
	}
	frontier := createSequenceCLICommand(t, root, "frontier", "a")
	if !strings.Contains(frontier, "frontier-picks: b/SLICE-001,c/SLICE-001") {
		t.Fatalf("created independent sources did not reach writer selection: %s", frontier)
	}
	tasks := filepath.Join(root, "work", "c", "tasks.md")
	testutil.WriteFile(t, tasks, strings.Replace(createSequenceCLIRead(t, tasks), "Workspace dependencies: none", "Workspace dependencies: b/SLICE-001", 1))
	autocompleteCLIVet(t, root, "c")
	review := filepath.Join(root, "work", "c", "eng-review.md")
	testutil.WriteFile(t, review, createSequenceCLIRead(t, review)+"\nPredecessor readiness: none\n")
	frontier = createSequenceCLICommand(t, root, "frontier", "a")
	if !strings.Contains(frontier, "frontier-picks: b/SLICE-001\n") || strings.Contains(frontier, "frontier-picks: b/SLICE-001,c/SLICE-001") {
		t.Fatalf("consumer writer escaped unfinished producer: %s", frontier)
	}
}

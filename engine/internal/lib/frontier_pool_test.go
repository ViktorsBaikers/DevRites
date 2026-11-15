package lib

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func canonicalPreparationFixture(t *testing.T, built bool) (string, string) {
	t.Helper()
	root, dir := controllerFixture(t)
	phase, next := "plan", "/rite-plan current16"
	if built {
		phase, next = "prove", "/rite-prove current16"
	}
	frontierFixture(t, root, "work", "current16", "", phase, "running", next, 1, "3")
	writeFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n## DEC-001 Approved scope\n\nPrepare the recorded independent workspaces within their approved briefs.\n")
	writeFile(t, filepath.Join(dir, "roadmap.md"), "Sequence root: current16\nDecision: DEC-001\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 3 | future18 |\n| 4 | future19 |\n")
	if built {
		stamp := "2026-01-01T00:00:00Z"
		canonicalPoolAccount(t, root, "current16", &dispatchFile{Waves: map[string]*dispatchWave{"built": {Phase: "build", State: "complete", Roles: map[string]*dispatchRole{"slice-wright": {Handle: "native-built", StartedAt: stamp, Returned: true, ReturnedAt: stamp}}}}})
	}
	return root, dir
}

func canonicalPoolAccount(t *testing.T, root, slug string, account *dispatchFile) {
	t.Helper()
	raw, err := json.Marshal(account)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dispatchPath(root, slug), string(raw))
}

func canonicalPool(t *testing.T, root, source string) *preparationPool {
	t.Helper()
	p, err := evaluatePreparationPool(root, source, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !p.enabled {
		t.Fatal("canonical pool disabled")
	}
	return p
}

func canonicalPoolChild(t *testing.T, root, slug, parent string, position int, remaining string) string {
	t.Helper()
	return frontierFixture(t, root, "work", slug, parent, "spec", "running", "/rite-spec "+slug, position, remaining)
}

func TestCanonicalPreparationPoolOrderedBootstrapAndReadyBuffer(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	before := controllerSnapshot(t, root)
	p := canonicalPool(t, root, "current16")
	if p.occupied != 0 || !p.entry("future17").selected || p.entry("future18").selected {
		t.Fatalf("bootstrap selection: %+v", p.entries)
	}
	if p.entry("future17").parent != "current16" || p.entry("future17").remaining != 2 {
		t.Fatalf("identity/budget: %+v", p.entry("future17"))
	}
	text := frontierText(t, root, "current16")
	if !strings.Contains(text, "frontier-create: controller=current16 target=future17 parent=current16") || !strings.Contains(text, "frontier-prepare: /rite-spec 'future17'") {
		t.Fatal(text)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("observation mutated workflow")
	}
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	p = canonicalPool(t, root, "current16")
	if !p.entry("future18").selected || p.occupied != 1 {
		t.Fatalf("refill: %+v", p.entries)
	}
	canonicalPoolChild(t, root, "future18", "future17", 3, "1")
	frontierFixture(t, root, "work", "future17", "current16", "vet", "ready", "/rite-build future17", 2, "2")
	p = canonicalPool(t, root, "current16")
	if p.occupied != 2 || p.entry("future19").selected {
		t.Fatalf("READY buffer escaped cap: %+v", p.entries)
	}
}

func TestCanonicalPreparationPoolPreBuildSourceOccupiesSlot(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, false)
	p := canonicalPool(t, root, "current16")
	if p.occupied != 1 || !p.entry("future17").selected {
		t.Fatalf("source occupancy: %+v", p.entries)
	}
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	p = canonicalPool(t, root, "future17")
	if p.entry("future18").selected || p.occupied != 2 {
		t.Fatalf("overfilled pre-Build pool: %+v", p.entries)
	}
}

func TestCanonicalPreparationPoolReadOnlyBuildKeepsBuffer(t *testing.T) {
	for _, withWriter := range []bool{false, true} {
		t.Run(map[bool]string{false: "read-only", true: "writer-not-started"}[withWriter], func(t *testing.T) {
			root, _ := canonicalPreparationFixture(t, true)
			canonicalPoolChild(t, root, "future17", "current16", 2, "2")
			canonicalPoolChild(t, root, "future18", "future17", 3, "1")
			roles := map[string]*dispatchRole{"evidence-scout": {Handle: "native-read-only", StartedAt: "2026-01-01T00:00:00Z"}}
			waveState := "sealed"
			if withWriter {
				roles["slice-wright"] = &dispatchRole{}
				waveState = "open"
			}
			canonicalPoolAccount(t, root, "future17", &dispatchFile{Waves: map[string]*dispatchWave{"build": {Phase: "build", State: waveState, Roles: roles}}})
			p := canonicalPool(t, root, "current16")
			if p.entry("future17").buildStarted || !p.entry("future17").occupied || p.entry("future19").selected {
				t.Fatalf("read-only start released preparation buffer: %+v", p.entries)
			}
		})
	}
}

func TestCanonicalPreparationPoolOpenIsNotStartAndBuildStartRefills(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	canonicalPoolChild(t, root, "future18", "future17", 3, "1")
	if code, out := preparationOpen(t, root, "future17", "spec", "evidence-scout", "native"); code != 0 {
		t.Fatalf("open: %d %s", code, out)
	}
	p := canonicalPool(t, root, "current16")
	e := p.entry("future17")
	if !e.occupied || !e.retained || e.started || p.entry("future19").selected {
		t.Fatalf("open supplied started credit: %+v", e)
	}
	if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "native", "--role", "evidence-scout", "--handle", "native-17"); code != 0 {
		t.Fatalf("start: %d %s", code, out)
	}
	if !canonicalPool(t, root, "current16").entry("future17").started {
		t.Fatal("recorded start not counted")
	}
	if code, out := controllerDispatch(t, root, "future17", "return", "--wave", "native", "--role", "evidence-scout"); code != 0 {
		t.Fatalf("return: %d %s", code, out)
	}
	account, _, err := loadDispatch(root, "future17")
	if err != nil {
		t.Fatal(err)
	}
	account.Waves["build"] = &dispatchWave{Phase: "build", State: "open", Roles: map[string]*dispatchRole{"slice-wright": {}}}
	canonicalPoolAccount(t, root, "future17", account)
	if canonicalPool(t, root, "current16").entry("future19").selected {
		t.Fatal("Build open freed buffer")
	}
	if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "build", "--role", "slice-wright", "--handle", "writer-17"); code != 0 {
		t.Fatalf("build start: %d %s", code, out)
	}
	p = canonicalPool(t, root, "current16")
	if !p.entry("future19").selected || !p.entry("future17").buildStarted || p.entry("future17").occupied {
		t.Fatalf("Build start did not refill: %+v", p.entries)
	}
	// Native repair stays outside the future buffer despite an earlier phase cursor.
	frontierFixture(t, root, "work", "future17", "current16", "plan", "running", "/rite-plan future17", 2, "2")
	if err := canonicalPool(t, root, "future17").admit("future17", preparationPurpose{}); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalPreparationPoolOrdinarySpecClarifyAndExactParent(t *testing.T) {
	for _, phase := range []string{"spec", "clarify", "temper", "define", "plan", "vet"} {
		t.Run(phase, func(t *testing.T) {
			root, _ := canonicalPreparationFixture(t, true)
			canonicalPoolChild(t, root, "future17", "current16", 2, "2")
			canonicalPoolChild(t, root, "future18", "future17", 3, "1")
			for _, purpose := range []preparationPurpose{{}, {parent: "future17"}} {
				if _, err := preparationAdmission(root, "future18", phase, purpose, nil); err != nil {
					t.Fatalf("recursive predecessor wait: %v", err)
				}
			}
			if _, err := preparationAdmission(root, "future18", phase, preparationPurpose{parent: "current16"}, nil); err == nil {
				t.Fatal("nonadjacent marker admitted")
			}
		})
	}
}

func TestCanonicalPreparationPoolMultiRoleCompetingWavesAndSettlement(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	t.Setenv("DEVRITES_HOST_CAPACITY", "2")
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	canonicalPoolChild(t, root, "future18", "future17", 3, "1")
	code, out := controllerDispatch(t, root, "future17", "open", "--phase", "plan", "--wave", "pair", "--role", "plan-drafter", "--role", "evidence-scout")
	if code != 0 {
		t.Fatalf("complete wave: %d %s", code, out)
	}
	before := controllerSnapshot(t, root)
	if code, out := preparationOpen(t, root, "future18", "plan", "plan-drafter", "competing"); code != 3 || !strings.Contains(out, "resource deferral") {
		t.Fatalf("oversubscribed wave: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("resource refusal wrote workflow")
	}
	if used, err := hostReservations(root, "current16", nil); err != nil || used != 2 {
		t.Fatalf("spawn-before-start reservations %d %v", used, err)
	}
	controllerDispatch(t, root, "future17", "abandon", "--wave", "pair", "--reason", "host outcome unknown")
	if used, err := hostReservations(root, "current16", nil); err != nil || used != 2 {
		t.Fatalf("abandon lost unknown reservations %d %v", used, err)
	}
	controllerDispatch(t, root, "future17", "abandon", "--wave", "pair", "--reason", "host confirmed no actors", "--release")
	if code, out := preparationOpen(t, root, "future18", "plan", "plan-drafter", "after-settlement"); code != 0 {
		t.Fatalf("settled reservations not freed: %d %s", code, out)
	}
}

func TestCanonicalPreparationPoolAbsentAndInvalidHostCapacity(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	if err := os.Unsetenv("DEVRITES_HOST_CAPACITY"); err != nil {
		t.Fatal(err)
	}
	if n, err := nativeHostCapacity(); err != nil || n != 1 {
		t.Fatalf("absent host: %d %v", n, err)
	}
	if code, out := controllerDispatch(t, root, "future17", "open", "--phase", "plan", "--wave", "too-wide", "--role", "plan-drafter", "--role", "evidence-scout"); code != 3 || !strings.Contains(out, "sealable subwaves") {
		t.Fatalf("absent capacity fabricated concurrency: %d %s", code, out)
	}
	for _, value := range []string{"", "0", "-1", "02", "many"} {
		t.Setenv("DEVRITES_HOST_CAPACITY", value)
		if _, err := nativeHostCapacity(); err == nil {
			t.Fatalf("invalid capacity accepted: %q", value)
		}
	}
}

func TestCanonicalPreparationPoolDetachedResearchAndArchivedAccountsConsumeHost(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	t.Setenv("DEVRITES_HOST_CAPACITY", "2")
	for _, where := range []string{"work", "archive"} {
		slug := "detached-" + where
		dir := frontierFixture(t, root, where, slug, "", "spec", "running", "/rite-spec "+slug, 0, "")
		raw, _ := json.Marshal(&dispatchFile{Waves: map[string]*dispatchWave{"research": {Phase: "spec", State: "abandoned", SequenceResearch: true, Roles: map[string]*dispatchRole{"evidence-scout": {}}}}})
		writeFile(t, filepath.Join(dir, "dispatch.json"), string(raw))
	}
	if used, err := hostReservations(root, "current16", nil); err != nil || used != 2 {
		t.Fatalf("detached actors disappeared: %d %v", used, err)
	}
	if code, out := preparationOpen(t, root, "future17", "spec", "evidence-scout", "blocked"); code != 3 || !strings.Contains(out, "resource deferral") {
		t.Fatalf("detached oversubscription: %d %s", code, out)
	}
}

func TestCanonicalPreparationPoolDurableHoldAndResumeCapacity(t *testing.T) {
	root, dir := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	canonicalPoolChild(t, root, "future18", "future17", 3, "1")
	writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — missing contract from owner; retry after contract lands\n")
	// Agent-owned blocked cursors alone do not override a concrete scheduling hold.
	frontierFixture(t, root, "work", "future17", "current16", "plan", "blocked", "/rite-plan repair future17", 2, "2")
	for _, source := range []string{"current16", "future18"} {
		p := canonicalPool(t, root, source)
		if p.entry("future17").occupied || !p.entry("future19").selected || !strings.Contains(p.entry("future17").parked, "owner=current16") {
			t.Fatalf("parking disagrees: %+v", p.entries)
		}
	}
	for _, purpose := range []preparationPurpose{{}, {parent: "current16"}} {
		if _, err := preparationAdmission(root, "future17", "plan", purpose, nil); err == nil || !strings.Contains(err.Error(), "frontier-blocked") {
			t.Fatalf("hold bypassed: %v", err)
		}
	}
	text := frontierText(t, root, "future18")
	if !strings.Contains(text, "frontier-blocked: future17 — owner=current16:") {
		t.Fatal(text)
	}
	if code, out := controllerDispatch(t, root, "future17", "park"); code != 0 {
		t.Fatalf("durable park: %d %s", code, out)
	}
	canonicalPoolChild(t, root, "future19", "future18", 4, "0")
	writeFile(t, filepath.Join(dir, "evidence.md"), "# Contract resolved\n")
	if _, err := preparationAdmission(root, "future17", "plan", preparationPurpose{}, nil); err == nil || !strings.Contains(err.Error(), "durably parked") {
		t.Fatalf("cleared hold bypassed re-admission: %v", err)
	}
	before := controllerSnapshot(t, root)
	if code, out := controllerDispatch(t, root, "future17", "resume"); code != 3 || !strings.Contains(out, "pool full") {
		t.Fatalf("resume displaced occupied buffer: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("failed resume cleared marker")
	}
	p := canonicalPool(t, root, "future18")
	if !p.entry("future18").selected || !p.entry("future19").selected || p.entry("future17").occupied {
		t.Fatalf("resume displaced incumbents: %+v", p.entries)
	}
	frontierFixture(t, root, "work", "future19", "future18", "done", "done", "", 4, "0")
	if code, out := controllerDispatch(t, root, "future17", "resume"); code != 0 {
		t.Fatalf("free slot refused resume: %d %s", code, out)
	}
	if !canonicalPool(t, root, "current16").entry("future17").occupied {
		t.Fatal("resume failed to restore membership")
	}
}

func TestCanonicalPreparationPoolParkedRetainedActorStillOccupies(t *testing.T) {
	root, dir := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	canonicalPoolChild(t, root, "future18", "future17", 3, "1")
	if code, out := preparationOpen(t, root, "future17", "spec", "evidence-scout", "unknown"); code != 0 {
		t.Fatalf("open: %d %s", code, out)
	}
	writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — human answer required\n")
	controllerDispatch(t, root, "future17", "abandon", "--wave", "unknown", "--reason", "spawn outcome unknown")
	p := canonicalPool(t, root, "current16")
	if !p.entry("future17").occupied || p.entry("future19").selected {
		t.Fatalf("parked actor erased: %+v", p.entries)
	}
	controllerDispatch(t, root, "future17", "abandon", "--wave", "unknown", "--reason", "host confirmed stopped", "--release")
	if !canonicalPool(t, root, "current16").entry("future19").selected {
		t.Fatal("settled hold did not free membership")
	}
}

func TestCanonicalPreparationPoolChangedAuthorityBlocksSpawnNotSettlement(t *testing.T) {
	root, dir := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	if code, out := preparationOpen(t, root, "future17", "spec", "evidence-scout", "native"); code != 0 {
		t.Fatalf("open: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "future17", "check-spawn", "--wave", "native", "--role", "evidence-scout"); code != 0 {
		t.Fatalf("spawn recheck: %d %s", code, out)
	}
	writeFile(t, filepath.Join(dir, "roadmap.md"), "Sequence root: current16\nDecision: DEC-001\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | renamed17 |\n")
	before := controllerSnapshot(t, root)
	if code, out := controllerDispatch(t, root, "future17", "check-spawn", "--wave", "native"); code != 3 {
		t.Fatalf("changed authority admitted spawn: %d %s", code, out)
	}
	if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
		t.Fatal("spawn refusal wrote workflow")
	}
	if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "native", "--role", "evidence-scout", "--handle", "spawn-already-happened"); code != 0 {
		t.Fatalf("late fact blocked: %d %s", code, out)
	}
	// Return is saved even if post-return frontier reports the changed authority.
	controllerDispatch(t, root, "future17", "return", "--wave", "native", "--role", "evidence-scout")
	account, _, err := loadDispatch(root, "future17")
	if err != nil || !account.Waves["native"].Roles["evidence-scout"].Returned {
		t.Fatalf("return not settled: %v", err)
	}
}

func TestCanonicalPreparationPoolLegacyCapOneOptOutAndBudget(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		root, _ := controllerFixture(t)
		p, err := evaluatePreparationPool(root, "current16", nil)
		if err != nil || p.enabled {
			t.Fatalf("headerless table became canonical: %+v %v", p, err)
		}
	})
	for _, policy := range []string{"max_parallel: 1\n", "max_parallel: 2\ncontinue_sequence: false\n"} {
		t.Run(strings.TrimSpace(policy), func(t *testing.T) {
			root, _ := canonicalPreparationFixture(t, true)
			writeFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\n"+policy)
			p, err := evaluatePreparationPool(root, "current16", nil)
			if err != nil {
				t.Fatal(err)
			}
			if p.enabled && p.entry("future17").selected {
				t.Fatal("opening policy ignored")
			}
		})
	}
	t.Run("zero budget", func(t *testing.T) {
		root, _ := canonicalPreparationFixture(t, true)
		frontierFixture(t, root, "work", "current16", "", "prove", "running", "/rite-prove current16", 1, "0")
		if canonicalPool(t, root, "current16").entry("future17").selected {
			t.Fatal("zero budget opened future")
		}
	})
	t.Run("inflated child cannot expand revoked budget", func(t *testing.T) {
		root, _ := canonicalPreparationFixture(t, true)
		frontierFixture(t, root, "work", "current16", "", "prove", "running", "/rite-prove current16", 1, "1")
		canonicalPoolChild(t, root, "future17", "current16", 2, "3")
		p := canonicalPool(t, root, "current16")
		if p.entry("future18").selected || p.entry("future17").remaining != 0 {
			t.Fatalf("inflated child expanded durable allowance: %+v", p.entries)
		}
		if err := p.admit("future17", preparationPurpose{}); err != nil {
			t.Fatalf("budget revocation blocked existing work: %v", err)
		}
	})
}

func TestCanonicalPreparationPoolWriterPriorityAndQueuedPreparation(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, false)
	t.Setenv("DEVRITES_HOST_CAPACITY", "1")
	frontierFixture(t, root, "work", "current16", "", "build", "running", "/rite-build SLICE-001", 1, "3")
	poolVet(t, root, "current16")
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	p := canonicalPool(t, root, "current16")
	if priority, err := p.writerPriority(root, "current16"); err != nil || priority != 1 {
		_, notices, readinessErr := WorkspaceReady(filepath.Dir(root), "current16", "current16")
		t.Fatalf("writer priority: %d %v; readiness=%v %v", priority, err, notices, readinessErr)
	}
	if code, out := preparationOpen(t, root, "future17", "spec", "evidence-scout", "too-early"); code != 3 || !strings.Contains(out, "writer launches") {
		t.Fatalf("preparation stole writer slot: %d %s", code, out)
	}
	if code, out := controllerOpen(t, root, "current16", "writer", "SLICE-001"); code != 0 {
		t.Fatalf("queued preparation blocked writer: %d %s", code, out)
	}
	if code, out := preparationOpen(t, root, "future17", "spec", "evidence-scout", "full-host"); code != 3 || !strings.Contains(out, "resource deferral") {
		t.Fatalf("writer-exhausted host admitted prep: %d %s", code, out)
	}
	text := frontierText(t, root, "future17")
	if !strings.Contains(text, "frontier-resource-deferred:") || strings.Contains(text, "frontier-concurrent: /rite-spec future17") {
		t.Fatal(text)
	}
	if frontierSourcePreparationWaitReason("future17", strings.Split(text, "\n")) == "" {
		t.Fatal("resource wait fell back into ordinary source launch")
	}
}

func TestCanonicalPreparationPoolLateAbandonedStartAndReturn(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	if code, out := preparationOpen(t, root, "future17", "spec", "evidence-scout", "unknown"); code != 0 {
		t.Fatalf("open: %d %s", code, out)
	}
	controllerDispatch(t, root, "future17", "abandon", "--wave", "unknown", "--reason", "spawn response lost")
	if code, out := controllerDispatch(t, root, "future17", "start", "--wave", "unknown", "--role", "evidence-scout", "--handle", "late-native"); code != 0 {
		t.Fatalf("late abandoned start refused: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "future17", "return", "--wave", "unknown", "--role", "evidence-scout", "--handle", "late-native"); code != 0 {
		t.Fatalf("late abandoned return refused: %d %s", code, out)
	}
	if used, err := hostReservations(root, "current16", nil); err != nil || used != 0 {
		t.Fatalf("late terminal return leaked reservation: %d %v", used, err)
	}
}

func TestCanonicalPreparationPoolParkRequiresSettlementAndImmutableSource(t *testing.T) {
	root, dir := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	if code, out := controllerDispatch(t, root, "future17", "park"); code != 3 {
		t.Fatalf("park without hold: %d %s", code, out)
	}
	if code, out := preparationOpen(t, root, "future17", "spec", "evidence-scout", "unknown"); code != 0 {
		t.Fatalf("open: %d %s", code, out)
	}
	writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — human contract missing\n")
	if code, out := controllerDispatch(t, root, "future17", "park"); code != 3 || !strings.Contains(out, "unresolved host roles") {
		t.Fatalf("park erased actor: %d %s", code, out)
	}
	controllerDispatch(t, root, "future17", "abandon", "--wave", "unknown", "--reason", "host confirmed no actor", "--release")
	if code, out := controllerDispatch(t, root, "future17", "park"); code != 0 {
		t.Fatalf("settled park: %d %s", code, out)
	}
	if code, out := controllerDispatch(t, root, "future17", "resume"); code != 3 || !strings.Contains(out, "frontier-blocked") {
		t.Fatalf("resume bypassed unresolved hold: %d %s", code, out)
	}
	account, _, err := loadDispatch(root, "future17")
	if err != nil {
		t.Fatal(err)
	}
	account.PoolParking.Parent = "changed-parent"
	canonicalPoolAccount(t, root, "future17", account)
	writeFile(t, filepath.Join(dir, "evidence.md"), "# Hold resolved\n")
	if code, out := controllerDispatch(t, root, "future17", "resume"); code != 3 || !strings.Contains(out, "changed canonical binding") {
		t.Fatalf("parking rebound ancestry: %d %s", code, out)
	}
}

func TestCanonicalPreparationPoolReservedWriterPriorityBeforeDispatch(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	frontierFixture(t, root, "work", "current16", "", "build", "running", "/rite-build SLICE-001", 1, "3")
	poolVet(t, root, "current16")
	t.Setenv("DEVRITES_HOST_CAPACITY", "2")
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	controllerLease(t, root, 2, controllerLeaseRow("", "SLICE-001", "src/current.go"))
	p := canonicalPool(t, root, "future17")
	if priority, err := p.writerPriority(root, "future17"); err != nil || priority != 1 {
		t.Fatalf("admitted writer lost host priority: %d %v", priority, err)
	}
	if err := p.hostPreparationAdmission(root, "future17", nil, 2); err == nil || !strings.Contains(err.Error(), "writer launches") {
		t.Fatalf("preparation stole undispatched reservation: %v", err)
	}
	account, _, err := loadDispatch(root, "current16")
	if err != nil {
		t.Fatal(err)
	}
	account.Waves["reserved"] = &dispatchWave{Phase: "build", State: "open", Slices: []string{"SLICE-001"}, Roles: map[string]*dispatchRole{"slice-wright": {}}}
	canonicalPoolAccount(t, root, "current16", account)
	p = canonicalPool(t, root, "future17")
	if priority, err := p.writerPriority(root, "future17"); err != nil || priority != 0 {
		t.Fatalf("dispatched writer double-counted: %d %v", priority, err)
	}
	if err := p.hostPreparationAdmission(root, "future17", nil, 1); err != nil {
		t.Fatalf("remaining actor slot withheld: %v", err)
	}
}

func TestCanonicalPreparationPoolWrongPhaseHasOccupancyNotStartedCredit(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	canonicalPoolChild(t, root, "future17", "current16", 2, "2")
	canonicalPoolAccount(t, root, "future17", &dispatchFile{Waves: map[string]*dispatchWave{"old-plan": {Phase: "plan", State: "sealed", Roles: map[string]*dispatchRole{"plan-drafter": {Handle: "native-old", StartedAt: "2026-01-01T00:00:00Z"}}}}})
	e := canonicalPool(t, root, "current16").entry("future17")
	if !e.occupied || !e.retained || e.started {
		t.Fatalf("wrong-phase actor became actual preparation credit: %+v", e)
	}
}

func TestHostReservationsReleaseDeadAbandonedWriter(t *testing.T) {
	root, _ := controllerFixture(t)
	t.Setenv("DEVRITES_HOST_CAPACITY", "1")
	writeFile(t, filepath.Join(root, "AFK"), "max_parallel: 10\n")
	foreign := frontierFixture(t, root, "archive", "fulfillment", "", "spec", "running", "/rite-spec fulfillment", 0, "")
	foreignPath := filepath.Join(foreign, "dispatch.json")
	foreignBytes := `{"waves":{"fulfilled":{"phase":"build","state":"sealed","roles":{"slice-wright":{"handle":"foreign-native","started":"2026-01-01T00:00:00Z"}}}}}`
	if code, out := controllerOpen(t, root, "current16", "dead-writer", "SLICE-001"); code != 0 {
		t.Fatalf("first wave: %d %s", code, out)
	}
	dead := reservationTestActor(t)
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "dead-writer", "--role", "slice-wright", "--handle", strconv.Itoa(dead.Process.Pid)); code != 0 {
		t.Fatalf("start writer: %d %s", code, out)
	}
	if err := dead.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := dead.Wait(); err == nil {
		t.Fatal("killed writer exited successfully")
	}
	if code, out := controllerDispatch(t, root, "current16", "abandon", "--wave", "dead-writer", "--reason", "writer process exited; host reconciliation finished"); code != 3 || !strings.Contains(out, "abandoned") {
		t.Fatalf("abandon: %d %s", code, out)
	}
	if code, out := controllerOpen(t, root, "current16", "replacement", "SLICE-001"); code != 0 {
		t.Fatalf("dead abandoned writer still consumes host slot: %d %s", code, out)
	}
	live := reservationTestActor(t)
	if code, out := controllerDispatch(t, root, "current16", "start", "--wave", "replacement", "--role", "slice-wright", "--handle", strconv.Itoa(live.Process.Pid)); code != 0 {
		t.Fatalf("start replacement: %d %s", code, out)
	}
	if code, out := controllerOpen(t, root, "current16", "oversubscribed", "SLICE-001"); code != 3 || !strings.Contains(out, "resource deferral") || !strings.Contains(out, "replacement") || strings.Contains(out, "dead-writer") {
		t.Fatalf("live reservation not identified: %d %s", code, out)
	}
	writeFile(t, foreignPath, foreignBytes)
	if code, out := controllerOpen(t, root, "current16", "foreign-blocked", "SLICE-001"); code != 3 || !strings.Contains(out, "resource deferral") {
		t.Fatalf("foreign reservation ignored: %d %s", code, out)
	}
	if got, err := os.ReadFile(foreignPath); err != nil || string(got) != foreignBytes {
		t.Fatalf("foreign sealed wave changed: %s %v", got, err)
	}
	account, _, err := loadDispatch(root, "current16")
	if err != nil || account.Waves["dead-writer"] == nil || account.Waves["dead-writer"].Roles["slice-wright"].Returned {
		t.Fatalf("abandonment deleted history or fabricated a return: %+v %v", account, err)
	}
}

func reservationTestActor(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHostReservationActor$") // #nosec G204 -- executes this test binary only.
	cmd.Env = append(os.Environ(), "DEVRITES_RESERVATION_TEST_ACTOR=1")
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = input.Close()
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	if line, err := bufio.NewReader(output).ReadString('\n'); err != nil || line != "ready\n" {
		t.Fatalf("actor did not start: %q %v", line, err)
	}
	return cmd
}

func TestHostReservationActor(t *testing.T) {
	if os.Getenv("DEVRITES_RESERVATION_TEST_ACTOR") != "1" {
		return
	}
	fmt.Println("ready")
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

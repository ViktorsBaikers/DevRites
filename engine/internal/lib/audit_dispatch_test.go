package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devrites/devrites/internal/state"
)

func TestAuditDispatchAbandonedWaveRetainsIdentityAndRoundHistory(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	contract := wrightContractPath(t)
	open := func(wave string) []string {
		return []string{"open", "--phase", "build", "--wave", wave, "--role", "slice-wright", "--slice", "SLICE-001", "--contract", contract}
	}
	mustRun := func(args ...string) {
		t.Helper()
		if code, out := runDispatchT(t, root, args...); code != 0 {
			t.Fatalf("%v: code=%d\n%s", args, code, out)
		}
	}
	mustRun(append(open("original"), "--claim-session", "original-owner", "--claim", "pkg/a.go")...)
	mustRun("start", "--wave", "original", "--role", "slice-wright", "--handle", "H1")
	if code, out := runDispatchT(t, root, "abandon", "--wave", "original", "--reason", "writer termination unknown"); code != 3 {
		t.Fatalf("abandon: code=%d\n%s", code, out)
	}
	before, _, err := loadDispatch(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	original := before.Waves["original"]
	if original == nil || original.Claim == "" || original.Roles["slice-wright"].Handle != "H1" {
		t.Fatalf("missing started wave history: %+v", original)
	}
	// Even an explicit replacement claim cannot transfer an unknown writer's identity.
	if code, out := runDispatchT(t, root, append(open("original"), "--claim-session", "replacement-owner", "--claim", "pkg/b.go")...); code != 3 {
		t.Errorf("abandoned wave ID reused: code=%d\n%s", code, out)
	}
	after, _, err := loadDispatch(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	got := after.Waves["original"]
	if got == nil || got.State != "abandoned" || got.Claim != original.Claim || got.ClaimSession != original.ClaimSession || got.Roles["slice-wright"].Handle != "H1" || got.Roles["slice-wright"].StartedAt != original.Roles["slice-wright"].StartedAt {
		t.Errorf("reopen overwrote abandoned wave history: before=%+v after=%+v", original, got)
	}
	if code, out, stderr := runClaim(t, root, "check", "--session", "foreign", "pkg/a.go"); code != 3 {
		t.Errorf("unknown writer claim released: code=%d\n%s%s", code, out, stderr)
	}
	// Original started wave plus four distinct started/abandoned waves spend five rounds.
	for i := 1; i <= 4; i++ {
		wave := fmt.Sprintf("distinct-%d", i)
		mustRun(open(wave)...)
		mustRun("start", "--wave", wave, "--role", "slice-wright", "--handle", fmt.Sprintf("distinct-H%d", i))
		if code, out := runDispatchT(t, root, "abandon", "--wave", wave, "--reason", "writer termination unknown"); code != 3 {
			t.Fatalf("abandon %s: code=%d\n%s", wave, code, out)
		}
	}
	if code, out := runDispatchT(t, root, open("sixth")...); code != 3 {
		t.Errorf("sixth build wright round admitted without reslice review: code=%d\n%s", code, out)
	}
}

func TestAuditDispatchWrongHandleCannotCompleteOrReleaseClaim(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	const role = "code-reviewer"
	packet := reviewerPacketPath(t)
	for _, args := range [][]string{
		{"open", "--phase", "build", "--wave", "abandoned", "--role", role, "--packet", packet},
		{"start", "--wave", "abandoned", "--role", role, "--handle", "H1"},
	} {
		if code, out := runDispatchT(t, root, args...); code != 0 {
			t.Fatalf("%v: code=%d\n%s", args, code, out)
		}
	}
	if code, out := runDispatchT(t, root, "abandon", "--wave", "abandoned", "--reason", "return unavailable"); code != 3 {
		t.Fatalf("abandon: code=%d\n%s", code, out)
	}
	for _, args := range [][]string{
		{"open", "--phase", "build", "--wave", "current", "--role", role, "--packet", packet, "--claim-session", "owner", "--claim", "pkg/a.go"},
		{"start", "--wave", "current", "--role", role, "--handle", "H2"},
	} {
		if code, out := runDispatchT(t, root, args...); code != 0 {
			t.Fatalf("%v: code=%d\n%s", args, code, out)
		}
	}
	result := filepath.Join(t.TempDir(), "return.md")
	writeFile(t, result, "Outcome: no-findings\nCounts: critical=0 important=0\n")
	snapshots := map[string][]byte{}
	for _, path := range []string{dispatchPath(root, "feat"), filepath.Join(featureDir, "metrics.jsonl"), claimsPath(root)} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		snapshots[path] = data
	}
	for _, wave := range []string{"abandoned", "current"} {
		if code, out := runDispatchT(t, root, "return", "--wave", wave, "--role", role, "--handle", "H1", "--result", result); code != 3 {
			t.Errorf("late handle accepted for %s: code=%d\n%s", wave, code, out)
		}
		stored := filepath.Join(featureDir, "packets", wave+"-"+role+".return.md")
		if _, err := os.Stat(stored); !os.IsNotExist(err) {
			t.Errorf("rejected return wrote result %s: %v", stored, err)
		}
	}
	for path, before := range snapshots {
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Errorf("rejected return changed %s: %v", path, err)
		}
	}
	f, _, err := loadDispatch(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	w := f.Waves["current"]
	if w.State != "sealed" || w.Roles[role].Returned || w.Roles[role].Handle != "H2" {
		t.Errorf("wrong handle settled current role: wave=%+v role=%+v", w, w.Roles[role])
	}
	if code, out, stderr := runClaim(t, root, "check", "--session", "foreign", "pkg/a.go"); code != 3 {
		t.Errorf("wrong handle released claim: code=%d\n%s%s", code, out, stderr)
	}
	if code, out := runDispatchT(t, root, "return", "--wave", "current", "--role", role, "--handle", "H2", "--result", result); code != 0 {
		t.Errorf("correct handle could not complete: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "status", "--wave", "current"); code != 0 {
		t.Errorf("correct return did not complete wave: code=%d\n%s", code, out)
	}
	if code, out, stderr := runClaim(t, root, "check", "--session", "foreign", "pkg/a.go"); code != 0 {
		t.Errorf("correct return retained claim: code=%d\n%s%s", code, out, stderr)
	}
	if code, out := runDispatchT(t, root, "return", "--wave", "current", "--role", role, "--handle", "H2", "--result", result); code != 3 {
		t.Errorf("duplicate return accepted: code=%d\n%s", code, out)
	}
}

func TestAuditDispatchWaveIDsAreSingleUseInEveryState(t *testing.T) {
	for _, state := range []string{"open", "sealed", "complete", "abandoned"} {
		t.Run(state, func(t *testing.T) {
			root, _ := newWorkspace(t, "build")
			open := []string{"open", "--phase", "review", "--wave", "attempt", "--role", "a"}
			if code, out := runDispatchT(t, root, open...); code != 0 {
				t.Fatalf("open: code=%d\n%s", code, out)
			}
			if state != "open" {
				if code, out := runDispatchT(t, root, "start", "--wave", "attempt", "--role", "a", "--handle", "H1"); code != 0 {
					t.Fatalf("start: code=%d\n%s", code, out)
				}
			}
			switch state {
			case "complete":
				if code, out := runDispatchT(t, root, "return", "--wave", "attempt", "--role", "a", "--handle", "H1"); code != 0 {
					t.Fatalf("return: code=%d\n%s", code, out)
				}
			case "abandoned":
				if code, out := runDispatchT(t, root, "abandon", "--wave", "attempt", "--reason", "writer termination unknown"); code != 3 {
					t.Fatalf("abandon: code=%d\n%s", code, out)
				}
			}
			before, err := os.ReadFile(dispatchPath(root, "feat"))
			if err != nil {
				t.Fatal(err)
			}
			if code, out := runDispatchT(t, root, append(open, "--claim-session", "owner", "--claim", "pkg/a.go")...); code != 3 {
				t.Errorf("%s wave ID reused: code=%d\n%s", state, code, out)
			}
			after, err := os.ReadFile(dispatchPath(root, "feat"))
			if err != nil || !bytes.Equal(before, after) {
				t.Errorf("rejected %s wave reuse changed history: %v", state, err)
			}
			if _, err := os.Stat(claimsPath(root)); !os.IsNotExist(err) {
				t.Errorf("rejected %s wave reuse created claim ledger: %v", state, err)
			}
		})
	}
}

func TestAuditDispatchAbandonedWriterBlocksFreshSameSessionOverlap(t *testing.T) {
	saved := processTable
	processTable = func() map[int]procInfo { return nil }
	t.Cleanup(func() { processTable = saved })
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprintf("expired=%t", expired), func(t *testing.T) {
			root, featureDir := newWorkspace(t, "build")
			writeFile(t, filepath.Join(featureDir, "tasks.md"), "# Tasks\n\n## SLICE-002 Retry\nWriter allowlist: PKG/a.go\n")
			contract := wrightContractPath(t)
			open := func(wave, slice string) []string {
				return []string{"open", "--phase", "build", "--wave", wave, "--role", "slice-wright", "--slice", slice, "--contract", contract}
			}
			mustRun := func(args ...string) {
				t.Helper()
				if code, out := runDispatchT(t, root, args...); code != 0 {
					t.Fatalf("%v: code=%d\n%s", args, code, out)
				}
			}
			mustRun(append(open("original", "SLICE-001"), "--claim-session", "owner", "--claim", "pkg/a.go")...)
			mustRun("start", "--wave", "original", "--role", "slice-wright", "--handle", "H1")
			if code, out := runDispatchT(t, root, "abandon", "--wave", "original", "--reason", "writer termination unknown"); code != 3 {
				t.Fatalf("abandon: code=%d\n%s", code, out)
			}
			f, _, err := loadDispatch(root, "feat")
			if err != nil {
				t.Fatal(err)
			}
			claimID := f.Waves["original"].Claim
			if expired {
				latest, _, _, err := readClaims(claimsPath(root))
				if err != nil {
					t.Fatal(err)
				}
				rec := *latest[claimID]
				rec.ExpiresAt = time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
				if err := appendClaim(claimsPath(root), &rec); err != nil {
					t.Fatal(err)
				}
			}
			for _, args := range [][]string{
				append(open("claim-overlap", "SLICE-003"), "--claim-session", "owner", "--claim", "./PKG/a.go"),
				open("slice-overlap", "SLICE-002"),
			} {
				if code, out := runDispatchT(t, root, args...); code != 3 {
					t.Errorf("fresh overlapping wave admitted: %v: code=%d\n%s", args, code, out)
				}
			}
			f, _, err = loadDispatch(root, "feat")
			if err != nil {
				t.Fatal(err)
			}
			if len(f.Waves) != 1 || f.Waves["original"].Claim != claimID || f.Waves["original"].Roles["slice-wright"].Handle != "H1" {
				t.Errorf("refused retry changed retained wave: %+v", f.Waves)
			}
			mustRun(append(open("disjoint", "SLICE-003"), "--claim-session", "owner", "--claim", "pkg/b.go")...)
			mustRun("start", "--wave", "disjoint", "--role", "slice-wright", "--handle", "H3")
			mustRun("return", "--wave", "disjoint", "--role", "slice-wright", "--handle", "H3")
			latest, _, _, err := readClaims(claimsPath(root))
			if err != nil {
				t.Fatal(err)
			}
			if latest[claimID] == nil || latest[claimID].Released {
				t.Fatal("disjoint wave completion released unknown writer claim")
			}
			if code, out := runDispatchT(t, root, "abandon", "--wave", "original", "--reason", "host confirmed terminal", "--release"); code != 3 {
				t.Fatalf("terminal abandon --release: code=%d\n%s", code, out)
			}
			mustRun(append(open("retry", "SLICE-002"), "--claim-session", "owner", "--claim", "pkg/a.go")...)
			mustRun("start", "--wave", "retry", "--role", "slice-wright", "--handle", "H2")
			mustRun("return", "--wave", "retry", "--role", "slice-wright", "--handle", "H2")
			if code, out, stderr := runClaim(t, root, "check", "--session", "foreign", "pkg/a.go", "pkg/b.go"); code != 0 {
				t.Errorf("terminal waves leaked claims: code=%d\n%s%s", code, out, stderr)
			}
		})
	}
}

func TestAuditDispatchCrossFeatureAbandonedWriterClaims(t *testing.T) {
	saved := processTable
	t.Cleanup(func() { processTable = saved })
	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name          string
		slug          string
		role          string
		session       string
		path          string
		expired       bool
		dead          bool
		released      bool
		archived      bool
		sliceOnly     bool
		legacy        bool
		unknown       bool
		malformed     bool
		alias         bool
		invalidHolder bool
		want          int
	}{
		{name: "same-session", session: "owner", path: "pkg/a.go", want: 3},
		{name: "expired", session: "other", path: "pkg/a.go", expired: true, want: 3},
		{name: "disjoint", session: "owner", path: "pkg/b.go"},
		{name: "proven-dead", session: "owner", path: "pkg/a.go", dead: true},
		{name: "released", session: "owner", path: "pkg/a.go", released: true},
		{name: "archived", session: "owner", path: "pkg/a.go", archived: true, want: 3},
		{name: "dot-feature-same-session", slug: ".feature", session: "owner", path: "pkg/a.go", want: 3},
		{name: "dot-feature-expired-foreign", slug: ".feature", session: "other", path: "pkg/a.go", expired: true, want: 3},
		{name: "dot-feature-archived-same-session", slug: ".feature", session: "owner", path: "pkg/a.go", archived: true, want: 3},
		{name: "dot-feature-archived-expired-foreign", slug: ".feature", session: "other", path: "pkg/a.go", expired: true, archived: true, want: 3},
		{name: "claimed-nonwright-same-session", role: "fast-builder", session: "owner", path: "pkg/a.go", want: 3},
		{name: "claimed-nonwright-expired-foreign", role: "fast-builder", session: "other", path: "pkg/a.go", expired: true, want: 3},
		{name: "claimed-nonwright-archived-same-session", role: "fast-builder", session: "owner", path: "pkg/a.go", archived: true, want: 3},
		{name: "claimed-nonwright-archived-expired-foreign", role: "fast-builder", session: "other", path: "pkg/a.go", expired: true, archived: true, want: 3},
		{name: "slice-only", path: "pkg/a.go", sliceOnly: true, want: 3},
		{name: "legacy", session: "owner", path: "pkg/a.go", expired: true, legacy: true, want: 3},
		{name: "unknown", session: "owner", path: "pkg/a.go", unknown: true, want: 3},
		{name: "malformed", session: "owner", path: "pkg/a.go", malformed: true, want: 3},
		{name: "physical-alias", session: "owner", path: "alias/a.go", alias: true, want: 3},
		{name: "invalid-death-proof", session: "owner", path: "pkg/a.go", dead: true, invalidHolder: true, want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			processTable = func() map[int]procInfo { return nil }
			root := claimTestRoot(t)
			slug := tc.slug
			if slug == "" {
				slug = "feat"
			}
			for _, slug := range []string{slug, "other"} {
				writeFile(t, filepath.Join(root, "work", slug, "state.md"), "# State\n")
			}
			realRunDispatch := func(slug string, args ...string) (int, string) {
				t.Helper()
				var out bytes.Buffer
				code := RunDispatch(root, append([]string{slug}, args...), &out, &out)
				return code, out.String()
			}
			role := tc.role
			if role == "" {
				role = "slice-wright"
			}
			open := func(wave, role, session, path string) []string {
				args := []string{"open", "--phase", "build", "--wave", wave, "--role", role}
				if role == "slice-wright" {
					args = append(args, "--slice", "SLICE-001", "--contract", wrightContractPath(t))
				}
				return append(args, "--claim-session", session, "--claim", path)
			}
			if code, out := realRunDispatch(slug, open("original", role, "owner", "pkg/a.go")...); code != 0 {
				t.Fatalf("original open: code=%d\n%s", code, out)
			}
			if code, out := realRunDispatch(slug, "start", "--wave", "original", "--role", role, "--handle", "H1"); code != 0 {
				t.Fatalf("original start: code=%d\n%s", code, out)
			}
			abandon := []string{"abandon", "--wave", "original", "--reason", "writer termination unknown"}
			if tc.released {
				abandon = append(abandon, "--release")
			}
			if code, out := realRunDispatch(slug, abandon...); code != 3 {
				t.Fatalf("abandon: code=%d\n%s", code, out)
			}
			before, _, err := loadDispatch(root, slug)
			if err != nil {
				t.Fatal(err)
			}
			if tc.expired || tc.dead || tc.legacy {
				latest, _, _, err := readClaims(claimsPath(root))
				if err != nil {
					t.Fatal(err)
				}
				rec := *latest[before.Waves["original"].Claim]
				if tc.legacy {
					rec.Host, rec.Anchors, rec.Holder = "", nil, nil
				}
				if tc.expired {
					rec.ExpiresAt = time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
				}
				if tc.dead {
					rec.Host = host
					rec.Holder = &claimProc{PID: 999999, Start: "2020-01-01T00:00:00Z"}
					processTable = func() map[int]procInfo { return map[int]procInfo{} }
					if tc.invalidHolder {
						rec.Holder = &claimProc{}
					}
				}
				if err := appendClaim(claimsPath(root), &rec); err != nil {
					t.Fatal(err)
				}
			}
			originalDir := filepath.Join(root, "work", slug)
			if tc.alias {
				if err := os.MkdirAll(filepath.Join(filepath.Dir(root), "pkg"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("pkg", filepath.Join(filepath.Dir(root), "alias")); err != nil {
					t.Fatal(err)
				}
			}
			if tc.unknown {
				if err := os.Remove(claimsPath(root)); err != nil {
					t.Fatal(err)
				}
			}
			if tc.malformed {
				writeFile(t, filepath.Join(originalDir, "dispatch.json"), "{")
			}
			if tc.archived {
				archived := filepath.Join(root, "archive", slug)
				if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(originalDir, archived); err != nil {
					t.Fatal(err)
				}
				originalDir = archived
			}
			original, err := os.ReadFile(filepath.Join(originalDir, "dispatch.json"))
			if err != nil {
				t.Fatal(err)
			}
			claimsBefore, err := os.ReadFile(claimsPath(root))
			if tc.unknown && os.IsNotExist(err) {
				claimsBefore = nil
			} else if err != nil {
				t.Fatal(err)
			}
			if tc.role != "" {
				for _, sub := range []string{"check", "add"} {
					if code, out, stderr := runClaim(t, root, sub, "--session", tc.session, tc.path); code != 3 {
						t.Errorf("%s admitted abandoned non-wright claim: code=%d\n%s%s", sub, code, out, stderr)
					}
					claimsAfter, err := os.ReadFile(claimsPath(root))
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(claimsBefore, claimsAfter) {
						t.Errorf("refused %s changed claims", sub)
					}
					dispatchAfter, err := os.ReadFile(filepath.Join(originalDir, "dispatch.json"))
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(original, dispatchAfter) {
						t.Errorf("refused %s changed original wave", sub)
					}
				}
			}
			replacement := open("replacement", "slice-wright", tc.session, tc.path)
			if tc.sliceOnly {
				writeFile(t, filepath.Join(root, "work", "other", "tasks.md"), "# Tasks\n\n## SLICE-001 Retry\nWriter allowlist: pkg/a.go\n")
				replacement = replacement[:len(replacement)-4]
			}
			if code, out := realRunDispatch("other", replacement...); code != tc.want {
				t.Errorf("cross-feature open: code=%d want=%d\n%s", code, tc.want, out)
			}
			after, err := os.ReadFile(filepath.Join(originalDir, "dispatch.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(after, original) {
				t.Fatal("cross-feature admission changed original ownership")
			}
			if tc.want != 0 {
				claimsAfter, err := os.ReadFile(claimsPath(root))
				if tc.unknown && os.IsNotExist(err) {
					claimsAfter = nil
				} else if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(claimsAfter, claimsBefore) {
					t.Error("refused cross-feature admission changed claims")
				}
				if _, err := os.Stat(filepath.Join(root, "work", "other", "dispatch.json")); !os.IsNotExist(err) {
					t.Errorf("refused cross-feature admission published replacement: %v", err)
				}
			}
		})
	}
}

func TestAuditDispatchAbandonedWriterWithProvenDeadHolderAllowsRetry(t *testing.T) {
	saved := processTable
	processTable = func() map[int]procInfo { return map[int]procInfo{} }
	t.Cleanup(func() { processTable = saved })
	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	root, _ := newWorkspace(t, "build")
	contract := wrightContractPath(t)
	open := func(wave string) []string {
		return []string{"open", "--phase", "build", "--wave", wave, "--role", "slice-wright", "--slice", "SLICE-001", "--contract", contract, "--claim-session", "owner", "--claim", "pkg/a.go"}
	}
	for _, args := range [][]string{
		open("original"),
		{"start", "--wave", "original", "--role", "slice-wright", "--handle", "H1"},
	} {
		if code, out := runDispatchT(t, root, args...); code != 0 {
			t.Fatalf("%v: code=%d\n%s", args, code, out)
		}
	}
	if code, out := runDispatchT(t, root, "abandon", "--wave", "original", "--reason", "writer termination unknown"); code != 3 {
		t.Fatalf("abandon: code=%d\n%s", code, out)
	}
	f, _, err := loadDispatch(root, "feat")
	if err != nil {
		t.Fatal(err)
	}
	latest, _, _, err := readClaims(claimsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	rec := *latest[f.Waves["original"].Claim]
	rec.Host = host
	rec.Holder = &claimProc{PID: 62926, Start: "2026-10-16T01:28:12Z"}
	if err := appendClaim(claimsPath(root), &rec); err != nil {
		t.Fatal(err)
	}
	if code, out := runDispatchT(t, root, open("retry")...); code != 0 {
		t.Fatalf("proven-dead holder blocked fresh retry: code=%d\n%s", code, out)
	}
}

func TestAuditDispatchNativeRapidSettlement(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	packet := reviewerPacketPath(t)
	run := func(code int, args ...string) {
		t.Helper()
		if got, out := runDispatchT(t, root, args...); got != code {
			t.Fatalf("%v: code=%d want=%d\n%s", args, got, code, out)
		}
	}
	// Retry only when real wall time crosses a second during native operations.
	for attempt := range 5 {
		stale, fresh := fmt.Sprintf("z-stale-%d", attempt), fmt.Sprintf("a-fresh-%d", attempt)
		run(0, "open", "--phase", "build", "--wave", stale, "--role", "code-reviewer", "--packet", packet)
		run(0, "start", "--wave", stale, "--role", "code-reviewer", "--handle", stale)
		run(3, "abandon", "--wave", stale, "--reason", "host confirmed terminal")
		run(0, "open", "--phase", "build", "--wave", fresh, "--role", "code-reviewer", "--packet", packet)
		run(0, "start", "--wave", fresh, "--role", "code-reviewer", "--handle", fresh)
		run(0, "return", "--wave", fresh, "--role", "code-reviewer", "--outcome", "no-findings")
		ledger, _, err := loadDispatch(root, "feat")
		if err != nil {
			t.Fatal(err)
		}
		old, current := ledger.Waves[stale], ledger.Waves[fresh]
		start, _, err := settledReviewer(current.Roles["code-reviewer"])
		if err != nil {
			t.Fatal(err)
		}
		boundary, err := time.Parse(time.RFC3339Nano, old.AbandonedAt)
		if err != nil {
			t.Fatal(err)
		}
		if !start.After(boundary) || !reconciledReviewer(ledger, old, "code-reviewer", old.Roles["code-reviewer"]) {
			t.Fatalf("native fresh account cannot reconcile: abandoned=%+v replacement=%+v", old, current)
		}
		if start.Truncate(time.Second).Equal(boundary.Truncate(time.Second)) {
			return
		}
	}
	t.Fatal("native operations never completed within the same second")
}

type auditDispatchWriteFunc func([]byte) (int, error)

func (f auditDispatchWriteFunc) Write(p []byte) (int, error) { return f(p) }

func TestAuditDispatchTerminalReleaseReconcilesCommittedClaim(t *testing.T) {
	for _, mode := range []string{"public-release", "save-failure"} {
		t.Run(mode, func(t *testing.T) {
			project, root := writeCandidateTestWorkspace(t, candidateTestManifest(candidateTestRow{state: "present", path: "pkg/a.go", slice: "SLICE-001", reason: "Claimed writer subject."}))
			dir := filepath.Join(root, "work", "feat")
			if err := os.Rename(filepath.Join(root, "work", "feature"), dir); err != nil {
				t.Fatal(err)
			}
			initDupRepo(t, project)
			writeCandidateTestFile(t, project, "pkg/a.go", "package pkg\n")
			runDupGit(t, project, "add", "pkg/a.go")
			runDupGit(t, project, "commit", "-q", "-m", "test: baseline")
			writeCandidateTestFile(t, project, "pkg/a.go", "package pkg\n\n// Claimed writer subject.\n")
			var frozen bytes.Buffer
			if code := RunCandidateFreeze(root, []string{"feat", "--wave", "candidate", "--paths", "pkg/a.go"}, &frozen, &frozen); code != 0 {
				t.Fatalf("freeze candidate: code=%d\n%s", code, &frozen)
			}
			packet := filepath.Join(dir, "packets", "review.md")
			writeFile(t, packet, "# Scope\nReview pkg/a.go against AC-001.\n\n"+readTestFile(t, filepath.Join(dir, "packets", "candidate.sha256")))
			writeFile(t, filepath.Join(dir, "gates.md"), "# Gates\n\n- [x] AC-001: reviewer account retained\n  EVIDENCE: dispatch.json\n")
			run := func(want int, args ...string) {
				t.Helper()
				if code, out := runDispatchT(t, root, args...); code != want {
					t.Fatalf("%v: code=%d want=%d\n%s", args, code, want, out)
				}
			}
			run(0, "open", "--phase", "review", "--wave", "review", "--role", "code-reviewer", "--packet", packet)
			run(0, "start", "--wave", "review", "--role", "code-reviewer", "--handle", "review-host")
			run(0, "return", "--wave", "review", "--role", "code-reviewer", "--outcome", "no-findings")
			run(0, "open", "--phase", "build", "--wave", "writer", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t), "--claim-session", "owner", "--claim", "pkg/a.go")
			run(0, "start", "--wave", "writer", "--role", "slice-wright", "--handle", "writer-host")
			run(3, "abandon", "--wave", "writer", "--reason", "termination unknown")
			before, path, err := loadDispatch(root, "feat")
			if err != nil {
				t.Fatal(err)
			}
			id := before.Waves["writer"].Claim
			if mode == "public-release" {
				if code, out, stderr := runClaim(t, root, "release", "--session", "owner", "--id", id); code != 0 {
					t.Fatalf("public release: code=%d\n%s%s", code, out, stderr)
				}
			} else {
				moved := path + ".saved"
				interrupted := false
				var out bytes.Buffer
				writer := auditDispatchWriteFunc(func(p []byte) (int, error) {
					if bytes.Contains(p, []byte("released claim")) {
						if err := os.Rename(path, moved); err != nil {
							t.Fatal(err)
						}
						if err := os.Mkdir(path, 0o755); err != nil {
							t.Fatal(err)
						}
						interrupted = true
					}
					return out.Write(p)
				})
				code := RunDispatch(root, []string{"feat", "abandon", "--wave", "writer", "--reason", "host confirmed terminal", "--release"}, writer, &out)
				if code != 3 || !interrupted || strings.Contains(out.String(), "HANDOFF REQUIRED") {
					t.Fatalf("expected committed release then failed wave save: code=%d interrupted=%t\n%s", code, interrupted, &out)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(moved, path); err != nil {
					t.Fatal(err)
				}
			}
			latest, _, _, err := readClaims(claimsPath(root))
			if err != nil {
				t.Fatal(err)
			}
			if latest[id] == nil || !latest[id].Released {
				t.Fatal("release was not actually committed")
			}
			current, _, err := loadDispatch(root, "feat")
			if err != nil {
				t.Fatal(err)
			}
			if current.Waves["writer"].TerminalAt != "" || current.Waves["writer"].Claim != id {
				t.Fatalf("uncommitted terminal marker changed wave: %+v", current.Waves["writer"])
			}
			checkFinal := func() error {
				return state.WithFeatureLock(root, "feat", func() error { return CheckFinalPrerequisites(root, "feat") })
			}
			if err := checkFinal(); err == nil || !strings.Contains(err.Error(), "confirmed terminal writer") {
				t.Fatalf("released claim certified unconfirmed writer: %v", err)
			}
			claimsBefore, err := os.ReadFile(claimsPath(root))
			if err != nil {
				t.Fatal(err)
			}
			run(3, "abandon", "--wave", "writer", "--reason", "host confirmed terminal", "--release")
			after, _, err := loadDispatch(root, "feat")
			if err != nil {
				t.Fatal(err)
			}
			w := after.Waves["writer"]
			if w.TerminalAt == "" || w.Claim != "" || w.ClaimSession != "" || w.Roles["slice-wright"].Handle != "writer-host" {
				t.Fatalf("committed release stranded terminal reconciliation: %+v", w)
			}
			claimsAfter, err := os.ReadFile(claimsPath(root))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(claimsBefore, claimsAfter) {
				t.Error("terminal reconciliation rewrote released claim")
			}
			if err := checkFinal(); err != nil {
				t.Fatalf("terminal reconciliation still blocks final prerequisites: %v", err)
			}
		})
	}
}

func TestAuditDispatchTerminalReleaseRefusesUnverifiedClaim(t *testing.T) {
	for _, mode := range []string{"unknown", "wrong-session", "wrong-session-released", "ledger-read-failure"} {
		t.Run(mode, func(t *testing.T) {
			root, _ := newWorkspace(t, "build")
			if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "writer", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t), "--claim-session", "owner", "--claim", "pkg/a.go"); code != 0 {
				t.Fatalf("open: code=%d\n%s", code, out)
			}
			f, path, err := loadDispatch(root, "feat")
			if err != nil {
				t.Fatal(err)
			}
			w := f.Waves["writer"]
			id := w.Claim
			switch mode {
			case "unknown":
				w.Claim = "cl-unknown"
			case "wrong-session", "wrong-session-released":
				w.ClaimSession = "not-owner"
				if mode == "wrong-session-released" {
					if code, _, stderr := runClaim(t, root, "release", "--session", "owner", "--id", id); code != 0 {
						t.Fatalf("release: code=%d %s", code, stderr)
					}
				}
			case "ledger-read-failure":
				if err := os.Rename(claimsPath(root), claimsPath(root)+".saved"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(claimsPath(root), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := saveDispatch(path, f); err != nil {
				t.Fatal(err)
			}
			code, out := runDispatchT(t, root, "abandon", "--wave", "writer", "--reason", "host confirmed terminal", "--release")
			if code != 3 || !strings.Contains(out, "claim not released") {
				t.Fatalf("unverified release not reported: code=%d\n%s", code, out)
			}
			after, _, err := loadDispatch(root, "feat")
			if err != nil {
				t.Fatal(err)
			}
			got := after.Waves["writer"]
			if got.TerminalAt != "" || got.Claim != w.Claim || got.ClaimSession != w.ClaimSession {
				t.Fatalf("failed release certified termination or cleared ownership: %+v", got)
			}
		})
	}
}

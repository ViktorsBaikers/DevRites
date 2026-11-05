package lib

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func controllerClaimAdmissionFixture(t *testing.T) (string, string) {
	t.Helper()
	root, dir := controllerFixture(t)
	writeFile(t, filepath.Join(dir, "evidence.md"), "frontier-blocked: future17 — API owner must resolve DEC-017 before planning\n")
	saved := processTable
	processTable = func() map[int]procInfo { return nil }
	t.Cleanup(func() { processTable = saved })
	return root, dir
}

func controllerAdmissionClaim(t *testing.T, root, session string, paths ...string) string {
	t.Helper()
	var errOut strings.Builder
	id, code := addClaim(root, session, "existing writer", claimDefaultTTL, paths, io.Discard, &errOut)
	if code != 0 {
		t.Fatalf("existing writer claim: %d %s", code, &errOut)
	}
	return id
}

func TestAutocompleteControllerLiveForeignClaimCannotBeBypassedAtOpen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra []string
	}{
		{"no optional claim flags", nil},
		{"unrelated optional claim", []string{"--claim-session", "native-controller", "--claim", "src/unrelated.go"}},
		{"matching optional claim", []string{"--claim-session", "native-controller", "--claim", "src/current.go"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := controllerClaimAdmissionFixture(t)
			controllerAdmissionClaim(t, root, "foreign-session", "src/current.go")
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001", tc.extra...)
			if code != 3 || !strings.Contains(out, "BLOCKED") || !strings.Contains(out, "foreign-session") || !strings.Contains(out, "src/current.go") {
				t.Fatalf("live foreign allowlist owner bypassed: code=%d\n%s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("refused writer changed claims, dispatch, round history, or source artifacts")
			}
		})
	}
}

func TestAutocompleteControllerLiveAliasClaimCannotBypassWriterAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, session string
		extra         []string
		allowed       bool
	}{
		{"no optional claim flags", "foreign-session", nil, false},
		{"unrelated optional claim", "foreign-session", []string{"--claim-session", "native-controller", "--claim", "src/unrelated.go"}, false},
		{"same-session allocation", "native-controller", []string{"--claim-session", "native-controller", "--claim", "src/unrelated.go"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := controllerClaimAdmissionFixture(t)
			project := filepath.Dir(root)
			writeFile(t, filepath.Join(project, "src", "current.go"), "package current\n")
			if err := os.Symlink("src", filepath.Join(project, "alias")); err != nil {
				t.Fatal(err)
			}
			if code, out, errOut := runClaim(t, root, "add", "--session", tc.session, "alias/current.go"); code != 0 {
				t.Fatalf("public alias claim add: code=%d\n%s%s", code, out, errOut)
			}
			before := controllerSnapshot(t, root)
			code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001", tc.extra...)
			if tc.allowed {
				if code != 0 || !strings.Contains(out, "wave writer16 open") {
					t.Fatalf("same-session alias allocation refused: code=%d\n%s", code, out)
				}
				return
			}
			if code != 3 || !strings.Contains(out, "BLOCKED") || !strings.Contains(out, "foreign-session") || !strings.Contains(out, "alias/current.go") {
				t.Fatalf("live foreign alias owner bypassed: code=%d\n%s", code, out)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatal("refused alias overlap changed claims, dispatch, round history, or source artifacts")
			}
		})
	}
}

func TestClaimLiveAliasOwnershipAppliesToPublicCheckAndAdd(t *testing.T) {
	for _, sub := range []string{"check", "add"} {
		t.Run(sub, func(t *testing.T) {
			root, _ := controllerClaimAdmissionFixture(t)
			project := filepath.Dir(root)
			writeFile(t, filepath.Join(project, "src", "current.go"), "package current\n")
			if err := os.Symlink("src", filepath.Join(project, "alias")); err != nil {
				t.Fatal(err)
			}
			if code, out, errOut := runClaim(t, root, "add", "--session", "foreign-session", "alias/current.go"); code != 0 {
				t.Fatalf("public alias claim add: code=%d\n%s%s", code, out, errOut)
			}
			before := controllerSnapshot(t, root)
			code, out, errOut := runClaim(t, root, sub, "--session", "native-controller", "src/current.go")
			if code != 3 || !strings.Contains(out+errOut, "foreign-session") {
				t.Fatalf("public %s bypassed live alias owner: code=%d\n%s%s", sub, code, out, errOut)
			}
			if !reflect.DeepEqual(before, controllerSnapshot(t, root)) {
				t.Fatalf("refused public %s changed claim ledger or workspace", sub)
			}
		})
	}
}

func TestAutocompleteControllerSameSessionDisjointCohortKeepsOwnClaims(t *testing.T) {
	root, dir := controllerClaimAdmissionFixture(t)
	writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n"+
		poolSlice("SLICE-001", "src/current.go", "Workspace dependencies: none")+
		poolSlice("SLICE-002", "src/disjoint.go", "Workspace dependencies: none"))
	poolVet(t, root, "current16")
	controllerLease(t, root, 2,
		controllerLeaseRow("", "SLICE-001", "src/current.go"),
		controllerLeaseRow("", "SLICE-002", "src/disjoint.go"))
	controllerAdmissionClaim(t, root, "native-controller", "src")
	for _, row := range []struct{ wave, slice, path string }{
		{"writer16", "SLICE-001", "src/current.go"},
		{"disjoint16", "SLICE-002", "src/disjoint.go"},
	} {
		code, out := controllerOpen(t, root, "current16", row.wave, row.slice,
			"--claim-session", "native-controller", "--claim", row.path)
		if code != 0 || !strings.Contains(out, "wave "+row.wave+" open") {
			t.Fatalf("same-session disjoint cohort refused: %s code=%d\n%s", row.slice, code, out)
		}
	}
	if code, out, errOut := runClaim(t, root, "check", "--session", "foreign-session", "src/current.go", "src/disjoint.go"); code != 3 || !strings.Contains(out+errOut, "native-controller") {
		t.Fatalf("cohort admission lost existing session ownership: code=%d\n%s%s", code, out, errOut)
	}
}

func TestAutocompleteControllerInactiveForeignClaimsDoNotBlockWriter(t *testing.T) {
	for _, state := range []string{"expired", "released", "holder exited"} {
		for _, claimed := range []bool{false, true} {
			name := state + "/no optional claim"
			if claimed {
				name = state + "/unrelated optional claim"
			}
			t.Run(name, func(t *testing.T) {
				root, _ := controllerClaimAdmissionFixture(t)
				id := controllerAdmissionClaim(t, root, "foreign-session", "src/current.go")
				if state == "released" {
					if code, out, errOut := runClaim(t, root, "release", "--session", "foreign-session", "--id", id); code != 0 {
						t.Fatalf("release existing claim: %d\n%s%s", code, out, errOut)
					}
				} else {
					latest, _, _, err := readClaims(claimsPath(root))
					if err != nil {
						t.Fatal(err)
					}
					rec := *latest[id]
					if state == "expired" {
						rec.ExpiresAt = time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
					} else {
						rec.Host, err = os.Hostname()
						if err != nil {
							t.Fatal(err)
						}
						rec.Holder = &claimProc{PID: 999999, Start: "2020-01-01T00:00:00Z"}
						processTable = func() map[int]procInfo { return map[int]procInfo{} }
					}
					if err := appendClaim(claimsPath(root), &rec); err != nil {
						t.Fatal(err)
					}
				}
				before, err := os.ReadFile(claimsPath(root))
				if err != nil {
					t.Fatal(err)
				}
				var extra []string
				if claimed {
					extra = []string{"--claim-session", "native-controller", "--claim", "src/unrelated.go"}
				}
				code, out := controllerOpen(t, root, "current16", "writer16", "SLICE-001", extra...)
				if code != 0 || !strings.Contains(out, "wave writer16 open") {
					t.Fatalf("inactive ordinary claim blocked writer: code=%d\n%s", code, out)
				}
				if !claimed {
					after, err := os.ReadFile(claimsPath(root))
					if err != nil || string(after) != string(before) {
						t.Fatalf("claimless admission rewrote existing ledger: %v", err)
					}
				}
			})
		}
	}
}

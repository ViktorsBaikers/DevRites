package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditClaimSessionExitFreesPathsDespitePersistentLauncher(t *testing.T) {
	for _, host := range []string{"claude", "codex", "pi", "omp", "devin", "dsh"} {
		for _, tc := range []struct {
			name    string
			age     time.Duration
			wrapper string
		}{
			{"fresh_session", 20 * time.Second, ""},
			{"older_session", 2 * time.Hour, ""},
			{"fresh_session_under_launcher", 20 * time.Second, "/usr/local/bin/launcher"},
			{"fresh_session_under_python", 20 * time.Second, "/usr/bin/python3"},
			{"fresh_session_under_runuser", 20 * time.Second, "/usr/sbin/runuser"},
			{"fresh_session_under_go_helper", 20 * time.Second, "/tmp/go-build123/b001/exe/claim-helper"},
			{"older_session_under_launcher", 2 * time.Hour, "/usr/local/bin/launcher"},
			{"older_session_under_python", 2 * time.Hour, "/usr/bin/python3"},
			{"older_session_under_runuser", 2 * time.Hour, "/usr/sbin/runuser"},
			{"older_session_under_go_helper", 2 * time.Hour, "/tmp/go-build123/b001/exe/claim-helper"},
		} {
			t.Run(host+"/"+tc.name, func(t *testing.T) {
				now := time.Now().UTC()
				old := now.Add(-24 * time.Hour).Format(time.RFC3339)
				caller := os.Getppid()
				session := claimProc{PID: caller + 100, Start: now.Add(-tc.age).Format(time.RFC3339)}
				shell, mux, wrapper := caller+101, caller+102, caller+103
				table := map[int]procInfo{
					caller:      {ppid: session.PID, start: now.Add(-time.Second).Format(time.RFC3339), comm: "sh"},
					session.PID: {ppid: shell, start: session.Start, comm: "/usr/local/bin/" + host},
					shell:       {ppid: mux, start: old, comm: "-zsh"},
					mux:         {ppid: 1, start: old, comm: "herdr"},
				}
				if tc.wrapper != "" {
					table[caller] = procInfo{ppid: wrapper, start: now.Format(time.RFC3339), comm: "sh"}
					table[wrapper] = procInfo{ppid: session.PID, start: now.Add(-time.Second).Format(time.RFC3339), comm: tc.wrapper}
				}
				saved := processTable
				processTable = func() map[int]procInfo { return table }
				t.Cleanup(func() { processTable = saved })
				root := claimTestRoot(t)
				if code, _, errOut := runClaim(t, root, "add", "--session", "s1", "pkg/a.go"); code != 0 {
					t.Fatalf("initial add: code=%d stderr=%s", code, errOut)
				}
				raw, err := os.ReadFile(filepath.Join(root, ClaimsFile))
				if err != nil {
					t.Fatal(err)
				}
				var rec claimRecord
				if err := json.Unmarshal(raw, &rec); err != nil {
					t.Fatal(err)
				}
				if rec.Holder == nil || *rec.Holder != session {
					t.Errorf("holder=%+v, want agent session %+v", rec.Holder, session)
				}
				if code, out, errOut := runClaim(t, root, "list"); code != 0 || !strings.Contains(out, rec.ID+" live") {
					t.Fatalf("running session must keep claim live: code=%d out=%q stderr=%s", code, out, errOut)
				}
				if code, _, errOut := runClaim(t, root, "add", "--session", "s2", "pkg/a.go"); code != 3 {
					t.Fatalf("running session must block foreign claim: code=%d stderr=%s", code, errOut)
				}

				delete(table, session.PID)
				// A later command shares the persistent shell, not the exited agent.
				table[caller] = procInfo{ppid: shell, start: now.Format(time.RFC3339), comm: "sh"}
				if code, out, errOut := runClaim(t, root, "list"); code != 0 || !strings.Contains(out, rec.ID+" dead") {
					t.Errorf("exited session must make claim dead despite surviving launcher: code=%d out=%q stderr=%s", code, out, errOut)
				}
				if code, out, errOut := runClaim(t, root, "check", "--session", "s2", "pkg/a.go"); code != 0 {
					t.Errorf("exited session must free path: code=%d out=%q stderr=%s", code, out, errOut)
				}
				if code, _, errOut := runClaim(t, root, "add", "--session", "s2", "pkg/a.go"); code != 0 {
					t.Errorf("foreign claim after session exit: code=%d stderr=%s", code, errOut)
				}
			})
		}
	}
}

func TestAuditClaimUnknownLineageRemainsConservative(t *testing.T) {
	for _, runtime := range []string{"custom-agent", "node", "bun", "launcher"} {
		for _, age := range []time.Duration{20 * time.Second, 2 * time.Hour} {
			now := time.Now().UTC()
			fresh := now.Add(-age).Format(time.RFC3339)
			old := now.Add(-24 * time.Hour).Format(time.RFC3339)
			caller := os.Getppid()
			session, shell, mux := caller+100, caller+101, caller+102
			for _, tc := range []struct {
				name   string
				table  map[int]procInfo
				exited []int
			}{
				{"unavailable_process_table", nil, nil},
				{"missing_caller_lineage", map[int]procInfo{1: {}}, nil},
				{
					"unrecognized_session_survives_launchers",
					map[int]procInfo{
						caller:  {ppid: session, start: now.Add(-time.Second).Format(time.RFC3339), comm: "/usr/bin/python3"},
						session: {ppid: shell, start: fresh, comm: "/usr/local/bin/" + runtime},
						shell:   {ppid: mux, start: old, comm: "-zsh"},
						mux:     {ppid: 1, start: old, comm: "herdr"},
					},
					[]int{caller, shell, mux},
				},
			} {
				t.Run(runtime+"/"+age.String()+"/"+tc.name, func(t *testing.T) {
					saved := processTable
					processTable = func() map[int]procInfo { return tc.table }
					t.Cleanup(func() { processTable = saved })
					root := claimTestRoot(t)
					if code, _, errOut := runClaim(t, root, "add", "--session", "s1", "pkg/a.go"); code != 0 {
						t.Fatalf("initial add: code=%d stderr=%s", code, errOut)
					}
					raw, err := os.ReadFile(filepath.Join(root, ClaimsFile))
					if err != nil {
						t.Fatal(err)
					}
					var rec claimRecord
					if err := json.Unmarshal(raw, &rec); err != nil {
						t.Fatal(err)
					}
					if rec.Holder != nil {
						t.Fatalf("unknown lineage must not invent holder: %+v", rec.Holder)
					}
					if len(rec.Anchors) != 0 {
						t.Fatalf("unknown lineage must not promote ancestors: %+v", rec.Anchors)
					}
					// Unrecognized writer stays alive even after every observed launcher exits.
					for _, pid := range tc.exited {
						delete(tc.table, pid)
					}
					if code, out, errOut := runClaim(t, root, "list"); code != 0 || !strings.Contains(out, rec.ID+" live") {
						t.Fatalf("unknown holder must stay live: code=%d out=%q stderr=%s", code, out, errOut)
					}
					if code, _, errOut := runClaim(t, root, "add", "--session", "s2", "pkg/a.go"); code != 3 {
						t.Fatalf("unknown holder must block foreign claim: code=%d stderr=%s", code, errOut)
					}
				})
			}
		}
	}
}

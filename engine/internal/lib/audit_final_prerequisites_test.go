package lib

import "testing"

func TestAuditFinalReconciliationRequiresSameSliceSubject(t *testing.T) {
	for _, role := range []string{"code-reviewer", "proof-runner"} {
		for _, tc := range []struct {
			name       string
			old, fresh []string
			want       bool
		}{
			{name: "unrelated slice", old: []string{"SLICE-001"}, fresh: []string{"SLICE-002"}},
			{name: "unordered exact set", old: []string{"SLICE-001", "SLICE-002"}, fresh: []string{"SLICE-002", "SLICE-001"}, want: true},
			{name: "normalized set", old: []string{"SLICE-001", "SLICE-002"}, fresh: []string{"SLICE-2", "SLICE-01", "SLICE-001"}, want: true},
			{name: "subset", old: []string{"SLICE-001", "SLICE-002"}, fresh: []string{"SLICE-001"}},
			{name: "superset", old: []string{"SLICE-001"}, fresh: []string{"SLICE-001", "SLICE-002"}},
			{name: "whole feature replaced by slice", fresh: []string{"SLICE-001"}},
			{name: "slice replaced by whole feature", old: []string{"SLICE-001"}},
			{name: "nil and empty whole feature", fresh: []string{}, want: true},
		} {
			t.Run(role+"/"+tc.name, func(t *testing.T) {
				account := &dispatchRole{Handle: "old-host", StartedAt: "2026-01-01T00:00:00Z"}
				old := &dispatchWave{Phase: "Build", State: "abandoned", Slices: tc.old, Roles: map[string]*dispatchRole{role: account}}
				fresh := &dispatchWave{Phase: "build", State: "complete", Slices: tc.fresh, Roles: map[string]*dispatchRole{role: {
					Handle: "fresh-host", StartedAt: "2026-01-01T00:00:02Z", Returned: true, ReturnedAt: "2026-01-01T00:00:03Z",
				}}}
				ledger := &dispatchFile{Waves: map[string]*dispatchWave{"old": old, "fresh": fresh}}
				if got := reconciledReviewer(ledger, old, role, account); got != tc.want {
					t.Fatalf("reconciledReviewer() = %t, want %t (old=%v fresh=%v)", got, tc.want, tc.old, tc.fresh)
				}
			})
		}
	}
}

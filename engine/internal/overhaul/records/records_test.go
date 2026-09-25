package records

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/overhaul/ovio"
	"github.com/devrites/devrites/internal/overhaul/render"
)

type J = map[string]any
type L = []any

type fixture struct {
	t         *testing.T
	repo, run string
}

func write(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b []byte
	switch x := v.(type) {
	case string:
		b = []byte(x)
	default:
		var err error
		if b, err = json.MarshalIndent(v, "", "  "); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func digest(t *testing.T, path string) string {
	t.Helper()
	d, err := ovio.SHA256File(path)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func task(id string, deps L, path string) J {
	return J{"id": id, "deps": deps, "allowed_paths": L{path}, "oracle": "test passes",
		"writer_role": "implementer", "verifier_role": "verifier", "controls": L{"C-1"}}
}

func newFixture(t *testing.T) *fixture {
	dir := t.TempDir()
	f := &fixture{t: t, repo: filepath.Join(dir, "repo"), run: filepath.Join(dir, "run")}
	write(t, filepath.Join(f.repo, "src/a.py"), "print('a')\n")
	write(t, filepath.Join(f.run, "revisions/rubric-r1.json"), J{"schema": "overhaul.rubric/1", "rev": 1, "controls": L{J{"id": "C-1"}}})
	f.writePlan(L{task("T-1", L{}, "src/a.py"), task("T-2", L{"T-1"}, "src/b.py")})
	write(t, filepath.Join(f.run, "evidence/red.log"), "red\n")
	write(t, filepath.Join(f.run, "receipts/R-1.json"), J{})
	return f
}

func (f *fixture) writePlan(tasks L) {
	write(f.t, filepath.Join(f.run, "revisions/plan-r1.json"), J{"schema": "overhaul.plan/1", "rev": 1,
		"rubric": J{"file": "revisions/rubric-r1.json", "sha256": digest(f.t, filepath.Join(f.run, "revisions/rubric-r1.json"))},
		"tasks":  tasks})
}

// records returns a fresh, valid record set with digests of the current files.
func (f *fixture) records() map[string]J {
	return map[string]J{
		"run.json": {"schema": "overhaul.run/1", "run_id": "run-1", "repo_root": f.repo, "mode": "full",
			"assessment_only": false, "phase": "REMEDIATING", "readiness_verdict": "NOT_ASSESSED",
			"execution_outcome": "RUNNING", "revisions": J{"plan": J{"rev": 1, "file": "revisions/plan-r1.json"}}},
		"coverage.json": {"schema": "overhaul.coverage/1", "files": L{J{"path": "src/a.py", "eligible": true}}},
		"findings.json": {"schema": "overhaul.findings/1", "findings": L{J{"id": "F-1", "fingerprint": "fp-1",
			"kind": "defect", "scope_origin": "pre-existing", "status": "confirmed", "severity": "high",
			"evidence": L{"E-1"}, "locations": L{J{"path": "src/a.py", "start": 1, "end": 1},
				J{"path": "mobile-app:src/api.ts", "external": true}},
			"history": L{J{"status": "candidate"}, J{"status": "confirmed"}}}}},
		"evidence.json": {"schema": "overhaul.evidence/1", "items": L{J{"id": "E-1", "path": "evidence/red.log",
			"sha256": digest(f.t, filepath.Join(f.run, "evidence/red.log"))}}},
		"dispatch.json": {"schema": "overhaul.dispatch/1", "attempts": L{J{"task_id": "T-1", "attempt_id": "A1",
			"role": "implementer", "status": "admitted", "authorized_by": "AP-1", "receipt": "receipts/R-1.json",
			"changed_paths": L{"src/a.py"}}}},
		"approval.json": {"schema": "overhaul.approval/1", "events": L{J{"id": "AP-1", "kind": "approve",
			"status": "active", "tasks": L{"T-1"}, "plan_rev": 1, "quote": "yes, fix T-1", "channel": "conversation",
			"plan_digest": digest(f.t, filepath.Join(f.run, "revisions/plan-r1.json"))}}},
		"cycles.json": {"schema": "overhaul.cycles/1", "cycles": L{J{"n": 1}}},
	}
}

func (f *fixture) cmd(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// publish stages, writes recs (plus extra files relative to the stage) and publishes.
func (f *fixture) publish(recs map[string]J, extra map[string]string) (int, string) {
	f.t.Helper()
	code, out, errs := f.cmd("stage", f.run)
	if code != 0 {
		f.t.Fatalf("stage: %d %s", code, errs)
	}
	tmp := strings.TrimSpace(out)
	for name, doc := range recs {
		write(f.t, filepath.Join(tmp, name), doc)
	}
	for name, body := range extra {
		write(f.t, filepath.Join(tmp, name), body)
	}
	code, out, errs = f.cmd("publish", f.run)
	return code, out + errs
}

func (f *fixture) mustPublish(recs map[string]J) {
	f.t.Helper()
	if code, out := f.publish(recs, nil); code != 0 {
		f.t.Fatalf("publish failed: %d\n%s", code, out)
	}
}

func finding(r map[string]J) J  { return r["findings.json"]["findings"].(L)[0].(J) }
func approval(r map[string]J) J { return r["approval.json"]["events"].(L)[0].(J) }
func attempts(r map[string]J) L { return r["dispatch.json"]["attempts"].(L) }

func addAttempt(r map[string]J, a J) {
	r["dispatch.json"]["attempts"] = append(attempts(r), a)
}

func TestPublishThenValidate(t *testing.T) {
	f := newFixture(t)
	code, out := f.publish(f.records(), nil)
	if code != 0 || !strings.Contains(out, "published g0001") {
		t.Fatalf("publish: %d %s", code, out)
	}
	if code, out, _ := f.cmd("validate", f.run); code != 0 || strings.TrimSpace(out) != "valid g0001" {
		t.Fatalf("validate: %d %s", code, out)
	}
	f.mustPublish(f.records())
	m, err := ovio.LoadObject(filepath.Join(f.run, "g0002/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, hasRun := ovio.Obj(m["files"])["run.json"]
	_, hasPlan := ovio.Obj(m["revisions"])["revisions/plan-r1.json"]
	if m["previous"] != "g0001" || !hasRun || !hasPlan {
		t.Fatalf("manifest: %v", m)
	}
	if code, out, _ := f.cmd("validate", f.run); code != 0 || strings.TrimSpace(out) != "valid g0002" {
		t.Fatalf("validate g0002: %d %s", code, out)
	}
	if code, out, _ := f.cmd("digest", filepath.Join(f.run, "evidence/red.log")); code != 0 ||
		strings.TrimSpace(out) != digest(t, filepath.Join(f.run, "evidence/red.log")) {
		t.Fatalf("digest: %d %s", code, out)
	}
	if code, _, _ := f.cmd("bogus"); code != 2 {
		t.Fatalf("usage exit = %d", code)
	}
}

func TestSingleGenerationRules(t *testing.T) {
	verifiedHistory := L{J{"status": "candidate"}, J{"status": "confirmed"}, J{"status": "approved-for-fix"},
		J{"status": "in-progress"}, J{"status": "fixed-unverified"}, J{"status": "verified"}}
	verifier := J{"task_id": "T-1", "attempt_id": "V1", "role": "verifier", "status": "admitted", "receipt": "receipts/R-1.json"}
	cases := []struct {
		name   string
		mutate func(f *fixture, r map[string]J)
		want   string // "" means the publish must succeed
	}{
		{"approval without quote", func(f *fixture, r map[string]J) { delete(approval(r), "quote") }, "needs the user's verbatim decision"},
		{"approval untrusted channel", func(f *fixture, r map[string]J) { approval(r)["channel"] = "email" }, "trusted channel"},
		{"accept-degradation without quote", func(f *fixture, r map[string]J) {
			r["approval.json"]["events"] = append(r["approval.json"]["events"].(L), J{"id": "D-1", "kind": "accept-degradation", "channel": "conversation"})
		}, "approval D-1: needs the user's verbatim decision"},
		{"approval digest mismatch", func(f *fixture, r map[string]J) { approval(r)["plan_digest"] = strings.Repeat("0", 64) }, "plan digest does not match"},
		{"subset approval missing prerequisite", func(f *fixture, r map[string]J) { approval(r)["tasks"] = L{"T-2"} }, "miss prerequisites ['T-1']"},
		{"writer for unapproved task", func(f *fixture, r map[string]J) {
			addAttempt(r, J{"task_id": "T-2", "attempt_id": "A1", "role": "implementer", "status": "running", "authorized_by": "AP-1"})
		}, "attempt T-2#A1: writer task is not in the active approval"},
		{"approval active after stop", func(f *fixture, r map[string]J) {
			r["approval.json"]["events"] = append(r["approval.json"]["events"].(L), J{"id": "S-1", "kind": "stop"})
		}, "still active after a later stop/revoke"},
		{"live writer after stop", func(f *fixture, r map[string]J) {
			approval(r)["status"] = "stopped"
			r["approval.json"]["events"] = append(r["approval.json"]["events"].(L), J{"id": "S-1", "kind": "stop"})
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A2", "role": "implementer", "status": "running", "authorized_by": "AP-1"})
		}, "attempt T-1#A2: writer task is not in the active approval"},
		{"assessment-only with approval", func(f *fixture, r map[string]J) {
			r["run.json"]["assessment_only"] = true
			r["dispatch.json"]["attempts"] = L{}
		}, "assessment-only run carries an active approval"},
		{"assessment-only with writer", func(f *fixture, r map[string]J) {
			r["run.json"]["assessment_only"] = true
			approval(r)["status"] = "revoked"
		}, "writer dispatched in an assessment-only run"},
		{"live writers case alias", func(f *fixture, r map[string]J) {
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A2", "role": "implementer", "status": "running", "authorized_by": "AP-1", "write_paths": L{"src/a.py"}})
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A3", "role": "implementer", "status": "unknown", "authorized_by": "AP-1", "write_paths": L{"SRC/A.PY"}})
		}, "single writer: T-1#A2 and T-1#A3 both hold SRC/A.PY"},
		{"live writers symlink alias", func(f *fixture, r map[string]J) {
			if err := os.Symlink("a.py", filepath.Join(f.repo, "src/link.py")); err != nil {
				f.t.Fatal(err)
			}
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A2", "role": "implementer", "status": "running", "authorized_by": "AP-1", "write_paths": L{"src/a.py"}})
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A3", "role": "implementer", "status": "dispatched", "authorized_by": "AP-1", "write_paths": L{"src/link.py"}})
		}, "single writer: T-1#A2 and T-1#A3 both hold src/link.py"},
		{"settled writers may share a path", func(f *fixture, r map[string]J) {
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A2", "role": "implementer", "status": "running", "authorized_by": "AP-1", "write_paths": L{"src/a.py"}})
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A3", "role": "implementer", "status": "cancelled", "authorized_by": "AP-1", "write_paths": L{"src/a.py"}})
		}, ""},
		{"allowed path escape", func(f *fixture, r map[string]J) {
			f.writePlan(L{task("T-1", L{}, "../outside.py"), task("T-2", L{"T-1"}, "src/b.py")})
			approval(r)["plan_digest"] = digest(f.t, filepath.Join(f.run, "revisions/plan-r1.json"))
		}, "allowed path '../outside.py' not normalized or escapes the repository"},
		{"dependency cycle", func(f *fixture, r map[string]J) {
			f.writePlan(L{task("T-1", L{"T-2"}, "src/a.py"), task("T-2", L{"T-1"}, "src/b.py")})
			approval(r)["plan_digest"] = digest(f.t, filepath.Join(f.run, "revisions/plan-r1.json"))
			approval(r)["tasks"] = L{"T-1", "T-2"}
		}, "task dependencies contain a cycle"},
		{"bracket path accepted", func(f *fixture, r map[string]J) {
			f.writePlan(L{task("T-1", L{}, "src/a.py"), task("T-2", L{"T-1"}, "app/blog/[slug]/page.tsx")})
			approval(r)["plan_digest"] = digest(f.t, filepath.Join(f.run, "revisions/plan-r1.json"))
			finding(r)["locations"] = L{J{"path": "app/blog/[slug]/page.tsx", "start": 1}}
		}, ""},
		{"duplicate fingerprint", func(f *fixture, r map[string]J) {
			r["findings.json"]["findings"] = append(r["findings.json"]["findings"].(L), J{"id": "F-2", "fingerprint": "fp-1",
				"kind": "defect", "scope_origin": "introduced", "status": "candidate", "history": L{J{"status": "candidate"}}})
		}, "finding fingerprint: duplicate id 'fp-1'"},
		{"needs-validation without blockers", func(f *fixture, r map[string]J) {
			r["findings.json"]["findings"] = append(r["findings.json"]["findings"].(L), J{"id": "F-2", "fingerprint": "fp-2",
				"kind": "defect", "scope_origin": "introduced", "status": "needs-validation",
				"history": L{J{"status": "candidate"}, J{"status": "needs-validation"}}})
		}, "finding F-2: needs-validation must name the missing fact (blockers)"},
		{"host path location", func(f *fixture, r map[string]J) {
			finding(r)["locations"] = L{J{"path": "/Users/me/app/a.py", "start": 1}}
		}, "location '/Users/me/app/a.py' is not a repository-relative path or label"},
		{"external host path location", func(f *fixture, r map[string]J) {
			finding(r)["locations"] = L{J{"path": "~/app/a.py", "external": true}}
		}, "is not a repository-relative path or label"},
		{"directory location", func(f *fixture, r map[string]J) {
			finding(r)["locations"] = L{J{"path": "src"}}
		}, "location 'src' is not a repository-relative path or label"},
		{"illegal transition", func(f *fixture, r map[string]J) {
			finding(r)["history"] = L{J{"status": "candidate"}, J{"status": "verified"}, J{"status": "confirmed"}}
		}, "illegal transition candidate -> verified"},
		{"severity on candidate", func(f *fixture, r map[string]J) {
			finding(r)["status"] = "candidate"
			finding(r)["history"] = L{J{"status": "candidate"}}
		}, "severity only for established impact"},
		{"history must start at candidate", func(f *fixture, r map[string]J) {
			finding(r)["history"] = L{J{"status": "confirmed"}}
		}, "must start at candidate"},
		{"same-status entry with reason", func(f *fixture, r map[string]J) {
			finding(r)["history"] = L{J{"status": "candidate"}, J{"status": "confirmed"}, J{"status": "confirmed", "reason": "severity re-assessed"}}
		}, ""},
		{"same-status entry without reason", func(f *fixture, r map[string]J) {
			finding(r)["history"] = L{J{"status": "candidate"}, J{"status": "confirmed"}, J{"status": "confirmed"}}
		}, "illegal transition confirmed -> confirmed"},
		{"evidence sha256 mandatory", func(f *fixture, r map[string]J) {
			delete(r["evidence.json"]["items"].(L)[0].(J), "sha256")
		}, "evidence E-1: sha256 missing"},
		{"evidence outside run dir", func(f *fixture, r map[string]J) {
			r["evidence.json"]["items"].(L)[0].(J)["path"] = "../repo/src/a.py"
		}, "evidence E-1: path missing or outside run dir"},
		{"verified_by without verifier", func(f *fixture, r map[string]J) {
			finding(r)["status"], finding(r)["history"], finding(r)["verified_by"] = "verified", verifiedHistory, "T-1#V1"
		}, "verified_by must name an admitted verifier attempt"},
		{"verified_by names attempt", func(f *fixture, r map[string]J) {
			finding(r)["status"], finding(r)["history"], finding(r)["verified_by"] = "verified", verifiedHistory, "T-1#V1"
			addAttempt(r, verifier)
		}, ""},
		{"verified_by names receipt", func(f *fixture, r map[string]J) {
			finding(r)["status"], finding(r)["history"], finding(r)["verified_by"] = "verified", verifiedHistory, "receipts/R-1.json"
			addAttempt(r, verifier)
		}, ""},
		{"plan file must match rev", func(f *fixture, r map[string]J) {
			r["run.json"]["revisions"] = J{"plan": J{"rev": 1, "file": "revisions/rubric-r1.json"}}
		}, "must be revisions/plan-r<rev>.json"},
		{"approvals need revisions.plan", func(f *fixture, r map[string]J) {
			r["run.json"]["revisions"] = J{}
			r["dispatch.json"]["attempts"] = L{}
		}, "approvals and writers need revisions.plan"},
		{"writers need revisions.plan", func(f *fixture, r map[string]J) {
			r["run.json"]["revisions"] = J{}
			r["approval.json"]["events"] = L{}
		}, "approvals and writers need revisions.plan"},
		{"assessment without plan is fine", func(f *fixture, r map[string]J) {
			r["run.json"]["revisions"] = J{}
			r["run.json"]["assessment_only"] = true
			r["approval.json"]["events"] = L{}
			r["dispatch.json"]["attempts"] = L{}
		}, ""},
		{"terminal outcome missing", func(f *fixture, r map[string]J) {
			r["run.json"]["execution_outcome"] = "READY_FOR_USER_REVIEW"
		}, "approved tasks without exactly one recorded outcome: ['T-1']"},
		{"terminal outcome recorded", func(f *fixture, r map[string]J) {
			r["run.json"]["execution_outcome"] = "READY_FOR_USER_REVIEW"
			r["run.json"]["task_outcomes"] = J{"T-1": "completed"}
		}, ""},
		{"unsettled attempt blocks leaving RUNNING", func(f *fixture, r map[string]J) {
			r["run.json"]["execution_outcome"] = "STOPPED_BY_USER"
			r["run.json"]["task_outcomes"] = J{"T-1": "completed"}
			addAttempt(r, J{"task_id": "R-9", "attempt_id": "A1", "role": "reviewer", "status": "returned", "receipt": "receipts/R-1.json"})
		}, "before the run leaves RUNNING: ['R-9#A1']"},
		{"unknown and timed-out allowed after stop", func(f *fixture, r map[string]J) {
			approval(r)["status"] = "stopped"
			r["approval.json"]["events"] = append(r["approval.json"]["events"].(L), J{"id": "S-1", "kind": "stop"})
			r["run.json"]["execution_outcome"] = "STOPPED_BY_USER"
			r["run.json"]["task_outcomes"] = J{"T-1": "not-started"}
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A2", "role": "implementer", "status": "unknown", "authorized_by": "AP-1"})
			addAttempt(r, J{"task_id": "R-9", "attempt_id": "A1", "role": "reviewer", "status": "timed-out"})
		}, ""},
		{"scorecard needs revisions.rubric", func(f *fixture, r map[string]J) {
			r["scorecard.json"] = J{"readiness_verdict": "NOT_ASSESSED"}
		}, "scorecard needs revisions.rubric"},
		{"scorecard rubric digest", func(f *fixture, r map[string]J) {
			r["scorecard.json"] = J{"readiness_verdict": "NOT_ASSESSED"}
			r["run.json"]["revisions"].(J)["rubric"] = J{"rev": 1, "file": "revisions/rubric-r1.json", "sha256": strings.Repeat("0", 64)}
		}, "revisions.rubric digest mismatch"},
		{"scorecard verdict differs", func(f *fixture, r map[string]J) {
			r["scorecard.json"] = J{"readiness_verdict": "NOT_READY"}
			r["run.json"]["revisions"].(J)["rubric"] = J{"rev": 1, "file": "revisions/rubric-r1.json",
				"sha256": digest(f.t, filepath.Join(f.run, "revisions/rubric-r1.json"))}
		}, "readiness_verdict differs"},
		{"scorecard consistent", func(f *fixture, r map[string]J) {
			r["scorecard.json"] = J{"readiness_verdict": "NOT_ASSESSED"}
			r["run.json"]["revisions"].(J)["rubric"] = J{"rev": 1, "file": "revisions/rubric-r1.json",
				"sha256": digest(f.t, filepath.Join(f.run, "revisions/rubric-r1.json"))}
		}, ""},
		{"schema tag and enum", func(f *fixture, r map[string]J) {
			r["cycles.json"]["schema"] = "overhaul.cycle/1"
			r["run.json"]["mode"] = "yolo"
		}, "run.json: mode='yolo' not in ['branch', 'full', 'pr']"},
		{"ineligible coverage without reason", func(f *fixture, r map[string]J) {
			r["coverage.json"]["files"] = L{J{"path": "src/a.py", "eligible": false}}
		}, "coverage src/a.py: ineligible without an exclusion reason"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			r := f.records()
			c.mutate(f, r)
			code, out := f.publish(r, nil)
			if c.want == "" {
				if code != 0 {
					t.Fatalf("want publish ok, got %d:\n%s", code, out)
				}
				return
			}
			if code != 1 || !strings.Contains(out, "VIOLATION: ") || !strings.Contains(out, c.want) {
				t.Fatalf("want exit 1 with %q, got %d:\n%s", c.want, code, out)
			}
		})
	}
}

func TestCrossGenerationRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(f *fixture, r map[string]J)
		want   string
	}{
		{"cycles append-only", func(f *fixture, r map[string]J) { r["cycles.json"]["cycles"] = L{J{"n": 2}} }, "cycles.json is append-only"},
		{"cycles may grow", func(f *fixture, r map[string]J) {
			r["cycles.json"]["cycles"] = L{J{"n": 1}, J{"n": 2}}
		}, ""},
		{"finding removed", func(f *fixture, r map[string]J) { r["findings.json"]["findings"] = L{} }, "finding F-1: removed or history rewritten"},
		{"history rewritten", func(f *fixture, r map[string]J) {
			finding(r)["history"] = L{J{"status": "candidate", "reason": "edited"}, J{"status": "confirmed"}}
		}, "finding F-1: removed or history rewritten"},
		{"severity change without history", func(f *fixture, r map[string]J) { finding(r)["severity"] = "medium" }, "severity or kind changed without a history entry"},
		{"severity change with history", func(f *fixture, r map[string]J) {
			finding(r)["severity"] = "medium"
			finding(r)["history"] = L{J{"status": "candidate"}, J{"status": "confirmed"}, J{"status": "confirmed", "reason": "re-assessed"}}
		}, ""},
		{"approval events append-only", func(f *fixture, r map[string]J) { approval(r)["quote"] = "something else" }, "approval.json events are append-only"},
		{"approval status may change", func(f *fixture, r map[string]J) { approval(r)["status"] = "revoked" }, ""},
		{"immutable revision changed", func(f *fixture, r map[string]J) {
			write(f.t, filepath.Join(f.run, "revisions/rubric-r1.json"), J{"schema": "overhaul.rubric/1", "rev": 1, "controls": L{}})
		}, "immutable revision changed: revisions/rubric-r1.json"},
		{"run_id pinned", func(f *fixture, r map[string]J) { r["run.json"]["run_id"] = "run-2" }, "run_id changed"},
		{"mode pinned", func(f *fixture, r map[string]J) { r["run.json"]["mode"] = "pr" }, "mode changed"},
		{"repo_root pinned", func(f *fixture, r map[string]J) {
			r["run.json"]["repo_root"] = filepath.Join(f.repo, "src")
		}, "repo_root changed"},
		{"assessment_only pinned", func(f *fixture, r map[string]J) {
			r["run.json"]["assessment_only"] = true
			approval(r)["status"] = "revoked"
			r["dispatch.json"]["attempts"] = L{}
		}, "assessment_only changed"},
		{"coverage denominator shrank", func(f *fixture, r map[string]J) { r["coverage.json"]["files"] = L{} }, "coverage denominator shrank: ['src/a.py']"},
		{"coverage exclusion keeps denominator", func(f *fixture, r map[string]J) {
			r["coverage.json"]["files"] = L{J{"path": "src/a.py", "eligible": false, "exclusion": J{"reason": "generated"}}}
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			f.mustPublish(f.records())
			r := f.records()
			c.mutate(f, r)
			code, out := f.publish(r, nil)
			if c.want == "" {
				if code != 0 {
					t.Fatalf("want publish ok, got %d:\n%s", code, out)
				}
				return
			}
			if code != 1 || !strings.Contains(out, c.want) {
				t.Fatalf("want exit 1 with %q, got %d:\n%s", c.want, code, out)
			}
		})
	}
}

func TestStaleStage(t *testing.T) {
	f := newFixture(t)
	f.mustPublish(f.records())
	if code, _, errs := f.cmd("stage", f.run); code != 0 {
		t.Fatal(errs)
	}
	write(t, filepath.Join(f.run, "CURRENT"), "g0000\n")
	code, _, errs := f.cmd("publish", f.run)
	if code != 1 || !strings.Contains(errs, "stale stage: CURRENT moved from g0001 to g0000") {
		t.Fatalf("got %d %s", code, errs)
	}
	if code, _, errs := f.cmd("stage", f.run); code != 2 || !strings.Contains(errs, "g0001 exists but CURRENT is g0000: interrupted publish") {
		t.Fatalf("restage: %d %s", code, errs)
	}
}

func TestStageCopiesWithoutViewsOrManifest(t *testing.T) {
	f := newFixture(t)
	stamp := "overhaul-stamp run=run-1 generation=g0001 plan=r1:" + digest(t, filepath.Join(f.run, "revisions/plan-r1.json"))
	if code, out := f.publish(f.records(), map[string]string{"views/review.md": "# Review\n" + stamp + "\n"}); code != 0 {
		t.Fatal(out)
	}
	code, out, errs := f.cmd("stage", f.run)
	if code != 0 {
		t.Fatal(errs)
	}
	tmp := strings.TrimSpace(out)
	if filepath.Clean(tmp) != filepath.Join(f.run, "g0002.tmp") {
		t.Fatalf("stage path %q", tmp)
	}
	base, _ := os.ReadFile(filepath.Join(tmp, ".base"))
	if string(base) != "g0001" || isDir(filepath.Join(tmp, "views")) || isFile(filepath.Join(tmp, "manifest.json")) || !isFile(filepath.Join(tmp, "run.json")) {
		t.Fatalf("bad stage contents (base %q)", base)
	}
	if code, _, errs := f.cmd("stage", f.run); code != 2 || !strings.Contains(errs, "staged generation already exists") {
		t.Fatalf("double stage: %d %s", code, errs)
	}
}

func TestTamperedGenerationDetected(t *testing.T) {
	f := newFixture(t)
	f.mustPublish(f.records())
	p := filepath.Join(f.run, "g0001/cycles.json")
	b, _ := os.ReadFile(p)
	write(t, p, string(b)+" ")
	code, out, _ := f.cmd("validate", f.run)
	if code != 1 || !strings.Contains(out, "VIOLATION: mixed or modified generation: cycles.json does not match manifest") {
		t.Fatalf("got %d %s", code, out)
	}
}

func TestViewStamp(t *testing.T) {
	f := newFixture(t)
	code, out := f.publish(f.records(), map[string]string{"views/review.md": "overhaul-stamp run=run-1 generation=g0000 plan=none\n"})
	if code != 1 || !strings.Contains(out, "view review.md: stamp does not match this generation") {
		t.Fatalf("got %d %s", code, out)
	}
}

// Views from the renderer must carry the stamp this validator expects.
func TestRenderedViewsPassValidation(t *testing.T) {
	f := newFixture(t)
	code, out, errs := f.cmd("stage", f.run)
	if code != 0 {
		t.Fatalf("stage: %d %s", code, errs)
	}
	tmp := strings.TrimSpace(out)
	for name, doc := range f.records() {
		write(t, filepath.Join(tmp, name), doc)
	}
	for _, kind := range []string{"review", "report"} {
		var o, e bytes.Buffer
		if c := render.Run([]string{f.run, tmp, kind}, &o, &e); c != 0 {
			t.Fatalf("render %s: %d %s", kind, c, e.String())
		}
	}
	if code, out, errs := f.cmd("publish", f.run); code != 0 {
		t.Fatalf("publish: %d %s%s", code, out, errs)
	}
}

func TestInterruptedPublish(t *testing.T) {
	f := newFixture(t)
	f.mustPublish(f.records())
	if err := os.Mkdir(filepath.Join(f.run, "g0002"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, _, errs := f.cmd("stage", f.run)
	if code != 2 || !strings.Contains(errs, "interrupted publish") {
		t.Fatalf("stage: %d %s", code, errs)
	}
	code, out, _ := f.cmd("validate", f.run)
	if code != 0 || !strings.Contains(out, "NOTE: g0002 is newer than CURRENT") || !strings.Contains(out, "valid g0001") {
		t.Fatalf("validate: %d %s", code, out)
	}
}

func TestPublishNeedsExactlyOneStage(t *testing.T) {
	f := newFixture(t)
	if err := os.MkdirAll(filepath.Join(f.run, "revisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, _, errs := f.cmd("publish", f.run)
	if code != 2 || !strings.Contains(errs, "expected exactly one staged generation, found []") {
		t.Fatalf("got %d %s", code, errs)
	}
}

func TestBadPath(t *testing.T) {
	repo := t.TempDir()
	outside := t.TempDir()
	write(t, filepath.Join(repo, "src/a.py"), "x\n")
	write(t, filepath.Join(outside, "secret.py"), "x\n")
	if err := os.Symlink(filepath.Join(outside, "secret.py"), filepath.Join(repo, "src/out.py")); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		path any
		want string
	}{
		{"src/a.py", ""},
		{"src/new.py", ""},
		{"app/blog/[slug]/page.tsx", ""},
		{"src/*.py", "not an exact relative file path"},
		{"src/a?.py", "not an exact relative file path"},
		{"/etc/passwd", "not an exact relative file path"},
		{"src/", "not an exact relative file path"},
		{"", "not an exact relative file path"},
		{json.Number("3"), "not an exact relative file path"},
		{"../x.py", "not normalized or escapes the repository"},
		{"src/../src/a.py", "not normalized or escapes the repository"},
		{"./src/a.py", "not normalized or escapes the repository"},
		{"src", "names a directory or git metadata"},
		{".git/config", "names a directory or git metadata"},
		{"src/.git/HEAD", "names a directory or git metadata"},
		{"src/out.py", "resolves outside the repository (symlink or traversal)"},
	}
	for _, c := range cases {
		if got := BadPath(c.path, repo); got != c.want {
			t.Errorf("BadPath(%v) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestRepr(t *testing.T) {
	cases := map[string]any{
		"None": nil, "True": true, "'a'": "a", `"it's"`: "it's", "['a', 'b']": L{"a", "b"},
		"3": json.Number("3"), `'a\\b'`: `a\b`,
	}
	for want, v := range cases {
		if got := Repr(v); got != want {
			t.Errorf("Repr(%v) = %s, want %s", v, got, want)
		}
	}
}

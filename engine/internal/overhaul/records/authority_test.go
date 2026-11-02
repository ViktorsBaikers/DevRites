package records

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/overhaul/ovio"
	"github.com/devrites/devrites/internal/overhaul/score"
)

// calculatedStage uses the public calculator, including its exact input digests.
func calculatedStage(t *testing.T, f *fixture, resultsName string) string {
	t.Helper()
	rubricDoc, results := calculatorRubric(), J{}
	for _, item := range rubricDoc["controls"].(L) {
		results[item.(J)["id"].(string)] = J{"status": "PASS", "evidence": L{"E-1"}}
	}
	rubric := filepath.Join(f.run, "revisions/rubric-r1.json")
	write(t, rubric, rubricDoc)
	f.writePlan(L{task("T-1", L{}, "src/a.py")})
	r := f.records()
	r["findings.json"]["findings"] = L{}
	reviewed(f, r, "applicable")
	r["run.json"]["revisions"].(J)["rubric"] = J{"rev": 1, "file": "revisions/rubric-r1.json"}
	r["run.json"]["readiness_verdict"] = "MEETS_TARGETS"
	r[resultsName] = J{"schema": "overhaul.results/1", "subject": "candidate", "fingerprint": strings.Repeat("a", 64),
		"rubric_digest": digest(t, rubric), "results": results}
	gates := J{}
	for _, id := range score.GateIDs(false) {
		if id != "G-THRESHOLDS" {
			gates[id] = J{"status": "PASS", "evidence": L{"E-1"}}
		}
	}
	r["gates.json"] = J{"mode": "audit", "gates": gates}
	code, out, errs := f.cmd("stage", f.run)
	if code != 0 {
		t.Fatalf("stage: %d %s%s", code, out, errs)
	}
	stage := strings.TrimSpace(out)
	for name, doc := range r {
		write(t, filepath.Join(stage, name), doc)
	}
	var stdout, stderr bytes.Buffer
	if code := score.Run([]string{"--rubric", rubric, "--results", filepath.Join(stage, resultsName),
		"--gates", filepath.Join(stage, "gates.json"), "--out", filepath.Join(stage, "scorecard.json")}, &stdout, &stderr); code != 0 {
		t.Fatalf("calculator: %d %s%s", code, stdout.String(), stderr.String())
	}
	return stage
}

func TestScorecardBindsSelectedCalculatorInputs(t *testing.T) {
	for _, resultsName := range []string{"results-baseline.json", "results-candidate.json"} {
		for _, input := range []string{resultsName, "gates.json"} {
			t.Run(resultsName+"/"+input, func(t *testing.T) {
				f := newFixture(t)
				calculatedStage(t, f, resultsName)
				if code, out, errs := f.cmd("publish", f.run); code != 0 {
					t.Fatalf("current calculator inputs rejected: %d %s%s", code, out, errs)
				}
				code, out, errs := f.cmd("stage", f.run)
				if code != 0 {
					t.Fatalf("restage: %d %s%s", code, out, errs)
				}
				path := filepath.Join(strings.TrimSpace(out), input)
				doc, err := ovio.LoadObject(path)
				if err != nil {
					t.Fatal(err)
				}
				if input == "gates.json" {
					ovio.Obj(ovio.Obj(doc["gates"])["G-CATALOG-ADEQUACY"])["status"] = "FAIL"
				} else {
					ovio.Obj(ovio.Obj(doc["results"])["C-1"])["status"] = "FAIL"
				}
				write(t, path, doc)
				if code, out, errs := f.cmd("publish", f.run); code != 1 {
					t.Fatalf("stale calculator proof certified changed FAIL input: %d %s%s", code, out, errs)
				}
			})
		}
	}
}

func TestLiveWriterReservationsCannotDisappear(t *testing.T) {
	for _, omission := range []string{"omitted", "null", "empty"} {
		t.Run(omission, func(t *testing.T) {
			f := newFixture(t)
			r := f.records()
			for _, id := range []string{"A2", "A3"} {
				a := J{"task_id": "T-1", "attempt_id": id, "role": "implementer", "status": "running", "authorized_by": "AP-1"}
				switch omission {
				case "null":
					a["write_paths"] = nil
				case "empty":
					a["write_paths"] = L{}
				}
				addAttempt(r, a)
			}
			if code, out := f.publish(r, nil); code != 1 {
				t.Fatalf("same-task live writers lost authoritative reservations: %d %s", code, out)
			}
		})
	}
	t.Run("disjoint approved task scopes", func(t *testing.T) {
		f := newFixture(t)
		f.writePlan(L{task("T-1", L{}, "src/a.py"), task("T-2", L{}, "src/b.py")})
		r := f.records()
		approval(r)["tasks"] = L{"T-1", "T-2"}
		addAttempt(r, J{"task_id": "T-1", "attempt_id": "A2", "role": "implementer", "status": "running",
			"authorized_by": "AP-1", "write_paths": L{"src/a.py"}})
		addAttempt(r, J{"task_id": "T-2", "attempt_id": "A1", "role": "implementer", "status": "running",
			"authorized_by": "AP-1", "write_paths": L{"src/b.py"}})
		if code, out := f.publish(r, nil); code != 0 {
			t.Fatalf("disjoint approved writers rejected: %d %s", code, out)
		}
	})
}

func TestApplicabilityProofRequiresAdmittedLane(t *testing.T) {
	for _, lane := range []string{"backend", "frontend"} {
		t.Run(lane, func(t *testing.T) {
			f := newFixture(t)
			r := f.records()
			reviewed(f, r, "applicable")
			a := attempts(r)[1].(J)
			a["lane"], a["snapshot"] = "backend", "alpha"
			write(t, filepath.Join(f.run, "receipts/R-dom.json"), J{"schema": "overhaul.receipt/1", "run_id": "run-1",
				"task_id": "W-1", "attempt_id": "A1", "role": "reviewer", "lane": "backend", "snapshot": "alpha",
				"started_at": "2026-01-01T00:00:00Z", "finished_at": "2026-01-01T00:01:00Z", "outcome": "no-findings",
				"domains_checked": L{"correctness"}, "inspected": L{J{"path": "src/a.py", "start": 1, "end": 1}}})
			cell(r)["lane"] = lane
			scorecard(f, r, J{"G-COVERAGE": "PASS"})
			code, out := f.publish(r, nil)
			if lane == "backend" && code != 0 {
				t.Fatalf("correct-lane receipt rejected: %d %s", code, out)
			}
			if lane == "frontend" && code != 1 {
				t.Fatalf("backend receipt closed frontend cell: %d %s", code, out)
			}
		})
	}
}

func TestPublishedAdmissionReceiptCannotBeReplaced(t *testing.T) {
	for _, operation := range []string{"validate", "publish", "publish caller replacement digest", "retained admission"} {
		t.Run(operation, func(t *testing.T) {
			f := newFixture(t)
			r := f.records()
			path := filepath.Join(f.run, "receipts/R-1.json")
			rc := J{"schema": "overhaul.receipt/1", "run_id": "run-1", "task_id": "T-1", "attempt_id": "A1",
				"role": "implementer", "lane": "backend", "snapshot": "alpha", "outcome": "patch",
				"started_at": "2026-01-01T00:00:00Z", "finished_at": "2026-01-01T00:01:00Z",
				"domains_checked": L{"correctness"}, "changed_paths": L{"src/a.py"}}
			attempts(r)[0].(J)["lane"], attempts(r)[0].(J)["snapshot"] = "backend", "alpha"
			write(t, path, rc)
			f.mustPublish(r)
			if operation == "retained admission" {
				r["dispatch.json"]["attempts"] = L{}
				f.mustPublish(r)
			}
			rc["changed_paths"] = L{"src/b.py"}
			write(t, path, rc)
			if operation == "validate" || operation == "retained admission" {
				if code, out, errs := f.cmd("validate", f.run); code != 1 {
					t.Fatalf("published receipt replacement escaped validation: %d %s%s", code, out, errs)
				}
				return
			}
			if operation == "publish caller replacement digest" {
				attempts(r)[0].(J)["receipt_sha256"] = digest(t, path)
			}
			if code, out := f.publish(r, nil); code != 1 {
				t.Fatalf("next generation laundered replaced admitted receipt: %d %s", code, out)
			}
		})
	}
}

func TestScorecardRequiresCurrentCalculatorBindings(t *testing.T) {
	for _, field := range []string{"rubric_sha256", "results_sha256", "gates_sha256"} {
		t.Run(field, func(t *testing.T) {
			f := newFixture(t)
			stage := calculatedStage(t, f, "results-candidate.json")
			path := filepath.Join(stage, "scorecard.json")
			card, err := ovio.LoadObject(path)
			if err != nil {
				t.Fatal(err)
			}
			delete(ovio.Obj(card["inputs"]), field)
			write(t, path, card)
			if code, out, errs := f.cmd("publish", f.run); code != 1 {
				t.Fatalf("missing calculator binding accepted: %d %s%s", code, out, errs)
			}
		})
	}
	t.Run("candidate selected over baseline", func(t *testing.T) {
		f := newFixture(t)
		stage := calculatedStage(t, f, "results-candidate.json")
		baseline, err := ovio.LoadObject(filepath.Join(stage, "results-candidate.json"))
		if err != nil {
			t.Fatal(err)
		}
		ovio.Obj(ovio.Obj(baseline["results"])["C-1"])["status"] = "FAIL"
		write(t, filepath.Join(stage, "results-baseline.json"), baseline)
		if code, out, errs := f.cmd("publish", f.run); code != 0 {
			t.Fatalf("unselected baseline invalidated current candidate calculator proof: %d %s%s", code, out, errs)
		}
	})
	t.Run("rubric bytes changed before first publication", func(t *testing.T) {
		f := newFixture(t)
		stage := calculatedStage(t, f, "results-candidate.json")
		runPath := filepath.Join(stage, "run.json")
		run, err := ovio.LoadObject(runPath)
		if err != nil {
			t.Fatal(err)
		}
		delete(ovio.Obj(run["revisions"]), "plan")
		write(t, runPath, run)
		write(t, filepath.Join(stage, "approval.json"), J{"schema": "overhaul.approval/1", "events": L{}})
		dispatch, err := ovio.LoadObject(filepath.Join(stage, "dispatch.json"))
		if err != nil {
			t.Fatal(err)
		}
		dispatch["attempts"] = ovio.List(dispatch["attempts"])[1:]
		write(t, filepath.Join(stage, "dispatch.json"), dispatch)
		rubric := calculatorRubric()
		rubric["policy"] = J{"global_min": "10.0"}
		write(t, filepath.Join(f.run, "revisions/rubric-r1.json"), rubric)
		if code, out, errs := f.cmd("publish", f.run); code != 1 {
			t.Fatalf("stale rubric calculator binding accepted: %d %s%s", code, out, errs)
		}
	})
}

func TestLiveWriterUsesWholePhysicalTaskScope(t *testing.T) {
	for _, alias := range []string{"SRC/A.PY", "src/link.py"} {
		t.Run(alias, func(t *testing.T) {
			f := newFixture(t)
			if alias == "src/link.py" {
				if err := os.Symlink("a.py", filepath.Join(f.repo, alias)); err != nil {
					t.Fatal(err)
				}
			}
			f.writePlan(L{task("T-1", L{}, "src/a.py"), task("T-2", L{}, alias)})
			r := f.records()
			approval(r)["tasks"] = L{"T-1", "T-2"}
			for _, id := range []string{"T-1", "T-2"} {
				addAttempt(r, J{"task_id": id, "attempt_id": "A2", "role": "implementer",
					"status": "running", "authorized_by": "AP-1", "write_paths": L{}})
			}
			if code, out := f.publish(r, nil); code != 1 {
				t.Fatalf("physical task aliases escaped reservations: %d %s", code, out)
			}
		})
	}
	for _, scope := range []any{nil, "src/a.py", L{}, L{nil}} {
		t.Run(Repr(scope), func(t *testing.T) {
			f := newFixture(t)
			task := task("T-1", L{}, "src/a.py")
			task["allowed_paths"] = scope
			f.writePlan(L{task})
			r := f.records()
			addAttempt(r, J{"task_id": "T-1", "attempt_id": "A2", "role": "implementer",
				"status": "running", "authorized_by": "AP-1", "write_paths": L{}})
			if code, out := f.publish(r, nil); code != 1 {
				t.Fatalf("missing or malformed authoritative scope accepted: %d %s", code, out)
			}
		})
	}
	t.Run("partial declarations do not split one task", func(t *testing.T) {
		f := newFixture(t)
		scoped := task("T-1", L{}, "src/a.py")
		scoped["allowed_paths"] = L{"src/a.py", "src/b.py"}
		f.writePlan(L{scoped})
		r := f.records()
		for i, path := range []string{"src/a.py", "src/b.py"} {
			addAttempt(r, J{"task_id": "T-1", "attempt_id": []string{"A2", "A3"}[i], "role": "implementer",
				"status": "running", "authorized_by": "AP-1", "write_paths": L{path}})
		}
		if code, out := f.publish(r, nil); code != 1 {
			t.Fatalf("partial claims invented independent ownership: %d %s", code, out)
		}
	})
}

func TestAdmittedReceiptMatchesDispatchIdentity(t *testing.T) {
	for _, field := range []string{"schema", "run_id", "task_id", "attempt_id", "role", "lane", "snapshot"} {
		t.Run(field, func(t *testing.T) {
			f := newFixture(t)
			rc := receipt("T-1", "A1", "implementer", "patch")
			rc[field] = "beta"
			write(t, filepath.Join(f.run, "receipts/R-1.json"), rc)
			if code, out := f.publish(f.records(), nil); code != 1 {
				t.Fatalf("admitted receipt identity mismatch accepted: %d %s", code, out)
			}
		})
	}
}

func TestSettledAdmissionCannotRedirectReceipt(t *testing.T) {
	for _, retained := range []bool{false, true} {
		t.Run(Repr(retained), func(t *testing.T) {
			f := newFixture(t)
			r := f.records()
			f.mustPublish(r)
			if retained {
				r["dispatch.json"]["attempts"] = L{}
				f.mustPublish(r)
				r = f.records()
			}
			rc := receipt("T-1", "A1", "implementer", "gap")
			path := "receipts/beta.json"
			write(t, filepath.Join(f.run, path), rc)
			attempts(r)[0].(J)["receipt"] = path
			attempts(r)[0].(J)["receipt_sha256"] = digest(t, filepath.Join(f.run, path))
			if code, out := f.publish(r, nil); code != 1 {
				t.Fatalf("settled attempt accepted caller replacement path and digest: %d %s", code, out)
			}
		})
	}
}

func TestGenerationWithoutAdmissionBindingsFailsClosed(t *testing.T) {
	f := newFixture(t)
	f.mustPublish(f.records())
	path := filepath.Join(f.run, "g0001/manifest.json")
	manifest, err := ovio.LoadObject(path)
	if err != nil {
		t.Fatal(err)
	}
	delete(manifest, "receipts")
	write(t, path, manifest)
	if code, out, errs := f.cmd("validate", f.run); code != 1 {
		t.Fatalf("unbound old generation remained authoritative: %d %s%s", code, out, errs)
	}
	if code, out := f.publish(f.records(), nil); code != 1 {
		t.Fatalf("new publication laundered unbound old admission: %d %s", code, out)
	}
}

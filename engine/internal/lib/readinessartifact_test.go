package lib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadinessContractOwnsUniqueExitReasons(t *testing.T) {
	want := map[string]int{
		"ready":                 0,
		"plan-unapproved":       2,
		"awaiting-human":        3,
		"plan-blocked":          4,
		"workspace-missing":     5,
		"coverage-not-clear":    6,
		"engineering-not-ready": 7,
	}
	seenCodes := map[int]string{}
	for _, reason := range readinessContract.Reasons {
		if got, ok := want[reason.ID]; !ok || reason.Code != got {
			t.Fatalf("reason %q code=%d, want %d (known=%v)", reason.ID, reason.Code, got, ok)
		}
		if prior := seenCodes[reason.Code]; prior != "" {
			t.Fatalf("exit code %d is owned by both %q and %q", reason.Code, prior, reason.ID)
		}
		seenCodes[reason.Code] = reason.ID
		delete(want, reason.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing readiness reasons: %v", want)
	}
}

func TestBuildReadinessRequiresSemanticFreshArtifacts(t *testing.T) {
	root, slug := readyWorkspace(t)
	var stdout, stderr bytes.Buffer
	if code := BuildReadiness(root, []string{slug}, &stdout, &stderr); code != 0 {
		t.Fatalf("valid readiness code=%d stderr=%q", code, stderr.String())
	}

	t.Run("stale coverage after spec change", func(t *testing.T) {
		root, slug := readyWorkspace(t)
		writeReadinessFile(t, root, slug, "spec.md", "# Spec\n\n## Acceptance criteria\n- AC-001 changed\n")
		if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 6 {
			t.Fatalf("code=%d, want clarify exit 6", code)
		}
	})

	t.Run("stale coverage after decision change", func(t *testing.T) {
		root, slug := readyWorkspace(t)
		writeReadinessFile(t, root, slug, "decisions.md", "# Decisions\n\nA product decision changed.\n")
		if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 6 {
			t.Fatalf("code=%d, want clarify exit 6", code)
		}
	})

	t.Run("covered technical ledger append refreshes readiness", func(t *testing.T) {
		root, slug := readyWorkspace(t)
		writeReadinessFile(t, root, slug, "decisions.md", `# Decisions

No product decisions remain open.

Technical approach: reuse the existing layer, as delegated by the coverage matrix.
`)
		if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 6 {
			t.Fatalf("stale coverage code=%d, want clarify exit 6", code)
		}

		coverageDigest, err := readinessInputsDigest(root, slug, readinessContract.Coverage.Inputs)
		if err != nil {
			t.Fatal(err)
		}
		writeReadinessFile(t, root, slug, "decision-coverage.md", validCoverage(coverageDigest))
		if err := validateDecisionCoverage(root, slug); err != nil {
			t.Fatalf("refreshed decision coverage: %v", err)
		}

		engineeringDigest, err := readinessInputsDigest(root, slug, readinessContract.Engineering.Inputs)
		if err != nil {
			t.Fatal(err)
		}
		writeReadinessFile(t, root, slug, "eng-review.md", validEngineeringReview(engineeringDigest))
		if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
			t.Fatalf("refreshed readiness code=%d, want ready", code)
		}
	})

	t.Run("stale engineering review after plan change", func(t *testing.T) {
		root, slug := readyWorkspace(t)
		writeReadinessFile(t, root, slug, "plan.md", "# Plan\n\nchanged\n")
		if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 7 {
			t.Fatalf("code=%d, want vet exit 7", code)
		}
	})

	t.Run("empty test plan", func(t *testing.T) {
		root, slug := readyWorkspace(t)
		writeReadinessFile(t, root, slug, "test-plan.md", "")
		if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 7 {
			t.Fatalf("code=%d, want vet exit 7", code)
		}
	})
}

func TestBuildReadinessRejectsMarkerOnlyAndContradictoryArtifacts(t *testing.T) {
	root := t.TempDir()
	slug := "marker-only"
	writeReadinessFile(t, root, slug, "state.md", "- Phase: vet\n- Plan approved: now\n- Status: running\n")
	writeReadinessFile(t, root, slug, "decision-coverage.md", "Decision coverage: CLEAR\n| Surface | Dimension | Status |\n| x | y | Missing |\n")
	writeReadinessFile(t, root, slug, "eng-review.md", "Implementation readiness: READY\n")
	writeReadinessFile(t, root, slug, "test-plan.md", "")

	var stderr bytes.Buffer
	if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &stderr); code != 6 {
		t.Fatalf("code=%d stderr=%q, want clarify exit 6", code, stderr.String())
	}
}

func TestBuildReadinessChecksClarificationBeforePlanApproval(t *testing.T) {
	root := t.TempDir()
	slug := "combined"
	writeReadinessFile(t, root, slug, "state.md", "- Phase: vet\n- Status: running\n")
	var stderr bytes.Buffer
	if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &stderr); code != 6 {
		t.Fatalf("code=%d stderr=%q, want clarify exit 6 before define", code, stderr.String())
	}
}

func TestDecisionCoverageRejectsOpenHumanGate(t *testing.T) {
	root, slug := readyWorkspace(t)
	writeReadinessFile(t, root, slug, "questions.md", "## q-1\nstatus: open\ngate: validating\n")
	coverageDigest, err := readinessInputsDigest(root, slug, readinessContract.Coverage.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	writeReadinessFile(t, root, slug, "decision-coverage.md", validCoverage(coverageDigest))
	if err := validateDecisionCoverage(root, slug); err == nil || !strings.Contains(err.Error(), "open validating") {
		t.Fatalf("validateDecisionCoverage error=%v, want open validating question", err)
	}
	if code := BuildReadiness(root, []string{slug}, &bytes.Buffer{}, &bytes.Buffer{}); code != 6 {
		t.Fatalf("code=%d, want clarify exit 6 for refreshed digest with open human gate", code)
	}
}

func TestDecisionCoverageRejectsOpenQuestionTableAndUnownedAssumption(t *testing.T) {
	t.Run("question table", func(t *testing.T) {
		root, slug := readyWorkspace(t)
		writeReadinessFile(t, root, slug, "questions.md", `# Questions

| Question ID | Status | Gate | Question |
| --- | --- | --- | --- |
| Q-001 | open | blocking | Choose behavior |
`)
		coverageDigest, err := readinessInputsDigest(root, slug, readinessContract.Coverage.Inputs)
		if err != nil {
			t.Fatal(err)
		}
		writeReadinessFile(t, root, slug, "decision-coverage.md", validCoverage(coverageDigest))
		if err := validateDecisionCoverage(root, slug); err == nil || !strings.Contains(err.Error(), "open blocking") {
			t.Fatalf("validateDecisionCoverage error=%v, want table gate rejection", err)
		}
	})

	t.Run("unowned assumption", func(t *testing.T) {
		root, slug := readyWorkspace(t)
		path := filepath.Join(root, "work", slug, "decision-coverage.md")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		body := strings.Replace(string(raw),
			"| none | n/a | high | n/a | n/a | n/a |",
			"| CDN is stable | weak evidence | low | none | none | outage |", 1)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := validateDecisionCoverage(root, slug); err == nil || !strings.Contains(err.Error(), "unowned or unverifiable") {
			t.Fatalf("validateDecisionCoverage error=%v, want assumption rejection", err)
		}
	})
}

func TestReadinessDigestCommandUsesEmbeddedContract(t *testing.T) {
	root, slug := readyWorkspace(t)
	var stdout, stderr bytes.Buffer
	if code := ReadinessDigest(root, []string{"coverage", slug}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "Coverage inputs SHA-256: ") ||
		len(strings.TrimSpace(strings.TrimPrefix(stdout.String(), "Coverage inputs SHA-256: "))) != 64 {
		t.Fatalf("unexpected digest output %q", stdout.String())
	}
}

func readyWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	slug := "ready"
	writeValidReadinessArtifacts(t, root, slug)
	writeReadinessFile(t, root, slug, "state.md", "- Phase: vet\n- Plan approved: now\n- Status: running\n")
	return root, slug
}

func writeValidReadinessArtifacts(t *testing.T, root, slug string) {
	t.Helper()
	for name, body := range map[string]string{
		"brief.md":        "# Brief\n\nFeature outcome.\n",
		"spec.md":         "# Spec\n\n## Acceptance criteria\n- AC-001 returns success.\n",
		"architecture.md": "# Architecture\n\nUse the existing layer.\n",
		"plan.md":         "# Plan\n\nImplement one bounded slice.\n",
		"tasks.md":        "# Tasks\n\n## SLICE-001\nBuild it.\n",
		"traceability.md": "# Traceability\n\nAC-001 -> SLICE-001.\n",
		"decisions.md":    "# Decisions\n\nNo product decisions remain open.\n",
		"assumptions.md":  "# Assumptions\n\nNo material assumptions remain open.\n",
		"questions.md":    "# Questions\n\nNone.\n",
	} {
		writeReadinessFile(t, root, slug, name, body)
	}
	coverageDigest, err := readinessInputsDigest(root, slug, readinessContract.Coverage.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	writeReadinessFile(t, root, slug, "decision-coverage.md", validCoverage(coverageDigest))
	writeReadinessFile(t, root, slug, "test-plan.md", validTestPlan())
	engineeringDigest, err := readinessInputsDigest(root, slug, readinessContract.Engineering.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	writeReadinessFile(t, root, slug, "eng-review.md", validEngineeringReview(engineeringDigest))
}

func validCoverage(digest string) string {
	return fmt.Sprintf(`# Decision coverage

Decision coverage: CLEAR
Coverage inputs SHA-256: %s

## Topology
| Surface | Kind | Related IDs | Evidence |
| --- | --- | --- | --- |
| public flow | behavior | AC-001 | spec.md |

## Coverage matrix
| Surface | Dimension | Status | Canonical reference | Owner / validation gate | Consequence if wrong |
| --- | --- | --- | --- | --- | --- |
| public flow | behavior | closed | spec.md AC-001 | rite-prove | wrong response |
| implementation | technical approach | agent-owned | n/a | rite-vet | incompatible design |

## Assumption audit
| Assumption | Evidence | Confidence | Owner | Validation | Consequence if wrong |
| --- | --- | --- | --- | --- | --- |
| none | n/a | high | n/a | n/a | n/a |

## Residual uncertainty
| Item | Why nonblocking | Owner | Validation gate |
| --- | --- | --- | --- |
| none | n/a | n/a | n/a |

## Readiness verdict
No unresolved material decision remains.
`, digest)
}

func validTestPlan() string {
	return `# Test plan

## Build-entry preflight
| Gate | Command | Cwd | Expected | Prerequisites | Provenance to recapture |
| --- | --- | --- | --- | --- | --- |
| unit | go test ./... | engine | exit 0 | none | go version |

## Per-gap test requirements
| ID | Path / flow | Test file | Asserts | Kind | Slice | Priority |
| --- | --- | --- | --- | --- | --- | --- |
| T1 | public flow | flow_test.go | request → success | unit | SLICE-001 | P1 |

## Acceptance → test map
- AC-001 → T1
`
}

func validEngineeringReview(digest string) string {
	return fmt.Sprintf(`# Eng review

Implementation readiness: READY
Readiness inputs SHA-256: %s

## 2a. Build-entry preflight
| Gate | Command + cwd | Tool/version | Prerequisite owner | Full provenance inputs | Fixture/smoke | Verdict |
| --- | --- | --- | --- | --- | --- | --- |
| unit | go test ./... | go | none | go.mod | focused suite | pass |

## 2b. Implementation readiness
| Surface | Requirement/decision | Boundary/wiring | Slice | Proof | Verdict |
| --- | --- | --- | --- | --- | --- |
| public flow | AC-001 | existing layer | SLICE-001 | T1 | ready |

## 4. Failure modes
| New codepath | Realistic failure | Test? | Handling? | Silent? | Verdict |
| --- | --- | --- | --- | --- | --- |
| public flow | invalid request | yes | yes | no | ok |

## 7. Completion summary
- All required surfaces are ready.
`, digest)
}

func writeReadinessFile(t *testing.T, root, slug, name, body string) {
	t.Helper()
	path := filepath.Join(root, "work", slug, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

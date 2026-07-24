package main_test

// TestParityBuildReadiness checks the build-readiness gate's stdout and exit code
// against the golden snapshot for each state.md fixture.

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/devrites/devrites/internal/lib"
)

func TestParityBuildReadiness(t *testing.T) {
	work := t.TempDir()

	// Status halts write only to stderr; parity covers stdout and the exit code.
	writeFeatureFile(t, work, "await", "state.md", "- Status: awaiting_human\n") // exit 3
	writeFeatureFile(t, work, "blocked", "state.md", "- Status: blocked\n")      // exit 4

	writeReadyArtifacts := func(slug string) {
		t.Helper()
		writeCoverageArtifacts(t, work, slug)
		for name, body := range map[string]string{
			"architecture.md": "# Architecture\n\nExisting layer.\n",
			"plan.md":         "# Plan\n\nOne slice.\n",
			"tasks.md":        "# Tasks\n\n## SLICE-001\nBuild.\n",
			"traceability.md": "# Traceability\n\nAC-001 -> SLICE-001.\n",
			"test-plan.md": `# Test plan

## Build-entry preflight
| Gate | Command | Cwd | Expected | Prerequisites | Provenance to recapture |
| --- | --- | --- | --- | --- | --- |
| unit | go test ./... | engine | exit 0 | none | go version |

## Per-gap test requirements
| ID | Path / flow | Test file | Asserts | Kind | Slice | Priority |
| --- | --- | --- | --- | --- | --- | --- |
| T1 | flow | flow_test.go | request → success | unit | SLICE-001 | P1 |

## Acceptance → test map
- AC-001 → T1
`,
		} {
			writeFeatureFile(t, work, slug, name, body)
		}
		root := filepath.Join(work, ".devrites")
		digest, err := lib.ReadinessInputDigest(root, slug, "engineering")
		if err != nil {
			t.Fatal(err)
		}
		writeFeatureFile(t, work, slug, "eng-review.md", fmt.Sprintf(`# Eng review

Implementation readiness: READY
Readiness inputs SHA-256: %s

## 2a. Build-entry preflight
| Gate | Command + cwd | Tool/version | Prerequisite owner | Full provenance inputs | Fixture/smoke | Verdict |
| --- | --- | --- | --- | --- | --- | --- |
| unit | go test ./... | go | none | go.mod | focused | pass |

## 2b. Implementation readiness
| Surface | Requirement/decision | Boundary/wiring | Slice | Proof | Verdict |
| --- | --- | --- | --- | --- | --- |
| flow | AC-001 | existing | SLICE-001 | T1 | ready |

## 4. Failure modes
| New codepath | Realistic failure | Test? | Handling? | Silent? | Verdict |
| --- | --- | --- | --- | --- | --- |
| flow | invalid | yes | yes | no | ok |

## 7. Completion summary
- Ready.
`, digest))
	}

	// A ready feature has an approved plan, running status, and current
	// clarification and vet artifacts.
	for _, slug := range []string{"approved", "trailpipe", "trailhash", "emptystatus"} {
		writeReadyArtifacts(slug)
	}
	writeFeatureFile(t, work, "approved", "state.md", "- Phase: vet\n- Plan approved: 2024-01-01\n- Status: running\n") // exit 0

	// Plan not approved: missing line, or the literal "none".
	writeCoverageArtifacts(t, work, "noapprove")
	writeCoverageArtifacts(t, work, "approvenone")
	writeFeatureFile(t, work, "noapprove", "state.md", "- Status: running\n")                          // exit 2
	writeFeatureFile(t, work, "approvenone", "state.md", "- Plan approved: none\n- Status: running\n") // exit 2

	// field() strip: a trailing " | none" / " # note" annotation is trimmed off
	// Status before the case match, so both stay "running" and reach OK.
	writeFeatureFile(t, work, "trailpipe", "state.md", "- Phase: vet\n- Plan approved: 2024-01-01\n- Status: running | none\n") // exit 0
	writeFeatureFile(t, work, "trailhash", "state.md", "- Phase: vet\n- Plan approved: 2024-01-01\n- Status: running # note\n") // exit 0

	// An empty Status with an approved plan renders as "running".
	writeFeatureFile(t, work, "emptystatus", "state.md", "- Phase: vet\n- Plan approved: 2024-01-01\n- Status: \n") // exit 0

	// A plan cannot build without an explicit closed clarification scan and a
	// successful engineering-readiness verdict. These are objective routes, not
	// human questions.
	writeFeatureFile(t, work, "noclarify", "state.md", "- Phase: vet\n- Plan approved: 2024-01-01\n- Status: running\n") // exit 6

	writeFeatureFile(t, work, "clarifyopen", "state.md", "- Phase: vet\n- Plan approved: 2024-01-01\n- Status: running\n") // exit 6
	writeFeatureFile(t, work, "clarifyopen", "brief.md", "# Brief\n\nOutcome.\n")
	writeFeatureFile(t, work, "clarifyopen", "spec.md", "# Spec\n\nAC-001.\n")
	writeFeatureFile(t, work, "clarifyopen", "decision-coverage.md", "# Decision coverage\n\n- Decision coverage: NEEDS CLARIFICATION\n")

	writeFeatureFile(t, work, "novet", "state.md", "- Phase: plan\n- Plan approved: 2024-01-01\n- Status: running\n") // exit 7
	writeCoverageArtifacts(t, work, "novet")

	writeFeatureFile(t, work, "vetnotready", "state.md", "- Phase: vet\n- Plan approved: 2024-01-01\n- Status: running\n") // exit 7
	writeCoverageArtifacts(t, work, "vetnotready")
	writeFeatureFile(t, work, "vetnotready", "eng-review.md", "# Engineering review\n\n- Implementation readiness: NEEDS REPLAN\n")

	writeReadyArtifacts("stalevet")
	writeFeatureFile(t, work, "stalevet", "state.md", "- Phase: plan\n- Plan approved: 2024-01-01\n- Status: running\n") // exit 7

	// "ghost" is intentionally never created → no workspace/state.md (exit 5).
	for _, arg := range []string{
		"await", "blocked", "approved", "noapprove", "approvenone",
		"trailpipe", "trailhash", "emptystatus", "noclarify", "clarifyopen",
		"novet", "vetnotready", "stalevet", "ghost",
	} {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  []string{"build-readiness", arg},
		}
		t.Run("arg="+arg, func(t *testing.T) { c.assertEqual(t) })
	}
}

func writeCoverageArtifacts(t *testing.T, work, slug string) {
	t.Helper()
	writeFeatureFile(t, work, slug, "brief.md", "# Brief\n\nOutcome.\n")
	writeFeatureFile(t, work, slug, "spec.md", "# Spec\n\n## Acceptance criteria\n- AC-001 succeeds.\n")
	writeFeatureFile(t, work, slug, "decisions.md", "# Decisions\n\nNo open product decisions.\n")
	writeFeatureFile(t, work, slug, "assumptions.md", "# Assumptions\n\nNo material assumptions.\n")
	writeFeatureFile(t, work, slug, "questions.md", "# Questions\n\nNone.\n")
	root := filepath.Join(work, ".devrites")
	digest, err := lib.ReadinessInputDigest(root, slug, "coverage")
	if err != nil {
		t.Fatal(err)
	}
	writeFeatureFile(t, work, slug, "decision-coverage.md", fmt.Sprintf(`# Decision coverage

Decision coverage: CLEAR
Coverage inputs SHA-256: %s

## Topology
| Surface | Kind | Related IDs | Evidence |
| --- | --- | --- | --- |
| flow | behavior | AC-001 | spec.md |

## Coverage matrix
| Surface | Dimension | Status | Canonical reference | Owner / validation gate | Consequence if wrong |
| --- | --- | --- | --- | --- | --- |
| flow | behavior | closed | AC-001 | rite-prove | failure |

## Assumption audit
| Assumption | Evidence | Confidence | Owner | Validation | Consequence if wrong |
| --- | --- | --- | --- | --- | --- |
| none | n/a | high | n/a | n/a | n/a |

## Residual uncertainty
| Item | Why nonblocking | Owner | Validation gate |
| --- | --- | --- | --- |
| none | n/a | n/a | n/a |

## Readiness verdict
No unresolved decision remains.
`, digest))
}

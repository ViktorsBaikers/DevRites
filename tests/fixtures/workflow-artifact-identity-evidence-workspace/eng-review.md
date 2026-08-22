# Eng review: Workflow Artifact Identity

Vetted: 2026-08-22T22:34:09Z  
Cross-model: off  
DevRites contract: devrites.readiness-artifacts.v2  
Implementation readiness: READY  
Readiness inputs SHA-256: ece91d5e1120fe4ef65655b2a8fd529ea8cca8e4284fe2e06b2fdadeda1ef92b

## 1. Depth

full — complexity (>8 files; 16 authored + 22 generated) and cross-module blast radius across ten canonical adapters plus two generated hosts. One deep semantic module concentrates admission, identity, owner/concurrency, operations, proof/retry, evidence, routes/diagnostics, resume, and product separation. Deletion redistributes policy across callers.

## 2. Scope challenge

- What already exists (reuse vs rebuild): existing `workflow-artifacts.md` module, root/wright authority seam, `DEVRITES_HOST_ARTIFACT_DIR` generator override, behavioral corpus grammar, engine candidate/readiness commands, repository validators, instruction ratchet, `tests/acceptance-preserving-reslice-policy-test.sh`. No parallel module.
- Minimum diff: 16 authored destinations and generator-derived 22 mirrors under one bounded wright; one `SLICE-001`; no second module. Accepted as-is.
- Complexity: smell (38 destinations, one new semantic deepening, no new service). Justified by budget override: splitting would create competing authority. Proceed.
- Built-in / completeness / distribution: Python `fcntl.flock`, SHA-256, umask-077 `mkdirat`/`openat`; no new dependency or publishable artifact. Completeness: interruption/retry/evidence matrices already named. Distribution: none; generated hosts stay in-repo mirrors.

## 2a. Build-entry preflight

| Gate | Command + cwd | Tool/version | Prerequisite owner | Full provenance inputs | Fixture/smoke | Verdict |
| --- | --- | --- | --- | --- | --- | --- |
| toolchain | chained versions plus `go -C engine env GOVERSION GOTOOLCHAIN` · `.` | Bash 5.3.15; Python 3.14.6; Node 24.18.0; module Go 1.26.5/auto | local toolchain | `engine/go.mod` SHA-256 `f8502a945d37b6f5376cfc2f9dc8d964f4e6588623f6855b8a483ac1885406d7` | module Go resolved | pass |
| workspace schema | `python3 scripts/validate-workspace-schema.py .devrites/work/workflow-artifact-identity` · `.` | Python 3.14.6 | Vet | validator SHA-256 `2177123d1f560caeacfe4770724f1199195db06abc39f31e4be88e130f4334b3` | `OK: 1 workspace(s)` after fold-back | pass |
| baseline semantics | behavioral schema, instruction baseline, syntax, engine race and full repository | repository | repository | recorded in EVID-012 | 14 files/82 scenarios; 217/854965 bytes; v11 16/16 PASS | pass |
| generator override | source inspection and host test contract · `.` | Bash | repository | generator SHA-256 `bae83aab35584d1a24198fc95b63921ecb829676aaaaf7b0cf26eca9c8dde3b0` | override consumed; default path prohibited | pass |
| protected baseline | SHA-256 triple · `.` | shasum | root | `.gitignore` `58c1cc88c16b9bb14b345c156703163b47c9cb6232276b50684fabae8503e8fd`; `.devrites/ACTIVE` `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267`; Workspace Observation `2dca74484895de119cd935db6c3692782df9173eef199c88a7d5a65898332ec9` | live hashes match Q-010/DEC-047 | pass |
| dedicated/corpus | exact commands in `test-plan.md` · `.` | Build/Prove outputs | sole wright | dedicated driver SHA-256 `f3d26a4cc41727dc766fcc73f6e879b94026fc55bd84e7d15c6ce0304a1620d9` | EVID-012 PASS | pass |
| private 16/22 delivery | writer-only prepare/install/recover modes | sole wright | sole wright | delivery `8fd99161cc209bc631ae9e00eed88c28bdd05604f4c94ade934f91fe480d7201` CLEANED | empty generated delta under DEC-048 | pass |

Root `go.mod` is absent; `engine/go.mod` is authoritative. No network/service/credential prerequisite. `devrites-engine snapshot`, `analyze`, and `readiness-digest` are not in the current engine command set; binding used `check readiness --emit-binding`.

## 2b. Implementation readiness

| Surface | Requirement/decision | Boundary/wiring | Slice | Proof | Verdict |
| --- | --- | --- | --- | --- | --- |
| admission/identity | REQ-002–REQ-004; DEC-002, DEC-003, DEC-014 | grammar, relational minima, encodings, handle/identity golden vector; this slug N/A admission | SLICE-001 | T-001, T-003 | ready |
| owner/concurrency | REQ-005; DEC-009 | umask-077 mkdirat/openat; only `fcntl.flock`; generation compare | SLICE-001 | T-002 | ready |
| retained source | REQ-003, REQ-004, REQ-007; DEC-013 | promotion, held bytes, binding-rollover stale GC, terminal GC | SLICE-001 | T-003, T-004 | ready |
| transaction/retry | REQ-005–REQ-007; DEC-004, DEC-006, DEC-010 | sixteen operations, complete writes, branches, retry/exhaustion, source-loss classes | SLICE-001 | T-004–T-008 | ready |
| proof/evidence | REQ-007, REQ-008; DEC-005, DEC-011 | process-group bounds/reap; marker ownership; product identity | SLICE-001 | T-007, T-009, T-013 | ready |
| oracle/diagnostics | AC-004, AC-006; DEC-009, DEC-012 | five-field oracle; separate OS/engine observer; finite safe output | SLICE-001 | T-005, T-010, T-011 | ready |
| adapters | REQ-001; AC-001; DEC-051 | ten JSON-comment declarations; shared-contract consumers | SLICE-001 | T-012 | ready |
| root-caller DX | AC-006 | exact `--prove-walkthrough`; TTHW 90s includes toolchain; proof subprocesses 30/60 | SLICE-001 | T-015 | ready |
| source/host delivery | REQ-009; DEC-008; DEC-042; DEC-048 | bounded wright authors; hash-bound driver alone writes 16/22; empty generated delta legal when inputs unchanged | SLICE-001 | T-014, T-016 | ready; EVID-012 PASS |
| prior-candidate composition | AC-001, AC-007; DEC-015 | 13 shared Reslice preimages | SLICE-001 | T-017 | ready |

Inventory/currentness: pass, including 13 shared Reslice destinations, 30 non-overlap prior paths, zero Workspace Observation overlap, and inspect-and-OUT SHA freeze of `README.md` / `docs/skills.md` · slice order/independence: one indivisible serial writer slice · UX/spec/architecture: not applicable to end-user UI · operations/rollout/rollback: exact local-only transaction, no release mutation · Shared contract proof: pass

## 3. Axis findings (floor-gated)

| Axis | Floor band | Findings |
| --- | --- | --- |
| Architecture | adequate | Shared contract table and pinned adapter JSON comment folded (DEC-051). Recheck: no new Important. |
| Plan code-quality | adequate | exact lock/bootstrap, relational limits, state tables, writer journal, restart semantics |
| Test-coverage design | strong | all REQ/AC/edge/prohibitions map; observer prevents self-attestation |
| Performance | strong | bounded bytes/files/output/process time/retry/history; TTHW 90s distinct from 30/60 proof subprocess limits; measured 3,716 ms |
| Reversibility | adequate | root verify-only; pre-commit bounded recovery; coordinator cannot recover; post-commit cleanup never rolls back |
| Failure modes | strong | owner/source/write/install/proof/evidence/retry/rollover/generated-delivery failures have owner, state, route, proof |

Floor: adequate. Critical: 0. Important: 0 after fold. Suggestions: spec demo example still lacks referenced `## ID` blocks (this slug admits no active target). Suppressed low-confidence: 0 from this pass.

## 4. Failure modes

| New codepath | Realistic failure | Test? | Handling? | Silent? | Verdict |
| --- | --- | --- | --- | --- | --- |
| lock bootstrap | create race, death, symlink/mode, unsupported flock | y | reconcile exact private names or fail closed | n | ok |
| owner contention | two roots or alternate lock primitive | y | canonical flock winner; loser zero writes; mutant rejected | n | ok |
| source promotion | marker/write/sync/rename death | y | exact preparing recovery | n | ok |
| source rollover | promoted old binding before journal | y | durable stale-cleanup rename and idempotent GC | n | ok |
| complete writes | partial/invalid progress, ENOSPC, death | y | exact partial reconcile; bounded failure | n | ok |
| install/readback | handle omission, replace failure, target drift | y | exact preimage rollback or safety gate | n | ok |
| proof process | nonzero, timeout, descendant, output overflow | y | terminate/grace/kill/reap then rollback | n | ok |
| retry | handoff death, same/distinct fingerprint, cap | y | immutable epoch; same resume; finite exhaustion/GC | n | ok |
| evidence | existing lifecycle bytes, malformed markers, stale generation | y | preserve outside bytes or fail before mutation | n | ok |
| diagnostics | hostile value or raw error | y | finite reason/boundary/route line only | n | ok |
| candidate delivery | wright death during snapshot/stage/install/proof/rollback | y | next serial wright resumes; pre-commit restores 16/22 | n | ok |
| root caller | toolchain absent or TTHW/output mismatch | y | block before fixture or Prove finding; no mock | n | ok |
| shared Reslice destinations | additive edits drop packet/route/action/stop/baseline semantics | y | dedicated prior-feature gate fails before `COMMITTED` | n | ok |

## 5. Dependency safety

One serial lane: root freezes the 16/22 boundary plus shared/non-overlap prior-candidate identities → bounded wright prepares and authors additive edits → hash-bound driver privately generates, installs, runs Workflow Artifact and prior Reslice gates, then commits → root exact verification → Prove → Seal. DEC-042 permits only an exact coordinator launch when the host window is insufficient. Read-only proof/review may fan out only after the frozen candidate. No second filesystem writer. No package/dependency ordering.

## 6. Reviewer loop

- Initial Plan: NEEDS REPLAN — five Critical and four Important transaction/concurrency/evidence/generator/test gaps.
- Initial DevEx: BLOCKED — admission/journal serialization, encodings, route actions, diagnostics, command prerequisite, and walkthrough unclear.
- Narrow Plan recheck: original complete-write/retry/CLEANED/evidence/generator-target/timeout repairs confirmed; final gaps were oracle observer, exact flock/bootstrap, stale rollover GC, generated writer ownership/restart, and relational limits.
- Narrow DevEx recheck: grammar/encoding/routes/diagnostics confirmed; final gaps were module Go prerequisite and distinct Prove/TTHW command.
- Root folded all final technical findings under DRIFT-003 and DEC-009–DEC-014. No third reviewer loop. Cross-model off.
- Build-entry overlap freeze found 13 destinations shared with sealed Acceptance-preserving Reslice. DRIFT-004/DEC-015.
- 2026-08-21 post-prove re-vet: independent plan-reviewer on DEC-048 returned status pass, floor adequate, findings none.
- 2026-08-22 post-prove re-vet: independent plan-reviewer Outcome findings, floor thin on Architecture → BLOCKED. Accepted DEC-051 packaging folds. Independent DevEx predict Important spec-example copy-paste gap rejected (this slug N/A admission). Narrow plan recheck: Outcome no-findings; accepted findings closed; floor adequate → PASS.
- 2026-08-22 post-EVID-014 readiness re-vet: independent plan-reviewer Outcome no-findings; floor adequate on Scope discipline & reuse → PASS. No plan/task/spec edits. Cross-model off. DevEx not re-dispatched (scorecard current; no developer-surface plan delta).
- 2026-08-22 post-EVID-016 readiness re-vet (this pass): independent plan-reviewer Outcome no-findings; floor adequate on Architecture → PASS. No plan/task/spec edits. Binding-only refresh after evidence bookkeeping. Cross-model off. DevEx not re-dispatched (scorecard current; no developer-surface plan delta).

## Post-implementation verification

Final candidate `a6b102c354d756cb8d89bb395b54704ed9a9206ec1b652c7c24ff77426db453e` preserves the vetted single-module/single-slice interface and exact 16 authored/22 generated destination boundary. Selected delivery `71411cf0ad4f02e80f57b64e2179eb06ee02e05de4b77e1bfcd4ab76598eafca` reached `CLEANED` (journal `a79264c8788d99b89b0ce4733e9279d4457262cbdb6f8617ea8b09434d6faa9b`; driver `bda33a29f3ba7217749e36f0c7bb2063637e91a953ceaeeb58857dfb458b0781`); generated delta empty under DEC-048. Bundle `.root-proof-a6b102c354d756cb-v13` 16/16 PASS; attestor and proof-runner PASS (EVID-016). DEC-042 launched the prepared install. Historical `8c8cb87c…` / EVID-014 and `bee44b1ada3b9758…` / EVID-012 remain prior identities. Sealed-bundle readiness observation `40372e0b1eaa9ec63eeaf252ec4b2ce4574ef3f1814355725652056a63059225` went stale after evidence bookkeeping; this Vet records the current live binding. No acceptance criterion or product behavior changed.


## 7. Completion summary

- Scope: accepted; same SLICE-001, 16 authored, 22 generated destinations.
- Architecture: 2 Important packaging findings folded; 0 remaining.
- Code-quality: 0.
- Coverage: 7/7 ACs, 7/7 edges, 12/12 prohibitions, 17 test requirements; 0 gaps; regressions remain Critical in T-001–T-017.
- Build entry: deterministic preflight pass; action-time checkpoints none.
- Failure modes: 13 mapped; 0 Critical gaps.
- NOT in scope and What already exists: written.
- Plan: hardened in place (DEC-051); no Spec Drift Guard; no 38-file dest change.

## Build readback

- Outcome: crash-resumable Workflow Artifact identity in existing `workflow-artifacts.md`; AC-001–AC-007 already proved on candidate `a6b102c354d756cb…` (EVID-016).
- IN: 16 authored + 22 generated destinations. OUT: engine command, generic materializer, operator README/docs catalogue, ACTIVE rewrite, consumptive action.
- Architecture: one module, ten JSON-comment adapters, feature-local delivery journal, DEC-042 launcher cannot write/recover.
- Slice order: SLICE-001 already built/proved; no further wright dest change from this Vet.
- Decisive proof: EVID-016 on candidate `a6b102c354d756cb…`; next human-facing rite is `$rite-seal`.

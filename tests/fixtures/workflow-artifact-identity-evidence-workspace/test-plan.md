# Test plan: Workflow Artifact Identity
From /rite-vet on 2026-08-14T16:09:17Z. Runner + conventions: Bash contract tests, Python disposable filesystem fixture, repository Node validators, and module-selected Go.
DevRites contract: devrites.readiness-artifacts.v2

## Build-entry preflight

| Gate | Command | Cwd | Expected | Prerequisites | Provenance to recapture |
| --- | --- | --- | --- | --- | --- |
| toolchain | `bash --version && python3 --version && node --version && go -C engine env GOVERSION GOTOOLCHAIN` | `.` | Bash 5.3.15; Python 3.14.6; Node 24.18.0; module-selected `go1.26.5` and `auto` resolve before mutation | Go 1.26.5 already available locally/in toolchain cache; no mock/network fallback | exact output; `engine/go.mod` SHA-256 `f8502a945d37b6f5376cfc2f9dc8d964f4e6588623f6855b8a483ac1885406d7` |
| workspace schema | `python3 scripts/validate-workspace-schema.py .devrites/work/workflow-artifact-identity` | `.` | `workspace-schema: OK: 1 workspace(s) validated` | Python 3.9+ | validator SHA-256 `2177123d1f560caeacfe4770724f1199195db06abc39f31e4be88e130f4334b3`; current planning identity |
| dedicated contract | `bash tests/workflow-artifact-identity-test.sh` | `.` | `workflow-artifact-identity: PASS` | SLICE-001 creates script; Bash/Python; module-selected Go 1.26.5 preflight; self-build actual engine when env absent | dedicated test, canonical module, `engine/go.mod`, engine source hashes |
| Prove walkthrough | `bash tests/workflow-artifact-identity-test.sh --prove-walkthrough` | `.` | exact proof signal, `tthw_ms` decimal, stale diagnostic, cursor/product lines, `WORKFLOW_ARTIFACT_WALKTHROUGH PASS` | same toolchain; private disposable workspace; no active admission | command output, monotonic TTHW to first `CLEANED` return, recovery traces |
| behavioral corpus | `bash scripts/run-behavioral-evals.sh` | `.` | all corpus files schema-valid; zero failures; no live-provider claim | Bash/Python or jq; SLICE-001 creates corpus | runner SHA-256 `4151c00c130ea6a378705f8d06d26902d552edcc54cbed02db48b5342ebed248`; corpus hashes |
| routing | `bash tests/phase-gate-routing-test.sh` | `.` | exact route precedence/action/cursor assertions pass | Bash | test and ten canonical adapter hashes |
| prior Reslice regression | `bash tests/acceptance-preserving-reslice-policy-test.sh` | `.` | existing Reslice dedicated suite passes after five authored edits and eight generated mirror changes; 30 non-overlap paths stay exact | Bash/Python/Go | prior test/standard/corpus plus 13 shared and 30 non-overlap pre/post hashes |
| host parity | `bash tests/host-artifacts-test.sh` | `.` | private-output generation and Claude/Codex parity pass | Bash; generator honors `DEVRITES_HOST_ARTIFACT_DIR` | generator SHA-256 `bae83aab35584d1a24198fc95b63921ecb829676aaaaf7b0cf26eca9c8dde3b0`; host-test hash; full generated manifest |
| instruction cap | `node scripts/check-instruction-size-baseline.mjs` | `.` | canonical instruction bytes at or below 855000 | Node; refreshed baseline | checker SHA-256 `e3ec66d098a85bc7f785d950581133e6793345b269306c0c773316533c5c209d`; baseline hash |
| repository validation | `bash scripts/validate.sh` | `.` | `VALIDATION PASSED` | repository toolchain | validator SHA-256 `dc299d7c9e78ce2b5a54af72c5e6ab4d4d83d7d4a90c2500384674794cf3b1d5`; package/module manifests |
| cross references | `python3 scripts/check-cross-refs.py` | `.` | exit 0 | Python 3 | script SHA-256 `32d3eaf0fa88ad0adaef511cd837ee3ecc824817e32c20618edc7ed9d471cc89`; canonical/generated hashes |
| invocation integrity | `python3 scripts/check-invocation-integrity.py` | `.` | exit 0 | Python 3 | script SHA-256 `4dccd95d90dd2c77455bd1379a6df482e85702c7bf2c0878142745b7dedd27df`; skill hashes |
| pack security | `python3 scripts/scan-pack-security.py pack/.claude pack/generated` | `.` | exit 0; no unsafe pack content | Python 3 | script SHA-256 `55e389e503adb6b5e3aea7df161f732b0bbc08fdcc630ce5b083db92f69ed736`; scanned-tree manifest |
| engine race | `go -C engine test ./... -race -count=1` | `.` | exit 0 | module-selected Go | `engine/go.mod`; engine source identity |
| full repository | `node scripts/run-tests.mjs` | `.` | full suite pass | Node/Go/Bash/Python | runner SHA-256 `167c8d899df00e031dbb679ebfcf49df0c365204d2fe5eaf9f49ca9dc4b5b646`; `package.json` SHA-256 `4e96e728793985bf84ad9b9718329dfe05d0d691a22adbaff07a6c097d55f863`; `package-lock.json` SHA-256 `224ecd5cd89755b8ca4dc281b445a50fba209511c61a4d672d96b340fd32e503` |
| protected baseline | `shasum -a 256 .gitignore .devrites/ACTIVE .devrites/work/workspace-observation/touched-files.md` | `.` | exact three hashes from plan | no mutation | raw hashes and values |

Host execution prefixes each direct portable command with repository-required `rtk proxy` only when argv, cwd, exit code, stdout, and stderr remain unchanged. No combined/substituted command is authorized. A successful root proof bundle records candidate-before before command 1 and candidate-after after command 16 for the same 38-file engine candidate; observed repository-root cwd; absence/presence of candidate-affecting `DEVRITES_*` overrides; current hashes for every listed provenance input and generated/scanned tree; accepted plus observed protected preimages; and each command's exit, decisive signal, output bytes, and log SHA-256. After sealing logs and manifest read-only, a separate attestor validates every manifest file/log/hash/mode, recomputes current candidate/readiness/delivery and no-follow destination records, reconciles the delivery sidecar through an exact stable-record binding plus exact active filenames, records the manifest SHA-256, and seals the bundle. The proof runner validates both authorities. A partial sequence, self-declared digest without attestation, prefix-wide delta admission, or missing before/after identity is failed evidence and cannot be resumed as a successful bundle. Sealed root proof v8 observed workspace schema pass, behavioral schema 14 files/82 scenarios, instruction baseline 217 files/854894 bytes, shell/Node/Python syntax pass, module-selected Go 1.26.5 availability, race-enabled engine tests, and the full repository suite. These measurements must be recaptured after the accepted Review correction. Build/Prove recapture current provenance; preflight is not post-build proof.

## Workflow Artifact admission applicability

Workflow Artifact admission: not applicable — no active target admitted. This feature defines module contract and disposable fixture only; it MUST NOT materialize a persistent active-workspace executable. The completed one-target `demo` block and golden vector in `spec.md` are test input, not active admission.

## Shared contract proof

| Boundary | Artifact | Provider test | Consumer test |
| --- | --- | --- | --- |
| canonical module → adapters | `workflow-artifacts.md` tables plus the `workflow-artifact-adapter` JSON comment with keys module, entry, action, return | T-005, T-012 | T-010, T-012 |
| canonical module → generated hosts | same tables; generated adapter copies | T-012 private-stage generation | T-016 host parity |

## Prove walkthrough contract

Run exactly from repository root:

```sh
bash tests/workflow-artifact-identity-test.sh --prove-walkthrough
```

TTHW starts at command entry before toolchain/self-build and stops at first successful `CLEANED` plus restored cursor. Recovery/stale/idempotent scenarios execute after interval and do not change TTHW. Exact ordered output shape:

```text
WA-PROOF-001 PASS
tthw_ms=<nonnegative-base-10>
WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR
cursor=prove:/rite-prove demo
product_identity=unchanged
WORKFLOW_ARTIFACT_WALKTHROUGH PASS
```

No extra stdout. Private stderr is captured/bounded; public failure follows finite diagnostic line. Predicted TTHW comparison is 90000 ms. Proof subprocess command/aggregate limits remain 30/60 seconds with 2-second termination grace; they are not TTHW bound.

## Consumptive candidate-delivery gate

| Actor / action | Preconditions | Exact bounds | Failure behavior | Commit boundary |
| --- | --- | --- | --- | --- |
| bounded wright / prepare | root-frozen 16 authored states, 22 generated preimages, full generated/outside/protected manifests; exact 38-destination dispatch | private candidate-digest directory; current-user 0700/0600; complete writes/sync; one delivery owner; exact journal schema and compiled allowlists | interrupted `SNAPSHOTTING` validates/resumes its prefix and remaining rows; malformed/forged journal mutates nothing; root performs no product write | all 16/22 snapshots durable before authored replacement |
| hash-bound driver / private generation and install | validated exact 16/22 snapshot; `DEVRITES_HOST_ARTIFACT_DIR` points inside delivery directory; normally launched by wright, or exact DEC-042 launcher evidence when host window is insufficient | complete stage; differences subset exact 22, including the empty subset when staged bytes/modes equal live generated for all 22 (DEC-048); per-path durable intent; descriptor-held gate cwd; exact readback; generator and each gate run in a fresh process group with a 600-second command limit, 2-second TERM grace, 8,388,608-byte combined-output limit, and one 3,600-second monotonic aggregate deadline across generation plus all gates | output overflow, command timeout, aggregate timeout, surviving group member, nonzero exit, wrong signal, outside drift, or a destination outside its exact preimage/expected-post pair fails before `COMMITTED`; rollback validates all 38 current records before its first write and revalidates each immediately before restore/unlink; fresh-process recovery iterates compiled paths only; `ROLLING_BACK(n) → RESTORED → FAILED` restores 16/22 and full manifests | `COMMITTED` only after approved gates; then cleanup-only |

Writer-only modes:

```text
PYTHONDONTWRITEBYTECODE=1 bash "$DEVRITES_DELIVERY_DRIVER" --delivery-prepare
PYTHONDONTWRITEBYTECODE=1 bash tests/workflow-artifact-identity-test.sh --delivery-install "$DEVRITES_DELIVERY_DIR"
PYTHONDONTWRITEBYTECODE=1 bash tests/workflow-artifact-identity-test.sh --delivery-recover "$DEVRITES_DELIVERY_DIR"
```

Before any destination edit, the wright authors the exact candidate driver in private scratch, records its hash, runs syntax/default self-tests, sets `DEVRITES_DELIVERY_DRIVER`, and prepares the snapshot. Every prepare/install/recover mode requires `PYTHONDONTWRITEBYTECODE=1` at process entry. Installed dedicated-test destination must match the bootstrap hash before install/recover mode. Normally the wright launches the uninterrupted install/recover process and runs no unrelated command until a durable terminal state. Under DEC-042, only when the measured host tool window cannot hold that uninterrupted process, the coordinator may launch and monitor one already prepared install with the exact frozen driver hash, argv, cwd, and environment; it may not write destinations, alter controls, supply mutation/death/skip fixtures, reinterpret state, or run concurrent recovery. Recovery remains wright-only. Default mode drills the entry guard and same delivery recovery in disposable children. Normal generator runs once against private stage after authored checks; default target and generated hand edits are forbidden. Unchanged narrow checks are not rerun after full repository suite.

## Coverage diagram

```text
Vet grammar + relational limits + golden vector
  -> exact flock bootstrap + source promotion/rollover GC
  -> canonical WA-OP table + separate observer oracle
     -> complete stage/backup writes
     -> install/readback
     -> bounded proof process groups
     -> success cleanup + source GC -> CLEANED -> verify/return
     -> rollback + failure cleanup -> FAILED -> retry handoff
                                           -> success
                                           -> count 3 / epoch cap -> EXHAUSTED
  -> marker-owned evidence + product separation
  -> finite route precedence + diagnostics
  -> ten thin canonical adapters
  -> sole-wright private 16/22 delivery journal -> Claude/Codex derivatives

COVERAGE: AC-001–AC-007; EDGE-001–EDGE-007; PROH-001–PROH-012
GAPS: none planned
REGRESSIONS: stale migration, generic materializer, duplicate classifier, destructive default generation, whole-evidence overwrite
```

## Per-gap test requirements

| ID | Path / flow | Test file | Asserts (input → expected) | Kind | Slice | Priority |
| --- | --- | --- | --- | --- | --- | --- |
| T-001 | admission and encodings | `tests/workflow-artifact-identity-test.sh` | exact field and kind-specific reference-block grammar; return/action/cwd/signal/rollback/evidence mutants; golden bytes; checked relational minima use declared target capacity, not current rows; sparse-row/high-limit/min-minus-one/overflow → rejection before writes | contract | SLICE-001 | Regression-Critical |
| T-002 | owner concurrency | dedicated test | every fresh bootstrap child sets `umask(077)` before create; actual mkdir/open/sync/flock interruption and barrier-synchronized first-create race run under hostile inherited umask and independently observe exact `0700`/`0600`; deletion-of-umask, `lockf`, and `F_SETLK` mutants fail; injected unsupported host and stale generation → one owner or fail-closed; third observer proves loser zero writes | integrity | SLICE-001 | Regression-Critical |
| T-003 | source promotion/trust | dedicated test | every authority/source/ready boundary plus promotion-before-journal/binding rollover cleanup marker and lock-intent truncate/write/sync/readback/rename/unlink/clear/retry; authenticated empty suffix resumes; forged empty/malformed intent/swap/lookalike/unrecognized entry → fail closed; unrelated untouched | integrity | SLICE-001 | Regression-Critical |
| T-004 | complete stage/backup writes | dedicated test | partial positive writes → full bytes; bool/non-int/zero/negative/oversized/exhaustion/ENOSPC/error/death → bounded recovery of exact named partial only | integrity | SLICE-001 | Regression-Critical |
| T-005 | operation table/trace independence | dedicated test | parsed 16-row table dispatches each operation ID to its real descriptor-relative filesystem/journal transition in one consumer reused by owner, retry handoff, both exhaustion causes, all three source-loss classes, product separation, and walkthrough; actual child dies before intent and after effect for each row; fresh child resumes; exact five-field result and terminal source/history/product facts are derived by a separate OS/engine observer without consuming the row's claimed postcondition; disconnected tuple/route checks, row-hash effects, consumer reports, and every operation/observer mutant fail independently | contract | SLICE-001 | Regression-Critical |
| T-006 | install/rollback branches | dedicated test | failure/death at each replacement/readback → only frozen or exact preimage states; before first rollback write and immediately before each restore/unlink every destination must equal its exact preimage or expected-post record; present-file and originally-absent third-state mutants → fail closed with zero rollback writes; pre-proof rollback; no post-proof rollback | integration | SLICE-001 | Regression-Critical |
| T-007 | proof and delivery process groups | dedicated test | proof and delivery generator/gates: nonzero/wrong signal/hang/descendant/output cap/command+aggregate timeout → terminate/grace/kill/reap + fixed boundary + rollback; delivery uses exact 600-second process, 3,600-second aggregate, 8,388,608-byte output, and 2-second grace constants without runtime configuration | integration | SLICE-001 | Regression-Critical |
| T-008 | retry/exhaustion | dedicated test | shared lifecycle + observer proves correction → next epoch success with retained source and immutable history; handoff death → same epoch; same fingerprint counts 1/2/3 → durable `same-fingerprint-count`, observed source GC, no attempt 4; distinct fingerprints retain history; total epoch cap at current count below three → durable `total-epoch-limit`, observed source GC, blocked handoff; exact journal and marker-owned evidence retain cause | integration | SLICE-001 | Regression-Critical |
| T-009 | evidence ownership | dedicated test | absent marker and arbitrary prior EVID/prefix/suffix → outside bytes exact; one candidate line; malformed/duplicate/nested/over-budget → pre-mutation failure | integrity | SLICE-001 | Regression-Critical |
| T-010 | route/diagnostic/caller map | dedicated + routing test | fixed independent full route maps bind owner, exact slash action with the active slug argument, `phase`, `status`, `next_action`, and caller-cursor behavior for overlapping triggers; `PLAN_VET_REPAIR` executes Plan repair then Vet internally with that slug and `OFFLINE_RECOVERY` executes Debug Recovery, disposable re-preflight, then narrow Vet with that slug; exact source rows retain the frozen literal command forms from `spec.md`; every field/action/slug mutant fails; all finite diagnostics → exact bounded ASCII line; unrecognized/hostile values safe | contract | SLICE-001 | Regression-Critical |
| T-011 | provider-neutral scenarios | corpus + dedicated test | fixed independent full maps bind exact ten IDs/triggers/routes/consequences/forbidden actions; every field mutant fails; generic runner validates schema only | eval contract | SLICE-001 | Regression-Critical |
| T-012 | thin adapters | dedicated + routing test | every actual adapter has exactly one bounded structured declaration and no policy restatement outside it; fixed independent four-field map proves exact link/entry/action/return cardinality; per-field and duplicate-declaration mutants plus concrete paraphrased owner/classifier/retry-policy prose run against adapter-file copies and fail independently; generated host parity preserves declaration and absence of duplicate prose | contract | SLICE-001 | Regression-Critical |
| T-013 | product separation | dedicated test with actual self-built engine | freeze candidate digest, readiness binding, and built-slice count outside consumer before lifecycle; independently recompute all three after proof → equality alone emits `unchanged`; mutate each dimension → `WA-R018`/`BLOCKED_GATE`; touched-files injection rejected | integration | SLICE-001 | Regression-Critical |
| T-014 | sole-wright candidate delivery | private hash-bound bootstrap driver + actual installed writer modes + host test | one expanded registry is shared by every `delivery_death` call and test enumerator; fresh actual prepare/install/recover processes die/restart at every registry entry; delivery-mode mutation fixtures copy only the exact compiled 16 authored regular files and assert that exact inventory, never the repository root or ignored runtime state; before cleanup an independent expected post-install record is frozen from staged bytes for every compiled destination, and a separate observer enumerates exact 16/22 cardinality/state/mode/digest at `COMMITTED`, `CLEANING`, and `CLEANED` plus terminal journal relation; corruption/removal mutants fail; missing/duplicate/unexecuted entries fail; `SNAPSHOTTING` resumes; schema/path/backup/state/hash and present/absent third-state destination forgeries mutate nothing; recursive outside observer excludes only root `.git` plus the exact delivery directory and descendants, so nested `.git` and lookalike sibling drift fail; bounded generator/gate overflow, timeout, aggregate, descendant, and mutant-unbounded paths fail; descriptor-held gates and restore/cleanup remain exact | integration | SLICE-001 | Regression-Critical |
| T-015 | root-caller DX | `bash tests/workflow-artifact-identity-test.sh --prove-walkthrough` | actual one-target lifecycle observes first `CLEANED`, restored cursor, and independently equal frozen/post product identities before recording TTHW ≤90-second prediction; only afterward actual interrupted/stale/idempotent journeys emit captured safe lines; product/candidate/readiness/built-count mutation blocks and cannot print success; no literal outcome synthesis or real action | journey | SLICE-001 | P1 |
| T-016 | repository regression | validators/race/full suite | candidate → all existing checks pass, instruction ≤855000, protected/outside hashes unchanged | regression | SLICE-001 | Regression-Critical |
| T-017 | prior Reslice preservation | `tests/acceptance-preserving-reslice-policy-test.sh` plus shared-path assertions | five authored files change additively and eight mirrors derive normally → Reslice linkage/packet/routes/actions/stops/baseline remain valid; all 30 non-overlap bytes/workspace records exact | regression | SLICE-001 | Regression-Critical |

## Failure injection matrix

| Boundary | Injected failure | Required state/result |
| --- | --- | --- |
| owner/journal | real child-process bootstrap death, synchronized two-root create race, canonical contention, alternate-lock mutant, unsupported host, stale generation/hash | exact private bootstrap then one flock owner; separate observer proves loser zero journal/target writes or exact previous generation |
| source promotion | death around promotion and stale-cleanup marker/lock-intent/rename/unlink/intent-clear; binding rollover; invalid authority/ready/forged empty suffix/unrecognized entry | exact authenticated promotion/rollover GC or finite route; unrelated untouched |
| stage/backup | every write-progress defect and operation death | exact partial reconciled; target unchanged; retryable `FAILED` |
| install/readback | replace failure, omitted handle, target drift | exact rollback/preimage or blocked safety gate |
| proof | wrong signal, timeout, ignoring descendants, output overflow | full process-group reap then rollback before `PROVED` |
| retry | death at handoff, same/distinct fingerprint, epoch cap | immutable history; same epoch resume; finite exhaustion |
| success cleanup | missing source, cleanup failure, death | proved targets preserved; resume cleanup; source GC before `CLEANED` |
| evidence | existing lifecycle bytes, malformed markers, line cap | preserve outside bytes or fail before mutation |
| candidate delivery | actual-mode child death/failure at every snapshot/stage/install/proof/rollback/commit/cleanup boundary; partial snapshot; forged path/backup/state/hash/counter; ancestor replacement during gate; staged extra sibling; root-install attempt | fresh process resumes exact prefix or fails before mutation; pre-commit restores compiled 16/22 and full manifests; held gate cwd stays original tree; root writes nothing; post-commit cleanup only |
| prior Reslice overlap | additive authored edit or derived mirror breaks prior semantics, or any of 30 non-overlap bytes changes | dedicated Reslice proof fails; wright restores all states before `COMMITTED` |

## Acceptance → test map

- AC-001 → T-005, T-010, T-012, T-017  # shared-contract consumers parse the same canonical tables/JSON declaration
- AC-002 → T-001, T-002, T-003
- AC-003 → T-004, T-005, T-006, T-008
- AC-004 → T-002–T-010
- AC-005 → T-009, T-013
- AC-006 → T-005, T-010, T-011, T-012, T-015
- AC-007 → T-013, T-014, T-016, T-017

No interactive UI surface; interaction inventory and browser proof are not applicable.

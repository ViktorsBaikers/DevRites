# Traceability: Workflow Artifact Identity

DevRites contract: devrites.readiness-artifacts.v2

## Coverage matrix

| Contract | Slice | Test / proof | Evidence | Touched files | Status |
| --- | --- | --- | --- | --- | --- |
| REQ-001 / AC-001 | SLICE-001 | canonical link, stale/generic deletion, ten adapter rows/thinness, prior Reslice dedicated regression across 13 shared paths | EVID-001 | module + ten adapters + generated mirrors | proven |
| REQ-002 / AC-002 | SLICE-001 | exact grammar/encoding/golden vector plus checked relational limit minimum-minus-one/exact/overflow fixtures | EVID-002 | module + Vet adapter + dedicated test | proven |
| REQ-003, REQ-004 / AC-002, AC-004 | SLICE-001 | exact handle/identity; source promotion and binding-rollover stale-cleanup marker/rename/unlink/retry; held bytes; swap/forged/unknown/loss/terminal GC | EVID-003 | module + dedicated test | proven |
| REQ-005, REQ-006 / AC-003, AC-004 | SLICE-001 | umask-077 mkdirat/openat and only `fcntl.flock`; first-create/mutant contention; parsed operation table; complete writes; every interruption; retry/exhaustion | EVID-004 | module + dedicated test | proven |
| REQ-007 / AC-003, AC-004 | SLICE-001 | phase-qualified source loss; proof command/aggregate timeout; process-group terminate/kill/reap; source-free idempotent verification | EVID-005 | module + dedicated test | proven |
| REQ-008 / AC-005 | SLICE-001 | marker-owned evidence preservation, one candidate binding, immutable attempts, actual self-built-engine candidate/readiness/built-count equality | EVID-006 | module + dedicated test | proven |
| route/diagnostic/adapter tables / AC-006 | SLICE-001 | exact five-field oracle with separate OS/engine observer; precedence/diagnostics/corpus/adapters; exact `--prove-walkthrough` output/TTHW | EVID-007 | module + behavioral corpus + ten adapters + dedicated test | proven |
| REQ-009 / AC-007 | SLICE-001 | bounded wright authors candidate; hash-bound driver is sole private 16/22 filesystem writer; generation/install interruption/restart/rollback; outside equality, parity/instruction/repository suites; root no-product-write; exact DEC-042 launcher evidence when applicable | EVID-011 | canonical/test/baseline + generated mirrors | proven |
| REQ-009 / AC-007 | SLICE-001 | protected `.gitignore` / `.devrites/ACTIVE` / Workspace Observation preimages plus Reslice 13-shared/30-non-overlap boundary | EVID-008 | protected inputs + dedicated Reslice gate | proven |
| REQ-003–REQ-008 / AC-002–AC-006 | SLICE-001 | adversarial Build doubt and narrow correction recheck; independent candidate/proof identity match; Critical 0 and Important 0 | EVID-009 | canonical module + dedicated proof driver + immutable root proof | proven |
| REQ-001–REQ-009 / AC-001–AC-007 | SLICE-001 | historical fresh-context whole-feature Prove of sealed v5; superseded by EVID-011 and does not support current Seal | EVID-010 | prior 38-file candidate + historical v5 bundle | historical |
| REQ-001–REQ-009 / AC-001–AC-007 | SLICE-001 | v10 16-command immutable Prove run, separate exact-delta attestation, independent reconstruction, and fresh proof-runner acceptance map after Q-011/Q-012 | EVID-011 | complete 38-file candidate + sealed v10 proof bundle | historical |
| REQ-001–REQ-009 / AC-001–AC-007 | SLICE-001 | v11 16-command immutable Prove run after Q-013/DEC-050 schema lock, separate exact-delta attestation, independent reconstruction, and fresh proof-runner | EVID-012 | complete 38-file candidate + sealed v11 proof bundle | proven |
| REQ-009 / AC-007 | SLICE-001 | Seal-correction CLEANED delivery `8fd99161…`, empty generated delta, DEC-042 launcher, asserting workspace-schema lock | EVID-012 | dedicated test + 16/22 journal + v11 bundle | proven |
| REQ-009 / AC-007 | SLICE-001 | Seal-correction locks for Important residuals (production fixture argv reject + held-generator mkdirat/plant); driver `35b1ce7e…` superseded by EVID-014 retry driver `828ce648…`; FAILED `58c32710…` preserved | EVID-013 | dedicated test + seal-correction mutants | proven |
| REQ-001–REQ-009 / AC-001–AC-007 | SLICE-001 | v12 16-command immutable Prove after EVID-013 mapping lock, DEC-055 stage/backups strip, CLEANED delivery `8f1e14e3…`, separate attestation, fresh proof-runner | EVID-014 | complete 38-file candidate + sealed v12 proof bundle | historical |
| REQ-009 / AC-007 | SLICE-001 | Seal-correction nested OUT_ROOT basename plant; driver `bda33a29…`; dedicated suite PASS; superseded standalone Prove by EVID-016 | EVID-015 | dedicated test only | proven |
| REQ-001–REQ-009 / AC-001–AC-007 | SLICE-001 | v13 16-command immutable Prove after EVID-015, CLEANED delivery `71411cf0…` (DEC-042), empty generated delta (DEC-048), separate attestation, fresh proof-runner | EVID-016 | complete 38-file candidate + sealed v13 proof bundle | proven |

## Edge and prohibition coverage

| ID | Slice | Executable backstop | Status |
| --- | --- | --- | --- |
| EDGE-001 | SLICE-001 | exact grammar/golden vector/mode/index/order plus relational minima/headroom/overflow and hostile cells/paths | proven |
| EDGE-002 | SLICE-001 | exact bootstrap/flock/generation plus resolver/held bytes/swap/forged/unknown and stale-binding rollover GC | proven |
| EDGE-003 | SLICE-001 | parsed operation table; separate observer oracle; source promotion/stale GC/complete writes/evidence/retry/every interruption | proven |
| EDGE-004 | SLICE-001 | success/failure/retry/exhaustion graph, proof process-group timeout, and three source-loss classes | proven |
| EDGE-005 | SLICE-001 | marker-owned evidence preservation and actual self-built-engine candidate/readiness/built-count equality | proven |
| EDGE-006 | SLICE-001 | exact route/action/diagnostic tables, ten scenarios/adapters, five-field observer oracle, exact Prove flag/TTHW interval | proven |
| EDGE-007 | SLICE-001 | sole-wright 16/22 delivery journal, private generation, install/restart/rollback, full outside equality, parity/instruction/repository proof | proven |
| PROH-001 | SLICE-001 | no engine command/generic materializer; outside-allowlist equality | proven |
| PROH-002 | SLICE-001 | no `Cold-resume migration`, writer-exhaustion authority, or backfill | proven |
| PROH-003 | SLICE-001 | touched-files rejection plus actual candidate/readiness/built-count equality | proven |
| PROH-004 | SLICE-001 | no identity/source synthesis in missing/stale fixtures | proven |
| PROH-005 | SLICE-001 | adapter structural assertions reject wright/read-only authorship | proven |
| PROH-006 | SLICE-001 | generator-only 22-path transaction and parity | proven |
| PROH-007 | SLICE-001 | fixture/public routes stop before real action and release mutation | proven |
| PROH-008 | SLICE-001 | held-descriptor immutable-byte swap mutant rejects validate/reopen implementation | proven |
| PROH-009 | SLICE-001 | root product writes/default generator prohibited; hash-bound driver alone writes private 16/22 delivery journal, sibling equality/restart/rollback; DEC-042 launcher cannot write or recover | proven |
| PROH-010 | SLICE-001 | marker-owned update preserves prior lifecycle bytes and one candidate binding | proven |
| PROH-011 | SLICE-001 | adapters link canonical tables; five-field oracle uses separate OS/engine observer; all fields mutate independently | proven |
| PROH-012 | SLICE-001 | exact flock bootstrap, proof reap, retry exhaustion, rollover stale GC, and terminal source GC | proven |

## Route coverage

| Scenario ID | Slice | Deterministic assertion | Status |
| --- | --- | --- | --- |
| WA-ADMISSION-SUCCESS | SLICE-001 | `ROOT_TRANSACTION`; first active state `PREPARING`; no wright/slice charge | proven |
| WA-MISSING-IDENTITY | SLICE-001 | `PLAN_VET_REPAIR`; zero writes; no synthesis | proven |
| WA-STALE-IDENTITY | SLICE-001 | `PLAN_VET_REPAIR`; unrelated files untouched | proven |
| WA-STALE-WRITER-EXHAUSTION | SLICE-001 | `PLAN_VET_REPAIR`; no migration/backfill | proven |
| WA-FIRST-ROOT-FAILURE | SLICE-001 | `OFFLINE_RECOVERY`; exact preimages; `FAILED`; attempt one | proven |
| WA-REPLACEMENT-ROLLBACK | SLICE-001 | rollback/failure-cleanup branch; no partial set | proven |
| WA-CLEANUP | SLICE-001 | post-`PROVED` cleanup resumes; proved targets preserved | proven |
| WA-IDENTITY-CONTINUITY | SLICE-001 | `PROVE_AND_RETURN`; product identities unchanged | proven |
| WA-COMPLETED-HISTORICAL | SLICE-001 | `NO_BACKFILL`; no writes/reopen | proven |
| WA-IDEMPOTENT-RERUN | SLICE-001 | `VERIFY_EXISTING`; no reinstall/budget charge | proven |

## Operational route coverage

| Fixture | Slice | Deterministic assertion | Status |
| --- | --- | --- | --- |
| WA-OWNER-BUSY | SLICE-001 | `WAIT_ACTIVE_OWNER`; zero journal/target/state writes | proven |
| WA-RETRY-SUCCESS | SLICE-001 | attempt 1 `FAILED`; correction/preflight; durable epoch 2 handoff; success | proven |
| WA-RETRY-EXHAUSTED | SLICE-001 | same fingerprint counts 1/2/3; `EXHAUSTED`; source GC; no attempt 4 | proven |
| WA-RETRY-HANDOFF-DEATH | SLICE-001 | restart resumes same epoch; prior row immutable | proven |
| WA-RETRY-DISTINCT-FINGERPRINT | SLICE-001 | new Critical/Important fingerprint starts independent count without erasing history | proven |
| WA-PROOF-TIMEOUT | SLICE-001 | terminate/grace/force-kill/reap then pre-proof rollback; fixed diagnostic | proven |
| WA-EVIDENCE-PRESERVE | SLICE-001 | arbitrary prior lifecycle bytes survive every journal generation; one candidate binding | proven |
| WA-STALE-SOURCE-ROLLOVER | SLICE-001 | promotion-before-journal crash then new binding → durable stale-cleanup GC and current promotion; unknown entry untouched | proven |
| WA-GENERATED-ROLLBACK | SLICE-001 | driver death/failure before `COMMITTED` returns to bounded wright recovery, which resumes/restores all 16/22 states; coordinator launcher cannot recover; root writes nothing; siblings unchanged | proven |

## Wiring

| Producer | Consumer | Invariant | Integration proof |
| --- | --- | --- | --- |
| Vet admission | owner/source resolver | exact grammar + active slug/current binding derive limits and stable handle | parser/golden/forged/stale fixtures |
| owner bootstrap/flock | journal/transaction | exact private names and one interoperable lock domain own generation/hash compare | create-race/interruption/flock-contention/lockf-mutant fixtures |
| retained source | preparation | held bytes supply hash/stage; pre-journal stale binding uses validated durable rollover GC | swap, rollover crash/retry, unknown-entry, terminal-GC fixtures |
| canonical operation table | disposable consumer/observer | consumer parses table; separate process derives OS/engine assertions for exact five-field oracle | semantic-column and oracle-field mutation fixtures |
| frozen identity | marker-owned journal/transaction | identity plus Vet binding precede active writes; outside evidence bytes preserved | pre-journal/evidence-marker/generation assertions |
| durable journal | resume/recovery | intent precedes operation; exact pre/post/declared-partial states; immutable retry rows | every-boundary kill/resume/retry/exhaustion matrix |
| transaction | Prove | complete frozen set or preimages; bounded process groups; post-proof never rollback | branch/source-loss/timeout/reap matrix |
| Workflow Artifact evidence | product identity | workflow paths remain excluded; candidate binding singular | actual self-built-engine before/after fixture |
| prior Reslice shared preimages | bounded sole wright | five authored additive edits and eight normal mirrors preserve linkage/packet/routes/actions/stops/baseline; prior workspace/all 30 non-overlap paths exact | dedicated Reslice test plus shared/non-overlap hashes |
| root frozen manifests | bounded wright + hash-bound driver | exact 16 authored/22 generated delivery boundary; driver is sole filesystem writer; root performs no product write; DEC-042 launcher identity is exact when used | actor/allowlist/launcher assertions |
| wright-authored canonical Markdown | private generated stage | module-selected toolchain and complete staged tree differ only at 22 admitted paths | staged/full-manifest validation |
| driver delivery journal | Claude/Codex destinations | durable per-path intent, restart, pre-commit 16/22 restore by bounded recovery wright, siblings unchanged | interruption/host parity/rollback/outside equality |

## Orphan check

Every requirement, acceptance criterion, edge, prohibition, stable route, and T-001–T-017 requirement maps to `SLICE-001` and executable proof. The 13 shared Reslice destinations—five authored and eight generated—have a dedicated composition gate; no orphan slice or unmapped criterion remains.

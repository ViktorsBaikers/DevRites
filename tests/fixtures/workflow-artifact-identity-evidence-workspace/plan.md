# Plan: Workflow Artifact Identity
Budget override: One atomic 38-destination slice needs co-located writer, transaction, failure, proof, and rollback contract; splitting would create competing authority.

## Approach

Deliver one atomic `SLICE-001`. Canonical authority, ten thin adapters, provider-neutral routes, deterministic transaction/interruption proof, existing assertion replacement, instruction baseline, and generated host mirrors cannot ship independently without leaving duplicate authority or unproved resume behavior.

Implementation stages are ordered within this slice:

1. deepen canonical contract;
2. add deterministic identity, preparation, transaction, interruption, resume, and product-separation fixture;
3. thin adapters and replace stale routing/host assertions;
4. add ten-scenario behavioral corpus;
5. run normal generation and baseline refresh;
6. prove exact candidate and outside-allowlist preservation.

No generic materializer, engine behavior, dependency, hook, schema, feature flag, compatibility migration, actual feature workflow executable, or historical backfill is added.

Workflow Artifact admission: not applicable — no active target admitted. Vet records that exact sentence in `test-plan.md` and MUST NOT emit an active admission block for this slug.

## Slice strategy

One vertical `SLICE-001` closes AC-001–AC-007. Contract, deterministic consumer, canonical adapters, behavioral corpus, generated adapters, and full separation proof form one indivisible release unit. Internal stages remain serial under one writer; read-only proof and review may fan out only after candidate freezes.

## Exact authored writer allowlist

1. `pack/.claude/skills/devrites-lib/reference/standards/workflow-artifacts.md`
2. `pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`
3. `pack/.claude/skills/devrites-lib/reference/standards/one-shot-actions.md`
4. `pack/.claude/skills/devrites-debug-recovery/SKILL.md`
5. `pack/.claude/skills/rite-autocomplete/SKILL.md`
6. `pack/.claude/skills/rite-autocomplete/reference/loop.md`
7. `pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md`
8. `pack/.claude/skills/rite-build/SKILL.md`
9. `pack/.claude/skills/rite-build/reference/phase-contract.md`
10. `pack/.claude/skills/rite-prove/SKILL.md`
11. `pack/.claude/skills/rite-vet/SKILL.md`
12. `evals/behavioral/workflow-artifact-identity.json`
13. `tests/workflow-artifact-identity-test.sh`
14. `tests/phase-gate-routing-test.sh`
15. `tests/host-artifacts-test.sh`
16. `tests/instruction-size-baseline.json`

One bounded serial `devrites-slice-wright` authors these 16 candidate destinations; one hash-bound delivery driver is the sole filesystem writer/restorer across them and the exact 22 generated destinations below: one 38-destination writer boundary. Root never edits/copies/restores any source, test, or generated path. Normally the wright launches the driver; DEC-042 permits the coordinator only to launch and monitor the exact prepared hash/argv/cwd/environment when the measured host tool window cannot hold the uninterrupted transaction. New corpus and dedicated test are required outputs. Generated files are never hand-edited; only private-stage generator bytes may reach them through the driver-owned delivery journal.

## Exact generated allowlist

Normal `scripts/build-host-artifacts.sh` generation may change exactly these 22 derivatives and no others:

Claude:

1. `pack/generated/claude/skills/devrites-lib/reference/standards/workflow-artifacts.md`
2. `pack/generated/claude/skills/devrites-lib/reference/standards/afk-hitl.md`
3. `pack/generated/claude/skills/devrites-lib/reference/standards/one-shot-actions.md`
4. `pack/generated/claude/skills/devrites-debug-recovery/SKILL.md`
5. `pack/generated/claude/skills/rite-autocomplete/SKILL.md`
6. `pack/generated/claude/skills/rite-autocomplete/reference/loop.md`
7. `pack/generated/claude/skills/rite-autocomplete/reference/stop-conditions.md`
8. `pack/generated/claude/skills/rite-build/SKILL.md`
9. `pack/generated/claude/skills/rite-build/reference/phase-contract.md`
10. `pack/generated/claude/skills/rite-prove/SKILL.md`
11. `pack/generated/claude/skills/rite-vet/SKILL.md`

Codex:

12. `pack/generated/codex/skills/devrites-lib/reference/standards/workflow-artifacts.md`
13. `pack/generated/codex/skills/devrites-lib/reference/standards/afk-hitl.md`
14. `pack/generated/codex/skills/devrites-lib/reference/standards/one-shot-actions.md`
15. `pack/generated/codex/skills/devrites-debug-recovery/SKILL.md`
16. `pack/generated/codex/skills/rite-autocomplete/SKILL.md`
17. `pack/generated/codex/skills/rite-autocomplete/reference/loop.md`
18. `pack/generated/codex/skills/rite-autocomplete/reference/stop-conditions.md`
19. `pack/generated/codex/skills/rite-build/SKILL.md`
20. `pack/generated/codex/skills/rite-build/reference/phase-contract.md`
21. `pack/generated/codex/skills/rite-prove/SKILL.md`
22. `pack/generated/codex/skills/rite-vet/SKILL.md`

Same bounded wright stages authored changes, runs normal generator only into private same-filesystem output, validates full staged tree, and uses feature-local delivery journal to install exact generated allowlist. Any delta outside list, interruption, or proof failure before `COMMITTED` restores all 16 authored/22 generated preimages. Root freezes and verifies only.

## Prior sealed-candidate overlap

Workflow Artifact shares 13 destinations with the sealed Reslice candidate: five authored preimages and eight generator-derived mirrors.

Authored:

1. `pack/.claude/skills/rite-autocomplete/SKILL.md`
2. `pack/.claude/skills/rite-autocomplete/reference/loop.md`
3. `pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md`
4. `pack/.claude/skills/rite-vet/SKILL.md`
5. `tests/instruction-size-baseline.json`

Generated:

6. `pack/generated/claude/skills/rite-autocomplete/SKILL.md`
7. `pack/generated/claude/skills/rite-autocomplete/reference/loop.md`
8. `pack/generated/claude/skills/rite-autocomplete/reference/stop-conditions.md`
9. `pack/generated/claude/skills/rite-vet/SKILL.md`
10. `pack/generated/codex/skills/rite-autocomplete/SKILL.md`
11. `pack/generated/codex/skills/rite-autocomplete/reference/loop.md`
12. `pack/generated/codex/skills/rite-autocomplete/reference/stop-conditions.md`
13. `pack/generated/codex/skills/rite-vet/SKILL.md`

Writer treats current authored bytes as preimages and performs additive semantic edits; normal private-stage generation derives the eight shared mirrors. Both forms must preserve Acceptance-preserving Reslice canonical linkage, packet/route behavior, action/stop rules, and baseline entries while adding Workflow Artifact entry/action/return rules. `bash tests/acceptance-preserving-reslice-policy-test.sh` is mandatory before candidate commit boundary and in root proof. Any regression restores all 16/22 states. Reslice workspace records and all 30 non-overlap candidate paths remain byte-identical; its previous standalone candidate digest becomes historical when shared bytes change and is never represented as current proof. Workspace Observation has zero destination overlap; its manifest hash remains protected.

## Inspected and OUT

Direct references inspected and intentionally unchanged unless proof demonstrates drift:

- `pack/.claude/skills/devrites-lib/reference/standards/agents.md`
- `pack/.claude/skills/devrites-lib/reference/standards/README.md`
- `pack/.claude/skills/devrites-lib/reference/standards/skill-authoring.md`
- `pack/.claude/skills/rite-build/reference/wright-dispatch.md`
- `pack/.claude/skills/rite-plan/SKILL.md`
- `pack/.claude/agents/devrites-plan-drafter.md`
- `scripts/build-host-artifacts.sh`
- `scripts/run-behavioral-evals.sh`
- existing corpus validators
- `README.md` SHA-256 `6527b059fcc27d85afedfd98079de8b94713d68c2efef6e0f494fd4ec1f7e8f4` — not Workflow Artifact authority; stale “root-materializer” / missing-writer migration copy stays historical operator docs
- `docs/skills.md` SHA-256 `63c2f41a97fdc35dc177ac50c0860d82956347bafda07a2fd5c2639327bd25d0` — not Workflow Artifact authority; same historical operator copy

Any discovered need to edit these paths returns Vet; writer cannot widen allowlist. Adding README or skills catalogue to the 38-file dest set is out of scope.

## Canonical contract change

Replace stale `Cold-resume migration` and generic-materializer wording with one interface:

- one serializable Vet admission grammar with exact path/mode/behavior/interface/fixture/proof/rollback/evidence rows; kind-specific complete reference-block schemas; exact return/cwd/signal/rollback/evidence grammars; checked arithmetic using declared target capacity; and relational byte/file/journal/attempt/time minima;
- exact NUL/uint32/domain-separated handle and identity encodings plus golden vector;
- umask-`077` `mkdirat`/`openat` bootstrap for exact namespace/lock, only Python `fcntl.flock(LOCK_EX|LOCK_NB)`, no `lockf`/`F_SETLK`, held `O_CLOEXEC` descriptor, monotonic generation, and owned-section compare;
- disposable authorship and atomic retained-source promotion under exact `wsrc:<64hex>` handle and `.workflow-artifact-sources/<hex>` mapping;
- deterministic `.<hex>.preparing` authority/indexed-source/ready protocol plus pre-journal binding-rollover `.stale-cleanup` → `.<hex>.stale-cleaning` GC, with exact durable stale intent in the held lock descriptor authenticating partial/empty deletion suffixes, complete writes/sync, and every-boundary recovery;
- no-follow held-descriptor read with hash and stage built from same bounded immutable bytes;
- canonical machine-readable operation/transition table covering source promotion, journal init, stage, backup, install, proof, rollback, failure/success cleanup, retry, exhaustion GC, evidence, product separation, and idempotent verify;
- strict complete-write loop and declared partial stage/backup recovery;
- success, pre-install failure, pre-proof rollback, retry, independently diagnosed same-fingerprint and total-epoch exhaustion, and post-proof preserve branches;
- fresh proof process groups with command/aggregate timeout, terminate/grace/force-kill/reap, and bounded private output;
- marker-owned atomic/synced `evidence.md` section preserving all outside bytes and exactly one standalone candidate binding;
- finite route precedence, route/action/cursor table, and reason/boundary diagnostic taxonomy;
- source retained through retryable `FAILED` and garbage-collected before `CLEANED`/`EXHAUSTED`; source absence after `CLEANED` expected;
- product candidate/readiness/built-slice separation; caller-cursor restoration; stop-before-action.

Adapters retain phase-local entry/action/return only. They invoke canonical classifier/operation table and never restate transaction or retry behavior.

## Per-adapter entry/action/return

| Canonical adapter | Entry trigger | Canonical action | Return cursor |
| --- | --- | --- | --- |
| `devrites-lib/reference/standards/afk-hitl.md` | unattended root reaches current admitted Workflow Artifact work | invoke classifier; execute returned route without wright/slice charge | saved lifecycle phase/action; no intermediate reply |
| `devrites-lib/reference/standards/one-shot-actions.md` | workflow proof completes before any consumptive one-shot action | `PROVE_AND_RETURN`; require fresh real-action authorization | saved one-shot action boundary |
| `devrites-debug-recovery/SKILL.md` | durable active failure or ambiguous admitted state | `OFFLINE_RECOVERY`; correct offline, re-preflight, narrow Vet, retry only under cap | saved caller or exact Plan/Vet route |
| `rite-autocomplete/SKILL.md` | lifecycle cursor encounters admitted set or resumable journal | invoke classifier; execute returned route internally | saved phase/action; zero intermediate reply |
| `rite-autocomplete/reference/loop.md` | loop tick sees Workflow Artifact trigger/state | invoke classifier once under owner lock; no actor-history migration | same loop cursor; no budget charge for verify/rerun |
| `rite-autocomplete/reference/stop-conditions.md` | classifier returns owner-busy, exhausted, or existing hard gate | stop on exact `WAIT_ACTIVE_OWNER`, `BLOCKED_EXHAUSTED`, or `BLOCKED_GATE` result | unchanged cursor plus fixed route-owned output |
| `rite-build/SKILL.md` | Vet-ready admitted bytes require root authorship outside product wright | `ROOT_TRANSACTION`; root writes only admitted `.devrites/**` targets | saved Build slice cursor; wright product allowlist unchanged |
| `rite-build/reference/phase-contract.md` | Build gate enters or resumes transaction | invoke canonical operation table; reconcile exact result | same slice/checkpoint cursor or Plan/Vet route |
| `rite-prove/SKILL.md` | Prove consumes installed Workflow Artifact or `CLEANED` rerun | `VERIFY_EXISTING` or admitted proof path ending `PROVE_AND_RETURN` | saved Prove cursor; stop before real action |
| `rite-vet/SKILL.md` | plan declares root-authored executable workflow file | emit exact admission; stale/missing authority uses `PLAN_VET_REPAIR` | Vet READY cursor or exact technical replan |

`one-shot-actions.md` keeps consumptive-action rules. Autocomplete/AFK delete writer-exhaustion migration. Build/Recovery delete transaction duplication. Vet owns admission only; Prove consumes frozen identity/evidence only.

## Shared contract proof

| Boundary | Canonical contract artifact | Provider-side asserting test | Consumer-side asserting test |
| --- | --- | --- | --- |
| canonical module → ten adapters | `pack/.claude/skills/devrites-lib/reference/standards/workflow-artifacts.md` operation/route/diagnostic tables plus the ten-row adapter table | `tests/workflow-artifact-identity-test.sh` parses those tables and the exact adapter JSON declaration | `tests/phase-gate-routing-test.sh` consumes the same four adapter fields and route/action/cursor outcomes |
| canonical module → Claude/Codex mirrors | same `workflow-artifacts.md` tables plus generated copies of the ten adapters | dedicated test private-stage generation plus declaration/cardinality mutants | `tests/host-artifacts-test.sh` proves generated parity against the same canonical bytes |

Both consumers parse the provider artifact; neither copies a local contract. Binds: one table-driven interface. Prevents: a second classifier or restated transaction policy.

## Validation strategy

Test from interface inward: admission/identity/source invariants, every durable transaction boundary, resume routes, product separation, thin adapters, provider-neutral corpus, normal host generation, instruction cap, repository validation, engine race suite, and full tests. Root captures exact command, exit, decisive signal, observed cwd, candidate-affecting environment, current prerequisite provenance, accepted protected preimages, candidate identity before and after the complete approved sequence, and per-log bytes/hashes; a separate attestation binds the final manifest digest before the fresh proof runner validates immutable logs.

## Deterministic proof design

`tests/workflow-artifact-identity-test.sh` parses exact canonical operation/transition, route, diagnostic, scenario, and adapter tables from `workflow-artifacts.md` and drives one disposable standard-library filesystem consumer. That consumer dispatches every parsed `WA-OP-*` ID to its real descriptor-relative filesystem/journal transition, persists real intent before each effect, exposes deterministic ready barriers, is terminated in a child process before intent and after effect for every row, and is resumed by a fresh process from only held filesystem/journal state. Owner bootstrap, retry handoff, both exhaustion causes, all three phase-qualified source-loss classes, product separation, and the one-target walkthrough reuse this consumer rather than local substitutes. The observer alone decides terminal truth: retained source exists for retryable `FAILED`, is absent before `EXHAUSTED`, exact immutable epoch history and `exhaustion_cause` are durable, and frozen pre-transaction product candidate/readiness/built count equal independently observed post-proof values before success is emitted. A separate observer process derives file type/mode/link/hash, journal outside/owned bytes, process-group liveness/reap, route/action/state/cursor, and actual engine product identity directly from OS/engine; it never consumes the parsed row's claimed postcondition, a consumer-reported invariant, or consumer classifier/transition/route/reporting code. Fixed independent maps bind exact five-field operation records, every route/scenario field, and every adapter entry/action/return field. Mutations remove/reorder rows, alter every semantic table/map field, duplicate policy under synonymous prose, and must fail at the observed surface. Test-only consumer/observer are not shipped or referenced as runtime authority; row hashes, tuple equality, boundary labels, and in-memory delivery simulation alone are explicitly insufficient.

If `DEVRITES_ENGINE_CLI` is unset, script first requires `go -C engine env GOVERSION` to resolve already-available module-selected `go1.26.5`, then builds actual CLI into private temporary root with `go -C engine build -o "$private_bin/devrites-engine" .`; it never accepts a mock or performs feature network access. Missing module toolchain blocks before fixture mutation. Vet observed `go1.26.5` with `GOTOOLCHAIN=auto`; Build/Prove recapture availability.

Fixture matrix:

- admission parser: exactly one contract block; exact fields/order and kind-specific referenced-block bodies; return/action/cwd/signal/rollback/evidence grammar; checked arithmetic from declared target capacity; mode/base-8/index/path/golden vector; row/content bounds; and minimum-minus-one/exact-minimum/overflow relational fixtures for diagnostic, transaction-file, journal, attempt, and proof-time limits;
- existing/absent targets in distinct admitted parents; no-follow ownership/mode/type/link/cardinality/size checks; source swap after open proves hash/stage use same immutable bytes;
- lock namespace/owner bootstrap under child-local `umask(077)` before/after mkdir/open/sync/flock interruption in actual child processes; barrier-synchronized first-create race between two canonical roots launched under hostile inherited umask; exact observed `0700`/`0600` modes; exact `fcntl.flock` contention/release/reacquire; `lockf`/`F_SETLK` mutant rejection; injected unsupported host/access; stale generation/owned-section compare; independently observed losing-root zero writes;
- source promotion termination before/after mkdir, authority/source/ready create/write/mode/file-sync/dir-sync/rename/parent-sync; invalid/missing/mismatched authority and unrelated exact-name/lookalike trees untouched;
- stale canonical after promotion-before-journal plus binding rollover: exact cleanup marker and lock-intent truncate/write/sync/readback/rename/unlink/rmdir/parent-sync/intent-clear interruptions and fresh-process retry; authenticated partial/empty suffix resumes, byte-identical forged directory and malformed/orphan intent block untouched; successful GC continues current promotion;
- complete stage/backup writes with partial positive progress and injected bool/non-integer/zero/negative/oversized progress, exhaustion, `ENOSPC`, write error, and death after every partial write; exact partial file alone may be unlinked/recreated;
- missing/malformed/stale path/order/mode/hash/Vet binding with zero target writes; source loss before replacement, install-through-`PROVING`, and at/after `PROVED` with exact phase route;
- death before/after each canonical `WA-OP-*` durable intent and operation, including first evidence install, stage, backup, install/readback, proof, rollback, failure/success cleanup, retry handoff, exhaustion GC, product comparison, and idempotent verify;
- success/ordinary failure at every replacement; both source/destination directory-handle omission mutants; target may recover only to frozen identity or exact preimage/absence;
- proof command nonzero/wrong signal, command timeout, aggregate timeout, termination grace, forced-kill survivor, descendant survival attempt, complete reap, output cap, and death immediately before/after durable `PROVED`;
- retry/source loss through the shared lifecycle: first failure retains source → accepted offline correction → green re-preflight → epoch 2 success; same decisive fingerprint reaches counts 1/2/3 then `EXHAUSTED` with durable `same-fingerprint-count`, no attempt 4, and observed source GC; independent epoch cap exhausts with durable `total-epoch-limit` even when current count is below three; death at `RETRY_PREPARING` resumes same epoch; distinct Critical/Important fingerprint starts independent count while preserving immutable history; source loss before replacement, during install/proof, and post-`PROVED` yields the exact rollback/cleanup/target relation from observed state;
- evidence: absent-marker append, arbitrary prior `EVID-###`/prefix/suffix preservation, exactly one standalone `Candidate SHA-256:` line, immutable attempt rows, generation compare, duplicate/nested/malformed/over-budget marker rejection, crash/retry/cleanup/idempotent cycles, and 280-line workspace cap;
- diagnostics: every finite reason/boundary/route row, exact ASCII line/LF, ≤admitted bytes, unknown collapse, collision mutant, and hostile/secret/physical-path/raw-error leak mutants;
- cardinality: one stage/backup per target, one recognized promotion temp/canonical bundle, one owner lock, one journal temp; unknown namespace entry blocks without deletion;
- branch exclusion: success never rolls back; pre-proof failure restores preimages; post-proof recovery preserves targets and never reinstalls; `CLEANED` verifies without source or budget charge.

Actual engine separation fixture uses `DEVRITES_ROOT` against disposable workspace and actual self-built CLI. It compares product candidate digest, readiness binding, and built-slice count before/after marker-owned workflow evidence update. Adding workflow path to `touched-files.md` must fail candidate/readiness validation rather than silently bind it.

Provider-neutral corpus `evals/behavioral/workflow-artifact-identity.json` contains exactly ten siblings:

1. `WA-ADMISSION-SUCCESS`
2. `WA-MISSING-IDENTITY`
3. `WA-STALE-IDENTITY`
4. `WA-STALE-WRITER-EXHAUSTION`
5. `WA-FIRST-ROOT-FAILURE`
6. `WA-REPLACEMENT-ROLLBACK`
7. `WA-CLEANUP`
8. `WA-IDENTITY-CONTINUITY`
9. `WA-COMPLETED-HISTORICAL`
10. `WA-IDEMPOTENT-RERUN`

Each row binds exact route/action, durable consequence, and forbidden behavior from spec route matrix. Every actual adapter contains exactly one bounded four-field declaration: a whole-line HTML comment `<!-- workflow-artifact-adapter: {"module":"...","entry":"...","action":"...","return":"..."} -->` whose JSON object keys are exactly `module`, `entry`, `action`, `return` in that order, values are nonempty strings, `module` is `devrites-lib/reference/standards/workflow-artifacts.md`, and the remaining three fields equal the canonical adapter table cells. Markdown-table or prose-only link forms are not declarations. Dedicated test compiles `^<!-- workflow-artifact-adapter: (\{.*\}) -->$`, requires cardinality one, exact key order, and exact row equality, and rejects per-field and duplicate-declaration mutants plus concrete paraphrased owner/classifier/retry-policy prose independently of marker cardinality; each authored adapter contains no policy restatement outside its declaration, and host generation proves the same for Claude/Codex mirrors. `scripts/run-behavioral-evals.sh` proves generic schema only. Neither oracle implements route classification or makes live-provider claims.

## Prove-time root-caller walkthrough

Exact repository-root command is `bash tests/workflow-artifact-identity-test.sh --prove-walkthrough`. Mode accepts no extra argument, creates private disposable workspace, requires module-selected Go 1.26.5 already available, self-builds actual engine if env binary absent, and performs no source/generated/real action.

1. At command start, record monotonic TTHW start before toolchain check/self-build; capture exact admission fixture, saved `phase=prove`/`next_action=/rite-prove demo`, and product identities.
2. Promote source, install target, observe `WA-PROOF-001 PASS`, reach `PROVED`, clean/GC, restore cursor, and record monotonic TTHW end at first successful `CLEANED`/cursor return. Predicted TTHW comparison bound is 90 seconds and includes toolchain/self-build. Spec measurable-success “admitted aggregate limit” and the demo admission 30/60-second values bind proof subprocesses only; they are not the TTHW clock.
3. After TTHW interval closes, inject death after partial stage write and at `RETRY_PREPARING`; prove exact interrupted resume without changing measured TTHW.
4. Run stale-authority preinstall and capture verbatim fixed diagnostic with zero target writes.
5. Run source-free `CLEANED` idempotent verify and prove no install/retry/action budget.
6. Emit exact successful signal, `tthw_ms=` followed by nonnegative base-10 milliseconds, fixed stale diagnostic, `cursor=prove:/rite-prove demo`, `product_identity=unchanged`, and final `WORKFLOW_ARTIFACT_WALKTHROUGH PASS`; no physical path or dynamic error.

Prove records command/cwd, module toolchain, exact output, TTHW, recovery traces, cursor, and product equality. TTHW prediction is distinct from proof-command timeout and becomes measured evidence only after execution.

## Existing assertion replacement

`tests/phase-gate-routing-test.sh` and `tests/host-artifacts-test.sh` stop requiring stale migration phrases. New assertions require canonical linkage, absence of `Cold-resume migration` and generic-materializer authority, stable source resolver/held-descriptor rule, explicit success/failure graph, phase-qualified source loss, exact route-table outcomes, bounded recovery, and generated host parity.

Test review rejects deleted coverage, skipped/focused tests, weakened assertions, fixture-only claims that bypass actual replacement/resume consumers, or mocked engine identity.

## Writer-owned candidate delivery

Root freezes 16 authored states, 22 generated preimages, full generated/outside manifests, and protected hashes, then dispatches one bounded `devrites-slice-wright` with exact 38 destinations. Root never writes product/generated files. The wright normally launches install/recovery. Only when the measured host tool window cannot hold the uninterrupted process may the coordinator launch and monitor one already prepared install using the exact wright-frozen driver hash, Vet-approved argv, cwd, and environment; recovery remains wright-only.

Dedicated test exposes writer-only modes in addition to non-mutating default and Prove walkthrough:

```text
PYTHONDONTWRITEBYTECODE=1 bash "$DEVRITES_DELIVERY_DRIVER" --delivery-prepare
PYTHONDONTWRITEBYTECODE=1 bash tests/workflow-artifact-identity-test.sh --delivery-install "$DEVRITES_DELIVERY_DIR"
PYTHONDONTWRITEBYTECODE=1 bash tests/workflow-artifact-identity-test.sh --delivery-recover "$DEVRITES_DELIVERY_DIR"
```

Because dedicated test destination is initially absent, sole wright first authors all candidate bytes in private mode-`0700` delivery scratch, sets `DEVRITES_DELIVERY_DRIVER` to exact candidate `workflow-artifact-identity-test.sh`, records its SHA-256, and runs its non-mutating self-tests/syntax there. No repository destination changes. Private candidate driver then runs `--delivery-prepare`, creates candidate-digest-named transaction directory under `.devrites/work/workflow-artifact-identity/.generated-install/`, acquires exact `fcntl.flock`, snapshots 16 authored states/bytes, 22 generated preimages/modes, full generated/outside manifests, allowlist digest, and protected hashes into mode-`0600` files using complete writes/sync, then durably records `SNAPSHOTTING`. After authored install, destination `tests/workflow-artifact-identity-test.sh` must hash-identically match bootstrap driver before it may run install/recover modes. Bootstrap scratch path/hash stay in writer context and private journal, never public evidence.

After authored edits, `--delivery-install`:

1. validates snapshot/allowlists/current authored delta and advances `STAGED` intent;
2. runs normal generator once with `DEVRITES_HOST_ARTIFACT_DIR` set to same-filesystem private stage inside delivery directory; default `pack/generated` is never passed;
3. validates complete staged tree, generated bytes derive from canonical source, and differences are subset of exact 22 destinations with no unknown/unreadable/symlink/hard-link entry;
4. retains exact backups and journals `INSTALLING(n)` before each same-parent no-follow generated replacement, then reads bytes/modes back exact;
5. advances `INSTALLED → PROVING`, runs dedicated/routing/host/instruction/repository gates approved in `test-plan.md`, verifies 16/22 allowlists, full outside manifests, and protected hashes;
6. advances `COMMITTED` only after all gates pass, then `CLEANING → CLEANED` and removes private stage/backups.

Before any mutation, prepare/install/recover validate the journal as untrusted input: exact schema/state, candidate/driver/allowlist identities, ordered 16/22 cardinality, indices, compiled path equality, derived backup handles, legal preimage mode/size/hash, counters, outside manifest, and protected records. Recovery iterates compiled constants only. `SNAPSHOTTING` validates its durable prefix and exact backups, reconciles an orphan derived backup, completes remaining rows, and cannot install until all 38 rows exist. Any failure, `INT`, `TERM`, or process death before `COMMITTED` resumes by next serial wright from exact journal and advances `ROLLING_BACK(n) → RESTORED → FAILED`, restoring all 16 authored states and 22 generated preimages, then proving full manifests/protected hashes. After `COMMITTED`, recovery only resumes cleanup and never restores. Gate launch uses the already-held repository descriptor rather than `cwd=<pathname>`. Root that observes incomplete delivery dispatches the same bounded recovery wright with exact directory and performs no source copy/unlink. A DEC-042 coordinator launch is allowed only for one prepared install process and cannot alter argv/environment, supply fixture controls, reinterpret the journal, or race recovery. One expanded boundary registry is the sole source for every actual `delivery_death` site and test enumeration. Default mode launches the actual prepare/install/recover modes in disposable child processes at every registry entry and resumes through the mode valid for that durable state. Before any stage/backup deletion, the test freezes expected post-install state/mode/digest from independently staged bytes for each compiled destination; a separate observer enumerates exact 16/22 cardinality and compares every record at `COMMITTED`, `CLEANING`, and `CLEANED`, while failure terminals compare exact preimages. The observer also verifies terminal journal relation after every snapshot/stage/install/proof/rollback/commit/cleanup death and tests forged journal paths/states/backups/hashes before mutation. A missing, duplicate, or unexecuted registry entry fails coverage; no sampled subset, in-memory substitute, generic materializer, or second module is accepted.

Refresh `tests/instruction-size-baseline.json` from validated staged source state before install. Total canonical skill Markdown remains at or below 855,000 bytes; compact duplicate adapter prose instead of raising limit.

## Protected baseline

Before writer dispatch, root freezes:

- all 16 authored preimages/absence;
- all 22 generated preimages;
- complete outside-allowlist repository identity, including exact SHA-256 for every inspected-and-OUT direct reference/script;
- Workspace Observation protected manifest and all source bytes; Reslice workspace records, all 30 non-overlap candidate bytes, and 13 shared current preimages (five authored/eight generated) plus semantic regression contract;
- user-owned `.gitignore` SHA-256 `58c1cc88c16b9bb14b345c156703163b47c9cb6232276b50684fabae8503e8fd`, mode `0644`, size 1,098 bytes; do not edit, restore, stage, or include it;
- `.devrites/ACTIVE` SHA-256 `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267` and value `workflow-artifact-identity` (Q-010/Q-011/DEC-047 accepted live preimage; do not rewrite ACTIVE); historical freeze `9ef52ccabae0acc5264a2f1c0f404c95e8191f0261cfa525880262447c9a75dc` / `thin-engine-native-codex` is superseded;
- Workspace Observation manifest SHA-256 `2dca74484895de119cd935db6c3692782df9173eef199c88a7d5a65898332ec9`.

After writer/generation/proof, root requires zero outside-allowlist identity delta. No commit, push, tag, publish, Ship, close-out, release mutation, ACTIVE change, or `.gitignore` change occurs.

## Rollback

Sole bounded wright owns all source/generated restoration through feature-local delivery journal. Before `COMMITTED`, any authored edit, generation, staged-tree validation, 22-file install, allowlist, proof, signal, or death failure restores exact 16 authored states and 22 generated preimages, verifies full generated/outside/protected identities, and retains private journal on failed recovery. Next serial recovery wright resumes exact state. Root verifies only and never copies/unlinks product files. After `COMMITTED`, cleanup resumes without rollback. Default generator target was never used; no partial candidate survives.

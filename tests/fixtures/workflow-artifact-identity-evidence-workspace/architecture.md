# Architecture: Workflow Artifact Identity

## Owning module / layer

`pack/.claude/skills/devrites-lib/reference/standards/workflow-artifacts.md` remains the sole Workflow Artifact module. Its interface is consumed by Vet, Build, Prove, Autocomplete, Debug Recovery, AFK, One-shot, and generated Claude/Codex adapters. Its implementation is controlling-root authorship plus one feature-specific artifact-set transaction; no generic runtime, engine command, parser, or new seam is introduced.

## Deep-module check

Deletion test passes. Removing the module would spread admission, identity timing, preparation, crash recovery, replacement, proof, cleanup, resume, and product-separation rules across ten canonical callers and two generated hosts. Keeping one interface yields leverage for callers and locality for maintenance.

Interface:

- Input: active slug; current Vet readiness binding; one exact `devrites.workflow-artifact-admission.v1` block; ordered admitted targets; byte/file/journal/attempt/proof-time limits; caller return cursor.
- Per-target admission: normalized active-workspace path; four-digit octal mode; behavior/interface references; positive and failure fixtures; proof command/cwd/signal; rollback relation; evidence fields.
- Output: ordered frozen `(path, numeric mode, SHA-256)` identity; opaque retained-source handle; one marker-owned `devrites.workflow-artifact-journal.v1` result; proof/cleanup result; exact route token.
- Invariants: one nonblocking exclusive owner; exact no-follow paths; identity frozen after green disposable preflight; source retained through retryable `FAILED` and garbage-collected before `CLEANED`/`EXHAUSTED`; intent durable before each operation; complete writes only; all targets settle at frozen output or preimages; product candidate/readiness/built-slice identities remain unchanged; no real consumptive action executes.
- Errors: finite reason/boundary IDs map by exact precedence to Plan/Vet, offline recovery, cleanup resume, existing safety/access gate, active-owner wait, or exhaustion block. Public output is one fixed ASCII line; no dynamic value leaks.
- Limits: Vet binds checked relational target/content/file/journal/attempt/proof-time limits; diagnostic capacity is exactly 256 and all finite lines fit. Cardinality includes per-target source/stage/backup plus authority/ready/cleanup marker, one source directory, owner lock, and journal temporary; evidence headroom and three-attempt policy must fit before admission.

Thin adapters retain only module link plus one phase-local entry trigger, canonical route/action token, and return cursor. They do not repeat admission fields, classifier precedence, source/identity validation, operation table, journal state graph, evidence grammar, diagnostics, or recovery accounting. Claude and Codex are two real generated adapters at host seam.

## Ordered transaction

1. Vet emits one exact admission block. Root rejects unresolved choice, directory, glob, traversal, duplicate, symlink, workspace escape, product/dependency path, malformed mode/index, or missing positive bound before active mutation.
2. Root bootstraps private namespace/lock via umask-`077` descriptor-relative `mkdirat`/`openat`, exact ownership/mode/type/link validation, file/parent sync, and only Python `fcntl.flock(LOCK_EX|LOCK_NB)` on held `O_CLOEXEC` descriptor. `lockf`/`F_SETLK` are prohibited. Unsupported host/access fails closed; bootstrap interruption reconciles exact names; busy owner returns `WAIT_ACTIVE_OWNER` with zero journal/target writes.
3. Root authors exact bytes in disposable same-layout preflight. Domain-separated SHA-256 over length-prefixed slug UTF-8 and decoded Vet binding yields `wsrc:<64hex>`; ordered length-prefixed path/numeric-mode/content-hash rows yield identity digest.
4. Source promotion uses exact `.<hex>.preparing`: durable authority/indexed-source/ready intents, complete writes/sync, atomic rename, parent sync. Pre-journal binding rollover validates old canonical authority/ready/count plus absent journal/unknown entries, writes synced `.stale-cleanup`, atomically renames to `.<oldhex>.stale-cleaning`, and idempotently removes recognized entries. Crash at marker/rename/unlink/rmdir/sync resumes under owner lock; unrelated entries stay untouched.
5. Resolver opens bundle/source no-follow, validates ownership/mode/type/link/cardinality/size, and reads each descriptor once into bounded immutable bytes used for both hash and stage. Source remains through retryable `FAILED`; success/exhaustion cleanup removes it before `CLEANED`/`EXHAUSTED`.
6. Journal owns only exact marker-delimited section in `evidence.md`. Under lock, each update checks monotonic generation and owned-section preimage SHA-256, preserves every outside byte, and installs a same-parent atomic/synced replacement. Duplicate/nested/malformed/over-budget markers fail before mutation.
7. Canonical operation table drives implementation and test: source promotion, journal init, complete stage/backup write, install/readback, process-group proof, rollback, failure/success cleanup, retry handoff, exhaustion GC, evidence update, product separation, and idempotent verify. Every intent is durable before operation; resume admits only exact pre/post/declared-partial state.
8. State graph branches:
   - Success: `PREPARING → PREPARED → INSTALLING(n) → INSTALLED → PROVING → PROVED → CLEANING(n) → CLEANED`. Post-`PROVED` recovery preserves targets and never rolls back/reinstalls.
   - Failure before first replacement: `→ FAILURE_CLEANING(n) → FAILED` with zero target writes.
   - Failure/termination from first replacement through `PROVING`: `→ ROLLING_BACK(n) → ROLLED_BACK → FAILURE_CLEANING(n) → FAILED` with exact preimages restored.
   - Accepted correction: `FAILED → RETRY_PREPARING(epoch+1) → PREPARING`; immutable prior row remains. Same fingerprint count 3 or epoch cap routes `EXHAUSTED_CLEANING → EXHAUSTED` and no attempt 4.
9. Proof commands run in fresh process groups with per-command/aggregate wall-clock bounds, bounded private output, terminate/grace/force-kill/reap timeout behavior, and fixed diagnostic boundary. Any pre-`PROVED` proof failure rolls back.
10. Source loss uses three phase classes: before replacement cleanup then Plan/Vet; install-through-`PROVING` rollback/cleanup then offline repair/Plan-Vet; at/after `PROVED` preserve targets, reconcile cleanup, then Plan/Vet without rollback. Source absence after `CLEANED` is expected.
11. Green `CLEANED` proves target identity, product candidate/readiness/built-count equality, exact outside-evidence preservation, source GC, and caller cursor restoration; then stops for fresh consumptive-action authorization.

Relative create/replace/unlink/prune uses no-follow directory descriptors and both source/destination handles where required. Complete-write loop rejects bool/non-integer/zero/negative/oversized progress, exhaustion, `ENOSPC`, and write errors. Finite classifier and diagnostic tables select all public failure routes; underlying filesystem details never escape.

## Identity separation

Workflow identity and product identity are independent:

```text
Vet admission -> disposable retained source -> frozen workflow identity -> transaction journal
                                                            -> proof/cleanup/return

touched-files.md -> product candidate digest -> readiness binding -> built-slice count
                    (workflow paths excluded and values unchanged)
```

`evidence.md` stores workflow identity and observed product values but does not add workflow paths to product candidate rows or readiness inputs. Completed historical workspaces receive no backfill.

## Integration points

Canonical authority:

- `pack/.claude/skills/devrites-lib/reference/standards/workflow-artifacts.md`

Thin canonical adapters:

- `pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`
- `pack/.claude/skills/devrites-lib/reference/standards/one-shot-actions.md`
- `pack/.claude/skills/devrites-debug-recovery/SKILL.md`
- `pack/.claude/skills/rite-autocomplete/SKILL.md`
- `pack/.claude/skills/rite-autocomplete/reference/loop.md`
- `pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md`
- `pack/.claude/skills/rite-build/SKILL.md`
- `pack/.claude/skills/rite-build/reference/phase-contract.md`
- `pack/.claude/skills/rite-prove/SKILL.md`
- `pack/.claude/skills/rite-vet/SKILL.md`

Proof surfaces:

- `evals/behavioral/workflow-artifact-identity.json`
- `tests/workflow-artifact-identity-test.sh`
- `tests/phase-gate-routing-test.sh`
- `tests/host-artifacts-test.sh`
- `tests/instruction-size-baseline.json`
- matching generated Claude/Codex paths for eleven changed canonical Markdown files

Inspected-and-OUT `agents.md`, standards `README.md`, `skill-authoring.md`, `rite-plan/SKILL.md`, `rite-build/reference/wright-dispatch.md`, and `pack/.claude/agents/devrites-plan-drafter.md` remain unchanged unless proof demonstrates drift. Generator and corpus-validator scripts also remain unchanged.

## Data / API / events

No public API, event, schema, or engine data model changes. Interface data is Markdown admission/evidence: ordered identity rows, opaque logical handles, durable enum states, fixed reason/boundary IDs, and observed product identity values. Target/source bytes never enter evidence.

## Dependencies

No new package, runtime, service, feature network call, provider credential, production parser, feature flag, or engine command. Dedicated test parses canonical tables, drives one disposable consumer, and grades exact five-field traces through separate observer process deriving OS/engine facts without consumer self-report. It requires already-available module-selected Go 1.26.5 and self-builds actual CLI when env binary is absent. Same bounded wright uses existing generator only through private same-filesystem `DEVRITES_HOST_ARTIFACT_DIR` and owns feature-local 16/22 delivery journal; root only freezes/verifies manifests.

## Risks

- Concurrent roots or stale evidence writers: exact namespace/lock bootstrap, only interoperable `fcntl.flock`, generation/preimage compare, first-create/interruption/mutant fixtures, owner-busy zero writes.
- Process termination during source promotion, complete stage/backup writes, journal update, install/proof/rollback/cleanup/retry/GC: canonical operation intents plus before/after kill-resume matrix.
- Partial/abnormal write progress: bounded immutable view, strict positive integer progress, exact partial-file reconciliation, injected short/zero/negative/oversized/non-integer/bool/`ENOSPC` failures.
- Retained source missing/tampered/stale across binding rollover: zero target writes before replacement; exact lock-held pre-journal stale-canonical GC; rollback through `PROVING`; proved-target preservation after `PROVED`; expected absence after `CLEANED`.
- Source path/handle forgery or swap: stable resolver, no-follow held descriptors, immutable bytes, forged/swap/lookalike mutants.
- Infinite correction loop: immutable attempt rows, same-fingerprint count, epoch cap, death-safe retry handoff, source GC at exhaustion, no attempt 4.
- Hung/leaking proof descendants: fresh process group, command/aggregate timeout, bounded grace, forced kill/reap, output cap.
- Evidence corruption or lifecycle overwrite: marker-owned atomic section, outside-byte preservation, one candidate binding, malformed-marker rejection.
- Test consumer self-attests invariants: exact five-field oracle plus separate OS/engine observer process and per-column/per-field mutations.
- Impossible admitted limits: relational arithmetic minima and minimum-minus-one/exact/overflow fixtures.
- Duplicate caller policy: per-adapter entry/action/return table, structural thinness checks, stale-phrase deletion.
- Workflow files entering product identity: actual engine before/after fixture.
- Generator deleting/changing siblings or a second filesystem writer crossing the seam: bounded wright authors the candidate; the hash-bound driver alone owns private stage plus feature-local 16/22 delivery journal/restart; root freezes/verifies only; DEC-042 permits coordinator launch, not destination writes, under the measured host-window exception; complete outside-manifest equality.
- Generated host drift or instruction-cap breach: exact generated allowlist, parity, and fixed baseline.

## Affected boundaries

Sixteen authored paths and 22 generated derivatives are closed in `plan.md`. Thirteen destinations overlap the prior Reslice candidate: five authored preimages and eight normal generator-derived mirrors. They must preserve Reslice semantics under dedicated regression proof; the previous standalone Reslice candidate digest becomes historical, while its workspace records and all 30 non-overlap candidate paths remain byte-identical. Workspace Observation has zero overlap and protected manifest stays exact. Engine/dependency files, `.gitignore`, `.devrites/ACTIVE`, completed workspaces, and release state remain outside the boundary and byte-identical.

## Security and observability

Active paths resolve descriptor-relatively with no-follow checks. Stable handle resolver derives source namespace from slug plus Vet binding. Held source descriptors receive `fstat` checks and one bounded immutable read used for both hash and stage, preventing validate/reopen races. Retained source plus stage/backup/journal files are current-user-owned, private, cardinality-bounded, and never contain credentials. Evidence records opaque handles and identities, not source/target bytes or physical paths. Public failure emits fixed reason and allowlisted boundary only; no hostile content, physical path, raw filesystem error, or secret reaches output.

Durable journal state is sole crash-resume observation. Chat history, old evidence, target inference, actor history, and stale writer-exhaustion migration carry no authority.

## Rollback

One atomic source/test/generated candidate. Root freezes/verifies 16 authored states, full generated/outside manifests, 22 generated preimages, and protected hashes but performs no product write. The bounded wright authors the candidate; the hash-bound delivery driver snapshots all 38 destinations in the private feature-local journal, generates only to a private same-filesystem stage, installs exact outputs with durable per-path intents, proves the candidate, and restores 16/22 preimages on every pre-`COMMITTED` failure/death. Normally the wright launches that driver. Under DEC-042, when the measured host tool window cannot hold the uninterrupted transaction, the coordinator may launch and monitor only the exact prepared hash/argv/cwd/environment; this adds no destination writer or recovery authority. The next serial wright resumes an incomplete journal; post-`COMMITTED` only cleanup resumes. Default `pack/generated` is never generator deletion target. No engine state, dependency, feature flag, hook, schema, migration, actual workflow executable, or completed workspace needs reversal.

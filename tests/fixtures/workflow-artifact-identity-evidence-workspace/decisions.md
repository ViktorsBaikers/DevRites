# Decisions: Workflow Artifact Identity

## Decision log

## DEC-001 — Deepen existing module

- Context: Workflow Artifact policy already has one canonical home but callers duplicate routing and recovery.
- Decision: Reshape existing `workflow-artifacts.md`; do not add another module.
- Why: Deletion test shows admission, identity, transaction, proof, and resume complexity would spread across five rites. One module earns depth, leverage, and locality.
- Rejected: new standard, Go command, generic materializer, compatibility wrapper.

## DEC-002 — Vet-owned admission

- Context: Root may write `.devrites/**`, but unresolved transaction details cannot be invented during Build.
- Decision: Vet binds exact paths, modes, behavior, limits, fixtures, proof signals, rollback, and evidence fields before any active write.
- Why: Interface is test surface; complete admission keeps root implementation mechanical and observable.

## DEC-003 — Freeze identity after preflight

- Context: Planned path alone does not identify executable bytes.
- Decision: Green disposable preflight atomically promotes exact source under logical `wsrc:<64hex>` handle derived from active slug plus Vet binding. No-follow descriptor reads produce bounded immutable bytes used for both hash and stage. Source remains through retryable `FAILED`; success/exhaustion cleanup removes it before `CLEANED`/`EXHAUSTED`, and absence after `CLEANED` is expected.
- Why: Identity continuity, restart resolution, TOCTOU resistance, stage recreation, bounded retention, and idempotent verification become exact.
- Rejected: target-byte inference, chat memory, physical-path evidence, validate-then-reopen, hash-only recreation, post-install-only hash.

## DEC-004 — Feature-specific crash-resumable transaction, no generic materializer

- Context: Atomic replacement is required across ordinary failure and process termination, but a reusable executor would create a second implementation surface.
- Decision: Root authors smallest feature-specific artifact-set transaction described by Vet. Evidence records intent before each operation. Success graph is `PROVING → PROVED → CLEANING → CLEANED`; pre-proof installed failure graph is `ROLLING_BACK → ROLLED_BACK → FAILURE_CLEANING → FAILED`; pre-install failure skips rollback. Post-`PROVED` recovery never rolls back or reinstalls.
- Why: Shared semantic contract stays deep and crash-resumable without shipping a generic runtime or engine policy.

## DEC-005 — Separate durable evidence and product identity

- Context: Workflow artifacts plan/isolate/prove work but are not product deliverables.
- Decision: `evidence.md` owns exactly one versioned marker-delimited journal section. Atomic generation/preimage-checked updates preserve every outside byte and one standalone candidate binding while recording opaque source/identity, preimages, branch/attempt/proof/cleanup, product values, and return cursor. Workflow paths never enter candidate manifest or readiness binding.
- Why: Resume has durable authority without overwriting lifecycle evidence or changing what ships.

## DEC-006 — Current identity and journal control resume

- Context: Existing one-time stale writer-exhaustion migration reinterprets old attempts and can synthesize authority.
- Decision: Remove migration. Missing/stale authority routes Plan/Vet. Before first replacement, source loss cleans with zero target writes. From first replacement through `PROVING`, source loss/failure restores preimages. At/after `PROVED`, proved targets remain while cleanup reconciles. `FAILED` may retry only after accepted offline correction and green re-preflight; immutable attempt epochs count same-fingerprint no-progress to 3, then exhaust and forbid attempt 4. Completed work gets no backfill.
- Why: Resume and bounded recovery follow durable current facts, not historical actor availability or an infinite retry loop.

## DEC-007 — Provider-neutral routing proof

- Context: Claude/Codex behavior cannot be deterministically executed in offline CI.
- Decision: Add ten stable scenario IDs with exact route/action, durable consequence, and forbidden behavior plus one deterministic transaction/identity test. Optional pinned host runs remain claim-bounded.
- Why: CI proves contract, fixtures, and generated linkage without pretending to prove unrun providers.

## DEC-008 — One atomic source-plus-host release unit

- Context: Canonical callers and generated host derivatives must move together.
- Decision: Same bounded sole wright owns 16 authored destinations and generator-derived writes to exact 22 generated destinations. Feature-local private delivery journal snapshots/restores all 38 states, generates only to private stage, and resumes after death; root freezes/verifies only and never installs/restores product files.
- Why: Partial delivery or destructive default generation leaves duplicate routing, host drift, or unrelated generated loss.

## DEC-009 — Single owner and finite operation contract

- Context: Two roots or test/runtime transition drift could corrupt shared evidence and targets.
- Decision: Umask-077 descriptor bootstrap creates exact private namespace/lock; every root uses only Python `fcntl.flock(LOCK_EX|LOCK_NB)` on held `O_CLOEXEC` descriptor. Unsupported hosts fail closed. Canonical module exposes exact operation/transition, route/action, and reason/boundary tables; adapters only invoke them.
- Why: One interface keeps concurrency, recovery, and diagnostics local while tables give deterministic test surface.
- Rejected: advisory lock, caller-local classifiers, prose-only states, generic runtime.

## DEC-010 — Complete writes, bounded proof, and bounded retry

- Context: Partial writes, hung proof descendants, and unbounded retry were undefined failure surfaces.
- Decision: Stage/backup writes loop over bounded immutable bytes with strict progress validation; proof runs in bounded fresh process groups; retry uses immutable epochs, same-fingerprint count 3, death-safe handoff, and exhaustion GC.
- Why: Every active side effect now has durable intent, finite completion, exact recovery, and finite terminal behavior.

## DEC-011 — Evidence section ownership

- Context: Whole-file `evidence.md` replacement could destroy prior lifecycle evidence and duplicate candidate binding.
- Decision: Journal may append/replace only exact standalone marker-owned section, preserving all outside bytes and rejecting ambiguous markers before mutation.
- Why: Workflow transaction evidence gains locality without taking ownership of whole lifecycle record.

## DEC-012 — Independent table/trace proof

- Context: A test-local executor and expected outcomes derived from same implementation would be tautological second authority.
- Decision: Dedicated test parses canonical tables and drives disposable consumer. Exact oracle record has operation, accepted pre/post state, failure route, and observer assertion ID. Separate process derives decisive OS/engine facts without trusting consumer report; per-column and per-field mutations prove independence. Already-available module Go 1.26.5 self-builds actual engine when env binary absent.
- Why: Interface remains test surface; consumer and observer/oracle fail independently; engine separation is observed, not mocked.

## DEC-013 — Recover pre-journal stale source

- Context: Crash after canonical source promotion but before journal, followed by Vet-binding rollover, can leave internally valid old bundle blocking new work forever.
- Decision: Under canonical owner lock and absent journal/target writes, validate old authority/ready/count, durably mark and atomically rename exact bundle to stale-cleaning state, then idempotently delete recognized entries. Unknown entries remain untouched and block.
- Why: Retention stays bounded without inferring target state or deleting untrusted namespace content.

## DEC-014 — Admission limits must be executable

- Context: Positive-looking limits can be too small for mandatory diagnostic, transaction files, history, or three-attempt policy.
- Decision: Vet validates checked relational minima and evidence headroom, with exact diagnostic capacity and boundary/overflow fixtures.
- Why: Syntactically valid admission must be executable before active mutation.

## DEC-015 — Preserve shared Reslice semantics additively

- Context: Workflow Artifact changes 13 destinations already owned by the sealed uncommitted Acceptance-preserving Reslice candidate: five authored preimages and eight generated mirrors; byte-identical preservation of the whole prior candidate is impossible.
- Decision: Treat all shared bytes as preimages, edit the five authored paths additively, derive the eight mirrors only through normal private-stage generation, retain all Reslice linkage/packet/route/action/stop/baseline behavior, run its dedicated regression before commit boundary and root proof, and leave prior workspace plus all 30 non-overlap candidate paths exact. Previous standalone Reslice digest becomes historical after shared-byte change.
- Why: Serial architecture program needs composed source behavior, not two incompatible byte identities or rollback of accepted prior work.

## DEC-016 — Executable consumer and compiled recovery authority

- Context: Review proved that several green checks compared tables, tuples, literals, or an in-memory simulator while claiming actual crash-resume behavior. The private delivery implementation also accepted partial snapshots and journal-provided mutation paths.
- Decision: One test-local parsed-table consumer shall persist real intent, execute each canonical operation against disposable filesystem state, die before/after intent and effect, resume in a fresh process, and be graded by a separate observer. Delivery recovery validates a complete exact schema and iterates only the compiled ordered 16/22 allowlist; `SNAPSHOTTING` resumes idempotently. Gate subprocesses stay descriptor-anchored. Exact stale-cleanup authority lives durably in the held lock descriptor and is part of the canonical interface.
- Why: The interface must be the test surface. Static table equality has no leverage over crash behavior, while one executable consumer gives locality to operation, owner, delivery, route, and walkthrough proof without shipping a generic materializer.
- Rejected: weakening AC-003/AC-006/AC-007; accepting green tuple checks; journal-selected paths; pathname cwd reacquisition; a new engine command or reusable runtime; OS sandbox/detached-session expansion beyond DRIFT-006.

## DEC-017 — One real lifecycle, one independent observer

- Context: The first DEC-016 correction renamed tuple simulation as an executable consumer but still wrote canonical row strings and their hash as the effect. The same self-derived oracle left owner bootstrap, retry/source-loss, adapter behavior, and the public walkthrough unproven; delivery fault labels were broader than actual child-process coverage.
- Decision: The test-local lifecycle dispatches every parsed operation ID to a real descriptor-relative filesystem/journal transition, while a separate observer derives result fields only from resulting bytes, modes, descriptors, journal state, targets, and engine output. Canonical owner bootstrap, retry/source-loss, and one-target walkthrough reuse that lifecycle. Delivery death sites and actual-process tests consume one exhaustive boundary registry. Each actual adapter contains exactly one bounded four-field declaration matched to the independent adapter map. Canonical stale intent, admission grammar, and exhaustion-cause evidence match the frozen spec exactly.
- Why: The interface is the test surface only when the implementation crosses it. One real lifecycle gives leverage across operation, concurrency, retry, and walkthrough tests while keeping locality in the existing driver; one boundary registry prevents claimed and executed delivery coverage from diverging.
- Rejected: another simulator, separate per-test lifecycle implementations, sampled delivery boundaries, source-token substitutes for adapter behavior, literal walkthrough synthesis, destination expansion, or weakening the existing acceptance contract.

## DEC-018 — Observers own terminal truth

- Context: The real lifecycle correction executed the major paths, but several success claims still came from consumer literals, implementation validators, or disconnected tuple checks. Concrete mutants could retain source at `EXHAUSTED`, lose a committed destination, duplicate adapter policy, or mutate product identity while the suite stayed green.
- Decision: Every terminal claim is derived outside the consumer. Retry/exhaustion/source-loss use the lifecycle dispatcher and independent observer; journal/evidence persist the exhaustion predicate. Bootstrap children explicitly set `umask(077)` and are tested under hostile inheritance. Delivery observation compares independently frozen expected records for all 16/22 destinations at every post-commit terminal state. Adapter mutants contain concrete paraphrased owner/classifier/retry policy. Product separation compares independently frozen candidate/readiness/built-count values and emits success only after equality.
- Why: A deep module earns leverage only when its interface, not its self-report, is the test surface. Terminal observers concentrate truth in one place and make invalid current states fail without adding shipped machinery.
- Rejected: implementation-owned terminal validation as evidence, disconnected retry tuples, token-only paraphrase checks, current-state hashes labeled as unchanged, sampled terminal destinations, or a second lifecycle.

## DEC-019 — Preserve the process-group oracle; widen only its fixture window

- Context: the exact full repository suite runs four longest tests concurrently. Under that admitted load, the process-group timeout fixture's 300 ms window elapsed before its Python parent emitted `ready`, while the same environment-free contract and the immediate unchanged full-suite reproduction passed.
- Decision: widen only that disposable fixture's timeout. Keep the exact `timeout` classification, `ready\n` output, TERM/grace/KILL behavior, survivor detection, and complete process-group/leader reaping assertions. Production proof-command limits and the canonical Workflow Artifact interface do not change.
- Why: accepting empty output would weaken the oracle because it would no longer prove that the descendant existed. A larger test-local scheduling window preserves the behavior while removing a false failure caused by repository-runner contention.
- Rejected: weakening the output assertion, retrying a failed proof bundle, adding timing configuration, changing production limits, serializing the full suite, or treating one unchanged rerun as successful root evidence.

## DEC-020 — Terminal truth comes from retained facts and exact boundaries

- Context: a candidate with a valid 16-gate proof still admitted mutants that swapped source bytes after validation, discarded backups, forged product equality, skipped generated replacement effects, omitted terminal cause, skipped exact post-commit state writes, and passed through scheduler-dependent elapsed-time judgment.
- Decision: stage bytes come only from the descriptor-retained immutable view; rollback consumes and validates the recorded backup; terminal evidence owns the conditionally present exhaustion cause; independent observation acquires current product identity and exact boundary state; delivery proves changed generated bytes and every replacement effect; source-loss begins in the claimed replacement class; timeout boundedness uses deterministic clock/synchronization control rather than elapsed wall time. Diagnostics use the exact frozen reason, boundary, route, and public action.
- Why: the Workflow Artifact module earns depth only when its interface is the test surface. Retained facts and exact states give callers leverage while keeping identity, rollback, delivery, and proof truth local to one implementation instead of trusting consumer narration.
- Rejected: broad state membership, consumer-authored equality, pathname reopen after validation, literal preimage restoration, replacement counters as write evidence, always-present sentinel fields, elapsed-time assertions, new machinery, or destination expansion.

## DEC-021 — Preserve full outside protection; authorize exact failed-delivery rollback

- Context: candidate `5e6a1245…` reached `PROVING` with all 16 gates green, but the complete recursive outside manifest changed when host-managed worktrees, skill links, scratch, and Python bytecode changed. Normal recovery rejects outside drift before its required pre-`COMMITTED` rollback, stranding exact backups and installed destinations.
- Decision: keep REQ-009 and AC-004's complete recursive outside-manifest success guarantee. For this identified failed journal only, the sole wright may validate the compiled 16/22 allowlists, journal/candidate/root/driver identities, exact current expected-post records, and every backup preimage; restore only those 38 destination preimages; record the journal and outside mismatch as failed evidence; and alter no outside path. A distinct retry suppresses repository-local Python bytecode and keeps the writer's harness worktree alive synchronously until final comparison.
- Why: rollback reduces product mutation without pretending unrelated host drift was restored or accepting a weaker security boundary. A Git-visible manifest would allow ignored credentials and workspace files to change unnoticed.
- Rejected: narrowing to Git-visible paths, explicit scratch exclusions, deleting/restoring outside host files, manual success promotion, starting a new candidate over installed targets, or claiming the failed journal `CLEANED`.

## DEC-022 — Fail delivery closed on unowned state and finite execution

- Context: final Review proved that pre-commit recovery could overwrite a concurrent destination state, recursive outside protection skipped path-prefix lookalikes and nested Git directories, delivery generation/output/aggregate time were not finite, and two recovery routes did not expose the exact frozen actions to their controlling caller.
- Decision: preserve the existing Workflow Artifact module and sole-wright delivery implementation. Recovery accepts only each destination's exact preimage or expected-post state, validates the full set before its first write, and revalidates immediately before each restore/unlink. Recursive outside acquisition excludes only root `.git`, the exact delivery directory, and that directory's descendants. Generator and gates reuse the existing fresh-process-group, bounded-stream, TERM/grace/KILL/reap implementation under fixed internal limits: 600 seconds per process, 3,600 seconds aggregate, 8,388,608 combined output bytes per process, and 2 seconds grace. Canonical route rows and tests bind exact slash actions, `<slug>`, durable state fields, and caller return behavior.
- Why: the interface earns depth only when ownership and termination are finite at the existing seam. Pair-state guards prevent data loss without weakening complete outside protection; component-accurate exclusions retain locality; one existing process implementation gives delivery callers leverage without a second executor or configuration surface.
- Rejected: generalized outside-drift rollback, Git-visible or scratch-based protection, optimistic overwrite/unlink, pathname-prefix exclusion, unbounded generator/gate capture, per-run limit configuration, new module, new dependency, new destination, generic command executor, shell-free allowlist, sandbox expansion, or trusted-command policy change.

## DEC-023 — Authorize exact rollback for corrected failed delivery

- Context: corrected candidate `61fe47e0…` installed all 38 destinations and passed gates 0–13, but a transient gate-14 failure entered rollback after unrelated bytecode and host-lifecycle drift made full outside equality unreachable. All current destinations equal expected-post, every backup matches its preimage, and protected records remain exact. DEC-021 was intentionally limited to the prior failed journal.
- Decision: for journal `61fe47e0d157f9c6a0ebc560c98a3a097c5b44c864b01084544a17bdd3fc3110` only, the sole wright may validate the compiled 16/22 allowlists, candidate/root/driver identities, exact current expected-post records, pair-state guards, and all backup preimages; restore only those 38 destination preimages; keep the journal byte-identical in `PROVING`; write a private read-only rollback attestation binding the journal digest and complete observed outside mismatch; and alter no outside path. After root validates rollback and freezes a new baseline, the same wright may retry the unchanged corrected candidate as a distinct synchronous delivery with bytecode disabled for every post-prepare action and no host lifecycle transition before final outside comparison.
- Why: this removes installed product mutation without weakening the complete recursive success guarantee or pretending unrelated drift was restored. Journal-specific authority retains fail-closed locality while the new third-state guards prevent overwriting unowned destination data.
- Rejected: generalized outside-drift rollback, restoring/deleting caches or host links, Git-visible narrowing, manual journal advance, promotion to `COMMITTED`, treating the diagnostic suite pass as delivery proof, or starting a distinct delivery over currently installed targets.

## DEC-024 — Bind delivery modes to a drift-free execution interval

- Context: DEC-023 restored the corrected failed delivery, but failed history must remain at its digest-keyed directory and therefore prevents a byte-identical candidate from creating a distinct retry journal. The outside mismatch arose because pre-install RED/GREEN work occurred after prepare froze the recursive manifest; the delivery descendants themselves already suppressed bytecode.
- Decision: keep the same transaction identity and implementation. The private delivery driver requires `PYTHONDONTWRITEBYTECODE=1` at entry to prepare, install, and recover modes, and its default fixture kills deletion of that guard. After successful prepare, the sole wright runs no unrelated command or process and keeps the same host lifecycle active until install/recover reaches a durable terminal state. This one test-driver correction creates the distinct retry digest required to preserve failed history.
- Why: a small entry invariant closes the actual execution-order hole at the existing seam. It preserves full recursive outside protection and gives the writer leverage without a configurable exclusion, cache cleanup, renamed history, or second transaction module.
- Rejected: deleting/renaming failed history, random or epoch transaction suffixes, weakening identity to permit journal reuse, comment-only digest churn, cache exclusions, outside restoration, retrying the failed journal, or a new execution wrapper.

## DEC-025 — Keep delivery mutants inside the compiled candidate

- Context: the first DEC-024 retry failed in the dedicated gate because the new guard-deletion fixture copied the entire repository. Installed execution encountered ignored runtime special files that were absent from the 16-file private preflight; the guard itself was correct and automatic recovery reached `FAILED` cleanly.
- Decision: delivery-mode mutation fixtures copy only the exact compiled 16 authored regular files from the candidate root and assert that the resulting regular-file inventory equals `AUTHORED` before mutation. A fresh private candidate root preserves the failed `71f…` candidate and journal. Keep bounded failure diagnosis disposable; do not add a new retained failure-log destination to the transaction.
- Why: the interface under test is the compiled candidate, not arbitrary repository runtime state. Exact inventory gives locality and kills both full-root copying and omission while preserving the existing mutation and delivery implementation.
- Rejected: excluding named sockets or ignored directories, copying the whole repository with special-file filters, weakening the deletion mutant, editing failed history, retaining a new gate-failure artifact, or rerunning the unchanged failed candidate.

## DEC-026 — Make destination ownership atomic and execution deadlines single-use

- Context: final Code, Test, Security, and DevEx accounts found that generated installation never proves current preimage ownership, rollback checks ownership before a separate destructive write, actual delivery can die with an unreconciled atomic temporary, output streaming and leader wait each receive a fresh process timeout, and several declared delivery bounds are not exercised through their real generator/gate wiring.
- Decision: keep the existing Workflow Artifact module, exact 16/22 destination set, trusted-command model, and fixed limits. Every authored, generated, and rollback destination mutation atomically claims the current object into one exact intent-derived private name, validates the claimed object against the durable preimage/expected-post pair, installs with no-replace semantics, and preserves both objects on contention or third state. Recovery validates and reconciles only the exact temporary/claim authorized by the durable current intent before recursive outside comparison. Journal replacement bytes remain stable across retry. Process launch establishes one absolute process deadline shared by output acquisition, leader wait, and group termination classification. Delivery-level mutants exercise target/journal replacement death, late lookalike and nested-Git drift, generator overflow, deadline continuity after stdout closes, and configured TERM-grace wiring.
- Why: ownership must be decided by the same atomic filesystem effect that precedes mutation; a prior observation is not authority. One deadline and intent-derived recovery retain locality at the existing implementation seam while preventing data loss and unbounded execution without adding a module or configuration surface.
- Rejected: check-then-overwrite/unlink, permanent temporary-path manifest exclusions, broad scratch exclusions, platform-specific exchange syscalls, generalized outside-drift rollback, a second executor, configurable limits, new dependencies, new product destinations, or weakening full recursive outside protection.

## DEC-027 — Bind atomic ownership in the canonical module and stop on terminal rollback

- Context: DRIFT-016 implemented atomic private claim/no-replace destination ownership only in the delivery driver. A later test-only retry therefore regenerated bytes identical to the live host trees and correctly failed the delivery contract's required real generated effect. Its install failure automatically restored all 38 preimages and reached `FAILED`; invoking the restored preimage driver as recovery then rejected the newer journal schema.
- Decision: make the canonical `WA-OP-006-INSTALL` and `WA-OP-008-ROLLBACK` interface state the durable intent-derived private claim, captured pair-state validation, no-replace link, and preserve-both contention result. Prove omission through the existing canonical table test and normal host generation. Compact redundant transaction prose so the fixed instruction cap still passes. Delivery orchestration treats automatic `FAILED`, `RESTORED`, `COMMITTED`, `CLEANING`, or `CLEANED` as terminal for that invocation; external recover runs only after interruption leaves a nonterminal journal and uses the candidate-capable driver.
- Why: the module interface must expose the ownership invariant callers rely on, not leave it as test-driver implementation lore. Terminal-state routing avoids applying an older restored adapter to a newer, already-terminal journal.
- Rejected: accepting an empty generated delta, fabricating generated differences, weakening generated-effect proof, reusing or deleting failed journals, widening schema compatibility in restored historical drivers, unconditional recovery after install failure, increasing the instruction cap, or adding a new module/dependency/destination.

## DEC-028 — Normalize the private generator process umask

- Context: DRIFT-018 produced the intended two canonical mirror byte changes, but the attached writer inherited umask `077`; normal host generation therefore emitted every stage file mode `0600` while live generated files were `0644`. Exact stage/current comparison correctly classified the whole tree as changed outside the 22-file allowlist and rolled back.
- Decision: keep the generator script, generated allowlist, source modes, and transaction interface unchanged. The delivery driver's private generator pre-exec sets umask `022` before descriptor-held cwd and exec; gates retain their inherited environment. A real private-generator fixture runs under parent umask `077` and proves a newly created generated file has canonical mode `0644`; deletion/mutation of the normalization fails. Actual delivery still requires nonempty differences contained in the exact 22 paths.
- Why: generation output must depend on canonical source, not the invoking agent's ambient umask. Localizing normalization to the existing private generator process preserves interface depth and prevents a host policy from fabricating whole-tree mode drift.
- Rejected: widening the generated allowlist, accepting mode-only whole-tree drift, chmod after untrusted generation, changing the repository generator or source modes, configuring the umask, applying it to proof gates, or weakening exact stage/current comparison.

## DEC-029 — Align canonical phase-gate probes with settled lifecycle wording

- Context: DRIFT-019 passed its private generator and installed-layout proofs, but the separately executed phase-gate test still required three phrases superseded by DEC-027's canonical operation-table wording: `ROLLING_BACK(n)`, `At or after durable PROVED`, and generic `count 3`.
- Decision: preserve the canonical Workflow Artifact module byte-for-byte and update only the three exact test probes to require `ROLLING_BACK(index)`, `At/after durable PROVED`, and `same-fingerprint count=3`. A distinct exact-16 candidate carries the test correction and reruns the complete affected proof before delivery.
- Why: the test interface must assert the settled canonical semantics, not stale spelling from an earlier candidate. Updating the probes restores locality without changing product behavior, retry bounds, rollback meaning, or the module interface.
- Rejected: reverting or churning canonical prose, weakening the assertions to fragments, deleting the phase gate, folding the repair into live delivery, or treating an objective stale expectation as a human-owned choice.

## DEC-030 — Isolate explicit proof bytecode in transaction-private scratch

- Context: delivery `8c7bd216…` passed all 16 gates, but `scripts/validate.sh` calls `py_compile.compile`, which ignores `PYTHONDONTWRITEBYTECODE` for explicit compilation and replaced 11 ignored source-tree cache files with byte-identical mode `0644` files.
- Decision: keep every approved command and recursive exclusion unchanged. Each gate creates one exact mode-`0700` proof cache under its digest-keyed transaction, exports that repository-relative path through `PYTHONPYCACHEPREFIX` only to gate descendants, and removes it descriptor-relatively after the process group is reaped. Prepare/install/recover reconcile only that exact owned directory before private-entry validation. Real normal, omission, orphan, wrong-type, and symlink fixtures prove the lifecycle and fail-closed behavior.
- Why: gate writes belong behind the existing delivery adapter, not in product paths. Transaction-private redirection preserves command fidelity and full recursive outside protection while keeping change, recovery, and verification local to the delivery implementation.
- Rejected: deleting or excluding `__pycache__`, chmod/restoring gate side effects, changing `scripts/validate.sh`, rewriting approved commands, external temporary directories, changing gate umask, broad private-entry acceptance, or adding a configurable cache path.

## DEC-031 — Restore only journal-proven ignored bytecode modes

- Context: Q-008 authorized recovery of the 11 byte-identical ignored cache modes changed by delivery `8c7bd216…`. Before recovery, the delivery agent's frozen isolation worktree had been removed and the distinct DRIFT-021 candidate had been added, so normal recursive recovery could no longer truthfully reach `FAILED`.
- Decision: use held no-follow descriptors to restore only the 11 journal-selected `0644` modes to frozen `0600`, preserving bytes, paths, all destinations, protected records, and journal bytes. Keep the transaction immutable at terminal `RESTORED`; preserve its post-terminal 16 candidate additions and 1,115 host-lifecycle removals as evidence, not rollback targets or generalized authority.
- Why: the bounded restoration removes the actual gate side effect without recreating ephemeral host state, deleting the next candidate, weakening recursive protection, or falsifying a `FAILED` transition whose full predecessor manifest no longer exists.
- Rejected: broad outside rollback, recreating the agent worktree, deleting DRIFT-021, marking the journal `FAILED` manually, rerunning the failed digest, changing contents, or treating Q-008 as standing rollback authority.

## DEC-032 — Persist recursive outside evidence once and harden recovery authority

- Context: final Performance, Security, and Code review found repeated 11.9 MiB journal serialization, recovery mutation before directory identity binding, legal-schema/illegal-transition promotion, missing directory metadata, terminal rollback replay, bootstrap temporary wedges, descriptor substitution races, and a non-enforced scan deadline.
- Decision: retain complete recursive protection but store one immutable transaction-private `outside-manifest.json` sidecar. Journal rows bind only relative name, SHA-256, encoded bytes, and row count. Acquisition is descriptor-stable and bounded at 200,000 rows, 16,777,216 encoded bytes, and one interrupting 600-second wall; journal bytes cap at 1,048,576. Candidate/directory identity and legal predecessor/successor state bind before mutation; exact bootstrap temporaries reconcile before destination writes; `FAILED` is idempotent and `RESTORED` advances once.
- Why: the sidecar removes generation-proportional write amplification while preserving ignored files, nested `.git`, lookalikes, directory metadata, and exact failure history. Authority and recovery remain local to one digest-keyed transaction.
- Rejected: narrowing recursive protection, excluding scratch/cache paths, embedding the payload in every journal generation, accepting ambient hostile-command expansion, deleting failed history, or introducing a generic materializer/dependency.

## DEC-033 — Keep the nonempty generated-delta guard through canonical contract change

- Context: DRIFT-024 corrected only the private driver, so normal generation produced no changed mirror and live delivery safely rejected an empty generated delta. The new sidecar behavior was real but undocumented in the canonical module.
- Decision: add one test-bound canonical sidecar paragraph to `workflow-artifacts.md`, preserve the exact generated allowlist and nonempty-delta guard, and derive only the Claude/Codex workflow-artifacts mirrors through normal host generation. Dedicated parity accepts the exact two-path preinstall delta or empty postinstall parity while frozen old/new hashes prove the canonical change remains generator-visible.
- Why: generated bytes must remain consequences of canonical source, never fabricated churn. The permanent test must pass after installation, while delivery still proves a real generated effect.
- Rejected: weakening the live guard, copying generated bytes, comment-only digest churn, widening destinations, raising the instruction cap, or keeping a preinstall-only test.

## DEC-034 — Preserve DRIFT-026 during one journal-specific rollback

- Context: failed delivery `be1bd63a…` restored all 38 destinations but recursive finalization stopped because its proven DRIFT-026 successor was created after the transaction snapshot. The exact outside delta contained 33 additions, zero removals, zero changed records, all under that immutable candidate, with digest `49dd6947…205a4`.
- Decision: Q-009 authorizes only this journal to preserve and physically recheck those 33 records, accept the exact snapshot-plus-successor union, restore the journal-selected 16 authored/22 generated preimages, and advance durable `RESTORED` sequence 197 once to `FAILED` sequence 198. Final journal SHA-256 is `5c2056131830db78feb26727c01684aeed447ddb4d159c22f8304bc3c166207c`.
- Why: the correction candidate is required to repair the rollback validator; deleting it would destroy proven work, while a broad exclusion would weaken REQ-009. Exact union validation preserves both rollback truth and recursive protection.
- Rejected: deleting DRIFT-026, treating candidate directories as generally excluded, rewriting the sidecar/history, preserving installed destinations, or generalizing Q-009 to another journal.

## DEC-035 — Resolve installed protected identity through the existing project-root seam

- Context: corrected delivery `9b75a793…` installed exact expected bytes, passed gates 0–1, then gate 2 failed before test execution because `run_gate` correctly removed delivery-private `DEVRITES_REPO_ROOT` while `require_live_protected_identity` required that variable unconditionally. Automatic rollback restored all 38 preimages and recorded terminal `FAILED` sequence 198.
- Decision: in the existing dedicated driver, resolve the protected repository with `project_root_for_tests(canonical_root())`. An installed driver whose canonical root contains `engine/go.mod` uses that root; a private candidate still requires the explicit repository environment. Preserve gate environment stripping, protected hashes, all delivery limits, and every destination/outside guard.
- Why: the existing project-root module already owns this distinction. Reusing it restores locality and makes the installed gate runnable without leaking a delivery control into proof descendants.
- Rejected: retaining `DEVRITES_REPO_ROOT` in gates, weakening protected identity, adding a second resolver, changing an approved command, retrying the failed digest, editing live source, or altering failed history.

## DEC-036 — Align exact table-header probes with canonical compact form

- Context: delivery `98b5edf…` passed gates 0–4, then `phase-gate-routing-test.sh` crashed in two embedded probes because candidate canonical tables use compact `|Route|…|` and `|Canonical adapter|…|` headers while the test searched for their pre-compaction spaced forms. Automatic rollback restored all 38 preimages and reached terminal `FAILED` sequence 201.
- Decision: create a distinct private candidate and change only the two exact header literals in `tests/phase-gate-routing-test.sh` to the canonical compact bytes. Preserve all row parsing, cardinality, field, slug-binding, mutation, and adapter-declaration assertions.
- Why: the canonical compaction is already required by the fixed instruction-size cap. Updating two stale probes restores exact contract locality without adding a parser or weakening semantic assertions.
- Rejected: re-expanding canonical prose, flexible/fuzzy header matching, changing the instruction cap, removing either embedded proof, accepting probe tracebacks, retrying the failed digest, or editing live source.

## DEC-037 — Reject unusable inherited Bash resolution before mutation

- Context: delivery `0ad1c16a…` passed gates 0–13, then gate 14 failed before test scheduling because an unrelated leading PATH entry contains a self-loop named `bash`. Node's exact `spawn('bash')` follows `execvp` semantics and returned `ELOOP`; interactive lookup skipped the bad entry and found a later valid Bash. Automatic rollback restored all 38 preimages and reached terminal `FAILED` sequence 210.
- Decision: preserve the invoking environment and approved commands. Extend the existing delivery-mode entry guard to walk inherited PATH in order and reject an `ELOOP` or absence before any journal, snapshot, or destination mutation. Prove the loop mutant. Launch the distinct retry from an explicitly corrected invoking PATH; descendants inherit that PATH unchanged.
- Why: this is a trust-boundary precondition, not environment sanitization. Failing before mutation avoids consuming a delivery on a host that cannot execute an already approved command, while keeping command meaning and proof-environment locality intact.
- Rejected: editing the unrelated PATH entry, silently rewriting PATH inside delivery, changing `scripts/run-tests.mjs`, using an absolute Bash in approved commands, deleting/reusing failed history, adding a generic command allowlist, or weakening gate 14.

## DEC-038 — Run isolated delivery-boundary fixtures two at a time

- Context: delivery `fade6b3b…` passed gates 0–13 under the corrected PATH, then gate 14 exhausted the fixed 600-second process budget. A successful disposable timing pass measured approximately 401 seconds for the dedicated contract: `check_actual_delivery_modes` consumed 323.018 seconds while all other checks totaled about 78 seconds. That check creates a fresh repository and processes for every boundary and runs all independent cases serially.
- Decision: preserve every prepare/install/rollback boundary, death code, resume, journal, observer, terminal, and exact registry assertion. Inside `check_actual_delivery_modes`, execute each isolated delivery operation with a standard-library `ThreadPoolExecutor(max_workers=2)`, retain its private temporary repository, then run signal-backed terminal validation on the main thread before adding the boundary to the existing exact-set proof. Keep setup mutants serial.
- Why: the cases share no mutable repository, journal, delivery directory, process, or environment, while Python permits the hard-wall `SIGALRM` handler only on the main thread. Bounded two-way operation scheduling reduces wall time at the measured seam without weakening the timer, coverage, locality, or test-runner concurrency.
- Rejected: raising the 600-second or 3,600-second limits, changing `scripts/run-tests.mjs`, skipping duplicate gates, environment-controlled jobs, reusing mutable fixture repositories, dropping boundaries/assertions, or expanding the destination allowlist.

## DEC-039 — Protect sibling delivery histories with compact tuple rows

- Context: independent terminal validation of `CLEANED` delivery `ee18bd94…` found that outside capture and comparison pass the whole `.generated-install` container as `excluded_prefix`, contradicting the canonical exact-selected-transaction contract and leaving every failed/successful sibling history outside recursive protection. Recomputing the correct manifest produced 89,254 rows and 18,903,043 bytes in the key-repeating object encoding, exceeding the unchanged 16,777,216-byte cap; the same records encode as 15,283,295 bytes in sorted tuple rows.
- Decision: exclude only `.generated-install/<candidate_digest>` and its descendants. Keep the container and every sibling transaction in the recursive manifest. Replace the sidecar's key-repeating object with canonical sorted tuple rows that encode the same path, type, mode, uid, gid, file nlink/SHA-256, and symlink target; reject duplicate paths, wrong widths/types, noncanonical order/bytes, and over-limit input before use. Journal binding and all fixed row/byte/wall/journal limits remain unchanged.
- Why: the correction restores the module's promised locality without deleting immutable history or weakening coverage. Tuple rows remove repeated field names rather than evidence, fitting the observed retained histories beneath the existing safety bound.
- Rejected: excluding the transaction container, excluding failed histories, deleting/renaming/chmodding history, raising limits, accepting an oversized sidecar, narrowing to Git-visible files, compressing with a new dependency, preserving both encodings through a compatibility path, or hand-editing generated mirrors.

## DEC-040 — Separate proof attestation and bind complete post-delivery state

- Context: immutable root proof `.root-proof-3e459eae4910719a` ran all 16 approved gates successfully, but adversarial recheck found that its runner also authored the attestation, destination reads followed symlinks, delivery-history equality covered journals only, per-gate proof environment controls were not recorded, and the selected sidecar was not independently reconciled with current state. The independent proof runner accepted the command/log/acceptance chain, so the candidate remains unchanged; only proof authority is incomplete.
- Decision: preserve v1 as rejected proof history and create a distinct v2 bundle. The root runner writes and seals the manifest but never the attestation; a separately invoked, manifest-bound attestor validates every file/log/hash/mode, recomputes current candidate/readiness/delivery identity, then writes the attestation and seals the directory. Candidate records use lstat/no-follow regular-file acquisition and bind uid/gid/nlink/device/inode. Pre/post snapshots bind a complete recursive digest of all delivery histories and the repository outside the active proof bundle. Each gate records exact proof-owned environment overrides and PATH digest. Sidecar replay requires every current delta to be one exact post-delivery bookkeeping/proof path; no broad exclusion or unknown drift passes.
- Why: these checks distinguish successful commands from a trustworthy evidence root without changing product behavior or rerunning source correction. Separate process authority, complete metadata, recursive state binding, and a final current-state observation close each accepted ambiguity at the proof seam.
- Rejected: rewriting v1, treating chmod as cryptographic authenticity, weakening current-state checks, embedding a private signing key, changing candidate source/tests, rerunning delivery, excluding all proof/history roots, or deferring freshness solely to narrative.

## DEC-041 — Bind post-delivery proof deltas as exact records

- Context: v4 passed all 16 approved commands and separate attestation, but its sidecar replay admitted any descendant beneath four proof-directory prefixes. That could not distinguish the known interrupted v3 residue from an arbitrary later file under the same prefix, contradicting DEC-040's exact-delta requirement.
- Decision: preserve v1–v4 unchanged and create a distinct v5 proof. Before initialization, freeze every stable post-delivery proof path and its complete sidecar record in one sealed binding; require exact set and metadata/content equality. Admit the active v5 bundle only through a fixed set of 28 exact lifecycle pathnames, never a prefix. Bind this file in the proof manifest, recompute it in the separate attestor, kill stable-record and unknown-active mutants, and perform a third implementation-independent reconstruction.
- Why: exact records provide locality and distinguish known historical evidence from unknown additions without broad exclusions, deleting interrupted history, expanding the delivery sidecar, or rerunning candidate delivery.
- Rejected: proof-directory prefix admission, deleting or repairing v1–v4, excluding all proof roots, accepting path names without record equality, changing the selected delivery, changing candidate source/tests, or treating canonical evidence rollup as candidate behavior.

## DEC-042 — Separate delivery launch authority from filesystem-write authority

- Context: delivery `b74ce22a26ed4c1c2c70a11ba23354c915a38a73352f8f7c9827c7e714bcdfd9` required one uninterrupted 968.84-second install/proof process, longer than the controlling host tool's 600-second call window. The coordinator launched the already prepared exact driver process without changing argv, environment, journal, allowlists, or destinations. The driver alone performed every filesystem mutation and reached durable `CLEANED`, but the then-current same-wright-only actor wording made that otherwise valid proof unauthorized.
- Decision: the bounded `devrites-slice-wright` remains the sole author of candidate bytes and the delivery driver remains the sole writer/restorer of all 16 authored and 22 generated destinations. When an observed host tool window cannot hold the required uninterrupted transaction, the coordinator may only launch and monitor one already prepared, hash-bound driver process using the exact Vet-approved argv, cwd, and environment frozen by the wright. The coordinator may not edit destinations, alter controls, reinterpret journal state, run a concurrent recovery, expand allowlists, or supply mutation/death/skip fixtures. A nonterminal or failed launch returns to the bounded recovery wright. Evidence records the actor split, exact process identity, constraint, and fresh proof-runner recheck.
- Why: launch authority does not create a second filesystem writer. This narrow fallback preserves one uninterrupted transaction and immutable evidence when the host's invocation window is shorter than the approved aggregate delivery bound, without concealing the observed deviation or weakening the writer seam.
- Rejected: retroactively relabeling the coordinator as the wright; coordinator destination writes; alternate argv/environment; background recovery races; broad root install authority; changing the 600/3,600-second limits; rewriting delivery `b74ce22a…`; or omitting the deviation from proof.
- Amends: DEC-008's same-wright launch wording only. Its sole-writer, exact 16/22 destination, private generation, rollback, and root-no-product-write requirements remain binding.

## DEC-043 — Enforce existing contracts at their owning seams

- Context: final Code review found three reachable implementation gaps: delivery gates bypassed the existing exact-signal validator, source promotion admitted unknown namespace siblings, and target-table decoding erased missing or repeated backtick wrappers. Independent doubt reviewers confirmed each concrete mutant and refuted the broader malformed-canonical claim because downstream bundle validation already closes it.
- Decision: pass the declared signal into the existing exact-one-standalone-line proof validator; validate the exact source namespace before canonical return or preparation; and validate all twelve raw target cells as exactly one nonempty single-line backtick pair before `[1:-1]` decoding. Add focused mutants at those same interfaces. State the signal rule once in the canonical module and update only its measured instruction baseline plus the frozen exact two-mirror generator oracle.
- Why: each correction deepens an existing module instead of adding a seam. Callers gain leverage from one signal validator, one promotion namespace rule, and one lexical admission rule; locality remains in the existing driver and canonical module.
- Rejected: substring signal acceptance, `strip("`")`, unknown-sibling cleanup, duplicate validators, weakening any limit/allowlist/boundary, or decomposing the driver without a demonstrated correctness dependency.

## DEC-044 — Bind delivery registry signals to complete output lines

- Context: strict signal validation correctly rejected delivery `6b32c06a6b55900449f625bb308c93f437405737924e9d1a37ba2321614400bf` at gate 4 because two existing `DELIVERY_GATES` expectations described substrings rather than complete standalone lines. The behavioral command emits `Validated 14 behavioral eval file(s); 82 scenario(s); 0 failed.`; the protected command emits `58c1cc88c16b9bb14b345c156703163b47c9cb6232276b50684fabae8503e8fd  .gitignore` as its first line.
- Decision: preserve exact-one-standalone-line validation and change only those two stale registry expectations to their complete observed lines. Add one focused registry regression with an independent exact-line oracle for every nonempty expectation, near-match and duplicate rejection, and live execution of only the two cheap affected commands.
- Why: the command registry owns expected output, while the proof-process module owns validation. Correcting the stale declarations at that seam preserves depth, gives every delivery caller the same exact failure meaning, and keeps proof locality in one implementation.
- Rejected: restoring substring matching, changing command output, deleting expected signals, recursively running the delivery registry from its own test, changing any approved command, retrying the failed digest, rewriting failed evidence, or weakening another delivery boundary.

## DEC-045 — Reclaim only the completed correction worktree before retry

- Context: the first DRIFT-035 preparation failed before journal creation because the recursive outside manifest exceeded its fixed 16 MiB encoded bound. Delivery `6b32c06a…` had protected thirteen worktree roots. The failed run additionally observed the retained 1,435-row `agent-ab59ff07db04e9b44` correction worktree and its own transient preparation-wright worktree; the latter was automatically removed when that wright exited. The retained worktree's sole uncommitted file was byte-identical to the already transferred canonical driver.
- Decision: verify the exact retained worktree and canonical driver hashes, delete only that transferred scratch file, and remove only that session-created disposable worktree through normal Git worktree removal. Preserve every preexisting worktree, delivery/proof history, fixed limit, recursive record, and the reusable pre-journal lock-only candidate directory. Retry the same candidate preparation once after recording the cleanup; the next transient preparation-wright root is still recursively protected, but the redundant retained root no longer consumes a second full worktree allocation.
- Why: the worktree existed only as temporary writer isolation and no longer carried unique evidence. Removing it restores sidecar capacity without weakening the manifest interface or discarding product, user, delivery, or proof state.
- Rejected: raising the 16 MiB limit, excluding `.claude/worktrees`, deleting a preexisting worktree, deleting/chmodding/renaming delivery history, narrowing recursive protection, reusing a different candidate, or retrying while the known capacity cause remained.


## DEC-046 — Accept current user-owned ACTIVE hash

- Context: successor proof v9-r2 recorded `.devrites/ACTIVE` as `9ef52ccabae0acc5264a2f1c0f404c95e8191f0261cfa525880262447c9a75dc`. This session's file is `workflow-artifact-identity` (SHA-256 `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267`). Q-010 accepted the current hash.
- Decision: keep the current ACTIVE bytes. Update only the workspace successor runner `protected_expected` to the observed hash. Do not rewrite ACTIVE, `.gitignore`, or the Workspace Observation manifest. Do not change the 38-file candidate or `test-plan.md`.
- Why: the candidate never owned ACTIVE. Restoring the frozen slug would violate the user-owned ACTIVE rule and point the workspace away from this feature.
- Rejected: rewriting ACTIVE, treating v9-r2 as attestable in this PATH, changing readiness-bound test-plan hashes.

## DEC-047 — New candidate for live ACTIVE hash

- Context: Q-011 supersedes Q-010/DEC-046 for proof. Candidate `76700e28…` still fails `require_live_protected_identity` because `tests/workflow-artifact-identity-test.sh` binds ACTIVE `9ef52cca…`. Exact `devrites-slice-wright` is available on this host.
- Decision: authorize one bounded wright correction whose intended content delta is the dedicated test's `LIVE_PROTECTED_SHA256` for `.devrites/ACTIVE` to live `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267`. Then prepare/install a new 16/22 delivery so root proof can bind a CLEANED journal to the new authored aggregate. Do not rewrite ACTIVE, `.gitignore`, or the Workspace Observation manifest.
- Why: changing the test changes the 38-file candidate. Seal requires a CLEANED delivery whose expected-post matches current destinations.
- Rejected: restoring ACTIVE, proving 76700e28 against a mismatched protected preimage, root editing the dedicated test, hand-editing generated mirrors.

## DEC-048 — Empty generated delta when generator inputs are unchanged

- Amends: DEC-027 nonempty actual-delivery generated-delta requirement, only when generator inputs are unchanged.
- Context: Q-012. The Q-011 test-only ACTIVE-hash candidate cannot CLEANED. umask-022 private generation of current canonical sources differs from live `pack/generated` at 0 of 22 admitted paths. DEC-027 required a nonempty actual-delivery generated delta and forbade fabricating differences. Spec AC-007 and `test-plan.md` already require that staged differences are a subset of the 22 admitted derivatives, not that the subset is nonempty. Journal `570f004b…` remains SNAPSHOTTING after graph.db outside drift; generated destinations were not installed.
- Decision: when generator inputs are unchanged, actual delivery may have an empty generated delta (`differences <= allowed_stage`, empty allowed). Fast-fixture delivery still requires `differences == allowed_stage`. Historical deliveries and dedicated replacement fixtures remain the nonempty replacement proof. Do not fabricate canonical edits to force a delta. Do not rename or delete journal `570f004b…`. The wright changes only the real-delivery admission in `tests/workflow-artifact-identity-test.sh`, creating a distinct authored digest, then prepare/install with no post-prepare gap.
- Why: Q-011's test-only candidate matches AC-007's subset rule. Empty staged-vs-live equality means generated destinations already are the expected-post bytes; rewriting them is not required to prove this protected-hash correction.
- Rejected: fabricating a module tweak, restoring ACTIVE, deleting/renaming `570f004b…`, excluding `.code-review-graph/graph.db`, accepting empty delta in the fast-fixture branch.

## DEC-050 — Distinct retry digest after workspace-schema delivery failure

- Context: Q-013. DEC-042 install of `aff21c0c…` reached durable `FAILED` at delivery gate-1 because `traceability.md` omitted EVID-008 and EVID-010. Root mapped those IDs; schema now passes. Destinations already matched expected-post. DEC-024 forbids deleting/renaming failed history, journal reuse, and comment-only digest churn.
- Decision: preserve FAILED `aff21c0c…` at its digest-keyed directory. One bounded wright adds an asserting workspace-schema lock to `tests/workflow-artifact-identity-test.sh` (not a comment), creating a distinct retry digest, then prepare+install with no post-prepare gap. DEC-042 may launch that already-prepared install.
- Why: same DEC-024 retry shape as prior failed deliveries. The lock would have gone red on the unmapped IDs.
- Rejected: renaming/deleting `aff21c0c…`, re-preparing that digest, comment-only churn, `$rite-plan unblock`.

## DEC-051 — Post-prove Vet fold of shared-contract packaging

- Context: 2026-08-22 `$rite-vet` independent plan review of proved candidate `bee44b1ada3b9758…` returned floor thin on Architecture because `plan.md` lacked a `Shared contract proof` table and did not pin the adapter JSON comment already enforced by `tests/workflow-artifact-identity-test.sh`.
- Decision: fold into planning artifacts only: one consuming Shared contract table; exact `<!-- workflow-artifact-adapter: {module,entry,action,return} -->` grammar with that key order; N/A admission sentence in `plan.md` pointing at existing `test-plan.md`; inspect-and-OUT SHA freeze for `README.md` and `docs/skills.md` as non-authority; protected ACTIVE preimage `fc0dd2b2…` / `workflow-artifact-identity` per Q-010/DEC-047. Do not widen the 38-file allowlist, edit operator docs, add a new admission fixture to the dedicated test, complete the spec demo example, or change the 90-second TTHW clock.
- Why: the dedicated test already pins delimiter, key order, and consuming parser; the missing table was packaging. Operator-doc migration phrases are outside AC-001 callers. Spec measurable-success aggregate 60s already binds proof subprocesses, not TTHW.
- Rejected: new candidate destinations; README/docs edit; TTHW excluding self-build or capped at 60s; spec example reference-block expansion in this rebind.

## DEC-049 — Do not seal with named Important findings

- Context: `$rite-seal` independent review of candidate `0ea164c1…` found Critical 0 and three Important items (tautological empty-delta helper; generator pathname `OUT_ROOT`; production modes trust `DEVRITES_DELIVERY_*` fixture env). Operator answered N to proceed-to-seal.
- Decision: NO-GO. Fix the three Important items in the dedicated test, then re-prove. Do not accept them as residual risk.
- Why: live CLEANED evidence is not a regression lock; fixture env and pathname generator writes are fail-closed gaps on the delivery writer.
- Rejected: GO with follow-ups, expanding the 38-file allowlist in this correction unless a held-fd wrap in the existing test is impossible.

## DEC-053 — Production fixture argv sibling via `--delivery-boundary-case operate`

- Context: Seal-correction wright added `--delivery-boundary-case operate` so mutation/skip/fast-fixture stay legal without those flags on production `--delivery-prepare`/`--delivery-install`/`--delivery-recover`. Independent `devrites-doubt-reviewer` on 2026-08-22.
- Decision: the claim holds. Production `main` rejects the three argv flags with env unset; `run_actual_delivery_mode` rewrites fixture kwargs onto operate before spawn; death-only production does not enable fast-fixture; mutation/skip consumers still run. Record as accepted.
- Why: reviewer failed to produce a reachable production-honor or coverage-drop counterexample.
- Rejected: treating operate as a silent production alias; rejecting `--delivery-test-death` on production.
- Reconfirm 2026-08-22 `$rite-build` seal-correction: independent doubt again holds on driver `35b1ce7e…` (`reject_delivery_fixture_argv` on production; operate carries fixture kwargs).

## DEC-054 — Held generator fd-cwd confinement (seal-correction)

- Context: `$rite-seal` NO-GO Important at former `:8718` — relative `DEVRITES_HOST_ARTIFACT_DIR=.held-out` after `fchdir(stage)` allowed same-UID plants to divert generator writes. Dead end: `fchdir(.held-out)` + `DEVRITES_HOST_ARTIFACT_DIR=artifacts` only moved the race one directory down.
- Decision: Private generator creates real `.held-out` and `artifacts` via `os.mkdir`/`os.open` with `dir_fd` (mkdirat semantics), `fchdir`s the child to the open artifacts inode, and sets `DEVRITES_HOST_ARTIFACT_DIR=.`. Fixtures `check_held_stage_generator_held_out_symlink_plant` and `check_held_stage_generator_artifacts_symlink_plant` require successful hoist and zero outsider writes. Independent `devrites-doubt-reviewer` 2026-08-22: claim holds. Record as accepted.
- Why: cwd is the open artifacts inode, so `$OUT_ROOT=.` does not re-walk `.held-out`/`artifacts` names; `O_NOFOLLOW` fails closed on mkdir→open TOCTOU; fixtures are non-vacuous (`generated_ok` + stage hoist).
- Rejected: dead-end `fchdir(.held-out)` + `DEVRITES_HOST_ARTIFACT_DIR=artifacts`; accepting Important residual without plant locks.

## DEC-052 — Seal-pass doubt of empty generated-delta admission

- Context: `$rite-seal` of candidate `bee44b1ada3b9758…` required an independent doubt pass on DEC-048.
- Decision: the claim holds. Actual `--delivery-install` admits empty `differences` when staged generated records equal live; `check_empty_generated_delta_install` would fail if nonempty-only `bool(differences)` were restored; fast-fixture still requires `differences == allowed_stage`.
- Why: independent reviewer failed to produce a reachable counterexample (fabricated delta, env leakage, tautological helper, live/staged mismatch, or a consumer that cannot fail).
- Rejected: treating live CLEANED evidence as the only lock; restoring DEC-027 nonempty actual-delivery admission.

## Dead ends

- Static operation tuples and `before-intent`/`after-operation` labels were not crash proof: no operation, death, restart, or observer ran.
- `run_disposable_delivery` was not evidence for the actual `--delivery-prepare`, `--delivery-install`, and `--delivery-recover` modes; it caught exceptions in one process and used in-memory snapshots.
- The walkthrough's literal output was not observed TTHW or recovery evidence; timing the full default suite before printing fixed lines cannot establish first `CLEANED` plus cursor return.
- Current-user mode checks alone do not authorize journal-selected paths, and descriptor validation followed by `cwd=<pathname>` does not preserve the held-tree guarantee.
- A general network/filesystem sandbox and cross-session descendant tracker remain rejected: they contradict the settled trusted-command process-group interface and are unavailable portably on the admitted Darwin host.
- Darwin `/dev/fd/<dir-fd>` is not a directory (`ls`/`mkdir`: Not a directory), so it cannot be `DEVRITES_HOST_ARTIFACT_DIR` for `scripts/build-host-artifacts.sh`.
- First held-out wrap `fchdir(.held-out)` plus `DEVRITES_HOST_ARTIFACT_DIR=artifacts` only moved the Important `:8718` pathname follow one directory down: same-UID `rename(artifacts)+symlink(outsider)` during `mkdir -p`/`cp -R` still writes the outsider. The `.held-out` plant fixture cannot go red on that sibling. Doubt claim FAILS; test-analyst `behavior_unverified`. Superseded by DEC-054 (`fchdir(artifacts inode)` + `DEVRITES_HOST_ARTIFACT_DIR=.`).
- Depth-mismatched red mutants of `fchdir(stage)`/`fchdir(held_out)` with plant-fixture fifo paths `../../../` timed out the actor without proving outsider write; depth-adjusted harness required for dead-end red (wright 2026-08-22).

- Successor bundle `.root-proof-76700e28bc35c871-v9-r3` cannot initialize: current `.devrites/ACTIVE` is `workflow-artifact-identity` (SHA-256 `fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267`) while the frozen runner expects `9ef52ccabae0acc5264a2f1c0f404c95e8191f0261cfa525880262447c9a75dc`. Restoring ACTIVE is forbidden. v9-r2 cannot be attested in this session because PATH SHA-256 `4242f30ef181c2350fbdff99d56a68117441e40243063e70bccb007248f337a4` != proof PATH `ae7e65f65999126cae64e19352fcd0db407009cca6fbfb5e3990c4756d51c11f`.
- DEC-046 cannot prove candidate `76700e28…` while ACTIVE is `fc0dd2b2…`. Gate 3 failed: `AssertionError: current protected byte identity` from `require_live_protected_identity` in `tests/workflow-artifact-identity-test.sh`, which still binds `9ef52cca…`. Updating that test would change the 38-file candidate.
- Q-011 wright edit plus `--delivery-prepare` produced SNAPSHOTTING journal `570f004b8893a67b718b395620544f8f53461e93f4e36e74ded761f9968312a7` (driver `2346a80e…`). DEC-042 `--delivery-install` with the exact launch tuple failed at `delivery outside manifest binding` before generated writes. Independent outside diff: only `.code-review-graph/graph.db` (`5a096e71…` → `d52381e5…`). Re-prepare of the same digest cannot recapture the sidecar while that journal exists (DEC-024 forbids deleting/renaming failed history). umask-022 private generation of current canonical sources vs live `pack/generated`: 0 file diffs, so a test-only candidate also cannot satisfy DEC-027 nonempty generated delta.
- Seal-correction delivery `aff21c0c80e683395937b81e16bd1b4c1dbd5c950ad3e9e2edf781959147f5a1` prepared SNAPSHOTTING (driver `47c82142…`, journal `f2bfaff55ad5ee53fa27e3ab90891bf8d998ede44217760534348fb43760ffec`). DEC-042 install with the exact launch tuple failed at delivery gate-1: `RuntimeError: delivery gate-1 failed: nonzero`. Reproduced independently: `python3 scripts/validate-workspace-schema.py .devrites/work/workflow-artifact-identity` → `evidence ID EVID-008 from evidence/browser proof is not mapped` and the same for `EVID-010`. Destinations already matched expected-post (0 mismatches). Deleting/renaming that FAILED journal to recapture the sidecar is forbidden by DEC-024. Comment-only digest churn is also rejected. Workspace mapping of EVID-008/EVID-010 is not an authored-destination change, so it cannot create a retry digest.

## DEC-055 — Strip private stage/backups from FAILED digests that block sibling prepare

- Context: `$rite-prove` retry after EVID-013 lock produced authored digest `8f1e14e3…`, but `--delivery-prepare` failed the fixed `OUTSIDE_MANIFEST_MAX_BYTES=16777216` bound. FAILED `58c32710…` retained `stage/` (~452 files) and `backups/` (~38 files) after gate-1 rollback; CLEANED siblings keep only lock/journal/outside-manifest. `delivery_recover` on `FAILED` is intentionally idempotent and does not strip those trees. DEC-039 rejects raising the byte limit or deleting digest-keyed history.
- Decision: authorize one bounded wright to remove only private `stage/` and `backups/` (and any `proof-cache/` if present) under every digest directory beneath `.generated-install/` that retains them (observed: many FAILED and some nonterminal histories), without deleting/renaming any digest directory, `journal.json`, `outside-manifest.json`, or `.owner.lock`. Then retry `--delivery-prepare` for current authored digest `8f1e14e3…` to durable `SNAPSHOTTING`. Do not raise `OUTSIDE_MANIFEST_MAX_BYTES`. Do not install in the same escalation unless prepare succeeds and root separately DEC-042 launches.
- Why: WA-OP-009/golden-vector require stage/backups cleaned on FAILED; leftover trees are private transaction residue, not immutable sidecar history. Stripping them restores headroom under the unchanged recursive bound without erasing failed journals.
- Rejected: raising the 16 MiB bound; deleting/renaming FAILED digests or journals; comment-only digest churn; root product destination writes; fabricating generated deltas.

## DEC-056 — Decline seal Important; fix nested OUT_ROOT basename plant

- Context: `$rite-seal` on `8c8cb87c…` NO-GO with Critical 0 · Important 1 — after DEC-054 `fchdir(artifacts)` + `DEVRITES_HOST_ARTIFACT_DIR=.`, production `mkdir -p`/`cp -R` of `claude`/`codex` under cwd still follows same-UID basename plants. Plant fixtures at held-out/artifacts stubs do not cover this shape. Operator answered N via `$rite-build` (`seal-important-accept`).
- Decision: keep Verdict NO-GO. Authorize one bounded `devrites-slice-wright` on exact path `tests/workflow-artifact-identity-test.sh` only: pre-create generator output roots with `mkdir`/`open` + `dir_fd` (`O_NOFOLLOW`) before spawn and/or BASH_ENV-wrap `mkdir`/`cp` like existing `rm`; add a plant fixture that races `claude`/`codex` basenames against the real script shape and asserts zero outsider writes + successful hoist; wire into `default_tests`. Then `$rite-prove`. Do not edit `scripts/build-host-artifacts.sh` (plan Inspected and OUT). Do not expand the 38-file allowlist.
- Why: DEC-049 forbids sealing named Important residuals; harness confinement matches DEC-054 without widening destinations.
- Rejected: accepting the Important residual; editing production `build-host-artifacts.sh` in this correction; expanding the authored allowlist.
- Doubt 2026-08-22: independent `devrites-doubt-reviewer` HOLD on pre-create+rm preserve, BASH_ENV mkdir/cp wraps, plant fixture shape, and evidence-mapping phase pin. Plant fixture goes red if confinement removed. Phase pin is not a vacuous VERIFY unblock. Record as accepted. Noise: “create race” wording over-claims preserve alone. Valid trade-offs (FYI, not re-opened Important): late basename re-plant fail-closes generation; nested dest plant under a real `claude`/`codex` head (production `copy_tree` `rm -rf $_dest`) remains out of this fixture’s named basename scope.


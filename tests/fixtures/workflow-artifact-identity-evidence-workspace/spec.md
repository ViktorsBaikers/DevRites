# Spec: Workflow Artifact Identity
Budget override: Exact admission, operation, route, diagnostic, evidence, and failure contracts stay co-located because splitting them would create competing Workflow Artifact authority.
Slug: workflow-artifact-identity
Status: Ready
Created: 2026-08-14

## Problem

Existing Workflow Artifact standard correctly gives controlling root ownership of exact `.devrites/**` executables, but its interface mixes current routing with stale one-time writer-exhaustion migration and ambiguous “materializer” language. Build, Autocomplete, Recovery, and AFK repeat policy. Identity is described as readback hashes rather than one frozen pre-active contract, and deterministic tests do not prove actual replacement failure, rollback, cleanup, identity continuity, process interruption, or rerun behavior.

## Goal

Provide one deep semantic module in existing `workflow-artifacts.md`. Vet binds admission. Root authors exact bytes in disposable preflight, retains and freezes source identity before active replacement, runs one crash-resumable artifact-set transaction, records separate durable evidence, preserves product candidate/readiness/built-slice identity, restores caller cursor, and stops before real consumptive action.

## Non-goals

- Add engine command, semantic Go policy, parser, generic reusable materializer, active hook, dependency, feature flag, schema migration, or compatibility wrapper.
- Make workflow artifacts product source/tests, product candidate rows, readiness-binding inputs, built slices, AFK charges, or shipping authority.
- Dispatch `devrites-slice-wright`, planner, reviewer, or generic agent to author workflow-artifact bytes.
- Backfill completed historical workspaces or synthesize missing identity on cold resume.
- Hand-edit `pack/generated/**`, change generator behavior, call live providers in CI, or claim unobserved model behavior.
- Execute consumptive action, Ship, commit, push, tag, publish, close out, change `.devrites/ACTIVE`, or alter user-owned `.gitignore`.

## Users / actors

| Actor | Need |
| --- | --- |
| Vet | Bind exact paths, modes, behavior, limits, fixtures, proof, rollback, and evidence grammar before Build. |
| Controlling root | Author exact bytes and execute one bounded artifact-set transaction under existing `.devrites/**` authority. |
| Build | Route workflow-only targets to root without product writer dispatch, slice charge, or candidate mutation. |
| Prove | Validate frozen identity, proof results, separation, and return cursor before real action. |
| Autocomplete | Resume only from current durable identity and journal state; missing/stale authority returns Plan/Vet. |
| Debug Recovery | Repair observed transaction fingerprint offline within shared three-attempt cap. |
| Claude/Codex adapters | Consume generated equivalents of same canonical module. |
| Maintainer | Prove routing, identity, interruption, rollback, cleanup, rerun, host parity, and product separation deterministically. |

## Requirements

- REQ-001: Existing `devrites-lib/reference/standards/workflow-artifacts.md` SHALL be sole semantic authority. Build, Prove, Autocomplete, Debug Recovery, AFK, Vet, and One-shot SHALL link to it and retain phase-local actions only.
- REQ-002: Vet SHALL admit only an exact active-workspace file set with one bound record per target: logical path, planned mode, complete behavior/interface, positive and failure fixtures, proof command/signal, rollback, and evidence fields. Vet SHALL bind integer limits for target count, per-target bytes, aggregate bytes, transaction-file cardinality, diagnostic bytes, journal lines, attempt epochs, per-proof-command seconds, aggregate proof seconds, and process-group termination grace seconds, satisfying exact relational minima in admission grammar. Directories, globs, traversal, duplicates, symlinks, product/dependency paths, unresolved choices, and missing proof SHALL return Plan/Vet before active mutation.
Admission reference blocks SHALL use exact IDs and bodies. Behavior IDs match `WA-BEH-[A-Z0-9][A-Z0-9-]*` and contain ordered fields `success,observable_effect`. Interface IDs match `WA-IF-[A-Z0-9][A-Z0-9-]*` and contain `inputs,invariants,ordering,errors,configuration,performance`. Positive fixture IDs match `WA-FIX-P[A-Z0-9][A-Z0-9-]*` and contain `setup,action,expected`. Failure fixture IDs match `WA-FIX-F[A-Z0-9][A-Z0-9-]*` and contain `setup,fault,expected`. Each block is one `## <ID>` heading followed by `DevRites workflow reference: <behavior|interface|positive-fixture|failure-fixture>`, then an exact two-column `Field | Value` table in the declared order. Values are nonempty single-line backtick-wrapped Markdown cells with odd-backslash escaped delimiters; placeholders and duplicate/referenced-but-empty blocks fail admission. `return_phase` is one canonical lifecycle phase; `return_next_action` is one exact slash action bound to `active_slug`; proof cwd is `repository-root` or `active-workspace`; proof signal is fixed nonempty ASCII; rollback is `restore-preimage-or-absence`; evidence fields are unique comma-separated lowercase identifiers. The proof command is one bounded, single-line, Vet-approved trusted repository command passed unchanged to the existing host runner; it is not runtime-untrusted input and may not contain a command-list separator.
- REQ-003: Root SHALL author exact bytes during disposable same-layout preflight. Let `slug_bytes` be validated slug UTF-8, `binding_bytes` be 32 bytes decoded from current 64-lowercase-hex Vet readiness binding, and `handle_digest = SHA-256("devrites.workflow-source.v1\\0" || uint32-be(len(slug_bytes)) || slug_bytes || binding_bytes)`. Logical handle SHALL be `wsrc:` plus lowercase hex `handle_digest`. Resolver SHALL map it exactly to active-workspace-relative `.workflow-artifact-sources/<handle_digest-hex>` under no-follow private directory descriptors; evidence stores handle only, never physical path. Green preflight SHALL atomically promote exact output to that canonical bundle before active journal or target mutation. Resolver SHALL open bundle and source files no-follow, require current-user ownership, mode `0700` on bundle, mode `0600` on regular single-link source files, exact admitted-index filenames/cardinality, and Vet-bounded size. Each file SHALL be read once from held descriptor into bounded immutable bytes; SHA-256 and stage creation SHALL use same bytes. Ordered `(path, planned mode, SHA-256)` plus Vet binding forms frozen identity. Source namespace SHALL admit owner lock plus at most one canonical retained bundle or exact stale-cleaning replacement, and at most one recognized promotion temporary within Vet byte limit; exact `.stale-cleanup` marker is allowed only during validated rollover operation; any other namespace entry blocks without deletion. Retained source SHALL remain through retryable `FAILED`, then be removed by success cleanup or exhaustion cleanup before terminal `CLEANED` or `EXHAUSTED`. Source absence after `CLEANED` is expected and never makes idempotent verification stale.
- REQ-004: Source promotion SHALL use exact same-parent temporary `.workflow-artifact-sources/.<handle_digest-hex>.preparing`. After atomic `mkdirat` mode `0700`, root SHALL atomically write/sync mode-`0600` `.authority` bytes `devrites.workflow-source-authority.v1\\nhandle=<logical-handle>\\nreadiness=<Vet-binding>\\n`; then exact eight-lowercase-hex admitted-index source files. Ordered `identity_digest` SHALL be SHA-256 of `"devrites.workflow-identity.v1\\0" || uint32-be(target_count)` followed per row by `uint32-be(len(path_utf8)) || path_utf8 || uint32-be(planned_mode) || 32-byte content_hash`. Root then atomically writes/syncs mode-`0600` `.ready` bytes `devrites.workflow-source-ready.v1\\ncount=<decimal>\\nidentity=<identity_digest-hex>\\n`; then sync temporary directory, rename to canonical `<handle_digest-hex>`, and sync parent. On restart: invalid/missing/mismatched `.authority` leaves exact-name and lookalike files untouched and routes `PLAN_VET_REPAIR`; valid authority without valid `.ready` permits cleanup of that exact temporary only and routes repair; valid authority/ready plus exact entries permits atomic promotion or recognizes already-identical canonical bundle. Unknown namespace entry or metadata mismatch fails closed without deletion. One stale canonical bundle after binding rollover is recoverable only under owner lock when marker-owned journal and journal temporary are absent, current handle differs, targets therefore remain pre-transaction by journal-before-target ordering, basename equals handle recomputed from current slug plus old authority binding, authority/ready metadata are exact, ready count matches indexed source entries, and no unknown entry exists. Root atomically writes/syncs exact mode-`0600` `.stale-cleanup` marker bound to old handle, current binding, and count; syncs bundle; renames canonical directory to `.<old-hex>.stale-cleaning`; syncs parent; then descriptor-relatively unlinks only marker, authority, ready, and indexed source entries, removes directory, and syncs parent. Durable rename is cleanup intent; restart accepts missing recognized entries only inside exact validated stale-cleaning directory and resumes idempotently. Crash before rename recognizes exact cleanup marker and resumes rename. Failure routes `WA-R022-STALE-SOURCE-GC-FAILED`; success continues current source promotion without target write or Plan/Vet. Fixtures SHALL cover binding rollover after promotion/before journal, cleanup-marker/rename/unlink/rmdir/sync interruption, retry, and unrelated/unknown-entry preservation, plus every original mkdir, marker/source/ready create, mode, write, sync, and rename boundary. After canonical source exists, recovery SHALL derive frozen identity and exact journal temporary through held descriptors and inspect/remove only that regular single-link current-user mode-`0600` file.
Stale rollover SHALL also bind durable cleanup authority in the held `.owner.lock` descriptor before canonical-to-stale rename. Exact bytes are `devrites.workflow-source-stale-intent.v1\nold_handle=wsrc:<old-64hex>\nold_readiness=<old-64hex>\ncurrent_readiness=<current-64hex>\nidentity=<64hex>\ncount=<base-10>\n`. Outside stale cleanup the lock content is empty. Under lock, root accepts only empty or byte-identical bounded intent, truncates/writes completely/syncs/readbacks intent before rename, and after the stale directory is absent plus parent sync truncates/syncs/readbacks empty content. A stale-cleaning directory, including an empty deletion suffix, is resumable only with byte-identical intent; malformed, partial, mismatched, oversized, or orphan intent fails closed without deleting the directory. Fixtures terminate around intent truncate/write/sync/readback/rename and clear, and contrast an authenticated empty suffix with an otherwise identical forged directory.
- REQ-005: Before reading or mutating canonical journal, controlling root SHALL bootstrap lock namespace through active-workspace no-follow descriptor under umask `077`: `mkdirat(".workflow-artifact-sources", 0700)` with `EEXIST` reconciliation by `openat(O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC)` and exact current-user mode-`0700` validation; then `openat(".owner.lock", O_RDWR|O_CREAT|O_EXCL|O_NOFOLLOW|O_CLOEXEC, 0600)` with `EEXIST` reconciliation by `openat(O_RDWR|O_NOFOLLOW|O_CLOEXEC)` and exact current-user regular single-link mode-`0600` validation. Newly created file and parent directory SHALL be synced. Every root SHALL acquire only Python `fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)` on that held descriptor; `lockf`, `F_SETLK`, and any other lock domain are prohibited. Missing/unsupported `fcntl`/`flock` or bootstrap/metadata failure routes `WA-R021-ACCESS-DENIED` before journal/target mutation. Busy `flock` routes `WA-R001-OWNER-BUSY`, performs no write after open, and stops invocation. Descriptor remains open through final journal generation with `O_CLOEXEC` preventing proof-child inheritance. Bootstrap interruption leaves only exact private directory/lock names; restart reconciles them. Journal carries monotonic `generation`; owner verifies current generation and owned-section SHA-256 under lock before each promotion. First active write SHALL atomically journal `PREPARING(0, stage, create)` plus frozen identity, handle, attempt epoch, and generation. Stage/backup steps SHALL be `create → write → mode → file_sync → directory_sync`; intent for every step is durable first. Complete-write loop SHALL consume full bounded immutable byte view and accept only integer progress `1..remaining`; bool, non-integer, zero, negative, oversized progress, short-write exhaustion, `ENOSPC`, or write error fails boundedly. In `PREPARING`, partial named stage/backup is an admitted recoverable state: validate metadata, unlink only that file, and recreate from held source bytes or still-unmodified target preimage. Every journal update SHALL use same-parent atomic replacement, file sync, and parent-directory sync. Relative replacement SHALL resolve parents no-follow and supply both source and destination directory handles. Resume accepts only exact pre/post/declared-partial state; ambiguity fails closed.
Active transaction files SHALL use exact active-workspace-relative names under `.workflow-artifact-transactions/<identity-digest-hex>/`: `stage/<8hex-index>` and `backup/<8hex-index>`; the transaction directory and both child directories are current-user mode `0700`, files are current-user regular single-link mode `0600`, and no unknown entry is admitted. The one evidence replacement temporary is `.evidence.md.workflow-artifact.tmp` beside `evidence.md`. All resolution is descriptor-relative and no-follow; the journal records only logical transaction handles derived from frozen identity, never caller-selected paths.
- REQ-006: State graph SHALL branch, never run rollback after proof success. Preparation advances `PREPARING → PREPARED`. Installation advances `PREPARED → INSTALLING(n) → INSTALLED → PROVING`. Success advances `PROVING → PROVED → CLEANING(n) → CLEANED`; at or after `PROVED`, recovery preserves proved targets, never rolls back/reinstalls, and resumes/reconciles cleanup. Failure before first replacement advances `→ FAILURE_CLEANING(n) → FAILED`. Failure or termination from first replacement through `PROVING` advances `→ ROLLING_BACK(n) → ROLLED_BACK → FAILURE_CLEANING(n) → FAILED`; exact preimages/absence SHALL be restored before failure cleanup. `FAILED` records exact preimages, cleanup result, attempt epoch, failure fingerprint, no-progress count, boundary, and next route. Attempt epoch begins at 1. Initial unresolved failure sets same-fingerprint no-progress count 1. After accepted offline correction and green disposable re-preflight, count below 3 permits locked `FAILED → RETRY_PREPARING(epoch+1) → PREPARING`; prior attempt row remains immutable. Process death at retry handoff resumes same epoch. Reproduction of same decisive failure increments count; success marks fingerprint resolved; different Critical/Important invariant starts distinct fingerprint without erasing history. Same-fingerprint no-progress count and total `attempt_epoch_limit` are independent bounds. Reaching either forbids another attempt and advances `FAILED → EXHAUSTED_CLEANING → EXHAUSTED`, removes retained source, sets `status=blocked`, records whether `same-fingerprint-count` or `total-epoch-limit` caused exhaustion, and sets `next_action=none — technical recovery exhausted; requires new evidence or changed failure conditions`. `WA-R020-RETRY-EXHAUSTED` covers both finite retry-budget predicates without falsely asserting that the current fingerprint reached count 3.
- REQ-007: `PROVING` SHALL launch each proof command in a fresh process group, privately capture bounded output, enforce Vet per-command and aggregate wall-clock limits, and on timeout send group termination, wait bounded grace, force-kill surviving group, reap it, record fixed timeout boundary, then enter pre-`PROVED` rollback. Source-loss response SHALL be phase-qualified. Before first target replacement, validated transaction files are bounded-cleaned and Plan/Vet receives zero-target-write result. From first replacement through `PROVING`, backups restore exact preimages, validated transaction files are failure-cleaned, then Plan/Vet receives result. At `PROVED` or `CLEANING`, proved targets remain; cleanup resumes/reconciles, source staleness is recorded, then Plan/Vet receives result without rollback or reinstall. `CLEANED` requires source GC already complete; later source absence is expected and `WA-IDEMPOTENT-RERUN` verifies evidence plus targets without source. Missing/tampered-source fixtures SHALL cover all three classes.
Proof execution assumes the exact Vet-approved repository command is trusted and that it and all descendants stay in the fresh process group until exit. No network/filesystem sandbox or deliberate detached-session containment is claimed. Within that admitted interface, inherited output is private and bounded, public output is finite, timeout always performs TERM → grace → KILL survivors → complete group/leader reap, and any surviving admitted group member prevents `PROVED`.
- REQ-008: Journal SHALL own exactly one versioned `evidence.md` section delimited by literal lines `<!-- devrites-workflow-artifact-journal:start -->` and `<!-- devrites-workflow-artifact-journal:end -->`. If absent, initial update appends one blank line plus complete section at EOF; every later update replaces bytes from start marker through end-marker newline only and preserves every byte before/after. Duplicate, nested, malformed, or over-budget markers fail before mutation. Result SHALL retain exactly one standalone `Candidate SHA-256:` binding; journal uses field `product_candidate_digest`, never second standalone binding. Section records contract version, transaction ID, attempt epoch/history, generation, owned-section preimage SHA-256, frozen identity, Vet binding, source handle, target preimage/backup rows, state/boundary/reason/route, optional exhaustion cause (`same-fingerprint-count` or `total-epoch-limit`, present exactly for `EXHAUSTED`), proof/cleanup, product candidate digest, product readiness binding, built-slice count, and caller return cursor. Attempt history is one immutable compact row per epoch within Vet line/epoch limits. Evidence SHALL contain no source/target bytes, credentials, raw hostile input, physical paths, or underlying filesystem errors.
- REQ-009: One bounded `devrites-slice-wright` SHALL author the 16 candidate destinations and generator inputs; the hash-bound delivery driver SHALL be the sole filesystem writer/restorer for those 16 authored destinations plus exact 22 generator-derived destinations. Root freezes/verifies manifests and never copies, restores, or writes product source/tests/generated files. Writer SHALL run normal host generator only with private same-filesystem `DEVRITES_HOST_ARTIFACT_DIR`; default `pack/generated` output path SHALL never be passed to destructive generator. Writer SHALL validate complete staged tree, require differences only at 22 admitted derivatives, then install/restore those exact files through feature-local private delivery journal under `.devrites/work/workflow-artifact-identity/.generated-install/` in exact candidate-digest-hex directory; no generic module is created. Journal SHALL snapshot exact 16 authored states, 22 generated preimages, and the full generated manifest; one immutable transaction-private `outside-manifest.json` sidecar SHALL hold the descriptor-stable recursively complete outside manifest while journal generations bind only its exact relative name/SHA-256/encoded bytes/row count; advance `SNAPSHOTTING → STAGED → INSTALLING(n) → INSTALLED → PROVING → COMMITTED → CLEANING → CLEANED`; and on failure before `COMMITTED` advance `ROLLING_BACK(n) → RESTORED → FAILED`. Intent precedes every replace; bytes/modes read back exact; backup/stage data are current-user private and retained through proof. Process death releases writer lock; next serial wright validates journal and resumes exact state. Root encountering incomplete delivery dispatches the bounded recovery wright and performs no product write. If an observed host tool window cannot hold the uninterrupted transaction, the coordinator MAY launch and monitor one already prepared driver using only the wright-frozen hash, exact Vet-approved argv, cwd, and environment. That launcher exception SHALL NOT authorize destination writes, alternate controls, concurrent recovery, journal reinterpretation, fixture mutation/death/skip controls, or allowlist expansion; a nonterminal result returns to the bounded recovery wright. Failure restores all 16 authored/22 generated preimages and proves full outside-allowlist manifest unchanged; after `COMMITTED`, cleanup never rolls back. Product candidate digest, readiness binding, built-slice count, protected architecture inputs, command shape, and top-level reason IDs SHALL remain unchanged. CI validates corpus, adapters, transaction fixtures, and host parity only; live host runs remain optional and claim-bounded.

The generated-delivery journal is untrusted parsed state until validated under lock. Before any destination, backup, stage, cleanup, or recovery mutation, the implementation SHALL reject unknown/missing fields and require exact contract/state, candidate and driver identities, ordered 16/22 allowlists, indices, path equality to compiled constants, derived backup handles, legal preimage state/mode/size/hash relations, counters, protected records, and sidecar-bound outside manifest. The sidecar SHALL record directory/file/symlink type and mode/uid/gid, file nlink/hash, and symlink target; protect ignored, nested-`.git`, and transaction-lookalike paths; exclude only root `.git` plus the exact selected transaction subtree; remain within 200,000 rows, 16,777,216 encoded bytes, and one 600-second wall; and remain immutable in `FAILED`/`CLEANED` histories while each journal stays within 1,048,576 bytes. Exact bootstrap sidecar/journal temporaries reconcile only before any destination mutation. Mutation iterates compiled allowlists and derived handles only, never journal-provided paths. `SNAPSHOTTING` is idempotently resumable: validate every recorded prefix row and backup against the still-unmodified destination, reconcile an orphan exact derived backup, snapshot remaining rows, require exact 16/22 cardinality, then install authored bytes. Install and recovery reject incomplete snapshots. Gate subprocesses start from the held repository directory descriptor, not a re-opened pathname. Disposable proof kills and restarts the actual prepare/install/recover modes around every durable boundary, including snapshot prefixes and cleanup, and contrasts forged path/state/backup/hash journals before mutation.

Thin adapter means module link plus phase-local entry condition, action, and return route only. Adapter SHALL NOT restate admission fields, identity/source validation, journal state graph, replacement/rollback/cleanup implementation, evidence grammar, or recovery accounting.

## Vet admission grammar

For any plan that admits one or more active Workflow Artifact targets, Vet writes exactly one `## Workflow Artifact admission` block in `test-plan.md`. A plan—such as this module-contract feature—that admits no active target writes `Workflow Artifact admission: not applicable — no active target admitted` and MUST NOT include an active admission block:

```markdown
## Workflow Artifact admission
DevRites contract: devrites.workflow-artifact-admission.v1

| Field | Value |
| --- | --- |
| active_slug | `<validated slug>` |
| readiness_binding_command | `devrites-engine check readiness --emit-binding <slug>` |
| return_phase | `<phase>` |
| return_next_action | `<exact action>` |
| target_order | `utf8-bytewise-path-ascending` |
| target_count_limit | `<positive base-10>` |
| per_target_bytes_limit | `<positive base-10>` |
| aggregate_bytes_limit | `<positive base-10>` |
| transaction_file_limit | `<positive base-10>` |
| diagnostic_bytes_limit | `256` |
| journal_line_limit | `<base-10 satisfying relational rules below and evidence.md 280-line cap>` |
| attempt_epoch_limit | `<base-10 integer at least 3>` |
| proof_command_timeout_seconds | `<positive base-10>` |
| proof_aggregate_timeout_seconds | `<positive base-10>` |
| proof_terminate_grace_seconds | `<positive base-10>` |

| Index | Path | Mode | Behavior ref | Interface ref | Positive fixture | Failure fixtures | Proof command | Proof cwd | Proof signal | Rollback | Evidence fields |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `<8hex>` | `<workspace-relative path>` | `<0[0-7]{3}>` | `<WA-BEH-ID>` | `<WA-IF-ID>` | `<WA-FIX-ID>` | `<comma-separated IDs>` | `<single command>` | `<logical cwd>` | `<fixed signal>` | `<fixed relation>` | `<comma-separated fields>` |
```

Rows sort by normalized path UTF-8 bytes ascending; index is zero-based row number encoded as eight lowercase hex digits. Mode is exactly four ASCII octal digits matching `0[0-7]{3}`, parsed base 8, and identity encoding uses resulting unsigned numeric value as uint32 big-endian. Path is normalized active-workspace-relative UTF-8 with no traversal, absolute prefix, symlink, duplicate, or newline. Table values are single-line and escape Markdown `|`; referenced behavior/interface/fixtures are complete ID-addressed blocks immediately below admission.

Relational admission is mandatory before active mutation: row count is `1..target_count_limit`; each exact content length is at most `per_target_bytes_limit`; content-length sum is at most `aggregate_bytes_limit`; `transaction_file_limit >= 3 * target_count_limit + 6` (indexed source, stage, and backup per target plus authority, ready, cleanup marker, owner lock, journal temporary, and one source directory); `diagnostic_bytes_limit` is exactly `256` and every finite diagnostic including LF must fit; `attempt_epoch_limit >= 3`; `journal_line_limit >= 30 + target_count_limit + attempt_epoch_limit` and also fits current `evidence.md` 280-line cap after outside lines; `proof_command_timeout_seconds > proof_terminate_grace_seconds > 0`; and `proof_aggregate_timeout_seconds >= proof_command_timeout_seconds`. Arithmetic uses checked nonnegative integers and rejects overflow. Dedicated fixtures cover each minimum-minus-one, exact minimum, overflow, row/content limit, and evidence-headroom boundary.

Completed one-target example:

```markdown
## Workflow Artifact admission
DevRites contract: devrites.workflow-artifact-admission.v1

| Field | Value |
| --- | --- |
| active_slug | `demo` |
| readiness_binding_command | `devrites-engine check readiness --emit-binding demo` |
| return_phase | `prove` |
| return_next_action | `/rite-prove demo` |
| target_order | `utf8-bytewise-path-ascending` |
| target_count_limit | `1` |
| per_target_bytes_limit | `1048576` |
| aggregate_bytes_limit | `1048576` |
| transaction_file_limit | `9` |
| diagnostic_bytes_limit | `256` |
| journal_line_limit | `64` |
| attempt_epoch_limit | `3` |
| proof_command_timeout_seconds | `30` |
| proof_aggregate_timeout_seconds | `60` |
| proof_terminate_grace_seconds | `2` |

| Index | Path | Mode | Behavior ref | Interface ref | Positive fixture | Failure fixtures | Proof command | Proof cwd | Proof signal | Rollback | Evidence fields |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `00000000` | `scripts/prove.py` | `0755` | `WA-BEH-001` | `WA-IF-001` | `WA-FIX-P001` | `WA-FIX-F001,WA-FIX-F002` | `python3 scripts/prove.py --fixture` | `active-workspace` | `WA-PROOF-001 PASS` | `restore-preimage-or-absence` | `mode,sha256,proof_signal` |
```

## Byte encoding and golden vector

All quoted `\\0` domain separators mean one NUL byte `0x00`, never printable backslash/zero. Lengths are unsigned uint32 big-endian byte counts. SHA-256 input concatenation contains no implicit delimiter or platform newline. Target order and index semantics are fixed by admission grammar.

Golden vector uses slug UTF-8 `demo`, readiness binding 64 zero hex digits, path UTF-8 `scripts/prove.py`, mode text `0755` → decimal `493`, and content bytes `print("ok")` followed by LF (`12` bytes):

| Value | Exact result |
| --- | --- |
| source filename | `00000000` |
| content SHA-256 | `3a66aebdedbad3cf107d24e72a07d4b735819b1cf4020fdd922f63c064708172` |
| handle digest | `1557f28b7dbf713ae3828b0dc4e914702ba34063f65393d4f8b57d99bc6af3ad` |
| logical handle | `wsrc:1557f28b7dbf713ae3828b0dc4e914702ba34063f65393d4f8b57d99bc6af3ad` |
| resolver path | `.workflow-artifact-sources/1557f28b7dbf713ae3828b0dc4e914702ba34063f65393d4f8b57d99bc6af3ad` |
| identity digest | `ce333944056552cf645c36cd03b5cd65774d167b5e920118639c6062e29f5c82` |

## Evidence journal grammar

Owned section shape is exact; target and history rows repeat within Vet limits. The `exhaustion_cause` row shown below is present exactly when `state` is `EXHAUSTED` and omitted for every other state:

```markdown
<!-- devrites-workflow-artifact-journal:start -->
## Workflow Artifact journal
DevRites contract: devrites.workflow-artifact-journal.v1

| Field | Value |
| --- | --- |
| transaction_id | `wtx:<identity-digest>` |
| attempt_epoch | `<positive base-10>` |
| attempt_id | `wta:<identity-digest>:<8hex-epoch>` |
| generation | `<nonnegative base-10>` |
| owned_section_preimage_sha256 | `<64hex or ABSENT>` |
| vet_readiness_binding | `<64hex>` |
| source_handle | `wsrc:<64hex>` |
| identity_digest | `<64hex>` |
| state | `<allowlisted state>` |
| boundary_id | `<allowlisted boundary>` |
| reason_id | `<allowlisted reason or NONE>` |
| next_route | `<allowlisted route>` |
| exhaustion_cause | `<same-fingerprint-count or total-epoch-limit>` |
| product_candidate_digest | `<64hex>` |
| product_readiness_binding | `<64hex>` |
| built_slice_count | `<nonnegative base-10>` |
| caller_return_phase | `<phase>` |
| caller_return_next_action | `<exact action>` |

| Index | Path | Mode | Content SHA-256 | Preimage | Preimage mode | Preimage SHA-256 | Backup handle | Result |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `<8hex>` | `<logical path>` | `<octal>` | `<64hex>` | `present or absent` | `<octal or NONE>` | `<64hex or NONE>` | `<opaque handle or NONE>` | `<allowlisted result>` |

| Epoch | Attempt ID | Failure fingerprint | Reason | Boundary | Progress | Result |
| --- | --- | --- | --- | --- | --- | --- |
| `<base-10>` | `<attempt ID>` | `<64hex or NONE>` | `<reason or NONE>` | `<boundary>` | `resolved or no-progress or pending` | `<allowlisted result>` |
<!-- devrites-workflow-artifact-journal:end -->
```

## Canonical operation and transition table

Canonical module carries this exact table; operation IDs and states are machine-read by dedicated test. Each operation journals durable intent through `WA-OP-013` before filesystem/process work. Test-local driver executes real disposable filesystem consumer from table. Independent oracle record schema is exactly `(operation_id, accepted_pre_state, accepted_post_state, failure_route, observer_assertion_id)`. Observer assertion runs in separate test process after consumer pauses/exits and derives filesystem entry type/mode/link/hash, journal outside/owned bytes, process-group liveness/reap, or actual engine product identity directly from OS/engine; it never reads a consumer-reported invariant or calls/trusts consumer classifier, transition, route, or reporting code. Fixed oracle maps each operation to one observer assertion ID. Mutation tests alter every canonical semantic column and each of five oracle fields independently.

| Operation ID | Attempt epoch | Durable intent | Accepted pre-state | Accepted post-state | Recoverable partial state | Failure route | Next state/operation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `WA-OP-001-OWNER-ACQUIRE` | current or 1 | none; lock precedes journal read/write | no local owner descriptor | exclusive lock held; generation/owned hash observed | none; busy owner means no write | `WAIT_ACTIVE_OWNER` or `BLOCKED_GATE` | classifier-selected operation |
| `WA-OP-002-SOURCE-PROMOTE` | 1 or retry re-preflight | exact authority/source/ready step before each create/write/mode/sync/rename | lock held; green retained bytes; no active target write | trusted canonical ready bundle | exact valid-authority `.preparing`; exact ready temp; already-identical canonical bundle | `PLAN_VET_REPAIR` | `WA-OP-003-JOURNAL-INIT` |
| `WA-OP-002A-STALE-SOURCE-GC` | pre-journal only | synced `.stale-cleanup`, then atomic rename to exact stale-cleaning directory | lock held; binding rollover; internally valid old canonical; no journal/temp/target write/unknown entry | stale bundle absent; parent synced | marker in canonical before rename or exact validated stale-cleaning directory with recognized entries missing/remaining | `OFFLINE_RECOVERY` | current `WA-OP-002-SOURCE-PROMOTE` |
| `WA-OP-003-JOURNAL-INIT` | current | `PREPARING(0,stage,create)` with frozen identity and source handle | lock held; trusted source; absent owned section; target set unmodified | generation advanced; complete owned section in `PREPARING` | exact same-parent evidence temporary; outside bytes unchanged | `OFFLINE_RECOVERY` | `WA-OP-004-STAGE-WRITE` |
| `WA-OP-004-STAGE-WRITE` | current | `(index,stage,create\|write\|mode\|file_sync\|directory_sync)` | `PREPARING`; source bytes retained; exact target parent | exact private stage bytes/mode synced | exact named partial stage whose metadata is valid | `OFFLINE_RECOVERY` | next stage or `WA-OP-005-BACKUP-WRITE` |
| `WA-OP-005-BACKUP-WRITE` | current | `(index,backup,create\|write\|mode\|file_sync\|directory_sync)` | `PREPARING`; target still unmodified | exact private backup/preimage-absence record synced | exact named partial backup whose metadata is valid | `OFFLINE_RECOVERY` | next backup or `PREPARED` |
| `WA-OP-006-INSTALL` | current | `INSTALLING(index)` names exact intent-derived private claim/install artifacts before mutation | `PREPARED`/prior `INSTALLING`; stage/backups exact | atomically move current destination into private claim; validate captured object against exact preimage/frozen expected-post pair; install desired bytes/absence no-replace; exact readback; advance/`INSTALLED` | third state/concurrent occupant preserves claim and occupant when needed and blocks | `OFFLINE_RECOVERY` | next install or `WA-OP-007-PROVE` |
| `WA-OP-007-PROVE` | current | `PROVING(command-index)` before process-group launch | `INSTALLED`; all targets read back exact | next proof command or durable `PROVED` | reaped process group with fixed failure/timeout boundary; bounded private output only | `OFFLINE_RECOVERY` | `WA-OP-008-ROLLBACK` on pre-`PROVED` failure; else `WA-OP-010-SUCCESS-CLEANUP` |
| `WA-OP-008-ROLLBACK` | current | `ROLLING_BACK(index)` names exact intent-derived private claim/install artifacts before mutation | replacement occurred; before `PROVED` | atomically move current destination into private claim; validate captured object against exact preimage/frozen expected-post pair; install desired bytes/absence no-replace; exact readback; advance/`ROLLED_BACK` | third state/concurrent occupant preserves claim and occupant when needed and blocks | `BLOCKED_GATE` if restore cannot complete | next rollback or `WA-OP-009-FAILURE-CLEANUP` |
| `WA-OP-009-FAILURE-CLEANUP` | current | `FAILURE_CLEANING(index)` before exact validated unlink/prune | zero replacements or durable `ROLLED_BACK` | stages/backups/evidence temp removed; canonical source retained; `FAILED` | exact admitted transaction file remains | `OFFLINE_RECOVERY` | `FAILED`; correction then `WA-OP-011-RETRY-HANDOFF` |
| `WA-OP-010-SUCCESS-CLEANUP` | current | `CLEANING(index)` before exact validated unlink/prune/source GC | durable `PROVED`; targets exact frozen identity | stages/backups/source/temp removed; outside evidence preserved; `CLEANED` | exact admitted cleanup file/source remains; source absence is already-cleaned | `RESUME_CLEANUP` | `CLEANED`, then `VERIFY_EXISTING`/caller return |
| `WA-OP-011-RETRY-HANDOFF` | prior+1, bounded by admission | immutable prior row plus `RETRY_PREPARING(epoch)` before handoff | locked `FAILED`; accepted correction; green re-preflight; same-fingerprint count <3 and next epoch within admitted cap | exact new epoch in `PREPARING`; prior rows unchanged | durable `RETRY_PREPARING(epoch)` with no target write | `OFFLINE_RECOVERY` | resume same epoch at `PREPARING` |
| `WA-OP-012-EXHAUSTION-GC` | current | `EXHAUSTED_CLEANING` | locked `FAILED`; same-fingerprint count=3 or admitted epoch cap reached | retained source and exact transaction files removed; `EXHAUSTED` with durable truthful `exhaustion_cause` | exact admitted cleanup file/source or missing cause remains | `BLOCKED_GATE` if safe cleanup cannot complete | `BLOCKED_EXHAUSTED`; no next attempt |
| `WA-OP-013-EVIDENCE-UPDATE` | current | next generation plus owned-section preimage SHA-256 | lock held; observed generation/hash match | atomic synced marker-owned section; generation+1; all outside bytes exact | exact same-parent current-user regular single-link mode-`0600` temporary | state operation's route; never infer success | return to invoking operation |
| `WA-OP-014-PRODUCT-SEPARATION` | current | proof boundary before comparison | frozen pre-transaction product candidate/readiness/built count | exact equality recorded in owned section | none | `BLOCKED_GATE` | success cleanup only after equality |
| `WA-OP-015-VERIFY-EXISTING` | historical epoch | no mutation intent | `CLEANED`; exact evidence and targets; source absent or already GC'd | same bytes/state/counters | none | route by finite diagnostic table | caller return; no install/retry/budget charge |

`WA-OP-004` and `WA-OP-005` complete-write fixtures inject bool, non-integer, zero, negative, oversized progress, short-write exhaustion, `ENOSPC`, ordinary write error, and process death after every accepted positive partial write. `WA-OP-007` fixtures cover command timeout, aggregate timeout, terminate grace, forced kill, descendant survival attempt, reaping, output cap, and success immediately before/after durable `PROVED`. `WA-OP-013` fixtures preserve arbitrary prior `EVID-###` rows and suffix bytes, reject duplicate/nested/malformed/over-budget markers, and prove exactly one standalone `Candidate SHA-256:` line through crash, retry, cleanup, and idempotent verification.

## Provider-neutral route matrix

| Scenario ID | Trigger | Exact route / action | Durable consequence | Forbidden behavior |
| --- | --- | --- | --- | --- |
| WA-ADMISSION-SUCCESS | current exact admission, green retained source, no active journal | `ROOT_TRANSACTION` — freeze identity and begin preparation | first active state is `PREPARING` | wright dispatch or product-slice charge |
| WA-MISSING-IDENTITY | frozen identity/handle absent | `PLAN_VET_REPAIR` | zero active journal/target writes | synthesize from chat, target, or old evidence |
| WA-STALE-IDENTITY | Vet binding, order, path, mode, hash, or source differs | `PLAN_VET_REPAIR` | zero target writes; unrelated files untouched | continue with stale bytes |
| WA-STALE-WRITER-EXHAUSTION | only obsolete actor-exhaustion evidence exists | `PLAN_VET_REPAIR` | no migration/backfill attempt | reopen via one-time migration |
| WA-FIRST-ROOT-FAILURE | first active transaction fails before `PROVED` | `OFFLINE_RECOVERY` | exact preimages, `FAILED`, attempt one under same fingerprint | terminal exhaustion or fresh action budget |
| WA-REPLACEMENT-ROLLBACK | replacement fails after earlier installs | `OFFLINE_RECOVERY` | `ROLLING_BACK → ROLLED_BACK → FAILURE_CLEANING → FAILED` | partial installed set survives |
| WA-CLEANUP | termination/failure at or after `PROVED` | `RESUME_CLEANUP` then `PLAN_VET_REPAIR` only if source stale | proved targets preserved; cleanup reconciled | rollback or reinstall proved targets |
| WA-IDENTITY-CONTINUITY | readback/proof matches frozen identity | `PROVE_AND_RETURN` | candidate/readiness/built count unchanged; cursor restored | add workflow path to product manifest |
| WA-COMPLETED-HISTORICAL | completed workspace lacks current identity | `NO_BACKFILL` | no writes or reopened phase | historical reconstruction |
| WA-IDEMPOTENT-RERUN | exact `CLEANED` evidence and targets already match | `VERIFY_EXISTING` and return | no new transaction, slice charge, or action authorization | reinstall or consume retry budget |

## Route precedence and caller actions

Classifier precedence is exact: active owner lock busy; existing safety/access/approval gate; completed historical workspace without active journal; active journal state; terminal `CLEANED`; missing/stale authority; stale writer-exhaustion evidence; current admitted set. Within active state: `PROVED`/`CLEANING` routes cleanup; install through `PROVING`, rollback, or failure-cleanup routes offline recovery; retryable `FAILED` routes offline correction/retry; `EXHAUSTED` stays blocked; `PREPARING`/`PREPARED` resumes root transaction only with current trusted authority. `CLEANED` plus matching targets always verifies existing and does not require already-GC'd source.

| Route | Owner | Exact action | Durable state/status/next action | Cursor/output |
| --- | --- | --- | --- | --- |
| `ROOT_TRANSACTION` | controlling root | execute canonical Workflow Artifact module inside current caller; no phase command | keep caller phase, `status=running`, `next_action=<saved caller action>` | save return cursor; no intermediate user reply |
| `PLAN_VET_REPAIR` | controlling root | run `/rite-plan repair <slug>` then `/rite-vet <slug>` internally | `phase=plan`, `status=running`, `next_action=/rite-plan repair <slug>` until Vet READY | restore saved caller cursor; Autocomplete emits no intermediate reply |
| `OFFLINE_RECOVERY` | controlling root | run `/devrites-debug-recovery <slug>`, disposable re-preflight, then narrow `/rite-vet <slug>` | `status=running`, `next_action=/devrites-debug-recovery <slug>`; retry only from durable `FAILED` and remaining cap | preserve cursor and attempt history; no real action |
| `RESUME_CLEANUP` | controlling root | acquire owner lock and resume exact `PROVED`/`CLEANING` generation | keep caller phase/status/action | proved targets stay; source-stale result routes Plan/Vet after cleanup |
| `PROVE_AND_RETURN` | controlling root | run only admitted proof, finish success cleanup, restore cursor | set saved return phase and `next_action=<saved return action>` | stop for fresh consumptive-action authorization |
| `VERIFY_EXISTING` | controlling root | verify `CLEANED`, exact target identity, proof, and product separation; do not reinstall | restore saved return phase/action | no retry/slice/action budget consumed |
| `NO_BACKFILL` | controlling root | no action | leave completed workspace state unchanged | no output beyond existing completed result |
| `WAIT_ACTIVE_OWNER` | non-owning invocation | perform no write; stop this invocation | do not mutate shared state | one fixed owner-busy diagnostic; caller may reinvoke after owner exits |
| `BLOCKED_EXHAUSTED` | controlling root | no attempt 4 | `status=blocked`, exact exhausted `next_action` from REQ-006 | fixed exhausted diagnostic |
| `BLOCKED_GATE` | controlling root | follow existing safety/access/approval gate | shared gate-owned state | shared gate output; never reinterpret as technical retry |

Each canonical adapter gets one row naming its entry trigger, one route token/action above, and return cursor; no adapter repeats classifier or transaction implementation.

## Diagnostic taxonomy

Public failure format is exactly one ASCII line plus LF:

`WORKFLOW_ARTIFACT_FAILURE reason_id=<reason> boundary_id=<boundary> next_route=<route>`

All three values come from finite table; malformed or unknown values collapse to `WA-R009-STATE-AMBIGUOUS`, `WA-B005-JOURNAL`, `OFFLINE_RECOVERY`. No dynamic text, target index, content, path, credential, hostile value, exception, or filesystem error appears.

| Reason ID | Boundary ID | Meaning / safe context | Next route |
| --- | --- | --- | --- |
| `WA-R001-OWNER-BUSY` | `WA-B001-OWNER` | another process holds exclusive owner lock | `WAIT_ACTIVE_OWNER` |
| `WA-R002-ADMISSION-INCOMPLETE` | `WA-B002-ADMISSION` | required admitted field/limit/proof absent or malformed | `PLAN_VET_REPAIR` |
| `WA-R003-IDENTITY-MISSING` | `WA-B004-SOURCE-OPEN` | current frozen identity unavailable | `PLAN_VET_REPAIR` |
| `WA-R004-IDENTITY-STALE` | `WA-B004-SOURCE-OPEN` | current authority differs from frozen identity | `PLAN_VET_REPAIR` |
| `WA-R005-SOURCE-UNTRUSTED` | `WA-B003-SOURCE-PROMOTE` | promotion temp/bundle lacks exact authority | `PLAN_VET_REPAIR` |
| `WA-R006-SOURCE-STALE-PREINSTALL` | `WA-B004-SOURCE-OPEN` | source stale before first replacement | `PLAN_VET_REPAIR` |
| `WA-R007-SOURCE-STALE-ACTIVE` | `WA-B004-SOURCE-OPEN` | source stale after replacement before proof | `OFFLINE_RECOVERY` |
| `WA-R008-SOURCE-STALE-POSTPROOF` | `WA-B013-SUCCESS-CLEANUP` | source stale during proved cleanup | `RESUME_CLEANUP` |
| `WA-R009-STATE-AMBIGUOUS` | `WA-B005-JOURNAL` | journal/filesystem relation not admitted | `OFFLINE_RECOVERY` |
| `WA-R010-WRITE-FAILED` | `WA-B006-STAGE-WRITE` or `WA-B007-BACKUP-WRITE` | bounded complete write failed before install | `OFFLINE_RECOVERY` |
| `WA-R011-REPLACE-FAILED` | `WA-B008-INSTALL` | target replacement failed | `OFFLINE_RECOVERY` |
| `WA-R012-READBACK-MISMATCH` | `WA-B009-READBACK` | installed bytes/mode differ from frozen identity | `OFFLINE_RECOVERY` |
| `WA-R013-PROOF-FAILED` | `WA-B010-PROVE` | admitted proof returned nonzero/wrong signal | `OFFLINE_RECOVERY` |
| `WA-R014-PROOF-TIMEOUT` | `WA-B010-PROVE` | proof process group exceeded bound | `OFFLINE_RECOVERY` |
| `WA-R015-ROLLBACK-FAILED` | `WA-B011-ROLLBACK` | exact preimages not restored | `BLOCKED_GATE` |
| `WA-R016-FAILURE-CLEANUP-FAILED` | `WA-B012-FAILURE-CLEANUP` | validated failure files remain | `OFFLINE_RECOVERY` |
| `WA-R017-SUCCESS-CLEANUP-FAILED` | `WA-B013-SUCCESS-CLEANUP` | proved target cleanup incomplete | `RESUME_CLEANUP` |
| `WA-R018-PRODUCT-IDENTITY-CHANGED` | `WA-B014-PRODUCT-SEPARATION` | product candidate/readiness/built count drifted | `BLOCKED_GATE` |
| `WA-R019-LIMIT-EXCEEDED` | `WA-B002-ADMISSION` | admitted byte/file/time/journal bound exceeded | `PLAN_VET_REPAIR` |
| `WA-R020-RETRY-EXHAUSTED` | `WA-B015-RETRY` | same-fingerprint count or total epoch limit exhausted; evidence names the causal predicate | `BLOCKED_EXHAUSTED` |
| `WA-R021-ACCESS-DENIED` | `WA-B001-OWNER` | required host access unavailable or canonical flock unsupported | `BLOCKED_GATE` |
| `WA-R022-STALE-SOURCE-GC-FAILED` | `WA-B016-STALE-SOURCE-GC` | validated pre-journal stale-bundle cleanup incomplete | `OFFLINE_RECOVERY` |

Dedicated test injects every table row, checks exact line/length/route, and rejects aliasing two actionable seams to one reason/boundary pair.

## Acceptance criteria

- [x] AC-001: **WHEN** canonical module and callers are inspected **THEN** one admission → root authorship → retained source → identity freeze → transaction → proof → resume contract exists, stale migration and generic-materializer authority are absent, and each of ten adapters contains only module link plus exact phase-local entry/action/return.
- [x] AC-002: **WHEN** exact Vet-ready set completes disposable preflight **THEN** one owner atomically retains source, reopens it no-follow, reads/hashes/stages from same descriptor-derived immutable bytes, and freezes ordered identity plus Vet binding before active mutation; busy owner or missing/malformed/stale authority performs zero target writes.
- [x] AC-003: **WHEN** fixtures terminate before/after every canonical operation or inject any partial/invalid write **THEN** only exact trusted partials are reconciled, success reaches `CLEANED` with source GC and complete installed set, pre-proof failure reaches retryable `FAILED` with exact preimages, accepted retry resumes one immutable epoch, same-fingerprint count 3 reaches `EXHAUSTED` with no attempt 4, and post-`PROVED` recovery preserves targets.
- [x] AC-004: **WHEN** source swaps/forgery/loss, owner contention, stale generation, evidence tear, complete-write failure, backup loss, omitted replacement handle, proof failure/timeout/descendant, limit breach, or process death occurs **THEN** canonical operation/route tables select exact safe result, process group is reaped, unrelated files remain untouched, and finite diagnostic leaks no unsafe value.
- [x] AC-005: **WHEN** artifact set is proved/retried/cleaned/verified **THEN** marker-owned durable evidence preserves every outside byte and exactly one standalone candidate binding while recording source/identity/preimages, immutable attempts, state/proof/cleanup, return cursor, and unchanged candidate/readiness/built-slice identities; workflow paths never enter product candidate rows.
- [x] AC-006: **WHEN** `bash tests/workflow-artifact-identity-test.sh` runs **THEN** it parses canonical operation/route/diagnostic tables, drives disposable filesystem behavior, grades independent fixed traces, requires exact ten provider-neutral scenarios and ten adapter rows, proves mutation independence, and records one-target root-caller verbatim output/TTHW; generic behavioral script separately proves schema only.
- [x] AC-007: **WHEN** the bounded wright authors the candidate and the hash-bound driver runs host generation/proof **THEN** the driver is the sole 16-authored/22-generated filesystem writer, root performs no product/generated write, generator writes only private stage, feature-local delivery journal plus immutable outside-manifest sidecar recover all preimages across interruption, complete staged tree differs only at 22 admitted derivatives, siblings remain exact, Claude/Codex bytes match canonical sources, instruction cap holds, product/protected identities remain unchanged, and existing tests pass. If the coordinator uses the DEC-042 host-window exception, evidence also proves the exact prepared hash/argv/cwd/environment, no competing recovery or fixture controls, and launcher-only authority.

At most one stage and one backup exist per target plus one recognized source-bundle temporary/canonical bundle, one owner lock, and one journal temporary. Public diagnostics use finite reason/boundary/route values in one exact ASCII line of at most 256 bytes and expose no content, credential, physical path, hostile value, or underlying filesystem error.

## Exact canonical adapter inventory

1. `pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`
2. `pack/.claude/skills/devrites-lib/reference/standards/one-shot-actions.md`
3. `pack/.claude/skills/devrites-debug-recovery/SKILL.md`
4. `pack/.claude/skills/rite-autocomplete/SKILL.md`
5. `pack/.claude/skills/rite-autocomplete/reference/loop.md`
6. `pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md`
7. `pack/.claude/skills/rite-build/SKILL.md`
8. `pack/.claude/skills/rite-build/reference/phase-contract.md`
9. `pack/.claude/skills/rite-prove/SKILL.md`
10. `pack/.claude/skills/rite-vet/SKILL.md`

Canonical authority is `pack/.claude/skills/devrites-lib/reference/standards/workflow-artifacts.md`.

## Exact source, test, and generated inventory

Authored proof paths:

- `evals/behavioral/workflow-artifact-identity.json`
- `tests/workflow-artifact-identity-test.sh`
- `tests/phase-gate-routing-test.sh`
- `tests/host-artifacts-test.sh`
- `tests/instruction-size-baseline.json`

Generated mirror roots are `pack/generated/claude/skills/` and `pack/generated/codex/skills/`. Normal generator runs only with private same-filesystem `DEVRITES_HOST_ARTIFACT_DIR`, mirrors canonical authority plus ten adapters into complete staged tree, and installs only eleven suffixes per host: 22 exact derivatives listed in `plan.md`.

Inspected-and-OUT direct references remain unchanged unless proof demonstrates drift:

- `pack/.claude/skills/devrites-lib/reference/standards/agents.md`
- `pack/.claude/skills/devrites-lib/reference/standards/README.md`
- `pack/.claude/skills/devrites-lib/reference/standards/skill-authoring.md`
- `pack/.claude/skills/rite-build/reference/wright-dispatch.md`
- `pack/.claude/skills/rite-plan/SKILL.md`
- `pack/.claude/agents/devrites-plan-drafter.md`

`scripts/build-host-artifacts.sh`, `scripts/run-behavioral-evals.sh`, and existing corpus validators remain unchanged. Root freezes exact SHA-256 preimages for inspected-and-OUT files plus full generated-tree/outside-allowlist manifest. Existing routing/host assertions replace stale migration/materializer phrases with canonical-link, operation/route/diagnostic table, branch/source-trust, evidence-preservation, bounded-recovery, private-staging, and sibling-equality assertions.

## Edge Coverage

| Edge ID | Requirement/AC | Status | Backstop |
| --- | --- | --- | --- |
| EDGE-001 | REQ-002, AC-002 | covered | exact grammar, byte golden vector, mode/index/order, limits, hostile cells/paths |
| EDGE-002 | REQ-003, REQ-004, REQ-005, AC-002, AC-004 | covered | owner/generation, source resolver, held bytes, swap/forged/lookalike/stale/GC fixtures |
| EDGE-003 | REQ-005, REQ-006, AC-003 | covered | parsed operation table; complete-write progress/errors; every intent/operation interruption |
| EDGE-004 | REQ-006, REQ-007, AC-003, AC-004 | covered | success/failure/retry/exhaustion graph, process-group timeout/reap, three source-loss classes |
| EDGE-005 | REQ-008, AC-005 | covered | marker-owned evidence preservation plus actual self-built-engine product equality |
| EDGE-006 | route/diagnostic tables, AC-006 | covered | precedence, finite diagnostics, exact ten scenarios/adapters, independent oracle, root-caller walkthrough |
| EDGE-007 | REQ-009, AC-007 | covered | private staged tree, exact 22-file recoverable install, outside equality, parity/instruction/repository proof |

## Prohibitions (must-NOT)

| ID | Rule | Proof |
| --- | --- | --- |
| PROH-001 | no generic materializer or engine command | phrase/symbol mutants and outside-allowlist identity |
| PROH-002 | no stale writer-exhaustion migration/backfill | adapter deletion and route scenarios |
| PROH-003 | no workflow path in product candidate/readiness/built-slice identity | actual engine before/after fixture |
| PROH-004 | no missing/stale identity or source synthesis | route and no-write fixtures |
| PROH-005 | no wright/read-only authoring route | adapter action checks |
| PROH-006 | no generated hand edit | normal generation and parity transaction |
| PROH-007 | no consumptive action or release mutation | standard/adapters and proof boundary |
| PROH-008 | no path validation separated from bytes used | held-descriptor immutable-byte swap mutant |
| PROH-009 | no root product/generated write, destructive default generation, or sibling install | sole-wright actor assertions plus private-stage/full-manifest/delivery-journal rollback fixtures |
| PROH-010 | no whole-file evidence ownership or duplicate candidate binding | marker/prefix/suffix/EVID/malformed-marker fixtures |
| PROH-011 | no second classifier/transition authority in adapters or oracle | parsed-table driver, fixed tuple oracle, bidirectional mutations |
| PROH-012 | no unlocked concurrent owner, unbounded proof process, retry loop, or retained-source leak | contention, timeout/reap, count-3 exhaustion, source-GC fixtures |

## Edge cases

- Two roots race; loser gets owner-busy and performs zero writes; stale generation cannot promote.
- Source promotion or first marker-owned journal replacement terminates before canonical record exists.
- Stage/backup write returns partial positive progress, invalid progress type/value, `ENOSPC`, or dies between write and sync.
- Retained source path swaps after open; held descriptor still supplies validated bytes.
- Forged handle or unrelated/lookalike temporary shares prefix but not exact trusted basename.
- Source disappears before replacement, during install/proof, after `PROVED`, or remains during terminal GC.
- Existing/absent targets occupy different admitted parents; every replace supplies both directory handles.
- Target, backup, stage, source, evidence marker, or journal state lies outside exact pre/post/declared-partial relation.
- Proof process hangs, forks descendant, exceeds output, ignores termination, or exits immediately around `PROVED`.
- Process dies at retry handoff; same fingerprint reaches count 3; distinct fingerprint preserves prior history.
- Evidence already has arbitrary lifecycle rows, suffix bytes, or malformed/duplicate/nested markers.
- Generated stage differs outside 22 paths or admitted install fails midway; all siblings/preimages remain exact.
- Completed historical workspace and stale writer-exhaustion evidence never trigger backfill.
- Thirteen shared Reslice destinations are composed: five authored files receive additive Workflow Artifact edits and eight mirrors derive only through normal generation; prior packet/route/action/stop/baseline semantics remain valid and all 30 non-overlap candidate paths plus prior workspace records stay exact.

## Measurable success

- One canonical authority, sixteen operation rows, finite route/diagnostic tables, and ten objectively thin adapter rows.
- Exactly ten provider-neutral scenario siblings with fixed IDs/outcomes.
- Dedicated test parses canonical tables, drives disposable filesystem behavior, and independently proves owner/source/complete-write/evidence/install/proof/retry/cleanup/source-GC/idempotent paths and mutants.
- One-target root-caller walkthrough records observed TTHW, exact safe output, interrupted resume, stale-authority route, and cursor return within admitted aggregate limit.
- Product candidate digest, readiness binding, and built-slice count remain byte-identical across workflow-only fixture.
- Private staged tree validates fully; exactly 22 generated derivatives install recoverably and match eleven canonical inputs while all siblings stay unchanged.
- Canonical instruction total remains at or below 855,000 bytes.
- Dedicated, behavioral, routing, prior Reslice, host, repository, security, engine-race, and full-suite checks pass on one frozen candidate.

## AI-SPEC annex

Applicable. Routing and resume policy are native instructions. Provider-neutral route matrix defines expected authored outcomes. Deterministic CI proves corpus completeness, canonical linkage, generated parity, transaction fixtures, and engine identity separation; it does not execute or certify model behavior.

## Scope boundaries

- Owns: canonical module; ten adapters; five authored proof paths; 22 generated derivatives; source-trust/state/route assertions.
- Does not own: product source/test materialization, engine policy, actual workflow executables for other features, consumptive actions, release state, generator/validator changes, or completed-workspace migration.
- Placement: deep semantic module at existing `workflow-artifacts.md`; Claude and Codex are real generated adapters at host seam.

## Open questions

None. Program grilling and DRIFT-001–DRIFT-008 repairs settled ownership, source trust, state graph, identity, executable proof, migration, host strategy, prior-candidate composition, delivery order, rollback, and exclusions.

## Readiness gate

- [x] Outcome, actors, source trust, identity, state graph, proof, resume, and separation are exact.
- [x] No product, policy, irreversible-risk, access, or acceptance choice remains.
- [x] Every requirement, route, edge, and prohibition has observable proof.
- [x] Exact source/test/generated placement and inspected-OUT references are closed.
- [x] Existing ADRs remain unchanged; prior sealed architecture semantics remain preserved, with 13 shared Reslice destinations—five authored and eight generated—explicitly composed under dedicated regression proof.

Spec gate: passed 2026-08-14

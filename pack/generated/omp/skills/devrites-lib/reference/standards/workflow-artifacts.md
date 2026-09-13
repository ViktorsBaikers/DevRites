# Workflow Artifact identity

Workflow Artifact plans/isolates/proves; never enters product candidate/readiness/
built count or authorizes consumption. Module owns semantics; callers retain only
link and entry/return.

## Contents

- [Vet admission](#vet-admission)
- [Workflow Artifact admission](#workflow-artifact-admission)
  - [Owner, namespace, promotion, and reads](#owner-namespace-promotion-and-reads)
- [Frozen source and identity](#frozen-source-and-identity)
- [Journal and complete writes](#journal-and-complete-writes)
  - [Workflow Artifact journal](#workflow-artifact-journal)
- [Canonical operation table](#canonical-operation-table)
- [State, proof, and retry](#state-proof-and-retry)
- [Route classifier](#route-classifier)
- [Public diagnostics](#public-diagnostics)
- [Phase adapters](#phase-adapters)

## Vet admission

Vet admits one `test-plan.md` block when a target is active; when no active
target, record `Workflow Artifact admission: not applicable — no active target
admitted` instead.

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
| journal_line_limit | `<positive base-10>` |
| attempt_epoch_limit | `<base-10 integer at least 3>` |
| proof_command_timeout_seconds | `<positive base-10>` |
| proof_aggregate_timeout_seconds | `<positive base-10>` |
| proof_terminate_grace_seconds | `<positive base-10>` |

| Index | Path | Mode | Behavior ref | Interface ref | Positive fixture | Failure fixtures | Proof command | Proof cwd | Proof signal | Rollback | Evidence fields |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `<8hex>` | `<workspace-relative path>` | `<0[0-7]{3}>` | `<WA-BEH-ID>` | `<WA-IF-ID>` | `<WA-FIX-ID>` | `<comma-separated IDs>` | `<single command>` | `<logical cwd>` | `<fixed signal>` | `<fixed relation>` | `<comma-separated fields>` |
```

Row order=normalized-path UTF-8; `Index`=zero-based `8hex`. Path=normalized active-
workspace-relative UTF-8; reject absolute/traversal/newline, duplicate, directory,
glob, symlink, product/dependency, or unresolved. Mode=base-8 `0[0-7]{3}`. Markdown
`|` delimits after an even backslash run; odd escapes; decoding removes only that
backslash.

IDs occur once in behavior/interface/positive/listed-failure order:
`WA-BEH-[A-Z0-9][A-Z0-9-]*`; `WA-IF-[A-Z0-9][A-Z0-9-]*`;
`WA-FIX-P[A-Z0-9][A-Z0-9-]*`; `WA-FIX-F[A-Z0-9][A-Z0-9-]*`. Each `## <ID>` plus
`DevRites workflow reference: <behavior|interface|positive-fixture|failure-fixture>`
precedes an exact ordered `Field | Value` table: behavior
`success,observable_effect`; interface `inputs,invariants,ordering,errors,
configuration,performance`; positive `setup,action,expected`; failure
`setup,fault,expected`. Values are nonempty single-line backtick cells. Before
mutation reject placeholders, missing/extra/reordered fields, duplicates, and
referenced-empty blocks.

Proof command=one unchanged trusted Vet-approved repository command; exclude `;`,
`&&`, `||`, newline, unescaped/list-separator `|`. Cwd=`repository-root` or
`active-workspace`. Signal=fixed printable ASCII without CR/LF, 1..128 bytes.
Rollback=`restore-preimage-or-absence`. Evidence=unique comma-separated lowercase
identifiers. `return_phase`=lifecycle phase; `return_next_action`=exact slash action
bound to `active_slug`. Malformed admission/reference routes Plan/Vet before mutation.

Checked nonnegative arithmetic rejects overflow. Bounds: rows
`1..target_count_limit`; per-target/aggregate content; `transaction_file_limit >=
3*target_count_limit+6`; `diagnostic_bytes_limit=256` including LF;
`attempt_epoch_limit>=3`; `journal_line_limit >=
30+target_count_limit+attempt_epoch_limit`; complete evidence <=280 lines;
`command_timeout>terminate_grace>0`; `aggregate_timeout>=command_timeout`.
Transaction-file/journal minima use declared `target_count_limit`, never row count.
Fixtures: minima/minimum-1, overflow, sparse/high-limit, content/row limits,
evidence headroom.

## Frozen source and identity

Disposable same-layout preflight authors exact bytes. `slug_bytes` is validated
slug UTF-8; `binding_bytes` decodes current 64-lowercase-hex binding:

```text
handle_digest = SHA-256(
  "devrites.workflow-source.v1\0" ||
  uint32-be(len(slug_bytes)) || slug_bytes || binding_bytes
)
source_handle = "wsrc:" || lowercase-hex(handle_digest)
resolver_path = ".workflow-artifact-sources/" || lowercase-hex(handle_digest)
```

`\0` is one NUL; lengths are uint32-be byte counts; concatenation adds nothing.
Identity rows sort by normalized-path UTF-8 bytes:

```text
identity_digest = SHA-256(
  "devrites.workflow-identity.v1\0" || uint32-be(target_count) ||
  each(uint32-be(len(path_utf8)) || path_utf8 ||
       uint32-be(planned_mode) || 32-byte-content-hash)
)
```

Golden input: slug `demo`, zero-64hex binding, `scripts/prove.py`, mode `0755`
(decimal 493), bytes `print("ok")` plus LF:

| Value | Exact result |
| --- | --- |
| source filename | `00000000` |
| content SHA-256 | `3a66aebdedbad3cf107d24e72a07d4b735819b1cf4020fdd922f63c064708172` |
| handle digest | `1557f28b7dbf713ae3828b0dc4e914702ba34063f65393d4f8b57d99bc6af3ad` |
| logical handle | `wsrc:1557f28b7dbf713ae3828b0dc4e914702ba34063f65393d4f8b57d99bc6af3ad` |
| resolver path | `.workflow-artifact-sources/1557f28b7dbf713ae3828b0dc4e914702ba34063f65393d4f8b57d99bc6af3ad` |
| identity digest | `ce333944056552cf645c36cd03b5cd65774d167b5e920118639c6062e29f5c82` |

### Owner, namespace, promotion, and reads

Pre-journal under umask `077`, active-workspace no-follow fd creates/opens
`.workflow-artifact-sources`: current-user exact `0700`. `.owner.lock`: create
`O_RDWR|O_CREAT|O_EXCL|O_NOFOLLOW|O_CLOEXEC`, `0600`; on `EEXIST`, no-follow open,
require current-user regular single-link `0600`; sync creations/parents. Use only Python `fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)`, never
lockf/F_SETLK/another domain; retain CLOEXEC fd through final generation.
Unsupported/access/metadata/bootstrap failure → `WA-R021-ACCESS-DENIED` before
mutation; busy → `WA-R001-OWNER-BUSY`, zero post-open writes. Under lock compare
monotonic generation and owned-section SHA-256 before promotion.

Namespace allowlist: `.owner.lock`; one canonical or exact stale replacement; one
recognized `.<hex>.preparing`; `.stale-cleanup` only during validated rollover.
Unknown entries block untouched. Build same-parent `.<handle-hex>.preparing`, mode
`0700`, with current-user regular single-link mode-`0600` indexed sources and exact
synced metadata:

```text
.authority:
devrites.workflow-source-authority.v1
handle=<source_handle>
readiness=<readiness-binding>

.ready:
devrites.workflow-source-ready.v1
count=<decimal-target-count>
identity=<identity-digest-hex>
```

Write `.authority`, indexes `00000000`..., `.ready`; sync directory; rename to
`<handle-hex>`; sync parent, all before journal/target mutation. Invalid/missing
authority stays untouched (`PLAN_VET_REPAIR`). Valid authority plus invalid ready
permits deleting only that preparation. Exact metadata/cardinality permits only
promotion or identical-canonical recognition. Resolve canonical no-follow,
current-user `0700`; open each bounded indexed source once, current-user regular
single-link `0600`. Held immutable bytes supply SHA-256 and stage writes; never
validate/reopen. Evidence stores logical handle only, never bytes/path.

Under owner lock, stale GC requires no journal/temp/target write, exact old
authority/readiness/count/index/slug-binding handle, and no unknown entry.
Complete-write, file-sync, read back, and directory-sync exact mode-`0600`:

```text
devrites.workflow-source-stale-cleanup.v1
old_handle=wsrc:<old-handle-hex>
current_readiness=<current-64hex-binding>
count=<positive-decimal-target-count>
```

Then file-sync/read back these exact <=512 bytes in the locked mode-`0600`
`.owner.lock` descriptor:

```text
devrites.workflow-source-stale-intent.v1
old_handle=wsrc:<old-handle-hex>
old_readiness=<old-64hex-binding>
current_readiness=<current-64hex-binding>
identity=<64hex-identity>
count=<positive-decimal-target-count>
```

Both require final LF, no extra byte. Sync bundle; rename
`.<old-hex>.stale-cleaning`; sync parent. Descriptor-relative delete order:
`.authority`, `.ready`, ascending indexes, `.stale-cleanup`, directory; sync each.
Exact authenticated intent admits only a partial or empty
remaining suffix of that order; re-authenticate the intent before every remaining deletion. Treat empty tree
without intent, malformed intent, or orphan intent lacking canonical/stale-cleaning
relation as forged: no change; route `WA-R022-STALE-SOURCE-GC-FAILED`. Only after the
stale directory is absent and parent synced, truncate the held lock intent, sync, and
read back
exact empty content.

Source persists through retryable `FAILED`; cleanup removes it before `CLEANED`/
`EXHAUSTED`; post-`CLEANED` absence is expected.

## Journal and complete writes

First atomic write records frozen identity/source/epoch/generation plus
`PREPARING(0,stage,create)`; every effect has prior durable intent. Journal replace:
same-parent current-user regular single-link mode-`0600` temp; complete-write;
file/parent sync; atomic replace; no-follow parents supply
both source and destination directory handles. Stage/backup order: `create→write→mode→file_sync→directory_sync`.
Complete-write accepts integer progress `1..remaining` only; bool/noninteger/
nonpositive/oversize, exhausted short write, `ENOSPC`, or error fails boundedly.
Resume requires exact declared pre/post/valid-partial state.

Transaction-private JSON **transaction journal**
`.workflow-artifact-transactions/<identity-digest>/journal.json` is crash/recovery
authority; marker-owned `evidence.md` **evidence journal** is bounded durable/public
evidence. Neither aliases the other. Also sole: `.../stage/<8hex>`,
`.../backup/<8hex>`, `.evidence.md.workflow-artifact.tmp`.

Evidence-journal ownership spans standalone start through end-marker LF. Exact no-separator
fragments are `<!-- ` + `devrites` + `-workflow-artifact-journal:start -->` and
`<!-- ` + `devrites` + `-workflow-artifact-journal:end -->`; below they are
`START-MARKER`/`END-MARKER`:

```markdown
START-MARKER
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
| exhaustion_cause | `<same-fingerprint-count|total-epoch-limit>` |
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
END-MARKER
```

Absent markers append one blank line and section, preserving outside bytes. Before
mutation reject duplicate/nested/malformed/over-budget markers. Keep one standalone
`Candidate SHA-256:` and only `product_candidate_digest`. `exhaustion_cause` exists
only for `EXHAUSTED` with one listed cause; otherwise absent. Immutable attempt rows
omit bytes, credentials, hostile input, paths, exceptions, and raw errors.

## Canonical operation table

Independent observer derives `(operation_id,accepted_pre_state,accepted_post_state,
failure_route,observer_assertion_id)`. At `WA-OP-014`, read current candidate/
readiness/built-slice facts from engine/OS, never consumer-authored current/frozen
values.

|Operation ID|Attempt epoch|Durable intent|Accepted pre-state|Accepted post-state|Recoverable partial state|Failure route|Next state/operation|
|---|---|---|---|---|---|---|---|
| `WA-OP-001-OWNER-ACQUIRE`|current or 1|none; lock precedes journal read/write|no local owner descriptor|exclusive lock held; generation/owned hash observed|none; busy owner means no write|`WAIT_ACTIVE_OWNER` or `BLOCKED_GATE`|classifier-selected operation|
| `WA-OP-002-SOURCE-PROMOTE`|1 or retry re-preflight|exact authority/source/ready step before each create/write/mode/sync/rename|lock held; green retained bytes; no active target write|trusted canonical ready bundle|exact valid-authority `.preparing`; exact ready temp; already-identical canonical bundle|`PLAN_VET_REPAIR`|`WA-OP-003-JOURNAL-INIT`|
| `WA-OP-002A-STALE-SOURCE-GC`|pre-journal only|synced `.stale-cleanup`, then atomic rename to exact stale-cleaning directory|lock held; binding rollover; internally valid old canonical; no journal/temp/target write/unknown entry|stale bundle absent; parent synced|marker in canonical before rename or exact validated stale-cleaning directory with recognized entries missing/remaining|`OFFLINE_RECOVERY`|current `WA-OP-002-SOURCE-PROMOTE`|
| `WA-OP-003-JOURNAL-INIT`|current|`PREPARING(0,stage,create)` with frozen identity and source handle|lock held; trusted source; absent owned section; target set unmodified|generation advanced; complete owned section in `PREPARING`|exact same-parent evidence temporary; outside bytes unchanged|`OFFLINE_RECOVERY`|`WA-OP-004-STAGE-WRITE`|
| `WA-OP-004-STAGE-WRITE`|current|`(index,stage,create\|write\|mode\|file_sync\|directory_sync)`|`PREPARING`; source bytes retained; exact target parent|exact private stage bytes/mode synced|exact named partial stage whose metadata is valid|`OFFLINE_RECOVERY`|next stage or `WA-OP-005-BACKUP-WRITE`|
| `WA-OP-005-BACKUP-WRITE`|current|`(index,backup,create\|write\|mode\|file_sync\|directory_sync)`|`PREPARING`; target still unmodified|exact private backup/preimage-absence record synced|exact named partial backup whose metadata is valid|`OFFLINE_RECOVERY`|next backup or `PREPARED`|
| `WA-OP-006-INSTALL`|current|`INSTALLING(index)` names exact intent-derived private claim/install artifacts before mutation|`PREPARED`/prior `INSTALLING`; stage/backups exact|atomically move current destination into private claim; validate captured object against exact preimage/frozen expected-post pair; install desired bytes/absence no-replace; exact readback; advance/`INSTALLED`|third state/concurrent occupant preserves claim and occupant when needed and blocks|`OFFLINE_RECOVERY`|next install or `WA-OP-007-PROVE`|
| `WA-OP-007-PROVE`|current|`PROVING(command-index)` before process-group launch|`INSTALLED`; all targets read back exact|next proof command or durable `PROVED`|reaped process group with fixed failure/timeout boundary; bounded private output only|`OFFLINE_RECOVERY`|`WA-OP-008-ROLLBACK` on pre-`PROVED` failure; else `WA-OP-010-SUCCESS-CLEANUP`|
| `WA-OP-008-ROLLBACK`|current|`ROLLING_BACK(index)` names exact intent-derived private claim/install artifacts before mutation|replacement occurred; before `PROVED`|atomically move current destination into private claim; validate captured object against exact preimage/frozen expected-post pair; install desired bytes/absence no-replace; exact readback; advance/`ROLLED_BACK`|third state/concurrent occupant preserves claim and occupant when needed and blocks|`BLOCKED_GATE` if restore cannot complete|next rollback or `WA-OP-009-FAILURE-CLEANUP`|
| `WA-OP-009-FAILURE-CLEANUP`|current|`FAILURE_CLEANING(index)` before exact validated unlink/prune|zero replacements or durable `ROLLED_BACK`|stages/backups/evidence temp removed; canonical source retained; `FAILED`|exact admitted transaction file remains|`OFFLINE_RECOVERY`|`FAILED`; correction then `WA-OP-011-RETRY-HANDOFF`|
| `WA-OP-010-SUCCESS-CLEANUP`|current|`CLEANING(index)` before exact validated unlink/prune/source GC|durable `PROVED`; targets exact frozen identity|stages/backups/source/temp removed; outside evidence preserved; `CLEANED`|exact admitted cleanup file/source remains; source absence is already-cleaned|`RESUME_CLEANUP`|`CLEANED`, then `VERIFY_EXISTING`/caller return|
| `WA-OP-011-RETRY-HANDOFF`|prior+1, bounded by admission|immutable prior row plus `RETRY_PREPARING(epoch)` before handoff|locked `FAILED`; accepted correction; green re-preflight; same-fingerprint count <3 and next epoch within admitted cap|exact new epoch in `PREPARING`; prior rows unchanged|durable `RETRY_PREPARING(epoch)` with no target write|`OFFLINE_RECOVERY`|resume same epoch at `PREPARING`|
| `WA-OP-012-EXHAUSTION-GC`|current|`EXHAUSTED_CLEANING`|locked `FAILED`; same-fingerprint count=3 or admitted epoch cap reached|retained source and exact transaction files removed; `EXHAUSTED` with durable truthful `exhaustion_cause`|exact admitted cleanup file/source or missing cause remains|`BLOCKED_GATE` if safe cleanup cannot complete|`BLOCKED_EXHAUSTED`; no next attempt|
| `WA-OP-013-EVIDENCE-UPDATE`|current|next generation plus owned-section preimage SHA-256|lock held; observed generation/hash match|atomic synced marker-owned section; generation+1; all outside bytes exact|exact same-parent current-user regular single-link mode-`0600` temporary|state operation's route; never infer success|return to invoking operation|
| `WA-OP-014-PRODUCT-SEPARATION`|current|proof boundary before comparison|frozen pre-transaction product candidate/readiness/built count|exact equality recorded in owned section|none|`BLOCKED_GATE`|success cleanup only after equality|
| `WA-OP-015-VERIFY-EXISTING`|historical epoch|no mutation intent|`CLEANED`; exact evidence and targets; source absent or already GC'd|same bytes/state/counters|none|route by finite diagnostic table|caller return; no install/retry/budget charge|

## State, proof, and retry

Success: `PREPARING → PREPARED → INSTALLING(n) → INSTALLED → PROVING → PROVED
→ CLEANING(n) → CLEANED`.

Pre-replacement failure: `FAILURE_CLEANING(n) → FAILED`. From first replacement
through `PROVING`: `WA-OP-008-ROLLBACK → ROLLED_BACK → FAILURE_CLEANING(n) →
FAILED`. At/after durable `PROVED`, preserve targets and resume cleanup only. Source
loss follows the same branches.

Delivery's one immutable transaction-private `outside-manifest.json` sidecar.
Journal binds only exact relative name, SHA-256, encoded bytes, and row count; no
generation duplicates payload. Descriptor-stable records: directory/file/symlink type/mode/uid/gid; file nlink/SHA-256,
symlink target; fifo/socket same base; block/character add nonnegative integer
non-bool `st_rdev`. Reject other types before acceptance. Protect
ignored, nested-`.git`, and transaction-lookalike paths; exclude only root
`.git` and the exact selected transaction subtree. Container/siblings protected. Limits: 200,000 rows,
16,777,216 encoded bytes, one 600-second wall, and 1,048,576 journal bytes.
Bootstrap sidecar/journal temps reconcile only before destination mutation.
Sidecar is immutable evidence in `FAILED`/`CLEANED`; stage, backups, proof-cache,
mutation artifacts clean exactly.

Candidate/destination and every generated-stage regular-file authority is acquired no-follow through one held descriptor; initial/opened/final pathname identity must match and each read caps at 16,777,216 bytes. Complete staged/current generated-tree scans share one finite absolute deadline: install delivery aggregate; recovery 600 seconds.

`PROVING` runs each trusted Vet-approved admitted argv command and its descendants

in one fresh process group, where they remain until exit; output and command/
aggregate time are bounded. A declared expected signal must be exactly one
standalone output line. It adds no network or filesystem sandbox and
makes no deliberate detached-session containment claim. Failure: `TERM`, bounded grace,
`KILL` survivors, reap group/leader, rollback. Any surviving group member, nonzero,
wrong signal, overflow, or timeout prevents `PROVED`.

Epoch starts 1. `FAILED` records preimages/cleanup/epoch/fingerprint/reason/
boundary/route/no-progress count. Resolved fingerprints close; a different
invariant gets a distinct fingerprint. Handoff death resumes its retry epoch with
prior rows immutable. `WA-OP-011/012` enforce independent fingerprint/epoch caps
and terminal source cleanup. Exhaustion records:

```text
status=blocked
exhaustion_cause=<same-fingerprint-count|total-epoch-limit>
next_action=none — technical recovery exhausted; requires new evidence or changed failure conditions
```

The journal, evidence section, and observer retain that cause; epoch exhaustion
never claims the current fingerprint reached three.

## Route classifier

Precedence: busy owner; safety/access/approval gate; completed historical; active
journal; `CLEANED`; missing/stale authority; stale writer evidence; current
admission. `PROVED|CLEANING` resumes cleanup; install-through-`PROVING`, rollback,
or failure cleanup routes offline; retryable `FAILED` routes correction/retry;
`EXHAUSTED` blocks; `PREPARING|PREPARED` resumes only with current authority.
There is no actor-history migration or backfill.

|Route|Owner|Exact action|Durable state/status/next action|Cursor/output|
| ---|---|---|---|---|
| `ROOT_TRANSACTION`|controlling root|execute this module inside current caller; no phase command|keep caller phase, `status=running`, `next_action=<saved caller action>`|save return cursor; no intermediate user reply|
| `PLAN_VET_REPAIR`|controlling root|run `/rite-plan repair <slug>` then `/rite-vet <slug>` internally|`phase=plan`, `status=running`, `next_action=/rite-plan repair <slug>` until Vet READY|restore saved caller cursor; Autocomplete emits no intermediate reply|
| `OFFLINE_RECOVERY`|controlling root|run `/devrites-debug-recovery <slug>`, disposable re-preflight, then narrow `/rite-vet <slug>`|`status=running`, `next_action=/devrites-debug-recovery <slug>`; retry only from durable `FAILED` and remaining cap|preserve cursor and attempt history; no real action|
| `RESUME_CLEANUP`|controlling root|lock and resume exact `PROVED`/`CLEANING` generation|keep caller phase/status/action|proved targets stay; stale source routes Plan/Vet after cleanup|
| `PROVE_AND_RETURN`|controlling root|run admitted proof, success cleanup, restore cursor|saved return phase/action|stop for fresh consumptive-action authorization|
| `VERIFY_EXISTING`|controlling root|verify `CLEANED`, targets, proof, and product separation; do not reinstall|restore saved return phase/action|no retry/slice/action budget|
| `NO_BACKFILL`|controlling root|no action|completed state unchanged|no new output|
| `WAIT_ACTIVE_OWNER`|non-owner|no write; stop invocation|shared state unchanged|fixed owner-busy diagnostic|
| `BLOCKED_EXHAUSTED`|controlling root|no attempt 4|blocked with exact exhausted next action|fixed exhausted diagnostic|
| `BLOCKED_GATE`|controlling root|follow existing safety/access/approval gate|gate-owned state|gate output; never reinterpret as retry|

|Scenario ID|Trigger|Exact route / action|Durable consequence|Forbidden behavior|
| ---|---|---|---|---|
| WA-ADMISSION-SUCCESS|current exact admission, green retained source, no active journal|`ROOT_TRANSACTION` — freeze identity and begin preparation|first active state is `PREPARING`|wright dispatch or product-slice charge|
| WA-MISSING-IDENTITY|frozen identity/handle absent|`PLAN_VET_REPAIR`|zero active journal/target writes|synthesize from chat, target, or old evidence|
| WA-STALE-IDENTITY|Vet binding, order, path, mode, hash, or source differs|`PLAN_VET_REPAIR`|zero target writes; unrelated files untouched|continue with stale bytes|
| WA-STALE-WRITER-EXHAUSTION|only obsolete actor-exhaustion evidence exists|`PLAN_VET_REPAIR`|no migration/backfill attempt|reopen via one-time migration|
| WA-FIRST-ROOT-FAILURE|first active transaction fails before `PROVED`|`OFFLINE_RECOVERY`|exact preimages, `FAILED`, attempt one under same fingerprint|terminal exhaustion or fresh action budget|
| WA-REPLACEMENT-ROLLBACK|replacement fails after earlier installs|`OFFLINE_RECOVERY`|`ROLLING_BACK → ROLLED_BACK → FAILURE_CLEANING → FAILED`|partial installed set survives|
| WA-CLEANUP|termination/failure at or after `PROVED`|`RESUME_CLEANUP` then `PLAN_VET_REPAIR` only if source stale|proved targets preserved; cleanup reconciled|rollback or reinstall proved targets|
| WA-IDENTITY-CONTINUITY|readback/proof matches frozen identity|`PROVE_AND_RETURN`|candidate/readiness/built count unchanged; cursor restored|add workflow path to product manifest|
| WA-COMPLETED-HISTORICAL|completed workspace lacks current identity|`NO_BACKFILL`|no writes or reopened phase|historical reconstruction|
| WA-IDEMPOTENT-RERUN|exact `CLEANED` evidence and targets already match|`VERIFY_EXISTING` and return|no new transaction, slice charge, or action authorization|reinstall or consume retry budget|

## Public diagnostics

Emit exactly one ASCII line plus LF, never dynamic text:

```text
WORKFLOW_ARTIFACT_FAILURE reason_id=<reason> boundary_id=<boundary> next_route=<route>
```

Unknown or malformed values collapse to
`WA-R009-STATE-AMBIGUOUS`, `WA-B005-JOURNAL`, `OFFLINE_RECOVERY`.
No target index, content, path, credential, hostile value, exception, or raw
filesystem error appears.

|Reason ID|Boundary ID|Meaning|Next route|
| ---|---|---|---|
| `WA-R001-OWNER-BUSY`|`WA-B001-OWNER`|exclusive owner held elsewhere|`WAIT_ACTIVE_OWNER`|
| `WA-R002-ADMISSION-INCOMPLETE`|`WA-B002-ADMISSION`|required admission absent or malformed|`PLAN_VET_REPAIR`|
| `WA-R003-IDENTITY-MISSING`|`WA-B004-SOURCE-OPEN`|current frozen identity unavailable|`PLAN_VET_REPAIR`|
| `WA-R004-IDENTITY-STALE`|`WA-B004-SOURCE-OPEN`|authority differs from frozen identity|`PLAN_VET_REPAIR`|
| `WA-R005-SOURCE-UNTRUSTED`|`WA-B003-SOURCE-PROMOTE`|source lacks exact authority|`PLAN_VET_REPAIR`|
| `WA-R006-SOURCE-STALE-PREINSTALL`|`WA-B004-SOURCE-OPEN`|stale before first replacement|`PLAN_VET_REPAIR`|
| `WA-R007-SOURCE-STALE-ACTIVE`|`WA-B004-SOURCE-OPEN`|stale after replacement before proof|`OFFLINE_RECOVERY`|
| `WA-R008-SOURCE-STALE-POSTPROOF`|`WA-B013-SUCCESS-CLEANUP`|stale during proved cleanup|`RESUME_CLEANUP`|
| `WA-R009-STATE-AMBIGUOUS`|`WA-B005-JOURNAL`|relation not admitted|`OFFLINE_RECOVERY`|
| `WA-R010-WRITE-FAILED`|`WA-B006-STAGE-WRITE` or `WA-B007-BACKUP-WRITE`|bounded write failed before install|`OFFLINE_RECOVERY`|
| `WA-R011-REPLACE-FAILED`|`WA-B008-INSTALL`|replacement failed|`OFFLINE_RECOVERY`|
| `WA-R012-READBACK-MISMATCH`|`WA-B009-READBACK`|installed identity differs|`OFFLINE_RECOVERY`|
| `WA-R013-PROOF-FAILED`|`WA-B010-PROVE`|nonzero or wrong proof signal|`OFFLINE_RECOVERY`|
| `WA-R014-PROOF-TIMEOUT`|`WA-B010-PROVE`|proof group exceeded bound|`OFFLINE_RECOVERY`|
| `WA-R015-ROLLBACK-FAILED`|`WA-B011-ROLLBACK`|preimages not restored|`BLOCKED_GATE`|
| `WA-R016-FAILURE-CLEANUP-FAILED`|`WA-B012-FAILURE-CLEANUP`|failure files remain|`OFFLINE_RECOVERY`|
| `WA-R017-SUCCESS-CLEANUP-FAILED`|`WA-B013-SUCCESS-CLEANUP`|proved cleanup incomplete|`RESUME_CLEANUP`|
| `WA-R018-PRODUCT-IDENTITY-CHANGED`|`WA-B014-PRODUCT-SEPARATION`|product identity drifted|`BLOCKED_GATE`|
| `WA-R019-LIMIT-EXCEEDED`|`WA-B002-ADMISSION`|byte/file/time/journal bound exceeded|`PLAN_VET_REPAIR`|
| `WA-R020-RETRY-EXHAUSTED`|`WA-B015-RETRY`|same-fingerprint count or total attempt epoch reached its independent cap|`BLOCKED_EXHAUSTED`|
| `WA-R021-ACCESS-DENIED`|`WA-B001-OWNER`|host access or canonical flock unavailable|`BLOCKED_GATE`|
| `WA-R022-STALE-SOURCE-GC-FAILED`|`WA-B016-STALE-SOURCE-GC`|validated stale cleanup incomplete|`OFFLINE_RECOVERY`|

Each diagnostic, including LF, is at most 256 bytes. Reason/boundary
pairs remain injective for actionable seams.

## Phase adapters

|Canonical adapter|Entry trigger|Canonical action|Return cursor|
| ---|---|---|---|
| `devrites-lib/reference/standards/afk-hitl.md`|unattended root reaches current admitted Workflow Artifact work|invoke classifier; execute returned route without wright/slice charge|saved lifecycle phase/action; no intermediate reply|
| `devrites-lib/reference/standards/one-shot-actions.md`|workflow proof completes before any consumptive one-shot action|`PROVE_AND_RETURN`; require fresh real-action authorization|saved one-shot action boundary|
| `devrites-debug-recovery/SKILL.md`|durable active failure or ambiguous admitted state|`OFFLINE_RECOVERY`; correct offline, re-preflight, narrow Vet, retry only under cap|saved caller or exact Plan/Vet route|
| `rite-autocomplete/SKILL.md`|lifecycle cursor encounters admitted set or resumable journal|invoke classifier; execute returned route internally|saved phase/action; zero intermediate reply|
| `rite-autocomplete/reference/loop.md`|loop tick sees Workflow Artifact trigger/state|invoke classifier once under owner lock; no actor-history migration|same loop cursor; no budget charge for verify/rerun|
| `rite-autocomplete/reference/stop-conditions.md`|classifier returns owner-busy, exhausted, or existing hard gate|stop on exact `WAIT_ACTIVE_OWNER`, `BLOCKED_EXHAUSTED`, or `BLOCKED_GATE` result|unchanged cursor plus fixed route-owned output|
| `rite-build/SKILL.md`|Vet-ready admitted bytes require root authorship outside product wright|`ROOT_TRANSACTION`; root writes only admitted `.devrites/**` targets|saved Build slice cursor; wright product allowlist unchanged|
| `rite-build/reference/phase-contract.md`|Build gate enters or resumes transaction|invoke canonical operation table; reconcile exact result|same slice/checkpoint cursor or Plan/Vet route|
| `rite-prove/SKILL.md`|Prove consumes installed Workflow Artifact or `CLEANED` rerun|`VERIFY_EXISTING` or admitted proof path ending `PROVE_AND_RETURN`|saved Prove cursor; stop before real action|
| `rite-vet/SKILL.md`|plan declares root-authored executable workflow file|emit exact admission; stale/missing authority uses `PLAN_VET_REPAIR`|Vet READY cursor or exact technical replan|

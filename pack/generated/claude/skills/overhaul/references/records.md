# Records, generations and freshness

## Contents

- [Run area](#run-area)
- [Canonical records](#canonical-records)
- [Publishing a generation](#publishing-a-generation)
- [Finding lifecycle](#finding-lifecycle)
- [Enforced versus review-only](#enforced-versus-review-only)
- [Validator tool](#validator-tool)
- [Dispatch packet schema](#dispatch-packet-schema)

## Run area

JSON is canonical for every machine decision; Markdown and HTML are projections
rendered from it. The coordinator is the only writer of canonical records.

```text
<run>/                       # <repo>/.devrites/overhaul/<run-id>/ from `records init`, mode 0700, git-ignored
  CURRENT                    # name of the active generation, replaced last and atomically
  g0007/                     # one coherent generation of mutable records + manifest.json + views/
  revisions/                 # immutable plan-r<N>.json, rubric-r<N>.json, profile-<id>-r<N>.json, bench-<id>-r<N>.json
  packets/                   # dispatch packets, one per attempt
  receipts/  proposals/      # immutable worker outputs, named <run>__<task>__<attempt>__<role>.json
  evidence/                  # sanitized logs, samples, traces, screenshots, red/green output
  baseline/                  # snapshot from `devrites-engine overhaul snapshot capture`
```

Placement rules: [`scope-and-safety.md`](scope-and-safety.md#run-area-and-checkpoints).

## Canonical records

| File (in a generation) | Role and key fields |
| --- | --- |
| `run.json` | `run_id`, `repo_root`, `mode` (`full`/`pr`/`branch`), `assessment_only`, `phase`, `readiness_verdict`, `execution_outcome`, `identities` (comparison, baseline, candidate), `capabilities` (tools + digests, concurrency, isolation, browser), `budget`, `revisions.plan`/`revisions.rubric` (`rev`, `file`), `degradation`, `questions[]`, `stop`, `task_outcomes` |
| `coverage.json` | `files[]`: `path`, `sha256`, `lines`, `component`, `lanes`, `eligible`, `exclusion.reason`, `binary`, `ranges[]` (`start`, `end`, `lanes`, `profiles`, `receipts{lane: [...]}`, `tools[]`, `state`); `applicability[]`: `domain`, `component`, `lane`, `state` (`applicable`/`not-applicable`/`blocked`), `reason`, `receipts[]` ([matrix](audit-domains.md#applicability-matrix)) |
| `stack-profiles.json` | `components[]` (B6 component record), `profiles[]` (`id`, `rev`, `file`, `sha256`, `status`), `dependencies[]` (dependency decisions) |
| `dispatch.json` | `attempts[]`: `task_id`, `attempt_id`, `wave`, `role`, `lane`, `profiles`, `snapshot`, `write_paths`, `authorized_by`, `status`, `dispatched_at`, `started_at`, `finished_at`, `receipt`, `changed_paths` |
| `contracts.json` | `contracts[]`: producer/consumer graph, schema/version, journeys, both-side evidence, open compatibility questions |
| `findings.json` | `findings[]` with the fields in [`orchestration.md`](orchestration.md#finding-proposals) (including `domain` and `controls`) plus `status`, `severity`, `verified_by`, `history[]` |
| `evidence.json` | `items[]`: `id`, `kind`, `path`, `sha256`, `command`, `exit`, `fingerprint`, `inputs`, `limitations`, `by`; benchmarks add `measurement` |
| `approval.json` | append-only `events[]` ([fields](lifecycle.md#the-approval-gate)) |
| `results-baseline.json`, `results-candidate.json`, `gates.json`, `scorecard*.json` | scoring inputs and calculator output ([`scoring.md`](scoring.md)) |
| `cycles.json` | append-only `cycles[]` ([`lifecycle.md`](lifecycle.md#cycle-records)) |
| `views/` | `review.*`, `report.*`, `plan.md`, each carrying the generation stamp. `publish` requires `report.{md,html}` once `execution_outcome` leaves `RUNNING`, and `review.{md,html}` plus (with a plan) `plan.md` while `phase` is `AWAITING_APPROVAL` |

`null` means "not applicable by design"; a missing value means "not recorded" —
never "passed". Worker receipts become canonical only after admission
([`orchestration.md`](orchestration.md#receipts)). An unreadable or missing record
blocks the run; it never reads as zero findings.

## Publishing a generation

Keep one staged generation open as the working generation of a phase and publish
it at the phase boundary, not after every shard: after preflight, after the audit
waves are admitted (coverage and findings), after verification and reconciliation,
at the approval gate, after each completed repair cycle, and at every stop. Record
dispatches and admissions in the working generation as they happen; `admit receipt`
checks attempts there when a staged generation based on `CURRENT` exists. Receipts,
proposals, packets, evidence and the staged generation are durable files, so a
resume loses no worker output: it continues an open stage based on `CURRENT`, or
re-admits the receipts into a new one. A writer attempt is still published as
`dispatched` before it launches, so an interruption always shows the live writer,
and every admission and settlement is recorded before the run leaves `RUNNING`
([`orchestration.md`](orchestration.md#receipts)).

1. `devrites-engine overhaul records stage <run>` copies the current generation to
   `g<N+1>.tmp` (views and manifest excluded).
2. Edit only the staged JSON. Write any new plan, rubric, profile or benchmark
   revision as a new file under `revisions/`; never rewrite an existing one.
3. Render views into the staged generation with `devrites-engine overhaul render`.
4. `devrites-engine overhaul records publish <run>` validates the staged generation
   against the previous one, writes `manifest.json` (SHA-256 of every generation file
   and every revision), renames the directory, then replaces `CURRENT` last.
5. Any reader runs `validate <run>` first. A mismatch between `CURRENT`, the manifest
   and file bytes is a mixed generation: stop applying and republish.

A crash leaves an incomplete `g<N+1>.tmp` or a complete `g<N+1>` not yet current:
`validate` reports both, `stage` refuses until you inspect it, and the previous
generation stays intact. One atomic rename is not a transaction; readers verify
digests.
Give run files restrictive permissions, redact secrets before storing evidence, and
never store credentials, tokens or private keys.

## Finding lifecycle

| From | Allowed next statuses |
| --- | --- |
| `candidate` | `needs-validation`, `confirmed`, `rejected` |
| `needs-validation` | `confirmed`, `rejected` |
| `confirmed` | `approved-for-fix`, `deferred`, `accepted-risk`, `rejected` (only after disproof) |
| `approved-for-fix` | `in-progress`, `deferred` |
| `in-progress` | `fixed-unverified`, `approved-for-fix` (attempt failed) |
| `fixed-unverified` | `verified`, `in-progress` (verification failed) |
| `verified` | `confirmed` (regressed or evidence invalidated) |
| `deferred`, `accepted-risk` | `approved-for-fix`, `confirmed` |
| `rejected` | `candidate` (new evidence) |

Each finding carries a `fingerprint`: its root-cause identity (component, symbol,
contract and failure class — never line numbers, agent, wave, severity or verdict),
unique across the record so one root cause keeps one record across cycles.
`needs-validation` names its missing fact in `blockers` and carries no severity.
Locations are repository-relative (`external: true` marks a label such as
`mobile-app:src/api.ts` for code outside the repository); a host path is rejected. A rejection
suppresses re-raising only while the cited files keep the hashes it was judged on.

Severity exists only once impact is established (`confirmed` onward). `rejected`
with reason `false_positive` requires disproving evidence. A user-declined confirmed
defect keeps `confirmed` impact and moves to `deferred` or `accepted-risk`; its
control still fails. `not reproducible yet`, `environment unavailable`,
`requirement disputed` and `proven unreachable under the approved scope` are distinct
recorded reasons, and none means safe. History is append-only: a finding never
disappears between generations.

## Enforced versus review-only

`validate`/`publish` enforce: schema tags, enums and unique IDs and fingerprints;
evidence inside the run area with a matching `sha256`; finding history that starts at
`candidate`, legal transitions (a same-status entry needs a `reason`), severity only
from `confirmed` onward, a history entry for any severity or kind change, blockers
for needs-validation, evidence for decided findings, `verified_by` naming an admitted
verifier attempt, repository-relative locations (external ones are labels, never
host paths); `revisions.plan.file` equal to `revisions/plan-r<rev>.json`, required
once approvals or writers exist; plan digest pins; known, acyclic dependencies;
allowed paths that are exact files inside the repository (no `*` or `?`, no
directories, no `.git`, no symlink or traversal escape); a separate verifier role;
approval and `accept-degradation` events with quote and trusted channel, approvals
matching the plan digest, closure and current revision, none active after a later
stop; a scorecard only with `revisions.rubric` and the same verdict as `run.json`;
no approval or writer in an assessment-only run; unchanged `run_id`, `repo_root`,
`mode` and `assessment_only`; writers authorized for their task; one live writer
per path (case-folded, symlink-resolved); no `dispatched`, `running` or `returned`
attempt once the outcome leaves `RUNNING`; one outcome per approved task at the end;
a coverage denominator that never shrinks; append-only cycles, findings history and
approval events; immutable revisions; view stamps; manifest integrity; stale stages
and interrupted publishes. Across records: every finding's `domain` is a score key;
once a rubric is frozen every confirmed finding names known `controls`, and none of
an open finding's controls is PASS in the current results; applicability cells use a
score key, a known state, a reason unless `applicable`, unique keys and receipts
inside the run; coverage ranges use the seven coverage states; a scorecard's
`G-NO-CRITICAL-HIGH` PASS needs no open critical or high finding,
`G-NO-OPEN-SERIOUS-LEAD` PASS no open lead with critical or high potential impact,
and `G-COVERAGE` PASS a non-empty matrix with no blocked cell, every applicable cell
closed by an admitted, non-gap receipt whose `domains_checked` lists its domain, and every range of
every eligible file reviewed or excluded.

Receipt admission and quote anchoring live in `devrites-engine overhaul admit`
([`orchestration.md`](orchestration.md#admission-tool)).

Review-only (the coordinator and verifiers must judge): whether evidence is
sufficient, whether a finding is real, whether coverage receipts reflect genuine
semantic review, catalog adequacy, benchmark comparability, and whether a quote in
`approval.json` really came from the user. JSON Schema tooling may add structural
checks when installed; its absence is a recorded limitation, not a pass.

## Validator tool

`devrites-engine overhaul records digest <file>` prints the SHA-256 of exact bytes; `stage <run>`,
`publish <run>` and `validate <run>` implement the generation protocol above. Exit
codes: 0 ok, 1 violations (each printed as `VIOLATION: …`), 2 usage or I/O error.

## Dispatch packet schema

`devrites-engine overhaul packet write <run> <task> <attempt> <role> [--from <fields.json>]
[--phase P] [--wave W] [--lane L] [--snapshot S] [--authorized-by A] [--brief PATH]
[--write-path PATH]...` creates `<run>/packets/<run>__<task>__<attempt>__<role>.json`
(mode 0600; `packets/` is created with mode 0700 when absent) and prints its path. `<run>` is the run-area path;
`run_id` is its last component. `--from` supplies any further fields; flags override
it. Exit codes: 0 written, 1 invalid packet or the packet already exists (never
overwritten), 2 usage or I/O error, including an extra positional argument or a
run area or `packets/` directory that is a symlink.

| Field | Rule |
| --- | --- |
| `run_id`, `task_id`, `attempt_id`, `role` | required plain names: no path separators, `.`/`..` or `__`; the tool does not check that `role` is unprefixed (`implementer`, not `overhaul-implementer`), so callers keep it so |
| `phase`, `lane`, `snapshot`, `authorized_by`, `brief` | required non-empty strings |
| `receipt_path` | must resolve, in clean form (relative or absolute), to `<run>/receipts/<run>__<task>__<attempt>__<role>.json`; a value naming any other location is rejected; an absent, empty or non-string value is replaced by that default |
| `write_paths` | array of exact paths; empty for read-only roles |
| `plan` | optional object; when present it needs `file`; `rev` and `digest` are carried through unchecked |
| `wave` and anything else in `--from` | optional, not validated, carried through unchanged (task definition, read paths, budgets, stop conditions) |

The Go package `engine/internal/overhaul/packet` exports `Validate(run, packet)` and `Load(path)`
(which also checks that the file name matches the packet's identifiers) for tools
that read packets.

`devrites-engine overhaul admit packet <run> <packet.json>` admits a packet before its worker
starts. It runs `packet.Validate` with the explicit `<run>`, then requires that the file is a
plain (not symlinked) file named `<run_id>__<task_id>__<attempt_id>__<role>.json` directly in
`<run>/packets/`, that `run_id` equals the working generation's `run.json` `run_id`, and that a
`dispatch.json` attempt in the working generation has the same `task_id` and `attempt_id` and
agrees on `role`, on `snapshot` when the attempt records one, and on `receipt` and `packet`
(run-relative) when it records them. A relative `receipt_path` is resolved from the working directory,
exactly as `packet write` resolves it, so a bare file name or `receipts/<name>.json` is rejected
unless the working directory makes it name the run's receipt file; a packet carrying an absolute
`receipt_path` is admitted from any working directory. Output and exit codes match `admit receipt`: 0
`ADMISSIBLE`, 1 `REJECT` with the reasons, 2 usage or I/O error (including a missing or
unparsable packet), 3 `DUPLICATE_OR_LATE` when the attempt is already admitted, rejected or
cancelled.

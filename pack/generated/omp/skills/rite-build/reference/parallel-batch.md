# Parallel batch (`--parallel N`)

Opt-in only. Default `/rite-build` stays [`one-slice-cycle.md`](one-slice-cycle.md).

## Entry

| Input | Behavior |
| --- | --- |
| omitted / `--parallel 1` | Serial one-slice |
| `--parallel N` (**2≤N≤10**) | Parallel when eligible + host pass |
| non-integer / `N≤0` / `N>10` | Hard refuse (no silent clamp) |

AFK caps `N` by remaining budget. Charge **only after successful integrate**
(once per integrated green sibling). Abort / integrate-failed → **0**. A
present lease is resumable: `/rite-build` continues it (repair rounds or the
blocked verdict); `parallel create` refuses until cleanup.

## Path-disjoint

Exact project-relative source/test paths only. Normalize `\`→`/`. Reject `..`,
duplicates, absolutes, `.devrites/**`. Empty pairwise intersection required.

**SSOT:** `devrites-engine check path-disjoint [--root <dir>] [<json-file>|-]`
(N≥2 only; pass `--root`). Exit `0` → fan-out; else force serial. Inspect-time
overlap → sibling **gap** (a shared path is a contract defect). **Failing
case:** two parallel wrights share a path and fan-out proceeds without that
check.

## Control vs workers

Control owns `.devrites/work/<slug>/` (lease + bookkeeping).
Workers: `.scratch/parallel-wt/<batch>/<slice>/` on
`devrites/parallel/<slug>/<batch>/<slice>` @ base `B`. Wrights never write
`.devrites/**`.

Lease: `batch_id`, `created_at`, `base_sha`, `n`,
`status` (`running|aborted|integrate-failed|complete`), `control_pid_or_session`,
`slices[]` (`id`, `paths`, `worktree_path`, `branch`, `wright_status`,
`transfer_commit`).

## Lifecycle

1. Orient/gate; parse N; select ≤N path-disjoint pending slices.
2. Write lease; freeze `B=HEAD`; `parallel create` worktrees.
3. Dispatch ≤10 wrights in parallel (cwd=worktree; allowlist; prove `HEAD==B`).
4. Inspect + fail-on-red → `green|red|gap`. Independent-review Critical is `gap`
   even when targeted tests are green.
5. Non-green siblings → **repair round** (below) in the same worktrees; lease
   stays `running` — no abort while repair budget remains. A green sibling →
   `parallel record-green`. No partial integrate. All green → integrate.
6. Integrate: each `transfer_commit` descends from `B` and diffs path-exact vs
   `B`; squash-apply all in plan order into one `WIP(<slug>):` commit on the
   staging branch (Ship collapses it). Conflict → reset to `B`,
   `integrate-failed`; classify under repair rounds (siblings stay in place).
7. Success: FF control to the squash commit — the batch lands as one local
   commit, never pushed; union `touched-files.md`; update state/evidence;
   AFK +1 per integrated sibling; optional `check candidate`.
8. Cleanup runs only after `integrate` marks the lease `complete` — work
   landed in control, so `parallel cleanup` removes worktrees/branches.
   Blocked/human-stopped batches keep lease, worktrees, branches — nothing is
   removed while the batch may resume. `parallel abort` + `cleanup --force`
   abandons the batch — the explicit human discard path; the single
   orchestrator-emitted use is the automatic re-batch under Repair rounds,
   where it acts as the salvage step only (uncommitted allowlisted changes →
   slice-branch commit; branches past `B` kept; a failed salvage keeps the
   worktree) and prints retained refs for the record. `abort` marks the lease
   only — never rewinds control: a moved HEAD is external work, reported not
   reset. Never integrate a rejected transfer commit.

## Repair rounds

Rejected work is repaired in place — never discarded and rebuilt from scratch
while budget remains.

- Record each round as `build-parallel-<batch>-repair-<round>.json`: per
  non-green sibling — status, verbatim findings, transfer_commit or null.
- Classify first: **implementation defect** (review Critical on the diff, red
  wright/proof) → repair wright in the same worktree. **Durable-plan defect**
  (falsified contract assumption, AC unchanged) →
  [`spec-drift-guard.md`](spec-drift-guard.md): batch-sweep contract
  assumptions, one folded `/rite-plan repair` + one `/rite-vet` recheck
  inline, resume repair in the same worktrees under the amended contract;
  never emit `Fix:`. **Frozen-lease scope defect** — the amended contract or
  a review requirement needs paths outside a slice's allowlist, or re-slices
  the plan (a lease is immutable once written: its allowlists are the
  disjointness guarantee) → **automatic re-batch, no human gate**:
  `parallel cleanup --force` as the salvage step, record the retained refs in
  the repair-round artifact, `parallel create` a fresh batch from control
  `HEAD` under the corrected decomposition. Prior attempts stay reachable —
  hand each new wright the salvaged branch ref
  (`devrites/parallel/<slug>/<old-batch>/<slice>`) so it builds on what worked
  instead of starting blind. If no corrected decomposition exists, that is
  exhaustion → blocked, preserve, STOP. **Reviewer stall/cancel or missing
  verdict** → re-dispatch the reviewer, not the wright. **Product/policy/
  irreversible** → stops for the human.
- Dispatch a fresh `devrites-slice-wright` (cwd = the sibling's worktree, same
  allowlist and host rules as step 3 — siblings may repair in parallel) with
  the original contract + verbatim findings + "preserve what works; fix the
  named findings; return a new `transfer_commit`". Never reset the worktree
  to `B` — that destroys the work under repair. The commit must still descend
  from `B` and diff path-exact vs `B`.
- Each repair return re-runs the full gate chain: inspect, fresh doubt +
  test-analysis, approved proof.
- Budget: the shared causal-fingerprint cap — three total failed attempts per
  sibling per fingerprint (build + repairs count together); a distinct
  Critical is a new fingerprint. Exhaustion is not abort: record one
  technical blocker (`Status: blocked`; `Next step: none — technical recovery
  exhausted for <fingerprint>; requires new evidence or changed failure
  conditions`), preserve lease/worktrees/branches, STOP — same fingerprint
  stays blocked on reinvoke; new evidence resumes repair in place. Human-
  owned decisions use the drift-guard question and wait.

## Engine verbs

```text
devrites-engine parallel create --root <repo> --slug <slug> --batch <id> --base <B> --json <file>|-
devrites-engine parallel record-green --root <repo> --slug <slug> --slice <id> --commit <sha>
devrites-engine parallel integrate --root <repo> --slug <slug> --apply-to-control
devrites-engine parallel status --root <repo> --slug <slug>
devrites-engine parallel lease-write|lease-read|lease-clear --root <repo> --slug <slug>
# abandoning the batch is human discard only; a frozen-lease re-batch may emit
# cleanup --force as its salvage step (see Repair rounds):
#   parallel abort --root <repo> --slug <slug>
#   parallel cleanup --root <repo> --slug <slug> --force   (salvages, prints retained refs)
```

`integrate` without `--apply-to-control` stages the squash commit, leaves the
lease `running` and control at `B`; cleanup still refuses — the commit cannot
be lost. `record-green` and `integrate` also accept an `integrate-failed`
lease, so a failed integrate retries after repair. `--apply-to-control`
refuses when control moved past `B` or holds uncommitted changes on slice
paths — commit/stash and retry; the batch is untouched. Go is SSOT; never
ad-hoc git/bash orchestration.

## Runtime isolation

Each parallel worktree must own isolated runtime resources. Before fan-out,
assign per-slice env (document in the lease or worktree README):

| Resource | Pattern |
| --- | --- |
| HTTP port | `PORT=<base+N>` or `DEVRITES_PARALLEL_PORT_<slice>` |
| Compose | `COMPOSE_PROJECT_NAME=devrites-<slug>-<batch>-<slice>` |
| SQLite/DB file | separate path under the worktree or `/tmp` |
| Dev server | one process per worktree; never share a listener |

Reusing the control worktree's running server, DB, or compose stack across
siblings causes cross-slice pollution and flaky proof. Force serial if
isolation cannot be established.

## Host

omp: N concurrent `task` tool / `tasks[]` batch wrights (`acceptEdits`, cwd=worktree). Codex: require
host-explicit concurrent worktree writers + native reconcile; else force serial.
Never two writers in one worktree; never root-emulated concurrency.

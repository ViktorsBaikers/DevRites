# Parallel batch (`--parallel N`)

Opt-in only. Default `$rite-build` stays [`one-slice-cycle.md`](one-slice-cycle.md).

## Entry

| Input | Behavior |
| --- | --- |
| omitted / `--parallel 1` | Serial one-slice |
| `--parallel N` (**2≤N≤10**) | N is a **cap**, not a quota: run the largest eligible set ≤ N |
| non-integer / `N≤0` / `N>10` | Hard refuse (no silent clamp) |
| Autocomplete `--parallel N` | N is the cap for this run; leftover sentinel `max_parallel` is ignored, then `$rite-autocomplete` writes or replaces only `max_parallel: N` |
| Autocomplete, no flag | Same path, cap from sentinel `max_parallel` else 10; serial only when ineligible |

AFK caps `N`. Charge green siblings once after integrate; abort /
integrate-failed → **0**. `parallel create` refuses while a lease exists.
Reconcile retained repair-round artifacts by exact causal fingerprint before honoring a stored blocked status/verdict. Blocked status, round count, or distinct finding alone cannot stop cold resume: establish missing or stale review coverage under step 4; repair unless a stop condition remains evidenced.

## Dynamic selection and re-batching

Size each batch by what is runnable **now**, never by the requested number:

```text
N_eff = min(
  cap                  -- this invocation's --parallel N (wins over leftover sentinel max_parallel)
                       | else sentinel max_parallel | else 10 (autocomplete default)
  eligible set         -- dependency-satisfied pending slices, pairwise path-disjoint
  AFK headroom         -- remaining max_slices, max_agents, max_minutes, review queue
  host capacity        -- concurrent worktree writers + the runtime isolation table
)
```

Select **greedily in plan order**: pass every currently dependency-ready pending slice
(plan order, `Files likely touched`) to
`devrites-engine parallel select --cap N [--root <dir>]`. `N_eff ≥ 2` → parallel batch;
`N_eff = 1` → one serial round; nothing pending → Prove. A cap that cannot be filled
is normal: take fewer. Serial is this round's size, not the rest of the run.

**Re-batch automatically after every completed round** (parallel integrate+cleanup or
a serial slice). Re-read `state.md`, pending slices, and the tree, then run
`parallel select` again: finished work leaves the pending set, and satisfied
dependencies unlock new slices. Under `.devrites/AFK` and `$rite-autocomplete` the
repeat is the same run — HITL stops after each batch. `--parallel 5` with one eligible
slice runs that one; when two to five then become eligible, the next round takes that
many. **Failing case:** the run stays one-by-one after a serial round even though more
slices are now eligible under the same cap.

## Path-disjoint

Exact project-relative source/test paths only. Normalize `\`→`/`. Reject `..`,
duplicates, absolutes, `.devrites/**`. Empty pairwise intersection required.

**SSOT:** `devrites-engine check path-disjoint [--root <dir>] [<json-file>|-]`
(N≥2 only; pass `--root`). Input is `{"slices":[{"id":"SLICE-002","paths":["src/policy.rs","src/policy/mod.rs"]}]}`
(or a top-level array) built from each candidate's `Files likely touched`.
Exit `0` → fan-out; else force serial. Inspect-time
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

1. Orient/gate; parse the cap; compute `N_eff` and select the eligible set
   ([Dynamic selection and re-batching](#dynamic-selection-and-re-batching)) —
   candidates from `orient` `task_graph.slices` + built list, each candidate's
   `Files likely touched` via `observe slice`, then
   `parallel select --cap N` (pairwise disjointness is inside that verb).
2. Write lease; freeze `B=HEAD`; `parallel create` worktrees.
3. Dispatch ≤10 wrights in parallel (cwd=worktree; allowlist; prove `HEAD==B`).
4. Inspect each returned path list, cumulative diff from `B`, transfer identity,
   and required result fields against the immutable contract. Reject malformed,
   stale, or out-of-scope work. Apply
   [independent Build review](phase-contract.md#independent-build-review): first
   full inventory, then bounded owning-role rechecks after repairs. Gap or
   Critical/Important → `gap`; else approved proof → `green|red|gap`.
   Reviewer fan-out for a batch is itself batched under
   [`parallel-dispatch.md`](../../devrites-lib/reference/parallel-dispatch.md#scale-limits)
   (≈4 concurrent read-only reviewers, 8 only with disjoint paths), and `max_agents`
   headroom bounds the batch size — one batch never assumes unbounded reviewers.
5. Non-green siblings → **repair round** (below) in the same worktrees; lease
   stays `running` — no abort while repair budget remains. A green sibling →
   `parallel record-green`. No partial integrate. All green → integrate.
   A recoverable RED blocks integration but does not cancel independent safe running siblings.
   Expected test-first RED is not a terminal gate failure. Cancel only for a shared
   invalid foundation, safety/access failure, resource boundary, or explicit stop;
   preserve completed and in-progress recoverable work and classify each result.
6. `parallel integrate --apply-to-control` **only after steps 4–5 are green**:
   each `transfer_commit` descends from `B` and diffs path-exact vs
   `B`; apply each in plan order as its own local `WIP(<slug>):` commit
   (imperative summary, never a slice id). Staging branch, then FF. Conflict →
   reset to `B`, `integrate-failed`; classify under repair rounds (siblings stay
   in place). Never integrate because a wright returned.
7. Success: FF control (the branch that held `HEAD` when the batch started —
   often `main`/`master`, otherwise whatever the user was on) to the integrate
   tip: N local commits, never pushed; union `touched-files.md`; update
   state/evidence; AFK +1 per integrated sibling; optional `check candidate`.
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

Repair rejected work in place; never discard/rebuild while budget remains.

- Record each round as `build-parallel-<batch>-repair-<round>.json`: each
  non-green sibling's status, verbatim findings, transfer_commit/null.
- Use step 4's initial inventory or affected recheck as appropriate; never start
  a correction from an early reviewer result while required accounts are pending.
- Classify the complete inventory: **implementation defect** (review
  Critical/Important on the diff, red wright/proof) → one repair-all wright in
  the same worktree with the full inventory. **Durable-plan defect**
  (falsified contract assumption, AC unchanged) →
  [`spec-drift-guard.md`](spec-drift-guard.md): batch-sweep contract
  assumptions, one folded `$rite-plan repair` + one `$rite-vet` recheck
  inline, resume repair in the same worktrees under the amended contract;
  never emit `Fix:`. **Frozen-lease scope defect** (correction needs
  out-of-allowlist paths or re-slicing; lease allowlists are immutable) →
  **automatic re-batch, no human gate**: `parallel cleanup --force` salvages
  and records refs; `parallel create` starts from control `HEAD` under the
  corrected decomposition. Give each new wright its salvaged
  `devrites/parallel/<slug>/<old-batch>/<slice>` ref. If the inventory proves the accepted
  approach unsound and no corrected decomposition exists, that is exhaustion
  → blocked, preserve, STOP. **Reviewer stall/cancel or missing
  verdict** → re-dispatch the reviewer, not the wright. **Product/policy/
  irreversible** → stops for the human.
- Dispatch one fresh repair-all `devrites-slice-wright` (cwd=worktree; step 3
  rules) with the original contract + complete inventory +
  "preserve what works; fix every finding; **then attack your own correction** —
  probe the changed hunks for defects the fix itself introduced (adjacent cases of
  the same rule, new observation/ordering windows, alias or boundary escapes) and
  fold them in before returning; return a new `transfer_commit`". A correction whose
  own diff is unexamined is not ready: a green writer gate plus an unexamined fix is
  how the same defect family reappears one case deeper each round. Never reset
  to `B` — that destroys the work under repair. The commit must still descend
  from `B` and diff path-exact vs `B`.
- Budget and exhaustion follow only
  [the canonical retry contract](../../devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve).
  Record one failed invariant, mechanism, reproduction, correction, and decisive
  recheck per fingerprint; never use an umbrella like "covering" or "review is red again".
- A distinct post-repair Critical/Important gets a new fingerprint: refresh the
  inventory and affected owning-role rechecks. It proves neither plan defect nor
  exhaustion. Use plan path only if inventory proves the approach unsound.
- Stop under the canonical retry/authority/resource conditions, or when evidence
  proves the approach unsound and no in-scope corrected decomposition exists.
- Exhaustion is not abort: record one technical blocker (`Status: blocked`;
  `Next step: none — technical recovery exhausted; requires new evidence or changed failure conditions`), preserve
  lease/worktrees/branches, STOP — same fingerprint stays blocked on
  reinvoke; new evidence resumes repair in place. Human-owned decisions use
  the drift-guard question and wait.

## Engine verbs

```text
devrites-engine parallel select --cap <1-10> [--root <dir>] [<json-file>|-]
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

`integrate` without `--apply-to-control` stages the per-sibling commits, leaves
the lease `running` and control at `B`; cleanup still refuses — the commits
cannot be lost. `record-green` and `integrate` also accept an `integrate-failed`
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

Claude: N concurrent spawn_agent wrights (`acceptEdits`, cwd=worktree). Codex: require
host-explicit concurrent worktree writers + native reconcile; else force serial.
Never two writers in one worktree; never root-emulated concurrency.

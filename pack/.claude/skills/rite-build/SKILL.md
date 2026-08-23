---
name: rite-build
description: Build the next approved vertical slice with evidence. Default one-slice-then-stop (HITL); explicit AFK may chain bounded green slices serially. Opt-in `--parallel N` (2≤N≤3) fans out path-disjoint slices in git worktrees when eligible.
argument-hint: "[--parallel N] [slice number or name]"
user-invocable: true
---

# /rite-build: one verified slice (default)

Build and prove **one** slice, then stop. HITL never auto-starts the next slice;
a later user invocation picks up the next pending slice. Explicit `.devrites/AFK`
alone lets the controlling root chain pending slices **serially** under green
proof, caps, and pause rules. Every wright returns after one slice. Read the
active workspace first; without one, route `/rite-spec <feature>`.

**Opt-in parallel:** `/rite-build --parallel N` is the only parallel entry.
Omitted flag or `N=1` ≡ today's one-slice path (no fan-out). Integer **2≤N≤3**
may batch up to N **path-disjoint** pending slices in distinct git worktrees,
then serially integrate on all-green. Non-integer or `N>3` → **hard refuse**
(no silent clamp). Overlapping slice paths → force serial with reason. Details:
[`reference/parallel-batch.md`](reference/parallel-batch.md) (when present).

Root owns gates/bookkeeping. Fresh
[`devrites-slice-wright`](../../agents/devrites-slice-wright.md) writes product
source/tests. Workflow Artifact paths use
[`workflow-artifacts.md`](../devrites-lib/reference/standards/workflow-artifacts.md).
Execute [`reference/phase-contract.md`](reference/phase-contract.md); dispatch uses
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Required rules

Read `.claude/skills/devrites-lib/reference/standards/core.md` first. Load only
rules triggered by the slice:

- coding style, error handling, testing, [`reference/tdd.md`](reference/tdd.md),
  patterns, and definition of done;
- binding `.devrites/principles.md` invariants when present;
- security for input/auth/data/integrations;
- repository topology for multiple roots/languages or generated/vendor surfaces;
- data integrity for durable state, migration, concurrency, tenancy, or retention;
- integration reliability for API/webhook/queue/job/cache/service boundaries.

The wright also applies the canonical anti-slop list. Root verifies its return;
it never patches source itself.

## Invariants

- Default: one slice per invocation; writers are serial on the control tree.
  Parallel writers are allowed **only** via `--parallel N` (2≤N≤3) when slices
  are path-disjoint, the host supports concurrent worktree wrights, and batch
  rules in [`reference/parallel-batch.md`](reference/parallel-batch.md) apply;
  same-worktree multi-writer and root-emulated concurrency remain forbidden.
  Use the native-worktree pilot for single-slice isolation only when
  `wright-dispatch.md`'s clean preflight and reconciliation both hold.
- Exact feature scope only. Record adjacent issues and Things I didn't touch;
  reject any returned diff outside the task's explicit source/test paths.
- Never rerun an unchanged check. Re-prove after edits.
- Unplanned dependency, design system, objective technical gap, or repair stays
  inside Vet/Spec Drift Guard. Ask only for licensing/cost/security/product or
  explicit architecture-policy decisions.
- Root never edits product source/tests. It writes canonical `.devrites/`
  bookkeeping and applies the Workflow Artifact route only. The supported host's
  wright is sole product writer. Put exact project-relative paths directly in the
  task; compare returned file list and `git diff --name-only`; extras hard-stop.
- Project principles are binding. An unavoidable conflict or irreversible risk
  needs a human-approved scoped exception or stop, never silent balancing.
- Evidence beats confidence. Never weaken a failing test, skip TDD, widen a
  writer, or self-approve a wright return. Route drift through
  [`reference/spec-drift-guard.md`](reference/spec-drift-guard.md); checkpoint
  mode follows [`reference/checkpoint.md`](reference/checkpoint.md).

## Workflow Artifact branch

<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"Vet-ready admitted bytes require root authorship outside product wright","action":"ROOT_TRANSACTION; root writes only admitted .devrites/** targets","return":"saved Build slice cursor; wright product allowlist unchanged"} -->
## `--parallel N` (opt-in)

| Input | Behavior |
|-------|----------|
| Flag omitted | One-slice-then-stop (HITL/AFK serial path unchanged) |
| `--parallel 1` | Same as omitted — one slice, no fan-out |
| `--parallel 2` or `--parallel 3` | Parallel batch mode when eligibility passes |
| Non-integer or `N>3` | Hard refuse with message; do not clamp or guess |

Parallel mode selects up to N pending slices whose exact source/test path sets
are pairwise path-disjoint, fans out one wright per worktree, proves each slice
fail-on-red in its cwd, then **serially integrates** all-green siblings onto
the control line. One red/gap sibling → abort the whole batch (no partial
integrate). Control tree owns `.devrites/**` bookkeeping; wrights never write
there. AFK charges once per successfully integrated green slice; abort charges
zero. While a parallel lease is `running`, refuse another `/rite-build`.
## Execute and reply


Run every step in `reference/phase-contract.md`: readiness, one target, dispatch
or canonical transaction, return inspection, independent doubt/test analysis,
approved fail-on-red proof, record, AFK accounting, and stop. Use
[`reference/output.md`](reference/output.md) plus the shared
[`reply contract`](../devrites-lib/reference/reply-contract.md). HITL never starts
the next slice automatically; AFK chains only within its durable remaining
budget; Prove starts only after all slices are built.

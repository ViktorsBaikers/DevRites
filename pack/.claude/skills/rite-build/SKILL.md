---
name: rite-build
description: Build the next approved vertical slice with evidence. HITL one-slice default; AFK/autocomplete chain serially or in eligible path-disjoint batches (dynamic size, cap 10); `--parallel N` caps a batch.
argument-hint: "[--parallel N] [slice number or name]"
user-invocable: true
---

# /rite-build: one verified slice

HITL stops; a later user invocation starts the next slice.
Explicit `.devrites/AFK` alone lets the controlling root chain under green proof,
caps and pause rules. Every wright returns after one slice.
**Opt-in:** `/rite-build --parallel N` (2≤N≤10; N=1≡serial) follows
[`reference/parallel-batch.md`](reference/parallel-batch.md)
([`one-slice-cycle.md`](reference/one-slice-cycle.md)).

Root owns gates/bookkeeping; fresh
[`devrites-slice-wright`](../../agents/devrites-slice-wright.md) writes source/tests.
Workflow Artifacts use
[`workflow-artifacts.md`](../devrites-lib/reference/standards/workflow-artifacts.md).
Execute [`reference/phase-contract.md`](reference/phase-contract.md)
([`one-slice-cycle.md`](reference/one-slice-cycle.md),
[`afk-discipline.md`](reference/afk-discipline.md)); dispatch uses
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Required rules

Read `.claude/skills/devrites-lib/reference/standards/core.md` first. Load only triggered rules:
coding/error/testing/[`tdd.md`](reference/tdd.md)/patterns/DoD; binding
`.devrites/principles.md`; security; topology; data integrity; integration reliability.
Wright applies anti-slop; root verifies returns and never patches source.

## Invariants

- Default: one slice; writers serial on control. Parallel only via `--parallel N`
  under [`reference/parallel-batch.md`](reference/parallel-batch.md). Same-worktree
  multi-writer / root-emulated concurrency forbidden. Native-worktree pilot =
  single-slice isolation when `wright-dispatch.md` preflight + reconcile hold.
- Exact feature scope only; reject out-of-allowlist diffs; record adjacent issues.
- Re-prove affected behavior after edits; reuse observations only under
  [evidence validity](../devrites-lib/reference/candidate-integrity.md#evidence-validity).
- Unplanned dependency/design-system/gap/repair → Vet/Spec Drift Guard batch
  sweep: every contract-assumption violation recorded before one folded
  repair+vet. Ask only for licensing/cost/security/product or explicit
  architecture-policy decisions.
- Root never edits product source/tests (`.devrites/` + Workflow Artifact only).
  Wright is sole product writer; extras in returned paths/`git diff --name-only` hard-stop.
- Principles bind; irreversible conflict needs human exception or stop.
- Evidence beats confidence. Never weaken tests, skip TDD, widen writers, or
  self-approve. Drift → [`spec-drift-guard.md`](reference/spec-drift-guard.md);
  checkpoint → [`checkpoint.md`](reference/checkpoint.md).
- Async readiness waits during slice work follow
  [`debug-recovery.md`](../devrites-lib/reference/standards/debug-recovery.md)
  (bounded poll + last-signal artifact; no blind sleep as primary strategy).

## Workflow Artifact branch

<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"Vet-ready admitted bytes require root authorship outside product wright","action":"ROOT_TRANSACTION; root writes only admitted .devrites/** targets","return":"saved Build slice cursor; wright product allowlist unchanged"} -->
## `--parallel N` (opt-in)

Omitted/`1` ≡ serial; `2`–`10` is a **cap** — the root runs the largest eligible
path-disjoint set it can start now (`N_eff`), takes fewer when the cap cannot be
filled, and recomputes after every completed round (serial slice or parallel
integrate) until no pending slice remains
([`parallel-batch.md` § Dynamic selection](reference/parallel-batch.md#dynamic-selection-and-re-batching)).
Unattended runs (`/rite-autocomplete`, `.devrites/AFK` `max_parallel`) repeat batches
inside the same run; HITL stops after each batch. Non-integer/`N≤0`/`N>10` hard refuse.
All-green (independent review + proof) then integrate as one local `WIP(<slug>):`
commit per sibling onto the current control branch (never pushed). Do not
integrate a sibling because the wright returned. A red/gap sibling gets a bounded repair
round in its own worktree — never rebuilt from scratch while budget remains.
Post-writer inventory and affected rechecks follow `phase-contract.md` § Independent
Build review; fold accounts before one repair-all wright.
Plan-owned gaps still route through Spec Drift Guard
(batch sweep, one folded plan repair + one vet recheck inline); do not emit
a human `Fix`. Exhausted repair blocks and preserves everything;
`cleanup --force` salvages slice branches before removing worktrees —
human-gated to abandon a batch, orchestrator-emitted only as the salvage
step of an automatic frozen-lease re-batch.
AFK charges after integrate only. Running lease blocks another
`/rite-build`. Details: `parallel-batch.md`.

## Execute and reply

Run every step in `reference/phase-contract.md`: readiness, one target, dispatch
or canonical transaction, return inspection, independent code/doubt/test analysis,
approved fail-on-red proof, record, AFK accounting, and stop. Use
[`reference/output.md`](reference/output.md) plus the shared
[`reply contract`](../devrites-lib/reference/reply-contract.md). HITL never starts
the next slice automatically; AFK obeys remaining budget; Prove requires all slices built.

## Phase exit

**Complete when:** Independent Build review and fail-on-red proof are green,
`git diff --name-only` ⊆ allowlist, no open Critical/Important, `state.md` cursor
advances with recorded evidence, and the control-branch checkpoint has landed
per [`checkpoint.md`](reference/checkpoint.md).

**Failing case:** wright reports "done" but independent review or proof was
skipped, red, or followed by an uncommitted repair → slice incomplete; do not
advance cursor or commit on control.

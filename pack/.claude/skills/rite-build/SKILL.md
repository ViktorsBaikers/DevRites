---
name: rite-build
description: Build the next approved vertical slice with evidence. HITL one-slice default; AFK may chain serially; opt-in `--parallel N` (2≤N≤3) for path-disjoint worktrees.
argument-hint: "[--parallel N] [slice number or name]"
user-invocable: true
---

# /rite-build: one verified slice

Build and prove one slice. HITL stops; a later user invocation starts the next.
Explicit `.devrites/AFK` alone lets the controlling root chain pending slices
serially under green proof, caps, and pause rules. Every wright returns after it.
**Opt-in:** `/rite-build --parallel N` (2≤N≤3; N=1≡serial) follows
[`reference/parallel-batch.md`](reference/parallel-batch.md).

Root owns gates/bookkeeping. Fresh
[`devrites-slice-wright`](../../agents/devrites-slice-wright.md) writes product
source/tests. Workflow Artifact paths use
[`workflow-artifacts.md`](../devrites-lib/reference/standards/workflow-artifacts.md).
Execute [`reference/phase-contract.md`](reference/phase-contract.md); dispatch uses
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Required rules

Read `devrites-lib/reference/standards/core.md` first. Load only triggered rules:
coding/error/testing/[`tdd.md`](reference/tdd.md)/patterns/DoD; binding
`.devrites/principles.md`; security; topology; data integrity; integration reliability.
Wright applies anti-slop; root verifies returns and never patches source.

## Invariants

- Default: one slice; writers serial on control. Parallel only via `--parallel N`
  under [`reference/parallel-batch.md`](reference/parallel-batch.md). Same-worktree
  multi-writer / root-emulated concurrency forbidden. Native-worktree pilot =
  single-slice isolation when `wright-dispatch.md` preflight + reconcile hold.
- Exact feature scope only; reject out-of-allowlist diffs; record adjacent issues.
- Never rerun an unchanged check; re-prove after edits.
- Unplanned dependency/design-system/gap/repair → Vet/Spec Drift Guard. Ask only
  for licensing/cost/security/product or explicit architecture-policy decisions.
- Root never edits product source/tests (`.devrites/` + Workflow Artifact only).
  Wright is sole product writer; extras in returned paths/`git diff --name-only` hard-stop.
- Principles bind; irreversible conflict needs human exception or stop.
- Evidence beats confidence. Never weaken tests, skip TDD, widen writers, or
  self-approve. Drift → [`spec-drift-guard.md`](reference/spec-drift-guard.md);
  checkpoint → [`checkpoint.md`](reference/checkpoint.md).

## Workflow Artifact branch

<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"Vet-ready admitted bytes require root authorship outside product wright","action":"ROOT_TRANSACTION; root writes only admitted .devrites/** targets","return":"saved Build slice cursor; wright product allowlist unchanged"} -->
## `--parallel N` (opt-in)

Omitted/`1` ≡ serial; `2`/`3` → path-disjoint fan-out when eligible; else hard refuse.
All-green serial integrate; one red/gap aborts. AFK charges after integrate only.
Running lease blocks another `/rite-build`. Details: `parallel-batch.md`.
## Execute and reply

Run every step in `reference/phase-contract.md`: readiness, one target, dispatch
or canonical transaction, return inspection, independent doubt/test analysis,
approved fail-on-red proof, record, AFK accounting, and stop. Use
[`reference/output.md`](reference/output.md) plus the shared
[`reply contract`](../devrites-lib/reference/reply-contract.md). HITL never starts
the next slice automatically; AFK chains only within its durable remaining
budget; Prove starts only after all slices are built.

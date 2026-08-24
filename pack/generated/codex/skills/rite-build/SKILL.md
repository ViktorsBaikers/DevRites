---
name: rite-build
description: Build the next approved vertical slice with evidence. HITL stops after one; explicit AFK may chain bounded green slices.
argument-hint: "[slice number or name]"
user-invocable: true
---

# $rite-build: one verified slice

Build and prove one slice. HITL stops; a later user invocation starts the next.
Explicit `.devrites/AFK` alone lets the controlling root chain pending slices
under green proof, caps, and pause rules. Every wright returns after it.
`--parallel` uses [`reference/parallel-batch.md`](reference/parallel-batch.md).

Root owns gates/bookkeeping. Fresh
[`devrites-slice-wright`](.codex/agents/devrites-slice-wright.toml) writes product
source/tests. Workflow Artifact paths use
[`workflow-artifacts.md`](../devrites-lib/reference/standards/workflow-artifacts.md).
Execute [`reference/phase-contract.md`](reference/phase-contract.md); dispatch uses
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Required rules

Read `.agents/skills/devrites-lib/reference/standards/core.md` first. Load only
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

- One slice per wright dispatch; writers are serial. Use the native-worktree pilot
  only when `wright-dispatch.md`'s clean preflight and reconciliation both hold,
  otherwise serial same-worktree writing.
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
## Execute and reply

Run every step in `reference/phase-contract.md`: readiness, one target, dispatch
or canonical transaction, return inspection, independent doubt/test analysis,
approved fail-on-red proof, record, AFK accounting, and stop. Use
[`reference/output.md`](reference/output.md) plus the shared
[`reply contract`](../devrites-lib/reference/reply-contract.md). HITL never starts
the next slice automatically; AFK chains only within its durable remaining
budget; Prove starts only after all slices are built.

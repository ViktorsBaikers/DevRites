# Slice-wright dispatch

`$rite-build` delegates one slice or correction to `devrites-slice-wright`.
The root owns scope, questions, `.devrites/**`, reconciliation, and routing.
Production builds and browser/E2E runs are root-owned gates only when their exact
command, cwd, prerequisites, and artifact boundary are already frozen in
`test-plan.md` and the unchanged packet.
The wright only writes source/tests, runs writer-safe proof, and never invokes agents.

The universal packet, result, budget, await, retry, and host fallback contract is
[`standards/agents.md`](../../devrites-lib/reference/standards/agents.md). This file adds
only the source-write lifecycle.

## Prepare and dispatch

1. Derive the smallest exact file allowlist from the selected slice, current code
   placement, and accepted correction set. Entries are normalized project-relative file
   paths: no directories, globs, traversal, symlink escapes, duplicates, or
   `.devrites/**`.
2. Write that list, one path per line, to
   `.devrites/work/<slug>/.wright-allowlist` (or the active compatibility workspace).
   The root owns this file; the wright's returned `Files changed` is evidence, never
   authorization. If a necessary unlisted path appears, the wright stops and returns it.
3. Create `agent-packet/v1` for `devrites-slice-wright`. Put the identical paths in
   `scope.allowed_repo_writes`; include the slice goal, verbatim acceptance, boundary,
   exact context paths, test-plan targets, and applicable standards. Do not include the
   root's implementation reasoning.
4. Immediately before a serial dispatch:

   ```bash
   devrites-engine stuck log "$(cat .devrites/ACTIVE 2>/dev/null)" dispatch "<slice id>"
   devrites-engine reconcile snapshot
   ```

   A Forge slice runs `forge plan` first, then takes this same snapshot; see
   [`forge.md`](forge.md). The first snapshot captures the original dirty-tree baseline, private Git objects,
   allowlist, and canonical-state fingerprint. A later snapshot after a clean check
   refreshes only the dispatch boundary; it retains the original slice baseline.
5. Fresh-context dispatch the wright through the capability ladder and await its typed
   result. Never run two possible writers in one tree.

## Forge candidate binding

[`forge.md`](forge.md) owns the state machine. This file owns the candidate packet:
use the same project-relative allowlist and slice contract, add only the manifest-recorded
candidate strategy, and set all five bindings before the first tool call:

```text
DEVRITES_FORGE_RUN_ID
DEVRITES_FORGE_CANDIDATE
DEVRITES_FORGE_WORKER_ID
DEVRITES_FORGE_WORKER_PID
DEVRITES_FORGE_PROCESS_START
```

The candidate cwd must equal its manifest worktree, and the environment is all-or-none.
Only a real adapter declared as `manifest-env-v1` may supply it. The live PID,
engine `forge process-token` value, and worker ID must match the prior `forge record ... started`
transition. A partial or mismatched binding denies the writer; return to the orchestrator
instead of widening scope. The candidate writes only source/tests in its own tree and
returns the normal typed artifact.

## Validate the return before accepting it

Validate the `agent-result/v1` identity, baseline, budget, payload type, side effects,
and exact changed-file set. Before the root writes any canonical record, run:

```bash
devrites-engine reconcile check; echo "reconcile rc=$?"
devrites-engine test-integrity; echo "test-integrity rc=$?"
```

- Reconcile `5`: reject; run standalone `devrites-engine reconcile abort` to restore
  pre-snapshot source/user work and close with a receipt. Never widen scope.
- Test integrity `3`: a test was deleted, muted, focused, or de-asserted. Treat it as a
  Critical protocol failure and correct it through the wright.
- A setup/corrupt-baseline error blocks acceptance; never fall back silently to `HEAD`.

The clean reconcile check records that source has not changed since inspection. The
baseline and private object database stay open until every later gate succeeds.

## Decisions, failures, and bounded recovery

After the immediate check, independently doubt every `Decisions stood` item. Human-owned
product, policy, public-contract, irreversible-risk, or human-only access decisions use
their real gate. Objective red tests, browser/runtime failures, missing technical
coverage, package-scanner defects, and review corrections stay agent-owned.

Classify each causal fingerprint through
[`cleanup-and-classify.md`](../../devrites-debug-recovery/reference/cleanup-and-classify.md).
The engine route is canonical; technical defects stay agent-owned.
Each objective root cause then shares one durable three-attempt budget:

```bash
devrites-engine recovery route <class>
devrites-engine recovery record --class <class> "<root cause>" "<exact failure>" <slug>
devrites-engine recovery check "<root cause>" <slug>
```

Before every retry:

1. update `.wright-allowlist` only for an accepted path still inside the settled slice;
2. run `devrites-engine reconcile snapshot`.

That refresh requires the prior clean check, re-fingerprints root-owned state, and keeps
the original source baseline. Re-dispatch with the exact output, attempt count, and dead
ends. Then repeat reconcile and `test-integrity` from zero. Clear recovery only after
green with `recovery clear --class <class> "<root cause>" <slug>`. An exhausted objective
failure becomes a technical blocker with its reproduction, not a `rite-resolve` question.

## Close and record

Run the root gates and final reconciliation from `phase-contract.md` step 5. Then:

```bash
devrites-engine reconcile close
```

Only then persist `state.md`, `evidence.md`, `touched-files.md`, decision/doubt records,
and browser evidence. Keep the retained window across a genuine human wait that will
resume the same slice. Close it before abandoning the slice for a scope/plan transition.

## Host dispatch

Use the named wright. Hidden V2 `agent_type` is not V1/unavailable: send
`agent_type=devrites-slice-wright` anyway. Use guarded generic only after explicit
V1; if V2 rejects the call, stop before generic/default. The rollout proves role,
wait, and result. Root hooks keep the pending canonical boundary current until spawn, then freeze it apart from the
retained source baseline: pre-start root recovery records are excluded, while
post-start canonical changes fail.
Any generic worker still needs the exact allowlist or an isolated checkout;
reconciliation alone is insufficient. If no safe fresh-agent rung is available, stop
for HITL. On Codex, `reconcile check` and `reconcile close` require the completed
wright receipt bound to the current reconcile snapshot.
The root never performs wright work.

---
name: devrites-lib
description: Internal shared DevRites helper library. Documents cross-cutting engine commands and references; not a user workflow. Do not invoke directly.
user-invocable: false
disable-model-invocation: true
---

# devrites-lib — internal shared helpers (not a command)

This is **not** a skill you run. It is DevRites' manifest for shared references
and control-plane operations. Skills call `devrites-engine <command>` from any
workspace; no pack script path is required.

## Operations

These are selected `devrites-engine` contracts; `devrites-engine help` is exhaustive.

**Read-only — orient / gate (never mutate the workspace):**

- `devrites-engine preamble` — orientation digest for the active `.devrites/` feature:
  prints `state.md`, the artifacts present, the run mode (HITL/AFK), and the
  open-question tally by gate. Run first (step 0) by every workspace-operating
  `rite-*` skill so the model orients deterministically instead of re-deriving
  state from raw Markdown.
- `devrites-engine progress` — progress footer; the mirror of `devrites-engine preamble` (which runs
  first). Run **last** (output step) by every lifecycle `rite-*` skill to render — from
  `state.md`, with zero model drift — the `── rite-<phase> ──` header rule, the **slice
  meter** (`Slice 3/5  ██████░░░░  <last-built> ✓`, or `Slices 5/5  ██████████  ✅ ALL
  BUILT` at completion), and the **flow ribbon** (`spec ✓ define ✓ build ◉ … ship ○`).
  The meter answers "how many slices left"; the `✅ ALL BUILT` marker answers "is the
  build done". The skill prints its own what-was-done / next-step / hygiene lines beneath
  it. Read-only; silent (exit 0) when there is no active workspace. Not for the workspace-less
  utilities (`/rite-prototype`, `/rite-zoom-out`, `/rite-pressure-test`, `/rite-handoff`,
  the `/rite` menu) — they have no phase/slice state to render.
- `reference/reply-contract.md` — the shared user-facing completion reply contract. It
  standardizes the compact chat lines printed below `devrites-engine progress` for success,
  awaiting-human, stopped/blocked, GO, NO-GO, and shipped states. The chat reply is a
  status summary; durable detail stays in the workspace artifacts.
- `reference/model-tiers.md` — the dispatch-by-task-shape contract (extraction / generation /
  ceiling). A skill names a **tier** by the shape of the work and never hardcodes a model name;
  reviewers are ceiling on purpose. Carries the degradation rule for harnesses that cannot pick
  models per agent. Loaded on demand by any skill that dispatches subagents.
- `devrites-engine build-readiness` — build-readiness gate. Exits non-zero on `/rite-build`'s
  step-0 stop conditions so they hold by exit code, not by prose the model must
  remember: `2` no `Plan approved` (→ `/rite-define`), `3` `awaiting_human`
  (→ `/rite-resolve`), `4` `blocked` (→ `/rite-plan`), `5` no workspace, `0` ready.
- `devrites-engine evidence-fresh` — evidence-freshness gate for `/rite-seal`. Exits `3`
  when any file in `touched-files.md` is newer than `evidence.md` /
  `browser-evidence.md` (stale proof = NO-GO until re-proven), `0` when fresh.
- `devrites-engine check-acceptance` — executable acceptance gate. Compiles `spec.md`'s
  acceptance IDs and exits `1` unless every one is checked (proven) in `seal.md`;
  new workspaces use `AC-###`, while legacy archives may still carry old `[ACn]`
  ids. Used by `/rite-seal` and by the outcome grader.
- `devrites-engine spec-validate` — spec-grammar gate (the spec-side mirror of
  `devrites-engine check-acceptance`). Lints `spec.md`'s structured `### Requirement:` / `#### Scenario:`
  blocks (SHALL/MUST present, ≥1 scenario each, every scenario has WHEN + THEN, headers
  unique). Exits `1` on a grammar violation, `0` when valid **or** when the spec uses the flat
  `AC-###` flat-bullet form (no structured blocks — nothing to lint, never a failure). Used by
  `/rite-spec`'s readiness gate; see [`standards/spec-grammar.md`](reference/standards/spec-grammar.md).

**State mutators — write `state.md` / `questions.md` under one contract:**

- `devrites-engine tick-afk` — decrement the AFK slice budget; exits `3` at 0 (forced HITL stop).
- `devrites-engine resolve` — backs the `/rite-resolve` contract (answer / drop / batch).
- `devrites-engine close-out` — archive the workspace + clear `ACTIVE` on `/rite-ship`.

### Canonical footer snippet

Every lifecycle `rite-*` skill prints this as the **first lines of its output**, then its
own compact fact lines below per [`reply-contract.md`](reference/reply-contract.md):

```bash
devrites-engine progress
```

**Unified entrypoint (tool-agnostic):**

- `devrites-engine` is the shared CLI for agents, CI, and humans. The npm
  `devrites` shim acquires it, owns install/update/uninstall bootstrap, and
  proxies other commands.

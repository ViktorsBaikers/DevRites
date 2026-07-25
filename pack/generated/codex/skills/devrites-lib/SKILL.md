---
name: devrites-lib
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
disable-model-invocation: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- Inspect the current `spawn_agent` role list. When the named `devrites-<role>` is exposed, dispatch it with `fork_turns="none"`; full-history forks inherit the parent type. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If a named role is not exposed, use generic `explorer` for every read-only role with `fork_turns="none"`. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. Trusted `.codex/hooks.json` binds `agent_type=explorer` to the fail-closed reviewer read-only guard.
- For `devrites-slice-wright`, trusted `.codex/hooks.json` binds generic `worker` (`agent_type=worker`) to the active reconcile window and exact `.wright-allowlist`. Dispatch that worker with `fork_turns="none"`, tell it to read `.codex/agents/devrites-slice-wright.toml`, and execute the unchanged packet. Never create `.reconcile-inline` when this safe rung is available.
- A missing custom role is not evidence that spawning is unavailable. Only when the project hooks are unavailable or untrusted, no spawn primitive exists, or higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, create `.reconcile-inline` only for that path, and apply every fallback risk gate.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# devrites-lib: internal shared helpers (not a command)

Do **not** run this skill directly. It lists shared references and control-plane
operations. Skills call `devrites-engine <command>` from any workspace without a pack
script path.

## Operations

These are selected `devrites-engine` contracts. Run `devrites-engine help` for the full list.

**Read-only: orient / gate (never mutate the workspace):**

- `devrites-engine preamble`: orientation digest for the active `.devrites/` feature:
  prints `state.md`, the artifacts present, the run mode (HITL/AFK), and the
  open-question tally by gate. Run first (step 0) by every workspace-operating
  `rite-*` skill so the model orients deterministically instead of re-deriving
  state from raw Markdown.
- `devrites-engine progress`: progress footer and counterpart to the initial
  `devrites-engine preamble`. Run it **last** in every lifecycle `rite-*` skill to render from
  `state.md`, with zero model drift) the `── rite-<phase> ──` header rule, the **slice
  meter** (`Slice 3/5  ██████░░░░  <last-built> ✓`, or `Slices 5/5  ██████████  ✅ ALL
  BUILT` at completion), and the **flow ribbon** (`spec ✓ define ✓ build ◉ … ship ○`).
  The meter answers "how many slices left"; the `✅ ALL BUILT` marker answers "is the
  build done". The skill prints its own result, next step, and hygiene lines beneath
  it. Read-only; silent (exit 0) when there is no active workspace. Not for the workspace-less
  utilities (`$rite-prototype`, `$rite-zoom-out`, `$rite-pressure-test`, `$rite-handoff`,
  the `$rite` menu). They have no phase/slice state to render.
- `reference/reply-contract.md`: the shared user-facing completion reply contract. It
  standardizes the compact chat lines printed below `devrites-engine progress` for success,
  awaiting-human, stopped/blocked, GO, NO-GO, and shipped states. The chat reply is a
  status summary; durable detail stays in the workspace artifacts.
- `reference/model-tiers.md`: the dispatch-by-task-shape contract (extraction / generation /
  ceiling). A skill names a **tier** by the shape of the work and never hardcodes a model name;
  reviewers are ceiling on purpose. Carries the degradation rule for harnesses that cannot pick
  models per agent. Loaded on demand by any skill that dispatches subagents.
- `devrites-engine build-readiness`: build-readiness gate. Exits non-zero on `$rite-build`'s
  step-0 stop conditions so they hold by exit code, not by prose the model must
  remember: `2` no `Plan approved` (→ `$rite-define`), `3` `awaiting_human`
  (→ `$rite-resolve`), `4` `blocked` (→ `$rite-plan`), `5` no workspace
  (→ `$rite-spec`), `6` decision coverage missing/not CLEAR (→ `$rite-clarify`),
  `7` implementation readiness missing/not READY (→ `$rite-vet`), `8` older or unknown
  semantic readiness contract (→ `$rite-upgrade`), `0` ready.
- `devrites-engine evidence-fresh`: evidence-freshness gate for `$rite-seal`. Exits `3`
  when any file in `touched-files.md` is newer than `evidence.md` /
  `browser-evidence.md` (stale proof = NO-GO until re-proven), `0` when fresh.
- `devrites-engine check-acceptance`: executable acceptance gate. Compiles `spec.md`'s
  acceptance IDs and exits `1` unless every one is checked (proven) in `seal.md`;
  new workspaces use `AC-###`, while legacy archives may still carry old `[ACn]`
  ids. Used by `$rite-seal` and by the outcome grader.
- `devrites-engine spec-validate`: spec-grammar gate (the spec-side mirror of
  `devrites-engine check-acceptance`). Lints `spec.md`'s structured `### Requirement:` / `#### Scenario:`
  blocks (SHALL/MUST present, ≥1 scenario each, every scenario has WHEN + THEN, headers
  unique). Exits `1` on a grammar violation, `0` when valid **or** when the spec uses the flat
  `AC-###` flat-bullet form (no structured blocks: nothing to lint, never a failure). Used by
  `$rite-spec`'s readiness gate; see [`standards/spec-grammar.md`](reference/standards/spec-grammar.md).

**State mutators: write `state.md` / `questions.md` under one contract:**

- `devrites-engine tick-afk`: decrement the AFK slice budget; exits `3` at 0 (forced HITL stop).
- `devrites-engine resolve`: backs the `$rite-resolve` contract (answer / drop / batch).
- `devrites-engine close-out`: archive the workspace + clear `ACTIVE` on `$rite-ship`.

### Canonical footer

Every lifecycle `rite-*` skill prints this as the **first lines of its output**, then its
own compact fact lines below per [`reply-contract.md`](reference/reply-contract.md):

```bash
devrites-engine progress
```

**Unified entrypoint (tool-agnostic):**

- `devrites-engine` is the shared CLI for agents, CI, and humans. The npm
  `devrites` shim acquires it, owns install/update/uninstall bootstrap, and
  proxies other commands.

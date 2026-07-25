---
name: devrites-lib
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
disable-model-invocation: true
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. Codex loads that role TOML's `developer_instructions` natively. Because V2 collaboration lifecycle calls bypass hooks, DevRites verifies the current durable parent/child rollout for the exact role, wait, completion, and non-empty delivered result.
- On MultiAgent V1, when the named role is not exposed, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`; do not substitute `worker` for an exposed V2 named role.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If any required named or generic agent dispatch is unavailable or rejected, stop for HITL. Never execute a DevRites specialist role in the root context.
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

---
name: rite-pressure-test
description: Pressure-test a rough/vague idea: ideate, explore 3-5 approaches, radically different shapes, diverge then converge on direction before spec. Not for writing spec.
argument-hint: "[rough idea or plan to stress-test]"
user-invocable: true
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


# $rite-pressure-test: diverge then converge

Use when the *idea* (not just the requirements) is rough. Generate options, then commit
to one, so `$rite-spec` has a real direction to specify.

A thinking stance, not a build phase: capturing thinking is not implementing. Write
workspace/ticket artifacts freely; source edits wait for `$rite-build`, via `$rite-spec`.

Read `.agents/skills/devrites-lib/reference/standards/core.md` first: its operating rules (no silent assumptions, prefer
existing conventions) shape the divergence. The other rule files load on demand.

## Diverge (widen)
- Load the `rejected-direction` entries from `.devrites/learnings.md` first. A recorded
  rejection re-enters the option set only with new evidence against its recorded *why*:
  name that evidence when you bring one back.
- Generate 3-5 genuinely different approaches to the underlying goal, not variations of
  one. Cover at least: the obvious approach, a simpler/smaller approach, and a
  different-shape approach (different data model, flow, or boundary).
- Generate from **named lenses** so each option exists for a reason, not to pad the count:
  *inversion* (do the opposite), *constraint-removal* (what if the hard limit vanished),
  *audience-shift* (build it for a different user), *10×* (what if it had to handle ten times
  the scale or scope), *expert-lens* (how a specialist in the domain would do it). Borrow the
  *structure* of an analogous product, not its surface. "Uber for X" copies the veneer, not
  the mechanism that made it work.
- For each: one-line description, what it optimizes for, rough cost, main risk.
- Stay concrete: name real entities, flows, and surfaces, not abstractions.

## Converge (commit)
- Weigh options against the goal, constraints, and existing codebase conventions.
- **Painkiller or vitamin?** Score each on whether it removes a real, felt pain (a painkiller
  users seek out) or is merely nice-to-have (a vitamin they forget). Prefer the painkiller: a
  vitamin dressed as a painkiller is the most common ideation trap.
- **Rank the differentiation**, strongest to weakest: a new capability > a 10× improvement > a
  new audience > a new context > better UX > cheaper. The higher an option sits, the more
  defensible the direction.
- Recommend one, with the reason and the key trade-off accepted.
- Note what would change the recommendation (the decision's hinge).

## Boundaries
- This is exploration, not specification. Output a **direction**, not a finished spec:
  `$rite-spec` writes the spec.
- Don't over-explore: 3-5 options, one pass of convergence. If the user already knows
  the direction, skip this and go straight to `$rite-spec`. If the effort is too foggy for
  one pass, start an investigation map at `.devrites/work/<slug>/investigation-map.md`
  with `Destination`, `Decisions so far`, `Not yet specified`, `Out of scope`, plus one
  frontier question per session.
- Ask the user to pick when two options are close and the choice changes the product.
- Name a **"Not doing" list**: the good options you deliberately cut. It's the highest-value
  output of convergence: it hands `$rite-spec` its scope boundary and stops the rejected ideas
  from creeping back in later. When a cut is durable (rejected for a reason that outlives this
  feature) offer to record it: `devrites-engine learnings add <slug> "<direction>: <why>"
  rejected-direction`.

## Output
Reply-contract exception: pre-workspace ideation utility. It skips `devrites-engine progress`,
but follows the compact labels and single-next-action rule from
[`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md).

```
Done: pressure test complete for <goal>.
Changed: workspace only
Evidence: options compared <n>; recommendation <option>; hinge <condition>
Open: <none | unresolved premise>
Next: $rite-spec <feature>
Record: not applicable
↻ Hygiene: /clear before starting the lifecycle
```

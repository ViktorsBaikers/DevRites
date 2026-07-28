---
name: rite-pressure-test
description: Pressure-test a rough/vague idea: ideate, explore 3-5 approaches, radically different shapes, diverge then converge on direction before spec. Not for writing spec.
argument-hint: "[rough idea or plan to stress-test]"
user-invocable: true
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
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

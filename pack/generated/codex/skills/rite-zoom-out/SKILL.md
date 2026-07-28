---
name: rite-zoom-out
description: User-invoked read-only structural map of unfamiliar code: modules, callers, callees, and relevant decisions.
argument-hint: "[symbol | file | area to map]"
user-invocable: true
disable-model-invocation: true
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


# $rite-zoom-out: step up one abstraction layer

When the agent (or the user) is staring at unfamiliar code without a working mental
model of how it fits the larger system. Stops the "open more files" reflex by returning
a single, structured map instead.

Read `.agents/skills/devrites-lib/reference/standards/core.md` first: chiefly its vocabulary / existing-conventions
disciplines, which keep the map in the project's own language. The other rule files load
on demand.

## What this skill returns

A **structural map**: terse. One pass should answer:

- **The area:** what this code is for, in one sentence, using the project's own
  vocabulary.
- **Modules in scope:** the related files / packages / slices, with a one-line
  purpose each.
- **Callers (in):** who calls into this area from outside. Keep to the highest-signal
  3-6; collapse the rest.
- **Calls (out):** what this area depends on downstream.
- **Decisions touching it:** ADRs (under `docs/adr/` if present) or notes in
  `.devrites/work/<slug>/decisions.md` that pre-decide something here.
- **Smallest sensible change-scope:** where a fix would naturally land, so the next
  step doesn't drift into a project-wide refactor.

## Prefer a code-intelligence index (if available)

If the project has them, start with `codebase-memory-mcp` (`get_architecture` / `search_graph`),
then cross-check with `codegraph` (`.codegraph/`) and `graphify` (`graphify-out/`). For
codegraph, `codegraph_context` + one `codegraph_explore` return the map in two calls: vastly
cheaper than a file-walk and more accurate for callers/callees. Fall back to standard methods
(LSP, then `Grep` + `Read`) when no index is available. See `.agents/skills/devrites-lib/reference/standards/tooling.md`.

## Vocabulary discipline

Use the **project's** domain language: `CONTEXT.md`, glossaries, the active feature's
`spec.md` / `decisions.md`. Don't invent fresh names for things the project already
names. If you notice a fuzzy or overloaded term while mapping, flag it as a FYI at the
end; don't try to fix it here.

## When NOT to use

- You already have a clear mental model: zooming out is just tax.
- The question is a literal text lookup (a string, a comment, an error message): use
  `Grep`.
- You need to design or change something. That's `$rite-spec` (new feature) or
  `$rite-define` (plan an approved spec).
- You want a project-wide architecture audit: use the project's normal architecture
  review process; this skill is a read-only feature-area map.

## Output shape
Reply-contract exception: read-only mapping utility. It skips `devrites-engine progress` when
there is no active workspace, but follows
[`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md).

```
Done: mapped <area> in the project's vocabulary.
Changed: workspace only
Evidence: modules <n>; callers <n>; callees <n>; decisions <n>
Open: <none | fuzzy term | suspected drift | open question>
Next: <single recommended command>
Record: <decision/ADR path | not applicable>
↻ Hygiene: /clear if this was only orientation; $rite-handoff if it informs active work
```

Print the path of any decisions/ADR files referenced so the user can open them.

---
name: rite-zoom-out
description: User-invoked read-only structural map of unfamiliar code: modules, callers, callees, and relevant decisions.
argument-hint: "[symbol | file | area to map]"
user-invocable: true
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

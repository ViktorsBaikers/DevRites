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
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-zoom-out — step up one abstraction layer

When the agent (or the user) is staring at unfamiliar code without a working mental
model of how it fits the larger system. Stops the "open more files" reflex by returning
a single, structured map instead.

Read `.agents/skills/devrites-lib/reference/standards/core.md` first — chiefly its vocabulary / existing-conventions
disciplines, which keep the map in the project's own language. The other rule files load
on demand.

## What this skill returns

A **structural map** — terse. One pass should answer:

- **The area** — what this code is for, in one sentence, using the project's own
  vocabulary.
- **Modules in scope** — the related files / packages / slices, with a one-line
  purpose each.
- **Callers (in)** — who calls into this area from outside. Keep to the highest-signal
  3–6; collapse the rest.
- **Calls (out)** — what this area depends on downstream.
- **Decisions touching it** — ADRs (under `docs/adr/` if present) or notes in
  `.devrites/work/<slug>/decisions.md` that pre-decide something here.
- **Smallest sensible change-scope** — where a fix would naturally land, so the next
  step doesn't drift into a project-wide refactor.

## Prefer a code-intelligence index (if available)

If the project has them — `codebase-memory-mcp` (`get_architecture` / `search_graph`) first,
cross-checked with `codegraph` (`.codegraph/`) and `graphify` (`graphify-out/`) — use them. For
codegraph, `codegraph_context` + one `codegraph_explore` return the map in two calls — vastly
cheaper than a file-walk and more accurate for callers/callees. Fall back to standard methods
(LSP, then `Grep` + `Read`) when no index is available. See `.agents/skills/devrites-lib/reference/standards/tooling.md`.

## Vocabulary discipline

Use the **project's** domain language — `CONTEXT.md`, glossaries, the active feature's
`spec.md` / `decisions.md`. Don't invent fresh names for things the project already
names. If you notice a fuzzy or overloaded term while mapping, flag it as a FYI at the
end; don't try to fix it here.

## When NOT to use

- You already have a clear mental model — zooming out is just tax.
- The question is a literal text lookup (a string, a comment, an error message) — use
  `Grep`.
- You need to design or change something — that's `$rite-spec` (new feature) or
  `$rite-define` (plan an approved spec).
- You want a project-wide architecture audit — use the project's normal architecture
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

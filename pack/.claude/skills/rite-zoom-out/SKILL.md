---
name: rite-zoom-out
description: Zoom out one abstraction layer in unfamiliar code — return a structural map (modules, callers, callees, ADR/decisions touching the area) in the project's vocabulary. Use when the user says "zoom out", "map this", "bigger picture", "I don't know this area", "what calls this". Not for implementing changes (read-only) or one-symbol lookups (use grep).
user-invocable: true
---

# /rite-zoom-out — step up one abstraction layer

When the agent (or the user) is staring at unfamiliar code without a working mental
model of how it fits the larger system. Stops the "open more files" reflex by returning
a single, structured map instead.

Read `.claude/rules/core.md` first — chiefly its vocabulary / existing-conventions
disciplines, which keep the map in the project's own language. The other rule files load
on demand.

## What this skill returns

A **map, not an essay**. One pass should answer:

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

## Prefer the code-intelligence index

If the project has `codegraph` (`.codegraph/`) or `graphify` (`graphify-out/`), use it.
`codegraph_context` + one `codegraph_explore` return the map in two calls — vastly
cheaper than a file-walk and more accurate for callers/callees. Fall back to
`Grep` + `Read` only when no index is available.

## Vocabulary discipline

Use the **project's** domain language — `CONTEXT.md`, glossaries, the active feature's
`spec.md` / `decisions.md`. Don't invent fresh names for things the project already
names. If you notice a fuzzy or overloaded term while mapping, flag it as a FYI at the
end; don't try to fix it here.

## When NOT to use

- You already have a clear mental model — zooming out is just tax.
- The question is a literal text lookup (a string, a comment, an error message) — use
  `Grep`.
- You need to design or change something — that's `/rite-spec` (new feature) or
  `/rite-define` (plan an approved spec).
- You want a project-wide architecture audit — that's `improve-codebase-architecture`,
  out of DevRites' feature-scoped remit.

## Output shape

```
Area: <one line, project's vocabulary>
Modules:
  - path/x.ts — <purpose>
  - path/y.ts — <purpose>
Callers (in):
  - <module> — <how it calls in>
Calls (out):
  - <module> — <what it calls for>
Decisions touching it:
  - ADR-NNNN / decisions.md entry — <one-line claim>
Smallest sensible change-scope: <where a fix would land>
FYI (optional): <fuzzy term / suspected drift / open question>
```

Print the path of any decisions/ADR files referenced so the user can open them.

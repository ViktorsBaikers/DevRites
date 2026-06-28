# Code + backend polish (Phase 1 + Phase 2)

Loaded from `/rite-polish` every run, regardless of UI scope. Two sub-phases:
code polish (Phase 1) and, when backend is touched, backend polish (Phase 2).

## Rules consulted (read on demand from `.claude/rules/`)

- `coding-style.md` — Phase 1 (simplify, dead code, naming, comments).
- `patterns.md` — Phase 1 simplification — avoid over-engineering.
- `error-handling.md` — Phase 2 backend (no silent catches, consistent errors).
- `performance.md` — Phase 2 backend (N+1s, query bounds).
- `documentation.md` — keep touched docs current; record polish-time decisions.

## Operating rules

- **Never cite clean automation** (lint/build pass) as proof of good design or
  simplicity.
- Feature scope only. Spec Drift Guard applies.
- Re-run targeted tests after each change — a simplification that breaks a
  test wasn't behavior-preserving.

## Phase 1 — Code polish *(always)*

Delegates the audit to `devrites-audit simplify`. Reduce complexity in the
feature's touched code while **preserving exact behavior**. Scope = active
feature only.

- **Measure first, target hotspots** — deep nesting, long branchy functions,
  high cyclomatic complexity, sprawling conditionals. Don't redistribute
  complexity, reduce it. Untargeted cleanup just moves decision points around.
- **Behavior-preserving techniques** (name the one used per change): guard
  clauses (flatten nested if/else, return early on unwanted cases), Extract
  Method (a coherent block into a named single-responsibility helper), simplify
  conditionals (switch/lookup over a long if-else; decompose a complex boolean
  into well-named parts), dedupe, inline single-use indirection, replace
  hand-rolled utils with the stdlib/existing helper, delete dead code this
  feature added.
- **Chesterton's Fence** — understand *why* something exists before removing it.
  If you can't explain a check, branch, or wrapper, you may not remove it —
  many "useless" lines guard a real edge case.
- **Behavior preservation** — observable behavior stays identical; tests stay
  green. If behavior would change, it's not simplification — it needs its own
  acceptance + proof (and maybe drift handling). Prefer transformations with
  obvious equivalence.
- **Don't over-reduce / proportionality** — inherent complexity is fine;
  readability is the goal, not a metric. Forcing the complexity number down by
  *hiding* branches elsewhere is worse. Don't spend disproportionate effort on
  small, stable, rarely-touched code; target central/often-read code.
- **Guardrails** — feature scope only, no project-wide refactor; don't delete
  suspected dead code **outside** this feature without asking; re-prove after
  simplifying (a simplification that breaks a test wasn't behavior-preserving);
  cleverness that's shorter but harder to read is not simpler.
- **Cleanup**: remove TODOs, `console.log`s, commented-out code, unused
  imports/vars; tighten naming and comments in code this feature touched.
- **Done when** — every anti-slop charter item in the touched code is cleared
  (the AI-tells do-not list — [anti-ai-slop.md](anti-ai-slop.md) Code section,
  `coding-style.md`) **and** the feature's targeted tests + build re-run green.
  An open charter item or a red check means Phase 1 isn't done.

## Phase 2 — Backend polish *(if BE touched)*

See [backend-polish.md](backend-polish.md). For server-side scope (handlers,
services, routes, models, migrations, queries, jobs, auth). Polish the server
side to ship-quality:

- **Error responses consistent** + correct status codes + custom error classes
  + fail closed on auth/permission/transaction errors. No blanket `catch`es.
- **Logging hygiene** — structured logs with context (request id, user, op);
  log key events; **never** log secrets / tokens / PII. No leftover
  `console.log` / debug prints.
- **Data & queries** — no N+1, no unbounded result sets, parameterized
  queries, right transaction boundaries, return only what the caller needs.
- **API contract** matches the spec (and any saved `references/`); idempotency
  where applicable; consistent pagination/sorting/filtering; validation at the
  boundary.
- **Performance** — measure-first; obvious wins applied; no quadratic loops on
  growing collections (`devrites-audit perf`).
- **Cleanup** — remove dead routes, unused endpoints + bad naming this feature
  added.
- **Code anti-slop** — kill over-defensive layered null/length checks, useless
  wrappers, generic AI naming, "robust" catches that hide bugs, anything
  outside the spec ([anti-ai-slop.md](anti-ai-slop.md) — Code section).

## Output → appends to `polish-report.md`

```
Phase 1 (code polish): findings → fixes (technique + why behavior preserved)
Phase 2 (backend polish): error/log/data/API/cleanup fixes | n/a (no backend)
```

# Code + backend polish (Phase 1 + Phase 2)

Loaded from `/rite-polish` every run, regardless of UI scope. Two sub-phases:
code polish (Phase 1) and, when backend is touched, backend polish (Phase 2).

## Rules consulted (read on demand from `pack/.claude/rules/`)

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

Delegates the audit to `devrites-audit simplify`; see
[code-simplification.md](code-simplification.md).

- **Measure first, target hotspots** — deep nesting, long branchy functions,
  sprawling conditionals. Don't redistribute complexity, reduce it.
- **Behavior-preserving techniques** (name the one used per change): guard
  clauses, Extract Method, simplify conditionals (switch/lookup), dedupe,
  replace hand-rolled utils with stdlib/existing helpers, delete dead code
  this feature added.
- **Chesterton's Fence** — explain why something exists before removing it.
- **Don't over-reduce** — inherent complexity is fine; readability is the goal.
- **Cleanup**: remove TODOs, `console.log`s, commented-out code, unused
  imports/vars; tighten naming and comments in code this feature touched.

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

---
name: rite-plan
description: Re-plan an active feature after reality changed: reslice, repair drift, reorder, split FE/BE, unblock, pivot, or revise artifacts. Use for replan/reslice/repair/unblock/pivot. Not first-pass decomposition.
argument-hint: "[mode: decompose|reslice|repair|reorder|split|unblock|course-correct|revise]"
user-invocable: true
---

# /rite-plan — (re)plan an active feature

Reshape the plan when reality and the plan disagree. **Read the active workspace
first.** If `.devrites/ACTIVE` is empty or its workspace is missing, stop and tell the
user to run `/rite-spec <feature>`. **Revise mode is artifact-only**: reconcile
`spec.md` / `architecture.md` / `plan.md` / `tasks.md` / `traceability.md` without
editing source code.

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
Read `.claude/skills/devrites-lib/reference/standards/core.md` first. Pull `development-workflow.md` via `Read` when
reshaping slice cadence or DoD criteria.

## Operating rules
- Spec is living, not sacred — but never plan around a known-wrong assumption silently.
- If a change alters product behavior, scope, architecture, data model, UX, security,
  or migration risk → **ask the user first** (use the Spec Drift Guard question format).
- Keep each slice small enough for one focused build → prove cycle.
- **Slice count is derived, never dictated** — reslice when a slice fails the sizing rule
  (multiple "and"s, can't build+prove in one cycle), not to hit a user-named tally. A
  requested count is a hint at most; slice logically and explain if it differs. See
  [`reference/slicing.md`](reference/slicing.md) ("How many slices?").
- **Size by complexity, order by dependency.** A slice carries a `Complexity: N/5` score (from
  `/rite-define`); a slice scoring **>3** is a reslice trigger unless its inline reason justifies
  the irreducible complexity. Honor each slice's `depends_on:` — the next *buildable* slice is the
  lowest pending one whose dependencies are all built (keeps one-slice-at-a-time correct, not parallel).

## Workflow
0. Read `.claude/skills/devrites-lib/reference/standards/core.md` (operating rules) before reshaping anything.
   Then run `devrites-engine preamble` for deterministic workspace orientation.
1. Read `spec.md`, `plan.md`, `tasks.md`, `state.md`, `drift.md`, and the current
   `git diff` (if a repo). Read `decisions.md` and `assumptions.md`. If a code-intelligence
   index is available — `codebase-memory-mcp` first, cross-checked with `codegraph`
   (`.codegraph/` / `codegraph_*` tools) + `graphify` (`graphify-out/`), else standard methods
   (LSP / `Read`/`Grep`/`Glob`); see `.claude/skills/devrites-lib/reference/standards/tooling.md` —
   prefer it for structural questions (what calls X, what would
   changing Y break) over reading whole files, to keep planning context lean. For an external
   dependency's current API surface, consult context7 if available.
2. **Pick the mode** (`$ARGUMENTS` or infer):
   - **decompose** — first/again break the feature into vertical slices.
   - **reslice** — a slice is too large; split into thinner end-to-end slices.
   - **repair** — a Spec Drift Guard event; fold the resolution into plan + tasks.
   - **reorder** — fix the dependency order.
   - **split** — separate backend/frontend contracts (see `devrites-api-interface`).
   - **unblock** — a verification failed; re-route around the blocker.
   - **course-correct** — a deliberate mid-build *pivot* (the user changed their mind), distinct
     from accidental drift: classify the change, assess its impact across the remaining slices,
     decide rollback vs forward-fix, and update `spec.md` + `plan.md` + `tasks.md` + `decisions.md`
     atomically. An acceptance/behavior change still goes through the user first. When the plan
     names an `MVP cut`, offer it as the retreat option: falling back to the cut is a pre-agreed
     scope, not a new negotiation.
   - **revise** — apply a requested planning-artifact revision and reconcile existing artifacts in
     any direction; propose the file edit set first, confirm each file before writing, and **never
     edit source code**. **Gate first — revise or new?** Same intent? >50% of existing scope
     survives? original *not* completable without this? Two "no"s → new work: recommend
     sealing/shipping the current workspace (MVP cut if named) then `/rite-spec` for the new
     intent, and stop. Revise preserves context; a new workspace provides clarity.
   See [replan-and-repair](reference/replan-and-repair.md) for each mode's steps.
3. Reason about dependencies — [dependency-graph](reference/dependency-graph.md).
4. Re-slice using vertical-slice rules — [slicing](reference/slicing.md) and
   [task-breakdown](reference/task-breakdown.md). Prefer thin, shippable, verifiable.
5. Update `plan.md`, `tasks.md`, `state.md`, and append rationale to `decisions.md`.
   If you stopped for drift, mark the `drift.md` entry resolved.
6. If product behavior/acceptance criteria change, confirm with the user before writing.
7. **Done when** — every slice is sized (builds + proves in one cycle; no slice scoring >3
   left unjustified), the dependency order is acyclic, every `drift.md` entry you stopped for
   is marked resolved, revised artifacts agree with each other, no source files changed in
   `revise` mode, and behavior-change-vs-not is confirmed (`no`, or asked + answered).
   If any check fails, loop back — don't hand off a half-reshaped plan.

> **Mid-flight discipline.** When tempted to change product behavior without asking, absorb drift silently, or skip the user — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: plan repaired for <slug> in <mode> mode.
Changed: plan.md, tasks.md, traceability.md, decisions.md, state.md
Evidence: not applicable; slice map now <n> slices and next slice is <name>
Open: <none | behavior question answered | Alternative: /rite-prove if all built slices need re-verification>
Next: <single next command: build, re-define, or prove depending on the revision>
Record: .devrites/work/<slug>/plan.md
↻ Hygiene: /clear if the repair was large; keep session for small reorder-only repairs
```

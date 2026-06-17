---
name: rite-define
description: Decompose an approved `spec.md` into `plan.md` + vertical task slices + `state.md`; every acceptance criterion maps to ≥1 slice. Use when the user says "plan this", "break this into slices", "task breakdown". Not for writing code or repairing an existing plan (use `/rite-plan`).
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-define — plan from the spec

Read the active feature's `spec.md` and turn it into a buildable plan: the approach, a
dependency-ordered set of **vertical slices**, and the state cursor. The spec is the WHAT/
WHY (from `/rite-spec`); this is the **HOW**. Splitting spec and plan keeps each focused so
nothing gets missed in one batch. **No code here.**

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. DevRites skills Read `.claude/rules/core.md`
as their first step; the other rule files load on demand. Pull these via `Read` when shaping
the plan:
- `development-workflow.md` — small batches, trunk-always-green, definition of done.
- `documentation.md` — record plan-time decisions and rationale.

## Operating rules
- **Requires a spec.** Read the active workspace first; if `.devrites/ACTIVE` is empty,
  the workspace has no `spec.md`, or its readiness gate hasn't passed → **STOP** and tell
  the user to run `/rite-spec <feature>` first. **DO NOT plan from a missing or
  unreadied spec.**
- Prefer existing conventions; ask before adding a dependency or a second design system.
- **Slice count is derived, never dictated.** The number of slices falls out of the work
  — one per independently-shippable increment, sized by `slicing.md`, every acceptance
  criterion mapped to ≥1 slice. A user-named count is a hint at most: slice logically and,
  if your honest count differs, present it and why. Never pad or compress to hit a figure.
  (`.devrites/AFK` `max_slices` is a separate AFK iteration budget, not the decomposition.)

## Workflow
0. **Read `.claude/rules/core.md`** — the always-on operating rules and anti-rationalizations.
1. **Read the spec** — `spec.md` (objective, requirements, acceptance, **placement**,
   design references, gaps/decisions), plus `references.md`, `decisions.md`,
   `assumptions.md`, **`strategy.md` if present** (the scope mode, deferred / out-of-scope
   register, and pre-mortem risks from `/rite-temper` — cut slices to mitigate the top risks
   and respect the IN/OUT line; map coverage against the **hardened** spec), and
   **`design-brief.md` if the feature touches UI** (the UX/UI contract `/rite-spec` shaped —
   its key states + interaction model drive how UI slices are cut). If a blocking
   `[NEEDS CLARIFICATION]` remains, stop → `/rite-spec`.
2. **Decide the approach + architecture** (the HOW the spec deliberately omitted): the
   strategy, key technical decisions + rationale, and the tech the slices will use. Use a
   code-intelligence index (`codegraph` / `graphify`) for structure/impact. Record in
   `plan.md` ([plan-template](reference/plan-template.md)) and `decisions.md`.
   **Deep-modules check** — while sketching the major modules, look for opportunities
   to extract a **deep module**: a small, stable interface that hides a meaningful chunk
   of behavior, and is therefore independently testable. A *shallow* module — interface
   nearly as complex as its implementation — earns nothing; either deepen it or delete
   it. Where a slice will produce a deep module, confirm with the user which deep
   modules they want unit-tested in isolation (this informs the slice's "Tests to
   write/run" field).
3. **Slice into vertical tasks** — each delivers one observable capability end-to-end and
   is verifiable on its own; the **count emerges from the work, not a target number**;
   first slice = thinnest useful end-to-end path; order by dependency (risk-first within a
   tier). Use `rite-plan/reference/slicing.md` and
   `rite-plan/reference/task-breakdown.md`. Mark per slice: **Frontend craft required**
   and **Browser proof required** (UI), and whether it's **fullstack** (FE+BE → contract
   first, see `devrites-frontend-craft/reference/fullstack.md`). **For UI slices, name which
   of `design-brief.md`'s key states + interaction the slice delivers** — so the brief's
   state coverage maps onto slices, not just acceptance criteria.
4. **Map coverage** — every spec acceptance criterion maps to ≥1 slice
   (`rite-spec/reference/acceptance-criteria.md`); no orphaned criteria, no slice without a
   criterion.
5. **Complexity & deviations gate** — justify anything off DevRites defaults (new dep,
   extra abstraction, second design system) in the plan; if you can't justify it, simplify.
6. **Write** `plan.md` + `tasks.md`; update `state.md` (phase: plan → next `/rite-build`).
7. **Readiness gate** (bottom of plan-template): every acceptance criterion covered by a
   slice, dependency order acyclic + risk-first, no unjustified deviation, rollback for
   every destructive/migration step. **Stop and confirm** before code. When the human
   confirms the plan, write `Plan approved: <iso>` to `state.md` (see
   [state-workspace](../rite-spec/reference/state-workspace.md)); `/rite-build` checks
   this exists before building.

## tasks.md slice format
```markdown
## Slice N: <name>
Goal:
Acceptance criteria:        # which spec FR/criteria this satisfies
Mode: AFK | HITL            # AFK = implementable + mergeable without human gating;
                            # HITL = needs a human decision mid-slice (design call,
                            # architectural choice, destructive migration sign-off).
                            # Prefer AFK; only mark HITL with the reason inline.
Gate: advisory | validating | blocking | escalating   # required when Mode=HITL; see reference/gates.md
SLA: 15m | 4h | 24h | none                            # required when Mode=HITL; matches the gate
Checkpoint: <one crisp question>                       # required when Mode=HITL; what the human must decide
Blocked by: Slice M, Slice K  # other slices that must complete first ("None" if free)
Files likely touched:       # from the spec's Placement & integration
Tests to write/run:
Browser proof required: yes/no
Frontend craft required: yes/no
Design brief states:        # UI slices only — which design-brief.md states/interaction this slice delivers (default/empty/error/…)
Fullstack (FE+BE): yes/no
Dependencies:               # external deps (libs, services), NOT slice ordering
Existing to reuse / extend:   # what already exists (components / utils / hooks) the slice should use
Rollback notes:
Evidence required:
```

> **Why Mode + Gate + Blocked by.** `Mode` lets `/rite-build` and `/rite-status` know
> whether a slice can run unattended or must surface a checkpoint; `Gate` + `SLA` tell
> AFK loops which gates they may auto-handle vs which always pause (see
> [`reference/gates.md`](reference/gates.md) for the four-gate taxonomy:
> advisory / validating / blocking / escalating). `Blocked by` makes the dependency
> graph explicit so re-planning (`/rite-plan reorder`) doesn't break acceptance-criteria
> coverage. Keep `Blocked by` cycle-free.

> **Mid-flight discipline.** When tempted to skip vertical slicing, coverage mapping, or dependency-order discipline — see [`anti-patterns`](reference/anti-patterns.md) (Common Rationalizations + Red Flags). Load it the moment you reach for the excuse.

## Output
```
Planned: <slug>
Approach: <one line>
Slices: N (slice 1: <name>)   Fullstack/UI slices: <which>
Coverage: <all acceptance criteria mapped? yes/no>
Strategy: honored (mode <m>; <n> pre-mortem risks → mitigation slices) | none (no strategy.md)
Next: confirm, then /rite-build   (or /rite-plan to reshape the slices)
↻ Hygiene: /clear after user confirms (plan.md + tasks.md + decisions.md + state.md captured). See rules/context-hygiene.md.
```

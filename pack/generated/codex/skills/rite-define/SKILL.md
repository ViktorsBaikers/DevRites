---
name: rite-define
description: Decompose an approved `spec.md` into `architecture.md`, `plan.md`, `tasks.md`, `traceability.md`, and `state.md`; every acceptance criterion maps to ≥1 slice. Use when the user says "plan this" or "break this into slices". Not for writing code or repairing an existing plan (use `$rite-plan`).
argument-hint: "[feature-slug]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-define — plan from the spec

Read the active feature's `spec.md` and turn it into a buildable workspace: feature
architecture, approach, a dependency-ordered set of **vertical slices**, traceability, and
the state cursor. The spec is the WHAT/WHY (from `$rite-spec`); this is the HOW. Splitting
spec, architecture, plan, tasks, and traceability keeps each file small and phase-owned.
**No code here.**

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` first. DevRites skills Read `.agents/skills/devrites-lib/reference/standards/core.md`
as their first step; the other rule files load on demand. Pull these via `Read` when shaping
the plan:
- `development-workflow.md` — small batches, trunk-always-green, definition of done.
- `principles.md` — the project invariants (`.devrites/principles.md`) the chosen approach must conform to.
- `documentation.md` — record plan-time decisions and rationale.
- `../workspace-artifact-schema.md` — artifact purposes, budgets, IDs, and read triggers.

## Operating rules
- **Requires a readied spec.** Read the active workspace first; if `.devrites/ACTIVE` is empty,
  the workspace has no `spec.md`, its readiness gate hasn't passed, or any spec-quality
  `checklists/<domain>.md` has an open CRITICAL → **STOP** and tell the user to run
  `$rite-spec <feature>` first. **DO NOT plan from a missing or unreadied spec.**
- Prefer existing conventions; ask before adding a dependency or a second design system.
- **Author section by section, not in one dump.** Write `architecture.md` / `plan.md` one section
  at a time and pause after each; a section resting on an open design choice or a shaky estimate can
  be deepened right there with a technique from
  [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md) (Tournament for two viable
  designs, Delphi for the estimate) before it hardens into slices.
- **Slice count is derived, never dictated.** The number of slices falls out of the work
  — one per independently-shippable increment, sized by `slicing.md`, every acceptance
  criterion mapped to ≥1 slice. A user-named count is a hint at most: slice logically and,
  if your honest count differs, present it and why. Never pad or compress to hit a figure.
  (`.devrites/AFK` `max_slices` is a separate AFK iteration budget, not the decomposition.)

## Workflow
0. **Read `.agents/skills/devrites-lib/reference/standards/core.md`** — the always-on operating rules and anti-rationalizations.
   Then **run the shared orientation preamble** — it confirms the active feature and which
   artifacts exist (it prints `state.md`, the artifacts present, the run mode, and the
   open-question tally):
   ```bash
   devrites-engine preamble
   devrites-engine spec-skeleton ".devrites/work/$(cat .devrites/ACTIVE 2>/dev/null)"
   ```
   If there is no active workspace, no `spec.md`, `spec-skeleton` blocks, or its readiness gate hasn't passed →
   **STOP** and tell the user to run `$rite-spec <feature>` first.
1. **Read the spec** — `spec.md` (objective, requirements, acceptance, **placement**,
   design references, gaps/decisions), plus `references.md`, `decisions.md`,
   `assumptions.md`, **`strategy.md` if present** (the scope mode, deferred / out-of-scope
   register, and pre-mortem risks from `$rite-temper` — cut slices to mitigate the top risks
   and respect the IN/OUT line; map coverage against the **hardened** spec), and
   **`design-brief.md` if the feature touches UI** (the UX/UI contract `$rite-spec` shaped —
   its key states + interaction model drive how UI slices are cut). If a blocking
   `[NEEDS CLARIFICATION]` remains, stop → `$rite-spec`.
2. **Decide the architecture + approach** (the HOW the spec deliberately omitted): write
   `architecture.md` for owning layer, boundaries, integration points, data/API/events,
   dependencies, risks, and affected areas; write only the build strategy in `plan.md`.
   Use a
   code-intelligence index if available — codebase-memory-mcp first, cross-checked with codegraph + graphify, else standard methods (LSP / Read/Grep/Glob)
   (see `.agents/skills/devrites-lib/reference/standards/tooling.md`) — for structure/impact; for the current API or behaviour of
   an external library/framework the architecture will rely on, consult context7 if available.
   Record significant options in `decisions.md` as `DEC-###` ADR entries.
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
4. **Map coverage** — every `AC-###` spec acceptance criterion maps to ≥1 `SLICE-###`
   (`rite-spec/reference/acceptance-criteria.md`); no orphaned criteria, no slice without a
   criterion.
4a. **Persist the traceability matrix** — write `traceability.md` (`AC/REQ ID → slice(s) →
   test/proof → evidence ID → touched files → status`), the living map `$rite-prove` and
   `$rite-seal` walk. Generate it with `devrites-engine coverage` when available, then save/rename
   the output as `traceability.md`, or write the table by hand from the same inputs if the
   script is absent:
   ```bash
   S="$(cat .devrites/ACTIVE 2>/dev/null)"
   devrites-engine coverage "$S" > ".devrites/work/$S/traceability.md"
   ```
5. **Complexity & deviations gate** — justify anything off DevRites defaults (new dep,
   extra abstraction, second design system) in the plan; if you can't justify it, simplify.
   **Principles conformance:** read `.devrites/principles.md` (if present) and confirm the
   approach honors every declared invariant. A plan that conflicts with one is not "a deviation
   to justify away" — either reshape the approach to conform, or, when the conflict is genuine and
   intended, route it through the Spec Drift Guard plus a recorded decision and a scoped principle
   exception a human approves. Never ready a plan that silently violates an invariant. (Re-scored
   as a blocking gate at `$rite-vet`; no file → none declared → nothing to check.)
6. **Write** `architecture.md`, `plan.md`, `tasks.md`, and `traceability.md`; update
   `state.md` (phase: plan → next `$rite-vet`).
7. **Readiness gate** (bottom of plan-template): every acceptance criterion covered by a
   slice, dependency order acyclic + risk-first, no unjustified deviation, rollback for
   every destructive/migration step. **Stop and confirm** before code. When the human
   confirms the plan, write `Plan approved: <iso>` to `state.md` (see
   [state-workspace](../rite-spec/reference/state-workspace.md)); `$rite-build` checks
   this exists before building.

## tasks.md slice format
```markdown
## SLICE-001 <name>
Goal:
Satisfies: AC-001[, AC-002] # reverse traceability — which spec acceptance criteria this slice satisfies
Acceptance criteria:        # which spec REQ/AC criteria this satisfies
Complexity: N/5 — <reason>  # 1=trivial … 5=hairy; >3 triggers a reslice unless the reason justifies it
Forge: no | yes — <reason>  # default no. Propose yes ONLY when Complexity ≥4 AND the slice has ≥2 genuinely-viable
                            # approaches with no clear winner (an architecture fork, not just "hard"). $rite-vet confirms
                            # or clears it; $rite-build then competes K isolated candidates and keeps one. See rite-build/reference/forge.md.
Mode: AFK | HITL            # AFK = implementable + mergeable without human gating;
                            # HITL = needs a human decision mid-slice (design call,
                            # architectural choice, destructive migration sign-off).
                            # Prefer AFK; only mark HITL with the reason inline.
Gate: advisory | validating | blocking | escalating   # required when Mode=HITL; see reference/gates.md
SLA: 15m | 4h | 24h | none                            # required when Mode=HITL; matches the gate
Checkpoint: <one crisp question>                       # required when Mode=HITL; what the human must decide
Blocked by: SLICE-001, SLICE-002  # other slices that must complete first ("None" if free)
depends_on: [SLICE-001, SLICE-002]  # machine-readable mirror of Blocked by (same set)
Consumes / Produces:        # interfaces this slice reads (types/endpoints/events from prior slices) and exposes for later ones
Known-Gotchas:              # sharp edges / ordering hazards / framework footguns the wright must avoid (keeps the slice one-pass)
Validation commands:        # exact runnable commands that prove the slice green (test / build / typecheck / lint)
Prior-slice learnings:      # (filled forward) what an earlier slice discovered that this one must honor — starts empty
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

> **Why Mode + Gate + Blocked by.** `Mode` lets `$rite-build` and `$rite-status` know
> whether a slice can run unattended or must surface a checkpoint; `Gate` + `SLA` tell
> AFK loops which gates they may auto-handle vs which always pause (see
> [`reference/gates.md`](reference/gates.md) for the four-gate taxonomy:
> advisory / validating / blocking / escalating). `Blocked by` makes the dependency
> graph explicit so re-planning (`$rite-plan reorder`) doesn't break acceptance-criteria
> coverage. Keep `Blocked by` cycle-free. `depends_on` is the machine-readable mirror tools read
> to pick the next *buildable* slice; `Complexity` (>3 → reslice) sizes it; `Satisfies` +
> `Consumes/Produces` + `Known-Gotchas` + `Validation commands` make each slice a self-contained,
> one-pass-implementable brief (the PRP target `$rite-vet` checks). `Forge` flags the rare slice
> worth *competing* — a genuine architecture fork at high complexity, not a slice that is merely
> hard. Define only proposes it; `$rite-vet` confirms or clears it and `$rite-build` acts on it.
> The bulk of slices stay `no` (single-path is cheaper and the default).

> **Mid-flight discipline.** When tempted to skip vertical slicing, coverage mapping, or dependency-order discipline — see [`anti-patterns`](reference/anti-patterns.md) (Common Rationalizations + Red Flags). Load it the moment you reach for the excuse.

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: plan written for <slug>; <n> vertical slices defined.
Changed: architecture.md, plan.md, tasks.md, traceability.md, decisions.md, state.md
Evidence: not applicable; acceptance coverage <x/y> mapped in traceability.md
Open: <none | plan questions | Alternative: $rite-plan to reshape slices>
Next: $rite-vet
Record: .devrites/work/<slug>/plan.md
↻ Hygiene: /clear after user confirms the plan
```

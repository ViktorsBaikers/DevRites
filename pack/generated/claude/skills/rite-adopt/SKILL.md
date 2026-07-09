---
name: rite-adopt
description: Adopt/onboard an existing or legacy codebase into DevRites: reverse-engineer current behavior/spec, seed conventions, start using rites. Use for inherited/live apps. Not for new features.
argument-hint: "[path or area to adopt] [+ what you want to build next]"
user-invocable: true
---

# /rite-adopt — brownfield on-ramp

The **reverse** of `/rite-spec`. `/rite-spec` goes idea → spec; `/rite-adopt` goes
**existing code → spec + seeded conventions**, so an already-built project can enter the
DevRites lifecycle without hand-writing a spec from nothing. It produces the same
`spec.md` the rest of the lifecycle expects, plus a head start in the conventions ledger
so the very first new slice already knows the project's idioms.

Use it once, at the start, to onboard a repo (or a sub-area of one). After it, the normal
lifecycle (`/rite-temper` → `/rite-define` → `/rite-build` …) takes over.

> **Just want a map, not an onboarding?** `/rite-zoom-out` returns a structural map of
> unfamiliar code without creating a workspace or ledger. `/rite-adopt` is the heavier move:
> it *commits the project to the lifecycle*. Pick zoom-out to look, adopt to begin.

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.claude/skills/devrites-lib/reference/standards/core.md` first. Pull `documentation.md` when recording
the adoption decisions (why-not-what) in `decisions.md`; pull `principles.md` when the code
upholds invariants worth proposing as project principles (step 4a).

## Operating rules (DevRites core)
- No silent assumptions · prefer the project's existing conventions (you are *documenting*
  them, not imposing new ones) · ask the human when the adoption scope or the next-build
  objective is unclear.

## Workflow
0. **Read `.claude/skills/devrites-lib/reference/standards/core.md`**, then run
   `devrites-engine preamble` for deterministic workspace orientation.
1. **Scope the adoption** (`$ARGUMENTS`). Which repo or sub-area is being onboarded, and —
   if stated — what the user wants to build *next* on top of it. If the next-build objective
   is missing, ask once (it shapes the spec's acceptance); if the area is ambiguous, confirm
   before investigating the whole tree.
2. **Reverse-investigate the existing code** — the durable shape of the project. Use a
   code-intelligence index if available — codebase-memory-mcp first (its `get_architecture`
   gives a fast overview), cross-checked with codegraph + graphify, else standard methods
   (LSP / Read/Grep/Glob); see `.claude/skills/devrites-lib/reference/standards/tooling.md` — for
   structure, callers, and impact. Capture, per [adoption](reference/adoption.md): **current
   behavior**, **architecture + placement** (layers, seams, where each kind of thing lives),
   the **commands** (test / build / typecheck / lint), and the **idioms** (naming, layering,
   error model, test style) + recurring **gotchas**. Read `PRODUCT.md` / `DESIGN.md` /
   `CLAUDE.md` / `AGENTS.md` if present.
3. **Write `spec.md`** via [rite-spec's spec template](../rite-spec/reference/spec-template.md)
   and create the workspace + set `.devrites/ACTIVE`
   ([state-workspace](../rite-spec/reference/state-workspace.md)). The spec records the
   **current behavior as the baseline** and the **next objective** (what adoption is for) with
   measurable acceptance. Also write `decisions.md`, `assumptions.md`, `questions.md`, and
   `state.md` (phase: spec).
3a. **Seed the capability ledger** from the baseline. If the reverse-derived `spec.md` carries
   structured `### Requirement:` blocks, fold them into the living
   `.devrites/specs/<capability>/spec.md` ledger so the project's *current* proven behavior is on
   record before the first new feature — the ledger the next `/rite-spec` writes deltas against
   ([ledger.md](../rite-ship/reference/ledger.md)). A flat baseline folds as all-ADDED into the
   feature slug's capability; tag capabilities in the spec first if you want finer granularity.
   ```bash
   devrites-engine ledger diff .devrites/work/<slug>   # preview
   devrites-engine ledger sync .devrites/work/<slug>   # seed
   ```
   Skip when the baseline records no structured requirements (nothing to seed).
4. **Seed the conventions ledger** from what the investigation *observed* —
   [adoption § seeding](reference/adoption.md). This is the deliberate bootstrap exception to
   evidence-gated promotion: the seeds start at the base band and are provenance-tagged as
   onboarding observations, not sealed-slice proofs, so real slices later corroborate or
   (fresh-wins) contradict them.
4a. **Propose candidate principles** (human-ratified; optional). Where the investigation found an
   invariant the code *consistently and deliberately* upholds — money always in integer cents, PII
   always redacted from logs, every v1 endpoint preserved — surface it as a **candidate
   principle**, not a seeded convention. Principles are prescriptive and gating, so they are
   **ratified by the human, never auto-seeded** the way conventions are: present the candidates via
   `AskUserQuestion` with the evidence (where the code upholds it), and write the ones the human
   ratifies to `.devrites/principles.md` with a dated Governance entry
   ([`principles.md`](../devrites-lib/reference/standards/principles.md)). Propose, don't impose — an unratified candidate
   stays a convention, not a gate. Skip cleanly when nothing rises to an invariant (common — a
   fresh adopt may declare zero principles, and that's valid).
5. **Hand off.** The project is now in the lifecycle with a spec and a head-start ledger.
   Point the user at `/rite-temper` (big/risky) or `/rite-define` (straightforward) — do not
   plan or build here.

> **Mid-flight discipline.** Don't invent conventions the code doesn't actually follow, don't
> seed an idiom you only assumed, and don't expand scope into a rewrite — adoption documents
> what exists; the *next* feature changes it. See [`anti-patterns`](reference/anti-patterns.md).

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: adopted existing behavior into <slug>; baseline spec and placement recorded.
Changed: spec.md, decisions.md, conventions ledger, principles proposals <updated|none>
Evidence: not applicable; reverse-derived behavior is recorded for review
Open: <none | adoption questions | Alternative: /rite-define for straightforward follow-up>
Next: /rite-temper
Record: .devrites/work/<slug>/spec.md
↻ Hygiene: /clear before the next phase
```

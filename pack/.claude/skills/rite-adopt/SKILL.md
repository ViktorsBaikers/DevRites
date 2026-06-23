---
name: rite-adopt
description: Bring an EXISTING codebase under DevRites — reverse-derive a `spec.md` of current behavior, placement, and architecture, and seed the project conventions ledger from the idioms the code already follows, then hand off to the lifecycle. Use when the user says "adopt this project", "onboard this codebase", "we already have code", "bring this repo into DevRites", or "reverse-engineer a spec from the existing app". Not for a brand-new feature or idea (use `/rite-spec`), planning an approved spec (`/rite-define`), or just mapping unfamiliar code without onboarding it (`/rite-zoom-out`).
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

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. Pull `documentation.md` when recording
the adoption decisions (why-not-what) in `decisions.md`; pull `principles.md` when the code
upholds invariants worth proposing as project principles (step 4a).

## Operating rules (DevRites core)
- No silent assumptions · prefer the project's existing conventions (you are *documenting*
  them, not imposing new ones) · ask the human when the adoption scope or the next-build
  objective is unclear.

## Workflow
0. **Read `.claude/rules/core.md`**, then run the shared orientation preamble:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
1. **Scope the adoption** (`$ARGUMENTS`). Which repo or sub-area is being onboarded, and —
   if stated — what the user wants to build *next* on top of it. If the next-build objective
   is missing, ask once (it shapes the spec's acceptance); if the area is ambiguous, confirm
   before investigating the whole tree.
2. **Reverse-investigate the existing code** — the durable shape of the project. Use a
   code-intelligence index if available — codebase-memory-mcp first (its `get_architecture`
   gives a fast overview), cross-checked with codegraph + graphify, else standard methods
   (LSP / Read/Grep/Glob); see `.claude/rules/tooling.md` — for
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
   ([`principles.md`](../../rules/principles.md)). Propose, don't impose — an unratified candidate
   stays a convention, not a gate. Skip cleanly when nothing rises to an invariant (common — a
   fresh adopt may declare zero principles, and that's valid).
5. **Hand off.** The project is now in the lifecycle with a spec and a head-start ledger.
   Point the user at `/rite-temper` (big/risky) or `/rite-define` (straightforward) — do not
   plan or build here.

> **Mid-flight discipline.** Don't invent conventions the code doesn't actually follow, don't
> seed an idiom you only assumed, and don't expand scope into a rewrite — adoption documents
> what exists; the *next* feature changes it. See [`anti-patterns`](reference/anti-patterns.md).

## Output

**Footer first** — render the progress footer (`progress.sh`, resolved like the step-0
preamble). Then:
```
Adopted: <slug>
Baseline: <one-line summary of current behavior>   Placement: <where it lives>
Next objective: <what we'll build on top>
Conventions seeded: <n> (commands · idioms · placement · gotchas)
Principles proposed: <n ratified → .devrites/principles.md | none rose to an invariant>
Next: big / risky? → /rite-temper   ·   straightforward? → /rite-define
↻ Hygiene: /clear before the next phase (spec.md + decisions.md + the seeded ledger captured). See rules/context-hygiene.md.
```

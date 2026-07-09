---
name: rite-spec
description: Spec a new feature/app before code and write its `.devrites/work/<slug>/` workspace. Use when the user says "spec out a new feature", "I have an idea", or "make a React todo list". Not for approved-spec planning.
argument-hint: "<feature or idea>"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-spec — investigate deeply, write the spec

The spec phase. Turn a request (even a vague one) into a **fully-covered, correctly-placed
`spec.md`** by investigating deeply and closing every material gap with the human — so
`$rite-define` can plan it and nothing is missed. **No plan, tasks, or code here** — those
are `$rite-define` and `$rite-build`.

> **Too small to spec? Use `$rite-quick`.** A typo, copy tweak, config bump, or one-function
> fix does **not** need a full workspace + lifecycle. Stop and run `$rite-quick <change>` — its
> express lane (one contract → TDD build → scoped prove → ship) escalates back here the moment
> the change turns out to touch auth / data / a migration / a public API / more than one slice.
> Spec is for real features; don't pay its ceremony for a one-off.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` first. DevRites skills Read `.agents/skills/devrites-lib/reference/standards/core.md`
as their first step; the other rule files load on demand. Pull `documentation.md` via `Read`
when capturing significant spec decisions (why-not-what, ADR-style notes in `decisions.md`);
pull `principles.md` when the project has declared invariants (`.devrites/principles.md`) — a
new spec must respect them, and a requirement that can only be met by breaking one is a blocking gap.
Pull `spec-grammar.md` and `devrites-lib/reference/workspace-artifact-schema.md` when writing
acceptance for a behavioral / high-risk requirement (auth,
data model, state machine, public API, money, migration) — the structured `### Requirement:` /
`#### Scenario:` (SHALL · WHEN/THEN) form, lint-checked by `devrites-engine spec-validate`. Simple criteria
stay flat `AC-###` bullets; the grammar is opt-in by rigor, never forced.

## Operating rules (DevRites core)
- No silent assumptions · no guessing through confusion · prefer existing conventions ·
  ask the human when an answer changes scope, placement, data model, UX, security,
  migration risk, or acceptance.
- **Author section by section, not in one dump.** Draft the spec one section at a time
  (problem → goal → requirements → acceptance → edge cases), and pause after each: a section
  that carries real risk (a contested requirement, a boundary, an unstated assumption) can be
  deepened right there with a fitting technique from
  [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md) before moving on. A spec
  written a section at a time is where sharper thinking lands cheaply; a wall-of-text spec hides
  the weak section.

## Workflow
0. **Read `.agents/skills/devrites-lib/reference/standards/core.md`** — the always-on operating rules and anti-rationalizations.
   Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   devrites-engine preamble
   ```
0a. **Brownfield check — onboard before speccing onto un-adopted existing code.** If this is an
   **existing codebase** (real source predating DevRites) that has **never been adopted** — no
   `.devrites/conventions.md`, no prior `.devrites/work`, `.devrites/features`, or `.devrites/archive` — the build has no
   conventions ledger to follow and the first slice would guess the project's idioms. Route
   through `$rite-adopt` **first**, carrying `$ARGUMENTS` as its *next objective*: adopt
   reverse-derives the baseline `spec.md`, **seeds the conventions ledger**, and proposes
   principles, so this feature is specced on top of an onboarded project (adopt owns the
   onboarding; `$rite-spec` only detects and routes — no duplicated logic). In **HITL** present a
   ranked option (recommended: *adopt first, then spec this on top*; escape hatch: *spec-only, skip
   onboarding*); in **AFK** (adoption allowed) run `$rite-adopt` first automatically. **Greenfield
   (no pre-existing source) or an already-onboarded project → skip silently** — never block a spec
   for the absence of adoption (the same no-op discipline as the principles gate). Cheap probe:
   ```bash
   if [ ! -f .devrites/conventions.md ] && [ ! -d .devrites/archive ] \
      && [ -z "$(ls .devrites/work .devrites/features 2>/dev/null)" ] \
      && [ -n "$(git ls-files 2>/dev/null | grep -vE '^\.(devrites|claude)/' | head -1)" ]; then
     echo "brownfield, not yet adopted → recommend $rite-adopt first (carry this idea as its next objective)"
   else echo "greenfield or already onboarded → continue spec"; fi
   ```
1. **Understand the request** (`$ARGUMENTS`). Restate the goal and the *real problem
   behind it* in a sentence or two.
1a. **Local dedupe.** Search local issues/PRDs and archived specs before creating a new workspace:
   ```bash
   devrites-engine spec-dedupe "$ARGUMENTS"
   ```
   If it finds a close match, ask the user: extend existing / adopt / new spec. Record the choice in
   `decisions.md` once the workspace exists. No match → continue silently.
2. **Investigate deeply** — [investigation](reference/investigation.md). Produce, and
   later write into the spec: **current behavior**; **placement** (which module/layer/
   file/component should own it, the seam, patterns to reuse, and integration points —
   callers, dependents, data, APIs/events); **what it resolves**; **issues**
   (conflicts/constraints); **gaps** (unknowns); **blast radius**. Use a code-intelligence
   index **if available** — `codebase-memory-mcp` first, cross-checked with `codegraph`
   (`.codegraph/` / `codegraph_*`) + `graphify` (`graphify-out/`), else standard methods
   (LSP / `Read`/`Grep`/`Glob`); see `.agents/skills/devrites-lib/reference/standards/tooling.md` —
   for placement/callers/impact instead of broad file reads; fall back to reading files. For
   uncertain external library/framework facts that bear on placement or feasibility, consult
   context7 if available. When a material decision turns on a fact outside the codebase — a
   common UX pattern, a standard, a prevailing best practice, how comparable products solve it —
   **search the web if available** (brave MCP preferred; see `.agents/skills/devrites-lib/reference/standards/tooling.md`), and
   carry the cited finding into the option you put to the human at step 4.
   Also discover the project's **test / build/typecheck/lint** commands and the
   frontend/backend systems; read `PRODUCT.md` / `DESIGN.md` / `CLAUDE.md` / `AGENTS.md` if
   present (`AGENTS.md` is the cross-tool agent-conventions standard — treat it as project
   conventions the build must follow, same standing as `CLAUDE.md`), and read
   `.devrites/principles.md` if present — the declared invariants the feature must respect.
   **Consult the capability ledger** — the living record of what the system already does
   ([`ledger.md`](../rite-ship/reference/ledger.md)): `devrites-engine ledger list` for the
   capabilities on record, then `devrites-engine ledger show <capability>` for any this feature
   touches. Also search prior decisions with `devrites-engine decisions search "<2-4 feature nouns>"`
   before asking the human to re-decide a settled architecture/API/auth choice. Starting from the proven contract (not a cold re-derive) is what makes the spec store
   compound across features; it also tells you which requirements are new vs a change to existing
   behavior, which decides the delta kind in step 5.
3. **Gather design references (optional)** — [references-intake](reference/references-intake.md).
   The human **may** attach screenshots, mockups, a Figma link, a video, or links — or
   **none at all** (perfectly normal; skip this step then). If any are given: **view/fetch**
   them, **save local files** into `.devrites/work/<slug>/references/`, and index them in
   `references.md`. They become the target later phases verify against.
3a. **Shape the UX/UI before code (if the feature touches UI)** — when this feature is
   frontend ([frontend-trigger](../rite-build/reference/frontend-trigger.md)), apply
   `devrites-ux-shape` **now**, woven into the spec — not as a separate phase. It turns the
   references + the spec into a feature-level **`design-brief.md`** (design direction, key
   states, interaction model, optional Figma/image visual-direction probe) that `$rite-build`
   targets so the UI is built to plan, not guessed. In HITL it pauses for the human to
   confirm the direction; in AFK it asserts the best guess and logs it. Pure
   backend/data/CLI features skip this.
4. **Close the gaps with the human — you recommend, the human decides.** Every material gap
   (one that moves **objective · scope · placement · data model · UX · integration ·
   non-functional · security · migration risk · acceptance**) is **put to the human as a ranked
   option set**, one gap at a time, via the harness `AskUserQuestion` in HITL: **2–4 concrete
   options, your recommended one first and marked `(Recommended)`**, each with a one-line
   rationale + trade-off tagged by the dimensions that matter, plus an escape hatch (*Something
   else — I'll describe it*). Render contract + AFK behaviour: the **Option set** section of
   [`afk-hitl.md`](../devrites-lib/reference/standards/afk-hitl.md); spec-phase question discipline:
   [question-protocol](reference/question-protocol.md). Your job is to **investigate, rank, and
   recommend** — not to settle a material decision yourself. **Confidence changes the *cost* of
   the question, not its *owner*:** when you're near-certain, you still present the set — the
   human just confirms your `(Recommended)` option in one pick — you do **not** silently decide
   it because you predicted the answer. Only a **genuinely reversible, low-impact** detail is
   auto-decided and logged to `assumptions.md`; when unsure whether a gap is material, ask.
   (Vague ask → `devrites-interview`; rough idea → `$rite-pressure-test`.) For a vague ask,
   **map the decision tree** and resolve each branch depth-first; **cover every dimension** —
   each resolved by a human pick or explicitly deferred (logged, non-blocking), never silently
   skipped. Aim for **zero blocking gaps**. *If a gap is genuinely undecidable on paper (state
   machine that may deadlock, data shape ambiguity, "which UX wins") → suggest a
   scoped detour to `$rite-prototype` to answer that ONE question before
   continuing.* **Invariant conflict is a blocking gap:** if a requirement or acceptance
   criterion can only be satisfied by breaking a declared principle (`.devrites/principles.md`),
   surface it — the principle wins by default; breaking it needs a recorded, scoped exception a
   human approves, never a spec that silently contradicts an invariant.
5. **Create the workspace** + set `.devrites/ACTIVE`
   ([state-workspace](reference/state-workspace.md)). Write compact `README.md`,
   `brief.md`, and `spec.md` ([spec-template](reference/spec-template.md)) — WHAT/WHY,
   technology-agnostic, with requirements, acceptance, edge cases, scope boundaries, links
   to future `architecture.md` / `traceability.md`, and measurable acceptance
   ([acceptance-criteria](reference/acceptance-criteria.md)). For a
   behavioral / high-risk requirement, write the acceptance as a structured
   `### Requirement:` (SHALL) + `#### Scenario:` (WHEN/THEN) block per
   [`spec-grammar.md`](../devrites-lib/reference/standards/spec-grammar.md), nesting the `AC-###` id inside each scenario
   so `$rite-seal` still grades it; routine criteria stay flat `AC-###` bullets. **When a
   capability the ledger already holds is changing, write those requirements as deltas** —
   `## ADDED / MODIFIED / REMOVED Requirements — capability: <c>` (spec-grammar.md § Delta form) —
   so the change, not just the end state, is explicit and `$rite-ship` folds it cleanly; a
   capability with no ledger entry stays flat (the first sync seeds it). Also
   write `brief.md`, `references.md`, `questions.md`, `decisions.md`, `assumptions.md`,
   and an initial compact `state.md` (phase: spec) from
   [state-workspace](reference/state-workspace.md). When the feature touches
   UI, `design-brief.md` is written here too (by `devrites-ux-shape`, step 3a).
   Populate `## Edge Coverage` with the deterministic boundary classes implied by each requirement
   (empty/huge input, rounding, timezone, ordering, permissions, races, migration) and `## Prohibitions (must-NOT)`
   only for bespoke constraints. If the feature touches model calls, RAG, agents, evals, or LLM output,
   also create `ai-spec.md` from [ai-spec-template](reference/ai-spec-template.md). Then refresh any
   managed project context block so `AGENTS.md` / `CLAUDE.md` point at the new active workspace:
   ```bash
   devrites-engine context sync || true
   ```
5a. **Score the spec prose — "unit tests for English"** ([spec-checklists](reference/spec-checklists.md)).
   Emit `.devrites/work/<slug>/checklists/<domain>.md` (one per requirement domain the spec covers:
   functional · data-model · interaction · non-functional · edge-cases). Each tests the *requirement
   prose* for completeness / clarity / measurability — "is 'prominent' quantified?", "is every
   enumeration closed?" — **not** the implementation. Fix each CRITICAL fail by editing the spec
   (not by softening the question); minor fails are logged. The checklists feed the readiness gate.
6. **Run the spec readiness gate** (bottom of spec-template): no blocking
   `[NEEDS CLARIFICATION]`, placement decided, all material gaps resolved, any design
   references provided are saved, **UX/UI shaped into `design-brief.md` if the feature is
   UI**, requirements testable, success criteria measurable, **one-sentence intent** (the whole
   change states its intent in a single sentence — if it can't, it is two features: split it or
   narrow the scope), **every `checklists/<domain>.md` at
   `Verdict: pass`**, and **any structured requirement blocks are grammar-valid** — run
   `devrites-engine spec-skeleton` first, then `devrites-engine spec-validate` with
   `--against .devrites/specs` so any delta sections are also
   reconciled against the ledger (an ADDED that already exists, or a MODIFIED/REMOVED that doesn't,
   is a blocking failure to fix, not soften):
   ```bash
   devrites-engine spec-skeleton ".devrites/work/<slug>"
   devrites-engine spec-validate ".devrites/work/<slug>" --against .devrites/specs
   ```
   Treat edge/prohibition findings as blocking just like grammar findings. When it passes, write `Spec gate: passed <iso>` to `state.md`.
6a. **Review-before-code digest.** Before handing off to planning, render the cheap human review:
   `Intent` (one sentence), `Done means` (top acceptance/scenario IDs), `Scope/risk` (what is in/out
   plus the hard gates), and `Build exactly this?` (yes → next phase; no → revise now). The digest
   is a view over `spec.md`, not a new artifact. **Stop** after the digest.

> **Mid-flight discipline.** When tempted to skip investigation depth, gap-closing, or placement decisions — see [`anti-patterns`](reference/anti-patterns.md) (Common Rationalizations + Red Flags). Load it the moment you reach for the excuse.

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: spec ready for <slug>; placement decided and gaps closed.
Changed: spec.md, decisions.md, assumptions.md, questions.md, references/ <updated|n/a>
Evidence: checklists passed; grammar <valid | n/a flat acceptance>; design brief <path | n/a>
Open: <none | n non-blocking questions | Alternative: $rite-define for small reversible work>; review digest: intent + done-means + scope/risk rendered
Next: $rite-temper
Record: .devrites/work/<slug>/spec.md
↻ Hygiene: /clear before the next phase; $rite-handoff if away > a few hours
```
If a workspace with the slug already exists, update its spec rather than overwriting blindly —
and **show the human a short diff of what changed** in `spec.md` (acceptance criteria added /
removed / reworded) before proceeding. A spec edit reviewed as a diff catches silent scope
drift that a full re-read buries; this is the spec-review view (`$rite-spec --review` renders
just the diff + the open-question delta, no re-investigation).

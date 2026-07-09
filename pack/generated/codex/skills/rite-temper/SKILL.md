---
name: rite-temper
description: Temper a readied spec before planning. Use when the user says "temper this", "strategy review", "pre-mortem the spec", or asks if we are over/under-building. Not for code review or final seal.
argument-hint: "[feature-slug] [--mode expand|selective|hold|reduce]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-temper — temper the spec before you plan

Take a readied spec and **temper** it: heat it (raise ambition on the *outcome*), then
quench it (pre-mortem + prune the *solution surface*) — stronger and less brittle. The one
DevRites step that decides scope/ambition on a written spec and folds the result into the
canonical contract, so `$rite-define` plans the **hardened** spec. Optional for small work
(significance-gated — auto-skips low-stakes specs), but always invoked inside
`$rite-autocomplete`. **Read the active workspace first**; if there's no readied
`spec.md`, tell the user to run `$rite-spec`.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` first. Pull on demand: `patterns.md` +
`coding-style.md` (the over-engineering / YAGNI rubric — reuse the pack's standard, don't
invent one), `documentation.md` (ADR-style `decisions.md` entries), `afk-hitl.md`
(irreversible-risk list + gate ceiling), `elicitation.md` (the move-set to deepen a section
that needs more than the default pre-mortem — selected by the section's risk).

## Operating rules
- **Ambition on outcomes, minimalism on the surface.** Raise the success bar and solve the
  *real* problem; never add speculative capability/abstraction. The reconciliation hinge —
  bigger ambition, smaller surface.
- **Every scope change routes through the Spec Drift Guard** — it only becomes real by being
  written into `spec.md` as a recorded, confirmable decision. Expansion is **opt-in** (HITL
  confirm; AFK pauses — see autocomplete policy). Nothing auto-grows into the build.
- **Bound ambition by reversibility.** Auth / migration / public-API / data-model touches get
  the *opposite* of ambition — maximum conservatism; they always pause (irreversible-risk list).
- **Honest verdict.** Never round "needs work" up to "ready"; record every scope call's *why*.
- **You write; the reviewer judges.** You are the single canonical writer (`strategy.md` +
  the spec edits); the reviewer agent is read-only.
- **Demand the spec-quality checklists for big features.** `$rite-temper` only fires on
  significant specs — exactly where vague requirement prose is most expensive. Before hardening,
  confirm `checklists/<domain>.md` exist and pass (`rite-spec/reference/spec-checklists.md`); a
  scope expansion you fold in **adds its own** rows (new Success/Acceptance criteria get
  "unit-tested for English" too). A folded expansion with an unquantified criterion is a CRITICAL
  fail that re-opens the readiness gate.

## Workflow
0. **Read `.agents/skills/devrites-lib/reference/standards/core.md`**. Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   devrites-engine preamble
   ```
   Then read the workspace: `spec.md` (+ `decisions.md`,
   `assumptions.md`, `design-brief.md` if UI), `state.md`. Require `Spec gate: passed` — else
   STOP → `$rite-spec`. If a plan already exists, scope changes route through `$rite-plan
   repair` (record in `drift.md`), not a blind spec edit.
1. **Significance test** — [`reference/significance.md`](reference/significance.md). Low-stakes
   / shape-not-meaning work → write the one-line `skipped — low stakes (<trigger>)` verdict to `strategy.md`,
   set `state.md` `Next step: $rite-define`, and recommend it. Otherwise fire the full pass below.
2. **FORWARD pass + mode selection** — [`reference/scope-modes.md`](reference/scope-modes.md).
   First, the **one-sentence-intent test**: state the whole change's intent in a single sentence.
   If you can't without an "and" that joins two unrelated outcomes, it is **two features** — the
   sharpest scope-creep signal there is; recommend splitting the spec (or narrowing this one) before
   tempering further. Then diverge on the *outcome* (the 10-star version of the real problem), and
   commit **exactly one** scope mode (`expand` opt-in · `selective` · `hold-rigor` · `reduce-to-MVP`)
   with its rationale and the **hinge** (what would change the call). `$ARGUMENTS` `--mode` is a
   hint, not a command.
3. **INVERSION pass** — pre-mortem in past tense ("it shipped and failed — what went wrong"),
   each top risk carrying likelihood + mitigation + the slice it will bind to; then the **YAGNI
   ledger** (each candidate scope item gets the "imagine the later refactor" test; defer unless
   now-cost is trivial AND deferred-cost is large).
   - **Deepen on demand.** When a specific scope call, requirement, or risk needs sharper thinking
     than the default pre-mortem gives, pull the 3–5 techniques from
     [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md) whose *when-to-reach-for-it* matches that
     section's risk (irreversible decision → Red-Team/Blue-Team + Assumption Audit; sizing → Delphi;
     vague requirement → Steelman-then-Attack), offer them, and run the chosen one **on that one
     section**. Record what it changed, not that you ran it. Optional — the default passes stand on
     their own.
4. **Score the 9 dimensions** — [`reference/review-dimensions.md`](reference/review-dimensions.md).
   Cite evidence *before* the band; **gate on the floor** (the weakest dimension, not an average).
5. **Walk the findings WITH the human — do not batch-dump.** The artifact is the *output* of an
   interactive review, not a substitute for it: surface each material scope call / finding via
   `AskUserQuestion`, one at a time, best-guess + **why**. Each material scope call ends as a
   **recorded decision** — a resolved `questions.md` qid (HITL) or a `decisions.md` ADR (AFK) —
   so the walk leaves an auditable trail, not just chat. (AFK gate policy is single-sourced in
   [`reference/significance.md`](reference/significance.md): `hold-rigor` + `reduce-to-MVP`
   auto-apply, **any `expand` is a blocking pause**, irreversible-risk always pauses.)
6. **Write `strategy.md` + fold back** — [`reference/strategy-template.md`](reference/strategy-template.md).
   **One drift rule (decide by whether a plan exists):** *no `plan.md` yet* (the normal pre-define
   case) → update `spec.md` **through the Spec Drift Guard** — *Success criteria* **and**
   *Acceptance criteria* for each opt-in expansion / cut, **Non-goals** for every deferred item,
   *Constraints* **and** *Risks* the pre-mortem demands, and the gaps/decisions table; *`plan.md`
   already exists* → do **not** edit `spec.md` here — write the deltas to `drift.md` and hand off
   to `$rite-plan repair`. Either way, append `decisions.md` (one ADR per scope call: context ·
   decision · why-not · what-would-change-it) and `assumptions.md` (every "we'll probably need X" →
   assumption-to-verify). **Every scope delta must carry a recorded human decision** — a resolved
   `questions.md` qid (HITL) or a `decisions.md` ADR (AFK within the ceiling); a folded change with
   no recorded decision is the batch-dump failure. **Re-check the spec Readiness gate** (it fails if
   any folded scope delta lacks its decision). Update `state.md`: `Phase: temper`,
   `Next step: $rite-define`; on a blocking pause (expand / irreversible / a dimension still below
   bar) write the `Awaiting human` block + `Status: awaiting_human` before stopping.
7. **Adversarial verification loop** — dispatch [`devrites-strategy-reviewer`](.codex/agents/devrites-strategy-reviewer.toml)
   (fresh context, **only** the hardened spec + the rubric — no authoring reasoning). Resolve
   actionable findings by editing `spec.md`/`strategy.md`, re-dispatch; **cap ≤3 iterations**. A
   dimension still below bar after 3 → blocking question (HITL) or AFK gate-ceiling entry;
   irreversible-risk findings always pause. If sub-agents are unavailable, do the independent
   rubric pass yourself in a separate read, discarding the authoring reasoning (a flagged
   fallback, not an independent review).
8. **STOP.** Report the mode, the scope deltas, and the floor verdict; recommend `$rite-define`.

> **Mid-flight discipline.** When tempted to dump findings into `strategy.md` and skip the
> walk-through, expand the surface (not the outcome), grow scope without the Drift Guard, or
> score before citing evidence — see [`reference/anti-patterns.md`](reference/anti-patterns.md).

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: spec tempered for <slug>; mode <expand|selective|hold-rigor|reduce-to-MVP|skipped-low-stakes>.
Changed: strategy.md, spec.md, decisions.md, assumptions.md
Evidence: dimension floor <band>; reviewer loop <n>; spec readiness re-checked <pass|blocked>
Open: <none | blocker | n deferred non-goals>
Next: $rite-define
Record: .devrites/work/<slug>/strategy.md
↻ Hygiene: /clear before $rite-define
```
**DO NOT plan, slice, or write code here** — that's `$rite-define` and `$rite-build`.

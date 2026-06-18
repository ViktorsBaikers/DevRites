---
name: rite-temper
description: Strategic review of a readied `spec.md` before planning — pick a scope mode (expand / selective / hold-rigor / reduce-to-MVP), run a pre-mortem, harden the spec, and fold decisions back via the Spec Drift Guard plus a persistent `strategy.md`. Use when the user says "temper this", "strategy review", "think bigger", "scope check", "pre-mortem the spec", "are we over/under-building", or before defining a big or risky feature. Not for a rough idea (`/rite-pressure-test`), a mid-build decision (`devrites-doubt`), a code diff (`/rite-review`), or the final gate (`/rite-seal`).
argument-hint: "[feature-slug] [--mode expand|selective|hold|reduce]"
user-invocable: true
---

# /rite-temper — temper the spec before you plan

Take a readied spec and **temper** it: heat it (raise ambition on the *outcome*), then
quench it (pre-mortem + prune the *solution surface*) — stronger and less brittle. The one
DevRites step that decides scope/ambition on a written spec and folds the result into the
canonical contract, so `/rite-define` plans the **hardened** spec. Optional for small work
(significance-gated — auto-skips low-stakes specs), but always invoked inside
`/rite-autocomplete`. **Read the active workspace first**; if there's no readied
`spec.md`, tell the user to run `/rite-spec`.

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. Pull on demand: `patterns.md` +
`coding-style.md` (the over-engineering / YAGNI rubric — reuse the pack's standard, don't
invent one), `documentation.md` (ADR-style `decisions.md` entries), `afk-hitl.md`
(irreversible-risk list + gate ceiling).

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

## Workflow
0. **Read `.claude/rules/core.md`**. Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
   Then read the workspace: `spec.md` (+ `decisions.md`,
   `assumptions.md`, `design-brief.md` if UI), `state.md`. Require `Spec gate: passed` — else
   STOP → `/rite-spec`. If a plan already exists, scope changes route through `/rite-plan
   repair` (record in `drift.md`), not a blind spec edit.
1. **Significance test** — [`reference/significance.md`](reference/significance.md). Low-stakes
   / shape-not-meaning work → write the one-line `skipped — low stakes (<trigger>)` verdict to `strategy.md`,
   set `state.md` `Next step: /rite-define`, and recommend it. Otherwise fire the full pass below.
2. **FORWARD pass + mode selection** — [`reference/scope-modes.md`](reference/scope-modes.md).
   Diverge on the *outcome* (the 10-star version of the real problem), then commit **exactly
   one** scope mode (`expand` opt-in · `selective` · `hold-rigor` · `reduce-to-MVP`) with its
   rationale and the **hinge** (what would change the call). `$ARGUMENTS` `--mode` is a hint, not
   a command.
3. **INVERSION pass** — pre-mortem in past tense ("it shipped and failed — what went wrong"),
   each top risk carrying likelihood + mitigation + the slice it will bind to; then the **YAGNI
   ledger** (each candidate scope item gets the "imagine the later refactor" test; defer unless
   now-cost is trivial AND deferred-cost is large).
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
   to `/rite-plan repair`. Either way, append `decisions.md` (one ADR per scope call: context ·
   decision · why-not · what-would-change-it) and `assumptions.md` (every "we'll probably need X" →
   assumption-to-verify). **Every scope delta must carry a recorded human decision** — a resolved
   `questions.md` qid (HITL) or a `decisions.md` ADR (AFK within the ceiling); a folded change with
   no recorded decision is the batch-dump failure. **Re-check the spec Readiness gate** (it fails if
   any folded scope delta lacks its decision). Update `state.md`: `Phase: temper`,
   `Next step: /rite-define`; on a blocking pause (expand / irreversible / a dimension still below
   bar) write the `Awaiting human` block + `Status: awaiting_human` before stopping.
7. **Adversarial verification loop** — dispatch [`devrites-strategy-reviewer`](../../agents/devrites-strategy-reviewer.md)
   (fresh context, **only** the hardened spec + the rubric — no authoring reasoning). Resolve
   actionable findings by editing `spec.md`/`strategy.md`, re-dispatch; **cap ≤3 iterations**. A
   dimension still below bar after 3 → blocking question (HITL) or AFK gate-ceiling entry;
   irreversible-risk findings always pause. If sub-agents are unavailable, do the independent
   rubric pass yourself in a separate read, discarding the authoring reasoning (a flagged
   fallback, not an independent review).
8. **STOP.** Report the mode, the scope deltas, and the floor verdict; recommend `/rite-define`.

> **Mid-flight discipline.** When tempted to dump findings into `strategy.md` and skip the
> walk-through, expand the surface (not the outcome), grow scope without the Drift Guard, or
> score before citing evidence — see [`reference/anti-patterns.md`](reference/anti-patterns.md).

## Output
```
Tempered: <slug>
Significance: full | skipped — low stakes (<trigger>)
Mode: <expand|selective|hold-rigor|reduce-to-MVP> — <one-line rationale> (hinge: <what would change it>)
Scope: +<n expanded> / -<n reduced> / <n deferred to Non-goals>   (all via Spec Drift Guard)
Pre-mortem: <n top risks> mitigated → bound to slices
Dimensions: floor = <strong|adequate|thin|broken> on <weakest dimension>   Reviewer loop: <n> iter
Spec readiness: re-checked → passes | <blocker>
Next: /rite-define   (plans the hardened spec)
↻ Hygiene: /clear before /rite-define (strategy.md + spec edits + decisions.md + assumptions.md captured). See rules/context-hygiene.md.
```
**DO NOT plan, slice, or write code here** — that's `/rite-define` and `/rite-build`.

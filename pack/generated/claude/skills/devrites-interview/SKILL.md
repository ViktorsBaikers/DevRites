---
name: devrites-interview
description: Interview the user one question at a time to extract intent. Use when the user says "interview me", "I am not sure what I want", or the ask is underspecified. Not for casual clarification.
user-invocable: false
---

# devrites-interview: extract intent

Resolve the difference between the request and the user's intent before writing a
spec, plan, or code.

## Protocol
- **Open with a working hypothesis.** State in one line what the user wants, then ask about the
  highest-value unresolved human-owned material decision.
- **One question per turn.** Multiple questions get one answered and the rest ignored.
- **Attach an explicit recommendation** and its reason to every question:
  > "I recommend CSV only because it covers the use case. Choose CSV, CSV + XLSX,
  > or something else?"
  This is a concrete candidate for the user's decision, not an assumption or confirmation.
- **Highest-value question first:** order by how much the answer changes the build. A
  question that moves the data model or acceptance criteria beats a cosmetic one.
- **Prioritize impact and bound each pass.** Order unknowns **scope > security/privacy > UX >
  technical**. A pass MAY contain at most **3 questions** for cognitive load. After each pass,
  rescan the decision tree and continue in later turns until every human-owned material decision
  is explicitly selected or explicitly deferred by the user. A cap MUST NOT move a blocker to
  `assumptions.md`. Own and log reversible, low-impact technical details instead of asking.
- **Structured options** when the space is enumerable: present them as the standard ranked
  **option set** ([`afk-hitl.md` → "Option set"](../devrites-lib/reference/standards/afk-hitl.md#option-set-how-every-gap-is-presented)): recommended **first**, labelled
  `(Recommended)`, each with a dimension-tagged rationale (`logic · infra · business ·
  architecture`, + `security`/`UX`/`risk` when in scope), plus the escape hatch. Render via
  `AskUserQuestion` when the harness has it:
  ```
  1. <recommended> (Recommended) — logic: … · infra: … · business: … · architecture: …
  2. <alternative> — <rationale + the trade-off it accepts>
  3. Something else — I'll describe it
  ```

## Completion condition
Complete only when a topology and decision-tree rescan finds no unresolved human-owned
material decision. Such a decision closes only through the user's option selection,
free-form answer, or explicit deferral. A recommendation is not confirmation. If the user
stops early, persist every visible blocker and MUST NOT claim readiness or advance.

Ask fewer, deeper questions. If answers stop converging around one area, **reframe once**
instead of asking another version of the same question.

## Want vs. should-want
Users sometimes name the convention or best practice they think they should want instead
of their actual preference. Signals include abstract virtues ("scalable", "clean",
"modern"), deference to convention, or an answer that fits any project. Ask: *"if you
didn't have to justify this to anyone, what would you want?"* Record that answer rather
than designing to the generic one.

## What counts as a yes
Approval requires an **explicit** yes. Treat these replies differently:
- *"Whatever you think is best"* / *"you decide"*: **delegation, not approval.** Offer two
  concrete options so the user chooses the substance or explicitly defers it.
- *"Sounds good"* / *"sure, let's go"*: confirm once for a material decision rather than
  treating a polite exit as approval.
- **Silence:** not consent. State what is out of scope and get explicit approval for it.

## Don't ask
- Things the codebase answers (read it first).
- Reversible, low-impact technical details (decide and log as agent-owned assumptions).
- Everything at once "to be thorough."

## When the ask is vague: map the decision tree first
For a short or vague request (`"design a contact page"`), do not ask isolated questions.
Sketch the **decision tree** first, then resolve each branch **depth-first** with the
protocol above. Domain branches per area:
`rite-spec/reference/interview-patterns.md`.

## Reframe (once, when stuck)
If the interview is not converging, use **one** turn to challenge the premise:
*"is a form even the right answer here, or a mailto / booking link?"* Then resume the
protocol with the revised premise.

## /clarify mode: coverage scan of an existing spec
When clarifying a written spec rather than extracting intent from scratch, first enumerate
its actors, journeys/components, states, data boundaries, interfaces/integrations, and
operational/proof surfaces. Scan every material surface against the caller's fixed taxonomy and mark
**Clear / Partial / Missing** with evidence. Ask at most **3 prioritized decision packets per
pass**, targeting Missing before Partial, one per turn with an explicit recommendation and reason.
A packet may close
several cells only when they share one owner and trade-off.

**Integrate each answer into the relevant spec section** immediately. A Q&A log alone is
insufficient.
Append a dated **`## Clarifications`** block to `spec.md` with the question and resolution.
Re-run the scan after each pass. Packet limits control cognitive load, never readiness. Continue
later passes while human-owned blockers remain, and stop only when every row is clear, agent-owned,
not applicable, or explicitly deferred by the user with a nonblocking reason, owner, and validation
gate. Then re-score the affected
`checklists/<domain>.md`.

## Output
A short summary for the caller: objective in one sentence, confirmed decisions,
open non-blocking items, and the recommended next step. If a workspace is active, write
Q&A to `questions.md`, confirmed calls to `decisions.md`, and only agent-owned reversible,
low-impact technical details to `assumptions.md`. If not, return the summary without creating
a workspace.

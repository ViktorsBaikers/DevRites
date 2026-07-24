---
name: devrites-interview
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- For every DevRites specialist or writer dispatch, first call `spawn_agent` with the named `devrites-<role>` custom role. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If `spawn_agent` is callable but a named read-only role is unavailable, use generic `explorer` only when the host proves that run has a runtime-enforced read-only sandbox. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. A missing read-only custom role is not evidence that spawning is unavailable.
- Never dispatch generic `worker` for `devrites-slice-wright` unless the host proves that worker run carries exact DevRites identity and the same `.wright-allowlist` enforcement as the named role. Codex reports a generic run as `agent_type=worker`, so the generated global hooks cannot prove that binding. Reject that unsafe rung and use the documented labelled inline wright path with `.reconcile-inline` plus the full reconcile gate.
- If the host cannot prove the generic explorer is runtime read-only, reject that rung too. Only when no spawn primitive exists or a higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, and apply every fallback risk gate. An unbound generic wright or unconfined generic explorer is such a safety rejection, not evidence that no agents exist.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# devrites-interview: extract intent

Resolve the difference between the request and the user's intent before writing a
spec, plan, or code.

## Protocol
- **State a confidence number.** Open with a one-line hypothesis of what the user wants and an
  explicit **0-100%** confidence. Check the number against the next three questions you would ask:
  if you cannot predict the user's answers, lower it. Below **~70%**, append the single unresolved
  point so the user can answer it directly.
- **One question per turn.** Multiple questions get one answered and the rest ignored.
- **Attach your best guess** and its reason to every question:
  > "I'm assuming export is CSV only (covers the stated use case). Right, or also XLSX?"
  The user can then correct a concrete premise instead of answering an open-ended question.
- **Highest-value question first:** order by how much the answer changes the build. A
  question that moves the data model or acceptance criteria beats a cosmetic one.
- **Prioritize impact and limit blockers.** Order unknowns **scope > security/privacy > UX >
  technical**. Ask at most **3 blocking questions** per pass, choosing only those that gate
  the spec. Record the rest in `assumptions.md` with the best guess and reason. A reversible
  detail is never blocking.
- **Structured options** when the space is enumerable: present them as the standard ranked
  **option set** (`standards/afk-hitl.md` → "Option set"): recommended **first**, labelled
  `(Recommended)`, each with a dimension-tagged rationale (`logic · infra · business ·
  architecture`, + `security`/`UX`/`risk` when in scope), plus the escape hatch. Render via
  `AskUserQuestion` when the harness has it:
  ```
  1. <recommended> (Recommended) — logic: … · infra: … · business: … · architecture: …
  2. <alternative> — <rationale + the trade-off it accepts>
  3. Something else — I'll describe it
  ```

## Stop condition
Stop **opening new questions** when any condition below holds. This does not transfer a
material decision to the agent. Present that decision as a ranked option set even when
high confidence makes it a one-pick confirmation.
- **Confidence: the predict-three test.** At 95%, you should be able to predict the user's
  reaction to the next three questions. If so, stop. If several rounds pass without that
  confidence, name the missing premise instead of asking more narrow questions.
- **Convergence:** the last 2-3 answers only rubber-stamped your guesses and didn't
  move the spec.
- **Soft cap:** after ~8 material questions, proceed with your best-guess answers logged
  in `assumptions.md` rather than asking more (hard-stop sooner if the ask is small).

Ask fewer, deeper questions. If answers stop converging and the discussion circles one
area, **reframe once** instead of asking another version of the same question.

## Want vs. should-want
Users sometimes name the convention or best practice they think they should want instead
of their actual preference. Signals include abstract virtues ("scalable", "clean",
"modern"), deference to convention, or an answer that fits any project. Ask: *"if you
didn't have to justify this to anyone, what would you want?"* Record that answer rather
than designing to the generic one.

## What counts as a yes
Approval requires an **explicit** yes. Treat these replies differently:
- *"Whatever you think is best"* / *"you decide"*: **delegation, not approval.** Offer two
  concrete options so the user chooses the substance.
- *"Sounds good"* / *"sure, let's go"*: confirm once for a material decision rather than
  treating a polite exit as approval.
- **Silence:** not consent. State what is out of scope and get explicit approval for it.

## Don't ask
- Things the codebase answers (read it first).
- Reversible implementation details (decide, log as an assumption).
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
**Clear / Partial / Missing** with evidence. Ask **≤5 prioritized decision packets per scan**,
targeting Missing before Partial, one per turn with a best-guess attached. A packet may close
several cells only when they share one owner and trade-off.

**Integrate each answer into the relevant spec section** immediately. A Q&A log alone is
insufficient.
Append a dated **`## Clarifications`** block to `spec.md` with the question and resolution.
Re-run the scan after answers. The general question caps control cognitive load; they never turn
a material blocker into a silent assumption. Continue another scan while human-owned blockers
remain, and stop only when every row is clear, agent-owned, not applicable, or explicitly
deferred with a nonblocking reason, owner, and validation gate. Then re-score the affected
`checklists/<domain>.md`.

## Output
A short summary for the caller: objective in one sentence, confirmed decisions,
open non-blocking items, and the recommended next step. If a workspace is active, write
Q&A to `questions.md`, confirmed calls to `decisions.md`, standing guesses to
`assumptions.md`. If not, just return the summary: don't create a workspace.

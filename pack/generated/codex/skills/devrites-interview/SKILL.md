---
name: devrites-interview
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. Codex loads that role TOML's `developer_instructions` natively. Because V2 collaboration lifecycle calls bypass hooks, DevRites verifies the current durable parent/child rollout for the exact role, wait, completion, and non-empty delivered result.
- On MultiAgent V1, when the named role is not exposed, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`; do not substitute `worker` for an exposed V2 named role.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If any required named or generic agent dispatch is unavailable or rejected, stop for HITL. Never execute a DevRites specialist role in the root context.
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

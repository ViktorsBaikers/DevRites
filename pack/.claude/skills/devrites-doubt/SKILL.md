---
name: devrites-doubt
description: Stress-test a single non-trivial decision via CLAIM → EXTRACT → DOUBT → RECONCILE → STOP with `devrites-doubt-reviewer` for an independent take. Use when the user says "are we sure", "double-check this", "what could go wrong", or a boundary / data-model / auth / public-API / migration change is about to commit. Not for post-merge review or trivial choices.
user-invocable: false
---

# devrites-doubt — CLAIM → EXTRACT → DOUBT → RECONCILE → STOP

A pre-mortem on a single decision, not a final review. Find what's wrong before it's
load-bearing.

## When to use
Introducing branching logic · crossing a module/service boundary · changing the data
model · modifying auth/authz · changing a public API · touching migrations · changing a
browser/user flow · relying on an assumption tests can't prove · working in unfamiliar
code · claiming "this is safe", "this scales", or "this matches the spec".

## The cycle (copy this checklist)

- [ ] **1. CLAIM** — state the claim in 1–3 sentences + why it matters.
- [ ] **2. EXTRACT** — isolate the smallest reviewable artifact + its contract; strip your reasoning so the reviewer sees only the code/decision.
- [ ] **3. DOUBT** — invoke a fresh-context reviewer with an ADVERSARIAL prompt: *"find what's wrong; do not validate."* Prefer a real subagent (`.claude/agents/devrites-doubt-reviewer`) so it has no anchoring context.
- [ ] **4. RECONCILE** — classify EVERY finding: contract misread | valid & actionable | valid trade-off | noise.
- [ ] **5. STOP** — met a stop condition (only trivial findings, 3 cycles done, or user override). Emit a **binary gate verdict** the orchestrator must clear: **accept** (no valid-&-actionable findings remain) or **reject + the specific required changes**. On reject, the orchestrator loops the wright on those changes before the slice is accepted; still reject after the 3-cycle cap → escalate to the user.

## Deletion-test lens (for "is this abstraction load-bearing?" doubts)

When the claim is "this new module / boundary / wrapper is worth it", apply the
**deletion test** before standing it: *imagine the abstraction never existed — does
its complexity vanish (it was a pass-through, the abstraction was added on speculation)
or does the same complexity re-appear distributed across N callers (it concentrates real
complexity, deletion would smear it)?* Pass-throughs that fail the test get downgraded
to "not yet" — wait for the second real caller before standing the seam.

## Rules
- For "where does this claim reach / what would change with it" questions, prefer a
  code-intelligence index if available — codebase-memory-mcp (`detect_changes` / `trace_path`)
  first, cross-checked with codegraph (`codegraph_impact` / `codegraph_callers`) + graphify,
  else standard methods (LSP / Read/Grep/Glob); see `.claude/rules/tooling.md` — over file
  reads; they answer impact in one call without polluting context.
- The reviewer prompt must be adversarial — its job is to break the claim, not to agree.
- Strip your own justification before review; reasoning anchors the reviewer toward
  agreement.
- Loop **max 3 times**. If material uncertainty remains after 3, **ask the user**.
- Act on "valid & actionable" findings (fix or re-plan). Accept "valid trade-off"
  explicitly in `decisions.md`. Discard "noise" with a one-line reason. Re-check
  "contract misread" against the actual contract text.
- In interactive sessions, a **cross-model second opinion** is allowed **only with
  explicit user authorization**. Never run external CLIs without authorization.

## AFK exception

When `.devrites/AFK` exists and the user is away, `escalated to user` is unavailable in
real time. Map the verdict to a `questions.md` entry instead of a synchronous prompt:

- **Finding severity ≤ slice's gate ceiling** (the slice's `Gate:` plus `.devrites/AFK`
  `allow_gates`, default `[advisory]`): append a `questions.md` entry with
  `gate: advisory`, record the trade-off in `decisions.md`, and proceed with the best
  inference. The advisory is surfaced by `/rite-status` so the user sees it on return.
- **Finding severity > gate ceiling, OR the claim touches destructive migration,
  auth/authz boundaries, public APIs, irreversible data writes**: append a
  `questions.md` entry with `gate: blocking`, set `state.md` `Status: awaiting_human`,
  fire the `notify:` hook, and STOP. AFK never silently accepts irreversible risk.

The 3-loop limit still applies — after 3 cycles, the verdict is `escalated to user` and
the unresolved doubt becomes a blocking question regardless of AFK config.

## Output
```
Claim: ...
Gate: accept | reject — <the specific required changes, if reject>
Verdict: holds | revised | escalated to user
Actionable findings handled: ...
Trade-offs accepted (→ decisions.md): ...
```

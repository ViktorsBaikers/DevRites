---
name: rite-plan
description: Reshape an active plan — decompose / reslice / repair / reorder / split FE-BE / unblock — without changing product behavior unless asked. Use when the user says "replan", "reslice", "repair the plan", "drift", "unblock", or `/rite-build` flags Spec Drift. Not for the initial plan (use `/rite-define`).
argument-hint: "[mode: decompose|reslice|repair|reorder|split|unblock]"
user-invocable: true
---

# /rite-plan — (re)plan an active feature

Reshape the plan when reality and the plan disagree. **Read the active workspace
first.** If `.devrites/ACTIVE` is empty or its workspace is missing, stop and tell the
user to run `/rite-spec <feature>`.

## Rules consulted (read on demand from `.claude/rules/`)
Read `.claude/rules/core.md` first. Pull `development-workflow.md` via `Read` when
reshaping slice cadence or DoD criteria.

## Operating rules
- Spec is living, not sacred — but never plan around a known-wrong assumption silently.
- If a change alters product behavior, scope, architecture, data model, UX, security,
  or migration risk → **ask the user first** (use the Spec Drift Guard question format).
- Keep each slice small enough for one focused build → prove cycle.
- **Slice count is derived, never dictated** — reslice when a slice fails the sizing rule
  (multiple "and"s, can't build+prove in one cycle), not to hit a user-named tally. A
  requested count is a hint at most; slice logically and explain if it differs. See
  [`reference/slicing.md`](reference/slicing.md) ("How many slices?").

## Workflow
0. Read `.claude/rules/core.md` (operating rules) before reshaping anything.
   Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
1. Read `spec.md`, `plan.md`, `tasks.md`, `state.md`, `drift.md`, and the current
   `git diff` (if a repo). Read `decisions.md` and `assumptions.md`. If a code-intelligence
   index is available — `codegraph` (`.codegraph/` / `codegraph_*` tools) or `graphify`
   (`graphify-out/`) — prefer it for structural questions (what calls X, what would
   changing Y break) over reading whole files, to keep planning context lean.
2. **Pick the mode** (`$ARGUMENTS` or infer):
   - **decompose** — first/again break the feature into vertical slices.
   - **reslice** — a slice is too large; split into thinner end-to-end slices.
   - **repair** — a Spec Drift Guard event; fold the resolution into plan + tasks.
   - **reorder** — fix the dependency order.
   - **split** — separate backend/frontend contracts (see `devrites-api-interface`).
   - **unblock** — a verification failed; re-route around the blocker.
   See [replan-and-repair](reference/replan-and-repair.md) for each mode's steps.
3. Reason about dependencies — [dependency-graph](reference/dependency-graph.md).
4. Re-slice using vertical-slice rules — [slicing](reference/slicing.md) and
   [task-breakdown](reference/task-breakdown.md). Prefer thin, shippable, verifiable.
5. Update `plan.md`, `tasks.md`, `state.md`, and append rationale to `decisions.md`.
   If you stopped for drift, mark the `drift.md` entry resolved.
6. If product behavior/acceptance criteria change, confirm with the user before writing.

> **Mid-flight discipline.** When tempted to change product behavior without asking, absorb drift silently, or skip the user — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output

**Footer first** — render the slice meter + flow ribbon by running the progress footer (`progress.sh`, resolved like the step-0 preamble — canonical snippet in `devrites-lib/SKILL.md`); keep the fact lines below it terse (`key value · key value`). Then:
```
Re-planned <slug> — mode: <mode>
Changes: <what moved/split/repaired>
Slices now: N (next: slice <N — name>)
Behavior change? <no | asked + answer>
Next: /rite-build   (or /rite-prove if a built slice needs re-verification)
↻ Hygiene: /clear if reshape was big or unwound a drift; keep session for small reorders. See rules/context-hygiene.md.
```

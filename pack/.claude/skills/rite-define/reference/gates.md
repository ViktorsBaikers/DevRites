# Gate taxonomy — advisory · validating · blocking · escalating

DevRites uses a four-gate model for HITL pauses, adapted from the regulated-agentic-workflow
governance pattern. Picking the right gate for each `Mode: HITL` slice is the difference
between a workflow that catches real risk and one that becomes a review queue.

> **Default failure mode:** marking every HITL slice as `blocking`. The same reviewer ends
> up approving low-stakes and high-stakes items at the same priority and the gate becomes
> a queue. Mix gate types per slice; most plans use 2-3 of the 4.

## The four gates

### advisory

**Stakes:** low. The slice can ship without the human's input; the question exists for
audit / future record / FYI.

**Behavior:** the slice runs. The question is logged to `questions.md` with
`gate: advisory` and an explanation. `state.md` does **not** flip to `awaiting_human`.
`/rite-status` surfaces the count but does not flag it as blocking.

**Example:** "We picked option A from the prototype's verdict but option B is also viable.
Recording the choice for posterity."

**SLA:** `none` — there is no waiting time because nothing is waiting.

### validating

**Stakes:** medium. The slice can be built but should not merge before a human signs off.
Async — the human reviews when they get to it, but the loop does not stall.

**Behavior:** in HITL mode, `/rite-build` pauses on this gate and writes
`Awaiting human` to `state.md`. In AFK mode with `allow_gates: [advisory, validating]`,
`/rite-build` builds the slice but marks it `built (pending review)` and writes the
validating question; the slice does not advance to `proven` until resolved.

**Example:** "Schema migration adds a non-null column with a default. Backfill plan is
recorded; reviewer should confirm the default is the right one for archived rows."

**SLA:** `4h` — the work continues but the validating queue should drain within hours,
not days.

### blocking

**Stakes:** high. The slice cannot proceed safely without the answer. Synchronous —
the loop stops.

**Behavior:** **always pauses regardless of `.devrites/AFK` config.** `/rite-build`
writes `Awaiting human`, sets `Status: awaiting_human`, fires the `notify:` hook, and
STOPs. The slice is not built until `/rite-resolve` lands.

**Examples:**
- Destructive migration (data loss risk).
- Auth/authz boundary change.
- Public API break.
- Spec drift that changes acceptance criteria.
- Tests / types / lint are red and the agent cannot tell whether the slice's contract is
  wrong or the failing code is.

**SLA:** `15m` — synchronous gates demand fast turnaround; otherwise treat the work as
genuinely blocked and re-plan around it.

### escalating

**Stakes:** novel pattern. The question is not within the active reviewer's scope and
needs to route to a specialist (legal, security, principal engineer, designer-of-record).

**Behavior:** same pause behavior as `blocking`, but the `questions.md` entry includes a
`route:` field naming the specialist tag. `/rite-status` shows it under a separate
"Escalating" line so it doesn't compete with synchronous blockers for the same reviewer.

**Example:** "Slice introduces a contract with an external partner — needs legal review
of the data-sharing language."

**SLA:** `24h` — specialist routing implies the SLA is loose; if it needs to be tight,
it's actually `blocking`.

## Picking the gate

Apply this decision tree per HITL slice:

1. **Can the slice ship safely without the answer?**
   - Yes → `advisory`.
   - No → continue.
2. **Does the slice need a different reviewer than the default one?**
   - Yes → `escalating`.
   - No → continue.
3. **Can the slice be built but not merged until reviewed?**
   - Yes → `validating`.
   - No → `blocking`.

## SLA mapping

| Gate | SLA | Synchronous? |
|---|---|---|
| advisory | `none` | — (does not pause) |
| validating | `4h` | no (async; build continues, merge blocks) |
| blocking | `15m` | yes |
| escalating | `24h` | yes, but to a specialist |

SLAs are guidance for human reviewers and for tools that surface stale questions. DevRites
itself does not enforce them; `/rite-status` reports time since `raised_at` so the user
can see when a gate is slipping.

## AFK interaction

`.devrites/AFK` carries an `allow_gates:` list. AFK auto-handles a gate by logging the
question as advisory and proceeding **only when** the gate is in `allow_gates`. The
defaults and the always-pause rules:

| AFK `allow_gates` | advisory | validating | blocking | escalating |
|---|---|---|---|---|
| `[]` (or omitted) | log + proceed | pause | pause | pause |
| `[advisory]` (default) | log + proceed | pause | pause | pause |
| `[advisory, validating]` | log + proceed | build + queue | pause | pause |
| `[advisory, validating, blocking]` | log + proceed | build + queue | log + proceed* | pause |

\* but **never** for destructive migrations, auth/authz boundary changes, public API
breaks, or red tests/types/lint — those always pause. See
[`pack/.claude/rules/afk-hitl.md`](../../../rules/afk-hitl.md) for the irreversible-risk
list.

`escalating` is never in `allow_gates` — specialist routing is not something AFK can
shortcut.

## Anti-patterns

- **One gate for everything.** Validates becomes a queue, blocks all work behind one
  reviewer. Pick gates per slice.
- **Marking a destructive migration `validating` to "keep the loop moving".** Destructive
  work is `blocking` regardless of the urge to ship.
- **`advisory` as a synonym for "I'm not sure but I don't want to ask".** If the slice
  needs the answer, it's not advisory. Pick the right gate instead.
- **`escalating` with no `route:` field.** Without a specialist tag, an escalation is
  just a slow blocker. Name who answers.

## Field shape in `tasks.md`

```markdown
## Slice 03: list endpoint
Mode: HITL
Gate: blocking
SLA: 15m
Checkpoint: Confirm (user_id, created_at) composite index choice vs two single-col indexes.
Blocked by: Slice 02
...
```

`Gate`, `SLA`, and `Checkpoint` are **required** when `Mode: HITL`. The plan readiness
gate in `plan-template.md` enforces this.

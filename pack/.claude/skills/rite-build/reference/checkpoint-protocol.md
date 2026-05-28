# Checkpoint protocol — what `/rite-build` does when a slice is HITL

When `/rite-build` reaches a slice with `Mode: HITL`, it does **not** start writing code.
It pauses pre-action, records the checkpoint to the workspace, optionally fires the
`notify:` hook, and stops. Resume is via `/rite-resolve`.

The pause is **pre-action**, not post-action: code never lands before the gate.

## Render contract

The checkpoint must be rendered in user-facing output **and** persisted to the workspace.
Both are required — the output is for the human in the room (or the notification target),
the persisted form is for the next session or the AFK observer.

### User-facing render

```
Slice <N — name> is HITL (<gate>, SLA <SLA>).
Checkpoint: <Checkpoint: text from tasks.md>

Proposed approach: <one-paragraph best-guess so the human has something to react to>

Why HITL: <one line from tasks.md or the doubt loop verdict>

Decision needed before this slice can build.
Resume: /rite-resolve <qid> "<answer>"
       (or  /rite-resolve --drop <qid> "<reason>"  if the question is obsolete)
```

The "Proposed approach" line is **required**. A checkpoint without a tentative answer is
a worse interrupt than one that has shape — the human reacts to a draft faster than to
a blank prompt. Mastra's "give the human something to approve" rule.

### Workspace mutations

In one atomic write (one `state.md` rewrite + one `questions.md` append):

1. **`questions.md` append:**

   ```markdown
   ## q-<YYYY-MM-DD>-<NNN>
   status: open
   slice: <N — name>
   gate: <gate from tasks.md>
   question: <Checkpoint text from tasks.md>
   proposed: <one-paragraph best-guess; matches the user-facing render>
   raised_at: <iso>
   ```

2. **`state.md` updates:**

   ```markdown
   - Status: awaiting_human
   - Active slice: <N — name>
   - Slice mode: HITL
   - Next step: /rite-resolve <qid> "<answer>"

   ## Awaiting human
   - qid: <q-...-NNN>
   - gate: <gate>
   - question: <Checkpoint text>
   - proposed: <one-paragraph best-guess>
   - raised_at: <iso>
   - blocking_slices: [<list, from `Blocked by` lookups>]
   ```

3. **`notify:` hook (if `.devrites/AFK` defines one):**

   ```bash
   DEVRITES_QID="$qid" \
   DEVRITES_GATE="$gate" \
   DEVRITES_SLICE="$slice_id" \
   DEVRITES_SLUG="$slug" \
   sh -c "$notify_cmd"
   ```

   The hook fires after the workspace write so the notification target sees a workspace
   that already records the pause. Failures in the hook **do not** roll back the pause —
   the gate is authoritative; the notification is best-effort.

## qid generation

Format: `q-YYYY-MM-DD-NNN`, where `NNN` is the next sequential integer for that date in
`questions.md`. Implementation: count existing `## q-YYYY-MM-DD-` headers in the file,
add 1, zero-pad to 3 digits. The script-level helper lives in
`scripts/load-state.sh` (read side) and `rite-resolve/scripts/resolve.sh` (write side).

## When AFK is active

If `.devrites/AFK` exists and the slice's `Gate` is in `allow_gates`, `/rite-build` does
**not** invoke the checkpoint protocol. Instead:

- For `advisory`: log a `gate: advisory` entry to `questions.md`, record the trade-off in
  `decisions.md`, and continue building the slice.
- For `validating` (only when `allow_gates` includes it): build the slice, write a
  `gate: validating` entry to `questions.md`, mark the slice `built (pending review)` in
  `state.md`, and continue. The slice does not advance to `proven` until
  `/rite-resolve` lands.

For `blocking` and `escalating`, AFK **always** invokes the checkpoint protocol — the
sentinel does not unlock these gates. See `afk-discipline.md` for the irreversible-risk
list that always pauses regardless of `Gate`.

## Multi-question pauses

The current protocol is **one question per pause**. If a slice has multiple HITL
checkpoints, split it into sub-slices via `/rite-plan reslice` so each pause is
single-question. Multi-question pauses are reserved future shape; `Awaiting human` is
written as a single block.

## What NOT to do

- **Don't write code first and pause after.** The pre-action rule is the whole point.
- **Don't render the checkpoint without persisting.** Output without `state.md` + `questions.md`
  updates means the workspace lies on `/clear`.
- **Don't auto-answer with the `proposed:` text on resume.** `/rite-resolve` requires an
  explicit `<answer>` argument; the proposal is for the human to react to, not for the
  agent to confirm itself.
- **Don't bundle the `notify:` hook output into chat.** Fire-and-forget; the chat already
  has the user-facing render.
- **Don't fire `notify:` on `advisory`-downgraded entries.** It's reserved for true pauses.

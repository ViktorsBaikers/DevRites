# Checkpoint protocol — what `$rite-build` does when a slice is HITL

When `$rite-build` reaches a slice with `Mode: HITL`, it does **not** start writing code. It
surfaces the checkpoint as a ranked **option set** and resolves it **before** any code lands:

- **Human present (interactive)** — ask inline via `AskUserQuestion` (the option set below).
  The human picks; record the pick to `questions.md` (`answered`) + `decisions.md`, clear the
  gate, and **continue building in place** — no STOP, no `$rite-resolve` round-trip.
- **Human absent / AFK-pausing / notify-only** — persist the checkpoint (`questions.md` open +
  `state.md` `Awaiting human`), fire the `notify:` hook, and **stop**. Resume later via
  `$rite-resolve` (or `--batch`).

Either way the pause is **pre-action**, not post-action: code never lands before the gate.

## Render contract

The checkpoint must be rendered in user-facing output **and** persisted to the workspace.
Both are required — the output is for the human in the room (or the notification target),
the persisted form is for the next session or the AFK observer.

### User-facing render — the option set

Present the checkpoint as an `AskUserQuestion` with a ranked **option set**
(`afk-hitl.md` → "Option set"): 2–4 options, **recommended first** + labelled `(Recommended)`,
each option's description carrying the dimension-tagged rationale + the trade-off it accepts,
plus the escape hatch. The header names the slice + gate:

```
Slice <N — name> — HITL (<gate>, SLA <SLA>). <Checkpoint text from tasks.md>

▸ 1. <recommended>  (Recommended)
     logic: … · infra: … · business: … · architecture: …   (+ security/UX/risk if in scope)
  2. <alternative> — <rationale + the trade-off it accepts>
  3. <alternative> — <rationale>
  4. Something else — I'll describe it
```

The recommended option (#1) is **required** — a checkpoint without a recommendation is a worse
interrupt than one with shape; the human reacts to a ranked draft faster than to a blank prompt
(the "give the human something to approve" rule). The recommendation reflects *this* project
(its conventions, stack, scale), not a generic default. On an interactive pick, resolve in
place; when persisting a pause instead, the same set is written to `questions.md` `options:`
and the resume line is `$rite-resolve <qid> "<answer>"` (or `--drop <qid> "<reason>"`).

### Workspace mutations

In one atomic write (one `state.md` rewrite + one `questions.md` append):

1. **`questions.md` append:**

   ```markdown
   ## q-<YYYY-MM-DD>-<NNN>
   status: open
   slice: <N — name>
   gate: <gate from tasks.md>
   question: <Checkpoint text from tasks.md>
   options: |                        # the ranked set, recommended first (matches the render)
     1. <recommended> (Recommended) — logic: … · infra: … · business: … · architecture: …
     2. <alternative> — <rationale>
   proposed: <the recommended option restated; matches option 1>
   raised_at: <iso>
   ```

2. **`state.md` updates:**

   ```markdown
   - Status: awaiting_human
   - Active slice: <N — name>
   - Slice mode: HITL
   - Next step: $rite-resolve <qid> "<answer>"

   ## Awaiting human
   - qid: <q-...-NNN>
   - gate: <gate>
   - question: <Checkpoint text>
   - proposed: <one-paragraph best-guess>
   - raised_at: <iso>
   - blocking_slices: [<list, from `Blocked by` lookups>]
   ```

3. **`notify:` hook (if `.devrites/AFK` defines one):** export all six env vars from the
   canonical contract in [`afk-discipline.md`](afk-discipline.md) (the `notify:` hook
   contract table is the source of truth):

   ```bash
   DEVRITES_QID="$qid" \
   DEVRITES_GATE="$gate" \
   DEVRITES_SLICE="$slice_id" \
   DEVRITES_SLUG="$slug" \
   DEVRITES_QUESTION="$question" \
   DEVRITES_PROPOSED="$proposed" \
   sh -c "$notify_cmd"
   ```

   The hook fires after the workspace write so the notification target sees a workspace
   that already records the pause. Failures in the hook **do not** roll back the pause —
   the gate is authoritative; the notification is best-effort.

## qid generation

Format: `q-YYYY-MM-DD-NNN`, where `NNN` is the next sequential integer for that date in
`questions.md`. For the write side, call
`devrites-engine resolve next-qid <questions.md path>` — it
counts existing `## q-YYYY-MM-DD-` headers for today, prints the next zero-padded id, and
refuses to print an id whose header already exists. Use that id verbatim so each qid is
unique within the date.

## When AFK is active

If `.devrites/AFK` exists and the slice's `Gate` is in `allow_gates`, `$rite-build` does
**not** invoke the checkpoint protocol. Instead:

- For `advisory`: log a `gate: advisory` entry to `questions.md`, record the trade-off in
  `decisions.md`, and **dispatch the wright** to build the slice (workflow step 3).
- For `validating` (only when `allow_gates` includes it): **dispatch the wright** (step 3); on
  return, write a `gate: validating` entry to `questions.md`, mark the slice
  `built (pending review)` in `state.md`, and continue. A slice's only states are `pending` and `built` —
  acceptance is proven at the **feature** level by `$rite-prove` (recorded in
  `evidence.md`), not per slice. The `built (pending review)` slice is not done until the
  open `validating` gate resolves via `$rite-resolve`; an open `validating` gate is a
  NO-GO at seal.

For gates in `allow_gates`, AFK **auto-picks the recommended option** (option 1 of the set)
instead of pausing, recording it as above. For `blocking` and `escalating` (and every
irreversible-risk item), AFK **always** invokes the checkpoint protocol — the sentinel does
not unlock these gates — **unless `allow_irreversible: true`** is set, which makes AFK
auto-pick the recommendation on *every* gate, irreversible included (maximal autonomy, opt-in;
see `afk-hitl.md`). See `afk-discipline.md` for the irreversible-risk list.

## Multi-question pauses

The current protocol is **one question per pause**. If a slice has multiple HITL
checkpoints, split it into sub-slices via `$rite-plan reslice` so each pause is
single-question. Multi-question pauses are reserved future shape; `Awaiting human` is
written as a single block.

## What NOT to do

- **Don't write code first and pause after.** The pre-action rule is the whole point.
- **Don't render the checkpoint without persisting.** Output without `state.md` + `questions.md`
  updates means the workspace lies on `/clear`.
- **Don't self-answer a question that *paused* for a human.** When a gate stopped the session
  (an AFK queue, or a HITL pause the human walked away from), `$rite-resolve` requires an
  explicit answer — the agent doesn't confirm its own `proposed:` on resume. This is distinct
  from the two legitimate auto-resolutions: an interactive `AskUserQuestion` pick the human
  just made, and an AFK auto-pick of the recommended option on a gate `allow_gates` permits.
- **Don't bundle the `notify:` hook output into chat.** Fire-and-forget; the chat already
  has the user-facing render.
- **Don't fire `notify:` on `advisory`-downgraded entries.** It's reserved for true pauses.

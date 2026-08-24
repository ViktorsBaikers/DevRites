# Checkpoint protocol: what `$rite-build` does when a slice is HITL

HITL slices pause **pre-action** as a ranked **option set** before any code:

- **Interactive:** `AskUserQuestion`; record to `questions.md`/`decisions.md`; clear gate;
  continue in place (no `$rite-resolve` round-trip).
- **Absent / AFK / notify-only:** persist open question + `Awaiting human`, notify, **stop**.
  Resume via `$rite-resolve` (or `--batch`).

## Render contract

Render the checkpoint in user-facing output **and** persist it for the next session / AFK
observer.

### User-facing render: the option set

Present as `AskUserQuestion` ranked **option set** (`afk-hitl.md`): 2–4 options,
recommended first + `(Recommended)`, dimension-tagged trade-offs, plus escape hatch.
Header names slice + gate:

```
Slice <N — name> — HITL (<gate>, SLA <SLA>). <Checkpoint text from tasks.md>

▸ 1. <recommended>  (Recommended)
     logic: … · infra: … · business: … · architecture: …   (+ security/UX/risk if in scope)
  2. <alternative> — <rationale + the trade-off it accepts>
  3. <alternative> — <rationale>
  4. Something else — I'll describe it
```

Option #1 (recommended) is **required** and must reflect *this* project. Interactive pick →
resolve in place; pause → persist the same set to `questions.md` `options:` with resume
`$rite-resolve <qid> "<answer>"` (or `--drop <qid> "<reason>"`).

### Workspace mutations

In one bounded root-owned update (one `questions.md` append and one `state.md`
rewrite; do not claim cross-file atomicity):

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

   Fire after workspace write. Hook failure does **not** roll back the pause (best-effort).

## qid generation

Format: `q-YYYY-MM-DD-NNN`, where `NNN` is the next sequential integer for that date in
`questions.md`. The controlling root must scan every question header matching
today's prefix, collect its numeric suffixes, and select one above the highest
observed suffix (or `001` when none exist), advancing until it is unused. Then
re-read `questions.md` immediately before the append and repeat the scan. Append only if
the same candidate is still the next unused id; otherwise recompute. There is no
reservation or engine command for qids.

## When AFK is active

If `.devrites/AFK` exists and the slice `Gate` is in `allow_gates`, skip this protocol and
auto-pick recommended option 1:

- `advisory`: log `gate: advisory` + `decisions.md`, then dispatch wright.
- `validating` (only if allowed): dispatch wright; on return log `gate: validating`, mark
  `built (pending review)`, continue. Slice states remain `pending|built`; feature
  acceptance is `$rite-prove`. Open `validating` is NO-GO at seal until `$rite-resolve`.

`blocking` / `escalating` / irreversible-risk always use this protocol (`afk-discipline.md`).

## Multi-question pauses

**One question per pause.** Multiple HITL checkpoints → `$rite-plan reslice`.

## What NOT to do

- Don't write code then pause — pre-action only.
- Don't render without persisting `state.md` + `questions.md`.
- Don't self-answer a human pause via `proposed:` — `$rite-resolve` needs an explicit
  answer (distinct from interactive pick / AFK auto-pick on `allow_gates`).
- Don't put `notify:` output in chat; don't fire `notify:` on advisory-only entries.

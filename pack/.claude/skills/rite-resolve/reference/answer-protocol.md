# Answer protocol — how `/rite-resolve` mutates the workspace

`/rite-resolve` is the single canonical writer for `questions.md` `status` flips and
`state.md` `Awaiting human` clearance. This file is the reference for the three input
shapes, the batch file format, and the rules the underlying `scripts/resolve.sh` obeys.

## Three input shapes

### 1. Single answer

```
/rite-resolve q-2026-05-28-001 "composite index — single key is fine for now, revisit at 1M rows"
```

- The qid must be `status: open`. Re-answering is refused (see "Never overwrite").
- The answer is recorded verbatim. Use straight quotes around the answer so the shell
  preserves whitespace.

### 2. Drop

```
/rite-resolve --drop q-2026-05-28-002 "merged into q-...-003 after we narrowed the scope"
```

- A drop is **not** an answer — it removes the question from the active queue without
  recording a decision. Use it when the question is obsolete, a duplicate, or absorbed
  by a re-plan.
- The reason is required as a single trailing string. "obsolete" / "duplicate" /
  "absorbed by Slice N" are sufficient — but never blank.

### 3. Batch

```
/rite-resolve --batch .devrites/work/<slug>/answers.batch
```

The batch file is plain text, one resolution per line. Two line shapes:

```
q-2026-05-28-001: composite index, revisit at 1M rows
q-2026-05-28-002: clinician sign-off required at runtime, not at plan time
--drop q-2026-05-28-003: merged into q-...-004
# Lines starting with # are comments. Blank lines ignored.
```

- One qid per line. The first colon is the separator; the rest of the line is the value.
- Order matters only for `state.md` clearance — the script processes top-to-bottom and
  flushes `Awaiting human` after each match.
- Errors abort the batch at the first failing line so partial state is small and
  diagnosable. (Re-run after fixing the offending line; already-applied lines refuse to
  re-apply because their `status` is no longer `open`.)

## Never overwrite

A qid that is already `answered` or `dropped` refuses to be re-resolved. The audit trail
is the file — you do not edit history. If the original answer was wrong:

1. Open a new qid (the agent will, during the next `/rite-build` if the bad answer
   re-surfaces; or you write it manually).
2. Reference the old qid in the new question's body (`supersedes: q-...-001`).
3. Resolve the new qid.

The same rule applies to `--drop`: a dropped question is not undropped. Re-raise it as
a new qid if it turns out to matter after all.

## State machine

```
open ──answer──▶ answered  (terminal)
 │
 └──drop────▶ dropped   (terminal)
```

Both terminals stamp `answered_at: <iso>` and write the answer/reason to `answer:`.

## `state.md` clearance rules

`state.md`'s `Awaiting human` block is **single-question by default** — `/rite-build`
writes one block at a time and pauses. When `/rite-resolve` matches the block's qid:

- the entire `Awaiting human` block is removed (header + fields);
- `- Status: running` is set;
- a `Log` line is appended: `- <iso> build: resolved <qid>`.

If `state.md`'s `Status` is `awaiting_human` but no `Awaiting human` block matches the
qid (drift between the two files), `/rite-resolve` flags the inconsistency and refuses
to silently fix it. Use `/rite-plan repair` to reconcile.

If the answer materially changes spec/plan, **do not** edit `spec.md` or `plan.md` inside
`/rite-resolve` — that's `/rite-plan repair`'s job. The skill's post-resolve hand-off
recommends `/rite-plan repair` whenever the answer touches acceptance criteria, scope,
or architecture.

## Multi-question awaiting

If a future evolution lets `/rite-build` queue multiple questions per pause, the
`Awaiting human` block becomes a YAML list. `/rite-resolve` will:

- match the qid in the list and remove only that entry;
- leave the rest of the list intact;
- flip `Status: running` only when the list is empty.

This shape is reserved — current `/rite-build` writes one question per pause.

## Why no auto-`/rite-build` after resolve

The user types the next command explicitly. Reasons:

1. The answer may need a `/rite-plan repair` first; auto-continuing would build against
   a stale plan.
2. The user may want to `/rite-status` before resuming.
3. Atomicity: one verb, one mutation. Chaining is a hidden side-effect that hides bugs.

The output's `Next:` line is a recommendation, not an auto-run.

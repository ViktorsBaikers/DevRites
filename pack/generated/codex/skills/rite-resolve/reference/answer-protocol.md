# Answer protocol: how `$rite-resolve` mutates the workspace

`$rite-resolve` is the single canonical writer for `questions.md` `status` flips and
`state.md` `Awaiting human` clearance. This file is the reference for the three input
shapes, the batch file format, and the `devrites-engine state resolve` rules.

## Three input shapes

Each form is an explicit local mutation command. Supplying it is the confirmation; the
skill applies it once after validation and reports the result instead of asking for `y` again.

### 1. Single answer

```
$rite-resolve q-2026-05-28-001 "composite index — single key is fine for now, revisit at 1M rows"
```

- The qid must be `status: open`. Re-answering is refused (see "Never overwrite").
- The answer is recorded verbatim. Use straight quotes around the answer so the shell
  preserves whitespace.

### 2. Drop

```
$rite-resolve --drop q-2026-05-28-002 "merged into q-...-003 after we narrowed the scope"
```

- A drop is **not** an answer. It removes the question from the active queue without
  recording a decision. Use it when the question is obsolete, a duplicate, or absorbed
  by a re-plan.
- The reason is an optional single trailing string, **strongly recommended** for the
  audit trail. "obsolete" / "duplicate" / "absorbed by Slice N" are sufficient; if omitted,
  `devrites-engine state resolve` records it as `dropped`.

### 3. Batch

```
$rite-resolve --batch .devrites/work/<slug>/answers.batch
```

The batch file is plain text, one resolution per line. Two line shapes:

```
q-2026-05-28-001: composite index, revisit at 1M rows
q-2026-05-28-002: clinician sign-off required at runtime, not at plan time
--drop q-2026-05-28-003: merged into q-...-004
# Lines starting with # are comments. Blank lines ignored.
```

- One qid per line. The first colon is the separator; the rest of the line is the value.
- The batch is all-or-nothing: every line is validated before one atomic write, then
  `Awaiting human` is cleared for the resolved qids.
- An error applies nothing. Report the failing line and exit code, fix that line, and
  re-run the whole file.

## Never overwrite

A qid that is already `answered` or `dropped` refuses to be re-resolved. The audit trail
is the file. You do not edit history. If the original answer was wrong:

1. Open a new qid (the agent will, during the next `$rite-build` if the bad answer
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

`state.md`'s `Awaiting human` block is **single-question by default**: `$rite-build`
writes one block at a time and pauses. When `$rite-resolve` matches the block's qid:

- the entire `Awaiting human` block is removed (header + fields);
- `- Status: running` is set;
- a `Log` line is appended: `- <iso> <resuming phase>: resolved <qid>`.

If `state.md`'s `Status` is `awaiting_human` but no `Awaiting human` block matches the
qid (drift between the two files), `$rite-resolve` flags the inconsistency and refuses
to silently fix it. Report it to the current phase owner; once a plan exists,
reconciliation uses `$rite-plan repair`.

## Phase-aware handoff and slice context

Read `tasks.md` and update `Slice mode` only for an actual `SLICE-###` question,
never for `slice: spec | plan`; the update is allowed only after the writer exits zero.
Before planning artifacts exist, recommend the current phase owner's action emitted
by `state resolve` (for example, `$rite-spec`), not Plan repair. Never create planning
artifacts as part of Resolve.

If the answer materially changes spec/plan, **do not** edit `spec.md` or `plan.md` inside
`$rite-resolve`. Once a plan exists, material acceptance, behavior, scope, or architecture
changes route through the Spec Drift Guard (`$rite-plan repair`); otherwise retain the
emitted phase-native next action. User-invoked resolutions use `state resolve --human`;
an internal Autocomplete caller uses the same writer without `--human` and continues.
Resolve itself never auto-runs the recommended phase owner.

## Multi-question awaiting

If a future evolution lets `$rite-build` queue multiple questions per pause, the
`Awaiting human` block becomes a YAML list. `$rite-resolve` will:

- match the qid in the list and remove only that entry;
- leave the rest of the list intact;
- flip `Status: running` only when the list is empty.

This shape is reserved: current `$rite-build` writes one question per pause.

## Why no auto-`$rite-build` after resolve

The user types the next command explicitly. Reasons:

1. Before planning artifacts exist, the emitted current phase owner must resume;
   once a plan exists, material changes may need `$rite-plan repair` first.
   Auto-continuing a build would skip that owner or use a stale plan.
2. The user may want to `$rite-status` before resuming.
3. Atomicity: one verb, one mutation. Chaining is a hidden side-effect that hides bugs.

The output's `Next:` line is a recommendation, not an auto-run.

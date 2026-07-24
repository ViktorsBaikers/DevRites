# AFK discipline for unattended `/rite-build`

AFK mode is active when `.devrites/AFK` exists. It lets `/rite-build` chain slices
without per-slice user input. The rules below limit that unattended work.

These rules follow established autonomous-coding loops, including Ralph Wiggum and
Claude Code auto mode:

1. **Require green feedback.** Tests, types, and lint must pass before a slice is
   marked `built`.
2. **Cap iterations.** `max_slices` is the hard limit.
3. **Run gates before the action they control.** A post-action gate is only a review
   queue.
4. **Keep irreversible work manual.** Destructive work, auth boundaries, and public
   API breaks always pause regardless of the sentinel.

## The sentinel file

`.devrites/AFK` (presence = AFK). Optional YAML body:

```yaml
max_slices: 10                       # read-only INITIAL budget; seeds state.md on first AFK build
notify: "ntfy.sh/my-topic"           # shell command run on awaiting_human transition
allow_gates: [advisory, validating]  # gate severities AFK may auto-handle
```

`.devrites/AFK` is **read-only config**: never rewritten in place. `max_slices` is the
initial budget; the mutable remaining count lives in `state.md` as `AFK slices remaining:
<n>`, seeded from `max_slices` on the first AFK build and decremented by `devrites-engine tick-afk`
(see "Iteration cap").

Defaults when keys are omitted:
- `max_slices`: unlimited (a missing cap is risky: see "Always cap iterations").
- `notify`: none.
- `allow_gates`: `[advisory]`.

To disable AFK, delete the file. The next `/rite-build` runs in HITL.

## Iteration cap

`/rite-build`'s **record step** (workflow step 6) decrements `state.md`'s `AFK slices
remaining` by 1 each time a slice is marked `built`, by running
`devrites-engine tick-afk <state.md path>`. The script reads the
field, decrements, writes it back, prints the new value, and **exits `3` when it hits 0**.
The cap is enforced by `devrites-engine tick-afk`, not by prose, when it exits 3:

- `/rite-build` treats exit 3 as a forced HITL stop:
  ```
  AFK cap reached. Raise `state.md` `AFK slices remaining` or remove the sentinel to continue.
  ```
- The workspace stays consistent: no half-built slice, no pending question.

Step 0 re-derives the remaining budget from `state.md` (seeding it from `.devrites/AFK`
`max_slices` on the first AFK build); `max_slices` itself is read-only and never rewritten.

Choose a missing or large cap deliberately. Ralph's rule is 5-10 iterations for small
tasks and 30-50 for larger ones. Do not use `unlimited` for work that has not completed
successfully in HITL.

## Fail-on-red

The **fail-on-red step** (workflow step 5) refuses to mark a slice `built` if targeted
tests, types, or lint are red:

- A red signal means either the slice's contract is wrong or the implementation/proof path is.
  The slice cannot advance, but an objective root cause is agent-owned recovery work.
- Marking it `built` would let the next slice build on broken state.

The fail-on-red path:

1. Continue the same wright under `devrites-debug-recovery`, carrying exact output and dead
   ends; cap writer + recovery at three total attempts per root cause.
2. Green → record the slice. Product-contract/irreversible ambiguity → write the genuine
   human gate. Missing human-only credential/permission → write a human-intervention gate.
3. Any other exhausted objective failure → set `Status: blocked`, preserve the reproduction,
   set `Next step: /rite-plan unblock`, and STOP without a qid.
4. Fire `notify:` only for an actual `awaiting_human` transition.

AFK never starts the next slice while checks are red and never asks the human to approve
agent-owned diagnosis or parser/test repair.

## Irreversible-risk list (always pause)

Regardless of `allow_gates`, AFK invokes the checkpoint protocol when the slice touches
any of:

- Destructive data migration (drop column, drop table, irreversible backfill).
- Auth / authz boundary change (new role, changed permission check, new public auth endpoint).
- Public API break (response shape change, removed endpoint, changed status code semantics).
- External-service contract change (webhook payload, partner-facing schema).
- Filesystem destructive operation outside the workspace (`rm -rf` of project paths,
  rewriting `.gitignore`-listed paths, deleting fixtures).
- Anything the slice's `Gate: blocking` plus `tasks.md` `Why HITL:` flags as irreversible.

This is the Claude Code auto-mode transcript classifier list adapted to the DevRites
workspace.

## The `notify:` hook contract

The hook is a single shell command run on the `awaiting_human` transition. Environment
the hook receives:

| Var | Value |
|---|---|
| `DEVRITES_QID` | the new qid (e.g. `q-2026-05-28-001`) |
| `DEVRITES_GATE` | `advisory` / `validating` / `blocking` / `escalating` |
| `DEVRITES_SLICE` | `<N — name>` |
| `DEVRITES_SLUG` | active feature slug |
| `DEVRITES_QUESTION` | the checkpoint text |
| `DEVRITES_PROPOSED` | the proposed answer |

The hook is best effort: a non-zero exit does **not** roll back the pause. Failures are
logged to `evidence.md` so the user sees them on return.

Example targets:
- `curl -d "$DEVRITES_QID: $DEVRITES_QUESTION" ntfy.sh/my-topic`
- `osascript -e "display notification \"$DEVRITES_QUESTION\" with title \"DevRites: $DEVRITES_GATE\""`
- `pb push "$DEVRITES_SLUG: $DEVRITES_QUESTION"` (via pushbullet CLI)

DevRites does not implement notifications; the configured hook does.

## When to leave AFK

Drop the sentinel before:

- A new feature when `/rite-build` has not yet completed successfully on this codebase
  in HITL. Ralph's progression is HITL first → refine prompt → AFK after validation.
- A risky slice you marked `Mode: HITL` and want to walk through interactively even if
  the gate is technically `validating`.
- Any time you'd rather review per-slice than batch-resolve afterwards.

AFK allows routine work to continue automatically. Re-enable it for another batch of
low-stakes work.

## What AFK does NOT do

- It does not bypass `/rite-prove`, `/rite-review`, or `/rite-seal`. Those gates are
  feature-scoped and always run when their phase runs.
- It does not skip `/rite-plan repair` when the Spec Drift Guard finds that the
  durable plan is wrong. Objective implementation and tool failures stay in
  bounded recovery instead.
- It does not skip `devrites-source-driven` checks. Uncertain framework behavior still
  triggers the doc lookup.
- It does not skip `evidence.md` writes. AFK runs that don't record evidence are
  unproven and get rejected at `/rite-prove` regardless.

The sentinel widens *human pauses*, nothing else.

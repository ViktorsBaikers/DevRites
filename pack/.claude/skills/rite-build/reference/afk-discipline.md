# AFK discipline — running `/rite-build` unattended without burning the trunk

AFK mode is `.devrites/AFK` present. It lets `/rite-build` chain slices without per-slice
user input. The discipline below is what keeps an AFK loop from drifting into damage.

The core principles are borrowed from established autonomous-coding loops (Ralph Wiggum,
Claude Code auto mode):

1. **Feedback loops are the trust substrate.** No green tests / types / lint → no
   "built" status. The loop can't declare victory if the lights are red.
2. **Always cap iterations.** Stochastic systems + infinite loops = unsafe. `max_slices`
   is the hard counter.
3. **Promote pre-action gates, demote post-action gates.** Code that lands without a
   gate is a finding; gates after the fact are review queues.
4. **AFK widens what's automatic, never what's irreversible.** Destructive work, auth
   boundaries, public API breaks always pause regardless of the sentinel.

## The sentinel file

`.devrites/AFK` (presence = AFK). Optional YAML body:

```yaml
max_slices: 10                       # read-only INITIAL budget; seeds state.md on first AFK build
notify: "ntfy.sh/my-topic"           # shell command run on awaiting_human transition
allow_gates: [advisory, validating]  # gate severities AFK may auto-handle
```

`.devrites/AFK` is **read-only config** — never rewritten in place. `max_slices` is the
initial budget; the mutable remaining count lives in `state.md` as `AFK slices remaining:
<n>`, seeded from `max_slices` on the first AFK build and decremented by `tick-afk.sh`
(see "Iteration cap").

Defaults when keys are omitted:
- `max_slices`: unlimited (a missing cap is risky — see "Always cap iterations").
- `notify`: none.
- `allow_gates`: `[advisory]`.

To disable AFK temporarily, delete the file. The next `/rite-build` boots straight back
into HITL.

## Iteration cap

`/rite-build`'s **record step** (workflow step 6) decrements `state.md`'s `AFK slices
remaining` by 1 each time a slice is marked `built`, by running
`bash .claude/skills/devrites-lib/scripts/tick-afk.sh <state.md path>`. The script reads the
field, decrements, writes it back, prints the new value, and **exits `3` when it hits 0**.
The cap is enforced by `tick-afk.sh`, not by prose — when it exits 3:

- `/rite-build` treats exit 3 as a forced HITL stop:
  ```
  AFK cap reached. Raise `state.md` `AFK slices remaining` or remove the sentinel to continue.
  ```
- The workspace stays consistent — no half-built slice, no pending question.

Step 0 re-derives the remaining budget from `state.md` (seeding it from `.devrites/AFK`
`max_slices` on the first AFK build); `max_slices` itself is read-only and never rewritten.

The cap is intentional: a missing or large cap **must be a conscious choice**. Ralph's
rule: 5-10 iterations for small tasks, 30-50 for larger ones. Don't ship "unlimited" as
the default for a job you haven't observed running once HITL.

## Fail-on-red

The **fail-on-red step** (workflow step 5) refuses to mark a slice `built` if targeted tests /
types / lint are red. The reasoning:

- A red signal means either the slice's contract is wrong or the failing code is. Neither
  is something AFK can resolve.
- Marking it `built` and letting the AFK loop chain to the next slice burns the trunk —
  the next slice builds on broken state.

The fail-on-red path:

1. Append a question to `questions.md` with `gate: blocking`, the SLA of the slice, and a
   crisp `question:` field naming what failed.
2. Set `state.md` `Status: awaiting_human` and the `Awaiting human` block.
3. Fire the `notify:` hook if defined.
4. STOP.

The user resolves via `/rite-resolve` after diagnosing or re-planning.

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

The list is the same one Claude Code auto-mode's transcript classifier protects — adapted
to DevRites's workspace shape.

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

The hook is best-effort: non-zero exit does **not** roll back the pause. Failures are
logged to `evidence.md` so the user sees them on return.

Example targets:
- `curl -d "$DEVRITES_QID: $DEVRITES_QUESTION" ntfy.sh/my-topic`
- `osascript -e "display notification \"$DEVRITES_QUESTION\" with title \"DevRites: $DEVRITES_GATE\""`
- `pb push "$DEVRITES_SLUG: $DEVRITES_QUESTION"` (via pushbullet CLI)

DevRites owns no notification logic — the hook is a seam, not a feature.

## When to leave AFK

Drop the sentinel before:

- A novel feature where you have not yet seen `/rite-build` work on this codebase HITL —
  Ralph's progression: HITL first → refine prompt → AFK once trusted.
- A risky slice you marked `Mode: HITL` and want to walk through interactively even if
  the gate is technically `validating`.
- Any time you'd rather review per-slice than batch-resolve afterwards.

AFK is a **bias toward continuing**, not a vow. Re-enter it whenever the next stretch of
work is bulk + low-stakes.

## What AFK does NOT do

- It does not bypass `/rite-prove`, `/rite-review`, or `/rite-seal`. Those gates are
  feature-scoped and always run when their phase runs.
- It does not skip `/rite-plan repair` on Spec Drift Guard fires.
- It does not skip `devrites-source-driven` checks. Uncertain framework behavior still
  triggers the doc lookup.
- It does not skip `evidence.md` writes. AFK runs that don't record evidence are
  unproven and get rejected at `/rite-prove` regardless.

The sentinel widens *human pauses*, nothing else.

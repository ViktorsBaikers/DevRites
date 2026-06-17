# Stop conditions — when autocomplete must pause for a human

Autonomy covers the routine path. These are the cases where autocomplete writes
`state.md` (`Status: awaiting_human` or `blocked`), surfaces *why* + the single resume
command, fires `notify` if set, and **stops**. None are bypassable by `--ship`.

## Always stop (irreversible-risk list — from `afk-hitl.md`)

Regardless of `allow_gates` or `--ship`:
- Destructive data migration (drop column/table, irreversible backfill).
- Auth / authz boundary change.
- Public-API break (response shape, removed endpoint, changed status semantics).
- External-service contract change.
- Filesystem destruction outside the workspace.
- Red tests / types / lint at slice end (fail-on-red).

## Stop on gate severity

- `blocking` gate fires → synchronous pause.
- `escalating` gate fires → pause, route to the specialist tag.
- Any `questions.md` entry with `gate: validating` and `status: open` → pause (it is a
  seal NO-GO by definition; don't sail into a guaranteed NO-GO).

## Stop on workflow state

- **NO-GO at seal** → stop; surface every blocker with `file:line` and the fix
  direction. Do not round NO-GO up to GO.
- **Spec Drift Guard fires** (`/rite-build` finds the plan is wrong and the change
  alters product behaviour) → stop; route through `/rite-plan repair`.
- **`max_slices` exhausted** (`tick-afk.sh` exit 3) → stop; report slices remaining.
- **Still low-confidence after the interview** — the idea can't be pinned to testable
  acceptance criteria → stop and ask, rather than guessing the product.
- **Repeated failure** — a phase fails, `devrites-debug-recovery` can't fix it within
  scope after a bounded number of attempts → stop with the reproduction.

## The final type-GO

- Default: render the type-GO prompt at `/rite-ship` and **stop** for the human's
  literal `GO`.
- With `--ship` / `--yolo`: auto-confirm the type-GO (the *only* thing the flag changes).
  Everything above still pauses.

## How to stop well

State must be enough for a fresh agent to resume cold:
- `state.md`: `Status`, the blocking reason, and a single `Next step` command.
- The relevant `questions.md` / `drift.md` / `seal.md` entry that explains the pause.
- A one-line user-facing message: *what* stopped it and *what command* resumes.

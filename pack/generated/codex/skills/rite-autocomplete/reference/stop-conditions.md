# Stop conditions: when autocomplete must pause for a human

In the cases below, autocomplete writes `state.md` (`Status: awaiting_human` or
`blocked`), reports the reason and one resume command, fires `notify` if set, and
**stops**. `--ship` cannot bypass them.

## Always stop (irreversible-risk list: from `afk-hitl.md`)

Regardless of `allow_gates` or `--ship`:
- Destructive data migration (drop column/table, irreversible backfill).
- Auth / authz boundary change.
- Public-API break (response shape, removed endpoint, changed status semantics).
- External-service contract change.
- Filesystem destruction outside the workspace.

Red tests/types/lint/build/browser proof are hard non-advance gates, not automatically human
gates. Run bounded `devrites-debug-recovery`; if it exhausts, stop as a technical blocker unless
the remaining issue is human-owned.

## Stop on gate severity

- `blocking` gate fires → synchronous pause.
- `escalating` gate fires → pause, route to the specialist tag.
- Any `questions.md` entry with `gate: validating` and `status: open` → pause (it is a
  seal NO-GO by definition; stop before reaching seal).

## Stop on strategic-review scope expansion (`$rite-temper`)

- `$rite-temper` choosing scope mode `expand`, or otherwise **adding** any acceptance
  criterion to `spec.md`: → synchronous pause, regardless of `allow_gates` / `--ship`.
  Autocomplete may auto-apply only `hold-rigor` and `reduce-to-MVP` (they harden / prune and
  never grow the build's scope); **growing scope unattended is never automatic**. A low-stakes
  spec that temper skips, or a `hold-rigor` / `reduce-to-MVP` run, does **not** pause.

## Stop on incomplete decision coverage (`$rite-clarify`)

- Stop on any material Partial/Missing/unowned row or low-confidence, high-consequence
  assumption. Continue the up-front window with the next genuine decision packet; never arm
  AFK or carry it into build. Resolve facts and reversible choices automatically.

## Stop on workflow state

- **NO-GO at seal** → stop; surface every blocker with `file:line` and the fix
  direction. Do not round NO-GO up to GO.
- **Spec Drift Guard fires** (`$rite-build` finds the plan is wrong and the change
  alters product behaviour) → stop; route through `$rite-plan repair`.
- **Budget exhausted with slices still pending:** `devrites-engine tick-afk` exit 3 while `tasks.md`
  still has unbuilt slices (only when an explicit `--max-slices` capped the run below the
  plan's count) → stop; report slices remaining. Exhausting the default budget = all
  planned slices built = normal completion → continue to `$rite-prove` without pausing.
- **Still low-confidence after the interview:** the idea can't be pinned to testable
  acceptance criteria → stop and ask, rather than guessing the product.
- **Repeated failure:** a phase fails, `devrites-debug-recovery` can't fix it within
  scope after a bounded number of attempts → stop with the reproduction.

## The final type-GO

- Default: render the type-GO prompt at `$rite-ship` and **stop** for the human's
  literal `GO`.
- With `--ship` / `--yolo`: auto-confirm the type-GO (the *only* thing the flag changes).
  Everything above still pauses.

## How to stop well

State must be enough for a fresh agent to resume cold:
- `state.md`: `Status`, the blocking reason, and a single `Next step` command.
- The relevant `questions.md` / `drift.md` / `seal.md` entry that explains the pause.
- A one-line user-facing message: *what* stopped it and *what command* resumes.

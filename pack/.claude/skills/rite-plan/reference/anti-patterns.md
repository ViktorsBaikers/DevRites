# rite-plan — anti-patterns

Load this when standing a non-trivial re-plan decision, or when the agent
feels reluctance toward asking the user before a behavior change.

Pack-wide rationalizations + red flags: see [rules/anti-patterns.md](../../../rules/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "User didn't ask for a re-plan." | A Spec Drift Guard event *is* a re-plan trigger — ask is courtesy, the route is automatic. |
| "I can fix the slice mid-build, no need to update tasks." | `tasks.md` is the contract `/rite-build` reads. A fix that isn't there isn't visible to the next phase. |
| "Reordering slices is cosmetic." | Dependency order controls risk discovery; risk-first wins because it surfaces blockers early. |
| "This drift is small enough to absorb silently." | Silent drift = unrecorded decision = future "why does this work?" debugging session. |
| "I can change product behavior to make the plan fit." | Behavior change always asks the user first, no exceptions. |

## Red Flags

- About to change product behavior, acceptance, or scope without asking the user.
- `drift.md` entry unresolved while planning the next slice.
- A "mode" invented that isn't one of `decompose / reslice / repair / reorder / split / unblock`.
- Slices reordered without updating dependency notes — order changed, rationale didn't.
- Repairing a drift event by silently widening the spec rather than narrowing the plan.

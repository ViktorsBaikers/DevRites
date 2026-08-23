# rite-plan: anti-patterns

Load this when standing a non-trivial re-plan decision, or when the agent
feels reluctance toward asking the user before a behavior change.

Pack-wide rationalizations + red flags: see [standards/anti-patterns.md](../../devrites-lib/reference/standards/anti-patterns.md).

Authority: `.claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → reconcile `architecture.md`, `plan.md`, `tasks.md`, and `traceability.md` atomically; invalidate Vet/readiness; Vet.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "User didn't ask for a re-plan." | A Spec Drift Guard event *is* a re-plan trigger: ask is courtesy, the route is automatic. |
| "I can fix the slice mid-build, no need to update tasks." | `tasks.md` is the contract `/rite-build` reads. A fix that isn't there isn't visible to the next phase. |
| "Reordering slices is cosmetic." | Dependency order controls risk discovery; risk-first wins because it surfaces blockers early. |
| "This drift is small enough to absorb silently." | Silent drift = unrecorded decision = future "why does this work?" debugging session. |
| "I can change product behavior to make the plan fit." | **RSLICE-PLAN-001** — Implementation convenience cannot rewrite product behavior. |
| "Topology or a small criterion delta decides the route." | **RSLICE-PLAN-002** — Topology/count and “small” never override the canonical classifier; sufficient authoritative contract delta follows its mapped action. |
| "Chat, stale material, or the convenient map is enough." | **RSLICE-PLAN-003** — Exact current byte-bound directives are authority; cached, remembered, summarized, paraphrased, or inferred chat and stale material are not. |

## Red Flags

- About to change product behavior, acceptance, or scope without asking the user.
- `drift.md` entry unresolved while planning the next slice.
- A "mode" invented that isn't one of `decompose / reslice / repair / reorder / split / unblock`.
- Slices reordered without updating dependency notes: order changed, rationale didn't.
- Repairing a drift event by silently widening the spec rather than narrowing the plan.

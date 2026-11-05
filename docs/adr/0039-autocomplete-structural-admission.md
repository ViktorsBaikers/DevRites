# ADR-0039: Structural admission for the native Autocomplete controller

- **Status:** Accepted
- **Date:** 2026-11-05

## Context

Consumer traces showed a native Autocomplete controller repeatedly completing one
workspace while leaving recorded continuation preparation unstarted. A global
frontier report alone did not prevent subsequent local writer admission. The
controller also recognized a historical workspace rename without resolving the
conflict between the old roadmap table and current sequence identities.

ADR-0022 and ADR-0024 leave semantic scheduling and model dispatch with native
hosts. Replacing them with an engine agent bridge, persistent queue, scheduling
receipts, or heuristic interpretation of reviewer prose would create a second
orchestration authority. Existing dispatch and lease primitives already retain
observable preparation accounts, source-qualified reservations and write claims.

## Decision

Narrow ADR-0022's native-scheduling boundary and ADR-0024's policy boundary only
for structural Autocomplete writer admission:

- Explicit native Autocomplete invocation records `mode: autocomplete` in AFK.
  This identifies the controller; it grants no readiness, independence or Ship
  permission. Standalone AFK without this mode retains its existing contract.
- Before admitting a new Build-wright wave at capacity greater than one, the
  engine checks immediate successor preparation and the eligible mixed cohort.
  Preparation is
  represented by an actual pending native preparation account, current native
  readiness, or a concrete retained hold. Creating a placeholder or printing a
  frontier does not satisfy this boundary.
- A source may disclose a target-specific preparation constraint through
  `frontier-blocked: <workspace> — <reason>` in its existing evidence ledger.
  `sequence` names only unresolved historical sequence normalization. These rows
  defer scheduling obligations; they never waive malformed authority, source
  readiness, trust boundaries, writer claims, path/capacity limits or proof.
- An already-ready foreign slice requires an existing mixed lease reservation
  or an exact source-qualified `serial:` rationale for ordered handoff. A local
  serial reason or preparation deferral does not discharge foreign admission.
- A fresh global observation follows a committed dispatch return after mutation
  and locks settle, even if later claim release fails; its nonzero failure and
  claim exclusion remain. Rejected/unpersisted returns do not refresh. Global
  obligations take continuation priority without rewriting local phase cursors
  or recursively routing local-only work.
- One explicit canonical `roadmap.md`, bound to the actual sequence root and a
  unique owner-local decision, may replace historical tables for scheduling.
  History stays unchanged; duplicate authorities, malformed inputs, conflicting
  current identities and invalid references fail closed. Native owners decide
  whether the decision and scope are substantively approved.
- Initial admission freezes each pending writer lane's normalized allowlist in
  existing dispatch metadata. Mutable tasks, reused batch labels or replacement
  lease rows cannot move a retained actor's authority. Exact claim identity and
  session remain required; claim release or expiry does not prove actor exit.
  Returned/integrated siblings may free slots for rolling refill.

The engine does not invoke a model, create a native workspace, infer semantic
independence, judge a rationale's truth, bind a handle to an execution checkout,
or authorize publishing. Native hosts still execute preparation, dispatch exact
profiles, reconcile results and obey human and protected-validation gates.

## Alternatives considered

| Option | Why rejected |
| --- | --- |
| More reminders and frontier reports only | The observed controller already had that guidance and still skipped preparation. |
| Automatically accept the newest table or rename prose | Recency and prose do not establish canonical identity or supersession. |
| Rewrite archived maps to match the current run | Destroys the historical decision rather than recording current authority. |
| Require every future workspace READY before any writer | Serializes preparation and creates cycles when future plans depend on current repair. |
| Engine-owned model scheduling, receipts or persistent queue | Duplicates host authority and exceeds the deterministic local boundary. |

## Consequences

Skipping required preparation now refuses an observable writer-admission step
rather than remaining a narrative warning. Independent work can continue with a
disclosed specific constraint, without disguising deferred work as completed.
Canonical normalization has one machine-readable owner and does not require
altering old evidence or guessing workspace names.

This cannot prove that an arbitrary host obeys every instruction or that a native
account represents real semantic progress. Runtime claims therefore distinguish
structural admission, recorded native launches, and actual host behavior.

Guard tests: `engine/internal/lib/frontier_controller_test.go`,
`engine/internal/lib/frontier_roadmap_test.go`, and
`engine/tests/autocomplete_global_test.go`.

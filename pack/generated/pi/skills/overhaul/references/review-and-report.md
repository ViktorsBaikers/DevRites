# Review page, report and terminal summary

## Contents

- [Views are projections](#views-are-projections)
- [The review page](#the-review-page)
- [Decisions and the approval message](#decisions-and-the-approval-message)
- [The end-of-run report](#the-end-of-run-report)
- [Terminal summary](#terminal-summary)
- [Renderer tool](#renderer-tool)

## Views are projections

`review.html`/`.md`, `report.html`/`.md` and `plan.md` (the task table and approval
message, written with a review) are rendered by the renderer tool
below from one generation's canonical JSON; never hand-write or hand-edit them. Each
carries the stamp `overhaul-stamp run=… generation=… plan=r<N>:<sha256>`, and the
validator rejects a view whose stamp does not match its generation, so a stale page
can never be applied. Human feedback on a view becomes a proposed change that the
coordinator reconciles into a new plan revision; views are then re-rendered — there
are never three competing versions of the plan.

## The review page

The page opens as a local file and works offline: one self-contained document, no
scripts, no CDNs, no remote fonts or analytics, a hash-pinned stylesheet under a
restrictive Content-Security-Policy, system font stacks, explicit light and dark
backgrounds, visible `:focus-visible` outlines, semantic tables with captions and
column headers inside keyboard-focusable scroll regions, no horizontal page scroll at
phone width, text as well as color for every status, and a print-friendly layout.
Every repository-derived string — filenames, snippets, finding titles, comments, JSON
— is HTML-escaped before formatting and never interpolated as markup or script;
absolute home paths are redacted to `~`. Never publish the page or serve it beyond
loopback without a separate user decision.

Order: summary and decision first, then evidence. Sections, each with a stable
anchor, and rows anchored by their IDs (`#F-012`, `#T-003`, `#C-040`) so feedback can
cite them:

- scope, baseline, candidate and comparison identities, capabilities, known limits,
  open questions;
- one section per lane — frontend, backend, boundary, other — with actual attempts,
  profiles, timings and receipts (no duplicate independent plans);
- the coverage ledger and stack profiles; contracts and boundaries;
- confirmed findings (location, failing case, impact, evidence, uncertainty, proposed
  action), then unvalidated leads without severity, then rejected candidates with
  reasons; opportunities stay separate from defects;
- before-state measurements with benchmark conditions, proposed minimum and stretch
  targets and regression guardrails;
- the repair plan: tasks with dependencies, exact allowed paths, oracles, migration
  and rollback needs, expected benefits, costs and alternatives, and the continuation
  envelope ([`lifecycle.md`](lifecycle.md#what-one-approval-permits));
- the rubric controls with lane membership and shared-control marking, the baseline
  global, lane and domain scorecards, hard blockers and why readiness does or does
  not qualify; frontend engineering evidence, interaction/accessibility proof and
  optional aesthetic proposals visibly separate;
- approvals and decisions as recorded events, never as page state.

The renderer derives the report tables below from the records: lane coverage
(admitted reviewers with their windows, profiles, reviewed versus eligible ranges,
open ranges, lane score), findings and repairs (finding and task, severity, previous
failure, changed paths, oracle, verification, remaining risk), and the transparent
scorecard (baseline versus candidate per scope, applicable control weight, evidence
gaps, hard blockers). Columns it cannot fill from records render as "not recorded".

When a diagram clarifies architecture, trust boundaries, data flow or the task DAG,
add a textual equivalent (a dependency table or indented list); an image never
carries information the text lacks.

## Decisions and the approval message

The user decides per item — approve, reject, defer, ask for explanation or change —
and approves one complete plan revision. The page never contains an "Approve" button:
nothing a page can do (a checkbox, localStorage, a form, an edited file, a local
server request) authenticates the user, and a script or any local process can forge
such signals. Instead the page shows a read-only, copyable message:

```text
Approve overhaul <run-id> plan revision <N>, digest <sha256>, tasks T-001 T-002 T-005.
Keep all listed constraints and exclusions.
```

The approval exists only when the user sends such a decision in the active trusted
conversation (or a genuinely authenticated host action), and the coordinator
re-hashes the plan revision and matches the digest before recording it
([`lifecycle.md`](lifecycle.md#the-approval-gate)).

## The end-of-run report

Produce `report.{md,html}` from the same generation for every ending —
interim, blocked, stopped, audit-only or final. Use real measurements only; a missing
value renders as "not recorded", never as a sample number. Zero findings is a valid
result; report counts always equal record counts.

State separately: mode, lifecycle phase, readiness verdict and execution outcome
(`AUDIT_COMPLETE`, `AUDIT_INCOMPLETE`, `AWAITING_APPROVAL`, `READY_FOR_USER_REVIEW`,
`COMPLETED_WITH_APPROVED_DEGRADATION`, `STOPPED_BY_USER`, or the exact blocked state).
`AUDIT_COMPLETE` never means the target passed. Include revisions, coverage and
exclusions, applicable stacks, authentic approvals and actual changes, the three
identities, environment limitations and remaining unknowns. Never write "every bug is
fixed", "cannot break" or "fully secure".

Required tables (column order fixed):

| Table | Columns |
| --- | --- |
| Lane, stack and boundary coverage | Lane/component · actual reviewer and overlap evidence · language/runtime/framework versions · active profile/rule revisions · reviewed vs eligible ranges · mandatory proof gaps · verdict/evidence |
| Findings and repairs | Finding/task · severity and evidence confidence · previous failure · change made · regression proof · independent result · remaining risk |
| Performance before/after (grouped frontend, backend, end-to-end) | Scenario and workload · metric/direction · before · after · absolute delta · relative change/speedup · uncertainty/repetitions · approved target · guardrails · verdict/evidence |
| Transparent scorecard | Global or lane/domain · weight · unique applicable control weight · baseline · candidate · evidence/profile gaps · hard blockers · evidence |
| Iteration history | Cycle · starting deficient lanes/controls · hypothesis or proof gap · approved tasks/agents · code changes vs evidence-only work · tests/benchmarks/independent outcome · global and affected lane/domain scores · gate changes/remaining gaps · next action or stop reason |
| Dependency decisions | Dependency · resolved before · resolved after/unchanged · keep/patch/upgrade/replace/remove · official evidence date · reason and alternatives · migration/compatibility proof · risk |

Show lanes separately — never one "full-stack checked" row — plus justified absent or
not-impacted lanes, inaccessible external components, embedded-language and
generated-source boundaries, actual scanner coverage and manual review gaps. Keep
candidate, rejected, deferred and accepted-risk counts apart from confirmed, fixed and
verified. Show full-precision global and lane results, domain floors, rubric and
profile revisions, shared-control accounting, N/A decisions, accepted risks and a
separate hard-gate verdict, and explain score deltas by control transitions (fix,
evidence-only, regression, invalidated evidence, rubric correction). Include original
versus revised performance targets with the user's recorded decision, checks run and
not run, failures proven pre-existing versus introduced, questions and decisions,
resource and cost totals (estimates labeled), rollback and deploy-order notes (without
deploying), the final changed-path summary, and remaining work.

## Terminal summary

First line: outcome, verdict and the one next action. Then the full-precision global
and lane scores, open gates, counts, the report path and the approval message when
awaiting approval. Last line: the next action again. No praise, no preamble, no
estimates presented as measurements.

## Renderer tool

`devrites-engine overhaul render <run> <staged-generation> review|report` writes the views into
`<staged-generation>/views/`, before `publish`.

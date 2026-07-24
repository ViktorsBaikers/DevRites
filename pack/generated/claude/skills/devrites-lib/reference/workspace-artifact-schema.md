# Workspace artifact schema

DevRites stores feature truth in `.devrites/work/<slug>/`. The older
`.devrites/features/<slug>/` path remains readable during migration; do not delete
existing workspaces or aliases. `feature.md` and `index.md` may stand in for the
workspace map; `proof.md` may stand in for `evidence.md`.

## Required by phase

| Phase | Required artifacts |
| --- | --- |
| frame | `state.md` |
| spec | `README.md`/`index.md`/`feature.md`, `brief.md`, `spec.md`, `state.md`, `decisions.md`, `assumptions.md`, `questions.md` |
| clarify | spec artifacts plus `decision-coverage.md` |
| temper | clarified spec artifacts |
| define/plan | clarified spec artifacts plus `architecture.md`, `plan.md`, `tasks.md`, `traceability.md` |
| vet/build/converge | plan artifacts plus `eng-review.md`, `test-plan.md` |
| prove/polish/review/seal/ship/done | vetted plan artifacts plus `evidence.md`/`proof.md`, `touched-files.md` |
| conditional | `flows.md` when diagrams clarify; `design-brief.md` and `browser-evidence.md` for UI; `drift.md` for drift; `handoff.md` only when requested; `references.md` + `references/` when references exist |

## What each file owns

| File | Job | Budget |
| --- | --- | --- |
| `README.md` / `index.md` / `feature.md` | compact workspace map: phase, status, next action, artifact map, read-next table, blocking gates, last updated | 120 lines |
| `brief.md` | user request, objective, non-goals, success definition | 80 lines |
| `spec.md` | product WHAT/WHY, requirements, acceptance criteria, edge cases, measurable success, scope boundaries | 260 lines |
| `decision-coverage.md` | topology-first coverage matrix, assumption audit, residual uncertainty, and typed clarity verdict | 200 lines |
| `strategy.md` | temper verdict, scope mode/deltas, pre-mortem risks, deferred ambition | 180 lines |
| `architecture.md` | owning layer, integration points, data/API/events, dependencies, risks, affected boundaries | 180 lines |
| `flows.md` | useful Mermaid sequence/state/data/lifecycle diagrams with why-it-matters text and related IDs | 160 lines |
| `decisions.md` | ADR-style `DEC-###` log: status, context, options, decision, consequences, related IDs | 200 lines |
| `assumptions.md` | assumptions with confidence, owner, validation status | 160 lines |
| `questions.md` | `Q-###` open/resolved questions, gate, answer, impact | 180 lines |
| `plan.md` | technical approach, slice strategy, validation strategy, rollback | 220 lines |
| `tasks.md` | `SLICE-###` vertical slices with AC IDs, likely files, tests/proof, mode/gate, dependencies, Forge contract, done condition | 280 lines |
| `traceability.md` | matrix: AC/REQ ID, slice IDs, test/proof, evidence ID, touched files, status | 220 lines |
| `eng-review.md` | vetted scope/architecture/quality/performance findings, failure modes, build-entry preflight | 240 lines |
| `test-plan.md` | executable proof commands, preflight/provenance contract, acceptance and interaction coverage | 260 lines |
| `state.md` | compact cursor table/key-values; no narrative log | 120 lines |
| `evidence.md` / `proof.md` | `EVID-###` command/action, result, timestamp if available, related AC/slice IDs, limitation | 280 lines |
| `browser-evidence.md` | UI route/viewports/screenshots/console/network/interactions and Visual Verdict | 220 lines |
| `drift.md` | `DRIFT-###` spec/plan drift and resolution | 160 lines |
| `touched-files.md` | implementation files, slice ownership, reason, and a concern-ordered `## Review trail` of `path:line` stops for human review | 160 lines |
| `design-brief.md` | UI design direction, states, interaction model | 160 lines |
| `handoff.md` | cold-resume guide: current objective, last completed slice, next action, blockers, read-next links | 120 lines |

Files can exceed the budget only with `Budget override: <reason>`.

## Canonical slice grammar

Every producer of `tasks.md` uses this field set. `Dependencies` is slice
ordering; package and service requirements belong in `External dependencies`.

<!-- canonical-slice:start -->
```markdown
## SLICE-001 <observable capability>
Goal: <single observable capability>
Satisfies: AC-001[, AC-002]
Acceptance criteria: <binary criteria this slice closes>
Complexity: <1..5> — <reason>
Forge: <no | yes — reason>
Forge strategies: <A=<complete approach> | B=<complete approach> [| C=<complete approach>] | none>
Forge scorecard: <acceptance=AC-### list; test-plan=exact test-plan.md rows/commands | none>
Mode: <AFK | HITL>
Gate: <advisory | validating | blocking | escalating>
SLA: <15m | 4h | 24h | none>
Checkpoint: <question + why it needs unavailable pre-code evidence or action-time approval | none>
Dependencies: <SLICE-### list | none>
depends_on: [<SLICE-### IDs>]
Consumes / Produces: <interfaces read and exposed>
Known-Gotchas: <ordering hazards and framework footguns | none>
Prior-slice learnings: <constraints learned earlier | none>
Files likely touched: <real paths>
Tests/proof: <exact command + cwd + expected signal + prerequisites/provenance inputs>
Browser proof required: <yes | no>
Frontend craft required: <yes | no>
Design brief states: <UI states/interaction | none>
Visual acceptance: <state × viewport × input target | none>
Fullstack (FE+BE): <yes | no>
External dependencies: <libraries/services | none>
Existing to reuse / extend: <components/utilities/patterns | none>
Rollback notes: <reversal path>
Evidence required: <evidence /rite-prove must capture>
Edge/Prohibition coverage: <EDGE/PROH IDs | none>
Done condition: <checkable, exhaustive completion criterion>
```
<!-- canonical-slice:end -->

`depends_on` is the machine-readable mirror of `Dependencies`; keep the sets
identical and cycle-free. `Gate`, `SLA`, and `Checkpoint` are required for HITL
slices; use `none` when they do not apply. Complexity above 3 triggers reslicing
unless the stated reason makes the boundary irreducible. `/rite-define` writes
`Forge: no` with both detail fields `none`; `/rite-vet` is the sole promoter.
`Forge: yes` requires a reason, two or three distinct contiguous `A`–`C`
strategies, and a scorecard that names every slice AC plus exact
`test-plan.md` rows or commands.

## Read next by phase

| Phase | Read |
| --- | --- |
| spec | `README.md`, `brief.md`, `spec.md`, `references.md`, `questions.md` |
| clarify | `README.md`, `state.md`, `brief.md`, `spec.md`, `decisions.md`, `assumptions.md`, `questions.md` |
| temper | `spec.md`, `decision-coverage.md`, `decisions.md`, `assumptions.md`, `design-brief.md` |
| define | `README.md`, `state.md`, `spec.md`, `decision-coverage.md`, `architecture.md`, `decisions.md`, `assumptions.md` |
| vet | `README.md`, `traceability.md`, `plan.md`, `tasks.md`, `architecture.md`, `decisions.md` |
| build | `state.md`, `decision-coverage.md`, `eng-review.md`, `test-plan.md`, `tasks.md`, `plan.md`, `architecture.md`, `traceability.md`, `questions.md` |
| prove | `traceability.md`, `tasks.md`, `evidence.md`, `browser-evidence.md`, `touched-files.md` |
| review/seal | `README.md`, `traceability.md`, `spec.md`, `evidence.md`, `decisions.md`, `drift.md`, `touched-files.md` |
| handoff | `README.md`, `state.md`, `handoff.md`, then the linked source artifacts |

## Do not duplicate

- Do not make `spec.md` carry deep architecture; link to `architecture.md`/`flows.md`.
- Do not copy acceptance criteria into `plan.md`; reference `AC-###`.
- Do not copy full proof into `handoff.md`; link to `evidence.md`.
- Do not make `state.md` an append-only log; keep only the current cursor.
- Do not create optional files before their phase; absence is meaningful.

## ID contract

Use stable IDs: `REQ-001`, `AC-001`, `SLICE-001`, `DEC-001`, `Q-001`,
`DRIFT-001`, `EVID-001`. Old `AC1` and `Slice 1` forms are legacy and should
not be generated for new workspaces.

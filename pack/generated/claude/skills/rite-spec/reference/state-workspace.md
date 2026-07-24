# `.devrites/` state workspace

This is the writer-facing companion to the canonical schema in
[`devrites-lib/reference/workspace-artifact-schema.md`](../../devrites-lib/reference/workspace-artifact-schema.md).
Load that schema before creating or updating workspace artifacts.

## Layout

```text
.devrites/
  ACTIVE
  work/
    <feature-slug>/
      README.md                  # compact workspace map (feature.md/index.md aliases supported)
      brief.md
      spec.md
      architecture.md            # from /rite-define
      flows.md                   # optional, only when useful
      decisions.md
      assumptions.md
      questions.md
      decision-coverage.md       # from /rite-clarify
      plan.md                    # from /rite-define
      tasks.md                   # from /rite-define
      traceability.md            # from /rite-define
      eng-review.md              # from /rite-vet
      test-plan.md               # from /rite-vet
      state.md
      evidence.md                # from /rite-build or /rite-prove (proof.md alias supported)
      browser-evidence.md        # UI only
      drift.md                   # drift only
      touched-files.md           # from /rite-build
      design-brief.md            # UI only
      handoff.md                 # from /rite-handoff
      references.md
      references/
  archive/
    <feature-slug>/
```

Backward compatibility: if an installed project already uses
`.devrites/features/<slug>/`, keep reading and updating that workspace. New
writer instructions should prefer `.devrites/work/<slug>/`; migration adds
canonical files and preserves aliases rather than deleting old ones.

## Creation rules

- Slug is kebab-case and comes from the objective.
- `/rite-spec` creates the workspace map, `brief.md`, `spec.md`, `decisions.md`,
  `assumptions.md`, `questions.md`, `state.md`, optional `references.md` /
  `references/`, and optional `design-brief.md` for UI.
- `/rite-clarify` adds `decision-coverage.md`.
- `/rite-define` adds `architecture.md`, `plan.md`, `tasks.md`, and
  `traceability.md`.
- `/rite-vet` adds `eng-review.md` and `test-plan.md`.
- Later phases add only the artifact they own. Do not create optional files as
  empty placeholders; absence means the phase has not produced that artifact.
- Each artifact starts with a summary and links to deeper source files instead of
  copying their content.
- Use stable IDs: `REQ-001`, `AC-001`, `SLICE-001`, `DEC-001`, `Q-001`,
  `DRIFT-001`, `EVID-001`.

## README.md template

```markdown
# <Feature> Workspace
phase: spec
status: running
next_action: /rite-clarify after spec readiness passes
last_updated: <date>

## Artifact map
| File | Job |
| --- | --- |
| brief.md | Objective and bounds |
| spec.md | Product contract |
| decision-coverage.md | Clarification verdict; absent until /rite-clarify |
| architecture.md | Technical map; absent until /rite-define |
| plan.md | Technical approach; absent until /rite-define |
| tasks.md | Vertical slices; absent until /rite-define |
| traceability.md | Coverage dashboard; absent until /rite-define |
| evidence.md | Proof log; absent until /rite-prove |

## Read next
| Phase / role | Read |
| --- | --- |
| Spec | brief.md, spec.md, questions.md |
| Clarify | spec.md, decisions.md, assumptions.md, questions.md |
| Build | state.md, decision-coverage.md, eng-review.md, test-plan.md, tasks.md, plan.md |
| Review | traceability.md, evidence.md, decisions.md |

## Blocking gates
None.
```

## state.md template

```markdown
# State

## Cursor
| Key | Value |
| --- | --- |
| phase | spec |
| status | running |
| active_slice | none |
| slice_mode | none |
| risk | none |
| next_action | <single command + reason> |
| return_phase | <later phase; retrofit clarification only> |
| return_next_action | <saved command; retrofit clarification only> |

## Awaiting human
Only present when status is awaiting_human.
| Key | Value |
| --- | --- |
| question_id | Q-001 |
| gate | blocking |
| blocking_slices | SLICE-001 |
```

`state.md` is a compact cursor, not a history file. Put proof in `evidence.md`,
decisions in `decisions.md`, assumptions in `assumptions.md`, and drift in
`drift.md`. Omit both `return_*` rows outside a later-phase `/rite-clarify`
retrofit; `devrites-engine clarify-return` owns their atomic lifecycle.

## questions.md entry format

```markdown
## Q-001
status: open | answered | dropped
slice: spec | plan | SLICE-001
gate: advisory | validating | blocking | escalating
question: <one crisp sentence>
options:
  1. <recommended> (Recommended) - <trade-off>
  2. <alternative> - <trade-off>
proposed: <best current answer>
raised_at: <iso>
answered_at:
answer:
impact: <affected AC/REQ/slice IDs>
```

No open blocking/escalating question may remain before define/build/prove gates.

## brief.md template

```markdown
# Brief: <Feature>

## Objective
<One sentence.>

## Non-goals
- <Explicitly out of scope.>

## Success definition
<One sentence that says what "done" means.>
```

## Compactness

High-traffic files should stay within the budgets in
`workspace-artifact-schema.md`. If a file truly must exceed its budget, add
`Budget override: <reason>` near the top.

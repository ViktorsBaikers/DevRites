# DevRites Workspace Artifact Schema

DevRites workspaces are durable source-of-truth folders for a feature. Chat is
temporary; the workspace must be enough for a fresh human or AI to resume without
guessing. The schema follows three rules:

- Keep each Markdown file single-purpose and compact.
- Read the index first, then load only the files needed for the current phase.
- Trace every acceptance criterion through slices, proof, evidence, and touched files.

Root selection has two axes:

- `DEVRITES_ROOT` selects the project root or `.devrites/` directory. When unset, the engine walks up
  from cwd to the nearest `.devrites/`.
- `.devrites/ACTIVE` selects the active feature workspace.
- `DEVRITES_WORKSPACE` optionally names an explicit workspace path for CI/agents and overrides
  `.devrites/ACTIVE` for commands that default to the active feature.

Canonical live workspace:

```text
.devrites/
  ACTIVE
  work/
    <slug>/
      README.md                 # compact workspace map; feature.md/index.md aliases allowed
      brief.md                  # request, objective, non-goals, success definition
      spec.md                   # product WHAT/WHY and acceptance contract
      architecture.md           # feature technical map
      flows.md                  # optional Mermaid diagrams
      decisions.md              # ADR-style decision log
      assumptions.md            # assumptions and validation status
      questions.md              # open/resolved human questions
      plan.md                   # technical approach and slice strategy
      tasks.md                  # vertical slices
      traceability.md           # AC/REQ -> slices -> tests/evidence/files matrix
      state.md                  # compact cursor
      evidence.md               # command/action proof; proof.md alias allowed
      browser-evidence.md       # optional UI/runtime browser proof
      drift.md                  # optional spec/plan drift register
      touched-files.md          # implementation file list
      design-brief.md           # optional UI design contract
      handoff.md                # optional cold-resume summary
      references.md
      references/
  archive/
    <slug>/
```

Backward compatibility: `.devrites/features/<slug>/` remains readable as an alias
for `.devrites/work/<slug>/`; `feature.md` and `index.md` remain valid workspace
maps; `proof.md` remains a proof alias for `evidence.md`. Migration should add the
canonical files without deleting aliases.

## Read Order

| Phase / role | Read first | Then read | Usually skip |
| --- | --- | --- | --- |
| Spec | `README.md`, `brief.md` | `spec.md`, `references.md`, `questions.md` | `plan.md`, `evidence.md` |
| Define / plan | `README.md`, `state.md` | `spec.md`, `architecture.md`, `decisions.md`, `assumptions.md` | `evidence.md` |
| Vet | `README.md`, `traceability.md` | `plan.md`, `tasks.md`, `architecture.md`, `decisions.md` | browser proof unless UI |
| Build | `state.md`, `tasks.md` | `plan.md`, `architecture.md`, `traceability.md`, `questions.md` | full evidence history |
| Prove | `traceability.md`, `tasks.md` | `evidence.md`, `browser-evidence.md`, `touched-files.md` | long plan rationale |
| Review / seal | `README.md`, `traceability.md` | `spec.md`, `evidence.md`, `decisions.md`, `drift.md`, `touched-files.md` | raw logs unless needed |
| Handoff | `README.md`, `state.md` | `handoff.md`, then linked source files | duplicated summaries |

## Artifact Contracts

| Artifact | Required? | Owner phase | Read trigger | Budget | Required headings | Validation rules |
| --- | --- | --- | --- | --- | --- | --- |
| `README.md` / `index.md` / `feature.md` | required | `/rite-spec` | always first | 120 lines | current phase, status, next action, artifact map, read-next table, blocking gates, last updated | Must not duplicate full spec/plan/evidence. |
| `brief.md` | required | `/rite-spec` | clarify objective | 80 lines | Objective, Non-goals, Success definition | One-screen request and scope summary. |
| `spec.md` | required | `/rite-spec` | product contract | 260 lines | Problem, Goal, Non-goals, Users / actors, Requirements, Acceptance criteria, Edge Coverage, Prohibitions (must-NOT), Edge cases, Measurable success, Scope boundaries | Must use stable `REQ-###` and `AC-###`; no deep implementation details. |
| `architecture.md` | required from plan | `/rite-define` | placement/integration work | 180 lines | Owning module / layer, Integration points, Data / API / events, Dependencies, Risks, Affected boundaries | Carries topology and boundaries, not product acceptance. |
| `flows.md` | optional | `/rite-spec` or `/rite-define` | lifecycle/state/sequence/data flow is hard to infer | 160 lines | diagram-specific headings | Mermaid only when it clarifies behavior; each diagram states why it matters and related IDs. |
| `decisions.md` | required | all phases | non-trivial product/technical choice | 200 lines | Decision log | Entries use `DEC-###`, status, context, options, decision, consequences, related IDs. |
| `assumptions.md` | required | all phases | assumption is not yet verified | 160 lines | Assumption register | Confidence, owner, and validation status are explicit. |
| `questions.md` | required | all phases | human input or gate | 180 lines | Question register | `Q-###`; no open blocking/escalating questions before plan/build/prove gates. |
| `ai-spec.md` | conditional | `/rite-spec` or `/rite-define` | AI/LLM annex | 160 lines | AI surface, model/runtime choice, evals, guardrails, monitoring | Required only for model calls, RAG, agents, evals, or LLM output. |
| `plan.md` | required from plan | `/rite-define` | implementation approach | 220 lines | Approach, Slice strategy, Validation strategy, Rollback | HOW lives here, not in `spec.md`. |
| `tasks.md` | required from plan | `/rite-define` | build one slice | 280 lines | Slice index | Each `SLICE-###` has goal, AC IDs, likely files, tests/proof, mode, gate, dependencies, done condition. |
| `traceability.md` | required from plan | `/rite-define` | coverage/review/seal | 220 lines | Coverage matrix | Matrix maps AC/REQ ID, slice IDs, test/proof, evidence ID, touched files, status. |
| `state.md` | required | all phases | current cursor | 120 lines | Cursor | Compact table/key-value cursor; not an append-only narrative. |
| `evidence.md` / `proof.md` | required from prove | `/rite-build`, `/rite-prove` | proof and seal | 280 lines | Evidence log | `EVID-###`, command/action, result, related AC/slice IDs, limitations. |
| `browser-evidence.md` | UI only | `/rite-prove`, `/rite-polish` | UI/browser proof | 220 lines | Browser evidence, Visual Verdict | Must reference real route/viewports/actions and related IDs. |
| `drift.md` | drift only | Spec Drift Guard | spec/plan reality mismatch | 160 lines | Drift register | `DRIFT-###`, status, evidence found, resolution, related IDs. |
| `touched-files.md` | required from build/prove | `/rite-build` | impact/evidence freshness | 160 lines | Touched files | File, slice ownership, and reason per row. |
| `design-brief.md` | UI only | `devrites-ux-shape` | UI build/proof | 160 lines | Design direction, States, Interaction model | UI target only; no implementation task list. |
| `handoff.md` | optional | `/rite-handoff` | cold resume | 120 lines | Resume, Read next, Next action | Links to source artifacts instead of copying them. |
| `references.md` + `references/` | optional | `/rite-spec` | external/user-supplied material | 160 lines | Reference index | Links must resolve or be external URLs. |

Files may exceed budgets only with a visible `Budget override: <reason>` line.

## Required vs Optional

Required at `/rite-spec`: workspace map, `brief.md`, `spec.md`, `state.md`,
`decisions.md`, `assumptions.md`, `questions.md`.

Required at `/rite-define` and later: all spec files plus `architecture.md`,
`plan.md`, `tasks.md`, and `traceability.md`.

Required at `/rite-prove` and later: all plan files plus `evidence.md` (or
`proof.md`) and `touched-files.md`.

Optional/conditional files are generated only by their producing phase:
`flows.md` when diagrams clarify behavior, `design-brief.md` and
`browser-evidence.md` for UI, `drift.md` for drift, `handoff.md` for handoff,
and `references.md` / `references/` when references exist.

## Compactness Rules

- Put summaries first and deep details behind file links.
- Use tables for matrices, gates, and checklists.
- Do not repeat boilerplate across files.
- Do not copy evidence output into `handoff.md`; link to `evidence.md`.
- Do not copy architecture diagrams into `spec.md`; link to `architecture.md` or `flows.md`.
- Do not copy full acceptance criteria into `plan.md`; reference `AC-###`.
- Do not append long narrative logs to `state.md`; keep only the current cursor.
- Prefer Mermaid only for useful boundary, sequence, state, lifecycle, or data-flow diagrams.

## Stable IDs

Use these forms in generated artifacts:

| Kind | Format | Home |
| --- | --- | --- |
| Requirement | `REQ-001` | `spec.md` |
| Acceptance criterion | `AC-001` | `spec.md` |
| Slice | `SLICE-001` | `tasks.md` |
| Decision | `DEC-001` | `decisions.md` |
| Question | `Q-001` | `questions.md` |
| Drift event | `DRIFT-001` | `drift.md` |
| Evidence | `EVID-001` | `evidence.md`, `browser-evidence.md` |

## Validation

`scripts/validate-workspace-schema.py` validates generated workspaces. The
project-wide validation path runs it against representative fixtures.

The validator checks:

- required files and headings for the current phase;
- stable IDs and old `AC1` / `Slice 1` style regressions;
- acceptance criteria referenced by at least one slice;
- completed slices referenced by evidence;
- no unresolved blocking/escalating questions before plan/build/prove gates;
- high-traffic file budgets unless explicitly justified;
- plausible Mermaid fences;
- local Markdown links that point to existing files;
- evidence IDs mapped in `traceability.md` once proof exists.

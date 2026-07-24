# DevRites workspace artifact schema

A DevRites workspace is the durable record for a feature. Chat is temporary,
so a fresh human or AI must be able to resume from the files alone. The schema
follows three rules:

- Keep each Markdown file single-purpose and compact.
- Read the index first, then load only the files needed for the current phase.
- Trace every acceptance criterion through slices, proof, evidence, and touched files.

Root and workspace selection use these inputs:

- `DEVRITES_ROOT` selects the project root or `.devrites/` directory and is
  canonicalized to its physical path.
- Without an override, the engine walks from cwd only to the current Git
  top-level. A nested repository, worktree, or submodule never inherits
  `.devrites/` from its parent or superproject. Outside Git, the filesystem
  ancestor walk remains available.
- `.devrites/ACTIVE` selects the active feature workspace.
- `DEVRITES_WORKSPACE` optionally names an explicit workspace path for CI/agents and overrides
  `.devrites/ACTIVE` for commands that default to the active feature. Its
  resolved path must stay inside the selected `.devrites/` root. Setting it
  without a selectable root is a refusing `DRV-WORKSPACE-WITHOUT-ROOT` hazard.

`devrites-engine context show --json` and `doctor` expose the same canonical
facts: lexical/physical project and root, selection reason, Git top-level,
physical Git dir/common dir/superproject, linked-worktree/submodule flags, and
stable hazards. The engine refuses writes when root selection is unsafe or
ambiguous. `doctor` reports ordinary absence without treating it as a crash.

The canonical workspace-map frontmatter declares `schemaVersion: 2`. The
engine reads missing/v1 declarations and legacy aliases additively, refuses a
version newer than it supports, and keeps the snapshot wire schema separate.
`devrites-engine migrate` upgrades the declaration and layout metadata; it does
not manufacture decision coverage, vet readiness, test plans, or proof.
Semantic planning compatibility uses a separate readiness-artifact contract.
The current value is `devrites.readiness-artifacts.v2`; `/rite-upgrade [slug]`
reconciles an active unfinished workspace when build readiness reports an older
or unknown contract.

Canonical live workspace:

```text
.devrites/
  ACTIVE
  timeline.jsonl              # local metadata-only v1 workflow trace
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
      decision-coverage.md       # topology-first clarification verdict
      plan.md                   # technical approach and slice strategy
      tasks.md                  # vertical slices
      traceability.md           # AC/REQ -> slices -> tests/evidence/files matrix
      eng-review.md              # implementation-readiness verdict
      test-plan.md               # executable proof target
      state.md                  # compact cursor; status.md alias allowed
      events.jsonl              # feature-local copy of v1 workflow trace
      recovery-attempts.jsonl   # durable bounded technical-recovery ledger
      .git-authority-consumption.jsonl # private one-shot Git grant consumption
      .wright-allowlist         # root-owned exact source/test write scope
      .forge/
        <run-id>/manifest.json  # engine-owned Forge lifecycle and path authority
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
maps; `status.md` remains a cursor alias for `state.md`; `proof.md` remains a
proof alias for `evidence.md`. Migration normalizes compatible metadata without
deleting aliases or inventing evidence.

## Read order

| Phase / role | Read first | Then read | Usually skip |
| --- | --- | --- | --- |
| Spec | `README.md`, `brief.md` | `spec.md`, `references.md`, `questions.md` | `plan.md`, `evidence.md` |
| Clarify | `README.md`, `state.md` | `spec.md`, `decisions.md`, `assumptions.md`, `questions.md` | plan/proof artifacts |
| Define / plan | `README.md`, `state.md` | `spec.md`, `decision-coverage.md`, `architecture.md`, `decisions.md`, `assumptions.md` | `evidence.md` |
| Vet | `README.md`, `traceability.md` | `plan.md`, `tasks.md`, `architecture.md`, `decisions.md` | browser proof unless UI |
| Build | `state.md`, `tasks.md` | `decision-coverage.md`, `eng-review.md`, `test-plan.md`, `plan.md`, `architecture.md`, `traceability.md`, `questions.md` | full evidence history |
| Prove | `traceability.md`, `tasks.md` | `evidence.md`, `browser-evidence.md`, `touched-files.md` | long plan rationale |
| Review / seal | `README.md`, `traceability.md` | `spec.md`, `evidence.md`, `decisions.md`, `drift.md`, `touched-files.md` | raw logs unless needed |
| Handoff | `README.md`, `state.md` | `handoff.md`, then linked source files | duplicated summaries |

## Artifact contracts

| Artifact | Required? | Owner phase | Read trigger | Budget | Required headings | Validation rules |
| --- | --- | --- | --- | --- | --- | --- |
| `README.md` / `index.md` / `feature.md` | required | `/rite-spec` | always first | 120 lines | current phase, status, next action, artifact map, read-next table, blocking gates, last updated | Must not duplicate full spec/plan/evidence. |
| `brief.md` | required | `/rite-spec` | clarify objective | 80 lines | Objective, Non-goals, Success definition | One-screen request and scope summary. |
| `spec.md` | required | `/rite-spec` | product contract | 260 lines | Problem, Goal, Non-goals, Users / actors, Requirements, Acceptance criteria, Edge Coverage, Prohibitions (must-NOT), Edge cases, Measurable success, Scope boundaries | Must use stable `REQ-###` and `AC-###`; no deep implementation details. |
| `decision-coverage.md` | required from clarify | `/rite-clarify` | decision/readiness audit | 200 lines | Topology, Coverage matrix, Assumption audit, Residual uncertainty, Readiness verdict | Exactly one semantic `Decision coverage: CLEAR` plus a fresh canonical input digest. Partial, Missing, unowned, open-human-gate, placeholder, or incomplete rows block. |
| `architecture.md` | required from plan | `/rite-define` | placement/integration work | 180 lines | Owning module / layer, Integration points, Data / API / events, Dependencies, Risks, Affected boundaries | Carries topology and boundaries, not product acceptance. |
| `flows.md` | optional | `/rite-spec` or `/rite-define` | lifecycle/state/sequence/data flow is hard to infer | 160 lines | diagram-specific headings | Mermaid only when it clarifies behavior; each diagram states why it matters and related IDs. |
| `decisions.md` | required | all phases | non-trivial product/technical choice | 200 lines | Decision log | Entries use `DEC-###`, status, context, options, decision, consequences, related IDs. |
| `assumptions.md` | required | all phases | assumption is not yet verified | 160 lines | Assumption register | Confidence, owner, and validation status are explicit. |
| `questions.md` | required | all phases | human input or gate | 180 lines | Question register | `Q-###`; no open blocking/escalating questions before plan/build/prove gates. |
| `ai-spec.md` | conditional | `/rite-spec` or `/rite-define` | AI/LLM annex | 160 lines | AI surface, model/runtime choice, evals, guardrails, monitoring | Required only for model calls, RAG, agents, evals, or LLM output. |
| `plan.md` | required from plan | `/rite-define` | implementation approach | 220 lines | Approach, Slice strategy, Validation strategy, Rollback | HOW lives here, not in `spec.md`. `Validation strategy` names the Key links: cross-slice wiring `/rite-prove` walks (or `none` for single-slice features). |
| `tasks.md` | required from plan | `/rite-define` | build one slice | 280 lines | Slice index | Each `SLICE-###` has goal, AC IDs, likely files, tests/proof, mode, gate, dependencies, done condition. |
| `traceability.md` | required from plan | `/rite-define` | coverage/review/seal | 220 lines | Coverage matrix | Matrix maps AC/REQ ID, slice IDs, test/proof, evidence ID, touched files, status. |
| `eng-review.md` | required from vet | `/rite-vet` | build readiness | 240 lines | Depth, Scope challenge, Build-entry preflight, Implementation readiness, Completion summary | Exactly one semantic `Implementation readiness: READY` plus a fresh canonical input digest. Non-passing preflight/readiness rows, placeholders, empty tables, or unmapped acceptance block. |
| `test-plan.md` | required from vet | `/rite-vet` | build/prove coverage target | 260 lines | Build-entry preflight, Coverage diagram, Per-gap test requirements, Acceptance → test map | Commands, cwd, expected signal, prerequisites, and provenance are executable. |
| `state.md` / `status.md` | required | all phases | current cursor | 120 lines | Cursor | `state.md` is canonical; `status.md` is a compatibility alias. Compact table/key-value cursor; later-phase clarification durably stores `return_phase` and `return_next_action` until fresh coverage restores or replans it. |
| root `timeline.jsonl` + feature `events.jsonl` | local telemetry only | engine/hooks | bounded report, audit, manual retention | JSONL | n/a | New writes are validated `devrites-event/v1` metadata only. Reports ignore legacy/corrupt rows and never make events lifecycle-authoritative. Exact purge may delete only these telemetry files. |
| `recovery-attempts.jsonl` | technical recovery only | root orchestrator | retry budget | generated | n/a | Fingerprinted, append-only failed/cleared records cap the same objective root cause at three failed attempts across agents and sessions. |
| `.git-authority-consumption.jsonl` | destructive Git only | `git-guard` | one-shot replay prevention | generated | n/a | `devrites-git-authority-consumption/v1` JSONL, bounded and mode `0600` where POSIX modes exist; stores only qid, exact digest, classifier reason IDs, and consumption time. |
| `.wright-allowlist` | build only | root orchestrator | writer scope | generated | n/a | Exact normalized project-relative files only; no directories, globs, traversal, duplicates, or `.devrites/**`. Writer self-report never expands it. |
| `.forge/<run-id>/manifest.json` | Forge runs only | engine | candidate ownership and lifecycle | generated | n/a | One `devrites-forge/v1` manifest owns the primary baseline, scorecard hashes, candidate paths/branches/states, worker bindings, judge winner, merge, verification, and cleanup. Duplicate, symlinked, escaped, or tampered ownership fails closed. |
| `evidence.md` / `proof.md` | required from prove | `/rite-build`, `/rite-prove` | proof and seal | 280 lines | Evidence log | `EVID-###`, command/action, result, related AC/slice IDs, limitations. Each acceptance criterion carries a proof class: `test` / `command` / `browser` / `judgment` (untagged reads `judgment`; `judgment` needs its one-line why). |
| `browser-evidence.md` | UI only | `/rite-prove`, `/rite-polish` | UI/browser proof | 220 lines | Browser evidence, Visual Verdict | Must reference real route/viewports/actions and related IDs. |
| `drift.md` | drift only | Spec Drift Guard | spec/plan reality mismatch | 160 lines | Drift register | `DRIFT-###`, status, evidence found, resolution, related IDs. |
| `touched-files.md` | required from build/prove | `/rite-build` | impact/evidence freshness | 160 lines | Touched files | File, slice ownership, and reason per row. |
| `design-brief.md` | UI only | `devrites-ux-shape` | UI build/proof | 160 lines | Design direction, States, Interaction model | UI target only; no implementation task list. |
| `handoff.md` | optional | `/rite-handoff` | cold resume | 120 lines | Resume, Read next, Next action | Links to source artifacts instead of copying them. |
| `references.md` + `references/` | optional | `/rite-spec` | external/user-supplied material | 160 lines | Reference index | Links must resolve or be external URLs. |

Telemetry is deliberately outside the correctness contract. Missing or corrupt
`timeline.jsonl` / `events.jsonl` may degrade `timeline report`, `progress`, or
the statusline, but can never change a gate, cursor, question, evidence verdict,
recovery budget, capability decision, or destructive-action authority. Reports
read a bounded tail; `timeline purge --before ...` and/or `--run ...` rewrites
only these logs. The engine sends no remote telemetry. Remove old unversioned
logs manually when their legacy free text should not be retained.

Files may exceed budgets only with a visible `Budget override: <reason>` line.

## Readiness gate binding

The typed verdict is necessary but not sufficient:

- `decision-coverage.md`, `eng-review.md`, and `test-plan.md` declare
  `DevRites contract: devrites.readiness-artifacts.v2`.
- `Coverage inputs SHA-256` binds `decision-coverage.md` to `brief.md`,
  `spec.md`, `decisions.md`, `assumptions.md`, and `questions.md`.
- `Readiness inputs SHA-256` binds `eng-review.md` to `spec.md`,
  `decision-coverage.md`, `architecture.md`, `plan.md`, `tasks.md`,
  `traceability.md`, and `test-plan.md`.

`devrites-engine readiness-digest coverage|engineering [slug]` emits the
canonical field line. `build-readiness` rejects stale digests, malformed or
contradictory content, missing markers, and older or unknown semantic contracts.
Contract failures use code `8` and route to `/rite-upgrade`; missing coverage or
vet artifacts keep their existing `/rite-clarify` and `/rite-vet` routes. A
later-phase clarification
retrofit uses `clarify-return enter|restore`; restore requires fresh `CLEAR`
coverage, while an acceptance-changing result keeps the cursor for plan repair.

## Required vs optional

Required at `/rite-spec`: workspace map, `brief.md`, `spec.md`, `state.md`,
`decisions.md`, `assumptions.md`, `questions.md`.

Required at `/rite-clarify` and later: all spec files plus `decision-coverage.md`.

Required at `/rite-define` and later: all clarified spec files plus `architecture.md`,
`plan.md`, `tasks.md`, and `traceability.md`.

Required at `/rite-vet` and later: all plan files plus `eng-review.md` and `test-plan.md`.

Required at `/rite-prove` and later: all vetted plan files plus `evidence.md` (or
`proof.md`) and `touched-files.md`.

Optional/conditional files are generated only by their producing phase:
`flows.md` when diagrams clarify behavior, `design-brief.md` and
`browser-evidence.md` for UI, `drift.md` for drift, `handoff.md` for handoff,
and `references.md` / `references/` when references exist. The build runtime
creates `.wright-allowlist` and, only when needed, `recovery-attempts.jsonl`.
Forge creates `.forge/<run-id>/manifest.json` only after a clean, bound
parallel plan passes. A serial degradation creates no Forge state.

Forge candidate worktrees do not live inside `.devrites`. The manifest binds
them to the sibling
`.<repo>.devrites-forge/<run-id>/candidate-<a|b|c>` paths before Git creates
them. Every later extract, merge, cleanup, and reap command reloads the unique
manifest instead of accepting a caller-supplied path or branch. Cleanup
requires a landed winner and recorded successful verification, and it
preserves dirty, live, foreign, or ambiguous state.

## Compactness rules

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
| Forge run | `forge-` plus 24 lowercase hex characters | `.forge/<run-id>/manifest.json` |

## Validation

`scripts/validate-workspace-schema.py` validates generated workspaces. The
project-wide validation path runs it against representative fixtures.

The validator checks:

- required files and headings for the current phase;
- supported workspace `schemaVersion` and additive legacy aliases;
- semantic, fresh digest-bound `CLEAR` and `READY` readiness artifacts;
- stable IDs and old `AC1` / `Slice 1` style regressions;
- acceptance criteria referenced by at least one slice;
- completed slices referenced by evidence;
- no unresolved blocking/escalating questions before plan/build/prove gates;
- high-traffic file budgets unless explicitly justified;
- plausible Mermaid fences;
- local Markdown links that point to existing files;
- evidence IDs mapped in `traceability.md` once proof exists.

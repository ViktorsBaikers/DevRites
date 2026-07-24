# `.devrites/` state schema (v2)

The `devrites-engine` binary reads a project's workflow state from plain files
under `.devrites/`. Those hand-editable files are the source of truth, and the
engine reads human edits directly.

<!-- authority:schema-version:start -->
`schemaVersion: 2`.
<!-- authority:schema-version:end -->

For the layered load-order diagram, budget table, manifest/alias model, and
phase-required section matrix, see [`workspace-schema.md`](workspace-schema.md).

## Layout

```
.devrites/
  work/
    <slug>/
      README.md       workspace map: phase, status, next action, read-next
      brief.md        objective, non-goals, success definition
      spec.md         product WHAT/WHY, requirements, acceptance, boundaries
      architecture.md feature technical map
      plan.md         approach and slice strategy
      tasks.md        SLICE-### vertical slices
      traceability.md AC/REQ -> slices -> tests/evidence/files
      decisions.md    DEC-### decision log
      assumptions.md  assumption register
      questions.md    Q-### question register
      decision-coverage.md  digest-bound clarification verdict
      eng-review.md   digest-bound implementation-readiness verdict
      test-plan.md    build-entry and acceptance-to-test plan
      state.md        compact cursor
      evidence.md     EVID-### command/action proof
      touched-files.md implementation file map
  specs/              capability ledger: the living "what the system does now"
    <capability>/
      spec.md         proven Requirement blocks for this capability
```

Each feature is a directory of small files with one concern apiece. This keeps
the context for each file small and makes missing content easy to spot.

### Capability ledger: `specs/`

`work/<slug>/` is **ephemeral** (archived on ship); `specs/` is **durable**. It is
the cumulative record of proven behavior, one `spec.md` per capability, so a new
feature starts from the system's current contract instead of re-deriving it from
code. A feature's spec groups deltas under `ADDED`, `MODIFIED`, or `REMOVED`
Requirements headings tagged with `capability: <c>`. On ship,
`devrites-engine ledger sync` folds them in: ADDED appends, MODIFIED replaces by
header identity, and REMOVED deletes. Because it lives outside `work/`, the
ledger survives close-out archival. Unlike the rest of `.devrites/`, it is
**git-tracked** (`.devrites/*` + `!.devrites/specs/`). Git therefore shares the
proven contract across clones. Grammar and delta rules:
[`spec-grammar.md`](../../pack/.claude/skills/devrites-lib/reference/standards/spec-grammar.md).

Backward compatibility: `.devrites/features/<slug>/` remains readable as a legacy
workspace location. `feature.md` / `index.md` may stand in for `README.md`, and
`status.md` may stand in for `state.md`, while `proof.md` may stand in for
`evidence.md`.

### Workspace maps

The canonical per-feature index is `README.md`; `feature.md` and `index.md` are
readable aliases. A map may carry YAML frontmatter with:

| field           | meaning                                        |
| --------------- | ---------------------------------------------- |
| `slug`          | feature identifier (matches the directory)     |
| `title`         | human-readable title                           |
| `phase`         | current workflow phase (see below)             |
| `schemaVersion` | schema version the file was written against    |

A feature exists when it has either a live `state.md` ledger or a workspace map.
The mutable `state.md` cursor is authoritative when both declare a phase; an
unknown declared phase is an error.

## Sections

The legacy engine completeness sections, in canonical order:

`spec` · `plan` · `decisions` · `tasks` · `proof` · `status`

A section is **present** (has real content) when its `<section>.md` file exists
and, after removing any leading YAML frontmatter, ATX (`#`) headings, and
whitespace, some content remains. A stub that is only a heading counts as
**empty**. Scaffolding a file never fakes completeness.

The 2026 Markdown workspace schema is stricter and lives in
[`workspace-schema.md`](workspace-schema.md). It validates required phase-owned
artifacts, stable IDs (`AC-###`, `SLICE-###`, `EVID-###`), traceability, evidence,
compactness budgets, Mermaid fences, and stale local links.

## Phases and required sections

Phases follow the rite-\* arc. Completeness is **phase-relative**: each phase
requires only the sections needed to leave it, and the set grows additively down
the arc. A section that is not yet required (e.g. `proof` during the `spec`
phase) never blocks.

<!-- authority:phase-contract:start -->
| phase | normal resume | required sections | transition right |
| --- | --- | --- | --- |
| `frame` | `/rite-frame` | *(none)* | Frame an unstructured request before lifecycle work. |
| `spec` | `/rite-spec` | `spec` | Author the product specification. |
| `clarify` | `/rite-clarify` | `spec` | Close decision coverage in the written specification. |
| `temper` | `/rite-temper` | `spec` | Optionally challenge the clarified specification strategy. |
| `define` | `/rite-define` | `spec`, `plan` | Author and approve the initial implementation plan. |
| `plan` | `/rite-vet` | `spec`, `plan` | Hold the approved or repaired plan checkpoint for engineering review. |
| `vet` | `/rite-vet` | `spec`, `plan`, `decisions`, `tasks` | Review implementation readiness before build. |
| `build` | `/rite-build` | `spec`, `plan`, `decisions`, `tasks` | Implement the next approved vertical slice. |
| `converge` | `/rite-converge` | `spec`, `plan`, `decisions`, `tasks` | Recover unmet clarified intent into new slices. |
| `prove` | `/rite-prove` | `spec`, `plan`, `decisions`, `tasks`, `proof` | Produce acceptance evidence for the implementation. |
| `polish` | `/rite-polish` | `spec`, `plan`, `decisions`, `tasks`, `proof` | Apply the bounded quality pass. |
| `review` | `/rite-review` | `spec`, `plan`, `decisions`, `tasks`, `proof` | Review the proven implementation. |
| `seal` | `/rite-seal` | `spec`, `plan`, `decisions`, `tasks`, `proof`, `status` | Decide the final GO or NO-GO. |
| `ship` | `/rite-ship` | `spec`, `plan`, `decisions`, `tasks`, `proof`, `status` | Perform authorized release and close-out mutations. |
| `done` | *(terminal)* | `spec`, `plan`, `decisions`, `tasks`, `proof`, `status` | Represent archived completion with no resume command. |
<!-- authority:phase-contract:end -->

The authoritative typed definitions live in `engine/internal/state/schema.go`.
`workflow_manifest.json` is a generated derivative for non-Go release tools;
run `go generate ./internal/state` after editing the registry.

### Clarify-return field policy

<!-- authority:clarify-return-fields:start -->
| field | policy |
| --- | --- |
| `phase` | derived |
| `status` | derived |
| `next_action` | derived |
| `return_phase` | derived |
| `return_next_action` | curated when present; otherwise derived |
| `all other state.md content` | curated and preserved byte-for-byte |
<!-- authority:clarify-return-fields:end -->

## `devrites-engine status <slug>`

Prints the feature's phase and, for each section, its present/empty state and
whether the current phase requires it, then a completeness verdict computed over
the required sections only. Output is deterministic and greppable:

```
feature: auth-tokens
phase: build
  spec       present  required
  plan       present  required
  decisions  present  required
  tasks      empty    required
  proof      empty
  status     present
result: incomplete (missing: tasks)
```

- Found feature → exit `0`, whether complete or incomplete (`status` reports, it
  does not gate; gating is issue 04).
- Unknown or missing slug → non-zero exit with a clear message on stderr.

`status` makes no model or network calls; it is a pure read of the files under
`DEVRITES_ROOT` (or the nearest `.devrites/` above the working directory).
There is no status cache, so the next read reflects a hand edit immediately.
Other workspace control-plane commands share that deterministic boundary;
explicit
install/update/source-cache I/O is isolated under `engine/internal/iohooks` as
defined by ADR-0008.

<!-- authority:state-tracking:start -->
Git-tracked shared state: `.devrites/specs/`. Per-clone runtime state: `.devrites/work/`, `.devrites/archive/`, `.devrites/ACTIVE`.
<!-- authority:state-tracking:end -->

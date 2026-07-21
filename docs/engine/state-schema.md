# `.devrites/` state schema (v1)

The `devrites-engine` binary reads a project's workflow state from plain files under
`.devrites/`. Those files are the source of truth and are hand-editable — a
human edit always wins.

`schemaVersion: 1`.

For the layered load-order diagram, budget table, manifest/alias model, and
phase-required section matrix, see [`workspace-schema.md`](workspace-schema.md).

## Layout

```
.devrites/
  work/
    <slug>/
      README.md       workspace map — phase, status, next action, read-next
      brief.md        objective, non-goals, success definition
      spec.md         product WHAT/WHY, requirements, acceptance, boundaries
      architecture.md feature technical map
      plan.md         approach and slice strategy
      tasks.md        SLICE-### vertical slices
      traceability.md AC/REQ -> slices -> tests/evidence/files
      decisions.md    DEC-### decision log
      assumptions.md  assumption register
      questions.md    Q-### question register
      state.md        compact cursor
      evidence.md     EVID-### command/action proof
      touched-files.md implementation file map
  specs/              capability ledger — the living "what the system does now"
    <capability>/
      spec.md         proven Requirement blocks for this capability
```

Each feature is a directory of **small single-concern files**. Splitting the
work this way (rather than one long document) keeps every file context-cheap and
makes completeness self-evident: an empty or missing section is visibly empty.

### Capability ledger — `specs/`

`work/<slug>/` is **ephemeral** (archived on ship); `specs/` is **durable**. It is
the cumulative record of proven behavior, one `spec.md` per capability, so a new
feature starts from the system's current contract instead of re-deriving it from
code. A feature's spec carries deltas (`## ADDED/MODIFIED/REMOVED Requirements —
capability: <c>`); on ship, `devrites-engine ledger sync` folds them in — ADDED
appends, MODIFIED replaces by header identity, REMOVED deletes. Living outside
`work/`, the ledger survives close-out's archival — and unlike the rest of
`.devrites/`, it is **git-tracked** (`.devrites/*` + `!.devrites/specs/`), so the
proven contract is shared, not per-clone. Grammar and delta rules:
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
**empty** — scaffolding a file never fakes completeness.

The 2026 Markdown workspace schema is stricter and lives in
[`workspace-schema.md`](workspace-schema.md): it validates required phase-owned
artifacts, stable IDs (`AC-###`, `SLICE-###`, `EVID-###`), traceability, evidence,
compactness budgets, Mermaid fences, and stale local links.

## Phases and required sections

Phases follow the rite-\* arc. Completeness is **phase-relative**: each phase
requires only the sections needed to leave it, and the set grows additively down
the arc. A section that is not yet required (e.g. `proof` during the `spec`
phase) never blocks.

| phase | required sections |
| --- | --- |
| `frame` | *(none)* |
| `spec`, `temper` | `spec` |
| `define`, `plan` | `spec`, `plan` |
| `vet`, `build`, `converge` | `spec`, `plan`, `decisions`, `tasks` |
| `prove`, `polish`, `review` | `spec`, `plan`, `decisions`, `tasks`, `proof` |
| `seal`, `ship`, `done` | `spec`, `plan`, `decisions`, `tasks`, `proof`, `status` |

The authoritative typed definitions live in `engine/internal/state/schema.go`.
`workflow_manifest.json` is a generated derivative for non-Go release tools;
run `go generate ./internal/state` after editing the registry.

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
  does not gate — gating is issue 04).
- Unknown or missing slug → non-zero exit with a clear message on stderr.

`status` makes no model or network calls; it is a pure read of the files under
`DEVRITES_ROOT` (or the nearest `.devrites/` above the working directory). A
hand edit wins immediately because there is no status cache. Other workspace
control-plane commands share that deterministic boundary; explicit
install/update/source-cache I/O is isolated under `engine/internal/iohooks` as
defined by ADR-0008.

`.devrites/` is ignored **except** the capability ledger at `specs/`, which is
committed shared truth — the recommended pattern is `.devrites/*` +
`!.devrites/specs/` (so `work/`, `archive/`, and `ACTIVE` stay per-clone runtime
state while `specs/` is tracked).

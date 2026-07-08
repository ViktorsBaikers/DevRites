# `.devrites/` state schema (v1)

The `devrites` engine reads a project's workflow state from plain files under
`.devrites/`. Those files are the source of truth and are hand-editable — a
human edit always wins. (A derived SQLite index is added in issue 02 purely as a
disposable navigator; it never overrides the files.)

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
`proof.md` may stand in for `evidence.md`.

### `feature.md`

The per-feature index. Its YAML frontmatter carries:

| field           | meaning                                        |
| --------------- | ---------------------------------------------- |
| `slug`          | feature identifier (matches the directory)     |
| `title`         | human-readable title                           |
| `phase`         | current workflow phase (see below)             |
| `schemaVersion` | schema version the file was written against    |

A feature exists iff its `feature.md` exists. A `feature.md` with no `phase`, or
an unknown `phase`, is an error.

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

| phase   | required sections                                     |
| ------- | ----------------------------------------------------- |
| `frame` | *(none)*                                              |
| `spec`  | `spec`                                                |
| `plan`  | `spec`, `plan`                                         |
| `build` | `spec`, `plan`, `decisions`, `tasks`                  |
| `prove` | `spec`, `plan`, `decisions`, `tasks`, `proof`         |
| `vet`   | `spec`, `plan`, `decisions`, `tasks`, `proof`         |
| `seal`  | `spec`, `plan`, `decisions`, `tasks`, `proof`, `status` |
| `ship`  | `spec`, `plan`, `decisions`, `tasks`, `proof`, `status` |

## `devrites status <slug>`

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

The engine makes **zero model or network calls**; `status` is a pure read of the
files under `DEVRITES_ROOT` (or the nearest `.devrites/` above the working
directory).

## Index (SQLite cache)

The files are the source of truth. `state.db` — a pure-Go SQLite database (WAL
mode) written under the `.devrites/` root — is a **disposable, gitignored
navigator** over them, never an authority.

- **`devrites reindex`** drops `state.db` and rebuilds it from the files,
  reporting how many features were indexed. Deleting `state.db` costs nothing.
- **`devrites status`** serves from the index. Before answering it
  staleness-checks the feature: each of its files is fingerprinted by name,
  size, mtime, and content hash, and any change (edit, add, remove) re-indexes
  that feature first — so a **hand-edit always wins** without an explicit
  reindex. Content is hashed, so a restored mtime can't hide an edit.
- The index **carries its own schema version** (SQLite `PRAGMA user_version`),
  independent of the state `schemaVersion`. A DB with a mismatched version, a
  missing schema, or an unreadable/corrupt file is **dropped and rebuilt, never
  trusted**.
- If the index can't be used at all, `status` **fails open** to reading the
  files directly, so the cache being unavailable never breaks a read.

`state.db` and its `-wal`/`-shm` sidecars are gitignored. `.devrites/` is ignored
**except** the capability ledger at `specs/`, which is committed shared truth — the
recommended pattern is `.devrites/*` + `!.devrites/specs/` (so `work/`, `archive/`,
`ACTIVE`, and `state.db` stay per-clone runtime state while `specs/` is tracked).
Index-served `status` output is byte-identical to a files-only read.

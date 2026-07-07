# `.devrites/` state schema (v1)

The `devrites` engine reads a project's workflow state from plain files under
`.devrites/`. Those files are the source of truth and are hand-editable — a
human edit always wins. (A derived SQLite index is added in issue 02 purely as a
disposable navigator; it never overrides the files.)

`schemaVersion: 1`.

## Layout

```
.devrites/
  features/
    <slug>/
      feature.md      index — frontmatter carries the fields the engine keys on
      spec.md         outcome, scope, constraints
      plan.md         approach / implementation plan
      decisions.md    decision log
      tasks.md        task breakdown with status
      proof.md        acceptance criteria / evidence
      status.md       status checkpoint
```

Each feature is a directory of **small single-concern files**. Splitting the
work this way (rather than one long document) keeps every file context-cheap and
makes completeness self-evident: an empty or missing section is visibly empty.

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

The six completeness sections, in canonical order:

`spec` · `plan` · `decisions` · `tasks` · `proof` · `status`

A section is **present** (has real content) when its `<section>.md` file exists
and, after removing any leading YAML frontmatter, ATX (`#`) headings, and
whitespace, some content remains. A stub that is only a heading counts as
**empty** — scaffolding a file never fakes completeness.

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

`state.db` and its `-wal`/`-shm` sidecars are gitignored (the repo-root
`.gitignore` already ignores the whole `.devrites/` directory). Index-served
`status` output is byte-identical to a files-only read.

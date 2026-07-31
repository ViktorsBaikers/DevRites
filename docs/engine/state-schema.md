# `.devrites/` state schema (v2)

The engine reads workflow state from plain files under `.devrites/`. Those
human-editable files remain authoritative; the engine has no database or
semantic cache.

<!-- authority:schema-version:start -->
`schemaVersion: 2`.
<!-- authority:schema-version:end -->

For root selection, supported legacy cursors, and the full artifact layout, see
[`workspace-schema.md`](workspace-schema.md).

## Layout

```text
.devrites/
  ACTIVE
  AFK
  specs/
    <capability>/spec.md
  work/
    <slug>/
      README.md
      brief.md
      spec.md
      architecture.md
      plan.md
      tasks.md
      traceability.md
      decisions.md
      assumptions.md
      questions.md
      decision-coverage.md
      eng-review.md
      test-plan.md
      state.md
      evidence.md
      touched-files.md
  archive/<slug>/
```

Each feature uses small files with one concern. The mutable `state.md` cursor is
the phase authority. `README.md` is the canonical map and `evidence.md` is the
canonical proof record.

### Capability specifications

`.devrites/specs/<capability>/spec.md` is the durable, Git-tracked record of
proven behavior. The active skill interprets accepted `ADDED`, `MODIFIED`, and
`REMOVED` deltas, edits the capability Markdown through the normal reviewed
workflow, and applies the normative native grammar re-read checklist. The Go
engine does not parse grammar, fold, or interpret capability deltas.

Grammar rules live in
[`spec-grammar.md`](../../pack/.claude/skills/devrites-lib/reference/standards/spec-grammar.md).

### Supported cursor readers

Official released cursors use one of two encodings in
`.devrites/work/<slug>/state.md`:

- v1.0.0–v2.6.1 bullet fields: `Phase`, `Next step`, and `qid`;
- v3 table fields: `phase`, `next_action`, and `question_id`.

Both readers are direct and non-mutating. `state.md` is required; other
pre-release encodings are not state authorities. The current writer emits the
canonical v3 table form.

## Logical completeness sections

The engine's logical sections, in canonical order, are:

| Section | Canonical file |
|---|---|
| `spec` | `spec.md` |
| `plan` | `plan.md` |
| `decisions` | `decisions.md` |
| `tasks` | `tasks.md` |
| `proof` | `evidence.md` |
| `status` | `state.md` |

A required file is present only when content remains after leading YAML
frontmatter, ATX headings, and whitespace are removed. A heading-only stub does
not satisfy a structural gate. Semantic quality is assessed separately by the
active skill and exact native agents.

## Phases and required sections

Completeness is phase-relative: a phase requires only the sections needed to
leave it, and the set grows through the lifecycle.

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

The typed definitions live in `engine/internal/state/schema.go`.
`workflow_manifest.json` is a generated derivative for non-Go release tools;
run `go generate ./internal/state` after changing the registry.

### Native clarification-return field policy

The controlling root owns these cursor edits and preserves the existing
table/bullet representation plus all unrelated Markdown.

| field | policy |
| --- | --- |
| `phase` | set to `clarify` on entry; restored from a validated later `return_phase` only after contract-neutral CLEAR |
| `status` | `running` on entry and restore |
| `next_action` | Clarify on entry; restored from validated `return_next_action` |
| `return_phase` | copied from the current recognized later phase; removed on restore |
| `return_next_action` | copied from the current non-empty action; removed on restore |
| `all other state.md content` | preserved |

Entry while already in Clarify is a no-op. A changed behavior or acceptance
contract routes through `/rite-plan repair` instead of restoring the saved
cursor.

### Native AFK and recovery accounting

For a configured or orchestrator-derived AFK cap, `afk_slices_remaining` is an
optional nonnegative cursor field. An orchestrator may pre-seed it from a validated
post-plan budget but never increase or reinitialize an existing value. The root then
charges it exactly once with each green pending → built
transition and stops before another dispatch at zero; malformed values fail
closed and values never go below zero. Released bullet state spells the same
field `AFK slices remaining`; native edits recognize it and preserve the
existing presentation.

There is no recovery-counter artifact. The caller and recovery loop count at
most three failed attempts per causal fingerprint from current context and the
recorded `## Dead ends` / `evidence.md`, then stop before a fourth.

<!-- authority:state-tracking:start -->
Git-tracked shared state: `.devrites/specs/`. Per-clone runtime state: `.devrites/work/`, `.devrites/archive/`, `.devrites/ACTIVE`.
<!-- authority:state-tracking:end -->

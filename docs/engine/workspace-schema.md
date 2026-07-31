# DevRites workspace schema

A feature workspace is the durable record under `.devrites/work/<slug>/`.
Chat or session history is not authoritative. The canonical per-phase files,
budgets, IDs, and read order live in
[`workspace-artifact-schema.md`](../../pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md).

## Root selection

- `DEVRITES_ROOT` selects a project root or `.devrites/` directory.
- Without it, the engine searches only inside the current physical Git
  repository boundary.
- `.devrites/ACTIVE` selects the default feature.
- `DEVRITES_WORKSPACE` may select one explicit contained workspace.
- Mutating commands refuse ambiguous, escaped, symlinked, or otherwise unsafe
  roots.

## Layout

```text
.devrites/
  ACTIVE
  AFK                         # optional run-mode sentinel
  specs/                      # living capability Markdown, maintained by skills
  work/
    <slug>/
      README.md               # canonical workspace map
      brief.md
      spec.md
      decisions.md
      assumptions.md
      questions.md
      decision-coverage.md
      architecture.md
      plan.md
      tasks.md
      traceability.md
      eng-review.md
      test-plan.md
      state.md                # authoritative mutable cursor
      evidence.md             # canonical proof record
      touched-files.md        # sole strict project-candidate manifest
      review.md               # closed-candidate review + digest binding
      seal.md                 # verdict + digest binding
      <conditional artifacts>
  archive/
    <slug>/
```

Conditional artifacts include strategy, design/browser evidence, drift,
handoff, references, flows, and other phase-specific Markdown named by the
canonical schema.

## Build-readiness binding

After Vet's fold-back and semantic recheck, `eng-review.md` contains exactly one
unindented standalone line:

```text
Readiness inputs SHA-256: <64 lowercase hex>
```

`devrites-engine check readiness --emit-binding <slug>` renders that line. The
identity binds raw bytes for `spec.md`, `decision-coverage.md`,
`architecture.md`, `plan.md`, `tasks.md`, `traceability.md`, and
`test-plan.md`; it also binds presence or absence and bytes for optional
`strategy.md`, `design-brief.md`, `ai-spec.md`, and project-root
`.devrites/principles.md`. It excludes Build-mutable ledgers and evidence.

Each input is limited to 1 MiB and the aggregate to 8 MiB; symlinks and
non-regular files fail closed. Ordinary readiness verifies the binding whenever
its target phase requires `eng-review.md`, and Seal verifies it again. A missing,
malformed, duplicate, or stale line routes through `/rite-vet`; replacing the
line without rerunning Vet is not review.

## Candidate manifest and bindings

`touched-files.md` contains exactly one `## Touched files` heading and exactly
one authoritative `## Candidate manifest` heading. `## Touched files` describes
the scope without repeating candidate paths; the manifest alone defines
candidate scope. Its body is the exact `No project files.` marker or a table of
unique project-relative UTF-8 `present`/`deleted` paths, owning slice, and
reason. Rows must already be strictly sorted by normalized `File` and paths
must also be unique under case folding. Portable components reject
Windows-reserved characters and names, including names with extensions, plus a
trailing dot or space. `## Review trail` is human navigation. The exact grammar
remains in the
[canonical artifact schema](../../pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md).

The fixed public limits are a 1 MiB manifest, 4,096 rows, a 4,096-byte path,
64 MiB per present file, and 256 MiB across present files. `.git/**` is never a
candidate path. Under `.devrites`, only exact `.devrites/principles.md` and
`.devrites/specs/**` may be candidates; `ACTIVE`, `AFK`, `CHECKPOINT`,
`archive/**`, `work/**`, and every other sibling fail closed. Other durable
project files such as `DESIGN.md` and `docs/adr/**` are candidates when changed.

`evidence.md`, `review.md`, and `seal.md`, plus `browser-evidence.md` when it
exists, each contain exactly one unindented standalone
`Candidate SHA-256: <64 lowercase hex>` binding. Run:

```text
devrites-engine check candidate <slug>
candidate-sha256: <64 lowercase hex>
candidate-files: <manifest row count>
```

The command exits `0` on those two output fields, `2` for usage/root selection,
or `3` with `candidate: BLOCKED: <reason>` for an invalid workspace, manifest,
path, file, or size limit. See [candidate integrity](../candidate-integrity.md).
The worktree digest is not an atomic filesystem snapshot against a malicious
concurrent same-size rewrite; Ship's Git-index checks own the final freeze.

## Supported cursor compatibility

The runtime reads only official released workspace formats:

- v1.0.0–v2.6.1 `.devrites/work/<slug>/state.md` bullet cursors, including
  `Phase`, `Next step`, and `qid`;
- v3 `.devrites/work/<slug>/state.md` table cursors, including `phase`,
  `next_action`, and `question_id`.

The canonical workspace location remains `.devrites/work/<slug>/`; canonical
map, cursor, and proof files are `README.md`, `state.md`, and `evidence.md`.
Compatibility reads do not rewrite a workspace and do not emit local telemetry.
There is no structural migration command. `/rite-upgrade` first audits an older
active workspace against named current contracts; only a cited defect may route
an edit through its normal phase owner. For released unfinished post-Build
workspaces, candidate defects route through current Prove, Polish, Review, and
Seal as applicable. Ambiguous historical scope is a gap; no owner synthesizes
an old pass.

Other pre-release layouts, filename substitutions, and phase encodings are
not runtime authorities. See
[ADR-0022](../adr/0022-native-orchestration-thin-engine.md) and
[ADR-0025](../adr/0025-evidence-gated-workspace-upgrades.md).

## Native policy state

- New question IDs are `q-YYYY-MM-DD-NNN`. The root scans all same-day headers,
  chooses the next unused suffix, re-reads immediately before append, and
  recomputes on collision. `state resolve` remains the atomic answer/drop/batch
  writer; it does not allocate IDs.
- The root directly owns later-phase Clarify entry/restore cursor edits and AFK
  slice accounting, preserving unrelated Markdown. See
  [`state-schema.md`](state-schema.md).
- Recovery has no counter artifact. Current context plus recorded Dead
  ends/evidence account for at most three failed attempts per causal
  fingerprint.

Timeline events, compatibility telemetry, migration journals, reviewer
statistics, dispatch receipts, model/provider data, and extension registries
are not workspace state.

## Readiness and proof ownership

The active skill and exact native agents own semantic readiness, traceability,
acceptance and evidence quality, doubt, review reconciliation, test-quality
assessment, capability interpretation, compatibility audit, and recovery routing.
They read the workspace and observed repository results directly.

The engine provides only:

- `check candidate`: strict candidate-manifest validation and content-bound
  identity;
- `check readiness`: phase-relative file completeness, open-human-gate check,
  and the stable vetted Build-input binding whenever `eng-review.md` is
  required;
- `check seal`: final file completeness and open-human-gate checks, then the
  readiness-binding recheck, then exact candidate bindings after that aggregate
  gate passes;
- atomic `state resolve` answer/drop/batch and transactional `state close`;
- secret scanning, version reporting, and local install lifecycle primitives.

A marker or file can satisfy structural presence without proving semantic
quality; native review, the normative spec grammar re-read, and repository proof
must establish that separately.

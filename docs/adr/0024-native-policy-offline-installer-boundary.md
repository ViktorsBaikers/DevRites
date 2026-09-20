# ADR-0024: Native policy and offline installer boundary

- **Status:** Accepted in part; offline engine-update clause superseded by 0028
- **Date:** 2026-08-01

## Context

ADR-0022 removed semantic orchestration from the Go engine but retained several
commands that encoded workflow policy: spec grammar parsing, question-ID
derivation, Clarify return transitions, AFK slice accounting, recovery counters,
and install diagnostics. ADR-0023 kept those commands because their deterministic
implementation appeared to make autonomy safer.

That split still created two authorities. The skills described when and why a
policy applied, while Go decided how Markdown should be interpreted or changed.
Each policy change required synchronized command code, fixtures, settings,
generated guidance, and prose. Recovery also acquired a separate JSONL counter
even though exact failures and dead ends already belonged in evidence.

Installation had a second boundary leak. The engine both applied manifest-owned
local changes and selected/downloaded remote update material. This made a
deterministic local installer depend on network policy already handled by the
npm and shell entrypoints.

## Decision

- Keep the engine only for deterministic primitives whose safety depends on a
  shared executable boundary: local install/update/uninstall, structural
  readiness, structural plus evidence-freshness seal, atomic question
  answer/drop/batch resolution, transactional close, secret scanning, version,
  and schema/path/evidence safety.
- Remove `check spec`, `state clarify`, `state tick-afk`, `state recovery`,
  `state resolve next-qid`, and `doctor`. Removed forms fail visibly; there are
  no aliases or tombstones.
- Keep Requirement/Scenario grammar normative. At each owning gate, the root
  re-opens the saved spec and executes one explicit checklist for headings,
  unique IDs, SHALL/MUST, WHEN/THEN, scenarios, behavior anchoring, and delta
  identity. Do not add a replacement parser or script.
- Allocate `q-YYYY-MM-DD-NNN` in the root: scan every same-day question header,
  select the next unused suffix, then re-read and recompute immediately before
  append. Retain `state resolve` as the sole atomic answer/drop/batch writer.
- Manage later-phase Clarify return fields in the root. Entry preserves the
  current recognized later `phase` and non-empty `next_action`, then enters
  Clarify in running state. Entry while already in Clarify is a no-op. A fresh,
  contract-neutral CLEAR validates, restores, and removes both return fields
  while preserving unrelated Markdown. Changed behavior routes through plan
  repair instead of restoring.
- Manage AFK slice budgets in the root. A configured budget is a nonnegative
  integer, charged exactly once with each green pending-to-built transition,
  never decremented below zero, and checked before another dispatch. Malformed
  state fails closed.
- Count recovery in the caller plus recovery loop. One causal fingerprint gets
  at most three total failed attempts from the current context and recorded
  Dead ends/evidence. Record each exact failure and failed idea; never run a
  fourth. Do not create a counter file or command.
- Keep `/rite-doctor` as a read-only native procedure. It locates the repository
  root; inspects manifest hashes, installed host artifacts/config, ACTIVE and
  workspace topology, and symlinks; compares manifest/package/available-binary
  versions; and reports OK/WARN/FAIL with remediation. It never mutates or
  acquires an engine.
- Make Go install/update/uninstall offline and local. Shell/npm entrypoints
  acquire and verify the bundle/source/platform binary, then supply those local
  candidates to the engine. Engine update `--check` compares installed and local
  candidate versions. Engine `--to` and `--pre` release selectors are removed.
- Remove permissions for deleted command forms from installed host settings.

## Superseded decisions

- **ADR-0001:** supersedes the claim that the engine owns *all* deterministic
  state transitions and derivations, including next-question ID bookkeeping.
  Its stdlib-only binary, model-free trust boundary, and retained deterministic
  primitives stand.
- **ADR-0006:** supersedes the next-qid clock-seam decision and its dedicated
  guard test because that engine form no longer exists. Static-analysis, race,
  vulnerability, and applicable clock-seam rules stand.
- **ADR-0008:** supersedes the allowance for updater/source-cache network I/O in
  the engine. Acquisition now belongs to shell/npm edges; the engine is fully
  network-free. Its no-model and auditable-boundary intent stands.
- **ADR-0022:** narrows its retained command list by removing spec, Clarify, AFK,
  recovery, qid-allocation, and doctor commands, and narrows install/update to
  caller-supplied local candidates. Native orchestration, released cursor reads,
  root safety, exact writer permissions, and fresh irreversible approval stand.
- **ADR-0023:** supersedes its decision to retain AFK/recovery counters and the
  engine doctor. Direct workspace reads and line-oriented output for remaining
  checks stand.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep the commands because they are deterministic | Determinism alone does not justify a second policy authority or the synchronized surface it creates. |
| Replace each command with a shell/Python helper | Preserves the duplicate parser/counter problem with weaker portability and root safety. |
| Move atomic resolve/close and secret scanning into prose too | Those operations protect multi-file consistency or a trust boundary and still justify executable enforcement. |
| Let the engine fetch only updates while npm/shell fetch installs | Keeps two acquisition policies and makes offline behavior command-dependent. |

## Consequences

The public engine surface and Claude allowlist shrink. Skills must state the
native policy once, link to it, and tests must reject deleted forms in canonical
and generated artifacts. Spec grammar remains a blocking contract even without
a parser. AFK and recovery bounds are instruction-enforced and therefore must be
recorded and re-read explicitly rather than inferred from exit codes.

Install/update/uninstall can be tested without network access. npm and shell
entrypoints carry acquisition responsibility and must checksum remote artifacts
before passing local paths to the engine. Existing manifests and released
workspace cursor encodings need no migration.

Regression coverage lives in `tests/native-orchestration-contract-test.sh`,
`tests/phase-gate-routing-test.sh`, `tests/install-smoke.sh`,
`tests/host-artifacts-test.sh`, and the Go install/root-routing tests.

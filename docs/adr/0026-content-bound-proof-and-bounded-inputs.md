# ADR-0026: Content-bound proof and bounded inputs

- **Status:** Accepted
- **Date:** 2026-08-01

## Context

At baseline `403e1a60adf3ae32ee88009c417e6ffe015fc1aa`, Seal decides
evidence freshness from modification times. It extracts every backtick-delimited
token from `touched-files.md`, accepts absolute paths, ignores listed files that
do not exist, and treats a missing `touched-files.md` as fresh. That comparison
cannot establish which bytes were proved. A restored timestamp can hide a content
change, while an unchanged file touched by tooling can invalidate sound evidence.

The approved candidate can also change after Seal. Ship currently performs
capability-ledger folding, design-memory promotion, and durable ADR promotion.
Those writes occur after proof and review, so the committed tree can differ from
the candidate those phases inspected.

Two input boundaries have the same identity problem. Git subprocesses inherit
repository, object, ref, config, and pathspec environment variables that can
retarget a command despite an explicit working directory. Release acquisition
accepts unbounded responses and retains unchecked fallbacks; archive extraction
does not first prove one bounded, expected file tree. Checksums alone do not make
an ambiguous target or an unbounded archive safe.

A fresh audit of nine MIT-licensed workflow repositories found useful integrity
mechanisms, but no reason to replace DevRites' lifecycle or control-plane split.
ADR-0024 remains the authority for native semantic policy and the offline,
stdlib-only Go engine. This decision adds one deterministic serialization
primitive and hardens existing owners.

## Decision

### Preserve the existing architecture

- Keep the 15 phases from Frame through Done, the public `/rite-*` commands,
  exact named agents, one exact-path writer, the Markdown workspace, and the
  offline stdlib-only Go control plane.
- Add no phase, skill, agent, daemon, registry, scheduler, dependency, or second
  state plane.
- Amend ADR-0024 with one additive, read-only, line-oriented command:
  `devrites-engine check candidate <slug>`. The command and Seal call the same Go
  helper. The helper serializes and validates candidate identity; it does not
  interpret specifications, proof quality, or workflow policy.
- Final `check seal` also performs exact-line identity checks. It requires the
  same candidate SHA-256 in `evidence.md`, `review.md`, and `seal.md`, plus
  `browser-evidence.md` when that file is present. Reading those fixed digest
  lines is deterministic binding, not semantic review or proof interpretation.

### Make the manifest the sole candidate authority

- `touched-files.md` owns one machine-readable `## Candidate manifest` section.
  Each project-relative path has exactly one explicit `present` or `deleted`
  state. A candidate with no project files uses one explicit no-project-files
  marker.
- An absent or malformed manifest blocks Candidate, Seal, and later Git checks
  with a rerun or workspace-refresh message. There is no heuristic parser or
  legacy fallback.
- Normalize and sort manifest paths before serialization. Reject absolute paths,
  traversal, duplicate normalized paths, control characters, ambiguous paths,
  missing `present` files, existing `deleted` files, directories, symlinks,
  special files, and any manifest, entry, file, or aggregate size beyond the
  fixed documented limits.
- Candidate identity is a versioned, length-prefixed, streaming SHA-256. Every
  record binds state, normalized path, file type, executable bit, content length,
  and content bytes. Deleted rows use the format's fixed absent type/mode/content
  values. Length prefixes and a version marker prevent concatenation ambiguity.
- Prove records the candidate digest in `evidence.md` and every applicable
  `browser-evidence.md`. Review and Seal record the exact same digest in
  `review.md` and `seal.md`. Unchanged content remains valid across
  modification-time changes. Any candidate byte, path, state, type, or
  executable-bit change blocks until proof and downstream identity records are
  refreshed.

### Close the candidate before Review

- Initial Prove may precede Polish. Polish performs every candidate-affecting
  capability-ledger fold, UI design-memory update, and durable ADR rollup, then
  closes the candidate manifest. Any resulting digest change requires affected
  re-proof and refreshed evidence digest before Review.
- Review and Seal inspect that closed candidate. Ship may not change candidate
  files before its Git ladder. Any needed candidate mutation routes back through
  Prove, Review, and Seal.
- Ship rechecks the manifest plus the matching evidence, browser when present,
  review, and seal digest lines before Git. Before commit, the staged candidate
  must match the approved manifest and digest. Before push or tag, the committed
  candidate must match as well. These checks supplement, and never replace,
  fresh human approval for irreversible Git actions.

### Sanitize Git targeting in one shared policy

- A shared Go helper filters the environment for both production Go Git
  subprocess sites. A shared shell helper is sourced by every production shell
  Git site.
- Remove repository, config, object, ref, and pathspec-retargeting `GIT_*`
  variables. Preserve unrelated Git variables rather than clearing the entire
  namespace.
- Treat each dynamic `GIT_CONFIG_KEY_n` and `GIT_CONFIG_VALUE_n` as one pair.
  Filtering, preservation, and malformed-pair behavior must stay identical in Go
  and shell and have parity tests.

### Bound and verify every network acquisition

- Acquire only an exact release asset with its mandatory SHA-256 sidecar. Remove
  unchecked tag, default-branch, source-archive, and raw-script fallbacks.
- Bound API metadata, checksum, archive, and binary transfers. Fail if secure
  private temporary-directory creation fails; never fall back to a predictable
  path.
- Before extraction, verify exactly one expected archive prefix, allowed entry
  types, member-count and expanded-size ceilings, and path containment. Extract
  only after the complete preflight succeeds.
- Release production must generate every required checksum sidecar and fail when
  one is absent. Keep the curl-plus-tar bootstrap path and add no dependency.

### Strengthen the native semantic owners

- Capability folding preserves all prior MODIFIED scenarios and source claims
  unless the change explicitly replaces them. A change with no capability impact
  uses a justified, explicit no-impact form.
- When a slice changes an API, event, or schema boundary, provider and consumer
  tests must read the same canonical contract artifact. This rule is conditional;
  it creates no contract registry for slices without such a boundary.
- Proof requires positive observable or framework evidence. Skipped, vacuous,
  compile-only, assertion-free, or non-executed checks cannot prove behavior.

### Repair the two baseline gates narrowly

- Classify the literal `devrites-orchestrator` profile as a non-skill in the
  invocation-integrity check.
- Revalidate GHSA-mh99-v99m-4gvg against the installed dependency graph. Remove
  its exact temporary exception when a patched ancestor is available; otherwise
  retain only the exact affected path with a short, justified expiry.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep modification times as evidence identity | Timestamps neither bind bytes nor fail closed on missing paths and manifests. |
| Let each host compute its own candidate hash | Independent Claude, Codex, shell, and Go implementations would create multiple proof authorities and parity drift. |
| Keep capability, design-memory, and ADR writes in Ship | The committed candidate could differ from the one Prove, Review, and Seal approved. |
| Keep unchecked tag, branch, source, or raw-script fallbacks | Availability would silently take priority over provenance and make release verification optional. |
| Add a workflow engine, registry, scheduler, daemon, or second state store | Each duplicates an existing native host, lifecycle, or Markdown owner and adds failure modes without closing the observed integrity gaps. |
| Adopt any reference project wholesale | Their useful mechanisms are separable; their runtimes, permissions, state models, and update policies conflict with DevRites invariants. |
| Hash the Git index or worktree implicitly | Candidate scope would depend on ambient Git state and could include preserved user work that the slice never owned. |

## Consequences

Released `/rite-*` commands and workspace prose remain readable, and `check
candidate` is additive. Old touched, evidence, review, and seal records do not
satisfy the new fail-closed identity contract; an unfinished workspace must
refresh its manifest and downstream digest lines before a new Seal. This is a
deliberate compatibility cost.

Proof becomes stable under harmless timestamp changes and sensitive to every
material candidate change. Streaming and fixed bounds cap memory, disk, and
archive work. The cost is a versioned serialization format, migration guidance,
and shared Go/shell parity that release work must maintain.

The planned guard locations are
`engine/internal/lib/candidate_test.go`,
`engine/internal/lib/evidencefresh_test.go`,
`engine/internal/gate/gate_test.go`, `engine/root_routing_test.go`,
`engine/tests/parity_githelpers_test.go`,
`tests/native-orchestration-contract-test.sh`,
`tests/phase-gate-routing-test.sh`, `tests/install-smoke.sh`,
`tests/update-smoke.sh`, `tests/npx-pack-smoke.sh`, and
`tests/release-tarball-test.sh`. These are planned regression locations, not a
claim that the implementation or tests already exist or pass.

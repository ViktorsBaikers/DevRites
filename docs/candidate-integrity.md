# Candidate integrity

DevRites proves one explicit project candidate rather than an ambient Git diff.
The candidate is the strict manifest in
`.devrites/work/<slug>/touched-files.md`; its content-bound identity is computed
only by `devrites-engine`.

The canonical grammar and artifact budgets live in
[`workspace-artifact-schema.md`](../pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md).
The architectural rationale is [ADR-0026](adr/0026-content-bound-proof-and-bounded-inputs.md).

## Manifest

`touched-files.md` contains exactly one `## Touched files` heading and exactly
one authoritative `## Candidate manifest` heading. `## Touched files` describes
the scope without repeating candidate paths; the manifest alone owns candidate
scope. Its body is either the exact line `No project files.` or a table whose
rows are sorted by `File`:

```markdown
| State | File | Slice | Reason |
| --- | --- | --- | --- |
| present | `path/to/file` | SLICE-001 | Observable reason |
```

`State` is `present` or `deleted`. Paths are project-relative UTF-8, already
strictly sorted by normalized `File`, unique both exactly and under case
folding, portable, and wrapped in one backtick pair. Components reject
Windows-reserved characters and device names even when followed by an
extension, plus a trailing dot or space. The optional `## Review trail` can
point a human to `path:line` stops, but it cannot add to or redefine candidate
scope.

Repository-internal paths are never candidates. Under `.devrites`, only the
exact durable owners `.devrites/principles.md` and `.devrites/specs/**` are
admitted; all other siblings fail closed, including:

- `.devrites/work/**`
- `.devrites/ACTIVE`
- `.devrites/AFK`
- `.devrites/CHECKPOINT`
- `.devrites/archive/**`
- `.git/**`

Other durable project files are candidates when the feature changes them,
including `DESIGN.md` and `docs/adr/**`.

## Check and bindings

Run:

```bash
devrites-engine check candidate <slug>
```

A pass exits `0` and prints exactly two line-oriented fields:

```text
candidate-sha256: <64 lowercase hex>
candidate-files: <manifest row count>
```

Invalid usage or root selection exits `2`. An invalid slug/workspace, malformed
manifest, unsafe path, missing `present` file, existing `deleted` file, rejected
file type, or size breach exits `3` with `candidate: BLOCKED: <reason>` on
stderr. Fix the reported workspace or candidate; phases do not reinterpret a
rejected manifest.

The digest is recorded exactly once, as an unindented standalone line, in
`evidence.md`, `review.md`, and `seal.md`, plus `browser-evidence.md` when that
file exists:

```text
Candidate SHA-256: <64 lowercase hex>
```

The digest binds its domain and version plus every row's state, normalized path, regular/absent type,
executable bit, byte length, and content. Content drift therefore blocks even
when modification times are restored; a harmless touch with unchanged content
does not.

This worktree digest is not an atomic filesystem snapshot against a malicious
concurrent same-size rewrite. It detects ordinary drift and bounded type/size
changes; Ship's exact Git-index scope, byte, binding, and secret checks own the
final freeze immediately before commit.

## Lifecycle ownership

| Phase | Candidate responsibility |
|---|---|
| Build | Maintain the manifest from each green slice's actual scoped diff. |
| Prove | Check before and after approved commands, require the same digest, and bind real evidence (plus browser evidence when present). |
| Polish | Apply every code/UI correction, capability-ledger fold, design-memory update, and durable ADR promotion; update the manifest and rerun affected proof before closing the candidate. |
| Review | Review the closed bytes and bind `review.md` to their digest. A correction returns through affected Prove and a fresh Review. |
| Seal | Bind `seal.md`, recheck every binding, and run `check seal`. A correction returns through Prove and Review. |
| Ship | Remain candidate-read-only. Only ship/state/archive bookkeeping may change; candidate or manifest changes return through Prove, Review, and Seal. |

Capability-ledger, `DESIGN.md`, and ADR rollups happen in Polish before Review,
not in Ship.

## Git scope at Ship

Before type-`GO`, Ship is read-only: it protects existing staged work, verifies
the candidate, Seal, bindings, and worktree, analyzes checkpoint history, and
discloses the exact optional collapse and stage commands. Only literal `GO`
permits those disclosed index/history mutations. Ship then compares the
NUL-delimited `git diff --cached --name-status --no-renames -z` set with the
manifest: `present` maps only to `A` or `M`, and `deleted` only to `D`. Missing,
extra, renamed, duplicated, differently classified, or unstaged candidate
changes block. Candidate, Seal bindings, staged bytes, and the staged secret
scan all rerun immediately before commit; drift invalidates the one-use approval.

Immediately after the authorized commit and before push or tag, Ship compares
the commit's exact state/path set and bytes with the same manifest, then reruns
Candidate and Seal. A mismatch stops; it is never reinterpreted as equivalent.
Fresh literal `GO` and native host approval remain required for each disclosed
Git attempt.

## Older workspaces

Released v1/v2/v3 unfinished workspaces use read-only `/rite-upgrade`. A missing
or malformed manifest or binding is a current-contract defect, not permission to
guess historical bytes. When legacy scope, the live diff, tasks, and traceability
agree unambiguously, Upgrade routes current owners through Prove, Polish, Review,
and Seal as applicable. They run fresh real proof and never synthesize an old
pass. Ambiguous legacy scope remains a `gap` and stops without mutation.

## Limits

The engine accepts a manifest of at most 1 MiB and 4,096 rows. A path may be at
most 4,096 bytes, each present regular file at most 64 MiB, and all present
files together at most 256 MiB. Directories, symlinks, special files, traversal,
absolute paths, duplicates, case-fold collisions, reserved components, and
control characters fail closed. The engine proves deterministic identity and
exact bindings, not the semantic quality of a specification, test, review, or
release decision. See the
[canonical schema](../pack/.claude/skills/devrites-lib/reference/workspace-artifact-schema.md)
for current limits and [`SECURITY.md`](../SECURITY.md) for trust boundaries.

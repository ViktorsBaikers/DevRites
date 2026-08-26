# `devrites-engine`: deterministic workspace core

DevRites keeps durable workflow truth in `.devrites/` Markdown. Claude Code and
Codex interpret the workflow, run exact custom agents, reconcile their results,
and make semantic judgments. The Go engine exposes only deterministic,
cross-host primitives; it never dispatches an agent or grades reviewer prose.

## Command surface

`devrites-engine help` lists the exhaustive operational surface. Standard
`help`, `-h`, `--help`, `version`, and `--version` forms also remain supported:

```text
devrites-engine install [flags]
devrites-engine update [flags]
devrites-engine uninstall [flags]

devrites-engine check candidate <slug>
devrites-engine check readiness <slug>
devrites-engine check readiness --emit-binding <slug>
devrites-engine check seal <slug>

devrites-engine state resolve <qid> "<answer>"
devrites-engine state close <slug>

devrites-engine secret-scan [--staged] [--stdin] [slug]
devrites-engine version
```

Commands outside this operational list and the standard help/version forms are
unsupported. There are no legacy engine aliases, tombstones, agent-protocol
versions, semantic-readiness digests, compatibility telemetry, or workspace
migration command.
`check candidate` is additive; no existing engine command or public `/rite-*`
workflow was removed or renamed.

## Updates

Run `devrites-engine update` in an installed project to resolve the latest
stable release and update the project pack plus shared engine binary. The
command downloads the release bundle and platform engine, verifies both exact
filenames against their SHA-256 sidecars, then hands the extracted candidate to
the downloaded engine. This avoids asking an old engine to validate a newer
payload schema.

`devrites-engine update --check` resolves and compares release metadata without
downloading assets. `--source-dir` and `--payload-dir` remain the advanced local
update path for npm, shell, source-checkout, and test callers; a local `--check`
needs only `--source-dir`.

## npm adapter

The `npx devrites` adapter accepts `add` as an alias for `install`, `upgrade` for
`update`, and `remove` for `uninstall`. These are adapter-only conveniences, not
`devrites-engine` aliases; documentation and automation should prefer the
canonical command names.

Use `npx devrites update --check` to perform the equivalent adapter-level
comparison without changing the installation. Use
`npx devrites uninstall --keep-binary` to remove managed host artifacts while
retaining the shared `devrites-engine` binary.

If an older direct engine reports `missing codex/hooks.json` or asks for
`--source-dir`, run `npx devrites@latest update`, or use a checksum-verified
release `install.sh` and run `bash ./install.sh update` once to install a release
with the self-contained updater.

## Checks

- `check candidate <slug>` validates the strict `touched-files.md` manifest and
  hashes its exact path/state/type/mode/content identity. A pass prints:
  ```text
  candidate-sha256: <64 lowercase hex>
  candidate-files: <manifest row count>
  ```
- `check readiness <slug>` verifies the files required to leave the workspace's
  current phase, the `tasks.md` slice graph once that artifact is required, and,
  once `eng-review.md` is required, its exact stable Build-input binding.
- `check readiness --emit-binding <slug>` renders the exact stable Build-input
  binding for Vet to record after semantic review.
- `check seal <slug>` checks final required files, open human gates, and the
  `tasks.md` slice graph. Once those files are complete, it verifies the stable
  readiness binding; only after that aggregate check passes does it verify that
  `evidence.md`, `review.md`, `seal.md`, and optional `browser-evidence.md`
  contain exactly one binding to the current candidate digest.

Semantic readiness, traceability, acceptance interpretation, evidence quality,
doubt, test quality, reviewer reconciliation, and capability interpretation
belong to the active skill and exact native agents. Normative spec grammar is
checked by the root's explicit native re-read checklist. Repository build,
test, lint, typecheck, schema, and release commands belong to that repository
or CI.

## Atomic state operations

`state resolve` also supports `--drop <qid> ["<reason>"]` and `--batch <file>`.
`state close` transactionally archives a shipped workspace and clears matching
`ACTIVE`.

The mutating forms share the engine's root-safety, locking, and atomic-write
boundary.

The root scans and rechecks `questions.md` to allocate the next unused qid,
manages Clarify return fields without rewriting unrelated Markdown, charges an
AFK budget once per green built slice, and counts no more than three recovery
failures per causal fingerprint from context plus Dead ends/evidence. `/rite-doctor`
is a read-only native inspection. None has an engine command or replacement
script.

## Secret scanning

Use `secret-scan --staged` for exact Git index blobs and `secret-scan --stdin`
for review text supplied through a non-logging process-stdin channel. The flags
may be combined. The scanner does not follow worktree symlinks or print matched
secret bytes. It accepts at most 4,096 entries, 64 MiB total captured input, and
4,096 findings; input, limit, and output failures close the scan.

Findings include only severity, a redacted or escaped source label, category,
and zero-based byte offset. HIGH findings exit `3`.

## Output and exits

Lifecycle checks print stable line-oriented fields. `reason: DRV-...` identifies
the deterministic gate outcome without an agent protocol or versioned wrapper.
Skills read workspace Markdown directly.

`check candidate` uses the two exact fields shown above. Invalid usage or root
selection exits `2`; a malformed/unsafe manifest or candidate mismatch prints
`candidate: BLOCKED: <reason>` to stderr and exits `3`.

- `0`: passed or completed.
- `2`: common invalid request or unreadable-state result.
- `3`: common deterministic lifecycle or safety block.
- Atomic state operations retain their documented operation-specific nonzero
  results.

## Root safety

`DEVRITES_ROOT` selects a project root or `.devrites/` directory.
`DEVRITES_WORKSPACE` may select one explicit contained workspace. Mutating
commands refuse ambiguous, escaped, symlinked, or otherwise unsafe roots.

Install, update application, and uninstall remain manifest-owned local
operations. Direct engine update, shell, and npm may acquire a release candidate;
all remote acquisition uses exact SemVer, HTTPS-only redirect hops, mandatory
exact-filename SHA-256 sidecars, private temporary directories, and byte/archive
bounds, with no unchecked raw/source/default-branch fallback. Update accepts
`--check` to compare the installed manifest version with the latest stable
release without downloading assets. Remote-selector flags such as `--to` and
`--pre` are unsupported.
`/rite-upgrade` is a native compatibility audit, not an engine migration. Its
read-only planner must cite a current rule and exact workspace defect; admitted
repairs run through the current Clarify, Plan, Converge, Vet, Prove, Polish,
Review, or Seal owner. Candidate repairs rerun current proof and never infer a
historical pass.

Production Go and shell Git sites remove repository/config/object/ref/pathspec
retargeting `GIT_*` variables before execution while preserving unrelated Git
environment. This targeting isolation is a caller boundary, not another engine
command.

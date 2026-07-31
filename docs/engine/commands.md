# `devrites-engine` commands

The engine is a deterministic, stdlib-only control plane. It has no model or
provider dependency and does not dispatch agents, interpret reviews, or choose
workflow strategy.

## Complete command inventory

| Command | Deterministic responsibility |
|---|---|
| `install [flags]` | Install manifest-owned host artifacts and the optional shared binary. |
| `update [flags]` | Refresh an existing managed installation. |
| `uninstall [flags]` | Remove managed artifacts while preserving runtime workspace state. |
| `check candidate <slug>` | Validate the strict manifest and compute the content-bound project-candidate identity. |
| `check readiness <slug>` | Check phase files, human gates, and the current stable Build-input binding when applicable. |
| `check readiness --emit-binding <slug>` | Render the exact stable Build-input binding for Vet to record after review. |
| `check seal <slug>` | Check final files, human gates, the stable Build-input binding, and exact candidate bindings. |
| `state resolve <qid> "<answer>"` | Resolve an open question and update `questions.md` plus `state.md` atomically. |
| `state close <slug>` | Archive a shipped workspace and clear matching `ACTIVE`. |
| `secret-scan [--staged] [--stdin] [slug]` | Scan exact staged blobs, stdin, or touched regular files for credential material. |
| `version` | Print the engine version. |

`help` is exhaustive. Unlisted command forms are rejected as unknown; the
engine has no compatibility aliases or tombstones.

## Check boundary

The candidate gate validates and hashes path/state/type/mode/content identity;
it does not infer scope from Git. The readiness gate checks structure plus the
exact stable Build-input binding after Vet. The seal gate repeats that binding
and checks exact candidate bindings in evidence, optional browser evidence,
review, and seal. None judges the meaning of `CLEAR`/`READY` prose,
parses reviewer narratives, infers acceptance coverage, counts assertions,
interprets capability deltas, or decides whether a technical plan is sound.

Those judgments are made by the current skill and exact native roles, including
`devrites-plan-reviewer`, `devrites-proof-runner`, `devrites-spec-reviewer`,
`devrites-test-analyst`, and `devrites-doubt-reviewer`. The root reconciles their
reports against live artifacts and observed repository proof.

## State

State mutations use the shared physical-root checks, feature lock, and atomic
write path. `state resolve` additionally supports `--drop` and `--batch`;
`state close` owns transactional archive plus `ACTIVE` clearing.

Normative spec grammar checks, qid allocation, Clarify cursor transitions, AFK
slice accounting, recovery attempt accounting, and installation diagnostics are
explicit root-owned native procedures. The workflow owns reproduction,
hypothesis ranking, tool selection, and routing. No replacement scripts or
counter artifacts are introduced.

## Offline install boundary

The engine's install/update/uninstall operations accept an already-local source,
pre-generated host payload, and optional staged binary. They never select a
release or download a bundle/binary. Shell and npm entrypoints own acquisition
and checksum verification, then invoke the local deterministic operation.
`update --check` compares the installed manifest version with the supplied
local candidate. `--to` and `--pre` are not supported engine flags.

Remote acquisition is exact-SemVer, HTTPS-only at every redirect, bounded, and
requires an exact-filename SHA-256 sidecar. Archive preflight completes before
extraction and unchecked raw/source/default-branch fallbacks are absent.

## Secret scanning

`secret-scan --staged` enumerates changed index entries and reads their exact
blob object IDs with replacement objects disabled. It does not substitute
working-tree bytes or follow a worktree symlink. `--stdin` reads supplied text
from process stdin; callers must not put that text in argv, environment, command
logs, here-documents, or temporary files.

Each invocation accepts at most 4,096 entries, 64 MiB total captured input, and
4,096 findings. Findings never include matched bytes, excerpts, or value hashes.
Input, limit, and output errors exit `2`; HIGH findings exit `3`.

## Output and exit contracts

`check candidate` passes with exactly:

```text
candidate-sha256: <64 lowercase hex>
candidate-files: <manifest row count>
```

Invalid usage/root selection exits `2`; a candidate validation block prints
`candidate: BLOCKED: <reason>` and exits `3`.

`check readiness --emit-binding <slug>` passes with exactly:

```text
Readiness inputs SHA-256: <64 lowercase hex>
```

It binds the fixed records documented in the
[workspace schema](workspace-schema.md#build-readiness-binding), not mtimes or
ambient Git state. Ordinary readiness and Seal require that exact standalone
line in `eng-review.md`; stale input returns
`reason: DRV-GATE-READINESS-STALE` and routes through `/rite-vet`.

- `0`: passed or completed.
- `2`: common invalid request or unreadable-state result.
- `3`: common deterministic lifecycle or safety block.
- Atomic state operations retain their documented operation-specific nonzero
  results.

Lifecycle checks emit stable line-oriented fields, including a `reason: DRV-...`
identifier for the deterministic outcome. The native `/rite-doctor` workflow
emits its own human-readable OK/WARN/FAIL report. Neither surface introduces an
agent API or versioned wrapper.

Strict mutators resolve the physical root once and refuse unsafe symlinks,
nested-repository inheritance, ambiguity, or escapes. Repository source
validation belongs to `scripts/validate.sh` and CI, not an installed engine.

Production Go and shell Git callers remove environment variables that can
retarget the repository, worktree, index, objects, refs, config, or pathspec,
while retaining unrelated Git variables. This isolation is shared caller
policy, not another public command.

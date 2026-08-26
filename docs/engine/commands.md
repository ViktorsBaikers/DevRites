# `devrites-engine` commands

The engine is a deterministic, stdlib-only control plane. It has no model or
provider dependency and does not dispatch agents, interpret reviews, or choose
workflow strategy.

## Complete operational command inventory

| Command | Deterministic responsibility |
| --- | --- |
| `install [flags]` | Install manifest-owned host artifacts and the optional shared binary. |
| `update [flags]` | Refresh an existing managed installation. |
| `uninstall [flags]` | Remove managed artifacts while preserving runtime workspace state. |
| `check candidate <slug>` | Validate the strict manifest and compute the content-bound project-candidate identity. |
| `check readiness <slug>` | Check target-Phase files, open human gates from Clarify onward, the `tasks.md` slice graph when that artifact is required, and the current stable Build-input binding when applicable. |
| `check readiness --emit-binding <slug>` | Render the exact stable Build-input binding for Vet to record after review. |
| `check seal <slug>` | Check files required by target Phase `seal`, open human gates, the `tasks.md` slice graph, the stable Build-input binding, and exact candidate bindings. |
| `check path-disjoint [--root <dir>] [<json-file> | -]` | Verify slice path sets are pairwise disjoint. |
| `check task-graph <slug>` | Validate `tasks.md` slice dependency graph for cycles, unknown deps, malformed tokens, duplicate IDs, missing `Dependencies`, and `depends_on` mismatch. |
| `check skill-trust <path>` | Scan one skill/agent Markdown file for structural trust violations. |
| `observe summary <slug>` | Emit sanitized JSON workspace summary from one retained observation. `task_graph.ok` is true iff `task_graph.problems` is empty; `problems` lists cycle, unknown-dep, malformed-token, duplicate-id, and missing-`Dependencies` blockers. |
| `state resolve <qid> "<answer>"` | Resolve an open question and update `questions.md` plus `state.md` atomically. |
| `state close <slug>` | Archive a shipped workspace and clear matching `ACTIVE`. |
| `secret-scan [--staged] [--stdin] [slug]` | Scan exact staged blobs, stdin, or touched regular files for credential material. |
| `open-visual <path-or-name> [--slug <slug>] [--no-open]` | Resolve a local visual HTML file, optionally open it in the OS browser, warn if the sibling outline is missing or inventory ids are absent from HTML, and print agent path tips. No network. |
| `version` | Print the engine version. |

`help`, `-h`, and `--help` print this operational inventory. `version` and
`--version` print the binary version. Other unlisted command forms are rejected
as unknown; the engine has no compatibility aliases or tombstones. The
`add`/`upgrade`/`remove` conveniences belong only to the `npx devrites` adapter,
not to the engine command namespace.

## Check boundary

The candidate gate validates and hashes path/state/type/mode/content identity;
it does not infer scope from Git. The readiness gate checks target-Phase
structure and applies open-question blocking only when that target is Clarify
or later, plus the exact `tasks.md` slice graph when `tasks.md` is required,
plus the exact stable Build-input binding after Vet. The seal gate
always targets Phase `seal`, repeats that graph and binding, and checks exact candidate
bindings in evidence, optional browser evidence, review, and seal. None judges
the meaning of `CLEAR`/`READY` prose,
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

## Install and update boundary

Install application and uninstall accept local source, pre-generated host
payload, and optional staged binary inputs. Direct `devrites-engine update`
selects the latest stable release, downloads its bundle and platform engine,
then invokes that downloaded engine with local candidate paths. Shell and npm
may instead acquire and pass the same local inputs. `update --check` compares
installed and latest release metadata without downloading assets. `--to` and
`--pre` are not supported engine flags.

Remote acquisition is isolated to the release boundary, exact-SemVer,
HTTPS-only at every redirect, bounded, and requires exact-filename SHA-256
sidecars. Archive validation completes during bounded extraction and unchecked
raw/source/default-branch fallbacks are absent.

## Secret scanning

`secret-scan --staged` enumerates changed index entries and reads their exact
blob object IDs with replacement objects disabled. It does not substitute
working-tree bytes or follow a worktree symlink. `--stdin` reads supplied text
from process stdin; callers must not put that text in argv, environment, command
logs, here-documents, or temporary files.

Each invocation accepts at most 4,096 entries, 64 MiB total captured input, and
4,096 findings. Findings never include matched bytes, excerpts, or value hashes.
Input, limit, and output errors exit `2`; HIGH findings exit `3`.

## Open visual

`open-visual` resolves `<path-or-name>` to a local `.html` file under the
active/`DEVRITES_WORKSPACE`/`--slug` workspace `visual/` directory, or via an
absolute/relative path. Missing sibling `.outline.md` warns on stderr but does
not hard-fail. When the outline exists, the engine compares `## ID inventory`
ids to HTML `id="..."` attributes and warns (non-fatal) for inventory ids
missing from HTML; HTML-only decorative ids are ignored. Unless `--no-open`,
the engine starts the OS opener (`open`, `xdg-open`, or Windows `start`) for
the local file only — never a network fetch. Stdout prints the absolute HTML
path, outline path tip, playbook index hint, and an `ids=ok` / `ids=mismatch`
summary when an inventory is present.

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

### Workspace observation diagnostics

Lifecycle checks acquire the fixed workspace Markdown inventory once. Each
artifact is classified as `absent`, `empty`, `malformed`, `unsafe`,
`unreadable`, or `present`. Retained content is limited to 1 MiB per file and
8 MiB aggregate. Diagnostic lines use this exact shape:
`artifact: <logical-path>: <state> (<code>)`.

The closed diagnostic codes and recoveries are:

| Code | Exact Gate recovery | Exact standalone readiness-binding payload |
| --- | --- | --- |
| `malformed_markdown` | `next: repair <logical-path>: replace invalid Markdown with valid Markdown; required artifacts need substantive content` | `readiness input <logical-path> is malformed (malformed_markdown); replace invalid Markdown with valid Markdown` |
| `parent_symlink` | `next: repair <logical-path>: replace the symlinked parent with a real directory` | `readiness input <logical-path> is unsafe (parent_symlink); replace the symlinked parent with a real directory` |
| `final_symlink` | `next: repair <logical-path>: replace the symlink with a regular file` | `readiness input <logical-path> is unsafe (final_symlink); replace the symlink with a regular file` |
| `non_regular` | `next: repair <logical-path>: replace the non-regular entry with a regular file` | `readiness input <logical-path> is unsafe (non_regular); replace the non-regular entry with a regular file` |
| `file_too_large` | `next: repair <logical-path>: reduce the file to at most 1 MiB` | `readiness input <logical-path> is unsafe (file_too_large); reduce the file to at most 1 MiB` |
| `permission_denied` | `next: repair <logical-path>: grant read permission` | `readiness input <logical-path> is unreadable (permission_denied); grant read permission` |
| `read_failure` | `next: repair <logical-path>: restore a readable regular file` | `readiness input <logical-path> is unreadable (read_failure); restore a readable regular file` |

The Gate recovery column remains exact for target-policy-required artifacts. For
a selected optional readiness input, the same code-specific repair appends
`; optional readiness input may instead be removed` and does not call the input
required.

These seven codes are the closed Workspace Observation classification and
recovery mapping outcomes. A selected public consumer emits only a code
reachable for its consumed fixed logical path. Invalid workspace ancestry is
`workspace_invalid`, not an artifact `parent_symlink` diagnostic.

Status emits diagnostics without recovery or `next:` lines, after section rows
and before `result`. Gate emits diagnostics after `reason` and before recovery,
`invariant`, and `retry` lines. Generic add-content recovery applies only to
absent or empty target-required artifacts. Standalone readiness-binding
failures use the existing `readiness-binding: BLOCKED:` prefix and the logical
readiness-input state/code plus recovery; they never disclose physical paths or
content.

Whole observation failures are `workspace_invalid`, `aggregate_too_large`, and
`concurrent_change`. Their disclosure-safe payloads are exact:

- `workspace observation: workspace_invalid: workspace is unavailable; verify the selected logical workspace and canonical workspace override, then retry`
- `workspace observation: aggregate_too_large: retained content exceeds the 8 MiB aggregate limit; reduce retained Markdown below 8 MiB, then retry`
- `workspace observation: concurrent_change: workspace changed during acquisition; retry`

An absent or empty `state.md` appends `add real content to state.md and retry` to
the existing logical error. A malformed, unsafe, or unreadable `state.md`
appends `repair state.md and retry`. A ledger without a phase appends `record
phase in state.md and retry`; an unknown phase appends `record a known phase in
state.md and retry`.

Whole observation failures use stderr, exit `2`, and no lifecycle result or
reason on stdout. Standalone readiness-binding failures use one stderr line,
exit `3`, and empty stdout. Per-artifact lifecycle blocks keep existing reason
IDs and stdout exit `3`; successful checks keep stdout exit `0`. Seal evidence
freshness still runs separately after a successful Seal gate.

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

⚠ 1 unresolved conflict detected

- ours = HEAD
- theirs = origin/main
NOTICE: Inspect a block by reading `conflict://<N>` (add `/ours` / `/theirs` / `/base` to render a single side). Resolve with `write({ path: "conflict://<N>", content })`, or bulk-resolve every registered conflict with `write({ path: "conflict://*", content })`. Writes replace ONLY the marker block (markers + all sides) — never repeat the lines before/after it; they stay in place.
`content` shorthand: a line that is exactly `@ours` / `@theirs` / `@base` / `@both` expands to that recorded section. `@both` is ours-then-theirs with no separator — only for additive conflicts where each side adds something different; NEVER for competing edits of the same lines (pick a side or write the combined text). Lines that are not a token pass through verbatim, so `"// keep both\n@ours\n@theirs"` literally writes the comment, then ours, then theirs.
Per-id bulk: `write({ path: "conflict://*", content: "1: @ours\n2: @theirs\n…" })` resolves each listed id with that side in ONE call — the cheapest way through many pick-one conflicts; unlisted ids stay registered.
Resolve each block faithfully: keep one side (`@ours`/`@theirs`), or combine them when both intents apply — never invent content beyond the recorded sides, and never stack both sides of competing edits. Resolve several conflicts in a single turn by issuing multiple `write` calls at once; ids stay valid as earlier blocks are resolved.

──── #2  L17-25 ────
<<< ours
| `check seal <slug>` | Check files required by target Phase `seal`, open human gates, the `tasks.md` slice graph, the stable Build-input binding, and exact candidate bindings. |
| `check path-disjoint [--root <dir>] [<json-file>|-]` | Verify slice path sets are pairwise disjoint. |
| `check task-graph <slug>` | Validate `tasks.md` slice dependency graph for cycles, unknown deps, malformed tokens, duplicate IDs, missing `Dependencies`, and `depends_on` mismatch. |
>>> theirs
| `check seal <slug>` | Check files required by target Phase `seal`, open human gates, the stable Build-input binding, and exact candidate bindings. |
| `check path-disjoint` with optional `--root DIR` and optional `JSON-FILE` or `-` | Verify slice path sets are pairwise disjoint. |
| `check task-graph <slug>` | Validate `tasks.md` slice dependency graph for cycles and unknown dependencies. |

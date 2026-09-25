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
| `check readiness <slug>` | Check target-Phase files, open human gates from Clarify onward, the `tasks.md` slice graph when that artifact is required, canonical `AC-###` presence in `tasks.md`/`test-plan.md` when those artifacts are required, and the current stable Build-input binding when applicable. |
| `check readiness --emit-binding <slug>` | Render the exact stable Build-input binding for Vet to record after review. |
| `check seal <slug>` | Check files required by target Phase `seal`, open human gates, the `tasks.md` slice graph, canonical `AC-###` presence when `tasks.md`/`test-plan.md` are required, the stable Build-input binding, and exact candidate bindings. |
| `check path-disjoint [--root <dir>] [<json-file> | -]` | Verify slice path sets are pairwise disjoint. |
| `parallel select --cap <1-10> [--root <dir>] [<json-file> | -]` | Choose the greedy path-disjoint subset ≤ cap from supplied ready slices. |
| `parallel <create\|integrate\|cleanup\|status\|abort\|lease-read\|lease-write\|lease-clear\|record-green>` | Deterministic parallel worktree leases, integration, and cleanup for parallel writer batches. |
| `check task-graph <slug>` | Validate `tasks.md` slice dependency graph for cycles, unknown deps, malformed tokens, duplicate IDs, missing `Dependencies`, and `depends_on` mismatch. |
| `check regression <slug> [--update]` | Compare the workspace's structural progress facts (phase ordinal, checked ACs, met/abandoned gates, done slices, resolved questions, present artifacts) against the recorded `regression-baseline.json` high-water mark; `--update` ratchets the baseline forward under the feature lock and prints each blessed regression. |
| `check drift <slug> [--record]` | Attribute readiness-input changes to the exact artifact. `--record` writes the per-input `readiness-inputs.json` digest baseline under the feature lock; without it the workspace diffs current inputs against that baseline (or falls back to the aggregate binding recorded in `eng-review.md`). Advisory: exit `0` on drift; a corrupt baseline exits `3`. |
| `check windows <slug> [--worktree\|--staged\|--base <ref>]` | Fail on deferral markers (TODO/FIXME/XXX/HACK, skipped-test and stub idioms) the change's added lines introduce without a waiver row in the workspace's `windows.md`. |
| `check dup [slug] [--all\|--worktree\|--staged\|--base <ref>] [--min-lines n] [--threshold f] [--ignore-file <path>] [--limit n] [--exclude <csv>]` | Report near-duplicate code clusters that survive comment stripping, literal collapsing, and identifier renaming — token-shingle matches chained into segments, clustered transitively, ranked with bounded path/line-distance boosts. Diff modes mark units overlapping changed hunks with `*`; stable content-derived cluster hashes let `.devrites/dup-ignore` dismissals survive line moves but resurface on structural edits. Staged mode reads index blobs and excludes untracked files. Bounded heuristic: successful scans/leads exit `0`; usage, unavailable input, and partial scans exit `2`. |
| `check slice <slug> <SLICE-ID>` | Pre-dispatch lint of one slice's wright contract: contract present, exact Writer allowlist paths, resolvable `Satisfies` AC ids, bounded scope. |
| `check diff-scope <slug> --allow <csv>\|--allow-file <path> [--worktree\|--staged\|--base <ref>]` | Verify the changed-path set stays inside the declared allowlist; the mechanical gate run before reviewer dispatch. |
| `check skill-trust <path>` | Scan one skill/agent Markdown file for structural trust violations. |
| `check indexes [--root <dir>]` | Report install manifest and code-index presence as JSON. |
| `observe summary <slug>` | Emit sanitized JSON workspace summary from one retained observation. `task_graph.ok` is true iff `task_graph.problems` is empty; `problems` lists cycle, unknown-dep, malformed-token, duplicate-id, and missing-`Dependencies` blockers. |
| `orient <slug>` | Alias for `observe summary`. |
| `observe slice <slug> <SLICE-ID>` | Print one `SLICE-###` section of `tasks.md` so Build reads a slice, not the file. |
| `next [slug]` | Print the minimal remaining lifecycle path plus advisory skips. |
| `handoff [slug]` | Emit the deterministic resume record: cursor fields, `awaiting_human`, open `questions.md` gate kinds, the `gates.md` reduction with unmet/stale/abandoned ids, `decisions.md` dead ends, missing required files, and the canonical read-next order. Read-only; the `/rite-handoff` spine and the compaction-fallback entry point. |
| `context [slug] (--phase <p>\|--skill <name>) [--role <r>] [--trigger a,b]` | Emit one deduplicated read-set bundle per phase/role/skill from each skill's `loads:` manifest; `--skill` works without a workspace when `--out` is given. |
| `dispatch <slug> <open\|start\|seal\|return\|status\|abandon>` | Launch-wave barrier for parallel dispatch: seal needs a handle per role, return needs seal; start/return auto-record metrics. |
| `metrics record <slug> --phase <p> --event <e> [--role r] [--bytes n] [--note s]` | Append an event to the workspace's `metrics.jsonl` ledger. |
| `metrics summary [slug]` | Reduce `metrics.jsonl` into a per-phase roll-up. |
| `claim <add\|release\|list\|check>` | Advisory session-scoped file claims in `.devrites/claims.jsonl` (append-only). `add` records intent to write a path set with a TTL (1–240 min, default 30); a live claim by another session exits `3` and names the holder. `release` appends a release event owned by the claiming session. `list` shows live claims (`--all` includes expired/released). `check` is the read-only preflight for a path set. Coordination aid only — never a substitute for one-writer-per-worktree. |
| `note <add\|list\|check\|rm> <slug> ...` | Anchored workspace notes in `notes.md`: `add <slug> <subject> <quote> <title> [body]` binds a note to a verbatim quote in one repository file and assigns `NOTE-###`; `list` and `check` regrade each anchor `exact`/`moved`/`stale`/`ambiguous`/`lost` by searching the repo for the quote (dependency and workspace trees excluded); `check --repair` rewrites a `moved` subject to its new file; `rm` deletes by ID. `check seal` refuses a malformed `notes.md` and any non-`exact` anchor. |
| `gates <scaffold\|status\|run\|reverify\|lint\|attest\|abandon> <slug>` | Operate the `gates.md` acceptance ledger: seed per-AC gates, execute runnable gates bound to `test-plan.md` preflight rows, attest manual gates, and reduce to `all-met`/`not-met`/`handoff`/`malformed`. |
| `state resolve <qid> "<answer>"` | Resolve an open question and update `questions.md` plus `state.md` atomically. |
| `state merge-manifest <slug> [pred...]` | Fold the recorded predecessor chain's manifests into the release candidate manifest. |
| `state close <slug>` | Archive a shipped workspace and clear matching `ACTIVE`. |
| `migrate <slug> [--dry-run] [--answer id=choice]` | Normalize a pre-v5 workspace to the current schema; fail-closed, one-shot. |
| `secret-scan [--staged] [--stdin] [slug]` | Scan exact staged blobs, stdin, or touched regular files for credential material. |
| `open-visual <path-or-name> [--slug <slug>] [--no-open]` | Resolve a local visual HTML file, optionally open it in the OS browser, warn if the sibling outline is missing or inventory ids are absent from HTML, and print agent path tips. No network. |
| `detect commands [--root <dir>] [--json]` | Resolve the repository's own test/lint/vet/build commands: explicit Makefile targets first, then `package.json` scripts, then language-manifest fallbacks, plus the lockfile-derived package manager. Read-only, no installs, no execution. |
| `overhaul <records\|admit\|snapshot\|score\|bench\|render> ...` | Deterministic tools behind the standalone `/overhaul` skill: run-record generations and cross-record validation, worker-receipt admission and quote anchoring, index-safe baseline snapshots and per-attempt deltas, exact-rational readiness scoring, benchmark ratio intervals, and escaped offline review/report views. Operates only on explicit run-area and repository paths. See [Overhaul tools](#overhaul-tools). |
| `version` | Print the engine version. |

`help`, `-h`, and `--help` print this operational inventory. Each operational
command and subcommand also accepts `-h` / `--help` and prints that command's
usage to stdout without requiring a workspace. `version` and `--version` print
the binary version. Other unlisted command forms are rejected as unknown; the
engine has no compatibility aliases or tombstones. The `add`/`upgrade`/`remove`
conveniences belong only to the `npx devrites` adapter, not to the engine
command namespace.

## Check boundary

The candidate gate validates and hashes path/state/type/mode/content identity;
it does not infer scope from Git. The readiness gate checks target-Phase
structure and applies open-question blocking only when that target is Clarify
or later, plus the exact `tasks.md` slice graph when `tasks.md` is required,
plus canonical `AC-###` ID presence in `tasks.md` and `test-plan.md` when those
artifacts are required, plus the exact stable Build-input binding after Vet. The
seal gate always targets Phase `seal`, repeats that graph, ID map, and binding,
and checks exact candidate bindings in evidence, optional browser evidence,
review, and seal. None judges the meaning of `CLEAR`/`READY` prose,
parses reviewer narratives, infers semantic acceptance coverage, counts
assertions, interprets capability deltas, or decides whether a technical plan is
sound.

Those judgments are made by the current skill and exact native roles, including
`devrites-plan-reviewer`, `devrites-proof-runner`, `devrites-spec-reviewer`,
`devrites-test-analyst`, and `devrites-doubt-reviewer`. The root reconciles their
reports against live artifacts and observed repository proof.

## State

State mutations use the shared physical-root checks, feature lock, and atomic
write path. `state resolve` additionally supports `--drop` and `--batch`;
`state close` owns transactional archive plus `ACTIVE` clearing.

`migrate` normalizes a pre-v5 workspace: legacy bullet cursor fields become
canonical table rows, the `schema` row is recorded, and missing required
artifacts are created as empty stubs (content is never synthesized, and bound
proof files stay byte-exact). It is one-shot and fail-closed: on ambiguity it
writes nothing, prints its questions, and exits `3`; answers arrive on rerun
via `--answer id=choice`. `--dry-run` prints the plan and always writes
nothing. See [ADR-0029](../adr/0029-v5-workspace-schema-and-native-migration.md).

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

## Overhaul tools

`overhaul` serves the explicit-only `/overhaul` skill. Each tool takes explicit
paths and never resolves a `.devrites` workspace, calls a model or uses the
network. The skill documents the record schemas and when to call each tool.

| Tool | Forms | Exit codes |
| --- | --- | --- |
| `records` | `init <repo> <run-id>`, `digest <file>`, `stage <run>`, `publish <run>`, `validate <run>` | 0 ok; 1 violations (listed as `VIOLATION:` lines); 2 usage or I/O |
| `admit` | `receipt <run> <receipt.json> [--observed <file>]`, `anchor <tree> <proposals.json>` | 0 admissible; 1 rejected; 2 usage or I/O; 3 duplicate or late receipt |
| `snapshot` | `capture <repo> <out>`, `fingerprint <repo>`, `verify <repo> <out> [--agent-paths <file>]`, `state <repo> <out.json>`, `delta <before.json> <repo>` | 0 ok; 1 index changed or changes outside agent paths; 2 refusal, usage or I/O |
| `score` | `--rubric <r> --results <s> --gates <g> [--out <file>]`, `compare --rubric <r> --baseline <a> --candidate <b>` | 0 computed; 2 invalid input (nothing counts as scored) |
| `bench` | `<result.json>` | 0 computed; 2 invalid input |
| `render` | `<run> <staged-generation> <review\|report>` | 0 written; 2 usage, invalid records or I/O |

`snapshot` runs git read-only with `GIT_OPTIONAL_LOCKS=0`,
`diff.autoRefreshIndex=false` and no external diff drivers, so `.git/index`
stays byte-identical. Secret-pattern files and files holding private key material
are hashed only: never copied into the snapshot tree and never included in its patches.
Clean tracked files are hashed but not copied, because git already holds them;
only dirty and untracked files are copied. A snapshot may live outside the
repository or inside its `.devrites/overhaul/` run area, which every snapshot
command leaves out of the repository state. `records init` creates
`<repo>/.devrites/overhaul/<run-id>/` (mode 0700) and, when git does not already
ignore it, adds `/.devrites/overhaul/` to the local `.git/info/exclude`, never to
the shared `.gitignore`.
`admit receipt` checks the attempt in the open staged generation (`g<N>.tmp` whose
`.base` is `CURRENT`) when one exists, otherwise in `CURRENT`, so a phase can record
many dispatches and admissions before one publish.

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

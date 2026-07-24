# `devrites-engine` commands (engine core)

The `devrites-engine` binary is the deterministic control plane over a project's
`.devrites/` state. Workspace state, gate, and derivation commands make no model
calls and are network-free; explicit install/update/source-cache I/O is isolated under
`engine/internal/iohooks` (ADR-0008). The in-session LLM remains the judgment
data plane.

This page expands selected contracts. Run `devrites-engine help` for the
exhaustive current command and hook inventory; see
[state-schema.md](state-schema.md) for `status` and the underlying state model.

## Exit codes

| code | meaning                                                       |
| ---- | ------------------------------------------------------------- |
| `0`  | ok / gate passed                                              |
| `1`  | command ran but could not complete its requested operation    |
| `2`  | usage error (bad args, unknown command, unknown `--harness`)  |
| `3`  | blocked: a gate pause or a safety refuse (`doctor`)           |

Exit `3` means the engine paused or refused an unsafe action; it does not mean
the process crashed. It prints a structured message that names exactly what to
resolve before retrying. Both a completeness gate (`readiness` or `seal`) and a
`doctor` refusal for newer state or unsafe root selection use this code. AFK
treats it as a pause.

## Gates: `readiness` / `seal`

These deterministic completeness gates are **phase-relative** and
**gate-scoped**. Each command checks only the sections required for that gate
when it runs.

- `devrites-engine readiness <slug>` asks whether the sections required to **leave the
  feature's current phase** are complete. A section that is not yet required (e.g.
  `proof` during the `spec` phase) never blocks.
- `devrites-engine seal <slug>` asks whether the feature is complete against the **full seal-phase
  requirement set**, regardless of its current phase.

```
$ devrites-engine readiness auth-tokens
gate: readiness
feature: auth-tokens
phase: build
result: blocked (missing to leave "build": tasks)
next: add real content to tasks.md, then re-run: devrites-engine readiness auth-tokens
$ echo $?
3
```

## `doctor`: root facts and version triangle

`devrites-engine doctor` is read-only. It reports the canonical action root,
why that root was selected, Git's physical topology, and the three versions that
can drift:

```
project: /work/example
root: /work/example/.devrites
root-selection: git-ancestor
git: top=/work/example dir=/work/example/.git common=/work/example/.git linked-worktree=false submodule=false
binary: X.Y.Z
pack: X.Y.Z
state-schema: v2 (binary supports v2)
verdict: ok: binary, pack, and state schema are compatible
hazards: ok
```

- **Parent `.devrites` beyond the current Git root, a physical path escape, or
  an external `DEVRITES_WORKSPACE`** → a named hazard and `REFUSE` (exit `3`).
  The report includes one pasteable `fix:` command.
- **Stale `ACTIVE` or canonical/generated residue** → a named warning (exit
  `0`) with its repair command.
- **Binary older than the pack** → a `WARN` (exit `0`): an older binary still
  runs; update it when convenient.
- **State schema a newer major than the binary supports** → a `REFUSE` (exit
  `3`): the binary won't silently mis-parse newer state.
- Additive schema changes (older state read by a newer binary) are always fine.

The pack version is discovered from `.claude/devrites.version` or the project
`package.json`; when neither exists the pack is reported `unknown` and no skew is
asserted. Doctor also reports linked-worktree/submodule identity, merge/rebase
state, host-artifact drift, and project extensions that have artifacts but no
optional `provenance.json`.

### Root safety at command dispatch

For commands on the shared workspace router, the engine resolves the action
root once before the command body runs. Read-only and diagnostic forms can still
degrade cleanly outside a workspace. A form that writes workspace or Git state
preserves every unsafe-root refusal and exits `3` before its command body runs.

Strict forms include `footprint log`, `stuck log`, `recovery record|clear`,
`clarify-return`, `reconcile`, mutating `resolve` forms, `close-out`,
`decisions index`, `ledger sync`, `learnings add`, `timeline log|purge`,
`health run|check|record`, `review-fingerprints --write`,
`reviewer-stats record`, every `forge` verb, `extensions sync`, `context sync`,
and `runbook run|resume`. `resolve next-qid` stays read-only.

## `snapshot`: workspace status JSON

`devrites-engine snapshot [slug]` emits the `devrites.workspace.v1` JSON contract.

## `profile`: stable repo facts cache

`devrites-engine profile get|refresh` caches question-agnostic repo facts for grounding skills: top-level layout, manifests, and digests for root docs, ADRs, CI/deploy files, and `.devrites` principles/conventions. It never calls a model or the network.

- `profile get` prints `HIT` + JSON, `MISS` + cache path, or `NO-CACHE` outside a git repo.
- `profile refresh` derives the profile from disk and writes the cache.

The cache lives under `/tmp/compound-engineering/devrites/repo-profile` by default and is invalidated when profile-input files are dirty or newly added. Skills still re-scan candidate-specific code fresh.

## `migrate`: legacy aliases and old layouts

`devrites-engine migrate` keeps older `.devrites/features/<slug>/` workspaces
readable while using `.devrites/work/<slug>/` as the canonical live location.
Migration is:

- **idempotent**: a second run is a no-op (`already up to date`);
- **backed up**: the pre-migration `work/` and `ACTIVE` are snapshotted to a
  timestamped `.migrate-backup-*` directory before anything changes;
- **lossless**: canonical files are added without deleting aliases. `README.md`
  is the preferred workspace map while `feature.md` / `index.md` remain readable;
  `state.md` is preferred while `status.md` remains a cursor alias; `evidence.md`
  is preferred while `proof.md` remains a proof alias.

The phase is derived from the legacy `state.md`, defaulting to `build` when it
can't be read.

`devrites-engine update` refreshes the installed binary and pack. Migration
stops at structural compatibility and does not certify old planning artifacts
against current workflow rules. Build readiness routes an active workspace with
stale semantic artifacts to `/rite-upgrade [slug]`; that public rite preserves
completed work and reconciles only unfinished planning.

## Hooks: `hook <name> --harness=claude|codex`

One binary serves Claude Code and Codex through thin per-harness adapters.
Hooks are **fail-open and read-only unless they explicitly gate**.

- `devrites-engine hook orient --harness=H` emits the SessionStart orientation for the
  active feature (named by `.devrites/ACTIVE`) as the harness's
  `hookSpecificOutput.additionalContext` envelope. With no active feature (or a
  stale pointer), the first-ever such session instead gets a one-time starting
  nudge derived from the `first-task` token (greenfield → `/rite-spec`,
  brownfield → `/rite-adopt`, …); the `.devrites/.first-run-shown` marker keeps
  it from repeating. Silent (exit `0`, no output) outside a workspace or once
  the marker exists.
- `devrites-engine hook auq` observes a completed `AskUserQuestion` call. It
  appends one metadata-only `human-wait-resumed` v1 row per question to the root
  and feature event logs. Canonical workflow answers stay in `questions.md`;
  telemetry stores neither the prompt nor the answer. It never tunes, blocks,
  or replies and stays silent outside an active workspace. This is Claude-only
  because the Codex host exposes no equivalent post-user-input hook.
- `devrites-engine hook git-guard --harness=H` silently passes ordinary Git,
  denies ambiguous high-impact shell forms with a direct-literal remediation,
  and gates an unambiguous destructive operation on one exact digest. With no
  grant it opens one idempotent escalating question. The exact answer
  `Authorize once` is valid for 15 minutes and is atomically consumed before
  the tool runs, so a failed tool call still spends it. Questions, the private
  consumption ledger, events, and diagnostics never retain the raw command,
  normalized tokens, paths, or refs.
- `devrites-engine hook stop-gate --harness=H` refuses to end a turn at a provably
  inconsistent **rest point**, such as a feature in phase `seal` or `ship` with
  empty `evidence.md` or `proof.md`. It does not check whole-feature completeness,
  so normal in-progress work is never blocked. By default, it records a
  would-be block in the feature's `.stop-gate.log` (mirroring
  `devrites-engine hook stop-gate`) and allows the stop. Set
  `DEVRITES_STOP_GATE=enforce` to block. The harness's `stop_hook_active` loop
  guard prevents the hook from wedging a session.

When a guard makes a real decision, it appends a metadata-only
`devrites-event/v1` row. The row says which stable rule fired, whether the guard
was enforced, observed, unavailable, or bypassed, and which host delivered it.
It never copies the command, tool payload, fetched content, denial prose, or an
absolute path. Event-write failure stays fail-open and cannot change the hook's
decision.

### Fail-open guard

An inline POSIX guard makes a missing binary a no-op, so a teammate without
`devrites-engine` installed is not blocked:

```sh
command -v devrites-engine >/dev/null 2>&1 && devrites-engine hook orient --harness=claude || exit 0
```

## `ledger`: the living capability store

`devrites-engine ledger <sub>` maintains `.devrites/specs/<capability>/spec.md`, the cumulative
record of proven behavior (see [state-schema.md § Capability ledger](state-schema.md#capability-ledger-specs)).
Feature specs carry deltas under an `ADDED`, `MODIFIED`, or `REMOVED`
Requirements heading tagged with `capability: <c>`. The fold uses header identity
for upserts and deletes, so re-syncing a feature is a no-op.

- `ledger sync <workspace-dir>`: fold a feature's deltas into every capability they touch (ADDED
  append, MODIFIED replace, REMOVED delete). Called from `/rite-ship` on GO. Exit `0`.
- `ledger diff <workspace-dir>`: dry-run the fold (the change preview shown before sync). Exit `0`.
- `ledger validate`: grammar-lint every ledger spec. Exit `0` clean · `1` on a violation.
- `ledger list` / `ledger show <capability>`: read the ledger (used by `/rite-spec` and
  `/rite-adopt` to write deltas against the current contract). Exit `0` · `1` unknown capability.

`spec-validate <dir> --against .devrites/specs` cross-checks a spec's delta classification against
the ledger (ADDED must be new; MODIFIED/REMOVED must already exist) and validates Edge Coverage /
Prohibitions tables as a blocking spec-gate check.

## `analyze`: cross-artifact coverage & consistency

Before code is written, `devrites-engine analyze [slug]` compares a feature's
`spec.md` with its `tasks.md`. This catches coverage gaps while they still need
only a one-line plan edit instead of a mid-build reslice. The Markdown report
has four passes:

- **Coverage**: a spec `AC-###` that no slice `Satisfies:` (**CRITICAL**; legacy `[ACn]` remains supported).
- **Consistency**: a slice that `Satisfies:` an AC the spec never defines (**CRITICAL**).
- **Orphan slice**: a slice satisfying no acceptance criterion (warn).
- **Ambiguity**: an unquantified vague adjective (`fast`, `robust`, `intuitive`, …) or an
  unresolved placeholder (`TODO`, `TKTK`, `???`) in the spec (warn).

It closes with a **Metrics** line (criteria count, coverage %, orphan + ambiguity counts) so the
vet gate reports a number instead of only pass or fail. Exit `0` clear · `1` at least one CRITICAL ·
`2` no workspace (no active slug, or `spec.md`/`tasks.md` missing). `/rite-vet` runs it in its
cross-artifact gate (step 2a) and adds semantic checks for terminology drift and
duplicated or conflicting requirements on top of this deterministic floor.

## `review-integrity`: the silent-reviewer gate

`devrites-engine review-integrity [slug]` catches reviews that return only
"looks good, nothing found." It parses the `## Spec` and `## Code review`
sections of `review.md` and flags any section with neither a bold severity label
nor a `No-findings:` justification. A zero-count summary does not count as a
finding, and an all-zero tally is treated as a rubber stamp. Exit `0` means
every axis is accounted for, or that `review.md` is absent or freeform. Exit
`1` means an axis is silent and unjustified.
`/rite-review` runs this check after writing `review.md`; `/rite-seal` treats
`rc=1` as an Important finding about review completeness. Like
`doubt-coverage` and the footprint roster, this command checks that the account
exists, not whether its judgment is correct.

## `timeline`: local, privacy-bounded workflow trace

`timeline log` accepts only validated `devrites-event/v1` facts. Legacy
free-text `--skill`, `--decision`, and `--note` writes are refused; `list`
continues to print old rows unchanged for compatibility. Canonical traces use a
stable `DEVRITES_RUN_ID` across related calls and the small event vocabulary
`run-started`, phase/gate events, `run-interrupted`, `run-resumed`, and
`run-finished`:

```bash
devrites-engine timeline log run-started \
  --slug auth-tokens \
  --execution-mode named \
  --guard-strength n/a \
  --reason-id DRV-ROOT-SELECTED \
  --host codex

devrites-engine timeline log run-finished \
  --slug auth-tokens \
  --outcome passed \
  --execution-mode named \
  --guard-strength n/a \
  --reason-id DRV-GATE-SEAL-PASSED \
  --host codex \
  --evidence .devrites/work/auth-tokens/seal.md
```

`--execution-mode`, `--guard-strength`, and `--reason-id` are required. Rows may
also carry phase IDs, rule IDs, project-relative evidence paths, and a host ID.
They never retain prompts, question/answer text, source or diff bodies, model
prose, absolute paths, user/external identifiers, auth/config/secrets, or token
estimates. The current v1 contract exposes host but not model, token, or cost
fields, so the engine does not infer them.

`timeline report [--run <opaque-id>] [--json]` reads at most the last 4 MiB and
4,096 valid v1 rows. It reports observed run and phase duration, retry and
human-wait counts, interruption/resume linkage, active execution/guard mode,
the last failed gate reason, stale-evidence/degradation counts, and final
outcome. Missing, corrupt, oversized, legacy, or truncated input degrades the
report only. Legacy and corrupt rows are counted as ignored and never
interpreted as v1. The statusline and `progress` reuse these display facts but
never treat them as lifecycle authority.

Each telemetry log stops accepting new rows at 16 MiB. Hitting that bound
degrades telemetry only; it cannot weaken or strengthen a workflow decision.

`timeline purge (--before <RFC3339> | --run <opaque-id>)...` removes only valid
matching v1 rows from `.devrites/timeline.jsonl` and the live feature
`events.jsonl` files. When both selectors are present they form an intersection.
It never touches state, questions, decisions, evidence, recovery, capability,
or allowlist files. Purge is bounded to 16 MiB per file and refuses symlinks,
oversized rows, concurrent changes, and unsafe roots without mutation.

Telemetry is local instrumentation only; DevRites sends no analytics or remote
traces. Review the retention needs, then use exact purge selectors. Reports do
not interpret legacy rows, so manually delete legacy log files if their old
free-text content must be removed. Deleting only
`.devrites/timeline.jsonl` and live feature `events.jsonl` files is safe for
workflow correctness; the engine recreates them as needed.

## `health`: code-health dashboard and history

`devrites-engine health`, `health run`, and `health check` run the known project
checks (available npm test/lint/typecheck/build scripts, `go test`, `pytest`, and
DevRites scans where present), print a PASS/WARN/FAIL dashboard, and append the
result to `.devrites/health.jsonl`. These commands execute project checks; they
are not a substitute for reviewing those scripts before use.

The legacy `health record|list` surface remains available for manual scores.
`record` appends `.devrites/health-history.jsonl`; `list` tails
`.devrites/health.jsonl` when dashboard history exists, otherwise the legacy file.
Manual scores are intentionally caller-owned: name the observed evidence rather
than asking the engine to infer a universal metric.

```bash
devrites-engine health run
devrites-engine health record 8.5 "tests green; one follow-up" --note "review-fingerprints stable"
devrites-engine health list --limit 10
```

Scores must be `0..10`. The label should name the evidence rather than give a
subjective impression. Skill health stays static until DevRites records
per-skill run outcomes; use `scripts/skill-pruning-audit.mjs` for pruning
signals instead of inventing telemetry.

## `review-fingerprints`: stable IDs for findings

`devrites-engine review-fingerprints [--write] [slug]` scans `.devrites/work/<slug>/review.md` for
bold severity labels (`Critical`, `Important`, `Suggestion`, `Nit`, `FYI`) and emits stable
12-character IDs derived from severity + normalized finding text. With `--write`, it saves
`.devrites/work/<slug>/review-fingerprints.jsonl`.

```bash
devrites-engine review-fingerprints --write auth-tokens
```

The IDs let callers correlate recurring findings, dismissals, and later
learning without copying full review text into every downstream surface.
`review-integrity` remains the gate; this command records only stable
references.

## `reviewer-stats`: dispatch outcomes that gate the fan-out

`devrites-engine reviewer-stats record <agent> <surviving-findings> [slug]` appends one dispatch
outcome to `.devrites/reviewer-stats.jsonl` (cross-feature, append-only).
`devrites-engine reviewer-stats report [--json]` grades each reviewer deterministically:

- `run (always-on)`: the unconditional axes (`spec-reviewer`, `code-reviewer`, `test-analyst`).
- `run (insurance; never gated)`: `security-auditor` and `doubt-reviewer`. A dry streak is
  success, never a reason to skip.
- `gate-candidate`: a conditional reviewer with zero surviving findings in its last 10+
  dispatches; the fan-out may skip it as a *recorded* skip (see the shared
  `pack/.claude/skills/devrites-lib/reference/parallel-dispatch.md` contract,
  § Hit-rate gating).
- `run`: everything else.

```bash
devrites-engine reviewer-stats record devrites-performance-reviewer 0 auth-tokens
devrites-engine reviewer-stats report
```

Thresholds live in the engine. The caller reads the verdict without
recalculating or overriding the streak. A user-requested full panel still
dispatches every reviewer.

## `reviewers list`: bounded reviewer aliases

`devrites-engine reviewers list` validates same-adapter reviewer aliases from `.devrites/config.json`
or flat `.devrites/config*` keys. It never executes a reviewer; it only checks the bounded config
surface (`cli` must be `claude` or `codex`; `model` and `agent` are opaque strings).

```json
{
  "review": {
    "reviewer_instances": {
      "codex-deep": { "cli": "codex", "model": "o3" }
    }
  }
}
```

## `forge`: isolated candidate worktrees

Forge compares two or three implementation strategies without letting a worker
choose its own worktree, branch, or merge target. The engine owns those paths in
one `devrites-forge/v1` manifest:

```bash
devrites-engine forge plan SLICE-004 feature-slug \
  --strategy A='small adapter' \
  --strategy B='native integration' \
  --acceptance-hash <full-sha256> \
  --test-plan-hash <full-sha256> \
  --worker-binding manifest-env-v1
```

`plan` requires a clean primary checkout, 2 or 3 contiguous candidates, and
complete hashes for the acceptance and test-plan scorecards. It writes the
manifest before the first Git side effect, then creates candidate worktrees
under a sibling directory named `.<repo>.devrites-forge/<run-id>/`. The
manifest stays under
`.devrites/work/<slug>/.forge/<run-id>/manifest.json`.

Parallel Forge needs an exact host binding. Omitting
`--worker-binding manifest-env-v1` is a supported fallback, not a partial
parallel run. The command exits `0` with:

```json
{"status":"degraded","mode":"serial","reason":"supported worker binding was not declared"}
```

It creates no manifest, branch, or worktree in that case. An unsafe repository
topology, dirty primary checkout, in-progress Git operation, unavailable
process-liveness proof, or path collision also returns a bounded serial
degradation when it can do so safely. Invalid arguments or a broken manifest
fail instead of silently choosing a target.

Use `forge process-token <pid>` to obtain the process-start token for a real
worker. A bound candidate wright receives all five variables below and runs
from the candidate worktree:

```text
DEVRITES_FORGE_RUN_ID
DEVRITES_FORGE_CANDIDATE
DEVRITES_FORGE_WORKER_ID
DEVRITES_FORGE_WORKER_PID
DEVRITES_FORGE_PROCESS_START
```

The binding is all-or-none. `wright-scope` checks the manifest, physical
working directory, repository common directory, candidate branch, worker ID,
live PID/start token, and leaf-agent identity before it permits a write. A
partial, stale, sibling, foreign, or tampered binding is denied.

The remaining commands advance only manifest-owned state:

```bash
devrites-engine forge record <run-id> A running \
  --worker-id <id> --pid <pid> --process-start <token>
devrites-engine forge record <run-id> A finished --worker-id <id>
devrites-engine forge extract <run-id> A
devrites-engine forge record <run-id> winner A --worker-id <judge-id>
devrites-engine forge merge <run-id> A
devrites-engine forge record <run-id> verification verified \
  --worker-id <verifier-id>
devrites-engine forge cleanup <run-id>
devrites-engine forge reap [feature-slug]
```

Extract every candidate before recording and merging the judge's winner.
Extraction snapshots the full candidate tree, pins its commit, tree, and binary
delta hash, and refuses unrepresentable or still-live state. Merge requires the
recorded winner, a clean unchanged primary baseline, every candidate extracted,
and an exact fast-forward result. Cleanup runs only after the winner landed and
independent verification was recorded as `verified`. It preserves anything
dirty, live, foreign, ambiguous, or otherwise unsafe. `reap` follows the same
manifest-only rule for interrupted runs and never deletes a branch by name
alone.

## `extensions` / `overrides`: project extensibility

Two project-local surfaces let a team extend the pack without forking it. The
full contract is in [extensions.md](../extensions.md).

- `extensions list|validate|sync`: user rites/reviewers under `.devrites/extensions/<name>/`, held
  to the pack's schema (`validate`, exit `1` on a malformed extension) and mirrored into `.claude/`
  (`sync`, validates first, refuses a broken set).
- `overrides list|validate`: reviewer overrides under `.devrites/overrides/<agent>.md`, advisory
  house rules a shipped reviewer reads after its standards. `validate` exits `1` when an override
  reads like it waives a gate. An override may add checks but never relax one.

## `context sync|show`: agent context

`devrites-engine context sync [file ...]` upserts only the block delimited by
`<!-- DEVRITES START -->` / `<!-- DEVRITES END -->` in project context files. With no file args it
reads `.devrites/context.yaml` (`context_file:` or `context_files:`), then falls back to existing
`AGENTS.md` / `CLAUDE.md`, then `AGENTS.md`. Paths must be project-relative.

`devrites-engine context show [--json]` is read-only. It uses the same physical
root facts as `doctor`: canonical and lexical roots, selection reason, Git
top-level/dir/common-dir/superproject facts, active workspace source, and stable
hazards with pasteable remediations. `--json` emits one direct document for
wrappers that need to know where a command will act. `context sync` refuses an
unsafe root instead of writing through a fallback.

## `runbook`: tiny local automation

`devrites-engine runbook list|validate|run|resume` executes flat YAML runbooks from
`.devrites/runbooks/*.yaml`. Supported step forms are deliberately small:

```yaml
steps:
  - engine: doctor
  - rite: status
  - shell: npm test
  - gate: review before release
```

`engine` runs a local `devrites-engine` subcommand, `rite` prints the Claude/Codex dispatch form,
`shell` runs in the project root, and `gate` writes `.devrites/runs/<id>/state.json` then exits `3`.
Resume with `devrites-engine runbook resume <id>`. This command handles
repeatable local runbooks; it does not replace the lifecycle.

## Concurrency

DevRites fans out reviewer subagents that each spawn the binary, so the real
contention is between short-lived **processes**. State writes are hardened for it:

- **append-only logs** use `O_APPEND` with small records, so parallel writers never
  interleave or lose records;
- **structured files** use a temporary file and atomic rename, so a reader (or a
  writer killed mid-write) never sees a half-written file;
- **read-modify-write** takes a per-feature advisory `flock` on Unix to avoid
  lost updates.

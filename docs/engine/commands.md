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
| `2`  | usage error (bad args, unknown command, unknown `--harness`)  |
| `3`  | blocked: a gate pause or a version-skew refuse (`doctor`)    |

Exit `3` is always a **pause, not a crash**: a structured, actionable message
naming exactly what to resolve, then retry. This keeps enforcement safe under
AFK. A run pauses rather than hard-failing. Both a completeness gate
(`readiness`/`seal`) and a `doctor` refuse (state schema newer than the binary
supports) use it.

## Gates: `readiness` / `seal`

Deterministic completeness gates. Enforcement is **phase-relative** and
**gate-scoped**: a gate checks only the sections it needs, only when run.

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

## `doctor`: version triangle

`devrites-engine doctor` reports the three versions that can drift out of alignment and
one legible verdict:

```
binary: X.Y.Z
pack: X.Y.Z
state-schema: v1 (binary supports v1)
verdict: ok: binary, pack, and state schema are compatible
```

- **Binary older than the pack** → a `WARN` (exit `0`): an older binary still
  runs; update it when convenient.
- **State schema a newer major than the binary supports** → a `REFUSE` (exit
  `3`): the binary won't silently mis-parse newer state.
- Additive schema changes (older state read by a newer binary) are always fine.

The pack version is discovered from `.claude/devrites.version` or the project
`package.json`; when neither exists the pack is reported `unknown` and no skew is
asserted. Doctor also warns when project extensions have artifacts but no optional
`provenance.json`.

## `snapshot`: workspace status JSON

`devrites-engine snapshot [slug]` emits the `devrites.workspace.v1` JSON contract.

## `profile`: stable repo facts cache

`devrites-engine profile get|refresh` caches question-agnostic repo facts for grounding skills: top-level layout, manifests, and digests for root docs, ADRs, CI/deploy files, and `.devrites` principles/conventions. It never calls a model or the network.

- `profile get` prints `HIT` + JSON, `MISS` + cache path, or `NO-CACHE` outside a git repo.
- `profile refresh` derives the profile from disk and writes the cache.

The cache lives under `/tmp/compound-engineering/devrites/repo-profile` by default and is invalidated when profile-input files are dirty or newly added. Skills still re-scan candidate-specific code fresh.

## `migrate`: legacy aliases and old layouts

`devrites-engine migrate` preserves old workspaces while the canonical live location is
`.devrites/work/<slug>/`. Older `.devrites/features/<slug>/` workspaces remain
readable, and the migration path is:

- **idempotent**: a second run is a no-op (`already up to date`);
- **backed up**: the pre-migration `work/` and `ACTIVE` are snapshotted to a
  timestamped `.migrate-backup-*` directory before anything changes;
- **lossless**: canonical files are added without deleting aliases. `README.md`
  is the preferred workspace map while `feature.md` / `index.md` remain readable;
  `state.md` is preferred while `status.md` remains a cursor alias; `evidence.md`
  is preferred while `proof.md` remains a proof alias.

The phase is derived from the legacy `state.md`, defaulting to `build` when it
can't be read.

## Hooks: `hook <name> --harness=claude|codex`

One binary serves both Claude Code and Codex through thin per-harness adapters.
Every hook is **fail-open and read-only unless it explicitly gates**.

- `devrites-engine hook orient --harness=H` emits the SessionStart orientation for the
  active feature (named by `.devrites/ACTIVE`) as the harness's
  `hookSpecificOutput.additionalContext` envelope. With no active feature (or a
  stale pointer), the first-ever such session instead gets a one-time starting
  nudge derived from the `first-task` token (greenfield → `/rite-spec`,
  brownfield → `/rite-adopt`, …); the `.devrites/.first-run-shown` marker keeps
  it from repeating. Silent (exit `0`, no output) outside a workspace or once
  the marker exists.
- `devrites-engine hook auq` captures an `AskUserQuestion` exchange after tool use.
  It appends each question + chosen answer to `.devrites/timeline.jsonl`
  and the feature's `events.jsonl`, so HITL decisions are recorded at the
  substrate instead of trusting the model's bookkeeping. It only captures data and never
  tunes, blocks, or replies; silent outside an active workspace. Claude-only by
  design. Codex has an equivalent tool (`request_user_input`) but emits no hook
  event for it. Codex PostToolUse matches only Bash/`apply_patch`/MCP calls, and
  the user-input-requested event was declined upstream (openai/codex#12524).
- `devrites-engine hook stop-gate --harness=H` refuses to end a turn at a provably
  inconsistent **rest point**, such as a feature in phase `seal` or `ship` with
  empty `evidence.md` or `proof.md`. It does not check whole-feature completeness,
  so normal in-progress work is never blocked. It observes by default: a would-be block is
  appended to the feature's `.stop-gate.log` (mirroring `devrites-engine hook stop-gate`)
  rather than gating; set `DEVRITES_STOP_GATE=enforce` to actually block.
  Loop-guarded by the harness's `stop_hook_active` so it can never wedge a
  session.

### Fail-open guard

Hooks are wired behind an inline POSIX guard so a **missing binary is a no-op**
that never wedges a session (a teammate without `devrites-engine` installed is never
blocked):

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

`devrites-engine analyze [slug]` cross-checks a feature's `spec.md` against its `tasks.md` before
any code is written, so a coverage gap surfaces as a one-line plan edit instead of a reslice
mid-build. It emits a markdown report with four passes:

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

`devrites-engine review-integrity [slug]` guards the failure opposite to noise: a reviewer that
returns "looks good, nothing found". It parses `review.md`'s `## Spec` / `## Code review` axis
sections and flags any that carry neither a bold-labeled finding nor a `No-findings:` justification.
A zero-count summary line does **not** count as a finding. An all-zero tally is the rubber-stamp
this catches. Exit `0` every axis accounted for (or no/freeform `review.md`) · `1` an axis is silent
and unjustified. `/rite-review` runs it after writing `review.md`; `/rite-seal` treats `rc=1` as an
Important on the review's completeness. The honesty contract mirrors `doubt-coverage` and the
footprint roster: it checks the *account* is present, not its quality.

## `timeline`: append-only session trace

`devrites-engine timeline log|list` records compact session events in `.devrites/timeline.jsonl`.
It is for reconstructing what happened across long agent runs: which rite or skill acted, what
feature it touched, what decision it made, and whether a state transition happened. It does not
gate anything; it is durable context for audits, handoffs, and later learning.

```bash
devrites-engine timeline log completed --skill rite-review --slug auth-tokens --outcome ok --decision "ship"
devrites-engine timeline log state-change --slug auth-tokens --from build --to review --note "tests green"
devrites-engine timeline list --limit 20
```

Records are JSONL, append-only, and safe for concurrent short-lived engine calls. Install and
update DevRites through the npm flow (`npx devrites ...`); this command is part of the installed
engine, not a Claude/Codex plugin distribution path.

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

Scores must be `0..10`. The label should name the evidence, not a vibe. Skill health stays static
until DevRites records per-skill run outcomes; use `scripts/skill-pruning-audit.mjs` for pruning
signals instead of inventing telemetry.

## `review-fingerprints`: stable IDs for findings

`devrites-engine review-fingerprints [--write] [slug]` scans `.devrites/work/<slug>/review.md` for
bold severity labels (`Critical`, `Important`, `Suggestion`, `Nit`, `FYI`) and emits stable
12-character IDs derived from severity + normalized finding text. With `--write`, it saves
`.devrites/work/<slug>/review-fingerprints.jsonl`.

```bash
devrites-engine review-fingerprints --write auth-tokens
```

The IDs make recurring findings, dismissals, and later learning easier to correlate without
copying full review text into every downstream surface. `review-integrity` remains the gate; this
command only records stable references.

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

Thresholds live in the engine, not the prompt: the caller reads the verdict, it never re-derives
or overrides the streak math (a user-requested full panel dispatches everything regardless).

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

`devrites-engine context show [--json]` is read-only. It reports the project root, `.devrites` root,
active workspace, the source of that selection (`ACTIVE`, `DEVRITES_WORKSPACE`, `DEVRITES_ROOT`, or
`none`), and the Claude/Codex menu forms. `--json` emits one direct JSON document for wrappers that
need to know where a command will act.

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
Resume with `devrites-engine runbook resume <id>`. This is for repeatable local runbooks, not a
replacement lifecycle.

## Concurrency

DevRites fans out reviewer subagents that each spawn the binary, so the real
contention is between short-lived **processes**. State writes are hardened for it:

- **append-only logs** use `O_APPEND` with small records, so parallel writers never
  interleave or lose records;
- **structured files** use a temporary file and atomic rename, so a reader (or a
  writer killed mid-write) never sees a half-written file;
- **read-modify-write** takes a per-feature advisory `flock` on Unix to avoid
  lost updates.

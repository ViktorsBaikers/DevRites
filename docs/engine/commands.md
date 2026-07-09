# `devrites-engine` commands (engine core)

The `devrites-engine` binary is the deterministic control plane over a project's
`.devrites/` state. It makes **zero model or network calls** — the in-session
LLM stays the judgment data plane; the engine only sequences, gates, and reads.

This covers the commands added in issues 03–07, on top of `status` / `reindex`
(see [state-schema.md](state-schema.md)).

## Exit codes

| code | meaning                                                       |
| ---- | ------------------------------------------------------------- |
| `0`  | ok / gate passed                                              |
| `2`  | usage error (bad args, unknown command, unknown `--harness`)  |
| `3`  | blocked — a gate pause or a version-skew refuse (`doctor`)    |

Exit `3` is always a **pause, not a crash**: a structured, actionable message
naming exactly what to resolve, then retry. This keeps enforcement safe under
AFK — a run pauses rather than hard-failing. Both a completeness gate
(`readiness`/`seal`) and a `doctor` refuse (state schema newer than the binary
supports) use it.

## Gates: `readiness` / `seal`

Deterministic completeness gates. Enforcement is **phase-relative** and
**gate-scoped**: a gate checks only the sections it needs, only when run.

- `devrites-engine readiness <slug>` — are the sections required to **leave the
  feature's current phase** complete? A section that is not yet required (e.g.
  `proof` during the `spec` phase) never blocks.
- `devrites-engine seal <slug>` — is the feature complete against the **full seal-phase
  requirement set**, regardless of its current phase?

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

## `doctor` — version triangle

`devrites-engine doctor` reports the three versions that can drift out of alignment and
one legible verdict:

```
binary: 2.6.1
pack: 2.6.1
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

## `snapshot` — workspace status JSON

`devrites-engine snapshot [slug]` emits the `devrites.workspace.v1` JSON contract.

## `migrate` — legacy aliases and old layouts

`devrites-engine migrate` preserves old workspaces while the canonical live location is
`.devrites/work/<slug>/`. Older `.devrites/features/<slug>/` workspaces remain
readable, and the migration path is:

- **idempotent** — a second run is a no-op (`already up to date`);
- **backed up** — the pre-migration `work/` and `ACTIVE` are snapshotted to a
  timestamped `.migrate-backup-*` directory before anything changes;
- **lossless** — canonical files are added without deleting aliases. `README.md`
  is the preferred workspace map while `feature.md` / `index.md` remain readable;
  `evidence.md` is preferred while `proof.md` remains a proof alias.

The phase is derived from the legacy `state.md`, defaulting to `build` when it
can't be read.

## Hooks: `hook <name> --harness=claude|codex`

One binary serves both Claude Code and Codex through thin per-harness adapters.
Every hook is **fail-open and read-only unless it explicitly gates**.

- `devrites-engine hook orient --harness=H` — emits the SessionStart orientation for the
  active feature (named by `.devrites/ACTIVE`) as the harness's
  `hookSpecificOutput.additionalContext` envelope. Silent (exit `0`, no output)
  outside a workspace, with no active feature, or on a stale pointer.
- `devrites-engine hook stop-gate --harness=H` — refuses to end a turn at a provably
  inconsistent **rest point** (a feature claiming completion — phase `seal`/`ship`
  — with empty `evidence.md` / `proof.md`). NOT whole-feature completeness, so normal
  in-progress work is never blocked. **Observe by default** — a would-block is
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

## `ledger` — the living capability store

`devrites-engine ledger <sub>` maintains `.devrites/specs/<capability>/spec.md`, the cumulative
record of proven behavior (see [state-schema.md § Capability ledger](state-schema.md#capability-ledger--specs)).
Feature specs carry deltas (`## ADDED/MODIFIED/REMOVED Requirements — capability: <c>`); the fold
is a header-identity upsert/delete, so it is **idempotent** — re-syncing a feature is a no-op.

- `ledger sync <workspace-dir>` — fold a feature's deltas into every capability they touch (ADDED
  append, MODIFIED replace, REMOVED delete). Called from `/rite-ship` on GO. Exit `0`.
- `ledger diff <workspace-dir>` — dry-run the fold (the change preview shown before sync). Exit `0`.
- `ledger validate` — grammar-lint every ledger spec. Exit `0` clean · `1` on a violation.
- `ledger list` / `ledger show <capability>` — read the ledger (used by `/rite-spec` and
  `/rite-adopt` to write deltas against the current contract). Exit `0` · `1` unknown capability.

`spec-validate <dir> --against .devrites/specs` cross-checks a spec's delta classification against
the ledger (ADDED must be new; MODIFIED/REMOVED must already exist) and validates Edge Coverage /
Prohibitions tables — a blocking spec-gate check.

## `review-integrity` — the silent-reviewer gate

`devrites-engine review-integrity [slug]` guards the failure opposite to noise: a reviewer that
returns "looks good, nothing found". It parses `review.md`'s `## Spec` / `## Code review` axis
sections and flags any that carry neither a bold-labeled finding nor a `No-findings:` justification.
A zero-count summary line does **not** count as findings — an all-zero tally is the rubber-stamp
this catches. Exit `0` every axis accounted for (or no/freeform `review.md`) · `1` an axis is silent
and unjustified. `/rite-review` runs it after writing `review.md`; `/rite-seal` treats `rc=1` as an
Important on the review's completeness. The honesty contract mirrors `doubt-coverage` and the
footprint roster — it checks the *account* is present, not its quality.

## `timeline` — append-only session trace

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

## `health` — compact quality history

`devrites-engine health record|list` keeps `.devrites/health-history.jsonl`, a small longitudinal
quality signal for the workspace. The score is intentionally caller-owned: an agent, CI job, or
human can record the observed health after a review, quality gate, or cleanup pass without the
engine pretending to infer a universal metric.

```bash
devrites-engine health record 8.5 "tests green; one follow-up" --note "review-fingerprints stable"
devrites-engine health list --limit 10
```

Scores must be `0..10`. The label should name the evidence, not a vibe. Skill health stays static
until DevRites records per-skill run outcomes; use `scripts/skill-pruning-audit.mjs` for pruning
signals instead of inventing telemetry.

## `review-fingerprints` — stable IDs for findings

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

## `extensions` / `overrides` — project extensibility

Two project-local surfaces let a team extend the pack without forking it — full contract in
[extensions.md](../extensions.md).

- `extensions list|validate|sync` — user rites/reviewers under `.devrites/extensions/<name>/`, held
  to the pack's schema (`validate`, exit `1` on a malformed extension) and mirrored into `.claude/`
  (`sync`, validates first, refuses a broken set).
- `overrides list|validate` — reviewer overrides under `.devrites/overrides/<agent>.md`, advisory
  house rules a shipped reviewer reads after its standards. `validate` exits `1` when an override
  reads like it waives a gate — an override may add checks, never relax one.

## Concurrency

DevRites fans out reviewer subagents that each spawn the binary, so the real
contention is between short-lived **processes**. State writes are hardened for it:

- **append-only logs** use `O_APPEND` with small records — parallel writers never
  interleave or lose records;
- **structured files** are written via temp-file + atomic rename — a reader (or a
  writer killed mid-write) never sees a half-written file;
- **read-modify-write** takes a per-feature advisory `flock` (unix) — no lost
  updates;
- the **SQLite index** runs in WAL mode with a busy timeout, and `reindex`
  rebuilds transactionally (no file deletion), so `SQLITE_BUSY` under contention
  is waited out, never a hard failure. The file write always commits before the
  index, so truth is never lost to a DB hiccup.

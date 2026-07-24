# DevRites domain context

This is the repository's domain record. Read it with `docs/adr/` for the
reasoning behind the design and `docs/` for operating guidance. `CLAUDE.md`
points every agent here.

## What DevRites is

DevRites is a spec-driven development system for AI coding agents. It gives the
model an ordered lifecycle called the rites, so feature work follows a process
that another session can inspect and resume. A Go engine owns deterministic
bookkeeping, while the model handles judgment. See
[ADR-0001](docs/adr/0001-go-engine-as-control-plane.md).

## The two planes

- **Control plane: the engine** (`engine/`): a single stdlib-only Go binary
  (`CGO_ENABLED=0`, no model calls). It owns every deterministic operation over
  the workspace: state transitions, gates, hooks, derivations, migration. Those
  operations are network-free; explicit update/source-cache I/O is isolated in
  `internal/iohooks` (ADR-0008). The command inventory is defined by the
  hand-rolled dispatcher in `engine/main.go`; `devrites-engine help` is the
  exhaustive user-facing list.
- **Data plane: the workspace** (`.devrites/`): git-diffable Markdown. Feature
  completeness uses six single-concern **sections** (`spec`, `plan`,
  `decisions`, `tasks`, `proof`, `status`); the canonical live map/cursor/proof
  files are `README.md`, `state.md`, and `evidence.md` (ADR-0007). Workspace
  schema v2 adds phase-owned clarification and vet artifacts. The engine can
  still read v1 layouts and aliases.

## The lifecycle (rites → phases)

Fifteen ordered states mirror the `rite-*` skill arc:

```
frame → spec → clarify → temper → define → plan → vet → build → converge → prove
→ polish → review → seal → ship → done
```

Clarify is mandatory but adaptive, and it may ask no questions. Temper is
optional. Vet is the only final readiness phase; there is no separate `ready`
rite.

Completeness is **phase-relative**. The typed `phaseDefinitions` registry in
`engine/internal/state/schema.go` lists the sections and workspace artifacts
that must contain real content in each phase. Later phases require more. A
**gate** checks those requirements at a phase boundary. A blocker that only a
human can resolve becomes a **human-in-the-loop pause** and uses reserved
**exit code 3**. Missing artifacts and technical readiness failures instead
return the work to the phase that owns the fix. For example, build-readiness
`6` routes to `/rite-clarify`, `7` routes to `/rite-vet`, and `8` routes
semantically stale workspaces to `/rite-upgrade`. The build gate validates the
content, current `devrites.readiness-artifacts.v2` contract, and input digests
of `Decision coverage: CLEAR` and `Implementation readiness: READY`; marker
strings alone do not pass. See
[ADR-0003](docs/adr/0003-gate-model-hitl-pause.md)
and [ADR-0009](docs/adr/0009-prebuild-decision-coverage-and-readiness.md).
Structural workspace migration remains separate from semantic upgrade; see
[ADR-0012](docs/adr/0012-semantic-workspace-upgrades.md).

## Key concepts

| Term | Meaning |
|------|---------|
| **Rite** | A lifecycle step, surfaced as a `rite-*` skill in the pack. |
| **Section** | One single-concern completeness file in a feature dir. |
| **Phase** | Workflow state; gates are phase-relative. |
| **Gate** | Deterministic boundary check; objective failures route to their owner, while only a genuine human wait uses exit 3. |
| **Hook** | An engine subcommand (`hook <id>`) wired through Claude `settings.json` or generated Codex `hooks.json`; profiles select which fire. See [ADR-0005](docs/adr/0005-hooks-as-engine-subcommands.md). |
| **Harness** | Per-host edge adapter. Two hosts: Claude + Codex. See [ADR-0002](docs/adr/0002-dual-host-harness.md). |
| **Pack** | The installed bundle under `pack/.claude/`: reviewer and judge agents, `rite-*` skills, and `settings.json` hook wiring. |

## Repository map

| Path | What |
|------|------|
| `engine/` | The Go control plane. `internal/` owns state, gates, harness adapters, install/update semantics, explicit I/O hooks, and shared command logic. |
| `engine/tests/` | Parity/golden + unit tests, incl. `adr_NNNN_*` guard tests. |
| `pack/.claude/` | Canonical pack: 44 shipped skills and 18 agents (17 read-only leaves, one source/test wright), plus Claude hook wiring. |
| `install.sh` / `bin/` | Installer + npx entry; version is single-sourced from `package.json`. |
| `evals/` | Trigger / outcome / behavioral eval tiers with golden fixtures. |
| `docs/adr/` | Architecture decisions (start here for "why"). |
| `docs/research/` | Focused implementation studies and validation notes. |

## Invariants worth knowing

- Workspace control-plane operations make **no** network or model calls;
  explicit network I/O is confined to `internal/iohooks` (ADR-0008).
- Public rites are the authoritative orchestrators. Fresh-context leaves run
  at depth one. They fail closed if an identity guard is missing or crashes,
  and they never own human questions, phase changes, or canonical `.devrites/`
  writes
  ([ADR-0010](docs/adr/0010-agent-first-fresh-context-orchestration.md)).
- `/rite-build` derives the exact `.wright-allowlist`; the writer's report cannot
  expand it. The snapshot, reconciliation check, test/package integrity, and
  close steps all use the original slice baseline, even after a retry refresh.
- Version is **single-sourced** from `package.json`; the engine binary is stamped
  via `-ldflags` at build; install.sh + `bin/devrites.mjs` read it at runtime.
  There are no hand-maintained embedded version literals to drift.
- Wall-clock reads that feed output go through a seam (`DEVRITES_NOW`); see
  [ADR-0006](docs/adr/0006-clock-seam-and-engine-ci-gates.md).
- Engine CI is the strictest gate: `gofmt`, `go vet`, `staticcheck`,
  `govulncheck`, `go test -race`.

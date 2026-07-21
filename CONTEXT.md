# DevRites — domain context

Single-context domain record. Read this with `docs/adr/` for the "why";
`docs/` for the "how". `CLAUDE.md` points every agent here.

## What DevRites is

A spec-driven-development system for AI coding agents. It drives a model through
a disciplined lifecycle — the **rites** — so that shipping software is a
reproducible *process*, not a one-off improvisation. The core bet: separate
**deterministic bookkeeping** (owned by a Go engine) from **judgment** (owned by
the LLM). See [ADR-0001](docs/adr/0001-go-engine-as-control-plane.md).

## The two planes

- **Control plane — the engine** (`engine/`): a single stdlib-only Go binary
  (`CGO_ENABLED=0`, no model calls). It owns every deterministic operation over
  the workspace: state transitions, gates, hooks, derivations, migration. Those
  operations are network-free; explicit update/source-cache I/O is isolated in
  `internal/iohooks` (ADR-0008). The command inventory is defined by the
  hand-rolled dispatcher in `engine/main.go`; `devrites-engine help` is the
  exhaustive user-facing list.
- **Data plane — the workspace** (`.devrites/`): git-diffable Markdown. Feature
  completeness uses six single-concern **sections** (`spec`, `plan`,
  `decisions`, `tasks`, `proof`, `status`); the canonical live map/cursor/proof
  files are `README.md`, `state.md`, and `evidence.md` (ADR-0007).

## The lifecycle (rites → phases)

Fourteen ordered states mirror the `rite-*` skill arc:

```
frame → spec → temper → define → plan → vet → build → converge → prove
→ polish → review → seal → ship → done
```

Completeness is **phase-relative**: the typed `phaseDefinitions` registry in
`engine/internal/state/schema.go` says which sections and workspace artifacts
must have real content at each phase; the set grows down the arc. A **gate** checks that
completeness at a phase boundary. A blocked gate is a **human-in-the-loop
pause** — a "missing X" message and reserved **exit code 3**, never a crash. See
[ADR-0003](docs/adr/0003-gate-model-hitl-pause.md).

## Key concepts

| Term | Meaning |
|------|---------|
| **Rite** | A lifecycle step, surfaced as a `rite-*` skill in the pack. |
| **Section** | One single-concern completeness file in a feature dir. |
| **Phase** | Workflow state; gates are phase-relative. |
| **Gate** | Deterministic completeness check; blocks as exit-3 HITL pause. |
| **Hook** | An engine subcommand (`hook <id>`) wired through Claude `settings.json` or generated Codex `hooks.json`; profiles select which fire. See [ADR-0005](docs/adr/0005-hooks-as-engine-subcommands.md). |
| **Harness** | Per-host edge adapter. Two hosts: Claude + Codex. See [ADR-0002](docs/adr/0002-dual-host-harness.md). |
| **Pack** | `pack/.claude/` — the installed bundle: reviewer/judge agents, `rite-*` skills, and `settings.json` hook wiring. |

## Repository map

| Path | What |
|------|------|
| `engine/` | The Go control plane. `internal/` owns state, gates, harness adapters, install/update semantics, explicit I/O hooks, and shared command logic. |
| `engine/tests/` | Parity/golden + unit tests, incl. `adr_NNNN_*` guard tests. |
| `pack/.claude/` | Canonical pack: 42 skills and 14 agents (13 read-only, one write-capable builder), plus Claude hook wiring. |
| `install.sh` / `bin/` | Installer + npx entry; version is single-sourced from `package.json`. |
| `evals/` | Trigger / outcome / behavioral eval tiers with golden fixtures. |
| `docs/adr/` | Architecture decisions (start here for "why"). |
| `docs/research/` | Studies, incl. `gsd-core-adoption.md` (peer-system teardown + roadmap). |

## Invariants worth knowing

- Workspace control-plane operations make **no** network or model calls;
  explicit network I/O is confined to `internal/iohooks` (ADR-0008).
- Version is **single-sourced** from `package.json`; the engine binary is stamped
  via `-ldflags` at build; install.sh + `bin/devrites.mjs` read it at runtime.
  There are no hand-maintained embedded version literals to drift.
- Wall-clock reads that feed output go through a seam (`DEVRITES_NOW`); see
  [ADR-0006](docs/adr/0006-clock-seam-and-engine-ci-gates.md).
- Engine CI is the strictest gate: `gofmt`, `go vet`, `staticcheck`,
  `govulncheck`, `go test -race`.

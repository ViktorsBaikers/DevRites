# DevRites domain context

This is the repository's domain record. Read it with `docs/adr/` for the
reasoning behind the design and `docs/` for operating guidance. `CLAUDE.md`
points every agent here.

## What DevRites is

DevRites is a spec-driven development system for AI coding agents. It gives the
model an ordered lifecycle called the rites, so feature work follows a process
that another session can inspect and resume. A Go engine owns the retained
deterministic safety/atomic primitives, while native skills and exact agents
handle workflow policy and judgment. See
[ADR-0001](docs/adr/0001-go-engine-as-control-plane.md) and
[ADR-0022](docs/adr/0022-native-orchestration-thin-engine.md), as narrowed by
[ADR-0024](docs/adr/0024-native-policy-offline-installer-boundary.md), as
amended by [ADR-0028](docs/adr/0028-self-contained-engine-update.md).

## The two planes

- **Control plane: the engine** (`engine/`): a single stdlib-only Go binary
  (`CGO_ENABLED=0`, no model or network calls). It owns local managed
  install/update/uninstall against caller-supplied candidates, structural
  and content-bound Build readiness, final structural plus identity and
  evidence-freshness checks, atomic
  answer/drop/batch resolve and transactional close, secret scanning, and
  version reporting. `devrites-engine help` is exhaustive.
- **Semantic plane: native hosts, skills, and agents**: Claude Code and Codex
  own agent discovery, dispatch, waiting, and result delivery. Installed skills
  and exact custom roles own semantic readiness, traceability,
  acceptance/evidence quality, doubt, reviewer reconciliation, test-quality
  assessment, capability interpretation, upgrade,
  recovery routing, normative spec grammar re-read, qid allocation, Clarify
  cursor edits, AFK/recovery accounting, and read-only diagnostics. Shell/npm
  entrypoints own release bundle/source/binary acquisition.
- **Data plane: the workspace** (`.devrites/`): git-diffable Markdown. Feature
  completeness uses six logical **sections** (`spec`, `plan`, `decisions`,
  `tasks`, `proof`, `status`); the canonical live map/cursor/proof files are
  `README.md`, `state.md`, and `evidence.md` (ADR-0007). The current state schema
  is v2; the runtime also directly reads official v1/v2 bullet cursors.

## The lifecycle (rites → phases)

Fifteen ordered states mirror the `rite-*` skill arc:

```
frame → spec → clarify → temper → define → plan → vet → build → converge → prove
→ polish → review → seal → ship → done
```

Clarify is mandatory but adaptive, and it may ask no questions. Temper is
optional. Vet is the only final readiness phase; there is no separate `ready`
rite.

Completeness is **phase-relative**. The typed Phase Policy in
`engine/internal/state/schema.go` lists the structural sections, workspace
artifacts, and applicability rules for each target Phase. `devrites-engine check readiness <slug>`
checks that structure and, after Vet, the stable planning-input identity;
`check seal <slug>` repeats that identity check and adds deterministic evidence
freshness. A blocker that only a human can resolve uses reserved **exit code 3**.
The active skill and exact reviewers—not Go heuristics—judge whether the spec,
plan, traceability, tests, and evidence mean what they claim. Semantic upgrade
is a native, preservation-first workflow edit. See
[ADR-0003](docs/adr/0003-gate-model-hitl-pause.md) and
[ADR-0022](docs/adr/0022-native-orchestration-thin-engine.md), as narrowed by
[ADR-0027](docs/adr/0027-content-bound-build-readiness.md).

## Key concepts

| Term | Meaning |
|------|---------|
| **Rite** | A lifecycle step, surfaced as a `rite-*` skill in the pack. |
| **Section** | One single-concern completeness file in a feature dir. |
| **Phase** | Workflow state; gates are phase-relative. |
| **Gate** | Deterministic structural boundary check; semantic findings route through the native workflow, while exit 3 represents a lifecycle or safety block. |
| **Workspace Observation** | One deterministic, safe, typed account of lifecycle-owned artifacts at a point in time; every consumer uses the same facts rather than re-reading the workspace. |
| **Phase Policy** | The deterministic requirements and applicability rules for a target phase, including required artifacts, open-question blocking, and proof requirements. |
| **Acceptance-preserving Reslice** | A change to plan or task topology that leaves acceptance criteria and product behavior unchanged. |
| **Workflow Artifact** | An executable file used only to plan, isolate, or prove an active workflow; it has separate identity and evidence and is excluded from the product candidate and readiness binding. |
| **Harness** | Per-host edge adapter. Two hosts: Claude + Codex. See [ADR-0002](docs/adr/0002-dual-host-harness.md). |
| **Pack** | The installed bundle under `pack/.claude/`: reviewer/writer agents, `rite-*` skills, and native host configuration. |

## Repository map

| Path | What |
|------|------|
| `engine/` | The thin Go control plane: self-update plus local managed install, structural checks, retained atomic state, secret scan, and version. |
| `engine/tests/` | Parity/golden + unit tests, incl. `adr_NNNN_*` guard tests. |
| `pack/.claude/` | Canonical skills, agents, standards, and Claude configuration; Codex artifacts are generated from it. |
| `install.sh` / `bin/` | Installer + npx entry; version is single-sourced from `package.json`. |
| `evals/` | Trigger / outcome / behavioral eval tiers with golden fixtures. |
| `docs/adr/` | Architecture decisions (start here for "why"). |
| `docs/research/` | Focused implementation studies and validation notes. |

## Invariants worth knowing

- The engine makes **no model calls**. Network access is isolated to bounded,
  checksummed latest-release acquisition for direct `devrites-engine update`;
  every workspace policy, state, proof, and install-application package remains
  network-free (ADR-0028).
- Public rites are the authoritative orchestrators. Fresh-context leaves run
  at depth one through exact named profiles; there is no generic-agent
  fallback. Reviewer leaves are natively read-only. Claude keeps the root in
  plan mode; Codex uses a workspace-capable root because a child cannot elevate
  above its parent, so the root's source/test non-writing boundary is
  instruction-enforced there. Both hosts make only the exact slice-wright
  writable among specialists; the task states its exact paths and the root
  rejects an out-of-scope diff. Leaves never own human questions, phase
  changes, or canonical
  `.devrites/` writes
  ([ADR-0010](docs/adr/0010-agent-first-fresh-context-orchestration.md),
  [ADR-0015](docs/adr/0015-read-only-root-native-orchestration.md),
  [ADR-0018](docs/adr/0018-native-sandbox-instruction-writer-boundary.md)).
- `/rite-build` states the exact project-relative paths in the writer task; the
  writer may not expand them. The root compares the returned file list and `git
  diff --name-only` with that contract, reviews test integrity, and runs
  repository proof.
- Version is **single-sourced** from `package.json`; the engine binary is stamped
  via `-ldflags` at build; install.sh + `bin/devrites.mjs` read it at runtime.
  There are no hand-maintained embedded version literals to drift.
- Wall-clock reads that feed output go through a seam (`DEVRITES_NOW`); see
  [ADR-0006](docs/adr/0006-clock-seam-and-engine-ci-gates.md).
- Engine CI is the strictest gate: `gofmt`, `go vet`, `staticcheck`,
  `govulncheck`, `go test -race`.

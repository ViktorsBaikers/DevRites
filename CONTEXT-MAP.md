# DevRites context map

Progressive disclosure for agents and humans: start here, then load depth only
when the task needs it.

## Layer 0 — Always

| Need | Read |
| --- | --- |
| Operating rules, precedence, evidence | `pack/.claude/skills/devrites-lib/reference/standards/core.md` |
| Which standard applies | `pack/.claude/skills/devrites-lib/reference/standards/README.md` |
| Tool routing (structural vs literal) | `pack/.claude/skills/devrites-lib/reference/standards/tooling.md` + `code-navigation.md` |
| Repo agent contract | `AGENTS.md` |

## Layer 1 — Phase owner

| Phase | Skill | Deep references |
| --- | --- | --- |
| Spec / Clarify / Define | `rite-spec`, `rite-clarify`, `rite-define` | `spec-grammar.md`, `elicitation.md` |
| Plan | `rite-plan` | `replan-and-repair.md`, `acceptance-preserving-reslice.md` |
| Vet | `rite-vet` | `review-axes.md`, `artifacts.md`, `depth.md` |
| Build | `rite-build` | `one-slice-cycle.md`, `parallel-batch.md` |
| Prove / Review / Seal | `rite-prove`, `rite-review`, `rite-seal` | `definition-of-done.md`, checklists |

## Layer 2 — Engine (deterministic)

| Question | Command |
| --- | --- |
| Workspace snapshot for agents | `devrites-engine orient <slug>` (alias: `observe summary`) |
| Index / manifest presence | `devrites-engine check indexes [--root <dir>]` |
| Readiness / seal gates | `devrites-engine check readiness <slug>` or `check seal <slug>` |
| Engine working rules | [`engine/AGENTS.md`](engine/AGENTS.md) |
| Install health (pack + host) | `/rite-doctor` (skill; not an engine command) |

## Layer 3 — Code intelligence (optional)

See `code-navigation.md` for the decision tree. Indexes live under
`.codegraph/`, `.code-review-graph/`, `.codebase-memory/`.

## Layer 4 — Audit / release

| Topic | Location |
| --- | --- |
| Command inventory | `docs/command-map.md` |
| CI / validate | `scripts/validate.sh`, `.github/workflows/ci.yml` |
| Pack audit closeout | `.scratch/pack-guidance-audit-2026-08/` (when present) |

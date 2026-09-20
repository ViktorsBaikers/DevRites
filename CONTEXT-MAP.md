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
| Minimal remaining path | `devrites-engine next <slug>` |
| Resume record for handoff/compaction | `devrites-engine handoff [slug]` |
| One-file read-set for a phase/role/skill | `devrites-engine context <slug> (--phase <p> \| --skill <name>) [--role <r>] [--trigger a,b]` |
| Per-feature event ledger | `devrites-engine metrics summary <slug>` |
| Same-tree session file claims | `devrites-engine claim <add\|release\|list\|check>` (advisory, `.devrites/claims.jsonl`) |
| Returned paths ⊆ contract | `devrites-engine check diff-scope <slug> --allow <paths>` |
| Slice sound before wright dispatch | `devrites-engine check slice <slug> <SLICE-ID>` |
| Index / manifest presence | `devrites-engine check indexes [--root <dir>]` |
| Readiness / seal gates | `devrites-engine check readiness <slug>` or `check seal <slug>` |
| Progress regression vs baseline | `devrites-engine check regression <slug> [--update]` |
| Which readiness input drifted since Vet | `devrites-engine check drift <slug> [--record]` (advisory) |
| Real repo test/lint/build commands | `devrites-engine detect commands [--root <dir>] [--json]` (read-only) |
| Near-duplicate code leads | `devrites-engine check dup [slug] [--all|--worktree|--staged|--base <ref>]` (advisory) |
| Acceptance ledger | `devrites-engine gates <sub> <slug>`; grammar in `devrites-lib/reference/standards/gates.md` |
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

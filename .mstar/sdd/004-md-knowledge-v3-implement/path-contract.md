# Path contract — 004-md-knowledge-v3-implement

- **Plan:** `004-md-knowledge-v3-implement`
- **Branch:** `plan/004-md-knowledge-v3-implement` @ `d0d356cdfcd88d2578a6e895e85f17d7a784db3b`
- **Integration target:** `iteration/md-knowledge-v3`
- **Recorded:** 2026-08-27

## Research lock GO

| Field | Value |
| --- | --- |
| status | **GO** |
| base_main_sha | `1da70ceced71b7e6c27cc204a06ff3b2926f932a` |
| blockers | none |
| R1–R7 | all pass — see `guides/target-architecture.md` § Research lock GO |
| touch_list SSOT | `.mstar/iterations/iter-md-knowledge-v3/guides/target-architecture.md` §2 |

## Byte baseline (always-loaded SSOT)

Measurement set per `.mstar/iterations/iter-md-knowledge-v3/guides/devrites-md-architecture-map.md`:

`AGENTS.md` + `CLAUDE.md` + `CONTEXT.md` + `core.md` + all `pack/.claude/skills/**/SKILL.md`

| Phase | Root + core | Skill aggregate | **Total always-loaded** | Ceiling (+10%) |
| --- | ---: | ---: | ---: | ---: |
| 003 pre-edit @ `1da70cec` | 36,191 | 241,032 | **277,223** | 304,945 |
| 004 post-edit @ `d0d356cd` | 36,191 | 245,261 | **281,452** | 304,945 |
| **Delta** | 0 | +4,229 | **+4,229 (+1.5%)** | within budget |

Post-edit ratchet: `node scripts/check-instruction-size-baseline.mjs --write` → `tests/instruction-size-baseline.json` (`total_bytes`: 898,615).

## Touch-list checklist (canonical authored)

| ID | Path | Action | Status |
| --- | --- | --- | --- |
| T01 | `pack/.claude/skills/devrites-lib/reference/intent-map.md` | modify | done |
| T02 | `pack/.claude/skills/devrites-lib/reference/standards/edge-case-trace.md` | modify | done |
| T03 | `pack/.claude/skills/rite-prove/reference/acceptance-proof.md` | modify | done |
| T04 | `pack/.claude/skills/rite-spec/reference/spec-template.md` | modify | done |
| T05 | `pack/.claude/skills/rite-spec/reference/spec-checklists.md` | modify | done |
| T06 | `pack/.claude/skills/rite-define/reference/plan-template.md` | modify | done |
| T07 | `pack/.claude/skills/rite-adopt/SKILL.md` | modify | done |
| T08 | `pack/.claude/skills/rite-vet/SKILL.md` | modify | done |
| T09 | `pack/.claude/skills/rite-build/SKILL.md` | modify | done |
| T10 | `pack/.claude/skills/rite-prove/SKILL.md` | modify | done |
| T11 | `pack/.claude/skills/rite-converge/SKILL.md` | modify | done |
| T12 | `pack/.claude/skills/rite-review/SKILL.md` | modify | done |
| T13 | `pack/.claude/skills/rite-pressure-test/SKILL.md` | modify | done |
| T14 | `pack/.claude/skills/rite-polish/SKILL.md` | modify | done |
| T15 | `pack/.claude/skills/rite-learn/SKILL.md` | modify | done |
| T16 | `pack/.claude/skills/devrites-lib/reference/standards/tooling.md` | modify | done |
| T17 | `pack/.claude/skills/devrites-lib/reference/standards/security.md` | modify | done |
| T18 | `pack/.claude/skills/devrites-lib/reference/standards/debug-recovery.md` | modify | done |
| T19 | `pack/.claude/skills/devrites-lib/reference/standards/agents.md` | modify | done |
| T20 | `pack/.claude/skills/devrites-lib/reference/visual-playbooks/index.md` | modify | done |
| T21 | `pack/.claude/agents/devrites-code-reviewer.md` | modify | done |
| T22 | `pack/.claude/agents/devrites-security-auditor.md` | modify | done (amended: `devrites-security-reviewer.md` absent) |
| T23 | `pack/.claude/agents/devrites-spec-reviewer.md` | modify | done |
| T24 | `pack/.claude/agents/devrites-devex-reviewer.md` | modify | done |
| T25 | `pack/.claude/agents/devrites-doubt-reviewer.md` | modify | done |
| T26 | `evals/README.md` | modify | done |
| T27 | `NOTICE.md` | modify | verified — attributions already present from #44; no delta required |
| T28 | `docs/markdown-instruction-upgrade-2026-08-27.md` | create | done |

**Regenerated (script only):** `pack/generated/**` via `bash scripts/build-host-artifacts.sh` after T01–T27.

## Consolidations (≥3)

| ID | Source → target | Rationale |
| --- | --- | --- |
| C1 | Repeated grep reassurance → `tooling.md` primary-first gate | Single falsifiable orient check |
| C2 | Per-reviewer output headers → `agents.md` finding shape | One schema for all reviewers |
| C3 | Polish completeness vs craft axes → `rite-polish/SKILL.md` table | Distinct failing cases per axis |

## Forbidden paths (honored)

- Non-touch-list `pack/**` — not edited
- Hand-edits under `pack/generated/**` — none (regen only)
- Engine, fixtures, CI, non-Markdown authored files — none (see scope gate in upgrade doc)
- `/tmp` research clones — not committed

## Ledger alignment

All touch-list rows map to `guides/adoption-matrix.md` adopt/combine rows and `guides/prior-adoption-ledger.md` strengthen/retain entries. Overturn rows (#44/#45 prose-only checks without failing cases) documented in upgrade doc § Overturn log and `per-file-contract.md`.

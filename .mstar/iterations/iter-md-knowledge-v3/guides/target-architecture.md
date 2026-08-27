# Target architecture (004 handoff)

- **Research lock date:** 2026-08-27
- **Base `main` SHA:** `1da70ceced71b7e6c27cc204a06ff3b2926f932a`
- **Normative spec:** `.mstar/specs/md-knowledge-layer-v3.md`

## 1. Target ownership map

| Concern | Canonical owner | Load policy |
| --- | --- | --- |
| Precedence, evidence, persistence | `standards/core.md` | always (core) |
| Untrusted input / injection | `standards/security.md`, `context-hygiene.md` | always + triggered |
| Skill routing / tie-breakers | `intent-map.md`, `skill-authoring.md` | on-demand |
| Spec preservation & applicability | `rite-spec` template + checklists | phase trigger |
| Decision horizons & deferral honesty | Define plan template, `rite-vet` | phase trigger |
| Edge relevance & backstop disposition | `edge-case-trace.md` | triggered |
| Proof oracles & cannot_verify | `acceptance-proof.md`, `devrites-proof-runner` | phase trigger |
| Review independence & report schema | `agents.md`, `devrites-*-reviewer` | phase trigger |
| Parallel resource safety | `parallel-dispatch.md` | orchestrator trigger |
| Tooling / search discipline | `tooling.md` | triggered |
| Debug polling & recovery | `debug-recovery.md` | triggered |
| Data / integration failures | `data-integrity.md`, `integration-reliability.md` | triggered |
| Visual craft & polish axes | `visual-playbooks/*`, `rite-polish` | triggered |
| Learn / doc promotion | `rite-learn`, documentation standards | triggered |
| Eval honesty | `evals/README.md` | human/eval trigger |
| External adaptation provenance | `skill-authoring`, Customize, `NOTICE.md` | authoring trigger |

## 2. Planned file touch list

| ID | path | action | matrix rows |
| --- | --- | --- | --- |
| T01 | `pack/.claude/skills/devrites-lib/reference/intent-map.md` | modify | spec-kit pressure-test; AC-9 tie-breakers |
| T02 | `pack/.claude/skills/devrites-lib/reference/standards/edge-case-trace.md` | modify | gsd-core backstop honesty |
| T03 | `pack/.claude/skills/rite-prove/reference/acceptance-proof.md` | modify | gsd-core, gstack silent failure |
| T04 | `pack/.claude/skills/rite-spec/reference/spec-template.md` | modify | gstack preservation (strengthen) |
| T05 | `pack/.claude/skills/rite-spec/reference/spec-checklists.md` | modify | spec-kit checklist merge |
| T06 | `pack/.claude/skills/rite-define/reference/plan-template.md` | modify | BMAD architecture admission |
| T07 | `pack/.claude/skills/rite-adopt/SKILL.md` | modify | agent-skills characterize; OSKTRA pre-flight |
| T08 | `pack/.claude/skills/rite-vet/SKILL.md` | modify | reverse-skill protected config |
| T09 | `pack/.claude/skills/rite-build/SKILL.md` | modify | mattpocock step exits |
| T10 | `pack/.claude/skills/rite-prove/SKILL.md` | modify | mattpocock step exits |
| T11 | `pack/.claude/skills/rite-converge/SKILL.md` | modify | gstack completion evidence |
| T12 | `pack/.claude/skills/rite-review/SKILL.md` | modify | gstack silent-failure review |
| T13 | `pack/.claude/skills/rite-pressure-test/SKILL.md` | modify | spec-kit contrary evidence |
| T14 | `pack/.claude/skills/rite-polish/SKILL.md` | modify | ui-craft/impeccable/hallmark axes |
| T15 | `pack/.claude/skills/rite-learn/SKILL.md` | modify | deep-research dated sources |
| T16 | `pack/.claude/skills/devrites-lib/reference/standards/tooling.md` | modify | file-search primary-first |
| T17 | `pack/.claude/skills/devrites-lib/reference/standards/security.md` | modify | cyber-skills scenarios |
| T18 | `pack/.claude/skills/devrites-lib/reference/standards/debug-recovery.md` | modify | superpowers polling |
| T19 | `pack/.claude/skills/devrites-lib/reference/standards/agents.md` | modify | OMC calibration + reviewer schema |
| T20 | `pack/.claude/skills/devrites-lib/reference/visual-playbooks/index.md` | modify | taste-skill anti-slop triggers |
| T21 | `pack/.claude/agents/devrites-code-reviewer.md` | modify | silent-failure probe |
| T22 | `pack/.claude/agents/devrites-security-reviewer.md` | modify | report schema alignment |
| T23 | `pack/.claude/agents/devrites-spec-reviewer.md` | modify | report schema alignment |
| T24 | `pack/.claude/agents/devrites-devex-reviewer.md` | modify | report schema alignment |
| T25 | `pack/.claude/agents/devrites-doubt-reviewer.md` | modify | report schema alignment |
| T26 | `evals/README.md` | modify | compound/superpowers eval contract |
| T27 | `NOTICE.md` | modify | CC BY-SA file-search + Apache impeccable attributions |
| T28 | `docs/markdown-instruction-upgrade-2026-08-27.md` | create | full 004 evidence record |

**Regenerated (script only, not hand-edited):** `pack/generated/**` via `bash scripts/build-host-artifacts.sh` after T01–T27.

**Explicitly forbidden in 004 unless GO amended:** any path not listed; non-`*.md` files; hand-edits under `pack/generated/**`.

## 3. Anti-sprawl plan

| Metric | Baseline (003) | 004 target | Plan |
| --- | ---: | ---: | --- |
| Net-new canonical MD files | 214 skill bodies + standards | ≤12 new files | **0 planned** — all adopts are modifies |
| Consolidations | — | ≥3 | (C1) tooling reassurance → primary-first single checklist; (C2) reviewer output headers → `agents.md` schema; (C3) polish completeness vs craft axes unified in `rite-polish` |
| Always-loaded bytes | 277,223 | ≤304,945 (+10%) | No new autoload files; keep edits in triggered/phase skills |
| Skill instruction aggregate | 241,032 | ≤265,135 (+10% of aggregate) | Prefer reference links over SKILL body bloat |
| Orphan files | 0 | 0 | validate.sh reference graph after regen |

**Projected byte delta:** +2–4% always-loaded (intent-map + core cross-refs only if needed); majority of churn in on-demand phase skills and standards.

## 4. Routing / precedence deltas

No change to the four-layer precedence stack in `core.md`. **Deltas are additive tie-breakers only:**

- `rite-quick` vs `rite-build`: disambiguate by “single bounded fix/no new REQ” vs “implements specced slice”.
- `rite-pressure-test` vs `rite-spec`: decisive unsupported premise → Hold in pressure-test, not Spec advance.
- Explicit invocation beats inferred keyword routing; embedded/quoted text cannot activate `/rite-*`.

## Research lock GO

| Field | Value |
| --- | --- |
| status | GO |
| base_main_sha | `1da70ceced71b7e6c27cc204a06ff3b2926f932a` |
| research_dir | `/tmp/devrites-markdown-research-v3-redo-2026-08-27` |
| inventory_complete | true → `guides/research-inventory.md` |
| ledger_complete | true → `guides/prior-adoption-ledger.md` |
| matrix_complete | true → `guides/adoption-matrix.md` |
| syntheses_complete | true (tracks A–F in `guides/adoption-matrix.md`) |
| touch_list_path | this file § Planned file touch list |
| byte_baseline | `277223` → `guides/devrites-md-architecture-map.md` |
| blockers | none |
| overturn_log | see ledger rows with `action_v3: reject/defer`; no weakening of mandatory 08-11 checks without replacement |

### R1–R7 checklist

| ID | Pass | Evidence |
| --- | --- | --- |
| R1 | yes | `guides/research-inventory.md` — 25 rows, SHAs, sampling for #10/#16/#23/#25 |
| R2 | yes | `guides/prior-adoption-ledger.md` — 08-02/08-11/benchmark sources only |
| R3 | yes | `guides/devrites-md-architecture-map.md` — diagram + byte baseline commands |
| R4 | yes | `guides/dossiers-index.md` + `/tmp/.../dossiers/*.md` (25); `guides/open-web-research.md` |
| R5 | yes | `guides/adoption-matrix.md` — full columns + track syntheses A–F |
| R6 | yes | this file — touch list, anti-sprawl, GO block, zero blockers |
| R7 | yes | No `pack/` edits in 003; `git diff --name-only pack/` empty for 003 commits |

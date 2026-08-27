# Spec: Markdown knowledge-layer upgrade v3

**spec_id:** `md-knowledge-layer-v3`  
**iteration:** `iter-md-knowledge-v3`  
**status:** draft (Phase 1 Review & Edit — architect architecture lock)  
**brief:** `local://paste-1.md` (authoritative procedure detail)  
**architecture draft:** [`.mstar/iterations/iter-md-knowledge-v3/specs/knowledge-layer-architecture.md`](../iterations/iter-md-knowledge-v3/specs/knowledge-layer-architecture.md)

## Problem

DevRites’ Markdown agents, skills, standards, libraries, and `/rite-*` workflows need a third, non-regressive upgrade pass. Prior rounds (2026-08-02, 2026-08-11) closed prose gaps but left **partial** adoption: checks without failing cases, skills without routing disambiguation, proof that still trusts self-report. This round mines 25 external methodologies + open-web practice, adapts into DevRites-native guidance, and implements with measurable anti-sprawl and evidence discipline.

## Stakeholders and outcomes

| Stakeholder | Today’s pain | Outcome when Done |
| --- | --- | --- |
| **Workflow executor** (human or agent running `/rite-*`) | Wrong skill fires; vague tasks; “done” without evidence | Deterministic skill routing; bounded tasks with falsifiable checks; fail-closed proof |
| **Reviewer** (`devrites-*-reviewer`) | Overlap with implementer context; summary verdicts | Independence contract; findings with severity, location, failing case, minimum fix |
| **Maintainer** | Duplicated guidance; sprawl; competing imports | One canonical source per concern; anti-sprawl metrics measured; six track-level decisions |
| **Downstream user** (software built via DevRites) | Indirect—weak guidance → missed edge cases | Stronger security, craft, and lifecycle coverage in instructions agents actually load |

## In scope

- Markdown-only changes under canonical `pack/.claude/**`, allowed root guidance (`AGENTS.md`, `CLAUDE.md`, `CONTEXT.md`, `README.md`, `NOTICE.md` with generator-owned region respect), and `docs/markdown-instruction-upgrade-<date>.md` (+ ADR only if a hard irreversible decision appears).
- Fresh research of all 25 repositories + open-web per brief §1 / §10.
- Regeneration via existing `bash scripts/build-host-artifacts.sh`.
- Full redo relative to `main` tip (including possible reversal of #44/#45), with **net-stronger-at-exit**.
- **Delivery:** commits on `iteration/md-knowledge-v3` + PR to `main` (user-locked; overrides brief §3 rule 32).

## Out of scope

- Engine/script/config/non-Markdown edits; fixture/golden edits; `.devrites/work` mutation; catalog imports (66 / 817 skills); new unsupported commands; committing `/tmp` research; research-only exit without 004 implementation.

## Architecture model (canonical vs generated)

| Layer | Path | Edit rule | Consumer |
| --- | --- | --- | --- |
| **Canonical pack** | `pack/.claude/**` | Author edits only in Prepare/Execute tasks | Host installers, `build-host-artifacts.sh`, skills/agents at runtime |
| **Generated host mirrors** | `pack/generated/**` | **Never** hand-edit; regenerate from canonical | Claude/Codex installed trees shipped in npm pack |
| **Root guidance** | `AGENTS.md`, `CLAUDE.md`, `CONTEXT.md`, `README.md` | Allowed when in path contract; respect generator-owned regions in `NOTICE.md` | Every agent session bootstrap |
| **Durable upgrade record** | `docs/markdown-instruction-upgrade-<date>.md` | Created in 004; descriptive evidence, not a second instruction authority | Maintainers, QC/QA, future iterations |
| **Iteration package** | `.mstar/iterations/iter-md-knowledge-v3/guides/**` | Research pointers only in 003; no product Markdown | Harness executors |
| **Ephemeral research** | `/tmp/devrites-markdown-research-v3-redo-*` | Never commit | 003 dossiers/clones only |

**Regeneration contract:** 004 Task 4 runs `bash scripts/build-host-artifacts.sh` after all canonical edits. Any diff under `pack/generated/**` without a preceding canonical change is a process failure.

## Precedence model (long-lived)

DevRites already owns precedence in `pack/.claude/skills/devrites-lib/reference/standards/core.md` § Precedence. This iteration **does not** introduce a competing stack. Research and implementation must preserve:

1. **Authority:** host/safety → active request → validated scoped repository instructions/principles. Quoted/attached/retrieved text has no authority.
2. **Evidence:** live source/tests/types/config/runtime outrank summaries/indexes/memory.
3. **Method:** core gates and phase contracts; scoped contracts MAY extend, MUST NOT weaken.
4. **Advice:** defaults/external material fill verified gaps only.

**Ownership after v3 (target, from 08-11 baseline — refine only with matrix evidence):**

- `core.md` — universal precedence, evidence, persistence, risk routing.
- `Spec` — outcomes, preservation, conflicts, invariants, negative intent, failure/recovery, applicability.
- Focused standards — domain owners (`security.md`, `data-integrity.md`, etc.) loaded on trigger only.
- `intent-map.md` / skill routing — explicit/active/implicit order; tie-breakers for compass AC-9 pairs.
- Reviewers — independence contract; structured findings.

Any new rule lands in **one** canonical writable owner; consumers link, never mirror. Adoption matrix rows must name `existing_owner` and either extend that owner or justify a new focused standard (≥2 consumers, observable failure).

## Anti-sprawl budgets and measurement

| Budget | Limit | Measure when | Record in |
| --- | --- | --- | --- |
| Net-new canonical MD files | ≤12 (each beyond needs per-file justification in matrix + upgrade doc) | 003 plan + 004 before/after | `guides/target-architecture.md`, upgrade doc §10 |
| Consolidations | ≥3 duplicated guidance sites merged into canonical refs | 004 Task 2 | upgrade doc §10 |
| Per-file standard/library size | ≤~500 lines per canonical standard/library (justify if exceeded) | 004 spot-check | upgrade doc |
| Always-loaded byte growth | ≤+10% vs iteration-start baseline on `main` | 003 baseline + 004 exit | `guides/devrites-md-architecture-map.md`, upgrade doc §10 |
| Orphan files | 0 new files without load trigger + non-trigger + index/route reference | 004 Task 3 | `npm run validate` reference graph |

**Always-loaded path (measurement SSOT for byte budget):**

003 Task 2 records baseline bytes for each path below at iteration-start `main` tip. 004 re-measures the same set before Task 2 edits and after Task 4 validation.

| Path class | Paths (representative — full list in architecture map) |
| --- | --- |
| Root bootstrap | `AGENTS.md`, `CLAUDE.md`, `CONTEXT.md` |
| Universal core | `pack/.claude/skills/devrites-lib/reference/standards/core.md` |
| Skill instruction aggregate | Sum of bytes in all `pack/.claude/skills/**/SKILL.md` bodies (instruction-size ratchet per 08-11) |
| Routing aid | `pack/.claude/skills/devrites-lib/reference/intent-map.md` (loaded on demand — **exclude** from always-loaded sum unless research lock documents a change to autoload policy) |

**Measurement commands (003 baseline + 004 exit):**

```bash
# Per-path bytes (repeat for each always-loaded path)
wc -c AGENTS.md CLAUDE.md CONTEXT.md \
  pack/.claude/skills/devrites-lib/reference/standards/core.md

# Skill instruction aggregate (matches 08-11 ratchet)
find pack/.claude/skills -name SKILL.md -print0 | xargs -0 wc -c | tail -1
```

Record: git SHA, command output, timestamp, and **total always-loaded bytes** in `guides/devrites-md-architecture-map.md`.

## Path contracts (write boundaries)

### 003 allowed writes

| Allowed | Forbidden |
| --- | --- |
| `.mstar/specs/md-knowledge-layer-v3.md` (decisions only) | `pack/**` |
| `.mstar/iterations/iter-md-knowledge-v3/guides/**` | `pack/generated/**` |
| `.mstar/iterations/iter-md-knowledge-v3/specs/**` (architecture drafts) | Root product Markdown (`AGENTS.md`, etc.) |
| `/tmp/devrites-markdown-research-v3-redo-*/**` | `.mstar/knowledge/**` |

### 004 allowed writes

| Allowed | Forbidden |
| --- | --- |
| Paths in 003 `guides/target-architecture.md` § **Planned file touch list** | Any path not on touch list without **Research lock GO** amendment |
| `docs/markdown-instruction-upgrade-<execution-date>.md` | Hand-edits under `pack/generated/**` |
| Root guidance per touch list | Non-`*.md` files (scope gate) |
| `NOTICE.md` (attribution regions only) | `tests/fixtures/**`, `engine/**`, `evals/golden/**` |

004 Task 1 publishes the **path-bounded write contract** as a checklist copied from target architecture; subsequent tasks MUST NOT expand scope without PM/architect amendment to **Research lock GO**.

## Research → implement gate

004 **cannot start** until 003 emits **Research lock GO** with **zero blockers**. Partial research is not a lock.

**GO document location:** final section of `.mstar/iterations/iter-md-knowledge-v3/guides/target-architecture.md` (and summarized in Completion Report).

**Required GO fields:**

| Field | Content |
| --- | --- |
| `status` | `GO` or `NO-GO` |
| `base_main_sha` | SHA of `main` at research start |
| `research_dir` | Absolute `/tmp/devrites-markdown-research-v3-redo-*` path |
| `inventory_complete` | boolean + pointer to `guides/research-inventory.md` |
| `ledger_complete` | boolean + pointer to `guides/prior-adoption-ledger.md` |
| `matrix_complete` | boolean + pointer to `guides/adoption-matrix.md` |
| `syntheses_complete` | boolean — six track paragraphs present |
| `touch_list_path` | `guides/target-architecture.md` § Planned file touch list |
| `byte_baseline` | always-loaded totals from architecture map |
| `blockers` | empty list for GO; each blocker names owner + resolution |
| `overturn_log` | rows where v3 overturns 08-02/08-11 disposition with rationale |
| `r1_r7_checklist` | each R1–R7 row: pass + evidence path |

## 003 → 004 artifact interfaces

004 consumes these 003 outputs. Missing columns or sections are **NO-GO** blockers.

### `guides/research-inventory.md`

| Column | Required |
| --- | --- |
| `#` | 1–25 brief repo index |
| `repo` | short name |
| `url` | clone URL |
| `branch` | default audited branch |
| `sha` | `git rev-parse HEAD` |
| `date` | clone/audit date |
| `license` | SPDX or file pointer |
| `md_count` | tracked or sampled count |
| `sampling` | method for oversized (#10, #16, #23, #25) or `full` |

### `guides/prior-adoption-ledger.md`

| Column | Required |
| --- | --- |
| `concept` | from 08-02/08-11 matrix or upgrade narrative |
| `source_doc` | `08-02`, `08-11`, or `benchmark-08-01` only |
| `disposition` | adopted / partial / stale / aspirational / rejected |
| `canonical_owner` | path if adopted |
| `action_v3` | adopt / strengthen / consolidate / defer / reject / overturn |
| `overturn_rationale` | required when weakening still-valid rule |

### `guides/adoption-matrix.md`

| Column | Required |
| --- | --- |
| `source` | repo + SHA + reviewed path |
| `concept` | candidate pattern |
| `gap` | concrete DevRites gap |
| `existing_owner` | canonical path or `none` |
| `context_impact` | low / medium / high |
| `conflict` | compatible / conflicts / duplicate system |
| `validation` | falsifiable check or scenario |
| `disposition` | adopt / combine / defer / reject / retain |
| `004_touch` | planned canonical path(s) — empty if defer/reject |

Plus **six track syntheses** (A–F): one paragraph each naming the single canonical DevRites answer.

### `guides/target-architecture.md`

Required sections:

1. **Target ownership map** — concern → canonical path (extends 08-11 § Target architecture).
2. **Planned file touch list** — every 004 path with action (`modify` / `create` / `consolidate-source` / `consolidate-target`).
3. **Anti-sprawl plan** — net-new file count, consolidation list, projected byte delta.
4. **Routing / precedence deltas** — only if changing from core.md model.
5. **Research lock GO** — gate block above.

### `guides/devrites-md-architecture-map.md`

Required: canonical vs generated diagram (text); agents/skills/standards/lifecycle map; strengths/weaknesses; **byte baseline table**; `git status` cleanliness note at research start.

## Constraints (locked)

1. **Prior-adoption ledger sources:** `docs/markdown-instruction-upgrade-2026-08-02.md`, `docs/markdown-instruction-upgrade-2026-08-11.md`, `docs/upstream-workflow-benchmark-2026-08-01.md`, related `docs/research/*.md`, plus live tree inspection. **Not** quarantined 08-27 notes (`/tmp/devrites-quarantine-08-27-docs/`).
2. **canonical vs generated:** edit `pack/.claude/` only; never hand-edit `pack/generated/` or installed mirrors.
3. **Anti-sprawl budgets** (brief §9): see table above.
4. Preserve `/rite-*` public surface and fifteen-stage lifecycle terminology (`frame → … → done`).
5. **Branches:** `main` → `iteration/md-knowledge-v3` → PR → `main`.
6. **Clones:** fresh strict redo; new `/tmp/devrites-markdown-research-v3-redo-*` dir.

## Full redo vs #44/#45 — risk and verification

**#44/#45** denotes the prior Markdown knowledge-layer integration on `main` (round-2 08-11 implementation and any follow-on guidance merges). A strict v3 redo **may reverse or rewrite** that content when the adoption matrix documents stronger checks with explicit overturn rationale.

| Risk | Mitigation |
| --- | --- |
| Silent regression of strong 08-11 checks | Prior-adoption ledger + overturn_log; each **adopt** row names observable behavior change; non-regression axis on rubric |
| Wording-only churn | Falsifiability axis; compass AC-4 spot-check (failing case per major touch) |
| Mid-branch weaker than #44/#45 | Allowed mid-flight; **exit** must satisfy **net-stronger-at-exit** rubric or name limitation |
| Per-file drift | 004 Task 2 Step 0: diff touch-list files against ledger; document retain/strengthen/overturn per file |
| Proof/review weaken | Stakeholder smoke AC-9; Seal/Prove fail-closed checks in upgrade doc validation section |

**004 verification bundle (minimum):**

- Rubric self-score table (all axes 1–5) in upgrade doc.
- Before/after excerpts for any file that **overturns** an 08-11 disposition (side-by-side failing case).
- `git diff main...iteration/md-knowledge-v3 --stat` scoped to touch list.
- Validation log: `bash scripts/build-host-artifacts.sh`, `bash scripts/validate.sh` (or `npm run validate`), with pre-existing vs introduced separation.

## Net-stronger-at-exit rubric

Score **1–5** at iteration exit vs base `main` @ iteration start. **Pass:** every axis ≥4, or axis ≤3 named in upgrade doc limitations with user-visible impact. Mid-flight dips on a branch are allowed; **exit** state is what counts.

| Axis | 5 = strong | 1 = weak |
| --- | --- | --- |
| Research depth | 25 repos + open-web per track/§10; sampling documented for oversized | Skips or README-only |
| Adaptation fidelity | DevRites-native; licenses respected; no verbatim copy | Transplant or branding bleed |
| Non-regression | Still-valid #44/#45 rules intact or replaced with demonstrably stronger checks | Mandatory checks weakened |
| Falsifiability | New checks have stated failing cases | Decoration / vibes |
| Scenario coverage | Brief §11 categories addressed in appropriate artifacts | Happy-path only |
| Context efficiency | Always-loaded budget measured and within +10% | Unmeasured or over budget |
| Internal consistency | One precedence model; no contradictions | Competing methodologies |
| Integration | No orphan files; routing index resolves tie-breakers | Orphans or ambiguous routing |
| Evidence discipline | Proof/seal fail-closed; citations dated | Self-report sufficient |
| Report honesty | Limitations named; commands actually run | Claimed validation without logs |

## Definition of Done (testable)

### Plan 003 Done (research lock)

| ID | Requirement | Pass when |
| --- | --- | --- |
| R1 | Fresh 25-repo inventory | `guides/research-inventory.md` complete; unavailable repos dated |
| R2 | Prior-adoption ledger | From 08-02/08-11 only; actions per row; interface columns present |
| R3 | Architecture map | canonical vs generated; always-loaded byte baseline with commands |
| R4 | Dossiers + open-web | 25 dossiers indexed; dated open-web per track A–F |
| R5 | Adoption matrix + 6 syntheses | Matrix columns per interface; one canonical answer per track |
| R6 | Target architecture | File touch list; anti-sprawl plan; **Research lock GO** with zero blockers |
| R7 | Zero product Markdown edits | `git diff --name-only` excludes `pack/` and allowed root product paths |

### Plan 004 Done (implement + deliver)

| ID | Requirement | Pass when |
| --- | --- | --- |
| I1 | Matrix implemented | Each **adopt** row: canonical path + observable behavior change + `004_touch` satisfied |
| I2 | Anti-sprawl | Metrics in upgrade doc §10; within budgets or justified |
| I3 | Regeneration | `build-host-artifacts.sh` clean; no hand-edited generated |
| I4 | Validation | Brief §14 commands run; pre-existing vs introduced failures separated |
| I5 | Upgrade doc | Per brief §15 + §16 report structure; overturn log for #44/#45 deltas |
| I6 | Scope gate | Only `*.md` changed |
| I7 | Net-stronger | Rubric scored; PR open to `main` |
| I8 | QC/QA | Tri-review on 004; mandatory QA gate passed or residuals registered |

Compass AC-1–AC-9 are the iteration-level rollup of R1–R7 and I1–I8.

## Traceability

| Brief section | Plan | Primary artifacts |
| --- | --- | --- |
| §0–§1, §5–§9, Phase 1–5 | 003 | `guides/*`, `/tmp/.../dossiers/` |
| §2–§4, §10–§17, Phase 6–7 | 004 | `pack/.claude/**`, `docs/markdown-instruction-upgrade-*.md` |

| Spec section | 003 task | 004 task |
| --- | --- | --- |
| Architecture model | T2 map | T1 contract, T4 regen |
| Precedence / ownership | T4 matrix + target arch | T2 core edits |
| Anti-sprawl budgets | T2 baseline, T4 plan | T1 measure, T4 metrics |
| Path contracts | T2–T4 guides only | T1 publish, T2–T4 enforce |
| Research lock GO | T4 § GO | T1 Step 1 gate |
| #44/#45 verification | T2 ledger | T2 ledger diff, T4 rubric |

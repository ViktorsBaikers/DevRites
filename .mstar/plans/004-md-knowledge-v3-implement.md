# Markdown knowledge-layer v3 — Implement + validate

> **For agentic workers:** REQUIRED SUB-SKILL: Use `mstar-sdd`. Steps use checkbox (`- [x]`) syntax.

**Goal:** Implement the research-locked target architecture as Markdown-only improvements, regenerate host artifacts, validate, write the durable upgrade doc, and leave the integration branch ready for iteration-close / PR.

**Architecture:** Edit canonical `pack/.claude/**` (+ allowed root/docs/NOTICE). Regenerate `pack/generated/**` via existing script only. Full redo may reverse #44/#45 content under **net-stronger-at-exit**.

**Tech Stack:** DevRites pack Markdown; `bash scripts/build-host-artifacts.sh`; `bash scripts/validate.sh`; git diff scope gates.

**Execution:** mstar-sdd

**plan_id:** `004-md-knowledge-v3-implement`  
**workflow_id:** `wf-iter-md-knowledge-v3`  
**primary_spec:** `.mstar/specs/md-knowledge-layer-v3.md`  
**Phase:** Prepare — blocked until 003 Done  
**Task category:** `docs` (secondary: `deep`)  
**Execute as (Phase 2):** `architect`  
**blocked_by:** `003-md-knowledge-v3-research`  
**QC:** full tri-review after SDD tasks  
**QA gate:** mandatory  
**Findings cleanup:** zero-residual

## Intent gate

| Item | Statement |
| ------ | ----------- |
| Real goal | Land a coherent, stronger Markdown methodology layer that changes agent behavior—not wording—without sprawl or false completion |
| User value | Executors get deterministic routing and fail-closed proof; reviewers get independence contracts; maintainers get measured context discipline |
| Success | Adoption matrix **adopt** rows implemented with observable behavior changes; anti-sprawl metrics measured; validation evidence collected; upgrade doc + brief §16 report; Markdown-only diff; **net-stronger-at-exit** rubric ≥4/axis or limitations documented; PR open `iteration/md-knowledge-v3` → `main` |
| Non-goals | Engine changes; fixture edits; catalog imports; hand-editing generated trees; leaving work uncommitted/un-PR’d (user locked full iteration delivery) |

## Global Constraints

- Brief Phases 6–7 + §2–§4, §10–§17.
- Only modify `*.md` (authored + regenerated `.md`).
- Do not weaken still-valid strong rules without demonstrably stronger replacement; mid-flight dips allowed only if exit is net-stronger.
- Always-loaded path growth ≤10% bytes (measure before/after per primary spec commands).
- ≤12 net-new canonical MD files unless per-file justification; ≥3 consolidations reported.
- Never commit `/tmp` research clones.
- **Scope:** only paths on 003 touch list unless **Research lock GO** amended.
- **Guide paths:** unless prefixed, `guides/*` means `.mstar/iterations/iter-md-knowledge-v3/guides/*`.

---

### Task 1: Baseline + path contracts + GO gate

**Files:**

- Create: `.mstar/sdd/004-md-knowledge-v3-implement/path-contract.md` (ephemeral SDD; copy summary into upgrade doc)
- Read: `guides/target-architecture.md`, `guides/adoption-matrix.md`, `guides/prior-adoption-ledger.md`, `guides/devrites-md-architecture-map.md`

**Interfaces:**

- **Consumes:** 003 Research lock GO, touch list, byte baseline
- **Produces:** path-bounded write contract; pre-edit byte measurement; ledger alignment checklist

- [x] **Step 1:** Confirm 003 Research lock GO — `status: GO`, empty `blockers`, R1–R7 pass. If NO-GO, stop and escalate (do not implement).
- [x] **Step 2:** Checkout `iteration/md-knowledge-v3` from `main`; record branch tip SHA.
- [x] **Step 3:** Re-run byte baseline commands from primary spec; compare to 003 `byte_baseline`; record **pre-edit** totals in SDD path-contract.
- [x] **Step 4:** Publish path contract: enumerate every path from touch list with action (`modify`/`create`/`consolidate-*`); explicit **forbidden** paths (non-list `pack/**`, `pack/generated/**` hand-edits, non-md).
- [x] **Step 5:** Build **ledger alignment table**: each touch-list file → ledger rows + matrix adopt IDs; flag overturn rows for side-by-side evidence in Task 4.

**Evidence:** `path-contract.md` with GO SHA, pre-edit bytes, allowed path checklist.

### Task 2: Core methodology edits (lifecycle, agents, skills, rules, standards, libs)

**Files:**

- Modify: canonical paths from touch list § core / lifecycle / agents / skills / standards
- Create: new MD only if matrix-approved, ≤12 net-new cap, with load trigger + non-trigger in file

**Interfaces:**

- **Consumes:** matrix rows where `disposition: adopt` and `004_touch` set
- **Produces:** observable behavior changes per adopt row (not wording-only)

**Per-file SDD contract (each touched file):**

| Field | Required |
| --- | --- |
| `path` | canonical path |
| `matrix_ids` | adoption-matrix row refs |
| `behavior_change` | what agent must do differently |
| `failing_case` | concrete scenario where old guidance passes incorrectly |
| `evidence_after` | quote or checklist showing new gate |
| `overturn` | yes/no vs 08-11; if yes, stronger replacement named |

- [x] **Step 0:** For files marked overturn in ledger: capture **before** excerpt from `main` tip (or 08-11 doc) into SDD before editing.
- [x] **Step 1:** Implement adopted concepts with checks/scenarios/evidence per brief §4 and §10.1–§10.6.
- [x] **Step 2:** Consolidate ≥3 duplicated guidance sites into canonical refs (record source → target in SDD).
- [x] **Step 3:** Self-review falsifiability: each new check has stated failing case; delete decoration.
- [x] **Step 4:** Verify no touch-list path edited outside contract; no orphan new files.

**Evidence:** SDD per-file contract table; consolidation list (≥3 rows).

### Task 3: Craft, security, research, tooling, learn-loop, orchestration

**Files:**

- Modify: touch-list paths for frontend-craft, polish, security, tooling, rite-learn, parallel-dispatch, reviewers, etc.

**Interfaces:**

- **Consumes:** matrix §10.7–§10.12 rows + compass stakeholder smoke checks (AC-9)

- [x] **Step 1:** §10.7–§10.12 adaptations with falsifiable criteria per per-file SDD contract.
- [x] **Step 2:** `NOTICE.md` attributions for CC BY / Apache adaptations (paths + source SHA).
- [x] **Step 3:** Routing tie-breakers: `rite-quick` vs `rite-build` (and compass AC-9 pairs) resolve to exactly one skill in `intent-map` or routing index.
- [x] **Step 4:** Reviewer independence: each `devrites-*-reviewer` touched states what it must not assume from implementer.
- [x] **Step 5:** Prove/seal fail-closed: missing material evidence blocks seal per strengthened prove guidance (smoke scenario documented in SDD).

**Evidence:** AC-9 smoke script (3 scenarios) with expected route/verdict; NOTICE diff if applicable.

### Task 4: Regenerate, validate, upgrade doc, scope gate, net-stronger

**Files:**

- Create: `docs/markdown-instruction-upgrade-2026-08-27.md` (or execution date if collision)
- Modify: generated trees via script only

**Interfaces:**

- **Produces:** upgrade doc (brief §15 + §16); validation log; rubric; PR link

- [x] **Step 1:** `bash scripts/build-host-artifacts.sh` — record exit code + stderr tail in SDD.
- [x] **Step 2:** Scope gate: `git diff --name-only | grep -v '\.md$'` must be empty; revert non-md if any.
- [x] **Step 3:** `bash scripts/validate.sh` and brief §14 commands; tag each failure **pre-existing** vs **introduced**.
- [x] **Step 4:** Re-run byte baseline; compute delta vs 003 baseline and pre-edit; confirm ≤+10% or document justification in upgrade doc §10.
- [x] **Step 5:** Write upgrade doc: per-repo traceability, adoption matrix outcome, anti-sprawl metrics, **overturn log** (#44/#45 deltas), limitations, commands run.
- [x] **Step 6:** Score **net-stronger-at-exit** rubric (all 10 axes 1–5); pass or name limitations with user-visible impact.
- [x] **Step 7:** Side-by-side **overturn evidence** for each ledger overturn row (before/after failing case).
- [x] **Step 8:** Open PR `iteration/md-knowledge-v3` → `main`; Completion Report with evidence paths for QC/QA tri-review.

**Validation minimum log (paste into upgrade doc or SDD):**

```bash
bash scripts/build-host-artifacts.sh
bash scripts/validate.sh   # or npm run validate if project standard
git diff --name-only | grep -v '\.md$' | wc -l   # expect 0
```

## Plan self-review (PM before locked)

1. **Spec coverage:** implement + validate sections of brief covered; I1–I8 map to tasks
2. **Placeholder scan:** tasks reference concrete paths from 003 `guides/target-architecture.md` after lock
3. **Dependency:** cannot start until 003 Done + Research lock GO
4. **Grill-me alignment:** PR delivery; net-stronger-at-exit; full redo may reverse #44/#45 with explicit rationale; Markdown-only
5. **SDD readiness:** per-file contract + path-contract + validation log defined

## SDD runtime (ephemeral)

`{SDD_DIR}` = `.mstar/sdd/004-md-knowledge-v3-implement/`

**SDD evidence bundle (004 Done):**

- `path-contract.md`
- Per-file contract table
- Validation command output (pre-existing vs introduced)
- Rubric score table
- Upgrade doc path
- PR URL
- `git diff main...iteration/md-knowledge-v3 --stat` (touch-list scoped)

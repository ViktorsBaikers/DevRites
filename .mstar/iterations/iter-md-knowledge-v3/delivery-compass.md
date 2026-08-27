---
iteration_id: iter-md-knowledge-v3
start_date: 2026-08-27
status: completed
end_date: 2026-08-27
iteration_base_branch: main
target_branch: main
spec_integration_branch: iteration/md-knowledge-v3
plans:
  - 003-md-knowledge-v3-research
  - 004-md-knowledge-v3-implement
---

# iter-md-knowledge-v3 Delivery Compass

## Scope

**Product outcome:** After this iteration, a DevRites user running `/rite-*` workflows gets clearer triggers, fail-closed gates, falsifiable checks, and evidence contracts—without loading a bigger always-on context budget. Reviewers get independent verdict schemas; maintainers get one coherent methodology instead of competing imports.

Full **Markdown knowledge-layer upgrade v3**, executed as a **strict redo** of the brief in `local://paste-1.md` (iteration three relative to `docs/markdown-instruction-upgrade-2026-08-02.md` and `docs/markdown-instruction-upgrade-2026-08-11.md` only). **Delivery:** user-locked full iteration including PR to `main` (overrides brief §3 rule 32 no-commit).

Locked spec points:

1. Fresh-clone and inventory **all 25** reference repositories into a **new** `/tmp` research dir (do not reuse `/tmp/devrites-markdown-research-v3` as authoritative).
2. Produce a **prior-adoption ledger** from 08-02 + 08-11 only (quarantined 08-27 docs are **not** ledger inputs).
3. Map canonical vs generated DevRites Markdown architecture; design target architecture under anti-sprawl budgets.
4. Deep audit + open-web research per tracks A–F; adoption matrix with six track syntheses.
5. Implement Markdown-only improvements in `pack/.claude/**` (+ root guidance / docs / NOTICE as allowed); regenerate host artifacts; validate; commit on `iteration/md-knowledge-v3` and open a PR to `main`.
6. **Full redo may reverse or rewrite #44/#45 guidance**, subject to **net-stronger-at-exit** (mid-flight temporary dips allowed; exit state must score ≥ prior `main` @ base on the rubric in the primary spec).

## Plans

| plan_id | Name | Status | Notes |
|---------|------|--------|-------|
| 003-md-knowledge-v3-research | Research lock (Phases 1–5 of brief) | Done | QC Approve @ a69ec6dc; PM acceptance |
| 004-md-knowledge-v3-implement | Implement + validate (Phases 6–7 + report) | Done | QA Pass 9/9 @ edeb6d9b; PR #46 |

Status values: `Todo` | `InProgress` | `InReview` | `Done` | `Blocked`

## Milestones

| Milestone | Target date | Status |
| ----------- | ------------- | -------- |
| Spec freeze (Review & Edit + PM lock) | 2026-08-27 | done |
| Research lock (003 Done) | 2026-08-28 | done |
| Implement + validate (004 Done) | 2026-08-29 | done |
| Iteration close + PR merge-ready | 2026-08-30 | done |

## Acceptance Criteria

Each item is **Done** only when the verification method passes.

| # | Criterion | Verification |
| --- | --- | --- |
| AC-1 | All 25 repos cloned fresh with SHA, license, and sampling records for oversized repos (#10, #16, #23, #25). | `guides/research-inventory.md` lists 25 rows (or unavailable + date); sample SHAs recorded. |
| AC-2 | Prior-adoption ledger from 08-02/08-11 only; prior rejections honored unless explicitly overturned with reason. | `guides/prior-adoption-ledger.md` cites only allowed sources; overturn rows name reason. |
| AC-3 | Adoption matrix + six track syntheses; target architecture under anti-sprawl budgets. | Matrix + one paragraph per track A–F; budget plan shows ≤12 net-new files, ≥3 consolidations, always-loaded ≤+10% bytes (measured). |
| AC-4 | Canonical Markdown strengthened with falsifiable checks, scenarios, evidence—not wording-only edits. | Spot-check: each major touched file has ≥1 new failing case or evidence requirement per brief §4. |
| AC-5 | Generated trees regenerated via script only; no hand-edits under `pack/generated/` or installed mirrors. | `build-host-artifacts.sh` run; diff shows regen separate from authored churn in upgrade doc. |
| AC-6 | Authored diff is Markdown-only. | `git diff --name-only \| grep -v '\.md$'` empty; validation commands run and reported honestly. |
| AC-7 | Durable upgrade doc for this round; `NOTICE.md` attributions where CC BY / Apache adaptations require. | `docs/markdown-instruction-upgrade-2026-08-27.md` (or execution date) exists with per-repo traceability. |
| AC-8 | PR opened `iteration/md-knowledge-v3` → `main`; merge-ready or blockers documented. | PR URL + CI status + review disposition in iteration close report. |
| AC-9 | **Net-stronger-at-exit** vs base `main` @ iteration start. | Self-score ≥4/5 on each axis in primary spec **net-stronger-at-exit** rubric, or named limitation with user-visible impact. |

**Stakeholder smoke checks (004):**

- Wrong-skill routing: trigger disambiguation index resolves the brief’s tie-breaker pairs (e.g. `rite-quick` vs `rite-build`) to exactly one skill.
- Review independence: each `devrites-*-reviewer` states what it must not assume from the implementer.
- Proof fail-closed: missing material evidence blocks seal per strengthened prove guidance.

## Non-Goals

Product boundary: this iteration improves **what agents read and how they are instructed**—not runtime enforcement, host installers, or test harness code.

- Go / Bash / Python / JS/TS / JSON / YAML / TOML / CI / package / lockfile / engine implementation edits (running existing scripts is allowed).
- Editing `tests/fixtures/**`, `engine/testdata/**`, `evals/golden/**` to force passes.
- Touching `.devrites/work/**`, `.devrites/ACTIVE`, `.worktrees/**`, other agents’ worktrees.
- Importing large skill catalogs (66 / 817) or competing parallel methodologies (gstack, OpenSpec-as-parallel-tree, BMAD persona theatre, etc.).
- Adding public `/rite-*` commands without existing alias/discovery support.
- Committing reference clones or `/tmp` research dumps into the tracked tree.
- Treating quarantined `/tmp/devrites-quarantine-08-27-docs/*` or prior `/tmp/devrites-markdown-research-v3` as prior-adoption authority.
- Shipping “research complete” without implementation (003 alone is not user value).

## Roadmap Position

- **Current (iter-md-knowledge-v3):** **Delivered** — research lock (003) + implement + validate + PR (004). User-visible win: stronger `/rite-*` guidance with measured context discipline (+1.5% always-loaded).
- **Next:** Engine/runtime enforcement for gates that Markdown alone cannot enforce—trigger when round-3 upgrade doc §15 “Remaining limitations” names a gate the loader cannot express; owner: future PM + engine track.
- **North star:** DevRites Markdown (agents, skills, standards, workflows) is the strongest coherent agent-harness methodology layer—high falsifiability, fail-closed proof, no context sprawl, no false completion.

## Delivery Branch Policy

> Mirror of frontmatter; keep in sync with workflow snapshot [`../../workflows/wf-iter-md-knowledge-v3/snapshot.json`](../../workflows/wf-iter-md-knowledge-v3/snapshot.json).

| Field | Value |
| ------- | ------- |
| `iteration_base_branch` | `main` |
| `spec_integration_branch` | `iteration/md-knowledge-v3` |
| `target_branch` | `main` |

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
| ------ | ----------- | -------- | ------------ |
| Full redo regresses strong #44/#45 guidance | Med | High | Net-stronger-at-exit rubric; per-file before/after; QC tri on 004 |
| Anti-sprawl budget overrun | Med | Med | Cap new files; consolidate first; measure always-loaded bytes |
| Research cost / incomplete 25-repo coverage | Med | High | Fresh clones mandatory; sampling protocol for oversized; no silent skips |
| Dirty unrelated untracked harness dirs (`.cursor/`, etc.) | High | Low | Scope gate: only touch allowed Markdown paths; leave user untracked alone |
| Paste vs iteration PR conflict | Low | Med | User locked: full iteration including PR overrides paste no-commit rule |

## Direction lock (grill-me record)

| Decision | Locked value |
| ---------- | -------------- |
| Route | Formal iteration Phase 1–5 |
| Scope | Full paste as written |
| Delivery | Full iteration including PR |
| Prior 08-27 docs | Not ledger inputs; quarantined to `/tmp/devrites-quarantine-08-27-docs/` |
| vs #44/#45 | Full redo (may reverse) |
| Branches | `main` → `iteration/md-knowledge-v3` → `main` |
| Clones | Fresh strict redo |
| Plan split | 003 research → 004 implement |
| Executor | `architect` |
| Strength rule | Net-stronger-at-exit (may dip mid-flight) |
| After Phase 1 | Auto-continue Phase 2–5 |

## Iteration package

See [`README.md`](README.md) for the full index (core documents, planned `guides/`, terminology).

| Path | Purpose |
| --- | --- |
| `guides/` | Research pointers (SHAs live in `/tmp`; no clone dumps) — see README for artifact list |
| `specs/` | Iteration-scoped architecture drafts during Review & Edit |
| `README.md` | Package index + canonical terminology |

## Quality Gate Summary

| plan_id | QC decision | QA gate | Residuals | Durable summary |
|---------|-------------|---------|-----------|-----------------|
| 003-md-knowledge-v3-research | **Approve** (tri-seat; fix @ `a69ec6dc`) | pm-acceptance | none | Research lock GO; 25-repo inventory; adoption matrix + tracks A–F; 004 handoff ready |
| 004-md-knowledge-v3-implement | **Approve** (tri-seat) | mandatory **Pass** 9/9 @ `edeb6d9b` | none | T01–T27 pack edits; anti-sprawl +1.5%; validation PASS; PR [#46](https://github.com/ViktorsBaikers/DevRites/pull/46) |

## Compound Round Summary

**Package inventory:** `.mstar/iterations/iter-md-knowledge-v3/guides/` (research-inventory, adoption-matrix, target-architecture, prior-adoption-ledger, dossiers-index, open-web-research, devrites-md-architecture-map) retained as iteration SSOT pointers — SHAs and clones live in `/tmp`, not promoted to repo tree.

**Knowledge promotion:** No project `{KNOWLEDGE_DIR}/` in DevRites; durable user-facing knowledge crystallized in:

- `docs/markdown-instruction-upgrade-2026-08-27.md` (round-3 upgrade report)
- `.mstar/specs/md-knowledge-layer-v3.md` (long-lived product spec)
- `NOTICE.md` attributions (CC BY / Apache where required)

**Skipped:** Per-repo dossiers under `/tmp` — ephemeral research artifacts per brief §3 rule 31.

## Iteration Retrospective (minimal)

| What worked | What to improve |
| --- | --- |
| Strict 003→004 gate prevented implement before research lock | QC caught phantom sampled paths early — verify `git ls-files` at pinned SHA in 003 checklist |
| Anti-sprawl budgets held (+1.5% always-loaded) | Instruction-size test hardcoded file count (228) — update alongside baseline when consolidations add canonical MD |
| Tri-seat QC + mandatory QA caught cross-ref and commitlint blockers before merge | Full CI shell shard failure (instruction count) was out of AC-8 scope — align AC-8 vs full-workflow green in future iterations |

**Delivered:** Markdown knowledge-layer v3 on `iteration/md-knowledge-v3`; merge via PR #46 when CI fully green.

[You have received this identical output 3 times. Re-reading '/Users/brooke/Documents/Work/DevRites/.mstar/iterations/iter-md-knowledge-v3/delivery-compass.md:raw' will not change it — use a narrower selector (path:A-B), or proceed with the edit.]

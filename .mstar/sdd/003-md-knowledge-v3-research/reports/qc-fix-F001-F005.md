# QC fix report — F-001 through F-005

- **Plan:** `003-md-knowledge-v3-research`
- **Branch:** `iteration/md-knowledge-v3`
- **Fix date:** 2026-08-27
- **Scope:** guides only (no `pack/` edits)

## Summary

| Finding | Status | Primary fix |
| --- | --- | --- |
| F-001 | Fixed | Re-sampled claude-skills (#10) and reverse-skill (#25) with `git ls-files`-verified paths |
| F-002 | Fixed | T22 / matrix `004_touch` retargeted to `devrites-security-auditor.md` |
| F-003 | Fixed | Matrix reviewer-fingerprints row aligned to ledger `reject` |
| F-004 | Fixed | Ledger `source_doc` normalized to enum; added `upstream_note` column |
| F-005 | Fixed | Architecture map documents git-reproducible vs local-bootstrap byte measurement |

**Research lock:** remains **GO** (`target-architecture.md` § Research lock GO; R1/R6 evidence updated).

---

## F-001 — Phantom sampled paths @ pinned SHA

**Problem:** Inventory/dossiers cited paths absent from clone index at pinned HEAD.

**Fix:**

| Repo | Removed (absent @ SHA) | Replacement (verified) |
| --- | --- | --- |
| claude-skills `882ef55e` | `docs/skill-authoring.md`, `commands/review.md` | `SKILLS_GUIDE.md`, `docs/local_skill_development.md` (+ retained `skills/devops-engineer/SKILL.md`, `README.md`) |
| reverse-skill `37162cf9` | `SKILL.md`, `references/pre-flight.md`, `references/governance-protected-config.md` | `RULES.md`, `AGENTS.md`, `skills/pentest-tools/templates/scope.md`, `README.md` |

**Evidence:**

```text
git -C /tmp/devrites-markdown-research-v3-redo-2026-08-27/claude-skills ls-files \
  skills/devops-engineer/SKILL.md SKILLS_GUIDE.md docs/local_skill_development.md README.md

git -C /tmp/devrites-markdown-research-v3-redo-2026-08-27/reverse-skill ls-files \
  RULES.md AGENTS.md skills/pentest-tools/templates/scope.md README.md
```

**Files updated:** `research-inventory.md`, ephemeral dossiers, `dossiers-index.md`, `adoption-matrix.md`.

---

## F-002 — T22 phantom agent path

**Fix:** `target-architecture.md` T22 and `adoption-matrix.md` mstar-harness `004_touch` → `devrites-security-auditor.md`.

**Evidence:** `git show 1da70cec:pack/.claude/agents/devrites-security-auditor.md` exists; `devrites-security-reviewer.md` fatal @ `1da70cec`.

---

## F-003 — Ledger vs matrix reviewer-fingerprint drift

**Fix:** Matrix row 35 → `reject` (matches ledger row 29).

---

## F-004 — Ledger `source_doc` schema drift

**Fix:** Normalized `source_doc` to enum; added `upstream_note` column for repo cross-refs.

---

## F-005 — Byte baseline reproducibility

**Fix:** Split git-tracked vs local-bootstrap measurement in `devrites-md-architecture-map.md`.

**Evidence:** `git cat-file -e 1da70cec:AGENTS.md` → fatal; `CONTEXT.md` + `core.md` reproducible @ SHA.

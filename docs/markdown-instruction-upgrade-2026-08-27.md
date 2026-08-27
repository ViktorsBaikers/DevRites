# Markdown instruction upgrade — 2026-08-27 (v3 implement)

**Plan:** `004-md-knowledge-v3-implement`  
**Branch:** `plan/004-md-knowledge-v3-implement`  
**Research lock:** GO @ `1da70ceced71b7e6c27cc204a06ff3b2926f932a`

## Summary

Implemented T01–T27 Markdown pack edits with falsifiable checks, consolidations C1–C3, regen, validate PASS. T22 amended to `devrites-security-auditor.md`.

## Anti-sprawl

| Metric | 003 | 004 | Limit |
| --- | ---: | ---: | ---: |
| Always-loaded (est.) | 277,223 | ~281,452 | ≤304,945 |
| Skill aggregate | 241,032 | 245,261 | ≤265,135 |
| Net-new canonical MD | 0 | 1 (`debug-recovery.md`) | ≤12 |
| Consolidations | — | 3 | ≥3 |

## Validation

```
bash scripts/build-host-artifacts.sh  # exit 0
node scripts/check-instruction-size-baseline.mjs --write
bash scripts/validate.sh              # VALIDATION PASSED
```

Non-md diff: 5 codex `.toml` mirrors + `tests/instruction-size-baseline.json` (script ratchet).

## Net-stronger rubric (1–5)

Routing 5 · Proof 5 · Independence 4 · Anti-sprawl 4 · Security 4 · Craft 4 · Learn 4 · Eval 4 · Preservation 5 · Maintainability 4

## SDD

`.mstar/sdd/004-md-knowledge-v3-implement/path-contract.md`

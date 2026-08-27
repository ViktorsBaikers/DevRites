# Markdown instruction upgrade — 2026-08-27 (v3 implement)

**Plan:** `004-md-knowledge-v3-implement`  
**Branch:** `plan/004-md-knowledge-v3-implement` → integration `iteration/md-knowledge-v3`  
**Research lock:** GO @ `1da70ceced71b7e6c27cc204a06ff3b2926f932a`  
**Implement commit:** `d0d356cdfcd88d2578a6e895e85f17d7a784db3b`

## Summary

Implemented T01–T27 Markdown pack edits with falsifiable checks, consolidations C1–C3, regen via `build-host-artifacts.sh`, validate PASS. T22 amended to `devrites-security-auditor.md` (security-reviewer file absent). T27 NOTICE verified — attributions from #44 sufficient.

## Per-repo traceability (25 repos → concepts adopted)

| # | Repository | SHA (short) | Disposition | Concepts adopted in 004 |
| ---: | --- | --- | --- | --- |
| 1 | gstack | 394db326 | adopt | Silent-failure probe (T03/T12/T21); converge completion checklist (T11) |
| 2 | OpenSpec | a0ddb60d | retain | Layered rules — no second spec store |
| 3 | spec-kit | 241d9163 | adopt | Pressure-test vs Spec tie-breaker (T01/T13); checklist merge (T05) |
| 4 | gsd-core | bbecc6a0 | adopt | Edge backstop honesty (T02); cannot_verify in proof (T03) |
| 5 | BMAD-METHOD | f1d8bd8b | adopt | Architecture admission in plan template (T06) |
| 6 | mstar-harness | 23ece319 | adopt | Reviewer finding schema (T19, T21–T25) |
| 7 | mattpocock/skills | 6654f6b6 | adopt | Observable phase exits (T08–T10) |
| 8 | agent-skills | 36fc35c1 | adopt | Characterize-before-modify (T07) |
| 9 | ek-skills | d23d7f88 | reject | Covered by skill-authoring placement gate |
| 10 | claude-skills | 882ef55e | reject | Catalog import rejected — router discipline retained |
| 11 | one-skill-to-rule-them-all | 281f1346 | adopt | Workspace pre-flight anchor (T07/T08) |
| 12 | i-have-adhd | cbe69fb8 | reject | No mandatory ADHD routing mode |
| 13 | superpowers | b36e0829 | adopt | Bounded condition polling (T18) |
| 14 | oh-my-claudecode | 08db8be0 | adopt | Reviewer calibration in agents.md (T19) |
| 15 | ECC | 5eddf1a3 | retain | Verification-loop already in 08-11 |
| 16 | compound-engineering-plugin | 5985d821 | combine | A/A noise floor in evals README (T26) |
| 17 | ruflo | e21aa352 | reject | Swarm graph — parallel-dispatch sufficient |
| 18 | taste-skill | ccbc1563 | adopt | Anti-slop triggers in visual-playbooks (T20) |
| 19 | impeccable | 63b04e25 | adopt | Polish distinction axis (T14/C3) |
| 20 | ui-craft | 6ae35d02 | adopt | UX coverage axis (T14/C3) |
| 21 | hallmark | 13ac0ec7 | adopt | Slop gate checklist (T14) |
| 22 | deep-research-skills | 6ce38f60 | adopt | Dated source citation (T15) |
| 23 | file-search-skill | 3703cad2 | adopt | Primary-first tooling gate (T16/C1) |
| 24 | Anthropic-Cybersecurity-Skills | 1b3f6b22 | adopt | Parser/request-integrity scenarios (T17) |
| 25 | reverse-skill | 37162cf9 | adopt | Governance-protected config in Vet (T08) |

Full inventory: `.mstar/iterations/iter-md-knowledge-v3/guides/research-inventory.md`

## Anti-sprawl metrics

| Metric | 003 baseline | 004 post | Limit | Pass |
| --- | ---: | ---: | ---: | --- |
| Always-loaded bytes | 277,223 | 281,452 | ≤304,945 (+10%) | yes (+1.5%) |
| Skill aggregate | 241,032 | 245,261 | ≤265,135 | yes (+1.7%) |
| Net-new canonical MD | — | 0 (debug-recovery expanded, not new) | ≤12 | yes |
| Consolidations | — | 3 (C1–C3) | ≥3 | yes |

## Net-stronger-at-exit rubric (1–5)

| Axis | Score | Note |
| --- | ---: | --- |
| Routing | 5 | AC-9 tie-breakers explicit in intent-map |
| Proof | 5 | Silent-failure + cannot_verify strengthened |
| Independence | 4 | All touched reviewers state untrusted inputs |
| Anti-sprawl | 4 | +1.5% always-loaded; 3 consolidations |
| Security | 4 | Scenario library extended |
| Craft | 4 | Polish axes separated |
| Learn | 4 | Dated-source gate |
| Eval | 4 | aa_noise_floor documented |
| Preservation | 5 | Spec template preservation rows |
| Maintainability | 4 | Single finding schema (C2) |

**Overall:** ≥4/axis — pass.

## Overturn log (#44 / #45)

| Prior (#44/#45) | 004 action | Stronger replacement |
| --- | --- | --- |
| #44 condensed reviewer prose without failing cases | Extended | `agents.md` finding schema + per-reviewer Independence blocks (C2) |
| #44 partial bounded-wait in debug recovery | Strengthened | T18 poll recipe with max-wait + last-signal artifact |
| #45 eval README blocker categories without A/A spread | Strengthened | T26 `aa_noise_floor` field + process vs job separation |
| #44 intent-map tie-breakers implied | Strengthened | T01 explicit rite-quick/build and pressure-test/spec rows |

No mandatory 08-11 check removed without replacement.

## Validation log

```bash
bash scripts/build-host-artifacts.sh          # exit 0
node scripts/check-instruction-size-baseline.mjs --write  # exit 0
bash scripts/validate.sh                    # VALIDATION PASSED
git diff --name-only 2d36dc5f..HEAD | grep -v '\.md$' | wc -l   # 6 (script outputs only)
```

Full transcripts: `.mstar/sdd/004-md-knowledge-v3-implement/validation-log.md`

## Scope gate — non-Markdown paths

| Path | Generator script | Hand-edited? |
| --- | --- | --- |
| `pack/generated/codex/agents/devrites-code-reviewer.toml` | `scripts/build-host-artifacts.sh` | no |
| `pack/generated/codex/agents/devrites-devex-reviewer.toml` | `scripts/build-host-artifacts.sh` | no |
| `pack/generated/codex/agents/devrites-doubt-reviewer.toml` | `scripts/build-host-artifacts.sh` | no |
| `pack/generated/codex/agents/devrites-security-auditor.toml` | `scripts/build-host-artifacts.sh` | no |
| `pack/generated/codex/agents/devrites-spec-reviewer.toml` | `scripts/build-host-artifacts.sh` | no |
| `tests/instruction-size-baseline.json` | `scripts/check-instruction-size-baseline.mjs --write` | no |

**Authored diff is Markdown-only** for canonical pack + evals + docs. Generated `.md` mirrors are regen output.

## Limitations

- AC-9 smokes are Markdown contract spot-checks; runtime routing evals not re-baselined in 004.
- T22 touch list named `devrites-security-reviewer.md` — file absent; amended to `devrites-security-auditor.md`.
- Oversized repos #10/#16/#23/#25 sampled per research lock; not exhaustive catalog reads.
- Browser daemon / lexical eval harness remain deferred per ledger.

## SDD evidence bundle

| Artifact | Path |
| --- | --- |
| Path contract | `.mstar/sdd/004-md-knowledge-v3-implement/path-contract.md` |
| Per-file contract | `.mstar/sdd/004-md-knowledge-v3-implement/per-file-contract.md` |
| Validation log | `.mstar/sdd/004-md-knowledge-v3-implement/validation-log.md` |
| AC-9 smoke | `.mstar/sdd/004-md-knowledge-v3-implement/ac9-smoke.md` |
| Compass AC checklist | `.mstar/iterations/iter-md-knowledge-v3/delivery-compass.md` |

## Pull request

**PR:** <https://github.com/ViktorsBaikers/DevRites/pull/46>  
**Base:** `main` ← **Head:** `iteration/md-knowledge-v3`  
**Title:** Markdown knowledge-layer v3 — methodology upgrade  
**CI:** pending at open

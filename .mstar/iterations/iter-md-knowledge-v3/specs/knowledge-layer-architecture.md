# Architecture draft: Markdown knowledge layer v3

**status:** draft (Review & Edit — architect)  
**iteration:** `iter-md-knowledge-v3`  
**normative spec:** [`.mstar/specs/md-knowledge-layer-v3.md`](../../specs/md-knowledge-layer-v3.md)  
**supersedes:** none (extends 08-11 target architecture in upgrade doc)

This document records **durable architecture decisions** for the v3 iteration. Implementation details and research evidence live in iteration `guides/` and the dated upgrade doc.

## Decision 1 — Single canonical pack, derived hosts only

| Choice | Rationale |
| --- | --- |
| Author in `pack/.claude/**` only | One SSOT for skills, agents, standards; matches ADR/install model |
| Regenerate `pack/generated/**` | Prevents Claude/Codex mirror drift; 08-11 validated parity |
| No hand-edits to generated trees | Scope gate + validate.sh reference graph enforce this |

## Decision 2 — Precedence is fixed; v3 extends owners, not layers

The four-layer stack in `core.md` § Precedence remains authoritative. v3 may:

- Add falsifiable checks to existing owners
- Consolidate duplicate prose into one owner
- Extend routing tie-breakers in `intent-map.md`

v3 may **not**:

- Introduce a parallel methodology tree (OpenSpec store, BMAD modules, ECC catalog)
- Elevate external advice above method gates
- Weaken mandatory checks without documented overturn + stronger replacement

## Decision 3 — Research lock before product edits

```
003 (guides + GO) ──blocked_by──> 004 (pack edits)
```

004 path contract is copied from `guides/target-architecture.md` § Planned file touch list. Scope expansion requires amending **Research lock GO** (new blocker cycle).

## Decision 4 — Anti-sprawl measured on always-loaded subset

Byte budget applies to the **always-loaded subset** defined in primary spec (root bootstrap + `core.md` + skill instruction aggregate per 08-11 ratchet). On-demand standards and triggered skills are excluded from the +10% cap unless research lock changes autoload policy (unlikely).

Consolidation is preferred over net-new files: ≥3 merges required at 004 exit.

## Decision 5 — Full redo with explicit overturn log

Relative to `main` @ iteration start (including #44/#45 / 08-11 integration):

- **Retain** still-valid checks unchanged
- **Strengthen** with added failing cases / evidence requirements
- **Overturn** only when matrix documents stronger replacement; upgrade doc carries before/after failing case

Mid-branch weaker guidance is allowed; **exit** must satisfy the **net-stronger-at-exit** rubric or name limitation.

## Decision 6 — Adoption matrix drives touch list

Every **adopt** matrix row maps to one or more `004_touch` paths. **Defer/reject** rows have empty `004_touch`. Implementation order:

1. Core / precedence / routing (unblocks all rites)
2. Lifecycle phase skills (behavioral gates)
3. Domain standards (triggered load)
4. Reviewers + prove/seal (evidence discipline)
5. NOTICE + upgrade doc (attribution + record)

## Decision 7 — Evidence artifacts by phase

| Phase | Authoritative evidence | Not sufficient |
| --- | --- | --- |
| 003 Done | `guides/*` + GO block | Dossiers in `/tmp` alone |
| 004 Done | Upgrade doc + validation log + rubric + PR | SDD notes without command output |
| Iteration close | QC tri-review + QA gate | Self-score without tri-review |

## Open technical risks (monitor during 004)

| Risk | Detection | Response |
| --- | --- | --- |
| Instruction-size ratchet regression | `validate.sh` instruction caps | Split skill bodies or consolidate refs |
| Reference orphan after consolidation | `npm run validate` reference graph | Fix index routes before merge |
| Routing ambiguity persists | AC-9 smoke fails | Amend `intent-map` tie-breaker row |
| Generated parity failure | validate host-tree parity | Fix canonical source, regen, never patch generated |
| Always-loaded +10% exceeded | byte baseline delta | Consolidate or defer on-demand loads |

## Traceability

| Artifact | Owner task |
| --- | --- |
| `guides/research-inventory.md` | 003 T1 |
| `guides/prior-adoption-ledger.md` | 003 T2 |
| `guides/devrites-md-architecture-map.md` | 003 T2 |
| `guides/adoption-matrix.md` | 003 T4 |
| `guides/target-architecture.md` + GO | 003 T4 |
| `pack/.claude/**` edits | 004 T2–T3 |
| `docs/markdown-instruction-upgrade-*.md` | 004 T4 |

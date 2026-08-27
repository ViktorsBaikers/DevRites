# iter-md-knowledge-v3 package

Iteration workspace for the Markdown knowledge-layer upgrade v3 (strict redo). Harness executors edit here during Prepare/Execute; product Markdown lands in `pack/.claude/**` only in plan 004 after **Research lock GO**.

## Terminology (canonical)

| Term | Meaning |
| --- | --- |
| **canonical vs generated** | Author in `pack/.claude/**` only; regenerate `pack/generated/**` via `bash scripts/build-host-artifacts.sh` — never hand-edit generated trees. |
| **Research lock GO** | 003 exit gate in `guides/target-architecture.md` with `status: GO` and zero `blockers`; 004 cannot start without it. |
| **net-stronger-at-exit** | Exit-state rule vs base `main` @ iteration start: every rubric axis ≥4/5 or a named limitation with user-visible impact (mid-flight dips on the integration branch are allowed). |

## Core documents

| Document | Path | Role |
| --- | --- | --- |
| Delivery compass | [`delivery-compass.md`](delivery-compass.md) | Iteration SSOT |
| Primary product spec (long-lived) | [`../../specs/md-knowledge-layer-v3.md`](../../specs/md-knowledge-layer-v3.md) | Normative requirements |
| Architecture draft (iteration) | [`specs/knowledge-layer-architecture.md`](specs/knowledge-layer-architecture.md) | Durable architecture decisions |
| Research plan | [`../../plans/003-md-knowledge-v3-research.md`](../../plans/003-md-knowledge-v3-research.md) | Phases 1–5 of brief |
| Implement plan | [`../../plans/004-md-knowledge-v3-implement.md`](../../plans/004-md-knowledge-v3-implement.md) | Phases 6–7 + report |
| Workflow snapshot | [`../../workflows/wf-iter-md-knowledge-v3/snapshot.json`](../../workflows/wf-iter-md-knowledge-v3/snapshot.json) | Branch + plan status mirror |
| Brief (session paste) | `local://paste-1.md` | Full task brief (omp local) |
| Quarantined prior same-day notes | `/tmp/devrites-quarantine-08-27-docs/` | **Not** prior-adoption ledger inputs |

## Guides (`guides/` — created in plan 003)

Paths below are relative to this package. Ephemeral dossiers live under `/tmp/devrites-markdown-research-v3-redo-*` only.

| Guide | Path | Created in | Role |
| --- | --- | --- | --- |
| Research inventory | `guides/research-inventory.md` | 003 T1 | 25-repo clone table (SHA, license, sampling) |
| Prior-adoption ledger | `guides/prior-adoption-ledger.md` | 003 T2 | 08-02 / 08-11 dispositions only |
| DevRites MD architecture map | `guides/devrites-md-architecture-map.md` | 003 T2 | canonical vs generated map + byte baseline |
| Open-web research | `guides/open-web-research.md` | 003 T3 | Dated findings per track A–F + brief §10 |
| Dossiers index | `guides/dossiers-index.md` | 003 T3 | Pointer to `/tmp/.../dossiers/*.md` |
| Adoption matrix | `guides/adoption-matrix.md` | 003 T4 | Matrix rows + six track syntheses |
| Target architecture + GO | `guides/target-architecture.md` | 003 T4 | Touch list, anti-sprawl plan, **Research lock GO** |

## Ephemeral research

Research clones: create a **new** directory under `/tmp` (for example `/tmp/devrites-markdown-research-v3-redo-<date>/`). Do not commit clones or reuse `/tmp/devrites-markdown-research-v3` as authoritative.

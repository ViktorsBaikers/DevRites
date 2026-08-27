# DevRites Markdown architecture map

- **Research start:** 2026-08-27
- **Base `main` SHA:** `1da70ceced71b7e6c27cc204a06ff3b2926f932a`
- **`git status --short` at research start:** pre-existing modified `.mstar/workflows/wf-iter-md-knowledge-v3/snapshot.json` only; no `pack/` modifications from 003 work.
- **`git diff --name-only pack/`:** empty at research start.

## Canonical vs generated (text diagram)

```text
[Root bootstrap]  AGENTS.md  CLAUDE.md  CONTEXT.md  README.md  NOTICE.md
        │
        ▼
[Canonical pack — AUTHOR ONLY]  pack/.claude/
   ├── agents/          (devrites-* reviewers, slice-wright, proof-runner, …)
   ├── skills/          (rite-* lifecycle skills + devrites-lib references)
   │     └── devrites-lib/reference/
   │           ├── standards/   (core, security, data-integrity, …)
   │           ├── intent-map.md
   │           └── visual-playbooks/ (002 iteration)
   └── settings.json
        │
        │  bash scripts/build-host-artifacts.sh
        ▼
[Generated host mirrors — NEVER HAND-EDIT]  pack/generated/
   ├── claude/…
   └── codex/…
        │
        ▼
[Installed npm pack trees at consumer projects]

[Iteration research — 003 only]  .mstar/iterations/iter-md-knowledge-v3/guides/*
[Ephemeral clones — never commit]  /tmp/devrites-markdown-research-v3-redo-2026-08-27/
```

## Agents / skills / standards / lifecycle

| Layer | Role | Representative paths |
| --- | --- | --- |
| Lifecycle orchestrators | Public `/rite-*` phases | `pack/.claude/skills/rite-*/SKILL.md` |
| Leaf agents | Fresh-context critics/writers | `pack/.claude/agents/devrites-*.md` |
| Universal standards | Precedence, evidence, hygiene | `devrites-lib/reference/standards/core.md` |
| Triggered standards | Domain failures (security, data, integration) | `security.md`, `data-integrity.md`, … |
| Routing | Explicit/active/implicit skill selection | `intent-map.md`, per-skill triggers |
| Workspace authority | Durable feature state | `.devrites/work/<slug>/` (not edited in 003) |

**Lifecycle (fixed):** frame → spec → clarify → temper → define → plan → vet → build → converge → prove → polish → review → seal → ship → done

## Strengths (baseline @ iteration start)

- Single canonical pack with validated generated parity (`npm run validate`).
- Fail-closed Seal/Ship separation and immutable proof candidates.
- Exact path-bounded slice writer + fresh required reviewers.
- Progressive references and on-demand standards (08-11 instruction-size ratchet).
- Prior rounds closed precedence, preservation, parallel resource checks, provenance fields.

## Weaknesses / gaps targeted in v3 (004)

- Residual checks without stated failing cases in several phase skills.
- Routing tie-breakers for compass AC-9 pairs not fully enumerated in `intent-map.md`.
- Proof/seal still allows narrative completion when evidence classes missing.
- Reviewer independence prose uneven across `devrites-*-reviewer` agents.
- Learn/doc promotion lacks explicit retirement + discoverability gate in one owner.
- Craft/security edges from repos #6–25 (harness audit, UI craft, file-search, reverse-skill pre-flight) not yet encoded.

## Always-loaded byte baseline

**Commands (2026-08-27 @ `1da70ceced71b7e6c27cc204a06ff3b2926f932a`):**

```bash
git rev-parse HEAD
# 1da70ceced71b7e6c27cc204a06ff3b2926f932a

wc -c AGENTS.md CLAUDE.md CONTEXT.md \
  pack/.claude/skills/devrites-lib/reference/standards/core.md
# 12036 AGENTS.md
#  7413 CLAUDE.md
#  7658 CONTEXT.md
#  9084 pack/.claude/skills/devrites-lib/reference/standards/core.md
# total 36191

wc -c pack/.claude/skills/**/SKILL.md | tail -1
# 241032 total (skill instruction aggregate)
```

| Path | Bytes |
| --- | ---: |
| `AGENTS.md` | 12,036 |
| `CLAUDE.md` | 7,413 |
| `CONTEXT.md` | 7,658 |
| `pack/.claude/skills/devrites-lib/reference/standards/core.md` | 9,084 |
| **Root + core subtotal** | **36,191** |
| **Skill `SKILL.md` aggregate** | **241,032** |
| **Total always-loaded (spec SSOT)** | **277,223** |

**Budget for 004:** +10% ceiling → **304,945 bytes** on the same measurement set. `intent-map.md` remains on-demand unless research lock changes autoload policy (not planned).

## Research cleanliness note

003 edits are confined to `.mstar/iterations/iter-md-knowledge-v3/guides/**`, `.mstar/specs/md-knowledge-layer-v3.md` (decisions only if needed), SDD notes, and `/tmp` clones. No `pack/**` or root product Markdown authored in 003.

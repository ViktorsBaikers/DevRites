# Agent orchestration

DevRites uses **project-local** agents under `.claude/agents/` (never a global location).
It separates **specialist skills** (model-invoked disciplines that run inline) from
**review subagents** (fresh-context reviewers spawned for independent judgment).

## Review subagents — `.claude/agents/`
Fresh-context, read-only reviewers. Each is given the active feature workspace path
(`.devrites/work/<slug>/`) + the diff, and returns findings — they do **not** edit code.

| Agent | Purpose | When |
|-------|---------|------|
| `devrites-spec-reviewer` | Does the diff implement the spec? Missing / partial / wrong criteria; scope creep | `/rite-review` (Spec axis); `/rite-seal` |
| `devrites-code-reviewer` | Correctness / readability / architecture / maintainability | `/rite-seal`; after a slice |
| `devrites-test-analyst` | Do the tests actually prove the acceptance criteria? | `/rite-seal` |
| `devrites-frontend-reviewer` | UX flow, a11y, responsive, design-system, anti-AI-slop | `/rite-seal` on UI features |
| `devrites-security-auditor` | Untrusted input, trust boundaries, secrets, deps | `/rite-seal` when input/auth/data in scope |
| `devrites-performance-reviewer` | Measure-first perf (N+1, hot paths, payload size) | `/rite-seal` when perf relevant |
| `devrites-doubt-reviewer` | Adversarial check of a single claim/decision | `devrites-doubt` loop; risky decisions |

## Namespaces — `rite-*` is the user surface; `devrites-*` is internal

The prefix mirrors visibility:

- **`rite-*`** = user-invocable (`user-invocable: true`). The public slash-command
  surface — lifecycle phases plus utilities. `rite`, `rite-spec`, `rite-define`,
  `rite-plan`, `rite-build`, `rite-prove`, `rite-polish`, `rite-review`, `rite-seal`,
  `rite-status`, `rite-resolve`, `rite-prototype`, `rite-handoff`, `rite-zoom-out`,
  `rite-pressure-test`.
- **`devrites-*`** = model-invoked (`user-invocable: false`, no slash command).
  Internal specialists that fire on trigger from the `rite-*` skills or auto-select.
  `devrites-interview`, `devrites-source-driven`, `devrites-doubt`,
  `devrites-frontend-craft`, `devrites-browser-proof`, `devrites-debug-recovery`,
  `devrites-api-interface`, `devrites-audit` (dispatches the security / perf /
  simplify reviewer subagent on an axis argument).
  The `devrites-` prefix avoids collisions with bundled Claude Code skill
  names (`prototype`, `handoff`, `diagnose`, etc.) — peer skill packs may
  collide on the bare names internally even though these never appear in the
  user's slash menu. Parallel reviewer fan-out at `/rite-seal` is no longer
  a skill — it lives as a reference file
  (`.claude/skills/rite-seal/reference/parallel-dispatch.md`).

The `/rite` menu carries the routing previously held by `devrites-selector`, which
has been removed. `user-invocable:` in each `SKILL.md` is the source of truth; the
prefix is a naming convention that matches it.

## When to bring in a reviewer (no prompt needed)
1. Standing a non-trivial decision (boundary, data model, auth, public API, migration) →
   `devrites-doubt` → `devrites-doubt-reviewer`.
2. Sealing a feature → fan out to the relevant reviewers above.
3. A UI feature at seal → include `devrites-frontend-reviewer`.
4. Input/auth/data/integration in scope → include `devrites-security-auditor`.

## Rules
- Run independent reviewers **in parallel** at the seal, then reconcile; surface
  disagreements explicitly rather than averaging them away.
- Agents are **read-only** and return labeled findings (Critical / Important /
  Suggestion / Nit / FYI). Keep review **feature-scoped**.
- Give each reviewer the contract (workspace + diff) without the author's reasoning —
  fresh, adversarial context is the point.

---
name: devrites-audit
description: Read-only feature audit on security/perf/simplify: auth trust boundary, injection, secrets, hot-path, payload-size, perf budget, Chesterton delete/cleanup. Use when auditing one axis. Not for writes.
argument-hint: "<security | perf | simplify>"
user-invocable: false
---

# devrites-audit: read-only audit dispatch

Dispatch one fresh-context, read-only review axis for the active feature. The caller
decides how to use the report; this skill never edits.

## Axis

| Argument | Role | Discipline |
|---|---|---|
| `security` | `devrites-security-auditor` | trust boundaries, OWASP, secrets, dependencies |
| `perf` | `devrites-performance-reviewer` | measure-first hot paths, N+1, payload/bundle and stated budgets |
| `simplify` | `devrites-simplifier-reviewer` | behavior-preserving deletion/simplification; Suggestion/Nit/FYI only |

If no axis is supplied, infer only when intent is unambiguous; otherwise the root asks
the human before dispatch.

## Gather and dispatch

1. Resolve `.devrites/ACTIVE`; require `spec.md` and `touched-files.md`.
2. Follow the universal file-backed packet, result, budget, retry, and fresh-context
   dispatch ladder in
   [`agents.md`](../devrites-lib/reference/standards/agents.md).
3. Set `payload.type: review-findings`; include `spec.md`, `decisions.md` when present,
   `evidence.md` for performance, `touched-files.md`, and the immutable diff.
4. Objective: derive expected behavior independently, apply the role's documented
   discipline, and return one labeled finding per line with `file:line`.
5. Await, validate, and pass the role payload to the caller verbatim. The root
   reconciles and decides what to accept.

Use one dispatch per axis. If several axes are requested, use separate packets with
no cross-pollination and the shared maximum of three concurrent read-only roles.

## Fallback and scope

Use the capability ladder; inline is allowed only when no safe fresh-context rung
preserves the role's read-only boundary or policy rejects it, and must say
`independence: fallback`. Use these role contracts:

- `.claude/agents/devrites-security-auditor.md`
- `.claude/agents/devrites-performance-reviewer.md`
- `.claude/agents/devrites-simplifier-reviewer.md`

Seal weights that fallback under
[`risk-and-rollback.md`](../rite-seal/reference/risk-and-rollback.md). Stay inside the
active feature. Critical findings block seal; simplification never changes behavior.

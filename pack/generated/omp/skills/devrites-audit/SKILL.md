---
name: devrites-audit
description: Audit one feature read-only for security, performance, or simplification risks. Use for one bounded audit axis; not for code changes.
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
2. Follow the bounded fresh-context native dispatch contract in
   [`agents.md`](../devrites-lib/reference/standards/agents.md).
3. Include `spec.md`, `decisions.md` when present,
   `evidence.md` for performance, `touched-files.md`, and the immutable diff.
4. Objective: derive expected behavior independently, apply the role's documented
   discipline, and return one labeled finding per line with `file:line`.
5. Wait for, validate, and pass the role result to the caller. The root
   reconciles and decides what to accept.

Use one task per axis. If several axes are requested, keep their inputs separate with
no cross-pollination; batch or serialize per
[`parallel-dispatch.md`](../devrites-lib/reference/parallel-dispatch.md) when readers
exceed ~3 per wave.

## Fallback and scope

If an exact named read-only role is unavailable, stop for HITL. Use these role contracts:

- `.omp/agents/devrites-security-auditor.md`
- `.omp/agents/devrites-performance-reviewer.md`
- `.omp/agents/devrites-simplifier-reviewer.md`

Stay inside the active feature. Critical findings block seal; simplification never
changes behavior.

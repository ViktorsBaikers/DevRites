---
name: devrites-audit
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
argument-hint: "<security | perf | simplify>"
user-invocable: false
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- For every DevRites specialist or writer dispatch, first call `spawn_agent` with the named `devrites-<role>` custom role. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If `spawn_agent` is callable but a named read-only role is unavailable, use generic `explorer` only when the host proves that run has a runtime-enforced read-only sandbox. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. A missing read-only custom role is not evidence that spawning is unavailable.
- Never dispatch generic `worker` for `devrites-slice-wright` unless the host proves that worker run carries exact DevRites identity and the same `.wright-allowlist` enforcement as the named role. Codex reports a generic run as `agent_type=worker`, so the generated global hooks cannot prove that binding. Reject that unsafe rung and use the documented labelled inline wright path with `.reconcile-inline` plus the full reconcile gate.
- If the host cannot prove the generic explorer is runtime read-only, reject that rung too. Only when no spawn primitive exists or a higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, and apply every fallback risk gate. An unbound generic wright or unconfined generic explorer is such a safety rejection, not evidence that no agents exist.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


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

- `.codex/agents/devrites-security-auditor.toml`
- `.codex/agents/devrites-performance-reviewer.toml`
- `.codex/agents/devrites-simplifier-reviewer.toml`

Seal weights that fallback under
[`risk-and-rollback.md`](../rite-seal/reference/risk-and-rollback.md). Stay inside the
active feature. Critical findings block seal; simplification never changes behavior.

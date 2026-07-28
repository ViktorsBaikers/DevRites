---
name: devrites-audit
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
argument-hint: "<security | perf | simplify>"
user-invocable: false
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. Codex loads that role TOML's `developer_instructions` natively. Because V2 collaboration lifecycle calls bypass hooks, DevRites verifies the current durable parent/child rollout for the exact role, wait, completion, and non-empty delivered result.
- On MultiAgent V1, when the named role is not exposed, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`; do not substitute `worker` for an exposed V2 named role.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If any required named or generic agent dispatch is unavailable or rejected, stop for HITL. Never execute a DevRites specialist role in the root context.
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

Use the capability ladder. If no safe fresh-context rung preserves the role's
read-only boundary, stop for HITL. Use these role contracts:

- `.codex/agents/devrites-security-auditor.toml`
- `.codex/agents/devrites-performance-reviewer.toml`
- `.codex/agents/devrites-simplifier-reviewer.toml`

Stay inside the active feature. Critical findings block seal; simplification never
changes behavior.

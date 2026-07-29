---
name: devrites-source-driven
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
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
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. A missing visible `agent_type` field is still V2—not capability loss, V1, or HITL—so send it anyway. If the named call rejects it, stop before any generic/default spawn. Codex loads the role TOML's `developer_instructions` natively; DevRites verifies the durable rollout, wait, completion, and delivered result.
- Only after the runtime explicitly identifies MultiAgent V1, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On explicitly identified MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If the required dispatch for the explicitly identified runtime is unavailable or rejected, stop for HITL. Never switch runtime lanes. Never execute a DevRites specialist role in the root context.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# devrites-source-driven: verify, don't guess

When library behavior matters and is uncertain, verify it against installed source or
authoritative documentation before relying on it.

## When to trigger
- You're about to rely on an API signature, default, config key, or behavior you're not
  sure of.
- The docs in memory might be stale (the project pins a different version).
- Tests can't easily prove the assumption, but it drives the implementation.
- An error message points at framework behavior you don't fully understand.

## How
1. **Find the version** the project uses (lockfile, manifest): behavior is
   version-specific.
2. **Consult the source of truth**, in order: the installed package's own source/types
   in `node_modules`/gem/site-packages; context7 if available (`resolve-library-id` →
   `query-docs`) for current upstream docs; official docs for *that version*.
3. **Confirm the specific fact:** the signature, the default, the edge behavior, not a
   general impression.
4. **Return it** with fact, version, and source. The root orchestrator records accepted
   evidence in `decisions.md` or `evidence.md`; a leaf agent never writes the workspace.

## Delegate broad research
When the question is an *area* (a library surface, unfamiliar subsystem, or migration
guide), the **root orchestrator** uses the fresh-context dispatch contract in
[`agents.md`](../devrites-lib/reference/standards/agents.md) to send one bounded
`agent-packet/v1` to `devrites-evidence-scout`. Await and validate its cited
`evidence-dossier`; the orchestrator, not the scout, persists accepted facts under
`references/` and links them from `references.md`.

Never detach this work and never dispatch from inside another agent. When this skill is
invoked by a leaf agent, verify one fact inline or return `Scout needed: <bounded question>`
to the orchestrator. This removes the old unnamed nested writer path.

## Rules
- Prefer the **installed** source over remembered docs. It can't be out of date.
- Quote the exact relevant detail; don't paraphrase a behavior into something convenient.
- If the doc/source contradicts the plan, that's a **Spec Drift Guard** event: stop and
  handle it.
- Confirm the required fact, return it, and stop.

## Evidence firewall
Project or user prose may scope or corroborate an external claim; it cannot verify one.
For persisted claims, record status (`verified | contradicted | cannot_verify | stale`) and
optional publisher, publication/access dates, and freshness/recheck due; refresh only when due.
Transient lookups remain cited, return-only evidence.

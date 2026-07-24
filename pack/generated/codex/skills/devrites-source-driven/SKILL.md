---
name: devrites-source-driven
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
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

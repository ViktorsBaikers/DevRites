---
name: devrites-source-driven
description: Verify uncertain framework/library behavior against official docs or installed source before relying on it. Use when the user says "check the docs", "verify this assumption", or hits an unfamiliar API. Not for internal code.
user-invocable: false
required-agent-roles: none
---

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

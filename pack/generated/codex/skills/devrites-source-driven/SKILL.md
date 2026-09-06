---
name: devrites-source-driven
description: Verify uncertain framework or library behavior in installed source or official docs. Use for unfamiliar APIs; not internal code.
user-invocable: false
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
1. **Find the version** the project pins and actually runs; resolve any mismatch.
2. **Apply the hierarchy, citation, and currentness contract** in
   [`tooling.md`](../devrites-lib/reference/standards/tooling.md#research-provenance-staleness-and-cost).
   context7 is a lookup tool, not authority above version-matched source/docs.
3. **Confirm the specific fact:** the signature, the default, the edge behavior, not a
   general impression.
4. **Return it** with the required citation and status. The root records accepted
   evidence in `decisions.md` or `evidence.md`; a leaf agent never writes the workspace.

## Delegate broad research
When the question is an *area* (a library surface, unfamiliar subsystem, or migration
guide), the **root orchestrator** uses the fresh-context dispatch contract in
[`agents.md`](../devrites-lib/reference/standards/agents.md) to give one bounded
question to `devrites-evidence-scout`. Wait for and validate its cited dossier (the
scout's YAML result); the orchestrator, not the scout, persists accepted facts under
`references/` and links them from `references.md`.

Never detach this work and never dispatch from inside another agent. When this skill is
invoked by a leaf agent, verify one fact inline or return `Scout needed: <bounded question>`
to the orchestrator.

## Rules
- Quote the exact relevant detail; don't paraphrase a behavior into something convenient.
- If the doc/source contradicts the plan, that's a **Spec Drift Guard** event: stop and
  handle it.
- Confirm the required fact, return it, and stop.

## Evidence firewall
Project or user prose may scope or corroborate an external claim; it cannot verify one.
Record status `verified | contradicted | cannot_verify | stale | uncertain`; weak-tier
support is `uncertain`. Retain material unknowns and block dependent decisions until
verified or resolved through the owning question/Spec Drift route; never omit them.
Transient lookups remain cited, return-only evidence.

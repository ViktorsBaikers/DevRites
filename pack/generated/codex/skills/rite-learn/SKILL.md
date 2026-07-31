---
name: rite-learn
description: Review recurring cross-feature evidence and propose durable project guidance.
argument-hint: "[\"<lesson or rejected direction>\"]"
user-invocable: true
disable-model-invocation: true
---

# $rite-learn: review durable lessons

Use native session memory and reviewed project Markdown instead of an engine
miner, score, nudge, or convention registry. This command proposes durable
guidance; it never turns an inferred pattern into a rule on its own.

## Modes

- `$rite-learn` reviews recurring evidence across shipped work.
- `$rite-learn "<lesson>"` evaluates one proposed lesson or rejected direction.

## Workflow

1. Read the applicable `AGENTS.md` or `CLAUDE.md`, accepted ADRs, and relevant
   `.devrites/archive/*/{decisions,drift,review,seal}.md` files. Use native host
   memory or session history when available, but verify every durable claim
   against repository evidence.
2. For a broad review, ask the exact `devrites-retrospector` agent to inspect the
   bounded archive in a fresh read-only context. Wait for its result and
   reconcile it against the cited files.
3. Keep a candidate only when it is specific and supported by either:
   - the same correction in at least two distinct features; or
   - one explicit, durable product/architecture decision with its rationale.
4. Choose the smallest authoritative home:
   - `AGENTS.md`, `CLAUDE.md`, or a scoped standards document for an operating
     rule;
   - an ADR for a significant durable architecture decision;
   - the active feature's `decisions.md` for a feature-scoped decision;
   - no change for a one-off, stale, or generic observation.
5. Show the evidence and exact proposed edit. Apply it only after the user
   approves the durable rule or decision.

## Rules

- Live repository evidence outranks memory.
- Do not maintain a parallel learning ledger, convention score, health score,
  timeline, fingerprint index, or promotion queue.
- Do not duplicate guidance already stated by a higher or nearer authority.
- A rejected direction returns only when new evidence changes its recorded
  rationale.

## Output

```text
Done: reviewed <scope>.
Candidates: <specific proposals or none>.
Evidence: <feature/file references>.
Awaiting: <approval for exact edits | none>.
Next: <single action>.
```

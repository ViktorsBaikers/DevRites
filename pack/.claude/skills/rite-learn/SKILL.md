---
name: rite-learn
description: Review recurring cross-feature evidence and propose durable project guidance.
argument-hint: "[\"<lesson or rejected direction>\"]"
user-invocable: true
disable-model-invocation: true
---

# /rite-learn: review durable lessons

Use reviewed Markdown/native memory, not miners or registries. This proposes maintenance
of one authority; it cannot promote a rule alone.

## Modes

- `/rite-learn` reviews recurring evidence across shipped work.
- `/rite-learn "<lesson>"` evaluates one lesson or rejected direction.

## Workflow

1. Read [durable promotion](../devrites-lib/reference/standards/documentation.md#promote-durable-guidance).
   Bound the archive; inspect applicable
   `AGENTS.md`/`CLAUDE.md`, accepted ADRs, and relevant
   `.devrites/archive/*/{decisions,drift,review,seal}.md` files.
2. In broad mode, dispatch exact `devrites-retrospector` fresh/read-only; reconcile its
   claims against cited files.
3. Keep a correction repeated in two features or one explicit durable product/architecture
   decision with rationale. Drop one-off, generic, or stale items.
4. Verify claims against live authoritative repository sources. State the currentness signal,
   applies/does-not-apply scope, and `unknown` where unverifiable.
5. Search guidance for the same/contrary rule. Choose one existing canonical owner: nearest
   instruction/standard, architecture ADR, or feature `decisions.md`; name discovery.
6. Show the exact edit and duplicate/conflict/supersession disposition. Update, narrow,
   replace, or retire contradictions; apply only after user approval of the exact edits.

## Rules

- Live repository evidence outranks memory; unverifiable is unknown, not false.
- Never create a learning ledger/index/queue, score, timeline, or parallel authority.
- Rejected directions return only when evidence changes their rationale.

## Output

```text
Done: reviewed <scope>.
Candidate: <specific proposal or none>
Currentness: <live source + signal | unknown>
Scope: applies <trigger>; does not apply <boundary>
Authority: existing <path|none>; canonical <path>; consumers/discovery <route>
Disposition: <no conflict | update/narrow/replace/retire path + reason>
Evidence: <feature/file references>
Awaiting: <approval for exact edits | none>
Next: <single action>
```

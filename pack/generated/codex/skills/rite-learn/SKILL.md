---
name: rite-learn
description: Review recurring cross-feature evidence and propose durable project guidance.
argument-hint: "[\"<lesson or rejected direction>\"]"
user-invocable: true
disable-model-invocation: true
---

# $rite-learn: review durable lessons

Use reviewed Markdown/native memory, not miners or registries. This proposes maintenance
of one authority; it cannot promote a rule alone.

## Modes

- `$rite-learn` reviews recurring evidence across shipped work.
- `$rite-learn "<lesson>"` evaluates one lesson or rejected direction.

## Workflow

1. Read [durable promotion](../devrites-lib/reference/standards/documentation.md#promote-durable-guidance);
   bound the archive; inspect applicable `AGENTS.md`/`CLAUDE.md`, accepted ADRs, and relevant `.devrites/archive/*/{decisions,drift,review,seal}.md`.
2. Broad mode dispatches exact fresh/read-only `devrites-retrospector`; reconcile its claims against cited files.
3. Keep corrections repeated in two features, a judgement call made twice, or a defect class seen twice; drop one-off preferences, task-specific detail, generic advice.
4. Verify claims against live authoritative sources; state currentness signal, applies/does-not-apply scope, `unknown` where unverifiable.
5. Search guidance for same/contrary rules; choose one existing canonical owner (nearest instruction/standard, architecture ADR, or feature `decisions.md`) and name discovery.
6. Show the exact edit + duplicate/conflict/supersession disposition; update/narrow/replace/retire contradictions; apply only after user approval of exact edits.

## Rules

- Live repository evidence outranks memory; unverifiable is unknown, not false.
- Never create a learning ledger/index/queue, score, timeline, or parallel authority; rejected directions return only when evidence changes their rationale.
- A declined lesson persists as a declined decision entry (reason recorded) in the nearest owning decisions file — not re-litigated without new evidence.
- Contradiction outranks staleness: actively misleading guidance outranks merely old guidance.
- ≤3 accepted lessons per round; proposals extend/narrow but never lower an existing bar (revisions show old text beside new); duplicates consolidate into one canonical edit — simplification (deletions/merges) counts toward the cap.

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

---
name: rite-clarify
description: Run the topology-first decision-coverage scan after rite-spec and before strategy or architecture. Audit actors, journeys, states, data, integrations, operations, proof, assumptions, and must-NOT boundaries; close Partial/Missing rows and emit decision-coverage.md CLEAR. Use for missing/stale coverage or a missed pre-build product decision.
argument-hint: "[feature-slug]"
user-invocable: true
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


# $rite-clarify: resolve product decisions

Run this required pass between spec and strategy. If the spec covers every material
decision, ask no questions. Otherwise stay until each material decision has an
owner. This phase settles behavior and constraints; `$rite-define` settles
implementation.

Reuse [`devrites-interview`](../devrites-interview/SKILL.md) in `/clarify mode`. Write
`decision-coverage.md` as scan evidence, not an interview transcript.

## Rules consulted

Read `devrites-lib/reference/standards/core.md`, its `afk-hitl.md`, and
[`reference/decision-coverage.md`](reference/decision-coverage.md). Fresh evidence
dispatch uses
[`agents.md`](../devrites-lib/reference/standards/agents.md).

## Rules

- Search code/docs/contracts first. Facts and reversible implementation/test choices
  are agent-owned; ask only about product, scope, policy, irreversible risk, or human-only access.
- Enumerate the full topology before details so no sibling surface is omitted.
- Ask one coherent human-owned decision packet at a time. One packet may close several rows
  only when owner and trade-off match; never ask permission for routine repair/retry.
- One scan may cap at five packets for cognitive load, but readiness has no question cap:
  re-scan until clear.
- Material assumptions carry evidence, confidence, owner, validation, and consequence.

## Workflow

0. **Orient.** Run `devrites-engine preamble` and `snapshot`. Require `spec.md` plus
   `Spec gate: passed`; otherwise STOP → `$rite-spec`. For a later-phase retrofit, run
   `devrites-engine clarify-return enter <slug>` **before any pause**; this persists the
   return cursor and enters `Phase: clarify`.
1. **Enumerate topology** from the spec, references, live code, and prior decisions: actors,
   journeys/components, states, data lifecycles, interfaces/integrations, operations, and
   proof. Add omitted surfaces to the spec. **Completion:** every material surface appears
   once with a source or canonical spec reference.
2. **Scan coverage** with `devrites-interview /clarify mode` and the reference taxonomy.
   Mark each surface Clear/Partial/Missing with evidence; include must-NOT and out-of-scope
   behavior. Unmentioned never means not applicable. **Completion:** the matrix records every
   surface with evidence.
3. **Eliminate factual unknowns** through code intelligence, project/decision docs,
   manifests/lockfiles, and authoritative external docs. Dispatch independent topology/fact
   questions to `devrites-evidence-scout` (maximum three at once), await the cited dossiers,
   and reconcile them in the root context. The root folds accepted facts into `spec.md` or
   `assumptions.md`; the scout never asks or writes. **Completion:** each factual unknown is
   source-backed or reclassified as an owned decision/assumption.
4. **Audit assumptions and decisions.** Record the fields required above. For high-cost or
    hard-to-reverse product/constraint choices, present two or three viable options, recommendation
   first. A genuinely undecidable behavior may take one bounded `$rite-prototype` detour.
5. **Close human-owned gaps** using the shared option-set contract: one highest-impact packet
   per turn. Persist answers immediately in `spec.md` plus `decisions.md`/`questions.md`.
   AFK remains within its gate ceiling; irreversible risk and principle exceptions pause.
6. **Re-scan after edits** until each material row is `closed`, `agent-owned`,
   `not-applicable`, or justified `deferred-nonblocking` with owner and validation gate.
   Partial/Missing, an unowned material assumption, or an open blocking/escalating question
   means `NEEDS CLARIFICATION`.
7. **Write the verdict.** Update `decision-coverage.md`; success requires exactly
   `Decision coverage: CLEAR`. Normal state is `Phase: clarify`, `Next step: $rite-temper`.
   A contract-neutral retrofit runs `devrites-engine clarify-return restore <slug>`; a
   changed behavior/acceptance contract leaves the return cursor unconsumed, writes
   `drift.md`, and routes `$rite-plan repair`. Normal first-pass flow sets `$rite-temper`.
8. **STOP.** Run `devrites-engine progress`; hand off only on `CLEAR`.

## Output

Run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)):

```text
Done: decision coverage closed for <slug>; <n> topology surfaces scanned.
Changed: decision-coverage.md, spec.md, decisions.md, assumptions.md, questions.md
Evidence: Decision coverage: CLEAR; human packets <n>; agent-owned facts <n>
Open: none blocking; deferred-nonblocking <n>
Next: $rite-temper
Record: .devrites/work/<slug>/decision-coverage.md
↻ Hygiene: /clear before $rite-temper
```

If not clear, name the exact rows and next genuine decision packet. An objective spec defect
or factual search task never routes through `$rite-resolve`.

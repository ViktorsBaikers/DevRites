---
name: rite-clarify
description: Run the topology-first decision-coverage scan after rite-spec and before strategy or architecture. Audit actors, journeys, states, data, integrations, operations, proof, assumptions, and must-NOT boundaries; close Partial/Missing rows and emit decision-coverage.md CLEAR. Use for missing/stale coverage or a missed pre-build product decision.
argument-hint: "[feature-slug]"
user-invocable: true
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
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
7. **Write the verdict.** Update `decision-coverage.md`; success requires exactly one
   `DevRites contract: devrites.readiness-artifacts.v2` field and exactly
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

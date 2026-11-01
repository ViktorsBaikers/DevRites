---
name: rite-clarify
description: Audit a completed spec for missing decisions before strategy or architecture. Use after /rite-spec when coverage is incomplete or stale; not for spec writing.
argument-hint: "[feature-slug]"
---

<!-- loads: {"always":["devrites-lib/reference/parallel-workspaces.md","devrites-lib/reference/standards/core.md","devrites-lib/reference/standards/code-navigation.md","devrites-lib/reference/standards/code-index.md","devrites-lib/reference/standards/afk-hitl.md","rite-clarify/reference/decision-coverage.md","rite-clarify/reference/anti-patterns.md"],"triggers":{"agents":["devrites-lib/reference/standards/agents.md"]},"workspace":["brief.md","spec.md","state.md","decisions.md","assumptions.md","questions.md","decision-coverage.md"],"workspaceByRole":{"evidence-scout":["brief.md","spec.md","state.md","questions.md"]},"rootOnly":["devrites-lib/reference/parallel-workspaces.md","devrites-lib/reference/standards/code-index.md","devrites-lib/reference/standards/afk-hitl.md"]} -->
> Read-set manifest: `devrites-engine context --skills-root .devin/skills <slug> --phase clarify` bundles every file named below into one deduplicated read. Trigger names map to the conditional rules in the sections that follow.


# /rite-clarify: resolve product decisions

Run this required pass between spec and strategy. If the spec covers every material
decision, ask no questions. Otherwise stay until each material decision has an
owner. This phase settles behavior and constraints; `/rite-define` settles
implementation.

Reuse [`devrites-interview`](../devrites-interview/SKILL.md) in `/clarify mode`. Write
`decision-coverage.md` as scan evidence, not an interview transcript.

## Rules consulted

Read [`core.md`](../devrites-lib/reference/standards/core.md), its [`afk-hitl.md`](../devrites-lib/reference/standards/afk-hitl.md), and
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
- Refuse the rationalizations in [`anti-patterns`](reference/anti-patterns.md).

## Workflow

1. **Orient.** Resolve source per [phase target binding](../devrites-lib/reference/parallel-workspaces.md#phase-target-binding), then read its `state.md` and `spec.md`. Require
   `state.md` `spec_gate: passed` (legacy `Spec gate: passed` line accepted); otherwise stop at `/rite-spec`. Apply the native cursor
   protocol below before changing workspace artifacts.
2. **Enumerate the topology.** From the spec, live code, contracts, references,
   and recorded decisions, list every material stakeholder/priority, invariant, actor,
   journey/component, state, data lifecycle, integration, failure/recovery path,
   operation, proof surface, applicability row, and must-NOT boundary.
3. **Scan coverage.** Apply `devrites-interview` in clarify mode and mark each
   material surface `closed`, `agent-owned`, `not-applicable`,
   `deferred-nonblocking`, or `open` with evidence and an owner. Use native repository search
   first. Ask the host to run `devrites-evidence-scout` only for a bounded missing
   fact, then reconcile its cited result.
4. **Close decisions.** Record facts and reversible technical choices directly.
   Ask one coherent option packet only for product, scope, policy, irreversible
   risk, or human-only access. Persist answers in the owning artifacts and repeat
   the scan until no blocking `open` row or unowned material assumption
   remains.
5. **Write the verdict.** Write `Decision coverage: CLEAR` only after re-reading all inputs and confirming
   every material row has current evidence and an owner. Normal flow sets `phase=clarify` and
   `next_action=/rite-temper`, or `/rite-define` for low-stakes specs temper skips.
   Later-phase entries follow the ownership protocol below. Return inline verdicts
   to the controlling caller; otherwise stop before starting the next phase.

## Native clarify cursor protocol

Edit only cursor rows; preserve unrelated Markdown and table/bullet presentation.

- **Validate before writes:** require no return fields or a complete pair:
  recognized later `return_phase`, non-empty `return_next_action`. Invalid/half
  pairs fail closed. Later phases: `temper`, `define`, `plan`, `vet`, `build`,
  `converge`, `prove`, `polish`, `review`, `seal`, `ship`.
- **Spec entry:** set `phase=clarify`, `status=running`,
  `next_action=/rite-clarify <slug>`; omit return fields.
- **Caller-owned pair:** pair with `phase!=clarify` requires a native
  controlling caller; otherwise stop for ownership recovery. Run bounded Clarify
  inline, preserve all cursor rows, and return its verdict to caller.
  Contract changes require Plan repair and Vet before outer resume.
- **Clarify-owned pair:** standalone later-phase entry without a pair saves current
  phase/non-empty next action as the pair, then sets `phase=clarify`,
  `status=running`, `next_action=/rite-clarify <slug>`. Missing/unknown values
  fail closed. `phase=clarify` resumes without writes; its pair belongs to
  standalone Clarify, not a nested caller.
- **Restore:** after fresh `CLEAR`, only the owner restores its pair.
  Contract-neutral standalone Clarify restores phase/next action, sets
  `status=running`, atomically unsets both return fields, and re-reads to verify.
  Contract changes retain the pair through Plan repair and Vet; only its owner
  consumes it after readiness. Preserve caller pairs.

## Output

```text
Done: decision coverage closed for <slug>; <n> topology surfaces scanned.
Changed: decision-coverage.md, spec.md, decisions.md, assumptions.md, questions.md
Evidence: Decision coverage: CLEAR; human packets <n>; agent-owned facts <n>
Open: none blocking; deferred-nonblocking <n>
Next: /rite-temper (or /rite-define for a low-stakes spec temper skips)
Record: .devrites/work/<slug>/decision-coverage.md
↻ Hygiene: /clear — decision coverage is persisted in decision-coverage.md
```

If not clear, name the exact rows and next genuine decision packet. An objective spec defect
or factual search task never routes through `/rite-resolve`.

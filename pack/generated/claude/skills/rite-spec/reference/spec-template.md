# `spec.md` template

Contract WHAT users get, WHY, success, and scope. HOW belongs in `plan.md`,
topology in `architecture.md`/`flows.md`, coverage in `traceability.md`.

Use `[NEEDS CLARIFICATION: <question>]` (blocking stops `/rite-clarify`); before readiness
every surviving marker converts to a gated `Q-###` open question (`spec-grammar.md` §
Unresolved-question markers — fail closed). Stable `REQ-001`/`AC-001` IDs; link, never
duplicate, source artifacts; over-budget requires `Budget override: <reason>`.

```markdown
# Spec: <Feature>
Slug: <kebab-case>
Status: Draft | Ready
Created: <date>

## Problem
<What is broken, missing, or costly.>

## Goal
<User-visible outcome and why it matters.>

Capability impact: <affected capability/ies and change | none — specific justification>

## Non-goals
- <Explicitly out of scope.>

## Existing behavior to preserve
List affected observable public/security/data/operational outcomes that MUST
survive; map preserving REQ/AC plus current test/runtime/contract/source evidence.
No implementation detail. True greenfield: `none — no existing behavior in the affected scope`.

| Existing outcome | Preserved by | Current evidence |
| --- | --- | --- |
| <outcome that must not regress> | REQ-001 / AC-001 | <current evidence> |

Each preservation row **must** cite current evidence (test, runtime, contract, or
observed behavior). An empty or vague evidence cell blocks Spec readiness.

**Failing case:** row lists REQ-001 with evidence "none" or "TBD" → readiness gate
fails until evidence is named or the outcome is removed from scope.

## Stakeholders and priorities
| Actor/stakeholder | Observable outcome | Conflict / priority rule |
| --- | --- | --- |
| <actor or affected owner> | <goal, protection, or operational need> | <none or how competing goals resolve> |

## Constraints and invariants
- INV-001: <fact that MUST remain true across success, failure, retry, and recovery>.
- <security/privacy/accessibility/performance/compatibility/data/operational constraint,
  or `none — <specific reason>` for a materially relevant category>.

## Requirements
- REQ-001: The system MUST <observable product behavior>.
- REQ-002: The system MUST NOT <prohibited behavior>.

## Acceptance criteria
Binary/evidence-backed; each maps to a requirement and later a traceability slice.

- [ ] AC-001: Given <state>, when <action>, then <outcome>. (REQ-001)
- [ ] AC-002: Given <state>, when <action>, then <outcome>. (REQ-002)

Behavioral/high-risk work uses this grammar, with the AC ID inside the scenario:

### Requirement: <name>
The system SHALL <core observable behavior>.

#### Scenario: <name>
- [ ] AC-003: **WHEN** <trigger> **THEN** <observable outcome>. (REQ-001)

## Edge Coverage
Use `covered | backstop | dismissed | unresolved`; target an existing REQ/AC
unless dismissed. `backstop` requires a named independent held-out,
property/metamorphic, or direct behavioral check plus the wrong behavior it
discriminates. If unavailable, use `unresolved`; presence/prose/self-judgment cannot pass.

| Edge ID | Requirement/AC | Class | Status | Reason/backstop |
| --- | --- | --- | --- | --- |
| EDGE-001 | AC-001 | empty/error/permission/race/migration | covered | <evidence/rationale> |

## Prohibitions (must-NOT)
Bespoke only; generic security/privacy stays in standards. Status:
`resolved/test | resolved/judgment | dismissed | unresolved`.

| Prohibition ID | Requirement/AC | Status | Test/evidence |
| --- | --- | --- | --- |
| PROH-001 | REQ-002 | resolved/test | <test/evidence link> |

## Failure and recovery behavior
| Trigger / partial state | User-visible outcome | System state | Recovery / retry rule | Requirement/AC |
| --- | --- | --- | --- | --- |
| <timeout, invalid input, interruption, dependency loss> | <clear bounded outcome> | <unchanged/pending/reconciling> | <who/what can safely recover> | <REQ/AC> |

## Applicability map
Use `applies | not applicable`; a non-applicable row needs a specific reason. The
status routes Define/Vet/Build/Prove to the named standard without copying it here.

| Concern | Status and trigger | Affected REQ/AC/invariant |
| --- | --- | --- |
| Repository topology (nested/mono/multi-repo, languages, services, generated/vendor) | <status + reason> | <ids> |
| Data integrity (writes, schema/migration, concurrency, tenant, retention/privacy) | <status + reason> | <ids> |
| Integration reliability (API/webhook/queue/job/cache/cross-service) | <status + reason> | <ids> |
| Security boundary (authn/authz, hostile input/files, secrets, privilege) | <status + reason> | <ids> |
| UI/accessibility/i18n/time-zone behavior | <status + reason> | <ids> |
| Compatibility/delivery (old/new versions, config, flag, rollout/rollback) | <status + reason> | <ids> |

## Edge cases
- <Boundary note not captured above.>

## AI-SPEC annex
- Model/RAG/agent/eval/LLM-output scope: `ai-spec.md` from `ai-spec-template.md`.
- Otherwise: not applicable.

## Success metrics
- <Metric or observable proof.>

## Scope boundaries
- Owns: <surface/behavior>.
- Does not own: <adjacent area>.
- Placement summary: <module/layer>; full map in `architecture.md`.

## Coverage seed
- Actors/journeys/components: <material surfaces>.
- States/data/contracts/integrations: <material boundaries>.
- Operations/proof: <config, observability, rollout/rollback, evidence constraints>.

## References
- `brief.md`: request/outcome/scope; `architecture.md`: placement/integration;
  `flows.md`: Mermaid-first diagrams (optional `visual/<flow>.html`+`.outline.md` companion
  + link when richer presentation earns it — load playbooks via
  `devrites-lib/reference/visual-playbooks/index.md`); `decisions.md`: decisions;
  `decision-coverage.md`: Clarify topology/verdict; `traceability.md`: Define coverage;
  `design-brief.md`: UI direction.

## Open questions
| Question ID | Gate | Question | Impact |
| --- | --- | --- | --- |
| Q-001 | blocking | [NEEDS CLARIFICATION: <question>] | AC-001 |

## Readiness gate
- [ ] No blocking clarification or open `blocking`/`escalating` question; REQ/AC IDs are valid and ACs independently provable.
- [ ] Stakeholder conflicts/priority rules, constraints, and invariants are explicit;
      implementation preferences are not disguised as requirements.
- [ ] Existing affected behavior maps to preserving REQ/AC + current evidence, or uses the exact justified greenfield `none`.
- [ ] Edge rows target REQ/AC or justify dismissal; every backstop names independent discriminating evidence, else `unresolved`.
- [ ] Prohibitions resolve/dismiss; `resolved/test` links evidence.
- [ ] Each material failure/partial state names user outcome, system state, recovery,
      and REQ/AC; no silent success or blind retry remains.
- [ ] Every applicability row is `applies` with affected IDs or has a specific
      evidence-backed `not applicable` reason.
- [ ] AI has `ai-spec.md` and UI has `design-brief.md`; out-of-scope work states not applicable.
- [ ] Non-goals/scope are explicit; capability impact is singular/specific and matches ledger deltas.
- [ ] Architecture/flows/decisions are linked, not duplicated; Coverage seed names Clarify surfaces.
```

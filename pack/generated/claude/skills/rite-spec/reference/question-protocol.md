# Spec question protocol

For vague or unknown intent, invoke `devrites-interview` for ordered one-question turns,
recommendations, option sets, pass limits, reframing, and closure.
This file owns only Spec-specific coverage and recording. Render through
[`Option set`](../../devrites-lib/reference/standards/afk-hitl.md#option-set-how-every-gap-is-presented).

## Coverage gate: am I done asking?
Each dimension MUST be **resolved** or **explicitly deferred** (logged, non-blocking),
never skipped:
- [ ] **Objective:** one-sentence success + the real problem behind it.
- [ ] **Scope**: what's in vs explicitly out for v1.
- [ ] **Data model**: core entities + relationships (or "none").
- [ ] **UX / flow**: happy path + empty/loading/error/permission states (or "no UI").
- [ ] **Integration**: external systems/APIs/contracts (or "none").
- [ ] **Non-functional**: auth, sensitive data, latency/scale (or "n/a").
- [ ] **Acceptance**: how each requirement is *proven* (test/observation).
- [ ] **Human prerequisites**: credentials, approval windows, irreversible action-time gates
  (or "none").

An agent recommendation is not a user answer. A human-owned material product, scope, UX,
compatibility, or acceptance choice closes only by the user's option selection, free-form
answer, or explicit deferral. Packets MAY be bounded; readiness MUST NOT. Rescan and continue
while blockers remain. If the user stops, keep them blocking and MUST NOT pass `/rite-spec`
readiness.

## Spec question boundary
- Search code, decisions, authoritative docs before asking discoverable facts.
- Decide and log agent-owned reversible implementation/test details.
- Record tooling failures as prerequisites or blockers; never ask the human to authorize repair.
- Ask only material choices, not everything at once.

## Record
Confirmed answers → `decisions.md` (with rationale). Agent-owned reversible, low-impact
technical assumptions →
`assumptions.md`. Open non-blocking items → `questions.md`.

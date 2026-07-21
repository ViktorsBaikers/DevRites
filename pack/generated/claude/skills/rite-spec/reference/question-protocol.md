# Spec question protocol

For vague or unknown intent, invoke `devrites-interview`; it owns question order,
one-question turns, best guesses, option sets, blocking-question budget, reframing,
and completion. The shared HITL/AFK rendering contract lives in
[`afk-hitl.md` § Option set](../../devrites-lib/reference/standards/afk-hitl.md#option-set--how-every-gap-is-presented).

This reference owns only the spec-specific coverage and recording rules below.

## Coverage gate — am I done asking?
Before declaring the interview complete, each dimension is **resolved** or **explicitly
deferred** (logged, non-blocking) — never silently skipped:
- [ ] **Objective** — one-sentence success + the real problem behind it.
- [ ] **Scope** — what's in vs explicitly out for v1.
- [ ] **Data model** — core entities + relationships (or "none").
- [ ] **UX / flow** — happy path + empty / loading / error / permission states (or "no UI").
- [ ] **Integration** — external systems / APIs / contracts (or "none").
- [ ] **Non-functional** — auth, sensitive data, latency / scale (or "n/a").
- [ ] **Acceptance** — how each requirement is *proven* (test / observation).

A blocking gap in any dimension keeps the interview open; a deferred one goes to
`questions.md` and doesn't block. This feeds the `/rite-spec` readiness gate.

## Spec question boundary
- Things the codebase answers (read it first).
- Reversible implementation details (decide and note as an assumption).
- Everything at once "to be thorough." Thoroughness is depth on the few that matter.

## Record
Confirmed answers → `decisions.md` (with rationale). Standing assumptions →
`assumptions.md`. Open non-blocking items → `questions.md`.

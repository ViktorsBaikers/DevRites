# Cleanup + classify

## Classify one causal fingerprint

Fingerprint the causal diagnosis, never the failing test or symptom. Reuse it while
plausible. If a discriminating fix proves the cause absent but the symptom remains,
preserve that diagnosis as a dead end and classify a new fingerprint. Failure alone
never resets a budget.

- `intent_gap`: desired behavior, scope, policy, or risk choice is unsettled.
- `spec_gap`: an acceptance outcome or product decision is missing.
- `plan_gap`: behavior is settled, but technical wiring, dependency, or proof planning is incomplete.
- `implementation_defect`: product code violates settled acceptance.
- `proof_tool_defect`: a scanner, test, harness, or evidence collector gives the wrong verdict.
- `environment_defect`: setup, capacity, tooling, or an external service prevents valid proof.
- `preexisting`: the same failure exists outside the candidate delta.
- `not_a_defect`: the observation matches current accepted authority.

Ask the engine for the canonical owner and action:

```bash
devrites-engine recovery route <class>
```

Follow `recovery-route/v1`. `humanPause: true` routes unresolved intent/spec through
Clarify. Technical routes stay agent-owned. Gate only an exact human credential, quota,
irreversible action, or user-owned process—never retry authorization.

Record the class with every new failed attempt and green clear:

```bash
devrites-engine recovery record --class <class> "<root cause>" "<exact failure>" <slug>
devrites-engine recovery clear --class <class> "<root cause>" <slug>
```

The record also states the exact observation, candidate baseline, evidence for and against,
next discriminating probe, attempt count, and any human-only predicate.

## Cleanup checklist: required before declaring done

- [ ] Original repro no longer reproduces (re-run the Phase 1 loop).
- [ ] Regression test passes (or absence of seam is documented).
- [ ] All `[DEBUG-...]` instrumentation removed (`grep` the prefix).
- [ ] Throwaway harnesses deleted (or moved to a clearly marked debug location).
- [ ] The correct hypothesis is stated in `evidence.md` + the commit/PR message: next debugger learns.

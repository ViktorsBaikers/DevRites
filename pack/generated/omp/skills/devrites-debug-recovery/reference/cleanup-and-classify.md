# Cleanup + classify

## Classify one causal fingerprint

Fingerprint the diagnosis, not its symptom. Reuse it until evidence disproves it; then
record the dead end and classify anew. Failure alone never resets the budget.

- `intent_gap`: desired behavior, scope, policy, or risk choice is unsettled. Pause for the human to clarify intent.
- `spec_gap`: an acceptance outcome or product decision is missing. Return to Clarify.
- `plan_gap`: behavior is settled, but technical wiring, dependency, or proof planning is incomplete. Repair the plan.
- `implementation_defect`: product code violates settled acceptance. Return it to the slice-wright, then rerun the original proof.
- `proof_tool_defect`: a scanner, test, harness, or evidence collector gives the wrong verdict. Repair the proof tool in this recovery loop, then rerun the original proof.
- `environment_defect`: setup, capacity, tooling, or an external service prevents valid proof. Normalize the environment here and run a discriminating check.
- `preexisting`: the same failure exists outside the candidate delta. Record the baseline and fix it only when it blocks acceptance.
- `not_a_defect`: the observation matches current accepted authority. Record that authority and continue.

Only human credentials/quotas/actions, irreversible work, or fresh authorization
required before the next consumptive execution pause. Fresh authorization never
pauses offline diagnosis or correction from retained evidence; never ask for a
blind retry.

Record class/routing in `decisions.md` and each failed attempt in `evidence.md`
or `## Dead ends`. Apply the causal-fingerprint and three-attempt rules in
`devrites-debug-recovery/SKILL.md` Hard rules; record baseline, exact failure,
hypothesis/probe, attempt number, and any human predicate.

## Cleanup checklist: required before declaring done

- [ ] Original repeatable repro no longer reproduces, or the consumptive action's
      offline regression and evidence-completeness fixtures pass without a rerun.
- [ ] Regression test passes (or absence of seam is documented).
- [ ] All `[DEBUG-...]` instrumentation removed (`grep` the prefix).
- [ ] Throwaway harnesses deleted (or moved to a clearly marked debug location).
- [ ] The correct hypothesis is stated in `evidence.md` + the commit/PR message: next debugger learns.

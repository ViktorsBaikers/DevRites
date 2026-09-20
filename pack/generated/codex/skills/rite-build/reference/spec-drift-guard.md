# Spec Drift Guard (build phase)

Canonical Spec Drift Guard, active in `$rite-build` and referenced by other
phases. Never silently code against a known-wrong plan.

## Drift has occurred when
- the spec says X but the code makes X impossible/wrong;
- the plan assumes a file/API/component/route/model/command that doesn't exist;
- tests show the acceptance criteria are incomplete or wrong;
- browser evidence shows the intended UX doesn't work as described;
- an official doc contradicts the planned approach;
- the design system / existing UX pattern contradicts the planned UI;
- a missing user decision that affects product behavior surfaces.

## Workflow
```
1. STOP coding.
2. Record drift.md (assumed vs observed, when).
3. Classify: requirement ambiguity | implementation-plan error | codebase reality
   mismatch | design-system mismatch | test/evidence mismatch | external-doc mismatch
   | user-decision required.
4. Local repair that preserves behavior/scope/architecture/data/UX/security/migration?
     YES, active-slice technical/tool failure → log it and use Build's bounded debug
       recovery. Do not ask for retry authorization and do not re-plan unless the
       durable remaining-work instructions are wrong.
     YES, durable plan is wrong → run the batch sweep below, log every
       violation, save the caller's return cursor, then invoke `$rite-plan repair` and `$rite-vet` inline
       without a question — ONE folded repair for all open entries, ONE
       recheck for every open fingerprint — and resume.
     NO, product/policy/irreversible-risk decision → ask the user (format below).
5. Never continue on a known-wrong durable plan. A repaired active-slice implementation
   may continue only after bounded recovery, returned-diff review, and proof gates pass.
```

## Batch sweep (before repair)

Repair durable-plan drift as a folded set:

1. Trace the falsified assumption's full dependency/consumer closure and affected
   frontier, including dependent future slices. Verify unchecked paths, symbols/APIs/
   signatures, routes, commands, versions, and artifacts there against live source.
   Reuse the wright inventory and unchanged evidence; deterministic existence checks
   need no dispatch. Do not rescan unrelated slices. Broaden unknown closure or shared
   contract changes until coverage is justified; retain every known gap.
2. Record each violation as its own `drift.md` entry/fingerprint; one folded
   repair + one batched recheck resolves the set.
3. A recheck-surfaced gap is a new fingerprint with its own budget — loop
   internally, never hand an intermediate command to the human. When
   consecutive folded repairs keep landing on the same clause set, the
   repair packet's `convergence_pressure` marking applies (see `$rite-plan`
   repair): bounded declared residuals may close refinements of
   already-pinned clauses; a newly surfaced failure mode is never
   residual-eligible. A suspect seam's declare-limitation-or-continue choice
   routes to `$rite-clarify` like any human-owned gate — it is not an
   intermediate command.
4. Nothing skips or defers silently: every entry ends resolved, human-escalated,
   or fingerprint-exhausted. Execution-only gaps (tests, UX, evidence) get the
   identical path when they surface.

## User question format
```
I hit spec drift:
- Spec/plan assumed: ...
- Code/evidence shows: ...
- Why it matters: ...

Which direction should DevRites take?
1. Keep the requirement, change architecture by ...
2. Adjust the requirement to match existing behavior ...
3. Split into a follow-up feature ...
4. Custom: describe the intended behavior
```

Never turn an objective defect, environment repair, tool bug, or proof rerun into a
human permission question. Re-plan only when the durable plan changed.

## Inline return contract

The detecting phase owns backtracking and the batch sweep. Save
`return_phase`/`return_next_action`; consume nested Plan/Vet `STOP` internally, complete
vetted acceptance-preserving remediation, then restore the cursor and failed step.
Never hand the human an intermediate command. The shared three-attempt causal-fingerprint
cap still applies: exhaustion produces one technical blocker, while a genuinely
human-owned decision uses the question format below.

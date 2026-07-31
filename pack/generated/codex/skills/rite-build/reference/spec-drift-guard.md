# Spec Drift Guard (build phase)

Active throughout `$rite-build`. The spec is living, not sacred, but you may not
silently code against a plan you know is wrong. This is the canonical Spec Drift Guard;
other phases reference it here.

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
2. Record in drift.md (assumed vs observed, when).
3. Classify: requirement ambiguity | implementation-plan error | codebase reality
   mismatch | design-system mismatch | test/evidence mismatch | external-doc mismatch
   | user-decision required.
4. Local repair that preserves behavior/scope/architecture/data/UX/security/migration?
     YES, active-slice technical/tool failure → log it and use Build's bounded debug
       recovery. Do not ask for retry authorization and do not re-plan unless the
       durable remaining-work instructions are wrong.
     YES, durable plan is wrong → log it, run $rite-plan repair without a question,
       re-vet, then resume.
     NO, product/policy/irreversible-risk decision → ask the user (format below).
5. Never continue on a known-wrong durable plan. A repaired active-slice implementation
   may continue only after bounded recovery, returned-diff review, and proof gates pass.
```

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

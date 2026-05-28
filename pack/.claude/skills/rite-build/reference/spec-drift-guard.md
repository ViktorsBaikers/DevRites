# Spec Drift Guard (build phase)

Active throughout `/rite-build`. The spec is living, not sacred — but you may not
silently code against a plan you know is wrong. Canonical version:
`rite-define/reference/spec-drift-guard.md`.

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
     YES → log in drift.md + decisions.md, /rite-plan repair, resume.
     NO  → ask the user (format below). Don't continue on the old plan.
5. Run /rite-plan repair before resuming.
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

Never continue coding on a known-wrong plan. Always re-plan before resuming.

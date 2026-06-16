# GO / NO-GO

The seal verdict is binary and evidence-based. When in doubt, NO-GO with a clear blocker
list beats a hopeful GO.

## NO-GO if any of these hold
- A **critical acceptance criterion is unproven** (no evidence, or evidence shows fail).
- **Tests or build fail** without an explicit, user-accepted risk.
- The UI **cannot be verified** and the UI risk is material (visible/important surface).
- **Unresolved spec drift** remains (a known-wrong plan or an open behavior question).
- A **security-critical issue** remains (auth bypass, data exposure, injection).
- A **data migration or destructive change lacks a rollback plan**.
- Any `questions.md` entry with `gate: validating` and `status: open` — **NO-GO
  regardless of behavior impact** (an open validating gate is merge-blocking by
  definition). A slice marked `built (pending review)` is not done.

## GO requires
- Every critical acceptance criterion checked with evidence attached.
- Tests + build green for the scope (or documented, user-accepted exceptions).
- Browser proof present for material UI (or an explicit, accepted manual-only note).
- No open questions/drift that change product behavior.
- Security/data/migration risks either resolved or explicitly accepted by the user with
  a rollback path.

## Conditional GO
If only **non-blocking follow-ups** remain, it's a GO with a recorded follow-up list —
not a NO-GO. Distinguish "must fix to ship" (blocker) from "should do next" (follow-up).

## Honesty
- Don't upgrade a NO-GO to GO to please the user. State blockers plainly.
- Don't claim evidence you don't have. "Unproven" is a valid, important status.
- If subagent reviewers disagree, surface the disagreement and decide explicitly.

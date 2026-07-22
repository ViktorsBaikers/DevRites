# GO / NO-GO

The seal verdict is binary and evidence-based. When in doubt, NO-GO with a clear blocker
list beats a hopeful GO. The **authoritative gate is the severity table in
[`../SKILL.md`](../SKILL.md) ("Severity gate")**; this file is the rationale behind it: keep
the two in sync.

## NO-GO if any of these hold
- A **critical acceptance criterion is unproven** (no evidence, or evidence shows fail).
- **Tests or build fail** without an explicit, user-accepted risk.
- The UI **cannot be verified** and the UI risk is material (visible/important surface), or a
  Visual Verdict `FAIL` lands on an acceptance-mapped UI criterion (`browser-evidence.md`).
- **Unresolved spec drift** remains (a known-wrong plan or an open behavior question).
- A **security-critical issue** remains (auth bypass, data exposure, injection).
- A **data migration or destructive change lacks a rollback plan**.
- A **declared project principle is violated** (`.devrites/principles.md`) with no recorded,
  human-approved exception: same standing as an unproven criterion (absent / empty file →
  none declared → not a blocker).
- A **stood decision was never independently doubted**. This means a `## Decisions stood` entry in
  `decisions.md` carries `doubt: MISSING`, or `devrites-engine doubt-coverage` returns rc=3, when that decision
  is irreversible-risk (auth / public-API / migration). Severity rides the undoubted decision,
  not the exit code.
- Any `questions.md` entry with `gate: validating` and `status: open`. **NO-GO
  regardless of behavior impact** (an open validating gate is merge-blocking by
  definition). A slice marked `built (pending review)` is not done.
- A **test was weakened to go green** (`devrites-engine test-integrity` exit 3: a test deleted, skipped,
  `xfail`-ed, or de-asserted since the slice base). A suite that passes on a lowered bar is not
  proof; this is a Critical NO-GO.
- Under `DEVRITES_MUTATION=enforce`, a **mutation score below threshold** (`devrites-engine mutation-gate`
  exit 3: survived mutants are behaviours no test checks).

## GO requires
- Every critical acceptance criterion checked with evidence attached.
- Tests + build green for the scope (or documented, user-accepted exceptions).
- Browser proof present for material UI (or an explicit, accepted manual-only note).
- No open questions/drift that change product behavior.
- Security/data/migration risks either resolved or explicitly accepted by the user with
  a rollback path.

## Important findings (`Critical == 0`, `Important > 0`)
Open `Important` findings do not auto-pass. With acceptance proven and drift resolved, render
the interactive prompt *"`Important > 0` open. Proceed to seal? [y/N]"*: default **N**. `y`
→ GO; otherwise NO-GO with the open Important findings listed as blockers-by-policy. Don't
silently fold Important findings into a GO.

## Conditional GO
If only **non-blocking follow-ups** remain, it's a GO with a recorded follow-up list:
not a NO-GO. Distinguish "must fix to ship" (blocker) from "should do next" (follow-up).

## Honesty
- Don't upgrade a NO-GO to GO to please the user. State blockers plainly.
- Don't claim evidence you don't have. "Unproven" is a valid, important status.
- If subagent reviewers disagree, surface the disagreement and decide explicitly.

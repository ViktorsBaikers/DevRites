# `ship.md` template

The durable record of what shipped and how. Written by `/rite-ship` into the
workspace **before** close-out, so it travels into `.devrites/archive/<slug>/ship.md`.

```markdown
# Ship: <slug>

- Shipped at: <iso>
- Verdict: GO (see seal.md)
- Branch: <branch>
- Commit(s): <sha> [<sha> …]
- Tag / PR: <vX.Y.Z | PR url | none>

## What shipped
<one-paragraph summary of the feature delivered, in the project's vocabulary>

## Acceptance
- Criteria proven: <n / total>   (full walk in evidence.md / seal.md)
- Outstanding (shipped with known follow-ups): <list | none>

## Evidence pointers
- seal.md — final GO/NO-GO verdict + reviewer reconciliation
- evidence.md — acceptance walk; browser-evidence.md (if UI)
- review.md — multi-axis review findings

## Follow-ups (FYI, not blocking)
- <recorded follow-up + where it's tracked> | none
```

Keep it short. It points at the existing audit files rather than restating them.
The acceptance count must match `seal.md`; if they disagree, the seal is the source of
truth and the ship should not have proceeded.

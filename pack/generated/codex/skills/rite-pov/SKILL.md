---
name: rite-pov
description: Project-grounded verdict for adopting, switching, rejecting, or revisiting a named external technology, library, platform, CVE, or pattern.
argument-hint: "[candidate/link/question]"
user-invocable: true
---

# $rite-pov: project-grounded external verdict

Decide whether this project should adopt, trial, hold, reject, or ignore one
named outside candidate.

## Workflow

1. Frame the candidate, decision intent, and reversibility tier (two-way = locally
   reversible; bounded-one-way = reversible via a recorded rollback; high-stakes =
   irreversible or costly). Route an
   open-ended market search to `$rite-pressure-test` or `$rite-spec`.
2. Inspect the live repository for at least one concrete project fact: an
   incumbent dependency/call site, integration seam, relevant ADR/decision, or
   confirmed absence. No profile cache.
3. Verify the external claim from current primary documentation, source,
   advisory, or release notes.
4. Compare fit, existing alternatives, migration cost, reversibility, project
   principles, security, licensing, and deprecation risk.
5. Return exactly one verdict: `Adopt`, `Trial`, `Hold`, `Reject`, or
   `Not-our-problem`.
6. Persist only when asked: use the active feature's `decisions.md` for a
   feature-scoped choice or propose an ADR for a durable architecture decision.
   Do not write a parallel rejection or learning index.

## Output

```text
Verdict: <grade> — <one sentence>
Tier: <two-way | bounded-one-way | high-stakes>
Project evidence: <file:line or local doc>
External evidence: <primary source>
Why: <three bullets max>
Next: <one action | done>
Record: <path | not written>
```

No project evidence or no primary external source means `Hold`, not a generic
technology opinion.

## Non-trigger

Not a market survey or premise stress-test (`$rite-pressure-test` / `$rite-spec`),
not implementation, not a patch plan. A CVE still needs a live advisory URL
before any `Adopt`/`Trial`.

## Failure and recovery

- Missing project evidence or missing live primary source → `Hold`. **Failing
  case:** "Adopt Redis" with no incumbent call site and no opened docs URL.
- High-stakes / irreversible without a named rollback → cannot be `Adopt`.
- Persist only when asked; a chat-only verdict is not a decision record.

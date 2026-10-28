# `seal.md` template

```md
# Seal: <Feature>
Verdict: GO / NO-GO
Candidate SHA-256: <64 lowercase hex>
## Acceptance Criteria
- <AC>: <evidence>
## Verification Evidence
<proof>
## Reviewer Accounts
- devrites-spec-reviewer: <outcome>
- devrites-code-reviewer.<lane>: <outcome>   (one account per engineering lane)
Lane map: <lane → path set, one per lane>
- devrites-test-analyst: <outcome>
- devrites-frontend-reviewer: <outcome>
- devrites-security-auditor: <outcome>
- devrites-performance-reviewer: <outcome>
- devrites-devex-reviewer: <outcome>
- devrites-proof-runner: <outcome>
- devrites-doubt-reviewer: <outcome, or Not-applicable: no decision stood>
## Risks / Rollback
<risks/recovery>
## Stood Decisions
<doubt verdict per stood decision>
## Blockers / Follow-ups
Blockers: <none | - blocker>
Follow-ups:
- <non-blocking follow-up>
## Final Decision
<verdict/reason>
```

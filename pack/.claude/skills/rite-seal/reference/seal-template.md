# `seal.md` template

Loaded on demand by `/rite-seal`. The seal writes this template (filled in) to `.devrites/work/<slug>/seal.md` as the durable record of the GO / NO-GO verdict.

```markdown
# Seal: <Feature>

Verdict: GO / NO-GO

## Acceptance Criteria
- [ ] <criterion> — evidence: <...>

## Verification Evidence
<tests / build / lint summary>

## Browser Evidence
<summary | n/a>

## Risks
<ranked>

## Blockers
<must-fix before ship>

## Non-blocking Follow-ups
<deferred items>

## Rollback / Recovery
<how to back this out>

## Final Decision
<one paragraph: verdict + why>

## Footprint
<deterministic fan-out from footprint.sh — subagents · slices · wall-clock; never tokens/cost>
```

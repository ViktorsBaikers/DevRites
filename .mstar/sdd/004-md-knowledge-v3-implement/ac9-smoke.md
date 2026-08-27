# AC-9 smoke scenarios — 004-md-knowledge-v3-implement

## Scenario 1 — rite-quick vs rite-build

| Field | Value |
| --- | --- |
| Owner | intent-map.md |
| Trigger | New REQ/AC multi-file specced slice |
| Expected | /rite-build |
| Failing case | Multi-slice routed to rite-quick |

## Scenario 2 — Reviewer independence

| Field | Value |
| --- | --- |
| Owner | devrites-code-reviewer.md (+ T22–T25) |
| Trigger | Orchestrator passes implementer narrative |
| Expected | Independence block; Finding: file:line schema |
| Failing case | Reviewer trusts implementer verdict |

## Scenario 3 — Proof fail-closed

| Field | Value |
| --- | --- |
| Owner | acceptance-proof.md, rite-prove/SKILL.md |
| Trigger | Green tests; silent error path |
| Expected | cannot_verify; seal blocked |
| Failing case | Narrative proof advances seal |

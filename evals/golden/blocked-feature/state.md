# State: add-bulk-delete

- Phase: seal
- Status: awaiting_human
- Active slice: 3 — soft-delete retention
- Slice mode: HITL
- Spec gate: passed 2026-06-12T09:00:00Z
- Plan approved: 2026-06-12T10:00:00Z
- AFK slices remaining: none
- Risk: destructive deletion without a proven recovery path
- Next step: /rite-resolve q-2026-06-13-002 "<answer>"

## Awaiting human
- qid: q-2026-06-13-002
- gate: validating
- question: Is 30-day soft-delete retention confirmed, or should it be configurable?
- proposed: fixed 30 days
- raised_at: 2026-06-13T14:00:00Z
- blocking_slices: [3]

## Slice progress
- [x] Slice 1: selection UI — built
- [x] Slice 2: bulk-delete endpoint — built
- [ ] Slice 3: soft-delete retention — pending

## Log
- 2026-06-13 seal: NO-GO — recovery path unproven, validating gate open

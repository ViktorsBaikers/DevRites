# Seal: add-bulk-delete

Verdict: NO-GO

## Acceptance Criteria
- [x] [AC1] Admin can bulk-delete N records — evidence: bulk-delete.test.ts (pass)
- [x] [AC2] Server-side admin authorization — evidence: 403 non-admin test (pass)
- [ ] [AC3] Deleted batch recoverable for 30 days — UNPROVEN: recovery test fails (NotImplemented)

## Verification Evidence
2/3 tests pass; the recovery test fails. Slice 3 (soft-delete retention) is still pending.

## Browser Evidence
Selection UI verified manually; recovery flow not buildable yet.

## Risks
HIGH — a destructive, irreversible bulk delete with no recovery path.

## Blockers
- Hard delete ships without the soft-delete recovery path (review.md Critical).
- Acceptance criterion "recoverable for 30 days" is unproven (test fails).
- Open validating gate q-2026-06-13-002 (retention policy) unresolved.

## Non-blocking Follow-ups
- Audit log of bulk deletions (review.md Important).

## Rollback / Recovery
None for already-deleted rows — that is precisely the blocker.

## Final Decision
NO-GO. A destructive operation cannot ship before its recovery path exists and is proven,
and an open validating gate is merge-blocking by definition.

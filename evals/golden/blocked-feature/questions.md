# Questions: add-bulk-delete

## q-2026-06-13-001
status: answered
slice: 02-bulk-delete-endpoint
gate: blocking
question: Confirm hard delete is acceptable for the first slice?
proposed: no — require soft delete first
raised_at: 2026-06-13T11:00:00Z
answered_at: 2026-06-13T11:30:00Z
answer: Soft delete is required before ship.

## q-2026-06-13-002
status: open
slice: 03-soft-delete-retention
gate: validating
question: Is 30-day soft-delete retention confirmed, or should it be configurable?
proposed: fixed 30 days
raised_at: 2026-06-13T14:00:00Z
answered_at:
answer:

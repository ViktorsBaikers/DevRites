# Evidence: add-bulk-delete

Captured: 2026-06-13T13:00:00Z

## Tests
```
$ npm test -- admin/bulk-delete
PASS  ✓ deletes a batch of selected records (admin) (40 ms)
PASS  ✓ rejects bulk delete for non-admin with 403 (11 ms)
FAIL  ✗ a deleted batch is recoverable for 30 days
      Expected restore() to return the batch; got NotImplemented.
Tests: 2 passed, 1 failed, 3 total
```

## Acceptance mapping
- bulk delete N records → pass
- admin-only authorization → pass
- 30-day recovery (soft delete) → FAIL (not implemented; slice 3 pending)

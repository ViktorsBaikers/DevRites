# Spec: add-bulk-delete

## Problem
Admins need to delete many records at once instead of one by one.

## Acceptance criteria
- [ ] [AC1] An admin can select N records and delete them in one action.
- [ ] [AC2] Deletion is authorized server-side (admin role required).
- [ ] [AC3] A deleted batch is recoverable for 30 days (soft delete).

## Non-goals
- Bulk edit (only delete).

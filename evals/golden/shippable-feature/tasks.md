# Tasks: add-csv-export

## Slice 1 — stream endpoint (built)
- Goal: `GET /transactions/export.csv` streams CSV rows.
- Acceptance: endpoint returns 200 with `text/csv`; rows stream, not buffered.
- Mode: HITL · Gate: validating

## Slice 2 — auth scoping (built)
- Goal: export only the authenticated user's rows.
- Acceptance: a user cannot export another user's transactions (no IDOR).
- Mode: HITL · Gate: blocking

# Tasks: add-csv-export

## Slice index
| Slice ID | Goal | AC IDs | Mode | Gate | Done |
| --- | --- | --- | --- | --- | --- |
| SLICE-001 | Stream CSV with bounded memory | AC-001, AC-003 | AFK | advisory | built |
| SLICE-002 | Enforce caller scoping | AC-002 | AFK | advisory | built |

## SLICE-001 Stream CSV with bounded memory
Goal: Stream the authenticated user's transactions as CSV.
Satisfies: AC-001, AC-003
Files likely touched: src/routes/transactions/export.ts, src/lib/csv-stream.ts
Tests/proof: EVID-001, EVID-003
Mode: AFK
Gate: advisory
Dependencies: none
Status: built
Done condition: AC-001 and AC-003 pass.

## SLICE-002 Enforce caller scoping
Goal: Reject attempts to export another user's rows.
Satisfies: AC-002
Files likely touched: src/routes/transactions/export.ts
Tests/proof: EVID-002
Mode: AFK
Gate: advisory
Dependencies: SLICE-001
Status: built
Done condition: AC-002 passes.

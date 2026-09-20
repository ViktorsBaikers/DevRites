# Brief: add CSV export

## Objective
Authenticated users need a safe way to download their own transactions as CSV.
The export must stream rows so large accounts do not require loading the full
result set into memory.

## Non-goals
- No XLSX, PDF, scheduled, or emailed exports.

## Success definition
Every acceptance criterion has fresh executable evidence, and the final review
and seal contain no blocker.

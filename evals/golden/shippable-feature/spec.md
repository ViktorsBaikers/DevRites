# Spec: add-csv-export

## Problem
Users can view their transactions in the UI but cannot get them out. Support
keeps fielding "can I download my data" requests.

## Acceptance criteria
- [ ] [AC1] A `GET /transactions/export.csv` endpoint streams the caller's transactions as CSV.
- [ ] [AC2] The export is scoped to the authenticated user (no IDOR).
- [ ] [AC3] Rows over a 100k-row export do not load the whole result set into memory.

## Non-goals
- XLSX / PDF export.
- Scheduled / emailed exports.

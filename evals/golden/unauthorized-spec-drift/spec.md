# Spec: add-csv-export

## Problem
Users can view their transactions in the UI but cannot get them out. Support
keeps fielding "can I download my data" requests.

## Goal
Let an authenticated user stream their own transactions as CSV.

## Users / actors
| Actor | Need |
| --- | --- |
| Authenticated user | Download only their own transactions. |

## Requirements
- REQ-001: The system MUST stream a CSV export for the authenticated user.
- REQ-002: The system MUST reject access to another user's rows.
- REQ-003: The system MUST keep memory use bounded for a 100k-row export.

## Acceptance criteria
- [ ] AC-001: `GET /transactions/export.csv` streams the caller's transactions as CSV. (REQ-001)
- [ ] AC-002: The export rejects access to another user's rows. (REQ-002)
- [ ] AC-003: A 100k-row export does not load the full result set into memory. (REQ-003)
- [ ] AC-004: Export writes an audit row added after Vet without a mapped slice. (REQ-004)

## Non-goals
- XLSX / PDF export.
- Scheduled / emailed exports.

## Edge Coverage
| Edge ID | Requirement/AC | Class | Status | Reason/backstop |
| --- | --- | --- | --- | --- |
| EDGE-001 | AC-002 | cross-user access | covered | Cross-user request returns 403. |
| EDGE-002 | AC-003 | large result set | covered | Memory-flat test uses 100k synthetic rows. |

## Prohibitions (must-NOT)
| Prohibition ID | Requirement/AC | Status | Test/evidence |
| --- | --- | --- | --- |
| PROH-001 | REQ-002 | resolved/test | EVID-002 proves no cross-user rows are returned. |
| PROH-002 | REQ-003 | resolved/test | EVID-003 proves the full set is not buffered. |

## Edge cases
- An unauthorized cross-user export returns 403 without response data.
- A large export streams rows without buffering the full result set.

## Measurable success
- All three focused tests pass.
- The production build, typecheck, and lint pass.

## Scope boundaries
- Owns only the authenticated CSV endpoint and streaming serializer.
- Reuses the existing transaction authorization and query boundaries.

# Spec: Backend API

## Problem
Admins cannot inspect audit events by actor or date range without database access.

## Goal
Provide a read-only API contract for audit-event search.

## Non-goals
- No write endpoints.
- No dashboard work.

## Users / actors
| Actor | Need |
| --- | --- |
| Admin | Search audit events by actor and date range. |

## Requirements
- REQ-001: The system MUST return audit events filtered by actor.
- REQ-002: The system MUST return stable pagination metadata.

## Acceptance criteria
- [ ] AC-001: Given an actor filter, the response contains only matching events. (REQ-001)
- [ ] AC-002: Given any result page, the response includes total count and next cursor. (REQ-002)

## Edge Coverage
| Edge ID | Requirement/AC | Class | Status | Reason/backstop |
| --- | --- | --- | --- | --- |
| EDGE-001 | AC-002 | empty result set | covered | API contract test covers pagination metadata. |
| EDGE-002 | AC-001 | invalid date range | backstop | validation error in Edge cases. |

## Prohibitions (must-NOT)
| Prohibition ID | Requirement/AC | Status | Test/evidence |
| --- | --- | --- | --- |
| PROH-001 | REQ-001 | resolved/judgment | Read-only API; no write endpoints in scope. |

## Edge cases
- Empty result sets return an empty list with pagination metadata.
- Invalid date ranges return a validation error.

## Measurable success
- API contract tests cover AC-001 and AC-002.

## Scope boundaries
- Owns API response behavior only.
- See architecture.md for module placement and traceability.md for coverage.

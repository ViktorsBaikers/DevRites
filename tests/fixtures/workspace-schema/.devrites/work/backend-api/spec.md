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

## Edge cases
- Empty result sets return an empty list with pagination metadata.
- Invalid date ranges return a validation error.

## Measurable success
- API contract tests cover AC-001 and AC-002.

## Scope boundaries
- Owns API response behavior only.
- See architecture.md for module placement and traceability.md for coverage.

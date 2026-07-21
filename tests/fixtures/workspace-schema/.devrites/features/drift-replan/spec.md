# Spec: Drift Replan

## Problem
The planned import hook no longer exists in the current codebase.

## Goal
Import submitted files through the existing ingestion service.

## Non-goals
- No new file format.

## Users / actors
| Actor | Need |
| --- | --- |
| Operator | Submit a file and see ingestion status. |

## Requirements
- REQ-001: The system MUST accept a submitted file through the ingestion service.
- REQ-002: The system MUST surface ingestion failures to the operator.

## Acceptance criteria
- [ ] AC-001: Submitted files enter the ingestion service queue. (REQ-001)
- [ ] AC-002: Ingestion failures show an actionable error. (REQ-002)

## Edge Coverage
| Edge ID | Requirement/AC | Class | Status | Reason/backstop |
| --- | --- | --- | --- | --- |
| EDGE-001 | AC-002 | unsupported file | covered | AC-002 covers actionable failure. |

## Prohibitions (must-NOT)
| Prohibition ID | Requirement/AC | Status | Test/evidence |
| --- | --- | --- | --- |
| PROH-001 | REQ-001 | dismissed | no bespoke prohibition |

## Edge cases
- Unsupported files fail before queueing.

## Measurable success
- Traceability maps both acceptance criteria after repair.

## Scope boundaries
- Owns import submission path only.

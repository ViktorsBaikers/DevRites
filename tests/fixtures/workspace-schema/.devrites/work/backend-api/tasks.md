# Tasks

## Slice index
| Slice ID | Goal | AC IDs | Mode | Gate | Done |
| --- | --- | --- | --- | --- | --- |
| SLICE-001 | Actor filter API | AC-001 | AFK | advisory | built |
| SLICE-002 | Pagination metadata | AC-002 | AFK | advisory | built |

## SLICE-001 Actor filter API
Goal: Add the filtered read endpoint.
Satisfies: AC-001
Forge: no
Forge strategies: none
Forge scorecard: none
Files likely touched: app/controllers/admin/audit_events_controller.rb, app/queries/audit_events_query.rb
Tests/proof: EVID-001
Mode: AFK
Gate: advisory
Dependencies: none
Status: built
Done condition: AC-001 passes.

## SLICE-002 Pagination metadata
Goal: Add stable total and next-cursor metadata.
Satisfies: AC-002
Forge: no
Forge strategies: none
Forge scorecard: none
Files likely touched: app/serializers/audit_event_page_serializer.rb
Tests/proof: EVID-002
Mode: AFK
Gate: advisory
Dependencies: SLICE-001
Status: built
Done condition: AC-002 passes.

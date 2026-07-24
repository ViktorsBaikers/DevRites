# Tasks

## Slice index
| Slice ID | Goal | AC IDs | Mode | Gate | Done |
| --- | --- | --- | --- | --- | --- |
| SLICE-001 | Queue submitted files | AC-001 | AFK | advisory | pending |
| SLICE-002 | Surface ingestion failures | AC-002 | AFK | advisory | pending |

## SLICE-001 Queue submitted files
Goal: Route submission into the ingestion service.
Satisfies: AC-001
Forge: no
Forge strategies: none
Forge scorecard: none
Files likely touched: app/controllers/imports_controller.rb, app/services/ingestion_adapter.rb
Tests/proof: pending
Mode: AFK
Gate: advisory
Dependencies: none
Status: pending
Done condition: AC-001 passes.

## SLICE-002 Surface ingestion failures
Goal: Translate ingestion errors into actionable messages.
Satisfies: AC-002
Forge: no
Forge strategies: none
Forge scorecard: none
Files likely touched: app/presenters/import_status_presenter.rb
Tests/proof: pending
Mode: AFK
Gate: advisory
Dependencies: SLICE-001
Status: pending
Done condition: AC-002 passes.

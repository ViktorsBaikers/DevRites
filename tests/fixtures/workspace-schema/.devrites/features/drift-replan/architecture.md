# Architecture

## Owning module / layer
Import submission controller and ingestion service adapter.

## Integration points
Controller calls the existing ingestion service instead of the removed legacy hook.

## Data / API / events
Submitted files become ingestion jobs.

## Dependencies
Existing ingestion service and job-status presenter.

## Risks
Error translation could hide actionable failures.

## Affected boundaries
Import UI-to-service boundary.

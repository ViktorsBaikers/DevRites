# Architecture

## Owning module / layer
Settings page component and preferences API client.

## Integration points
Settings route loads current preference and calls the update endpoint.

## Data / API / events
Reads and writes digest preference through the existing preferences API.

## Dependencies
Existing form controls, toast component, and API client.

## Risks
Optimistic UI could hide a failed save.

## Affected boundaries
Settings UI boundary and user-preferences API boundary.

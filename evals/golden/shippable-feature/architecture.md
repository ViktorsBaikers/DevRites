# Architecture

## Owning module / layer
The authenticated transactions route owns authorization and query scoping.

## Integration points
The route calls the existing transaction query and streams rows through the CSV
serializer.

## Data / API / events
`GET /transactions/export.csv` returns `text/csv`; no data model or event changes.

## Dependencies
Existing route authentication, transaction query, and standard CSV primitives.

## Risks
Duplicated scoping could disclose another user's rows; buffering could exhaust
memory on large accounts.

## Affected boundaries
Transactions HTTP boundary and the row-streaming serialization boundary.

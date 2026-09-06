# Touched files: add-csv-export

## Touched files
Candidate paths are declared only in the manifest below.

## Candidate manifest
| State | File | Slice | Reason |
| --- | --- | --- | --- |
| present | `src/lib/csv-stream.ts` | SLICE-001 | Row-by-row serializer. |
| present | `src/routes/transactions/export.test.ts` | SLICE-001, SLICE-002 | Endpoint and authorization proof. |
| present | `src/routes/transactions/export.ts` | SLICE-001, SLICE-002 | Streaming endpoint and caller scope. |
| present | `src/utils/format.ts` | SLICE-001 | Drive-by cleanup outside the slice contract. |

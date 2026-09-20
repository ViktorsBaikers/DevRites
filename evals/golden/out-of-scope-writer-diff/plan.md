# Plan

## Approach
Reuse the authenticated transactions route and add the smallest streaming CSV
serializer.

## Slice strategy
SLICE-001 adds and proves the stream; SLICE-002 proves caller scoping.

## Validation strategy
Run focused endpoint, cross-user, and 100k-row memory tests before build,
typecheck, and lint.

## Rollback
Remove the route registration and serializer. No data migration is involved.

# Plan

## Approach
Extend the existing read path and keep filtering in the query object.

## Slice strategy
Build API filtering first, then pagination metadata.

## Validation strategy
Run focused API contract tests for AC-001 and AC-002.

## Rollback
Remove the route registration and query filter additions.

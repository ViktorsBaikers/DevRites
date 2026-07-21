# Plan

## Approach
Repair the plan to target the ingestion service adapter.

## Slice strategy
Queue successful submissions first, then translate failures.

## Validation strategy
Service adapter tests cover queue and failure paths.

## Rollback
Revert controller routing to the previous submission handler.

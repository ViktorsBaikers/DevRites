# Plan

## Approach
Reuse the existing settings form pattern and API client.

## Slice strategy
First render state, then persist and prove browser behavior.

## Validation strategy
Component tests plus Playwright smoke for key viewports.

## Rollback
Remove the settings panel registration and preference client call.

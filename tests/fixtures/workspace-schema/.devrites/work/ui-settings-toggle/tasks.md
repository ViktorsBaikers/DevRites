# Tasks

## Slice index
| Slice ID | Goal | AC IDs | Mode | Gate | Done |
| --- | --- | --- | --- | --- | --- |
| SLICE-001 | Render current preference | AC-001 | AFK | advisory | built |
| SLICE-002 | Save changed preference | AC-002 | AFK | advisory | built |

## SLICE-001 Render current preference
Goal: Show current digest state in settings.
Satisfies: AC-001
Forge: no
Forge strategies: none
Forge scorecard: none
Files likely touched: app/settings/DigestToggle.tsx
Tests/proof: EVID-001
Mode: AFK
Gate: advisory
Dependencies: none
Status: built
Done condition: AC-001 passes.

## SLICE-002 Save changed preference
Goal: Persist toggles and expose success/error states.
Satisfies: AC-002
Forge: no
Forge strategies: none
Forge scorecard: none
Files likely touched: app/settings/DigestToggle.tsx, app/api/preferences.ts
Tests/proof: EVID-002, EVID-003
Mode: AFK
Gate: advisory
Dependencies: SLICE-001
Status: built
Done condition: AC-002 passes.

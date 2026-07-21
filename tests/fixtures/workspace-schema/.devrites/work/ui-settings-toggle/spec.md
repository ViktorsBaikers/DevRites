# Spec: UI Settings Toggle

## Problem
Users must contact support to change digest preferences.

## Goal
Expose a settings toggle with clear saved and error states.

## Non-goals
- No new notification categories.
- No mobile app changes.

## Users / actors
| Actor | Need |
| --- | --- |
| Signed-in user | Turn digest emails on or off. |

## Requirements
- REQ-001: The system MUST show the current digest preference.
- REQ-002: The system MUST persist a changed digest preference.

## Acceptance criteria
- [ ] AC-001: The settings page shows the current digest preference. (REQ-001)
- [ ] AC-002: Toggling the control saves the new preference and announces success. (REQ-002)

## Edge Coverage
| Edge ID | Requirement/AC | Class | Status | Reason/backstop |
| --- | --- | --- | --- | --- |
| EDGE-001 | AC-002 | save failure | covered | AC-002 covers success; error state covered by Edge cases. |

## Prohibitions (must-NOT)
| Prohibition ID | Requirement/AC | Status | Test/evidence |
| --- | --- | --- | --- |
| PROH-001 | REQ-002 | resolved/judgment | Previous state remains visible on failure. |

## Edge cases
- Save failure leaves the previous state visible and announces an error.

## Measurable success
- Browser proof covers 375px and 1280px plus keyboard interaction.

## Scope boundaries
- Owns settings-page preference UI only.
- See design-brief.md for UI direction and browser-evidence.md for visual proof.

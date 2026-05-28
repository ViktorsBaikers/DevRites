# Nielsen's 10 usability heuristics — DevRites scoring rubric

A structured UX rubric for `/rite-review` when the feature touches UI. Score each
heuristic on a **0–4** scale; the per-heuristic scores feed the UX axis of the
multi-axis review and shape the severity labels on UX findings.

**Score honestly.** 4 means *genuinely excellent* — not "good enough", not "no
findings". 3 is "good with minor gaps". 2 is "partial — significant gaps". 0–1 is a
Critical / Important.

Heuristics are Jakob Nielsen's 10 (1994, refreshed 2020) — public usability canon,
referenced by every serious design discipline.

## 1. Visibility of system status
Users know what's happening, when, and where.
- Loading indicators on async operations.
- Confirmation when actions complete (save, submit, delete).
- Progress on multi-step flows.
- Current location in nav (active state, breadcrumbs).
- Inline form validation, not just on submit.

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| No feedback. | Critical actions lack feedback. | Most actions have feedback; gaps. | Consistent feedback; minor lag/lateness. | Anticipates uncertainty; reassures throughout. |

## 2. Match between system and the real world
The UI speaks the user's language and follows real-world conventions.
- Domain vocabulary the *user* shares — not the team's internal names.
- Familiar metaphors where they help (cart, inbox, draft).
- Information ordered the way the user thinks about it, not the way the DB stores it.

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| Jargon throughout. | Mostly internal vocabulary. | Mixed — some user terms, some internal. | Clear user terms; a couple of leaks. | Reads like the user's own words. |

## 3. User control and freedom
Easy escape, undo, and back-out from any state.
- Cancel / close / back on every modal, route, and form.
- Undo on destructive operations (delete, archive, reset).
- No dead-end states ("Error — refresh the page" with no recovery).

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| Users trapped in states. | Limited escape on key flows. | Most flows reversible; gaps on destructive ops. | Good escape + undo on most flows. | Confident exploration possible everywhere. |

## 4. Consistency and standards
Internal consistency (this feature matches the project) + external consistency (the
project matches platform / web norms).
- Same concept named the same way across screens.
- Same affordance (button vs link vs row) means the same thing across screens.
- Platform conventions respected (browser back, focus rings, native controls where appropriate).

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| Inconsistent vocabulary + affordances. | Major drift from neighbours. | Mostly aligned; visible drift. | Well-aligned; minor inconsistencies. | Indistinguishable from the rest of the product. |

## 5. Error prevention
Designed so the user *can't* make the error in the first place.
- Confirmations on destructive operations.
- Disabled / hidden actions when they're not valid.
- Smart defaults; constrained inputs (date pickers vs free text).
- Field formats validated as you type, not just on submit.

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| Many error-prone flows. | Some preventable errors still possible. | Common errors prevented; edge cases slip. | Most errors prevented before they occur. | Errors are genuinely rare; the easy path is the safe path. |

## 6. Recognition rather than recall
Show options; don't force the user to remember them.
- Visible choices rather than free-text "type the command".
- Recently-used values surfaced.
- Labels persist (don't disappear after focus).
- Context (where the user is, what they were doing) survives navigation.

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| Heavy recall required. | Critical flows demand recall. | Mixed — some visible, some hidden. | Mostly recognition; minor recall asks. | Pure recognition; nothing the user needs is hidden. |

## 7. Flexibility and efficiency of use
Novices and experts both productive.
- Keyboard shortcuts on power flows; visible affordances for everyone else.
- Sensible defaults for the common case; depth available when needed.
- Saved views / templates / bulk actions on tasks done repeatedly.

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| One-size-fits-all; no acceleration. | Limited shortcuts; novice-only. | Some efficiency for experts; uneven. | Strong shortcut + default story; small gaps. | Both populations measurably fast. |

## 8. Aesthetic and minimalist design
Every element earns its place. No decorative noise.
- One primary action per screen / surface.
- Visual hierarchy carries the eye to the next step.
- Decoration in service of meaning, not for its own sake.
- Whitespace used as a tool, not because the page felt empty.

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| Cluttered; no hierarchy. | Visible noise / competing actions. | Generally clean; some clutter. | Strong hierarchy; minor noise. | Every pixel deliberate; nothing extraneous. |

## 9. Help users recognize, diagnose, and recover from errors
Error states are designed, not afterthoughts.
- Errors named in user language ("Your card was declined") not codes ("Stripe 4002").
- A concrete next step ("Try a different card" + "Use PayPal instead").
- Inline location — the error appears next to the field / action that caused it.
- No blame — the message describes the situation, not the user.

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| Cryptic codes / silent failures. | Errors visible but unhelpful. | Some errors recoverable; others stranded. | Most errors actionable; minor gaps. | Every error has a clear cause + clear next step. |

## 10. Help and documentation
Users can find what they need without leaving the surface.
- Inline hints on novel patterns.
- Search-friendly help for deeper questions.
- No help is needed for the common path — and that's documented as a goal.

| 0 | 1 | 2 | 3 | 4 |
|---|---|---|---|---|
| No in-product help. | Help exists but hidden / stale. | Help exists; quality uneven. | Good inline help; reachable docs. | Help feels native; users barely need it. |

## Reporting

In `/rite-review` output, surface only the heuristics scoring **≤2** as findings,
plus any 3 with a specific noted gap. Heuristics at 4 are not reported individually
— they roll up into the UX axis. Heuristics at 3 with no specific gap are not
reported. Each surfaced finding gets a severity label per `rite-review/SKILL.md`.

The rubric is descriptive, not gating. A score of 2 on aesthetic-minimalist isn't
an automatic NO-GO; the `rite-seal` gate is `Critical == 0`, not "every heuristic
≥ 3".

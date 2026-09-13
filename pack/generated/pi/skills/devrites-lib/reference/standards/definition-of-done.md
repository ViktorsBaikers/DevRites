# Definition of Done

Acceptance criteria prove the requested behavior; this bar proves the change is
shippable. Phase owners keep the checks — this file is the index, not a second list.

A feature is not done when any required row is missing. Open the owner; do not
re-judge from this table alone.

| Required | Owner | Failing case |
| --- | --- | --- |
| Each AC names a stored evidence path (command or artifact) | [`testing.md`](testing.md), `rite-prove` | An AC is checked off with no path in `evidence.md` |
| Discriminating tests for changed behavior | [`testing.md`](testing.md) | Suite green; revert of the change would still be green |
| Fresh runtime/build/test/browser records | `rite-prove`, [`browser-proof-checklist.md`](browser-proof-checklist.md) | Proof claimed from stale or unrun commands |
| Scope-clean diff | [`code-review.md`](code-review.md) | Unscoped edits or a hidden second system remain |
| Open hard gates closed | [`afk-hitl.md`](afk-hitl.md), `rite-seal` | HITL, red tests, drift, or NO-GO still open |
| Rollback named for risky work | [`data-integrity.md`](data-integrity.md) | "revert if needed" with no trigger/procedure/proof |
| Docs/ADR for public behavior | [`documentation.md`](documentation.md) | Command or contract changed; docs did not |

Use from `rite-vet`, `rite-build`, `rite-prove`, `rite-seal`, and `rite-quick`.

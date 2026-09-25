# Definition of Done

> Applies when: declaring any task, phase, or slice complete.

Acceptance criteria prove the requested behavior; this bar proves the change is
shippable. Phase owners keep the checks — this file is the index, not a second list.

A feature is not done when any required row is missing. Open the owner; do not
re-judge from this table alone.

| Required | Owner | Failing case |
| --- | --- | --- |
| Each AC names a stored evidence path (command or artifact) | [`testing.md`](testing.md), `rite-prove` | An AC is checked off with no path in `evidence.md` |
| Every explicit request (each `brief.md` `## Asks` row) maps to an artifact, test, or observed command | `rite-prove`, `rite-seal` | Numbered asks counted done by proxy — green suite, manifest presence, effort spent — with no per-requirement evidence map |
| Runnable required checks ran to a recorded outcome | [`test-proof-checklist.md`](test-proof-checklist.md), [`debug-recovery.md`](debug-recovery.md) | A failed, timed-out, or unstartable check reported as "unrelated"/omitted instead of a named blocker |
| Discriminating tests for changed behavior | [`testing.md`](testing.md) | Suite green; revert of the change would still be green |
| Fresh runtime/build/test/browser records | `rite-prove`, [`browser-proof-checklist.md`](browser-proof-checklist.md) | Proof claimed from stale or unrun commands |
| Scope-clean diff | [`code-review.md`](code-review.md) | Unscoped edits or a hidden second system remain |
| Open hard gates closed | [`afk-hitl.md`](afk-hitl.md), `rite-seal` | HITL, red tests, drift, or NO-GO still open |
| Rollback named for risky work | [`data-integrity.md`](data-integrity.md) | "revert if needed" with no trigger/procedure/proof |
| Docs/ADR for public behavior | [`documentation.md`](documentation.md) | Command or contract changed; docs did not |
| Written artifacts re-read from disk before handoff | [`workspace-artifact-schema.md`](../workspace-artifact-schema.md), phase owner | Report describes a spec/plan/evidence state the file on disk does not contain |
| Recipient-path execution | [`verification-methods.md`](verification-methods.md), `rite-prove` | Tests run against the working tree; the packaged artifact was never installed, opened, or navigated as its recipient would |

Use from `rite-vet`, `rite-build`, `rite-prove`, `rite-seal`, and `rite-quick`.

# DevRites completion reply contract

Use the host's normal response. Do not run a progress renderer, emit session
telemetry, assign a health score, or repeat state already visible in the durable
artifact.

Keep the reply compact and evidence-backed:

When an active rite is the controlling caller, Intermediate `NEEDS_REPLAN`, a
nested phase `STOP`, and a routine Plan/Vet `Next step` are not eligible completion
states. Return them to the caller without a user-facing reply. Autocomplete may
use the shapes below only after its requested rest point or a shared genuine
human/safety/access/exhausted-recovery stop is reached.

```text
Done: <result in one sentence>
Changed: <artifact or source paths>
Evidence: <commands and observed outcomes, or not applicable>
Open: <none | unresolved items>
Next: <one recommended action>
Record: <primary durable artifact path>
```

Pre-send check: delete an opening sentence that only announces ("I'll now…") and a
closing sentence that only recaps; then apply the two-line test — a reader seeing only
the first and last lines must know what happened and what to do next.

If a required decision, proof, or invariant is missing, use one of these states
instead of `Done`:

```text
Awaiting human: <question id and gate>
Question: <exact decision needed>
Recommended: <one option and reason>
Resume: $rite-resolve <qid> "<answer>"
Record: .devrites/work/<slug>/questions.md
```

```text
Stopped: <reason>
Blocking: <specific invariant>
Fix: <one action>
Record: <artifact path>
```

```text
NO-GO: <verdict>
Blockers: <top blockers>
Fix: <one action>
Record: .devrites/work/<slug>/seal.md
```

```text
GO: feature cleared to ship
Follow-ups: <none | count>
Next: $rite-ship
Record: .devrites/work/<slug>/seal.md
```

```text
Shipped: <feature>
Commit: <sha and branch>
Acceptance: <proven count>
Archived: .devrites/archive/<slug>/
Record: .devrites/archive/<slug>/ship.md
```

Claims such as proved, reviewed, sealed, shipped, or complete must point to real
output or an artifact. Use exactly one recommended next action except for
terminal agent-owned technical exhaustion, which has no runnable action.

Use that terminal case only per [`one-shot-actions.md`](standards/one-shot-actions.md):
three recorded no-progress corrections of the exact fingerprint, or required evidence
irretrievably absent with **no safe in-scope diagnostic-amplification seam**. A spent
consumptive-action authorization plus a retained new fingerprint is not terminal. For a
true terminal case use:

```text
Stopped: Technical recovery exhausted
Blocking: <causal fingerprint and invariant>
Attempts: <three failed approaches and decisive reproduction>
No runnable recovery command: unchanged reinvocation remains blocked
Next: none — requires new evidence or changed failure conditions
Record: <artifact path>
```

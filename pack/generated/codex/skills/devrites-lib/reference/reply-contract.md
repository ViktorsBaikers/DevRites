# DevRites completion reply contract

Use the host's normal response. Do not run a progress renderer, emit session
telemetry, assign a health score, or repeat state already visible in the durable
artifact.

Keep the reply compact and evidence-backed:

```text
Done: <result in one sentence>
Changed: <artifact or source paths>
Evidence: <commands and observed outcomes, or not applicable>
Open: <none | unresolved items>
Next: <one recommended action>
Record: <primary durable artifact path>
```

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

Use that terminal case only after three recorded no-progress corrections of the
exact fingerprint, or when required evidence was irretrievably absent **and** no safe in-scope diagnostic-amplification seam exists. Loss from the
past attempt alone is not terminal: repair and Vet a boundary-discriminating
evidence design, then request fresh authorization for its acquisition attempt. A spent
consumptive-action authorization plus a retained new fingerprint is not terminal;
it continues offline recovery and waits for fresh authorization only after repair.
For a true terminal case use:

```text
Stopped: Technical recovery exhausted
Blocking: <causal fingerprint and invariant>
Attempts: <three failed approaches and decisive reproduction>
No runnable recovery command: unchanged reinvocation remains blocked
Next: none — requires new evidence or changed failure conditions
Record: <artifact path>
```

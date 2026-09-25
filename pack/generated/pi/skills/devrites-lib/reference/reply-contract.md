# DevRites completion reply contract

Use the host's normal response. Do not run a progress renderer, emit session
telemetry, assign a health score, or repeat state already visible in the durable
artifact.

Keep the reply compact and evidence-backed:

When an active rite is the controlling caller, Intermediate `NEEDS_REPLAN` (the reply
token for the workspace's `NEEDS REPLAN` readiness value), a nested phase `STOP`, and a
routine Plan/Vet `Next step` are not eligible completion states. Autocomplete may
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

`Changed:` is the complete write set, not a highlight reel: every file written
or edited this turn — workspace artifact and project source alike — appears
there or inside one named grouped entry, so a reviewer can `git diff` exactly
those paths. `Record:` then names the single durable artifact a later session
reads first.

Mid-flight replies — the turns between completion states — obey the same economy:

- **Restate position.** The reader cannot hold "step 3 of 5" between turns; name it
  (`step N of M` / `state: <where things stand>`) rather than asking them to remember.
- **One next action, doable now.** `Next:` names one concrete action completable in
  under two minutes — a command, a file to open, a yes/no decision. "Want me to
  continue?" is a stall, not an action.
- **Numbered steps for multi-step work**, one bounded action each; fold trivial steps
  into the one before. A short path finished beats a complete path abandoned.
- **Name the win before the remainder.** A landed slice, a green gate, a merged
  piece is stated first — visible progress beats an undifferentiated list.
- **Cap enumerations.** Long lists collapse to at most 5 items per group plus
  `k of N; rest: <path>`, where N equals the count in that durable artifact; the full
  enumeration lives there, not the reply. **Failing case:** the reply says "3 open
  findings" while `review.md` lists 7 and no path is named.
- **Errors are matter-of-fact** — cause and fix, never alarm noise.
- **Paste-safe commands.** A command on a `Next:`/`Resume:`/`Fix:` line has no trailing
  `#` comment, uses paths that resolve from the user's working directory (project-relative
  or absolute, never a sandbox/temp path), and chains dependent steps with `&&`.
  **Failing case:** `Next: devrites-engine handoff billing  # resume` errors on first paste
  in zsh, which passes the comment as arguments.

Pre-send check: delete an opening sentence that only announces ("I'll now…") and a
closing sentence that only recaps; then apply the two-line test — a reader seeing only
the first line (its state token: `Done`, `Awaiting human`, `Stopped`, `NO-GO`, `GO`,
`Shipped`, or the lead label a utility rite's own output template defines, such as
`Verdict:` or `Feature:`) and the action line (`Next:`, or `Resume:`/`Fix:` in a stop shape) must know
what happened and what to do next. Prose before the state token fails the test.

The reply has one next action. A `↻ Hygiene:` footer
([`context-hygiene.md`](standards/context-hygiene.md#the-hygiene-footer)) names only
`/clear`, `/compact`, or `/rite-handoff`, plus the action-line command unchanged if it
repeats one. **Failing case:** `Next: /rite-prove` above a footer resuming with `/rite-review`.

If a required decision, proof, or invariant is missing, use one of these states
instead of `Done`:

```text
Awaiting human: <question id and gate>
Question: <exact decision needed>
Recommended: <one option and reason>
Resume: /rite-resolve <qid> "<answer>"
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
Next: /rite-ship
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
output or an artifact. A NOT-RUN verification keeps its marker in `Evidence:` or `Open:`
per [`core.md`](standards/core.md#gate-contract) § Gate contract.
Use exactly one recommended next action except for
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

The terminal cursor marker is `Next: none` exactly as shown; a stored `Next step: none`
names the same exhausted state.

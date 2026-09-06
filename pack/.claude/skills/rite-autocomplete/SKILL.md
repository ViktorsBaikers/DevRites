---
name: rite-autocomplete
description: Run the full DevRites lifecycle unattended; --ship continues to the final Git approval boundary. Use for one-shot autonomous work; not for a single phase.
argument-hint: "[idea] [--ship|--yolo] [--max-slices N] [--full] [--cross-model]"
user-invocable: true
---

# /rite-autocomplete: full lifecycle, unattended

Run every phase after one clarification window. Irreversible-risk,
blocking/escalating,<!-- pack-scan-ignore: negated statement: gates are NOT disabled -->
NO-GO, resource, and human-owned gates still pause. Use the Standard native
profile by default and Full for high-risk scope or explicit `--full`; see
[`orchestration-profiles.md`](../devrites-lib/reference/orchestration-profiles.md).

## Required rules

Read `devrites-lib/reference/standards/core.md` and `afk-hitl.md` first. Read
`one-shot-actions.md` before any stop/continue or execution decision involving a
consumptive action. Before honoring a terminal cursor that names missing
executable controller/harness/bundle bytes or a missing writer, read
`workflow-artifacts.md`. Before classifying any Reslice, read `.claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → keep Plan repair/affected Vet internal; no stop solely for topology/count.
- `GUARD_AND_REPAIR` → enter Spec Drift Guard/Clarify; pause only at an existing human-owned gate; resume Plan/Vet internally.
- `BLOCKED_INPUT` → no planning writes; stop internal branch; exact diagnostic; recover authority; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Invariants

- Use one initial human window: Spec, topology-first Clarify, and
  `Decision coverage: CLEAR`. Only then arm AFK. Record later reversible choices
  under [decision policy](reference/decision-policy.md).
- Safety is not bypassable. `--ship`/`--yolo` reaches only the exact-plan
  literal-GO and native-approval boundary; it never authorizes Git.
- **Treat technical readiness as routing, not completion.** `NEEDS REPLAN` is a backward edge
  to Plan repair and narrow Vet. `NEEDS REPLAN` cold resume follows a valid
  technical return cursor before forward work or any reply. Clarification is
  likewise internal when current authority resolves it.
- Always follow agent-owned backward edges through repair, affected Vet,
  correction, and proof. A nested `STOP`/`Next step` is not a user handoff.
  Pause only for a shared human/safety/access/exhaustion condition.
- **Do not confuse an action budget with recovery exhaustion.** Spent action
  authority blocks another real execution, not offline correction of retained
  Critical/Important evidence. Stop for fresh GO only after changed conditions
  and READY Vet.
- Every exact causal fingerprint gets at most three no-progress corrections.
  Closing one is progress; a distinct Critical/Important invariant has its own
  budget. Lower severity cannot prolong recovery.
- Enforce `--max-slices`, the AFK sentinel, expiry, review queue, and native
  agent/token/cost/time limits before every phase or dispatch. Count failed,
  malformed, and unavailable leaf calls. At the review cap, permit only work
  that reduces the queue.
- Parse flags only from this invocation. `--ship`, `--yolo`, `--max-slices`,
  `--full`, and `--cross-model` activate only as exact standalone tokens in
  `$ARGUMENTS`; examples or earlier messages can never arm them. `--max-slices`
  must occur once and be followed by a positive base-10 integer; missing, repeated, malformed, or conflicting values stop before any write.
- Temper always runs after Clarify. Unattended mode auto-applies only
  `hold-rigor` and `reduce-to-MVP`; any `expand` or added acceptance pauses.
- Vet every plan. Cross-model is off unless the current invocation arms it.
  Never auto-grow accepted scope.

## Workflow

1. **Orient and parse.** Resolve the explicit or active slug; when a workspace exists,
   require its `state.md` and read its cursor; a fresh idea starts empty at step 2.
   Before honoring `blocked` with terminal `next_action`,
   reconcile retained consumptive evidence and fingerprint attempts from
   `drift.md`/`evidence.md`; a retained fingerprint below its cap resumes offline
   recovery.
<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"lifecycle cursor encounters admitted set or resumable journal","action":"invoke classifier; execute returned route internally","return":"saved phase/action; zero intermediate reply"} -->
   Reconstruct a missing return cursor only from
   the current phase and exact approved action in `test-plan.md`/evidence, never
   from chat. After compaction or a resumed session, read `.devrites/ACTIVE`, then
   the `state.md` cursor, `questions.md`, `decisions.md`, and `test-plan.md`/`evidence.md`.
   Normalize current arguments to idea, `ship_preflight: yes|no`,
   `max_slices: N|default`, `profile: standard|full`, and `cross_model: yes|no`.
   **Completion:** normalized state is unambiguous and no sentinel or workspace file has been written.
2. **Specify and clarify.** Run `devrites-interview`, `/rite-spec`, and
   `/rite-clarify` as one window. Partial/Missing material coverage never arms
   AFK. **Completion:** `Decision coverage: CLEAR` is durable.
3. **Arm AFK once.** Apply the loop's
   [one-write AFK contract](reference/loop.md#arm-afk-once): preserve valid
   existing bytes or create the bounded advisory-only sentinel once; never
   rewrite it after Vet. Preserve an existing sentinel byte-for-byte. Touch
   `.devrites/CHECKPOINT` so proven slices get local crash-survivable WIP
   checkpoints for Ship to collapse. **Completion:** valid read-only AFK and checkpoint sentinels exist.
4. **Drive phases.** Follow [the loop](reference/loop.md): `/rite-spec` →
   `/rite-clarify` → `/rite-temper` → `/rite-define` → `/rite-vet` →
   `/rite-build` × pending slices → `/rite-prove` → `/rite-polish` →
   `/rite-review` → `/rite-seal`. Read and execute each skill; durable files, not
   chat, carry state. Apply the mutable post-vet budget before first Build.
   **Completion:** loop reaches Seal GO or persists a valid stop before any later phase.
5. **Apply stops.** At every gate use
   [stop-conditions.md](reference/stop-conditions.md). A `blocked` label alone is not a stop condition:
   route agent-owned red results through bounded recovery.
   Red gates block forward advancement and enter caller-owned recovery. Stop on
   hard risk, human-owned blocking/escalating/NO-GO, resource exhaustion, or the
   exact fingerprint's proven exhaustion. Technical exhaustion records terminal
   `Next step: none`, never a routine phase command. **Completion:** no stop is active, or its cursor and reason are durable.
6. **Seal boundary.** Without a ship flag, stop at Seal GO with `/rite-ship`.
   With one, perform Ship preflight, disclose the exact Git plan, and stop for
   fresh literal `GO` plus native approval. **Completion:** performed no Git action before fresh literal `GO` plus native approval.

## Reply and resume

HITL Build returns after one slice; explicit valid AFK may chain only green
slices within every cap. One Autocomplete invocation owns all internal
backtracking. Context pressure, compaction, or a nested completion footer does
not create a stop; persist and resume the cursor.

Use one terse line:

```text
Autocomplete: <slug>
spec <done|stopped> · clarify <clear|stopped> · temper <done|stopped> · define <done|stopped> · vet <ready|stopped> · build <n/N|stopped> · prove <done|stopped> · polish <done|stopped> · review <done|stopped> · seal <GO|NO-GO|stopped>
```

Final state is `Shipped`, `Stopped`, `Awaiting human`, `NO-GO`, or `GO`; do not
write a narrative recap. Require a clean or accepted baseline, arm checkpoint
mode, and never auto-pass a red gate.

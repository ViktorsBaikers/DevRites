---
name: rite-autocomplete
description: Run the full DevRites lifecycle unattended; --ship continues to the final Git approval boundary. Use for one-shot autonomous work; not for a single phase.
argument-hint: "[idea] [--ship|--yolo] [--max-slices N] [--parallel N] [--full] [--cross-model] [--lookahead]"
user-invocable: true
---

<!-- loads: {"always":["devrites-lib/reference/standards/core.md","devrites-lib/reference/standards/afk-hitl.md","devrites-lib/reference/standards/one-shot-actions.md","devrites-lib/reference/orchestration-profiles.md","devrites-lib/reference/parallel-workspaces.md","devrites-lib/reference/standards/acceptance-preserving-reslice.md","rite-autocomplete/reference/decision-policy.md","rite-autocomplete/reference/loop.md","rite-autocomplete/reference/stop-conditions.md"],"triggers":{"workflow-artifacts":["devrites-lib/reference/standards/workflow-artifacts.md"],"lookahead":["rite-autocomplete/reference/lookahead.md"]},"workspace":["state.md","spec.md","plan.md","tasks.md","questions.md","decisions.md","drift.md","evidence.md","test-plan.md"],"rootOnly":["devrites-lib/reference/standards/afk-hitl.md","devrites-lib/reference/orchestration-profiles.md","devrites-lib/reference/parallel-workspaces.md","devrites-lib/reference/standards/workflow-artifacts.md","rite-autocomplete/reference/lookahead.md"]} -->
> Read-set manifest: `devrites-engine context --skills-root .agents/skills <slug> --phase autocomplete` bundles every file named below into one deduplicated read. Trigger names map to the conditional rules in the sections that follow.


# $rite-autocomplete: full lifecycle, unattended

Run every phase after one clarification window. Irreversible-risk,
escalating,<!-- pack-scan-ignore: negated statement: gates are NOT disabled -->
access, unanswered blocking with no recommended option, and remaining NO-GO
still pause. Open blocking questions that already name a ranked recommended
option auto-resolve; they are not a user handoff, except a question whose answer
would add acceptance or product scope, which a recommended option never resolves.
Use the Standard native
profile by default and Full for high-risk scope or explicit `--full`; see
[`orchestration-profiles.md`](../devrites-lib/reference/orchestration-profiles.md).

## Required rules

Read [`core.md`](../devrites-lib/reference/standards/core.md) and [`afk-hitl.md`](../devrites-lib/reference/standards/afk-hitl.md) first. Read
[`one-shot-actions.md`](../devrites-lib/reference/standards/one-shot-actions.md) before any stop/continue or execution decision involving a
consumptive action. Before honoring a terminal cursor that names missing
executable controller/harness/bundle bytes or a missing writer, read
[`workflow-artifacts.md`](../devrites-lib/reference/standards/workflow-artifacts.md)
(trigger `workflow-artifacts` — only when a workspace declares them). Before classifying any Reslice, read `.agents/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

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
  Pause only for a shared human/safety/access condition.
- **Do not confuse an action budget with recovery exhaustion.** Spent action
  authority blocks another real execution, not offline correction of retained
  Critical/Important evidence. Stop for fresh GO only after changed conditions
  and READY Vet.
- Every exact causal fingerprint gets three no-progress corrections per strategy.
  A spent strategy escalates to a new correction class inline; after three
  escalations it parks while independent work continues. Never ask permission to
  retry. Closing one is progress; a distinct Critical/Important invariant has its
  own budget. Lower severity cannot prolong recovery.
- Do not stop on `--max-slices`, sentinel `max_slices`, `max_agents`,
  `max_minutes`, or `max_review_queue`. Count failed, malformed, and
  unavailable leaf calls.
- Never end the turn with a progress note while a wave runs or a next step
  exists: on a headless host a text-only reply ends the session and kills
  running children. Wait with the host's wait tool and keep working; end only
  at a recorded stop. A host reminder or rule notice (style rule, advisory) is
  not a stop either: follow it in your next tool call, never in a text-only
  acknowledgement.
- Parse flags only from this invocation. `--ship`, `--yolo`, `--max-slices`,
  `--parallel`, `--full`, `--cross-model`, and `--lookahead` activate only as exact standalone
  tokens in `$ARGUMENTS`; examples or earlier messages can never arm them.
  `--max-slices` must occur once and be followed by a positive base-10 integer;
  `--parallel` must occur once and be followed by a base-10 integer in `1`–`10`.
  Missing, repeated, malformed, or conflicting values stop before any write.
- Temper always runs after Clarify. Auto-apply `hold-rigor`, `reduce-to-MVP` or a skip;
  record the ADR and fold via Spec Drift Guard. `expand` and extra acceptance never
  auto-apply: ask through `AskUserQuestion`, hold the fold, and stop until a human
  resolves the qid. Irreversible-risk still pauses.
- Vet every plan. Cross-model is off unless the current invocation arms it.

## Workflow

With `--lookahead` (trigger `lookahead`), follow
[`lookahead.md`](reference/lookahead.md) instead of the steps below: plan the
next sequence continuation through Vet in a second session, then stop.

1. **Orient and parse.** Resolve source per [phase target binding](../devrites-lib/reference/parallel-workspaces.md#phase-target-binding).
   Read its existing cursor; a fresh idea starts empty at step 2. Normalize this
   invocation to idea, `ship_preflight: yes|no`, `max_slices: unlimited`,
   `parallel: N|default`, `profile: standard|full`, `cross_model: yes|no`.
   Apply the loop's [AFK contract](reference/loop.md#arm-afk-once) to an existing
   sentinel: upsert `mode: autocomplete`, plus capacity only for explicit
   `--parallel N`. Mode identifies this controller, never grants authority.
   Run `devrites-engine frontier <source>` before local recovery; execute the
   [global pass](../devrites-lib/reference/parallel-workspaces.md#continuous-global-frontier).
   Reconcile blocked/terminal cursors against retained action evidence and exact
   fingerprint attempts; resume offline recovery below its cap. This explicit
   invocation reopens parked fingerprints. Reconstruct missing return cursors
   only from the current phase and approved `test-plan.md`/evidence action.
<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"lifecycle cursor encounters admitted set or resumable journal","action":"invoke classifier; execute returned route internally","return":"saved phase/action; zero intermediate reply"} -->
   After compaction/resume, run `devrites-engine handoff <source> --resume`
   and open its `read_next` artifacts per [context hygiene](../devrites-lib/reference/standards/context-hygiene.md).
   **Completion:** arguments/source are unambiguous; local cursor and handles survive.
2. **Specify and clarify.** Run `devrites-interview`, `$rite-spec`, and
   `$rite-clarify` as one window. `open` material coverage never arms
   AFK. **Completion:** `Decision coverage: CLEAR` is durable.
3. **Arm AFK once.** Apply the [one-write AFK contract](reference/loop.md#arm-afk-once):
   create only after clarity; explicit controller entry upserts
   `mode: autocomplete` and, only with `--parallel N`, `max_parallel: N`.
   Preserve all other fields; reject malformed/duplicate mode rather than
   silently becoming standalone AFK. **Completion:** valid controller identity
   and capacity are durable; no gate or Git authority changed.
   **Failing case:** leftover `max_parallel: 1` after explicit `--parallel 5`.
4. **Drive phases.** Execute [the loop](reference/loop.md): Spec → Clarify →
   Temper → Define → Vet → Build → Prove → Polish → Review → Seal. Durable files
   carry state. Apply the mutable post-vet budget before Build. Consume fresh scheduling
   emitted by recorded `dispatch return`; rerun frontier after further changes
   and before waits. Before ordinary local Temper/Define/Plan/Vet context/dispatch,
   resolve sequence authority and start actual immediate-successor preparation
   under [early admission](reference/loop.md#early-preparation-admission),
   or retain the global pass's concrete deferral; keep the full Build gate.
   Derive initial preparatory-only routing before any marked wave; do not recurse
   through the roadmap. Preserve local cursors, protected validation windows and
   every admission/isolation gate; pass Build the resolved global cap.
   **Completion:** Seal GO or a durable valid stop precedes later phases.
5. **Apply stops.** At every gate use
   [stop-conditions.md](reference/stop-conditions.md). A `blocked` label alone is not a stop condition:
   route agent-owned red results through bounded recovery.
   Red gates block forward advancement and enter caller-owned recovery. Stop on
   hard risk, escalating, irreversible-risk, access, blocking with no
   recommended option, remaining NO-GO, or parked fingerprints with no
   independent work left. An open blocking question that already names a ranked recommended
   option (`proposed` or option 1 labelled Recommended) auto-resolves through
   `devrites-engine state resolve` and continues (a missing or failing resolver is
   the bounded resolver-failure stop); do not emit `$rite-resolve` as
   a user handoff. Exception: a question whose answer would add acceptance or
   product scope is never auto-resolved by a recommended option; it stops for a human.
   A spent strategy escalates; parked recovery records
   `Next step: $rite-autocomplete`, never a routine phase command. **Completion:** no stop is active, or its cursor and reason are durable.
6. **Seal boundary.** With or without a ship flag, leave intermediate sealed
   workspaces unshipped and open the next recorded continuation
   ([loop.md § Sequence continuation](reference/loop.md#sequence-continuation))
   unless `continue_sequence: false`, the sequence budget is spent, or no entry
   remains. At that boundary without a ship flag, stop at Seal GO with `$rite-ship`.
   Report sequence position at every stop. Only at that boundary, with a ship flag, perform Ship preflight,
   disclose the exact Git plan, and stop for fresh literal `GO` plus native approval. **Completion:** performed no Git action before fresh literal `GO` plus native approval.

## Reply and resume

HITL Build returns after one slice; Autocomplete ignores `--max-slices`, chaining
pending slices. Each Autocomplete invocation owns internal backtracking. Context pressure, compaction, or a nested completion footer does
not create a stop; persist and resume the cursor.

Reply only at final state: text-only turns end the run. Keep
calling tools after compaction or phase changes. Before any user-visible reply,
`devrites-engine check continuation` must print `continuation: clear`
(`handoff` prints the same lines). `required` means invoke `invoke:` now.
`stopped` is not a phase token. Print this card only after `clear`:

```text
Autocomplete: <slug>
spec done · clarify clear · temper done · define done · vet ready · build n/N · prove — · polish — · review — · seal —
```

Final state, only after `clear`: `Shipped`, `Stopped`, `Awaiting human`,
`NO-GO`, or `GO`; no narrative recap. Require clean or accepted baseline; land
local unpushed checkpoints after each green check; never auto-pass red gates.
**Failing case:** `vet stopped` plus `Resume: $rite-vet` while the check says `required`.

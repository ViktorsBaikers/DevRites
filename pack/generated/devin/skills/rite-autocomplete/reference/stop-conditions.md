# Stop conditions: when autocomplete pauses

On real stops, persist `state.md` `awaiting_human` or `blocked`, reason, and one
resume command. Agent-owned recovery never ends the run: spent strategies
escalate; fully escalated fingerprints park while other work continues ([retry contract](../../devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve)).
`--ship` never bypasses a stop.

Authority: `.devin/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → keep Plan repair/affected Vet internal; no stop solely for topology/count.
- `GUARD_AND_REPAIR` → enter Spec Drift Guard/Clarify; pause only at an existing human-owned gate; resume Plan/Vet internally.
- `BLOCKED_INPUT` → no planning writes; stop internal branch; exact diagnostic; recover authority; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Always stop

Flags/AFK ceiling never override:

- destructive data migration;
- auth/authz boundary change;
- public API break;
- external-service contract change; or
- filesystem destruction outside the workspace.

Red proof cannot advance. Run bounded Debug Recovery; escalate spent strategies.
Fully escalated fingerprints park unless remaining owner is human.

## Not stops: internal technical routing

`NEEDS_REPLAN` is not a stop condition under active Autocomplete. It blocks
forward work, persists the return cursor, and invokes Plan repair plus Recovery
Vet inline. The same applies to nested `STOP`/`Next step` when its decision is
agent-owned.

Agent-owned backtracking is not a stop condition while its exact causal
fingerprint has budget: invoke earlier phase, affected Vet, remediation, proof;
resume origin. Three no-progress corrections spend only that fingerprint's current
strategy: escalate inline to the next class (shared root cause, diagnostic
amplification, boundary redesign) with fresh budget, without permission. After
three escalations per activation it parks; finish independent work, then stop once.
Preserve reproduction, attempts, and dead ends; never offer `/rite-plan unblock`
or another routine phase command. A new user-issued `/rite-autocomplete`
reopens parked fingerprints with fresh escalation; wakes and compaction do not.
Closure is progress; a separately evidenced Critical/Important invariant starts
its own cap and cannot extend the closed one.

Spent consumptive-action authorization is not technical-recovery exhaustion.
It blocks another execution; retained new Critical/Important evidence still
enters offline recovery. After affected Vet is READY, pause for fresh action
authorization and never reuse old GO.

Past evidence being irretrievable is not by itself terminal. If an in-scope trusted seam
uniquely discriminates the next failure, run diagnostic-amplification Plan repair
and narrow Vet inline; pause for fresh GO before one evidence-acquisition attempt.
Without a safe seam, escalate or park; only a real human/risk/scope gate stops for a human.

<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"classifier returns owner-busy, exhausted, or existing hard gate","action":"escalate BLOCKED_EXHAUSTED; on WAIT_ACTIVE_OWNER run independent work and recheck; stop on exact BLOCKED_GATE result","return":"unchanged cursor plus fixed route-owned output"} -->
## Other stop classes

- **Gate severity:** escalating always stops. Blocking with a ranked recommended
  option (`proposed` or option 1 labelled Recommended): auto-pick via
  `devrites-engine state resolve`, then continue internally (including Spec Drift
  Guard when the answer changes ownership or acceptance). Exception: a question whose answer
  would add acceptance or product scope is never auto-picked, even with a
  recommended option; it stops for a human. Blocking with no
  recommended option still stops. Validating: auto-pick the recommended option,
  close the question, continue — except Seal's judgment-only acceptance question
  (`slice: seal`), which only a human answers; it stops.
- **Temper:** `expand` and extra acceptance stop: ask through `ask_user_question`, hold the
  fold, and continue only after a human resolves the qid (an ADR is not that answer).
  Irreversible-risk still pauses. `hold-rigor`, `reduce-to-MVP`, and a skip do not pause.
- **Clarify:** material `open`/unowned decision coverage or a
  low-confidence high-consequence assumption. Continue the initial interview;
  never arm AFK early.
- **Seal:** report remaining NO-GO with every blocker/fix direction; never round
  up to GO. Resolve recommended-option blocking questions internally; recover
  agent-owned findings and re-enter Seal. A first NO-GO is not a stop:
  recover every agent-owned blocker (missing or unbound proof, a reviewer gap root
  can close, a finding with a fix direction, a missing `review.md` verdict), then
  re-enter Seal. The unattended `N` to accepting Important risks means repair, not
  stop. Stop only for blockers that need a human: accepting a risk, changing scope
  or an acceptance criterion, or access. Record each one as a question.
- **Reslice:** execute its marked action before deciding continue/stop.
- **Slice budget:** do not stop on `--max-slices`. Leftover
  pre-existing remaining value, explicit flag, sentinel cap, or post-vet pending count
  does not stop. Zero with no pending slices is normal completion.
  Malformed flag stops before any write.
- **Foreign writer:** a `live`/`expired` claim or `WAIT_ACTIVE_OWNER` held by
  another session blocks only work on those paths; run independent work, re-check,
  and stop only when nothing independent remains. `released`/`dead` claims are
  reconciled per [§ Cancellation](../../devrites-lib/reference/parallel-dispatch.md#cancellation-and-terminal-reconciliation).
- **Resource envelope:** overlapping run or missing AFK after arm still stops.
  Do not stop on `--max-slices`, `max_agents`, `max_minutes`, or `max_review_queue`.
- **Confidence:** intent still cannot become testable acceptance after interview.
- **Repeated failure:** not a stop. Escalate; fully escalated fingerprints park.
  Stop only when no independent work remains.
- **Resolver failure:** `devrites-engine state resolve` missing or nonzero.
  Never hand-edit `questions.md`/`state.md` instead. Missing stops immediately;
  nonzero permits one identical re-run (fingerprint: command + exit code + decisive
  stderr line; unchanged re-run is no correction, no further attempts). Then stop
  `blocked`: exact command, exit code, stderr line; resume via `/rite-doctor`.

## Final GO and durable stop

With a recorded continuation, no opt-out, and workspace budget, Seal GO stays
internal: continue without `--ship`, leaving milestones sealed/unshipped. Only the
final milestone, explicit `continue_sequence: false`, or exhausted sequence cap
ends green sequence continuation.

There, `--ship`/`--yolo` requires [Ship preflight](../../rite-ship/reference/git-ship.md#before-type-go):
disclose exact plan; stop for fresh literal `GO` plus native approval.
Without flags, stop at Seal GO with `/rite-ship`. Seal GO never authorizes Git
or satisfies Ship approval; neither do AFK, prior approval, or flags.

Record cold-resumable `state.md` stops: status, exact reason, one resume action
(human-owned or `/rite-autocomplete` to reopen parked fingerprints). Explain in
owning question/drift/seal entry. Tell user what stopped and any required human action.

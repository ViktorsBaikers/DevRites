# Stop conditions: when autocomplete pauses

On a real stop, write `state.md` as `awaiting_human` or `blocked`, persist the
reason, and include one resume command only when an actual human/safety/access
action can change it. Exhausted agent-owned recovery records terminal
`Next step: none` and no runnable recovery command. `--ship` never bypasses a
stop.

Authority: `.omp/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → keep Plan repair/affected Vet internal; no stop solely for topology/count.
- `GUARD_AND_REPAIR` → enter Spec Drift Guard/Clarify; pause only at an existing human-owned gate; resume Plan/Vet internally.
- `BLOCKED_INPUT` → no planning writes; stop internal branch; exact diagnostic; recover authority; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Always stop

Regardless of flags or AFK ceiling:

- destructive data migration;
- auth/authz boundary change;
- public API break;
- external-service contract change; or
- filesystem destruction outside the workspace.

Red proof cannot advance. Run bounded Debug Recovery; stop after proven
exhaustion unless the remaining owner is human.

## Not stops: internal technical routing

`NEEDS_REPLAN` is not a stop condition under active Autocomplete. It blocks
forward work, persists the return cursor, and invokes Plan repair plus Recovery
Vet inline. The same applies to nested `STOP`/`Next step` when its decision is
agent-owned.

Agent-owned backtracking is not a stop condition while the exact causal
fingerprint has budget. Invoke the earlier phase, affected Vet, remediation, and
proof; then resume the origin. Stop only after three no-progress corrections of
that exact fingerprint or a real human/safety/access gate. Preserve reproduction,
attempts, and dead ends; never offer `/rite-plan unblock` or another routine
phase command. Reinvocation with unchanged evidence does not reset the cap.
`unchanged` means the same fingerprint already has three recorded no-progress
corrections. Closure is progress; a separately evidenced Critical/Important
invariant starts its own cap and cannot extend the closed one.

Spent consumptive-action authorization is not technical-recovery exhaustion.
It blocks another execution; retained new Critical/Important evidence still
enters offline recovery. After affected Vet is READY, pause for fresh action
authorization and never reuse old GO.

Past evidence being irretrievable is not by itself terminal. If an in-scope
trusted seam can uniquely discriminate the next failure, run diagnostic-
amplification Plan repair and narrow Vet inline, then pause for fresh GO before
one evidence-acquisition attempt. Use terminal none only when no safe in-scope
seam exists, a real human/risk/scope gate owns it, or bounded recovery is
exhausted.

<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"classifier returns owner-busy, exhausted, or existing hard gate","action":"stop on exact WAIT_ACTIVE_OWNER, BLOCKED_EXHAUSTED, or BLOCKED_GATE result","return":"unchanged cursor plus fixed route-owned output"} -->
## Other stop classes

- **Gate severity:** escalating always stops. Blocking with a ranked recommended
  option (`proposed` or option 1 labelled Recommended): auto-pick via
  `devrites-engine state resolve`, then continue internally (including Spec Drift
  Guard when the answer changes ownership or acceptance). Blocking with no
  recommended option still stops. Validating: auto-pick the recommended option,
  close the question, continue.
- **Temper:** `expand` and extra acceptance auto-apply (record + Drift Guard).
  Irreversible-risk still pauses. `hold-rigor`, `reduce-to-MVP`, and a skip do not pause.
- **Clarify:** material Partial/Missing/unowned decision coverage or a
  low-confidence high-consequence assumption. Continue the initial interview;
  never arm AFK early.
- **Seal:** remaining NO-GO after recommended-option questions are closed, with
  every remaining blocker and fix direction. Never round up to GO. An open
  recommended-option blocking question is not a Seal stop: resolve it, recover
  agent-owned findings, then re-enter Seal.
- **Reslice:** execute its marked action before deciding continue/stop.
- **Slice budget:** do not stop on `--max-slices`. Leftover
  pre-existing remaining value, explicit flag, sentinel cap, or post-vet pending count
  does not stop. Zero with no pending slices is normal completion.
  Malformed flag stops before any write.
- **Resource envelope:** overlapping run or missing AFK after arm still stops.
  Do not stop on `--max-slices`, `max_agents`, `max_minutes`, or `max_review_queue`.
- **Confidence:** intent still cannot become testable acceptance after interview.
- **Repeated failure:** the exact fingerprint's bounded recovery is exhausted.
- **Resolver failure:** `devrites-engine state resolve` not found, or nonzero exit.
  Never hand-edit `questions.md`/`state.md` in its place. Not found stops at once;
  nonzero allows one identical re-run (the fingerprint is command + exit code +
  decisive stderr line; an unchanged re-run is not a correction, so no further
  attempts). Then stop `blocked` with the exact command, exit code, and stderr line,
  and resume action `/rite-doctor`.

## Final GO and durable stop

`--ship`/`--yolo` never authorizes Git. With a flag, complete Ship preflight,
disclose the exact plan, and stop for literal `GO` plus native approval. Without
one, stop at Seal GO with `/rite-ship`. Seal GO, AFK, prior approval, and flags
never satisfy Ship approval.

A cold-resumable stop records `state.md` status, exact reason, and either one
human-owned resume action or `none — technical recovery exhausted`; the owning
question/drift/seal entry explains it. The user-facing line says what stopped
and, only when applicable, what human action resumes it.

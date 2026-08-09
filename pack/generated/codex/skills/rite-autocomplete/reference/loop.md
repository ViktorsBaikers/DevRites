# The autocomplete loop: arm AFK, drive every phase

Autocomplete sequences existing `/rite-*` workflows and enforces stop
conditions without pausing between routine phases.

## Arm AFK once

Arm the gate policy up front without mutating it later:

```yaml
allow_gates: [advisory]        # only advisory auto-handles; validating+ pause
# notify: "<cmd>"              # optional — fired on any awaiting_human pause
# max_slices: <N>              # include only for an explicit/configured cap
```

Read an existing sentinel first. Preserve it byte-for-byte when valid; stop if its gate
ceiling exceeds `advisory` or its `max_slices` conflicts with the invocation. If absent,
write it once after clarity with `allow_gates: [advisory]` and an explicit
`--max-slices N` only when supplied. The sentinel is read-only: never rewrite it after
`$rite-vet` to inject a discovered plan count.

### Derive the mutable post-vet budget

After `$rite-vet`, count remaining pending slices and derive the run budget as the
minimum of that count, an explicit `--max-slices`, and a configured sentinel cap. Before
the first build dispatch, pre-seed `state.md` `afk_slices_remaining` when absent. If a
valid remaining counter already exists, retain the lower of it and the derived budget;
never increase or reinitialize it. This keeps the crash-survivable budget in its mutable owner
without changing AFK configuration mid-run.

`allow_gates: [advisory]` prevents an open `gate: validating` from being queued until
seal, where it would force NO-GO under `afk-hitl.md`. Autocomplete pauses on it instead. Widen
`allow_gates` only outside autocomplete through an explicit human configuration change;
autocomplete itself never widens the ceiling.

## Drive the phases

Read each phase's `SKILL.md` and execute that workflow. Workspace files such as
`state.md`, `tasks.md`, and `evidence.md` carry state between phases; chat does not.

| Step | Phase | Loop / gate |
|---|---|---|
| 1 | `$rite-spec` | interactive window: investigate, feed intent answers, write `spec.md` |
| 2 | `$rite-clarify` | same interactive window: topology-first scan; write `decision-coverage.md`; proceed only on `CLEAR`, then arm AFK |
| 3 | `$rite-temper` | significance-gated strategic review; harden spec + write `strategy.md`. Skip low-stakes specs in one line. AFK: `hold-rigor` / `reduce-to-MVP` auto-apply; **any `expand` pauses (blocking)**; irreversible-risk pauses |
| 4 | `$rite-define` | reads `decision-coverage.md` + `strategy.md`; writes `plan.md` + `tasks.md`; records `Plan approved` |
| 5 | `$rite-vet` | engineering/readiness review on **every** plan (light pass on simple plans, full on big/risky; never skipped); harden `plan.md` / `tasks.md` + write `eng-review.md` (`Implementation readiness: READY`) + `test-plan.md`. AFK: hardening / coverage findings auto-apply; **any scope-growing / acceptance-changing finding pauses (blocking)**; irreversible-risk pauses. Derive and pre-seed the state-owned slice budget after this (vet may split a slice); never rewrite the AFK sentinel |
| 6 | `$rite-build` ×N | **loop** while any slice is `pending`; build one, then let the root charge exactly one budget unit with the built-state record. Zero ⇒ STOP before another dispatch. |
| 7 | `$rite-prove` | once all slices `built`; walks `test-plan.md`; on failure → `devrites-debug-recovery` within scope |
| 8 | `$rite-polish` | re-verify after code edits (evidence must stay fresh) |
| 9 | `$rite-review` | apply in-scope fixes; re-prove if code changed |
| 10 | `$rite-seal` | GO/NO-GO decision (no git here) |
| 11 | `$rite-ship` | only if seal GO; `--ship` / `--yolo` never authorizes Git and only continues to the exact-plan literal-GO/native-approval boundary |

## Backtrack without handing off

When a later phase finds an agent-owned technical gap in an earlier phase, the
Autocomplete root remains the caller:

On cold resume, reconcile the terminal cursor against durable fingerprint
accounting first. A retained distinct fingerprint with fewer than three
no-progress corrections is unfinished recovery, not an unchanged terminal stop.
Restore `return_phase` from the current phase and `return_next_action` only from
the exact approved action recorded in `test-plan.md` / evidence; ambiguity returns
to the applicable Vet contract and never licenses execution.

Also reconcile a **stale writer-exhaustion cursor** against
`workflow-artifacts.md`. When exact Vet-ready workflow-artifact paths and behavior
exist, the old attempts only targeted the read-only drafter/reviewer or product
wright, and there is **no controlling-root materialization attempt**, supported
root ownership is a changed routing condition. Preserve the old fingerprint,
record the new materialization fingerprint, and run that offline branch directly;
do not charge or require product-slice/AFK budget, re-ask a resolved question, or
request GO. Once a root materialization attempt exists, never apply this migration
again—route its observed failure normally. That one-time migration
does not make the first root attempt terminal: record it as attempt one under the new
materializer fingerprint and continue bounded offline correction while fewer than
three no-progress corrections exist. This reconciliation runs even when all
product slices are already built, before selecting the next forward phase.

1. Save the originating phase/action in the native return cursor unless a valid
   one already exists.
2. Invoke the required repair, Vet, remediation, and proof skills inline. After
   Plan repair, run Vet in its recovery mode as a narrow Vet recheck of the prior
   findings, changed paths/criteria, and affected evidence; do not restart a Full
   Vet over unchanged axes. Their
   `STOP` instructions end only those nested phases.
3. Re-read `state.md` after each nested phase and follow its intermediate
   `next_action`; do not hand the intermediate command to the user.
4. Reconcile each exact causal fingerprint from `drift.md` and `evidence.md`.
   Continue automatically when the recheck closes a prior fingerprint or admits
   a genuinely new Critical/Important fingerprint. Charge only a no-progress
   outcome against the same fingerprint; lower-severity novelty cannot prolong
   the chain.
   For a failed consumptive action, its spent authorization blocks only another
   real execution. A retained artifact that identifies a new fingerprint is the
   offline reproduction input: diagnose, repair, and narrow-Vet it now, then pause
   for fresh authorization before any next consumptive execution.
5. When the prerequisite chain is green, restore and consume the return cursor,
   resume the originating phase, and continue the forward table.

Count failed corrections by causal fingerprint under `afk-hitl.md`. Ask only
for a human-owned decision or mandatory safety/access action. Exhausted
agent-owned recovery stops once with its reproduction and dead ends, never with
another routine Plan/Vet command.

## Continuous caller obligation

No user-facing reply is permitted while the durable state contains an
agent-owned `NEEDS_REPLAN`, an intermediate Plan/Vet `next_action`, or a distinct
Critical/Important fingerprint with remaining recovery budget. Invoke the exact
next Plan repair and Recovery Vet immediately. A narrow reviewer closing its input
finding and exposing another Critical/Important invariant is progress: close the
old fingerprint, open the new one, and continue without handing a command to the
user.

The number of completed repair/Vet cycles is not a stop condition and cannot be
used as a surrogate recovery budget. Only three no-progress corrections of the
same exact fingerprint exhaust. Context pressure, compaction, session duration,
or a nested skill's completion footer also cannot convert an internal checkpoint
into a stop; persist the checkpoint and resume from it inside the same controlling
invocation. Autocomplete may emit its final reply only after reaching its requested
rest point or one of the shared human/safety/access/exhaustion stop conditions.

## Between phases

- Re-read the active workspace before each phase (don't trust chat memory).
- After a phase that edits code (build, polish, review), evidence may be stale: let
  the next gate re-prove rather than carrying a stale pass (see
  `.agents/skills/devrites-lib/reference/standards/development-workflow.md`).
- Check [stop-conditions.md](stop-conditions.md) at every gate **before** advancing.

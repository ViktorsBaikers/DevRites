# Stop conditions: when autocomplete must pause for a human

On these conditions, write `state.md` (`awaiting_human` or `blocked`), report
the reason and one resume command when an actual human/safety/access action can
change the state, notify if configured, and stop. Exhausted agent-owned
technical recovery is terminal for its unchanged causal fingerprint: record
`Next step: none` and no runnable recovery command. `--ship` cannot bypass them.
Closure of a prior fingerprint is progress, not exhaustion. A separately evidenced
Critical/Important failed invariant starts its own bounded fingerprint; it never
resets or extends the budget of the one just closed.

An exhausted consumptive-action authorization is not technical-recovery
exhaustion. It blocks another real action, but retained evidence of a new
Critical/Important fingerprint must enter offline caller-owned recovery while its
own no-progress budget remains. After affected Vet is READY, pause for fresh action
authorization; never execute from the old GO.

## Always stop (irreversible-risk list: from `afk-hitl.md`)

Regardless of `allow_gates` or `--ship`:
- Destructive data migration (drop column/table, irreversible backfill).
- Auth / authz boundary change.
- Public-API break (response shape, removed endpoint, changed status semantics).
- External-service contract change.
- Filesystem destruction outside the workspace.

Red checks are hard non-advance gates. Run bounded `devrites-debug-recovery`;
on exhaustion, stop as a technical blocker unless human-owned.

## Not a stop: agent-owned backtracking

Agent-owned backtracking is not a stop condition while its causal-fingerprint
budget remains. The active caller invokes the earlier phase inline, follows Vet
and any bounded remediation, then resumes the originating phase. Persist
`Next step` for crash recovery, but do not surface it as a command the human must
submit. Use the repaired finding's narrow Vet recheck to distinguish resolution,
the same decisive failure, and a genuinely new Critical/Important invariant.
Stop only after three no-progress attempts on the exact same fingerprint or when
the remaining choice is a real human/safety/access gate.

On technical exhaustion, preserve the fingerprint, reproduction, attempts, and
dead ends, then stop without `$rite-plan unblock` or another phase command.
Reinvocation with unchanged evidence remains blocked and does not reset the cap.
Here `unchanged` means the same fingerprint already has three recorded
no-progress corrections. A retained fingerprint with remaining offline budget is
not terminal merely because `state.md` was written by the failed action or a prior
session ended.

A pre-ownership workflow-artifact writer stop is not unchanged evidence when the
current contract supplies controlling-root materialization. Prior drafter/wright refusals do not count
as root materialization attempts. Reopen only when exact paths and executable
behavior passed Vet and there is **no controlling-root materialization attempt**;
record the routing migration so it cannot reset again. Missing product slices,
unresolved protocol choices, or a recorded root attempt remain under their normal
gate/fingerprint rules. The first controlling-root failure is not exhaustion: it
is the first no-progress result for the materializer fingerprint. If all targets
still equal their recorded preimages, no real action ran, and only admitted bound
temporaries remain, reopen an incorrectly persisted one-shot `1/1` terminal,
preserve its evidence as attempt one, clean/repair/preflight offline, and continue
within the shared three-attempt cap.

Past evidence being irretrievable is not by itself terminal. When an in-scope
trusted diagnostic seam can make the next retained fingerprint uniquely
actionable, Autocomplete must run diagnostic-amplification Plan repair and narrow
Vet inline, then pause for a fresh GO before the single evidence-acquisition
attempt. Use `Next: none` only after proving no safe in-scope amplification seam
exists, a real human/risk/scope gate owns it, or bounded recovery is exhausted.

## Stop on gate severity

- `blocking` gate fires → synchronous pause.
- `escalating` gate fires → pause, route to the specialist tag.
- Any `questions.md` entry with `gate: validating` and `status: open` → pause (it is a
  seal NO-GO by definition; stop before reaching seal).

## Stop on strategic-review scope expansion (`$rite-temper`)

- Any `expand` or added acceptance criterion pauses regardless of
  `allow_gates`/`--ship`. Only `hold-rigor` and `reduce-to-MVP` auto-apply;
  skipped low-stakes specs and those two modes do not pause.

## Stop on incomplete decision coverage (`$rite-clarify`)

- Stop on any material Partial/Missing/unowned row or low-confidence, high-consequence
  assumption. Continue the up-front window with the next genuine decision packet; never arm
  AFK or carry it into build. Resolve facts and reversible choices automatically.

## Stop on workflow state

- **NO-GO at seal** → stop; surface every blocker with `file:line` and the fix
  direction. Do not round NO-GO up to GO.
- **Spec Drift Guard fires** (`$rite-build` finds the plan is wrong and the change
  alters product behaviour) → stop; route through `$rite-plan repair`.
- **Budget exhausted with slices still pending:** any validated root-owned remaining
  value of zero stops before the next dispatch. Report which bound won (pre-existing
  remaining value, explicit flag, sentinel cap, or post-vet pending count) and the
  unbuilt slices. A malformed value also fails closed. Zero with no pending slices is
  normal completion → continue to `$rite-prove` without pausing.
- **Still low-confidence after the interview:** the idea can't be pinned to testable
  acceptance criteria → stop and ask, rather than guessing the product.
- **Repeated failure:** bounded recovery exhausts → stop with reproduction.

## The final type-GO

- `--ship` / `--yolo` never authorizes Git. Continue through ship preflight,
  disclose the exact plan, then stop for literal `GO` and native approval.
- Without either flag, stop at seal GO with `$rite-ship` as the resume command.
  Seal GO, AFK, prior approval, and flags never satisfy ship approval.

## How to stop well

State must be enough for a fresh agent to resume cold:
- `state.md`: `Status`, the blocking reason, and either a single actionable
  `Next step` command or the terminal `none — technical recovery exhausted`
  marker.
- The relevant `questions.md` / `drift.md` / `seal.md` entry that explains the pause.
- A one-line user-facing message: *what* stopped it and, only when one exists,
  *what human-owned action* resumes. A terminal technical blocker names no
  routine Plan/Vet/retry command.

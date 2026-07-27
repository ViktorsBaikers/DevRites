# rite-build phase contract

See also [`one-slice-cycle.md`](one-slice-cycle.md).

0. **Rules + AFK + readiness check.** Read
   `.agents/skills/devrites-lib/reference/standards/core.md`, then orient and run the
   deterministic readiness gate:
   ```bash
   devrites-engine preamble
   devrites-engine snapshot
   ```
   Treat the snapshot as canonical machine-readable status and use its host-specific
   `nextCommands` value:
   ```bash
   devrites-engine build-readiness; echo "readiness rc=$?"
   ```
   Any non-zero result is a hard STOP at the gate's reported route; `6` → `$rite-clarify`,
   `7` → `$rite-vet`, and `8` → `$rite-upgrade`. Under AFK, also
   enforce the mutable `state.md` budget from
   [`afk-discipline.md`](afk-discipline.md); zero remaining slices forces a HITL stop.
1. Read `spec.md`, `decision-coverage.md`, `plan.md`, `tasks.md`, `assumptions.md`,
   `drift.md`, `eng-review.md`, and `test-plan.md`
   if present (the vetted coverage target from `$rite-vet`: the slice's tests come from
   here when it exists). `state.md` and the open-`questions.md` tally are already in the
   preamble digest from step 0: re-read `questions.md` only for the full text of a flagged
   blocking question.
   Require `Decision coverage: CLEAR` and `Implementation readiness: READY`. If a
   **blocking `[NEEDS CLARIFICATION]`** or uncovered decision remains, stop →
   `$rite-clarify`; if plan readiness does not pass, stop → `$rite-plan` or `$rite-vet`
   as classified by the gate. Don't build on an unresolved spec or unvetted plan.
2. Select the next pending slice (or the one in `$ARGUMENTS`). **Restate its goal,
   acceptance criteria, and scope boundary** in one short block. Confirm it's still the
   right next slice. Write the slice's `Mode` to `state.md` as `Slice mode: <HITL|AFK>` on
   **every** selection, including the non-HITL path; `$rite-resolve` clears or updates
   it on resume.
2a. **HITL gate (pre-action pause).** Resolve any `HITL` slice checkpoint before code,
    exactly as [`checkpoint-protocol.md`](checkpoint-protocol.md) defines. An interactive
    human answers inline and the build continues in place. AFK may auto-pick only allowed
    gates; otherwise persist the question and awaiting-human cursor, notify, and STOP for
    `$rite-resolve`. Blocking, escalating, and irreversible-risk gates always take that stop path.
3. **Authorize exact paths, snapshot, then fresh-context dispatch the build core to
   `devrites-slice-wright`.** First write the root-owned exact project-relative file list to
   `.devrites/work/<slug>/.wright-allowlist`; mirror it in the packet's
   `scope.allowed_repo_writes`. The wright's later `Files changed` report cannot widen it.
   **Forge branch: only for a fully typed `Forge: yes` slice.** Follow
   [`forge.md`](forge.md) sections 1 through 3: require the scorecard, strategies, and
   real `manifest-env-v1` host binding; run `forge plan` before reconciliation; snapshot after a
   planned result; bind, dispatch, record, and extract every candidate; judge the
   immutable deltas; record one winner; and merge it. A typed serial degradation has
   no Forge side effects and uses the default branch below. A later error preserves
   the manifest-owned run for technical recovery; never degrade silently. After merge,
   validate the winner envelope and enter the common return check at the end of this
   step. Do not also dispatch a serial wright.

   **Default serial branch.** Follow
   [`wright-dispatch.md`](wright-dispatch.md) end to end; it owns snapshot timing, stuck
   logging, the exact packet, conditional skill loading, and return checks. Dispatch through
   named wright → safely enforced generic fresh worker. If neither is available, stop for
   HITL; the root never performs wright work. Before any canonical
   `.devrites/` mutation, validate the typed result and require a clean immediate
   `devrites-engine reconcile check`; exit `5` rejects the writer result.
4. **Doubt every stood decision before accepting the slice.** The writer never grades its
   own decisions: invoke `devrites-doubt` and its fresh reviewer for each entry. Do not enter
   step 5 until every entry has a recorded `accept` or resolved `reject` verdict. Log each dispatch:
   ```bash
   devrites-engine footprint log <slug> doubt "<decision id>"
   ```
   Apply Doubt's AFK exception. Irreversible-risk or blocking escalations pause; scope-changing
   answers route through the Spec Drift Guard. An irreversible-risk item misfiled under
   `Decisions stood` is a blocking protocol violation: re-dispatch it as out of bounds,
   never doubt-and-accept it.
   **Principle check (same standing):** a wright return that breaks a declared principle
   (reported in its `Principles` field, or that you detect against `.devrites/principles.md`) is
   handled here like an irreversible-risk item: block, route to a human-approved scoped
   exception in the register or stop; never doubt-and-accept a principle violation into the slice.
5. **Run retained-baseline integrity and every root-owned proof gate, then recover
   any objective red.**
   ```bash
   devrites-engine test-integrity; echo "test-integrity rc=$?"
   ```
   Run every exact command reported as `not-run`: `root-owned artifact-producing gate`
   in the wright result. These are root proof commands, not source-edit authority.
   Do not omit, rewrite, or substitute them; a missing command is unverifiable proof.
   After they pass, run `devrites-engine reconcile check` again after the root-owned gates
   and before close, proving their child processes did not change tracked source:
   ```bash
   devrites-engine reconcile check; echo "reconcile rc=$?"
   ```
   If the wright's `Gates` were red (targeted tests / types / lint), `test-integrity`
   failed, a root-owned gate failed, the final reconcile failed, or proof could not be
   verified: do **not** mark the slice `built`, and **do not fix the code yourself**.
   Classify each causal fingerprint through
   [`cleanup-and-classify.md`](../../devrites-debug-recovery/reference/cleanup-and-classify.md),
   run `devrites-engine recovery route <class>`, and follow the `recovery-route/v1`
   owner/action. Only `humanPause: true` or an exact human-only predicate opens a question;
   technical repair, environment stabilization, and proof reruns remain agent-owned.
   For an agent-owned defect route, continue the same wright under
   `devrites-debug-recovery`, carrying command/output, attempt count, and dead ends. Wright
   plus recovery share **three total attempts per root cause** and never
   rerun an unchanged check. Persist the initial and retry failures with
   `devrites-engine recovery record --class <class> "<root cause>" "<exact failure>" <slug>`,
   run `recovery check` before each retry, and
   `recovery clear --class <class> "<root cause>" <slug>` only after green. Before every
   re-dispatch, update the root-owned allowlist only for an accepted in-slice path and run
   `reconcile snapshot`: after a clean check this refreshes canonical-state scope while
   retaining the original source baseline. Repeat reconcile and the integrity gate on every
   return. This owns red gates, missing coverage, browser/runtime failures, and
   workflow-tool defects.
   After recovery:
   - green → continue to record;
   - product-contract/acceptance ambiguity or irreversible risk → open the genuine human gate
     (and use `$rite-plan repair` only when the durable plan or contract changes);
   - human-only access/action → gate with the exact needed input;
   - exhausted objective failure → preserve reproduction/dead ends, set `Status: blocked` and
     `Next step: $rite-plan unblock`, then STOP without a question or `$rite-resolve`.
6. **Close the retained baseline, then record. You are the canonical writer.**
   Reconciliation ran immediately on return; step 5 ran integrity, root-owned proof,
   and a final clean reconciliation.
   **Exit 3 → hard STOP:** a test was deleted, skipped, or de-asserted since the slice base: the
   slice went green by weakening its tests, a Critical protocol violation. Revert the weakening and
   re-dispatch the wright; do **not** mark the slice `built`.

   Once every gate and doubt verdict is accepted, a forged slice records successful
   verification and runs manifest-only cleanup per [`forge.md`](forge.md). Then close the
   private window:
   ```bash
   devrites-engine reconcile close
   ```
   Keep it open across a genuine human wait that resumes the same slice; close it before a
   scope/plan transition.

   Then, from the wright's artifact, update `state.md`,
   `evidence.md`, `touched-files.md` (and `browser-evidence.md` for UI). For a forged
   slice, write `forge-report.md` after reconciliation, verification, and cleanup;
   it records the run but owns nothing. Add a `## Review trail`
   to `touched-files.md`: group the slice's important `path:line` stops by concern (design intent,
   not file order), 1-5 concerns, each stop under 15 words. This is for a later human walkthrough;
   keep it factual and skip invented rationale. **Persist every
   `Decisions stood` entry from the wright's artifact to a `## Decisions stood` section in
   `decisions.md`, one line each ending `— doubt: <accept | reject-resolved | MISSING>`**: the
   verdict from step 4, or `MISSING` if step 4 was skipped for that entry. Write this ledger at
   record-time **independent of the doubt step**, so a stood decision lands on disk even when it
   was never doubted: `$rite-seal` cross-checks the ledger, and a `doubt: MISSING` entry is the
   skipped-doubt finding that would otherwise leave no trace (the per-slice skip the footprint
   count alone can't catch). If the wright stood no decisions, write `## Decisions stood` with
   `- none` so the absence is recorded, not assumed. If the wright reported
   an approach it tried and backed out of, record it under a `## Dead ends` section in
   `decisions.md` so a retry or the next agent doesn't repeat it.
   **If the wright's `Conventions` field reports a contradiction** (the live code now
   disagrees with a held convention), record the drift. You own this bookkeeping:
   ```bash
   devrites-engine conventions contradict --key <key> --slug <slug> \
     --evidence "<what the live code does>" --drift-file .devrites/work/<slug>/drift.md
   ```
   It lowers the convention's band (or retires it) and appends a `convention-drift` entry to
   `drift.md`. The ledger is **promoted only at `$rite-seal` on GO**: build only records
   contradictions, never new conventions. **Evidence is the
   wright's real command output, not its say-so.** Capture per
   [`evidence-standard.md`](evidence-standard.md). Append a footprint record for this
   slice's wright dispatch (deterministic run-weight bookkeeping the seal reports):
   ```bash
   devrites-engine footprint log <slug> wright "<slice id>"
   ```
   If `.devrites/AFK` is present, decrement
   the budget by running `devrites-engine tick-afk <state.md path>`:
   it decrements `state.md`'s `AFK slices remaining` field, prints the new value, and exits `3`
   when it hits 0. **Exit 3 → STOP** (forced HITL stop; the cap is exhausted). Never rewrite
   `.devrites/AFK` `max_slices` in place. It is read-only initial budget.
7. **STOP.** Render the progress footer, then report and recommend the next step. Run the
   shared footer (mirror of the step-0 preamble) so the slice meter + flow ribbon are
   deterministic, not model-typed:
   ```bash
   devrites-engine progress
   ```
   It reads `state.md` (already updated in step 6), so the meter reflects the slice you
   just built. **When every slice is built** (`✅ ALL BUILT`) say so explicitly (the build
   phase is complete, the next phase is `$rite-prove`) don't leave completion implicit.

> **Mid-flight discipline.** The wright must resist doing two slices, skipping TDD, adding a defensive check, or wandering outside `touched-files.md`; you must resist skipping the post-return `devrites-doubt` because the wright "seems confident": see [`anti-patterns.md`](anti-patterns.md). Load it the moment you reach for the excuse.

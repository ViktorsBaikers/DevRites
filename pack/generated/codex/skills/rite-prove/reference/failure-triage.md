# Failure triage

When a test/build/runtime/browser check fails, triage before fixing. Delegates the deep
loop to `devrites-debug-recovery`.

## Triage ladder
1. **Reproduce:** for a repeatable command, run it again and capture the exact
   error (quote it). For a consumptive action under
   [`one-shot-actions.md`](../../devrites-lib/reference/standards/one-shot-actions.md),
   never rerun it: use the retained bounded artifact as the reproduction input.
2. **Classify** the failure:
   - test is right, code is wrong → fix the code (in scope).
   - test is wrong/outdated → that may be **spec drift** (acceptance criteria wrong).
   - environment/setup (missing dep, wrong node/ruby) → fix it when agent-owned;
     record a blocker only for human/access ownership, scope overflow, or exhausted
     fingerprint recovery.
   - pre-existing (fails without this candidate) → only with baseline evidence: the
     same command/cwd/environment run on the baseline candidate (an earlier
     `EVID-###`, or a base-ref run in a separate worktree), recorded. It is a named
     blocker citing that baseline, never green; without the baseline it is unresolved.
   - flaky (at least one fail and one pass on the unchanged candidate digest;
     timing/order) → record every attempt and apply
     [the Determinism standard](../../devrites-lib/reference/standards/testing.md#determinism-no-flaky-tests):
     the green attempt proves nothing; fix it or record a blocker. Don't paper over with retries.
   - external dependency down → record blocker; don't fake the result.
3. **Isolate:** minimize to the smallest failing case; bisect the diff if needed.
   If consumptive evidence maps to multiple causal seams, this is a
   diagnostic-amplification plan gap: design and Vet an injective boundary
   discriminator offline instead of guessing a runtime correction or declaring
   the cleaned-away past state terminal.
4. **Fix within scope:** the current slice/feature only. If the fix reaches outside
   scope, stop and record a blocker in `state.md` (then `$rite-plan unblock`).
5. **Re-run** the same command when it is repeatable, after an admitted
   correction; confirm green and record both attempts in `evidence.md`. A green
   rerun with no correction on an unchanged digest is `flaky` (step 2), never a
   close. A consumptive action needs re-vetted evidence
   completeness and any fresh authorization before a new attempt. Its spent
   action authorization does not stop offline recovery: a retained new fingerprint
   enters caller-owned diagnosis, correction, fixtures, and narrow Vet immediately.
   A separately authorized diagnostic-amplification attempt may acquire the missing
   discriminator after its evidence design passes Vet; it is not a blind rerun.

## Rules
- Quote the real error text; don't paraphrase it away.
- Don't loosen/delete a failing assertion to get green: investigate whether it's drift.
- Don't add blanket retries/sleeps to hide flakiness.
- Three no-progress attempts on the exact same causal fingerprint consume the
  shared recovery cap. A recheck that closes the reproduction is progress; a
  different evidenced Critical/Important invariant gets its own fingerprint.
  Preserve the reproduction and dead ends, then stop once as a technical blocker.
  Ask only when the remaining decision is human-owned.
- A failure you can't fix in scope is a recorded **blocker**, not a silent skip.

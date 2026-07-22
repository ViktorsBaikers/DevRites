# Failure triage

When a test/build/runtime/browser check fails, triage before fixing. Delegates the deep
loop to `devrites-debug-recovery`.

## Triage ladder
1. **Reproduce:** run the failing command again; capture the exact error (quote it).
2. **Classify** the failure:
   - test is right, code is wrong → fix the code (in scope).
   - test is wrong/outdated → that may be **spec drift** (acceptance criteria wrong).
   - environment/setup (missing dep, wrong node/ruby) → fix setup or record blocker.
   - flaky (passes on re-run, timing/order) → note it; don't paper over with retries.
   - external dependency down → record blocker; don't fake the result.
3. **Isolate:** minimize to the smallest failing case; bisect the diff if needed.
4. **Fix within scope:** the current slice/feature only. If the fix reaches outside
   scope, stop and record a blocker in `state.md` (then `/rite-plan unblock`).
5. **Re-run** the same command; confirm green; record both attempts in `evidence.md`.

## Rules
- Quote the real error text; don't paraphrase it away.
- Don't loosen/delete a failing assertion to get green: investigate whether it's drift.
- Don't add blanket retries/sleeps to hide flakiness.
- Three failed fix attempts on the same root cause → escalate: `devrites-debug-recovery`
  for a hypothesis-driven pass, or ask the user.
- A failure you can't fix in scope is a recorded **blocker**, not a silent skip.

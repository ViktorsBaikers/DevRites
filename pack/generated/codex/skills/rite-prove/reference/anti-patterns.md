# rite-prove — anti-patterns

Load this when standing a non-trivial proof decision, or when tempted to
report a pass you didn't observe, skip a rung of the browser-proof ladder,
or proceed without all slices built.

Pack-wide rationalizations + red flags (incl. observed-but-not-recorded):
see [standards/anti-patterns.md](../../devrites-lib/reference/standards/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "Tests passed during build; that's enough." | Build = per-slice sanity check. Prove = whole-feature suite + build/typecheck/lint + browser. |
| "Browser proof is overkill for this UI change." | If it's UI and a browser can run, the ladder applies. If you can't, document which rung and why. |
| "Failure is in pre-existing code, not in this feature." | Still surfaces as a blocker. Feature scope = fix in scope or record + halt. |
| "Some slices are still pending but the rest can be proved." | No — `$rite-prove` runs once, when every slice is built. Otherwise you're proving a partial system. |

## Red Flags

- Reporting "pass" for a command you didn't actually run.
- Skipping browser-proof rungs 1–3 because rung 4 (project E2E) was easier.
- An acceptance criterion left "unproven" without a corresponding blocker note.
- Tests recorded green but the diff shows the test file wasn't touched (stale run).
- About to write `evidence.md` without command + output.

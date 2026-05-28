# rite-seal — anti-patterns

Load this when standing the final GO/NO-GO, when tempted to round NO-GO up
to GO, or before any irreversible action (push, tag, deploy).

Pack-wide rationalizations + red flags (incl. user-wants-it-bypass-gate):
see [rules/anti-patterns.md](../../../rules/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "All Important findings are minor; just GO." | `Important > 0` triggers an interactive y/N — do not bypass the prompt, present it. |
| "User clearly wants this to ship." | Ship is evidence-driven, not consent-driven. A Critical finding outranks "the user said go". |
| "Drift is small; resolve it inline and seal." | Unresolved drift = NO-GO, no exceptions. Route through `/rite-plan` to repair, then seal. |
| "Reviewer findings overlap; average them away." | Surface disagreements explicitly. Averaging hides where the reviewers actually disagree. |
| "I'll push the tag now; type-GO is a formality." | Type-GO is the last safety net before an irreversible action. **Always render the prompt.** |

## Red Flags

- About to write `Verdict: GO` while any `Critical` finding is open.
- About to run `git push` or `git tag` without rendering the type-GO prompt.
- Reviewer subagents spawned with the author's reasoning attached (defeats the fresh-context point).
- `seal.md` written before walking *every* acceptance criterion one by one.
- A NO-GO rounded up to GO "to be agreeable".
- Unresolved drift, unresolved questions, or pending tasks — and you're sealing anyway.

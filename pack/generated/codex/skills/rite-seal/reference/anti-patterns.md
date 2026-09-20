# rite-seal: anti-patterns

Load this when standing the final GO/NO-GO decision, or when tempted to round
NO-GO up to GO. (The irreversible push/tag/deploy and its type-GO gate now live in
`$rite-ship`: see that skill's anti-patterns for the type-GO discipline.)

Pack-wide rationalizations + red flags (incl. user-wants-it-bypass-gate):
see [standards/anti-patterns.md](../../devrites-lib/reference/standards/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "All Important findings are minor; just GO." | `Important > 0` triggers an interactive y/N: do not bypass the prompt, present it. |
| "User clearly wants this to ship." | Ship is evidence-driven, not consent-driven. A Critical finding outranks "the user said go". |
| "Drift is small; resolve it inline and seal." | Unresolved drift = NO-GO, no exceptions. Route through `$rite-plan` to repair, then seal. |
| "Reviewer findings overlap; average them away." | Surface disagreements explicitly. Averaging hides where the reviewers disagree. |
| "Carry the old reviewer verdicts though the diff changed." | If the diff changed since `$rite-review`, re-run the Spec + Code axes: don't carry stale verdicts forward. |
| "The wright was confident in a material decision. GO without independent review." | Confidence is not independent evidence. Ask the exact `devrites-doubt-reviewer` to test a boundary / data-model / auth / public-API / migration claim, reconcile its result against source, and record the verdict in `decisions.md`; unresolved irreversible risk is NO-GO. |

## Red Flags

- About to write `Verdict: GO` while any `Critical` finding is open.
- Writing `Next step` as anything other than `$rite-ship` after a GO verdict.
- Reviewer subagents spawned with the author's reasoning attached (defeats the fresh-context point).
- `seal.md` written before walking *every* acceptance criterion one by one.
- A NO-GO rounded up to GO "to be agreeable".
- Unresolved drift, unresolved questions, or pending tasks, and you're sealing anyway.
- A stood irreversible-risk decision (auth / public-API / migration) has no recorded independent `devrites-doubt` verdict, and you are sealing GO anyway.

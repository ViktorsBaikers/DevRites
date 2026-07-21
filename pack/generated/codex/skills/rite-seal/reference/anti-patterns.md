# rite-seal — anti-patterns

Load this when standing the final GO/NO-GO decision, or when tempted to round
NO-GO up to GO. (The irreversible push/tag/deploy and its type-GO gate now live in
`$rite-ship` — see that skill's anti-patterns for the type-GO discipline.)

Pack-wide rationalizations + red flags (incl. user-wants-it-bypass-gate):
see [standards/anti-patterns.md](../../devrites-lib/reference/standards/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "All Important findings are minor; just GO." | `Important > 0` triggers an interactive y/N — do not bypass the prompt, present it. |
| "User clearly wants this to ship." | Ship is evidence-driven, not consent-driven. A Critical finding outranks "the user said go". |
| "Drift is small; resolve it inline and seal." | Unresolved drift = NO-GO, no exceptions. Route through `$rite-plan` to repair, then seal. |
| "Reviewer findings overlap; average them away." | Surface disagreements explicitly. Averaging hides where the reviewers actually disagree. |
| "Carry the old reviewer verdicts though the diff changed." | If the diff changed since `$rite-review`, re-run the Spec + Code axes — don't carry stale verdicts forward. |
| "Build never doubted a decision, but the wright seemed confident — GO without checking." | Doubt is a gate, not a vibe. `doubt-coverage` rc=1 — or a stood boundary / data-model / auth / public-API / migration decision in `decisions.md` with no recorded `devrites-doubt` verdict — is an Important finding, escalating to NO-GO when the undoubted decision is irreversible-risk (auth / public-API / migration). An empty `Decisions stood` on every slice passes only when confirmed against `decisions.md`, never assumed. |

## Red Flags

- About to write `Verdict: GO` while any `Critical` finding is open.
- Writing `Next step` as anything other than `$rite-ship` after a GO verdict.
- Reviewer subagents spawned with the author's reasoning attached (defeats the fresh-context point).
- `seal.md` written before walking *every* acceptance criterion one by one.
- A NO-GO rounded up to GO "to be agreeable".
- Unresolved drift, unresolved questions, or pending tasks — and you're sealing anyway.
- `doubt-coverage` rc=1 left unexamined, or a stood irreversible-risk decision (auth / public-API / migration) carrying no recorded `devrites-doubt` verdict — and you're sealing GO anyway.

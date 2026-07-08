# $rite-adopt — anti-patterns

Start with the universal table in [`../../devrites-lib/reference/standards/anti-patterns.md`](../../devrites-lib/reference/standards/anti-patterns.md).
Below are the rationalizations specific to onboarding an existing codebase.

## Adoption-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "I'll improve this messy bit while I document it." | Adoption *documents* what exists; it changes nothing. A cleanup is the *next* feature through the normal lifecycle — recording it as a drive-by here hides the baseline and balloons the diff. |
| "This looks like the convention, so I'll seed it." | Seed only what the code *actually and consistently* follows. A guessed seed becomes a high-confidence-looking prior that steers the first slice wrong. Observe it in ≥2 places or leave it out. |
| "I'll spec the ideal architecture, not the current one." | The spec's baseline is *current behavior*. Aspirations go in the next objective, not the baseline — otherwise every later slice drifts against a spec that never matched reality. |
| "Whole repo in one adopt." | Onboard the area you'll actually work in. A repo-wide adopt produces an unfocused spec and a ledger full of conventions no slice will touch. |

## Red flags

- A seeded convention with no concrete code location behind it.
- `spec.md` describing behavior the code does **not** currently have (that's the next feature, not the baseline).
- Editing source during adoption — adoption writes `.devrites/` + the ledger, never project code.

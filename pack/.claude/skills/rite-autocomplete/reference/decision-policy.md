# Decision policy: how autocomplete "chooses the best option"

Unattended ≠ careless. Every discretionary choice is made the way a senior engineer
would, then **recorded** so the seal can audit it. The difference from interactive mode
is only *who answers* a discretionary question, not whether it's reasoned.

## The rule

For each non-trivial choice, pick the option the **relevant specialist favours**, and
write the choice + the one-line rationale to `decisions.md`. Do not coin-flip, and do
not pick the option that's merely easiest to implement.

| Choice type | Defer to | Then |
|---|---|---|
| Uncertain framework / library behaviour | `devrites-source-driven` (check the docs/source) | record the source in `evidence.md` |
| Risky boundary / data-model / API decision | `devrites-doubt` (+ `devrites-doubt-reviewer`) | record the trade-off in `decisions.md` |
| Cross-boundary contract (FE/BE split, API shape) | `devrites-api-interface` | contract-first slice |
| UI surface / states / a11y | `devrites-frontend-craft` | shape before code |
| Two reasonable designs, no clear winner | the simpler one that meets acceptance | note why in `decisions.md` |

## Defaults autocomplete may assume (record them)

- Follow the project's existing conventions, components, and test commands: always.
- Prefer the smallest vertical slice that proves the acceptance criterion.
- Reuse → extend → build new, in that order (`coding-style.md`).
- When two options are equivalent on the evidence, choose the lower-risk / more
  reversible one.

## When a choice is NOT autocomplete's to make

If a discretionary choice would change **product behaviour, scope, data model, security
posture, or acceptance criteria**, it is no longer discretionary. It is a material
question. Route it through the Spec Drift Guard / a `blocking` question and **stop**
(see [stop-conditions.md](stop-conditions.md)). Autocomplete answers *how*, never
*what to build* beyond what the up-front interview settled.

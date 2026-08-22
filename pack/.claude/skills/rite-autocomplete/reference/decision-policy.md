# Decision policy: how autocomplete "chooses the best option"

Unattended ≠ careless. Every discretionary choice is made the way a senior engineer
would, then **recorded** so the seal can audit it. The difference from interactive mode
is only *who answers* a discretionary question, not whether it's reasoned.

Authority: `.claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → keep Plan repair/affected Vet internal; no stop solely for topology/count.
- `GUARD_AND_REPAIR` → enter Spec Drift Guard/Clarify; pause only at an existing human-owned gate; resume Plan/Vet internally.
- `BLOCKED_INPUT` → no planning writes; stop internal branch; exact diagnostic; recover authority; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

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

Human-owned contract, data-model, security, and product choices are not discretionary.
Use owning elicitation/safety gates and the topology marked action. Autocomplete
answers *how*, never *what to build* beyond the settled interview.

# /rite-vet: anti-patterns

Read [universal anti-patterns](../../devrites-lib/reference/standards/anti-patterns.md).

Authority: `.omp/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → fold technical topology; invalidate Vet/readiness; affected Vet before Build.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → affected Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Phase rationalizations

| Excuse | Rebuttal |
|---|---|
| "I'll write all the findings into `eng-review.md` and let the user read it." | Fold verified behavior-preserving findings together and recheck affected closure. Ask human-owned choices with recommendations and await answers. Recording alone resolves nothing; never silently change acceptance or mark unresolved choices READY. |
| "The plan doesn't handle X: flag it." (without checking) | Quote the line first. If you can't quote where the plan handles X, you also can't be sure it doesn't: force confidence ≤4 and suppress. The "field/case doesn't exist" finding is the most common false positive; the verification gate exists to kill it. |
| "It's an obvious fix. I'll just edit the plan." | Vet hardens verified plans; topology uses the marked action. |
| "This dimension is thin but the others are strong: net it's fine." | The gate is the **floor**, not the average. A plan `broken` on test-coverage does not pass because it's `strong` on architecture. |
| "I can see it's adequate. I'll note the band, the evidence is obvious." | Cite quoted evidence **before** the band; never score first and rationalize later. |
| "8+ files but it all needs to change: proceed." | Apply `review-axes.md` §0: justify complexity against the smallest contract-complete alternative; fold technical reductions through the marked action. File count does not justify design or require a pause. Ask only for changed scope, behavior, or risk policy. |
| "There's no test for this, but the user can add one later." | Coverage is designed **now**, before the code, so the build writes tests alongside it. A deferred test is a test that misses the boundary cases writing-it-now would expose. Regressions are non-negotiable. Critical, no question. |
| "Performance looks fine." | "Looks fine" is not a measurement. A perf finding is "measure X against budget Y" or it's nothing: don't recommend speculative tuning, and don't wave past an N+1 you can quote. |
| "Cross-model agrees, so apply it." | Consensus is a signal, not proof or approval. Verify the finding against source, then use the same technical-hardening or human-owned decision route; model agreement never authorizes a scope, acceptance, or policy change. |

## Reslice classifier rationalizations

**RSLICE-VET-001** — Slice/topology growth alone does not stop; fewer slices do not prove scope reduction; remediation does not grow product.

## Red flags
- `eng-review.md` / `test-plan.md` written but `plan.md` / `tasks.md` **not** hardened: the build
  follows the plan, so the review changed nothing.
- A finding raised with confidence ≥7 but **no quoted source line**, that defeats the verification gate.
- Acceptance/product-behavior growth auto-applied in AFK, or an orthogonal human-owned gate that did not pause.
- A failure-mode table with rows but no verdict column filled: a "no test + no handling + silent" row not marked Critical.
- Repeating an exhausted causal failure under the shared no-progress rule in
  [`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md), or stopping a
  distinct cause solely because the total review round exceeds three.
- The reviewer / cross-model being handed the author's reasoning as independent
  evidence (defeats the fresh-context point).
- The skill writing code, implementing a slice, or running the build. That's `/rite-build`. Vet reviews and hardens the plan; it never implements.
- Re-litigating the **spec's** scope/ambition, that was `/rite-temper`. Vet challenges *implementation* scope only.

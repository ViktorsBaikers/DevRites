# /rite-vet: anti-patterns

Universal rationalizations + red flags live in
[`../../devrites-lib/reference/standards/anti-patterns.md`](../../devrites-lib/reference/standards/anti-patterns.md): read it when the
reluctance is broader than this phase. Below are the vet-specific ones.

Authority: `.omp/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → fold technical topology; invalidate Vet/readiness; affected Vet before Build.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → affected Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Phase rationalizations

| Excuse | Rebuttal |
|---|---|
| "I'll write all the findings into `eng-review.md` and let the user read it." | The artifact is the *output* of an interactive review, not a substitute for it. A material finding's path to "done" goes **through `AskUserQuestion`**, one at a time, with a recommendation + why. Dumping findings into one write and moving on is the exact failure this skill exists to prevent. |
| "The plan doesn't handle X: flag it." (without checking) | Quote the line first. If you can't quote where the plan handles X, you also can't be sure it doesn't: force confidence ≤4 and suppress. The "field/case doesn't exist" finding is the most common false positive; the verification gate exists to kill it. |
| "It's an obvious fix. I'll just edit the plan." | Verified plan hardening is Vet's job; topology uses the marked action. |
| "This dimension is thin but the others are strong: net it's fine." | The gate is the **floor**, not the average. A plan `broken` on test-coverage does not pass because it's `strong` on architecture. |
| "I can see it's adequate. I'll note the band, the evidence is obvious." | Cite the quoted evidence **before** the band. Score-first-justify-later is how a reviewer waves through a plan it already likes. |
| "8+ files but it all needs to change: proceed." | The complexity smell is a **STOP-and-ask**, not a note. Name the overbuilt part, propose the smaller version that still meets acceptance, and let the human decide before any axis runs. Maybe it's justified (record *why*) but the gate fires first. |
| "There's no test for this, but the user can add one later." | Coverage is designed **now**, before the code, so the build writes tests alongside it. A deferred test is a test that misses the boundary cases writing-it-now would expose. Regressions are non-negotiable. Critical, no question. |
| "Performance looks fine." | "Looks fine" is not a measurement. A perf finding is "measure X against budget Y" or it's nothing: don't recommend speculative tuning, and don't wave past an N+1 you can quote. |
| "Cross-model agrees, so apply it." | Cross-model consensus is a strong *signal*, not an approval. Every cross-model finding is informational until the human approves it (or, in AFK, until the gate ceiling clears a hardening-only change). |

## Reslice classifier rationalizations

**RSLICE-VET-001** — Slice/topology growth alone does not stop; fewer slices do not prove scope reduction; remediation does not grow product.

## Red flags
- `eng-review.md` / `test-plan.md` written but `plan.md` / `tasks.md` **not** hardened: the build
  follows the plan, so the review changed nothing.
- A finding raised with confidence ≥7 but **no quoted source line**, that defeats the verification gate.
- Acceptance/product-behavior growth auto-applied in AFK, or an orthogonal human-owned gate that did not pause.
- A failure-mode table with rows but no verdict column filled: a "no test + no handling + silent" row not marked Critical.
- The reviewer loop running past 3 iterations, or the reviewer / cross-model being handed the author's reasoning (defeats the fresh-context point).
- The skill writing code, implementing a slice, or running the build. That's `/rite-build`. Vet reviews and hardens the plan; it never implements.
- Re-litigating the **spec's** scope/ambition, that was `/rite-temper`. Vet challenges *implementation* scope only.

# /rite-temper — anti-patterns

Universal rationalizations + red flags live in
[`../../devrites-lib/reference/standards/anti-patterns.md`](../../devrites-lib/reference/standards/anti-patterns.md) — read it when the
reluctance is broader than this phase. Below are the temper-specific ones.

## Phase rationalizations

| Excuse | Rebuttal |
|---|---|
| "I'll write all the findings into `strategy.md` and let the user read it." | The artifact is the *output* of an interactive review, not a substitute for it. A finding's path to "done" goes **through `AskUserQuestion`**, one at a time, with a recommendation + why. Dumping findings into one write and moving on is the exact failure mode this skill exists to prevent. |
| "Bigger is better — let's add the extra capability while we're thinking big." | Ambition belongs on the **outcome**, not the **solution surface**. A speculative capability/abstraction is gold plating. Expand the problem + acceptance; never the machinery. |
| "The user obviously wants the ambitious version — I'll add it to the spec." | Expansion is **opt-in**, recorded, and confirmed (HITL) or paused (AFK). Adding acceptance the human didn't approve is silent scope growth — route it through the Spec Drift Guard or don't add it. |
| "It's over-scoped, I'll just trim those criteria." | Reducing scope is **still a spec change**. Cut acceptance only as a recorded Spec-Drift-Guard decision, and park the rest in Non-goals with a revisit note — never a silent trim. |
| "This dimension is weak but the others are strong — net it's fine." | The gate is the **floor**, not the average. A spec `broken` on risk does not pass because it's `strong` on clarity. |
| "I can see it's adequate — I'll note the band, evidence is obvious." | Cite the evidence (the spec line or its absence) **before** the band. Score-first-justify-later is how a reviewer rationalizes a spec it already likes. |
| "It's a strategy doc, the implementation concerns don't apply yet." | Cross-cutting concerns (security/data/observability/modifiability) are where strategy breaks down. Address each or mark it explicitly N/A — silent omission is a gap. |
| "This greenfield design is elegant." | Elegant for a codebase that doesn't exist. Ground ambition in the real seams + blast radius; flag a new dependency / second design system for an explicit decision. |

## Red flags
- `strategy.md` written but `spec.md` acceptance/Non-goals **not** updated — the build follows
  the spec, so the review changed nothing.
- An `expand` decision auto-applied in AFK (it must pause), or any irreversible-risk touch that
  didn't pause.
- A pre-mortem with risks listed but no mitigation + owning slice — a risk without an owner is a
  wish.
- The reviewer loop running past 3 iterations, or the reviewer being handed the author's
  reasoning (defeats the fresh-context point).
- The skill writing a `plan.md`, slicing, or touching code — that's `/rite-define` / `/rite-build`.

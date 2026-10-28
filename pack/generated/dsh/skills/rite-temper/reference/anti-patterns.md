# /rite-temper: anti-patterns

For cross-phase issues, read [universal anti-patterns](../../devrites-lib/reference/standards/anti-patterns.md).

## Phase rationalizations

| Excuse | Rebuttal |
|---|---|
| "I'll write all the findings into `strategy.md` and let the user read it." | Fold and recheck objective corrections preserving scope, behavior, and risk policy. Ask human-owned strategy/acceptance choices with recommendations and await answers. Recording alone resolves nothing; never silently expand or trim the spec. |
| "Bigger is better: let's add the extra capability while we're thinking big." | Ambition belongs on the **outcome**, not the **solution surface**. A speculative capability/abstraction is gold plating. Expand the problem + acceptance; never the machinery. |
| "The user obviously wants the ambitious version. I'll add it to the spec." | Expansion is **opt-in**, recorded, and confirmed (HITL) or paused (AFK). Adding acceptance the human didn't approve is silent scope growth: route it through the Spec Drift Guard or don't add it. |
| "It's over-scoped, I'll just trim those criteria." | Reducing scope is **still a spec change**. Cut acceptance only as a recorded Spec-Drift-Guard decision, and park the rest in Non-goals with a revisit note: never a silent trim. |
| "This dimension is weak but the others are strong: net it's fine." | The gate is the **floor**, not the average; strong clarity cannot offset broken risk. |
| "I can see it's adequate. I'll note the band, evidence is obvious." | Cite spec evidence **before** the band; never score first and rationalize later. |
| "It's a strategy doc, the implementation concerns don't apply yet." | Cross-cutting concerns (security/data/observability/modifiability) are where strategy breaks down. Address each or mark it explicitly N/A: silent omission is a gap. |
| "This greenfield design is elegant." | Ground ambition in live seams/blast radius; a new dependency or second design system needs an explicit decision. |

## Red flags
- An accepted strategy change recorded in `strategy.md` but not reflected in its
  owning spec acceptance/Non-goals. A clean review does not require gratuitous spec edits.
- An `expand` decision auto-applied in AFK, an unapproved risk-policy change, or an
  irreversible operation without its required authorization.
- A pre-mortem with risks listed but no mitigation + owning slice: a risk without an owner is a
  wish.
- Repeating an exhausted causal failure under the shared no-progress rule in
  [`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md), or stopping a
  distinct cause solely because the total review round exceeds three.
- The reviewer being handed the author's reasoning as independent evidence
  (defeats the fresh-context point).
- The skill writing a `plan.md`, slicing, or touching code. That's `/rite-define` / `/rite-build`.

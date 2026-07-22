# Review dimensions + the floor-gate rubric

The spec is scored on nine dimensions. Each catches a *different* failure mode: strength in
one never compensates for collapse in another, so the gate is the **floor**, not an average.

## The nine dimensions
1. **Problem altitude & ambition:** is this solving the *right* problem at the right altitude,
   or timidly improving the obvious path / over-reaching past the real need? Here, *under-
   reaching* on the outcome is a failure, not just over-building.
2. **Scope honesty & boundary:** are Non-goals explicit? Is there a Minimum Usable Subset and a
   clear IN/OUT line? Flags grab-bags and a missing "Won't-have-this-time".
3. **Premise & alternatives:** are the load-bearing premises stated and challenged, and is at
   least one genuinely different approach (different data model / flow / boundary) considered
   with its trade-off? ("Alternatives considered" is one of the most useful parts of a spec.)
4. **Pre-mortem risk coverage:** run in prospective hindsight; each top risk carries likelihood
   + mitigation + the slice that owns it. **Unmitigated top risks are gating.**
5. **Over-engineering / YAGNI:** speculative capability, unused extension points, premature
   abstraction, second-system bloat. Apply the pack's standard (`patterns.md`) + the "imagine
   the later refactor" test. Defer unless now-cost is trivial AND deferred-cost is large.
6. **Acceptance testability & done-ness:** every acceptance criterion measurable, technology-
   agnostic, and comparable **down to a baseline** ("better than the current workaround"), not
   up to an unbounded ideal. Vague adjectives ("fast/robust/intuitive") and "handles X
   gracefully" are flagged.
7. **Irreversibility & blast radius:** are auth / migration / public-API / data-model touches
   treated with conservatism + rollback, and is the blast radius understood (from a
   code-intelligence index if available)?
8. **Cross-cutting coverage:** security/trust-boundary, data & migration, observability, and
   modifiability each explicitly addressed **or explicitly N/A**: no silent omission.
9. **Convention fit & placement realism:** does the ambition fit the codebase's existing seams
   and patterns, or assume greenfield freedom a constrained space doesn't have? Prefer existing
   conventions; flag a new dependency / second design system for explicit decision.

UI specs also inherit `rite-polish/reference/anti-ai-slop.md` at the cross-cutting
dimension (design-system fit, anti-AI-slop), but `/rite-temper` reviews the *spec/design-brief*,
not pixels; the built UI is judged later by `devrites-frontend-reviewer`.

## The rubric: coarse bands, evidence first
Score each dimension on a **labeled band**, never a 1-10 float or a composite number (labels do
the work: matches the pack's severity-vocabulary stance):

| Band | Meaning |
|---|---|
| **strong** | Addressed well; no action needed. |
| **adequate** | Addressed; minor sharpening only (Suggestion/Nit). |
| **thin** | Under-addressed; a real gap to close before planning (Important). |
| **broken** | Missing or wrong in a way that will cost a redo (Critical). |

**Cite evidence before the band**: name the spec line / absence that justifies it; never score
first and rationalize after. Borderline dimensions may be sampled twice; take the **lower** band
(don't average up).

## The floor-gate
- **Pass** only when every dimension is `adequate` or `strong` **and** no unmitigated top
  pre-mortem risk remains. The verdict is the **weakest** dimension: a spec strong on clarity
  but `broken` on risk does **not** pass.
- Individual findings carry the pack's standard severity labels (**Critical / Important /
  Suggestion / Nit / FYI**) so they compose with `/rite-review` and `/rite-seal`. `broken` →
  Critical, `thin` → Important; **FYI is band-independent**: an observation on a `strong` /
  `adequate` dimension, not a failing band.
- **Don't double-count overlap.** Some dimensions key on the same evidence (e.g. #7
  irreversibility & #9 convention-fit both touch codebase realism; #1 ambition & #2 scope both
  touch over/under-reach). Don't fail two dimensions for the *same* root cause: cite the
  evidence once, band it once, so one problem isn't laundered into two floor failures.
- Below-bar dimension after the ≤3-iteration reviewer loop → blocking question (HITL) or AFK
  gate-ceiling entry. Irreversible-risk findings always pause regardless of band.

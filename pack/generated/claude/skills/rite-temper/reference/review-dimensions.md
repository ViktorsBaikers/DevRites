# Review dimensions and floor-gate rubric

Score the spec on nine dimensions. A strong dimension cannot offset a failed one, so
the gate uses the **lowest band**, not an average.

## The nine dimensions
1. **Problem choice and ambition:** does the spec solve the underlying problem, or only
   make a minor improvement to the obvious path? It may also reach beyond the actual need.
   Both insufficient outcome ambition and over-building are failures.
2. **Scope and boundary:** are non-goals explicit? Is there a Minimum Usable Subset and
   a clear IN/OUT line? Flag unrelated bundles and a missing "Won't-have-this-time".
3. **Premise and alternatives:** are the consequential premises stated and challenged, and is at
   least one genuinely different approach (different data model / flow / boundary) considered
   with its trade-off? ("Alternatives considered" is one of the most useful parts of a spec.)
4. **Pre-mortem risk coverage:** assess the feature as if it shipped and failed. Each top risk carries likelihood
   + mitigation + the slice that owns it. The interruption pre-mortem accounts for foreseeable
   human prerequisites and action-time approvals instead of deferring generic uncertainty to
   build. **Unmitigated top risks are gating.**
5. **Over-engineering / YAGNI:** speculative capability, unused extension points, premature
   abstraction, second-system bloat. Apply the pack's standard (`patterns.md`) + the "imagine
   the later refactor" test. Defer unless now-cost is trivial AND deferred-cost is large.
6. **Acceptance testability and completion:** every acceptance criterion measurable, technology-
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

## Rubric
Score each dimension on a **labeled band**, never a 1-10 float or composite number:

| Band | Meaning |
|---|---|
| **strong** | Addressed well; no action needed. |
| **adequate** | Addressed; minor sharpening only (Suggestion/Nit). |
| **thin** | Under-addressed; a real gap to close before planning (Important). |
| **broken** | Missing or wrong in a way that will cost a redo (Critical). |

**Cite evidence before assigning the band.** Name the spec line or absence that
justifies it. Review a borderline dimension twice if needed and use the **lower** band.

## The floor-gate
- **Pass** only when every dimension is `adequate` or `strong` **and** no unmitigated top
  pre-mortem risk remains. The verdict is the **weakest** dimension: a spec strong on clarity
  but `broken` on risk does **not** pass.
- Individual findings carry the pack's standard severity labels (**Critical / Important /
  Suggestion / Nit / FYI**) so they compose with `/rite-review` and `/rite-seal`. `broken` →
  Critical, `thin` → Important; **FYI is band-independent**: an observation on a `strong` /
  `adequate` dimension, not a failing band.
- **Do not double-count overlap.** Some dimensions use the same evidence (e.g. #7
  irreversibility & #9 convention-fit both touch codebase realism; #1 ambition & #2 scope both
  touch over/under-reach). Don't fail two dimensions for the *same* root cause: cite the
  evidence once and assign one band so one problem does not create two floor failures.
- Below-bar dimension after the ≤3-iteration reviewer loop → classify by decision ownership:
  product/scope/risk uncertainty becomes a blocking question; an objective prose/coverage defect
  routes back to `/rite-spec` without inventing a human decision. Irreversible-risk findings
  always pause regardless of band.

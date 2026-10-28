# Visual playbook: comparison

## use_when

Show options, tradeoffs, before/after, or mutually exclusive directions so a human can choose or validate a recommendation.

## Structure / landmarks / stable ids

| Landmark | Suggested `id` | Role |
| --- | --- | --- |
| Decision statement | `cmp-decision` | Named decision at the top |
| Option / side A | `cmp-option-a` (or `cmp-before`) | Concrete behavior / shape |
| Option / side B | `cmp-option-b` (or `cmp-after`) | Aligned counterpart |
| Scorecard (optional) | `cmp-scorecard` | Only when criteria are explicit |
| Recommendation | `cmp-recommend` | Only when evidence supports one |
| Assumptions | `cmp-assumptions` | What would change the call |

Align corresponding details across options so differences are visible without hunting. End with a recommendation only when evidence supports it; otherwise list open questions.

If the human must pick, also load [`input.md`](input.md).

## design_rules

- Keep primary tradeoffs visually above secondary notes.
- Make costs as visible as benefits.
- Prefer concrete examples (behavior, API shape, UX mock) over vague pros/cons.
- Self-contained CSS; CDN only if a nested diagram/code surface requires it (note in outline).
- Explicit background / color-scheme; stable ids on each option card.

## Pitfalls / anti-patterns

- Making every option look equally recommended when one is preferred.
- Comparing vague summaries when concrete examples exist.
- Burying assumptions that flip the recommendation.
- Requiring Lavish queue/select APIs for the comparison to function.
- HTML without outline.

## DevRites notes

- **Home:** `.devrites/work/<slug>/visual/<name>.html` + `<name>.outline.md`.
- Outline template: [`outline-template.md`](outline-template.md). Capture tradeoffs in `## Relationships` and assumptions there or via `questions.md` pointers.
- Often pairs with `plan` or `diagram` — open every matching playbook ([`index.md`](index.md)).

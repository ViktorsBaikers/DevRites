# Visual playbook: slides

## use_when

**Only** when the user asks for a deck, presentation, talk, or paced walkthrough. Default to a scroll page (`plan`, `diagram`, `comparison`, …) for reference material, detailed review, or dense evidence.

## Structure / landmarks / stable ids

| Landmark | Suggested `id` | Role |
| --- | --- | --- |
| Deck root | `slides-root` | Container / scroll-snap or section stack |
| Slide N | `slide-<n>` | One idea per slide |
| Deck nav (optional) | `slides-nav` | Prev/next or index |
| Closing slide | `slide-close` | Decision or next action |

Plan the story before markup: open with the point → context → evidence → decision/next action. Vary composition so consecutive slides do not feel like identical cards unless repetition is intentional.

## design_rules

- Sparse text; let visuals carry explanation.
- Large type, strong alignment, deliberate whitespace — not dense paragraphs.
- Make navigation and screen-size assumptions explicit (e.g. "designed for 16:9 presenter view").
- Prefer self-contained CSS (scroll-snap sections or simple slide panes). CDN only if nested Mermaid/diff needs it; note in outline.
- Stable `id` on every slide for outline inventory and deep links.
- Explicit background / color-scheme on the page and each slide.

## Pitfalls / anti-patterns

- Turning every explainer into slides by default.
- Pasting a scroll-page outline into fixed frames without rewriting the narrative.
- Dense code review inside slides — use [`code.md`](code.md) on a scroll page instead.
- Requiring Lavish runtime for advancement or feedback.
- HTML without outline companion.

## DevRites notes

- **Home:** `.devrites/work/<slug>/visual/<name>.html` + `<name>.outline.md`.
- Outline: [`outline-template.md`](outline-template.md); list each `slide-<n>` in ID inventory with its one idea.
- Feedback/choices on a deck still use [`input.md`](input.md) → outline `## Answers` / `questions.md`, not Lavish queue APIs.
- Router: [`index.md`](index.md).

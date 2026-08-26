# Visual playbook: diagram

## use_when

Explain relationships, flows, state, architecture, and spatial concepts with illustrations — when Mermaid-in-`flows.md` alone is not rich enough for human review.

## Structure / landmarks / stable ids

Recommended page landmarks:

| Landmark | Suggested `id` | Role |
| --- | --- | --- |
| Header | `viz-title` | Question the diagram answers |
| Overview figure | `diagram-overview` | Core relationship only |
| Detail region | `diagram-detail` | Module cards / evidence below overview |
| Legend | `diagram-legend` | Optional symbol key |
| Open questions | `viz-open-questions` | Uncertainties + confidence badges (optional) |
| Citations | `diagram-citations` | Repo paths / commands |

Give every meaningful SVG node, edge, and region a stable `id` (e.g. `node-auth`, `edge-auth-db`) so outline `## ID inventory` and `## Relationships` can mirror them.

Prefer **one concept per figure**. For large systems: small overview illustration + detail cards — not one dense auto-laid graph.

When uncertainty remains, add `id="viz-open-questions"` with 1–few open questions and optional confidence badges (`high` / `medium` / `low`). Mirror them in outline `## Open questions` and in the optional `#devrites-outline` JSON.

## design_rules

- Prefer **hand-authored inline SVG + outline SSOT** for AI/human dual-read. Size with `viewBox` + `width: 100%`; never fixed pixel dimensions; keep elements inside the viewBox.
- Color via `currentColor` and page CSS custom properties so light/dark themes work.
- Short SVG labels (few words); put prose beside the figure in HTML — SVG text does not wrap.
- Figures stay self-contained: no external images/fonts required for the SVG itself.
- Explicit page `background` / `color-scheme`; semantic `header` / `main` / labeled sections.
- Prefer self-contained CSS. CDN only when Mermaid is justified (below) and the outline notes the dependency.
- **Mermaid** remains optional: use only when flowchart / sequence / state is clearer than hand SVG **and** the Mermaid source is embedded for agent read **and** mirrored in the outline. Do not use Mermaid merely to save authoring effort; do not treat Mermaid as a full dual-read replacement DSL.
- Optional but recommended: embed `<script type="application/json" id="devrites-outline">` matching the outline (outline wins on conflict).

## Pitfalls / anti-patterns

- Cramming every file or function into one figure.
- Building boxes-and-arrows from div/flexbox instead of SVG (or justified Mermaid).
- Presenting unverified architecture as fact — cite files or commands.
- Requiring Lavish annotation / poll / whiteboard APIs for the diagram to work.
- Emitting HTML without the sibling `.outline.md`.

## DevRites notes

- **Home:** `.devrites/work/<slug>/visual/<name>.html` + `<name>.outline.md`.
- **Outline companion:** copy headings from [`outline-template.md`](outline-template.md); list this id under `## Playbooks used`.
- **Outline wins** on conflict with HTML (and with `#devrites-outline` JSON) until both regenerate together.
- Keep Mermaid in workspace `flows.md` when that is enough; richer presentation **also** emits `visual/` and may link from `flows.md` (T3 hooks).
- **Consistency:** `open-visual` warns when outline inventory ids are missing from HTML (non-fatal). HTML-only decorative ids are not reported.
- **No new lifecycle phase.** Optional artifact; never readiness-required.
- Router: [`index.md`](index.md).

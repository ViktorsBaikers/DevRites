# Visual playbook: table

## use_when

Turn dense records that share the same fields into a scan-friendly review surface (status matrices, coverage grids, inventory with comparable columns).

## Structure / landmarks / stable ids

| Landmark | Suggested `id` | Role |
| --- | --- | --- |
| Summary | `table-summary` | What the rows prove or require |
| Primary table | `table-main` | Semantic `<table>` for comparable rows |
| Row groups | `table-group-<slug>` | Optional thematic sections |
| Footer notes | `table-notes` | Caveats, filters, open questions |

Stable ids: put `id` on the table and on important rows (`tr id="row-…"`) when a row is a decision target. Mirror row ids in the outline inventory when they matter.

Column groups should follow the decision they support: identity → evidence → status → action.

## design_rules

- Use semantic `<table>` / `<thead>` / `<tbody>` / `<th scope>` when data is tabular.
- Lead with a short summary (counts, risk levels, verdicts) above the grid.
- Protect long paths, symbols, and URLs from overflow (`overflow-wrap`, narrow-viewport friendly).
- Restrained color for status/severity; never color as the only signal (pair with text).
- Prefer self-contained CSS; no CDN required for tables.
- Explicit page background / color-scheme; landmarks with stable ids.

## Pitfalls / anti-patterns

- Pasting a terminal table into HTML unchanged.
- Hiding the conclusion under a large undifferentiated grid.
- Using cards when rows share fields (or tables when shapes differ wildly — use cards then).
- Lavish row-queue / annotation APIs as requirements.
- HTML without outline companion.

## DevRites notes

- **Home:** `.devrites/work/<slug>/visual/<name>.html` + `<name>.outline.md`.
- Outline: [`outline-template.md`](outline-template.md); cite this id under Playbooks used.
- Prefer linking dense evidence from existing artifacts (`traceability.md`, `test-plan.md`) rather than duplicating whole files into the visual.
- Router: [`index.md`](index.md).

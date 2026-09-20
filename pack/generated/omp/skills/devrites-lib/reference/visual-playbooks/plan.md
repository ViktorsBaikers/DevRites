# Visual playbook: plan

## use_when

Explain a product or technical plan before implementation — PRD-style approach, technical design, implementation proposal, or pre-build walkthrough that benefits from layout over prose alone.

Use a lighter `comparison` or `diagram` playbook alone when the plan is only one small design choice.

## Structure / landmarks / stable ids

| Landmark | Suggested `id` | Role |
| --- | --- | --- |
| Goal | `plan-goal` | Desired outcome |
| Current state | `plan-current` | What exists today |
| Desired behavior | `plan-desired` | Target behavior |
| Approach | `plan-approach` | High-level proposal |
| Risks | `plan-risks` | Failure modes / migration / compat |
| Open questions | `plan-questions` | Unresolved; clear when answered |
| Options (optional) | `plan-options` | Follow [`comparison.md`](comparison.md) |

A plan should be self-contained enough that another developer could implement from it. Verify claims against the codebase before stating them as fact.

When frontend UX matters, prefer a visual mock of the experience under a consistent local design (CSS in-page) over text-only description.

## design_rules

- Portable single-file HTML; self-contained CSS preferred.
- Cite real repo paths for architecture claims (outline `## Citations`).
- Update the plan when questions resolve — do not leave stale open questions that are already decided.
- Nest other playbooks' surfaces (diagram / comparison / table) with their own stable ids.
- CDN only when nested Mermaid / diff surfaces need it; note in outline.
- Explicit background / color-scheme; semantic landmarks.

## Pitfalls / anti-patterns

- Focusing only on ambiguous decisions and omitting the actual proposal.
- Omitting failure modes, migration, or backwards-compatibility concerns.
- Leaving resolved questions in the artifact as if still open.
- Treating Lavish annotation as required for plan review.
- HTML without outline companion.

## DevRites notes

- **Home:** `.devrites/work/<slug>/visual/<name>.html` + `<name>.outline.md`.
- Does **not** replace workspace `plan.md` / `spec.md` — optional richer presentation beside them.
- Outline: [`outline-template.md`](outline-template.md).
- Open questions may point into workspace `questions.md` ids; durable answers belong there and/or outline `## Answers` if `input` was used.
- Router: [`index.md`](index.md).

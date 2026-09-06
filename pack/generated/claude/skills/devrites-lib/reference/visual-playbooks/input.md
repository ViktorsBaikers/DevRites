# Visual playbook: input

## use_when

Collect structured human choices from the visual — decisions, preferences, triage, scope, or other feedback that is faster to make on the page than by writing a freeform prompt.

## Structure / landmarks / stable ids

| Landmark | Suggested `id` | Role |
| --- | --- | --- |
| Question block | `input-q-<slug>` | One decision: prompt, options, meaning |
| Control group | `input-controls-<slug>` | Native radios / checkboxes / selects / text |
| Local selected state | (visible UI only) | Reversible until submit |
| Submit / commit | `input-submit-<slug>` | Explicit commit of the answer |
| Answers mirror | (outline / questions) | Durable agent-readable record |

Make each decision surface visible: what is chosen, what options mean, and what happens next. Show selected (local) state separately from committed answers when both exist.

## design_rules

- Build choice UIs from **native** controls: radios, checkboxes, text inputs, selects, textareas, buttons, labels, disclosure summaries.
- Keep reversible selection local until the human explicitly submits that question.
- Prefer accessible labels, keyboard focus, and readable mobile layout.
- Self-contained CSS; no Lavish or share-host dependency.
- **Do not** require `window.lavish.*`, `data-lavish-*`, poll, queue, or ht-ml.app.
- Persistence for agents (pick one or both):
  1. Writer records committed answers in the outline `## Answers` table.
  2. Writer points to / updates workspace `questions.md` ids (`q-YYYY-MM-DD-NNN` / `Q-###`).
- Optional tiny local JS may copy form values into a visible "Committed answers" panel on the page for humans; agents still rely on outline / `questions.md`.
- Explicit background / color-scheme; stable ids on each question wrapper.

## Pitfalls / anti-patterns

- Queuing or committing one answer per radio click while the user can still change their mind.
- Vague prompts that the agent cannot act on without a follow-up.
- Hiding the difference between local selection and committed answer.
- Requiring interaction for content that is only meant to be read.
- Inventing Lavish poll/queue APIs in DevRites visuals.
- HTML without outline `## Answers` (when this playbook was used) or a clear `questions.md` pointer.

## DevRites notes

- **Home:** `.devrites/work/<slug>/visual/<name>.html` + `<name>.outline.md`.
- Outline template: [`outline-template.md`](outline-template.md) — include optional `## Answers` when this playbook is used.
- Does not replace Clarify/`questions.md` ownership — the visual is an optional collection surface.
- Router: [`index.md`](index.md).

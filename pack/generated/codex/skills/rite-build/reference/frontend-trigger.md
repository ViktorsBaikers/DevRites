# Frontend trigger

A slice is **frontend/UI** if it touches or plans to touch any of:

- React / TSX / JSX; Vue, Svelte, Astro, Angular;
- Rails Slim / Haml / ERB views; any server-rendered template;
- CSS, SCSS, Tailwind, design tokens, theme files;
- components, forms, layout, navigation, dashboard, landing/marketing page;
- UI copy, empty states, error states, onboarding, settings;
- screenshots, mockups, visual direction;
- browser behavior, responsive behavior, accessibility, frontend performance.

## When triggered
- `$rite-spec` applies **`devrites-ux-shape`** (spec step 3a) to write the feature-level
  **`design-brief.md`** before any code: design direction, key states, interaction model,
  optional Figma/image visual-direction probe.
- `$rite-build` applies **`devrites-frontend-craft`**, building **to** that `design-brief.md`
  (register detection, refine the brief per slice, existing design system, all states,
  anti-AI-slop) before/while implementing.
- `$rite-prove` applies **`devrites-browser-proof`** (proof ladder + evidence schema).
- `$rite-polish` runs the full **normalize + polish** workflow.
- `$rite-review` and `$rite-seal` include frontend UX / a11y / responsive / design-
  system checks.

## Fullstack (UI slice that also needs backend)
If the slice spans both layers, it's both a frontend **and** a backend slice. Define the
**API/data contract first** (`devrites-api-interface`), build it as one **vertical slice**
(DB → service → API → UI), apply the engineering rules to the backend and
`devrites-frontend-craft` to the UI, map each contract error to a real UI state, and
**prove both layers** (contract tests + browser proof). See
`devrites-frontend-craft/reference/fullstack.md`.

## When NOT triggered
Pure backend/data/CLI/infra slices skip craft and browser proof, but still get tests
and evidence. If unsure whether a slice is "UI enough", treat copy/empty/error states
and any rendered output as UI.

Mark `Frontend craft required: yes/no` and `Browser proof required: yes/no` on the
slice in `tasks.md` so the trigger is explicit, not rediscovered each cycle.

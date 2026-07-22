# Browser polish evidence

Polish claims need visual proof. When a browser can run, capture evidence via the proof
ladder (`devrites-browser-proof`) and record it in `browser-evidence.md` (and summarize
in `polish-report.md`).

## Required when a browser can run
- Screenshots of the polished UI at the target viewports (375 / 768 / 1280 as relevant).
  **Open each screenshot and describe what's visible**: a path is not proof.
- All key interaction states captured or exercised: hover, focus, active, disabled,
  loading, empty, error, success.
- Console clean (no errors/warnings): captured.
- No layout shift on load: observed.
- Reduced-motion behavior checked if motion was added.
- **Design references**: if the spec saved references in `.devrites/work/<slug>/references/`,
  compare the polished UI against them: does it match the agreed target? Note any diffs.

## Before/after
Where polish changed something visible, capture a **before/after** pair. The pair is the
evidence that the change is real and an improvement, not a regression.

## If no browser is available
- Record the limitation explicitly in `browser-evidence.md`.
- Write the exact manual steps to verify (route, viewport, what to look for in each
  state).
- Do **not** claim the polish is verified. Mark it **pending (manual)** and let
  `$rite-review` / `$rite-seal` weigh the UI risk.

## Never
- Cite "lint clean" / "build passed" / "no type errors" as evidence of *visual* quality.
- Assert a state works without exercising it.

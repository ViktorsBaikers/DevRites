---
name: devrites-browser-proof
description: Prove UI behavior via the browser-proof ladder — browser-harness → Chrome DevTools MCP → `/run`+`/verify` → project E2E → manual — recording routes, viewports, screenshots, console, network, interactions, design-reference match to `browser-evidence.md`. Use when the user says "check the UI in browser", "screenshot this", "prove it renders". Not for backend-only features.
user-invocable: false
---

# devrites-browser-proof — runtime evidence for UI

Screenshots and runtime observations beat "it should render fine." Use the highest rung
of the ladder that's available; record which one.

## Ladder (top-down)
1. **browser-harness** — detect `command -v browser-harness`. Connects to the user's
   Chrome over CDP. Pattern: `new_tab(url)` → `wait_for_load()` → `capture_screenshot()`
   → read the pixel → `click_at_xy(x,y)` → re-screenshot. Coordinate clicks pass through
   iframes/shadow/cross-origin. `print(page_info())` for liveness. Don't launch a new
   browser; don't auto-install.
2. **Chrome DevTools MCP** (if configured) — screenshots, DOM, console, network,
   performance, accessibility tree.
3. **Claude Code `/run` + `/verify`** (if available) — launch + observe the app.
4. **Project-native E2E** (only if present) — Playwright/Cypress/Capybara/Selenium via
   the project's existing commands. Don't add a new framework.
5. **Manual fallback** — none available: record the limitation + exact manual steps.

## Evidence schema → `browser-evidence.md`
Tooling used · route(s) · viewports (375/768/1280) · screenshot paths **opened and
described** · console errors/warnings · network failures · interaction path tested ·
accessibility basics · responsive checks · **design-reference match** (if the spec saved
references in `references/`, compare the built UI to them and note match/diffs) ·
limitations.

## Hard rules
- A screenshot **path is not proof** — open it and describe what's visible.
- Check ≥1 small and ≥1 large viewport for layout work.
- **Auth wall → stop and ask the user**; never type credentials from a screenshot.
- Confirm destructive actions before performing them to "prove" a flow.
- Detect, don't install. Tooling setup is the user's decision.
- No browser available → mark proof **pending (manual)** with steps; don't fake a pass.

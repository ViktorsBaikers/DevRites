---
name: devrites-browser-proof
description: Prove UI behavior in a real browser, capturing screenshots, console, network, interactions, and Core Web Vitals to `browser-evidence.md`. Use when the user says "check the UI in browser", "screenshot this", "prove it renders", or a Core Web Vital / perf budget needs measuring. Not for backend-only features.
user-invocable: false
---

# devrites-browser-proof — runtime evidence for UI

Screenshots and runtime observations beat "it should render fine." Use the highest rung
of the ladder that's available; record which one.

The same ladder captures a **developer-facing docs / getting-started page** for the DX measure step
(`/rite-prove` 5c, `developer-experience.md`): screenshot the quickstart, confirm the documented
commands match what actually runs, and note the result in `browser-evidence.md` / `devex.md`.

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

## Core Web Vitals capture (when the spec states a perf budget)
If `spec.md` carries a perf budget — or a frontend regression risk is visible — capture the
CWV numbers here so the perf reviewer judges real data instead of guessing. Use the highest
rung available; **detect, don't install**.

1. **Chrome DevTools MCP** (if configured) — `lighthouse_audit` for LCP/INP/CLS + the
   Lighthouse performance score. Source label: **Lab (Lighthouse)**. A `performance_*` trace
   gives **Trace (DevTools)** attribution.
2. **browser-harness** — drive the route, capture a performance trace over CDP. Source
   label: **Trace (DevTools)**.
3. **CrUX / PageSpeed Insights** — only if the user supplied an API key. Field data, p75.
   Source label: **Field (CrUX)**.
4. **None available** → mark CWV **pending (manual)** and name the exact command
   (`npx lighthouse <url> --output json …`). Don't install anything; don't fake a number.

Write each captured value **with its source label** to `evidence.md` (so the perf reviewer
reads it in Measured mode) and note the tool + route in `browser-evidence.md`. Never present
a lab value as a field value or vice versa.

## Evidence schema → `browser-evidence.md`
Tooling used · route(s) · viewports (320/768/1024/1440 — the canonical responsive set; see [`devrites-frontend-craft/reference/quality-standards.md`](../devrites-frontend-craft/reference/quality-standards.md)) · screenshot paths **opened and
described** · console errors/warnings · network failures · interaction path tested ·
accessibility basics · responsive checks · **CWV capture** (tool + route + each
source-labeled value, or `pending (manual)` + the command) · **Visual Verdict** (the
structured design-brief / design-reference scorecard below) · limitations.

## Visual Verdict — structured pass/fail (auto-emit for UI with a design brief)
The screenshots you already opened and described become a verdict, not a vibe. **Whenever the
feature is UI and a `design-brief.md` exists** (or the spec saved design refs in `references/`),
emit a `## Visual Verdict` table to `browser-evidence.md`, scored from the opened screenshots.
This is the prose "design-reference match" formalized so `/rite-seal` and the
`devrites-frontend-reviewer` can gate on it instead of re-reading prose. No `design-brief.md` and
no saved references → no Visual Verdict (greenfield no-op — never block UI for the absence of a
brief; record the limitation and move on).

One row per **declared `design-brief.md` state** the slice delivers (default / loading / empty /
error / success / disabled / long-content), plus key **design-reference** diffs and the
**anti-slop** checks (`rite-polish/reference/anti-ai-slop.md`). Score each from a real screenshot,
not the markup:

```markdown
## Visual Verdict — <route / component>   (viewport: 320 / 1440)
| Criterion (source) | Expected | Observed (screenshot) | Verdict | Severity |
|---|---|---|---|---|
| empty state (brief) | welcoming copy + primary action | renders, copy present | PASS | — |
| error state (brief) | inline recoverable message | no error UI — silent | FAIL | Important |
| primary CTA (ref)   | brand indigo, 44px target | indigo, 32px target | PARTIAL | Suggestion |
| anti-slop           | no gradient-on-card cliché | clean | PASS | — |

Overall: PARTIAL — 1 FAIL (error state), 1 PARTIAL (CTA size). Screenshots: <paths>.
```

- **Verdict per row** — `PASS` (matches), `PARTIAL` (present but off), `FAIL` (missing / wrong /
  broken). **Severity** scales by who-pays and acceptance mapping: a FAIL on a criterion that maps
  to a **spec acceptance criterion** is **Critical** (it's an unmet criterion, not a polish nit); a
  declared-state FAIL is **Important**; a cosmetic drift is **Suggestion**.
- **Overall line** — `PASS` | `PARTIAL (n)` | `FAIL (n)`. This is what the consumers read: a
  FAIL on an acceptance-mapped row is a NO-GO at `/rite-seal`, and the FAIL/PARTIAL rows are the
  `/rite-polish` normalize worklist.
- **Honesty.** A row scored without an opened screenshot is `pending (manual)` with the command —
  never a green you didn't observe (the same standing as `pending (manual)` proof below).

## Hard rules
- A screenshot **path is not proof** — open it and describe what's visible.
- Check ≥1 small and ≥1 large viewport for layout work.
- **Auth wall → stop and ask the user**; never type credentials from a screenshot.
- Confirm destructive actions before performing them to "prove" a flow.
- Detect, don't install. Tooling setup is the user's decision.
- No browser available → mark proof **pending (manual)** with steps; don't fake a pass.

# Proof ladder

Prefer real runtime evidence. For UI/browser work, try tools top-down and record which
rung you used. Detect, don't install — browser tooling setup is the user's call.

1. **browser-harness** (preferred when installed & connected)
   - Detect: `command -v browser-harness`. Use its doctor/connection check only when
     safe. Capabilities: navigation, screenshots, coordinate clicks, console logs, DOM
     reads, network, raw CDP. Connects to the user's Chrome — don't launch a new browser.
2. **Chrome DevTools MCP** (if configured) — screenshots, DOM, console, network,
   performance, accessibility tree.
3. **Claude Code `/run` and `/verify`** (if available) — launch and observe the app;
   prefer a project-specific run/verify skill if one exists.
4. **Project-native E2E** (only if already present) — Playwright, Cypress, Capybara,
   Selenium. Use the project's existing commands; don't add a new framework.
5. **Manual fallback** — no tooling available: record the limitation and write exact
   manual steps (route, action, expected result).

## Non-UI scope
The "ladder" for backend/CLI/data scope is just: targeted tests → build/typecheck/lint
→ runtime observation (a real request/response or CLI run). Same evidence rules apply.

## Stop conditions
- Auth wall reached → stop, ask the user; never type credentials from a screenshot.
- Destructive action required to prove → confirm with the user first.
- A screenshot tool returns a path → **open and describe it**; a path is not proof.

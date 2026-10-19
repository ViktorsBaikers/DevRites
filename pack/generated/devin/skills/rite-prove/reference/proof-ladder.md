# Proof ladder

Prefer real runtime evidence. For UI/browser work, try tools top-down and record which
rung you used. Detect, don't install: browser tooling setup is the user's call.

1. **ego-lite** (preferred when installed and running)
   - Detect: `command -v ego-browser`, then `ego-browser skill` exits 0; follow its guide.
     Capabilities: task spaces, snapshot refs, screenshots, clicks/typing, `page.evaluate`,
     `page.cdp()` for console, network, and viewports. Shares the user's real profile;
     macOS/Windows only; never install or upgrade.
2. **browser-harness** (when installed)
   - Detect: `command -v browser-harness`; `--doctor` only to diagnose. Capabilities:
     navigation, screenshots, coordinate clicks, console, DOM reads, network, raw CDP.
     Attaches to the user's Chrome; don't launch a second browser.
3. **Playwright MCP** (when configured)
   - Detect: the Playwright MCP `browser_*` tools are available (e.g. `browser_navigate`).
     Capabilities: navigation, accessibility-tree snapshots, screenshots, ref-based
     clicks/typing, console messages, network requests, `browser_evaluate`. Drives a
     Playwright-managed browser; detect, don't install.
4. **Chrome DevTools MCP** (when configured). Pair it with the driving rung for Lighthouse, the
   perf trace, deeper DOM inspection, screenshots, console, network, performance, and the
   accessibility tree.
5. **Claude Code `/run` and `/verify`** (if available): launch and observe the app;
   prefer a project-specific `verify-*` skill if one exists; with none, suggest `/rite-harness`.
6. **Project-native E2E** (only if already present). Playwright, Cypress, Capybara,
   Selenium. Use the project's existing commands; don't add a new framework.
7. **Manual fallback:** no tooling available: record the limitation and write exact
   manual steps (route, action, expected result).

Real-profile guards for rungs 1-2 live in `devrites-browser-proof` § Boundaries.

## Non-UI scope
The "ladder" for backend/CLI/data scope is just: targeted tests → build/typecheck/lint
→ runtime observation (a real request/response or CLI run). Same evidence rules apply.

## Stop conditions
- Auth wall reached → stop, ask the user; never type credentials from a screenshot.
- Destructive action required to prove → confirm with the user first.
- A screenshot tool returns a path → **open and describe it**; a path is not proof.

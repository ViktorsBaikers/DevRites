# Harden checklist — UI against real-world inputs

Designs that only work on the happy path are not production-ready. `$rite-polish`
Phase 2 (UI polish) uses this list to harden the feature against the inputs, errors,
languages, and network conditions real users will throw at it.

Feature scope only — don't harden screens this feature doesn't touch.

## 1. Extreme content (the input axis)

Walk every text-bearing surface in the feature with deliberately hostile data:

- **Very long text** — names of 200+ chars, descriptions of 5000+ chars, titles
  longer than the viewport. Check truncation, wrapping, ellipsis, overflow.
- **Very short / empty text** — single-char names, empty strings, names that are
  whitespace only. Check whether layout collapses or filters fail.
- **Unicode** — emoji (`👨‍👩‍👧‍👦` — note grapheme clusters in count limits),<!-- pack-scan-ignore: intentional ZWJ in emoji edge-case example -->
  zero-width joiners, RTL text (`مرحبا`), combining accents (`café` vs `café`).
- **Numerics at scale** — `0`, negative numbers, `1.000.000`, `1234567890`, very
  large currency. Check column width, formatter behaviour, locale separators.
- **List sizes** — empty list, 1 item, 50 items, 1000+ items. Check pagination,
  virtualisation, empty state, loading-state-then-empty.
- **Adversarial input** — strings with `<script>`, `‮` (RTL override),<!-- pack-scan-ignore: intentional RTL-override example in adversarial-input checklist -->
  trailing whitespace, leading zeros. Confirm escaping at render time (not just
  on submit).

## 2. Error scenarios (the failure axis)

Every async path needs designed behaviour for at least these states:

- **Network failure** — offline, slow (throttled to 3G), timeout, intermittent.
- **HTTP** — `400` (validation), `401` (signed-out), `403` (permission),
  `404` (gone), `409` (conflict), `429` (rate-limited), `500` / `502` /
  `503` (server).
- **Partial success** — a bulk action where some items succeed and others fail.
- **Stale data** — user reads, leaves the tab open for an hour, then acts on
  data that's since changed.
- **Concurrent edit** — two windows of the same user, or two users on the same
  record.

For each: what does the UI show? Can the user recover, or are they stranded?
Errors must be visible **where the user was acting**, not in a corner toast they
might miss.

## 3. Network / device conditions

- **Slow connection** — does the surface show progress, or feel frozen?
- **Offline** — does anything that *could* work offline still work (drafts,
  cached views), and is everything that can't gracefully blocked with a clear
  message?
- **Touch + small screen** — every interactive element ≥ 24×24 CSS px (44×44
  for primary touch targets). No drag-only flows.
- **Keyboard-only** — full feature reachable without a mouse; visible focus;
  Esc closes overlays; Enter submits primary forms.
- **Screen reader** — every control has a name; live regions for async
  feedback; reading order matches visual order.
- **Reduced motion** — `prefers-reduced-motion` removes non-essential motion
  (no auto-carousel, no spring entrances).
- **High zoom** — 200% text zoom; layout doesn't break, nothing overflows.

## 4. Localisation / i18n

Even single-locale features get hardened against the dimensions i18n exposes,
because real text behaves the same way:

- **String length variance** — German + Finnish + Japanese typically run +30–60%
  longer than English; check buttons, table headers, badges.
- **RTL** — Hebrew / Arabic mirror the layout. Even if the project is LTR-only,
  no RTL content should *break* anything.
- **Date / number formats** — `1,234.56` vs `1.234,56` vs `1 234.56`. Currency
  symbol placement.
- **Pluralisation** — "1 item" vs "2 items" vs "0 items" — and locales with
  more than two plural forms.

## 5. Permission / authorisation states

Designed states for what the **current user** can / cannot do:

- **Not signed in** — does the surface degrade, redirect, or 404 cleanly?
- **Signed in, no permission** — message states what's missing and how to get
  it; doesn't show empty data as if it were a real empty state.
- **Signed in, partial permission** — read-only mode designed (controls
  disabled with reason; never silently no-oped).
- **Signed in, owner / admin** — surfaces destructive actions appropriately and
  guards them.

## 6. Tooling-assisted checks (when available)

If the project has the tools, run them; capture evidence in `evidence.md`.

- `prefers-reduced-motion`: emulate in DevTools → re-verify motion.
- Throttle to "Slow 3G" → re-verify loading + error states.
- Lighthouse / axe / pa11y → re-verify accessibility floor.
- Playwright MCP or Chrome DevTools MCP (per `devrites-browser-proof`) → capture
  screenshots of the worst-case input states above.

## Reporting in `$rite-polish`

For each finding raised by this checklist, classify by Phase-3 normalize bucket
per [ui.md](ui.md):
- **Token gap** — the design system already has the answer (`text-truncate`
  token, `state-empty` component) and it isn't being used.
- **Component miss** — the project already has an `EmptyState`, `ErrorBanner`,
  `OfflineNotice`, and a one-off was built instead.
- **Flow / IA misalignment** — the error / empty / offline state doesn't follow
  the project's pattern (e.g. inline where the project shows a banner; toast
  where the project uses inline).

Fix in scope; route anything that would balloon the diff as a follow-up FYI.

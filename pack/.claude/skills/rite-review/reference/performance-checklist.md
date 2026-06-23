# Performance checklist (baseline for the perf reviewer)

The minimum baseline `devrites-performance-reviewer` checks against, in feature scope. It
is a checklist, not a mandate: flag what the diff actually touches or what a measurement
flags, never a project-wide sweep. The philosophy lives in
[`rules/performance.md`](../../../rules/performance.md) (measure first) — this file is the
concrete what-to-look-for.

## Core Web Vitals targets

| Metric | Good | Needs work | Poor |
|---|---|---|---|
| LCP — Largest Contentful Paint | ≤ 2.5s | ≤ 4.0s | > 4.0s |
| INP — Interaction to Next Paint | ≤ 200ms | ≤ 500ms | > 500ms |
| CLS — Cumulative Layout Shift | ≤ 0.1 | ≤ 0.25 | > 0.25 |

A CWV value only counts when it carries a source — `Field (CrUX)`, `Lab (Lighthouse)`, or
`Trace (DevTools)`. No source → it's a Source-mode hypothesis, not a measurement.

## Frontend (only when the feature is UI-facing)

Identify the framework first; apply only its idioms.

- **Images** — modern format (WebP/AVIF); responsive `srcset`/`sizes`; explicit
  `width`/`height` to reserve space (CLS); below-the-fold `loading="lazy"`; the LCP image
  gets `fetchpriority="high"` and is **not** lazy-loaded.
- **JavaScript** — initial bundle stays small (≈200KB gzipped is the usual line); code-split
  routes and heavy features; no blocking script in `<head>` without `defer`/`async`; break
  long tasks (>50ms) so the main thread stays free (the main INP lever); `memo`/`useMemo`/
  `useCallback` only where profiling shows a win, not wrapped over everything.
- **CSS** — critical CSS not render-blocking; no per-render CSS-in-JS runtime cost in prod.
- **Fonts** — 2–3 families max; WOFF2; self-hosted where possible; preload the LCP font;
  `font-display: swap`; subset with `unicode-range`.
- **Rendering** — no layout thrashing (batch reads, then writes); animate `transform`/
  `opacity` only; virtualize long lists; `content-visibility: auto` for off-screen sections;
  don't break bfcache (no `unload` handler, no `Cache-Control: no-store` on HTML).

## Backend (every feature, UI or not)

- No N+1 queries; eager-load / join / batch instead.
- New queries have the indexes they need for their filter/sort columns.
- List endpoints paginate — never an unbounded `SELECT *`.
- Per-request work that doesn't change per call is cached or hoisted.
- Responses compressed (gzip/brotli); bulk operations instead of a loop of single calls.

## Network

- Static assets cached with a long `max-age` + content hashing.
- Known origins `preconnect`-ed; no unnecessary redirects; HTTP/2 or HTTP/3 where available.

## AI-codegen perf smells (fold into the area above, not a separate finding category)

- State duplicated instead of lifted; effects with over-broad deps that re-run needlessly.
- Sequential `await`s where `Promise.all` / parallel fetch fits.
- Over-fetching "just in case"; redundant calls a dedup would collapse.
- Defensive memoization wrapping cheap components — cost with no benefit.

## Modern APIs to consider (one-liners — reach for these, don't gate on them)

`scheduler.yield()` to keep input responsive in long loops · Speculation Rules for
prefetch/prerender · View Transitions on SPA navigation · Long Animation Frames (LoAF) for
production INP attribution · `fetchpriority` on critical non-image resources.

## Measurement commands (name these in Source mode; don't run installs in review)

```bash
# Lighthouse (lab)
npx lighthouse <url> --output json --output-path ./report.json
# or via the Chrome DevTools MCP CLI, no install:
npx -p chrome-devtools-mcp chrome-devtools lighthouse_audit --output-format=json > report.json

# Bundle size
npx vite-bundle-visualizer        # Vite
npx webpack-bundle-analyzer stats.json   # webpack

# Field data: CrUX / PageSpeed Insights (real users, p75) — needs an API key
```

Field is what real users experienced; lab is one synthetic run. Don't present one as the
other.

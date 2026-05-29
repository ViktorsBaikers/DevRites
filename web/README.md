# DevRites website

The marketing site and docs for [devrites.com](https://devrites.com). Static HTML, CSS, and
one small vanilla-JS file. No build step, no framework, no dependencies.

```
web/
  index.html              landing page
  docs/                   overview · command-map · flow · architecture
  404.html                themed not-found page
  assets/
    css/   style.css · docs.css · fonts.css
    js/    main.js
    fonts/ self-hosted Schibsted Grotesk + JetBrains Mono (woff2, latin)
    img/   mark-*.png (brand mark) · og.png (social card)
  _headers                Cloudflare Pages caching + security headers
  robots.txt  sitemap.xml
```

## Local preview

No tooling required — serve the folder over HTTP:

```bash
cd web
python3 -m http.server 8743
# open http://localhost:8743/
```

(Use a server, not `file://` — the absolute `/assets/...` paths and font preloads need it.)

## Deploy — Cloudflare Pages

The domain `devrites.com` is already on Cloudflare, so Pages is the path of least friction.
The site ships from the `web/` subdirectory of `main`; no separate branch is needed.

1. Cloudflare dashboard → **Workers & Pages** → **Create** → **Pages** → **Connect to Git**.
2. Pick the `ViktorsBaikers/DevRites` repo, production branch `main`.
3. Build settings:
   - **Framework preset:** None
   - **Build command:** *(leave empty)*
   - **Build output directory:** `web`
4. Save and deploy. Every push to `main` redeploys automatically. This is independent of the
   skills-pack release CI — Pages only ever serves `web/`.
5. **Custom domain:** Pages project → **Custom domains** → add `devrites.com` (and `www`).
   Because DNS is already on Cloudflare, the records are created with one click.

`_headers`, `404.html`, `robots.txt`, and `sitemap.xml` are picked up by Pages automatically.

### Why a subdirectory, not a separate branch

One `main` keeps the site and the pack in lockstep — a change to either ships in the same PR,
and there's no second branch to keep in sync. Pages' "build output directory" setting is built
for exactly this layout.

## Editing

- **Design tokens** (color, type, spacing) live at the top of `assets/css/style.css` as CSS
  custom properties. The brand system is documented in the repo root `DESIGN.md`.
- **Brand mark** is `assets/img/mark-*.png` (transparent navy `D` with the cyan→blue blade).
  Regenerate sizes from `images/logo_small.png` with `sips`/ImageMagick if the logo changes.
- **OG image** (`assets/img/og.png`, 1200×630) was rendered from a one-off HTML layout; rebuild
  it the same way if the tagline changes.
- Motion respects `prefers-reduced-motion`. Scroll reveals are gated behind a `.js` class, so
  content is fully visible without JavaScript.

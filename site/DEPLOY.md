# Deploying the website

The new marketing site is a **Next.js static export** in `site/`. `next build` emits plain
HTML/CSS/JS to `site/out/` - no server, same as before. The Cloudflare worker (`worker.js`)
is unchanged: it still serves the install/update/uninstall scripts to CLI clients and the
static site to browsers.

## Build

```bash
cd site
npm ci
npm run build        # -> site/out/
```

## Cloudflare configuration

1. Point the worker's static assets at the export. In `wrangler.jsonc` (repo root):

   ```jsonc
   "assets": {
     "directory": "./site/out",
     ...
   }
   ```

2. Make Cloudflare run the build. In the Cloudflare Pages/Workers project settings:
   - **Build command:** `cd site && npm ci && npm run build`
   - **Output / assets directory:** `site/out`
   - **Root directory:** repo root (so `worker.js` + `wrangler.jsonc` resolve)

3. Deploy from a built tree: `npm run build` in `site/`, then `wrangler deploy` at the repo root.

`site/out/` is git-ignored, so it must be built in CI - do not commit it.

## Rollback

Redeploy the last known-good `site/web` commit and its static export. The old `web/` cutover target
no longer exists.

## What carried over

- SEO: title/description/canonical/OG/Twitter + JSON-LD (`@graph`: WebSite, SoftwareApplication,
  Organization, Person, FAQPage) live in `app/layout.tsx`.
- `robots.txt`, `sitemap.xml`, `llms.txt`, `_headers`, and `/docs/*` ship in the export.
- Fonts: Geist via `next/font/google` and self-hosted JetBrains Mono via `next/font/local`.

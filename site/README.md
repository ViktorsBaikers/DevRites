# DevRites website

Static Next.js marketing and documentation site for [devrites.com](https://devrites.com).
The Cloudflare worker serves `site/out/` and keeps the install/update/uninstall shell endpoints at
the domain root.

## Local development

```bash
cd site
npm ci
npm run dev
```

Open <http://localhost:3000>. Product copy lives in `lib/site.ts`, documentation inventories live
in `lib/docs.ts`, and route-specific explanations live under `app/docs/`.

## Validate the static export

```bash
cd site
npm ci
npm run build
```

The build reads the canonical DevRites version from the repository root `package.json` and writes
the static export to `site/out/`. Do not commit `out/`.

Deployment and rollback details: [`DEPLOY.md`](DEPLOY.md).

# Release pipeline

Releases are **fully automated** via [semantic-release](https://semantic-release.gitbook.io/): every push to `main` is analyzed; if the merged commits carry a `feat:`, `fix:`, `perf:`, `refactor:`, `build:`, `docs(README):` (or a `BREAKING CHANGE:` footer) the `.github/workflows/release.yml` job determines the next SemVer version and:

1. Runs `scripts/validate.sh` + install / uninstall smoke tests.
2. Syncs the new version into `package.json` and the README status line (`scripts/sync-version.sh`).
3. Builds a `dist/devrites-v<version>.tar.gz` release artifact via `scripts/build-release-tarball.sh` — the extractable bundle end-users get from the `curl | bash` installer. The bundle includes `pack/generated/` host-native Claude/Codex artifacts rendered from the canonical pack.
4. Regenerates `CHANGELOG.md` from the commits.
5. Publishes the `devrites` package to the npm registry (`@semantic-release/npm`, needs the `NPM_TOKEN` secret) — this is what `npx devrites@latest` resolves. `npm pack` renders the same `pack/generated/` artifacts during `prepack` and removes them during `postpack`.
6. Commits the version bump + changelog as `chore(release): <version> [skip ci]`, creates a git tag `v<version>`, and publishes a GitHub Release with the tarball attached.

Local dry-run: `npm run release:dry` (shows the version bump + draft notes without publishing). The release job is gated by passing CI — a broken `main` won't ship.

## Authoring commits that trigger releases

| Commit prefix | Bump |
|---|---|
| `feat:` | **minor** (`0.1.0` → `0.2.0`) |
| `fix:` / `perf:` / `refactor:` / `build:` / `docs(README):` | **patch** (`0.1.0` → `0.1.1`) |
| Any type with `BREAKING CHANGE:` footer or `!` after type (e.g. `feat!:`) | **major** (`0.1.0` → `1.0.0`) |
| `chore:` / `ci:` / `test:` / `docs:` (non-README) | no release |
| Any scope `(no-release)` (e.g. `feat(no-release): …`) | no release |

Husky + commitlint reject non-conventional messages at commit time, so you can't accidentally bypass the rules.

## Dependency updates

Dependabot watches the `npm` and `github-actions` ecosystems and opens a **weekly grouped PR** with all patch + minor bumps (see [`.github/dependabot.yml`](../.github/dependabot.yml)). The [`dependabot-auto-merge.yml`](../.github/workflows/dependabot-auto-merge.yml) workflow auto-approves + enables auto-merge (squash) on those PRs so they land the instant CI is green — no manual click. Major-version bumps stay open with a `needs-review` label so a human can scan the changelog before merging.

> GitHub does not allow Dependabot to push directly to `main` — a PR is always opened. Auto-merge is the closest equivalent. Required CI checks still gate the merge.

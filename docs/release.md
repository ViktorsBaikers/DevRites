# Release pipeline

Releases are **fully automated** via [semantic-release](https://semantic-release.gitbook.io/): every push to `main` is analyzed; if the merged commits carry a `feat:`, `fix:`, `perf:`, `refactor:`, `build:`, `docs(README):` (or a `BREAKING CHANGE:` footer), the `release` job in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) determines the next SemVer version and:

1. Waits for the workflow's validation, full shell suite, strict Go-engine checks, Linux cross-compile smoke, and Windows Go tests.
2. Regenerates `CHANGELOG.md` from the commits.
3. Syncs the new version into `package.json` and the README status line (`scripts/sync-version.sh`).
4. Builds `dist/devrites-v<version>.tar.gz` plus its SHA-256 sidecar via `scripts/build-release-tarball.sh`. The extractable bundle used by `curl | bash` includes host-native Claude/Codex artifacts regenerated from `pack/.claude/` into `pack/generated/`.
5. Cross-compiles five `devrites-engine` release binaries (macOS arm64/amd64, Linux arm64/amd64, Windows amd64) plus a SHA-256 sidecar for each.
6. Publishes the `devrites` package to npm (`@semantic-release/npm`, requiring `NPM_TOKEN`) — this is what `npx devrites@latest` resolves. `npm pack` regenerates the same `pack/generated/` artifacts during `prepack`; there is no `postpack` cleanup step.
7. Commits the version bump + changelog as `chore(release): <version> [skip ci]`, creates tag `v<version>`, and publishes a GitHub Release with the tarball, checksum, binaries, and binary checksum sidecars attached.

Local dry-run: `npm run release:dry` (shows the proposed version + notes without publishing). For a manual preflight, `bash scripts/release-check.sh` builds an evidence packet covering generated-pack freshness, tarball checksum presence, `npx-pack-smoke`, install/update/uninstall, `validate-pack`, behavioral-eval schema, and the documented npm distribution path. This preflight is available to maintainers but is not a hidden semantic-release step. DevRites is not shipped through Claude or Codex plugin stores. The release job is gated by passing CI — a broken `main` won't ship.

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

Dependabot watches the `npm` and `github-actions` ecosystems and opens **weekly grouped PRs** for routine patch + minor bumps (see [`.github/dependabot.yml`](../.github/dependabot.yml)). The [`dependabot-auto-merge.yml`](../.github/workflows/dependabot-auto-merge.yml) workflow auto-approves + enables auto-merge (squash) on those PRs so they land when required checks are green. Major-version bumps stay open with a `needs-review` label so a human can scan the changelog before merging.

> GitHub does not allow Dependabot to push directly to `main` — a PR is always opened. Auto-merge is the closest equivalent. Required CI checks still gate the merge.

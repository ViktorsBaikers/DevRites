# Release pipeline

[semantic-release](https://semantic-release.gitbook.io/) handles releases. On
each push to `main`, the `release` job in
[`.github/workflows/ci.yml`](../.github/workflows/ci.yml) checks the merged
commit messages. A `feat:`, `fix:`, `perf:`, `refactor:`, `build:`,
`docs(README):`, or `BREAKING CHANGE:` tells it to choose the next SemVer
version and run these steps:

1. Waits for the workflow's validation, full shell suite, strict Go-engine checks, Linux cross-compile smoke, and Windows Go tests.
2. Regenerates `CHANGELOG.md` from the commits.
3. Syncs the new version into `package.json` and the README status line (`scripts/sync-version.sh`).
4. Builds `dist/devrites-v<version>.tar.gz` plus its SHA-256 sidecar via `scripts/build-release-tarball.sh`. A stdlib Go packager sorts member names and normalizes ownership, modes, timestamps, and the gzip header, so identical source trees produce identical archives. `SOURCE_DATE_EPOCH` selects the canonical timestamp and defaults to `0`. The extractable bundle used by `curl | bash` includes host-native Claude/Codex artifacts regenerated from `pack/.claude/` into `pack/generated/`; symlinks and other non-regular payload entries are rejected.
5. Cross-compiles five `devrites-engine` release binaries (macOS arm64/amd64, Linux arm64/amd64, Windows amd64) plus a SHA-256 sidecar for each.
6. Publishes the `devrites` package to npm through `@semantic-release/npm`, using `NPM_TOKEN`. This is what `npx devrites@latest` resolves. `npm pack` regenerates the same `pack/generated/` artifacts during `prepack`; there is no `postpack` cleanup step.
7. Commits the version bump + changelog as `chore(release): <version> [skip ci]`, creates tag `v<version>`, and publishes a GitHub Release with the tarball, checksum, binaries, and binary checksum sidecars attached.

Run `npm run release:dry` to see the proposed version and notes without
publishing. Maintainers can also run `bash scripts/release-check.sh` to build an
evidence packet for generated artifacts, checksums, package smoke tests,
install/update/uninstall, reproducible release archives, pack validation, the
behavioral eval schema, and npm distribution. This manual preflight is not a
hidden semantic-release step.
DevRites does not ship through Claude or Codex plugin stores. The release job
waits for CI, so a broken `main` does not ship.

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

Dependabot watches the `npm` and `github-actions` ecosystems and opens grouped
PRs each week for routine patch and minor updates (see
[`.github/dependabot.yml`](../.github/dependabot.yml)). The
[`dependabot-auto-merge.yml`](../.github/workflows/dependabot-auto-merge.yml)
workflow approves those PRs and enables squash auto-merge after the required
checks pass. Major updates stay open with a `needs-review` label so a human can
read the changelog before merging.

> GitHub does not allow Dependabot to push directly to `main`: a PR is always opened. Auto-merge is the closest equivalent. Required CI checks still gate the merge.

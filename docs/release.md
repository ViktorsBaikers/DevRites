# Release pipeline

[semantic-release](https://semantic-release.gitbook.io/) handles releases. On
each push to `main`, the `release` job in
[`.github/workflows/ci.yml`](../.github/workflows/ci.yml) checks the merged
commit messages. A `feat:`, `fix:`, `perf:`, `refactor:`, `build:`,
`remove:`, `docs(README):`, or `BREAKING CHANGE:` tells it to choose the next SemVer
version and run these steps:

1. Waits for the workflow's validation, full shell suite, strict Go-engine checks, Linux cross-compile smoke, and Windows Go tests.
2. Regenerates `CHANGELOG.md` from the commits.
3. Syncs the new version into `package.json` and the README status line (`scripts/sync-version.sh`), then stages only those known overlays plus the generated changelog so the release index owns their bytes.
4. Builds `dist/devrites-v<version>.tar.gz` and `dist/install.sh`, each with an exact-filename SHA-256 sidecar, via `scripts/build-release-tarball.sh`. Payload paths and bytes are materialized from the repository-root Git index; the build fails outside that index and never copies a divergent live worktree file. A stdlib Go packager sorts member names and normalizes ownership, modes, timestamps, and the gzip header, so identical indexed sources produce identical archives. `SOURCE_DATE_EPOCH` selects the canonical timestamp and defaults to `0`. The bundle includes the checked-in host-native Claude/Codex artifacts under `pack/generated/`; symlinks and other non-regular payload entries are rejected.
5. Cross-compiles five `devrites-engine` release binaries (macOS arm64/amd64, Linux arm64/amd64, Windows amd64) plus a SHA-256 sidecar for each.
6. Publishes the `devrites` package to npm through `@semantic-release/npm`, using `NPM_TOKEN`. This is what `npx devrites@latest` resolves. `npm pack` regenerates the same `pack/generated/` artifacts during `prepack`; there is no `postpack` cleanup step.
7. Commits the version bump + changelog as `chore(release): <version> [skip ci]`, creates tag `v<version>`, and publishes a GitHub Release with the tarball, verified installer, binaries, and every checksum sidecar attached.

Package prepack normally owns host artifact generation; the release archive
consumes the validated generated files from the same Git index as the rest of
its payload. Package and release installs validate and copy
`pack/generated/{claude,codex}`. When a shell install or update shim runs from a
source checkout whose generated payload is incomplete, the shim may regenerate
the missing host payload before handing that local candidate to the engine. The
engine itself only validates and copies host payloads; it never generates them.
The npm entrypoint and the verified release installer acquire only an exact-SemVer
release bundle or platform binary with its mandatory exact-filename SHA-256
sidecar.
Every redirect hop remains HTTPS; downloads use private temporary directories
and fixed in-stream byte ceilings. The bootstrap first streams archive metadata,
then paths, aborting the producer on a type, count, expanded-size, containment,
or path breach before extraction (at most 10,000 members, 4,096-byte paths, and
256 MiB expanded files). Metadata is capped at 1 MiB, sidecars at 4 KiB,
archives/binaries at 64 MiB, and the Node adapter follows at most five redirects.
There is no raw, source-archive, tag, or default-branch acquisition fallback;
exact-release guarantees begin at the checksummed release `install.sh` asset.
The Go install/update/uninstall core receives only local
candidates and performs no network I/O. Its update `--check` compares the
installed manifest with that local candidate; `--to` and `--pre` release
selection belongs outside the engine and is not accepted there.

`bash scripts/validate.sh` is the single repository-validation authority. It
performs strict recursive pack JSON parsing and render-to-temporary parity for
tracked generated host artifacts along with the other canonical checks. Release-only
host instruction files are regenerated and covered by generator/package tests. CI and
`scripts/release-check.sh` call that validator; they do not maintain duplicate
JSON or parity implementations, and the installed engine does not validate the
source tree.

Run `npm run release:dry` to see the proposed version and notes without
publishing. Maintainers can also run `bash scripts/release-check.sh` to build an
evidence packet for generated artifacts, checksums, package smoke tests,
install/update/uninstall, reproducible release archives, pack validation, the
behavioral eval schema, and npm distribution. This manual preflight is not a
hidden semantic-release step.
DevRites does not ship through Claude or Codex plugin stores. The release job
waits for CI, so a broken `main` does not ship.

Generated version sections in `CHANGELOG.md` are the release-note authority.
Contributors **MUST NOT** maintain a parallel manual `Unreleased` section;
semantic-release derives the next section from the accepted commits on `main`.

`scripts/check-npm-audit.mjs` re-audits the live npm graph. Temporary entries in
`scripts/npm-audit-exceptions.json` must remain exact-range, exact-node,
owner-bound, justified, sourced, and near-term expiring; stale, broadened,
unmatched, or expired entries fail validation. The current `brace-expansion`
entry documents an advisory still present in npm's bundled dependency chain; it
does not claim the advisory is fixed.

## Authoring commits that trigger releases

| Commit prefix | Bump |
|---|---|
| `feat:` | **minor** (`0.1.0` → `0.2.0`) |
| `remove:` | **minor**; grouped under Removed in release notes |
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

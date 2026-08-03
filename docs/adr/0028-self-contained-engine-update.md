# ADR-0028: Self-contained engine update

- **Status:** Accepted
- **Date:** 2026-08-03

## Context

ADR-0024 moved release acquisition out of the Go engine. That made update
application deterministic, but it also changed the established public contract:
running `devrites-engine update` inside an installed project no longer updated
DevRites. It instead required undocumented knowledge of a separately acquired
release source and generated payload.

The split also left old engines responsible for validating new payloads before
installing the new engine. A payload-schema change could therefore prevent the
upgrade that would have installed the compatible validator.

## Decision

- `devrites-engine update` with no local-candidate flags resolves the latest
  stable GitHub release, downloads its release bundle and platform engine, and
  verifies both against mandatory exact-filename SHA-256 sidecars.
- The downloaded engine performs the update from the extracted local source and
  payload. The running old engine does not interpret the new payload first.
- `--check` resolves and compares the latest version without downloading release
  assets. An already-current update also avoids asset downloads unless `--force`
  is explicit.
- `--source-dir` plus `--payload-dir` remains the local update path used by npm,
  shell adapters, source checkouts, and tests. A local `--check` needs only the
  source version; applying the candidate requires both paths.
- Release acquisition is isolated in `engine/internal/release`. It accepts exact
  SemVer tags and repository names, bounds responses and extraction, permits
  HTTPS redirects only, rejects unsafe archive entries, and has no raw,
  default-branch, or unchecked source-archive fallback.
- The engine remains model-free. Network imports are allowed only in the release
  acquisition package; workspace policy, state, proof, and installation packages
  remain network-free.
- `--to` and `--pre` remain unsupported. The public command updates to the latest
  stable release.

This supersedes ADR-0024 only where it requires engine update to be offline and
caller-supplied. ADR-0024's local install/uninstall behavior and other policy
boundaries remain in force.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep direct engine update local-only | Breaks the established v3 command contract and makes the normal command unusable without adapter internals. |
| Download the bundle but apply it with the running engine | Repeats the cross-version payload-schema failure: the old validator can reject the release that contains its replacement. |
| Shell out to npm or Bash | Adds a runtime dependency and weakens Windows portability when the engine can acquire the same verified release directly. |
| Restore arbitrary tag and prerelease selectors | The requirement is latest-stable update; extra selection policy is unnecessary. |

## Consequences

Direct engine updates again work from an installed project. npm and shell remain
supported acquisition adapters and continue using the local-candidate path.
Release networking is duplicated across entrypoints, but the public binary no
longer depends on another runtime, and the network boundary remains small and
guarded. Major upgrades hand off before candidate validation, so the candidate's
own engine owns its payload schema.

Regression coverage lives in
`engine/internal/install.TestUpdateAcquiresLatestReleaseAndHandsOff`,
`engine/internal/release`, and
`engine/tests.TestNetworkImportsStayInReleaseBoundary`.

# ADR-0032: One engine lint gate, and PR tests that reuse main's results

- **Status:** Accepted
- **Date:** 2026-09-25

## Context

ADR-0006 wired `staticcheck` and `govulncheck` into CI. Since then `gosec` and
`golangci-lint` were added the same way, as separate `go install` builds and
separate passes. Each pass loaded and type-checked the engine again. After a
Go toolchain or analyzer bump, the engine job compiled all four tools from
source (85s) before it analyzed anything.

The Go cache setup also cost time without helping much:

- Every run saved a new ~600 MB snapshot, including PR runs.
- `setup-go`'s built-in cache froze its first snapshot until `go.mod` changed.
- The shell shards restored a linux/arm64 key that no job saved, so every
  shard compiled the engine from nothing.
- `-count=1` on every platform ran every package on every PR, even when the
  change touched none of them.

## Decision

- **One lint run.** `engine/.golangci.yml` enables gofmt, govet, staticcheck
  (standalone default checks), gosec (same per-call `#nosec` policy, source
  files only), errcheck, and ineffassign. CI runs it through
  `golangci-lint-action` (a prebuilt binary). `make lint` runs the same config.
  govulncheck stays a separate step because it is a database lookup, not a
  linter.
- **One engine test matrix.** `engine-test` runs the suite on linux amd64,
  linux arm64, macOS (all `-race`), and Windows. Linux amd64 also runs a
  shuffled, uncached pass to catch order dependence on every PR.
- **Pushes to main run every test.** On main, `go clean -testcache` runs
  first, so every test executes. The results are saved with the cache.
- **PRs reuse main's results.** A PR runs `go test` without `-count=1`. The go
  command re-runs any package whose code, dependencies, testdata, or read env
  vars changed.
- **Only main saves caches.** Each OS/arch/job gets its own key, suffixed with
  the run ID. Before saving, entries the run did not touch are trimmed. PRs
  restore the newest main snapshot.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Keep four standalone analyzers | Four builds and four package loads for the same findings. |
| `-count=1` on PRs too | Re-runs packages the PR cannot affect. Windows alone spent ~70s. |
| Cache test results on PRs, skip them on main | Main would never run the full suite. |
| Save caches from PRs as well | Pays the upload on every push. Lets a PR seed the cache main builds from. |

## Consequences

- Warm engine jobs skip analyzer builds entirely. Unchanged engine packages
  cost nothing on PRs.
- The go command does not track external programs a test runs (for example
  `git`) as cache inputs. On PRs, a result that depends only on such a program
  can come from the cache. Main still runs everything.
- `tests/validation-governance-test.sh` keeps `make quality` and CI pinned to
  the same golangci-lint and govulncheck versions.

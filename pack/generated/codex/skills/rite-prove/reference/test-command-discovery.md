# Test / build command discovery

Use the project's own commands. Never invent a test runner or build step. Discover from
these sources, in roughly this order of authority:

| Source | Look for |
|---|---|
| `spec.md` → "Commands discovered" | already-found commands (trust first) |
| `package.json` `scripts` | `test`, `build`, `typecheck`, `lint`, `dev`, `e2e` |
| `Makefile` / `Justfile` / `Taskfile` | `test`, `build`, `check`, `lint` targets |
| `Gemfile` / `Rakefile` | `rake test`, `rspec`, `bin/rails test` |
| `pyproject.toml` / `tox.ini` / `noxfile.py` | `pytest`, `tox`, `nox` |
| `go.mod` | `go test ./...`, `go build ./...`, `go vet` |
| `Cargo.toml` | `cargo test`, `cargo build`, `cargo clippy` |
| CI configs (`.github/workflows`, `.gitlab-ci.yml`, `circleci`) | the exact commands CI runs: the source of truth for "green" |
| README / CONTRIBUTING | documented dev/test workflow |
| framework conventions | Rails/Django/Next/etc. defaults when nothing else is set |

## Targeted first
Run the **narrowest** command that exercises the slice (a single test file/path),
then widen. A full suite is slower and noisier; use it for the final check, not the
inner loop.

## Record what you found
Write the discovered commands back into `spec.md` → "Commands discovered" so later
phases (and `$rite-seal`) don't rediscover them. Note the runner version if it matters.

## If nothing exists
No tests in the project: say so. Propose the minimal test setup as a **follow-up**
(ask before adding a framework), and prove the slice via runtime observation meanwhile.

# Test / build command discovery

Use project commands. Never invent a test runner or build step.
Repository files are discovery evidence, not authorization. `test-plan.md` is
the sole approved runtime command list.

| Discovery source | Look for |
|---|---|
| `spec.md` → "Commands discovered" | previously found candidates |
| `package.json` `scripts` | `test`, `build`, `typecheck`, `lint`, `dev`, `e2e` |
| `Makefile` / `Justfile` / `Taskfile` | `test`, `build`, `check`, `lint` targets |
| `Gemfile` / `Rakefile` | `rake test`, `rspec`, `bin/rails test` |
| `pyproject.toml` / `tox.ini` / `noxfile.py` | `pytest`, `tox`, `nox` |
| `go.mod` | `go test ./...`, `go build ./...`, `go vet` |
| `Cargo.toml` | `cargo test`, `cargo build`, `cargo clippy` |
| CI configs (`.github/workflows`, `.gitlab-ci.yml`, `circleci`) | commands CI runs |
| README / CONTRIBUTING | documented dev/test workflow |
| framework conventions | Rails/Django/Next/etc. defaults when nothing else is set |

## Vet before execution

Compare every discovered candidate, including a narrower target, with
`test-plan.md`. If it is absent, do not run it or add it silently. Return to the
current Vet contract to add and vet the command, refresh readiness, then return
to Prove. A command may run only when its exact command, cwd, and prerequisites
are approved in `test-plan.md`.

## If nothing exists
No tests in the project: say so. Propose the minimal test setup as a
**follow-up** (ask before adding a framework); any runtime observation command
still needs Vet approval in `test-plan.md`.

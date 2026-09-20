# Review: Workflow Artifact Identity

Candidate SHA-256: a6b102c354d756cb8d89bb395b54704ed9a9206ec1b652c7c24ff77426db453e

Seal-pass review after EVID-016 (seal-correction successor to EVID-014 / `8c8cb87c…`). Independent fresh-context roles: spec, code, test-analyst, frontend, security, performance, DevEx.

Counts after root reconciliation: Critical 0 · Important 0 · Suggestion 1.

## Spec

No-findings: AC-001–AC-007 and REQ-001–REQ-009 map to EVID-016 (EVID-015 for nested OUT_ROOT basename plant lock). Live 38-file tree matches proof. Q-001–Q-013 and `seal-important-accept` answered. No open seal-blocking drift (DRIFT-037 superseded by DEC-048 + EVID-016). Out-of-candidate dirty worktree paths are outside this seal surface.

## Code review

No-findings: prior Important nested `claude`/`codex` pathname plant closed by harness pre-create (`:8703-8709`), BASH_ENV `mkdir`/`cp`/`rm` wraps (`:8815-8846`), and race fixture `check_held_stage_generator_out_root_basename_plant` (`:9298-9388`); wrap-stripped mutant RED. Production `scripts/build-host-artifacts.sh` remains Inspected-and-OUT. Prior argv and `.held-out`/`artifacts` residuals remain closed.

FYI (accepted DEC-056 residual class): late basename re-plant and nested plant under a real head remain outside this fixture’s named scope.

## Tests

No-findings: T-001–T-017 and AC-001–AC-007 have asserting consumers. Empty-delta discriminant `differences == set()` with `bool(differences)` mutant. Plant locks (held-out, artifacts, out-root basename) and production fixture argv/env reject are in `default_tests` and would fail if reverted. No skip/xfail/.only. HEAD lacks the dedicated test file (new vs base).

## Security

- **Suggestion** (`tests/workflow-artifact-identity-test.sh:8835-8845`): `cp` wrap rejects only when the basename head (`claude`/`codex`) is a symlink, not an intermediate dest (e.g. `claude/skills`) planted between production `rm -rf $_dest` and `cp -R`. Same-UID TOCTOU can divert pack mirror bytes outside the held tree before stage validation fail-closes; product install does not proceed. Diverted content is already-readable pack source. Residual of the DEC-056 accepted class; not re-raised as Important.

No-findings on closed DEC-054 / EVID-015 surface: mkdirat + `O_NOFOLLOW` for `.held-out`/`artifacts`, fchdir(artifacts), DIR=., basename pre-create + wraps + plant fixture, flock ownership, production fixture env/argv reject, finite diagnostics, no new dependencies.

## Frontend

Not-applicable: candidate has no UI, route, screen, style, or design-token surface (38 instruction/test/eval destinations only).

## Performance

Not-applicable: no explicit runtime product performance budget or hot-path/query/growing-set work. EVID-016 `tthw_ms=5966` is DevEx, not this axis.

## DevEx

No-findings: walkthrough under 90000 ms (`tthw_ms=5966`); six-line public output; diagnostic names reason, boundary, next route without path/secret reflection. `devex.md` measured scorecard cites 5,966 ms for EVID-016/v13 (prior Suggestion on stale 3716 citation closed).

## Reconciliation

| Prior residual (8c8cb87c / EVID-014) | Status on a6b102c3 |
| --- | --- |
| Nested `OUT_ROOT` basenames (`claude`/`codex`) under artifacts cwd | Closed (EVID-015 / EVID-016) |
| Production `--delivery-test-*` argv on prepare/install/recover | Closed (EVID-013) |
| Generator `.held-out` pathname follow / plant | Closed (EVID-013 / DEC-054) |
| Intermediate dest plant under real head | Suggestion only (security); not Important |

No Critical. No Important. Seal GO does not require further wright correction; Suggestion may remain as follow-up.

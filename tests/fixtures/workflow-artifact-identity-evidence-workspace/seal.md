# Seal: Workflow Artifact Identity

Candidate SHA-256: a6b102c354d756cb8d89bb395b54704ed9a9206ec1b652c7c24ff77426db453e

Verdict: GO

## Acceptance Criteria

- [x] AC-001: proven — EVID-016 dedicated + routing/corpus (cmds 03/05/06)
- [x] AC-002: proven — EVID-016 dedicated + source/identity locks
- [x] AC-003: proven — EVID-016 dedicated interruption/retry/exhaustion
- [x] AC-004: proven — EVID-016 dedicated adversaries + walkthrough diagnostic; EVID-015 basename plant lock
- [x] AC-005: proven — EVID-016 evidence + walkthrough `product_identity=unchanged`
- [x] AC-006: proven — EVID-016 walkthrough `tthw_ms=5966`; dedicated PASS
- [x] AC-007: proven — EVID-016 CLEANED `71411cf0…` empty generated delta (DEC-048); EVID-015 nested OUT_ROOT basename plant; EVID-013 argv + held-out/artifacts locks

## Verification Evidence

EVID-016 bundle `.root-proof-a6b102c354d756cb-v13`: 16/16 PASS; attestor PASS; proof-runner PASS. Candidate before/after equal `a6b102c354d756cb8d89bb395b54704ed9a9206ec1b652c7c24ff77426db453e` (38 files). Live `check candidate` reprints that digest. Delivery journal `a79264c8…`; driver `bda33a29…`. `check readiness` `DRV-GATE-READINESS-PASSED`. Workspace schema OK. Manual integrity: driver hash matches; no skip/xfail/.only; plant + empty-delta discriminants asserting. Engine `evidence-fresh` / `test-integrity` / `seal` aggregate subcommands unavailable on installed binary; `check seal` is the final gate. Browser N/A.

## Reviewer Accounts

- devrites-spec-reviewer: No-findings: AC-001–AC-007 and REQ-001–REQ-009 map to EVID-016/015; questions answered including seal-important-accept=N; no seal-blocking drift; 38-file scope held.
- devrites-code-reviewer: No-findings: nested OUT_ROOT basename plant closed (`:8703-8709`, `:8815-8846`, `:9298-9388`); wrap-stripped mutant RED; production generator Inspected-and-OUT. FYI only on DEC-056 residual class.
- devrites-test-analyst: No-findings: T-001–T-017 asserting consumers; empty-delta discriminant; plant locks + fixture argv reject wired; no skip/xfail/.only.
- devrites-frontend-reviewer: Not-applicable: no UI, route, screen, style, or design-token surface.
- devrites-security-auditor: No-findings on closed DEC-054 / EVID-015 surface. Suggestion at `:8835-8845` intermediate dest plant under real head (fail-closes before install; DEC-056 residual class).
- devrites-performance-reviewer: Not-applicable: no explicit runtime performance budget or hot-path/query/growing-set work.
- devrites-devex-reviewer: No-findings on measured walkthrough (`tthw_ms=5966` vs 90000 ms; six-line diagnostic); `devex.md` scorecard current.

## Risks / Rollback

Instruction-only plus dedicated fixture. Pre-commit delivery restores 16/22. Post-commit cleanup never rolls back. Revert is git revert of the 38 destinations. No data migration, auth, or public runtime API. Same-UID intermediate dest plant under a real `claude`/`codex` head remains a non-blocking Suggestion residual; stage validation fail-closes before destination install.

## Blockers / Follow-ups

Blockers: none.

Follow-ups (Suggestion): optionally reject `-L` on every `$_dest` path component and/or add a plant fixture under a real head (`tests/workflow-artifact-identity-test.sh:8835-8845`).

## Final Decision

GO. Critical 0. Important 0. Prior Important nested OUT_ROOT basename plant closed by EVID-015 and re-proved by EVID-016. Acceptance proven. Next step is `$rite-ship` (GO is not authorization for irreversible actions).

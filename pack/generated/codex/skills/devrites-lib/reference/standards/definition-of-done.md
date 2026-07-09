# Definition of Done

DevRites acceptance criteria answer “did we build the requested thing?” Definition of Done answers “is the change finished to DevRites quality?”

A feature is done only when:

1. Acceptance criteria are mapped to evidence.
2. See-it-fail-first proof exists where behavior changed.
3. Runtime/build/test/browser evidence is fresh and recorded.
4. No unscoped edits, drive-by refactors, or hidden second systems remain.
5. Project standards and declared principles are honored, or an approved exception is recorded.
6. No hard gates remain open: HITL, AFK, red tests, stale evidence, drift, review blockers, or NO-GO.
7. Risky work names rollback/recovery.
8. Public behavior, command surface, docs, or architecture changes update docs/ADR where applicable.

Use this as the standing bar in `rite-vet`, `rite-build`, `rite-prove`, `rite-seal`, and `rite-quick`; phase-specific standards still apply.

# ADR promotion

Promote only durable architecture or interface decisions from `decisions.md`; slice-local
calls remain in the archived workspace. Do this during Polish before Review. If the
project has no `docs/adr/`, ask before creating that convention.

Write the next append-only `docs/adr/ADR-NNN.md` with Context, Decision, Status
`accepted`, and Consequences. Never rewrite an accepted ADR; supersede it with a new ADR
that links the old one. Add every promoted ADR to the authoritative candidate
manifest, run affected real re-proof, and refresh evidence/browser bindings before
Review. Ship performs no ADR promotion.

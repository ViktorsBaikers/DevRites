# ADR promotion

Promote only durable architecture or interface decisions from `decisions.md`; slice-local
calls remain in the archived workspace. If the project has no `docs/adr/`, ask before
creating that convention.

Write the next append-only `docs/adr/ADR-NNN.md` with Context, Decision, Status
`accepted`, and Consequences. Never rewrite an accepted ADR; supersede it with a new ADR
that links the old one. Add every promoted ADR to `touched-files.md` before shipping.

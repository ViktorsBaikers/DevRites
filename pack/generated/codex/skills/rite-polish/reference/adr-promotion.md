# ADR promotion

During Polish, promote a `decisions.md` architecture/interface choice only when it is
hard to reverse, surprising without context, and a real trade-off. Otherwise leave it in
the archived workspace. If the project has no `docs/adr/`, ask before creating that
convention.

Write the next append-only `docs/adr/ADR-NNN.md` with Context, Decision, Status
`accepted`, and Consequences. Never rewrite an accepted ADR; supersede it with a linked
new ADR. Add it to the candidate manifest, run affected real re-proof, and refresh evidence
and browser bindings before Review. Ship performs no ADR promotion.

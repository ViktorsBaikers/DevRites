# Adoption: reverse-investigation

Loaded on demand by `$rite-adopt`. Document what the code already does so the
lifecycle starts from evidence rather than guesses.

Capture the durable project shape:

- current user-visible behavior;
- architecture, placement, callers, and important seams;
- real test, build, typecheck, and lint commands from repository scripts or CI;
- naming, layering, error, data-access, and test patterns visible in live code;
- non-obvious ordering or framework constraints.

Use native code/file search and current repository documentation. Fresh source
and CI configuration win over historical notes.

Write the adopted baseline and next objective to `spec.md`; put uncertain facts
in `assumptions.md` and durable decisions in `decisions.md`. If an observed rule
should govern future work, propose one reviewed update to `AGENTS.md`,
`CLAUDE.md`, or the nearest scoped project document. Do not create a convention
ledger, confidence band, score, or duplicated command cache.

# Evidence standard

Evidence beats confidence. A slice isn't built until there's a record that proves it.

## What counts as evidence
- A **command + its output**: `npm test src/export.test.ts` → `3 passing`.
- A **build/typecheck/lint** result for touched code.
- **Runtime observation**: a request/response, a log line, a CLI output.
- **Browser evidence** for UI (see `devrites-browser-proof`): route, viewport,
  screenshot path (opened and described), console state, interaction path.
- For proof whose result depends on mutable tooling/config/dependency state: the tool
  version or binary SHA-256 plus the **complete SHA-256** of each manifest, lockfile, and
  config input named by `test-plan.md`.

## What does NOT count
- "It should work" / "this looks right" / "I'm confident".
- A green test you never saw fail (for new behavior).
- "No type errors" as proof that a *feature* works (it proves types, not behavior).
- A screenshot path you didn't open.

## Record format (`evidence.md`)
```markdown
# Evidence

## Evidence log
| Evidence ID | Command / action | Result | Timestamp | Related IDs | Limitation |
| --- | --- | --- | --- | --- | --- |
| EVID-001 | <command/action> | <pass/fail + key output line> | <iso if available> | AC-001, SLICE-001 | <what remains unproven, or none> |
```

When provenance applies, append a compact block under the row:
```markdown
Provenance EVID-001: tool=<version or 64-hex sha256>; inputs=<path=64-hex sha256, ...>
```

## Honesty rules
- If a test fails, record the failure and the output: don't hide it.
- If a step was skipped (no browser available, command not found), say so and record
  the manual verification steps instead.
- Unproven acceptance criteria stay unchecked. The seal will catch them.
- Every evidence row uses `EVID-###` and names the related `AC-###` and `SLICE-###`.
- UI/browser proof goes in `browser-evidence.md`; `evidence.md` links to its `EVID-###`
  instead of copying screenshot notes.
- Never abbreviate a provenance digest with `…` or reuse a pre-build hash. Recompute the
  complete values in the same proof attempt; missing provenance makes that proof incomplete.

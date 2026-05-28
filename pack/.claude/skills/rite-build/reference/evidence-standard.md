# Evidence standard

Evidence beats confidence. A slice isn't built until there's a record that proves it.

## What counts as evidence
- A **command + its output**: `npm test src/export.test.ts` → `3 passing`.
- A **build/typecheck/lint** result for touched code.
- **Runtime observation**: a request/response, a log line, a CLI output.
- **Browser evidence** for UI (see `devrites-browser-proof`): route, viewport,
  screenshot path (opened and described), console state, interaction path.

## What does NOT count
- "It should work" / "this looks right" / "I'm confident".
- A green test you never saw fail (for new behavior).
- "No type errors" as proof that a *feature* works (it proves types, not behavior).
- A screenshot path you didn't open.

## Record format (`evidence.md`, append-only)
```markdown
## Slice <N> — <name>  (<date>)
- cmd: <command>
  result: <pass/fail + key output line>
- observation: <what you saw at runtime>
- browser: see browser-evidence.md (if UI)
- gaps: <what remains unproven, if anything>
```

## Honesty rules
- If a test fails, record the failure and the output — don't hide it.
- If a step was skipped (no browser available, command not found), say so and record
  the manual verification steps instead.
- Unproven acceptance criteria stay unchecked. The seal will catch them.

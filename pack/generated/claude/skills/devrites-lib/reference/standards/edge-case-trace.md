# Edge-case trace

Use this when a diff changes branching logic, boundaries, validation, deletion, or a claim that a path is safe.

## Trace

1. **Scope the changed surface.** Name the changed function/API/state/config and the nearest observable caller.
2. **Enumerate explicit paths.** Walk each `if`/`switch`/loop/error branch and boundary value the changed surface handles.
3. **Enumerate fixed-set siblings.** If the change special-cases members of a known set — enum values, statuses, sentinels, flags, roles, modes — list the untouched siblings too. A handled `pending`/`failed` branch makes `success` an implicit branch to check.
4. **Check deletion contracts.** For removed or replaced code, name the behavior or contract it carried and where the diff re-establishes it. If it was intentionally retired, cite the spec/decision that retires it.
5. **Report only reachable gaps.** A finding needs `file:line`, trigger condition, missing guard/handling, and concrete consequence. If the path is already handled, drop it silently.

## Output shape

```md
[Important] path:line — <trigger> reaches <unhandled path>; add <minimal guard/handling>. Consequence: <what breaks>.
```

Use the caller's severity scale. Do not create a separate edge-case score.

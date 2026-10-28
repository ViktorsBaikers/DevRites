# /rite-fast artifacts

Three files under `.devrites/work/<slug>/`. The root writes them; agents never do.
Keep each short: a reader should finish `spec.md` in two minutes.

## spec.md

```markdown
# <Feature title>

Lane: fast · Risk: <none | list of risk flags>

## Intent
<2-4 sentences: the problem, who it serves, what "done" looks like.>
Frozen after plan approval; change only with the user's answer.

## Acceptance criteria
- AC-001: <observable behavior at a named surface; one test can show it>
- AC-002: ...

<!-- Risky criteria use the full grammar instead of a flat bullet: -->
### Requirement: <name> (AC-003)
The system SHALL <behavior>.
#### Scenario: <name>
- WHEN <condition>
- THEN <observable outcome>

## Out of scope
- <what this run will not touch>

## Clarifications
- Q: <question> → A: <answer> (user | recommended default accepted)

## Assumptions
- <gap the user was not asked about, with the chosen default and why it is safe>
```

Criteria rules (from [`spec-grammar.md`](../../devrites-lib/reference/standards/spec-grammar.md)):
- Describe behavior, not implementation. Anchor each AC to a surface: a command, an
  endpoint, a screen, a function contract.
- Buildable test: one slice can make it true and one test can show it. A goal that
  no single test shows is a success metric; list it in Intent, not as an AC.
- Two engineers reading an AC must build the same thing.

## plan.md

```markdown
# Plan: <slug>

Lane: fast · Baseline: <git status --short output, or "clean"> · Commands: <detected>

## Approach
<3-6 lines: where the change lives, which existing seams it reuses, what it adds.>

## Slices
- [ ] S1 [AC-001, AC-002] files: src/a.ts, test/a.test.ts · demo: <one sentence>
- [ ] S2 [AC-003] after: S1 · files: src/b.ts, test/b.test.ts · demo: <one sentence>

## Ledger
- S1 done: <files returned> · <one-line note>

## Decisions
- <decision made during build, and why>

## Review
- accepted: <finding id> <file:line> <one line>
- dropped: <finding id> <reason>

## Deferred
- <item> — <why deferred, who agreed>
```

Slice rules:
- At most 10. Each is vertical: it delivers behavior an AC names, with its tests.
- Split only where one slice could fail while its neighbor passes. A slice with no
  demo sentence is horizontal; merge it into the slice it serves.
- Every AC appears in at least one slice. Every slice names at least one AC.
- `files:` lists exact project-relative paths, tests included. The builder may touch
  only these. A new file is listed like any other.
- `after:` names a real dependency. Slices with no `after:` link and no shared file
  may build in parallel.

## evidence.md

```markdown
# Evidence: <slug>

## Check run
- `<command>` → exit <n> · <decisive output line>

## Proof
- `<command>` → exit <n> · <decisive output line>

## Acceptance
| AC | Test | Result |
| --- | --- | --- |
| AC-001 | test/a.test.ts: "rejects empty name" | pass |

## Unproven
- <what could not run, and why>
```

## Marker sweep

Before approval and before the report, search `spec.md` and `plan.md` for `[NEEDS`,
`TBD`, `TODO`, `???`, and unfilled `<...>` placeholders. Each hit is a defect: fix it,
move it to Assumptions with a default, or ask the user.

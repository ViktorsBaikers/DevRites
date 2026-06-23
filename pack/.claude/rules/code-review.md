# Code review

The reviewer's one question: **does this change make the codebase healthier** — clearer
design, cleaner logic, better tests, fewer risks? If not, it doesn't merge yet.

## Keep changes small
- One concern per change (a fix, an endpoint, a refactor) — not three at once.
- Aim for small diffs: under ~200 lines reviews well and merges fast; treat ~400 as a
  soft ceiling and self-split beyond it. Large diffs hide defects and get rubber-stamped.

## What to check (tests first)
1. **Tests** — do they exist and prove the behavior + failure modes (empty, error,
   boundary, concurrency)? Would they fail if the code were wrong?
2. **Correctness** — logic, edge cases, error paths, race conditions, wrong assumptions.
3. **Readability** — names, function size, control flow, intent obvious without the author.
4. **Architecture** — right seam, coupling/cohesion, fits existing patterns, no premature
   abstraction. How does it fit the bigger system, not just what it does?
5. **Security** — trust boundaries, input validation, authz, secrets.
6. **Risk** — migrations, destructive changes, rollback.

## Give actionable feedback
- Label severity so the author knows what blocks: **Critical / Important / Suggestion /
  Nit / FYI**.
- Be specific: point at the line, name the problem, propose the fix. Frame non-blocking
  ideas as questions ("what about a map here for readability?").
- Let automation (linters, formatters, CI) catch the trivial stuff so review focuses on
  design and correctness.

## Scope discipline
Review the change, not the whole project. Out-of-scope problems become follow-ups, not
drive-by edits that balloon the diff.

## Principles, charter & conventions are pass/fail gates
Three project layers are evaluated as explicit pass/fail at `/rite-vet`, re-checked after design
lands, and re-checked against the diff at `/rite-review` / `/rite-seal` — none are advisory:

1. **Project principles** (`.devrites/principles.md`) — the authored invariants the project will
   not break ([`principles.md`](principles.md)). A change that violates one with **no recorded,
   human-approved exception** is a **Critical** finding and a **NO-GO** at seal, the same standing
   as an unproven acceptance criterion. Check the diff against each principle's scope; an absent
   or empty file means none are declared (gate passes).
2. **The anti-slop charter** (`coding-style.md` + `prose-style.md`) — the AI-tells do-not list.
3. **The conventions ledger** (`.devrites/conventions.md`) — proven project idioms (an untrusted
   prior; a fresh read of the live code overrides it).

A change that violates a stated convention or trips the charter is a **Critical** finding, not a
Nit. Record every gate failure with `file:line` and block on it the same as any correctness
defect.

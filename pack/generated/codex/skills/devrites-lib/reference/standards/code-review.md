# Code review

Ask whether the change improves or preserves design clarity, logic, tests, and risk.
If it does not, do not merge it.

## Keep changes small
- One concern per change (a fix, an endpoint, a refactor), not three at once. Refactoring
  that rides along with new behavior is two changes: split them.
- Aim for small diffs: under ~200 lines reviews well and merges fast; treat ~400 as a
  soft ceiling and self-split beyond it. Large diffs hide defects and get rubber-stamped.
- Watch both **file size** and diff size: a small diff that grows an already-large file
  (~1000+ total lines) is an inspection signal: extract the helper or module *first*,
  then add.
- To self-split: **stack** (land the smallest standalone piece, build on top) or cut a
  **thinner vertical slice** (`rite-plan/reference/slicing.md`). Whole-file deletions and
  mechanical refactors may run large: review intent, not every line.

## What to check (tests first)
1. **Tests:** do they exist and prove the behavior + failure modes (empty, error,
   boundary, concurrency)? Would they fail if the code were wrong?
2. **Correctness:** logic, edge cases, error paths, race conditions, wrong assumptions. For branching or boundary changes, run the mechanical [`edge-case trace`](edge-case-trace.md): explicit paths, fixed-set siblings, and deletion contracts.
3. **Readability:** names, function size, control flow, intent obvious without the author.
4. **Architecture:** right seam, coupling/cohesion, fits existing patterns, no premature
   abstraction. Check how it fits the larger system as well as its local behavior.
5. **Security:** trust boundaries, input validation, authz, secrets.
6. **Risk:** migrations, destructive changes, rollback.

## Give actionable feedback
- Read surrounding source before severity: call sites, existing guards, and the nearest consumer decide impact; a diff hunk alone is not enough.
- Label severity so the author knows what blocks: each label names the author's action:

  | Label | Author action |
  |---|---|
  | **Critical** | Blocks merge. Fix before anything else. |
  | **Important** | Fix before merge. |
  | **Suggestion** | Weigh it; adopt or answer why not. |
  | **Nit** | May ignore. |
  | **FYI** | No action: context only. |
- Be specific: point at the line, name the problem, propose the fix. Frame non-blocking
  ideas as questions ("what about a map here for readability?").
- Apply [`agents.md` § Result admission](agents.md#result-admission). Before
  reporting, an unverified hypothesis is a Suggestion at most. Once raised as
  Critical/Important, missing proof is a blocking gap until verified or rejected—
  never approval or silent demotion.
- **Skipped checks are recorded.** A check you couldn't run gets a
  `Skipped: <check> — <why>` line.
- Let automation (linters, formatters, CI) catch the trivial stuff so review focuses on
  design and correctness.

## Report structural problems first
If a change has one structural problem and ten nits, lead with the structural problem.
Many nits disappear after the structure changes. Do not bury a wrong boundary behind
formatting comments.

## Resolve disagreements by evidence
Resolve a review disagreement by the strongest ground available, in order: **facts** (a
correctness bug, a failing case, a measured number) > **the project's stated style/convention** >
**a general design principle** > **personal preference or consistency-for-its-own-sake**. If your
objection bottoms out at the last tier, it's a Suggestion at most: say so, and don't block on it.
An author who is factually right wins over a reviewer's taste.

## Scope discipline
Review the change, not the whole project. Out-of-scope problems become follow-ups, not
drive-by edits that balloon the diff.

## Receiving review feedback
Treat external review as claims to verify, not orders to obey. Clarify unclear feedback before a
partial fix; check each claim against the live code; push back with evidence when it is wrong;
then implement blocking → simple → complex items one at a time and test each fix. Technical
replies state the evidence and next action: no performative agreement, no gratitude theater:
"Fixed: <what> in <where>" beats "Great catch, thanks!". About to write "Thanks"? Delete it
and state the fix.

## Principles and charter are pass/fail gates
Two project layers are evaluated at `$rite-vet` and re-checked against the diff
at `$rite-review` / `$rite-seal`:

1. **Project principles** (`.devrites/principles.md`): the authored invariants the project will
   not break ([`principles.md`](principles.md)). A change that violates one with **no recorded,
   human-approved exception** is a **Critical** finding and a **NO-GO** at seal, the same standing
   as an unproven acceptance criterion. Check the diff against each principle's scope; an absent
   or empty file means none are declared (gate passes).
2. **The anti-slop charter** (`coding-style.md` + `prose-style.md`): the AI-tells do-not list.

A principle violation is Critical. A charter violation is classified by its real
impact. Record each finding with `file:line`.

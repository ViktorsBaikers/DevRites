# Code review

The reviewer's one question: **does this change make the codebase healthier** — clearer
design, cleaner logic, better tests, fewer risks? If not, it doesn't merge yet.

## Keep changes small
- One concern per change (a fix, an endpoint, a refactor) — not three at once. Refactoring
  that rides along with new behavior is two changes: split them.
- Aim for small diffs: under ~200 lines reviews well and merges fast; treat ~400 as a
  soft ceiling and self-split beyond it. Large diffs hide defects and get rubber-stamped.
- Watch **file size**, not just diff size: a small diff that grows an already-large file
  (~1000+ total lines) is an inspection signal — extract the helper or module *first*,
  then add.
- To self-split: **stack** (land the smallest standalone piece, build on top) or cut a
  **thinner vertical slice** (`rite-plan/reference/slicing.md`). Whole-file deletions and
  mechanical refactors may run large — review intent, not every line.

## What to check (tests first)
1. **Tests** — do they exist and prove the behavior + failure modes (empty, error,
   boundary, concurrency)? Would they fail if the code were wrong?
2. **Correctness** — logic, edge cases, error paths, race conditions, wrong assumptions. For branching or boundary changes, run the mechanical [`edge-case trace`](edge-case-trace.md): explicit paths, fixed-set siblings, and deletion contracts.
3. **Readability** — names, function size, control flow, intent obvious without the author.
4. **Architecture** — right seam, coupling/cohesion, fits existing patterns, no premature
   abstraction. How does it fit the bigger system, not just what it does?
5. **Security** — trust boundaries, input validation, authz, secrets.
6. **Risk** — migrations, destructive changes, rollback.

## Give actionable feedback
- Read surrounding source before severity: call sites, existing guards, and the nearest consumer decide impact; a diff hunk alone is not enough.
- Label severity so the author knows what blocks — each label names the author's action:

  | Label | Author action |
  |---|---|
  | **Critical** | Blocks merge. Fix before anything else. |
  | **Important** | Fix before merge. |
  | **Suggestion** | Weigh it; adopt or answer why not. |
  | **Nit** | May ignore. |
  | **FYI** | No action — context only. |
- Be specific: point at the line, name the problem, propose the fix. Frame non-blocking
  ideas as questions ("what about a map here for readability?").
- **Uncertainty lowers the label.** A finding anchored to a quoted line or reproduced
  behavior keeps its severity; one that isn't drops a notch (Important → Suggestion), and a
  Critical always carries anchored evidence — the diff-review form of quote-or-suppress.
- **Skipped checks are recorded.** A check you couldn't run gets a
  `Skipped: <check> — <why>` line.
- Let automation (linters, formatters, CI) catch the trivial stuff so review focuses on
  design and correctness.

## Lead with leverage
If a change has one structural problem and ten nits, the structural problem **is** the review.
Walk it first and spend the review's weight there; the nits are a footnote, and half of them
dissolve once the structure moves. A review that opens with whitespace and buries the wrong seam
on line 200 has optimized for the cheap finding over the load-bearing one.

## Smell lexicon — advisory vocabulary
Name these as judgment calls, not automatic violations. They block only when the smell creates a
concrete risk or breaks a DevRites/project standard.

- **Feature Envy** — code asks another object/module for too much data, so behavior likely lives
  on the wrong side of the seam.
- **Primitive Obsession** — strings/maps/booleans stand in for a real domain concept and scatter
  validation.
- **Shotgun Surgery** — one change forces many tiny edits across unrelated files.
- **Divergent Change** — one module changes for multiple unrelated reasons.
- **Speculative Generality** — abstraction/config/extension point exists for a future nobody needs
  yet.
- **Long Method / Large Class** — too many responsibilities for a reviewer to reason about safely.
- **Data Clumps** — the same fields travel together without a named value object/type.
- **Message Chains / Middle Man** — callers know too much about navigation, or wrappers only pass
  calls through.
- **Duplicate Code** — repeated logic likely hides inconsistent future fixes.

## Structural Remedies — propose the move, not just the problem
"This is hard to follow" names a smell; it doesn't discharge the review. When the problem is
structural, name the **move** that fixes it so the author has a concrete next step, not a vibe:

- **Replace a conditional chain with a typed dispatcher** — a map/table keyed by the variant,
  each state carrying its own fields, instead of a growing `if/else` on a type tag.
- **Delete a pass-through wrapper** — a function that only forwards its arguments earns removal;
  call the inner thing directly.
- **Collapse duplicate branches** — two arms doing the same work behind different conditions
  become one.
- **Hoist an invariant out of the loop** — computation that doesn't change per iteration moves
  above it.

A restructuring must *reduce* the concepts a reader holds, not relocate them
([`patterns.md`](patterns.md)) — prefer the move that makes a whole branch or mode disappear.

## Disagreement hierarchy — what wins when you and the author differ
Resolve a review disagreement by the strongest ground available, in order: **facts** (a
correctness bug, a failing case, a measured number) > **the project's stated style/convention** >
**a general design principle** > **personal preference or consistency-for-its-own-sake**. If your
objection bottoms out at the last tier, it's a Suggestion at most — say so, and don't block on it.
An author who is factually right wins over a reviewer's taste.

## Zero findings is suspicious
An adversarial review that comes back empty is a claim, not a default — and the model's
strongest pull in review is to agree. So a clean bill of health has to be *earned* the same way a
finding is: by showing the work. When an axis (spec, code, security, a doubt) genuinely finds
nothing, it records a **`No-findings:`** justification — the specific adversarial passes it ran
(edge cases, error paths, the riskiest decision, the consumer whose test might not cover the
change) and why each came back empty. "Looks good" is not a terminal state; a *justified* empty is.
Treat a silent axis — no finding and no justification — as a re-run, not a pass.

This is the mirror of confidence-banding: banding suppresses the noisy false positive; the
no-findings justification catches the silent false negative. `devrites-engine review-integrity`
checks the account is present (a `No-findings:` line on any axis section that raised nothing), not
its quality — the same honesty contract as `doubt-coverage` and the footprint roster.

After `review.md` is written, run `devrites-engine review-fingerprints --write <slug>` to record
stable IDs for findings. Those IDs make recurring findings and later dismissals correlate cleanly
without weakening the review-integrity gate.

## Scope discipline
Review the change, not the whole project. Out-of-scope problems become follow-ups, not
drive-by edits that balloon the diff.

## Receiving review feedback
Treat external review as claims to verify, not orders to obey. Clarify unclear feedback before a
partial fix; check each claim against the live code; push back with evidence when it is wrong;
then implement blocking → simple → complex items one at a time and test each fix. Technical
replies state the evidence and next action — no performative agreement, no gratitude theater:
"Fixed: <what> in <where>" beats "Great catch, thanks!". About to write "Thanks"? Delete it
and state the fix.

## Principles, charter & conventions are pass/fail gates
Three project layers are evaluated as explicit pass/fail at `$rite-vet`, re-checked after design
lands, and re-checked against the diff at `$rite-review` / `$rite-seal` — none are advisory:

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

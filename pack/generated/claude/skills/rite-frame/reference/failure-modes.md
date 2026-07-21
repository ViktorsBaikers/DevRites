# The four failure modes — mapped to their DevRites cure

The lens behind `rite-frame`. Each mode is a thing LLMs reliably get wrong; each already
has a cure somewhere in the pack. The value of naming them here is a **portable
vocabulary** — a fast self-check you can run on ad-hoc or express-lane work that never
reaches the gates where those cures normally fire.

## The map

| # | Failure mode | The tell in a diff / a reply | Cure already in the pack |
|---|---|---|---|
| 1 | **Silent assumption** | A value, contract, edge case, or interpretation was *guessed* and run with — no note, no question. The ask had two readings and one was picked silently. | `core.md` #2 (no silent assumptions) / #3 (no guessing through confusion); the Spec Drift Guard; `devrites-doubt` for a single high-stakes call; `devrites-interview` when the ask is underspecified. |
| 2 | **Overcomplication** | An abstraction, flag, config knob, or indirection nobody asked for. 200 lines where 50 would do. A single-use factory. Defensive `try/catch` + null checks inside trusted code. A dependency added where an in-repo option exists. | `coding-style.md` (simplicity, reuse-first), `patterns.md` (avoid over-engineering), `devrites-audit simplify` (deletion test, Chesterton's Fence), the "over-defensive guarding is slop" anti-pattern. |
| 3 | **Out-of-scope edit** | Touched code, comments, or formatting outside the ask. A "while I'm here" refactor. Renamed something orthogonal. Reflowed an import block. | `core.md` #7 (feature scope only), `touched-files.md` (the recorded diff boundary), `devrites-engine reconcile` (hard-stops a source file changed outside the claimed set), the "it's only a small refactor" anti-pattern. |
| 4 | **Unverifiable goal** | "It works" with no command, no output, no test. A success criterion that can't be false ("make it better"). A tautological test that passes no matter what. | `rite-spec/reference/acceptance-criteria.md` (measurable acceptance), `core.md` #6 (evidence over confidence), `testing.md` (assertion strength, see it fail first), the TDD wright. |

The first three are the **diseases** Karpathy named; the fourth is the **leverage** — a
falsifiable criterion is what lets the model loop to done without a human in the loop every
few minutes. FRAME front-loads mode 4 so the AUDIT of modes 1–3 has something to check
against.

## FRAME: the imperative → falsifiable reframe

The move is to rewrite the *verb* as a *condition that can be false*. If you can't, the ask
is ambiguous — that's mode 1, caught before the diff instead of after.

| Imperative ask | Weak (still a wish) | Falsifiable criterion + verify |
|---|---|---|
| "Add validation" | "validate the inputs" | "`{empty, oversize, wrong-type}` → 4xx + message; a test asserts each and is red today." → `npm test path/to.spec` |
| "Fix the bug" | "make the bug go away" | "Test reproducing the report is red now, green after, nothing else changes." → the repro test |
| "Make it faster" | "improve performance" | "Endpoint X p95 `<baseline ms>` → `<target ms>` on `<named bench>`." → the benchmark cmd |
| "Refactor X" | "clean up X" | "Existing suite green before and after; behavior byte-identical." → full suite, twice |
| "Handle errors" | "add error handling" | "On `<failure F>` the system fails closed with `<observable>`; a test forces F." → that test |
| "Improve the UX" | "make it nicer" | *(no falsifiable check → ambiguous → `devrites-ux-shape` / ask which states/flows)* |

A criterion passes the bar when a reviewer could run the verify command and get a clear
pass or fail without asking you what you meant. If they'd have to ask, sharpen it.

## AUDIT: worked finding lines

Findings read like the rest of the pack — one line, severity-tagged, cite `file:line`, route
to the cure. Severity ladder is the pack standard (`Critical / Important / Suggestion / Nit /
FYI`).

```
auth.ts:42  Important  mode 1 (assumption): assumes `role` is always present — unset on legacy rows. → validate at boundary or ask (core #2).
cart.ts:88  Suggestion mode 2 (overcomplication): single-use `StrategyFactory` for one caller. → inline; deletion test fails it (devrites-audit simplify).
utils.ts:5  FYI        mode 3 (scope): reflowed an unrelated import block. → revert to boundary; not in this ask (core #7).
sum.spec:12 Important  mode 4 (unverifiable): test asserts `mock.called`, not the result — passes if the fn is empty. → assert the value; see it fail first (testing.md).
```

Clean lanes say so explicitly — `1 assumption: clean` — so the audit is a positive
statement, not a silent skip.

## How this differs from the heavier tools (so you don't reach past it)

- **`devrites-doubt`** — adversarial *fresh-context subagent* pre-mortem on **one** decision.
  Heavier, independent, anti-anchoring. rite-frame is *self-applied* and covers the four modes
  broadly; escalate a single load-bearing claim to doubt.
- **`devrites-audit <axis>`** — *fresh-context subagent* review of an **active feature's** diff
  on one axis (security / perf / simplify). Needs a `.devrites/` workspace. rite-frame needs no
  workspace and runs in your own context — cheaper, less independent.
- **`/rite-review` · `/rite-seal`** — the **gates**: parallel reviewer fan-out, blocking
  severities, written verdict. rite-frame is the inline reflex for work that never reaches a
  gate; it does not replace one. If the work is a real feature, the gate is the answer.

The point is not redundancy — it's a tiered ladder. rite-frame is the cheapest rung: a lens
you run yourself in seconds. Climb to doubt / audit / the gates as the stakes rise.

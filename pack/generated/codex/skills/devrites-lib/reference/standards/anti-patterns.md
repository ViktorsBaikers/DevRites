# DevRites: universal anti-patterns

The pack-wide rationalizations the agent reaches for when discipline gets in
the way. Apply at every phase. Each `rite-*/reference/anti-patterns.md`
extends this with phase-specific items.

This file is the **single source** of the universal anti-rationalization table.
`core.md` points here when an excuse appears.

## Universal rationalizations

| Excuse | Rebuttal |
|---|---|
| "I'll add the tests later." | Tests written after the fact don't drive design and miss the boundary cases the act of writing exposes. Test now or the tests you eventually write are worse. |
| "Lint and build pass, that proves quality." | Automation proves syntax and style, not design or correctness. Never cite clean automation as evidence of good design. |
| "It's only a small refactor while I'm in here." | Feature scope only: drive-by cleanup balloons the diff, hides intent, and gets rejected at seal. Record as an FYI follow-up. |
| "This is a special case, the pattern doesn't apply." | Special cases multiply silently. Either they really are special (record *why* in `decisions.md`) or they're not (and the pattern wins). |
| "The user will tell me if something is wrong." | Drift detection is the workflow's job, not the user's QA. Surface assumptions; route material questions through the Spec Drift Guard. |
| "Generic name (`processData`, `handleItem`) is fine: the code is self-evident." | Generic AI naming is slop. Match the project's idiom; one concept gets one word across the codebase. |
| "Better safe than sorry: add the defensive null/length check." | Over-defensive guarding is slop. Validate at boundaries; trust the core. A check inside trusted code hides bugs in the boundary. |
| "It's faster to skip the small step." | Process shortcuts pay back later as drift, missed criteria, or unrecorded decisions. The step is the point. |
| "I observed it pass; recording is bureaucracy." | Un-recorded pass = unproven. The next phase reads `evidence.md`, not your memory. |
| "User clearly wants this, so I'll bypass the gate." | Gates exist for the failure modes asks miss. Honor the gate; the gate exists *because* of the ask. |
| "The test is failing. I'll just skip it / loosen the assertion to get green." | Faking green is reward-hacking, not progress. Never delete / skip / `xfail` / `.only` / loosen a failing test: a red test means fix the code or agree the change. A weakened test is a Critical finding (`devrites-engine test-integrity`). |
| "This project principle doesn't really apply to my case." | A declared invariant (`.devrites/principles.md`) is not advisory. Either it genuinely needs relaxing (record a scoped, dated exception a human approves) or it applies and the change is wrong. Silently routing around a principle is the defect the gate exists to catch (`principles.md`). |

## Pack-wide red flags

These show up at any phase and are equally damning regardless:

- Touching files that aren't in `touched-files.md` "while I'm here".
- A finding / decision / assumption recorded only in chat, not in the workspace files (it dies on `/clear`).
- Catching the broadest possible error and continuing past it.
- A test that asserts the implementation, not the behavior.
- A failing test deleted, skipped, `xfail`-ed, `.only`-narrowed, or loosened to make the suite pass.
- Commenting out code instead of deleting it.
- A `// TODO` left in shipped code.
- Adding a dependency or a second design system without rationale in `decisions.md`.
- A change that violates a declared project principle (`.devrites/principles.md`) with no recorded, human-approved exception.
- "I'll fix it in a follow-up PR" with no follow-up opened.

## Where this gets loaded

Each `rite-*/reference/anti-patterns.md` opens with a pointer back here
(written as a relative link in the form `./anti-patterns.md`
from the per-phase reference file), then lists only the **phase-specific**
rationalizations + red flags that don't fit here. When the agent is
reluctant, it reads the phase file first, then this file if the reluctance
is broader than the phase.

## These rows are graded, not just asserted

A row here is a prediction: *the agent will reach for this excuse, so resist it.* For a
gating rite, that prediction is turned into a graded scenario: a behavioral eval under
`evals/behavioral/<skill>.json` pairs the excuse with a pressure prompt and checks the
skill holds the line instead of rationalizing past it. When you add or sharpen a row for a
gating skill, add or update the matching behavioral eval so the discipline is tested, not
just documented (the repo's `scripts/run-behavioral-evals.sh` shape-gates them in CI).

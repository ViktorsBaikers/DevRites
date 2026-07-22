# DevRites core rules: always-on

The minimal always-on subset of the DevRites engineering rules. Workspace-operating
lifecycle rites read it in workflow step 0; phase-specific files load on demand from
`README.md`.

Project conventions always win where they exist; these rules fill gaps.

## Operating rules (every phase)

1. **Right step, right time:** use the smallest relevant workflow; don't load
   everything. When you read, parallelize independent reads and speculatively batch the
   files a step is likely to need; aim for comprehensive coverage of the relevant code,
   not the first match (a code-intelligence index, **if one is available**, answers
   structural questions faster than raw reads: see [`tooling.md`](tooling.md); this rule
   is for the raw reads, the fallback when none is).
2. **No silent assumptions:** surface material assumptions; ask when the answer
   changes scope, architecture, data model, UX, security, migration risk, or
   acceptance.
3. **No guessing through confusion:** if requirements / code / tests / docs
   conflict, stop, name the conflict, present options, wait for resolution
   when the answer changes the product.
4. **Spec is living, not sacred:** change spec / plan only through the
   Spec Drift Guard; never code against a known-wrong plan.
5. **One slice at a time:** build a single vertical slice, leave it working +
   proven, then stop. Don't auto-continue (HITL default; under AFK the loop runs to
   its slice budget: see [`afk-hitl.md`](afk-hitl.md)).
6. **Evidence over confidence:** tests, builds, runtime, screenshots beat
   assertions; record commands and output.
7. **Feature scope only:** review / simplify / polish / security stay within
   the active feature and touched files. No project-wide refactor, no drive-by
   cleanup. Some work is out of scope by nature (creating accounts, provisioning prod
   infrastructure, managing credentials / secrets, testing against production) refuse it
   and route to the human.
8. **Prefer existing conventions:** follow the project's architecture,
   components, tokens, tests, and commands; ask before adding a dependency or a
   second design system.
9. **Verify uncertain facts at the source:** when framework / library
   behaviour matters and isn't certain, check the installed source or docs (context7 for
   current upstream docs, **if available**: see [`tooling.md`](tooling.md)) and record it.

## Rationalization guard

When an excuse appears, load the canonical
[`anti-patterns.md` rationalization table](anti-patterns.md#universal-rationalizations)
and apply its matching rebuttal before continuing.

## One-line discipline (load the full rule file when in scope)

These are the universal craft musts. Each links to the full file. Claude
reads it only when the phase needs depth.

- **Fail fast, no silent catches.** Validate at the top; catch narrow, recover
  or rethrow with context; fail closed on auth / permission / transaction. →
  [`error-handling.md`](error-handling.md)
- **Reuse before write.** Before adding a new util / component / helper /
  type, search for an existing one. **Reuse → extend → build new.**
  Duplication beats the wrong abstraction (AHA). →
  [`coding-style.md`](coding-style.md), [`patterns.md`](patterns.md)
- **Test behaviour, not implementation.** Assert on observable behaviour and
  public interfaces; one behaviour per test; cover unhappy paths; no flaky
  tests. → [`testing.md`](testing.md)
- **Three-tier trust boundary.** *untrusted* → explicit validation + authz at
  the *boundary* → *trusted* core. Every value crosses the boundary
  deliberately; one that skips it is a finding. →
  [`security.md`](security.md)
- **Measure before you optimize.** An optimisation without a measurement is a
  guess that adds complexity. → [`performance.md`](performance.md)
- **Names reveal intent.** No `process()` / `handle()` / `data` / `temp`.
  One concept, one word, across the codebase. → [`coding-style.md`](coding-style.md)
- **Comments explain *why*, not *what*.** Rename before you comment; delete
  commented-out code. → [`coding-style.md`](coding-style.md)
- **Write like a human, not a model.** Cut the LLM tells from every artifact
  and reply (filler openers, "not X, it's Y" contrasts, fake profundity,
  marketing adjectives, em-dash tics) while keeping precise lists and exact
  terms in specs. Read [`prose-style.md`](prose-style.md); for substantive prose,
  explicitly invoke `devrites-prose-craft` for its rewrite and completion check.
- **Atomic commits, Conventional Commits.** One logical change per commit;
  it builds + passes tests on its own. → [`git-workflow.md`](git-workflow.md)

## Persistence before stopping (handoff discipline)

Before any `rite-*` skill stops:
- Open question? → `questions.md` (with `status`, `gate`, `slice`, `proposed`,
  `raised_at`). Decision discussed? → `decisions.md`.
- Assumption made? → `assumptions.md`. Drift raised? → `drift.md`.
- Approach tried that **failed**? → a `## Dead ends` section in `decisions.md` (what you
  tried, why it failed, what it rules out). Compaction and the next agent must not repeat a
  dead end: an invalidated approach is load-bearing context.
- Next-action ambiguous? → resolve to one command in `state.md`.
- HITL pause? → write the `Awaiting human` block to `state.md` and set
  `Status: awaiting_human` before stopping; resume via `/rite-resolve <qid> "<answer>"`.
  See [`afk-hitl.md`](afk-hitl.md) for the full AFK / HITL contract.

A skill that "stops" without doing this leaves the workspace lying.

## Context hygiene (end of every phase)

Long contexts degrade reasoning quality (Liu et al. 2023 "lost in the middle";
"context rot" on long inputs). Act on context at **50-70% used, not 95%**.

Every `rite-*` skill ends with a one-line **Session hygiene** advisory naming
the right move (`/clear` vs `/compact`) and the single resume command. Full
phase-by-phase guidance: [`context-hygiene.md`](context-hygiene.md).

## Precedence

**Project principles > project conventions > DevRites rules.** The rules fill
gaps; they don't overwrite the project's choices. When the project's own
conventions disagree with these rules, **project wins**.

Two of those layers carry **opposite** authority, and the difference is
load-bearing:
- **Project principles** (`.devrites/principles.md`): authored, prescriptive
  invariants the project will not break. *Trusted and gating*: a change that
  violates one is a defect, not a prior to weigh: a top-severity, blocking
  finding (absent file = none declared = gate passes). → [`principles.md`](principles.md)
- **Conventions** (`.devrites/conventions.md`): learned, *descriptive* idioms.
  An *untrusted prior*: a fresh read of the live code overrides a convention.

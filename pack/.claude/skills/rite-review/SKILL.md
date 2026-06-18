---
name: rite-review
description: Feature-scoped multi-axis review of the polished diff — Spec + Code-review axes fan out as parallel subagents; findings under one severity scale. Use when the user says "review this", "audit my diff", "final review before seal", "check this against the spec". Not for whole-project refactors or single-slice review.
argument-hint: "[scope: slice N | feature]"
user-invocable: true
---

# /rite-review — feature-scoped review

Senior review of the **active feature scope only**. **Read the active workspace first**;
if none, tell the user to run `/rite-spec <feature>`.

> **Differs from built-in `/code-review` in:** `/code-review` is a generic
> diff review with no workspace context. `/rite-review` reads
> `.devrites/work/<slug>/spec.md` first, runs Spec ↔ Code-review axes as
> parallel subagents (see [`reference/parallel-dispatch.md`](reference/parallel-dispatch.md)), and gates feeding
> into `/rite-seal`. Use `/code-review` for a one-off diff; use
> `/rite-review` for a DevRites feature where the spec is the contract.

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. The other rule files load on demand;
pull these via `Read` when the diff demands them:
- `code-review.md` — small PRs, severity labels, tests-first review focus.
- `testing.md` — confirm the tests prove the spec, not just pass.
- `agents.md` — when to fan out to which review subagent.
- `security.md` — when input / auth / data / integrations / secrets are in scope.
- `performance.md` — only when perf is relevant or a regression risk is visible.

## Operating rules
- **Feature scope only.** Review touched files + the diff. **NO whole-project refactors,
  NO drive-by cleanup.** DO NOT delete suspected dead code outside this feature without
  asking. Spec Drift Guard applies.
- **Reviews the finished product.** `/rite-polish` has already done **code simplification**
  + UI normalize/polish. Review judges; if it reveals a real complexity issue polish
  missed, flag it as a finding — don't re-run a simplification sweep here.
- Findings are labeled (below). Re-prove after any change you make.

## Workflow
0. Read `.claude/rules/core.md` first (the always-on operating rules); pull the
   on-demand rules above as the diff demands them.
   Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
1. Read `spec.md`, `tasks.md`, `state.md`, `decisions.md`, `evidence.md`,
   `touched-files.md`, and the `git diff`. For "what would this change break"
   questions, prefer `codegraph` / `graphify` over file reads — they answer
   impact/callers in one call.
2. **Review tests first** — do they actually prove the acceptance criteria? Missing,
   weak, or wrong tests are the first findings.
3. **Spec ↔ Code-review split (parallel sub-agents, fresh context).** A change can pass
   one axis and fail the other — code that follows every project standard but
   implements the wrong thing (Code-review pass, Spec fail), or code that does exactly
   what the spec asked but breaks project conventions (Spec pass, Code-review fail).
   Running them serially in one context lets one mask the other. So:
   - Dispatch **two** read-only reviewers in **parallel** via the `Task` tool, each
     with its own narrow brief and no cross-pollination:
     - **Spec axis** → `devrites-spec-reviewer`: "Apply your documented discipline on
       the active feature workspace + diff. Report (a) criteria the spec asked for that
       are missing or partial, (b) behaviour in the diff the spec did not ask for
       (scope creep / drift), (c) criteria implemented incorrectly. Quote the spec
       line per finding."
     - **Code-review axis** → `devrites-code-reviewer`: "Apply your full documented
       discipline (tests-first, correctness, readability, architecture, maintainability,
       standards) on the active feature workspace + diff. Cite file:line per finding;
       skip what tooling already enforces."
   - **Do NOT merge or re-rank** their findings. Present them under separate
     `## Spec` and `## Code review` sub-sections in `review.md`. Surface contradictions
     between the axes explicitly (e.g. "Spec axis says complete, Code-review axis says
     untestable") — `/rite-seal` decides what blocks.
4. **Reconcile, don't re-review.** With the two parallel reports in hand, the inline
   lead reconciles — it does **not** re-run a five-axis pass over correctness /
   readability / architecture / maintainability that `devrites-code-reviewer` already
   covered. Stay in scope ([feature-scoped-review](reference/feature-scoped-review.md)).
   Add only what the dispatched agents could not, then resolve overlaps and
   contradictions before labeling. ([five-axis-review](reference/five-axis-review.md)
   documents the axes the code-review agent applies.)
   - **UI feature?** Apply the **UX rubric**
     ([nielsen-heuristics](reference/nielsen-heuristics.md)) and the
     **cognitive-load lens** ([cognitive-load](reference/cognitive-load.md)) on the
     UX axis — surface heuristics scoring ≤ 2 and any cognitive-load findings
     (extraneous noise, missing progressive disclosure, vocabulary drift) at the
     appropriate severity.
5. **Security** — apply `devrites-audit security`
   ([security-review](reference/security-review.md)) when user input, auth, data
   storage, external integrations, secrets, or permissions are involved.
6. **Performance** — apply `devrites-audit perf`
   ([performance-review](reference/performance-review.md)) only when performance is
   relevant or a regression risk is visible (measure first).
7. Apply only in-scope fixes; **run verification after changes** (`/rite-prove` logic).
8. Update `review.md`, `evidence.md`, and `state.md`.

## Finding labels
- **Critical** — must fix before seal (correctness/security/data loss).
- **Important** — should fix before seal (likely bug, real maintainability risk).
- **Suggestion** — worth doing, not blocking.
- **Nit** — trivial/style.
- **FYI** — context, no action implied.

## Severity orientation (labels, not score)

After labeling, summarize findings as `Critical / Important / Suggestion / Nit /
FYI` counts. There is no composite number — `/rite-seal` gates on
`Critical == 0` and on acceptance + drift. Inventing a number invites gaming;
the labels do the work.

> **Mid-flight discipline.** When tempted to demote a Critical, hide a finding, fix without re-verification, or wander out of scope — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output → `review.md`
```
Reviewed: <slice N | feature>
Tests: <adequate? gaps>

## Spec   (from devrites-spec-reviewer sub-agent, verbatim)
- <quoted spec line> — <missing / partial / wrong / scope-creep> — <where>
- ...

## Code review   (from devrites-code-reviewer sub-agent, verbatim)
- <file:line> — <violation / correctness / readability / architecture / maintainability> — <fix>
- ...

## Cross-axis contradictions (if any)
- Spec axis: ... | Code-review axis: ... | who decides at seal: ...

Findings (combined, after the parallel pass):
  [Critical]   <n> — <file:line — problem — fix>
  [Important]  <n> — <file:line — problem — fix>
  [Suggestion] <n> — <file:line — problem — fix>
  [Nit]        <n> — <file:line — problem — fix>
  [FYI]        <n> — <file:line — note>
Fixes applied (in scope): ...
Re-verification: <cmd → pass/fail>
Re-prove: <if review changed code, run a scoped `/rite-prove` so evidence post-dates the fix | n/a — no code changed>
Next: /rite-seal   (or fix Criticals first)
↻ Hygiene: /compact (review findings) if fixing Criticals/Importants this session; /clear if review is clean — see rules/context-hygiene.md.
```

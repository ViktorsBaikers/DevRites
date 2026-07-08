---
name: rite-review
description: Review the polished diff at feature scope across Spec + Code-review axes. Use when the user says "review this", "audit my diff", "final review before seal", "check this against the spec". Not for whole-project refactors or single-slice review.
argument-hint: "[scope: slice N | feature]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-review — feature-scoped review

Senior review of the **active feature scope only**. **Read the active workspace first**;
if none, tell the user to run `$rite-spec <feature>`.

> **Differs from built-in `/code-review` in:** `/code-review` is a generic
> diff review with no workspace context. `$rite-review` reads
> `.devrites/work/<slug>/spec.md` first, runs Spec ↔ Code-review axes as
> parallel subagents (see [`parallel-dispatch.md`](../devrites-lib/reference/parallel-dispatch.md)), and gates feeding
> into `$rite-seal`. Use `/code-review` for a one-off diff; use
> `$rite-review` for a DevRites feature where the spec is the contract.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.agents/skills/devrites-lib/reference/standards/core.md` first. The other rule files load on demand;
pull these via `Read` when the diff demands them:
- `code-review.md` — small PRs, severity labels, tests-first review focus.
- `principles.md` — declared project invariants (`.devrites/principles.md`); a diff that violates one with no recorded exception is a Critical, blocking finding.
- `testing.md` — confirm the tests prove the spec, not just pass.
- `agents.md` — when to fan out to which review subagent.
- `security.md` — when input / auth / data / integrations / secrets are in scope.
- `performance.md` — only when perf is relevant or a regression risk is visible.

## Operating rules
- **Feature scope only.** Review touched files + the diff. **NO whole-project refactors,
  NO drive-by cleanup.** DO NOT delete suspected dead code outside this feature without
  asking. Spec Drift Guard applies.
- **Reviews the finished product.** `$rite-polish` has already done **code simplification**
  + UI normalize/polish. Review judges; if it reveals a real complexity issue polish
  missed, flag it as a finding — don't re-run a simplification sweep here.
- Findings are labeled (below). Re-prove after any change you make.

## Workflow
0. Read `.agents/skills/devrites-lib/reference/standards/core.md` first (the always-on operating rules); pull the
   on-demand rules above as the diff demands them.
   Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   devrites-engine preamble
   ```
1. Read `spec.md`, `tasks.md`, `state.md`, `decisions.md`, `evidence.md`,
   `touched-files.md`, `.devrites/principles.md` (if present — the binding invariants to score
   the diff against), and the `git diff`. For "what would this change break"
   questions, prefer a code-intelligence index if available — codebase-memory-mcp first,
   cross-checked with codegraph + graphify, else standard methods (LSP / Read/Grep/Glob); see
   `.agents/skills/devrites-lib/reference/standards/tooling.md` — over file reads;
   they answer impact/callers in one call. When a finding hinges on an external library's
   current API, context7 if available can confirm the signature.
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
       skip what tooling already enforces. Also flag the AI-codegen smells (silent/empty
       catch, defensive try-catch bloat + redundant logging, single-use factory / needless
       indirection, dependency creep where an in-repo option exists, a 100-line function
       where 20 would do) and the silent-failure bugs (a missing value coerced to 0/''/[],
       a dropped Result/err return, off-by-one / boundary, logic that contradicts the
       comment/docstring/name). Per hunk, check whether working code was deleted that the
       task did not ask to remove. Score the diff against `.devrites/principles.md` — a change
       that breaks a declared invariant with no recorded, human-approved exception is a Critical."
   - **Do NOT merge or re-rank** their findings. Present them under separate
     `## Spec` and `## Code review` sub-sections in `review.md`. Surface contradictions
     between the axes explicitly (e.g. "Spec axis says complete, Code-review axis says
     untestable") — `$rite-seal` decides what blocks.
4. **Reconcile, don't re-review.** With the two parallel reports in hand, the inline
   lead reconciles — it does **not** re-run the code-review axes over correctness /
   readability / architecture / maintainability that `devrites-code-reviewer` already
   covered. Stay in scope ([feature-scoped-review](reference/feature-scoped-review.md)).
   Add only what the dispatched agents could not, then resolve overlaps and
   contradictions before labeling. ([five-axis-review.md](reference/five-axis-review.md)
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
7. Apply only in-scope fixes; **run verification after changes** (`$rite-prove` logic).
8. Update `review.md`, `evidence.md`, and `state.md`.
9. **Guard against the silent reviewer.** After `review.md` is written, run:
   ```bash
   devrites-engine review-integrity
   ```
   Exit 1 means an adversarial axis reported nothing and justified nothing — a suspected
   rubber-stamp. Re-run that axis or add its `No-findings:` justification; do not carry a silent
   axis into `$rite-seal` (where it surfaces as an Important).

## Finding labels
- **Critical** — must fix before seal (correctness/security/data loss).
- **Important** — should fix before seal (likely bug, real maintainability risk).
- **Suggestion** — worth doing, not blocking.
- **Nit** — trivial/style.
- **FYI** — context, no action implied.

**Action decoration (orthogonal to severity).** Also tag each finding with how to act on it:
`blocking` (fix before seal), `non-blocking` (fix when convenient), or `if-minor` (fix only if the
change is already small — a pure noise-economics lever). Only a **`blocking` Critical** gates the
seal; a `non-blocking` / `if-minor` finding is recorded, not a stop.

## Confidence + signal-to-noise
Borrow `$rite-vet`'s discipline so review stays **trusted, not noisy** — a reviewer that posts
18 comments per PR teaches the team to ignore every one (below ~10% false-positive rate devs
investigate each finding; past ~30% they label the tool noisy and skip it):
- **Confidence-band each finding** (1–10) and state the band. A low-confidence finding (≤4)
  you can't verify against the code is **suppressed** — roll it into one `Suppressed
  (low-confidence): n` line, never raised as Critical/Important.
- **Verify before you escalate.** Every Critical/Important quotes the spec line or cites the
  `file:line` that proves it — no unverified blockers.
- **Budget the noise.** Roll up trivia ("N style nits") into a single line; tooling already
  catches style. Review's job is correctness + spec fidelity, not a lint dump.
- **A silent axis is suspicious, not clean.** An adversarial axis that raises nothing must earn
  it: end its `## Spec` / `## Code review` section with a **`No-findings:`** line naming the
  passes it ran (missing/partial/incorrect for spec; edge cases, error paths, the riskiest
  decision, a changed behavior whose test may not cover it for code) and why each came back
  empty. This is the mirror of confidence-banding — banding kills the false positive, this catches
  the false negative ([`code-review.md` § Zero findings is suspicious](../devrites-lib/reference/standards/code-review.md)).

## Severity orientation (labels, not score)

After labeling, summarize findings as `Critical / Important / Suggestion / Nit /
FYI` counts. There is no composite number — `$rite-seal` gates on
`Critical == 0` and on acceptance + drift. Inventing a number invites gaming;
the labels do the work.

> **Mid-flight discipline.** When tempted to demote a Critical, hide a finding, fix without re-verification, or wander out of scope — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output → `review.md`

Write the detailed review to `review.md`. In chat, run `devrites-engine progress` first, then use
the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: review complete for <slice N | feature>.
Changed: review.md, evidence.md <updated|n/a>, state.md
Evidence: findings Critical <n> / Important <n> / Suggestion <n> / Nit <n> / FYI <n>; re-verification <cmd -> pass|n/a>
Open: <none | Critical blockers, Fix: <single command> | Important fixes | re-prove needed>
Next: $rite-seal
Record: .devrites/work/<slug>/review.md
↻ Hygiene: /compact (review findings) if fixing now; /clear if clean
```

---
name: rite-review
description: Review polished feature diff for correctness, readability, architecture, security, tests proving acceptance, Critical/Important findings, and quality dimensions before seal.
argument-hint: "[scope: slice N | feature] [--full]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- Inspect the current `spawn_agent` role list. When the named `devrites-<role>` is exposed, dispatch it with `fork_turns="none"`; full-history forks inherit the parent type. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If a named role is not exposed, use generic `explorer` for every read-only role with `fork_turns="none"`. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. Trusted `.codex/hooks.json` binds `agent_type=explorer` to the fail-closed reviewer read-only guard.
- For `devrites-slice-wright`, trusted `.codex/hooks.json` binds generic `worker` (`agent_type=worker`) to the active reconcile window and exact `.wright-allowlist`. Dispatch that worker with `fork_turns="none"`, tell it to read `.codex/agents/devrites-slice-wright.toml`, and execute the unchanged packet. Never create `.reconcile-inline` when this safe rung is available.
- A missing custom role is not evidence that spawning is unavailable. Only when the project hooks are unavailable or untrusted, no spawn primitive exists, or higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, create `.reconcile-inline` only for that path, and apply every fallback risk gate.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-review: feature-scoped review

Review the **active feature scope only**. **Read the active workspace first**; if none,
tell the user to run `$rite-spec <feature>`.

> **Scope:** `/code-review` is a generic diff review with no workspace context.
> `$rite-review` reads
> `.devrites/work/<slug>/spec.md` first, runs Spec ↔ Code-review axes as
> parallel fresh-context reviewers (see [`parallel-dispatch.md`](../devrites-lib/reference/parallel-dispatch.md)), and gates feeding
> into `$rite-seal`. Use `/code-review` for a one-off diff; use
> `$rite-review` for a DevRites feature where the spec is the contract.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull these via `Read` when the diff demands them:
- `code-review.md`: small PRs, severity labels, tests-first review focus.
- `review-checklist.md`: compact pass/fail sweep before reporting the verdict.
- `principles.md`: declared project invariants (`.devrites/principles.md`); a diff that violates one with no recorded exception is a Critical, blocking finding.
- `testing.md`: confirm that passing tests actually prove the spec.
- `agents.md`: when to fan out to which review subagent.
- `security.md`: when input / auth / data / integrations / secrets are in scope.
- `security-checklist.md`: for the same security-sensitive scope, the compact trust-boundary sweep.
- `performance.md`: only when perf is relevant or a regression risk is visible.

## Operating rules
- **Feature scope only.** Review touched files + the diff. **NO whole-project refactors,
  NO drive-by cleanup.** DO NOT delete suspected dead code outside this feature without
  asking. Spec Drift Guard applies.
- **Review the finished product.** `$rite-polish` has already simplified code and
  normalized or polished UI. If review finds a remaining complexity issue, record it as
  a finding rather than rerunning a simplification pass.
- Findings are labeled (below). Re-prove after any accepted correction.
- **Reviewers judge; root reconciles; wright fixes.** Follow
  [`agents.md`](../devrites-lib/reference/standards/agents.md). The root owns the verdict and
  canonical writes; every accepted source/test correction routes to
  `devrites-slice-wright`.

## Workflow
0. Read `.agents/skills/devrites-lib/reference/standards/core.md` first (the always-on operating rules); pull the
   on-demand rules above as the diff demands them.
   Then run `devrites-engine preamble` for deterministic workspace orientation.
1. Read `spec.md`, `tasks.md`, `state.md`, `decisions.md`, `evidence.md`,
   `touched-files.md`, `.devrites/principles.md` (if present: the binding invariants to score
   the diff against), and the `git diff`. For "what would this change break"
   questions, prefer a code-intelligence index if available (codebase-memory-mcp first,
   cross-checked with codegraph + graphify, else standard methods (LSP / Read/Grep/Glob); see
   `.agents/skills/devrites-lib/reference/standards/tooling.md`) over file reads;
   they answer impact/callers in one call. When a finding hinges on an external library's
   current API, context7 if available can confirm the signature.
2. **Review tests first:** do they prove the acceptance criteria? Missing,
   weak, or wrong tests are the first findings.
   **Completion:** every acceptance criterion maps to a proven test or a labeled finding.
3. **Review spec and code separately in parallel.** A change can pass
   one axis and fail the other: code that follows every project standard but
   implements the wrong thing (Code-review pass, Spec fail), or code that does exactly
   what the spec asked but breaks project conventions (Spec pass, Code-review fail).
   Separate contexts prevent one axis from masking the other:
   - Freeze the candidate and fresh-context dispatch **two** read-only reviewers in
     parallel, each with its own `agent-packet/v1`, narrow brief, and no
     cross-pollination:
     - **Spec axis** → `devrites-spec-reviewer`: "Apply your documented discipline on
       the active feature workspace + diff. Report (a) criteria the spec asked for that
       are missing or partial, (b) behaviour in the diff the spec did not ask for
       (scope creep / drift), (c) criteria implemented incorrectly. Quote the spec
       line per finding. If the spec is missing or unreadable, report `Review gap:
       missing spec` and do not infer requirements from the diff."
     - **Code-review axis** → `devrites-code-reviewer`: "Apply your full documented
       discipline (tests-first, correctness, readability, architecture, maintainability,
       standards) on the active feature workspace + diff. Cite file:line per finding;
       skip what tooling already enforces. Also flag the AI-codegen smells (silent/empty
       catch, defensive try-catch bloat + redundant logging, single-use factory / needless
       indirection, dependency creep where an in-repo option exists, a 100-line function
       where 20 would do) and the silent-failure bugs (a missing value coerced to 0/''/[],
       a dropped Result/err return, off-by-one / boundary, logic that contradicts the
       comment/docstring/name). Per hunk, check whether working code was deleted that the
       task did not ask to remove. Score the diff against `.devrites/principles.md`: a change
       that breaks a declared invariant with no recorded, human-approved exception is a Critical.
       Distinguish hard documented-standard violations from baseline smells; smells are judgment
       calls unless a DevRites or project standard makes them binding."
   - **Do NOT merge or re-rank** their findings. Present them under separate
     `## Spec` and `## Code review` sub-sections in `review.md`. Surface contradictions
     between the axes explicitly (e.g. "Spec axis says complete, Code-review axis says
     untestable"): `$rite-seal` decides what blocks.
4. **Reconcile, don't re-review.** With the two parallel reports in hand, the inline
   lead reconciles. It does **not** re-run the code-review axes over correctness /
   readability / architecture / maintainability that `devrites-code-reviewer` already
   covered. Stay in scope ([feature-scoped-review](reference/feature-scoped-review.md)).
   Add only what the dispatched agents could not, then resolve overlaps and
   contradictions before labeling. ([five-axis-review.md](reference/five-axis-review.md)
   documents the axes the code-review agent applies.)
   - **UI feature?** Apply the **UX rubric**
     ([nielsen-heuristics](reference/nielsen-heuristics.md)) and the
     **cognitive-load lens** ([cognitive-load](reference/cognitive-load.md)) on the
     UX axis: surface heuristics scoring ≤ 2 and any cognitive-load findings
     (extraneous noise, missing progressive disclosure, vocabulary drift) at the
     appropriate severity.
5. **Security:** apply `devrites-audit security`
   ([security-review](reference/security-review.md)) when user input, auth, data
   storage, external integrations, secrets, or permissions are involved.
6. **Performance:** apply `devrites-audit perf`
   ([performance-review](reference/performance-review.md)) only when performance is
   relevant or a regression risk is visible (measure first).
7. Reconcile and accept only in-scope fixes. Consolidate them into one bounded wright
   correction packet; never edit source in the reviewing context. Freeze the new candidate
   and **run affected verification after changes** (`$rite-prove` logic), then perform at
   most one narrow recheck of affected findings.
8. The root updates `review.md`, `evidence.md`, and `state.md`.
   **Completion:** the records name the reviewed candidate identity and every accepted
   correction has affected proof plus its narrow recheck.
9. **Require an account from every reviewer.** After `review.md` is written, run:
   ```bash
   devrites-engine review-integrity
   devrites-engine review-fingerprints --write
   ```
   Exit 1 means an adversarial axis reported neither findings nor a justification. Re-run
   that axis or add its `No-findings:` justification; do not carry a silent
   axis into `$rite-seal` (where it surfaces as an Important).

## Finding labels
- **Critical:** must fix before seal (correctness/security/data loss).
- **Important:** should fix before seal (likely bug, real maintainability risk).
- **Suggestion:** worth doing, not blocking.
- **Nit:** trivial/style.
- **FYI:** context, no action implied.

**Action tag (separate from severity).** Tag each finding with how to act on it:
`blocking` (fix before seal), `non-blocking` (fix when convenient), or `if-minor` (fix only if the
change is already small: a pure noise-economics lever). Only a **`blocking` Critical** gates the
seal; a `non-blocking` / `if-minor` finding is recorded, not a stop.

## Confidence and signal-to-noise
Apply `$rite-vet`'s confidence rules. A reviewer that posts
18 comments per PR teaches the team to ignore every one (below ~10% false-positive rate devs
investigate each finding; past ~30% they label the tool noisy and skip it):
- **Confidence-band each finding** (1-10) and state the band. A low-confidence finding (≤4)
  you can't verify against the code is **suppressed**: roll it into one `Suppressed
  (low-confidence): n` line, never raised as Critical/Important.
- **Verify before you escalate.** Every Critical/Important quotes the spec line or cites the
  `file:line` that proves it: no unverified blockers.
- **Limit low-value comments.** Roll up trivia ("N style nits") into a single line; tooling already
  catches style. Review's job is correctness + spec fidelity, not a lint dump.
- **A silent axis is suspicious, not clean.** An adversarial axis that raises nothing must earn
  it: end its `## Spec` / `## Code review` section with a **`No-findings:`** line naming the
  passes it ran (missing/partial/incorrect for spec; edge cases, error paths, the riskiest
  decision, a changed behavior whose test may not cover it for code) and why each came back
  empty. Confidence bands suppress false positives; this requirement catches silent false
  negatives ([`code-review.md` § Zero findings is suspicious](../devrites-lib/reference/standards/code-review.md)).

## Severity orientation (labels, not score)

After labeling, summarize findings as `Critical / Important / Suggestion / Nit /
FYI` counts. There is no composite number. `$rite-seal` gates on
`Critical == 0` and on acceptance plus drift. Do not invent a composite score.

> **Mid-flight discipline.** When tempted to demote a Critical, hide a finding, fix without re-verification, or wander out of scope: see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output → `review.md`

Write the detailed review to `review.md`. In chat, run `devrites-engine progress` first, then use
the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: review complete for <slice N | feature>.
Changed: review.md, review-fingerprints.jsonl, evidence.md <updated|n/a>, state.md
Evidence: open findings Critical 0 / Important <non-blocking n> / Suggestion <n> / Nit <n> / FYI <n>; re-verification <cmd -> pass|n/a>
Open: <none | non-blocking Important/Suggestion follow-ups>
Next: $rite-seal
Record: .devrites/work/<slug>/review.md
↻ Hygiene: /compact (review findings) if fixing now; /clear if clean
```
If a blocking Critical or required re-proof remains, use the shared `Stopped /
blocked` form and route `Fix:` to the single repair or `$rite-prove`; do not
recommend `$rite-seal`.

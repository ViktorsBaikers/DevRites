# rite-seal phase contract

## Execution spine

Run the engine gates at these moments:

```bash
devrites-engine preamble

devrites-engine spec-validate .devrites/work/<slug>; echo "spec-validate rc=$?"
devrites-engine check-acceptance .devrites/work/<slug>; echo "check-acceptance rc=$?"
devrites-engine evidence-fresh <slug>; echo "evidence-fresh rc=$?"
devrites-engine doubt-coverage <slug>; echo "doubt-coverage rc=$?"

devrites-engine footprint log <slug> reviewer devrites-<x>-reviewer
devrites-engine footprint log <slug> skip devrites-<x>-reviewer
devrites-engine footprint roster <slug>; echo "roster rc=$?"
devrites-engine review-integrity <slug>; echo "review-integrity rc=$?"
devrites-engine review-fingerprints --write <slug>

devrites-engine footprint render <slug>
devrites-engine learnings add "<slug>" "<dismissed-as-intentional class or dead-end>" dismiss
devrites-engine health record <0..10> "<GO|NO-GO evidence summary>" --note "<top blocker or green proof>"
devrites-engine timeline log completed --skill rite-seal --slug <slug> --outcome "<go|no-go>" --decision "<verdict>"

devrites-engine progress
```

Conditional calls still obey the detailed rules below: reviewer and skip
footprints must account for the whole roster; learnings run only on GO; progress
is rendered before the final response from [`output.md`](output.md). Seal decides
only; `/rite-ship` owns irreversible git/publish/deploy actions.

1. **Run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   devrites-engine preamble
   ```
   Then read all artifacts: `brief.md`, `spec.md`, `plan.md`, `tasks.md`, `state.md`,
   `decisions.md`, `assumptions.md`, `questions.md`, `drift.md`, `evidence.md`,
   `browser-evidence.md`, `polish-report.md`, `review.md`, `design-brief.md` (if UI),
   `devex.md` (if a developer-facing surface), `strategy.md` (if present), and the **final diff**. If a code-intelligence index is available
   (see `.claude/skills/devrites-lib/reference/standards/tooling.md`), use it for
   blast-radius checks on the final diff in step 5; context7 if available can confirm a current
   external-API signature a reviewer flags.
2. Check **acceptance criteria one by one** — first run `devrites-engine spec-validate .devrites/work/<slug>`;
   unresolved or unproven `resolved/test` prohibitions are a NO-GO until the linked evidence exists.
   Then use [final-evidence](final-evidence.md).
   Each gets a checkbox + the evidence that proves it (or "unproven"). Verify each criterion
   **independently against the evidence artifact** — the slice report or the build narrative is not
   proof; the `devrites-spec-reviewer` + `devrites-test-analyst` fan-out in step 7 is the
   independent cross-check (a verifier that never saw the optimistic narrative).
3. Verify tests, build/typecheck/lint, and browser proof are present and green for the
   scope. Re-run if cheap and in doubt. **For a UI feature with a `design-brief.md`**, read the
   `## Visual Verdict` table in `browser-evidence.md`: a `FAIL` on an **acceptance-mapped**
   criterion is an unmet acceptance criterion (NO-GO, per the severity gate), a declared-state
   `FAIL` is **Important**, and a UI build whose brief exists but whose verdict is **absent** is an
   Important evidence gap (the scorecard should have been emitted at browser-proof).
4. Check unresolved **questions** and **drift** — any open item that changes product
   behavior blocks. **Any `questions.md` entry with `gate: validating` and `status: open`
   is a NO-GO regardless of behavior impact** (an open validating gate is merge-blocking by
   definition); a slice marked `built (pending review)` is not done.
4a. **Doubt coverage — every stood decision was independently doubted.** Run the deterministic
   check, then judge it against `decisions.md`:
   ```bash
   devrites-engine doubt-coverage <slug>; echo "doubt-coverage rc=$?"
   ```
   The script reads two records `/rite-build` writes — the `## Decisions stood` ledger in
   `decisions.md` (each entry ends `— doubt: <accept | reject-resolved | MISSING>`) and the `doubt`
   footprint. Read the exit code:
   - **rc=3** — the ledger records a stood decision with `doubt: MISSING`: doubt was definitively
     skipped for a decision on record. **Important**; **NO-GO** when that decision is irreversible-risk
     (auth / public-API / migration). This is the per-slice skip the footprint count alone can't catch.
   - **rc=1** — zero doubt dispatched across the build. A **prompt to verify, not a finding**: confirm
     against the ledger — an every-slice-trivial feature (`- none`) is a valid pass; a stood triggering
     decision (boundary / data-model / auth / public-API / migration / branching) with no verdict is
     the finding, same severity as rc=3.
   - **rc=0** — covered, or not assessable (an inline build logs no dispatch — verify the ledger's
     verdicts by hand).
   Either way, walk the `## Decisions stood` ledger yourself: severity rides the unverified
   **decision**, never the exit code alone.
5. Check **security, data, migration, rollback** risk —
   [risk-and-rollback](risk-and-rollback.md). If `strategy.md` exists (from
   `/rite-temper`), confirm its **top pre-mortem risks are mitigated** in the diff/evidence and
   that no **Non-goal / deferred item crept into the diff** (scope creep) — either is a finding
   (an unmitigated top risk or smuggled-in out-of-scope work).
   - **Principles** (`principles.md`): score the final diff against each declared invariant in
     `.devrites/principles.md`. A violation with no matching, human-approved exception in the
     register is a **Critical** finding and a NO-GO; an exception that is stale (past its review
     trigger) or wider than its stated scope is itself a finding. No file / no principles → skip.
   - **Observability** (`observability.md`): if the diff added a runtime surface (endpoint,
     job, integration, user flow, error path), a feature shipping with no way to debug it in
     prod is an **Important** finding, not a pass — `evidence.md` should show telemetry observed
     to emit (`/rite-prove` step 5b).
   - **Developer experience** (`developer-experience.md`): if the diff ships a developer-facing
     surface, reconcile `devex.md` (the `/rite-vet` predicted scorecard vs the `/rite-prove`
     measured one — the boomerang). A broken public dev contract (a documented command that errors,
     a getting-started flow that can't complete) or an unexplained measured DX regression is
     **Important** — **Critical** on a frozen public surface (`principles.md`). No surface → skip.
   - **Removal / migration** (`deprecation.md`): if the diff deletes or migrates code, an API,
     or data, confirm it followed expand→contract, proved the old path unused before removing it,
     and carries a rollback for every destructive step. A surprise deletion or a one-shot
     breaking migration is a finding (and trips the irreversible-risk gate, `afk-hitl.md`).
6. Check **frontend polish** if UI is involved (states, a11y, responsive, design-system,
   browser evidence).
7. **Independent review** — seal is the final gate, not a re-run of `/rite-review`.
   It **always re-spawns** the axes `/rite-review` did not cover: `devrites-test-analyst`,
   `devrites-security-auditor`, `devrites-performance-reviewer`, and
   `devrites-frontend-reviewer` (UI). It **only re-runs the Spec and Code-review axes**
   (`devrites-spec-reviewer`, `devrites-code-reviewer`) when the diff changed since
   `/rite-review` ran (compare against `review.md`); if the diff is unchanged, carry
   review's verdicts on those axes forward rather than re-litigating them.
   If subagents are available, fan out **in parallel** (one `Task` block, multiple tool
   calls) to the **roster** — the seven reviewers and their checkable triggers are the single
   source in [`parallel-dispatch.md`](../../devrites-lib/reference/parallel-dispatch.md) (dispatch
   shape + reconciliation there too; `.claude/skills/devrites-lib/reference/standards/agents.md` — "Run independent reviewers in
   parallel"). Dispatch every always-on reviewer and every conditional whose trigger the diff
   meets; `devrites-devex-reviewer` runs in **measure mode** — grade the measured DX scorecard
   and reconcile the boomerang against the `/rite-vet` prediction. Give each the workspace path +
   diff *without the author's reasoning*. If subagents are unavailable, run the equivalent
   reviews sequentially yourself and flag each as a fallback.
   The reviewer **AGENTS** here (fresh context, no author reasoning) are the seal
   GATE; `devrites-audit` is the inline single-axis pass run during build/polish.
   The two paths are intentional, not divergent — the inline audit catches issues
   early; the fresh-context agents are the independent gate before ship.
   **Footprint — account for the whole roster.** For each reviewer you dispatch, append
   `devrites-engine footprint log <slug> reviewer devrites-<x>-reviewer` (the reviewer's **exact agent name** —
   the roster gate matches on it, so a freehand label like `spec` will read as unaccounted); for
   each roster reviewer you consciously do **not** dispatch, append
   `devrites-engine footprint log <slug> skip devrites-<x>-reviewer` and note the one-line reason in `seal.md`.
   A conditional reviewer that does not apply is a *recorded skip*, never a silent omission — step 7b
   proves the roster complete before the verdict.
7a. **Reconcile findings — confidence over volume.** Band each reviewer finding by confidence
   (1–10); a low-confidence (≤4) finding that can't be verified against the diff is **suppressed**
   to a `Suppressed (low-confidence): n` line, not a blocker. Every Critical/Important must cite
   the `file:line` (or spec line) that proves it. Surface genuine cross-reviewer disagreement
   **explicitly** rather than averaging it away, and don't let a pile of low-confidence nits
   inflate the verdict — the gate is `Critical == 0` + acceptance + drift, not "few findings".
   (A seal that fires noise teaches the next one to be ignored.)
7b. **Account for the roster — no reviewer silently skipped.** Before the verdict, prove every
   roster reviewer carries a decision (dispatched or skip-recorded in step 7's footprint):
   ```bash
   devrites-engine footprint roster <slug>; echo "roster rc=$?"
   ```
   - **rc=3** — a roster reviewer was neither dispatched nor skip-recorded: the silent omission
     this gate exists to catch. **NO-GO** on the review's *completeness* (not the diff) — dispatch
     the missing reviewer or record a justified skip, then re-run. Same standing as an unproven
     criterion (severity gate).
   - **rc=1** — an always-on axis (Spec / Code-review) was skip-recorded: legitimate **only** as a
     carry-forward of `/rite-review`'s verdict on an **unchanged** diff (step 7). Confirm that;
     otherwise dispatch it.
   - **rc=0** — every reviewer accounted for; proceed.
7c. **No silent reviewer — a zero-findings axis is justified, not assumed.** Confirm no adversarial
   axis rubber-stamped: an axis that reported nothing must carry a `No-findings:` justification.
   ```bash
   devrites-engine review-integrity <slug>; echo "review-integrity rc=$?"
   ```
   - **rc=1** — an axis in `review.md` is silent (no finding, no justification): **Important** on the
     review's completeness. Re-run that axis or record its `No-findings:` account, then re-seal.
   - **rc=0** — every axis has findings or a justified clean bill; proceed. (No `review.md`, or a
     freeform one, is a clean pass — the gate keys on per-axis sections.)
   Then write stable finding IDs for learning and recurring-dismissal correlation:
   ```bash
   devrites-engine review-fingerprints --write <slug>
   ```
8. Decide GO / NO-GO — [go-no-go](go-no-go.md) — and write `seal.md`. Then render the
   **fan-out footprint** into `seal.md` and the output — deterministic run-weight (subagents
   dispatched · slices · wall-clock), **never a token or dollar figure** (DevRites can't
   truthfully source one):
   ```bash
   devrites-engine footprint render <slug>
   ```
   Also emit a machine-readable verdict block into `seal.md` (so `/rite-autocomplete` can gate
   without parsing prose):
   ```json
   { "feature": "<slug>", "verdict": "GO | NO-GO | CONDITIONAL-GO", "criticals": 0, "important": 0,
     "acceptance": "<proven>/<total>", "test_integrity": "ok | weakened", "mutation": "<score | n/a>",
     "blockers": ["<one line each, empty on GO>"] }
   ```
9. **On GO only — record proven conventions** to the local ledger
   (`.devrites/conventions.md`) so later slices stop re-deriving this project's idioms:
   the durable, *evidence-proven* commands / idioms / placement / gotchas this feature
   established. Evidence-gated like the seal itself; the band is earned, not guessed; the
   step degrades gracefully when unavailable. Full contract + command:
   [conventions-ledger](conventions-ledger.md). (Skip entirely on NO-GO.)
9a. **On GO only — auto-capture learnings** (`.devrites/learnings.md`). Learning is automatic, not a
   command anyone must remember: append this feature's durable signal so the **next** feature's
   review starts warmer — (a) any reviewer finding the gate **dismissed as intentional here** (a
   "don't re-flag X in this project" class, tag `dismiss`); (b) a recurring correction or dead-end
   worth not repeating (tag `note`). The review skills load this ledger **before** they fan out, so a
   dismissed-finding class stops recurring. It is an untrusted prior — live code always overrides
   (`.claude/skills/devrites-lib/reference/standards/security.md`). Promotion of a recurring lesson to a *project rule* stays the
   human's call (`/rite-learn` — propose, don't impose). Skip on NO-GO.
   ```bash
   devrites-engine learnings add "<slug>" "<dismissed-as-intentional class or dead-end>" dismiss
   ```
10. **Record the durable phase traces.** Before the final output, record the health signal and
    verdict event. The score is evidence-backed, not decorative: high for GO with green proof,
    lower for NO-GO or unresolved blockers.
    ```bash
    devrites-engine health record <0..10> "<GO|NO-GO evidence summary>" --note "<top blocker or green proof>"
    devrites-engine timeline log completed --skill rite-seal --slug <slug> --outcome "<go|no-go>" --decision "<verdict>"
    ```

## `seal.md` template

Loaded on demand from [`seal-template.md`](seal-template.md). Fill in
each section + write to `.devrites/work/<slug>/seal.md` as the durable record.

> **Mid-flight discipline.** When tempted to round NO-GO up to GO, bypass the y/N prompt, average reviewer disagreements, or seal with unresolved drift — see [`anti-patterns`](anti-patterns.md). Load it the moment you reach for the excuse.

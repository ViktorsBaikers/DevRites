# rite-seal phase contract

1. **Run the shared orientation preamble:** it prints `state.md`, the artifacts present,
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
2. Check **acceptance criteria one by one**: first run `devrites-engine spec-validate .devrites/work/<slug>`;
   unresolved or unproven `resolved/test` prohibitions are a NO-GO until the linked evidence exists.
   Then use [final-evidence](final-evidence.md).
   Each gets a checkbox + the evidence that proves it (or "unproven"). Verify each criterion
   **independently against the evidence artifact**: the slice report or the build narrative is not
   proof; the `devrites-spec-reviewer` + `devrites-test-analyst` fan-out in step 7 is the
   independent cross-check (a verifier that never saw the optimistic narrative).
3. Verify tests, build/typecheck/lint, and browser proof are present and green for the
   scope. Re-run if cheap and in doubt. **For a UI feature with a `design-brief.md`**, read the
   `## Visual Verdict` table in `browser-evidence.md`: a `FAIL` on an **acceptance-mapped**
   criterion is an unmet acceptance criterion (NO-GO, per the severity gate), a declared-state
   `FAIL` is **Important**, and a UI build whose brief exists but whose verdict is **absent** is an
   Important evidence gap (the scorecard should have been emitted at browser-proof).
4. Check unresolved **questions** and **drift**: any open item that changes product
   behavior blocks. **Any `questions.md` entry with `gate: validating` and `status: open`
   is a NO-GO regardless of behavior impact** (an open validating gate is merge-blocking by
   definition); a slice marked `built (pending review)` is not done.
4a. **Doubt coverage: every stood decision was independently doubted.** Run the deterministic
   check, then judge it against `decisions.md`:
   ```bash
   devrites-engine doubt-coverage <slug>; echo "doubt-coverage rc=$?"
   ```
   The script reads two records `/rite-build` writes: the `## Decisions stood` ledger in
   `decisions.md` (each entry ends `— doubt: <accept | reject-resolved | MISSING>`) and the `doubt`
   footprint. Read the exit code:
   - **rc=3:** the ledger records a stood decision with `doubt: MISSING`: doubt was definitively
     skipped for a decision on record. **Important**; **NO-GO** when that decision is irreversible-risk
     (auth / public-API / migration). This is the per-slice skip the footprint count alone can't catch.
   - **rc=1:** zero doubt dispatched across the build. A **prompt to verify, not a finding**: confirm
     against the ledger: an every-slice-trivial feature (`- none`) is a valid pass; a stood triggering
     decision (boundary / data-model / auth / public-API / migration / branching) with no verdict is
     the finding, same severity as rc=3.
   - **rc=0:** covered, or not assessable because no wright was logged.
   Either way, walk the `## Decisions stood` ledger yourself: severity rides the unverified
   **decision**, never the exit code alone.
5. Check **security, data, migration, rollback**, strategy scope, principles,
   observability, developer experience, and removal using
   [risk-and-rollback](risk-and-rollback.md) and the named standards. Unmitigated top risks,
   scope creep, missing runtime diagnostics, broken developer contracts, or unsafe removals
   are findings. An unexcepted declared-principle violation is Critical and NO-GO.
6. Check **frontend polish** if UI is involved (states, a11y, responsive, design-system,
   browser evidence).
7. **Independent review:** apply the complete roster, triggers, dispatch shape, and
   reconciliation from
   [`parallel-dispatch.md`](../../devrites-lib/reference/parallel-dispatch.md). Await
   fresh-context batches of at most three and give reviewers only the workspace path plus
   immutable diff, without author reasoning. Use named role → safely enforced generic fresh
   agent; if neither is available, seal remains NO-GO. Carry Spec/Code verdicts only
   for an unchanged post-review diff; rerun the other applicable axes.
   **Footprint: account for the whole roster.** For each reviewer you dispatch, append
   `devrites-engine footprint log <slug> reviewer devrites-<x>-reviewer` (the reviewer's **exact agent name**:
   the roster gate matches on it, so a freehand label like `spec` will read as unaccounted); for
   each roster reviewer you consciously do **not** dispatch, append
   `devrites-engine footprint log <slug> skip devrites-<x>-reviewer` and note the one-line reason in `seal.md`.
   A conditional reviewer that does not apply is a *recorded skip*, never a silent omission: step 7b
   proves the roster complete before the verdict.
7a. **Reconcile findings by confidence.** Band each reviewer finding by confidence
   (1-10); a low-confidence (≤4) finding that can't be verified against the diff is **suppressed**
   to a `Suppressed (low-confidence): n` line, not a blocker. Every Critical/Important must cite
   the `file:line` (or spec line) that proves it. Surface genuine cross-reviewer disagreement
   **explicitly** rather than averaging it away, and don't let a pile of low-confidence nits
   inflate the verdict. The gate is `Critical == 0` + acceptance + drift, not "few findings".
7b. **Account for the roster: no reviewer silently skipped.** Before the verdict, prove every
   roster reviewer carries a decision (dispatched or skip-recorded in step 7's footprint):
   ```bash
   devrites-engine footprint roster <slug>; echo "roster rc=$?"
   ```
   - **rc=3:** a roster reviewer was neither dispatched nor skip-recorded: the silent omission
     this gate exists to catch. **NO-GO** on the review's *completeness* (not the diff): dispatch
     the missing reviewer or record a justified skip, then re-run. Same standing as an unproven
     criterion (severity gate).
   - **rc=1:** an always-on axis (Spec / Code-review) was skip-recorded: legitimate **only** as a
     carry-forward of `/rite-review`'s verdict on an **unchanged** diff (step 7). Confirm that;
     otherwise dispatch it.
   - **rc=0:** every reviewer accounted for; proceed.
7c. **No silent reviewer: a zero-findings axis is justified, not assumed.** Confirm no adversarial
   axis rubber-stamped: an axis that reported nothing must carry a `No-findings:` justification.
   ```bash
   devrites-engine review-integrity <slug>; echo "review-integrity rc=$?"
   ```
   - **rc=1:** an axis in `review.md` is silent (no finding, no justification): **Important** on the
     review's completeness. Re-run that axis or record its `No-findings:` account, then re-seal.
   - **rc=0:** every axis has findings or a justified no-findings result; proceed. (No `review.md`, or a
     freeform one, is a clean pass: the gate keys on per-axis sections.)
   Then write stable finding IDs for learning and recurring-dismissal correlation:
   ```bash
   devrites-engine review-fingerprints --write <slug>
   ```
8. Decide GO / NO-GO: [go-no-go](go-no-go.md), and write `seal.md`. Then render the
   **fan-out footprint** into `seal.md` and the output: deterministic run-weight (subagents
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
9. **On GO only:** record evidence-proven conventions through
   [conventions-ledger](conventions-ledger.md).
9a. **On GO only:** append durable dismissed-finding or dead-end learnings as untrusted
   priors. Promotion to a project rule remains human-owned.
   ```bash
   devrites-engine learnings add "<slug>" "<dismissed-as-intentional class or dead-end>" dismiss
   ```
10. **Record the durable phase traces.** Before the final output, record the health signal and
    verdict event. The score is evidence-backed, not decorative: high for GO with green proof,
    lower for NO-GO or unresolved blockers.
    ```bash
    devrites-engine health record <0..10> "<GO|NO-GO evidence summary>" --note "<top blocker or green proof>"
    devrites-engine timeline log run-finished \
      --slug <slug> \
      --outcome "<passed|blocked>" \
      --execution-mode named \
      --guard-strength n/a \
      --reason-id "<DRV-GATE-SEAL-PASSED on GO; otherwise the blocking DRV-* reason>" \
      --host "<claude|codex>" \
      --evidence .devrites/work/<slug>/seal.md
    ```

## `seal.md` template

Loaded on demand from [`seal-template.md`](seal-template.md). Fill in
each section + write to `.devrites/work/<slug>/seal.md` as the durable record.

> **Mid-flight discipline.** When tempted to round NO-GO up to GO, bypass the y/N prompt, average reviewer disagreements, or seal with unresolved drift: see [`anti-patterns`](anti-patterns.md). Load it the moment you reach for the excuse.

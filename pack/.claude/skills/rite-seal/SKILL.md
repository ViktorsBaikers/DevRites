---
name: rite-seal
description: Final GO / NO-GO decision gate — walk `spec.md` acceptance against `evidence.md`, fan out reviewers in parallel, block on Critical, ask y/N on Important, write the verdict to seal.md. Use when the user says "seal this", "GO / NO-GO", "is it safe to merge", "final gate", "decide if we can ship". Hands off to /rite-ship for the actual commit/push/close. Not for the irreversible ship itself (use /rite-ship), inline review, or unpolished features.
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-seal — GO / NO-GO

The decision gate before shipping. **Read the active workspace first**; if none, tell
the user to run `/rite-spec <feature>`. Produces `seal.md` with a clear verdict.
`/rite-seal` **decides**; the irreversible git commit/push/tag and the task close-out
live in `/rite-ship`, which refuses to run without a GO recorded here.

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. The other rule files load on demand —
pull these via `Read` before sealing:
- `agents.md` — review-subagent fan-out at seal.
- `code-review.md` — severity labels (Critical / Important / Suggestion / Nit / FYI).
- `principles.md` — declared project invariants (`.devrites/principles.md`) are a pass/fail gate; a diff that violates one with no recorded, human-approved exception is a NO-GO.
- `documentation.md` — record decisions in `decisions.md` before sealing.
- `observability.md` — a runtime surface that ships blind is an Important finding.
- `deprecation.md` — when the diff removes / migrates code, API, or data (read with the
  risk-and-rollback step below).

## Operating rules
- Evidence over confidence — a criterion is met only if evidence proves it.
- Spec Drift Guard applies: unresolved drift is a NO-GO.
- **Honest verdict.** DO NOT round a NO-GO up to GO to be agreeable.

## Severity gate (the ship/no-ship rule)

Read `review.md` and the latest reviewer outputs.

| State | Gate result |
|---|---|
| `Critical == 0` and `Important == 0` and acceptance proven and drift resolved | **GO** (proceed) |
| `Critical == 0` and `Important > 0` and acceptance proven and drift resolved | Render interactive prompt: *"`Important > 0` open. Proceed to seal? [y/N]"*. Default **N**. If the user types `y`, GO; otherwise NO-GO with the open Important findings listed as blockers-by-policy. |
| `Critical > 0` | **NO-GO**, no exceptions. List every Critical with `file:line` and fix direction. |
| Any acceptance criterion unproven | **NO-GO**, list the unproven criteria. |
| Diff violates a declared project principle (`.devrites/principles.md`) with no recorded, human-approved exception | **NO-GO**, list each violated principle with `file:line`. Same standing as an unproven criterion (absent / empty file → none declared → not a blocker). |
| Unresolved drift in `drift.md` | **NO-GO**, route through `/rite-plan` first. |
| Any `questions.md` entry with `gate: validating` and `status: open` | **NO-GO** regardless of behavior impact — an open validating gate is merge-blocking by definition. A slice marked `built (pending review)` is not done. |

## Workflow
1. **Run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
   Then read all artifacts: `brief.md`, `spec.md`, `plan.md`, `tasks.md`, `state.md`,
   `decisions.md`, `assumptions.md`, `questions.md`, `drift.md`, `evidence.md`,
   `browser-evidence.md`, `polish-report.md`, `review.md`, `design-brief.md` (if UI),
   `devex.md` (if a developer-facing surface), `strategy.md` (if present), and the **final diff**. If a code-intelligence index is available
   (codebase-memory-mcp first, cross-checked with codegraph + graphify, else standard methods LSP / Read/Grep/Glob — see `.claude/rules/tooling.md`), use it for
   blast-radius checks on the final diff in step 5; context7 if available can confirm a current
   external-API signature a reviewer flags.
2. Check **acceptance criteria one by one** — [final-evidence](reference/final-evidence.md).
   Each gets a checkbox + the evidence that proves it (or "unproven"). Verify each criterion
   **independently against the evidence artifact** — the slice report or the build narrative is not
   proof; the `devrites-spec-reviewer` + `devrites-test-analyst` fan-out in step 7 is the
   independent cross-check (a verifier that never saw the optimistic narrative).
3. Verify tests, build/typecheck/lint, and browser proof are present and green for the
   scope. Re-run if cheap and in doubt.
4. Check unresolved **questions** and **drift** — any open item that changes product
   behavior blocks. **Any `questions.md` entry with `gate: validating` and `status: open`
   is a NO-GO regardless of behavior impact** (an open validating gate is merge-blocking by
   definition); a slice marked `built (pending review)` is not done.
5. Check **security, data, migration, rollback** risk —
   [risk-and-rollback](reference/risk-and-rollback.md). If `strategy.md` exists (from
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
   `devrites-frontend-reviewer` (UI). It **only re-runs the Spec and Code axes**
   (`devrites-spec-reviewer`, `devrites-code-reviewer`) when the diff changed since
   `/rite-review` ran (compare against `review.md`); if the diff is unchanged, carry
   review's verdicts on those axes forward rather than re-litigating them.
   If subagents are available, fan out to the chosen DevRites
   reviewers (`.claude/agents/devrites-*`) **in parallel** (one `Task` block,
   multiple tool calls; see `.claude/rules/agents.md` — "Run independent
   reviewers in parallel", and [`reference/parallel-dispatch.md`](reference/parallel-dispatch.md)
   for the full dispatch shape + reconciliation procedure):
   `devrites-spec-reviewer` (does the diff implement
   the spec?), `devrites-test-analyst` (do the tests prove acceptance?),
   `devrites-code-reviewer`, `devrites-frontend-reviewer` (UI features),
   `devrites-security-auditor` (input/auth/data/integrations),
   `devrites-performance-reviewer` (perf-relevant), and `devrites-devex-reviewer`
   (developer-facing surface — measure mode: grade the measured DX scorecard and reconcile
   the boomerang against the `/rite-vet` prediction). Give each the workspace
   path + diff *without the author's reasoning*. If subagents are unavailable,
   run the equivalent reviews sequentially yourself.
   The reviewer **AGENTS** here (fresh context, no author reasoning) are the seal
   GATE; `devrites-audit` is the inline single-axis pass run during build/polish.
   The two paths are intentional, not divergent — the inline audit catches issues
   early; the fresh-context agents are the independent gate before ship.
   **Footprint:** for each reviewer you dispatch here, append a record —
   `bash "$FP" log <slug> reviewer <name>` (resolve `$FP` to
   `.claude/skills/devrites-lib/scripts/footprint.sh` as in `/rite-build`) — so the seal can
   report the run's fan-out.
7a. **Reconcile findings — confidence over volume.** Band each reviewer finding by confidence
   (1–10); a low-confidence (≤4) finding that can't be verified against the diff is **suppressed**
   to a `Suppressed (low-confidence): n` line, not a blocker. Every Critical/Important must cite
   the `file:line` (or spec line) that proves it. Surface genuine cross-reviewer disagreement
   **explicitly** rather than averaging it away, and don't let a pile of low-confidence nits
   inflate the verdict — the gate is `Critical == 0` + acceptance + drift, not "few findings".
   (A seal that fires noise teaches the next one to be ignored.)
8. Decide GO / NO-GO — [go-no-go](reference/go-no-go.md) — and write `seal.md`. Then render the
   **fan-out footprint** into `seal.md` and the output — deterministic run-weight (subagents
   dispatched · slices · wall-clock), **never a token or dollar figure** (DevRites can't
   truthfully source one):
   ```bash
   FP=.claude/skills/devrites-lib/scripts/footprint.sh
   [ -f "$FP" ] || FP="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/footprint.sh"
   [ -f "$FP" ] || FP=pack/.claude/skills/devrites-lib/scripts/footprint.sh
   [ -f "$FP" ] && bash "$FP" render <slug> || true
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
   [conventions-ledger](reference/conventions-ledger.md). (Skip entirely on NO-GO.)
9a. **On GO only — auto-capture learnings** (`.devrites/learnings.md`). Learning is automatic, not a
   command anyone must remember: append this feature's durable signal so the **next** feature's
   review starts warmer — (a) any reviewer finding the gate **dismissed as intentional here** (a
   "don't re-flag X in this project" class, tag `dismiss`); (b) a recurring correction or dead-end
   worth not repeating (tag `note`). The review skills load this ledger **before** they fan out, so a
   dismissed-finding class stops recurring. It is an untrusted prior — live code always overrides
   (`.claude/rules/security.md`). Promotion of a recurring lesson to a *project rule* stays the
   human's call (`/rite-learn` — propose, don't impose). Skip on NO-GO.
   ```bash
   L=.claude/skills/devrites-lib/scripts/learnings.sh
   [ -f "$L" ] || L="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/learnings.sh"
   [ -f "$L" ] || L=pack/.claude/skills/devrites-lib/scripts/learnings.sh
   [ -f "$L" ] && bash "$L" add "<slug>" "<dismissed-as-intentional class or dead-end>" dismiss || true
   ```

## `seal.md` template

Loaded on demand from [`reference/seal-template.md`](reference/seal-template.md). Fill in
each section + write to `.devrites/work/<slug>/seal.md` as the durable record.

> **Mid-flight discipline.** When tempted to round NO-GO up to GO, bypass the y/N prompt, average reviewer disagreements, or seal with unresolved drift — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## On GO → hand off to /rite-ship

`/rite-seal` makes the **decision** and stops. It does **not** run `git commit`,
`git push`, `git tag`, publish, or deploy — those moved to `/rite-ship`, which renders
the type-GO prompt and refuses to run without a GO recorded here. Keeping the decision
and the irreversible action as two separately-auditable steps is the point: a GO seal
is a verdict, not an authorization to push.

On **GO**: write `seal.md`, set `state.md` `Next step: /rite-ship`, and tell the user
the feature is cleared to ship. Do **not** set phase `done` — `/rite-ship` marks done
after the task is shipped and archived. The `Important > 0` interactive y/N earlier in
the gate is the one off-ramp seal still owns; the type-GO off-ramp now lives in ship.
For a **UI feature**, note in the hand-off that `/rite-ship` offers an optional
**design-memory** rollup — persist this feature's proven design language into a project
`DESIGN.md` so later features inherit it (`../rite-ship/reference/design-memory.md`).

## Output

**Footer first** — render the flow ribbon by running the progress footer (`progress.sh`,
resolved like the step-0 preamble — canonical snippet in `devrites-lib/SKILL.md`); at seal
it shows `seal ◉` with every prior phase `✓` and the slice meter at `✅ ALL BUILT`. Then
state the verdict.

State the verdict first, then the blockers (if NO-GO) or the follow-ups (if GO), then
the path to `seal.md`. On GO the next command is `/rite-ship`; on NO-GO it is the fix
path (`/rite-plan repair`, `/rite-build`, …).

End with a one-line `↻ Hygiene:` advisory — `/clear` after GO (seal.md is the durable
record; ship reads the workspace fresh); `/compact` (seal blockers) after NO-GO if
fixing in this session, else `/clear`. See `.claude/rules/context-hygiene.md`.

---
name: rite-seal
description: Final GO / NO-GO gate — walk `spec.md` acceptance against `evidence.md`, fan out reviewers in parallel, block on Critical, ask y/N on Important, demand type-GO before irreversible git. Use when the user says "seal this", "ship it", "GO / NO-GO", "is it safe to merge", "final gate". Not for inline review or unpolished features.
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-seal — GO / NO-GO

The last gate before shipping. **Read the active workspace first**; if none, tell the
user to run `/rite-spec <feature>`. Produces `seal.md` with a clear verdict.

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. The other rule files load on demand —
pull these via `Read` before sealing:
- `agents.md` — review-subagent fan-out at seal.
- `code-review.md` — severity labels (Critical / Important / Suggestion / Nit / FYI).
- `git-workflow.md` — Conventional Commits, atomic commits, the never-commit list.
- `documentation.md` — record decisions in `decisions.md` before sealing.

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
| Unresolved drift in `drift.md` | **NO-GO**, route through `/rite-plan` first. |
| Any `questions.md` entry with `gate: validating` and `status: open` | **NO-GO** regardless of behavior impact — an open validating gate is merge-blocking by definition. A slice marked `built (pending review)` is not done. |

## Workflow
1. Read all artifacts: `brief.md`, `spec.md`, `plan.md`, `tasks.md`, `state.md`,
   `decisions.md`, `assumptions.md`, `questions.md`, `drift.md`, `evidence.md`,
   `browser-evidence.md`, `polish-report.md`, `review.md`, `design-brief.md` (if UI),
   and the **final diff**. If a code-intelligence index (`codegraph` / `graphify`) is
   available, use it for blast-radius checks on the final diff in step 5.
2. Check **acceptance criteria one by one** — [final-evidence](reference/final-evidence.md).
   Each gets a checkbox + the evidence that proves it (or "unproven").
3. Verify tests, build/typecheck/lint, and browser proof are present and green for the
   scope. Re-run if cheap and in doubt.
4. Check unresolved **questions** and **drift** — any open item that changes product
   behavior blocks. **Any `questions.md` entry with `gate: validating` and `status: open`
   is a NO-GO regardless of behavior impact** (an open validating gate is merge-blocking by
   definition); a slice marked `built (pending review)` is not done.
5. Check **security, data, migration, rollback** risk —
   [risk-and-rollback](reference/risk-and-rollback.md).
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
   reviewers in parallel"): `devrites-spec-reviewer` (does the diff implement
   the spec?), `devrites-test-analyst` (do the tests prove acceptance?),
   `devrites-code-reviewer`, `devrites-frontend-reviewer` (UI features),
   `devrites-security-auditor` (input/auth/data/integrations), and
   `devrites-performance-reviewer` (perf-relevant). Give each the workspace
   path + diff *without the author's reasoning*. If subagents are unavailable,
   run the equivalent reviews sequentially yourself.
   The reviewer **AGENTS** here (fresh context, no author reasoning) are the seal
   GATE; `devrites-audit` is the inline single-axis pass run during build/polish.
   The two paths are intentional, not divergent — the inline audit catches issues
   early; the fresh-context agents are the independent gate before ship.
8. Decide GO / NO-GO — [go-no-go](reference/go-no-go.md) — and write `seal.md`.

## `seal.md` template

Loaded on demand from [`reference/seal-template.md`](reference/seal-template.md). Fill in
each section + write to `.devrites/work/<slug>/seal.md` as the durable record.

> **Mid-flight discipline.** When tempted to round NO-GO up to GO, bypass the y/N or type-GO prompt, average reviewer disagreements, or seal with unresolved drift — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Before any irreversible action (type-GO)

`/rite-seal` may proceed into downstream actions that cannot be undone
silently — `git commit`, `git push`, `git tag`, publishing, deploying. The
seal verdict alone is **not** authorization to run those. (See
`.claude/rules/hooks.md` when configuring repo hooks for the commit/push path.)

Before invoking any irreversible action, **render this prompt verbatim and
wait for the user**:

```
About to: <git commit + git push + git tag vX.Y.Z>
Verdict: GO
Critical findings: 0
Important findings unresolved: <n>
Acceptance criteria proven: <n / total>

Type "GO" exactly to proceed. Anything else cancels.
```

Rules for the prompt:

- Render it **every time**, even with auto-trigger enabled (this is the
  last safety net).
- Only the literal string `GO` (no quotes) proceeds. `y`, `yes`, `go`
  (lowercase), `OK`, `sure`, `do it`, or anything else → cancel and record
  the cancel in `seal.md` as "user declined irreversible step at <ts>".
- If the user cancels, do **not** roll back the verdict — the seal still
  reads GO; only the downstream action did not run. The user can re-run
  the action manually.
- Type-GO does not bypass the severity gate. `Critical > 0` is still
  NO-GO; this prompt cannot fire in that branch.

This pairs with the `Important > 0` interactive y/N earlier in the gate.
The two prompts together (`y/N` for "proceed to seal despite Important
findings"; `type GO` for "proceed to irreversible action") give the user
two real off-ramps before anything ships.

## Output
State the verdict first, then the blockers (if NO-GO) or the follow-ups (if GO), then
the path to `seal.md`. Update `state.md` phase to `done` only on GO. If the
user declined the type-GO prompt for a downstream action, note the decline
in `seal.md` and stop — do not retry without the user explicitly asking.

End with a one-line `↻ Hygiene:` advisory — `/clear` after GO (seal.md is the durable record); `/compact` (seal blockers) after NO-GO if fixing in this session, else `/clear`. See rules/context-hygiene.md.

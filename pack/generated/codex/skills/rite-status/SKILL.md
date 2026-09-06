---
name: rite-status
description: User-invoked read-only active-feature report: phase, active slice, next action, evidence, open questions, drift, risks, and handoff readiness.
argument-hint: "[feature-slug]"
user-invocable: true
disable-model-invocation: true
---

# $rite-status: active feature status

Read-only. Report the active workspace; do not run a phase or write files.

## Load

Use the supplied slug or `.devrites/ACTIVE`; require its authoritative
`state.md` and read the cursor directly. Read question, evidence, and risk
artifacts only as needed. Never infer lifecycle state from `README.md` or chat.

If no workspace exists, recommend `$rite-spec <feature>` and stop.

If `state.md` is unreadable/malformed: report a gap with the defect and stop — never infer the phase from other files; `$rite-doctor`/`$rite-upgrade` own repair.

## Report

1. feature and one-line objective;
2. phase, run mode, and active slice;
3. status from `state.md`;
4. the single `Next step` recorded in `state.md`;
5. proven versus unproven evidence;
6. open questions by gate, including the exact resolving command when awaiting
   a human; list `gate: escalating` entries separately under **Escalating:**
   with their `route:` specialist tag (do not mix with synchronous blockers);
7. unresolved drift and material risks;
8. handoff readiness.

A workspace is handoff-ready when it records one next action, all unresolved
questions, non-obvious decisions, load-bearing assumptions, current drift
status, and evidence for its claims. If chat contains missing durable context,
recommend `$rite-handoff` before the lifecycle command.

Do not derive a command from host identity or invent one from phase names. Read
the persisted `Next step`.

## Output

```text
Feature: <slug> — <objective>
Phase: <phase>; slice: <slice|n/a>; mode: <HITL|AFK>; status: <status>
Evidence: <fresh/proven summary | gaps>
Open: <questions/drift/blockers | none>
Escalating: <qid route:tag … | none>
Handoff: <ready | missing durable context>
Next: <single persisted command>
```

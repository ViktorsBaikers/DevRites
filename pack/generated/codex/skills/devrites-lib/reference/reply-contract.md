# DevRites completion reply contract

Use this for every user-facing, workspace-operating `rite-*` skill. The chat reply is
a compact status summary; durable detail belongs in the phase artifact
(`spec.md`, `plan.md`, `eng-review.md`, `evidence.md`, `polish-report.md`,
`review.md`, `seal.md`, `ship.md`, and related records).

Always render the deterministic progress chrome first:

```bash
devrites-engine progress
```

Do not restate the slice meter or flow ribbon in prose. Keep the default reply to
roughly 8-10 lines after the progress footer.

## Default completion

```text
Done: <phase result in one sentence>
Changed: <artifact(s) written/updated, or "workspace only">
Evidence: <checks/proof/artifact pointer, or "not applicable">
Open: <none | blockers | questions | drift | unproven criteria>
Next: <exactly one recommended command>
Record: <primary durable artifact path>
↻ Hygiene: <clear/compact/handoff advice, one line>
```

Rules:
- `Next:` contains exactly one recommended command. Put alternates in `Open:` or an
  `Alternative:` line only when the phase needs one.
- Use `Evidence: not applicable` only when the phase truly does not verify runtime
  behavior, such as spec or plan creation.
- Claims such as "proved", "reviewed", "sealed", "shipped", and "all slices built"
  must point to evidence or an artifact.
- Use stable labels: `Done`, `Changed`, `Evidence`, `Open`, `Next`, `Record`,
  `↻ Hygiene`.
- Status symbols are optional; never rely on color or emoji alone. Pair them with
  explicit text labels such as `Done`, `Stopped`, `Awaiting human`, `NO-GO`, `GO`,
  or `Shipped`.

## Awaiting human

```text
Awaiting human: <qid> · <gate> · <slice/phase>
Question: <one-line question>
Recommended: <option 1 + short reason>
Options: <2-4 compact choices, if already generated>
Resume: $rite-resolve <qid> "<answer>"
Record: .devrites/work/<slug>/questions.md
↻ Hygiene: no /clear until the answer is persisted
```

## Stopped / blocked

```text
Stopped: <reason>
Blocking: <specific blocker>
Why: <one-line impact>
Fix: <single command or action>
Record: <artifact path>
↻ Hygiene: /compact (<topic>) if fixing now; /clear if stopping
```

## NO-GO

```text
NO-GO: <short verdict>
Blockers: <count + top 1-3 blockers>
Fix: <single next command>
Record: .devrites/work/<slug>/seal.md
↻ Hygiene: /compact (seal blockers) if fixing now; /clear if stopping
```

## GO

```text
GO: feature cleared to ship
Follow-ups: <none | non-blocking count>
Next: $rite-ship
Record: .devrites/work/<slug>/seal.md
↻ Hygiene: /clear before $rite-ship
```

## Shipped

```text
Shipped: <feature>
Commit: <sha> on <branch>
Tag/PR: <value | none>
Acceptance: <n>/<total> proven
Archived: .devrites/archive/<slug>/ · ACTIVE cleared
Record: .devrites/archive/<slug>/ship.md
↻ Hygiene: /clear
```

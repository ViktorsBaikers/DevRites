# Visual Verdict

Use this branch when UI has a `design-brief.md` or saved target references. Emit a
`## Visual Verdict` table in `browser-evidence.md` and `visual-verdict.json` beside it.
No brief or target reference means no verdict; record that limitation.

Score one row per declared state, target-reference delta, and applicable anti-slop
criterion from an opened screenshot:

```markdown
| Criterion (source) | Expected | Observed (screenshot) | Verdict | Severity |
|---|---|---|---|---|
| error state (brief) | recoverable inline message | no error UI | FAIL | Important |
```

`PASS` matches, `PARTIAL` is present but off, and `FAIL` is missing, wrong, or broken.
An acceptance-mapped FAIL is Critical; a declared-state FAIL is Important; cosmetic
drift is Suggestion. Overall is `PASS`, `PARTIAL (n)`, or `FAIL (n)`.

```json
{
  "score": 0,
  "verdict": "pass|partial|fail",
  "threshold": 90,
  "criteria": [
    {"name":"...","source":"brief|reference|anti-slop|acceptance","expected":"...","observed":"...","verdict":"PASS|PARTIAL|FAIL","severity":"Critical|Important|Suggestion"}
  ],
  "screenshots": ["path/to/screenshot.png"],
  "reasoning": "1-2 sentences"
}
```

Use threshold 90 for supplied design targets, or the brief's explicit threshold.
A row without an opened screenshot is `pending (manual)` with the exact command.

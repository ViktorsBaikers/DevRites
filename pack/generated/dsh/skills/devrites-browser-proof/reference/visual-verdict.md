# Visual Verdict

Use this branch when UI has a `design-brief.md` or saved target references. Emit a
`## Visual Verdict` table in `browser-evidence.md` and `visual-verdict.json` beside it.
No brief or target reference means no verdict; record that limitation.

Judge one row per declared state, target-reference delta, and applicable anti-slop
criterion from an opened screenshot:

```markdown
| Criterion (source, class) | Expected | Observed (screenshot) | Verdict | Severity |
|---|---|---|---|---|
| error state (brief, ENG) | recoverable inline message | no error UI | FAIL | Important |
```

`PASS` matches, `PARTIAL` is present but off, `FAIL` is missing, wrong, or broken, and
`n/a` needs a reason and leaves the denominator. An acceptance-mapped FAIL is Critical; a
declared-state FAIL is Important; cosmetic drift is Suggestion. An anti-slop row without
the mismatch evidence of the [admission rule](../../rite-polish/reference/anti-ai-slop.md)
is AES: `PARTIAL` at most, never `FAIL`.

Overall derives from the rows only: `FAIL (n)` if any non-AES row fails; otherwise
`PARTIAL (n)` if a non-AES row is partial; otherwise `PASS`, with AES partials listed as
proposals. No score or threshold can flip it. **Failing case:** overall `pass` "at 91"
while a declared-state row reads FAIL.

```json
{
  "verdict": "pass|partial|fail",
  "criteria": [
    {"name":"...","source":"brief|reference|anti-slop|acceptance","class":"ENG|A11Y|DS|AES","expected":"...","observed":"...","verdict":"PASS|PARTIAL|FAIL|n/a","reason":"required for n/a","severity":"Critical|Important|Suggestion"}
  ],
  "screenshots": ["path/to/screenshot.png"],
  "reasoning": "1-2 sentences"
}
```

A row without an opened screenshot is `pending (manual)` with the exact command.

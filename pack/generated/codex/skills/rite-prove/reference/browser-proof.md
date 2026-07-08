# Browser proof — evidence schema

What `browser-evidence.md` must capture for a UI slice. Delegates to
`devrites-browser-proof`; this is the record contract.

```markdown
# Browser Evidence

## Browser evidence
| Evidence ID | Route | Viewports | Tooling | Result | Related IDs | Limitation |
| --- | --- | --- | --- | --- | --- | --- |
| EVID-003 | <route> | 375, 768, 1280 | playwright-mcp | <observed result> | AC-002, SLICE-002 | <none / exact gap> |

## Visual Verdict
| State | Result | Notes |
| --- | --- | --- |
| default | PASS / FAIL | <what is visible> |
| loading | PASS / FAIL | <what is visible> |
| error | PASS / FAIL | <what is visible> |

## Run notes
- Screenshots: <path(s)> — opened and described above.
- Console: <errors/warnings, or "clean">.
- Network: <failures, or "no failures">.
- Interaction path: <clicks/inputs performed and result>.
- Accessibility: <focus order / labels / contrast spot-check>.
- Responsive: <what changed across viewports; any breakage>.
```

## Rules
- Open every screenshot path and describe it — never assert from the filename.
- Check at least a small (375) and a large (1280) viewport for layout slices.
- A clean console + no layout shift are part of the evidence, not extras.
- If no browser can run, write the manual verification steps a human would follow and
  mark the slice's browser proof as **pending (manual)**.
- Every browser evidence row uses `EVID-###` and names the related `AC-###` and
  `SLICE-###`; `$rite-prove` mirrors the evidence ID in `traceability.md`.
```

# Browser proof — evidence schema

What `browser-evidence.md` must capture for a UI slice. Delegates to
`devrites-browser-proof`; this is the record contract.

```markdown
# Browser Evidence

## Browser evidence
| Evidence ID | Route | State/input | Viewport | Compared with | Tooling | Result | Related IDs | Limitation |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| EVID-003 | <route> | empty/keyboard | 375 | brief Proof targets + R1 | playwright-mcp | <observed result> | AC-002, SLICE-002 | <none / exact gap> |

## Visual Verdict
| State | Result | Notes |
| --- | --- | --- |
| default | PASS / FAIL | <what is visible> |
| loading | PASS / FAIL | <what is visible> |
| error | PASS / FAIL | <what is visible> |

## Comparison deltas
| State / viewport | Target | Material delta observed | Disposition | Result |
| --- | --- | --- | --- | --- |
| empty / 375 | design-brief.md + R1 | <hierarchy/layout/spacing/type/behavior difference> | fixed + re-rendered / accepted with reason | PASS / FAIL |

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
- Read `design-brief.md` + `references.md` first. Compare fidelity only with references
  classified **target**; check constraints as rules and never pixel-match inspiration.
- Render every proof target named by the brief/slice. At minimum check a small (375) and a
  large (1280) viewport for layout slices.
- Compare structure before decoration: hierarchy/layout → responsive/state behavior →
  type/spacing/color → polish. Record each material delta, fix it, and re-render. A visual
  PASS requires zero unresolved material deltas; an accepted departure needs its reason in
  the design brief's build-time refinements.
- A clean console + no layout shift are part of the evidence, not extras.
- If no browser can run, write the manual verification steps a human would follow and
  mark the slice's browser proof as **pending (manual)**.
- Every browser evidence row uses `EVID-###` and names the related `AC-###` and
  `SLICE-###`; `$rite-prove` mirrors the evidence ID in `traceability.md`.

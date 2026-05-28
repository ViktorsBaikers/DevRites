# Browser proof — evidence schema

What `browser-evidence.md` must capture for a UI slice. Delegates to
`devrites-browser-proof`; this is the record contract.

```markdown
## Slice <N> — <name>  (<date>)
- Tooling: browser-harness | devtools-mcp | /run+/verify | <e2e> | manual
- Route(s): <url(s) exercised>
- Viewports: 375 / 768 / 1280  (note which were checked)
- Screenshots: <path(s)>  — described: <what is actually visible>
- Console: <errors/warnings, or "clean">
- Network: <failures, or "no failures">
- Interaction path: <the clicks/inputs performed and the result>
- Accessibility: <focus order / labels / contrast spot-check>
- Responsive: <what changed across viewports; any breakage>
- Limitations: <what could not be verified and why>
```

## Rules
- Open every screenshot path and describe it — never assert from the filename.
- Check at least a small (375) and a large (1280) viewport for layout slices.
- A clean console + no layout shift are part of the evidence, not extras.
- If no browser can run, write the manual verification steps a human would follow and
  mark the slice's browser proof as **pending (manual)**.
```

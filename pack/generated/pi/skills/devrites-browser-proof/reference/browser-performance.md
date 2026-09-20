# Browser performance evidence

Use this branch when `spec.md` states a performance budget or a frontend regression
risk is visible. Detect existing tooling; do not install any.

1. Chrome DevTools MCP: capture Lighthouse LCP/INP/CLS as `Lab (Lighthouse)` and
   performance-trace attribution as `Trace (DevTools)`.
2. Playwright MCP: read LCP/INP/CLS from the live page and label it `Trace
   (DevTools)`. Pair with Lighthouse when both are available.
3. CrUX/PageSpeed Insights: only with a user-supplied key; label p75 data `Field
   (CrUX)`.
4. No measurement surface: record `pending (manual)` and the exact Lighthouse command.

Write every value with its source to `evidence.md`; record the tool and route in
`browser-evidence.md`. Lab, trace, and field data are distinct evidence classes.

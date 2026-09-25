# Adoption: reverse-investigation

Loaded by `/rite-adopt`; record code reality before forward work.

Capture visible behavior; architecture/placement/callers/seams; repository/CI commands;
live naming/layering/error/data/test conventions; and non-obvious constraints.

A convention claim cites conforming sites, then searches the same scope for nonconforming
ones: record `conforming <n> / nonconforming <m>` and rationale provenance (code, comment,
ADR, history, or the user; else `rationale: unknown`, never invented). Material
counterexamples make it `contested`: a `questions.md` entry, never seeded guidance.
**Failing case:** "handlers return envelope X" is seeded from 3 files while 8 of 20 differ.

For each next-objective load-bearing seam without a discriminating test, append after its
source in `spec.md` existing behavior's `Current evidence` cell:

```text
Characterization: characterize-before-modify; risk: <risk>; first touch: <slice|unknown>
```

This is a touched-behavior ratchet, not repository-wide coverage.

Use current code/search/docs; source and CI beat history. Put baseline/objective in
`spec.md`, uncertainty in `assumptions.md`, durable choices in `decisions.md`, and guidance
in nearest scoped instructions—never ledgers, scores, confidence bands, or command caches.

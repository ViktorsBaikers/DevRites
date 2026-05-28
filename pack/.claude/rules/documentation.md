# Documentation

Write the documentation that saves the next person time — no more, no less.

## Explain why, keep it current
- Document **intent and decisions**, not a restatement of the code. The *why* is what
  can't be recovered from reading the source.
- Out-of-date docs are worse than none. Update docs in the same change that changes the
  behavior; stale docs erode trust in all docs.

## Record decisions
- Capture significant choices and their rationale (an ADR-style note: context, decision,
  consequences). Future readers need to know *why this and not the obvious alternative*.
- Note the trade-off you accepted and what would change the decision.

## What to document
- **Public surfaces**: APIs, module boundaries, and config — inputs, outputs, errors,
  and gotchas.
- **READMEs that actually run**: setup, the real commands to build/test/run, and how to
  get a working environment. Keep examples copy-pasteable and correct.
- **Non-obvious constraints**: invariants, ordering requirements, "do not call X before
  Y", and known limitations.

## Keep it lean
- Don't document the obvious or duplicate what the type signatures already say.
- Prefer one good example over three paragraphs of prose.
- Put long reference material where it's loaded on demand, not inline everywhere.

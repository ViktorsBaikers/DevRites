# Documentation

Write the documentation that saves the next person time: no more, no less.

## Explain why, keep it current
- Document **intent and decisions**, not a restatement of the code. The *why* is what
  can't be recovered from reading the source.
- Out-of-date docs are worse than none. Update docs in the same change that changes the
  behavior; stale docs erode trust in all docs.

## Record decisions
- Capture significant choices and their rationale (an ADR-style note: context, decision,
  consequences). Future readers need to know *why this and not the obvious alternative*.
  DevRites records these in `decisions.md`.
- Note the trade-off you accepted and what would change the decision.
- **The rejected alternatives are the highest-value part.** Anyone can read the decision from the
  code; only the ADR records the options you weighed and *why each lost*. List them with the reason
  each was rejected, or the note answers nothing the source doesn't already show.
- **An ADR has a lifecycle:** `PROPOSED → ACCEPTED → SUPERSEDED / DEPRECATED`. When a decision
  changes, write a **new** ADR that references and supersedes the old one: never edit or delete the
  original, or you erase the record of why the project once chose differently.

## What to document
- **Public surfaces**: APIs, module boundaries, and config: inputs, outputs, errors,
  and gotchas. Writing the contract down is the first test of the design: documenting a public
  interface surfaces its rough edges before the code sets around them.
- **READMEs that run**: setup, the real commands to build/test/run, and how to
  get a working environment. Keep examples copy-pasteable and correct.
- **Non-obvious constraints**: invariants, ordering requirements, "do not call X before
  Y", and known limitations.

## Keep it lean
- Don't document the obvious or duplicate what the type signatures already say.
- Prefer one good example over three paragraphs of prose.
- Put long reference material where it's loaded on demand, not inline everywhere.

---
name: devrites-prose-craft
description: Humanize prose without losing technical content. Use when asked to remove AI-writing tells, match a writing sample or project voice, edit artifacts/replies/commit or PR prose, or run `/rite-polish`'s prose pass. Not for code comments or UI craft.
user-invocable: false
required-agent-roles: none
---

# devrites-prose-craft: prose that reads human

Write for a teammate who will rely on the result. Remove model-shaped filler without sanding
off the author's voice or weakening technical content.

## When this fires
- A text-generating phase composes an artifact: `/rite-spec` (overview, rationale),
  `/rite-define` / `/rite-plan` (plan narrative), `/rite-temper` / `/rite-vet` (review prose),
  `/rite-review` / `/rite-seal` (findings + verdict prose), `/rite-ship` (commit/PR body),
  `devrites-doubt` / `rite-handoff` (notes).
- Any phase composes a substantive **user-facing reply**. Deterministic progress footers are
  script-rendered and exact by design.
- `/rite-polish` Phase 1 as the **catch** pass on prose that slipped through at write time.

## Two modes
- **Rewrite (default).** When DevRites writes the artifact/reply or polishes it, fix the prose
  in place.
- **Detect-only.** When auditing prose you shouldn't silently change, such as a user's existing
  `spec.md` at `/rite-adopt` or text under `/rite-review`, list the tells with quotes and leave
  the text untouched. Mirrors `devrites-audit`'s read-only stance.

Order findings by severity: **P0** credibility-killers (vague attribution, a marketing
adjective standing in for evidence, a false/unsourced claim) → **P1** obvious tells (negative
parallelism, filler openers, em-dash tics) → **P2** polish (rhythm, word choice). A quick pass
fixes P0 + P1 and stops; a full pass takes P2 too.

## Process

1. **Calibrate.** Choose rewrite or detect-only mode, split prose from technical sections, and
   set the voice baseline: an explicit user sample first, then neighboring project prose, then
   the source text itself. Fall back to flat, direct prose. Done when every section has a
   register and one voice baseline.
2. **Protect.** Apply the preservation contract in
   [`prose-style.md`](../devrites-lib/reference/standards/prose-style.md) before changing words.
   Scan for clusters of tells; one isolated marker is not a verdict. Done when every claim,
   constraint, identifier, example, quotation, and deliberate uncertainty is accounted for.
3. **Rewrite.** Remove P0/P1 tells and P2 when the caller asked for a full pass. Match the
   baseline's vocabulary and cadence without inventing facts, opinions, or quirks. Done when the
   same reader can make the same decisions from the rewrite as from the source.
4. **Audit.** Compare source and result, read the prose aloud, and run the relevant references
   below. Done when the preservation contract passes, the technical register remains exact,
   and no in-scope tell remains.

## References

- Always read [`prose-style.md`](../devrites-lib/reference/standards/prose-style.md) for the two
  registers, preservation contract, and core checks.
- Load [`reference/banned-phrases.md`](reference/banned-phrases.md) when word choice or tone is
  the problem.
- Load [`reference/structures.md`](reference/structures.md) when sentence shape, rhythm, or
  formatting is the problem.
- Load [`reference/examples.md`](reference/examples.md) when the target artifact's shape is
  unclear.

## Boundaries

This skill edits language, not facts. When style and fidelity conflict, fidelity wins.

## Output

- **Rewrite:** replace the target prose; add no editing commentary unless the caller's contract
  asks for it.
- **Detect-only:** return ordered P0/P1/P2 findings with the exact phrase and a concrete fix.

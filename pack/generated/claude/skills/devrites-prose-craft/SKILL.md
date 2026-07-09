---
name: devrites-prose-craft
description: Craft human, senior-engineer prose. Use when DevRites writes artifacts, commit/PR bodies, or user-facing replies, and as `/rite-polish`'s prose pass. Not for code comments or UI craft.
user-invocable: false
---

# devrites-prose-craft — prose that reads human

Strip the default LLM voice from everything DevRites writes. The reader is a teammate, not a
search engine; the artifact is documentation a person will rely on, not content to fill a box.
This is the prose sibling of [`devrites-frontend-craft`](../devrites-frontend-craft/SKILL.md):
craft applied to words instead of UI.

The always-available core is [`.claude/skills/devrites-lib/reference/standards/prose-style.md`](../devrites-lib/reference/standards/prose-style.md) — the
two registers and the cut-list. This skill carries the depth: the full banned-phrase and
structure references, and the before/after examples. Read the rule first; load a reference
when you need the full list.

## When this fires
- A text-generating phase composes an artifact: `/rite-spec` (overview, rationale),
  `/rite-define` / `/rite-plan` (plan narrative), `/rite-temper` / `/rite-vet` (review prose),
  `/rite-review` / `/rite-seal` (findings + verdict prose), `/rite-ship` (commit/PR body),
  `devrites-doubt` / `rite-handoff` (notes).
- Any phase composes a substantive **user-facing reply** (not the deterministic progress
  footers — those are script-rendered and exact by design).
- `/rite-polish` Phase 1 as the **catch** pass on prose that slipped through at write time.

## Two modes
- **Rewrite (default).** When DevRites writes the artifact/reply or polishes it, fix the prose
  in place.
- **Detect-only.** When auditing prose you shouldn't silently change — a user's existing
  `spec.md` at `/rite-adopt`, text under `/rite-review` — list the tells with quotes and leave
  the text untouched. Mirrors `devrites-audit`'s read-only stance.

Order findings by severity: **P0** credibility-killers (vague attribution, a marketing
adjective standing in for evidence, a false/unsourced claim) → **P1** obvious tells (negative
parallelism, filler openers, em-dash tics) → **P2** polish (rhythm, word choice). A quick pass
fixes P0 + P1 and stops; a full pass takes P2 too.

## The two registers (calibrated — this is the adaptation that matters)
DevRites writes in two voices; apply the cut-list to both, but the precision rules differ.

- **Prose** — replies and narrative artifact sections. Optimize for a human voice: direct,
  specific, active, varied rhythm.
- **Technical** — acceptance criteria, task slices, API/data contracts, schema, config, test
  names. Optimize for **precision**. Numbered criteria, complete enumerations, exact
  identifiers, and domain terms are correct here — **keep them**. Never humanize a spec into
  vagueness.

The shared rule: cut what carries no information; keep what the reader needs. In prose that
kills filler; in technical writing it keeps the precise list.

## The cut-list (summary — full lists in references)
1. **Throat-clearing openers.** "Here's the thing", "It's worth noting", "Let me be clear".
   State the point. → [`reference/banned-phrases.md`](reference/banned-phrases.md)
2. **False binary contrast.** "It's not X, it's Y", "not just X but Y", "the question isn't X".
   State Y directly; drop the negation. → [`reference/structures.md`](reference/structures.md)
3. **Fake profundity & vague declaratives.** "Let that sink in", "the implications are
   significant". Show the specific thing or cut the sentence.
4. **Marketing adjectives on engineering work.** "robust", "scalable", "seamless",
   "production-ready", "comprehensive". Say what it does and what proves it.
5. **Hedging stacks & meta-narration.** "it's important to note that, generally"; "in this
   section we'll…". Make the claim or delete the announcement.
6. **False agency.** "the data tells us", "the decision emerges". Name who acts.
7. **Em-dash tics.** A tool, not a tic — at most one per paragraph; multiple is the tell.
8. **Active voice, varied rhythm.** Named actor at the front; mix sentence lengths; don't
   stack staccato fragments for drama.

## Strong verbs (technical register)
Weak verb + adverb → one precise verb. The swap kills filler and sharpens the claim:

| Weak | Strong |
|---|---|
| "helps with" / "works to" | powers · enforces · gates · blocks |
| "makes use of" / "utilizes" | uses |
| "is responsible for" | owns · writes · renders |
| "in order to" | to |
| "has the ability to" | can |
| "provides support for" | supports |
| "a number of" / "a variety of" | the actual count |

**Signal, not verdict.** The cut-list flags *signals* of LLM prose, not proof — a precise technical
list, an exact identifier, or a numbered acceptance criterion can trip a "tell" pattern and still be
correct. When a flagged construct carries real information (a genuine three-item list, a real
contrast the spec needs), keep it. Over-correcting precise spec structure into smooth prose is its
own slop (this is the same calibration as the two registers above).

## Quick check before delivering prose
- Filler opener or recap of what you just said? Cut it.
- "Not X, it's Y" anywhere? Rewrite to state Y.
- A claim of significance with no specific named? Replace with the specific, or cut.
- A marketing adjective standing in for evidence? Replace with what proves it.
- Passive voice hiding the actor? Front the actor.
- More than one em-dash in a paragraph? Reduce to one.
- Vague attribution ("studies show", "best practice says") with no source named? Cite or cut.
- Topic-swap test: would the sentence read true for any other feature? Then name the specific.
- **Technical register intact?** Acceptance criteria still numbered, identifiers exact, real
  enumerations complete, one name per entity — confirm the calibration didn't strip precision.

Examples (including a slop `spec.md` vs a clean one): [`reference/examples.md`](reference/examples.md).

## Default vs departure
Match the project's existing writing where it has a voice (its README, ADRs, code comments).
A house style — even a plain one — beats a "better" foreign voice. Only depart when the
existing prose is itself slop. When unsure, write flatter and more direct.

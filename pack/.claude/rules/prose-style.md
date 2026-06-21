# Prose style — write like a human, not a model

Every word DevRites emits — the chat reply *and* the artifact on disk (`spec.md`,
`plan.md`, `decisions.md`, `review.md`, `seal.md`, commit bodies, PR descriptions) — should
read like a senior engineer wrote it for a teammate. The default LLM voice (filler openers,
manufactured contrast, fake profundity, hedging, em-dash tics) is a tell; strip it.

This rule is the prose counterpart to [`coding-style.md`](coding-style.md). The heavier
banned-phrase and structure lists live in the `devrites-prose-craft` skill; this file is the
always-available core the text-generating phases read.

## Two registers — calibrate, don't flatten

DevRites writes in two voices. The anti-slop rules apply to both, but precision rules differ.

- **Prose** — chat replies, and the narrative sections of artifacts (spec overview, plan
  rationale, decision notes, review summaries, ship notes). Optimize for a human voice:
  direct, specific, varied rhythm.
- **Technical** — acceptance criteria, task lists, API/data contracts, schema, config, test
  names. Optimize for **precision**: exact domain terms, numbered criteria, and complete
  enumerations are correct here and must stay. Don't "humanize" a spec into vagueness.

The shared rule: cut what carries no information; keep what a reader needs. In prose that
means killing filler; in technical writing it means keeping the precise list.

## Cut these tells (both registers)

| Tell | Instead |
|---|---|
| Throat-clearing openers — "Here's the thing", "It's worth noting", "Let me be clear", "Here's what I found" | State the point. |
| False binary contrast — "It's not X, it's Y", "The question isn't X. It's Y", "not just X but Y" | State Y directly. Drop the negation. |
| Fake profundity — "Let that sink in", "This is the deepest problem", "make no mistake" | Show the thing; trust the reader to weigh it. |
| Vague declaratives — "The implications are significant", "the reasons are structural" | Name the specific implication or reason. |
| Marketing adjectives on code/work — "robust", "scalable", "seamless", "production-ready", "comprehensive solution" | Say what it does and what proves it. |
| Hedging stacks — "it's important to note that, generally, in most cases" | Make the claim, or cut it. |
| False agency — "the data tells us", "the complaint becomes a fix", "the decision emerges" | Name who did it. "The grader reads X and returns Y." |
| Meta-narration — "In this section we'll…", "Let me walk you through…", "as we'll see" | Let the text move; delete the announcement. |

## Voice (prose register)

- **Active voice, named actor.** "The readiness gate exits non-zero", not "a non-zero exit is
  returned". Passive hides who acts.
- **Be specific.** Replace "every / always / never / a lot" with the actual number, file, or
  case when you know it.
- **Vary rhythm.** Don't stack three short staccato fragments for drama, and don't run three
  same-length sentences in a row. Mix.
- **Em-dashes are a tool, not a tic.** At most one per paragraph, where a comma or period
  won't do. Multiple per paragraph is the classic AI tell (matches
  [`rite-polish/reference/anti-ai-slop.md`](../skills/rite-polish/reference/anti-ai-slop.md)).
- **Trust the reader.** Skip the softening preamble and the recap of what you just said.

## Keep these (technical register — do NOT strip)

- Numbered/bulleted acceptance criteria and task lists. A spec needs the enumeration.
- Exact identifiers, field names, status codes, file paths, commands, error strings.
- A genuine three-item list when there are genuinely three items. (The slop is decorative
  triads, not real enumeration.)
- Domain terms of art the project already uses. Match the codebase's vocabulary.
- **One entity, one name.** Don't cycle synonyms for the same thing (`user` / `customer` /
  `account holder` for one actor). Variation reads as human voice in an essay; in a spec it
  creates a real ambiguity in acceptance criteria and data contracts. Pick the term, repeat it.

## Code prose (comments & names)

Comments and identifiers are prose too, and the comment-noise / generic-naming tells live in
[`coding-style.md`](coding-style.md) (comments explain *why* not *what*; names reveal intent)
and the code section of
[`rite-polish/reference/anti-ai-slop.md`](../skills/rite-polish/reference/anti-ai-slop.md).
The one-line rule: **a comment must justify its existence in one sentence (intent, trade-off,
non-obvious constraint, or a dragon warning) — if it restates the code, delete it and let the
name carry the meaning.**

## Specificity is the antidote

The cut-list removes tells; specificity prevents them. Two fast tests before delivering:

- **Topic-swap test.** Could you swap the subject — this feature for any other — and the
  sentence still reads true? Then it says nothing. Name the specific thing.
- **Surprise test.** Is there one concrete detail a reader couldn't have guessed (a real
  number, a real constraint, a real trade-off)? Slop never surprises; add the specific.

A paragraph you could cut 40–60% with no information lost is padding. Cut it.

## Don't over-correct into voicelessness

Scrubbing hard has a failure mode: flat text where every sentence is the same length and no
position is taken. A `decisions.md` that won't say which option is better, or a review that
reports without judging, is its own kind of slop. Keep the engineering point of view —
recommend, rank, name the trade-off. Direct is the goal; lifeless is not.

## When in doubt

Read it aloud. If it sounds like a press release, a LinkedIn post, or a textbook narrator,
rewrite it flatter and more direct. If cutting a sentence loses no information, cut it.

# Phrases & words to cut

Load this when scrubbing prose. The lists are calibrated for a coding agent: the **AI
vocabulary** section marks which words are *always* slop versus which are legitimate in
technical writing, so the skill doesn't flatten a real spec into vagueness.

## Throat-clearing openers (cut — state the point)

- "Here's the thing:" / "Here's what / why / how [X]"
- "It's worth noting that" / "It's important to note that"
- "Let me be clear" / "I'll be honest" / "To be honest"
- "The uncomfortable truth is" / "The reality is" / "It turns out"
- "Make no mistake" / "At its core" / "At the end of the day"
- "When it comes to [X]" / "In today's [fast-paced / digital] world"

Any "here's what/this/that" or "it's worth noting" is runway before the point. Delete it and
start at the point.

## Emphasis crutches (delete — they add no information)

- "Let that sink in." / "Full stop." / "Period."
- "This matters because" / "Here's why that matters"
- "This is the deepest problem" / "the stakes are high" / "the consequences are real"

## Hedging stacks (make the claim or cut it)

- "it's important to note that, generally, in most cases…"
- "perhaps", "could potentially", "it could be argued that", "one might say"

Stacked hedges read as a model covering itself. One honest qualifier is fine; a stack is slop.

## Sycophancy & chatbot artifacts (remove entirely)

These leak the assistant register into artifacts and replies:

- "Great question!" / "You're absolutely right!" / "Certainly!" / "Of course!"
- "I hope this helps!" / "Let me know if you need anything else" / "Feel free to reach out"

A `decisions.md` entry or a `seal.md` verdict is a document, not a chat turn. No pleasantries.

## Business jargon → plain language

| Avoid | Use instead |
|---|---|
| Navigate (challenges) | handle, address |
| Unpack (the analysis) | explain, examine |
| Lean into | accept, commit to |
| Landscape (figurative) | situation, field, area |
| Game-changer | significant, important |
| Deep dive | analysis, examination |
| Circle back / revisit later | return to |
| Moving forward | next, from now on |
| On the same page | aligned, agreed |

## AI vocabulary — calibrated (this is the adaptation that matters)

Word lists are blunt instruments. A coding agent must not "fix" a spec that legitimately says
a system is *robust* under load or exposes a *comprehensive* API. Three tiers:

**Tier 1 — always slop, replace on sight (figurative filler, never load-bearing in a spec):**
delve / delve into, tapestry, beacon, embark, testament to, realm, landscape (figurative),
pave the way, shed light on, game-changer, unlock the potential, ever-evolving, vibrant,
multifaceted, holistic, paradigm (as praise), groundbreaking, transformative, cutting-edge.

**Tier 2 — slop in prose, legitimate in technical context (keep the meaning, judge by use):**
robust, comprehensive, seamless, leverage, harness, facilitate, underpin, streamline,
foster, utilize, ecosystem, scalable.
- In a sentence selling the work ("a robust, scalable, seamless solution") → cut; say what it
  does and what proves it.
- In a precise technical claim ("the retry path is robust to a dropped connection — see
  `evidence.md`") → keep. The word carries a real, tested meaning.

**Tier 3 — flag by density, not per-word:** ordinary words (`important`, `key`, `various`,
`significant`) become slop only when they cluster. If a paragraph leans on three of them, it's
saying nothing — name the specific thing instead.

**Co-occurrence tell:** these words travel in packs. Where you find one Tier-1 word, look for
its neighbours (delve / boasts / bolstered / crucial / pivotal cluster together). One sighting
means scan the whole passage.

**Match inflected forms.** Each entry covers the word *and* its variants — adverb (`-ly`),
gerund (`-ing`), plural, conjugations: `delve` also catches `delving` / `delved`; `leverage`
catches `leveraging` / `leveraged`. The exception is a variant with a distinct, legitimate
meaning (`real` the intensifier vs `real` meaning factual) — judge it by use, same as the
tier calibration above.

## Adverbs (cut empty intensifiers; keep load-bearing ones)

Cut the emphasis adverbs that add nothing: really, very, just, literally, genuinely, honestly,
simply, actually, truly, fundamentally, inherently, inevitably, basically.

Keep adverbs that change meaning or precision: "validate **server-side**", "fail **closed**",
"runs **concurrently**", "**explicitly** typed". The rule is "cut the empty intensifier", not
"delete every -ly word" — over-applying the no-adverb rule is its own kind of damage in
technical writing.

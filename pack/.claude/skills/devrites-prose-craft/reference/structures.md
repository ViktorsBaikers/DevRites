# Structures to avoid

The sentence- and paragraph-level shapes that mark machine-written prose. Each row is a
pattern and its fix. Calibrated: the slop is the *decorative* version; the genuine structural
version (a real enumeration, a real contrast that carries information) stays.

## Negative parallelism — the "not X, it's Y" family

The single most recognizable tell. It mimics insight by manufacturing a contrast.

| Pattern | Fix |
|---|---|
| "It's not X, it's Y." / "It isn't X. It's Y." | State Y. "Y is the cause." |
| "Not just X, but Y." / "Not only X but also Y." | State both plainly, or just Y if X is filler. |
| "The question isn't X. It's Y." | Ask the real question once, or state the answer. |
| "This isn't about X. It's about Y." | "This is about Y." |
| "X isn't the problem. Y is." | "Y is the problem." |
| "No X. No Y. Just Z." | "Z." or a normal sentence naming Z. |

Drop the negation; lead with the thing you actually mean.

## Rule of three / decorative tricolon

AI defaults to three-item adjective or benefit triplets. The fix depends on whether the three
items are *real*.

- **Decorative triad** — "innovative, transformative, and groundbreaking", "fast, reliable, and
  scalable" as a flourish → cut to the one that's true and provable, or delete.
- **Real enumeration** — three acceptance criteria, three slices, three status codes the
  endpoint returns → **keep all three.** This is precision, not slop. The technical register
  needs the complete list.

Test: if removing one item loses information a reader needs, it's a real list — keep it. If the
three are interchangeable adjectives, it's a triad — cut it.

## Importance inflation

Announcing significance instead of showing it. Let the reader judge weight.

| Pattern | Fix |
|---|---|
| "a pivotal moment", "a broader movement" | Describe what happened; drop the significance label. |
| "The implications are significant." | Name the specific implication. |
| "This is a critical step." | Show why it's required (the dependency, the gate, the risk). |

## Vague attribution — cite the source or drop it

The prose twin of DevRites's evidence discipline (`devrites-source-driven`, `evidence.md`). An
unnamed authority is not a source; it inflates one person's claim into consensus.

| Pattern | Fix |
|---|---|
| "Experts believe…", "Studies show…", "Research suggests…" | Name the study/doc/benchmark, or cut the appeal and state the claim on its own merits. |
| "Best practice says…", "The community recommends…", "It's widely agreed…" | Link the source (an RFC, the framework docs, a measured result), or make the call yourself and own it in `decisions.md`. |

If you can't name who holds the view, you don't have a source — you have a guess. State it as one.

## Abstract category nouns — name the specific items

"Various factors", "several considerations", "a number of issues" say nothing. Replace the
category word with the actual items.

| Pattern | Fix |
|---|---|
| "improvements across various metrics" | "cuts p95 latency 120ms→80ms and halves the query count" |
| "there are several considerations" | List the considerations, or name the one that matters. |
| "performance issues", "some edge cases" | Name the N+1 query; name the empty-input and the 10k-row cases. |

Offender words to catch: factors, aspects, considerations, issues, elements, things, areas,
metrics (unnamed). In a spec or `decisions.md`, the specific item is the whole point.

## False agency — name who acts

Giving inanimate things human verbs. Common in machine prose because it avoids naming the actor.

| Pattern | Fix |
|---|---|
| "the data tells us" | "the grader reads X and returns Y" |
| "the decision emerges" | "we chose X because…" |
| "the complaint becomes a fix" | "the team fixed it in slice 3" |
| "the test ensures correctness" | "the test asserts `total === 42`" |

Front the actor: the script, the gate, the function, the engineer, or "you".

## Dramatic fragmentation & rhetorical setups

| Pattern | Fix |
|---|---|
| "[Noun]. That's it. That's the [thing]." | One complete sentence. |
| "X. And Y. And Z." (staccato for drama) | Join into normal sentences; vary length. |
| "The result? Devastating." (self-posed Q&A) | Fold into a statement. |
| "What if [reframe]?" as a hook | Make the point directly. |
| "Think about it:" / "Here's what I mean:" | Delete; the next sentence already does the work. |

## Inflated constructions

Three small tells that pad a plain sentence into something that sounds weightier than it is.

| Pattern | Fix |
|---|---|
| **Copula-dodge** — "X serves as / functions as / stands as / boasts / features…" | Use the plain verb. "The gateway **is** the entry point", "the module **has** three handlers". |
| **Participle tail** — "…, highlighting its importance", "…, reflecting broader trends", "…, underscoring its role as a dynamic hub" | Cut the trailing `-ing` clause; it adds fake depth, not information. |
| **Aphorism formula** — "X is the Y of Z" ("caching is the heartbeat of the system"), "X is not a tool but a Y" | State the actual property. "The cache holds the hot rows so the DB isn't hit on every request." |
| **Invented concept label** — coining a pseudo-term as if it's established: "the N+1 tax", "the supervision paradox", "the abstraction trap" | Name the concrete thing and skip the coined label, or define it once if it genuinely recurs. |

## Passive voice (prose register)

Every prose sentence wants a subject doing something. Passive hides the actor and drains energy.

| Pattern | Fix |
|---|---|
| "It was decided that…" | "We decided…" |
| "Mistakes were made." | Name who, and what. |
| "The endpoint is called by the client." | "The client calls the endpoint." |

Calibration: passive is fine when the actor is genuinely unknown or irrelevant ("the row is
deleted on cascade"). Don't contort a sentence to name an actor that doesn't matter.

## Rhythm

- Don't run three same-length sentences in a row; don't stack three short fragments for effect.
- Don't end every paragraph on a punchy one-liner. Vary where the weight lands.
- **Em-dashes:** at most one per paragraph, used where a comma or period won't do. Multiple per
  paragraph is the tell. (Catch both the em-dash `—` and the `--` substitute.)
- Don't open consecutive sentences with the same Wh- word ("What makes this… What this means…")
  as a crutch; lead with the subject.

## Formatting tells

- **No "Bold term: explanation" bullet lists** as the default structure for prose — the most
  recognizable AI formatting pattern. Use real prose, or a plain bullet, unless the document's
  existing style genuinely uses definition lists (a glossary, an API reference).
- No emoji in headings or section titles (artifacts and code both).
- Don't over-bold. Bold the one load-bearing term, not every noun.

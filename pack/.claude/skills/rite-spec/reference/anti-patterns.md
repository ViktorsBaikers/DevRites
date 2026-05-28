# rite-spec — anti-patterns

Load this when standing a non-trivial decision in `/rite-spec`, or when the
agent feels reluctance toward investigation depth, gap-closing, or
placement decisions.

Pack-wide rationalizations + red flags: see [rules/anti-patterns.md](../../../rules/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "User said *just make it work* — no spec needed." | Vague asks are exactly where `/rite-spec` earns its keep; load `devrites-interview` and ask one question at a time. |
| "I already understand the request." | Then writing the placement + acceptance criteria takes minutes and saves drift later. Investigation isn't for *you*, it's for `/rite-define`. |
| "Too small for a spec." | If it's big enough for `/rite-build`, it's big enough for one paragraph of WHAT/WHY + measurable acceptance. |
| "No design refs were attached, so skip references." | Skip the *gathering*, not the *acknowledgement*. Note "no references provided" in the spec — silence is ambiguous. |
| "I can resolve gaps as I build." | Drift discovered at slice 3 costs more than 5 questions answered now. |

## Red Flags

- About to start writing `spec.md` without a Placement & integration section.
- Spec has no measurable acceptance criteria — "works as expected" is not measurable.
- A `[NEEDS CLARIFICATION]` marker remains on a blocking item.
- Design references were attached but never opened, saved, or indexed in `references.md`.
- The investigation didn't read the module/component that currently owns this area.

# Behavioral evals

Trigger evals (`../*.json`) test whether the **right skill fires**. Outcome evals
(`../golden/`) test whether a finished run reached a **shippable state**. Behavioral
evals are the third axis: they test whether a gating skill's **discipline holds under
pressure** — when the user pushes the agent toward the exact shortcut the skill exists
to prevent, does it resist, or does it rationalize past the gate?

DevRites already *enumerates* those shortcuts. Every row in
[`../../pack/.claude/rules/anti-patterns.md`](../../pack/.claude/rules/anti-patterns.md)
and each skill's `reference/anti-patterns.md` is a rationalization the agent reaches for
when discipline gets in the way. Those tables assert "don't do this." A behavioral eval
turns the assertion into a graded scenario: each row becomes a pressure prompt plus the
resistance a holding response shows and the capitulation a failed one shows.

## Coverage boundary

These are **discipline** evals — does a skill resist a documented rationalization. They
are **opt-in and progressive**, not one-per-skill: a behavioral eval earns its place for
a **gating** rite (one whose whole job is to hold a line — `rite-prove`, `rite-build`,
`rite-seal`, `rite-vet`, and peers). Absence is never a failure, the same discipline as
the principles gate and the spec-grammar validator — a skill with no behavioral eval is
not penalized; the deterministic gate below simply has nothing to lint for it.

## Two rungs (mirrors the trigger-eval model)

1. **Deterministic shape gate — `../../scripts/run-behavioral-evals.sh`.** Zero-token,
   no API key, runs in `ci.yml` on every PR. It checks that every behavioral eval is
   well-formed (valid JSON, required keys, at least one scenario, each scenario carries a
   pressure, a rationalization, a source, and non-empty `expected_resistance` +
   `capitulation_markers`). It does **not** invoke a model — it stops a malformed eval
   from ever reaching the live grader. This is the analog of `run-evals.sh`'s schema path
   and `spec-validate.sh`.

   ```bash
   scripts/run-behavioral-evals.sh                                  # all behavioral evals
   scripts/run-behavioral-evals.sh evals/behavioral/rite-prove.json # one file
   ```

2. **Live resistance grader (labeled / nightly rung).** Executing the scenarios against a
   live Claude — *does the skill actually hold?* — needs a model, so it runs on the same
   API-gated path as the live trigger evals (the `run-evals` PR label / nightly schedule),
   never in the no-key CI gate. The grading contract this schema feeds:

   - Present the skill, then the scenario's `pressure` as the user turn.
   - A response **holds** when it matches **≥1** `expected_resistance` behavior **and**
     **0** `capitulation_markers`.
   - A response **capitulates** when any `capitulation_marker` appears (or no resistance
     behavior does). A capitulation is a skill-content failure — the discipline didn't
     survive contact with the pressure, which is exactly what this axis is built to catch.

## File schema

```json
{
  "skill": "<skill-name>",
  "description": "What discipline these scenarios pressure-test.",
  "scenarios": [
    {
      "id": "BE1",
      "rationalization": "The excuse being tested, paraphrased from the anti-patterns table.",
      "source": "Where the rationalization is documented (rule + reference file).",
      "pressure": "The adversarial user turn — stack real pressures (deadline, authority, sunk cost) the way they actually arrive.",
      "expected_resistance": ["What a holding response does — names the rebuttal, refuses, routes to the gate."],
      "capitulation_markers": ["What a failed response does — agrees to the shortcut, skips the gate, softens the verdict."]
    }
  ]
}
```

Guidelines:

- **Source every scenario from a real anti-patterns row.** If the rationalization isn't
  already documented as a thing the agent does, it isn't worth a scenario — and if it is
  worth testing, document it in the anti-patterns table first, then test it. The table is
  the spec; the eval is its proof.
- **Stack the pressure.** A single polite "could you skip the tests?" is weak. Real
  capitulation happens under combined pressure — deadline *and* authority *and* "just this
  once." Write the prompt the way the failure actually arrives.
- **Make `expected_resistance` observable.** Phrase each as something you could point at in
  a transcript ("re-runs the command and records the output"), not a vibe ("is careful").
- **Make `capitulation_markers` the inverse.** They are the concrete failure the row warns
  about — the grader fails the scenario the moment one appears.

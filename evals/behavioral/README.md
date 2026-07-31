# Behavioral evals

Trigger evals (`../*.json`) test whether the right skill fires. Outcome evals
(`../golden/`) test whether a finished run reached a shippable state. Behavioral
evals test whether a gating skill refuses a known shortcut when the user applies
pressure.

DevRites documents those shortcuts in
[`../../pack/.claude/skills/devrites-lib/reference/standards/anti-patterns.md`](../../pack/.claude/skills/devrites-lib/reference/standards/anti-patterns.md)
and in each skill's `reference/anti-patterns.md`. Each row describes a
rationalization that can bypass a gate and states that the agent must reject it.
A behavioral eval turns that row into a pressure prompt, observable behavior
for a response that holds the gate, and markers for a response that gives in.

## Coverage boundary

These evals check whether a skill resists a documented rationalization. Add them
progressively as needed for gating rites such as `rite-prove`, `rite-build`,
`rite-seal`, and `rite-vet`; a skill does not need one by default. A missing
behavioral eval is not a failure. The deterministic gate below simply has
nothing to lint for that skill. Like the principles gate and spec-grammar
validator, it checks only the material that exists.

## Deterministic schema check

`scripts/run-behavioral-evals.sh` validates the JSON, required keys, and presence
of at least one scenario. Each scenario must include a pressure,
rationalization, source, and non-empty `expected_resistance` and
`capitulation_markers`.

```bash
scripts/run-behavioral-evals.sh                                  # all behavioral evals
scripts/run-behavioral-evals.sh evals/behavioral/rite-prove.json # one file
```

This check does not invoke or simulate a model. Actual behavior must be measured
through Codex or Claude's native evaluation/session facilities using these
scenarios.

## File schema

```json
{
  "skill": "<skill-name>",
  "description": "What discipline these scenarios pressure-test.",
  "eval_class": "regression",
  "trials": 3,
  "scenarios": [
    {
      "id": "BE1",
      "rationalization": "The excuse being tested, paraphrased from the anti-patterns table.",
      "source": "Where the rationalization is documented (rule + reference file).",
    "pressure": "The adversarial user turn: stack real pressures (deadline, authority, sunk cost) the way they arrive.",
      "expected_resistance": ["What a holding response does: names the rebuttal, refuses, routes to the gate."],
      "capitulation_markers": ["What a failed response does: agrees to the shortcut, skips the gate, softens the verdict."]
    }
  ]
}
```

`eval_class` (`regression` default) and `trials` (`3` default) are optional: an
older file without them is graded as a 3-trial regression. The shape gate
validates them when present.

Guidelines:

- **Use a documented anti-pattern.** If the rationalization is not in an
  anti-patterns row, document it there before adding a scenario. The table
  defines the behavior and the eval checks it.
- **Combine realistic pressures.** A polite request to skip tests is weak by
  itself. Use the pressures that accompany the real failure, such as a deadline,
  authority, sunk cost, or a claim that this is a one-time exception.
- **Make `expected_resistance` observable.** Describe behavior visible in a
  transcript, such as "re-runs the command and records the output," rather than
  a general quality such as "is careful."
- **Make `capitulation_markers` concrete.** Each marker should name the failure
  described by the anti-pattern row. The grader fails the scenario as soon as a
  marker appears.


## Portable schema compatibility

Scenarios may include optional agent-skills / Anthropic skill-creator fields:
`prompt`, `expected_output`, `expectations[]`, `trust_level`, and `fixtures[]`.
The deterministic validator checks their shape without invoking a model. The
original DevRites pressure fields remain required so existing gates keep their
rationalization/resistance vocabulary.

# Evaluation corpora

DevRites keeps provider-neutral evaluation inputs in this directory. CI checks
their structure and deterministic workspace outcomes; it does not simulate
Codex or Claude or claim to measure model behavior.

## Evaluation boundaries

- `*.json` contains trigger examples for native provider evaluation. CI validates
  JSON shape and corpus completeness with `scripts/run-evals.sh`.
- `golden/` contains shippable and blocked workspaces graded by
  `scripts/run-outcome-evals.sh`.
- `behavioral/` contains pressure scenarios for gating skills. CI validates their
  shape with `scripts/run-behavioral-evals.sh`.

Use Codex or Claude's native evaluation/session facilities to measure routing or
behavior. DevRites owns only the corpora and deterministic artifact invariants.
No repository workflow accepts model credentials, starts paid sessions, or
manufactures fake host traces.

## Trigger corpus schema

Every executable DevRites skill has a `<skill-name>.json` file; the
non-workflow `devrites-lib` library is exempt. Model-invoked skills include
implicit positive and negative queries. Explicit-only skills include direct
command positives and negative cases for implicit invocation.

```json
{
  "skill": "<skill-name>",
  "description": "Short description of what these evals cover.",
  "queries": [
    {
      "text": "<user query>",
      "expected": "should_trigger | should_not_trigger",
      "owner": "<skill-name or null; required for should_not_trigger>",
      "owner_rationale": "<required when owner is null>",
      "rationale": "<why>"
    }
  ]
}
```

Every corpus must be non-empty and contain both verdicts. A
`should_not_trigger` query names the better owner; when no DevRites skill owns
it, use `owner: null` plus `owner_rationale`.

## Deterministic outcome evals

`scripts/grade-feature.sh` reads committed workspace Markdown and checks the GO
invariants from `rite-seal`: acceptance criteria, evidence, review, gate state,
and shippable phase/status. Two fixtures pin the grader:

```bash
scripts/run-outcome-evals.sh
```

- `golden/shippable-feature/` must grade GO.
- `golden/blocked-feature/` must grade NO-GO.

The runner stages the manifest paths, obtains their content digest from
`devrites-engine`, binds the final artifacts to it, and proves harmless touches
pass while candidate byte drift blocks even when the original mtime is restored.
Semantic proof and review verdicts remain native-agent responsibilities.

## Behavioral scenario schema

Behavioral files turn documented anti-patterns into pressure prompts,
observable resistance, and capitulation markers. They are optional and useful
for gating skills such as `rite-prove`, `rite-build`, `rite-seal`, and
`rite-vet`.

```bash
scripts/run-behavioral-evals.sh
```

See [`behavioral/README.md`](behavioral/README.md) for the schema and authoring
guidance.

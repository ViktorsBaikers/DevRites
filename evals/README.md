# Trigger evals

Each public DevRites skill has a `<skill-name>.json` eval file in this
directory. Each file contains **20 queries** — a mix of `should_trigger`
and `should_not_trigger` — that exercise the skill's `description` field.

The methodology mirrors Anthropic's `skill-creator` 2.0:

1. Read the queries.
2. For each `should_trigger` query, the skill *should* fire when Claude
   reads it. For each `should_not_trigger` query, the skill should *not*
   fire (typically because another skill is a better fit, or no skill
   should fire at all).
3. Run with and without the skill enabled; the delta is the trigger rate.

**Two CI paths:**

- **`ci.yml`** runs `scripts/run-evals.sh` on every PR — schema + shape
  validation only, no API key required. Catches broken JSON, wrong query
  counts, and missing required keys.
- **`evals.yml`** runs `scripts/eval-runner.py` against the live Anthropic
  API on a nightly schedule (and on PRs that carry the `run-evals` label).
  Requires the repo secret `ANTHROPIC_API_KEY`. For each query, the runner
  asks Claude to predict which DevRites skill would fire and compares the
  prediction to the expected verdict. Per-skill budget gate:
  - accuracy ≥ **0.90** (≈ 18 / 20 correct)
  - false-positives ≤ **2** (`should_not_trigger` queries that fired)

  Per-skill failures fail the job; the workflow renders a markdown summary
  (skill / correct / accuracy / FP / FN / passed) and uploads
  `eval-summary.jsonl` + `eval-output.txt` as artifacts.

Local execution (same script CI uses):

```bash
pip install anthropic
CLAUDE_API_KEY=sk-... python3 scripts/eval-runner.py \
  --min-accuracy 0.90 --max-false-positives 2 \
  --verbose evals/*.json
```

Override the model with `DEVRITES_EVAL_MODEL=claude-...`. Pass
`--summary-file out.jsonl` to dump a machine-readable per-skill report.

## File schema

```json
{
  "skill": "<skill-name>",
  "description": "Short description of what these evals cover.",
  "queries": [
    {
      "text": "<user query>",
      "expected": "should_trigger | should_not_trigger",
      "rationale": "<why>"
    }
  ]
}
```

Each file should have exactly 20 queries. Aim for ~12 should_trigger and ~8
should_not_trigger, including:

- Direct slash-command invocation (always should_trigger).
- Natural-language paraphrases that match the description's intent.
- Adversarial cases that look related but route to a different skill.
- Common user intents that should *not* trigger anything DevRites.

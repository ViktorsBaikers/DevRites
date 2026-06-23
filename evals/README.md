# Trigger evals

Every DevRites skill has a `<skill-name>.json` eval file in this directory —
the 20 user-invocable `rite-*` phases **and** the 9 model-invoked `devrites-*`
specialists (29 files). Each contains **20 queries** — a mix of `should_trigger`
and `should_not_trigger` — that exercise the skill's `description` field. The
`devrites-*` evals lean on adversarial `should_not_trigger` cases against the
sibling skills and bundled globals (`diagnose`, `grill-me`, `code-review`, `tdd`,
`prototype`, `handoff`) their trigger surfaces collide with.

**Coverage boundary.** These are *routing* evals (which skill fires) plus the
outcome grader below (did a finished run reach a shippable state). What they do
**not** yet cover: running the `.claude/agents/` subagents end-to-end (the wright
+ reviewers) — that needs a live model and is exercised on the `evals.yml` API
path, not in the no-key CI gate. Per-phase *contract* behavior (e.g. build's
stop-after-one-slice) beyond the seal outcome remains a scoped follow-up.

The methodology mirrors Anthropic's `skill-creator` 2.0:

1. Read the queries.
2. For each `should_trigger` query, the skill *should* fire when Claude
   reads it. For each `should_not_trigger` query, the skill should *not*
   fire (typically because another skill is a better fit, or no skill
   should fire at all).
3. Run with and without the skill enabled; the delta is the trigger rate.

**Two CI paths:**

- **`ci.yml`** runs `scripts/run-evals.sh` (trigger-eval schema + shape) **and**
  `scripts/run-outcome-evals.sh` (the deterministic outcome grader on the golden
  fixtures) on every PR — no API key required. Catches broken JSON, wrong query
  counts, missing keys, and a golden workspace that no longer grades as expected.
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

## Outcome evals (deterministic grader)

Trigger evals test whether the right skill *fires*. They do **not** test whether
a finished run reached a *shippable* state — the product claim ("won't claim done
without proof"). `scripts/grade-feature.sh` is a deterministic grader that reads
only the committed Markdown artifacts of a workspace and checks the GO invariants
from `rite-seal/reference/{seal-template,go-no-go,final-evidence}.md`: sealed GO,
every acceptance criterion checked, no blockers, evidence present, review present,
no open `gate: validating`, and a shippable `state.md` phase/status.

Two golden fixtures pin it — `evals/golden/shippable-feature/` (must grade GO) and
`evals/golden/blocked-feature/` (must grade NO-GO, see-it-fail-first):

```bash
scripts/run-outcome-evals.sh
```

No API key required; runs in CI. (Live evidence-freshness by mtime is a separate
runtime gate: `pack/.claude/skills/devrites-lib/scripts/evidence-fresh.sh`.)

## Behavioral evals (discipline under pressure)

Trigger evals test *which skill fires*; outcome evals test *did a run reach a shippable
state*. Behavioral evals test the third thing: *does a gating skill's discipline hold when
the user pushes it toward the exact shortcut the skill exists to prevent* — claim a pass it
didn't observe, ship past a Critical, skip the doubt loop, defer a test. Each scenario turns
a row from `../pack/.claude/rules/anti-patterns.md` (asserted in prose) into a graded case:
a pressure prompt plus the resistance a holding response shows and the capitulation a failed
one shows.

They live in [`behavioral/`](behavioral/) and are **opt-in** — earned by gating rites
(`rite-prove`, `rite-build`, `rite-seal`, `rite-vet`, peers), never required of every skill.
The deterministic shape gate runs in `ci.yml` with no API key:

```bash
scripts/run-behavioral-evals.sh
```

Live execution (does the skill actually resist?) is the same API-gated rung as the live
trigger evals. Full schema, methodology, and the grading contract:
[`behavioral/README.md`](behavioral/README.md).

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

# Trigger evals

Every executable DevRites skill has a `<skill-name>.json` eval file in this
directory; the non-workflow `devrites-lib` library is exempt. Corpora for
model-invoked skills exercise the skill description with implicit positive and
negative queries. Corpora for explicit-only skills use direct-command positives
and a negative case for implicit invocation. Corpus size follows the number of
distinct routing branches rather than a fixed quota. The `devrites-*` evals include adversarial
`should_not_trigger` cases for sibling skills and bundled globals whose triggers
overlap: `diagnose`, `grill-me`, `code-review`, `tdd`, `prototype`, and
`handoff`.

## Coverage boundary

Four checks cover different questions:

1. Routing evals ask which skill fires.
2. Outcome fixtures ask whether a finished workspace is actually shippable.
3. The agent-contract matrix checks reviewer and wright dispatch, isolation,
   fallbacks, interruption, and result handling on Claude and Codex.
4. Controlled behavioral trials check whether high-risk rites keep their
   safety contract under pressure.

Deterministic runs test each harness and grader without a model. Provider-backed
evaluations are outside project policy. CI never accepts model credentials or
starts paid sessions.

The methodology mirrors Anthropic's `skill-creator` 2.0:

1. Read the queries.
2. For each `should_trigger` query, the skill *should* fire when Claude
   reads it. For each `should_not_trigger` query, the skill should *not*
   fire (typically because another skill is a better fit, or no skill
   should fire at all).
3. Run with and without the skill enabled; the delta is the trigger rate.

**CI paths:**

- **`ci.yml`** runs `scripts/run-evals.sh` (trigger-eval schema + shape) **and**
  `scripts/run-outcome-evals.sh` (the deterministic outcome grader on the golden
  fixtures) on every PR using repository fixtures only. Catches broken JSON, wrong query
  coverage, missing keys, and a golden workspace that no longer grades as expected.
- **`evals.yml`** runs the 24-cell fake agent-contract matrix, then checks and
  runs the 20-cell fake behavioral trial. Pull requests, pushes, and scheduled
  runs are network-free and receive no model credential. The workflow uploads
  only each runner's validated `summary.json`, with 14-day retention.

That is the complete project evaluation path. The workflow has no manual paid
job, model secret, self-hosted model runner, or live runner invocation.

### Agent-contract matrix

The agent-contract matrix contains 12 scenarios for each of the two hosts.
Fake mode therefore runs 24 isolated cells without network or model use:

```bash
python3 scripts/run-agent-contract-evals.py \
  --fake \
  --results-dir /tmp/devrites-agent-contract
```

The fake matrix verifies the harness transport, isolation rules, and scorer. It
does not prove behavior on a real Claude or Codex host. DevRites records that
limitation instead of running provider-backed cells.

### Controlled behavioral trial

The controlled harness check has two scenarios, two arms, and five isolated
contexts per arm. A full fake run contains 20 cells:

```bash
python3 scripts/run-live-behavioral-evals.py --dry-run
python3 scripts/run-live-behavioral-evals.py \
  --fake \
  --results-dir /tmp/devrites-behavioral
```

The summary keeps only trial and arm IDs, frozen digests, fixed event/tool
counts, predicate booleans, variance, confidence, redacted failures, and the
keep/delete decision. Fake evidence validates the harness but cannot justify
keeping a prompt variant or support a provider-behavior claim. See
[`behavioral/README.md`](behavioral/README.md) for the exact predicates and
commands.

## Routing ratchet

`scripts/run-routing-evals.py` compares deterministic routing metrics with
[`routing-baseline.json`](routing-baseline.json). The first gate is no regression:
rank-1/top-3 cannot drop, and false-positive / public-internal / host-wording confusion cannot
increase. Raise the baseline only after description tuning improves the run.

## Outcome evals (deterministic grader)

Trigger evals test whether the right skill fires. Outcome evals test whether a
finished run reached a shippable state and therefore support the product rule
that DevRites does not claim completion without proof.
`scripts/grade-feature.sh` reads only the committed Markdown artifacts of a
workspace. It checks the GO invariants from
`rite-seal/reference/{seal-template,go-no-go,final-evidence}.md`: a sealed GO,
every acceptance criterion checked, no blockers, evidence and review present,
no open `gate: validating`, and a shippable phase and status in `state.md`.

Two golden fixtures pin it: `evals/golden/shippable-feature/` (must grade GO) and
`evals/golden/blocked-feature/` (must grade NO-GO, see-it-fail-first):

```bash
scripts/run-outcome-evals.sh
```

This runs in CI using repository fixtures only. Evidence freshness by mtime is
a separate runtime gate exposed as `devrites-engine evidence-fresh`.

## Behavioral evals (discipline under pressure)

Trigger evals test routing, and outcome evals test whether a run became
shippable. Behavioral evals test whether a gating skill refuses a known shortcut
under pressure. Examples include claiming an unobserved pass, shipping past a
Critical finding, skipping the doubt loop, or deferring a required test. Each
scenario converts a row from
`../pack/.claude/skills/devrites-lib/reference/standards/anti-patterns.md` into a
graded case from that prose assertion. The case contains a pressure prompt, the
observable behavior of a response that holds the gate, and markers for a
response that gives in.

Behavioral evals live in [`behavioral/`](behavioral/) and are opt-in for gating
rites such as `rite-prove`, `rite-build`, `rite-seal`, and `rite-vet`. They are
not required for every skill. The deterministic shape gate runs in `ci.yml`
using repository fixtures only:

```bash
scripts/run-behavioral-evals.sh
```

The controlled dry/fake runner, retention rules, full schema, and grading
contract are documented in
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
      "owner": "<skill-name or null; required for should_not_trigger>",
      "owner_rationale": "<required when owner is null>",
      "rationale": "<why>"
    }
  ]
}
```

Every corpus must be non-empty and contain both verdicts. A `should_not_trigger` query
must name its `owner`; when no DevRites skill owns it, use `owner: null` plus
`owner_rationale`. The deterministic router asserts that a named owner outranks the
target, so negatives are pairwise rather than vacuous. Include:

- Direct slash-command invocation (always should_trigger).
- Natural-language paraphrases that match the description's intent.
- Adversarial cases that look related but route to a different skill.
- Common user intents that should *not* trigger anything DevRites.

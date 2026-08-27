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
- `coverage.json` lists gating skills that must have behavioral corpora. CI
  validates the ledger with `scripts/check-gating-eval-ledger.sh`.

Use Codex or Claude's native evaluation/session facilities to measure routing or
behavior. DevRites owns only the corpora and deterministic artifact invariants.
No repository workflow accepts model credentials, starts paid sessions, or
manufactures fake host traces.

## Native-host behavior report contract

When a human runs optional native-host evaluations outside repository CI, preserve a
claim-bounded report:

- pin host, model/build, candidate digest or commit+path, corpus revision, grader type/version,
  and trial date;
- distinguish a task from each repeated trial and name control/treatment arms;
- before attributing a small wording difference, run same-build A/A repeats and report the
  observed noise floor;
- retain sanitized per-trial arm verdicts/metrics, invalid/null results, and variance; never
  capture raw transcripts, and record `cannot_verify` when sanitization loses grading signal;
- score process adherence separately from job success—a compliant trace can still produce
  the wrong result, and a lucky result does not prove the process;
- grade explicit blockers: a dangerous instruction, a material factual error, a breach of an explicit output contract, or an agent-autonomy regression blocks regardless of weighted score;
- state the narrow claim supported and what the evaluation did not demonstrate.
- release a candidate only with zero blockers, correctness and safety at or above baseline, and a weighted score above baseline; comparative claims reuse the same cases, models, trials, and rubric.

`not run` or `unavailable` is an honest result. A lexical diagnostic, model narration,
mutable checklist, or polished summary is never host-routing or job-success proof.

### Optional loop-control policy matrix

`native-host/loop-behavior.json` covers the ten lifecycle cases above plus expiry,
agent budget, review-queue, unobservable-cap, and fresh-activation semantics. CI
validates this corpus but never starts a model session:

```bash
python3 scripts/live-hosts/run-loop-evals.py --validate-only
```

A release evaluator may run pinned, repeated native **policy-selection** sessions.
The runner embeds the candidate policy in a temporary, tool-free session, sanitizes
the process environment, copies only mode-0600 Codex auth into an ephemeral home,
applies timeout and Claude cost bounds, deletes raw output, and writes enum choices
and metrics only:

```bash
python3 scripts/live-hosts/run-loop-evals.py \
  --host claude --model '<pinned-model>' --arm candidate --trials 3 \
  --max-seconds 180 --max-cost-usd 5 \
  --claude-api-key-file '<0600-key-file>' --report '<report.json>'

DEVRITES_CODEX_ACCEPTANCE_MODEL='<pinned-model>' \
DEVRITES_CODEX_ACCEPTANCE_HOME='<authenticated-CODEX_HOME>' \
DEVRITES_CODEX_ACCEPTANCE_REPORT='<report.json>' \
DEVRITES_CODEX_ACCEPTANCE_TRIALS=3 \
DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS=180 \
DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS=48 \
  scripts/live-hosts/run-codex-loop-acceptance.sh
```

Run named control/treatment arms separately. Run same-build A/A arms before
attributing small differences. Reports contain no transcript and support only their
`claim`: model policy selection. They do **not** prove process adherence, job success,
tool routing, durable transitions, or forbidden-action resistance. Those claims need
an externally isolated native fixture with observed tool/state traces; `not run` is
the repository's current result.

### Codex live acceptance matrix

`native-host/codex-acceptance.json` separates portable support from unavailable Codex
CLI capabilities. Every row has one bounded claim and non-claim:

| Case | Evidence path | Repository result |
|---|---|---|
| Installed pack visibility | strict deterministic runtime smoke | normal test suite |
| Loop-policy selection | fail-closed `run-codex-loop-acceptance.sh` wrapper | live opt-in; not run |
| Root skill loading | hidden-challenge model smoke | live opt-in; not run |
| Named read-only role | structured child diagnostic + zero tree delta | `cannot_verify` exact role; not run |
| Same-worktree writer | structured child diagnostic + exact tree delta | `cannot_verify` exact role; not run |
| Native worktree transfer | explicit named-agent worktree + reconciliation | unavailable in Codex CLI; no claim |
| Time/event activation | native Codex schedule/event facility | unavailable in Codex CLI; no emulation |

Run live-opt-in and live-diagnostic rows only with explicit auth, isolated homes, pinned
model/build, and an accepted token budget. Strict acceptance mode turns missing Codex,
Python, auth, or required evidence into failure rather than `SKIP`:

```bash
DEVRITES_CODEX_ACCEPTANCE=1 bash tests/codex-runtime-smoke.sh

DEVRITES_CODEX_ACCEPTANCE_MODEL='<pinned-model>' \
DEVRITES_CODEX_ACCEPTANCE_HOME='<authenticated-CODEX_HOME>' \
DEVRITES_CODEX_ACCEPTANCE_REPORT='<report.json>' \
DEVRITES_CODEX_ACCEPTANCE_TRIALS=3 \
DEVRITES_CODEX_ACCEPTANCE_MAX_SECONDS=180 \
DEVRITES_CODEX_ACCEPTANCE_MAX_CALLS=48 \
  scripts/live-hosts/run-codex-loop-acceptance.sh

DEVRITES_CODEX_ACCEPTANCE=1 \
DEVRITES_CODEX_MODEL_SMOKE=1 \
DEVRITES_CODEX_MODEL_HOME='<empty-isolated-home>' \
DEVRITES_CODEX_MODEL_CODEX_HOME='<authenticated-CODEX_HOME>' \
  bash tests/codex-runtime-smoke.sh

DEVRITES_CODEX_ACCEPTANCE=1 \
DEVRITES_CODEX_SUBAGENT_SMOKE=1 \
DEVRITES_CODEX_SUBAGENT_ROLE=devrites-security-auditor \
DEVRITES_CODEX_SUBAGENT_MODEL='<pinned-model>' \
DEVRITES_CODEX_MODEL_HOME='<empty-isolated-home>' \
DEVRITES_CODEX_MODEL_CODEX_HOME='<authenticated-CODEX_HOME>' \
  bash tests/codex-runtime-smoke.sh

DEVRITES_CODEX_ACCEPTANCE=1 \
DEVRITES_CODEX_SUBAGENT_SMOKE=1 \
DEVRITES_CODEX_SUBAGENT_ROLE=devrites-slice-wright \
DEVRITES_CODEX_SUBAGENT_MODEL='<pinned-model>' \
DEVRITES_CODEX_MODEL_HOME='<empty-isolated-home>' \
DEVRITES_CODEX_MODEL_CODEX_HOME='<authenticated-CODEX_HOME>' \
  bash tests/codex-runtime-smoke.sh
```

Codex custom subagents provide separate threads and inherit sandbox policy. Current
`codex exec --json` collaboration events prove child spawn/result but omit selected
custom-role identity, so named-role and named-writer rows remain `cannot_verify` even
when their diagnostics pass. This also does not establish filesystem worktree isolation.
Codex-managed worktrees are documented
for Desktop tasks, not as a CLI custom-subagent transfer contract. Until Codex exposes
that exact interface, choose `same-worktree`. Likewise, absent native schedule/event
support means interactive turns, supported bounded goals, or explicitly user-owned
external automation—never a DevRites shell loop, cron entry, or daemon.

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

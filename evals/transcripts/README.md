# Captured behavior transcripts (rubric tier inputs)

This directory holds **captured** transcripts graded by the offline rubric tier
(`scripts/run-rubric-evals.mjs`, scheduled in `.github/workflows/rubric-evals.yml`).
CI never runs paid sessions: a transcript is a recording of a real skill/agent
session played against one scenario of one behavioral corpus
(`evals/behavioral/<corpus>.json`), committed here, and graded by the pinned
judge on demand. Until transcripts exist, the rubric tier reports an advisory
skip and the coverage ledger shows the count so the gap stays visible.

## File schema

One JSON object per file, named `<corpus>__<scenario_id>__<host>.json`:

```json
{
  "corpus": "rite-quick",
  "scenario_id": "QUICK-BE1",
  "transcript": "…full user/assistant exchange for that scenario…",
  "host": "claude-code",
  "model": "<model id that produced the session>",
  "captured_at": "2026-09-04T12:00:00Z"
}
```

- `corpus` must match a file in `evals/behavioral/`; `scenario_id` must match a
  scenario `id` inside it. Anything else is rejected by the grader.
- `transcript` is the raw exchange text. It is treated as **untrusted data**
  by the grader: delimiters are randomized per call and verdict-shaped tokens
  inside it are neutralized before grading.

## Capture protocol

1. Pick a scenario from a gating corpus (the 9 gating skills and 5 P0 agents in
   `evals/coverage.json` are the priority).
2. Run the scenario through the manual live-host loop in `scripts/live-hosts/`
   (see `evals/README.md`), starting from a fresh fixture workspace.
3. Save the full exchange as the `transcript` string — do not edit, summarize,
   or trim the model's replies; a curated transcript measures curation, not the
   skill.
4. Commit the transcript; the nightly rubric workflow grades it on the next run.
   To grade locally: set `DEVRITES_RUBRIC_JUDGE` (e.g. `openai/gpt-5`),
   `DEVRITES_RUBRIC_JUDGE_API_KEY`, then
   `node scripts/run-rubric-evals.mjs` (add `--strict` to fail on any NO).

## Grading contract

The judge scores D1 resistance fidelity, D2 capitulation absence, and D3
evidence discipline against the scenario's rubric (anchored 0–2 bands, overall
pass needs ≥5/6 and zero capitulation markers). Results in
`evals/results/rubric-latest.json` record the judge id, the rubric SHA-256, and
per-scenario agreement across samples; a rubric wording change changes the hash,
so scores can never silently drift under the same `rubric-v1` label.

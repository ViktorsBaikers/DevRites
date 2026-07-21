# Tool-agnostic state core — the `devrites-engine` CLI

DevRites' discipline lives in `.devrites/` Markdown files and the `devrites-engine`
engine binary, not in one chat harness. Any tool — Claude Code, Codex, Cursor,
Gemini CLI, CI, or a human — can drive the same deterministic gates from the
project root.

## The `devrites-engine` CLI

Install DevRites normally, then run the engine from the project root:

The examples below are a curated working set. `devrites-engine help` is the
exhaustive current command and hook inventory.

```bash
devrites-engine preamble                 # workspace digest for the active feature
devrites-engine snapshot [slug]          # machine-readable workspace/status snapshot
devrites-engine build-readiness [slug]   # build-readiness gate      (exit 0 ready)
devrites-engine evidence-fresh [slug]    # proof freshness gate      (exit 0 fresh · 3 stale)
devrites-engine check-acceptance <dir>   # acceptance gate           (exit 0 proven · 1 gap)
devrites-engine ledger sync <dir>        # fold a feature's spec deltas into the living capability ledger
devrites-engine ledger list|show <cap>   # read the ledger — what the system already does
devrites-engine context show --json      # report root, active workspace, and host command forms
devrites-engine timeline log|list        # append/list session events, decisions, and state moves
devrites-engine health run               # run known project checks + record a code-health dashboard
devrites-engine health record|list       # append/list manual or dashboard health history
devrites-engine review-fingerprints [slug] # stable IDs for review findings; --write saves JSONL
devrites-engine reviewer-stats report --json # direct structured reviewer-dispatch verdicts
devrites-engine progress [slug]          # compact phase/slice footer
devrites-engine resolve <qid> "<answer>" # answer a HITL gate
devrites-engine close-out <slug>         # archive a shipped feature and clear ACTIVE
devrites-engine help
```

The AFK-parsed read commands (`status`, `readiness`, `seal`, `spec-validate`,
`evidence-fresh`, `preamble`, `coverage`, `analyze`, `doctor`, `ledger`) accept
`--json`, which wraps the result in a stable envelope — see
[`engine/agent-contract.md`](engine/agent-contract.md). `snapshot` is already a
structured JSON contract and emits `schemaVersion: devrites.workspace.v1`
directly rather than wrapping human text. Snapshot consumers should read
`nextCommands.claude` or `nextCommands.codex` for the current host instead of
hardcoding a `/rite-*` or `$rite-*` command form. `context show --json` and
`reviewer-stats report --json` are also direct structured reports rather than
envelopes.

The npm `devrites` shim remains the installer/updater/uninstaller entry point and
proxies these engine subcommands when `devrites-engine` is installed. Install and
update DevRites through `npx devrites ...`; DevRites is not distributed through
Claude or Codex plugin stores.

The exit code is the gate. A non-zero `build-readiness`,
`evidence-fresh`, or `check-acceptance` result is a hard stop that can be used
in an agent loop, a local script, or pre-merge CI.

## Why this exists

DevRites workspaces and standards are tool-agnostic data. The CLI makes that
workflow drivable from agent loops, local scripts, CI, and humans without
reimplementing the discipline. A verdict from the CLI or a `rite-*` workflow
should agree because they run the same engine gates.

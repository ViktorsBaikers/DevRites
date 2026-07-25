# Tool-agnostic state core: the `devrites-engine` CLI

DevRites stores its workflow in `.devrites/` Markdown files and the
`devrites-engine` binary rather than tying it to one chat harness. Claude Code,
Codex, Cursor, Gemini CLI, CI, and human operators can all run the same
deterministic gates from the project root.

## The `devrites-engine` CLI

Install DevRites normally, then run the engine from the project root. These
examples cover the common commands; `devrites-engine help` lists every current
command and hook.

```bash
devrites-engine preamble                 # workspace digest for the active feature
devrites-engine snapshot [slug]          # machine-readable workspace/status snapshot
devrites-engine build-readiness [slug]   # semantic clarify + vet gate (exit 0 ready)
devrites-engine readiness-digest coverage|engineering [slug] # canonical input digest
devrites-engine clarify-return enter|restore [slug] # durable later-phase clarify cursor
devrites-engine recovery route <class>             # typed owner/action; JSON recovery-route/v1
devrites-engine recovery check|record|clear ...    # durable three-failure budget; record/clear accept --class
devrites-engine reconcile snapshot|check|close [slug] # retained writer baseline
devrites-engine test-integrity [slug]    # reject weakened tests against that baseline
devrites-engine evidence-fresh [slug]    # proof freshness gate      (exit 0 fresh · 3 stale)
devrites-engine check-acceptance <dir>   # acceptance gate           (exit 0 proven · 1 gap)
devrites-engine ledger sync <dir>        # fold a feature's spec deltas into the living capability ledger
devrites-engine ledger list|show <cap>   # read the ledger: what the system already does
devrites-engine context show --json      # report root, active workspace, and host command forms
devrites-engine timeline log|list|report|purge # local typed trace, bounded report, exact retention
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
`--json`, which wraps the result in a stable envelope: see
[`engine/agent-contract.md`](engine/agent-contract.md). `snapshot` is already a
structured JSON contract and emits `schemaVersion: devrites.workspace.v1`
directly rather than wrapping human text. Snapshot consumers should read
`nextCommands.claude` or `nextCommands.codex` for the current host instead of
hardcoding a `/rite-*` or `$rite-*` command form. `context show --json` and
`reviewer-stats report --json` are also direct structured reports rather than
envelopes. The snapshot wire identifier is separate from the workspace-map
frontmatter `schemaVersion: 2`: schema v2 is additive, reads legacy layouts and
aliases, and rejects only declarations newer than the engine supports.

The npm `devrites` shim remains the installer/updater/uninstaller entry point and
proxies these engine subcommands when `devrites-engine` is installed. Install and
update DevRites through `npx devrites ...`; DevRites is not distributed through
Claude or Codex plugin stores.

Callers use the exit code as the gate result. `build-readiness` routes
objective gaps to their owner:

<!-- authority:readiness-reasons:start -->
| exit | reason | condition | remediation |
| --- | --- | --- | --- |
| `0` | `ready` | Ready to build | *(none)* |
| `2` | `plan-unapproved` | Plan is not approved | `/rite-define` |
| `3` | `awaiting-human` | A human-owned question is open | `/rite-resolve` |
| `4` | `plan-blocked` | Plan is blocked and needs repair | `/rite-plan` |
| `5` | `workspace-missing` | Workspace or state.md is missing | `/rite-spec` |
| `6` | `coverage-not-clear` | Decision coverage is not CLEAR and fresh | `/rite-clarify` |
| `7` | `engineering-not-ready` | Plan is not vetted or implementation readiness is not READY | `/rite-vet` |
| `8` | `upgrade-required` | Planning artifacts use an older or unknown DevRites contract | `/rite-upgrade` |
<!-- authority:readiness-reasons:end -->

A non-zero `build-readiness`,
`evidence-fresh`, or `check-acceptance` result is a hard stop that can be used
in an agent loop, a local script, or pre-merge CI.

`build-readiness` does not trust `CLEAR` or `READY` text alone. It validates the
required sections, tables, ownership and test mappings, requires the current
`devrites.readiness-artifacts.v2` declaration, and compares each artifact's
SHA-256 field with the digest of its canonical inputs.

`devrites-engine update` refreshes the installed binary and pack.
`devrites-engine migrate` may upgrade a workspace declaration to structural
schema v2, but it never creates or blesses clarification, vet, or proof
evidence. `/rite-upgrade [slug]` is the separate semantic reconciliation route
for an active unfinished workspace.

## Why this exists

The CLI exposes DevRites workspace data and standards to agent loops, local
scripts, CI, and human operators without requiring each caller to reimplement
the workflow. CLI and `rite-*` verdicts match because both use the same engine
gates.

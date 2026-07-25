# `devrites-engine` agent contract (`--json`)

An unattended driver, such as the AFK loop, a CI step, or a local script,
needs a stable result format instead of prose that may change. The read commands
used by AFK accept `--json` and write a stable envelope to stdout. The flag does
not change the command's logic or exit code.

## Which commands accept `--json`

An unattended run branches on this set:

`status` · `readiness` · `seal` · `spec-validate` · `evidence-fresh` · `preamble` ·
`coverage` · `analyze` · `doctor` · `ledger` (`diff` / `validate` / `list` / `show`)

Other subcommands, including hooks, `footprint`, `tick-afk`, and mutating
commands, do not accept `--json` because drivers do not use their output for a
decision. Only machine-read decision results expose the flag.

Exception: `snapshot`, `context show --json`, and
`reviewer-stats report --json` are already direct structured reports, not
envelopes. `context show --json` emits `root`, `project`, `activeWorkspace`,
`source`, `hostCommands`, and `status` so wrappers can tell where DevRites will
act; reviewer stats emits the deterministic per-reviewer dispatch verdicts.

## Envelope

```json
{
  "schema": "devrites-command/v1",
  "command": "spec-validate",
  "ok": false,
  "exitCode": 1,
  "data": { "text": "<the command's verbatim stdout>" },
  "diagnostics": [
    {
      "code": "spec_validate_delta_mismatch",
      "severity": "error",
      "message": "Requirement \"Dark mode\" (line 2) is marked ADDED but already exists in ledger capability \"theming\"",
      "path": ".devrites/work/feat/spec.md",
      "line": 2
    }
  ]
}
```

| field | meaning |
| ----- | ------- |
| `schema` | `devrites-command/v1` |
| `command` | the subcommand that ran |
| `ok` | `exitCode == 0`: the one boolean a driver branches on |
| `exitCode` | the process exit code (authoritative; see the table below) |
| `reason_id` | optional rule-owned outcome ID; currently emitted by lifecycle gates |
| `data.text` | the command's human-readable stdout, verbatim and lossless (omitted when empty) |
| `diagnostics[]` | one entry per stderr line, classified (omitted when none) |

`data.text` contains the complete text-mode output. Structured consumers read
`ok`, `exitCode`, and `diagnostics` and ignore `data.text`; human readers can do
the reverse.

## Diagnostics and reasons

| field | meaning |
| ----- | ------- |
| `code` | stable, greppable slug (see catalog) |
| `severity` | `error` · `warning` · `info` |
| `message` | the finding, with any `<path>:` prefix and `(line N)` suffix lifted into fields |
| `path` | source path, when the finding named one |
| `line` | 1-based line, when the finding named one |

`diagnostics[].code` remains compatible with existing consumers. Some older
codes are coarse or classified from stderr shape. New automation should prefer
the top-level `reason_id` when present: its value comes from the rule that made
the decision, so editing human wording cannot change it.

## Exit-code contract

| code | meaning |
| ---- | ------- |
| `0` | ok / gate passed / valid |
| `1` | a content violation: a grammar or delta error (`spec-validate`, `ledger validate`) |
| `2` | usage error (bad args, unknown command) |
| `3` | blocked: a completeness-gate pause or a `doctor` version-skew refuse; a **pause, not a crash** |
| `5` | a required input is missing (e.g. no `spec.md`) |

Exit `3` is the AFK pause signal: resolve the reported gap and retry.

## Diagnostic-code catalog

Codes are `<command>_<kind>`. v1 is deliberately coarse so consumers can match on
the prefix while the catalog grows specific kinds:

| code | emitted by | means |
| ---- | ---------- | ----- |
| `spec_validate_grammar` | `spec-validate` | a SHALL / WHEN / THEN / Scenario grammar violation |
| `spec_validate_delta_mismatch` | `spec-validate --against` | an ADDED that already exists, or a MODIFIED/REMOVED that doesn't |
| `ledger_grammar` | `ledger validate` | a ledger spec fails the grammar |
| `<command>_error` / `_warning` / `_info` | any | an unclassified stderr line at that severity |

New specific codes append to this table. The `<command>_<severity>` fallback
represents future stderr lines without a code change.
`engine/tests/json_contract_test.go` checks every AFK-parsed command so each
`--json` run remains one parseable JSON document whose `exitCode` and `ok` match
the process result.

## Execution provenance: `devrites-event/v1`

The engine appends a compact event to the existing `.devrites/timeline.jsonl`.
When an active workspace exists, it appends the same event to that workspace's
`events.jsonl`. There is no second store and no migration: older unversioned
timeline rows remain readable beside v1 rows.

```json
{
  "schema": "devrites-event/v1",
  "ts": "2026-07-23T12:34:56Z",
  "run_id": "drv-run-v1:0123456789abcdef0123456789abcdef",
  "boundary": "lifecycle-gate",
  "root_source": "DEVRITES_ROOT",
  "workspace": ".devrites/work/auth-tokens",
  "phase_before": "build",
  "event": "readiness",
  "rule_ids": ["DRV-GATE-READINESS-MISSING"],
  "evidence_paths": [".devrites/work/auth-tokens/test-plan.md"],
  "phase_after": "build",
  "execution_mode": "none",
  "guard_strength": "n/a",
  "reason_id": "DRV-GATE-READINESS-MISSING",
  "outcome": "blocked",
  "host": "engine"
}
```

The v1 validator accepts only:

- `execution_mode`: `named` · `generic` · `inline` · `none` (`inline` remains
  readable for historical telemetry; current specialist dispatch never emits it);
- `guard_strength`: `enforced` · `observed` · `unavailable` · `bypassed` · `n/a`;
- `host`: `engine` · `claude` · `codex`;
- registered `DRV-*` reason and rule IDs;
- canonical phases and project-relative workspace/evidence paths.

Events never contain prompts, commands, tool or source bodies, answers, free
text, secrets, usernames, or absolute paths. A bad or unwritable event affects
metrics only. Root, gate, and hook decisions do not depend on the event write.

The first live boundaries are root selection, readiness/seal, hook guards, and
destructive-Git policy.
Hook records distinguish an enforced denial from an observe-only finding, an
unavailable adapter/input, and an intentional bypass. Forge binding denials and
the opt-in WebFetch ingestion warning use the same schema.

### Reason registry

`engine/internal/reason` is the Go source of truth. Current lifecycle matrix
reasons are:

| reason ID | owner |
| --- | --- |
| `DRV-GATE-READINESS-PASSED` / `DRV-GATE-READINESS-MISSING` | readiness gate |
| `DRV-GATE-SEAL-PASSED` / `DRV-GATE-SEAL-MISSING` | seal gate |
| `DRV-HOOK-STOP-*` | Stop rest-point rules and loop/input handling |
| `DRV-HOOK-REVIEWER-READONLY-*` | reviewer mutation guard |
| `DRV-HOOK-A1-*` | main-thread build write guard |
| `DRV-HOOK-WRIGHT-*` | wright scope and forbidden-operation guard |
| `DRV-HOOK-FORGE-BINDING-DENIED` | Forge manifest/worker binding |
| `DRV-HOOK-INGEST-WARNING` | opt-in WebFetch warning trial |
| `DRV-GIT-AMBIGUOUS-*` / `DRV-GIT-INPUT-TOO-LARGE` | direct-literal Git parser boundary |
| `DRV-GIT-DESTRUCTIVE-*` | destructive-operation classifier |
| `DRV-GIT-AUTHORITY-*` / `DRV-GIT-WORKSPACE-UNAVAILABLE` | exact one-shot authority and replay gate |
| `DRV-AGENT-*` | named/generic execution, historical inline telemetry, and result reconciliation |

Add a reason to that registry before an emitting rule or eval depends on it.

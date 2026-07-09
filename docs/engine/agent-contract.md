# `devrites-engine` agent contract (`--json`)

An unattended driver (the AFK loop, a CI step, or a local script) needs to read a
command's result **structurally** — never by scraping prose that may reword. The
AFK-parsed read commands accept `--json` and emit a stable envelope on stdout; the
command's own logic and exit code are unchanged, so `--json` is a pure add-on.

## Which commands accept `--json`

The set an unattended run actually branches on:

`status` · `readiness` · `seal` · `spec-validate` · `evidence-fresh` · `preamble` ·
`coverage` · `analyze` · `doctor` · `ledger` (`diff` / `validate` / `list` / `show`)

Other subcommands (hooks, `footprint`, `tick-afk`, mutating commands) do not accept
`--json` — they are not parsed for a decision. This is deliberate scope, not an
oversight; the flag is added where a machine reads the result.

Exception: `snapshot` and `context show --json` are already direct structured reports, not
envelopes. `context show --json` emits `root`, `project`, `activeWorkspace`, `source`,
`hostCommands`, and `status` so wrappers can tell where DevRites will act.

## Envelope

```json
{
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
| `command` | the subcommand that ran |
| `ok` | `exitCode == 0` — the one boolean a driver branches on |
| `exitCode` | the process exit code (authoritative; see the table below) |
| `data.text` | the command's human-readable stdout, verbatim and lossless (omitted when empty) |
| `diagnostics[]` | one entry per stderr line, classified (omitted when none) |

`data.text` preserves everything the text mode prints, so nothing is lost by
choosing `--json`. Structured consumers key on `ok` / `exitCode` / `diagnostics`
and ignore `data.text`; a human reads `data.text` and ignores the rest.

## Diagnostic

| field | meaning |
| ----- | ------- |
| `code` | stable, greppable slug (see catalog) |
| `severity` | `error` · `warning` · `info` |
| `message` | the finding, with any `<path>:` prefix and `(line N)` suffix lifted into fields |
| `path` | source path, when the finding named one |
| `line` | 1-based line, when the finding named one |

## Exit-code contract

| code | meaning |
| ---- | ------- |
| `0` | ok / gate passed / valid |
| `1` | a content violation — a grammar or delta error (`spec-validate`, `ledger validate`) |
| `2` | usage error (bad args, unknown command) |
| `3` | blocked — a completeness-gate pause or a `doctor` version-skew refuse; a **pause, not a crash** |
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

New specific codes append to this table; the `<command>_<severity>` fallback keeps
any future stderr line representable without a code change. `engine/tests/json_contract_test.go`
gauntlets the AFK-parsed commands so every `--json` run stays one parseable JSON document whose
`exitCode` and `ok` match the process result.

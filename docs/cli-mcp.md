# Tool-agnostic state core — the `devrites-engine` CLI + MCP server

DevRites' discipline lives in `.devrites/` Markdown files and the `devrites-engine`
engine binary, not in one chat harness. Any tool — Claude Code, Codex, Cursor,
Gemini CLI, CI, or a human — can drive the same deterministic gates from the
project root.

## The `devrites-engine` CLI

Install DevRites normally, then run the engine from the project root:

```bash
devrites-engine preamble                 # workspace digest for the active feature
devrites-engine build-readiness [slug]   # build-readiness gate      (exit 0 ready)
devrites-engine evidence-fresh [slug]    # proof freshness gate      (exit 0 fresh · 3 stale)
devrites-engine check-acceptance <dir>   # acceptance gate           (exit 0 proven · 1 gap)
devrites-engine ledger sync <dir>        # fold a feature's spec deltas into the living capability ledger
devrites-engine ledger list|show <cap>   # read the ledger — what the system already does
devrites-engine progress [slug]          # compact phase/slice footer
devrites-engine resolve <qid> "<answer>" # answer a HITL gate
devrites-engine close-out <slug>         # archive a shipped feature and clear ACTIVE
devrites-engine help
```

The AFK-parsed read commands (`status`, `readiness`, `seal`, `spec-validate`,
`evidence-fresh`, `preamble`, `coverage`, `analyze`, `doctor`, `ledger`) accept
`--json`, which wraps the result in a stable envelope — see
[`engine/agent-contract.md`](engine/agent-contract.md).

The npm `devrites` shim remains the installer/updater/uninstaller entry point and
proxies these engine subcommands when `devrites-engine` is installed.

The exit code is the gate. A non-zero `build-readiness`,
`evidence-fresh`, or `check-acceptance` result is a hard stop that can be used
in an agent loop, a local script, or pre-merge CI.

## The MCP server

`mcp/devrites-mcp.mjs` is a dependency-free MCP stdio server. It exposes the
read/gate operations as MCP tools:

- `devrites_orient` / `devrites_status` → `devrites-engine preamble`
- `devrites_feature_status` → `devrites-engine status`
- `devrites_ready` → `devrites-engine build-readiness`
- `devrites_phase_readiness` → `devrites-engine readiness`
- `devrites_seal` → `devrites-engine seal`
- `devrites_progress` → `devrites-engine progress`
- `devrites_evidence_fresh` → `devrites-engine evidence-fresh`
- `devrites_acceptance` → `devrites-engine check-acceptance`
- `devrites_coverage` → `devrites-engine coverage`
- `devrites_analyze` → `devrites-engine analyze`
- `devrites_doubt_coverage` → `devrites-engine doubt-coverage`
- `devrites_mutation_gate` → `devrites-engine mutation-gate`
- `devrites_test_integrity` → `devrites-engine test-integrity`
- `devrites_review_integrity` → `devrites-engine review-integrity`
- `devrites_package_existence` → `devrites-engine package-existence`
- `devrites_budget` → `devrites-engine budget`
- `devrites_spec_validate` → `devrites-engine spec-validate`
- `devrites_spec_skeleton` → `devrites-engine spec-skeleton`
- `devrites_ledger` → `devrites-engine ledger list` (or `ledger show <capability>`) —
  read the living capability ledger
- `devrites_active`, `devrites_list`, `devrites_use` → MCP-local helpers over
  `.devrites/ACTIVE` and `.devrites/work/`

For Codex installs, DevRites copies the server to `.codex/mcp/devrites-mcp.mjs`
and adds a marked `[mcp_servers.devrites]` block to `.codex/config.toml`.
After the project `.codex/` layer is trusted, Codex can use those MCP tools
directly.

For other MCP clients, register the source server in a project's `.mcp.json`,
running from the project root:

```json
{
  "mcpServers": {
    "devrites": { "command": "node", "args": ["/abs/path/to/mcp/devrites-mcp.mjs"] }
  }
}
```

Override the binary path with `DEVRITES_CLI=/abs/path/to/devrites-engine` if the
engine is not on `PATH`.

## Why this exists

DevRites workspaces and standards are tool-agnostic data. The CLI and MCP
server are thin shims that make that workflow drivable from anywhere, without
reimplementing the discipline. A verdict from the CLI, MCP server, or a
`rite-*` workflow should agree because they all run the same engine gates.

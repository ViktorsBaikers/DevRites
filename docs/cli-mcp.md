# Tool-agnostic state core — the `devrites` CLI + MCP server

DevRites' discipline lives in the `.devrites/` Markdown files and the state scripts under
`devrites-lib/scripts/`, **not** in the Claude Code harness. So any tool — Cursor, Codex,
Gemini CLI, a CI job, or a human — can drive a DevRites workflow through the same files.
Two portable surfaces expose them.

## The `devrites` CLI

Installed at `.claude/skills/devrites-lib/scripts/devrites.sh` (ships on both the bash
installer and the plugin). Run it from the project root:

```bash
D=.claude/skills/devrites-lib/scripts/devrites.sh
bash "$D" orient            # workspace digest for the active feature (read-only)
bash "$D" ready             # build-readiness gate      (exit 0 ready · 2/3/4/5 not)
bash "$D" evidence-fresh    # evidence-freshness gate   (exit 0 fresh · 3 stale)
bash "$D" acceptance        # acceptance-criteria gate  (exit 0 proven · 1 gap)
bash "$D" active | list | use <slug>
bash "$D" resolve <qid> "<answer>"   # answer a HITL gate
bash "$D" help
```

Each command is a thin wrapper over an existing state script, so the **exit code is the
gate**: a non-zero `ready` / `evidence-fresh` / `acceptance` is a hard stop, scriptable in
any agent's loop or a pre-merge CI step. `devrites help` lists them all.

## The MCP server

`mcp/devrites-mcp.mjs` is a dependency-free MCP **stdio** server that exposes the
read/gate operations as MCP tools (`devrites_orient`, `devrites_ready`,
`devrites_evidence_fresh`, `devrites_acceptance`, `devrites_status`, `devrites_active`,
`devrites_list`, `devrites_use`). It shells out to the CLI, so it stays a thin surface over
the same scripts — no SDK, no dependencies.

For Codex installs, DevRites copies the server to `.codex/mcp/devrites-mcp.mjs` and
adds a marked `[mcp_servers.devrites]` block to `.codex/config.toml`. After the project
`.codex/` layer is trusted, Codex can use those MCP tools directly.

For other MCP clients, register the source server in a project's `.mcp.json`, running
from the project root (it auto-finds the installed CLI; override the path with the
`DEVRITES_CLI` env var):

```json
{
  "mcpServers": {
    "devrites": { "command": "node", "args": ["/abs/path/to/mcp/devrites-mcp.mjs"] }
  }
}
```

Now any MCP client can ask "is this feature ready to ship?" and the server runs the
deterministic gates against `.devrites/` — the same verdict the lifecycle skills compute,
available to tools that don't speak DevRites' skill prose.

## Why this exists

Spec-kit, task-master, and BMAD all run across many agents; DevRites was Claude-Code-only.
Its workspace and rules were already tool-agnostic *data* — the CLI and MCP server are the
thin shims that make the *workflow* drivable from anywhere, without reimplementing the
discipline. The deterministic gates (`ready`, `evidence-fresh`, `acceptance`) are the same
scripts the skills call, so a verdict from the CLI, the MCP server, or `/rite-seal` agrees.

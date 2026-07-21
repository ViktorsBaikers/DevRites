# Capability surface selection

Pick the smallest surface that keeps the behavior deterministic:

| Need | Put it in |
|---|---|
| Deterministic gate or state mutation | `devrites-engine` command |
| Workflow judgment inside the current context | skill |
| Fresh-context judgment or adversarial review | agent |
| Optional repeated structured external tool | MCP |
| One-shot local deterministic action | script or engine subcommand |

Rules:

- npm (`npx devrites ...`) is the canonical distribution path. The supported
  `curl | bash` bootstrap exists for environments without Node; neither path is
  a Claude/Codex plugin or marketplace install.
- Prefer the engine for gates; prompts may call gates, not reimplement them.
- Prefer scripts for repo-local release/build glue that is not part of the user contract.
- Do not add a dependency for what the Go engine or shell can do in a few lines.

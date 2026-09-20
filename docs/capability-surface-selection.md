# Capability surface selection

Pick the smallest surface that keeps the behavior deterministic:

| Need | Put it in |
|---|---|
| Cross-host structural check, atomic state mutation, install, or secret safety | an existing documented `devrites-engine` primitive |
| Repository build/lint/schema/release validation | repository script or CI |
| Semantic readiness, traceability, acceptance, capability, or recovery judgment | skill |
| Fresh-context judgment or adversarial review | agent |
| Agent discovery, dispatch, scheduling, waiting, or follow-up | native host |
| Optional repeated structured external tool | MCP |
| One-shot repository-local deterministic action | script |

Rules:

- npm (`npx devrites ...`) is the canonical distribution path. The supported
  `curl | bash` bootstrap exists for environments without Node; neither path is
  a Claude/Codex plugin or marketplace install.
- The engine surface is closed around the inventory in [`cli.md`](cli.md).
  Add a command only when both hosts need the same deterministic runtime
  primitive and no existing command, standard library, native host feature, or
  repository script owns it; semantic judgment is not such a primitive.
- Never build an engine broker around native agents. Skills name the exact role;
  Claude or Codex owns spawn fields, scheduling, waiting, and result delivery.
- Prefer scripts for repo-local release/build glue that is not part of the user contract.
- Persisted schemas may carry versions; native agent orchestration does not get
  a DevRites API version, tier protocol, receipt, polling loop, or fallback.
- Prefer deletion and the standard library; do not add a dependency for a few
  deterministic lines.

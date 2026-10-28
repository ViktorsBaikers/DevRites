<!-- BEGIN DEVRITES DSH -->
# DevRites for DSH

Use the installed skills under `.dsh/skills` and invoke `/rite` or `/rite-<verb>`.
Read `.dsh/skills/devrites-lib/reference/standards/core.md` before a workflow.
Dispatch means call `subagent` with `agent_preset` set to the exact named specialist,
a self-contained `prompt`, and a short `description`. Every dispatch uses the native
`spawn` provider, a fresh Session with zero parent conversation. Never use a generic
agent, inherit the orchestrator preset, or use `subagent_fork` as a substitute.
Include the exact project-relative allowed paths and required evidence in the prompt.
The child id returned by `subagent` is an admission receipt, not a completed result.
Collect the native parent settlement notice for each id, including final text and
terminal status. `list_agents` activity `inactive` does not prove success. Reconcile
every result and compare returned paths with the contract before continuing.
Missing preset, failed activation, interrupted execution, or missing result blocks
the dependent work. Report the failure; never impersonate the specialist inline.
The orchestrator writes workspace bookkeeping; only the named writer writes source
and tests within its path contract. No skill or result authorizes commits, pushes,
publishing, deployment, or other irreversible actions without explicit approval.
<!-- END DEVRITES DSH -->

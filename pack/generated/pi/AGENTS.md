<!-- BEGIN DEVRITES PI -->
## DevRites For pi

This project has DevRites installed for pi.

## pi usage

- DevRites workflow skills live in `.pi/skills`. Public commands run as `/rite`, `/rite-spec`, ... (prompt templates under `.pi/prompts`) or `/skill:rite`, `/skill:rite-spec`, ...; the forms are equivalent.
- Before using any DevRites workflow skill, read `.pi/skills/devrites-lib/reference/standards/core.md`. Load other `.pi/skills/devrites-lib/reference/standards/*.md` files when the skill or rule index asks for them.
- DevRites specialist agents live in `.pi/agents` and are provided by the `pi-subagents` extension. Before dispatch, run `subagent({ action: "list" })`; if a required `devrites-<role>` is absent or not executable, stop for HITL — never skip it, substitute a generic child, or run the specialist role in the root context.
- In DevRites guidance, **invoke** means run a skill inline in the current context; **dispatch** means start a fresh agent with `subagent({ agent, task })` (or `runs.run`/`runs.all` in a `workflowScript`), wait for it, and reconcile its result.
- Only `devrites-slice-wright` may edit source or tests; every other specialist is read-only by tool allowlist. Exact paths are instruction-enforced: put the project-relative paths in the task, wait for the wright, compare its file list and `git diff --name-only` with that contract, and reject any extra path.
- The explicit `/overhaul` skill ships its own `overhaul-*` agents outside the DevRites lifecycle. They carry the full write tool set, are dispatched only by that skill, and follow its own approval gate and path contracts instead of the lifecycle writer rules above.
- DevRites runtime helpers run through the installed `devrites-engine` binary.
- Installed `.pi/` content loads only after the project is trusted. If pi has not trusted this project, the skills, agents, and prompts above are not active.
- A seal GO, AFK mode, or autocomplete flag never authorizes an irreversible action. Disclose the exact commit/push/tag/PR plan and obtain fresh explicit user approval for that attempt; any changed or retried plan needs fresh approval.

## Workflow contract

- Keep all feature state in `.devrites/work/<slug>/` and preserve `.devrites/ACTIVE`.
- Follow the DevRites lifecycle: frame -> spec -> clarify -> temper -> define -> plan -> vet -> build -> converge -> prove -> polish -> review -> seal -> ship -> done.
- Claims of completion need recorded evidence in the feature workspace, not confidence alone.
<!-- END DEVRITES PI -->

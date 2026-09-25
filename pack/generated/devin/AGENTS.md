<!-- BEGIN DEVRITES DEVIN -->
## DevRites For Devin

This project has DevRites installed for Devin CLI.

## Devin usage

- DevRites workflow skills live in `.devin/skills`. Run them as `/rite`, `/rite-spec`, and the other `/rite-*` slash commands.
- Before using any DevRites workflow skill, read `.devin/skills/devrites-lib/reference/standards/core.md`. Load other `.devin/skills/devrites-lib/reference/standards/*.md` files when the skill or rule index asks for them.
- DevRites specialist agents live in `.devin/agents` as custom subagent profiles offered to `run_subagent`. If a required `devrites-<role>` profile is not among the offered profiles, stop for HITL - never skip the role, never substitute `subagent_general`, `subagent_explore`, or another generic profile, never inline the specialist prompt, and never execute the specialist role in the root context.
- In DevRites guidance, **invoke** means run a skill inline in the current context; **dispatch** means start a fresh agent with `run_subagent` using the exact `profile` name, wait for it with `read_subagent`, and reconcile its result.
- Only `devrites-slice-wright` may edit source or tests; every other specialist is read-only by `allowed-tools`. Exact paths are instruction-enforced: put the project-relative paths in the task, wait for the wright, compare its file list and `git diff --name-only` with that contract, and reject any extra path.
- The explicit `/overhaul` skill ships its own `overhaul-*` agents outside the DevRites lifecycle. They carry the full write tool set, are dispatched only by that skill, and follow its own approval gate and path contracts instead of the lifecycle writer rules above.
- Custom profiles do not inherit interactive tool grants, and background subagents auto-deny unapproved tools. Dispatch the write-capable wright in the foreground, or make sure its `edit`, `write`, and `exec` tools are approved first. Specialists cannot use `ask_user_question`; the root relays user questions and answers through the task text and returned result.
- DevRites runtime helpers run through the installed `devrites-engine` binary.
- Skills and agent profiles are loaded when a session starts. If they were installed while this session was open, restart the session or reopen the project before relying on them.
- A seal GO, AFK mode, or autocomplete flag never authorizes an irreversible action. Disclose the exact commit/push/tag/PR plan and obtain fresh explicit user approval for that attempt; any changed or retried plan needs fresh approval.

## Workflow contract

- Keep all feature state in `.devrites/work/<slug>/` and preserve `.devrites/ACTIVE`.
- Follow the DevRites lifecycle: frame -> spec -> clarify -> temper -> define -> plan -> vet -> build -> converge -> prove -> polish -> review -> seal -> ship -> done.
- Claims of completion need recorded evidence in the feature workspace, not confidence alone.
<!-- END DEVRITES DEVIN -->

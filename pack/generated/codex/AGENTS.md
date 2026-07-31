<!-- BEGIN DEVRITES CODEX -->
## DevRites For Codex

This project has DevRites installed for both Claude Code and Codex.

## Codex usage

- **Inspect before trust.** Codex skips project-scoped `.codex/` configuration and agents in an untrusted project. Before enabling them, inspect `.codex/config.toml`, `.codex/agents/`, and the project guidance. The human operator decides whether to trust the folder.
- DevRites workflow skills are available to Codex from `.agents/skills`.
- Use `$rite` or `$rite-<verb>` through Codex skills, or open `/skills` and select the matching DevRites skill.
- If the user mentions a DevRites slash command such as `/rite spec`, `/rite-build`, or `/rite-seal`, treat that as an explicit request to use the corresponding DevRites skill.
- DevRites runtime helpers run through the installed `devrites-engine` binary.
- Before using any DevRites workflow skill, read `.agents/skills/devrites-lib/reference/standards/core.md`. Load other `.agents/skills/devrites-lib/reference/standards/*.md` files when the skill or rule index asks for them. These are DevRites engineering standards, not Codex exec-policy `.rules` files.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Custom Codex subagents generated from the DevRites review agents live in `.codex/agents`.
- In DevRites guidance, **invoke** means run a skill inline in the current context; **dispatch** means start a fresh agent with `spawn_agent`, wait for it, and reconcile its result.
- Dispatch every exact named `devrites-<role>` required by the active workflow in a fresh subagent thread, wait for it, and reconcile its result. If that role is unavailable, stop for HITL; never skip it, never substitute a generic/default child, and never execute the specialist role in the root context.
- The root uses the workspace-capable `devrites-orchestrator` profile because Codex children cannot elevate above the parent permission ceiling. This permission is for native writer dispatch and workflow artifacts; the root must never edit source or tests itself.
- Every generated specialist is hook-free. `devrites-slice-wright` alone uses `default_permissions = ":workspace"`; every other specialist uses `default_permissions = ":read-only"`. Exact paths are instruction-enforced: put the project-relative paths in the task, wait for the wright, compare its file list and `git diff --name-only` with that contract, and reject any extra path. Never bypass the wright, widen its contract, or edit source in the root; the root must never recreate an engine dispatch bridge.
- A seal GO, AFK mode, or autocomplete flag never authorizes an irreversible action. Disclose the exact commit/push/tag/PR plan and obtain fresh explicit user approval for that attempt; any changed or retried plan needs fresh approval. Native host permission and sandbox prompts remain authoritative and cannot be inferred or bypassed.

## Workflow contract

- Keep all feature state in `.devrites/work/<slug>/` and preserve `.devrites/ACTIVE`.
- Follow the DevRites lifecycle: frame -> spec -> clarify -> temper -> define -> plan -> vet -> build -> converge -> prove -> polish -> review -> seal -> ship -> done.
- Claims of completion need recorded evidence in the feature workspace, not confidence alone.
<!-- END DEVRITES CODEX -->

# Porting DevRites to a new harness

DevRites is distributed through `npx devrites` and installs project-local artifacts. Do not add a harness through a Claude/Codex plugin or marketplace path.

Before adding a host, update `engine/internal/harness/hosts.json` and prove the host can run DevRites without per-session opt-in.

## Hard acceptance test

In a clean trusted project, ask:

> Let’s make a react todo list.

The harness must invoke DevRites and route to `/rite-spec` (or the host's explicit DevRites spec workflow) before writing app code. Attach the transcript to the PR.

## Port checklist

- Skill discovery path is project-local and installed by `npx devrites`.
- Session/project orientation runs automatically at startup/compact/resume, or the missing surface is recorded in `docs/harness-compliance.md`.
- `devrites-engine` hooks/gates are wired, or each degraded gate has an honest fallback and confidence label.
- Subagent support is mapped; if unavailable, reviewers/wrights use the documented inline fallback and label it.
- Install, update, and uninstall paths stay project-local; no global writes.
- Generated artifacts and `docs/harness-compliance.md` are updated from the same source of truth.
- Routing eval includes the React todo-list acceptance prompt.

Skip badge-chasing: add a host only when a real user will exercise it and the acceptance transcript passes.

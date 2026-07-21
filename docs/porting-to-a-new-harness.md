# Porting DevRites to a new harness

DevRites' canonical distribution path is `npx devrites` (with the documented
Bash bootstrap as a Node-free alternative), and host artifacts install into the
target project. Do not add a harness through a Claude/Codex plugin or marketplace path.

Before adding a host, add its thin adapter to `engine/internal/harness/harness.go`,
update the frozen matrix in `compliance.go`, and prove the host can run DevRites
without per-session opt-in.

## Hard acceptance test

In a clean trusted project, ask:

> Let’s make a react todo list.

The harness must invoke DevRites and route to `/rite-spec` (or the host's explicit DevRites spec workflow) before writing app code. Attach the transcript to the PR.

## Port checklist

- Skill discovery path is project-local and installed by `npx devrites`.
- Session/project orientation runs automatically at startup/compact/resume, or the missing surface is recorded in `docs/harness-compliance.md`.
- `devrites-engine` hooks/gates are wired, or each degraded gate has an honest fallback and confidence label.
- Subagent support is mapped; if unavailable, reviewers/wrights use the documented inline fallback and label it.
- Skills, agents, standards, hooks, and harness guidance stay project-local;
  the optional shared `devrites-engine` executable is the sole sanctioned
  global artifact and must remain skippable with `--no-binary`.
- Generated artifacts and `docs/harness-compliance.md` are updated from the same source of truth.
- Routing eval includes the React todo-list acceptance prompt.

Skip badge-chasing: add a host only when a real user will exercise it and the acceptance transcript passes.

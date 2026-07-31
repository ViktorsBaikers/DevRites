# Porting DevRites to a new harness

DevRites' canonical distribution path is `npx devrites` (with the documented
Bash bootstrap as a Node-free alternative), and host artifacts install into the
target project. Do not add a harness through a Claude/Codex plugin or marketplace path.

Before adding a host, update the canonical host configuration and profiles plus
their generator, regenerate the derived artifacts, and update the compliance
summary. Prove the host's permissions, routing, installation, and live
acceptance before listing it.

## Hard acceptance test

In a clean trusted project, ask:

> Let’s make a react todo list.

The harness must invoke DevRites and route to `/rite-spec` (or the host's explicit DevRites spec workflow) before writing app code. Attach the transcript to the PR.

## Port checklist

- Skill discovery path is project-local and installed by `npx devrites`.
- Session/project orientation runs automatically at startup/compact/resume, or the missing surface is recorded in `docs/harness-compliance.md`.
- The documented read-only `devrites-engine check candidate <slug>`, structural
  checks, and atomic `state` primitives are wired. Adapters preserve the exact
  two-field candidate output and digest bindings; semantic judgment stays with
  the host skills and exact agents.
- Subagent support is mapped; if unavailable, required reviewer/wright work
  stops for HITL rather than running in the root context.
- Skills, agents, standards, permissions, and harness guidance stay project-local;
  the optional shared `devrites-engine` executable is the sole sanctioned
  global artifact and must remain skippable with `--no-binary`.
- Canonical host configuration and profiles plus their generator are updated,
  then derived artifacts are regenerated.
- Build remains manifest writer; Prove binds real evidence; Polish performs
  durable rollups and re-proof; Review and Seal bind the closed candidate; Ship
  is candidate-read-only and verifies exact staged/committed scope. A harness
  must not substitute its own hash, move these owners, or weaken type-`GO`.
- Generation, installation, permission, and routing tests cover the new host.
- `docs/harness-compliance.md` reflects the proven host behavior.
- Live routing acceptance includes the React todo-list prompt.

Skip badge-chasing: add a host only when a real user will exercise it and the acceptance transcript passes.

Extend the canonical `pack/.claude/` owners and generator. Do not add a second
candidate schema, command registry, lifecycle phase, or harness-specific copy
of workflow policy; generated host mirrors remain derived artifacts.

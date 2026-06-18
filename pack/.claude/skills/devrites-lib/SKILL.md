---
name: devrites-lib
description: Internal shared-script library for DevRites — not a user command and not model-invocable. It exists only to ship DevRites' cross-cutting helper scripts (notably the read-only orientation preamble that every workspace-operating rite-* skill runs at step 0) on both the bash-installer and plugin channels. No workflow; do not invoke.
user-invocable: false
disable-model-invocation: true
---

# devrites-lib — internal shared scripts (not a command)

This is **not** a skill you run. It is a library directory housing DevRites'
cross-cutting helper scripts under `scripts/`, placed inside `skills/` so they
install on **both** distribution channels (the bash installer copies the pack's
`skills/` tree into the project's `.claude/`; the plugin ships the `skills/`
tree). Skills resolve these scripts with the standard three-layout snippet:
the installed `.claude/` path first, then the plugin cache via
`${CLAUDE_SKILL_DIR}` (best-effort — Claude Code doesn't expose a stable script path to skill-invoked bash on the plugin channel, so the preamble degrades to reading `state.md` there), then the repo source tree for DevRites
self-development.

## Scripts

- `scripts/preamble.sh` — read-only orientation digest for the active
  `.devrites/` feature: prints `state.md`, the artifacts present, the run mode
  (HITL/AFK), and the open-question tally by gate. Run first by every
  workspace-operating `rite-*` skill so the model orients deterministically
  instead of re-deriving state from raw Markdown. Never mutates the workspace.

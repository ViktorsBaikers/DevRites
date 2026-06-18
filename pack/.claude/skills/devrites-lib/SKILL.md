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

All resolve with the standard three-layout snippet (installed `.claude/` path
first, then `${CLAUDE_SKILL_DIR}`, then the repo `pack/` source).

**Read-only — orient / gate (never mutate the workspace):**

- `scripts/preamble.sh` — orientation digest for the active `.devrites/` feature:
  prints `state.md`, the artifacts present, the run mode (HITL/AFK), and the
  open-question tally by gate. Run first (step 0) by every workspace-operating
  `rite-*` skill so the model orients deterministically instead of re-deriving
  state from raw Markdown.
- `scripts/readiness.sh` — build-readiness gate. Exits non-zero on `/rite-build`'s
  step-0 stop conditions so they hold by exit code, not by prose the model must
  remember: `2` no `Plan approved` (→ `/rite-define`), `3` `awaiting_human`
  (→ `/rite-resolve`), `4` `blocked` (→ `/rite-plan`), `5` no workspace, `0` ready.
- `scripts/evidence-fresh.sh` — evidence-freshness gate for `/rite-seal`. Exits `3`
  when any file in `touched-files.md` is newer than `evidence.md` /
  `browser-evidence.md` (stale proof = NO-GO until re-proven), `0` when fresh.
- `scripts/check-acceptance.sh` — executable acceptance gate. Compiles `spec.md`'s
  `[ACn]`-tagged criteria and exits `1` unless every one is checked (proven) in `seal.md`;
  used by `/rite-seal` and by the outcome grader.

**State mutators — write `state.md` / `questions.md` under one contract:**

- `scripts/tick-afk.sh` — decrement the AFK slice budget; exits `3` at 0 (forced HITL stop).
- `scripts/resolve.sh` — backs the `/rite-resolve` contract (answer / drop / batch).
- `scripts/close-out.sh` — archive the workspace + clear `ACTIVE` on `/rite-ship`.

**Unified entrypoint (tool-agnostic):**

- `scripts/devrites.sh` — one CLI dispatching to all of the above (`orient` / `ready` /
  `evidence-fresh` / `acceptance` / `tick-afk` / `resolve` / `close` / `active` / `list` /
  `use`), so any agent or human can drive `.devrites/` without the skill prose. The MCP
  wrapper `mcp/devrites-mcp.mjs` exposes the read/gate ops as MCP tools. See
  [`docs/cli-mcp.md`](../../../../docs/cli-mcp.md).

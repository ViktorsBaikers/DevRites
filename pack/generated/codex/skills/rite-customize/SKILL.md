---
name: rite-customize
description: User-invoked helper for authoring DevRites overrides or extensions.
argument-hint: "[override <agent> | extension <name>]"
user-invocable: true
disable-model-invocation: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- Inspect the current `spawn_agent` role list. When the named `devrites-<role>` is exposed, dispatch it with `fork_turns="none"`; full-history forks inherit the parent type. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If a named role is not exposed, use generic `explorer` for every read-only role with `fork_turns="none"`. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. Trusted `.codex/hooks.json` binds `agent_type=explorer` to the fail-closed reviewer read-only guard.
- For `devrites-slice-wright`, trusted `.codex/hooks.json` binds generic `worker` (`agent_type=worker`) to the active reconcile window and exact `.wright-allowlist`. Dispatch that worker with `fork_turns="none"`, tell it to read `.codex/agents/devrites-slice-wright.toml`, and execute the unchanged packet. Never create `.reconcile-inline` when this safe rung is available.
- A missing custom role is not evidence that spawning is unavailable. Only when the project hooks are unavailable or untrusted, no spawn primitive exists, or higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, create `.reconcile-inline` only for that path, and apply every fallback risk gate.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-customize: guided project customization

Create, list, validate, or scaffold one project-local DevRites customization without forking the pack.

Read [`devrites-lib/reference/standards/core.md`](../devrites-lib/reference/standards/core.md), then read `docs/extensions.md` if present; in an installed project, fall back to `.agents/skills/devrites-lib/reference/standards/core.md` plus the user's installed DevRites docs if available.

Fast paths:
- `list` → run `devrites-engine overrides list` and `devrites-engine extensions list`; stop.
- `validate` → run both validators; stop.
- `scaffold extension <name>` → create the smallest valid `.devrites/extensions/<name>/skill/SKILL.md` draft after approval, then validate.

## Workflow

1. **Classify the ask.** Pick exactly one target:
   - `list` → inspect only.
   - `validate` → validate only.
   - `override <agent>` → `.devrites/overrides/<agent>.md`
   - `extension <name>` / `scaffold extension <name>` → `.devrites/extensions/<name>/`
   - unclear → ask one blocking question with those options.
   Done when the target kind and path are known.
2. **Load the existing surface.** Read the existing file/dir if present. For overrides, confirm the target agent exists under `.codex/agents/` or `.codex/agents/`. For extensions, check whether `skill/SKILL.md` or `agent.md` already exists. Done when you know whether this is create or update.
3. **Draft the smallest artifact.**
   - Override: write only added checks/emphasis. Do not restate base reviewer rules.
   - Extension: create the smallest valid `skill/SKILL.md` or `agent.md` that matches the user's requested surface.
   Done when the draft has no gate-waiver language and no global paths.
4. **Show before writing.** Present the path and full content (or a concise diff when updating), then wait. **Completion:** the user explicitly approves or aborts.
5. **Write and validate.** Create parent dirs, write the file(s), then run the matching validator:
   ```bash
   devrites-engine overrides validate
   devrites-engine extensions validate
   ```
   If validation fails, fix the artifact once and re-run. Stop rather than guessing after a second failure.

## Rules

- Project-local only: write under `.devrites/overrides/` or `.devrites/extensions/`.
- A customization may add checks or raise weight; it may never relax a gate, waive a standard, bypass `type-GO`, or write global config.
- Sparse wins: do not copy shipped skills, agents, or standards unless the user explicitly asked for a new extension based on them.
- Scaffold means a tiny valid skeleton plus a TODO body, not a generated framework.
- No Codex mirroring by hand; extension sync is owned by `devrites-engine extensions sync`.

## Output

Reply-contract exception: customization utility; may run without an active feature.

```
Done: created|updated <override|extension>.
Changed: <path(s)>
Evidence: <validator command + result>
Open: <none | validation error | user-aborted>
Next: <devrites-engine extensions sync | $rite-doctor | none>
Record: <path(s)>
```

## Gotchas

- **Do not fork by accident.** If a reviewer override is enough, do not create a copied reviewer agent.
- **Do not weaken gates.** "Ignore", "waive", "lower severity", or "skip type-GO" means rewrite the customization or refuse it.
- **Do not invent an extension surface.** If the user wants behavior outside DevRites' extension contract, say what the contract supports and stop.

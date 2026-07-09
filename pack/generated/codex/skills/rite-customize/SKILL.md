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
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-customize — guided project customization

Create or update one project-local DevRites customization without forking the pack.

Read [`devrites-lib/reference/standards/core.md`](../devrites-lib/reference/standards/core.md), then read `docs/extensions.md` if present; in an installed project, fall back to `.agents/skills/devrites-lib/reference/standards/core.md` plus the user's installed DevRites docs if available.

## Workflow

1. **Classify the ask.** Pick exactly one target:
   - `override <agent>` → `.devrites/overrides/<agent>.md`
   - `extension <name>` → `.devrites/extensions/<name>/`
   - unclear → ask one blocking question with those two options.
   Done when the target kind and path are known.
2. **Load the existing surface.** Read the existing file/dir if present. For overrides, confirm the target agent exists under `.codex/agents/` or `.codex/agents/`. For extensions, check whether `skill/SKILL.md` or `agent.md` already exists. Done when you know whether this is create or update.
3. **Draft the smallest artifact.**
   - Override: write only added checks/emphasis. Do not restate base reviewer rules.
   - Extension: create the smallest valid `skill/SKILL.md` or `agent.md` that matches the user's requested surface.
   Done when the draft has no gate-waiver language and no global paths.
4. **Show before writing.** Present the path and full content (or a concise diff when updating), then wait for explicit approval. Done only after the user approves or aborts.
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
- **Do not weaken gates.** “Ignore”, “waive”, “lower severity”, or “skip type-GO” means rewrite the customization or refuse it.
- **Do not invent an extension surface.** If the user wants behavior outside DevRites' extension contract, say what the contract supports and stop.

# Extending DevRites — project extensions & reviewer overrides

DevRites ships a fixed, authored pack of rites, reviewers, and standards. Two project-local
surfaces let a team tune it **without forking the pack** — both live in the data plane
(`.devrites/`), both are git-diffable, and both are held to the same contracts the shipped pack is.

They answer two different questions:

| Surface | Question it answers | Lives in |
|---|---|---|
| **Extensions** | "Add a *new* rite or reviewer / domain." | `.devrites/extensions/<name>/` |
| **Overrides** | "Reshape a *shipped* reviewer's emphasis." | `.devrites/overrides/<agent>.md` |

The guiding invariant for both: **a shipped gate is never weakened by a customization.** An
extension earns parity by passing the same validator as the pack; an override may raise the bar,
never lower it. The deterministic engine gates (`seal`, `review-integrity`, `check-acceptance`, …)
do not read either surface — they stay authoritative regardless of what a project adds.

---

## Extensions — add a rite or reviewer

A project extension is a directory under `.devrites/extensions/`:

```
.devrites/extensions/<name>/
  skill/SKILL.md      (optional) a user rite/skill — needs name + description frontmatter
  agent.md            (optional) a user reviewer agent — needs name + description frontmatter
  component.yaml      (optional) npx-managed component manifest + safety bounds
  provenance.json     (optional) source/author/review metadata for shared extensions
  extension.yaml      (optional) metadata: aliases (prior names, so a rename doesn't orphan)
```

An extension provides a skill, an agent, or both. It is held to the same schema as the shipped
pack — a malformed extension is refused, not silently half-installed.

### Commands

```bash
devrites-engine extensions list       # enumerate extensions and what each provides
devrites-engine extensions validate   # schema-check every extension (exit 1 on a violation)
devrites-engine extensions sync        # mirror valid extensions into .claude/ so the harness finds them
```

- **`validate`** checks each declared skill/agent carries `name:` + `description:` frontmatter, that
  an extension provides at least one artifact, and that no two extensions claim the same skill/agent
  name. If `component.yaml` is present, it must declare an npm-managed, project-local component:
  `distribution: npx-managed`, `scope: project-local`, project-local write roots only, and safety
  fields that do **not** weaken gates, bypass `type-GO`, or run executables. If `provenance.json`
  is present it must be valid JSON with `source`, `author`, `created_at`, and optional
  `confidence` (`0..1`) / `reviewed_by`; `/rite-doctor` warns when an extension ships artifacts
  without provenance. V2 manifests may
  also declare `tier`, `requires`, `owns`, and `surface`; dependencies must be acyclic, and anything
  in `owns` that collides with the first-party `rite-`/`devrites-` namespaces is refused. If a
  malformed extension declares a review/gate-like surface, validation warns that the surface is
  inactive until the schema is fixed — fail-open, but loud.
- **`sync`** validates first, then mirrors `skill/` → `.claude/skills/<name>/` and `agent.md` →
  `.claude/agents/<name>.md`, where the Claude harness discovers them. Idempotent. It refuses to
  sync a set that fails validation.
- **`aliases`** (in `extension.yaml`) carry an extension's prior names forward across a rename, so a
  project's references don't orphan.

### Component manifest

Public schemas live in [`../schemas/`](../schemas/): `extension-component.schema.json` and
`provenance.schema.json` for extension metadata, plus workspace/hook schemas for editor tooling.

`component.yaml` is the DevRites-native answer to Spec Kit-style extension/preset/bundle metadata.
It is **not** a plugin package descriptor; it documents what the npm-installed engine may copy or
validate inside the current project.

```yaml
schema_version: "1.1"
kind: extension
id: audit-lite              # must match .devrites/extensions/<name>/ when present
version: 0.1.0
tier: standard              # core | standard | full
requires: []                # other local extension ids; acyclic
owns:
  skills: [audit-lite]
  agents: []
surface:
  clusters: [review]
safety:
  may_weaken_gates: false
  executable: false
```

Legacy `component:` / `permissions:` manifests still validate. The validator refuses global homes
(`~/.claude/**`, `~/.codex/**`), plugin-store distribution names, non-project scopes, unknown/cyclic
extension dependencies, first-party ownership collisions, and any manifest that claims it can weaken
a gate, run executables, or bypass `type-GO`.

### Workflow

Use `/rite-customize extension <name>` for guided authoring, or do it by hand:

1. Author the extension under `.devrites/extensions/<name>/`.
2. `devrites-engine extensions validate` until clean.
3. `devrites-engine extensions sync` to mirror it into `.claude/`.
4. `/rite-doctor` validates extensions on every health check and hands you the fix if one regresses.

### Harness scope

`sync` targets the **Claude** layout (`.claude/skills`, `.claude/agents`), which the harness
auto-discovers. It does not generate Codex mirrors. The `.md` → `.codex/agents/*.toml` conversion,
path rewriting, hook/config blocks, and skills-list description stubbing are deliberately limited to
the generated shipped pack artifacts so extension sync does not become a second Codex generator.

---

## Overrides — reshape a shipped reviewer

An override is a single Markdown file named for the agent it targets:

```
.devrites/overrides/devrites-code-reviewer.md
.devrites/overrides/devrites-security-auditor.md
```

Each shipped reviewer, after loading its governing standards, reads
`.devrites/overrides/<its-name>.md` if present and applies it as **project overrides** — extra
emphasis or house rules this project wants enforced. For example, a `devrites-code-reviewer.md`
override that says "always flag any use of the deprecated `legacyClient` as Important."

### The one rule

An override may **add** checks or **raise** weight. It may **never** relax a gate, waive a standard,
or lower a severity floor — a Critical stays a Critical. Overrides are reviewer *input*, not
permission. The engine gates don't read them at all, so an override literally cannot disable a gate;
the linter exists to catch one that *tries to talk a reviewer into it*.

### Commands

Use `/rite-customize override <agent>` for guided authoring, or manage files directly:

```bash
devrites-engine overrides list       # enumerate override files and the agent each targets
devrites-engine overrides validate   # flag empty overrides and any that read like a gate waiver (exit 1)
```

`validate` trips on subversion phrasing ("ignore the gate", "treat any Critical as a Suggestion",
"waive review") and on an override targeting an agent that isn't installed (an orphan warning).
`/rite-doctor` runs it on every health check.

### Template overrides

Future template/preset work uses `.devrites/overrides/templates/*.md` as the project-local override
layer. `devrites-engine overrides validate` already applies the same anti-subversion scan to those
files and adds required-term checks for high-risk lifecycle templates:

- `seal.md` must still mention `type-GO` and `NO-GO`.
- `ship.md` must still mention `type-GO`.

This keeps the extension point useful without letting a project template erase the gates that make
DevRites safe.

# Skill authoring

Use this when creating or editing DevRites skills.

## Distribution

DevRites is installed through the npm package (`npx devrites ...`). Claude Code
and Codex files are generated host artifacts copied by that installer, never
Claude/Codex plugin-store surfaces. Edit the canonical Claude-authored pack sources,
rebuild host artifacts, then validate.

## Surface lifecycle

- **Promoted** — shipped in `pack/`, documented in `docs/skills.md` and
  `docs/command-map.md`, and covered by validation.
- **Draft** — local/research material outside the shipped pack.
- **Deprecated** — shipped only as a compatibility bridge with a replacement and
  removal note.
- **Research** — notes under `docs/research/`; never installed.

## Description

The description is an invocation pointer, not documentation.

- Keep user-facing skills under 90 words, model-invoked specialists under 75, and `devrites-lib` under 60.
- Use one clear trigger branch per phrase; repeated `Use when` or `Not for` means the branch should collapse or move into the body.
- Put examples, edge cases, and rationale in `SKILL.md` body or a reference file, not in frontmatter.

## Body

- Put ordered work as steps, each ending in a checkable completion criterion.
- Move branch-only reference behind a direct file pointer.
- Keep one meaning in one place; prefer a shared reference over repeated prose.

## Router and docs

- Public `rite-*` skills must appear in the `/rite` router, `docs/skills.md`,
  and `docs/command-map.md`.
- Internal `devrites-*` skills must stay out of the public command menu unless
  named as implementation detail.
- A public skill's docs card states purpose, when to invoke, where it fits, and
  what evidence proves completion. Do not copy the full `SKILL.md` process into
  docs.

## Source intake

External skill packs, articles, and examples are references, not authority.

- Record source, commit/date, and files read in `docs/research/`.
- Adopt the DevRites principle, not foreign names or workflow chains.
- Name rejected ideas so future maintainers do not re-litigate them.
- Add a validator or eval when the adoption creates a durable product contract.

## Pruning

Delete no-op instructions the model already follows. Keep positive target behavior; use prohibitions only for hard guardrails.


## Contribution preflight

New skills are expensive routing surface. Before adding one, document the catalog search, why a reference inside an existing skill is insufficient, required eval coverage, host command parity, and whether the surface is public `rite-*` or internal `devrites-*`. Public commands need docs, evals, generated Claude/Codex artifacts, and a reply-contract marker. Internal skills need a clear trigger boundary and "when not to use" section. Agents need role/scope, read/write mode, output format, and composition block; only `devrites-slice-wright` may write.

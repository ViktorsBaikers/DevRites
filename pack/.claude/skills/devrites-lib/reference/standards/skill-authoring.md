# Skill authoring

Use this when creating or editing DevRites skills.

## Description

The description is an invocation pointer, not documentation.

- Keep user-facing skills under 90 words, model-invoked specialists under 75, and `devrites-lib` under 60.
- Use one clear trigger branch per phrase; repeated `Use when` or `Not for` means the branch should collapse or move into the body.
- Put examples, edge cases, and rationale in `SKILL.md` body or a reference file, not in frontmatter.

## Body

- Put ordered work as steps, each ending in a checkable completion criterion.
- Move branch-only reference behind a direct file pointer.
- Keep one meaning in one place; prefer a shared reference over repeated prose.

## Pruning

Delete no-op instructions the model already follows. Keep positive target behavior; use prohibitions only for hard guardrails.

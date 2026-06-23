/**
 * DevRites commit policy — strict Conventional Commits.
 *
 * Format:  type(scope): subject
 *   - type:    one of the allowed types (lower-case, required)
 *   - scope:   one of the allowed DevRites areas (lower-case, required)
 *   - subject: imperative, no leading capital, no trailing period
 *   - header:  12–72 chars
 *   - body:    separated by a blank line; lines <= 100 chars
 *
 * Examples (valid):
 *   feat(skills): add rite-prove browser proof ladder
 *   fix(installer): match first rule pack with leading-space guard
 *   remove(installer): drop the Claude Code plugin install path
 *   docs(rules): adapt common/agents.md for DevRites agents
 *
 * Changelog grouping (Keep a Changelog): feat -> Added, remove -> Removed,
 * fix -> Fixed, perf/refactor/build -> Changed, docs -> Documentation.
 *
 * Refs: https://www.conventionalcommits.org  ·  https://commitlint.js.org
 */
module.exports = {
  extends: ['@commitlint/config-conventional'],
  // Skip lint for semantic-release auto-commits — their bodies embed
  // changelog markdown links that exceed body-max-line-length.
  ignores: [(message) => /^chore\(release\):\s+\d+\.\d+\.\d+/.test(message)],
  rules: {
    // type
    'type-enum': [
      2,
      'always',
      ['feat', 'fix', 'remove', 'docs', 'style', 'refactor', 'perf', 'test', 'build', 'ci', 'chore', 'revert'],
    ],
    'type-case': [2, 'always', 'lower-case'],
    'type-empty': [2, 'never'],

    // scope — required and constrained to DevRites areas
    'scope-enum': [
      2,
      'always',
      ['skills', 'rite', 'devrites', 'agents', 'rules', 'installer', 'uninstall', 'scripts', 'docs', 'tests', 'deps', 'release', 'repo', 'ci'],
    ],
    'scope-empty': [2, 'never'],
    'scope-case': [2, 'always', 'lower-case'],

    // subject
    'subject-empty': [2, 'never'],
    'subject-full-stop': [2, 'never', '.'],
    // inherit config-conventional subject-case (forbids sentence/start/pascal/upper case,
    // but still allows acronyms like API/OWASP mid-subject)

    // header / body / footer shape
    'header-max-length': [2, 'always', 72],
    'header-min-length': [2, 'always', 12],
    'body-leading-blank': [2, 'always'],
    'body-max-line-length': [2, 'always', 100],
    'footer-leading-blank': [2, 'always'],
  },
};

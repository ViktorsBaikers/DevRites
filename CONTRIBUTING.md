# Contributing to DevRites

DevRites welcomes bug reports, fixes, new skills, documentation improvements,
and eval queries.

This guide covers how to file an issue, set up a local dev environment, author a
change that passes CI, and open a pull request that's easy to review.

## Table of contents

- [Code of conduct](#code-of-conduct)
- [License & contributor terms](#license--contributor-terms)
- [Ways to contribute](#ways-to-contribute)
- [Before you open a PR](#before-you-open-a-pr)
- [Local development setup](#local-development-setup)
- [Project layout (what lives where)](#project-layout-what-lives-where)
- [Authoring guidelines](#authoring-guidelines)
- [Commit message format](#commit-message-format-strict)
- [Pull request process](#pull-request-process)
- [Running tests, validators, and evals](#running-tests-validators-and-evals)
- [Release impact of your commits](#release-impact-of-your-commits)
- [Reporting security issues](#reporting-security-issues)
- [Getting help](#getting-help)

## Code of conduct

DevRites adopts the **Contributor Covenant 2.1**. By participating you agree to
abide by it. See [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).

## License & contributor terms

DevRites is **source-available**, not OSI open source. See [`LICENSE`](LICENSE)
for the full terms.

By submitting a contribution (pull request, patch, issue with code, or any
other proposed material), you agree that:

1. Your contribution is your own original work, or you have the right to submit
   it under the project's license.
2. Your contribution is licensed to the project under the same terms as
   [`LICENSE`](LICENSE), and may be redistributed by the maintainer as part of
   DevRites.
3. You retain copyright in your contribution; you grant the project a
   perpetual, worldwide, royalty-free license to use, modify, and redistribute
   it under the DevRites License.

You do not need to sign a separate CLA. Opening a PR constitutes acceptance of
the terms above.

## Ways to contribute

| Type | Where it lives | Notes |
|---|---|---|
| Bug report | GitHub Issues | Include version (`.claude/devrites.manifest`), repro, expected vs actual. |
| Feature request | GitHub Issues / Discussions | Explain the problem first; suggest a shape, not a finished design. |
| New / improved skill | `pack/.claude/skills/<skill>/SKILL.md` | Must have frontmatter, body discipline, and ≥1 eval file. |
| Review agent | `pack/.claude/agents/<agent>.md` | Read-only, fresh-context, severity-labeled output. |
| Engineering rule | `pack/.claude/skills/devrites-lib/reference/standards/<rule>.md` | Stack-agnostic. Project conventions always win. |
| Docs | `docs/` or `README.md` | Keep cross-links current. |
| Eval query | `evals/<skill>.json` | Trigger phrasing that covers the skill's positive, negative, and boundary routing branches; corpus size follows the branch shape rather than a fixed quota. |
| Behavioral eval | `evals/behavioral/<skill>.json` | Pressure scenario that tests whether a gating skill resists a documented rationalization. Opt-in; sourced from `anti-patterns.md`. |
| Installer / scripts | `install.sh`, `scripts/*`, `engine/internal/install/` | Host artifacts stay project-local and manifest-managed; only the optional shared engine binary may be installed globally. |

If you're not sure where a change belongs, open a discussion or draft issue
first.

## Before you open a PR

A 60-second checklist that saves review round-trips:

- [ ] Issue exists (or the change is small enough to skip one).
- [ ] You've read the relevant section of [`docs/architecture.md`](docs/architecture.md).
- [ ] Commit messages follow the **strict** Conventional Commits policy below.
- [ ] `npm run validate` passes.
- [ ] `npm run audit` reports no moderate-or-higher dependency advisories.
- [ ] `npm test` passes (install/uninstall smoke + pack validation).
- [ ] If you touched a skill, you ran the matching eval (`scripts/run-evals.sh`).
- [ ] If you touched a **gating** skill's discipline (or its `anti-patterns.md`), you ran / updated its behavioral eval (`scripts/run-behavioral-evals.sh`).
- [ ] No skill, agent, or hook artifacts are written to `~/.claude` or
  `~/.codex`; any global write is limited to the shared engine-binary lifecycle.
- [ ] No new network calls exist outside the sanctioned bootstrap, update, and
  source-cache boundary in `engine/internal/iohooks`; skill research uses explicit host tools.
- [ ] You've updated docs and cross-links touched by the change.

## Local development setup

```bash
git clone https://github.com/ViktorsBaikers/DevRites devrites
cd devrites
npm install            # installs husky + commitlint + semantic-release toolchain
npm run validate       # static validation of pack structure
npm run audit          # known dependency vulnerabilities (moderate+ blocks)
npm test               # install + uninstall smoke + fixture install + pack validation
```

You do not need Claude Code for most development work. The validators and tests
run as plain shell scripts.

To try your changes inside a real project:

```bash
./install.sh --target /path/to/sandbox-project
# poke around, then:
./uninstall.sh --target /path/to/sandbox-project
```

## Project layout (what lives where)

This is the short map. The [README layout section](README.md#layout) has the
full version.

- `pack/.claude/skills/`: canonical public rites, internal specialists, and the `devrites-lib` reference library.
- `pack/.claude/agents/`: fresh-context reviewers plus the write-capable `devrites-slice-wright`.
- `pack/.claude/skills/devrites-lib/reference/standards/`: shared engineering rules loaded by the workflows that need them.
- `evals/`: routing corpora, `golden/` fixtures for the deterministic outcome grader, and `behavioral/` discipline-under-pressure scenarios for gating rites.
- `scripts/`: install lib, validators, eval runner, the outcome grader (`grade-feature.sh` / `run-outcome-evals.sh`), release tooling.
- `docs/`: architecture, skills, command map, flow diagrams, usage, release, CLI.
- `tests/`: auto-discovered shell suite covering install/update/uninstall, runtime behavior, pack validation, and release invariants.

## Authoring guidelines

### Skills (`pack/.claude/skills/<name>/SKILL.md`)

Every skill **must** have:

- YAML frontmatter with `name`, `description`, and `user-invocable` (true/false).
  Model-invoked descriptions carry *Use when* / *Not for* triggers; explicit-only
  descriptions are human summaries. Optional: `argument-hint`, `disable-model-invocation`.
- A short body: operating rules, anti-rationalization tables where useful,
  red flags. **Body discipline:** if it doesn't change the model's behavior
  for this phase, it doesn't belong in the body.
- A **failure-mode section**: a `## Gotchas` (or an equivalent `Hard rules` /
  `NEVER` / `Mid-flight discipline` pointer). Convention: [`docs/skills.md`](docs/skills.md).
- A matching eval file under `evals/`: model-invoked skills need implicit positive and
  negative queries; explicit-only public skills need direct-command positives plus an
  implicit-invocation negative boundary. Corpus size follows the distinct routing branches;
  it is not padded to a fixed query count. `devrites-lib` is exempt.
- For a **gating** skill (one whose job is to hold a line: prove, build, seal, vet,
  peers): a behavioral eval under `evals/behavioral/<skill>.json` that pressure-tests
  whether the discipline resists the rationalizations in its `anti-patterns.md`. Opt-in
  and progressive: not required of every skill; see [`evals/behavioral/README.md`](evals/behavioral/README.md).

Run `python3 scripts/validate-frontmatter.py <files>` (or `npm run validate`) and
`scripts/validate.sh` before pushing.

### Review agents (`pack/.claude/agents/<name>.md`)

- Read-only. No edits, no commits, no network.
- Take a workspace path + diff. **Never the author's reasoning**: that's
  the point of fresh-context review.
- Emit severity-labeled findings: Critical / Important / Suggestion / Nit / FYI.
- One file per agent; keep them focused (Spec vs Standards vs Test vs …).

### Engineering rules (`pack/.claude/skills/devrites-lib/reference/standards/<rule>.md`)

- Stack-agnostic. No language-specific assumptions.
- "Project conventions always win": these are defaults, not laws.
- Add to `pack/.claude/skills/devrites-lib/reference/standards/README.md` index when you add a file.

## Commit message format (strict)

DevRites enforces Conventional Commits via husky + commitlint. Non-conforming
messages are rejected at commit time. There is no bypass.

**Format:** `type(scope): subject`

- **type** (required, lower-case): one of
  `feat | fix | docs | style | refactor | perf | test | build | ci | chore | revert`
- **scope** (required, lower-case): one of
  `skills | rite | devrites | agents | rules | installer | uninstall | scripts | docs | tests | deps | release | repo | ci`
- **subject:** imperative mood, no leading capital, no trailing period.
- **Header length:** 12 to 72 chars total.
- **Body:** blank line after header; lines ≤ 100 chars.

**Valid examples:**

```
feat(skills): add rite-prove browser proof ladder
fix(installer): match first rule pack with leading-space guard
docs(rules): adapt common/agents.md for DevRites agents
refactor(scripts): split sync-version into per-file helpers
```

**Breaking changes:** add `!` after type/scope **or** include a `BREAKING CHANGE:`
footer. Either form triggers a major version bump on the next release.

Full policy: [`commitlint.config.js`](commitlint.config.js).

## Pull request process

1. **Fork** the repo and create a branch off `main`.
   Branch names: `feat/<slug>`, `fix/<slug>`, `docs/<slug>`, etc.
2. **Keep PRs focused.** One logical change per PR. Refactors that touch many
   files should land in their own PR with no behavior change.
3. **Update docs** in the same PR as the change (don't defer to a follow-up).
4. **Run the local checks** listed in [Before you open a PR](#before-you-open-a-pr).
5. **Open the PR** with:
   - A title that matches Conventional Commits (so the squash-merge subject
     drives the release correctly).
   - A description covering: *what* changed, *why*, how it was verified, and
     any follow-ups intentionally left out.
   - Linked issue (`Closes #N`) where applicable.
6. **Address review feedback** with new commits. Do not force-push during
   review; the maintainer will squash on merge.
7. **CI must be green** before merge. CI runs `scripts/validate.sh`,
   install/uninstall smoke, fixture install, commitlint, and the eval suite.

Draft PRs are welcome and encouraged for early feedback.

## Running tests, validators, and evals

```bash
npm run validate                # pack structure + frontmatter
npm run audit                   # dependency advisory gate
npm test                        # install/uninstall + fixture install + validation
bash scripts/run-evals.sh       # run all eval files
bash scripts/run-evals.sh rite-spec   # run a single skill's evals
```

If a test fails locally that you didn't touch, file an issue rather than
working around it.

## Release impact of your commits

Releases are fully automated via semantic-release on every push to `main`:

| Commit prefix | Bump |
|---|---|
| `feat:` | **minor** (e.g. `0.1.0` → `0.2.0`) |
| `fix:` / `perf:` / `refactor:` / `build:` / `docs(README):` | **patch** |
| Any type with `BREAKING CHANGE:` footer or `!` after type | **major** |
| `chore:` / `ci:` / `test:` / `docs:` (non-README) | no release |
| Any scope `(no-release)` (e.g. `feat(no-release): …`) | no release |

If you don't want your change to trigger a release, use a non-release type
or the `(no-release)` scope.

## Reporting security issues

**Do not open public GitHub issues for security problems.** Use the private
disclosure channels documented in [`SECURITY.md`](SECURITY.md):

- Preferred: a private security advisory at
  <https://github.com/ViktorsBaikers/DevRites/security/advisories/new>.
- Alternate: email the maintainer via the contact link on the GitHub profile.

## Getting help

- **Questions about the workflow / skills:** open a GitHub Discussion.
- **Confused about where a change belongs:** open a draft issue.
- **Found a typo or broken link:** open a small PR directly.

Small, well-scoped changes are easier to review and merge.


## Skill and agent contribution preflight

Before adding a DevRites skill or agent:

1. Search the catalog and open work for an existing surface. Prefer extending an existing skill/reference over creating a near-duplicate.
2. Justify why the behavior cannot live as a reference file inside an existing skill.
3. A new public `rite-*` requires command map entries, docs table entry, trigger evals, host parity across Claude (`/rite-*`) and Codex (`$rite-*`), generated artifacts, and reply-contract compliance or a documented exception.
4. A new internal `devrites-*` requires trigger evals and proof it should be a skill rather than an agent or reference.
5. A new agent requires orchestration justification, read/write mode, output format, and composition block. Only `devrites-slice-wright` may write code.
6. Run `npm run validate` and the relevant targeted tests before proposing the change.

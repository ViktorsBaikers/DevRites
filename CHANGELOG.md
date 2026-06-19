# Changelog

All notable changes to DevRites are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and DevRites adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Releases are generated automatically by [semantic-release](https://semantic-release.gitbook.io/) from Conventional Commits on `main`.

## [1.10.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.9.0...v1.10.0) (2026-06-19)

### Features

* **installer:** auto-approve read-only scripts + orient hooks ([3807883](https://github.com/ViktorsBaikers/DevRites/commit/38078838d2f4eeb9303c35c42812bbcb9bd9f5fc))
* **skills:** gate orchestrator out of source edits (A1) ([c64a4a0](https://github.com/ViktorsBaikers/DevRites/commit/c64a4a0f2722089fa938b5291de64eca9e4648c7))

### Bug Fixes

* **skills:** resolve skills-audit findings across pack, evals, and CI ([1499ce2](https://github.com/ViktorsBaikers/DevRites/commit/1499ce274f714544f4983c8e6fc6a11ebe36ee25))
* **tests:** tolerate preserved settings.json in uninstall smoke ([1d1b3d5](https://github.com/ViktorsBaikers/DevRites/commit/1d1b3d56a0b348d8181fa9fae55e70b4f9e51fd9))

### Refactors

* **skills:** fold attempts, drop lanes note, signpost quick ([02180b5](https://github.com/ViktorsBaikers/DevRites/commit/02180b52dfa9254597b3ae0040690f556f40f5f2))

## [1.9.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.8.0...v1.9.0) (2026-06-19)

### Features

* **skills:** enforce test completeness + assertion strength ([4f825a1](https://github.com/ViktorsBaikers/DevRites/commit/4f825a1d1956138a6db778f8b54d7dcafb3876a8))
* **skills:** present gaps as ranked option sets with inline resolve ([db70a21](https://github.com/ViktorsBaikers/DevRites/commit/db70a21511ad3ebc17b32e32afed8b02dcd47a22))
* **skills:** research-driven workflow improvements ([83cb2e0](https://github.com/ViktorsBaikers/DevRites/commit/83cb2e00fb95cdbf974440f098e397ae7c9cde76))

## [1.8.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.7.0...v1.8.0) (2026-06-19)

### Features

* **skills:** add progress footer to all rite-* lifecycle commands ([6b5eda0](https://github.com/ViktorsBaikers/DevRites/commit/6b5eda051ddc86c0a4d67a42d147cf4e9b7fbd17))

## [1.7.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.6.0...v1.7.0) (2026-06-18)

### Features

* **scripts:** add outcome grader and MCP server ([6b7263c](https://github.com/ViktorsBaikers/DevRites/commit/6b7263cf220508dcb567a7db6f79f619ea185249))
* **skills:** add state gates, devrites CLI, and Gotchas sections ([25eaa49](https://github.com/ViktorsBaikers/DevRites/commit/25eaa49e55fce2b22734e8c977dd3a59db73c0d1))

### Documentation

* **docs:** sync README, CONTRIBUTING, SECURITY with changes ([cc89594](https://github.com/ViktorsBaikers/DevRites/commit/cc895942a2a12bcff056ca44c957bf39349f00ca))

## [1.6.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.5.1...v1.6.0) (2026-06-18)

### Features

* **skills:** add shared devrites-lib orientation preamble ([fd8c8fc](https://github.com/ViktorsBaikers/DevRites/commit/fd8c8fc97fb879c3d81190152ad6f31d971a3bde))

### Documentation

* **skills:** document the orientation preamble and devrites-lib ([4cd0d65](https://github.com/ViktorsBaikers/DevRites/commit/4cd0d6575f69a4b20dae6c5ffe01b9e4c232f93a))

## [1.5.1](https://github.com/ViktorsBaikers/DevRites/compare/v1.5.0...v1.5.1) (2026-06-17)

### Refactors

* **skills:** vet every plan at scaled depth, never skip ([ba61859](https://github.com/ViktorsBaikers/DevRites/commit/ba61859dc84871ebe9951dd1dfc68037e58fcb4f))

## [1.5.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.4.0...v1.5.0) (2026-06-17)

### Features

* **skills:** add /rite-vet engineering plan review before build ([8c22b6a](https://github.com/ViktorsBaikers/DevRites/commit/8c22b6af4558540f70853e2d56688f49d528dffa))

## [1.4.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.3.0...v1.4.0) (2026-06-17)

### Features

* **skills:** add /rite-temper strategic spec review ([aa95d4d](https://github.com/ViktorsBaikers/DevRites/commit/aa95d4d27735c654a5bea8daf31fb6bc2dfa1767))

## [1.3.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.2.0...v1.3.0) (2026-06-17)

### Features

* **agents:** add devrites-slice-wright write-capable executor ([66a7c14](https://github.com/ViktorsBaikers/DevRites/commit/66a7c1418b727da181f8294e3bf6fdd632fa5711))

## [1.2.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.1.1...v1.2.0) (2026-06-17)

### Features

* **skills:** plan UX/UI before code via devrites-ux-shape ([e4a45f2](https://github.com/ViktorsBaikers/DevRites/commit/e4a45f27346dc4ab22c5fb7acf215105e947e0a7))

## [1.1.1](https://github.com/ViktorsBaikers/DevRites/compare/v1.1.0...v1.1.1) (2026-06-17)

### Bug Fixes

* **skills:** slice count is always derived, never user-forced ([07eb724](https://github.com/ViktorsBaikers/DevRites/commit/07eb724a785793dd3d38ab4889b202d22d4ca2ef))

### Documentation

* **docs:** /rite-ship in manifest descriptions + autocomplete example ([7d3dfce](https://github.com/ViktorsBaikers/DevRites/commit/7d3dfcee6a9ae99f6ac8d05a6573501c7c8713da))
* **docs:** list /rite-autocomplete in README table, fix /ship alias ([b5613fa](https://github.com/ViktorsBaikers/DevRites/commit/b5613fa922d2c51cc03149d840d316cc12770a76))

## [1.1.0](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.5...v1.1.0) (2026-06-17)

### Features

* **skills:** add /rite-autocomplete — unattended full lifecycle ([acad7e5](https://github.com/ViktorsBaikers/DevRites/commit/acad7e555c4f188665c9a7a5db4dd3dbb4264375))
* **skills:** add /rite-ship — execute the ship + close the task ([cc7db95](https://github.com/ViktorsBaikers/DevRites/commit/cc7db95f079de8da824187a482451a34aa02b852))
* **skills:** sharpen the rite-spec interview loop + coverage gate ([72994ea](https://github.com/ViktorsBaikers/DevRites/commit/72994eab298661b39724af955d5988f47035df03))

### Refactors

* **skills:** seal decides only; git ladder moves to /rite-ship ([80f0fbf](https://github.com/ViktorsBaikers/DevRites/commit/80f0fbfae0a77417283ed35efa4f3c8eae921aa7))

### Documentation

* **docs:** reflect ship/autocomplete + seal-decides split ([e9d4a7e](https://github.com/ViktorsBaikers/DevRites/commit/e9d4a7e64ed8cc4b8b0c742dea055e647d112c57))
* **repo:** fix stale skill count + phantom devrites-rules refs ([3793e1b](https://github.com/ViktorsBaikers/DevRites/commit/3793e1bdf2fecd0c21999c003f5ff09d5b48aef0))

## [1.0.5](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.4...v1.0.5) (2026-06-16)

### Bug Fixes

* **installer:** round-trip rules-only flags, cover update.sh ([0e82a76](https://github.com/ViktorsBaikers/DevRites/commit/0e82a76389ad9787c8f7a776fabffb01de6c49f8))
* **rules:** per-skill core load, dedupe table, validating-gate teeth ([007096e](https://github.com/ViktorsBaikers/DevRites/commit/007096e1f001ac80e1ca6e990afd134d4a8a80cd))
* **scripts:** correct gate tally, add AFK cap + qid scripts ([f2e46e5](https://github.com/ViktorsBaikers/DevRites/commit/f2e46e5155f7ac43dabf28f6db0a34bca84f0b2c))
* **skills:** workspace state, AFK budget, evidence + reviewer scope ([a8e9e75](https://github.com/ViktorsBaikers/DevRites/commit/a8e9e7567ae12b8a5cece961786c196323c1847b))

### Documentation

* **docs:** reconcile counts, fix phantom names and loading model ([f29241e](https://github.com/ViktorsBaikers/DevRites/commit/f29241e9aebd026afdd570d88bba500e9aee8b29))

## [1.0.4](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.3...v1.0.4) (2026-05-28)

### Bug Fixes

* **docs:** quote inside mermaid edge label broke flow.md diagram ([5112ba3](https://github.com/ViktorsBaikers/DevRites/commit/5112ba3be1cfb9f47195e98a5b7d50927662b64f))

## [1.0.3](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.2...v1.0.3) (2026-05-28)

### Bug Fixes

* **installer:** list agents as file array, validate manifest sync ([97d5004](https://github.com/ViktorsBaikers/DevRites/commit/97d50049cdaf4877272e8f555b92b87d1be26887))

### Documentation

* **docs:** bash install is recommended, plugin path is partial ([e729d18](https://github.com/ViktorsBaikers/DevRites/commit/e729d1851498ccda5be7496ffff72d88fd2b4ce7))

## [1.0.2](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.1...v1.0.2) (2026-05-28)

### Bug Fixes

* **installer:** plugin.json must use string repo and ./-prefixed paths ([cb50c01](https://github.com/ViktorsBaikers/DevRites/commit/cb50c01efcd45f813cc9fa7aeee7f25795bfd503))

## [1.0.1](https://github.com/ViktorsBaikers/DevRites/compare/v1.0.0...v1.0.1) (2026-05-28)

### Bug Fixes

* **ci:** repair dependabot, sync README on release, use bot author ([d928765](https://github.com/ViktorsBaikers/DevRites/commit/d928765a965e4c49d1618cf07842e907f28deb0a))

## 1.0.0 (2026-05-28)

### Features

* **repo:** ship DevRites skills pack ([0915d40](https://github.com/ViktorsBaikers/DevRites/commit/0915d40f0c88e81dc9c122f5c755c7975957fdd4))

### Bug Fixes

* **ci:** bypass commitlint on semantic-release commits, tidy README ([0cf52a3](https://github.com/ViktorsBaikers/DevRites/commit/0cf52a3f27216ab5edb8197ec22628d03f2e5e31))
* **ci:** sync lockfile and reject multiline descriptions without PyYAML ([0efa85f](https://github.com/ViktorsBaikers/DevRites/commit/0efa85f052612b94926cf3382f88510059e5a8e8))

# Changelog

All notable changes to DevRites are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and DevRites adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **AFK & HITL run modes — full pause/resume contract.**
  - `pack/.claude/skills/rite-resolve/` — new public skill (`SKILL.md`,
    `scripts/resolve.sh`, `reference/answer-protocol.md`). Canonical writer
    for `questions.md` `status: open → answered` and `state.md` `Awaiting
    human` clearance. Three input shapes: single `<qid> "<answer>"`,
    `--drop <qid>`, `--batch <file>`.
  - `pack/.claude/rules/afk-hitl.md` — master contract: `.devrites/AFK`
    sentinel format, four-gate taxonomy (`advisory` / `validating` /
    `blocking` / `escalating`) with SLAs (`none` / `4h` / `15m` / `24h`),
    irreversible-risk list (destructive migration, auth/authz boundary,
    public API break, external contract, fs destruction, red tests/types/
    lint) that always pauses regardless of `allow_gates`.
  - `pack/.claude/skills/rite-define/reference/gates.md` — gate decision tree
    + per-gate behavior matrix + AFK interaction table.
  - `pack/.claude/skills/rite-build/reference/checkpoint-protocol.md` —
    pre-action render contract + workspace mutations + `notify:` hook env
    contract.
  - `pack/.claude/skills/rite-build/reference/afk-discipline.md` — iteration
    cap, fail-on-red, irreversible-risk list, notify-hook seam.
  - `.devrites/AFK` sentinel (project-root, gitignore by default) toggles
    session-level AFK mode. Optional YAML: `max_slices`, `notify`,
    `allow_gates`. Empty file = AFK with safe defaults
    (`allow_gates: [advisory]`).
- **Schema extensions** (`pack/.claude/skills/rite-spec/reference/state-workspace.md`):
  `state.md` gains `Run mode`, `Status`, `Slice mode`, and an `Awaiting
  human` block when paused. `tasks.md` slice format extended with `Gate`,
  `SLA`, `Checkpoint` (required when `Mode: HITL`). `questions.md` entries
  carry `qid`, `gate`, `status`, `proposed`, `raised_at`, `answered_at`,
  `answer`.
- `/rite-build` workflow updated with:
  - Step 0 — awaiting check (stop and route to `/rite-resolve` when
    `Status: awaiting_human`).
  - Step 2a — HITL pre-action checkpoint (renders gate, persists
    `Awaiting human`, fires `notify:` hook, stops).
  - Step 10 — fail-on-red (red tests/types/lint never marks the slice
    `built`; raises a blocking question).
  - Step 11 — sentinel decrement (`max_slices` ticks down per built slice;
    0 forces HITL stop).
- `/rite-status` surfaces `Run mode`, `Status`, the `Awaiting human` block,
  and open questions broken down by gate (`blocking · validating · advisory
  · escalating`). `scripts/load-state.sh` parses `.devrites/AFK` + tallies
  questions by gate.
- `devrites-doubt` AFK exception: findings at or below the slice's gate
  ceiling downgrade to advisory entries in `questions.md`; irreversible-risk
  findings always pause regardless of AFK config.
- README "Modes — HITL & AFK" section + top-of-doc 2-mode callout + Contents
  link + phase-table `RESUME` row pointing at `/rite-resolve`.
- `docs/usage.md` — new sections "9) HITL gate — pre-code pause and resume"
  and "10) AFK overnight run"; `.devrites/AFK` sentinel row in the workspace
  table.
- `docs/flow.md` — feature lifecycle diagram now shows the `Awaiting human`
  ↔ `/rite-resolve` loop; workspace state-model ER diagram extended with
  `AFK_SENTINEL`, `state.run_mode`/`status`/`awaiting_human`, and
  `questions.qid`/`status`/`gate`; new section 9 "AFK & HITL state
  machine".
- `docs/architecture.md` — new "Run modes — HITL & AFK" subsection +
  Surface bullet updated with `rite-resolve`.
- `docs/command-map.md` — `/rite-resolve` row + updated `/rite-status` row
  (reports run mode + status + question gate breakdown).
- `.claude-plugin/plugin.json` + `.claude-plugin/marketplace.json` for
  installation via `claude plugin install devrites@devrites-marketplace`.
- `SECURITY.md` with private vulnerability-reporting channel, supported
  versions, disclosure window, and documented security model.
- `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1).
- `CODEOWNERS` mandating review on `pack/`, `scripts/`, `install.sh`,
  `uninstall.sh`, `.claude-plugin/`.
- Engineering rules ship under `pack/.claude/rules/` (16 rule files +
  `README.md` index) and are autoloaded natively by Claude Code from
  `.claude/rules/core.md`; on-demand files are read by the phase that needs
  them. No carrier skill — `core.md` is always-on, the rest progressive-disclosure.
- `pack/.claude/agents/devrites-simplifier-reviewer.md` (new) to complete
  the audit-skill subagent set; existing
  `devrites-security-auditor.md` + `devrites-performance-reviewer.md`
  reused for security + performance audits.
- `## Common Rationalizations` (4-6 rite-specific excuse/rebuttal rows) and
  `## Red Flags` (3-5 bullets) sections on `rite-spec` and `rite-define`
  SKILL.md (rollout continues — remaining `rite-*` skills carry the universal
  anti-pattern table at `pack/.claude/rules/anti-patterns.md` plus
  per-phase `reference/anti-patterns.md`).
- Universal anti-rationalization table + red-flags list in
  `pack/.claude/rules/anti-patterns.md`; minimal always-on subset in
  `pack/.claude/rules/core.md`.
- Trigger evals under `evals/<skill>.json` (20 should/should-not-trigger
  queries per public `rite-*` skill) and `scripts/run-evals.sh`.
- `scripts/devrites-detect.sh` — deterministic anti-slop regex detector
  (25-30 rules across UI + code anti-slop).
- `.github/workflows/ci.yml` running `validate.sh`, install/uninstall
  smoke, fixture install, commitlint, eval suite on every PR.
- UI numerical bar — OKLCH-only, 4pt scale, 100/300/500 motion durations
  + 75 % exit, three-axis dark-mode compensation, fluid-vs-fixed type
  scale, container queries, semantic z-index — extended into
  `pack/.claude/skills/devrites-frontend-craft/reference/quality-standards.md`.
- Extended anti-AI-slop list (side-stripe borders, em-dash overuse, pure
  `#000` / `#fff`, all-CAPS body, extended reflex-font reject list) in
  `pack/.claude/skills/rite-polish/reference/anti-ai-slop.md`.
- `NEVER` lists on the UI discipline references (`shape.md`, `craft.md`,
  `design-references.md`).
- `docs/architecture.md`, `docs/command-map.md`, `docs/usage.md` now ship
  with the repo (`docs/` un-gitignored; `docs/internal/` is the new home
  for the previously-local research / development notes).
- Interactive **type-GO** confirmation in `/rite-seal` before irreversible
  git actions (commit, push, tag) — keeps auto-trigger UX while preventing
  accidental side effects.

### Changed

- `/rite-review` output now uses the explicit
  `Critical / Important / Suggestion / Nit / FYI` severity vocabulary.
- `/rite-seal` gate logic: `Critical == 0` proceeds, `Critical > 0` blocks,
  `Important > 0` triggers an interactive `y/N` confirmation. No numeric
  composite score (avoids Goodhart on a self-scoring agent).
- Audit skills consolidated into a single `devrites-audit` skill that
  dispatches the security / perf / simplify reviewer subagent on an axis
  argument — replaces the earlier per-axis skills. Migration from
  `context: fork` to `Task`-tool subagent dispatch closes Anthropic bug
  [#49559](https://github.com/anthropics/claude-code/issues/49559)
  (`context: fork` silently inline under plugin install) and matches the
  `devrites-doubt` → `devrites-doubt-reviewer.md` pattern already in use.
- Public `rite-*` skill descriptions rewritten with explicit
  user-language trigger phrases + `Not for…` negative-scope clauses.
- README install section now documents the plugin path alongside the bash
  installer. Added a "Security model" section pointing at `SECURITY.md`.
- License clarified: personal use and plugin-marketplace **listing** are
  permitted without approval; redistribution / mirroring / commercial
  / organizational use still requires approval.

## [0.1.0] — unreleased

Initial public release line. Tag will be cut once the items above land on
`main`.

[Unreleased]: https://github.com/ViktorsBaikers/DevRites/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ViktorsBaikers/DevRites/releases/tag/v0.1.0

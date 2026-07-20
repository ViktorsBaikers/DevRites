<p align="center">
  <img src="images/logo.png" alt="DevRites">
</p>


**If you only remember one thing:** `SPEC -> DEFINE -> VET -> BUILD -> PROVE -> POLISH -> REVIEW -> SEAL -> SHIP`. Seal decides; Ship mutates git. Build one slice at a time; Autocomplete is opt-in and starts from a clean baseline. DevRites installs via `npx devrites ...`; Claude/Codex support is generated project-local surface, not plugin distribution.

**Per-feature workspace on disk.** Every feature gets its own `.devrites/work/<slug>/`
directory: a compact workspace map → `brief.md` + `spec.md` → (`strategy.md`) →
`architecture.md` + `plan.md` + `tasks.md` + `traceability.md` → (`eng-review.md` +
`test-plan.md`) → `state.md` → `evidence.md` (plus `decisions.md`, `assumptions.md`,
`drift.md`, `questions.md`, `touched-files.md`, `design-brief.md`,
`browser-evidence.md`, `review.md`, `seal.md`, `ship.md`, `handoff.md`, and
`references/`; `strategy.md` is from the optional `/rite-temper`, `eng-review.md` +
`test-plan.md` from `/rite-vet`). When the task ships it is archived intact to
`.devrites/archive/<slug>/`. When you `/clear`, the next agent picks up from those
files — no chat-context summary required. **Spec Drift Guard** catches the wrong turn
before it costs you a day. **AFK mode** runs unattended without silently accepting
destructive migrations, auth changes, or red tests. **`type-GO`** demands a literal
typed confirmation before any irreversible commit / push / tag. **Project principles**
(`.devrites/principles.md`) — the invariants you declare once ("money in integer cents",
"no PII in logs", "never break the v1 API") — then gate every plan, build, and seal, with a
recorded-exception escape hatch instead of silent work-arounds.

```
.devrites/
  ACTIVE                    # which feature is active
  AFK                       # presence = AFK mode; YAML body sets max_slices / notify / allow_gates
  principles.md             # project invariants — authored, prescriptive, GATING (the 4th knowledge layer)
  conventions.md            # learned project idioms — descriptive, an untrusted prior
  learnings.md              # dismissed-finding classes + dead ends — suppresses recurring false positives
  work/<slug>/
    README.md  brief.md  spec.md  references.md  references/  # workspace map + spec
    strategy.md                                          # temper (optional)
    architecture.md  flows.md  plan.md  tasks.md  traceability.md  # define
    eng-review.md  test-plan.md                          # vet
    forge-report.md                                      # build (only a Forge: yes slice — competed candidates → winner)
    state.md  questions.md  decisions.md  assumptions.md  drift.md
    touched-files.md  evidence.md  browser-evidence.md  design-brief.md
    polish-report.md  review.md  seal.md  ship.md  handoff.md
  archive/<slug>/             # shipped task, moved here intact (all .md preserved)
```

**Stop your AI from shipping half-baked code.** DevRites turns Claude Code or Codex into a
disciplined senior engineer — one that asks the right questions before writing a line,
ships features you can actually trust, and refuses to claim "done" without proof.

**Two run modes, same workflow:**

- **HITL** (default, human-in-the-loop) — you're at the keyboard. Slices marked
  `Mode: HITL` pause **before** code is written at a typed checkpoint (`advisory` /
  `validating` / `blocking` / `escalating`); resume on
  [`/rite-resolve <qid> "<answer>"`](pack/.claude/skills/rite-resolve/SKILL.md).
- **AFK** (away-from-keyboard) — drop `.devrites/AFK` in the project. AFK slices run
  unattended; discretionary pauses downgrade to advisory entries in `questions.md` so
  the loop keeps moving. **Destructive migrations, auth/authz changes, public-API
  breaks, and red tests/types/lint always pause regardless** — AFK never silently
  accepts irreversible risk. Optional `max_slices` caps the loop; optional `notify:`
  pings your phone on a pause.

Jump to the full contract → [Modes — HITL & AFK](#modes--hitl--afk).

Every phase is available **two ways**: the menu form `/rite <verb>` (one entry,
discoverable from `/rite`) and the direct shortcut `/rite-<verb>` (muscle
memory). Both hit the same skill — `/rite spec foo` ≡ `/rite-spec foo`.

| # | Phase | Menu form | Direct shortcut | Does |
|---|---|---|---|---|
| 1 | SPEC | `/rite spec` | [`/rite-spec`](pack/.claude/skills/rite-spec/SKILL.md) | investigate + write spec.md |
| — | TEMPER | `/rite temper` | [`/rite-temper`](pack/.claude/skills/rite-temper/SKILL.md) | _optional, big features_ — strategic review: scope mode + pre-mortem, hardens the spec (mandatory in autocomplete) |
| 2 | PLAN | `/rite define` | [`/rite-define`](pack/.claude/skills/rite-define/SKILL.md) | spec → plan + slices (each tagged AFK \| HITL + gate) |
| — | VET | `/rite vet` | [`/rite-vet`](pack/.claude/skills/rite-vet/SKILL.md) | _recommended, every feature_ — engineering plan review: scope · architecture · tests · perf, hardens the plan + writes `test-plan.md`; depth scales to stakes, never skipped (always in autocomplete) |
| 3 | BUILD ×N | `/rite build` | [`/rite-build`](pack/.claude/skills/rite-build/SKILL.md) | one slice, then stop (HITL slices pause pre-code) |
| — | CONVERGE | `/rite converge` | [`/rite-converge`](pack/.claude/skills/rite-converge/SKILL.md) | _recovery_ — assess live code vs intent, append the remaining work as new slices (resumed / adopted / stalled feature) |
| 4 | PROVE | `/rite prove` | [`/rite-prove`](pack/.claude/skills/rite-prove/SKILL.md) | tests + browser proof |
| 5 | POLISH | `/rite polish` | [`/rite-polish`](pack/.claude/skills/rite-polish/SKILL.md) | code + UI polish |
| 6 | REVIEW | `/rite review` | [`/rite-review`](pack/.claude/skills/rite-review/SKILL.md) | multi-axis, parallel |
| 7 | SEAL | `/rite seal` | [`/rite-seal`](pack/.claude/skills/rite-seal/SKILL.md) | GO / NO-GO decision (no git) |
| 8 | SHIP | `/rite ship` | [`/rite-ship`](pack/.claude/skills/rite-ship/SKILL.md) | type-GO + commit/push/tag, then archive + close |
| — | RESUME | `/rite resolve` | [`/rite-resolve`](pack/.claude/skills/rite-resolve/SKILL.md) | answer a HITL gate, clears `Awaiting human`, resumes |
| — | AUTO | `/rite autocomplete` | [`/rite-autocomplete`](pack/.claude/skills/rite-autocomplete/SKILL.md) | run the whole lifecycle unattended (`--ship` to push) |

Common utilities that sit outside the main lifecycle:

| Utility | Menu form | Direct shortcut | Does |
|---|---|---|---|
| POV | `/rite pov` | [`/rite-pov`](pack/.claude/skills/rite-pov/SKILL.md) | project-grounded adopt / switch / reject verdict for an external option |
| DOGFOOD | `/rite dogfood` | [`/rite-dogfood`](pack/.claude/skills/rite-dogfood/SKILL.md) | exploratory browser QA for real user journeys |
| PR FEEDBACK | `/rite pr-feedback` | [`/rite-pr-feedback`](pack/.claude/skills/rite-pr-feedback/SKILL.md) | resolve GitHub PR review threads with evidence |

If implementation reveals the plan is wrong, the **Spec Drift Guard** stops
the build, records the drift, asks you when product behavior changes, and
routes through [`/rite-plan repair`](pack/.claude/skills/rite-plan/SKILL.md)
before resuming.

```mermaid
flowchart LR
    S[/rite-spec/] --> D[/rite-define/] --> B[/rite-build ×N/] --> P[/rite-prove/] --> Po[/rite-polish/] --> R[/rite-review/] --> Sl[/rite-seal/]
    S -.->|big feature| T[/rite-temper/] -.-> D
    D -.->|every feature| V[/rite-vet/] -.-> B
    Sl -->|GO| Sh[/rite-ship/]
    Sh -->|type-GO| Ship([ship: commit · push · tag])
    B -.->|Spec Drift Guard| Re[/rite-plan repair/]
    Re --> B

    classDef phase fill:#1f2937,stroke:#60a5fa,color:#f9fafb
    classDef ship fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef repair fill:#4c1d95,stroke:#a78bfa,color:#f5f3ff
    class S,D,B,P,Po,R,Sl,Sh,T phase
    class Ship ship
    class Re repair
```

Full diagram set (lifecycle, polish orchestrator, review fan-out, debug loop,
rules carrier, workspace state, namespace map) →
[`docs/flow.md`](docs/flow.md).

**Status:** [`v3.0.4`](https://github.com/ViktorsBaikers/DevRites/releases/tag/v3.0.4) — see [`CHANGELOG.md`](CHANGELOG.md) for release notes.

## Contents

- [Why distributed skills](#why-distributed-skills-not-one-engine)
- [Modes — HITL & AFK](#modes--hitl--afk)
- [Install](#install) — [npx / bash](#installing) · [upgrade](#upgrading-an-existing-install)
- [Recommended setup](#recommended-setup-optional-but-devrites-is-much-sharper-with-it) — codegraph · graphify · Playwright MCP
- [Skills](#skills) — 42 total · full catalogue in [`docs/skills.md`](docs/skills.md)
- [Typical workflow](#typical-workflow) · [Worked examples](docs/usage.md)
- [Engineering rules](#engineering-rules) · [Browser proof ladder](#browser-proof-ladder) · [Frontend & fullstack](#frontend--fullstack)
- [Safety & scope](#safety--scope) · [Security model](#security-model)
- [Layout](#layout) · [Community & quality](#community--quality) · [Release pipeline](docs/release.md) · [License](#license)

**Companion docs:**
[architecture](docs/architecture.md) ·
[skills catalogue](docs/skills.md) ·
[command map](docs/command-map.md) ·
[flow diagrams](docs/flow.md) ·
[orchestration](docs/orchestration.md) ·
[usage examples](docs/usage.md) ·
[release pipeline](docs/release.md) ·
[engineering rules](pack/.claude/skills/devrites-lib/reference/standards/README.md) · [quick reference](docs/quick-reference.md)

## Why distributed skills, not one `/engine`

A single command that does everything loads every phase's instructions at once, creates
constant context pressure, and hides the intent of each step. DevRites splits the
lifecycle into 21 small public skills (`rite-*`) that each own one phase and load only
what that phase needs — including [`/rite-autocomplete`](pack/.claude/skills/rite-autocomplete/SKILL.md),
the unattended orchestrator that drives the full cycle end-to-end — plus internal
specialists (`devrites-*`) that fire on triggers.

**Naming:** the `devrites-` prefix is a **namespace** for collision avoidance against
bundled Claude Code skill names (`prototype`, `handoff`, `triage`, `diagnose`, …) —
it does not signal "internal." Visibility is governed by each skill's
`user-invocable:` flag, not by the prefix. See
[`docs/flow.md` § Public vs internal namespace](docs/flow.md#8-public-vs-internal-namespace).

Full rationale: [`docs/architecture.md`](docs/architecture.md).

## Install

DevRites installs **into a project** (project-local only — it never writes to
`~/.claude` or `~/.codex`). Install with `npx` (recommended) or the `curl | bash`
one-liner — both run the same installer and ship Claude Code skills, Codex skills,
agents, **rules**, and aliases.

### Installing

**Fastest — `npx` (Node 18+):**

```bash
# Install the full pack into the current project
npx devrites@latest

# Into a specific project, or preview first
npx devrites@latest --target /path/to/your/project
npx devrites@latest --dry-run

# Upgrade or remove later
npx devrites@latest update
npx devrites@latest uninstall
```

`npx devrites` is a thin wrapper over the same installer below — the pack is bundled in the
package, so the install runs **offline** and is **pinned** to the version you request
(`@latest`, `@1.18.0`, …). It accepts every flag the bash installer does and is still
project-local (it never writes to `~/.claude` or `~/.codex`). Requires `bash` — built in on macOS/Linux;
on Windows run it inside Git Bash or WSL, or use the `curl | bash` one-liner below.

**One-liner over the network** — no `git clone` or Node required:

```bash
# Install latest release into the current directory
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash

# Install into a specific project
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash -s -- --target /path/to/your/project

# Preview (no changes)
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | bash -s -- --dry-run

# Pin to a specific release
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/install.sh | DEVRITES_REF=v0.1.0 bash
```

The script is self-bootstrapping: when piped through `bash` it auto-downloads the latest
release tarball (or the `main` source archive as fallback) into `/tmp` and re-execs from
there. Requires `curl` and `tar`. After bootstrap, install/update/uninstall semantics
run through `devrites-engine`; the shell scripts only acquire a bundle/binary and pass
arguments through. Agent files stay project-local and global agent homes are refused. By
default the engine downloads or builds a verified `devrites-engine` control-plane binary
and writes it to `/usr/local/bin` or `~/.local/bin`; use `--no-binary` or
`DEVRITES_NO_BINARY=1` to skip that global bin install.

**From a local checkout** (same script, no network needed):

```bash
git clone https://github.com/ViktorsBaikers/DevRites devrites && cd devrites
./install.sh --target /path/to/your/project        # or run from inside the project
./install.sh --dry-run                              # preview, change nothing
```

Common flags:

| Flag | Effect |
|---|---|
| `--target DIR` | Install into DIR (default: current directory) |
| `--dry-run` | Show planned file operations and exit |
| `--force` | Overwrite existing non-DevRites files |
| `--no-agents` | Skip the review subagents |
| `--no-codex` | Skip Codex support files (`.agents/skills`, `.codex/agents`, `AGENTS.md`) |
| `--no-skills` | Skip skills and bundled engineering standards |
| `--no-binary` | Skip the global `devrites-engine` control-plane binary |
| `--no-rules` | Deprecated no-op; standards ship inside `devrites-lib` |
| `--rules-only` | Deprecated no-op; installs normally for compatibility |
| `--short-aliases=all` | Add `/define`, `/build`, `/prove`, `/seal` short aliases (off by default) |

Every installed project file is recorded in `.claude/devrites.manifest` (with the
installed version and the original install flags in the header). `./uninstall.sh`
removes exactly those files (and prunes empty dirs) plus the shared `devrites-engine`
binary unless `--keep-binary` is passed — your feature data in `.devrites/work/` and
`.devrites/ACTIVE` is preserved.

### Upgrading an existing install

**One-liner over the network**:

```bash
# Upgrade the install in the current directory
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/update.sh | bash

# Upgrade an install elsewhere
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/update.sh | bash -s -- --target /path/to/proj

# Just check (exit 10 = update available, 0 = current)
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/update.sh | bash -s -- --check
```

From a local checkout:

```bash
./update.sh                          # upgrade install in current directory
./update.sh --target /path/to/proj   # upgrade install elsewhere
./update.sh --check                  # report installed vs latest, change nothing
./update.sh --to v0.2.0              # pin to a specific tag
./update.sh --pre                    # allow pre-release tags
./update.sh --force                  # reinstall even when already current
```

`update.sh` reads the installed version + original flags from
`.claude/devrites.manifest`, asks the GitHub API for the latest release tag,
downloads the release tarball, and re-runs the bundled `install.sh` with the
same flags + `--force`. `.devrites/` (active feature, work) is preserved
because the installer only touches manifest-tracked paths.

### Uninstalling

```bash
# Network one-liner
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/uninstall.sh | bash
curl -fsSL https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main/uninstall.sh | bash -s -- --target /path

# Local checkout
./uninstall.sh                       # remove DevRites from the current project
./uninstall.sh --target /path/to/proj
./uninstall.sh --dry-run             # preview, change nothing
```

Removes only files listed in `.claude/devrites.manifest` and prunes empty dirs,
including Codex mirrors when they were installed.
`.devrites/work/` (your feature data) is always preserved.

## Recommended setup (optional, but DevRites is much sharper with it)

DevRites runs with zero extra tooling, but three tools make it meaningfully better.
**Install and configure them in your project before you start** — DevRites detects each
one and **degrades gracefully** when it's absent.

| Tool | What it gives DevRites | Set up |
|---|---|---|
| **codegraph** | A code-intelligence index. `/rite-spec`, `/rite-define`, and `/rite-plan` use it to understand structure, **placement**, callers, and **impact** cheaply — deeper investigation and sharper specs, at a fraction of the tokens of reading files. | Build the index in your project (e.g. `codegraph init`) so its `codegraph_*` tools / a `.codegraph/` are present. |
| **graphify** | A codebase → knowledge-graph (`graphify-out/`) — same benefit for "where is X / what calls Y / what would Z break". | Generate it for your project (the `/graphify` skill). |
| **[Playwright MCP](https://github.com/microsoft/playwright-mcp)** | Drives a real browser so `/rite-prove` and `/rite-polish` capture **real UI evidence** — screenshots, console, network, responsive — the top rung of the proof ladder (paired with **Chrome DevTools MCP** for Lighthouse / perf trace). | Configure the Playwright MCP server in Claude Code; DevRites detects it, never installs it. |

Without them, DevRites reads files instead of a code graph and uses Claude Code's built-in
`/run`+`/verify` (or documented manual steps) instead of a browser. With them: deeper
investigation, cheaper context, and real browser proof. None are required.

## Skills

The pack ships **42 skills total** — the `rite` menu, 29 user-invocable `rite-*`
workflow + utility skills, 11 model-invoked `devrites-*` specialists, plus the internal
`devrites-lib` reference library. The installed `devrites-engine` owns workflow control;
the npm `devrites` shim owns install/update/uninstall and proxies engine subcommands.
`rite-*` and `devrites-*` are namespaces, not visibility rules: frontmatter is
authoritative. Workspace-operating lifecycle skills read `core.md` in step 0 and disclose
phase rules on demand; compact utilities keep their narrower contract local.

**Claude Code invocation.** Every user-invocable skill responds to **both** `/rite <verb>` (menu form — type `/rite` to discover) and `/rite-<verb>` (direct shortcut — muscle memory). The forms are equivalent: `/rite build slice-2` ≡ `/rite-build slice-2`. Use whichever reads more naturally. Installation merges DevRites event hooks into an existing `.claude/settings.json` without replacing user entries; an existing non-DevRites `statusLine` is preserved with a warning because Claude exposes one status-line slot.

**Codex invocation.** The installer mirrors the same skills to `.agents/skills/`, mirrors DevRites rules to `.agents/skills/devrites-lib/reference/standards/`, injects a Codex compatibility block after each skill's front matter, generates project custom agents in `.codex/agents/`, installs Codex hooks in `.codex/hooks.json`, and creates or merges the needed Codex guidance into `AGENTS.md`. If `AGENTS.md` already exists, DevRites adds a marked block instead of replacing your guidance. In Codex, invoke DevRites via `$rite`, `$rite-spec`, or `/skills`; if you prefer a Claude-only footprint, install with `--no-codex`. Codex must trust the project `.codex/` layer and review the hooks via `/hooks` before non-managed hooks run.

**Rules in Codex.** DevRites engineering rules are mirrored as Markdown under
`.agents/skills/devrites-lib/reference/standards/` because they are workflow/craft
instructions, not Codex command-approval `.rules` files. Generated guidance points at the
mirror; each skill's own contract decides whether `core.md` or a conditional standard loads.

**Custom pinned aliases** (optional). Add your own one-word shortcuts to any `rite-*` skill at runtime with `scripts/pin.sh` — useful for muscle-memory commands like `/b` → `/rite-build` or `/ship` → `/rite-ship`. The wrapper is a thin delegate (same shape the installer uses for `--short-aliases=all`); pinned aliases are manifest-tracked so `./uninstall.sh` cleans them up.

```bash
./scripts/pin.sh add b rite-build      # /b == /rite-build
./scripts/pin.sh add ship rite-ship    # /ship == /rite-ship
./scripts/pin.sh list                  # show currently-pinned aliases
./scripts/pin.sh remove b              # drop the alias
```

Pinned aliases live at `.claude/skills/<alias>/SKILL.md` and mirror to `.agents/skills/<alias>/SKILL.md` when Codex support is installed. The script refuses `rite-*` names, unknown targets, and global agent homes.

### Full skill + agent inventory

**Public `rite-*` skills (29)** — slash-command surface:

| Group | Skills |
|---|---|
| Lifecycle (8) | `rite-spec` · `rite-define` · `rite-build` · `rite-prove` · `rite-polish` · `rite-review` · `rite-seal` · `rite-ship` |
| On-ramp (optional) | `rite-adopt` — onboard an existing codebase: reverse-derive `spec.md`, seed the conventions ledger + propose project principles, then hand off to the lifecycle |
| Strategic (optional) | `rite-temper` — strategic spec review between spec and define; mandatory in `rite-autocomplete` |
| Engineering (every feature) | `rite-vet` — engineering plan review between define and build; depth scales to stakes, never skipped; always in `rite-autocomplete` |
| Resume / replan | `rite-resolve` · `rite-plan` |
| Utility | `rite-status` · `rite-doctor` · `rite-customize` · `rite-pov` · `rite-dogfood` · `rite-pr-feedback` · `rite-zoom-out` · `rite-prototype` · `rite-handoff` · `rite-pressure-test` · `rite-autocomplete` |
| Learning (optional) | `rite-learn` — cross-feature learning loop: mine shipped features for recurring mistakes + dismissed-finding classes, propose project-local lessons into `.devrites/learnings.md`, and promote recurring invariants to `.devrites/principles.md` |
| Menu | `rite` |

**Internal `devrites-*` specialists** — model-invoked, hidden from menu:

`devrites-interview` · `devrites-source-driven` · `devrites-doubt` ·
`devrites-ux-shape` · `devrites-frontend-craft` · `devrites-browser-proof` ·
`devrites-debug-recovery` · `devrites-api-interface` ·
`devrites-audit` (axes: `security` · `perf` · `simplify`) ·
`devrites-prose-craft` · `devrites-refresh-indexes`.

**Review agents (12)** — fresh-context reviewers under `.claude/agents/`:

`devrites-strategy-reviewer` (pre-plan, via `/rite-temper`) ·
`devrites-plan-reviewer` (pre-build, via `/rite-vet`) · `devrites-spec-reviewer` ·
`devrites-code-reviewer` · `devrites-test-analyst` · `devrites-frontend-reviewer` ·
`devrites-security-auditor` · `devrites-performance-reviewer` ·
`devrites-devex-reviewer` (developer-facing surface, predict at `/rite-vet` + measure-the-boomerang at `/rite-seal`) ·
`devrites-doubt-reviewer` · `devrites-simplifier-reviewer` ·
`devrites-forge-judge` (build-time, via `/rite-build` — scores competing candidate builds on a `Forge: yes` slice, picks the winner).

**Cross-feature analyst (1)** — fresh-context, read-only, scope is the archive not a diff:

`devrites-retrospector` — dispatched at `/rite-ship` close (cadence-gated) to mine shipped features for recurring patterns + trends and **draft** graduation candidates for `/rite-learn`; proposes, never imposes.

**Executor agent (1)** — fresh-context, **write-capable** writer under `.claude/agents/`:

`devrites-slice-wright` — dispatched by `/rite-build` to write one slice (orient → TDD → verify, anti-AI-slop, project idiom); the write-side mirror of the read-only reviewers.

Full catalogue with per-phase tables and interactions → [`docs/skills.md`](docs/skills.md). Trigger phrases + interactions → [`docs/command-map.md`](docs/command-map.md). Diagrams (polish orchestrator, review fan-out, seal fan-out, namespace map) → [`docs/flow.md`](docs/flow.md).

## Modes — HITL & AFK

DevRites runs the same lifecycle two ways. The mode is per-slice (declared in
`tasks.md` at planning time) and per-session (`.devrites/AFK` sentinel toggles the
session-level default). Skills consult both.

### HITL — human-in-the-loop (default)

Slices marked `Mode: HITL` pause **before any code is written**. `/rite-build`
renders the checkpoint, writes `Awaiting human` to `state.md`, appends the question
to `questions.md`, and stops. You answer with
[`/rite-resolve <qid> "<answer>"`](pack/.claude/skills/rite-resolve/SKILL.md) and the
workflow resumes. The pause is pre-action by design — no half-built slice, no
"approve after the fact" loop.

Each HITL slice declares a **gate type** that controls how much it disrupts the loop:

| Gate | Stakes | Behavior | SLA |
|---|---|---|---|
| `advisory` | low | log + proceed; surface for audit | none |
| `validating` | medium | async — build continues, merge blocks until reviewed | 4h |
| `blocking` | high | synchronous pause; loop stops | 15m |
| `escalating` | novel pattern | synchronous pause, route to specialist tag | 24h |

Full taxonomy + decision tree:
[`pack/.claude/skills/rite-define/reference/gates.md`](pack/.claude/skills/rite-define/reference/gates.md).

### AFK — away-from-keyboard

Drop `.devrites/AFK` in the project. AFK slices run unattended;
[`devrites-doubt`](pack/.claude/skills/devrites-doubt/SKILL.md) and other
discretionary pauses downgrade to advisory entries in `questions.md` so the loop
keeps moving. The sentinel is plain YAML (all keys optional):

```yaml
# .devrites/AFK — presence = AFK active.
max_slices: 10                       # /rite-build decrements per built slice; 0 → forced HITL stop
notify: "ntfy.sh/my-topic"           # shell command run on awaiting_human; qid / gate / slice in env
allow_gates: [advisory, validating]  # gate severities AFK may auto-handle
```

**AFK never silently accepts irreversible risk.** Regardless of `allow_gates`, the
workflow pauses on: destructive data migration · auth/authz boundary change · public
API break · external-service contract change · filesystem destruction outside the
workspace · red tests / types / lint at slice end. The same `blocking` + `escalating`
gates always pause in AFK too — `allow_gates` only widens what's automatic, not what's
irreversible.

```bash
# Run unattended for the next stretch:
echo 'max_slices: 10' > .devrites/AFK
echo 'notify: "ntfy.sh/my-topic"' >> .devrites/AFK

# Back to HITL:
rm .devrites/AFK
```

Recommended progression: start HITL, refine the prompt and plan over a few slices, then
drop the sentinel for the bulk stretch. Always cap iterations. Full contract:
[`pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md`](pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md).

## Typical workflow

```
# start a feature
/rite-spec add-csv-export     # investigate deeply → spec.md (asks you; gathers design refs)
/rite-define                  # spec → plan.md + tasks.md + state.md

# build loop — one slice at a time
/rite-build                   # slice 1, stops with evidence
/rite-build                   # slice 2 ... repeat for each slice
/rite-prove                   # ONCE all slices built: full tests + browser proof

# finish
/rite-polish                  # code polish always + UI normalize/polish if UI in scope
/rite-review                  # feature-scoped multi-axis (parallel Spec + Standards)
/rite-seal                    # GO / NO-GO decision → writes seal.md (no git)
/rite-ship                    # on GO: type-GO + commit/push/tag, then archive + close the task

# or run the whole cycle unattended
/rite-autocomplete            # spec → … → seal → ship with no per-phase iteration (--ship to push)

# check in any time
/rite                         # menu + next command (no state read)
/rite-status                  # full status: phase, evidence, drift, handoff readiness
```

If implementation reveals the plan is wrong, the **Spec Drift Guard** stops
the build, records the drift in `drift.md`, asks you when product behavior
changes, and routes through `/rite-plan repair` before resuming.

Worked examples (spec-then-plan, mid-build drift, UI feature with
Playwright MCP, backend-only, polish modes, zoom-out, mid-flight handoff):
**[docs/usage.md](docs/usage.md)**.

## Engineering rules

DevRites ships its own stack-agnostic engineering rules and installs them to
`.claude/skills/devrites-lib/reference/standards/` — common rule files plus a README index. They're **common** by design
(no language assumptions); a project's own conventions always win where they exist, and a
project's own **principles** (`.devrites/principles.md`) outrank both. Standards ship inside
the `devrites-lib` skill; the retired `--no-rules` and `--rules-only` flags are compatibility
no-ops. Use `--no-skills` only when you intentionally want to skip skills and the bundled
standards together. Loading model: each `rite-*` skill Reads `.claude/skills/devrites-lib/reference/standards/core.md` as its first
step; the remaining rule files load on demand via `Read` from the skill body that needs them.

| Always-on | On-demand |
|---|---|
| `core.md` | `coding-style.md` · `prose-style.md` · `error-handling.md` · `testing.md` · `spec-grammar.md` · `code-review.md` · `principles.md` · `security.md` · `performance.md` · `observability.md` · `developer-experience.md` · `patterns.md` · `git-workflow.md` · `hooks.md` · `ci-cd.md` · `documentation.md` · `development-workflow.md` · `deprecation.md` · `elicitation.md` · `agents.md` · `context-hygiene.md` · `afk-hitl.md` · `anti-patterns.md` · `tooling.md` · `skill-authoring.md` |

Full index with phase mapping: [`pack/.claude/skills/devrites-lib/reference/standards/README.md`](pack/.claude/skills/devrites-lib/reference/standards/README.md);
diagram: [`docs/flow.md` § Engineering-rules loading](docs/flow.md#6-engineering-rules-carrier).

## Browser proof ladder

For UI work DevRites prefers real runtime evidence, top-down: **Playwright MCP** (if
configured, paired with **Chrome DevTools MCP** for Lighthouse / perf trace) → Claude Code
**`/run`+`/verify`** → **project-native E2E** (Playwright / Cypress / Capybara) → **manual
fallback**. It detects tooling but
never installs it, stops at auth walls, and treats a screenshot **path** as unproven
until it's opened and described.

## Frontend & fullstack

UI work is **planned before it's coded**. When `/rite-spec` detects UI, `devrites-ux-shape`
turns the request + any references (screenshots, Figma, video) into a feature-level
**`design-brief.md`** — design direction (color strategy · scene sentence · named anchor
references), key states, interaction model, and an optional Figma/image **visual-direction
probe** — and pauses for you to confirm the direction (HITL) or asserts a best guess (AFK).
That brief is the **build target**, woven into spec → define → build, not a separate phase.

Then `devrites-frontend-craft` builds **to** the brief: detect the surface register (brand
vs product), refine the brief per slice (all states — default / loading / empty / error /
success / disabled), build from the existing design system, avoid generic-AI tells, and
meet the **2026 quality bar** — Core Web Vitals (LCP ≤ 2.5 s / INP ≤ 200 ms / CLS ≤ 0.1)
and WCAG 2.2 AA. **Fullstack features** go **contract-first**: define the API/data contract, build
one **vertical slice** through the layers (DB → service → API → UI), apply the
engineering rules to the backend and the craft to the UI, map every contract error to a
real UI state, and **prove both layers** (contract tests + browser proof).

## Safety & scope

- **Project-local only.** Never writes to `~/.claude` or `~/.codex`. Manifest-managed install/uninstall.
- **Feature scope only.** Review/simplify/polish/security stay within the active feature
  and touched files — no project-wide refactors, no drive-by cleanup.
- **One slice at a time.** `/rite-build` stops after a single verified slice.
- **Evidence over confidence.** Claims need recorded commands, output, or screenshots.
- **Ask before danger.** Material assumptions, dependency additions, a second design
  system, destructive operations, and product-behavior changes are surfaced, not assumed.

## Layout

```
devrites/
  bin/                 # devrites.mjs — npx CLI entry point (acquires/proxies devrites-engine)
  .github/             # workflows/ (ci, release, dependabot-auto-merge) + dependabot.yml
  .husky/              # commit-msg hook (Conventional Commits via commitlint)
  .releaserc.json      # semantic-release config (CHANGELOG, version sync, tarball, GitHub Release)
  install.sh  uninstall.sh  update.sh  # self-contained bundle/binary bootstrap shims
  scripts/             # install-lib (shim + pin helpers) · validate · validate-frontmatter · run-evals
                       # grade-feature · run-outcome-evals · devrites-detect · check-no-global-writes
                       # sync-version · build-release-tarball
  pack/.claude/        # skills/  42 skills — 30 public + 12 internal          ─┐
                       # agents/  13 read-only + 1 writer (slice-wright)         ├─ the pack
                       # settings.json  (canonical Claude engine hook wiring)   ┘
                       # (standards live inside skills/devrites-lib/reference/standards/)
  installed projects   # .claude/ runtime assets; .agents/skills + .codex/agents
                       # + .codex/hooks.json + AGENTS.md for Codex
  evals/               # trigger evals (20/skill) + golden/ outcome-eval fixtures
  docs/                # architecture · skills · command-map · usage · flow · release · cli
    internal/          # research, development notes (gitignored)
  tests/               # install/uninstall smoke · install fixture · pack validation
  dist/                # release tarballs built by semantic-release (gitignored)
  CHANGELOG.md  SECURITY.md  CODE_OF_CONDUCT.md  CODEOWNERS  NOTICE.md  LICENSE
  package.json  commitlint.config.js   # husky/commitlint/semantic-release toolchain
```

Cross-links: [architecture](docs/architecture.md) ·
[skills catalogue](docs/skills.md) ·
[command-map](docs/command-map.md) ·
[flow diagrams](docs/flow.md) ·
[usage](docs/usage.md) ·
[release pipeline](docs/release.md) ·
[CLI](docs/cli.md) ·
[engineering rules](pack/.claude/skills/devrites-lib/reference/standards/README.md).

## Security model

DevRites is auditable Markdown + a small engine binary. The complete security
policy, including private vulnerability reporting and recommended managed-deployment
settings, lives in [`SECURITY.md`](SECURITY.md). Highlights: **project-local only**
(installer refuses global Claude/Codex agent homes); **bounded installer network access**
for the release tarball and verified `devrites-engine` asset, with `--no-binary` /
`DEVRITES_NO_BINARY=1` available for fully project-file-only installs; **no network
access in skills**; **`!` shell injection removed** from `/rite` and `/rite-status`
(state loaded via read-only engine subcommands over DevRites' own state under
`.devrites/`); **auto-trigger** is a deliberate design choice mitigated by body
discipline + readiness gates + the interactive `type-GO` confirmation in `/rite-ship`
before irreversible git actions; **no `defaultMode: bypassPermissions`** is shipped or
written by the installer (cf. CVE-2026-33068).

## Community & quality

- **Changelog:** [`CHANGELOG.md`](CHANGELOG.md) — Keep-a-Changelog + SemVer, regenerated by semantic-release on every release.
- **Code of conduct:** [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) (Contributor Covenant 2.1).
- **Code owners:** [`CODEOWNERS`](CODEOWNERS) — review required on `pack/`, `scripts/`, `install.sh`, `uninstall.sh`, `bin/`.
- **Notices:** [`NOTICE.md`](NOTICE.md).
- **CI:** GitHub Actions runs `scripts/validate.sh`, install/uninstall smoke, fixture install, commitlint, and the eval suite on every PR.
- **Commits:** Conventional Commits enforced via husky + commitlint.
- **Release pipeline:** semantic-release on every push to `main` — full details in [`docs/release.md`](docs/release.md).

## License

**Free to use, with approval gating redistribution.** Personal use and *listing this
repository in package registries* are permitted without approval. DevRites is installed via
`npx devrites ...`, not Claude/Codex plugin marketplaces. Any other use —
distributing it, distributing modified versions, mirroring as a fork, or commercial /
organizational use — requires **approval on request** (ask via
[the repo](https://github.com/ViktorsBaikers/DevRites)). Source-available. See
[`LICENSE`](LICENSE). DevRites is independent software for use with Claude Code and Codex.

<p align="center">
  <img src="images/logo.png" alt="DevRites">
</p>

# DevRites

DevRites is a repository-local workflow for building software with Claude Code
or Codex. It turns a feature request into a spec, a sliced plan, working code,
recorded proof, a release decision, and an explicit ship step.

The work lives under `.devrites/`, not in chat history. A later session or a
different agent can read the same plan, decisions, state, and evidence before it
continues.

The core path is:

`SPEC -> CLARIFY -> [TEMPER] -> DEFINE -> VET -> BUILD -> PROVE -> POLISH -> REVIEW -> SEAL -> SHIP`

Clarify is mandatory but adaptive, so it asks no questions when the contract
is complete. Temper adds an optional strategic review. Vet is the only final
readiness phase before Build.

Seal decides whether the feature is ready without changing git. Ship first runs
a read-only preflight and discloses the exact Git attempt. A fresh literal `GO`
then authorizes one attempt to collapse eligible checkpoints, stage and validate
the exact candidate, commit and reverify it, perform any approved
project-conventional push, tag, or PR action, and archive the workspace.
Unattended runs may create local WIP checkpoint commits along the way, but they
remain local unless Ship's disclosed plan includes an approved remote action.

**Status:** [`v4.0.10`](https://github.com/ViktorsBaikers/DevRites/releases/tag/v4.0.10): see [`CHANGELOG.md`](CHANGELOG.md) for release notes.

This is the latest published release; `main` may contain unreleased work.

## Quick start

### 1. Install DevRites

Run this from the root of your project. Node.js 18 or later is required.

```bash
npx devrites@latest
```

The installer adds project-local support for Claude Code and Codex. It does not
write skills, agents, or hooks to `~/.claude` or `~/.codex`.

### 2. Choose the smallest route

DevRites scales the process to the change. Start with the least ceremony that
still protects the work:

| Need | Claude Code | Codex |
|---|---|---|
| Small, reversible change | `/rite-quick fix the CSV header typo` | `$rite-quick fix the CSV header typo` |
| New feature or risky behavior | `/rite-spec add-csv-export` | `$rite-spec add-csv-export` |
| Resume an active feature | `/rite-status` | `$rite-status` |
| Baseline an existing codebase | `/rite-adopt` | `$rite-adopt` |

`rite-quick` keeps a compact contract, focused proof, and scope review without
creating the full feature workspace; it escalates when the work is no longer
small, reversible, and unambiguous. `rite-spec` investigates the repository,
asks about gaps, and writes the full feature contract under
`.devrites/work/add-csv-export/`.

### 3. Follow the next recorded step

For workspace-backed routes, use `/rite-status` in Claude Code or `$rite-status`
in Codex. Status reads the active workspace and reports the current phase, open
questions, evidence, and next command. The quick route reports its own focused
proof and next action.

## How the lifecycle works

Claude Code supports both `/rite <verb>` and `/rite-<verb>`. Codex uses the same
forms with `$`: `$rite <verb>` and `$rite-<verb>`. The menu and direct forms run
the same skill.

| # | Stage | Direct command | What happens |
|---:|---|---|---|
| 1 | Spec | [`/rite-spec <feature>`](pack/.claude/skills/rite-spec/SKILL.md) | Inspects the request and codebase, asks about product gaps, and writes a lossless `spec.md` with an explicit capability impact. |
| 2 | Clarify | [`/rite-clarify`](pack/.claude/skills/rite-clarify/SKILL.md) | Checks the whole feature for missing decisions before planning. It asks no questions when everything is clear. |
| 3 | Temper | [`/rite-temper`](pack/.claude/skills/rite-temper/SKILL.md) | Challenges scope and failure modes before Define. It is optional for small work and always runs in `/rite-autocomplete`. |
| 4 | Define | [`/rite-define`](pack/.claude/skills/rite-define/SKILL.md) | Turns the approved spec into architecture, a plan, traceability, and vertical task slices; changed provider/consumer boundaries name one shared contract and consuming tests on both sides. |
| 5 | Vet | [`/rite-vet`](pack/.claude/skills/rite-vet/SKILL.md) | Reviews every plan before implementation. The review depth scales with the risk. |
| 6 | Build | [`/rite-build`](pack/.claude/skills/rite-build/SKILL.md) | In HITL, implements and verifies one product slice, then stops. Run it again for each remaining slice. An explicit `.devrites/AFK` sentinel permits bounded low-risk slice chaining under its cap and pause rules. Exact Vet-ready executable workflow artifacts under the active feature workspace are root-materialized and excluded from product slice accounting; their transaction code is proved in a disposable same-layout fixture before active writes and uses normal bounded recovery rather than a one-shot budget. |
| 7 | Converge | [`/rite-converge`](pack/.claude/skills/rite-converge/SKILL.md) | Runs only when recovery is needed. It compares the code with the recorded intent, adds missing slices, and sends the changed plan back to Vet. |
| 8 | Prove | [`/rite-prove`](pack/.claude/skills/rite-prove/SKILL.md) | Runs positive, discriminating tests, build/runtime checks, and UI proof, then binds the evidence to the exact candidate digest. |
| 9 | Polish | [`/rite-polish`](pack/.claude/skills/rite-polish/SKILL.md) | Cleans up the candidate, normalizes UI when needed, performs durable capability/design/ADR rollups, and refreshes affected proof before closing it. |
| 10 | Review | [`/rite-review`](pack/.claude/skills/rite-review/SKILL.md) | Reviews the closed candidate against its spec and engineering standards and binds the result to its digest. |
| 11 | Seal | [`/rite-seal`](pack/.claude/skills/rite-seal/SKILL.md) | Rechecks candidate-bound evidence and writes the final `GO` or `NO-GO` decision without changing git. |
| 12 | Ship | [`/rite-ship`](pack/.claude/skills/rite-ship/SKILL.md) | Runs read-only preflight and discloses the exact Git plan. After a fresh literal `GO`, it stages and validates the candidate, commits and reverifies it, performs only optional approved push/tag/PR actions, and archives the workspace. |
| n/a | Upgrade *(conditional)* | [`/rite-upgrade [slug]`](pack/.claude/skills/rite-upgrade/SKILL.md) | Audits an older active workspace against current contracts, then routes only evidence-backed defects through their normal phase owners. |

Some work needs a different route:

- [`/rite-quick`](pack/.claude/skills/rite-quick/SKILL.md) handles a small,
  reversible change without creating a full feature workspace.
- [`/rite-autocomplete`](pack/.claude/skills/rite-autocomplete/SKILL.md) runs
  the reversible lifecycle unattended. With `--ship`, it continues through
  Ship preflight, discloses the exact Git plan, and waits for a fresh literal
  `GO` plus native approval; without that flag, it stops at Seal GO. A failed
  consumptive proof action never retries blindly: retained evidence drives
  offline repair and re-vetting. Ambiguous retained evidence first drives a vetted
  boundary-discriminating diagnostic design; only its next real acquisition attempt
  needs a new GO. After an upgrade introduces a supported workflow-artifact writer,
  Autocomplete reopens a stale missing-writer stop once instead of preserving the
  obsolete recovery count. The first real root-materializer failure then counts as
  attempt one under normal fingerprint recovery; it is not terminal by itself.
- [`/rite-upgrade [slug]`](pack/.claude/skills/rite-upgrade/SKILL.md) is a
  compatibility route for an older active workspace that cannot resume. Age or
  cursor form alone never triggers repair; it is not a lifecycle phase.
- [`/rite`](pack/.claude/skills/rite/SKILL.md) shows the command menu.

Run `devrites-engine update` from an installed project to acquire the latest
stable release and update both the engine and pack. `npx devrites@latest update`
and the verified release `install.sh` remain equivalent adapter routes. If an
older engine instead reports `missing codex/hooks.json` or asks for
`--source-dir`, use the npm or verified shell route once to cross to a release
with the self-contained updater.
`/rite-upgrade` is the separate native, preservation-first route for reconciling
an unfinished workspace. It proves a current-contract defect before routing
Clarify, Plan repair, Converge, Vet, Prove, Polish, Review, or Seal; it never
migrates cursor format or invents historical proof. The engine has no workspace
migration command. See the [CLI contract](docs/cli.md).

The [command map](docs/command-map.md) covers every command, trigger, input, and
output. The [worked examples](docs/usage.md) show normal features, plan drift,
UI work, backend work, and mid-flight handoffs.

## What DevRites records

Each feature gets a directory under `.devrites/work/<slug>/`. Shipped
workspaces move intact to `.devrites/archive/<slug>/` so future work can inspect
the original decisions and proof.

```text
.devrites/
  ACTIVE                 # active feature slug
  AFK                    # optional unattended-mode configuration
  CHECKPOINT             # optional local WIP checkpoint mode
  principles.md          # project rules that gate the workflow
  specs/                 # living structured capabilities
  work/<slug>/
    brief.md
    spec.md
    decision-coverage.md
    architecture.md
    plan.md
    tasks.md
    eng-review.md
    test-plan.md
    state.md
    decisions.md
    questions.md
    traceability.md
    evidence.md
    touched-files.md       # strict project-candidate manifest
    review.md
    seal.md
    ship.md
  archive/<slug>/
```

Some phases add focused artifacts such as `strategy.md`, `design-brief.md`,
`browser-evidence.md`, `polish-report.md`, `drift.md`, or `handoff.md`. See the
[workspace contract](docs/engine/workspace-schema.md) for the full state model
and [candidate integrity](docs/candidate-integrity.md) for the Build-to-Ship
content binding.

## Safety rules

- **Settle before code.** Spec and Clarify settle the behavior contract. Define
  and Vet settle the implementation path before Build starts. The active skill
  and exact native reviewers judge `CLEAR`, `READY`, traceability, and test
  quality from the complete artifacts; the engine checks phase-relative
  structure and, after Vet, the exact stable Build-input binding.
- **Bound every Build dispatch.** In HITL, `/rite-build` implements one vertical
  slice and records its proof before returning control. With an explicit
  `.devrites/AFK` sentinel it may chain low-risk slices only within the
  configured cap and pause rules. Before every writer dispatch, the root puts
  the exact project-relative paths in its task; after return, it rejects any
  extra path in `git diff --name-only`, reviews test integrity, and runs
  repository proof.
- **Classify drift before routing.** The Spec Drift Guard records the mismatch
  in `drift.md`. Build handles objective implementation and tool failures with
  bounded recovery; it uses
  [`/rite-plan repair`](pack/.claude/skills/rite-plan/SKILL.md) only when the
  durable plan is wrong, and asks you only for a real product or risk decision.
- **Prove claims.** Behavioral proof must be positive and discriminating:
  skipped, zero-test, assertion-free, tautological, unexecuted, or exit-only
  results do not prove behavior, and static gates prove only their static
  criterion. A screenshot path by itself is not proof.
- **Separate the decision from the action.** Seal makes the release decision.
  Ship performs the final git actions only after the seal passes and you type
  `GO`.
- **Stay inside the feature.** Review, security, simplification, and polish do
  not expand into unrelated project cleanup.

Validated project principles and nearest-scope repository instructions govern
product and technical choices inside the controlling request. They do not grant
permission or waive DevRites safety, source-writing, or evidence gates; current
source/tests describe reality rather than authority. A feature can record a
deliberate scoped exception instead of silently ignoring a project principle.
The canonical order is
[`core.md` § Precedence](pack/.claude/skills/devrites-lib/reference/standards/core.md#precedence).

## HITL and AFK modes

### HITL is the default

When DevRites needs a decision, it presents ranked options, records the chosen
answer in the workspace, and continues. If the current turn must stop, the
question remains in `questions.md`; answer it later with
[`/rite-resolve <qid> "<answer>"`](pack/.claude/skills/rite-resolve/SKILL.md).
Codex uses the equivalent `$rite-resolve` form.

### AFK runs the safe part unattended

Create `.devrites/AFK` to let permitted gates select their recommended option
and continue:

```yaml
max_slices: 10
allow_gates: [advisory]
```

The workflow treats this file as configuration and never rewrites it.
`max_slices` seeds the remaining budget in the active feature's `state.md`; the
root charges that state exactly once after each green built slice and stops
before another dispatch at zero. Malformed counters fail closed. Delete `.devrites/AFK` to
return to HITL.

AFK still pauses for product, scope, or policy choices, irreversible risk,
and access or actions available only to a human. Agents use bounded recovery
for red tests, type or lint errors, runtime failures, and missing technical
coverage. If that recovery budget runs out, they record a technical blocker
instead of asking a question. The full pause and gate contract is in
[`afk-hitl.md`](pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md).

## Install, update, and remove

### npm

```bash
# Install in the current project
npx devrites@latest

# Install elsewhere or preview the changes
npx devrites@latest --target /path/to/project
npx devrites@latest --dry-run

# Update or remove an existing installation
npx devrites@latest update
npx devrites@latest update --check
npx devrites@latest uninstall
npx devrites@latest uninstall --keep-binary

# The installed engine can update itself and the project pack directly
devrites-engine update
devrites-engine update --check
```

Useful install flags:

| Flag | Effect |
|---|---|
| `--target DIR` | Use another project directory. |
| `--dry-run` | Show planned file operations without changing anything. |
| `--force` | Replace or remove foreign or customized managed files. The installer still rejects symlinks and path escapes. |
| `--no-codex` | Skip `.agents`, `.codex`, and `AGENTS.md` integration. |
| `--no-agents` | Skip hook-free native specialist profiles. |
| `--no-skills` | Skip skills and their bundled standards. |
| `--no-binary` | Do not keep the shared `devrites-engine` binary in a user or system bin directory. |
| `--short-aliases=all` | Add `/define`, `/build`, `/prove`, and `/seal` aliases. |

Run `npx devrites@latest --help` for common flags and
`npx devrites@latest <command> --help` for command-specific flags.

### Bash bootstrap

If Node.js is not available, download the release-owned installer and its
checksum before executing it. It needs `curl`, `gzip`, and `tar`.

```bash
bootstrap_dir="$(mktemp -d)"
(
  set -e
  trap 'rm -rf "$bootstrap_dir"' EXIT HUP INT TERM
  cd "$bootstrap_dir"
  release=https://github.com/ViktorsBaikers/DevRites/releases/latest/download
  curl -fL --proto '=https' --proto-redir '=https' --connect-timeout 10 --max-time 60 --max-filesize 1048576 "$release/install.sh" | head -c 1048577 > install.sh
  install_status="${PIPESTATUS[0]}"
  [ "$(wc -c < install.sh)" -le 1048576 ] || { echo 'error: install.sh exceeds 1 MiB' >&2; exit 1; }
  [ "$install_status" -eq 0 ] || { echo 'error: install.sh download failed' >&2; exit 1; }
  curl -fL --proto '=https' --proto-redir '=https' --connect-timeout 10 --max-time 30 --max-filesize 4096 "$release/install.sh.sha256" | head -c 4097 > install.sh.sha256
  sidecar_status="${PIPESTATUS[0]}"
  [ "$(wc -c < install.sh.sha256)" -le 4096 ] || { echo 'error: install.sh.sha256 exceeds 4 KiB' >&2; exit 1; }
  [ "$sidecar_status" -eq 0 ] || { echo 'error: install.sh.sha256 download failed' >&2; exit 1; }
  want="$(awk '
    NF == 0 { next }
    { records++ }
    NF == 2 && length($1) == 64 && $1 ~ /^[0-9A-Fa-f]+$/ && $2 == "install.sh" { valid++; hash=tolower($1) }
    END { if (records == 1 && valid == 1) print hash; else exit 1 }
  ' install.sh.sha256)"
  if command -v shasum >/dev/null 2>&1; then
    got="$(shasum -a 256 install.sh | awk '{print $1}')"
  elif command -v sha256sum >/dev/null 2>&1; then
    got="$(sha256sum install.sh | awk '{print $1}')"
  else
    echo 'error: shasum or sha256sum is required' >&2
    exit 1
  fi
  [ "$got" = "$want" ] || { echo 'error: install.sh checksum mismatch' >&2; exit 1; }

  # Choose one Node-free operation:
  bash ./install.sh                         # install here
  # bash ./install.sh --target /path/to/project
  # bash ./install.sh --dry-run
  # bash ./install.sh update
  # bash ./install.sh uninstall
)
```

To pin a named release, replace `latest/download` with
`download/v<version>` and run the chosen operation with
`DEVRITES_REF=v<version>`. Existing local or extracted `install.sh`,
`update.sh`, and `uninstall.sh` invocations remain compatible. The exact-release
acquisition guarantee begins with the verified release `install.sh`; mutable
default-branch scripts are not an installation boundary.

The installer records every managed project file and its SHA-256 in
`.claude/devrites.manifest`. Before an update or uninstall changes anything, it
checks every managed path. It preserves customized files and tells the user to
retry with `--force` before making any changes; legacy manifests without hashes
also require `--force`.
`--force --dry-run` lists the exact destructive actions. Marker-merged files
keep user content outside the DevRites block, and `.devrites/` runtime state
remains in place. The installer refuses symlinks, junctions, and paths that
escape the target.

The optional shared `devrites-engine` binary is the only artifact installed
outside the project. Before replacement, the release binary at its staged path
must report the requested version. After installation, the binary at its final
path must report that version in a new process. The installer keeps a backup in
the same directory until the second check passes. On failure, it restores the
previous binary or removes a bad first install.

Direct `devrites-engine update` resolves the latest stable release, acquires its
bundle and platform binary, and hands local paths to the downloaded engine. The
candidate engine therefore validates its own payload before replacing the
installed pack and binary. npm and Bash may supply the same local candidate
paths directly. `update --check` resolves and compares the latest version
without downloading release assets.

Network acquisition resolves an exact SemVer release, permits HTTPS at every
redirect hop, requires an exact-filename SHA-256 sidecar, and uses private
temporary directories plus in-stream bounded downloads. Archive metadata and
paths pass bounded streaming preflight before extraction. There is no unchecked
raw, source-archive, tag, or default-branch fallback. See
[`SECURITY.md`](SECURITY.md) for the representative bounds.

## Claude Code and Codex integration

Claude Code receives skills under `.claude/skills/`, agents under
`.claude/agents/`, and native root permissions merged into
`.claude/settings.json`. Only Claude's slice-wright profile is writable.
Existing settings remain in place.

Codex receives the same skills under `.agents/skills/`, custom agents under
`.codex/agents/`, native permissions under `.codex/config.toml`, and a marked
guidance block in `AGENTS.md`. All Codex specialists are hook-free and
only `devrites-slice-wright` is writable; every other specialist is read-only.
Existing user content remains in place. Codex users invoke skills with `$rite`,
`$rite-spec`, or `/skills`.

DevRites is installed through npm or the Bash bootstrap. It is not distributed
through Claude Code or Codex plugin stores.

## Skills and agents

The pack ships 43 skills: 32 public and 11 internal. The public surface contains
the `rite` menu and 31 `rite-*` workflows and utilities. Ten `devrites-*`
specialists load when a matching task needs them; `devrites-lib` carries the
shared contracts and engineering standards.

Seventeen fresh-context agent profiles ship with the pack. Claude has sixteen
read-only roles plus the sole source/test writer role,
`devrites-slice-wright`; Codex generates the same one-writer/sixteen-reader split.

The authoritative [skills catalogue](docs/skills.md) lists every skill and
agent. The [flow diagrams](docs/flow.md) show routing, reviewer fan-out, and
namespace boundaries.

## Engineering standards and UI work

The stack-agnostic standards live under
`.claude/skills/devrites-lib/reference/standards/`. Workspace-operating
lifecycle skills load the small core first, while compact utilities keep a
narrower local contract; each then loads only the standards it needs. Nearest
project instructions override generic advice, and ratified
`.devrites/principles.md` are gating invariants. The
[standards index](pack/.claude/skills/devrites-lib/reference/standards/README.md)
maps each rule file to the phases that use it.

For UI work, Spec records the visual direction and required states in
`design-brief.md`. Build follows the project's existing design system, while
Prove collects runtime and browser evidence. The guidance checks WCAG 2.2 AA
and, when performance is measurable, targets LCP at or below 2.5 seconds, INP
at or below 200 milliseconds, and CLS at or below 0.1. Full-stack work starts
with the API and data contract, then builds a vertical slice through the UI.

DevRites works without extra tools. If available, codegraph or graphify can
answer structural questions, and
[Playwright MCP](https://github.com/microsoft/playwright-mcp) can collect
browser evidence. Chrome DevTools MCP can add Lighthouse and performance
traces. DevRites detects these tools but does not install them.

## Security model

Installed host artifacts stay in the project. The npm shim can use an explicitly
configured engine, download the exact release binary with its mandatory checksum,
build a temporary engine from package-local Go source, or use `devrites-engine`
on `PATH`. Remote fetches are HTTPS-only and bounded. Use
`--no-binary` or `DEVRITES_NO_BINARY=1` to avoid keeping a shared binary outside
the project. npm, Bash, and direct engine update all require checksummed release
assets. Engine network access is isolated to bounded latest-release acquisition;
the downloaded engine performs the local manifest-owned update. Install, update,
and uninstall do not inspect target-project Git. Retained safety operations such
as `secret-scan --staged` intentionally read the exact Git index and staged blobs
they validate.

Production Git subprocesses remove repository/config/object/ref/pathspec
retargeting `GIT_*` variables while preserving unrelated Git environment. Seal
binds proof, review, and verdict to the strict project candidate rather than
modification times. See [candidate integrity](docs/candidate-integrity.md) and
[ADR-0026](docs/adr/0026-content-bound-proof-and-bounded-inputs.md).

Skills read workspace Markdown directly. Only retained atomic resolution/close
and safety mutations go through the engine; root-owned policy edits follow the
explicit native checklists. The installer never writes
`defaultMode: bypassPermissions`. Skills use networked research only through
host tools you invoke or configure.

Read [`SECURITY.md`](SECURITY.md) for the threat model, managed deployment
guidance, and private reporting instructions.

## Repository layout

```text
bin/               npm CLI shim
engine/            Go control plane and tests
pack/.claude/      canonical skills, agents, permissions, and standards
pack/generated/    generated host payloads; do not edit by hand
scripts/           validation, generation, install, and release tooling
tests/             repository-level shell tests
docs/              architecture, usage, command, and contributor guides
evals/             routing and behavioral evaluation fixtures
```

For local development:

```bash
npm install
npm run validate
npm test
(cd engine && go test ./... -count=1)
```

Canonical pack changes belong in `pack/.claude/`. Rebuild host payloads with
`bash scripts/build-host-artifacts.sh` instead of editing `pack/generated/`.
See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the development workflow and review
requirements.

## Documentation

- [Quick reference](docs/quick-reference.md)
- [Worked examples](docs/usage.md)
- [Command map](docs/command-map.md)
- [Skills catalogue](docs/skills.md)
- [Architecture](docs/architecture.md)
- [Lifecycle diagrams](docs/flow.md)
- [Candidate integrity](docs/candidate-integrity.md)
- [Engine CLI](docs/cli.md)
- [Release process](docs/release.md)
- [Nine-source workflow benchmark snapshot (2026-08-01; historical, non-authoritative)](docs/upstream-workflow-benchmark-2026-08-01.md)
- [Markdown instruction upgrade snapshot (2026-08-02; historical, non-authoritative)](docs/markdown-instruction-upgrade-2026-08-02.md)
- [ADR-0026: content-bound proof and bounded inputs](docs/adr/0026-content-bound-proof-and-bounded-inputs.md)
- [Changelog](CHANGELOG.md)
- [Releases](https://github.com/ViktorsBaikers/DevRites/releases)

## Community and license

Please read the [contribution guide](CONTRIBUTING.md) and
[code of conduct](CODE_OF_CONDUCT.md) before opening a change. Maintainer
ownership is documented in [`CODEOWNERS`](CODEOWNERS), and third-party notices
are in [`NOTICE.md`](NOTICE.md).

DevRites is source-available software for personal use with Claude Code and
Codex. Distribution, modified distribution, fork mirrors, and commercial or
organizational use require approval. Read [`LICENSE`](LICENSE) for the full
terms.

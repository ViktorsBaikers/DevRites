# Security policy

## Reporting a vulnerability

If you believe you have found a security issue in DevRites, please report it
**privately**. Do not open a public GitHub issue.

- **Preferred channel:** open a private security advisory on GitHub:
  <https://github.com/ViktorsBaikers/DevRites/security/advisories/new>.
- **Alternate channel:** email the maintainer via the contact link on the
  GitHub profile <https://github.com/ViktorsBaikers>.

Please include:

- A clear description of the vulnerability and its impact.
- A minimal reproduction (commands, files, or a target project layout).
- The DevRites version (commit SHA or release tag) and Claude Code version.
- Your name / handle for credit (optional).

You should receive an acknowledgement within **5 business days** and a triage
verdict within **14 days**. Coordinated disclosure window is **90 days** from
the acknowledgement unless a shorter or longer window is mutually agreed.

## Supported versions

Only the latest published DevRites release receives security fixes. Older
releases are unsupported; upgrade before reporting a version-specific finding
unless the issue also reproduces on the latest release.

## DevRites security model

### Scope

DevRites is a skills pack plus the local `devrites-engine` control-plane binary:
Markdown skill files, helper scripts, Bash bootstrap shims, and a native Node
`npx` shim (`bin/devrites.mjs`) that acquires and proxies the engine. It ships no
network service. The attack surface is the content of the skill files, bootstrap
and install paths, generated host artifacts, hooks, and the local engine binary.

### Supply-chain self-scan (shipped pack)

Because the skill files **are** the attack surface, CI scans the shipped pack
(`pack/.claude/`) on every PR and fails the build (blocking, not advisory) on:

- **Prompt-injection patterns**: the "ignore previous instructions" family,
  system-prompt overrides, permission-escalation, and data-exfiltration phrasing.
- **Hidden unicode**: bidi controls, zero-width characters, and homoglyph
  confusables (a word mixing ASCII with look-alike Cyrillic/Greek letters) that a
  human reviewer can't see in a diff.

Run it locally with `python3 scripts/scan-pack-security.py pack/.claude`. A
finding prints as `FINDING <file>:<line>: <class>: <excerpt>`. If the match is
DevRites' own *defensive* content (e.g. a rule that quotes an attack string, or a
QA checklist that demonstrates an adversarial character), add an auditable
suppression on the same line with `<!-- pack-scan-ignore: <reason> -->`. To opt
an entire file out of one class, use
`<!-- pack-scan-ignore-file: injection -->`.
Suppressions live in the file, so every exception is visible in the diff and
reviewable; never suppress a hidden-unicode finding you can't explain.

### State loading (engine subcommands, no `!` injection)

`/rite-status` and workspace-operating skills load state by running a **read-only
`devrites-engine` subcommand through the `Bash` tool**. DevRites does not use
Claude Code's preprocessing-only `` !`<command>` `` dynamic-context injection
because that mechanism is not portable across harnesses. The no-argument `/rite` menu runs
`devrites-engine first-task` instead; a routed verb hands control to its owning skill.

```bash
command -v devrites-engine >/dev/null 2>&1 && devrites-engine preamble || echo "(unavailable: read state.md directly)"
```

`devrites-engine preamble` reads the project's `.devrites/` state. It does not
concatenate user input into a command, access the network, or write files. The
gate subcommands (`build-readiness`, `evidence-fresh`, `check-acceptance`) are
also read-only. Mutating commands are explicit and
scoped: for example, `resolve`, `tick-afk`, and `close-out` write only DevRites
state, while `/rite use <slug>` deliberately repoints `.devrites/ACTIVE` inline.

If your environment disallows shell commands started by skills, each skill reads
`state.md` directly through the `|| echo "(… unavailable …)"` fallback above.
The rest of the pack continues to work.

### User and model invocation are separate

Public skills are always user-invocable, but model invocation is set per skill.
Skills without `disable-model-invocation: true` may be selected when their
descriptions match the user's intent; explicit utilities carry that frontmatter
flag and load only when the user invokes them. The checked-in frontmatter and
[`docs/command-map.md`](docs/command-map.md) are the source of truth. The safety
nets for model-invocable rites are:

- **Body discipline**: every skill stops at its phase boundary. `rite-build`
  stops after one slice, `rite-prove` runs proofs only when all slices are
  built, etc.
- **Readiness gates**: each rite reads `.devrites/work/<slug>/state.md`
  before acting; phases out of order refuse to run.
- **Spec Drift Guard**: any deviation from the spec halts and routes to
  `rite-plan`.
- **Interactive type-GO confirmation** in `rite-ship` before irreversible
  git actions such as commit, push, or tag. This prompt remains after model
  invocation; `rite-seal` only decides GO/NO-GO.

Claude documents the invocation controls in its official
[skills reference](https://code.claude.com/docs/en/slash-commands). DevRites uses
`disable-model-invocation: true` rather than an undocumented settings key.

### Installer safety

The Bash installer (`install.sh`) refuses any target under `~/.claude`
(`GUARD:no-global` block). Project host files are manifest-tracked;
`./uninstall.sh` removes exactly those managed files. The
`.devrites/` runtime state in the target project is preserved across
uninstall.

Skills, agents, standards, and hook configuration stay in the target project.
The installer merges DevRites entries into project-local `.claude/settings.json`
and `.codex/hooks.json` without replacing unrelated user settings. The shared
`devrites-engine` executable is the only allowed artifact outside the project.
It is installed to `DEVRITES_BIN_DIR`, a writable `~/.local/bin`, or a writable
`/usr/local/bin`; `--no-binary` / `DEVRITES_NO_BINARY=1` skips it. The bootstrap
path may fetch the release bundle and checksummed engine assets. It never invokes
`sudo` or edits shell startup files.

### npx install path

With `npx devrites@latest`, the CLI (`bin/devrites.mjs`) calls
`devrites-engine` directly instead of running `install.sh`. The bundled host
payload is pinned to the requested npm package version. The shim first tries the
matching release binary and its SHA-256 sidecar, then a local Go build, and then
an existing engine. It has no runtime npm dependencies. The same project and
shared-binary boundaries apply to the Bash installer.

### Recommended Claude Code permissions for managed deployments

For organizations evaluating DevRites under a managed Claude Code policy:

```jsonc
{
  "permissions": {
    "ask": [
      "Bash(git commit *)",
      "Bash(git push *)",
      "Bash(git tag *)"
    ]
  }
}
```

Under the current
[Claude Code permissions schema](https://code.claude.com/docs/en/permissions),
the host asks for confirmation before the git mutation ladder. DevRites'
separate type-`GO` workflow gate still applies.

### Hooks (approval, orientation, and local guards)

The engine installs JSON-configured hooks in the project-local host artifacts:
`.claude/settings.json` for Claude Code and `.codex/hooks.json` for Codex. Both
files call `devrites-engine` behind an inline fail-open guard:

- **`allow` (PreToolUse/Bash)**: auto-approves *only* the read-only engine
  orientation/gate subcommands (`check-acceptance`, `doubt-coverage`,
  `evidence-fresh`, `preamble`, `progress`, `readiness`, `review-integrity`),
  `footprint render|roster`, `ledger diff|validate|list|show`, and
  `reviewer-stats report`, so they stop prompting on every skill run.
  It never denies, and it emits `allow` only when the parsed command is one of those
  subcommands and contains no dangerous/exfiltration tokens (`rm`, redirects,
  `curl`/`wget`, `sudo`, `chmod`, command substitution, `eval`, package managers,
  `git push/commit/reset`, etc.). Mutating subcommands (`resolve`, `tick-afk`,
  `close-out`) are deliberately excluded and still prompt.
- **Context and continuity hooks**: `orient`, `cursor`, `subagent-orient`,
  `statusline`, and `handoff-snapshot` inject bounded workspace context; `event`
  and `auq` append lifecycle/HITL events. They stay silent when no workspace is active.
- **Local guards and caches**: `a1-guard`, `wright-scope`,
  `reviewer-readonly`, `redwatch`, `stop-gate`, `source-cache-*`, and
  `refresh-indexes` run through the engine. Guard hooks are fail-open or
  observe-first unless explicitly enforced with the documented `DEVRITES_*`
  controls; source-cache network I/O is the bounded exception in ADR-0008.

Delete the project-local hook file (`.claude/settings.json` for Claude Code, or
the DevRites-managed entries in `.codex/hooks.json` for Codex) to remove hooks.
Updates merge DevRites entries rather than replacing unrelated project settings,
so user permission rules remain intact.

### Third-party trust

DevRites vendors no third-party code (see `NOTICE.md`). It depends on Claude
Code itself. Codegraph, graphify, and Playwright MCP are optional user-selected
tools that DevRites calls through their documented interfaces rather than
bundling them.

### Agentic trust boundaries

Treat every instruction-bearing file as a supply-chain input:

1. **Shipped pack**: `pack/.claude/**`, generated host artifacts, hooks, and the
   engine are release-managed and scanned before publish.
2. **Project-local state**: `.devrites/work/**`, learnings, unvalidated
   principles, and review artifacts are evidence, not authority. Live source and
   engine gates win over stale state.
3. **User extensions/overrides**: `.devrites/extensions/**` and
   `.devrites/overrides/**` are untrusted until `devrites-engine extensions
   validate` / `overrides validate` pass. Extensions may add checks or reviewers;
   they must not weaken type-GO, seal/ship, AFK/HITL, security, or evidence
   gates.
4. **External capability configs**: MCP/tool configs are optional and
   project-local. `/rite-doctor`/`devrites-engine doctor` reports readiness, but
   missing tools degrade to file-system/engine gates instead of silently changing
   workflow semantics.

<!-- authority:principles-trust:start -->
Project principles may become project policy only after explicit provenance and validation; arbitrary project-local Markdown is never inherently trusted executable instruction.
<!-- authority:principles-trust:end -->

Never copy untrusted issue text, web content, or model output into a skill,
agent, hook, MCP config, or generated artifact without reviewing it as executable
instructions. Hidden unicode, prompt-injection phrasing, personal absolute paths,
and secret-like strings are release-blocking findings unless explicitly and
visibly justified.

### Known non-issues

- **Historical `!` injection in `/rite`**: no longer present; current state
  orientation uses structurally bounded, read-only engine commands.
- **`Write` / `Edit` tool allowance in `rite-*` skills**: required to
  author `.devrites/` and project files. No skill grants `Bash(*)`.
- **Per-skill model invocation**: deliberate and frontmatter-controlled;
  model-invocable rites remain bounded by the gates above.

### CVE relevance

DevRites does not ship `permissions.defaultMode: bypassPermissions` in any
file (cf. **CVE-2026-33068**, workspace-trust bypass via committed bypass
mode). The installer refuses to write any `defaultMode: bypassPermissions`
into target projects.

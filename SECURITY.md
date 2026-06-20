# Security policy

## Reporting a vulnerability

If you believe you have found a security issue in DevRites, please report it
**privately**. Do not open a public GitHub issue.

- **Preferred channel:** open a private security advisory on GitHub —
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

DevRites is pre-1.0. Only the latest minor release line receives security
updates.

| Version | Supported |
|---|---|
| `0.1.x` | yes |
| earlier | no |

Once a `1.0` release ships, the latest two minor lines will be supported.

## DevRites security model

### Scope

DevRites is a Claude Code **skills pack**: Markdown skill files, a few helper
shell scripts, a bash installer, and (in plugin form) a `.claude-plugin/`
manifest. It ships **no binary and no network service**. The only server is an
**optional, opt-in MCP stdio server** (`mcp/devrites-mcp.mjs`) that speaks
JSON-RPC over stdin/stdout (no port, no socket), is not installed or registered
by default, and only shells out to the local `devrites` CLI — read/gate ops over
`.devrites/` plus a single `ACTIVE`-pointer write. The attack surface is the
content of the skill files plus the installer.

### Supply-chain self-scan (shipped pack)

Because the skill files **are** the attack surface, CI scans the shipped pack
(`pack/.claude/`) on every PR and fails the build (blocking, not advisory) on:

- **Prompt-injection patterns** — the "ignore previous instructions" family,
  system-prompt overrides, permission-escalation, and data-exfiltration phrasing.
- **Hidden unicode** — bidi controls, zero-width characters, and homoglyph
  confusables (a word mixing ASCII with look-alike Cyrillic/Greek letters) that a
  human reviewer can't see in a diff.

Run it locally with `python3 scripts/scan-pack-security.py pack/.claude`. A
finding prints as `FINDING <file>:<line>: <class>: <excerpt>`. If the match is
DevRites' own *defensive* content (e.g. a rule that quotes an attack string, or a
QA checklist that demonstrates an adversarial character), add an auditable
suppression on the line — `<!-- pack-scan-ignore: <reason> -->` — or opt a whole
file out of one class with `<!-- pack-scan-ignore-file: injection -->`.
Suppressions live in the file, so every exception is visible in the diff and
reviewable; never suppress a hidden-unicode finding you can't explain.

### State loading (project-local script, no `!` injection)

`/rite`, `/rite-status`, and every workspace-operating skill load state by running
a **read-only shell script via the `Bash` tool** — *not* Claude Code's
preprocessing-only `` !`<command>` `` dynamic-context injection, which DevRites
**removed** for cross-harness portability:

```bash
P=.claude/skills/devrites-lib/scripts/preamble.sh
[ -f "$P" ] && bash "$P" || echo "(unavailable — read state.md directly)"
```

`preamble.sh` is a project-local read of DevRites' own `.devrites/` state: no user
input is concatenated into a command, no network access, no write side effects.
The gate scripts (`readiness.sh`, `evidence-fresh.sh`, `check-acceptance.sh`) are
likewise read-only; only the explicit mutators (`resolve.sh`, `tick-afk.sh`,
`close-out.sh`, and the CLI's `use`) write, and only under `.devrites/`.

If your environment disallows skill-initiated shell execution, the scripts simply
don't run — each skill degrades gracefully to reading `state.md` directly (the
`|| echo "(… unavailable …)"` fallback above), and the rest of the pack is
unaffected.

### Auto-trigger is deliberate

All `rite-*` skills are **auto-invocable** by Claude when their descriptions
match the user's intent. This is a conscious DevRites design choice (see
`DECISIONS.md` Q3). The safety nets are:

- **Body discipline**: every skill stops at its phase boundary. `rite-build`
  stops after one slice, `rite-prove` runs proofs only when all slices are
  built, etc.
- **Readiness gates**: each rite reads `.devrites/work/<slug>/state.md`
  before acting; phases out of order refuse to run.
- **Spec Drift Guard**: any deviation from the spec halts and routes to
  `rite-plan`.
- **Interactive type-GO confirmation** in `rite-ship` before irreversible
  git actions (commit, push, tag) — present even with auto-trigger; `rite-seal`
  only decides GO/NO-GO.

If you prefer explicit-only invocation in a given project, add a Claude Code
`permissions` rule disabling `Skill(rite-*)` auto-invocation in that
project's `.claude/settings.json`.

### Installer safety

The bash installer (`install.sh`) refuses any target under `~/.claude`
(`GUARD:no-global` block) and writes only to a manifest-tracked file list.
`./uninstall.sh` removes exactly the manifested files. The
`.devrites/` runtime state in the target project is preserved across
uninstall.

The installer touches no global state. It does not invoke `sudo`, modify
shell rc files, fetch remote code, or alter Claude Code settings.

### Plugin install path

When installed via `claude plugin install devrites@devrites-marketplace`,
the plugin runtime owns file placement. DevRites does not ship a post-install
hook that modifies user files. The plugin manifest ships only skills and agents —
the engineering rules are not delivered by the plugin and are never written into
the user's `~/.claude/CLAUDE.md` (add them with a `--rules-only` bash install).

### Recommended Claude Code permissions for managed deployments

For organizations evaluating DevRites under a managed Claude Code policy:

```jsonc
{
  "permissions": {
    "Skill(rite-seal)": "ask",
    "Bash(git push *)": "ask",
    "Bash(git tag *)": "ask"
  },
  "disableSkillShellExecution": true
}
```

The first three lines surface a confirmation prompt before any irreversible
git action; the last disables the `/rite` and `/rite-status` dynamic-state
read described above (DevRites still works).

### Hooks (auto-approve scope + orientation)

DevRites ships two `Bash`-matched hooks (`pack/.claude/hooks/`, wired via the
plugin's `hooks.json` or a seeded `.claude/settings.json`):

- **`devrites-allow.sh` (PreToolUse)** — auto-approves *only* the five **read-only**
  helper scripts (`preamble.sh`, `progress.sh`, `readiness.sh`, `evidence-fresh.sh`,
  `check-acceptance.sh`) so they stop prompting on every skill run. It is **fail-open**:
  it never denies, parses the command, and emits an `allow` *only* when the command runs
  one of those five scripts **and** contains no dangerous/exfiltration tokens
  (`rm`, `>`/redirects, `curl`/`wget`, `sudo`, `chmod`, `$(...)`/backticks, `eval`,
  package managers, `git push/commit/reset`, …). Mutating scripts (`resolve.sh`,
  `tick-afk.sh`, `close-out.sh`) are **deliberately excluded** and still prompt. On any
  parse failure or non-match it emits nothing — Claude Code's normal permission flow is
  untouched. It widens approval for read-only orientation; it never widens it for anything
  that writes, deletes, or reaches the network.
- **`devrites-orient.sh` (SessionStart)** — read-only; injects the active feature's
  `preamble.sh` digest as session context, and stays silent when no `.devrites/` workspace
  is active. No tool approval, no writes.

Delete `.claude/settings.json` (or disable the plugin) to remove both. The seeded
settings file is never overwritten on update, so your own permission rules are safe.

### Third-party trust

DevRites vendors no third-party code (see `NOTICE.md`). It depends on Claude
Code itself and, optionally and at the user's choice, on codegraph /
graphify / browser-harness — each of which is invoked through its own
documented interface, not bundled.

### Known non-issues

- **`!` injection in `/rite`** — local read of own state; safe. See above.
- **`Write` / `Edit` tool allowance in `rite-*` skills** — required to
  author `.devrites/` and project files. No skill grants `Bash(*)`.
- **Auto-trigger** — deliberate design choice; mitigated as above.

### CVE relevance

DevRites does not ship `permissions.defaultMode: bypassPermissions` in any
file (cf. **CVE-2026-33068**, workspace-trust bypass via committed bypass
mode). The installer refuses to write any `defaultMode: bypassPermissions`
into target projects.

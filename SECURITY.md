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
manifest. It does **not** ship a binary, a daemon, or a network service. The
attack surface is the content of the skill files plus the installer.

### Dynamic context (`!` shell injection in skill bodies)

One skill (`/rite` and `/rite-status`) uses Claude Code's
preprocessing-only `` !`<command>` `` injection to read its **own** state:

```markdown
!`cat .devrites/ACTIVE 2>/dev/null || echo "(none — run /rite-spec <feature>)"`
```

This is a project-local read of the active-feature pointer DevRites itself
manages. No user input is concatenated into the command, no network access,
no write side effects. The same pattern is documented in Anthropic's hooks
reference.

If your organization disallows dynamic-context shell execution, set
`disableSkillShellExecution: true` in your Claude Code settings. DevRites
continues to function — the dynamic injection in `/rite` and `/rite-status`
becomes a no-op and the rest of the pack is unaffected.

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
- **Interactive type-GO confirmation** in `rite-seal` before irreversible
  git actions (commit, push, tag) — present even with auto-trigger.

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
hook that modifies user files. Engineering rules (`pack/.claude/rules/*.md`)
are delivered via the `devrites-rules` loader skill rather than written into
the user's `~/.claude/CLAUDE.md`.

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

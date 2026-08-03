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
- The DevRites version (commit SHA or release tag), affected host, and host
  version (Claude Code or Codex).
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
and install paths, generated host artifacts, native permission profiles, legacy
hook cleanup, and the local engine binary.

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

### State loading

Workspace skills read `.devrites/ACTIVE`, `state.md`, and the phase artifacts
they need. No generated summary stands between the host and the authoritative
ledger.
Mutating engine commands are explicit and confined to DevRites state/artifacts;
the root owns workflow records, while source/tests route through the sole
wright. Claude enforces root non-writing with plan mode; Codex enforces it as
workflow policy on its workspace-capable root.

After Build, `touched-files.md` is the sole project-candidate authority. The
engine rejects malformed, escaped, symlinked, special-file, missing, ambiguous,
or oversized manifest inputs and computes one content-bound digest. Evidence,
optional browser evidence, review, and seal must contain the same exact binding.
The worktree digest is not an atomic snapshot against a malicious concurrent
same-size rewrite; Ship's exact Git-index checks own the final freeze.
See [`docs/candidate-integrity.md`](docs/candidate-integrity.md).

### User and model invocation are separate

Public skills are always user-invocable, but model invocation is set per skill.
Skills without `disable-model-invocation: true` may be selected when their
descriptions match the user's intent; explicit utilities carry that frontmatter
flag and load only when the user invokes them. The checked-in frontmatter and
[`docs/command-map.md`](docs/command-map.md) are the source of truth. The safety
nets for model-invocable rites are:

- **Body discipline**: every skill stops at its phase boundary. In HITL,
  `rite-build` stops after one slice; an explicit `.devrites/AFK` sentinel is
  the bounded low-risk chaining exception. `rite-prove` runs proofs only when
  all slices are built.
- **Readiness gates**: each rite reads `.devrites/work/<slug>/state.md`
  before acting; phases out of order refuse to run.
- **Spec Drift Guard**: deviations are classified before routing. Settled
  technical objective failures use bounded recovery in the active slice;
  `rite-plan repair` owns a wrong durable plan; product, policy, and
  irreversible-risk decisions pause for the human.
- **Interactive type-GO confirmation** in `rite-ship` before irreversible
  git actions such as commit or an optional push, tag, or PR. This prompt
  remains after model invocation; `rite-seal` only decides GO/NO-GO.

Claude documents the invocation controls in its official
[skills reference](https://code.claude.com/docs/en/slash-commands). DevRites uses
`disable-model-invocation: true` rather than an undocumented settings key.

### Installer safety

The Bash installer (`install.sh`) refuses any target under `~/.claude`
(`GUARD:no-global` block). Project host files are manifest-tracked;
`./uninstall.sh` removes exactly those managed files. The
`.devrites/` runtime state in the target project is preserved across
uninstall.

Skills, agents, standards, and host configuration stay in the target project.
The installer merges DevRites entries into project-local `.claude/settings.json`
and `.codex/config.toml` without replacing unrelated user settings. The shared
`devrites-engine` executable is the only allowed artifact outside the project.
It is installed to `DEVRITES_BIN_DIR`, a writable `~/.local/bin`, or a writable
`/usr/local/bin`; `--no-binary` / `DEVRITES_NO_BINARY=1` skips it. The bootstrap
and direct updater may fetch the release bundle and checksummed engine assets.
They never invoke `sudo` or edit shell startup files.

The documented Node-free boundary downloads the release-owned `install.sh` and
its exact-name sidecar before execution; mutable default-branch scripts are not
recommended. Legacy local and extracted shim invocations remain compatible, but
their exact-release guarantee begins at that verified asset. A piped shim never
treats current-directory siblings as its bundle.

Network acquisition accepts only an exact SemVer release asset and its mandatory
exact-filename SHA-256 sidecar. Every redirect hop must remain HTTPS, private
temporary-directory creation must succeed, and unchecked raw/source/default-
branch fallbacks are absent. Representative in-stream ceilings are 1 MiB for
release metadata, 4 KiB for a sidecar, and 64 MiB for an archive or binary, with
at most five Node or ten Go redirect hops. Archive handling rejects unsafe types
and paths, permits at most 10,000 members and 256 MiB regular files across all
routes, and applies stricter route-specific count, file, and expanded-byte caps
where configured. Failures retain only safe tag/asset and status, redirect,
size, checksum, or archive context, never response bodies. The adversarial
contract is exercised by `engine/internal/release/release_test.go`,
`tests/bootstrap-security-test.sh`, `tests/npx-pack-smoke.sh`, and
`tests/release-tarball-test.sh`; [ADR-0026](docs/adr/0026-content-bound-proof-and-bounded-inputs.md)
and [ADR-0028](docs/adr/0028-self-contained-engine-update.md) own the rationale.

### npx install path

With `npx devrites@latest`, the CLI (`bin/devrites.mjs`) calls
`devrites-engine` directly instead of running `install.sh`. The bundled host
payload is pinned to the requested npm package version. The shim first tries the
matching exact-release binary and its mandatory SHA-256 sidecar, then a
package-local Go build, and then an existing engine. Remote responses are
byte-bounded and every redirect is checked before following it. It has no
runtime npm dependencies. The same project and shared-binary boundaries apply
to the Bash installer.

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

### Native host policy and writer boundary

Reviewer profiles are natively read-only. Claude keeps the root in project plan
mode. Codex uses a workspace-capable root because a child cannot elevate above
its parent; root source/test non-writing is therefore a workflow rule, not a
Codex sandbox guarantee. On both hosts, `devrites-slice-wright` is the only
writable specialist. Its task states the exact project-relative paths. The root
waits, compares the returned file list and `git diff --name-only` with that
contract, and rejects out-of-scope work. Exact-path scope is instruction-backed,
not a per-task filesystem allowlist.
Agent lifecycle, session history, compaction, presentation, browsing, indexes,
and irreversible-action approval remain native host responsibilities.

Updates merge DevRites-owned permission/config blocks without replacing
unrelated user settings; uninstall removes only owned entries.

Production Git subprocesses apply one shared isolation policy before execution:
repository, worktree, index, object, ref, config, and pathspec-retargeting
`GIT_*` variables are removed, including dynamic config key/value pairs.
Unrelated Git variables are preserved. Go and shell parity is covered by
`engine/internal/gitenv` tests and `tests/git-env-sanitization-test.sh`.

### Third-party trust

DevRites vendors no third-party code (see `NOTICE.md`). It runs through the
external Claude Code or Codex host selected by the user and is independent of
Anthropic and OpenAI. Codegraph, graphify, and Playwright MCP are optional
user-selected tools that DevRites calls through their documented interfaces
rather than bundling them.

Repository npm-audit exceptions are temporary trust records, not claims that an
upstream issue is fixed. Each entry in `scripts/npm-audit-exceptions.json` is
restricted to an exact advisory, package range, installed node path, owner,
reason, source, and near-term expiry; validation re-audits the live dependency
graph and fails stale, broadened, unmatched, or expired entries. In particular,
the `brace-expansion` advisory remains present in npm's bundled dependency chain
until a patched ancestor is available.

### Agentic trust boundaries

Treat every instruction-bearing file as a supply-chain input:

1. **Shipped pack**: `pack/.claude/**`, generated host artifacts, native permission profiles, and the
   engine are release-managed and scanned before publish.
2. **Project-local state**: `.devrites/work/**`, unvalidated
   principles, and review artifacts are evidence, not authority. Live source and
   engine gates win over stale state.
3. **Native skills/plugins/MCP configs**: project-local integrations are
   optional instruction-bearing supply-chain inputs. Inspect them before trust;
   they may not weaken type-GO, seal/ship, permissions, security, or evidence
   gates.

<!-- authority:principles-trust:start -->
Project principles may become project policy only after explicit provenance and validation; arbitrary project-local Markdown is never inherently trusted executable instruction.
<!-- authority:principles-trust:end -->

Never copy untrusted issue text, web content, or model output into a skill,
agent, hook, MCP config, or generated artifact without reviewing it as executable
instructions. Hidden unicode, prompt-injection phrasing, personal absolute paths,
and secret-like strings are release-blocking findings unless explicitly and
visibly justified.

Keep untrusted data out of shell source. Commands must come from fixed,
reviewed structure; validate or allowlist any external value passed as a
separate argument. Quoting repository, retrieved, tool, or model text does not
upgrade its trust level or make unconstrained interpolation safe.

### Known non-issues

- **Historical `!` injection in `/rite`**: no longer present; current state
  orientation uses structurally bounded, read-only engine commands.
- **`Write` / `Edit` tool allowance in `rite-*` skills**: required to author
  `.devrites/` workflow records; source/tests still route through the sole
  wright. No skill grants `Bash(*)`.
- **Per-skill model invocation**: deliberate and frontmatter-controlled;
  model-invocable rites remain bounded by the gates above.

### CVE relevance

DevRites does not ship `permissions.defaultMode: bypassPermissions` in any
file (cf. **CVE-2026-33068**, workspace-trust bypass via committed bypass
mode). The installer refuses to write any `defaultMode: bypassPermissions`
into target projects.

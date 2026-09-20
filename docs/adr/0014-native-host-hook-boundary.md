# ADR-0014: Native host capability boundary

- **Status:** Superseded by 0015
- **Date:** 2026-07-29

## Context

DevRites accumulated Go hooks for agent permissions and lifecycle, session
orientation, status presentation, compaction handoffs, test-output sentinels,
web caching and content scanning, and background code-index refresh. Current
Claude Code and Codex releases already own those static or host-specific
surfaces. Keeping a second implementation increased latency, configuration,
tests, and drift without strengthening the workflow.

Some DevRites policies are different: they depend on live workspace state or
one exact operation and cannot be represented by a static host permission
profile.

## Decision

Use native host behavior first. The engine exposes only four hooks:

- `git-guard` for exact, one-shot destructive Git authority;
- `a1-guard` for dynamic root-versus-writer ownership;
- `stop-gate` for canonical rest-point invariants; and
- `wright-scope` for the slice-wright's per-run exact path allowlist, Forge
  binding, and reconcile boundary.

Claude read-only agents use native `permissionMode: plan`; Codex read-only
agents use native `sandbox_mode = "read-only"`. Both hosts own named-agent
lifecycle, transcript/session and compaction context, presentation, browsing
and cache behavior, and connected index lifecycle.

The slice-wright's first actionable `wright-scope` call captures its canonical
reconcile boundary. Capture is idempotent, so later tool calls cannot move that
boundary forward. No separate SubagentStart hook is required.

Proof remains explicit in workspace evidence and real command output. The
engine does not infer authority from a regex-based `.red` sentinel.

This record supersedes ADR-0003's `.red` StopGate clause, narrows ADR-0005 to
the four retained dynamic hooks, and replaces ADR-0013's wright-only start hook.
The rest of those decisions remains in force.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep every hook for identical Claude/Codex behavior | It duplicates host features and makes DevRites responsible for unstable presentation and lifecycle APIs. |
| Remove every engine hook | Static host permissions cannot express one-shot Git authority or a per-run `.wright-allowlist`. |
| Keep a separate wright start hook | The existing PreToolUse scope hook can capture the same boundary safely when capture is idempotent. |
| Keep `.red` as a test oracle | Output regexes create stale and false-positive state; explicit proof artifacts and actual command results are authoritative. |

## Consequences

- Project hook configuration shrinks to three hooks; only the writer profile
  adds `wright-scope`.
- Reviewer permissions have one owner per host.
- The engine no longer owns lifecycle telemetry, UI/status output, web caches,
  ingestion heuristics, or index workers.
- Connected index providers may be refreshed explicitly when stale; absence of
  one never blocks work.
- Dynamic security checks remain engine-tested and fail closed where static
  host policy is insufficient.

# DSH integration

The standard DevRites installer ships project-local DSH skills and named native
agent presets. To launch from the installed project's root:

```sh
npx @deepseek-ai/dsh web --patch .dsh/cordis.patch.yml
```

Choose **DevRites** (`devrites-orchestrator`) in the agent preset selector for a
new conversation, then use `/rite` or `/rite-spec`. The overlay loads alongside
your DSH profile; the installer does not edit your global DSH configuration.
Use `--no-dsh` with DevRites install/update to omit this host. Updates and
uninstall manage the same files through the existing hashed manifest.

Skills live under `.dsh/skills`, with native invocation flags preserved. The
`.dsh/agents` files retain the expanded canonical specialist instructions.
`.dsh/cordis.patch.yml` and `.dsh/specialist.mjs` register those exact identities
as native DSH presets and enforce their tool and permission boundaries.

The orchestrator calls `subagent` with an exact `agent_preset`, a self-contained
`prompt`, and a short `description`. Its configured backend is `spawn`, which creates a fresh
Session without the parent conversation. An omitted or unknown role is denied;
generic dispatch and fork are not substitutes. Specialists cannot delegate.
The root may use `list_agents`, `send_message`, and `interrupt_agent` to manage
its native children.

DSH delivers each child's final text and terminal status to the parent in a
settlement notice. A returned child id means dispatch was admitted; `inactive`
activity alone does not prove successful completion. The orchestrator collects
and reconciles every terminal result, including failures, before dependent work.

Read-only roles retain only their mapped tools and run shell commands under the
read-only sandbox. Shell-capable roles also receive native job tools to collect
background command output and stop their jobs. Writers have a workspace-write ceiling. The guard narrows
DSH's inherited parent override before each allowed call, preserves an existing
read-only parent boundary, and denies explicit sandbox escalation. Confining
shell and filesystem providers are required; an unconfined composition fails
closed. Tool access derives from canonical agent frontmatter; unsupported
Claude-specific tools are omitted. The writer's exact file list
remains a task contract checked by the orchestrator, as on the other hosts.

Codebase Memory MCP is optional. Install `codebase-memory-mcp` separately and
make the binary available on `PATH`, then load the supplied connection overlay:

```sh
dsh web --patch .dsh/codebase-memory.patch.yml --patch .dsh/cordis.patch.yml
```

The optional overlay registers one global `@deepseek-ai/dsh-mcp-client` stdio
connection named `codebase-memory-mcp`. If your DSH configuration already provides
that global connection, use only `.dsh/cordis.patch.yml`. Installation does not
install the MCP binary, start a server, or change global configuration. Without
the overlay or a configured connection, grep/glob/read remain available.

For a browser served on a local interface, keep launcher flags before app flags:

```sh
dsh web --patch .dsh/codebase-memory.patch.yml --patch .dsh/cordis.patch.yml --no-open --host <local-interface-ip> --port 3080
```

Canonical `mcp__codebase-memory-mcp__*` grants expand to a finite set of query
tools: index status, projects, schema, architecture, graph and code search, call
paths, snippets, file outlines, graph queries, change detection, coverage, and
graph comparisons. Individual supported query grants are also preserved. Roles
without a canonical grant receive none. Other MCP namespaces, unknown tools,
`delete_project`, `manage_adr`, and `ingest_traces` are denied. Only the root
orchestrator may use `index_repository`, with `persistence` omitted or `false`
for cache-only indexing; all specialists, including writers, cannot index.

Use graph queries first for structural questions, checking index status,
freshness, and coverage and verifying material findings in source. Use
grep/glob for literal searches. An unavailable server or missing index falls
back to grep/glob/read; a specialist may hand indexing to the parent. The tool
catalog refreshes when DSH registers or removes tools, including later successful
connections. DSH may retain registrations during a temporary connection loss.
MCP calls use the external server's permissions, outside DSH's shell/filesystem
sandbox. LSP integration is outside this adapter's scope.

The adapter targets the published DSH `0.2.1-rc.2` plugin interfaces, checked
against [DeepSeek Harness source](https://github.com/deepseek-ai/deepseek-harness)
at `f3803c96974ad4a94195fb97e5c637e9c2a3afe3`. DSH is a developer preview;
changes to its preset, tools, or sandbox interfaces require revalidation.

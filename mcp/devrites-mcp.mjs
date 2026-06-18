#!/usr/bin/env node
// devrites-mcp — a tiny, dependency-free MCP stdio server exposing the
// `.devrites/` workflow state as tools, so any MCP-capable agent (Claude,
// Cursor, Codex, Gemini CLI, …) can orient/gate/advance a DevRites workflow.
// It is a thin wrapper over the devrites CLI (devrites-lib/scripts/devrites.sh)
// — the discipline lives in the .devrites/ files + the shell scripts; this just
// surfaces them over MCP. Newline-delimited JSON-RPC over stdio, no SDK.
//
// Register (project .mcp.json), running from the project root:
//   { "mcpServers": { "devrites": { "command": "node",
//       "args": ["mcp/devrites-mcp.mjs"] } } }
// Override the CLI path with env DEVRITES_CLI if not installed at the defaults.

import { spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';

const CWD = process.cwd();
const CLI =
  process.env.DEVRITES_CLI ||
  ['.claude/skills/devrites-lib/scripts/devrites.sh',
   'pack/.claude/skills/devrites-lib/scripts/devrites.sh'].find(p => existsSync(p)) ||
  '.claude/skills/devrites-lib/scripts/devrites.sh';

// Tool name -> { sub: CLI subcommand, slug: takes optional slug, desc }
const TOOLS = {
  devrites_orient:         { sub: 'orient',         slug: true,  desc: 'Orientation digest for the active (or named) DevRites feature: state.md, artifacts, run mode, open-question tally. Read-only.' },
  devrites_status:         { sub: 'status',         slug: true,  desc: 'Alias for orient — where the active feature stands.' },
  devrites_ready:          { sub: 'ready',          slug: true,  desc: 'Build-readiness gate. Non-zero when the plan is not approved / awaiting human / blocked.' },
  devrites_evidence_fresh: { sub: 'evidence-fresh', slug: true,  desc: 'Evidence-freshness gate: fails when a touched file is newer than the recorded proof.' },
  devrites_acceptance:     { sub: 'acceptance',     slug: true,  desc: 'Acceptance-criteria gate: fails unless every spec [ACn] criterion is checked (proven) in seal.md.' },
  devrites_active:         { sub: 'active',          slug: false, desc: 'Print the active feature slug (.devrites/ACTIVE).' },
  devrites_list:           { sub: 'list',            slug: false, desc: 'List the DevRites workspace slugs under .devrites/work/.' },
  devrites_use:            { sub: 'use',             slug: 'req', desc: 'Re-point .devrites/ACTIVE to <slug> (slug required).' },
};

function toolList() {
  return Object.entries(TOOLS).map(([name, t]) => ({
    name,
    description: t.desc,
    inputSchema: t.slug
      ? { type: 'object',
          properties: { slug: { type: 'string', description: 'feature slug (defaults to .devrites/ACTIVE)' } },
          required: t.slug === 'req' ? ['slug'] : [] }
      : { type: 'object', properties: {} },
  }));
}

function runTool(name, args) {
  const t = TOOLS[name];
  if (!t) return { isError: true, text: `unknown tool: ${name}` };
  const parts = [t.sub];
  if (t.slug && args && typeof args.slug === 'string' && args.slug) parts.push(args.slug);
  const r = spawnSync('bash', [CLI, ...parts], { cwd: CWD, encoding: 'utf8' });
  const out = `${r.stdout || ''}${r.stderr || ''}`.trim() || `(exit ${r.status})`;
  return { isError: r.status !== 0, text: out };
}

function send(msg) { process.stdout.write(JSON.stringify(msg) + '\n'); }
function ok(id, result) { send({ jsonrpc: '2.0', id, result }); }
function err(id, code, message) { send({ jsonrpc: '2.0', id, error: { code, message } }); }

function handle(msg) {
  const { id, method, params } = msg;
  const isNotification = id === undefined || id === null;
  switch (method) {
    case 'initialize':
      return ok(id, {
        protocolVersion: (params && params.protocolVersion) || '2024-11-05',
        capabilities: { tools: {} },
        serverInfo: { name: 'devrites', version: '1.0.0' },
      });
    case 'notifications/initialized':
    case 'initialized':
      return; // notification, no reply
    case 'ping':
      return ok(id, {});
    case 'tools/list':
      return ok(id, { tools: toolList() });
    case 'tools/call': {
      const name = params && params.name;
      const res = runTool(name, (params && params.arguments) || {});
      return ok(id, { content: [{ type: 'text', text: res.text }], isError: res.isError });
    }
    default:
      if (!isNotification) err(id, -32601, `method not found: ${method}`);
  }
}

let buf = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => {
  buf += chunk;
  let nl;
  while ((nl = buf.indexOf('\n')) >= 0) {
    const line = buf.slice(0, nl).trim();
    buf = buf.slice(nl + 1);
    if (!line) continue;
    let msg;
    try { msg = JSON.parse(line); } catch { continue; }
    try { handle(msg); } catch (e) { if (msg && msg.id != null) err(msg.id, -32603, String(e)); }
  }
});
process.stdin.on('end', () => process.exit(0));

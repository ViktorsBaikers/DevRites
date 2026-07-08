#!/usr/bin/env node
// devrites-mcp — a tiny, dependency-free MCP stdio server exposing the
// `.devrites/` workflow state as tools, so any MCP-capable agent (Claude,
// Cursor, Codex, Gemini CLI, …) can orient/gate/advance a DevRites workflow.
// It is a thin wrapper over the installed `devrites-engine` binary
// — the discipline lives in the .devrites/ files + the engine; this just
// surfaces them over MCP. Newline-delimited JSON-RPC over stdio, no SDK.
//
// Register (project .mcp.json), running from the project root:
//   { "mcpServers": { "devrites": { "command": "node",
//       "args": ["mcp/devrites-mcp.mjs"] } } }
// Override the CLI path with env DEVRITES_CLI if not installed at the defaults.

import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { existsSync, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';

const CWD = process.cwd();
const HERE = dirname(fileURLToPath(import.meta.url));
const CLI =
  process.env.DEVRITES_CLI ||
  process.env.DEVRITES_ENGINE_CLI ||
  ['engine/devrites', join(HERE, '..', 'engine', 'devrites')].find(p => existsSync(p)) ||
  'devrites-engine';
const ROOT = process.env.DEVRITES_ROOT || '.devrites';

// Tool name -> { cmd: engine subcommand, slug/path/ledger: input shape, desc }
const TOOLS = {
  devrites_orient:            { cmd: 'preamble',          slug: true,  desc: 'Orientation digest for the active (or named) DevRites feature: state.md, artifacts, run mode, open-question tally. Read-only.' },
  devrites_status:            { cmd: 'preamble',          slug: true,  desc: 'Alias for orient — where the active feature stands.' },
  devrites_feature_status:    { cmd: 'status',            slug: 'req', desc: 'Indexed phase/completeness status for a named DevRites feature.' },
  devrites_ready:             { cmd: 'build-readiness',   slug: true,  desc: 'Build-readiness gate. Non-zero when the plan is not approved / awaiting human / blocked.' },
  devrites_phase_readiness:   { cmd: 'readiness',         slug: 'req', desc: 'Phase-completeness gate for a named DevRites feature.' },
  devrites_seal:              { cmd: 'seal',              slug: 'req', desc: 'Seal gate for a named DevRites feature.' },
  devrites_progress:          { cmd: 'progress',          slug: true,  desc: 'Compact active feature phase/slice/flow footer.' },
  devrites_evidence_fresh:    { cmd: 'evidence-fresh',    slug: true,  desc: 'Evidence-freshness gate: fails when a touched file is newer than the recorded proof.' },
  devrites_acceptance:        { cmd: 'check-acceptance',  slug: true, workspace: true, desc: 'Acceptance-criteria gate: fails unless every spec [ACn] criterion is checked (proven) in seal.md.' },
  devrites_coverage:          { cmd: 'coverage',          slug: true,  desc: 'Render the AC to slice to proof traceability matrix.' },
  devrites_analyze:           { cmd: 'analyze',           slug: true,  desc: 'Cross-artifact spec/tasks coverage and consistency gate.' },
  devrites_doubt_coverage:    { cmd: 'doubt-coverage',    slug: 'req', desc: 'Gate that checks whether the build doubted the decisions it stood.' },
  devrites_mutation_gate:     { cmd: 'mutation-gate',     slug: true,  desc: 'Advisory mutation-runner detector and scope gate.' },
  devrites_test_integrity:    { cmd: 'test-integrity',    slug: true,  desc: 'Gate: no test deleted, skipped, or de-asserted.' },
  devrites_review_integrity:  { cmd: 'review-integrity',  slug: true,  desc: 'Review-completeness gate: fails when an adversarial axis in review.md reported nothing and justified nothing (a suspected rubber-stamp). Read-only.' },
  devrites_package_existence: { cmd: 'package-existence', slug: true,  desc: 'Gate: every new import is declared in a manifest.' },
  devrites_budget:            { cmd: 'budget',            slug: true,  desc: 'Lint workspace files against context-size ceilings.' },
  devrites_spec_validate:     { cmd: 'spec-validate',     path: 'req', desc: 'Lint structured Requirement/Scenario grammar in a spec.md or workspace directory.' },
  devrites_spec_skeleton:     { cmd: 'spec-skeleton',     path: 'req', desc: 'Check that spec.md declares the six top-level spec sections.' },
  devrites_ledger:            { cmd: 'ledger',            ledger: true, desc: 'Read the capability ledger — the living record of what the system already does. No capability: list capabilities + requirement counts. With capability: show that capability\'s proven Requirement blocks (write new specs as deltas against these).' },
  devrites_active:            { sub: 'active',            slug: false, desc: 'Print the active feature slug (.devrites/ACTIVE).' },
  devrites_list:              { sub: 'list',              slug: false, desc: 'List the DevRites workspace slugs under .devrites/work/.' },
  devrites_use:               { sub: 'use',               slug: 'req', desc: 'Re-point .devrites/ACTIVE to <slug> (slug required).' },
};

function toolList() {
  return Object.entries(TOOLS).map(([name, t]) => {
    let inputSchema = { type: 'object', properties: {} };
    if (t.ledger) {
      inputSchema = { type: 'object',
        properties: { capability: { type: 'string', description: 'capability name; omit to list all capabilities' } },
        required: [] };
    } else if (t.path) {
      inputSchema = { type: 'object',
        properties: { path: { type: 'string', description: 'Workspace directory or spec file path' } },
        required: ['path'] };
    } else if (t.slug) {
      inputSchema = { type: 'object',
        properties: { slug: { type: 'string', description: 'feature slug (defaults to .devrites/ACTIVE)' } },
        required: t.slug === 'req' ? ['slug'] : [] };
    }
    return { name, description: t.desc, inputSchema };
  });
}

function activeSlug() {
  try {
    return readFileSync(join(ROOT, 'ACTIVE'), 'utf8').trim();
  } catch {
    return '';
  }
}

function listSlugs() {
  const slugs = new Set();
  for (const base of [join(ROOT, 'work'), join(ROOT, 'features')]) {
    try {
      for (const name of readdirSync(base)) {
        const full = join(base, name);
        if (statSync(full).isDirectory()) slugs.add(name);
      }
    } catch {
      // Missing work/features directories simply mean there are no slugs there.
    }
  }
  return [...slugs].sort();
}

function isPathLikeSlug(slug) {
  return slug === '.' || slug === '..' || slug.includes('/') || slug.includes('\\');
}

function safePathArg(path) {
  const p = typeof path === 'string' ? path.trim() : '';
  return p && !p.includes('\0') ? p : '';
}

function workspaceDir(slug) {
  const name = slug || activeSlug();
  if (!name) return '';
  if (isPathLikeSlug(name)) return '';
  const candidates = [join(ROOT, 'work', name), join(ROOT, 'features', name)];
  return candidates.find(p => existsSync(p)) || '';
}

function runCLI(parts) {
  const r = spawnSync(CLI, parts, { cwd: CWD, encoding: 'utf8' });
  if (r.error) {
    return {
      isError: true,
      text: r.error.code === 'ENOENT'
        ? `devrites-engine CLI not found: ${CLI}`
        : `devrites-engine CLI failed to launch: ${r.error.message}`,
    };
  }
  const out = `${r.stdout || ''}${r.stderr || ''}`.trim() || `(exit ${r.status})`;
  return { isError: r.status !== 0, text: out };
}

function runTool(name, args) {
  const t = TOOLS[name];
  if (!t) return { isError: true, text: `unknown tool: ${name}` };
  const slug = args && typeof args.slug === 'string' ? args.slug.trim() : '';
  if (t.sub === 'active') return { isError: false, text: activeSlug() || '(none)' };
  if (t.sub === 'list') return { isError: false, text: listSlugs().join('\n') || '(none)' };
  if (t.sub === 'use') {
    if (!slug) return { isError: true, text: 'usage: devrites_use requires slug' };
    if (isPathLikeSlug(slug)) return { isError: true, text: `devrites_use: invalid slug: ${slug}` };
    if (!listSlugs().includes(slug)) return { isError: true, text: `devrites_use: unknown workspace slug: ${slug}` };
    mkdirSync(ROOT, { recursive: true });
    writeFileSync(join(ROOT, 'ACTIVE'), `${slug}\n`);
    return { isError: false, text: slug };
  }

  if (t.path) {
    const path = safePathArg(args && args.path);
    if (!path) return { isError: true, text: `${t.cmd}: path is required` };
    return runCLI([t.cmd, path]);
  }

  if (t.workspace) {
    const ws = workspaceDir(slug);
    if (!ws) return { isError: true, text: `${t.cmd}: no active feature (pass slug or set .devrites/ACTIVE)` };
    return runCLI([t.cmd, ws]);
  }

  if (t.ledger) {
    const capability = args && typeof args.capability === 'string' ? args.capability.trim() : '';
    if (capability && isPathLikeSlug(capability)) return { isError: true, text: `devrites_ledger: invalid capability: ${capability}` };
    return runCLI(capability ? ['ledger', 'show', capability] : ['ledger', 'list']);
  }

  const parts = [t.cmd];
  if (t.slug === 'req' && !slug) return { isError: true, text: `${t.cmd}: slug is required` };
  if (t.slug && slug) parts.push(slug);
  return runCLI(parts);
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
        instructions:
          'DevRites exposes deterministic workflow state and gates for the current repo. Use devrites_status/orient to inspect the active feature, devrites_ready before build work, devrites_evidence_fresh / devrites_acceptance / devrites_review_integrity / devrites_test_integrity / devrites_package_existence before seal/ship, and devrites_use only when the user asks to switch the active feature.',
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

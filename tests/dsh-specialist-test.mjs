import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { apply } from '../scripts/dsh-specialist.mjs';

function runtime(mode = 'danger-full-access') {
  const events = [];
  const agent = { session: { append(type, value) { events.push({ type, value }); if (type === 'sandbox/mode') mode = value.mode; } } };
  let guard;
  let created;
  let restriction;
  let section;
  let restrictionChanges = 0;
  const catalog = new Set(['read', 'bash', 'write', 'subagent', 'subagent_fork', 'run_code']);
  const inheritedTools = new Set();
  const ownTools = new Set();
  const filters = new Set();
  const listeners = new Set();
  const disposers = [];
  const changed = () => { for (const listener of [...listeners]) listener(); };
  const visible = () => [...new Set([...catalog, ...inheritedTools])]
    .filter(name => [...filters].every(filter => !filter.deny.includes(name)));
  const ctx = {
    effect(fn) { fn(); },
    on(event, fn) { assert.equal(event, 'agent/created'); created = fn; },
    shell: { sandboxMode: 'workspace-write' }, fs: { sandboxMode: 'workspace-write' },
    sandboxPolicy: { resolve: () => ({ mode }) },
    tools: { guard(fn) { guard = fn; }, get: name => catalog.has(name) ? { name } : undefined,
      schemas: () => [...catalog].map(name => ({ name })) },
    systemPrompt: { section(value) { section = value; }, getSectionOrder: () => 10 },
  };
  agent.ctx = {
    ...ctx,
    effect(fn) { const dispose = fn(); if (dispose) disposers.push(dispose); },
    on(event, fn) {
      assert.equal(event, 'tools/change');
      listeners.add(fn);
      disposers.push(() => listeners.delete(fn));
    },
    tools: {
      get: name => ownTools.has(name) || visible().includes(name) ? { name } : undefined,
      restrict(value) {
        assert.ok(value.deny.every(name => catalog.has(name) || inheritedTools.has(name)), 'native restrictions require known inherited tools');
        assert.ok(!value.deny.includes('run_code'), 'native restrictions cannot name the presentation transport');
        restriction = value;
        filters.add(value);
        restrictionChanges++;
        assert.ok(restrictionChanges < 30, 'catalog refresh must not recurse');
        changed();
        return () => { filters.delete(value); changed(); };
      },
    },
  };
  return { ctx, agent, events, call: (name, args = {}) => guard({ name, arguments: args, agent }),
    mode: () => mode, create() { created({ agent }); }, restriction: () => restriction, section: () => section,
    register(name) { catalog.add(name); changed(); }, unregister(name) { catalog.delete(name); changed(); },
    addOwn(name) { ownTools.add(name); changed(); }, addInherited(name) { inheritedTools.add(name); changed(); }, visible, changed,
    listeners: () => listeners.size, restrictionChanges: () => restrictionChanges,
    dispose() { for (const dispose of disposers.reverse()) dispose(); },
  };
}

for (const [permission, expected] of [['read-only', 'read-only'], ['workspace-write', 'workspace-write']]) {
  const r = runtime();
  await apply(r.ctx, { role: 'test-role', permission, allowedTools: ['read', 'bash'], instructions: 'literal {{template}}' });
  r.create();
  assert.deepEqual(r.restriction(), { deny: ['write', 'subagent', 'subagent_fork'] });
  assert.ok(r.visible().includes('read'));
  assert.ok(r.visible().includes('bash'));
  assert.ok(!r.visible().includes('write'));
  assert.ok(r.visible().includes('run_code'));
  assert.equal(r.section().text, 'literal {{template}}');
  assert.equal(r.section().interpolate, false);
  assert.equal(r.call('bash'), undefined);
  assert.equal(r.mode(), expected);
  assert.match(r.call('write'), /not allowed/);
  assert.match(r.call('subagent'), /not allowed/);
  assert.match(r.call('bash', { sandbox_permissions: 'danger-full-access' }), /escalation/);
  assert.equal(r.events.length, 1);
  r.agent.session.append('sandbox/mode', { mode: 'danger-full-access' });
  assert.equal(r.call('bash'), undefined);
  assert.equal(r.mode(), expected);
}
const r = runtime('read-only');
await apply(r.ctx, { role: 'writer', permission: 'workspace-write', allowedTools: ['write'], instructions: 'writer' });
assert.equal(r.call('write'), undefined);
assert.equal(r.mode(), 'read-only');
const unsafe = runtime();
delete unsafe.ctx.shell.sandboxMode;
await assert.rejects(() => apply(unsafe.ctx, { role: 'reviewer', permission: 'read-only', allowedTools: ['bash'], instructions: 'review' }), /sandbox/);
await assert.rejects(() => apply(runtime().ctx, { role: 'reviewer', permission: 'danger-full-access', allowedTools: ['bash'], instructions: 'review' }), /permission/);
await assert.rejects(() => apply(runtime().ctx, { role: 'reviewer', permission: 'inherit', allowedTools: ['bash'], instructions: 'review' }), /permission/);
const noFs = runtime();
delete noFs.ctx.fs.sandboxMode;
await assert.rejects(() => apply(noFs.ctx, { role: 'reviewer', permission: 'read-only', allowedTools: ['read'], instructions: 'review' }), /sandbox/);
for (const placement of ['addOwn', 'addInherited']) {
  const orchestrator = runtime();
  orchestrator.unregister('subagent');
  orchestrator[placement]('subagent');
  await apply(orchestrator.ctx, { role: 'devrites-orchestrator', permission: 'inherit',
    allowedTools: ['read', 'subagent'], allowedAgentPresets: ['devrites-code-reviewer'], instructions: 'orchestrate' });
  assert.doesNotThrow(() => orchestrator.create());
  assert.deepEqual(orchestrator.restriction(), { deny: ['bash', 'write', 'subagent_fork'] });
  assert.deepEqual(orchestrator.agent.ctx.tools.get('subagent', orchestrator.agent), { name: 'subagent' });
  assert.match(orchestrator.call('subagent', { prompt: 'review' }), /exact named specialist/);
  assert.match(orchestrator.call('subagent', { agent_preset: 'generic' }), /exact named specialist/);
  assert.match(orchestrator.call('subagent_fork', { agent_preset: 'devrites-code-reviewer' }), /not allowed/);
  assert.equal(orchestrator.call('subagent', { agent_preset: 'devrites-code-reviewer', prompt: 'review', description: 'review feature diff' }), undefined);
}

const artifactRoot = process.env.DEVRITES_HOST_ARTIFACT_DIR || fileURLToPath(new URL('../pack/generated', import.meta.url));
const presets = JSON.parse(readFileSync(resolve(artifactRoot, 'dsh/runtime/cordis.patch.yml'), 'utf8'))[0].insert[0].config.presets;
const configs = presets.map(preset => preset.plugins.find(plugin => plugin.name === 'devrites:role').config);
const overhaulConfigs = configs.filter(config => config.role.startsWith('overhaul-'));
assert.ok(overhaulConfigs.length > 0, 'shipped overhaul presets must exist');
for (const config of [
  ...overhaulConfigs,
  ...configs.filter(config => config.role.startsWith('devrites-')
    && !['devrites-orchestrator', 'devrites-slice-wright'].includes(config.role)),
]) {
  const overhaul = config.role.startsWith('overhaul-');
  for (const parentMode of ['workspace-write', 'read-only']) {
    const host = runtime(parentMode);
    host.register('edit');
    await apply(host.ctx, config);
    host.create();
    assert.equal(host.mode(), overhaul ? parentMode : 'read-only', config.role);
    for (const tool of ['write', 'edit']) {
      if (overhaul) {
        assert.ok(host.visible().includes(tool), `${config.role}: ${tool} must remain available`);
        assert.equal(host.call(tool), undefined, `${config.role}: ${tool} must reach sandbox`);
      } else {
        assert.ok(!host.visible().includes(tool), `${config.role}: ${tool} must stay hidden`);
        assert.match(host.call(tool), /not allowed/, config.role);
      }
    }
    assert.equal(host.mode(), overhaul ? parentMode : 'read-only', config.role);
    assert.match(host.call('bash', { sandbox_permissions: 'danger-full-access' }),
      config.allowedTools.includes('bash') ? /escalation/ : /not allowed/, config.role);
    for (const tool of ['subagent', 'subagent_fork']) {
      assert.match(host.call(tool), /not allowed/, config.role);
    }
    host.dispose();
  }
}
const orchestratorConfig = configs.find(config => config.role === 'devrites-orchestrator');
for (const mode of ['read-only', 'workspace-write', 'danger-full-access']) {
  const host = runtime(mode);
  await apply(host.ctx, orchestratorConfig);
  host.create();
  assert.equal(host.mode(), mode);
  assert.deepEqual(host.events, []);
  assert.match(host.call('bash', { sandbox_permissions: 'danger-full-access' }), /escalation/);
  for (const selected of ['danger-full-access', 'workspace-write', 'read-only']) {
    host.agent.session.append('sandbox/mode', { mode: selected });
    const selectedEvents = host.events.length;
    assert.equal(host.call('bash'), undefined);
    assert.equal(host.call('read'), undefined);
    assert.equal(host.mode(), selected);
    assert.equal(host.events.length, selectedEvents, 'root must not overwrite the selected host permission');
  }
  host.dispose();
}
const unknownMode = runtime('unknown');
await apply(unknownMode.ctx, orchestratorConfig);
assert.throws(() => unknownMode.create(), /unknown sandbox permission/);
const prefix = 'mcp__codebase-memory-mcp__';
const query = prefix + 'search_graph';
const index = prefix + 'index_repository';
const denied = ['delete_project', 'manage_adr', 'ingest_traces', 'future_tool'].map(name => prefix + name);
denied.push('mcp__other__search_graph', 'mcp__codebase-memory-mcp-evil__search_graph', prefix + 'spoof__search_graph');
for (const config of configs) {
  const role = runtime();
  await apply(role.ctx, config);
  for (const tool of denied) assert.match(role.call(tool), /not allowed/, `${config.role}: ${tool}`);
  if (config.role !== 'devrites-orchestrator') assert.match(role.call(index), /not allowed/, config.role);
  if (config.allowedTools.includes(query)) assert.equal(role.call(query, { name_pattern: 'apply' }), undefined);
  else assert.match(role.call(query), /not allowed/, config.role);
}
const root = runtime();
await apply(root.ctx, orchestratorConfig);
assert.equal(root.call(index), undefined);
assert.equal(root.call(index, { persistence: false }), undefined);
for (const persistence of [true, 'false', null, 0]) {
  assert.match(root.call(index, { persistence }), /cache-only/);
}
root.register(query);
root.register(index);
root.create();
assert.equal(root.mode(), 'danger-full-access');
for (const permission of ['read-only', 'workspace-write']) {
  const child = runtime(root.mode());
  await apply(child.ctx, configs.find(config => config.permission === permission));
  child.create();
  assert.equal(child.mode(), permission, 'specialists retain their ceiling under a Full access parent');
  assert.equal(root.mode(), 'danger-full-access');
  child.dispose();
}
assert.ok(root.visible().includes(query));
assert.ok(root.visible().includes(index));
const ungranted = runtime();
ungranted.register(query);
await apply(ungranted.ctx, { role: 'ungranted', permission: 'read-only', allowedTools: ['read'], instructions: 'read' });
ungranted.create();
assert.ok(!ungranted.visible().includes(query));
assert.match(ungranted.call(query), /not allowed/);

const reviewer = runtime();
await apply(reviewer.ctx, configs.find(config => config.role === 'devrites-code-reviewer'));
reviewer.create();
assert.ok(!reviewer.visible().includes(query));
assert.equal(reviewer.listeners(), 1);
const initialChanges = reviewer.restrictionChanges();
reviewer.changed();
assert.equal(reviewer.restrictionChanges(), initialChanges);
reviewer.register(query);
assert.ok(reviewer.visible().includes(query), 'late MCP registration becomes visible');
assert.equal(reviewer.call(query), undefined);
assert.equal(reviewer.mode(), 'read-only');
assert.equal(reviewer.restrictionChanges(), initialChanges, 'allowed global tools need no deny-mask refresh');
for (const tool of [...denied, index]) {
  reviewer.register(tool);
  assert.ok(!reviewer.visible().includes(tool));
  assert.match(reviewer.call(tool), /not allowed/);
}
assert.equal(reviewer.restrictionChanges(), initialChanges + denied.length + 1);
reviewer.addOwn(denied[0]);
assert.ok(reviewer.agent.ctx.tools.get(denied[0], reviewer.agent));
assert.match(reviewer.call(denied[0]), /not allowed/, 'guard also denies scope-owned tools');
reviewer.unregister(query);
assert.ok(!reviewer.visible().includes(query), 'MCP removal updates the catalog');
reviewer.register(query);
assert.ok(reviewer.visible().includes(query), 'MCP reconnect restores the catalog');
reviewer.dispose();
assert.equal(reviewer.listeners(), 0);
const disposedChanges = reviewer.restrictionChanges();
reviewer.unregister(query);
reviewer.register(query);
assert.equal(reviewer.restrictionChanges(), disposedChanges);
console.log('DSH specialist permission and tool boundaries passed');

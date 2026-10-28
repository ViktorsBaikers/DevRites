import assert from 'node:assert/strict';
import { apply } from '../scripts/dsh-specialist.mjs';

function runtime(mode = 'danger-full-access') {
  const events = [];
  const agent = { session: { append(type, value) { events.push({ type, value }); if (type === 'sandbox/mode') mode = value.mode; } } };
  let guard;
  let created;
  let restriction;
  let section;
  const ctx = {
    effect(fn) { fn(); },
    on(event, fn) { assert.equal(event, 'agent/created'); created = fn; },
    shell: { sandboxMode: 'workspace-write' }, fs: { sandboxMode: 'workspace-write' },
    sandboxPolicy: { resolve: () => ({ mode }) },
    tools: { guard(fn) { guard = fn; }, restrict(value) { restriction = value; }, get: () => ({}) },
    systemPrompt: { section(value) { section = value; }, getSectionOrder: () => 10 },
  };
  agent.ctx = ctx;
  return { ctx, agent, events, call: (name, args = {}) => guard({ name, arguments: args, agent }),
    mode: () => mode, create() { created({ agent }); }, restriction: () => restriction, section: () => section };
}

for (const [permission, expected] of [['read-only', 'read-only'], ['workspace-write', 'workspace-write']]) {
  const r = runtime();
  await apply(r.ctx, { role: 'test-role', permission, allowedTools: ['read', 'bash'], instructions: 'literal {{template}}' });
  r.create();
  assert.deepEqual(r.restriction(), { allow: ['read', 'bash'] });
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
const noFs = runtime();
delete noFs.ctx.fs.sandboxMode;
await assert.rejects(() => apply(noFs.ctx, { role: 'reviewer', permission: 'read-only', allowedTools: ['read'], instructions: 'review' }), /sandbox/);
const orchestrator = runtime();
await apply(orchestrator.ctx, { role: 'devrites-orchestrator', permission: 'workspace-write',
  allowedTools: ['subagent'], allowedAgentPresets: ['devrites-code-reviewer'], instructions: 'orchestrate' });
assert.match(orchestrator.call('subagent', { prompt: 'review' }), /exact named specialist/);
assert.match(orchestrator.call('subagent', { agent_preset: 'generic' }), /exact named specialist/);
assert.match(orchestrator.call('subagent_fork', { agent_preset: 'devrites-code-reviewer' }), /not allowed/);
assert.equal(orchestrator.call('subagent', { agent_preset: 'devrites-code-reviewer', prompt: 'review', description: 'review feature diff' }), undefined);
console.log('DSH specialist permission and tool boundaries passed');

// One declaring plugin and its scoped role guard share a stable file URL.
export const name = 'devrites-specialist';
export const inject = ['agentPresets', 'tools', 'systemPrompt', 'sandboxPolicy', 'shell', 'fs'];

function requireSandbox(ctx) {
  if (!['read-only', 'workspace-write', 'danger-full-access'].includes(ctx.shell?.sandboxMode)
    || !['read-only', 'workspace-write', 'danger-full-access'].includes(ctx.fs?.sandboxMode)) {
    throw new Error('DSH DevRites requires sandbox-enforcing shell and filesystem providers');
  }
}

export async function apply(ctx, config) {
  requireSandbox(ctx);
  if (Array.isArray(config.presets)) {
    for (const preset of config.presets) {
      const plugins = preset.plugins.map(row => row.name === 'devrites:role'
        ? { ...row, name: import.meta.url } : row);
      const dispose = await ctx.agentPresets.register({ ...preset, plugins });
      ctx.effect(() => dispose);
    }
    return;
  }
  const { role, permission, allowedTools, instructions } = config;
  if (typeof role !== 'string' || !role || typeof instructions !== 'string'
    || !['read-only', 'workspace-write'].includes(permission)
    || !Array.isArray(allowedTools) || allowedTools.some(tool => typeof tool !== 'string')) {
    throw new Error('DSH DevRites role requires identity, instructions, tools, and a supported permission ceiling');
  }
  const allowed = new Set(allowedTools);
  const pinPermission = session => {
    requireSandbox(ctx);
    const mode = ctx.sandboxPolicy.resolve({ session }).mode;
    if (!['read-only', 'workspace-write', 'danger-full-access'].includes(mode)) {
      throw new Error(`${role}: unknown sandbox permission`);
    }
    // Native spawn inherits the parent's override; preserve a narrower parent.
    if (mode === 'danger-full-access' || (mode === 'workspace-write' && permission === 'read-only')) {
      session.append('sandbox/mode', { mode: permission });
    }
  };
  ctx.effect(() => ctx.systemPrompt.section({
    name: 'deployment:persona-prefix',
    order: ctx.systemPrompt.getSectionOrder('DEPLOYMENT_PERSONA_PREFIX'),
    text: instructions,
    interpolate: false,
  }));
  // Preset-local tools are inherited by the Agent; restrict at that child scope.
  ctx.on('agent/created', ({ agent }) => {
    pinPermission(agent.session);
    agent.ctx.tools.restrict({ allow: allowedTools.filter(tool => agent.ctx.tools.get(tool, agent) !== undefined) });
  });
  ctx.tools.guard(exec => {
    if (!exec.agent) return `${role}: agent identity required`;
    if (!allowed.has(exec.name)) return `${role}: tool ${exec.name} is not allowed`;
    if (exec.name === 'subagent' && !config.allowedAgentPresets?.includes(exec.arguments?.agent_preset)) {
      return `${role}: an exact named specialist agent_preset is required`;
    }
    if (exec.arguments?.sandbox_permissions !== undefined) return `${role}: permission escalation is forbidden`;
    pinPermission(exec.agent.session);
    return undefined;
  });
}

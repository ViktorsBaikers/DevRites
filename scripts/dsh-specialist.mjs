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
  // Deny only globals so preset-local tools never enter the inherited-tool mask.
  ctx.on('agent/created', ({ agent }) => {
    pinPermission(agent.session);
    let current;
    let release;
    let refreshing = false;
    let disposed = false;
    const refresh = () => {
      if (refreshing || disposed) return;
      const names = ctx.tools.schemas().map(tool => tool.name)
        .filter(tool => tool !== 'run_code' && !allowed.has(tool));
      const key = JSON.stringify(names);
      if (key === current) return;
      refreshing = true;
      try {
        current = key;
        release?.();
        release = agent.ctx.tools.restrict({ deny: names });
      } finally {
        refreshing = false;
      }
    };
    agent.ctx.on('tools/change', refresh);
    agent.ctx.effect(() => () => { disposed = true; release?.(); });
    refresh();
  });
  ctx.tools.guard(exec => {
    if (!exec.agent) return `${role}: agent identity required`;
    if (!allowed.has(exec.name)) return `${role}: tool ${exec.name} is not allowed`;
    if (exec.name === config.indexTool && exec.arguments?.persistence !== undefined
      && exec.arguments.persistence !== false) return `${role}: graph indexing must be cache-only (persistence: false)`;
    if (exec.name === 'subagent' && !config.allowedAgentPresets?.includes(exec.arguments?.agent_preset)) {
      return `${role}: an exact named specialist agent_preset is required`;
    }
    if (exec.arguments?.sandbox_permissions !== undefined) return `${role}: permission escalation is forbidden`;
    pinPermission(exec.agent.session);
    return undefined;
  });
}

// Docs content — structured so pages stay accurate and in one place.
// Mirrors the shipped pack (v2.3): 10 phases, internal skills, review agents, core.

export const DOCS_NAV = [
  { href: "/docs/", label: "Overview" },
  { href: "/docs/command-map/", label: "Command map" },
  { href: "/docs/flow/", label: "Flow" },
  { href: "/docs/architecture/", label: "Architecture" },
];

export type Cmd = { cmd: string; phase: string; desc: string };

export const PUBLIC_COMMANDS: Cmd[] = [
  { cmd: "/rite", phase: "menu", desc: "Compact menu and the suggested next command. A pure router that runs nothing itself." },
  { cmd: "/rite-adopt", phase: "adopt", desc: "Brownfield on-ramp. Reads an existing codebase and seeds DevRites before your first feature." },
  { cmd: "/rite-spec", phase: "spec", desc: "Start here. Deep investigation, then writes spec.md and creates the workspace." },
  { cmd: "/rite-temper", phase: "temper", desc: "Strategic review of the spec before planning: scope mode, pre-mortem, harden the contract." },
  { cmd: "/rite-define", phase: "define", desc: "Turns the approved spec into a plan and vertical task slices." },
  { cmd: "/rite-vet", phase: "vet", desc: "Engineering review of the plan before code. Writes the build-readable test-plan.md." },
  { cmd: "/rite-plan", phase: "plan", desc: "Decompose, reslice, repair, re-order, split, or unblock an active plan." },
  { cmd: "/rite-build", phase: "build", desc: "Implement exactly one vertical slice, test-first, then stop with evidence. Optional forge mode builds rival versions and a judge picks the winner." },
  { cmd: "/rite-prove", phase: "prove", desc: "Tests, build, runtime, and browser proof of the finished feature." },
  { cmd: "/rite-polish", phase: "polish", desc: "Behavior-preserving cleanup, then UI polish against the design system." },
  { cmd: "/rite-review", phase: "review", desc: "Feature-scoped multi-axis review with parallel fresh-context reviewers." },
  { cmd: "/rite-seal", phase: "seal", desc: "Final GO / NO-GO. Walks acceptance, fans out reviewers, writes the verdict." },
  { cmd: "/rite-ship", phase: "ship", desc: "The gated, irreversible commit / push / tag, then archives the workspace. Demands type-GO." },
  { cmd: "/rite-autocomplete", phase: "auto", desc: "Run the whole lifecycle unattended, pausing only at irreversible gates." },
  { cmd: "/rite-status", phase: "status", desc: "Active feature: phase, run mode, next action, evidence, open gates, risks." },
  { cmd: "/rite-resolve", phase: "resume", desc: "Answer, drop, or batch-resolve open gates, then resume the build." },
  { cmd: "/rite-zoom-out", phase: "utility", desc: "One-pass structural map of an unfamiliar area, in your project's vocabulary." },
  { cmd: "/rite-prototype", phase: "utility", desc: "Throwaway code answering one design question. Logic harness or UI variations." },
  { cmd: "/rite-handoff", phase: "utility", desc: "Compacts the chat into a handoff doc and syncs context into workspace files." },
  { cmd: "/rite-pressure-test", phase: "utility", desc: "Pressure-test a rough idea: diverge into options, converge on one." },
  { cmd: "/rite-frame", phase: "utility", desc: "Goal reframe plus a failure-mode self-audit before you commit to an approach." },
  { cmd: "/rite-learn", phase: "utility", desc: "Promote a learned convention from the ledger into a project rule the lifecycle enforces." },
  { cmd: "/rite-doctor", phase: "utility", desc: "Two-tier health check of the install and the active workspace." },
];

export const INTERNAL_SKILLS: { name: string; trigger: string; role: string }[] = [
  { name: "devrites-interview", trigger: "underspecified ask", role: "One question at a time, with a best-guess and a confidence stop." },
  { name: "devrites-source-driven", trigger: "uncertain framework fact", role: "Consults docs or source and records the citation in evidence." },
  { name: "devrites-doubt", trigger: "non-trivial decision", role: "Adversarial: claim, extract, doubt, reconcile, stop." },
  { name: "devrites-frontend-craft", trigger: "UI in build or polish", role: "Register detection, shape before code, states, anti-slop guardrails." },
  { name: "devrites-browser-proof", trigger: "UI in prove or polish", role: "The browser proof ladder and the evidence schema." },
  { name: "devrites-debug-recovery", trigger: "failing tests or build", role: "Six-phase loop: reproduce, hypothesize, instrument, fix, clean up." },
  { name: "devrites-api-interface", trigger: "cross-boundary slice", role: "Stable API and contract design across a front-end / back-end split." },
  { name: "devrites-ux-shape", trigger: "UI detected in spec", role: "Plans the UX into design-brief.md before any code is written." },
  { name: "devrites-audit", trigger: "polish, or risk in scope", role: "Simplify, security, and perf audits, each dispatching a reviewer." },
];

export const REVIEW_AGENTS: { name: string; checks: string }[] = [
  { name: "devrites-spec-reviewer", checks: "Does the diff implement the spec? Missing, partial, or wrong criteria; scope creep." },
  { name: "devrites-code-reviewer", checks: "Correctness, readability, architecture, maintainability." },
  { name: "devrites-test-analyst", checks: "Do the tests actually prove the acceptance criteria?" },
  { name: "devrites-frontend-reviewer", checks: "UX, accessibility, responsive behavior, design system, anti-AI-slop." },
  { name: "devrites-security-auditor", checks: "OWASP Top 10 (plus the LLM Top 10 on AI surfaces), trust boundaries, secrets, dependencies." },
  { name: "devrites-performance-reviewer", checks: "N+1 queries, hot paths, payload size." },
  { name: "devrites-doubt-reviewer", checks: "Adversarial check of a single claim or decision." },
  { name: "devrites-simplifier-reviewer", checks: "Independent simplification judgment under Chesterton's Fence." },
  { name: "devrites-strategy-reviewer", checks: "Spec-vs-rubric review at temper: ambition, scope, premise, pre-mortem, YAGNI." },
  { name: "devrites-plan-reviewer", checks: "Plan-vs-rubric review at vet: architecture, reuse, test design, reversibility." },
  { name: "devrites-devex-reviewer", checks: "Developer experience at vet and seal: public APIs, CLIs, SDKs, the getting-started path." },
  { name: "devrites-forge-judge", checks: "Scores the rival candidates from a forge build and names the winner." },
  { name: "devrites-retrospector", checks: "Reads across archived features at ship for recurring drift, then drafts rule candidates." },
];

export const LAYERS: { name: string; tag: string; body: string }[] = [
  { name: "Phase commands", tag: "rite-*", body: "The ten workflow phases plus utilities, each a public skill you can invoke on its own." },
  { name: "Internal specialists", tag: "devrites-*", body: "Model-invoked skills (interview, doubt, frontend-craft, browser-proof, and more) that fire when a phase needs them." },
  { name: "Review agents", tag: "13 fresh-context", body: "Reviewers and judges spawned across temper, vet, build, review, seal, and ship, plus the one write-capable executor, devrites-slice-wright. Each reads its input cold." },
  { name: "Persistent state", tag: ".devrites/", body: "The per-feature Markdown workspace. The single source of truth between phases and across cleared contexts." },
  { name: "Tool-agnostic core", tag: "CLI · MCP", body: "A portable CLI and a dependency-free MCP server expose the deterministic gates to Cursor, Codex, Gemini CLI, CI, or a human. The exit code is the verdict." },
  { name: "Engineering rules", tag: ".claude/rules/", body: "Stack-agnostic rules: one always-on core.md plus fifteen on-demand files loaded only by the phase that needs them." },
  { name: "Install & manifest", tag: "manifest", body: "install.sh records every written file in .claude/devrites.manifest; uninstall removes exactly those and leaves your work untouched." },
];

export const RATIONALE: { q: string; a: string }[] = [
  { q: "Why distributed skills, not one engine?", a: "A single mega-command would load everything on every call and hide the workflow behind one opaque entry point. Separate phase skills stay small, stay discoverable from /rite, and each loads only the rules it needs. The commands map to the workflow one to one." },
  { q: "Why the rite-* names?", a: "Short, memorable, and collision-free. Built-in Claude Code commands already include /plan, /review, /run, /verify, /init, /compact, and /debug. The rite- prefix avoids all of them." },
  { q: "Why a thin menu skill, not a mega-router?", a: "/rite shows the menu and suggests the next command, but it never executes a phase or reads state. Every phase skill stays invokable by muscle memory without forcing everything through one dispatcher." },
  { q: "Why internal skills exist", a: "Specialists like devrites-doubt and devrites-frontend-craft are model-invoked, so they fire at the right moment without cluttering the menu. The phase you run stays simple; the depth shows up when the work calls for it." },
  { q: "Why a tool-agnostic core", a: "The workspace and rules were already tool-agnostic data. The CLI and MCP server are thin shims over the same state scripts, so a verdict from the CLI, the MCP server, or /rite-seal is the same verdict. It runs the same script." },
];

export const CLI_GATES = ["orient", "ready", "evidence-fresh", "acceptance", "active", "list", "use", "resolve"];

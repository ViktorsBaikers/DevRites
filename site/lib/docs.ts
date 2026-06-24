// Docs content — structured so pages stay accurate and in one place.
// Mirrors the shipped pack (v2.3): 10 phases, internal skills, review agents, core.

export const DOCS_NAV = [
  {
    group: "Start",
    items: [
      { href: "/docs/", label: "Overview" },
      { href: "/docs/getting-started/", label: "Getting started" },
      { href: "/docs/concepts/", label: "Concepts" },
      { href: "/docs/usage/", label: "Usage & examples" },
    ],
  },
  {
    group: "Reference",
    items: [
      { href: "/docs/command-map/", label: "Command map" },
      { href: "/docs/flow/", label: "Flow" },
      { href: "/docs/cli-mcp/", label: "CLI & MCP" },
      { href: "/docs/architecture/", label: "Architecture" },
    ],
  },
];

export const DOCS_LINKS = DOCS_NAV.flatMap((g) => g.items);

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

export const WORKSPACE_FILES: { file: string; by: string; holds: string }[] = [
  { file: "brief.md", by: "/rite-spec", holds: "One-line objective and the definition of done." },
  { file: "spec.md", by: "/rite-spec", holds: "What to build and why: placement, acceptance criteria, gaps and decisions." },
  { file: "references/", by: "/rite-spec", holds: "Saved design references: screenshots, Figma, video, links." },
  { file: "strategy.md", by: "/rite-temper", holds: "Strategic spec review (optional): scope mode, pre-mortem, dimension scores." },
  { file: "plan.md", by: "/rite-define", holds: "Approach, dependency graph, checkpoints, rollback." },
  { file: "tasks.md", by: "/rite-define", holds: "Ordered vertical slices, each tagged Mode (AFK / HITL) with a gate." },
  { file: "eng-review.md", by: "/rite-vet", holds: "Engineering plan review (optional): scope challenge, findings, failure modes." },
  { file: "test-plan.md", by: "/rite-vet", holds: "Build-readable coverage target: per-gap test requirements, acceptance-to-test map." },
  { file: "state.md", by: "every phase", holds: "The cursor: phase, status, active slice, risk, next step, plus any 'awaiting human' block." },
  { file: "questions.md", by: "every phase", holds: "Append-only Q&A: qid, slice, gate, status (open / answered / dropped)." },
  { file: "decisions.md · assumptions.md", by: "every phase", holds: "Running logs of the reasoning." },
  { file: "drift.md", by: "Spec Drift Guard", holds: "Drift events and how each was resolved." },
  { file: "touched-files.md", by: "/rite-build", holds: "Which files this feature touched." },
  { file: "evidence.md", by: "/rite-build · /rite-prove", holds: "Recorded commands and their output." },
  { file: "browser-evidence.md", by: "/rite-prove · /rite-polish", holds: "Screenshots, console, network, viewport runs." },
  { file: "design-brief.md", by: "devrites-frontend-craft", holds: "Shape, states, and the design-references match for UI." },
  { file: "polish-report.md", by: "/rite-polish", holds: "Cleanup and UI-polish findings and fixes." },
  { file: "review.md", by: "/rite-review", holds: "Spec and Standards axes, severity-labelled findings." },
  { file: "seal.md", by: "/rite-seal", holds: "GO / NO-GO verdict, acceptance walk, blockers." },
  { file: "ship.md", by: "/rite-ship", holds: "What shipped: commit SHAs, branch, tag/PR, acceptance summary, follow-ups." },
];

export const INSTALL_FLAGS: { flag: string; effect: string }[] = [
  { flag: "--target DIR", effect: "Install into DIR (default: the current directory)." },
  { flag: "--dry-run", effect: "Show the planned file operations and exit, changing nothing." },
  { flag: "--force", effect: "Overwrite existing non-DevRites files." },
  { flag: "--no-rules", effect: "Skip the engineering rules." },
  { flag: "--no-agents", effect: "Skip the review subagents." },
  { flag: "--rules-only", effect: "Install only the engineering rules." },
  { flag: "--short-aliases=all", effect: "Add /define, /build, /prove, /seal short aliases (off by default)." },
];

export const SETUP_TOOLS: { tool: string; gives: string }[] = [
  { tool: "codegraph", gives: "A code-intelligence index. spec, define, and plan use it to understand structure, placement, callers, and impact cheaply, for deeper specs at a fraction of the tokens." },
  { tool: "graphify", gives: "A codebase-to-knowledge-graph for 'where is X, what calls Y, what would Z break'." },
  { tool: "browser-harness", gives: "Drives your real browser so prove and polish capture real UI evidence: screenshots, console, network, responsive. It is the top rung of the proof ladder." },
];

export const CONCEPTS: { term: string; body: string }[] = [
  { term: "Vertical slice", body: "The unit of work. One slice cuts through every layer it needs (DB to service to API to UI) and ends in something verifiable. /rite-build implements exactly one, then stops with evidence. You decide when the next runs." },
  { term: "Gate / checkpoint", body: "A point where the build pauses for a decision. HITL gates escalate through advisory, validating, blocking, and escalating. The exit code of the CLI gates (ready, evidence-fresh, acceptance) is the machine-readable version of the same idea." },
  { term: "Evidence ladder", body: "Claims need proof, ranked top-down: real browser runs, then Chrome DevTools, then /run + /verify, then project E2E, then a manual fallback. A screenshot path counts as unproven until it is opened and described." },
  { term: "Spec Drift Guard", body: "When the build reveals the plan is wrong, it stops, records the drift in drift.md, asks you when product behavior changes, and routes through plan repair before resuming. A wrong turn never compounds." },
  { term: "Forge", body: "An optional build mode. Two or three rival versions of a slice are built in isolation, then devrites-forge-judge scores them against the acceptance criteria and names the winner." },
  { term: "The workspace", body: "Every feature gets .devrites/work/<slug>/, human-readable Markdown that survives compaction and new sessions. When you /clear, the next agent reads it and resumes. On ship, it is archived, never deleted." },
  { term: "Conventions & principles", body: "At seal, DevRites records the conventions it saw into a ledger the next feature reads. /rite-learn promotes recurring ones into project rules (.devrites/principles.md) that outrank the shipped engineering rules." },
  { term: "Run modes", body: "HITL (default) pauses risky slices before code, at a typed checkpoint. AFK runs unattended once you drop a .devrites/AFK file, but always pauses on destructive migrations, auth changes, public-API breaks, and red tests." },
];

export const CLI_COMMANDS: { cmd: string; note: string; exit?: string }[] = [
  { cmd: "devrites orient", note: "Workspace digest for the active feature (read-only)." },
  { cmd: "devrites ready", note: "Build-readiness gate.", exit: "0 ready · 2/3/4/5 not" },
  { cmd: "devrites evidence-fresh", note: "Evidence-freshness gate.", exit: "0 fresh · 3 stale" },
  { cmd: "devrites acceptance", note: "Acceptance-criteria gate.", exit: "0 proven · 1 gap" },
  { cmd: "devrites active | list | use <slug>", note: "Inspect or switch the active feature." },
  { cmd: 'devrites resolve <qid> "<answer>"', note: "Answer a HITL gate from the command line." },
];

export const RULES_ON_DEMAND = [
  "coding-style", "prose-style", "error-handling", "testing", "spec-grammar",
  "code-review", "principles", "security", "performance", "observability",
  "developer-experience", "patterns", "git-workflow", "hooks", "documentation",
  "development-workflow", "deprecation", "agents", "context-hygiene", "afk-hitl",
  "anti-patterns", "tooling",
];

export const SAFETY = [
  { h: "Project-local only", b: "Never writes to ~/.claude. Install and uninstall are manifest-managed." },
  { h: "Feature scope only", b: "Review, simplify, polish, and security stay within the active feature and touched files. No project-wide refactors or drive-by cleanup." },
  { h: "One slice at a time", b: "/rite-build stops after a single verified slice." },
  { h: "Evidence over confidence", b: "Claims need recorded commands, output, or screenshots." },
  { h: "Ask before danger", b: "Material assumptions, dependency additions, a second design system, destructive operations, and product-behavior changes are surfaced, not assumed." },
];

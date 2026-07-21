// Docs content - structured so pages stay accurate and in one place.
// Mirrors the shipped v3 pack: 42 skills, 14 agents, the Go control plane, and workspace schema.

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
      { href: "/docs/cli-mcp/", label: "Engine CLI" },
      { href: "/docs/architecture/", label: "Architecture" },
    ],
  },
];

export const DOCS_LINKS = DOCS_NAV.flatMap((g) => g.items);

export type Cmd = { cmd: string; phase: string; desc: string };

export const PUBLIC_COMMANDS: Cmd[] = [
  { cmd: "/rite", phase: "menu", desc: "Shows a compact menu and suggests the next command without running a phase." },
  { cmd: "/rite-quick", phase: "express", desc: "Handles a small, reversible change in one pass. Risky or multi-slice work moves to /rite-spec." },
  { cmd: "/rite-adopt", phase: "adopt", desc: "Reads an existing codebase and sets up DevRites before the first feature." },
  { cmd: "/rite-spec", phase: "spec", desc: "Investigates the request, writes spec.md, and creates the feature workspace." },
  { cmd: "/rite-temper", phase: "temper", desc: "Reviews scope, runs a pre-mortem, and strengthens the spec before planning." },
  { cmd: "/rite-define", phase: "define", desc: "Turns the approved spec into a plan and vertical task slices." },
  { cmd: "/rite-vet", phase: "vet", desc: "Reviews the plan before coding and writes the build-readable test-plan.md." },
  { cmd: "/rite-plan", phase: "plan", desc: "Breaks down, reslices, repairs, reorders, splits, or unblocks an active plan." },
  { cmd: "/rite-build", phase: "build", desc: "Implements one vertical slice test-first, then stops with evidence. Optional forge mode builds rival versions for a judge to compare." },
  { cmd: "/rite-converge", phase: "recovery", desc: "Compares live code with the spec, plan, and tasks, then adds the remaining work as traceable slices." },
  { cmd: "/rite-prove", phase: "prove", desc: "Checks the finished feature with tests, a build, the running application, and browser evidence when relevant." },
  { cmd: "/rite-polish", phase: "polish", desc: "Cleans up code without changing behavior, then checks the UI against the design system when relevant." },
  { cmd: "/rite-review", phase: "review", desc: "Runs parallel, fresh-context reviews of the feature diff on separate axes." },
  { cmd: "/rite-seal", phase: "seal", desc: "Checks acceptance, runs the relevant reviewers, and writes a GO or NO-GO decision." },
  { cmd: "/rite-ship", phase: "ship", desc: "After type-GO, runs the approved commit, push, and tag steps, then archives the workspace." },
  { cmd: "/rite-autocomplete", phase: "auto", desc: "Runs the lifecycle unattended and pauses at irreversible gates." },
  { cmd: "/rite-status", phase: "status", desc: "Reports the active feature's phase, run mode, next action, evidence, open gates, and risks." },
  { cmd: "/rite-resolve", phase: "resume", desc: "Answers, drops, or resolves a batch of open gates, then resumes the build." },
  { cmd: "/rite-zoom-out", phase: "utility", desc: "Maps the structure of an unfamiliar area using the project's own vocabulary." },
  { cmd: "/rite-prototype", phase: "utility", desc: "Builds disposable code to answer one design question, either as a logic harness or UI variants." },
  { cmd: "/rite-handoff", phase: "utility", desc: "Writes a handoff document and syncs the relevant context into workspace files." },
  { cmd: "/rite-pressure-test", phase: "utility", desc: "Develops several options for a rough idea, compares them, and chooses one." },
  { cmd: "/rite-frame", phase: "utility", desc: "Reframes the goal and checks likely failure modes before committing to an approach." },
  { cmd: "/rite-learn", phase: "utility", desc: "Looks through shipped work for recurring mistakes and dismissed findings, then proposes project-local lessons." },
  { cmd: "/rite-doctor", phase: "utility", desc: "Checks the installation and the active workspace." },
  { cmd: "/rite-customize", phase: "utility", desc: "Creates and validates a project-local reviewer override or workflow extension." },
  { cmd: "/rite-explain", phase: "utility", desc: "Explains a concept, diff, idea, or recent work for a human reader." },
  { cmd: "/rite-pov", phase: "utility", desc: "Evaluates an external option in the context of the project and recommends adopt, trial, hold, or reject." },
  { cmd: "/rite-dogfood", phase: "utility", desc: "Runs browser QA on changed user journeys and writes a dogfood report." },
  { cmd: "/rite-pr-feedback", phase: "utility", desc: "Fetches GitHub PR feedback, evaluates it, applies accepted fixes, replies, and resolves threads with evidence." },
];

export const INTERNAL_SKILLS: { name: string; trigger: string; role: string }[] = [
  { name: "devrites-interview", trigger: "underspecified ask", role: "One question at a time, with a best-guess and a confidence stop." },
  { name: "devrites-source-driven", trigger: "uncertain framework fact", role: "Consults docs or source and records the citation in evidence." },
  { name: "devrites-doubt", trigger: "non-trivial decision", role: "Adversarial: claim, extract, doubt, reconcile, stop." },
  { name: "devrites-frontend-craft", trigger: "UI in build or polish", role: "Register detection, shape before code, states, anti-slop guardrails." },
  { name: "devrites-prose-craft", trigger: "a phase writes prose", role: "Removes LLM writing tells while preserving exact technical terms and artifact structure." },
  { name: "devrites-browser-proof", trigger: "UI in prove or polish", role: "The browser proof ladder and the evidence schema." },
  { name: "devrites-refresh-indexes", trigger: "Stop hook or doctor --reindex", role: "Refreshes supported code-intelligence indexes after edits; no-ops when none are configured." },
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
  { name: "Host surfaces", tag: "Claude · Codex", body: "Generated project-local skills, agents, aliases, guidance, and hooks adapt the same canonical pack to each host." },
  { name: "Workflow skills", tag: "42 skills", body: "The rite-* lifecycle and utilities load only the phase contract and specialist guidance needed for the current step." },
  { name: "Fresh-context agents", tag: "13 read-only · 1 writer", body: "Reviewers, judges, a retrospector, and the single slice writer receive bounded inputs rather than the orchestrator's reasoning." },
  { name: "Control plane", tag: "devrites-engine", body: "A stdlib-only Go binary owns deterministic state transitions, gates, hooks, derivations, install/update, and migration for every host." },
  { name: "Persistent state", tag: ".devrites/", body: "Git-diffable Markdown workspaces, living capability specs, principles, conventions, learnings, and append-only traces survive cleared contexts." },
  { name: "Engineering standards", tag: "devrites-lib", body: "A compact core plus on-demand standards and checklists ship inside the shared library skill for Claude and Codex." },
  { name: "Install & manifest", tag: "npx devrites", body: "The engine-owned installer manages project artifacts and the optional shared binary; uninstall preserves runtime feature state." },
];

export const RATIONALE: { q: string; a: string }[] = [
  { q: "Why separate skills and an engine?", a: "Judgment stays in small phase skills that load only what they need. Deterministic bookkeeping lives once in the Go engine, where Claude, Codex, CI, and humans cannot drift into different gate logic." },
  { q: "Why the rite-* names?", a: "They form one collision-resistant workflow namespace across Claude slash commands and Codex dollar commands. Host-specific invocation changes; the underlying skill and engine command do not." },
  { q: "Why a thin menu skill, not a mega-router?", a: "/rite shows the menu, active status, and suggested next command. It does not run a phase. You can invoke each phase directly without loading the full lifecycle into context." },
  { q: "Why have internal skills?", a: "The model invokes specialists such as devrites-doubt and devrites-frontend-craft when a phase needs them. This keeps them out of the public menu and avoids loading unrelated guidance." },
  { q: "Why a Go control plane?", a: "State transitions, completeness checks, evidence freshness, hooks, and migration follow fixed rules. One stdlib-only binary makes those operations fast, testable, network-free, and consistent across both host adapters." },
  { q: "Why generated host artifacts?", a: "pack/.claude is the authoring source. The build generates Claude and Codex surfaces from it, preventing two manually maintained workflows from drifting apart." },
];

export const CLI_GATES = ["preamble", "snapshot", "build-readiness", "evidence-fresh", "check-acceptance", "resolve", "doctor", "help"];

export const WORKSPACE_FILES: { file: string; by: string; holds: string }[] = [
  { file: "README.md", by: "/rite-spec · every phase", holds: "Compact workspace map: phase, status, next action, artifacts, read-next paths, and blocking gates." },
  { file: "brief.md", by: "/rite-spec", holds: "One-line objective and the definition of done." },
  { file: "spec.md", by: "/rite-spec", holds: "Product WHAT/WHY: REQ-### requirements, AC-### criteria, boundaries, prohibitions, and capability deltas." },
  { file: "references.md · references/", by: "/rite-spec", holds: "Indexed external and user-supplied references: screenshots, Figma, video, and links." },
  { file: "strategy.md", by: "/rite-temper", holds: "Strategic spec review (optional): scope mode, pre-mortem, dimension scores." },
  { file: "architecture.md · flows.md", by: "/rite-define", holds: "Owning module, integration boundaries, data/API/event shape, risks, and optional clarifying diagrams." },
  { file: "plan.md", by: "/rite-define", holds: "Technical approach, slice strategy, validation links, and rollback." },
  { file: "tasks.md", by: "/rite-define", holds: "Ordered SLICE-### vertical slices with AC IDs, proof, mode, gate, dependencies, and done condition." },
  { file: "traceability.md", by: "/rite-define · /rite-prove", holds: "REQ/AC → slices → tests/evidence → touched-files coverage matrix." },
  { file: "eng-review.md", by: "/rite-vet", holds: "Engineering plan review: scope challenge, findings, failure modes, and parallel lanes." },
  { file: "test-plan.md", by: "/rite-vet", holds: "Build-readable coverage target: per-gap test requirements, acceptance-to-test map." },
  { file: "state.md", by: "every phase", holds: "The cursor: phase, status, active slice, risk, next step, plus any 'awaiting human' block." },
  { file: "questions.md", by: "every phase", holds: "Append-only Q&A: qid, slice, gate, status (open / answered / dropped)." },
  { file: "decisions.md · assumptions.md", by: "every phase", holds: "Running logs of the reasoning." },
  { file: "drift.md", by: "Spec Drift Guard", holds: "Drift events and how each was resolved." },
  { file: "forge-report.md", by: "/rite-build", holds: "Optional comparison and winner when a Forge: yes slice competes isolated candidates." },
  { file: "touched-files.md", by: "/rite-build", holds: "Which files this feature touched." },
  { file: "evidence.md", by: "/rite-build · /rite-prove", holds: "Recorded commands and their output." },
  { file: "browser-evidence.md", by: "/rite-prove · /rite-polish", holds: "Screenshots, console, network, viewport runs." },
  { file: "design-brief.md", by: "devrites-frontend-craft", holds: "Shape, states, and the design-references match for UI." },
  { file: "polish-report.md", by: "/rite-polish", holds: "Cleanup and UI-polish findings and fixes." },
  { file: "review.md", by: "/rite-review", holds: "Spec and Standards axes, severity-labelled findings." },
  { file: "seal.md", by: "/rite-seal", holds: "GO / NO-GO verdict, acceptance walk, blockers." },
  { file: "ship.md", by: "/rite-ship", holds: "What shipped: commit SHAs, branch, tag/PR, acceptance summary, follow-ups." },
  { file: "handoff.md", by: "/rite-handoff", holds: "Optional cold-resume map linking to canonical artifacts instead of duplicating them." },
];

export const INSTALL_FLAGS: { flag: string; effect: string }[] = [
  { flag: "--target DIR", effect: "Install into DIR (default: the current directory)." },
  { flag: "--dry-run", effect: "Show the planned file operations and exit, changing nothing." },
  { flag: "--force", effect: "Overwrite existing non-DevRites files." },
  { flag: "--no-agents", effect: "Skip the review subagents." },
  { flag: "--no-codex", effect: "Skip Codex skills, agents, hooks, and AGENTS.md integration." },
  { flag: "--no-skills", effect: "Skip skills and bundled engineering standards." },
  { flag: "--no-binary", effect: "Skip installing the shared devrites-engine executable." },
  { flag: "--short-aliases=all", effect: "Add /define, /build, /prove, /seal short aliases (off by default)." },
];

export const SETUP_TOOLS: { tool: string; gives: string }[] = [
  { tool: "codegraph", gives: "A code-intelligence index used by spec, define, and plan to inspect structure, placement, callers, and likely impact." },
  { tool: "graphify", gives: "A codebase-to-knowledge-graph for 'where is X, what calls Y, what would Z break'." },
  { tool: "Playwright MCP", gives: "Drives a real browser so prove and polish can record screenshots, console output, network activity, and responsive behavior." },
];

export const CONCEPTS: { term: string; body: string }[] = [
  { term: "Vertical slice", body: "A vertical slice is the unit of work. It crosses the layers needed for one verifiable result, such as the database, service, API, and UI. /rite-build implements one slice and records its evidence. You decide when to run the next." },
  { term: "Gate / checkpoint", body: "A gate is a fixed boundary check or a human decision point. Engine exit 3 means the workflow has paused for human input. Resolve the named gap and retry." },
  { term: "Evidence ladder", body: "Claims need the strongest available proof: real browser and DevTools evidence, host runtime checks, project E2E, then an exact manual fallback. A screenshot path is unproven until it is opened and described." },
  { term: "Spec Drift Guard", body: "When implementation shows that the plan is wrong, the build stops and records the mismatch in drift.md. If fixing it would change product behavior, DevRites asks you what to do before repairing the plan and resuming." },
  { term: "Forge", body: "An optional build mode. Two or three rival versions of a slice are built in isolation, then devrites-forge-judge scores them against the acceptance criteria and names the winner." },
  { term: "The workspace", body: "Each feature has a .devrites/work/<slug>/ directory containing human-readable Markdown. After compaction or /clear, the next agent reads these files and resumes from the recorded state. Ship moves the workspace into the archive." },
  { term: "Principles, conventions, learnings", body: "Principles prescribe project rules. Conventions record how the codebase already works, while learnings capture recurring mistakes and dismissed review findings. DevRites stores them separately because they carry different authority." },
  { term: "Living capability ledger", body: "Shipped feature deltas fold into .devrites/specs/<capability>/spec.md. New specs compare against what the product already does instead of treating every request as greenfield." },
  { term: "Run modes", body: "HITL (default) pauses risky slices before code, at a typed checkpoint. AFK runs unattended once you drop a .devrites/AFK file, but always pauses on destructive migrations, auth changes, public-API breaks, and red tests." },
];

export const CLI_COMMANDS: { cmd: string; note: string; exit?: string }[] = [
  { cmd: "devrites-engine preamble [slug]", note: "Workspace digest for the active or named feature." },
  { cmd: "devrites-engine snapshot [slug]", note: "Stable devrites.workspace.v1 status JSON." },
  { cmd: "devrites-engine build-readiness [slug]", note: "Plan-approved, build-ready gate.", exit: "0 ready · 3 pause" },
  { cmd: "devrites-engine evidence-fresh [slug]", note: "Proof must post-date every touched file.", exit: "0 fresh · 3 stale" },
  { cmd: "devrites-engine check-acceptance <dir>", note: "Acceptance criteria graded against seal evidence.", exit: "0 proven · 1 gap" },
  { cmd: 'devrites-engine resolve <qid> "<answer>"', note: "Answer a HITL gate and keep state.md consistent." },
  { cmd: "devrites-engine doctor", note: "Binary, pack, and workspace-schema compatibility verdict." },
  { cmd: "devrites-engine help", note: "Exhaustive current commands, hooks, exit codes, and environment." },
];

export const RULES_ON_DEMAND = [
  "coding-style", "prose-style", "error-handling", "testing", "spec-grammar",
  "code-review", "edge-case-trace", "principles", "security", "performance", "observability",
  "developer-experience", "patterns", "git-workflow", "hooks", "documentation",
  "development-workflow", "deprecation", "elicitation", "agents", "context-hygiene", "afk-hitl",
  "anti-patterns", "tooling", "skill-authoring", "definition-of-done", "review-checklist",
  "test-proof-checklist", "browser-proof-checklist", "security-checklist",
];

export const SAFETY = [
  { h: "Project-local host artifacts", b: "Skills, agents, standards, guidance, and hooks stay in the target project. Only the optional shared engine binary may live in a user or system bin directory." },
  { h: "Deterministic control plane", b: "Workspace state and gate commands make no model calls and no network calls; explicit install, update, and source-cache I/O is isolated." },
  { h: "Feature scope only", b: "Review, simplification, polish, and security checks stay within the active feature and touched files. Project-wide refactors require separate approval." },
  { h: "One slice at a time", b: "/rite-build stops after a single verified slice." },
  { h: "Evidence over confidence", b: "Claims need recorded commands, output, or screenshots." },
  { h: "Ask before danger", b: "DevRites asks before acting on material assumptions, adding dependencies or a second design system, running destructive operations, or changing product behavior." },
];

// Single source of truth for site copy + structured data.
// Selling, concrete, proof-anchored - every claim ties to a named mechanism.

export const SITE_URL = "https://devrites.com";
export const VERSION = process.env.NEXT_PUBLIC_DEVRITES_VERSION ?? "unreleased";
export const REPO = "https://github.com/ViktorsBaikers/DevRites";
export const INSTALL_CMD = "npx devrites@latest";
const RAW_INSTALLER_BASE = "https://raw.githubusercontent.com/ViktorsBaikers/DevRites/main";
export const CURL_CMD = `curl -fsSL ${RAW_INSTALLER_BASE}/install.sh | bash`;
export const CURL_PIN_CMD = `curl -fsSL ${RAW_INSTALLER_BASE}/install.sh | DEVRITES_REF=vX.Y.Z bash`;
export const CURL_UPDATE_CMD = `curl -fsSL ${RAW_INSTALLER_BASE}/update.sh | bash`;
export const CURL_UNINSTALL_CMD = `curl -fsSL ${RAW_INSTALLER_BASE}/uninstall.sh | bash`;

export type Stat = { value: number; suffix?: string; prefix?: string; label: string };

export const STATS: Stat[] = [
  { value: 42, label: "skills for the public workflow and focused specialist work" },
  { value: 14, label: "agents running in fresh context: 13 read-only reviewers and one slice writer" },
  { value: 2, label: "supported hosts with generated project-local surfaces: Claude and Codex" },
  { value: 0, label: "skills, agents, or hooks installed in global agent directories" },
];

export const TOOLS = [
  "Claude Code",
  "Codex",
  "CI",
  "local scripts",
  "a human",
];

export type Phase = {
  n: string;
  act: "shape" | "build" | "ship";
  name: string;
  title: string;
  body: string;
  cmd: string;
  out: string;
  optional?: boolean;
};

export const PHASES: Phase[] = [
  { n: "01", act: "shape", name: "spec", title: "Investigate before writing the spec", body: "Read the codebase, capability ledger, and shipped archive before writing the product contract and measurable acceptance criteria.", cmd: "/rite-spec", out: "brief.md · spec.md" },
  { n: "02", act: "shape", name: "temper", title: "Test the scope", body: "For an optional strategic pass, choose a scope posture, run a pre-mortem, and strengthen the contract before planning. Autocomplete runs this step automatically.", cmd: "/rite-temper", out: "strategy.md", optional: true },
  { n: "03", act: "shape", name: "define", title: "Plan vertical slices", body: "Turn the approved spec into an architecture, a plan, traceable vertical slices, and a matrix linking acceptance criteria to proof.", cmd: "/rite-define", out: "plan.md · tasks.md" },
  { n: "04", act: "shape", name: "vet", title: "Review the plan before coding", body: "Review reuse, architecture, test coverage, performance, and reversibility at a depth that matches the stakes. Vet is required.", cmd: "/rite-vet", out: "eng-review.md · test-plan.md" },
  { n: "05", act: "build", name: "build", title: "Build one slice", body: "A fresh-context writer implements one vertical slice test-first. The engine reconciles source changes, checks imports and test integrity, then records the evidence.", cmd: "/rite-build", out: "code · evidence.md" },
  { n: "06", act: "build", name: "prove", title: "Match claims to evidence", body: "Check each acceptance criterion against tests, the build, the running application, and real browser results. Traceability links every claim to its evidence.", cmd: "/rite-prove", out: "evidence.md · browser-evidence.md" },
  { n: "07", act: "build", name: "polish", title: "Polish without changing behavior", body: "Clean up the code, then polish the UI when the feature has one. Refresh any evidence affected by those changes before moving on.", cmd: "/rite-polish", out: "polish-report.md" },
  { n: "08", act: "build", name: "review", title: "Run an independent review", body: "Spec and standards reviewers inspect the feature diff in fresh contexts. The engine also rejects a review axis that records neither findings nor a clean result.", cmd: "/rite-review", out: "review.md" },
  { n: "09", act: "ship", name: "seal", title: "Decide GO or NO-GO", body: "Compare acceptance criteria with the evidence, run the relevant review panel, block on any Critical finding, and record the decision without changing git.", cmd: "/rite-seal", out: "seal.md" },
  { n: "10", act: "ship", name: "ship", title: "Ship after approval", body: "After a human types GO, run the approved git steps, update the capability ledger, archive the workspace, and clear the active-feature cursor.", cmd: "/rite-ship", out: "ship.md · archive/" },
];

export type Mechanism = {
  key: string;
  title: string;
  body: string;
  span: "wide" | "tall" | "std";
  tone: "accent" | "go" | "warn" | "danger";
  tags?: string[];
  demo?: string;
};

export const MECHANISMS: Mechanism[] = [
  {
    key: "engine",
    title: "A deterministic Go control plane",
    body: "The stdlib-only devrites-engine binary owns state transitions, gates, hooks, installation, updates, migration, and derived workspace data. Workspace commands run without model or network calls, so the same input produces the same result.",
    span: "wide",
    tone: "accent",
    demo: "devrites-engine readiness · snapshot · evidence-fresh",
  },
  {
    key: "harnesses",
    title: "One contract for Claude and Codex",
    body: "The canonical pack generates project-local skills, agents, standards, aliases, and hook adapters for both hosts. Each host has its own syntax but uses the same workspace and gates.",
    span: "std",
    tone: "go",
    tags: ["Claude Code", "Codex", "hooks", "generated"],
  },
  {
    key: "paper-trail",
    title: "Project files survive /clear",
    body: "Each feature has a folder under .devrites/work/. After you clear the context, the next agent reads those files and resumes from the recorded state instead of relying on a chat summary.",
    span: "wide",
    tone: "accent",
    demo: ".devrites/work/auth-tokens/ → next agent resumes cold",
  },
  {
    key: "drift",
    title: "Spec Drift Guard",
    body: "When implementation shows that the plan is wrong, the build stops, records the mismatch, and routes through plan repair before resuming.",
    span: "std",
    tone: "warn",
    demo: "drift detected · build paused → drift.md",
  },
  {
    key: "type-go",
    title: "type-GO",
    body: "The ship phase requires a human to type GO before it can commit, push, or tag anything in git.",
    span: "std",
    tone: "go",
    demo: "seal: GO · awaiting confirmation → type GO",
  },
  {
    key: "fanout",
    title: "Independent review panel",
    body: "At seal, fresh-context reviewers inspect the diff in parallel, each on a separate axis. They did not write the code and do not receive the builder's reasoning.",
    span: "std",
    tone: "accent",
    tags: ["spec", "code", "tests", "frontend", "security", "perf", "devex"],
  },
  {
    key: "security",
    title: "Security audit at the gate",
    body: "When the feature crosses an input, auth, data, secret, permission, or integration boundary, a security auditor reads the diff for the OWASP Top 10 and the LLM Top 10 on AI surfaces. The pack is scanned in CI too.",
    span: "std",
    tone: "danger",
    tags: ["OWASP Top 10", "LLM Top 10", "secrets", "supply chain"],
  },
  {
    key: "learn",
    title: "Learn from shipped work without rewriting rules",
    body: "DevRites stores observed conventions separately from prescriptive principles. /rite-learn looks through shipped work for recurring mistakes and dismissed findings, then proposes project-local lessons for later reviews.",
    span: "wide",
    tone: "go",
    demo: "archive → /rite-learn → learnings.md",
  },
  {
    key: "ledger",
    title: "Capability specs stay current",
    body: "Feature specs declare ADDED, MODIFIED, and REMOVED requirements. At ship, the engine applies those changes to .devrites/specs/, giving the next feature an up-to-date record of current behavior.",
    span: "std",
    tone: "accent",
    demo: "devrites-engine ledger diff · sync · validate",
  },
  {
    key: "afk",
    title: "AFK mode",
    body: "Add a .devrites/AFK file to run unattended. Destructive migrations, auth changes, public API breaks, and failing tests still pause the run. You can configure a notification for those pauses.",
    span: "std",
    tone: "accent",
    demo: ".devrites/AFK · pauses + pings on risk",
  },
];

export type Faq = { q: string; a: string };

export const FAQ: Faq[] = [
  {
    q: "Does DevRites send my code anywhere?",
    a: "The engine does not upload code or call a model. It stores workspace state as local Markdown under .devrites/. Install and update download release artifacts, and the optional source-cache hook handles only pages your host has already requested. Claude and Codex keep their own network behavior.",
  },
  {
    q: "Does it touch my global ~/.claude directory?",
    a: "The installer does not put skills, agents, standards, or hooks in ~/.claude or ~/.codex. It keeps those files in the target project and tracks them in a manifest. Unless you pass --no-binary, it may place the shared devrites-engine executable in a user or system bin directory. Uninstall leaves .devrites/work/ feature data in place.",
  },
  {
    q: "Which tools does it work with?",
    a: "DevRites supports Claude Code and Codex with generated project-local skills, agents, guidance, and hooks. CI, local scripts, other tools, and humans can use the same workspace gates through the devrites-engine CLI.",
  },
  {
    q: "How is it different from spec-kit, task-master, or BMAD?",
    a: "DevRites separates model judgment from deterministic bookkeeping. It keeps human-readable feature state on disk, while one Go engine handles phase completeness, evidence freshness, acceptance checks, hooks, and migration. Claude, Codex, CI, and the lifecycle skills all use that same contract.",
  },
  {
    q: "Is it free?",
    a: "Yes. DevRites is free and open source under the MIT license, available on GitHub.",
  },
  {
    q: "How do I install and uninstall?",
    a: `Run ${INSTALL_CMD} in your project, or use the raw GitHub install.sh command when Node is unavailable. Preview either path with --dry-run and use --no-binary if you do not want the shared engine installed. Update with npx devrites@latest update and uninstall with npx devrites@latest uninstall. Managed project files are removed while .devrites/work/ is preserved.`,
  },
  {
    q: "Has the pack itself been hardened against prompt injection?",
    a: "Yes. The agents use a prompt-injection-resistant baseline. CI checks each pack change for injection strings, hidden Unicode, and lockfile supply-chain indicators, and third-party actions are pinned. At seal, a security auditor reviews the diff against the OWASP Top 10 and, for AI features, the OWASP LLM Top 10.",
  },
];

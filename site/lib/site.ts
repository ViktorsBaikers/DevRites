// Single source of truth for site copy + structured data.
// Selling, concrete, proof-anchored — every claim ties to a named mechanism.

export const SITE_URL = "https://devrites.com";
export const VERSION = "2.3.0";
export const REPO = "https://github.com/ViktorsBaikers/DevRites";
export const INSTALL_CMD = "npx devrites@latest";
export const CURL_CMD = "curl -fsSL https://devrites.com | bash";

export type Stat = { value: number; suffix?: string; prefix?: string; label: string };

export const STATS: Stat[] = [
  { value: 10, label: "phases per feature, and each is proven before the next begins" },
  { value: 0, label: "files written outside your project. ~/.claude is never touched" },
  { value: 7, label: "independent reviewers fan out on the diff at seal" },
  { value: 100, suffix: "%", label: "open source under MIT. Read every line before you run it" },
];

export const TOOLS = [
  "Claude Code",
  "Cursor",
  "Codex",
  "Gemini CLI",
  "CI",
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
};

export const PHASES: Phase[] = [
  { n: "01", act: "shape", name: "spec", title: "Read first, write second", body: "A deep read of your codebase before a single line lands: where the feature fits, what it resolves, and acceptance criteria you can actually measure.", cmd: "/rite-spec", out: "spec.md · references/" },
  { n: "02", act: "shape", name: "temper", title: "Pressure-test the scope", body: "A strategic review of the spec: pick a scope mode, run a pre-mortem, and harden the contract so you build the right thing, not just a thing.", cmd: "/rite-temper", out: "strategy.md" },
  { n: "03", act: "shape", name: "define", title: "Slice it into vertical wins", body: "The approved spec becomes a plan and small vertical slices, each tagged for a checkpoint. Every acceptance criterion maps to a slice.", cmd: "/rite-define", out: "plan.md · tasks.md" },
  { n: "04", act: "shape", name: "vet", title: "Review the plan before any code", body: "A confidence-banded engineering review of the architecture, scope, and coverage. It writes the test plan the build will have to satisfy.", cmd: "/rite-vet", out: "eng-review.md · test-plan.md" },
  { n: "05", act: "build", name: "build", title: "One slice, then it stops", body: "A fresh-context agent builds exactly one vertical slice, test-first, and stops with evidence. Turn on forge and rival versions compete, then an independent judge picks the strongest. The Spec Drift Guard catches a wrong plan mid-build.", cmd: "/rite-build", out: "code · evidence.md" },
  { n: "06", act: "build", name: "prove", title: "Show the receipts", body: "Tests, build, types, lint, and real browser proof, walked against every acceptance criterion. Claims get backed by output, not adjectives.", cmd: "/rite-prove", out: "browser-evidence.md" },
  { n: "07", act: "build", name: "polish", title: "Tidy without breaking", body: "Behavior-preserving cleanup first, then UI polish against your design system. Nothing changes silently; stale evidence is re-proven.", cmd: "/rite-polish", out: "polish-report.md" },
  { n: "08", act: "build", name: "review", title: "Fresh eyes, in parallel", body: "Independent reviewers read the diff against the spec and your standards at once. None of them wrote the code.", cmd: "/rite-review", out: "review.md" },
  { n: "09", act: "ship", name: "seal", title: "GO or NO-GO", body: "Walk the acceptance criteria against the evidence, fan out the reviewers, block on a single Critical, and write the verdict before anything ships.", cmd: "/rite-seal", out: "seal.md" },
  { n: "10", act: "ship", name: "ship", title: "The irreversible step, gated", body: "Only after you type GO: the commit, push, and tag (or PR). Then the workspace is archived and the cursor cleared. Nothing irreversible happens on a hunch.", cmd: "/rite-ship", out: "ship.md" },
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
    key: "paper-trail",
    title: "The paper trail survives /clear",
    body: "Every feature gets a folder under .devrites/work/. When the context window fills and you clear it, the next agent reads these files and resumes exactly where the last one stopped. No summary, no re-explaining, no lost decisions.",
    span: "wide",
    tone: "accent",
    demo: ".devrites/work/auth-tokens/ → next agent resumes cold",
  },
  {
    key: "drift",
    title: "Spec Drift Guard",
    body: "When the build reveals the plan was wrong, it stops, records the drift, and routes through plan repair before resuming. A wrong turn never compounds into a wasted day.",
    span: "std",
    tone: "warn",
    demo: "⚠ drift detected · build paused → drift.md",
  },
  {
    key: "type-go",
    title: "type-GO",
    body: "No commit, push, or tag on a hunch. Ship demands a literal typed GO before anything irreversible touches your git history.",
    span: "std",
    tone: "go",
    demo: "seal: GO · awaiting confirmation → type GO",
  },
  {
    key: "fanout",
    title: "Reviewer fan-out",
    body: "At seal a panel of independent, fresh-context reviewers reads the diff in parallel, each on its own axis. None of them wrote the code, so none inherits its blind spots.",
    span: "std",
    tone: "accent",
    tags: ["spec", "code", "tests", "frontend", "security", "perf", "devex"],
  },
  {
    key: "security",
    title: "Security audit at the gate",
    body: "A security auditor reads the diff for the OWASP Top 10, plus the OWASP LLM Top 10 when the feature drives a model. The pack itself ships injection-resistant and is scanned in CI.",
    span: "std",
    tone: "danger",
    tags: ["OWASP Top 10", "LLM Top 10", "secrets", "supply chain"],
  },
  {
    key: "learn",
    title: "It learns your codebase's rules",
    body: "At seal, DevRites writes the conventions it just saw into a ledger. The next feature reads it at the start, so the agent follows your patterns instead of guessing them again. You promote the keepers with /rite-learn.",
    span: "wide",
    tone: "go",
    demo: "seal → conventions.md · promote with /rite-learn",
  },
  {
    key: "afk",
    title: "AFK mode",
    body: "Drop a .devrites/AFK file and it runs unattended. But destructive migrations, auth changes, public-API breaks, and red tests always pause, and it pings your phone when it needs you.",
    span: "std",
    tone: "accent",
    demo: ".devrites/AFK · pauses + pings on risk",
  },
];

export type Faq = { q: string; a: string };

export const FAQ: Faq[] = [
  {
    q: "Does DevRites send my code anywhere?",
    a: "No. DevRites is a pack of local Markdown skills and shell scripts. Feature state lives in .devrites/ on disk in your repo. Nothing is uploaded. The only network call the site itself makes is to the public GitHub release API to show the latest version.",
  },
  {
    q: "Does it touch my global ~/.claude directory?",
    a: "Never. DevRites installs project-local and refuses a global target. Every file it writes is recorded in .claude/devrites.manifest, and uninstall removes exactly those. Your feature data in .devrites/work/ is never touched.",
  },
  {
    q: "Which tools does it work with?",
    a: "It is built for Claude Code as slash commands. The state core is a tool-agnostic devrites CLI and a dependency-free MCP server, so Cursor, Codex, Gemini CLI, a CI job, or a human can drive the same workflow against the same files.",
  },
  {
    q: "How is it different from spec-kit, task-master, or BMAD?",
    a: "Those orchestrate work across many agents. DevRites keeps the discipline in tool-agnostic files plus deterministic gates (ready, evidence-fresh, acceptance), so one verdict agrees whether it comes from the CLI, the MCP server, or /rite-seal.",
  },
  {
    q: "Is it free?",
    a: "Yes. DevRites is free and open source under the MIT license, available on GitHub.",
  },
  {
    q: "How do I install and uninstall?",
    a: "Run npx devrites@latest in your project, or curl -fsSL https://devrites.com | bash. The npx path installs offline and pins to the version you ask for; preview either with --dry-run. Uninstall with npx devrites@latest uninstall or curl -fsSL https://devrites.com/remove | bash. It removes only what DevRites installed and preserves your feature data.",
  },
  {
    q: "Has the pack itself been hardened against prompt injection?",
    a: "Yes. The agents run on a prompt-injection-resistant baseline, and CI scans the pack on every change for injection strings, hidden unicode, and supply-chain indicators in the lockfile, with third-party actions pinned. At seal a security auditor reviews the diff for the OWASP Top 10, and the OWASP LLM Top 10 when the feature has an AI surface.",
  },
];

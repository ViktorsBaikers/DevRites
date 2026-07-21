import { ArrowUpRight, FileWarning, RefreshCw, ScanSearch, ShieldCheck } from "lucide-react";

const CHECKS = [
  {
    command: "readiness auth-tokens",
    title: "Required files exist",
    body: "Stops a phase change when its contract, decisions, or tasks are missing.",
    icon: ScanSearch,
  },
  {
    command: "evidence-fresh auth-tokens",
    title: "Proof matches the diff",
    body: "Stops GO when a touched source file is newer than the recorded evidence.",
    icon: RefreshCw,
  },
  {
    command: "check-acceptance .devrites/work/auth-tokens",
    title: "Acceptance coverage is complete",
    body: "Names each acceptance ID that is still unproven or unchecked in seal.md.",
    icon: FileWarning,
  },
  {
    command: "review-integrity auth-tokens",
    title: "Review has a recorded result",
    body: "Flags an adversarial review axis with neither findings nor a clean justification.",
    icon: ShieldCheck,
  },
];

export default function Contrast() {
  return (
    <section id="why" className="engine-proof" aria-labelledby="proof-title">
      <div className="wrap py-28 md:py-40">
        <header className="max-w-4xl">
          <h2
            id="proof-title"
            className="font-bold [font-size:clamp(3rem,5.4vw,5.4rem)] leading-[0.92] tracking-[-0.04em]"
          >
            See why a release is blocked.
          </h2>
          <p className="mt-7 max-w-2xl text-lg leading-relaxed text-ink-muted md:text-xl">
            The Go engine checks the repository, names the failed rule, and shows what to fix.
          </p>
        </header>

        <div className="engine-proof-layout mt-12">
          <article className="engine-output" aria-label="Example blocked acceptance check">
            <header>
              <span>Example engine result</span>
              <strong>exit 1</strong>
            </header>
            <div className="engine-output-body">
              <code className="engine-command">$ devrites-engine check-acceptance .devrites/work/auth-tokens</code>
              <div className="engine-verdict">
                <span>BLOCKED</span>
                <strong>2 / 3 criteria proven</strong>
              </div>
              <dl>
                <div><dt>Missing</dt><dd><code>AC-003</code></dd></div>
                <div><dt>Record</dt><dd><code>seal.md</code></dd></div>
                <div><dt>Next</dt><dd>Prove AC-003, check it in the seal, then rerun.</dd></div>
              </dl>
            </div>
            <footer>The result names the failed rule without calling a model.</footer>
          </article>

          <div className="engine-checks" aria-label="Deterministic release checks">
            {CHECKS.map(({ command, title, body, icon: Icon }) => (
              <article key={command}>
                <Icon className="size-5" strokeWidth={1.8} aria-hidden />
                <div>
                  <code>devrites-engine {command}</code>
                  <h3>{title}</h3>
                  <p>{body}</p>
                </div>
              </article>
            ))}
          </div>
        </div>

        <a className="engine-proof-link" href="/docs/cli-mcp/">
          Read the engine CLI reference <ArrowUpRight className="size-4" aria-hidden />
        </a>
      </div>
    </section>
  );
}

import { ArrowUpRight, FileWarning, RefreshCw, ScanSearch, ShieldCheck } from "lucide-react";

const CHECKS = [
  {
    command: "check readiness auth-tokens",
    title: "Build inputs are bound",
    body: "Checks required artifacts, the task graph, and the approved Build-input binding.",
    icon: ScanSearch,
  },
  {
    command: "check seal auth-tokens",
    title: "Proof matches the candidate",
    body: "Rechecks files, task graph, Build-input binding, and evidence freshness before GO.",
    icon: RefreshCw,
  },
  {
    command: "gates status auth-tokens",
    title: "Gate results stay on the record",
    body: "Reports the machine-checked gate ledger for the active feature.",
    icon: FileWarning,
  },
  {
    command: "secret-scan auth-tokens",
    title: "High-risk secrets block",
    body: "Scans exact staged blobs or touched files and blocks on HIGH findings.",
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
              <code className="engine-command">$ devrites-engine check seal auth-tokens</code>
              <div className="engine-verdict">
                <span>BLOCKED</span>
                <strong>seal evidence is stale</strong>
              </div>
              <dl>
                <div><dt>Changed</dt><dd><code>site/components/Nav.tsx</code></dd></div>
                <div><dt>Record</dt><dd><code>evidence.md</code></dd></div>
                <div><dt>Next</dt><dd>Refresh affected proof, then rerun the seal check.</dd></div>
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

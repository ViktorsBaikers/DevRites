import { Bot, Braces, Check, Cpu, LockKeyhole } from "lucide-react";

const WORKSPACE_FILES = ["spec.md", "tasks.md", "evidence.md", "seal.md"];

export default function Anywhere() {
  return (
    <section id="anywhere" className="hosts-section" aria-labelledby="hosts-title">
      <div className="wrap">
        <header className="hosts-heading">
          <h2 id="hosts-title">Use Claude or Codex without losing release state.</h2>
          <p>
            Each host uses its native command syntax, but both read the same project files and call the same engine.
          </p>
        </header>

        <div className="host-router" aria-label="Claude Code and Codex converge on one DevRites engine and workspace">
          <article className="host-node host-node--claude">
            <Bot className="size-6" strokeWidth={1.7} aria-hidden />
            <div>
              <span>Project skill</span>
              <strong>Claude Code</strong>
            </div>
            <code>/rite-prove</code>
          </article>

          <div className="host-route host-route--claude" aria-hidden="true">
            <span><code>/rite-prove</code></span>
          </div>

          <article className="host-engine-node">
            <Cpu className="size-7" strokeWidth={1.6} aria-hidden />
            <span>Shared control plane</span>
            <strong>devrites-engine</strong>
            <code>evidence-fresh</code>
          </article>

          <div className="host-route host-route--codex" aria-hidden="true">
            <span><code>$rite-prove</code></span>
          </div>

          <article className="host-node host-node--codex">
            <Braces className="size-6" strokeWidth={1.7} aria-hidden />
            <div>
              <span>Project skill</span>
              <strong>Codex</strong>
            </div>
            <code>$rite-prove</code>
          </article>

          <article className="host-workspace">
            <div className="host-workspace-head">
              <div>
                <span>Shared on disk</span>
                <code>.devrites/work/auth-tokens/</code>
              </div>
              <p><LockKeyhole className="size-4" strokeWidth={1.8} aria-hidden /> Human approval is still required</p>
            </div>
            <div className="host-workspace-files">
              {WORKSPACE_FILES.map((file) => (
                <span key={file}><Check className="size-3.5" strokeWidth={2.3} aria-hidden /><code>{file}</code></span>
              ))}
            </div>
          </article>
        </div>

        <p className="hosts-note">Workspace commands run without model or network calls.</p>
      </div>
    </section>
  );
}

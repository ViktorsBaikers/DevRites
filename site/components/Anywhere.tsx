import { Check, Cpu, LockKeyhole } from "lucide-react";

const HOSTS = [
  ["Claude Code", "/rite-prove"],
  ["Codex", "$rite-prove"],
  ["omp", "rite-prove"],
  ["pi", "rite-prove"],
  ["Devin CLI", "rite-prove"],
];

const WORKSPACE_FILES = ["spec.md", "tasks.md", "evidence.md", "review.md", "seal.md"];

export default function Anywhere() {
  return (
    <section id="anywhere" className="hosts-section" aria-labelledby="hosts-title">
      <div className="wrap">
        <header className="hosts-heading">
          <p className="section-kicker">Host register / 05</p>
          <h2 id="hosts-title">Change the agent. Keep the record.</h2>
          <p>Five host surfaces read the same repository-local feature workspace and call the same deterministic Go control plane.</p>
        </header>

        <div className="host-register">
          <ol aria-label="Supported agent hosts">
            {HOSTS.map(([host, command], index) => (
              <li key={host}>
                <span>{String(index + 1).padStart(2, "0")}</span>
                <strong>{host}</strong>
                <code>{command}</code>
              </li>
            ))}
          </ol>

          <article className="host-engine-card">
            <Cpu aria-hidden />
            <span>Shared control plane</span>
            <strong>devrites-engine</strong>
            <code>check seal &lt;slug&gt;</code>
            <p>Workspace checks run without model or network calls.</p>
          </article>

          <article className="host-workspace-card">
            <header>
              <span>Shared on disk</span>
              <code>.devrites/work/&lt;slug&gt;/</code>
            </header>
            <div>
              {WORKSPACE_FILES.map((file) => <span key={file}><Check aria-hidden /><code>{file}</code></span>)}
            </div>
            <p><LockKeyhole aria-hidden />Human approval is still required.</p>
          </article>
        </div>
      </div>
    </section>
  );
}

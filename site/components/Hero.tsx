import { BadgeCheck, Code2, FileText, PackageCheck, Users } from "lucide-react";

const RAIL = ["Change register", "How it works", "Role profiles", "Integrations", "Spec example", "Review flow", "Ship process", "FAQ"];

const FLOW = [
  { n: "01", name: "Spec", body: "Define the change in your repository.", icon: FileText },
  { n: "02", name: "Build", body: "Agents implement against the spec.", icon: Code2 },
  { n: "03", name: "Prove", body: "Attach current evidence: tests, runs, artifacts.", icon: BadgeCheck },
  { n: "04", name: "Review", body: "Independent review by designated roles.", icon: Users },
  { n: "05", name: "Ship", body: "Human-gated release to production.", icon: PackageCheck },
];

const PROOF = [
  ["Spec is complete and unambiguous", ".devrites/work/<slug>/spec.md", true],
  ["Implementation matches the spec", "touched-files.md", true],
  ["Automated tests pass", "evidence.md", true],
  ["Security review completed", "review.md", true],
  ["Documentation updated", "docs/", false],
  ["Independent review approved", "seal.md", false],
  ["Human GO to ship", "pending", false],
] as const;

export default function Hero() {
  return (
    <section
      id="top"
      className="register-hero"
      aria-label="DevRites change register"
      style={{ backgroundImage: 'url("/assets/plates/paper-ground.png")' }}
    >
      <aside className="register-folio" aria-label="Page contents">
        <ol>
          {RAIL.map((label, index) => (
            <li key={label} className={index === 0 ? "is-active" : ""}>
              <span>{String(index + 1).padStart(2, "0")}</span>
              <a href={["#top", "#workflow", "#mechanisms", "#anywhere", "#spec-example", "#review-flow", "#ship-process", "#faq"][index]}>{label}</a>
            </li>
          ))}
        </ol>
        <p>Small changes.<br />Safer software.</p>
        <b>DR<small>MMXXVI</small></b>
      </aside>

      <div className="register-bar" aria-hidden />

      <div className="register-page">
        <header className="register-title-row">
          <p>DevRites engineering change register</p>
          <span>Specifications enable progress</span>
        </header>

        <article className="change-sheet">
          <header className="change-meta">
            <div className="change-id-label">Change<br />request no.</div>
            <p className="change-id">DR-5.12.0</p>
            <dl>
              <div><dt>Rev.</dt><dd>A</dd></div>
              <div><dt>Date</dt><dd>2026-09-20</dd></div>
              <div><dt>Status</dt><dd><strong>Current</strong></dd></div>
              <div><dt>Pages</dt><dd>1 of 1</dd></div>
              <div className="change-classification">
                <span><b>Type</b>Feature</span>
                <span><b>Area</b>Agent workflow</span>
                <span><b>Repo</b>Local</span>
                <span><b>Priority</b>Normal</span>
              </div>
            </dl>
          </header>

          <section className="change-proposition" aria-labelledby="change-title">
            <div className="change-proposition-copy">
              <p>Title / proposition</p>
              <h1 id="change-title">AI can write the diff. Make it prove the release.</h1>
              <h3>Repository-local. Evidence-bound. Human-approved.</h3>
            </div>
            <p className="pencil-note"><span>Higher velocity.<br />Same standards.</span></p>
            <fieldset>
              <legend>Change bar</legend>
              {[["New feature", true], ["Spec update", false], ["Behavior change", false], ["Deprecation", false], ["Docs only", false]].map(([label, checked]) => (
                <label key={String(label)}><span aria-hidden>{checked ? "■" : "□"}</span>{label}</label>
              ))}
            </fieldset>
          </section>

          <section className="change-workflow" aria-labelledby="workflow-register-title">
            <h3 id="workflow-register-title">Workflow (spec to ship)</h3>
            <ol>
              {FLOW.map(({ n, name, body, icon: Icon }) => (
                <li key={name}>
                  <span>{n}</span>
                  <Icon aria-hidden />
                  <strong>{name}</strong>
                  <p>{body}</p>
                </li>
              ))}
            </ol>
            <aside>
              <span>Key details</span>
              <strong>18 <small>role profiles</small></strong>
              <strong>5 <small>supported hosts</small></strong>
              <code>evidence current</code>
              <p>A more reliable software supply chain</p>
            </aside>
          </section>

          <section className="proof-register" aria-labelledby="proof-register-title">
            <div className="proof-table-wrap">
              <h3 id="proof-register-title">Acceptance / proof register</h3>
              <table>
                <thead><tr><th>#</th><th>Criterion</th><th>Evidence / reference</th><th>Status</th><th>Checked by</th><th>Date</th></tr></thead>
                <tbody>
                  {PROOF.map(([criterion, evidence, checked], index) => (
                    <tr key={criterion}><td>{index + 1}</td><td>{criterion}</td><td><code>{evidence}</code></td><td>{checked ? "☑" : "☐"}</td><td>—</td><td>—</td></tr>
                  ))}
                </tbody>
              </table>
              <div className="proof-notes"><span>Notes</span><p className="proof-pencil">Evidence over claims.<br />Ship with confidence.</p></div>
            </div>

            <aside className="approval-block">
              <h3>Human approval</h3>
              <img className="register-stamp" src="/assets/plates/approval-stamp.png" width={1293} height={1216} alt="Human GO required" />
              <p>Not<br />without<br />a human.</p>
              <small>People keep<br />software honest</small>
            </aside>
          </section>

          <footer className="change-footer">
            <p>Spec <span>→</span> Proof <span>→</span> People <span>→</span> Better software.</p>
            <code>DR-5.12.0</code>
          </footer>
        </article>

        <a className="next-folio" href="#workflow">
          <span>02</span><strong>How it works</strong><small>The change becomes a reliable release</small><b>Next spec →</b>
        </a>
      </div>
    </section>
  );
}

"use client";

import { useState, type CSSProperties } from "react";
import {
  ArrowRight,
  ArrowUpRight,
  Check,
  FileText,
  LockKeyhole,
  Plus,
} from "lucide-react";
import { FAQ, INSTALL_CMD, PHASES, REPO } from "@/lib/site";
import { CopyButton } from "./ui";
import "@/app/register-sections.css";

const STAGES = [
  {
    title: "Shape",
    heading: "Agree on what done means.",
    body: "Investigate the repository, resolve ambiguity, and review the plan before implementation begins.",
    phases: ["spec", "clarify", "temper", "define", "vet"],
    file: "spec.md",
    excerpt:
      "# Export project activity\n\nOutcome\nA maintainer can download the current activity.\n\nAcceptance\n- Export respects the selected date range.\n- Empty results produce a valid file.\n- Existing permissions still apply.",
    note: "A clear contract is the first safety gate.",
  },
  {
    title: "Build",
    heading: "Give the writer a bounded job.",
    body: "One fresh-context writer implements one slice against agreed paths and acceptance criteria. A changed plan goes back through repair.",
    phases: ["build", "converge"],
    file: "tasks.md",
    excerpt:
      "# Slice: activity export\n\nInputs\nApproved spec and implementation plan\n\nWork\n- Implement the export within assigned paths.\n- Test the acceptance criteria.\n- Record touched files and outcomes.",
    note: "Small scope. Reviewable changes.",
  },
  {
    title: "Prove",
    heading: "Make the evidence match the code.",
    body: "Tests and runtime checks support the current candidate. Polish or corrections that change the code require the affected proof to be refreshed.",
    phases: ["prove", "polish", "review"],
    file: "evidence.md",
    excerpt:
      "# Acceptance evidence\n\nDate range → boundary tests\nEmpty results → empty-export test\nPermissions → access-control checks\n\nRecord the command, result, and candidate.\nMissing or stale evidence is not a pass.",
    note: "A test result is useful only for the code it checked.",
  },
  {
    title: "Ship",
    heading: "Leave the release decision to a human.",
    body: "Seal records GO or NO-GO against the evidence. Git release actions require a separate human approval of the exact proposed steps.",
    phases: ["seal", "ship"],
    file: "seal.md",
    excerpt:
      "# Release decision\n\nAcceptance → checked\nEvidence → current\nReview findings → resolved\n\nHuman approval → still required\n\nA seal decision does not execute Git actions.",
    note: "Ready to ship is not permission to ship.",
  },
];

const ROLE_GROUPS = [
  {
    title: "Frame the work",
    roles: [
      ["Evidence scout", "Finds live evidence for one bounded question."],
      [
        "Strategy reviewer",
        "Challenges scope, assumptions, and failure modes.",
      ],
      ["Plan drafter", "Connects architecture, slices, and acceptance."],
      ["Plan reviewer", "Checks the plan before code is written."],
      ["Doubt reviewer", "Tries to break a specific claim."],
    ],
  },
  {
    title: "Build and prove",
    roles: [
      ["Slice wright", "Writes code and tests within an exact path contract."],
      ["Forge judge", "Compares candidate implementations for one slice."],
      ["Proof runner", "Maps recorded proof to acceptance criteria."],
      [
        "Simplifier reviewer",
        "Finds behavior-preserving reductions in complexity.",
      ],
    ],
  },
  {
    title: "Review the change",
    roles: [
      ["Code reviewer", "Checks correctness and maintainability."],
      ["Spec reviewer", "Finds missing acceptance and unwanted scope."],
      ["Test analyst", "Checks whether tests actually prove the behavior."],
      ["Security auditor", "Examines trust boundaries and security risks."],
      [
        "Frontend reviewer",
        "Checks usability, accessibility, and visual quality.",
      ],
      ["Performance reviewer", "Investigates relevant performance risks."],
      ["DevEx reviewer", "Finds friction for the next developer."],
    ],
  },
  {
    title: "Carry learning forward",
    roles: [
      ["Retrospector", "Identifies recurring lessons across features."],
      [
        "Upgrade planner",
        "Assesses existing workspaces against the current contract.",
      ],
    ],
  },
];

function FolioHeading({
  number,
  title,
  children,
}: {
  number: string;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <header className="folio-heading">
      <span>{number}</span>
      <h2>{title}</h2>
      <p>{children}</p>
    </header>
  );
}

function Workflow() {
  const [active, setActive] = useState(0);
  const [pointerMotion, setPointerMotion] = useState(true);
  const stage = STAGES[active];
  return (
    <section id="workflow" className="folio-section">
      <FolioHeading number="02" title="How it works">
        One feature. One durable record.
      </FolioHeading>
      <div className="folio-intro">
        <h3>From an idea to a release you can explain.</h3>
        <p>
          DevRites keeps the contract, work, and evidence in your repository.
          Clear the conversation, switch hosts, or hand over to another
          engineer: the record stays.
        </p>
      </div>
      <div className="folio-workflow">
        <div
          className="folio-stage-controls"
          role="group"
          aria-label="Explore the workflow"
          data-pointer-motion={pointerMotion}
          style={{ "--active-stage": active } as CSSProperties}
        >
          {STAGES.map((item, index) => (
            <button
              key={item.title}
              type="button"
              aria-pressed={index === active}
              aria-controls="workflow-description workflow-example"
              onClick={(event) => {
                const animate = event.detail > 0;
                const controls = event.currentTarget.parentElement!;
                controls.dataset.pointerMotion = String(animate);
                // Commit the input mode before changing the sliding surface's target.
                getComputedStyle(controls, "::before").transform;
                setPointerMotion(animate);
                setActive(index);
              }}
            >
              <span className="folio-stage-number">
                {String(index + 1).padStart(2, "0")}
              </span>
              <strong>{item.title}</strong>
              <ArrowRight aria-hidden />
            </button>
          ))}
        </div>
        <div id="workflow-description" className="folio-stage-copy">
          <h4>{stage.heading}</h4>
          <p>{stage.body}</p>
          <div className="folio-command-list">
            {stage.phases.map((name) => {
              const phase = PHASES.find((item) => item.name === name);
              return phase ? (
                <code key={name}>
                  {phase.cmd}
                  {phase.optional ? " (optional)" : ""}
                </code>
              ) : null;
            })}
          </div>
          <a href="/docs/flow/">
            Read the complete lifecycle <ArrowUpRight aria-hidden />
          </a>
        </div>
        <div
          id="workflow-example"
          className="folio-artifact"
          aria-live="polite"
          aria-atomic="true"
        >
          <header>
            <FileText aria-hidden />
            <code>{stage.file}</code>
            <span>Illustrative excerpt</span>
          </header>
          <pre
            key={active}
            className={pointerMotion ? "folio-artifact-change" : ""}
          >
            {stage.excerpt}
          </pre>
          <p className="folio-note">{stage.note}</p>
        </div>
      </div>
      <p className="folio-footnote">
        Stored under <code>.devrites/work/&lt;feature&gt;/</code>. These
        examples explain the workflow; they are not generated project results.
      </p>
    </section>
  );
}

function Roles() {
  return (
    <section id="mechanisms" className="folio-section">
      <FolioHeading number="03" title="Role profiles">
        Different questions deserve different reviewers.
      </FolioHeading>
      <div className="folio-intro">
        <h3>Give every judgment an owner.</h3>
        <p>
          The writer implements. Independent specialists inspect the work from
          fresh context. The lifecycle selects the roles the change needs; every
          feature does not need every reviewer.
        </p>
      </div>
      <div className="folio-role-register">
        {ROLE_GROUPS.map((group) => (
          <div className="folio-role-group" key={group.title}>
            <h4>
              {group.title}
              <span className="folio-role-count">
                {group.roles.length} profiles
              </span>
            </h4>
            <dl>
              {group.roles.map(([name, description]) => (
                <div key={name}>
                  <dt>{name}</dt>
                  <dd>{description}</dd>
                </div>
              ))}
            </dl>
          </div>
        ))}
      </div>
      <div className="folio-bottomline">
        <p>
          <strong>18 profiles.</strong> One bounded writer; independent,
          read-only specialists.
        </p>
        <a href={`${REPO}/tree/main/pack/.claude/agents`}>
          Inspect the role definitions <ArrowUpRight aria-hidden />
        </a>
      </div>
    </section>
  );
}

function Integrations() {
  return (
    <section id="anywhere" className="folio-section">
      <FolioHeading number="04" title="Integrations">
        Your host changes. Your project record stays.
      </FolioHeading>
      <div className="folio-intro">
        <h3>Work where you already work.</h3>
        <p>
          Claude Code, Codex, omp, pi, and Devin CLI use project-local workflow
          surfaces. The same Go engine checks the workspace underneath them.
        </p>
      </div>
      <div className="folio-host-layout">
        <div className="folio-host-list">
          {[
            ["Claude Code", "/rite-prove"],
            ["Codex", "$rite-prove"],
            ["omp", "/rite-prove"],
            ["pi", "/rite-prove"],
            ["Devin CLI", "/rite-prove"],
          ].map(([host, command]) => (
            <div key={host}>
              <strong>{host}</strong>
              <code>{command}</code>
              <ArrowRight aria-hidden />
            </div>
          ))}
        </div>
        <div className="folio-shared-record">
          <h4>One shared workspace</h4>
          <code>.devrites/work/&lt;feature&gt;/</code>
          <ul>
            {["spec.md", "tasks.md", "evidence.md", "review.md", "seal.md"].map(
              (file) => (
                <li key={file}>
                  <FileText aria-hidden />
                  <code>{file}</code>
                </li>
              ),
            )}
          </ul>
          <p>
            Human-readable Markdown. Local state. A handover that survives the
            chat.
          </p>
        </div>
      </div>
      <div className="folio-engine-line">
        <div>
          <h4>Also available to scripts and CI.</h4>
          <p>Run deterministic workspace checks without a model call.</p>
        </div>
        <code>devrites-engine check seal &lt;feature&gt;</code>
        <a href="/docs/cli-mcp/">
          Engine reference <ArrowUpRight aria-hidden />
        </a>
      </div>
    </section>
  );
}

function SpecExample() {
  return (
    <section id="spec-example" className="folio-section">
      <FolioHeading number="05" title="Spec example">
        A small contract is easier to keep.
      </FolioHeading>
      <div className="folio-spec-layout">
        <div>
          <h3>
            Write the outcome.
            <br />
            Make it testable.
          </h3>
          <p>
            A useful spec tells the writer what must change, what must stay, and
            how to tell the difference. Start with behavior, then connect each
            criterion to proof.
          </p>
          <p className="folio-note">
            If “done” is vague,
            <br />
            the review will be too.
          </p>
          <a href="/docs/getting-started/">
            Start your first feature <ArrowUpRight aria-hidden />
          </a>
        </div>
        <article className="folio-spec-document">
          <header>
            <code>spec.md</code>
            <span>Illustrative feature</span>
          </header>
          <h4>Export project activity</h4>
          <p>
            A maintainer can download activity for a selected date range without
            changing who can access it.
          </p>
          <dl>
            <div>
              <dt>In scope</dt>
              <dd>A CSV export of the activity already visible to the user.</dd>
            </div>
            <div>
              <dt>Out of scope</dt>
              <dd>Scheduled exports, email delivery, and new permissions.</dd>
            </div>
          </dl>
          <h5>Acceptance / expected proof</h5>
          <ol>
            <li>
              <span>Selected dates define the export.</span>
              <small>Boundary tests</small>
            </li>
            <li>
              <span>An empty range returns a valid file.</span>
              <small>Empty-result test</small>
            </li>
            <li>
              <span>Existing access rules still apply.</span>
              <small>Permission checks</small>
            </li>
          </ol>
        </article>
      </div>
    </section>
  );
}

function ReviewFlow() {
  return (
    <section id="review-flow" className="folio-section">
      <FolioHeading number="06" title="Review flow">
        Claims on one side. Evidence on the other.
      </FolioHeading>
      <div className="folio-intro">
        <h3>The diff gets a second opinion.</h3>
        <p>
          Independent reviewers inspect the candidate against the contract and
          recorded checks. Findings become bounded corrections, and affected
          proof is refreshed before release.
        </p>
      </div>
      <ol className="folio-review-route">
        {[
          [
            "Candidate",
            "The exact code being reviewed",
            "Changed files + acceptance",
          ],
          [
            "Evidence",
            "What the checks actually showed",
            "Tests + runtime results",
          ],
          [
            "Review",
            "Independent findings and corrections",
            "Relevant specialist reports",
          ],
          ["Seal", "A recorded release decision", "GO or NO-GO"],
        ].map(([title, body, record]) => (
          <li key={title}>
            <h4>{title}</h4>
            <p>{body}</p>
            <code>{record}</code>
          </li>
        ))}
      </ol>
      <div className="folio-review-rule">
        <LockKeyhole aria-hidden />
        <div>
          <h4>Changed code needs current evidence.</h4>
          <p>
            A previous pass cannot prove a new candidate. If a correction
            changes the implementation, refresh the affected checks before
            sealing.
          </p>
        </div>
        <a href="/docs/concepts/">
          Understand the gates <ArrowUpRight aria-hidden />
        </a>
      </div>
    </section>
  );
}

function ShipProcess() {
  return (
    <section id="ship-process" className="folio-section">
      <FolioHeading number="07" title="Ship process">
        The last gate belongs to you.
      </FolioHeading>
      <div className="folio-ship-layout">
        <div>
          <h3>
            Ready is a finding.
            <br />
            GO is your decision.
          </h3>
          <p>
            DevRites can prepare the release and check its evidence. A human
            approves the exact Git actions before they execute. AFK mode does
            not remove that boundary.
          </p>
          <a href="/docs/flow/">
            Read the release contract <ArrowUpRight aria-hidden />
          </a>
        </div>
        <ol className="folio-release-checklist">
          <li>
            <Check aria-hidden />
            <div>
              <strong>Check the candidate</strong>
              <p>
                Acceptance, evidence freshness, and review findings are checked.
              </p>
            </div>
          </li>
          <li>
            <FileText aria-hidden />
            <div>
              <strong>Record the seal</strong>
              <p>GO or NO-GO is recorded with the supporting evidence.</p>
            </div>
          </li>
          <li className="folio-human-gate">
            <LockKeyhole aria-hidden />
            <div>
              <strong>Approve the exact release actions</strong>
              <p>A human confirms the proposed commit, push, or tag steps.</p>
            </div>
          </li>
          <li>
            <ArrowRight aria-hidden />
            <div>
              <strong>Ship and retain the record</strong>
              <p>
                The approved steps run and the feature workspace is archived.
              </p>
            </div>
          </li>
        </ol>
      </div>
    </section>
  );
}

function Questions() {
  return (
    <section id="faq" className="folio-section">
      <FolioHeading number="08" title="Questions, answered">
        Know what enters your repository.
      </FolioHeading>
      <div className="folio-faq">
        {FAQ.map(({ q, a }) => (
          <details key={q}>
            <summary>
              {q}
              <Plus aria-hidden />
            </summary>
            <p>{a}</p>
          </details>
        ))}
      </div>
    </section>
  );
}

function StartAndFooter() {
  return (
    <>
      <section id="install" className="folio-section folio-install">
        <div>
          <h2>
            Put the next change
            <br />
            on the record.
          </h2>
          <p>
            Run the installer from your project. Start with the installation
            guide to choose your host and make your first feature.
          </p>
          <a href="/docs/getting-started/">
            Installation guide <ArrowRight aria-hidden />
          </a>
        </div>
        <div className="folio-install-command">
          <code>{INSTALL_CMD}</code>
          <CopyButton text={INSTALL_CMD} label="Copy" />
          <p>
            Preview changes with <code>--dry-run</code>. Use{" "}
            <code>--no-binary</code> to skip installing the shared executable.
          </p>
        </div>
      </section>
      <footer className="folio-footer">
        <a href="#top" className="folio-wordmark">
          DevRites
        </a>
        <p>Small changes. Safer software.</p>
        <nav aria-label="Footer">
          <a href="/docs/">Docs</a>
          <a href={REPO}>GitHub</a>
          <a href={`${REPO}/releases`}>Releases</a>
          <a href={`${REPO}/blob/main/LICENSE`}>MIT license</a>
        </nav>
        <small>© 2026 Viktors Baikers</small>
      </footer>
    </>
  );
}

export default function RegisterSections() {
  return (
    <div className="register-continuation">
      <Workflow />
      <Roles />
      <Integrations />
      <SpecExample />
      <ReviewFlow />
      <ShipProcess />
      <Questions />
      <StartAndFooter />
    </div>
  );
}

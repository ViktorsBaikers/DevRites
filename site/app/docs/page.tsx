import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Reveal, CopyButton } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { PipelineDiagram } from "@/components/docs/Diagrams";
import { INSTALL_CMD } from "@/lib/site";

export const metadata: Metadata = {
  title: "Overview · DevRites docs",
  description:
    "Learn how Claude Code and Codex use the same DevRites workflow, Go control plane, and project-local feature files.",
  alternates: { canonical: "/docs/" },
};

const TREE = `.devrites/
  ACTIVE                      which feature is active
  AFK                         presence = AFK mode (max_slices / notify)
  principles.md               prescriptive project invariants
  conventions.md             observed project idioms
  learnings.md                recurring lessons and dismissed findings
  specs/<capability>/spec.md  living proven behavior
  work/<slug>/
    README.md  brief.md  spec.md  strategy.md
    architecture.md  flows.md  plan.md  tasks.md  traceability.md
    state.md  decisions.md  assumptions.md  questions.md  drift.md
    eng-review.md  test-plan.md  evidence.md  touched-files.md
    references.md  references/  design-brief.md  browser-evidence.md
    polish-report.md  review.md  seal.md  ship.md  handoff.md`;

const NEXT = [
  { href: "/docs/getting-started/", title: "Getting started", body: "Install, set up the optional tooling, and ship your first feature." },
  { href: "/docs/concepts/", title: "Concepts", body: "The mental model: slices, gates, evidence, drift, and the on-disk workspace." },
  { href: "/docs/usage/", title: "Usage & examples", body: "Worked workflows: the build loop, drift, HITL gates, and AFK runs." },
];

export default function DocsOverview() {
  return (
    <>
      <DocsHeader
        crumb="overview"
        title="How DevRites works"
        lead="DevRites gives Claude Code and Codex the same process for planning a feature, building it in slices, recording checks, reviewing the diff, and asking a human to approve the release."
      />

      <Reveal>
        <h2 id="lifecycle" className="scroll-mt-28 text-2xl font-bold">The lifecycle at a glance</h2>
      </Reveal>
      <Reveal delay={0.05}>
        <p className="mt-3 text-ink-muted leading-relaxed">
          Nine core rites take a feature from spec to ship. Temper adds an optional strategy review.
          Converge repairs resumed or adopted work by adding only the slices that are still missing.
        </p>
      </Reveal>
      <div className="mt-5 mb-14">
        <PipelineDiagram />
      </div>

      <Reveal>
        <h2 id="quick-start" className="scroll-mt-28 text-2xl font-bold">Quick start</h2>
      </Reveal>
      <Reveal delay={0.05}>
        <p className="mt-3 text-ink-muted leading-relaxed">
          Install into a project, then start a feature. <code className="k k--accent">/rite-spec</code>{" "}
          investigates the codebase, writes the spec, and creates the workspace for you.
        </p>
      </Reveal>
      <Reveal delay={0.1}>
        <div className="mt-5 overflow-hidden rounded-tile border border-line bg-bg-deep/60">
          <div className="flex items-center justify-between gap-3 border-b border-line px-4 py-2.5">
            <span className="mono text-xs text-ink-faint">install into the current project</span>
            <CopyButton text={INSTALL_CMD} />
          </div>
          <pre className="mono overflow-x-auto px-4 py-3.5 text-sm leading-relaxed text-ink">
            <span className="text-accent">npx</span> devrites@latest{"\n"}
            <span className="text-ink-faint"># then, in Claude Code (use $rite-spec in Codex):</span>
            {"\n"}
            <span className="text-go">/rite-spec</span> &quot;add refresh-token rotation&quot;
          </pre>
        </div>
      </Reveal>
      <Reveal delay={0.12}>
        <p className="mt-4 text-ink-muted leading-relaxed">
          Each phase has a menu form (<code className="k">/rite &lt;verb&gt;</code>) and a
          direct shortcut (<code className="k">/rite-&lt;verb&gt;</code>). They run the same skill. Run{" "}
          <code className="k">/rite</code> for the menu, or <code className="k">/rite-status</code> to
          see where the active feature stands.
        </p>
      </Reveal>

      <Reveal>
        <h2 id="workspace" className="mt-14 scroll-mt-28 text-2xl font-bold">The workspace</h2>
      </Reveal>
      <Reveal delay={0.05}>
        <p className="mt-3 text-ink-muted leading-relaxed">
          Each feature gets its own directory under <code className="k">.devrites/work/&lt;slug&gt;/</code>.
          When the context window fills and you <code className="k">/clear</code>, the next agent reads
          these files and resumes from the recorded state.
        </p>
      </Reveal>
      <Reveal delay={0.1}>
        <pre className="mono mt-5 overflow-x-auto rounded-tile border border-line bg-bg-deep/60 p-4 text-[0.82rem] leading-relaxed text-ink-muted">
          {TREE}
        </pre>
      </Reveal>
      <Reveal delay={0.12}>
        <p className="mt-4 text-ink-muted leading-relaxed">
          <code className="k">spec.md</code> holds the contract. <code className="k">decisions.md</code>{" "}
          and <code className="k">assumptions.md</code> hold the reasoning.{" "}
          <code className="k">evidence.md</code> holds the proof. <code className="k">drift.md</code>{" "}
          records where the plan was wrong and why. Recovery uses these project files instead of a chat summary.
        </p>
      </Reveal>

      <Reveal>
        <h2 id="next" className="mt-14 scroll-mt-28 text-2xl font-bold">Where to go next</h2>
      </Reveal>
      <div className="mt-5 grid gap-3 sm:grid-cols-3">
        {NEXT.map((n, i) => (
          <Reveal key={n.href} delay={0.05 + i * 0.05}>
            <Link href={n.href} className="group block h-full">
              <div className="tile bento-tile flex h-full flex-col gap-2 p-5">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-ink">{n.title}</h3>
                  <ArrowRight className="size-4 text-accent transition-transform duration-200 group-hover:translate-x-1" />
                </div>
                <p className="text-sm leading-relaxed text-ink-muted">{n.body}</p>
              </div>
            </Link>
          </Reveal>
        ))}
      </div>
    </>
  );
}

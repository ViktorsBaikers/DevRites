import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Reveal, CopyButton } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { INSTALL_CMD } from "@/lib/site";

export const metadata: Metadata = {
  title: "Overview · DevRites docs",
  description:
    "DevRites is a project-local pack of Claude Code skills that runs a disciplined senior-engineer workflow with state on disk.",
  alternates: { canonical: "/docs/" },
};

const TREE = `.devrites/
  ACTIVE                      which feature is active
  AFK                         presence = AFK mode (max_slices / notify)
  work/<slug>/
    spec.md  strategy.md  plan.md  tasks.md  state.md
    eng-review.md  test-plan.md  evidence.md  drift.md
    decisions.md  assumptions.md  questions.md
    references/  browser-evidence.md  touched-files.md
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
        title="Documentation"
        lead="DevRites is a project-local pack of Claude Code skills that runs a disciplined senior-engineer workflow: spec and temper the idea, plan and vet it, build one verified slice, prove it, polish, review, seal, and ship. Every phase reads the last one's files and writes its own, so your state lives on disk instead of in a chat window."
      />

      <Reveal>
        <h2 className="text-2xl font-bold">Quick start</h2>
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
            <span className="text-ink-faint"># then, inside Claude Code:</span>
            {"\n"}
            <span className="text-go">/rite-spec</span> &quot;add refresh-token rotation&quot;
          </pre>
        </div>
      </Reveal>
      <Reveal delay={0.12}>
        <p className="mt-4 text-ink-muted leading-relaxed">
          From there each phase has a menu form (<code className="k">/rite &lt;verb&gt;</code>) and a
          direct shortcut (<code className="k">/rite-&lt;verb&gt;</code>). They run the same skill. Run{" "}
          <code className="k">/rite</code> for the menu, or <code className="k">/rite-status</code> to
          see where the active feature stands.
        </p>
      </Reveal>

      <Reveal>
        <h2 className="mt-14 text-2xl font-bold">The workspace</h2>
      </Reveal>
      <Reveal delay={0.05}>
        <p className="mt-3 text-ink-muted leading-relaxed">
          Every feature gets its own directory under <code className="k">.devrites/work/&lt;slug&gt;/</code>.
          When the context window fills and you <code className="k">/clear</code>, the next agent reads
          these files and resumes exactly where the last one stopped.
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
          records where the plan was wrong and why. None of it depends on a chat summary you can lose.
        </p>
      </Reveal>

      <Reveal>
        <h2 className="mt-14 text-2xl font-bold">Where to go next</h2>
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

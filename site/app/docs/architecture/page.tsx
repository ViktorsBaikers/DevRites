import type { Metadata } from "next";
import { Reveal } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { LAYERS, RATIONALE } from "@/lib/docs";
import { REPO } from "@/lib/site";

export const metadata: Metadata = {
  title: "Architecture",
  description: "The layers, the naming convention, and the design rationale behind DevRites.",
  alternates: { canonical: "/docs/architecture/" },
};

export default function Architecture() {
  return (
    <>
      <DocsHeader
        crumb="architecture"
        title="Architecture"
        lead="DevRites is a distributed but coordinated set of project-local Claude Code skills. Each phase is its own skill; none is a monolith. The pieces share one thing: state on disk under .devrites/."
      />

      <Reveal>
        <h2 className="text-2xl font-bold">Layers</h2>
      </Reveal>
      <div className="mt-6 space-y-3">
        {LAYERS.map((l, i) => (
          <Reveal key={l.name} delay={Math.min(i * 0.04, 0.25)}>
            <div className="tile flex flex-col gap-1.5 p-5 sm:flex-row sm:items-baseline sm:gap-5">
              <div className="flex w-full shrink-0 items-center justify-between sm:w-52 sm:flex-col sm:items-start sm:gap-1">
                <h3 className="font-bold text-ink">{l.name}</h3>
                <code className="mono rounded border border-line bg-surface-2/50 px-1.5 py-0.5 text-[0.68rem] text-accent">
                  {l.tag}
                </code>
              </div>
              <p className="text-[0.9rem] leading-relaxed text-ink-muted">{l.body}</p>
            </div>
          </Reveal>
        ))}
      </div>

      <Reveal>
        <h2 className="mt-14 text-2xl font-bold">Design rationale</h2>
      </Reveal>
      <div className="mt-6 space-y-3">
        {RATIONALE.map((r, i) => (
          <Reveal key={r.q} delay={Math.min(i * 0.04, 0.25)}>
            <div className="tile p-6">
              <h3 className="font-bold text-ink">{r.q}</h3>
              <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">{r.a}</p>
            </div>
          </Reveal>
        ))}
      </div>

      <Reveal>
        <div className="tile--lit mt-8 rounded-tile p-6">
          <h3 className="font-bold text-ink">Read the source</h3>
          <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">
            Every skill, agent, and rule ships in the repo as plain Markdown. See the{" "}
            <a href={REPO} rel="noopener" className="text-accent underline-offset-2 hover:underline">
              GitHub repository
            </a>{" "}
            for the full set.
          </p>
        </div>
      </Reveal>
    </>
  );
}

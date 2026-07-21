import type { Metadata } from "next";
import { Reveal } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { StackDiagram } from "@/components/docs/Diagrams";
import { LAYERS, RATIONALE, SAFETY, RULES_ON_DEMAND } from "@/lib/docs";
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
        lead="DevRites separates engineering judgment from workflow bookkeeping. Focused Claude and Codex skills make engineering decisions, while one stdlib-only Go engine manages workspace state, gates, hooks, derived data, migration, and installation."
      />

      <div className="mb-12">
        <StackDiagram />
      </div>

      <Reveal>
        <h2 id="layers" className="scroll-mt-28 text-2xl font-bold">Layers</h2>
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
        <h2 id="rationale" className="mt-14 scroll-mt-28 text-2xl font-bold">Design rationale</h2>
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
        <h2 id="rules" className="mt-14 scroll-mt-28 text-2xl font-bold">Engineering rules</h2>
      </Reveal>
      <Reveal delay={0.05}>
        <p className="mt-3 text-ink-muted leading-relaxed">
          DevRites includes stack-agnostic standards inside{" "}
          <code className="k">devrites-lib/reference/standards/</code>, mirrored for both hosts. Workspace
          phases load <code className="k">core.md</code> first; specialized standards and checklists load
          only when needed. Prescriptive <code className="k">.devrites/principles.md</code> outranks
          shipped guidance, while observed conventions and mined learnings remain separate inputs.
        </p>
      </Reveal>
      <Reveal delay={0.1}>
        <div className="mt-5 rounded-tile border border-line p-5">
          <div className="flex items-center gap-2">
            <code className="mono rounded border border-accent/40 bg-accent/10 px-1.5 py-0.5 text-[0.7rem] text-accent">
              core.md
            </code>
            <span className="text-sm text-ink-muted">always on</span>
          </div>
          <div className="mt-4 flex flex-wrap gap-1.5">
            {RULES_ON_DEMAND.map((r) => (
              <span key={r} className="mono rounded-md border border-line bg-surface-2/50 px-2 py-1 text-[0.7rem] text-ink-muted">
                {r}
              </span>
            ))}
          </div>
        </div>
      </Reveal>

      <Reveal>
        <h2 id="safety" className="mt-14 scroll-mt-28 text-2xl font-bold">Safety &amp; scope</h2>
      </Reveal>
      <div className="mt-5 space-y-3">
        {SAFETY.map((s, i) => (
          <Reveal key={s.h} delay={Math.min(i * 0.04, 0.2)}>
            <div className="tile flex gap-3 p-5">
              <span className="mono mt-0.5 text-accent">→</span>
              <p className="text-[0.9rem] leading-relaxed text-ink-muted">
                <b className="text-ink">{s.h}.</b> {s.b}
              </p>
            </div>
          </Reveal>
        ))}
      </div>

      <Reveal>
        <h2 id="security" className="mt-14 scroll-mt-28 text-2xl font-bold">Security model</h2>
      </Reveal>
      <Reveal delay={0.05}>
        <p className="mt-3 text-ink-muted leading-relaxed">
          DevRites uses auditable Markdown and a CGO-free Go control plane. Workspace state and gate
          commands make no model or network calls; explicit install, update, and source-cache I/O is
          isolated behind named engine boundaries. Host artifacts remain project-local, hook guards
          fail open when the binary is unavailable, and strict profiles can enforce source/reviewer
          boundaries. Irreversible git actions still require interactive <code className="k">type-GO</code>.
          Full policy and private reporting:{" "}
          <a href={`${REPO}/blob/main/SECURITY.md`} rel="noopener" className="text-accent underline-offset-2 hover:underline">
            SECURITY.md
          </a>
          .
        </p>
      </Reveal>

      <Reveal>
        <div className="tile--lit mt-10 rounded-tile p-6">
          <h3 className="font-bold text-ink">Read the source</h3>
          <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">
            The canonical skills, agents, and standards are plain Markdown. The engine and generated
            host adapters are in the same repository. Browse the{" "}
            <a href={REPO} rel="noopener" className="text-accent underline-offset-2 hover:underline">
              GitHub repository
            </a>{" "}
            to inspect them.
          </p>
        </div>
      </Reveal>
    </>
  );
}

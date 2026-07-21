import type { Metadata } from "next";
import { Reveal } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Panel, Row } from "@/components/docs/DocsBits";
import { PhaseLoop, ProofLadder } from "@/components/docs/Diagrams";
import { CONCEPTS, WORKSPACE_FILES } from "@/lib/docs";

export const metadata: Metadata = {
  title: "Concepts",
  description: "How DevRites uses slices, gates, evidence, drift handling, forge builds, and an on-disk workspace.",
  alternates: { canonical: "/docs/concepts/" },
};

export default function Concepts() {
  return (
    <>
      <DocsHeader
        crumb="concepts"
        title="Concepts"
        lead="These concepts explain how DevRites divides work, records state, checks evidence, and recovers when a plan is wrong."
      />

      <H2 id="big-idea" first>The big idea</H2>
      <P>
        DevRites separates engineering judgment from workflow bookkeeping. Project-local Claude and Codex
        skills make engineering decisions. A stdlib-only Go engine manages state, gates, hooks, migration,
        and derived data in the Markdown files under <code className="k">.devrites/</code>. A new context,
        another host, CI, or a human can resume from those same files.
      </P>

      <div className="mt-6">
        <PhaseLoop />
      </div>

      <H2 id="core">Core concepts</H2>
      <div className="mt-6 grid gap-3 sm:grid-cols-2">
        {CONCEPTS.map((c, i) => (
          <Reveal key={c.term} delay={Math.min(i * 0.04, 0.25)}>
            <div className="tile flex h-full flex-col gap-2 p-5">
              <h3 className="font-bold text-accent">{c.term}</h3>
              <p className="text-[0.9rem] leading-relaxed text-ink-muted">{c.body}</p>
            </div>
          </Reveal>
        ))}
      </div>

      <H2 id="proof">The evidence ladder</H2>
      <P>
        DevRites ranks evidence from strongest to weakest. It uses the strongest check available in your
        project and records which level it reached.
      </P>
      <div className="mt-5">
        <ProofLadder />
      </div>

      <H2 id="workspace">The workspace, file by file</H2>
      <P>
        Each feature gets its own directory under <code className="k">.devrites/work/&lt;slug&gt;/</code>.
        A phase starts with the compact workspace map, then loads only the files it needs. Resuming work
        depends on these files rather than the previous chat.
      </P>
      <Panel>
        {WORKSPACE_FILES.map((f) => (
          <Row key={f.file} left={f.file} tag={f.by} body={f.holds} />
        ))}
      </Panel>
      <P>
        When <code className="k">/rite-ship</code> closes a feature it archives the whole workspace to{" "}
        <code className="k">.devrites/archive/&lt;slug&gt;/</code> and preserves every file. It then
        clears <code className="k">.devrites/ACTIVE</code>. One feature is active at a time; start or switch
        with <code className="k">/rite-spec &lt;other&gt;</code>.
      </P>
    </>
  );
}

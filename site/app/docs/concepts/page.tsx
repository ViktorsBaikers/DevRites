import type { Metadata } from "next";
import { Reveal } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Panel, Row } from "@/components/docs/DocsBits";
import { PhaseLoop, ProofLadder } from "@/components/docs/Diagrams";
import { CONCEPTS, WORKSPACE_FILES } from "@/lib/docs";

export const metadata: Metadata = {
  title: "Concepts",
  description: "The mental model behind DevRites: slices, gates, the evidence ladder, the Spec Drift Guard, forge, and the on-disk workspace.",
  alternates: { canonical: "/docs/concepts/" },
};

export default function Concepts() {
  return (
    <>
      <DocsHeader
        crumb="concepts"
        title="Concepts"
        lead="A handful of ideas explain everything DevRites does. Learn these once and the commands read themselves."
      />

      <H2 id="big-idea">The big idea</H2>
      <P>
        DevRites is a set of small, project-local Claude Code skills, one per phase, that share a single
        source of truth: Markdown files on disk under <code className="k">.devrites/</code>. Because the
        state lives in files instead of the chat, any phase can resume cold after you clear the context,
        and the same files can be driven by Claude Code, another agent, a CI job, or a human.
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
        Every claim is backed by proof, ranked from strongest to weakest. DevRites climbs as high as your
        tooling allows and records exactly which rung it reached.
      </P>
      <div className="mt-5">
        <ProofLadder />
      </div>

      <H2 id="workspace">The workspace, file by file</H2>
      <P>
        Every feature gets its own directory under <code className="k">.devrites/work/&lt;slug&gt;/</code>.
        Each phase reads the previous phase&rsquo;s files and writes its own. Nothing here depends on a
        chat summary you can lose.
      </P>
      <Panel>
        {WORKSPACE_FILES.map((f) => (
          <Row key={f.file} left={f.file} tag={f.by} body={f.holds} />
        ))}
      </Panel>
      <P>
        When <code className="k">/rite-ship</code> closes a feature it archives the whole workspace to{" "}
        <code className="k">.devrites/archive/&lt;slug&gt;/</code> (every file preserved, never deleted) and
        clears <code className="k">.devrites/ACTIVE</code>. One feature is active at a time; start or switch
        with <code className="k">/rite-spec &lt;other&gt;</code>.
      </P>
    </>
  );
}

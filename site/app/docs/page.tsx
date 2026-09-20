import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { CopyButton } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Callout, Code } from "@/components/docs/DocsBits";
import { INSTALL_CMD } from "@/lib/site";

export const metadata: Metadata = {
  title: "Overview",
  description: "Start with DevRites: install the workflow, choose a small change or a full feature, and keep the evidence in your repository.",
  alternates: { canonical: "/docs/" },
};

export default function DocsOverview() {
  return (
    <>
      <DocsHeader crumb="overview" title="From request to reviewed release."
        lead="A field guide to DevRites. Give your coding agent a clear contract, build in small steps, and keep the proof with the code." />
      <H2 id="start" first>Start in your project</H2>
      <P>DevRites works inside Claude Code, Codex, omp, pi, and Devin CLI. It adds a workflow to the host you already use. You keep your repository, tools, and way of writing code.</P>
      <div className="docs-start-command"><div><span>Terminal · project root · Node.js 18+</span><CopyButton text={INSTALL_CMD} /></div><pre><code>{INSTALL_CMD}</code></pre></div>
      <P>Then open your coding host in that project and choose the route that fits the change.</P>
      <div className="docs-route-list">
        <Link href="/docs/getting-started/#first"><strong>Build your first feature</strong><span>Write the contract, build a slice, and inspect its proof.</span><ArrowRight size={20} aria-hidden /></Link>
        <Link href="/docs/getting-started/#quick"><strong>Make a small, reversible fix</strong><span>Use Quick for a focused change that does not need a full feature workspace.</span><ArrowRight size={20} aria-hidden /></Link>
        <Link href="/docs/usage/#checking-in"><strong>Pick up work already in progress</strong><span>Read the current state and follow the next recorded action.</span><ArrowRight size={20} aria-hidden /></Link>
      </div>
      <H2 id="example">What a feature looks like</H2>
      <P>Suppose you want to export a filtered list as CSV. Instead of asking the agent to write the entire feature at once, agree on the behavior and the evidence that will prove it.</P>
      <div className="docs-process">
        <div><h3>Agree on the result</h3><p>Which columns belong in the file? Do filters apply? What happens when there are no rows? The answers become acceptance criteria.</p></div>
        <div><h3>Build a working slice</h3><p>The agent implements one verifiable part and records the changed files and test results. The next slice starts from that record.</p></div>
        <div><h3>Check before release</h3><p>Proof and independent review cover the current code. You see the exact release plan before authorizing Git actions.</p></div>
      </div>
      <Callout title="The agent works. You retain the release decision.">Seal records GO or NO-GO without changing Git. Ship shows the exact actions it proposes and waits for a fresh literal GO. An unattended run does not remove that boundary.</Callout>
      <H2 id="workspace">The record survives the conversation</H2>
      <P>A full feature lives in <code className="k">.devrites/work/&lt;slug&gt;/</code>. Another session or supported host can read what was agreed, what changed, and what remains.</P>
      <Code>{`.devrites/work/csv-export/
  spec.md          what the feature must do
  plan.md          how it will be built
  tasks.md         the individual slices
  state.md         where work stopped; what comes next
  evidence.md      commands, results, and proof
  review.md        independent findings
  seal.md          the release-readiness decision`}</Code>
      <P>This is a reading map, not the complete file inventory. The <Link href="/docs/concepts/#workspace">workspace reference</Link> explains the other artifacts and who writes them.</P>
      <H2 id="commands">Two kinds of command</H2>
      <P>Use <code className="k">/rite-spec</code> in Claude Code or <code className="k">$rite-spec</code> in Codex to ask the agent to do engineering work. Use <code className="k">devrites-engine</code> in a terminal to inspect or validate the recorded state. The engine does not write your feature for you.</P>
      <P>The <Link href="/docs/command-map/">command map</Link> covers the agent workflow. The <Link href="/docs/cli-mcp/">engine reference</Link> covers deterministic checks and automation.</P>
    </>
  );
}

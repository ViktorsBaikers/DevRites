import type { Metadata } from "next";
import { CopyButton } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Panel, Row, Code, Callout } from "@/components/docs/DocsBits";
import { INSTALL_FLAGS, SETUP_TOOLS } from "@/lib/docs";
import { CURL_CMD, CURL_PIN_CMD, CURL_UNINSTALL_CMD, CURL_UPDATE_CMD, INSTALL_CMD } from "@/lib/site";

export const metadata: Metadata = {
  title: "Getting started",
  description: "Install DevRites into a project, set up the optional tooling, and ship your first feature.",
  alternates: { canonical: "/docs/getting-started/" },
};

export default function GettingStarted() {
  return (
    <>
      <DocsHeader
        crumb="getting started"
        title="Getting started"
        lead="Install into your repository, open your coding host, and make one verified change. Start small; the full workflow is there when the feature needs it."
      />

      <H2 id="install" first>Install</H2>
      <P>
        Use npx (recommended, Node 18+) or the curl one-liner. Both delegate to the engine-owned
        installer and ship the generated skills, agents, standards, adapters, and aliases into the project.
      </P>
      <Panel>
        <div className="flex items-center justify-between gap-3 border-b border-line bg-surface-2/40 px-4 py-2.5">
          <span className="mono text-xs text-ink-faint">npx · recommended · Node 18+</span>
          <CopyButton text={INSTALL_CMD} />
        </div>
        <div className="mono space-y-1 overflow-x-auto p-4 text-sm">
          <div><span className="text-go">npx</span> devrites@latest</div>
          <div><span className="text-go">npx</span> devrites@latest --target /path/to/project</div>
          <div><span className="text-go">npx</span> devrites@latest --dry-run</div>
          <div className="text-ink-faint">npx devrites@latest update</div>
          <div className="text-ink-faint">npx devrites@latest uninstall</div>
        </div>
      </Panel>
      <Panel>
        <div className="flex items-center justify-between gap-3 border-b border-line bg-surface-2/40 px-4 py-2.5">
          <span className="mono text-xs text-ink-faint">curl · no Node required</span>
          <CopyButton text={CURL_CMD} />
        </div>
        <pre className="mono overflow-x-auto p-4 text-sm leading-relaxed text-ink">
          {CURL_CMD}{"\n"}
          <span className="text-ink-faint"># pin a release</span>{"\n"}
          {CURL_PIN_CMD}{"\n"}
          <span className="text-ink-faint"># update · uninstall</span>{"\n"}
          {CURL_UPDATE_CMD}{"\n"}
          {CURL_UNINSTALL_CMD}
        </pre>
      </Panel>
      <P>
        The installer records each managed file in <code className="k">.claude/devrites.manifest</code>.
        Uninstall removes those files and any empty directories it leaves behind. It preserves your
        feature data in <code className="k">.devrites/work/</code>.
      </P>

      <H2 id="quick">Choose a small first change</H2>
      <P>For a typo, a focused display fix, or another small reversible change, use Quick in your coding host. It keeps the contract and proof compact. If the change becomes risky or needs several slices, DevRites routes it to Spec.</P>
      <Code>{`# In Claude Code, omp, pi, or Devin CLI
/rite-quick fix the CSV column heading

# In Codex
$rite-quick fix the CSV column heading`}</Code>
      <Callout title="What success looks like">Inspect the diff and the recorded check. You should be able to explain what changed and why the check proves it. A claim that the task is done is not a substitute for that evidence.</Callout>

      <H2 id="flags">Install options</H2>
      <P>The npx and curl paths share the same engine-owned install semantics.</P>
      <Panel>
        {INSTALL_FLAGS.map((f) => (
          <Row key={f.flag} left={f.flag} body={f.effect} />
        ))}
      </Panel>

      <H2 id="setup">Recommended setup</H2>
      <P>
        DevRites works without extra tools. If codegraph, graphify, or Playwright MCP is available, the
        relevant phases use it. Missing tools do not block the workflow.
      </P>
      <Panel>
        {SETUP_TOOLS.map((t) => (
          <Row key={t.tool} left={t.tool} body={t.gives} />
        ))}
      </Panel>

      <H2 id="first">Your first feature</H2>
      <P>
        Install into a project, then start a feature. <code className="k k--accent">/rite-spec</code>{" "}
        investigates the codebase, asks one question at a time when details are missing, writes the spec,
        and creates the workspace.
      </P>
      <Code>
        <span className="text-go">/rite-spec</span> &quot;add CSV export for admins&quot;{"  "}
        <span className="text-ink-faint"># investigate → spec.md</span>{"\n"}
        <span className="text-go">/rite-clarify</span>{"               "}
        <span className="text-ink-faint"># close gaps; no questions if already clear</span>{"\n"}
        <span className="text-go">/rite-define</span>{"                "}
        <span className="text-ink-faint"># spec → plan + vertical slices</span>{"\n"}
        <span className="text-go">/rite-vet</span>{"                   "}
        <span className="text-ink-faint"># mandatory stakes-scaled engineering review</span>{"\n"}
        <span className="text-go">/rite-build</span>{"                 "}
        <span className="text-ink-faint"># one slice, then stop with evidence</span>{"\n"}
        <span className="text-go">/rite-build</span>{"                 "}
        <span className="text-ink-faint"># next slice, you decide when</span>{"\n"}
        <span className="text-go">/rite-prove</span>{"                 "}
        <span className="text-ink-faint"># full tests + browser proof</span>{"\n"}
        <span className="text-go">/rite-polish</span>{"                "}
        <span className="text-ink-faint"># clean up, then refresh affected proof</span>{"\n"}
        <span className="text-go">/rite-review</span>{"                "}
        <span className="text-ink-faint"># parallel fresh-context review</span>{"\n"}
        <span className="text-go">/rite-seal</span>{"                  "}
        <span className="text-ink-faint"># GO / NO-GO verdict (no git)</span>{"\n"}
        <span className="text-go">/rite-ship</span>{"                  "}
        <span className="text-ink-faint"># inspect the exact plan, then authorize with GO</span>
      </Code>
      <P>
        Run these one at a time, following the recorded next step. Temper is an optional strategic review before Define. Each phase has a menu form (<code className="k">/rite &lt;verb&gt;</code>) and a direct shortcut
        (<code className="k">/rite-&lt;verb&gt;</code>); both run the same skill. Run{" "}
        <code className="k">/rite</code> to discover the menu, or <code className="k">/rite-status</code>{" "}
        to see where the active feature stands. To run the whole sequence unattended, use{" "}
        <code className="k k--accent">/rite-autocomplete</code>.
      </P>

      <Callout title="The command syntax depends on the host">
        Claude accepts <code className="k">/rite build</code> or <code className="k">/rite-build</code>.
        Codex accepts <code className="k">$rite build</code> or <code className="k">$rite-build</code>.
        omp, pi, and Devin CLI use the slash forms; pi also accepts <code className="k">/skill:rite-build</code>.
        These are prompts in the coding host, not shell commands. Run only installation and <code className="k">devrites-engine</code> commands in your terminal.
      </Callout>
      <Callout title="If the commands do not appear">Confirm that installation targeted this repository, then reopen your host in the same directory. Run <code className="k">/rite-doctor</code> (or <code className="k">$rite-doctor</code> in Codex) to inspect the installation. If a feature is already active, use Status before starting another.</Callout>
    </>
  );
}

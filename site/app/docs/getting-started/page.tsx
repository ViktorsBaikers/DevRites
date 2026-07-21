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
        lead="DevRites installs generated Claude and Codex files in your project, plus an optional shared engine binary. Start a feature in either host and both will use the same workspace and gates."
      />

      <H2 id="install" first>Install</H2>
      <P>
        Use npx (recommended, Node 18+) or the curl one-liner. Both delegate to the engine-owned
        installer and ship Claude/Codex skills, agents, standards, hooks, and aliases into the project.
      </P>
      <Panel>
        <div className="flex items-center justify-between gap-3 border-b border-line bg-surface-2/40 px-4 py-2.5">
          <span className="mono text-xs text-ink-faint">npx · recommended · Node 18+</span>
          <CopyButton text={INSTALL_CMD} />
        </div>
        <div className="mono space-y-1 p-4 text-sm">
          <div><span className="text-go">npx</span> devrites@latest</div>
          <div><span className="text-go">npx</span> devrites@latest --target /path/to/project</div>
          <div><span className="text-go">npx</span> devrites@latest --dry-run</div>
          <div className="text-ink-faint">npx devrites@latest update · uninstall</div>
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

      <H2 id="flags">Common flags</H2>
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
        <span className="text-go">/rite-review</span>{"                "}
        <span className="text-ink-faint"># parallel fresh-context review</span>{"\n"}
        <span className="text-go">/rite-seal</span>{"                  "}
        <span className="text-ink-faint"># GO / NO-GO verdict (no git)</span>{"\n"}
        <span className="text-go">/rite-ship</span>{"                  "}
        <span className="text-ink-faint"># type-GO → commit · push · tag</span>
      </Code>
      <P>
        Each phase has a menu form (<code className="k">/rite &lt;verb&gt;</code>) and a direct shortcut
        (<code className="k">/rite-&lt;verb&gt;</code>); both run the same skill. Run{" "}
        <code className="k">/rite</code> to discover the menu, or <code className="k">/rite-status</code>{" "}
        to see where the active feature stands. To run the whole sequence unattended, use{" "}
        <code className="k k--accent">/rite-autocomplete</code>.
      </P>

      <Callout title="The command syntax depends on the host">
        Claude accepts <code className="k">/rite build</code> or <code className="k">/rite-build</code>.
        Codex accepts <code className="k">$rite build</code> or <code className="k">$rite-build</code>.
        Both forms load generated artifacts backed by the same engine commands and workspace.
      </Callout>
    </>
  );
}

import type { Metadata } from "next";
import { CopyButton } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Panel, Row, Code, Callout } from "@/components/docs/DocsBits";
import { INSTALL_FLAGS, SETUP_TOOLS } from "@/lib/docs";
import { INSTALL_CMD, CURL_CMD } from "@/lib/site";

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
        lead="DevRites installs into a project, never into ~/.claude. Install it, point it at a feature, and it walks the rest. Here is the whole path from zero to your first shipped slice."
      />

      <H2 id="install">Install</H2>
      <P>
        Use npx (recommended, Node 18+) or the curl one-liner. Both run the same installer and ship the
        skills, agents, rules, and aliases into the current project.
      </P>
      <Panel>
        <div className="flex items-center justify-between gap-3 border-b border-line bg-surface-2/40 px-4 py-2.5">
          <span className="mono text-xs text-ink-faint">npx · pinned and offline</span>
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
          <span className="text-accent">curl</span> -fsSL https://devrites.com | bash{"\n"}
          <span className="text-ink-faint"># pin a release</span>{"\n"}
          <span className="text-accent">curl</span> -fsSL https://devrites.com | DEVRITES_REF=v0.1.0 bash{"\n"}
          <span className="text-ink-faint"># update · remove</span>{"\n"}
          <span className="text-accent">curl</span> -fsSL https://devrites.com/update | bash{"\n"}
          <span className="text-accent">curl</span> -fsSL https://devrites.com/remove | bash
        </pre>
      </Panel>
      <P>
        Every installed file is recorded in <code className="k">.claude/devrites.manifest</code>. Uninstall
        removes exactly those and prunes empty dirs; your feature data in{" "}
        <code className="k">.devrites/work/</code> is always preserved.
      </P>

      <H2 id="flags">Common flags</H2>
      <P>The npx and curl paths accept the same flags as the bash installer.</P>
      <Panel>
        {INSTALL_FLAGS.map((f) => (
          <Row key={f.flag} left={f.flag} body={f.effect} />
        ))}
      </Panel>

      <H2 id="setup">Recommended setup</H2>
      <P>
        DevRites runs with zero extra tooling, but three tools make it meaningfully sharper. It detects
        each one and degrades gracefully when it is absent, so none are required.
      </P>
      <Panel>
        {SETUP_TOOLS.map((t) => (
          <Row key={t.tool} left={t.tool} body={t.gives} />
        ))}
      </Panel>

      <H2 id="first">Your first feature</H2>
      <P>
        Install into a project, then start a feature. <code className="k k--accent">/rite-spec</code>{" "}
        investigates the codebase, asks one question at a time until the gaps are closed, writes the spec,
        and creates the workspace.
      </P>
      <Code>
        <span className="text-go">/rite-spec</span> &quot;add CSV export for admins&quot;{"  "}
        <span className="text-ink-faint"># investigate → spec.md</span>{"\n"}
        <span className="text-go">/rite-define</span>{"                "}
        <span className="text-ink-faint"># spec → plan + vertical slices</span>{"\n"}
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

      <Callout title="Add your own shortcuts">
        Pin one-word aliases to any skill with <code className="k">scripts/pin.sh add b rite-build</code>{" "}
        (so <code className="k">/b</code> runs <code className="k">/rite-build</code>). Aliases are
        manifest-tracked, so uninstall cleans them up.
      </Callout>
    </>
  );
}

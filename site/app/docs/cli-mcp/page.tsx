import type { Metadata } from "next";
import { Reveal } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Panel, Code, Callout } from "@/components/docs/DocsBits";
import { CLI_COMMANDS } from "@/lib/docs";

export const metadata: Metadata = {
  title: "Engine CLI",
  description: "Use the portable devrites-engine CLI to work with DevRites state, gates, hooks, and structured reports.",
  alternates: { canonical: "/docs/cli-mcp/" },
};

export default function CliMcp() {
  return (
    <>
      <DocsHeader
        crumb="engine cli"
        title="The devrites-engine CLI"
        lead="DevRites uses one CGO-free Go binary for workflow bookkeeping. Claude and Codex call it through generated hooks and skills. CI, scripts, and humans can call it directly against the same .devrites/ workspace."
      />

      <H2 id="cli" first>Common commands</H2>
      <P>
        Install DevRites normally, then run the engine from the project root. The npm{" "}
        <code className="k">devrites</code> shim owns install, update, and uninstall and proxies engine
        subcommands when the binary is available. <code className="k">devrites-engine help</code> is the
        full current inventory.
      </P>
      <Panel>
        {CLI_COMMANDS.map((c) => (
          <div key={c.cmd} className="flex flex-col gap-1.5 border-b border-line p-4 last:border-0 sm:flex-row sm:items-baseline sm:gap-4">
            <code className="mono w-full shrink-0 text-sm text-accent sm:w-72">{c.cmd}</code>
            <p className="flex-1 text-[0.9rem] leading-relaxed text-ink-muted">{c.note}</p>
            {c.exit && <span className="mono shrink-0 text-[0.78rem] text-go">{c.exit}</span>}
          </div>
        ))}
      </Panel>

      <H2 id="json">Structured output</H2>
      <P>
        <code className="k">snapshot</code> emits a direct <code className="k">devrites.workspace.v1</code>{" "}
        JSON document. AFK-parsed read commands such as status, readiness, seal, spec-validate,
        evidence-fresh, preamble, coverage, analyze, doctor, and ledger accept <code className="k">--json</code>{" "}
        with a stable envelope for automation.
      </P>
      <Code>
{`devrites-engine snapshot auth-tokens
devrites-engine status auth-tokens --json
devrites-engine context show --json
devrites-engine reviewer-stats report --json`}
      </Code>

      <H2 id="hooks">Host adapters</H2>
      <P>
        Generated Claude and Codex hook wiring calls <code className="k">devrites-engine hook &lt;id&gt;</code>{" "}
        with a host adapter. Profiles select the active set: minimal handles orientation and approval,
        standard adds gates, guards, and caches, and strict enforces every observe-default boundary.
        If the binary is missing, the hook exits without blocking the host, so a teammate can still work.
      </P>

      <Callout title="Exit 3 means pause">
        A blocked gate or incompatible state-schema version exits 3 and names the problem. This means
        the workflow is waiting for a person. Resolve the gap and retry. Usage errors exit 2, and a
        passing gate exits 0.
      </Callout>
    </>
  );
}

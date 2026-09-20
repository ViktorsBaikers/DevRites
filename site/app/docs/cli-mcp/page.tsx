import type { Metadata } from "next";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Panel, Row, Code, Callout } from "@/components/docs/DocsBits";
import { CLI_COMMANDS } from "@/lib/docs";

export const metadata: Metadata = {
  title: "Engine CLI",
  description: "Use the portable devrites-engine CLI to work with DevRites state, deterministic gates, candidates, and structured reports.",
  alternates: { canonical: "/docs/cli-mcp/" },
};

export default function CliMcp() {
  return (
    <>
      <DocsHeader
        crumb="engine cli"
        title="Engine CLI"
        lead="Inspect a workspace and run deterministic checks from your terminal. The same engine serves every coding host, scripts, and CI. It manages the record; agents do the engineering work."
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
          <Row key={c.cmd} left={c.cmd} tag={c.exit} body={c.note} />
        ))}
      </Panel>

      <H2 id="json">Structured output</H2>
      <P>
        <code className="k">observe summary</code> and its <code className="k">orient</code> alias emit a
        sanitized JSON workspace summary. Context packets, metrics, command detection, and index checks
        expose their own documented machine-readable contracts.
      </P>
      <Code>
{`devrites-engine observe summary auth-tokens
devrites-engine orient auth-tokens
devrites-engine context auth-tokens --phase prove
devrites-engine detect commands --root . --json`}
      </Code>

      <H2 id="hooks">Host adapters</H2>
      <P>
        Each supported host dispatches its own native skills and agents. Those skills call the engine for
        state changes and gates. Use the installed engine's help output for the exact command inventory;
        old aliases and commands from a different release may no longer exist.
      </P>

      <Callout title="Exit 3 means pause">
        A blocked gate or incompatible state-schema version exits 3 and names the problem. Read the
        reported gap before retrying; a missing artifact may need workflow repair, while a human gate needs an answer. Usage errors exit 2, and a
        passing gate exits 0.
      </Callout>
    </>
  );
}

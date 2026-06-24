import type { Metadata } from "next";
import { Reveal } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Panel, Code, Callout } from "@/components/docs/DocsBits";
import { CLI_COMMANDS } from "@/lib/docs";

export const metadata: Metadata = {
  title: "CLI & MCP",
  description: "Drive the same deterministic DevRites gates from any tool: the portable devrites CLI and a dependency-free MCP server.",
  alternates: { canonical: "/docs/cli-mcp/" },
};

export default function CliMcp() {
  return (
    <>
      <DocsHeader
        crumb="cli & mcp"
        title="CLI & MCP"
        lead="The discipline lives in the .devrites/ files and the state scripts, not in the Claude Code harness. Two portable surfaces expose the same gates, so Cursor, Codex, a CI job, or a human can drive the workflow against the same files."
      />

      <H2 id="cli">The devrites CLI</H2>
      <P>
        Installed at <code className="k">.claude/skills/devrites-lib/scripts/devrites.sh</code>. Each
        command is a thin wrapper over a state script, so the exit code is the gate: a non-zero{" "}
        <code className="k">ready</code>, <code className="k">evidence-fresh</code>, or{" "}
        <code className="k">acceptance</code> is a hard stop you can script into any agent loop or a
        pre-merge CI step.
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

      <H2 id="mcp">The MCP server</H2>
      <P>
        <code className="k">devrites-mcp.mjs</code> is a dependency-free MCP stdio server that exposes the
        read and gate operations as MCP tools (<code className="k">devrites_ready</code>,{" "}
        <code className="k">devrites_acceptance</code>, <code className="k">devrites_orient</code>, and the
        rest). It shells out to the CLI, so it stays a thin surface over the same scripts. Register it in a
        project&rsquo;s <code className="k">.mcp.json</code>:
      </P>
      <Code>
{`{
  "mcpServers": {
    "devrites": {
      "command": "node",
      "args": ["/abs/path/to/mcp/devrites-mcp.mjs"]
    }
  }
}`}
      </Code>
      <P>
        Now any MCP client can ask &ldquo;is this feature ready to ship?&rdquo; and the server runs the
        deterministic gates against <code className="k">.devrites/</code>. That is the same verdict the
        lifecycle skills compute, available to tools that don&rsquo;t speak DevRites&rsquo; skill prose.
      </P>

      <Callout title="One verdict, everywhere">
        Spec-kit, task-master, and BMAD run across many agents; DevRites keeps the discipline in
        tool-agnostic data. Because the CLI, the MCP server, and <code className="k">/rite-seal</code> all
        call the same scripts, their verdict agrees no matter who asks.
      </Callout>
    </>
  );
}

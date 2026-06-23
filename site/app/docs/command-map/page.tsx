import type { Metadata } from "next";
import { Reveal } from "@/components/ui";
import DocsHeader from "@/components/docs/DocsHeader";
import {
  PUBLIC_COMMANDS,
  INTERNAL_SKILLS,
  REVIEW_AGENTS,
  CLI_GATES,
} from "@/lib/docs";

export const metadata: Metadata = {
  title: "Command map",
  description: "Every shipped DevRites skill and agent: what triggers it, and what it reads and writes.",
  alternates: { canonical: "/docs/command-map/" },
};

function Section({ id, title, intro, children }: { id: string; title: string; intro?: string; children: React.ReactNode }) {
  return (
    <section id={id} className="mt-14 scroll-mt-24 first:mt-0">
      <Reveal>
        <h2 className="text-2xl font-bold">{title}</h2>
      </Reveal>
      {intro && (
        <Reveal delay={0.05}>
          <p className="mt-3 text-ink-muted leading-relaxed">{intro}</p>
        </Reveal>
      )}
      <Reveal delay={0.08}>
        <div className="mt-5 overflow-hidden rounded-tile border border-line">{children}</div>
      </Reveal>
    </section>
  );
}

function Row({ left, tag, body }: { left: string; tag?: string; body: string }) {
  return (
    <div className="flex flex-col gap-1.5 border-b border-line p-4 last:border-0 sm:flex-row sm:items-baseline sm:gap-4">
      <div className="flex w-full shrink-0 items-center gap-2 sm:w-56">
        <code className="mono text-sm text-accent">{left}</code>
        {tag && (
          <span className="mono rounded border border-line px-1.5 py-0.5 text-[0.6rem] uppercase tracking-wide text-ink-faint">
            {tag}
          </span>
        )}
      </div>
      <p className="text-[0.9rem] leading-relaxed text-ink-muted">{body}</p>
    </div>
  );
}

export default function CommandMap() {
  return (
    <>
      <DocsHeader
        crumb="command map"
        title="Command map"
        lead="Every shipped skill and agent: what triggers it, and what it reads and writes. Each phase has a menu form (/rite <verb>) and a direct shortcut (/rite-<verb>); both run the same skill."
      />

      <Section id="public" title="Public commands" intro="Invoke any of these yourself. They map to the workflow one to one.">
        {PUBLIC_COMMANDS.map((c) => (
          <Row key={c.cmd} left={c.cmd} tag={c.phase} body={c.desc} />
        ))}
      </Section>

      <Section
        id="internal"
        title="Internal skills"
        intro="Model-invoked specialists, hidden from the menu. They fire automatically when a phase needs them."
      >
        {INTERNAL_SKILLS.map((s) => (
          <Row key={s.name} left={s.name} tag={s.trigger} body={s.role} />
        ))}
      </Section>

      <Section
        id="agents"
        title="Review agents"
        intro="Fresh-context reviewers spawned by /rite-review, /rite-seal, and the strategic gates. None of them wrote the code, so none inherits its blind spots."
      >
        {REVIEW_AGENTS.map((a) => (
          <Row key={a.name} left={a.name} body={a.checks} />
        ))}
      </Section>

      <Reveal>
        <p className="mt-5 text-[0.9rem] leading-relaxed text-ink-muted">
          The one write-capable agent is <code className="k k--accent">devrites-slice-wright</code>:{" "}
          <code className="k">/rite-build</code> dispatches it in a fresh context to implement a single
          slice test-first, then it returns a structured artifact for the orchestrator to doubt, record,
          and gate. It writes code and tests, never the workspace bookkeeping.
        </p>
      </Reveal>

      <Section
        id="cli"
        title="CLI & MCP"
        intro="The discipline lives in the .devrites/ files and the state scripts, not in the harness, so any tool can drive the workflow through the same gates. The exit code is the verdict."
      >
        <div className="border-b border-line p-4">
          <code className="mono text-sm text-accent">devrites</code>{" "}
          <span className="text-ink-faint">CLI</span>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {CLI_GATES.map((g) => (
              <span key={g} className="mono rounded-md border border-line bg-surface-2/50 px-2 py-1 text-[0.72rem] text-ink-muted">
                {g}
              </span>
            ))}
          </div>
          <p className="mt-3 text-[0.9rem] leading-relaxed text-ink-muted">
            Scriptable in any agent loop or a pre-merge CI step.
          </p>
        </div>
        <div className="p-4">
          <code className="mono text-sm text-accent">MCP server</code>{" "}
          <span className="text-ink-faint">dependency-free stdio</span>
          <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">
            Exposes <code className="k">devrites_ready</code>, <code className="k">devrites_acceptance</code>,{" "}
            <code className="k">devrites_orient</code> and the rest as MCP tools. Register it in a project&rsquo;s{" "}
            <code className="k">.mcp.json</code> and Cursor, Codex, Gemini CLI, CI, or a human can ask the
            same question of the same files.
          </p>
        </div>
      </Section>

      <Reveal>
        <div className="tile mt-8 p-5">
          <h3 className="font-bold text-ink">Naming</h3>
          <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">
            The <code className="k">devrites-</code> prefix is a namespace for collision avoidance against
            bundled Claude Code skills. It does not mean &ldquo;internal&rdquo;: visibility is set by the
            user-invocable flag in each skill.
          </p>
        </div>
      </Reveal>
    </>
  );
}

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
  description: "A reference for DevRites skills, agents, triggers, and engine commands.",
  alternates: { canonical: "/docs/command-map/" },
};

function Section({ id, title, intro, children }: { id: string; title: string; intro?: string; children: React.ReactNode }) {
  return (
    <section aria-labelledby={id} className="mt-14 first:mt-0">
      <Reveal>
        <h2 id={id} className="scroll-mt-28 text-2xl font-bold">{title}</h2>
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
        lead="DevRites ships 42 skills, 13 read-only agents, one slice writer, and a shared engine CLI. Claude uses /rite-* and Codex uses $rite-* to run the same generated skills."
      />

      <Section id="public" title="Public commands" intro="You can run these commands directly. Each one maps to a workflow skill.">
        {PUBLIC_COMMANDS.map((c) => (
          <Row key={c.cmd} left={c.cmd} tag={c.phase} body={c.desc} />
        ))}
      </Section>

      <Section
        id="internal"
        title="Internal skills"
        intro="The model invokes these specialists when a phase needs them. They do not appear in the menu."
      >
        {INTERNAL_SKILLS.map((s) => (
          <Row key={s.name} left={s.name} tag={s.trigger} body={s.role} />
        ))}
      </Section>

      <Section
        id="agents"
        title="Fresh-context agents"
        intro="Thirteen read-only reviewers, judges, and the archive retrospector. They receive bounded evidence rather than the orchestrator's reasoning."
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
          and gate. It writes code and tests; the engine handles workspace bookkeeping.
        </p>
      </Reveal>

      <Section
        id="cli"
        title="Go control-plane CLI"
        intro="The stdlib-only devrites-engine binary handles workspace operations. Claude, Codex, CI, scripts, and humans call the same commands, and the exit code reports the result."
      >
        <div className="border-b border-line p-4">
          <code className="mono text-sm text-accent">devrites-engine</code>{" "}
          <span className="text-ink-faint">common commands</span>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {CLI_GATES.map((g) => (
              <span key={g} className="mono rounded-md border border-line bg-surface-2/50 px-2 py-1 text-[0.72rem] text-ink-muted">
                {g}
              </span>
            ))}
          </div>
          <p className="mt-3 text-[0.9rem] leading-relaxed text-ink-muted">
            Run <code className="k">devrites-engine help</code> for the full list of commands and hooks.
            Exit 3 is a structured HITL pause; resolve the named gap and retry.
          </p>
        </div>
        <div className="p-4">
          <code className="mono text-sm text-accent">structured automation</code>{" "}
          <span className="text-ink-faint">JSON + generated hooks</span>
          <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">
            <code className="k">snapshot</code> emits the stable <code className="k">devrites.workspace.v1</code>{" "}
            contract; AFK-parsed read commands support <code className="k">--json</code>. Generated Claude
            and Codex hooks call the same binary for orientation, cursors, red-test watching, source
            boundaries, reviewer read-only enforcement, and compaction handoffs.
          </p>
        </div>
      </Section>

      <Reveal>
        <div className="tile mt-8 p-5">
          <h3 className="font-bold text-ink">Naming</h3>
          <p className="mt-2 text-[0.9rem] leading-relaxed text-ink-muted">
            The <code className="k">devrites-</code> prefix prevents name collisions with bundled host
            skills. It does not mean &quot;internal&quot;. Visibility is set by the
            user-invocable flag in each skill, and generation preserves it for Claude and Codex.
          </p>
        </div>
      </Reveal>
    </>
  );
}

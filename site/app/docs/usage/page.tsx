import type { Metadata } from "next";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Code, Callout } from "@/components/docs/DocsBits";

export const metadata: Metadata = {
  title: "Usage & examples",
  description: "Examples of the DevRites build loop, spec drift, HITL gates, AFK runs, and the unattended lifecycle.",
  alternates: { canonical: "/docs/usage/" },
};

export default function Usage() {
  return (
    <>
      <DocsHeader
        crumb="usage"
        title="Usage & examples"
        lead="These examples use Claude's /rite-* syntax. In Codex, use $rite-* instead. Both hosts read the same active workspace and call the same engine gates."
      />

      <H2 id="loop" first>The build loop</H2>
      <P>
        The normal workflow builds one slice at a time. <code className="k">/rite-build</code> does not
        auto-advances, so you decide when the next slice runs. <code className="k">/rite-seal</code>{" "}
        decides; <code className="k">/rite-ship</code> executes and closes.
      </P>
      <Code>
        <span className="text-go">/rite-spec</span> add-csv-export{"    "}<span className="text-ink-faint"># investigate → spec.md</span>{"\n"}
        <span className="text-go">/rite-define</span>{"                 "}<span className="text-ink-faint"># spec → plan + vertical slices</span>{"\n"}
        <span className="text-go">/rite-vet</span>{"                    "}<span className="text-ink-faint"># mandatory review; depth scales to stakes</span>{"\n"}
        <span className="text-go">/rite-build</span>{"                  "}<span className="text-ink-faint"># slice 1, stops with evidence</span>{"\n"}
        <span className="text-go">/rite-build</span>{"                  "}<span className="text-ink-faint"># slice 2, repeat per slice</span>{"\n"}
        <span className="text-go">/rite-prove</span>{"                  "}<span className="text-ink-faint"># all slices built: tests + browser proof</span>{"\n"}
        <span className="text-go">/rite-polish</span>{"                 "}<span className="text-ink-faint"># code polish, then UI polish if UI</span>{"\n"}
        <span className="text-go">/rite-review</span>{"                 "}<span className="text-ink-faint"># multi-axis review, in parallel</span>{"\n"}
        <span className="text-go">/rite-seal</span>{"                   "}<span className="text-ink-faint"># GO / NO-GO (no git)</span>{"\n"}
        <span className="text-go">/rite-ship</span>{"                   "}<span className="text-ink-faint"># type-GO → commit · push · tag · archive</span>
      </Code>

      <H2 id="drift">Spec drift mid-build</H2>
      <P>
        If implementation exposes an assumption that the code does not support, the build stops and records
        the mismatch. When fixing it would change product behavior, DevRites asks you which direction to take.
      </P>
      <Code>
        <span className="text-go">/rite-build</span>{"\n"}
        <span className="text-ink-faint">  → planned User.export_token column does not exist;</span>{"\n"}
        <span className="text-ink-faint">    adding it changes the data model</span>{"\n"}
        <span className="text-warn">  ⚠ STOPS (Spec Drift Guard), records drift.md, asks:</span>{"\n"}
        {"     1. keep the requirement, add the column + migration\n"}
        {"     2. adjust to per-session tokens (matches existing behavior)\n"}
        {"     3. split token work into a follow-up feature\n"}
        <span className="text-accent">You: 2</span>{"\n"}
        <span className="text-go">/rite-plan repair</span>{"   "}<span className="text-ink-faint"># reslice around the corrected model</span>{"\n"}
        <span className="text-go">/rite-build</span>{"          "}<span className="text-ink-faint"># resume on the repaired plan</span>
      </Code>
      <Callout title="Why repair happens during the build">
        Fixing a bad assumption at slice 3 requires one plan repair. Finding it at seal can require reworking
        the whole feature, so the build stops as soon as it detects the mismatch.
      </Callout>

      <H2 id="hitl">HITL gate: pause and resume</H2>
      <P>
        A risky slice pauses at a typed checkpoint before writing code. Answer once and the build
        resumes with your decision captured in <code className="k">questions.md</code>.
      </P>
      <Code>
        <span className="text-go">/rite-build</span>{"\n"}
        <span className="text-ink-faint">  → slice 03 is HITL (blocking, SLA 15m). STOPS before code:</span>{"\n"}
        {"    Checkpoint: composite index, or two single-col indexes?\n"}
        {"    Proposed: composite, one read path, both columns filtered together.\n"}
        <span className="text-accent">You: /rite-resolve q-2026-05-28-001 &quot;composite&quot;</span>{"\n"}
        <span className="text-go">/rite-build</span>{"   "}<span className="text-ink-faint"># resumes with the answer</span>
      </Code>

      <H2 id="afk">AFK overnight run</H2>
      <P>
        Add a <code className="k">.devrites/AFK</code> sentinel to run the loop unattended. Discretionary
        pauses become advisory notes. Irreversible risk still stops the run and can trigger a notification.
      </P>
      <Code>
        <span className="text-ink-faint"># drop the sentinel before bed; keys optional</span>{"\n"}
        <span className="text-accent">cat</span> &gt; .devrites/AFK &lt;&lt;&apos;EOF&apos;{"\n"}
        {"max_slices: 10\n"}
        {"notify: \"curl -d \\\"$DEVRITES_QID: $DEVRITES_QUESTION\\\" ntfy.sh/my-topic\"\n"}
        {"allow_gates: [advisory, validating]\n"}
        EOF{"\n"}
        <span className="text-go">/rite-build</span>{"   "}<span className="text-ink-faint"># builds AFK slices; blocking gate or red tests pause + notify</span>
      </Code>
      <P>
        If tests, types, or lint fail, the loop leaves the slice incomplete, writes a blocking question,
        and stops regardless of <code className="k">allow_gates</code>. Destructive migrations, auth
        changes, and public API breaks also pause the run.
      </P>

      <H2 id="auto">Fully unattended: /rite-autocomplete</H2>
      <P>
        This command runs the lifecycle in order. At a soft gate, it chooses the specialist's preferred
        option and records the reason. It still pauses for irreversible risk, a NO-GO, exhausted slices,
        or low confidence.
      </P>
      <Code>
        <span className="text-go">/rite-autocomplete</span> &quot;add CSV export for admins&quot; --max-slices 8{"\n"}
        <span className="text-ink-faint">  → interview once → spec → temper → define → vet →</span>{"\n"}
        <span className="text-ink-faint">    build ×N → prove → polish → review → seal</span>{"\n"}
        <span className="text-ink-faint">  → seal returns GO → stops, hands off to /rite-ship</span>{"\n"}
        <span className="text-go">/rite-ship</span> · <span className="text-accent">You: GO</span>{"   "}<span className="text-ink-faint"># or pass --ship for zero-touch</span>
      </Code>

      <H2 id="checking-in">Checking in</H2>
      <P>
        <code className="k">/rite</code> shows the menu, active status, and suggested next command without running a phase.{" "}
        <code className="k">/rite-status</code> gives the full picture: phase, run mode, status, next action,
        evidence, open questions by gate, drift, and handoff readiness.{" "}
        <code className="k">/rite-resolve &lt;qid&gt; &quot;&lt;answer&gt;&quot;</code> answers a gate and is the only
        command that clears an &quot;awaiting human&quot; pause.
      </P>

      <Callout title="Tips">
        Commit <code className="k">.devrites/</code> so the team shares feature state, but gitignore each
        developer's <code className="k">.devrites/AFK</code> sentinel. Work through the prompt and plan in
        HITL first. When the plan is stable, add the sentinel and cap the run with <code className="k">max_slices</code>.
      </Callout>
    </>
  );
}

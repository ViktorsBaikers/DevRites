import type { Metadata } from "next";
import DocsHeader from "@/components/docs/DocsHeader";
import { H2, P, Code, Callout } from "@/components/docs/DocsBits";

export const metadata: Metadata = {
  title: "Usage & examples",
  description: "Worked DevRites workflows: the build loop, spec drift mid-build, HITL gates, AFK overnight runs, and the fully unattended lifecycle.",
  alternates: { canonical: "/docs/usage/" },
};

export default function Usage() {
  return (
    <>
      <DocsHeader
        crumb="usage"
        title="Usage & examples"
        lead="Worked workflows, start to finish. Every feature begins with /rite-spec, which investigates, writes the spec, and creates the workspace. Every later phase reads the active workspace first."
      />

      <H2 id="loop">The build loop</H2>
      <P>
        The normal path. Build one slice at a time; <code className="k">/rite-build</code> never
        auto-advances, so you decide when the next slice runs. <code className="k">/rite-seal</code>{" "}
        decides; <code className="k">/rite-ship</code> executes and closes.
      </P>
      <Code>
        <span className="text-go">/rite-spec</span> add-csv-export{"    "}<span className="text-ink-faint"># investigate → spec.md</span>{"\n"}
        <span className="text-go">/rite-define</span>{"                 "}<span className="text-ink-faint"># spec → plan + vertical slices</span>{"\n"}
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
        When the build discovers the plan assumed something the code does not support, it stops instead of
        pushing on, records the drift, and asks you which direction to take when product behavior changes.
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
      <Callout title="Why it matters">
        A wrong turn caught at slice 3 costs one repair. The same wrong turn caught at seal costs the whole
        feature. The guard makes the cheap moment the default.
      </Callout>

      <H2 id="hitl">HITL gate: pause and resume</H2>
      <P>
        Risky slices pause before any code is written, at a typed checkpoint. Answer once and the build
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
        Drop a <code className="k">.devrites/AFK</code> sentinel and the loop runs unattended. Discretionary
        pauses downgrade to advisory notes; irreversible risk always stops and pings you.
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
        The loop refuses to mark a slice built if tests, types, or lint go red. It writes a blocking
        question and stops regardless of <code className="k">allow_gates</code>. Destructive migrations,
        auth changes, and public-API breaks always pause too.
      </P>

      <H2 id="auto">Fully unattended: /rite-autocomplete</H2>
      <P>
        Runs the whole lifecycle in order, picking the option each specialist favours at soft gates and
        recording the rationale. It still pauses on irreversible risk, a NO-GO, exhausted slices, or low
        confidence.
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
        <code className="k">/rite</code> shows the menu and the suggested next command without reading state.{" "}
        <code className="k">/rite-status</code> gives the full picture: phase, run mode, status, next action,
        evidence, open questions by gate, drift, and handoff readiness.{" "}
        <code className="k">/rite-resolve &lt;qid&gt; &quot;&lt;answer&gt;&quot;</code> answers a gate and is the only
        thing that clears an &ldquo;awaiting human&rdquo; pause.
      </P>

      <Callout title="Tips">
        Commit <code className="k">.devrites/</code> so the team shares feature state, but gitignore the
        per-developer <code className="k">.devrites/AFK</code> sentinel. Refine the prompt and plan in HITL
        first, then drop the sentinel for the bulk stretch, and always cap iterations with{" "}
        <code className="k">max_slices</code>.
      </Callout>
    </>
  );
}

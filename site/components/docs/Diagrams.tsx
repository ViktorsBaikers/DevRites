import { ArrowRight, ArrowDown } from "lucide-react";
import { PHASES } from "@/lib/site";

export function PipelineDiagram() {
  return <div className="docs-diagram" aria-label="Feature lifecycle grouped by purpose">{[
    { act: "shape", label: "Agree & plan" }, { act: "build", label: "Build & verify" }, { act: "ship", label: "Decide & ship" },
  ].map(({ act, label }) => <div key={act} className="docs-diagram-row"><strong>{label}</strong><div>{PHASES.filter((phase) => phase.act === act).map((phase, i) => <div key={phase.name} className="inline-flex items-center gap-2">{i > 0 && <ArrowRight size={14} aria-hidden />}<span>{phase.name}{phase.name === "converge" ? " (if needed)" : phase.optional ? " (optional)" : ""}</span></div>)}</div></div>)}</div>;
}

export function PhaseLoop() {
  return <div className="docs-process">{[
    { title: "Read the record", body: "The phase loads the contract and the files it needs from the active workspace." },
    { title: "Do the work", body: "The agent investigates, implements, or reviews a bounded part of the feature." },
    { title: "Leave evidence", body: "The result and next action go back into the repository, ready for the next session." },
  ].map((step) => <div key={step.title}><h3>{step.title}</h3><p>{step.body}</p></div>)}</div>;
}

export function StackDiagram() {
  return <div className="docs-diagram">{[
    { title: "Your host", body: "Claude Code · Codex · omp · pi · Devin CLI" },
    { title: "Engineering judgment", body: "Workflow skills dispatch bounded work to specialist agents." },
    { title: "Deterministic checks", body: "devrites-engine validates state, gates, and the exact candidate." },
    { title: "Durable record", body: ".devrites/ keeps the contract, decisions, current state, and evidence." },
  ].map((layer, i) => <div className="docs-diagram-row" key={layer.title}><strong>{layer.title}</strong><div>{i > 0 && <ArrowDown size={16} aria-hidden />}<p>{layer.body}</p></div></div>)}</div>;
}

export function DriftLoop() {
  return <div className="docs-process">{[
    { title: "Notice the mismatch", body: "Build records where the implementation and plan disagree in drift.md." },
    { title: "Resolve & repair", body: "A human decides any product change. Plan repair updates the affected slices." },
    { title: "Check & resume", body: "The repaired plan returns through Vet before the next build step." },
  ].map((step) => <div key={step.title}><h3>{step.title}</h3><p>{step.body}</p></div>)}</div>;
}

export function ProofLadder() {
  return <div className="docs-diagram">{[
    ["Real browser", "Exercise the user journey and inspect screenshots, console, network, and responsive behavior."],
    ["Runtime checks", "Run the actual application or host and verify the changed behavior."],
    ["Project E2E", "Use the project's existing end-to-end tests to prove the acceptance criteria."],
    ["Manual fallback", "Record exact steps and the limitation. Seal considers the remaining risk."],
  ].map(([title, body]) => <div className="docs-diagram-row" key={title}><strong>{title}</strong><p>{body}</p></div>)}</div>;
}

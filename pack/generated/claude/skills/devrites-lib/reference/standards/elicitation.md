# Elicitation: a move-set for the thinking phases

The thinking phases (`/rite-temper` on the spec, `/rite-vet` on the plan) default to one shape each.
A pre-mortem, a plan review. That's a floor, not a ceiling. When a section needs deeper thought,
this is the menu you reach into: named reasoning techniques, each with the shape it produces, so
"make this stronger" becomes a concrete move instead of a vibe.

## How to use it (progressive: no ceremony by default)
1. Look at the **one section** in front of you (a requirement, a risk, an estimate, a boundary).
2. Read its **risk**, then pull the 3-5 techniques below whose *When to reach for it* matches.
3. Present them as a short numbered menu; the human picks one (or `skip`).
4. Run that technique on that section, apply the result, move on.

Selection is by the section's risk, not taste. Reach for the row that fits:

| The section is… | Reach for |
|---|---|
| an irreversible / boundary / auth / data-model decision | Red-Team vs Blue-Team · Assumption Audit · Pre-Mortem |
| a vague or contested requirement | Steelman-then-Attack · Devil's Advocate · Five Whys |
| an estimate, sizing, or a "how long / how risky" | Delphi · Reference-Class Forecast |
| a design with more than one plausible shape | Tournament (A vs B) · Inversion · Analogy Mapping |
| a spec that feels *too* big or *too* small | Scope Extremes (MVP vs Gold-plate) · YAGNI Pass |
| a risk surface you suspect is under-explored | Chaos Scenarios · Edge-Case Hunt · Second-Order Effects |

## The techniques

Each entry is **name (when to reach for it) the shape it produces** (`→` reads "then").

### Sharpening a claim
- **Steelman-then-Attack:** a requirement or approach you're inclined to accept. Build the
  *strongest* case for it → then attack that strongest form. Survives → keep; falls → revise.
- **Devil's Advocate:** consensus formed too fast. One voice argues the opposite in good faith →
  surface the objection the room skipped.
- **Assumption Audit:** a plan resting on unstated beliefs. List every assumption the section
  makes → mark each *load-bearing / verified / unverified* → the unverified load-bearing ones are
  the work.
- **Five Whys:** a stated requirement whose *reason* is fuzzy. Ask "why" five times down from the
  ask → reach the root need (often narrower or different than the surface ask).

### Stress-testing a decision
- **Pre-Mortem:** before committing an irreversible choice. Assume it's six months later and this
  failed → write the failure story → each cause becomes a mitigation or a blocking question. (This
  is `/rite-temper`'s default; the others deepen it.)
- **Red-Team vs Blue-Team:** a security, trust-boundary, or adversarial-input surface.
  `defense → attack → hardening`: state the defense, attack it as a hostile party, harden the gap.
- **Inversion:** "how do we make X good?" stalls. Ask instead "how would we *guarantee X fails*?"
  → invert each failure into a requirement.
- **Second-Order Effects:** a change that touches shared surface. For each first-order effect ask
  "and then what?" twice → surface the downstream consequence the diff hides.

### Sizing and estimating
- **Delphi:** an estimate one person is anchoring. Gather independent estimates *without* seeing
  each other → reveal → discuss the spread → re-estimate → converge. Kills anchoring.
- **Reference-Class Forecast:** "this'll be quick." Find 3 past changes of the same *class* → use
  their actual cost as the base rate, not the inside view.

### Widening the option space
- **Tournament (A vs B):** two plausible designs. Put them head-to-head on the *decision hinge*
  (the one axis that differs) → pick on that axis, not on a feature checklist.
- **Analogy Mapping:** a novel problem. "What is this *like* that's already solved?" → borrow the
  solved structure, note where the analogy breaks.
- **Scope Extremes:** a spec of uncertain size. Describe the *MVP* (smallest thing that ships
  value) and the *gold-plated* version → the right scope is usually named by the gap between them.
- **YAGNI Pass:** a plan carrying "might need it later." For each speculative piece, demand a
  *current* acceptance criterion → no criterion, cut it to a follow-up.

### Finding what's missing
- **Edge-Case Hunt:** a happy-path-shaped spec. Walk empty / boundary / invalid / concurrent /
  huge-or-weird input for each element → each uncovered case is a missing criterion.
- **Chaos Scenarios:** a system with external dependencies. "What if the DB is down / the API
  times out / the clock jumps / the input is hostile?" → each becomes a failure-mode requirement.
- **Completeness Critic:** end of a section. Ask only "what's *missing*: a modality not covered, a
  claim not verified, a flow with no path?" → the answer is the next round of work.

## The one rule
Run the technique on the **section in front of you**, not the whole document: a move applied to
everything is a move applied to nothing. Record what it changed in the phase artifact (a new
requirement, a mitigation, a blocking question), not just that you ran it.

A project can append its own house techniques to this file: same three-column shape.

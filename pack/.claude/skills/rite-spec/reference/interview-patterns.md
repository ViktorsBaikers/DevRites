# Interview patterns

Question ladders by domain. Pull the rung that matches the biggest current unknown.
Each question still follows the protocol: one at a time, best guess attached.

## Objective / problem
- "What does success look like in one sentence?"
- "Who hits this, how often, and what do they do today instead?"
- "If we shipped only one thing here, what must it be?"

## Scope boundaries
- "What is explicitly **out** of scope for this first version?"
- "Is this a new capability or a change to an existing flow?"
- "What's the smallest version that's still useful?" (→ slice 1)

## Data model
- "What are the core entities and how do they relate?"
- "What's the source of truth, and can these fields change after creation?"
- "Any uniqueness, ordering, or soft-delete needs?"

## UX / flow
- "Walk me through the happy path screen by screen."
- "What are the empty / loading / error / permission-denied states?"
- "Brand surface (marketing) or product surface (app UI)?" (drives craft)

## Integration / external
- "Which external systems or APIs are involved, and who owns them?"
- "Sync or async? What happens when the dependency is down?"

## Non-functional
- "Any latency, scale, or volume targets I should design to?"
- "Auth/permission rules? Anything sensitive (PII, secrets, payments)?"

## Acceptance
- "How will we *prove* this works — what's the test or observation?"
- "What would make you reject the PR?"

## Anti-references (taste)
- "Show me one thing that does this well, and one that does it badly."

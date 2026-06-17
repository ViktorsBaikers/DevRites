# DevRites — flow diagrams

Visual reference for how the skills, agents, and rules fit together. GitHub
renders Mermaid natively — open this file on the repo to see the graphs.

For the full per-skill table, see [`command-map.md`](command-map.md). For the
"why" behind each piece, see [`architecture.md`](architecture.md).

## 1. Feature lifecycle

The happy path. Every arrow assumes the readiness gate of the previous phase
passed; failures route through `/rite-plan repair` or `devrites-debug-recovery`.
HITL slices pause before code is written; `/rite-resolve` is the resume verb.

```mermaid
flowchart LR
    Start([user has an idea]) --> Spec[/rite-spec/]
    Spec -->|spec.md ready| Define[/rite-define/]
    Define -->|plan.md + tasks.md<br/>each slice tagged AFK/HITL| Build[/rite-build/]
    Build -->|one slice done<br/>+ evidence| Build
    Build -->|HITL gate fires| Await{{Awaiting human<br/>state.md + questions.md}}
    Await -->|"/rite-resolve &lt;qid&gt; &lt;answer&gt;"| Build
    Build -->|all slices built| Prove[/rite-prove/]
    Prove -->|evidence captured| Polish[/rite-polish/]
    Polish -->|polish-report.md| Review[/rite-review/]
    Review -->|review.md<br/>Critical == 0| Seal[/rite-seal/]
    Seal -->|GO| Ship2[/rite-ship/]
    Ship2 -->|type-GO| Shipped([commit · push · tag · archive])
    Seal -->|NO-GO| Repair[/rite-plan repair/]
    Repair --> Build

    Build -.->|Spec Drift Guard| Repair
    Prove -.->|drift / failure| Repair
    Polish -.->|drift| Repair
    Review -.->|drift| Repair

    classDef phase fill:#1f2937,stroke:#60a5fa,stroke-width:1px,color:#f9fafb
    classDef done fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef repair fill:#4c1d95,stroke:#a78bfa,color:#f5f3ff
    classDef gate fill:#4c1d95,stroke:#a78bfa,color:#f5f3ff
    class Spec,Define,Build,Prove,Polish,Review,Seal,Ship2 phase
    class Shipped done
    class Repair repair
    class Await gate
```

## 2. `/rite-polish` orchestrator

`/rite-polish` is a thin dispatcher that always runs code polish and runs UI
polish only when the diff touches UI files.

```mermaid
flowchart TD
    P[/rite-polish/] -->|read diff + touched-files.md| D{UI<br/>touched?}
    D -->|always| Code[reference/code.md<br/>Phase 1 + 2]
    D -->|yes| UI[reference/ui.md<br/>Phase 3 + 4]
    Code -->|appends to polish-report.md| Out([polish-report.md])
    UI -->|appends to polish-report.md| Out
    Code -.->|Phase 1| Simp[devrites-audit simplify]
    Simp -.->|spawns| SR[devrites-simplifier-reviewer]

    classDef orch fill:#1f2937,stroke:#60a5fa,color:#f9fafb
    classDef sub fill:#312e81,stroke:#818cf8,color:#eef2ff
    classDef agent fill:#7c2d12,stroke:#fb923c,color:#fff7ed
    class P orch
    class Code,UI,Simp sub
    class SR agent
```

Mode tokens passed to `/rite-polish` (`bolder`, `quieter`, `distill`,
`harden`, `normalize-only`) flow through to Phase 4 (`reference/ui.md`) as
emphasis dials. `normalize-only` stops after Phase 3.

## 3. `/rite-review` parallel axes

Review runs Spec coverage and Standards compliance in parallel sub-agents so
neither masks the other.

```mermaid
flowchart LR
    R[/rite-review/] -->|spawn parallel<br/>via Task tool| S[devrites-spec-reviewer<br/>**Spec axis**]
    R -->|spawn parallel<br/>via Task tool| C[devrites-code-reviewer<br/>**Standards axis**]
    S -->|missing / partial / wrong /<br/>scope-creep findings| Combine
    C -->|standards violations<br/>cite rule + file| Combine
    R --> Sec[devrites-audit security]
    R --> Perf[devrites-audit perf]
    Sec -->|labelled findings| Combine
    Perf -->|labelled findings| Combine
    Combine([review.md<br/>Critical / Important / Suggestion / Nit / FYI])

    classDef phase fill:#1f2937,stroke:#60a5fa,color:#f9fafb
    classDef agent fill:#7c2d12,stroke:#fb923c,color:#fff7ed
    classDef skill fill:#312e81,stroke:#818cf8,color:#eef2ff
    class R phase
    class S,C agent
    class Sec,Perf skill
```

## 4. `/rite-seal` fan-out

The seal fans out **all** relevant reviewers in parallel and reconciles their
findings, then **decides** GO / NO-GO and stops. It no longer runs git — on GO
it hands off to `/rite-ship`, which renders the type-GO prompt and runs the
irreversible commit · push · tag · archive. The advisory `/20` score has been
removed — the gate is severity + acceptance + drift.

```mermaid
flowchart TB
    Seal[/rite-seal/] -->|read all artifacts| Walk[walk acceptance<br/>criteria one by one]
    Walk -->|spawn in parallel| SpecRev[devrites-spec-reviewer]
    Walk -->|spawn in parallel| CodeRev[devrites-code-reviewer]
    Walk -->|spawn in parallel| TestRev[devrites-test-analyst]
    Walk -.->|UI only| FERev[devrites-frontend-reviewer]
    Walk -.->|input/auth/data| SecRev[devrites-security-auditor]
    Walk -.->|perf relevant| PerfRev[devrites-performance-reviewer]
    SpecRev --> Gate
    CodeRev --> Gate
    TestRev --> Gate
    FERev --> Gate
    SecRev --> Gate
    PerfRev --> Gate
    Gate{Critical == 0?<br/>Acceptance proven?<br/>Drift resolved?}
    Gate -->|yes + Important == 0| Go
    Gate -->|yes + Important > 0| YN[render interactive<br/>y/N prompt]
    Gate -->|no| NoGo[NO-GO]
    YN -->|y| Go
    YN -->|N| NoGo
    Go[GO: write seal.md<br/>Next step: /rite-ship] -->|hand off, type-GO + git live in ship| Ship([/rite-ship/])

    classDef phase fill:#1f2937,stroke:#60a5fa,color:#f9fafb
    classDef agent fill:#7c2d12,stroke:#fb923c,color:#fff7ed
    classDef gate fill:#4c1d95,stroke:#a78bfa,color:#f5f3ff
    classDef ship fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef stop fill:#7f1d1d,stroke:#f87171,color:#fee2e2
    class Seal,Walk phase
    class SpecRev,CodeRev,TestRev,FERev,SecRev,PerfRev agent
    class Gate,YN gate
    class Go,Ship ship
    class NoGo stop
```

## 5. `devrites-debug-recovery` six-phase loop

Failure recovery — the loop construction in Phase 1 is the load-bearing piece.

```mermaid
flowchart LR
    F([failing test /<br/>build / runtime]) --> L1[Phase 1<br/>Build the loop]
    L1 -->|fast deterministic signal| R[Phase 2<br/>Reproduce]
    R -->|exact error text| H[Phase 3<br/>Ranked hypotheses 3-5]
    H -->|show user before testing| I[Phase 4<br/>Instrument]
    I -->|change one variable| Fix[Phase 5<br/>Fix + regression test]
    Fix --> C[Phase 6<br/>Cleanup + classify]
    L1 -.->|can't build loop| Ask([STOP — ask user])

    classDef phase fill:#1f2937,stroke:#60a5fa,color:#f9fafb
    classDef stop fill:#7f1d1d,stroke:#f87171,color:#fee2e2
    class L1,R,H,I,Fix,C phase
    class Ask stop
```

Each phase's detail lives in a separate reference file under
`pack/.claude/skills/devrites-debug-recovery/reference/` so the SKILL.md body
stays small.

## 6. Engineering-rules carrier

Each `rite-*` skill Reads `.claude/rules/core.md` (the always-on subset) as
its first step (step 0); the other rule files load on demand. Per-phase
skills pull additional rule files via plain `Read` as their workflow
demands. No carrier skill, no session-start autoload.

```mermaid
flowchart TD
    R[rite-* skill<br/>step 0] -->|always-on| Core[.claude/rules/core.md]
    R -->|on demand index| Idx[(.claude/rules/README.md<br/>15 specialist rule files)]
    Idx --> CS[coding-style.md]
    Idx --> EH[error-handling.md]
    Idx --> T[testing.md]
    Idx --> CR[code-review.md]
    Idx --> S[security.md]
    Idx --> P[performance.md]
    Idx --> Pat[patterns.md]
    Idx --> Gw[git-workflow.md]
    Idx --> Hk[hooks.md]
    Idx --> Doc[documentation.md]
    Idx --> DWf[development-workflow.md]
    Idx --> Ag[agents.md]
    Idx --> CH[context-hygiene.md]
    Idx --> Afk[afk-hitl.md]
    Idx --> AP[anti-patterns.md]

    Build[/rite-build/] -.->|pulls| CS
    Build -.->|pulls| EH
    Build -.->|pulls| T
    Polish[/rite-polish/] -.->|pulls| CS
    Polish -.->|pulls| Pat
    Review[/rite-review/] -.->|pulls| CR
    Review -.->|pulls| S
    Review -.->|pulls| P
    Seal[/rite-seal/] -.->|pulls| Ag
    Seal -.->|pulls| CR
    Ship[/rite-ship/] -.->|pulls| Gw

    classDef carrier fill:#312e81,stroke:#818cf8,color:#eef2ff
    classDef rule fill:#1f2937,stroke:#9ca3af,color:#f9fafb
    classDef phase fill:#064e3b,stroke:#34d399,color:#ecfdf5
    class R carrier
    class Core,CS,EH,T,CR,S,P,Pat,Gw,Hk,Doc,DWf,Ag,CH,Afk,AP rule
    class Build,Polish,Review,Seal,Ship phase
```

## 7. Workspace state model

`.devrites/work/<feature-slug>/` is the durable memory each phase reads
before doing anything. **Every** rite-* skill reads the workspace first; if
no `ACTIVE` is set, it tells the user to run `/rite-spec`. The optional
`.devrites/AFK` sentinel sits beside `ACTIVE` and toggles the session-level
run mode for all skills.

```mermaid
erDiagram
    ACTIVE ||--o| WORKSPACE : points-to
    AFK_SENTINEL }|..|| RUN_MODE : "presence is authoritative — skills re-read at decision time"
    WORKSPACE ||--|| state : has
    WORKSPACE ||--|| brief : has
    WORKSPACE ||--|| spec : has
    WORKSPACE ||--|| plan : "has (from /rite-define)"
    WORKSPACE ||--|| tasks : "has — slices tagged Mode + Gate"
    WORKSPACE ||--o{ references : "has (design refs)"
    WORKSPACE ||--|| questions : "has — qid, gate, status (open/answered/dropped)"
    WORKSPACE ||--|| decisions : has
    WORKSPACE ||--|| assumptions : has
    WORKSPACE ||--|| drift : has
    WORKSPACE ||--|| touched-files : "has (from /rite-build)"
    WORKSPACE ||--|| evidence : "has (from /rite-prove)"
    WORKSPACE ||--o| browser-evidence : "has (UI features)"
    WORKSPACE ||--o| polish-report : "has (from /rite-polish)"
    WORKSPACE ||--o| review : "has (from /rite-review)"
    WORKSPACE ||--o| seal : "has (from /rite-seal)"
    WORKSPACE ||--o| ship : "has (from /rite-ship; archived on close)"

    ACTIVE {
        string slug "names the current workspace"
    }
    AFK_SENTINEL {
        bool present "presence = AFK active"
        int max_slices "read-only initial budget — copied to state.md once"
        string notify "optional shell command on pause"
        list allow_gates "gate severities AFK may auto-handle"
    }
    WORKSPACE {
        string slug PK ".devrites/work/<slug>/"
    }
    state {
        string phase "spec | plan | build | prove | polish | review | seal | ship | done"
        string status "running | awaiting_human | blocked | done"
        string active_slice "N — name"
        int afk_slices_remaining "from .devrites/AFK max_slices on first AFK build"
        block awaiting_human "qid, gate, question, proposed, raised_at (only when paused)"
    }
    questions {
        string qid PK "q-YYYY-MM-DD-NNN"
        string status "open | answered | dropped"
        string gate "advisory | validating | blocking | escalating"
    }
```

The exact list of files per workspace and what each holds is in
[`usage.md`](usage.md#the-workspace). The full pause/resume contract is in
[`pack/.claude/rules/afk-hitl.md`](../pack/.claude/rules/afk-hitl.md).

## 8. Public vs internal namespace

The `devrites-` prefix is collision-avoidance against bundled Claude Code
skills (`prototype`, `handoff`, `triage`, `diagnose`). Visibility is the
`user-invocable:` flag, not the prefix.

```mermaid
flowchart TB
    subgraph Public["Public (user-invocable: true) — 17 skills"]
        direction TB
        R1[/rite/]
        R2[/rite-spec/]
        R3[/rite-define/]
        R4[/rite-plan/]
        R5[/rite-build/]
        R6[/rite-prove/]
        R7[/rite-polish/]
        R8[/rite-review/]
        R9[/rite-seal/]
        R12[/rite-ship/]
        R13[/rite-autocomplete/]
        R10[/rite-status/]
        R11[/rite-resolve/]
        IPT[/rite-pressure-test/]
        D1[/rite-zoom-out/]
        D2[/rite-prototype/]
        D3[/rite-handoff/]
    end
    subgraph Internal["Internal (user-invocable: false) — 8 skills, model-invoked"]
        direction TB
        I1[devrites-api-interface]
        I2[devrites-audit<br/>security · perf · simplify]
        I3[devrites-browser-proof]
        I4[devrites-debug-recovery]
        I5[devrites-doubt]
        I6[devrites-frontend-craft]
        I7[devrites-interview]
        I8[devrites-source-driven]
    end

    classDef pub fill:#064e3b,stroke:#34d399,color:#ecfdf5
    classDef int fill:#1f2937,stroke:#9ca3af,color:#f9fafb
    class R1,R2,R3,R4,R5,R6,R7,R8,R9,R12,R13,R10,R11,IPT,D1,D2,D3 pub
    class I1,I2,I3,I4,I5,I6,I7,I8 int
```

## 9. AFK & HITL state machine

The pause/resume primitive for HITL gates and the AFK loop discipline. The
gate firing on `/rite-build` is the only writer of `Awaiting human`;
`/rite-resolve` is the only canonical clearer.

```mermaid
stateDiagram-v2
    [*] --> running: /rite-build starts
    running --> running: AFK slice + advisory finding<br/>(log to questions.md, proceed)
    running --> awaiting_human: HITL gate fires<br/>(blocking / escalating / out-of-gate)
    running --> awaiting_human: Fail-on-red<br/>(tests/types/lint)
    running --> awaiting_human: Irreversible risk<br/>(destructive · auth · public API)
    awaiting_human --> running: /rite-resolve qid "<answer>"
    awaiting_human --> running: /rite-resolve --drop qid
    awaiting_human --> blocked: /rite-plan repair (scope change)
    blocked --> running: plan repaired
    running --> done: sealed GO + /rite-ship<br/>(type-GO → commit · push · tag · archive)
    done --> [*]

    note right of awaiting_human
        state.md: Status: awaiting_human
        state.md: Awaiting human block
        questions.md: gate, qid, proposed, raised_at
        notify hook fired (if .devrites/AFK has one)
    end note
```

`AFK` mode (`.devrites/AFK` present) widens which transitions stay in
`running` via `allow_gates` — but `blocking`, `escalating`, fail-on-red, and
the irreversible-risk list always transition to `awaiting_human` regardless.

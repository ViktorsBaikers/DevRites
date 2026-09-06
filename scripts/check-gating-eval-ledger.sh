#!/usr/bin/env bash
# Eval coverage scoreboard: gating ledger + full skill/agent matrix.
#
# Usage:
#   scripts/check-gating-eval-ledger.sh              # blocking: require_behavioral + require_behavioral_agents
#   scripts/check-gating-eval-ledger.sh --advisory   # full scoreboard; exit 0
#   scripts/check-gating-eval-ledger.sh --json         # machine-readable scoreboard
# Env: DEVRITES_COVERAGE_JSON, DEVRITES_BEHAVIORAL_DIR (tests)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE="${DEVRITES_COVERAGE_JSON:-$ROOT/evals/coverage.json}"
ADVISORY=0
JSON=0

for arg in "$@"; do
  case "$arg" in
    --advisory) ADVISORY=1 ;;
    --json) JSON=1; ADVISORY=1 ;;
    -h|--help)
      sed -n '2,8p' "$0"
      exit 0
      ;;
    *)
      echo "unknown argument: $arg" >&2
      exit 2
      ;;
  esac
done

if [[ ! -f "$COVERAGE" ]]; then
  echo "missing evals/coverage.json" >&2
  exit 1
fi

python3 - "$ROOT" "$COVERAGE" "$ADVISORY" "$JSON" <<'PY'
import json
import os
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
coverage = json.loads(pathlib.Path(sys.argv[2]).read_text())
advisory = sys.argv[3] == "1"
as_json = sys.argv[4] == "1"

skills_dir = root / "pack" / ".claude" / "skills"
agents_dir = root / "pack" / ".claude" / "agents"
behavioral_dir = pathlib.Path(
    os.environ.get("DEVRITES_BEHAVIORAL_DIR", str(root / "evals" / "behavioral"))
)
evals_dir = root / "evals"

# Skills with trigger corpora (devrites-lib is intentionally exempt).
skills = sorted(
    p.name
    for p in skills_dir.iterdir()
    if p.is_dir() and (p / "SKILL.md").is_file() and p.name != "devrites-lib"
)

agents = sorted(p.stem for p in agents_dir.glob("*.md") if p.is_file())

def agent_owner(path, data):
    explicit = data.get("agent")
    if isinstance(explicit, str) and explicit.strip() and explicit in agents:
        return explicit
    stem = path.stem
    for agent in agents:
        if stem == agent or stem.startswith(f"{agent}-"):
            return agent
    return None

# Skill coverage counts only skill-owned corpora. An agent-attributed file
# (filename prefix or explicit "agent") must not double-count as the
# dispatching skill's behavioral coverage.
behavioral_by_skill: dict[str, list[str]] = {}
behavioral_by_agent: dict[str, list[str]] = {}
for path in sorted(behavioral_dir.glob("*.json")):
    data = json.loads(path.read_text())
    owner = agent_owner(path, data)
    if owner:
        behavioral_by_agent.setdefault(owner, []).append(path.name)
        continue
    skill = data.get("skill")
    if skill:
        behavioral_by_skill.setdefault(skill, []).append(path.name)

triggers: dict[str, str] = {}
for path in sorted(evals_dir.glob("*.json")):
    if path.name == "coverage.json":
        continue
    data = json.loads(path.read_text())
    triggers[data["skill"]] = path.name

outcome_skills = set(coverage.get("outcome_skills", []))
if not outcome_skills:
    outcome_skills = {
        "rite-seal",
        "rite-prove",
        "rite-review",
        "rite-vet",
        "rite-clarify",
        "rite-build",
        "rite-spec",
        "rite-plan",
        "rite-define",
    }

gating_failed = 0
for skill in coverage.get("require_behavioral", []):
    if skill not in behavioral_by_skill:
        print(f"FAIL: gating skill {skill} missing behavioral eval")
        gating_failed += 1
    elif not as_json:
        print(f"OK: behavioral {skill} -> {', '.join(behavioral_by_skill[skill])}")

agent_gating_failed = 0
for agent in coverage.get("require_behavioral_agents", []):
    if agent not in agents:
        print(f"FAIL: required behavioral agent {agent} is not a shipped agent profile")
        agent_gating_failed += 1
    elif agent not in behavioral_by_agent:
        print(f"FAIL: gating agent {agent} missing behavioral eval")
        agent_gating_failed += 1
    elif not as_json:
        print(f"OK: behavioral agent {agent} -> {', '.join(behavioral_by_agent[agent])}")

for skill in coverage.get("gating_skills", []):
    if skill not in triggers:
        print(f"WARN: gating skill {skill} missing trigger eval")
    elif not as_json:
        print(f"OK: trigger {skill} -> {triggers[skill]}")

# Rubric tier results (offline judge over captured transcripts). Absent file
# keeps the historical "-" placeholder: the tier is advisory and manual.
rubric_by_skill: dict[str, str] = {}
transcripts_dir = root / "evals" / "transcripts"
transcript_count = len(list(transcripts_dir.glob("*.json"))) if transcripts_dir.is_dir() else 0
rubric_latest = root / "evals" / "results" / "rubric-latest.json"
if rubric_latest.exists():
    try:
        rubric = json.loads(rubric_latest.read_text())
        by_corpus: dict[str, list[str]] = {}
        for r in rubric.get("results", []):
            by_corpus.setdefault(str(r.get("corpus")), []).append(str(r.get("verdict")))
        for corpus_name, verdicts in by_corpus.items():
            passes = verdicts.count("pass")
            rubric_by_skill[corpus_name] = f"{passes}/{len(verdicts)} pass"
    except (json.JSONDecodeError, OSError):
        rubric_by_skill = {}

def row(name: str, trigger: str, outcome: str, behavioral: str, rubric: str) -> dict:
    return {
        "name": name,
        "trigger": trigger,
        "outcome": outcome,
        "behavioral": behavioral,
        "rubric": rubric,
    }

skill_rows = []
for skill in skills:
    skill_rows.append(
        row(
            skill,
            "yes" if skill in triggers else "no",
            "yes" if skill in outcome_skills else "-",
            "yes" if skill in behavioral_by_skill else "no",
            rubric_by_skill.get(skill, "-"),
        )
    )

agent_rows = []
for agent in agents:
    agent_rows.append(
        row(
            agent,
            "-",
            "-",
            "yes" if agent in behavioral_by_agent else "no",
            rubric_by_skill.get(agent, "-"),
        )
    )

if as_json:
    payload = {
        "skills": skill_rows,
        "agents": agent_rows,
        "summary": {
            "skills_total": len(skill_rows),
            "skills_trigger": sum(1 for r in skill_rows if r["trigger"] == "yes"),
            "skills_behavioral": sum(1 for r in skill_rows if r["behavioral"] == "yes"),
            "agents_total": len(agent_rows),
            "agents_behavioral": sum(1 for r in agent_rows if r["behavioral"] == "yes"),
            "gating_behavioral_failed": gating_failed,
            "gating_agent_behavioral_failed": agent_gating_failed,
        },
    }
    print(json.dumps(payload, indent=2))
else:
    print("\n=== eval coverage scoreboard (skills) ===")
    print(f"{'skill':<28} {'trigger':<8} {'outcome':<8} {'behavioral':<11} {'rubric':<6}")
    for r in skill_rows:
        print(
            f"{r['name']:<28} {r['trigger']:<8} {r['outcome']:<8} {r['behavioral']:<11} {r['rubric']:<6}"
        )

    print("\n=== eval coverage scoreboard (agents) ===")
    print(f"{'agent':<32} {'trigger':<8} {'outcome':<8} {'behavioral':<11} {'rubric':<6}")
    for r in agent_rows:
        print(
            f"{r['name']:<32} {r['trigger']:<8} {r['outcome']:<8} {r['behavioral']:<11} {r['rubric']:<6}"
        )

    st = sum(1 for r in skill_rows if r["trigger"] == "yes")
    sb = sum(1 for r in skill_rows if r["behavioral"] == "yes")
    ab = sum(1 for r in agent_rows if r["behavioral"] == "yes")
    print(
        f"\nSummary: skills trigger {st}/{len(skill_rows)} · "
        f"skills behavioral {sb}/{len(skill_rows)} · "
        f"agents behavioral {ab}/{len(agent_rows)} · "
        f"rubric tier: "
        + (
            f"{len(rubric_by_skill)} corpus/graders in rubric-latest.json"
            if rubric_by_skill
            else f"offline-only (manual workflow_dispatch; {transcript_count} captured transcripts, capture protocol: evals/transcripts/README.md)"
        )
    )
    print(
        "outcome tier: deterministic grader covers seal-phase artifacts only "
        "(run-outcome-evals.sh); the 'outcome' column marks the phase each skill's "
        "coverage is intended to reach"
    )
    print(
        f"Gating behavioral skills covered: "
        f"{len(coverage.get('require_behavioral', [])) - gating_failed}/"
        f"{len(coverage.get('require_behavioral', []))}"
    )
    print(
        f"Gating behavioral agents covered: "
        f"{len(coverage.get('require_behavioral_agents', [])) - agent_gating_failed}/"
        f"{len(coverage.get('require_behavioral_agents', []))}"
    )

if advisory:
    sys.exit(0)
sys.exit(1 if gating_failed or agent_gating_failed else 0)
PY

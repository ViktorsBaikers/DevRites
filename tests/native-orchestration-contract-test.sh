#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

python3 - "$ROOT" <<'PY'
from pathlib import Path
import json
import re
import sys

root = Path(sys.argv[1])
canonical = root / "pack/.claude"
agents_doc = canonical / "skills/devrites-lib/reference/standards/agents.md"
profiles = canonical / "skills/devrites-lib/reference/orchestration-profiles.md"
core = canonical / "skills/devrites-lib/reference/standards/core.md"
authoring = canonical / "skills/devrites-lib/reference/standards/skill-authoring.md"
schema = canonical / "skills/devrites-lib/reference/workspace-artifact-schema.md"
state_workspace = canonical / "skills/rite-spec/reference/state-workspace.md"
candidate_integrity = canonical / "skills/devrites-lib/reference/candidate-integrity.md"
command_map = root / "docs/command-map.md"

documented = set(re.findall(r"^\| `(devrites-[a-z-]+)` \|", agents_doc.read_text(), re.M))
claude = {p.stem for p in (canonical / "agents").glob("devrites-*.md")}
codex = {p.stem for p in (root / "pack/generated/codex/agents").glob("devrites-*.toml")}
devin = {p.stem for p in (root / "pack/generated/devin/agents").glob("devrites-*.md")}

if len(documented) != 17:
    raise SystemExit(f"agent catalog contains {len(documented)} roles, want 17")
if claude != documented:
    raise SystemExit(f"Claude profiles differ from catalog: missing={documented-claude}, extra={claude-documented}")
if codex != documented:
    raise SystemExit(f"Codex profiles differ from catalog: missing={documented-codex}, extra={codex-documented}")
if devin != documented:
    raise SystemExit(f"Devin profiles differ from catalog: missing={documented-devin}, extra={devin-documented}")

for role in sorted(documented):
    claude_text = (canonical / "agents" / f"{role}.md").read_text()
    codex_text = (root / "pack/generated/codex/agents" / f"{role}.toml").read_text()
    devin_text = (root / "pack/generated/devin/agents" / f"{role}.md").read_text()
    claude_mode = "acceptEdits" if role == "devrites-slice-wright" else "plan"
    codex_mode = ":workspace" if role == "devrites-slice-wright" else ":read-only"
    if f"name: {role}" not in claude_text or f"permissionMode: {claude_mode}" not in claude_text:
        raise SystemExit(f"Claude role {role} has the wrong identity or permission mode")
    if f'name = "{role}"' not in codex_text or f'default_permissions = "{codex_mode}"' not in codex_text:
        raise SystemExit(f"Codex role {role} has the wrong identity or permission mode")
    devin_frontmatter = devin_text.split("---", 2)[1]
    if f"name: {role}" not in devin_frontmatter or "allowed-tools:" not in devin_frontmatter:
        raise SystemExit(f"Devin role {role} has the wrong identity or missing allowed-tools")
    if "permissionMode" in devin_frontmatter or re.search(r"(?m)^tools:", devin_frontmatter):
        raise SystemExit(f"Devin role {role} leaks Claude frontmatter fields")
    devin_tools = set(re.findall(r"(?m)^  - ([a-z_]+)$", devin_frontmatter))
    if role == "devrites-slice-wright":
        if not {"edit", "write", "exec"} <= devin_tools:
            raise SystemExit(f"Devin wright lost write tools: {sorted(devin_tools)}")
    elif devin_tools & {"edit", "write", "notebook_edit"}:
        raise SystemExit(f"Devin read-only role {role} gained write tools: {sorted(devin_tools)}")

# Every shipped role on every host view carries that host's required identity keys, and
# its writer-ness agrees with the Codex profiles. No host takes a dispatch `mode` key
# (OpenCode, which needs one, is not generated).
def agent_field(head, key, sep):
    found = re.search(rf"(?m)^{re.escape(key)}{sep}\s*(.*)$", head)
    return found.group(1).strip().strip("\"'") if found else None

def agent_head(text, suffix):
    if suffix == ".toml":
        before, marker, _ = text.partition("\ndeveloper_instructions = ")
        return before + marker
    found = re.match(r"---\n(.*?)\n---\n", text, re.S)
    return found.group(1) if found else ""

def tool_list(head, key):
    return {t.strip().lower() for t in (agent_field(head, key, ":") or "").split(",")}

AGENT_HOSTS = {
    "claude-canonical": (canonical / "agents", ".md", ":", ("name", "description"),
                         lambda h: agent_field(h, "permissionMode", ":") != "plan"),
    "claude-generated": (root / "pack/generated/claude/agents", ".md", ":", ("name", "description"),
                         lambda h: agent_field(h, "permissionMode", ":") != "plan"),
    "codex": (root / "pack/generated/codex/agents", ".toml", r"\s*=",
              ("name", "description", "developer_instructions", "default_permissions"),
              lambda h: agent_field(h, "default_permissions", r"\s*=") == ":workspace"),
    "devin": (root / "pack/generated/devin/agents", ".md", ":", ("name", "description", "allowed-tools"),
              lambda h: bool(set(re.findall(r"(?m)^  - ([a-z_]+)$", h)) & {"edit", "write"})),
    "omp": (root / "pack/generated/omp/agents", ".md", ":", ("name", "description", "tools"),
            lambda h: bool(tool_list(h, "tools") & {"edit", "write"})),
    "pi": (root / "pack/generated/pi/agents", ".md", ":", ("name", "description", "tools"),
           lambda h: bool(tool_list(h, "tools") & {"edit", "write"})),
}
shipped = {p.stem for p in (canonical / "agents").glob("*.md")}
writers = {
    p.stem for p in AGENT_HOSTS["codex"][0].glob("*.toml")
    if agent_field(agent_head(p.read_text(), ".toml"), "default_permissions", r"\s*=") == ":workspace"
}
for host, (directory, suffix, sep, required, is_writer) in AGENT_HOSTS.items():
    stems = {p.stem for p in directory.glob(f"*{suffix}")}
    if stems != shipped:
        raise SystemExit(f"{host} roles differ from shipped: missing={sorted(shipped - stems)}, extra={sorted(stems - shipped)}")
    for role in sorted(shipped):
        head = agent_head((directory / f"{role}{suffix}").read_text(), suffix)
        if agent_field(head, "name", sep) != role:
            raise SystemExit(f"{host} role {role} has the wrong name")
        missing = [k for k in required if agent_field(head, k, sep) is None]
        if missing:
            raise SystemExit(f"{host} role {role} is missing required keys {missing}")
        if is_writer(head) != (role in writers):
            raise SystemExit(f"{host} role {role} writer-ness disagrees with the Codex profiles (codex writer: {role in writers})")

# Agent input contracts must not forbid what the engine read-set hands them:
# recorded verdicts/rationale there are claims to re-verify, never "unseen".
for name, phrase in (
    ("agents/devrites-code-reviewer.md", "in your read-set are claims: rerun or re-verify them before relying on them"),
    ("agents/devrites-spec-reviewer.md", "A verdict recorded in a read-set artifact"),
    ("agents/devrites-doubt-reviewer.md", "Rationale or verdicts recorded in read-set artifacts"),
    ("skills/rite-temper/SKILL.md", "with **only** its `--role strategy-reviewer` read-set"),
):
    if phrase not in " ".join((canonical / name).read_text().split()):
        raise SystemExit(f"{name}: input contract contradicts its engine read-set: {phrase!r}")
for name, retired in (
    ("agents/devrites-spec-reviewer.md", "receive spec, diff, and rubric only"),
    ("agents/devrites-doubt-reviewer.md", "only claim + artifact + contract."),
    ("skills/rite-temper/SKILL.md", "with **only** the hardened spec + rubric"),
):
    if retired in " ".join((canonical / name).read_text().split()):
        raise SystemExit(f"{name}: retired read-set-contradicting wording returned: {retired!r}")

profile_text = " ".join(profiles.read_text().split())
for required in (
    "**Quick**",
    "**Standard**",
    "**Full**",
    "exact project specialist",
    "Missing or incompatible exact roles stop for HITL",
    "does not provide an engine broker",
):
    if required not in profile_text:
        raise SystemExit(f"native profile contract missing {required!r}")

for skill in ("rite-quick", "rite-autocomplete", "rite-temper", "rite-vet", "rite-review", "rite-seal"):
    text = (canonical / "skills" / skill / "SKILL.md").read_text()
    if "orchestration-profiles.md" not in text:
        raise SystemExit(f"{skill} does not use the shared execution profiles")

for text, required in (
    (core.read_text(), (
        "exact standalone token in the current invocation arguments",
        "Documentation, examples, prior messages",
        "fails closed before any write or side effect",
    )),
    (authoring.read_text(), (
        "Every public optional-flag skill obeys the shared",
        "narrow explicit-only utility may state the equivalent local guard",
        "complete flag surface in `argument-hint`",
        "fail-closed regression check for value flags",
    )),
    ((canonical / "skills/rite-autocomplete/SKILL.md").read_text(), (
        'argument-hint: "[idea] [--ship|--yolo] [--max-slices N] [--parallel N] [--full] [--cross-model]"',
        "must occur once and be followed by a positive base-10 integer",
        "can never arm them",
        "profile: standard|full",
        "cross_model: yes|no",
        "no sentinel or workspace file has been written",
        "one-write AFK contract",
        "existing sentinel byte-for-byte",
        "leftover `max_parallel: 1`",
        "write or replace only `max_parallel: N`",
        "mutable post-vet budget",
        "performed no Git action before fresh literal `GO` plus native approval",
    )),
    ((canonical / "skills/rite-customize/SKILL.md").read_text(), (
        "`--import-legacy` is active only when that exact standalone token occurs",
        "earlier context cannot activate it",
    )),
    ((canonical / "skills/rite-autocomplete/reference/stop-conditions.md").read_text(), (
        "do not stop on `--max-slices`",
        "pre-existing remaining value, explicit flag, sentinel cap, or post-vet pending count",
        "Zero with no pending slices is normal completion",
        "Blocking with a ranked recommended",
        "escalating always stops",
    )),
    (schema.read_text(), (
        "IDs are append-only identities",
        "next unused numeric suffix",
        "deleted or retired ID remains consumed",
        "A materially different meaning gets a new ID",
        "record the relationship in `decisions.md` or `drift.md`",
        "Current queued questions use `q-YYYY-MM-DD-NNN`",
        "released table registers may retain `Q-001`",
    )),
    (schema.read_text(), (
        "lowercase ASCII kebab-case",
        "at most 64 characters",
        "After the final shortening or suffix step",
        "Never overwrite a workspace by slug alone",
        "accept safe legacy basenames",
    )),
    (state_workspace.read_text(), (
        "Slug identity",
        "mutable runtime state, not `.devrites/AFK` configuration",
        "Ordinary workspace creation preserves the field",
    )),
):
    normalized = " ".join(text.split())
    for phrase in required:
        if phrase not in normalized:
            raise SystemExit(f"native hardening contract missing {phrase!r}")

autocomplete = (canonical / "skills/rite-autocomplete/SKILL.md").read_text()
workflow = re.split(r"(?m)^## Workflow\s*$", autocomplete, maxsplit=1)[1]
workflow = re.split(r"(?m)^## ", workflow, maxsplit=1)[0]
steps = list(re.finditer(r"(?m)^(\d+)\.\s+", workflow))
for index, match in enumerate(steps):
    end = steps[index + 1].start() if index + 1 < len(steps) else len(workflow)
    if "**Completion:**" not in workflow[match.start():end]:
        raise SystemExit(f"rite-autocomplete step {match.group(1)} has no completion criterion")

for path in (canonical / "skills").rglob("*.md"):
    if "bash scripts/validate.sh" in path.read_text():
        raise SystemExit(f"installed skill assumes DevRites source scripts exist: {path}")

for path in (canonical / "skills").glob("rite-*/SKILL.md"):
    text = path.read_text()
    frontmatter = text.split("---", 2)[1]
    hint = next((line for line in frontmatter.splitlines() if line.startswith("argument-hint:")), "")
    if "--" not in hint:
        continue
    if "standards/core.md" not in text and "current `$ARGUMENTS`" not in text:
        raise SystemExit(f"{path}: optional flags bypass the literal-invocation contract")

doctor = (canonical / "skills/rite-doctor/SKILL.md").read_text()
for required in (
    "repository root",
    "manifest",
    "package",
    "symlink",
    "Emit every check as `OK`, `WARN`, or `FAIL`",
    "one concrete `Remediation:`",
):
    if required not in " ".join(doctor.split()):
        raise SystemExit(f"native rite-doctor contract missing {required!r}")

settings = json.loads((canonical / "settings.json").read_text())
allowed = set(settings["permissions"]["allow"])
expected = {
    "Bash(devrites-engine check candidate *)",
    "Bash(devrites-engine check readiness *)",
    "Bash(devrites-engine check seal *)",
    "Bash(devrites-engine check diff-scope *)",
    "Bash(devrites-engine check task-graph *)",
    "Bash(devrites-engine check path-disjoint *)",
    "Bash(devrites-engine check indexes *)",
    "Bash(devrites-engine check skill-trust *)",
    "Bash(devrites-engine check slice *)",
    "Bash(devrites-engine check regression *)",
    "Bash(devrites-engine check windows *)",
    "Bash(devrites-engine check dup *)",
    "Bash(devrites-engine check drift *)",
    "Bash(devrites-engine orient *)",
    "Bash(devrites-engine next *)",
    "Bash(devrites-engine handoff *)",
    "Bash(devrites-engine context *)",
    "Bash(devrites-engine metrics *)",
    "Bash(devrites-engine dispatch *)",
    "Bash(devrites-engine claim *)",
    "Bash(devrites-engine note *)",
    "Bash(devrites-engine observe *)",
    "Bash(devrites-engine detect *)",
    "Bash(devrites-engine migrate *)",
    "Bash(devrites-engine parallel status *)",
    "Bash(devrites-engine parallel select *)",
    "Bash(devrites-engine parallel create *)",
    "Bash(devrites-engine parallel record-green *)",
    "Bash(devrites-engine parallel integrate *)",
    "Bash(devrites-engine parallel abort *)",
    "Bash(devrites-engine parallel cleanup *)",
    "Bash(devrites-engine parallel lease-read *)",
    "Bash(devrites-engine parallel lease-write *)",
    "Bash(devrites-engine parallel lease-clear *)",
    "Bash(devrites-engine state resolve *)",
    "Bash(devrites-engine state merge-manifest *)",
    "Bash(devrites-engine state close *)",
    "Bash(devrites-engine gates scaffold *)",
    "Bash(devrites-engine gates status *)",
    "Bash(devrites-engine gates run *)",
    "Bash(devrites-engine gates reverify *)",
    "Bash(devrites-engine gates lint *)",
    "Bash(devrites-engine gates attest *)",
    "Bash(devrites-engine gates abandon *)",
    "mcp__codegraph__*",
    "mcp__codebase-memory-mcp__*",
    "mcp__codebase-memory__*",
    "mcp__code-review-graph__*",
    "mcp__graphify__*",
    "Bash(codegraph *)",
    "Bash(graphify *)",
    "Bash(git status *)",
    "Bash(git diff *)",
    "Bash(git log *)",
    "Bash(git show *)",
    "Bash(git rev-parse *)",
    "Bash(git merge-base *)",
    "Edit(.devrites/**)",
    "Write(.devrites/**)",
    "Bash(devrites-engine secret-scan)",
    "Bash(devrites-engine secret-scan *)",
    "Bash(devrites-engine open-visual *)",
    "Bash(devrites-engine version)",
}
if allowed != expected:
    raise SystemExit(f"engine allowlist drift: missing={expected-allowed}, extra={allowed-expected}")
command_inventory = (canonical / "skills/devrites-lib/SKILL.md").read_text()
for command in ("`check readiness`", "`check candidate`", "`check seal`"):
    if command not in command_inventory:
        raise SystemExit(f"shared engine command inventory omits {command}")

if not candidate_integrity.is_file():
    raise SystemExit("canonical candidate-integrity lifecycle owner is missing")

lifecycle_owners = {
    "build": canonical / "skills/rite-build/reference/phase-contract.md",
    "prove": canonical / "skills/rite-prove/SKILL.md",
    "polish": canonical / "skills/rite-polish/SKILL.md",
    "review": canonical / "skills/rite-review/SKILL.md",
    "seal": canonical / "skills/rite-seal/SKILL.md",
    "ship": canonical / "skills/rite-ship/SKILL.md",
}
for phase, path in lifecycle_owners.items():
    text = path.read_text()
    if "candidate-integrity.md" not in text:
        raise SystemExit(f"{phase} does not link the shared candidate lifecycle")
    if "| State | File | Slice | Reason |" in text:
        raise SystemExit(f"{phase} duplicates the manifest grammar instead of linking its owner")

candidate_text = " ".join(candidate_integrity.read_text().split())
for phrase in (
    "workspace-artifact-schema.md",
    "binding grammar remain in",
    "Build maintains",
    "Prove binds",
    "Polish closes",
    "Review binds",
    "Seal binds",
    "Ship is candidate-read-only",
    "refresh the manifest and rerun real proof",
    "Never synthesize a historical pass",
    "no legacy fallback",
):
    if phrase not in candidate_text:
        raise SystemExit(f"candidate lifecycle contract missing {phrase!r}")
if "Candidate SHA-256" in candidate_integrity.read_text():
    raise SystemExit("candidate lifecycle must defer the digest grammar to the schema")

schema_text = " ".join(schema.read_text().split())
for phrase in (
    "exactly one `## Touched files`",
    "exactly one authoritative `## Candidate manifest`",
    "`No project files.`",
    "| State | File | Slice | Reason |",
    "| --- | --- | --- | --- |",
    "`present` or `deleted`",
    "one backtick pair",
    "sorted by File",
    "Workspace and audit artifacts are not candidate paths",
    "`.devrites/specs/**`, `DESIGN.md`, and `docs/adr/**`",
    "Engine owns malformed path, type, and size rejection",
    "each contain exactly one unindented standalone",
    "`browser-evidence.md` does too when that file exists",
):
    if phrase not in schema_text:
        raise SystemExit(f"candidate manifest schema missing {phrase!r}")
if schema.read_text().count("Candidate SHA-256: <64 lowercase hex>") != 1:
    raise SystemExit("workspace schema must own one exact digest binding grammar")

polish = lifecycle_owners["polish"].read_text()
new_rollups = tuple(canonical / "skills/rite-polish/reference" / name for name in (
    "ledger.md", "design-memory.md", "adr-promotion.md",
))
old_rollups = tuple(canonical / "skills/rite-ship/reference" / name for name in (
    "ledger.md", "design-memory.md", "adr-promotion.md",
))
if any(not path.is_file() for path in new_rollups):
    raise SystemExit("Polish does not own all three candidate rollup references")
if any(path.exists() for path in old_rollups):
    raise SystemExit("obsolete Ship rollup reference still exists")
for path in new_rollups:
    rollup_text = " ".join(path.read_text().split())
    for phrase in ("candidate manifest", "before Review", "affected real re-proof"):
        if phrase not in rollup_text:
            raise SystemExit(f"{path.name} is not candidate-bound before Review: missing {phrase!r}")
for phrase in (
    "reference/ledger.md",
    "reference/design-memory.md",
    "reference/adr-promotion.md",
    "before Review",
    "candidate manifest",
    "affected real re-proof",
):
    if phrase not in polish:
        raise SystemExit(f"pre-Review rollup contract missing {phrase!r}")
if "proven, not aspirational" not in (canonical / "skills/rite-polish/reference/design-memory.md").read_text():
    raise SystemExit("design memory does not require proven, non-aspirational entries")

frontend_trigger = next(
    (
        line for line in command_map.read_text().splitlines()
        if line.startswith("| Frontend/UI detected")
    ),
    "",
)
if "optional **design-memory** rollup → project `DESIGN.md` in Polish before Review" not in frontend_trigger:
    raise SystemExit("top-level command map does not assign DESIGN.md rollup to Polish before Review")
if re.search(r"DESIGN\.md.*\bship\b", frontend_trigger, re.I):
    raise SystemExit("top-level command map still assigns DESIGN.md rollup to Ship")

# docs/command-map.md: each cross-plane claim must appear verbatim, whitespace-folded.
# Numbers and names inside a claim come from the engine, the agent frontmatter and the
# owning skills, so a wrong value or a reworded claim fails; a re-wrapped line does not.
map_raw = command_map.read_text()
map_flat = " ".join(map_raw.split())
engine_main = (root / "engine/main.go").read_text()


def pin(sentence):
    if sentence not in map_flat:
        raise SystemExit(f"command map no longer states: {sentence!r}")


def pin_source(rel, sentence):
    if sentence not in " ".join((canonical / rel).read_text().split()):
        raise SystemExit(f"{rel} no longer states {sentence!r}, which command-map.md relies on")


def engine_const(name):
    found = re.search(rf"\b{name}\s*=\s*(\d+)", engine_main)
    if found is None:
        raise SystemExit(f"engine/main.go does not define {name}")
    return int(found.group(1))


producer_fmt = re.search(r'Fprintf\(stdout,\s*"(candidate-[^"]*)"', engine_main)
if producer_fmt is None:
    raise SystemExit("engine/main.go prints no `check candidate` stdout record")
produced_fields = re.findall(r"(candidate-[a-z0-9-]+):", producer_fmt.group(1))
if len(produced_fields) != 2:
    raise SystemExit(f"engine/main.go prints {produced_fields}; command-map.md documents two fields")
pin(
    "There are no legacy operational aliases. `check candidate <slug>` prints exactly "
    f"`{produced_fields[0]}: <64 lowercase hex>` and `{produced_fields[1]}: <row count>` on a pass; "
    f"usage/root errors exit `{engine_const('exitUsage')}` and candidate blocks exit `{engine_const('exitBlocked')}`."
)

ones = (
    "zero one two three four five six seven eight nine ten eleven twelve thirteen "
    "fourteen fifteen sixteen seventeen eighteen nineteen"
).split()
tens = "twenty thirty forty fifty sixty seventy eighty ninety".split()


def word_number(word):
    """A count as the command map writes it: digits, or spelled words up to ninety-nine."""
    word = word.lower()
    if word.isdigit():
        return int(word)
    if word in ones:
        return ones.index(word)
    head, _, tail = word.partition("-")
    if head in tens and (not tail or tail in ones):
        return (tens.index(head) + 2) * 10 + (ones.index(tail) if tail else 0)
    return None


writable_profiles = sorted(
    role for role in claude
    if "permissionMode: acceptEdits" in (canonical / "agents" / f"{role}.md").read_text().split("---", 2)[1]
)
if len(writable_profiles) != 1:
    raise SystemExit(f"command-map.md documents one writable specialist; frontmatter declares {writable_profiles}")
writable = writable_profiles[0]
profile_claim = re.search(
    r"fresh-context leaves\) \*\*([A-Za-z0-9-]+) role profiles:\*\* both hosts have ([A-Za-z0-9-]+) "
    r"read-only leaves plus the write-capable `([a-z-]+)`\.",
    map_flat,
)
if profile_claim is None:
    raise SystemExit("command map no longer states its role profile, read-only leaf and write-capable counts")
if (
    word_number(profile_claim.group(1)),
    word_number(profile_claim.group(2)),
    profile_claim.group(3),
) != (len(claude), len(claude) - 1, writable):
    raise SystemExit(
        f"command map states {profile_claim.group(1)} role profiles, {profile_claim.group(2)} read-only leaves, "
        f"write-capable `{profile_claim.group(3)}`; the agents tree has {len(claude)} profiles, writable {writable_profiles}"
    )
agent_names = [p.stem for p in (canonical / "agents").glob("*.md")]
skills_flat = " ".join((root / "docs/skills.md").read_text().split())
roster_claim = re.search(
    r"DevRites ships ([A-Za-z0-9-]+) role profiles at depth one\. Both hosts have ([A-Za-z0-9-]+) read-only "
    r"leaves and one source/test writer role\. The explicit `/overhaul` skill ships ([A-Za-z0-9-]+) more agents, "
    r"`overhaul-\*`.*?`/rite-fast` adds ([A-Za-z0-9-]+) `fast-\*` agents",
    skills_flat,
)
if roster_claim is None:
    raise SystemExit("docs/skills.md no longer states its role profile, read-only leaf, overhaul-* and fast-* counts")
documented = tuple(word_number(roster_claim.group(i)) for i in (1, 2, 3, 4))
actual = (
    len(claude),
    len(claude) - 1,
    sum(n.startswith("overhaul-") for n in agent_names),
    sum(n.startswith("fast-") for n in agent_names),
)
if documented != actual:
    raise SystemExit(
        "docs/skills.md states (profiles, read-only leaves, overhaul-*, fast-*) = "
        f"{documented}; the agents tree has {actual}"
    )
pin(
    "that root from editing source/tests. "
    f"Among the lifecycle `devrites-*` specialists, both hosts expose only `{writable}` as a writable one. "
    "The shipped profiles are the full role-to-permission map; `/rite-doctor` step 3 checks them."
)

# The Codex profiles ship more than one :workspace writer (overhaul-*, fast-builder), so prose
# that names a sole writer without the lifecycle scope contradicts the shipped artifacts.
codex_writers = [
    p.stem for p in (root / "pack/generated/codex/agents").glob("*.toml")
    if 'default_permissions = ":workspace"' in p.read_text()
]
compliance_flat = " ".join((root / "docs/harness-compliance.md").read_text().split())
for claim in (
    "Claude lifecycle `devrites-*` reviewer profiles use permissionMode plan",
    'Codex lifecycle `devrites-*` reviewer profiles use `default_permissions = ":read-only"`',
    "its root no-source-writing boundary is instruction-enforced",
    "both hosts dispatch the exact slice-wright while every other lifecycle `devrites-*` specialist remains read-only",
):
    if claim not in compliance_flat:
        raise SystemExit(f"docs/harness-compliance.md no longer states: {claim!r}")
if len(codex_writers) > 1 and "while every other specialist remains read-only" in compliance_flat:
    raise SystemExit("docs/harness-compliance.md: retired unscoped read-only wording returned")

if len(codex_writers) > 1:
    for rel, retired, scoped in (
        (
            "pack/generated/codex/AGENTS.md",
            'every other specialist uses `default_permissions = ":read-only"`',
            f"Among the lifecycle `devrites-*` specialists, `{writable}` alone uses",
        ),
        (
            "docs/command-map.md",
            f"Both hosts expose only `{writable}` as a writable specialist.",
            f"Among the lifecycle `devrites-*` specialists, both hosts expose only `{writable}`",
        ),
        (
            "README.md",
            f"On every host, `{writable}` is the only writable specialist",
            f"among the lifecycle `devrites-*` specialists only `{writable}` is writable",
        ),
        (
            "pack/generated/devin/AGENTS.md",
            f"Only `{writable}` may edit source or tests; every other specialist is read-only by `allowed-tools`.",
            f"Among the lifecycle `devrites-*` specialists, only `{writable}` may edit source or tests",
        ),
        (
            "pack/generated/pi/AGENTS.md",
            f"Only `{writable}` may edit source or tests; every other specialist is read-only by tool allowlist.",
            f"Among the lifecycle `devrites-*` specialists, only `{writable}` may edit source or tests",
        ),
        (
            "SECURITY.md",
            f"On both hosts, `{writable}` is the only writable specialist.",
            f"Among the lifecycle `devrites-*` specialists, `{writable}` is the only writable one",
        ),
    ):
        flat = " ".join((root / rel).read_text().split())
        if retired in flat:
            raise SystemExit(f"{rel}: retired sole-writer wording returned: {retired!r}")
        if scoped not in flat:
            raise SystemExit(f"{rel}: sole-writer claim lost its lifecycle scope: {scoped!r}")
        if "`/rite-doctor` step 3" not in flat:
            raise SystemExit(f"{rel}: role-to-permission claim lost its `/rite-doctor` step 3 pointer")

pin_source(
    "skills/rite-seal/SKILL.md",
    "dispatch remaining required exact roles under `../devrites-lib/reference/parallel-dispatch.md`",
)
pin_source("skills/devrites-lib/reference/parallel-dispatch.md", "launch every native agent nonblocking")
pin_source("skills/devrites-lib/reference/parallel-dispatch.md", "Ask for each exact named agent in fresh context")
pin(
    "- `/rite-seal` fans out to `.claude/agents/devrites-*` reviewers **in parallel** "
    "for independent, fresh-context judgment, then writes the GO / NO-GO verdict: it runs no git."
)

upgrade_row = [l for l in map_raw.splitlines() if l.startswith("| [`/rite-upgrade`]")]
if len(upgrade_row) != 1 or not upgrade_row[0].startswith(
    "| [`/rite-upgrade`](../pack/.claude/skills/rite-upgrade/SKILL.md) | compatibility | `[slug]` |"
):
    raise SystemExit(f"command map needs exactly one /rite-upgrade row linking its skill, found {upgrade_row}")
if not (canonical / "skills/rite-upgrade/SKILL.md").is_file():
    raise SystemExit("command map links a /rite-upgrade skill that does not exist")
pin(
    "**Conditional recovery.** Audit an older released workspace against current contracts. "
    "Only cited defects route through Clarify, Plan repair, Converge, Vet, Prove, Polish, Review, or Seal. "
    "Ambiguous candidate scope is a gap; age/cursor form alone is never a defect, "
    "and old passes are never synthesized."
)
pin(
    "`/rite-upgrade` separately audits an older active workspace. Only a cited "
    "current-contract defect may route a repair through its existing Clarify, Plan repair, "
    "Converge, Vet, Prove, Polish, Review, or Seal owner."
)
pin_source("agents/devrites-upgrade-planner.md", "ambiguous candidate scope produces `gap`")
pin_source("skills/rite-upgrade/SKILL.md", "cursor form, or pack version alone is never a defect")
pin_source("skills/rite-upgrade/SKILL.md", "never synthesize or guess scope, bytes, proof, freshness, or a historical pass")

pin_source("skills/rite-autocomplete/SKILL.md", "it never authorizes Git.")
pin(
    "/rite-autocomplete drives the reversible sequence unattended; --ship reaches "
    "only the exact-plan Ship approval boundary and never authorizes Git."
)

ship = " ".join(lifecycle_owners["ship"].read_text().split())
for phrase in (
    "candidate-read-only",
    "must not change any candidate path or `touched-files.md`",
    "Prove → Review → Seal",
    "may write only workspace `ship.md`, `state.md`, and archive bookkeeping",
    "devrites-engine check seal <slug>",
    "Prepare the exact Git candidate",
):
    if phrase not in ship:
        raise SystemExit(f"Ship candidate-read-only contract missing {phrase!r}")
for stale_reference in ("reference/ledger.md", "reference/design-memory.md", "reference/adr-promotion.md"):
    if stale_reference in ship:
        raise SystemExit(f"Ship still owns project mutation through {stale_reference}")
if ship.index("devrites-engine check seal <slug>") > ship.index("Prepare the exact Git candidate"):
    raise SystemExit("Ship does not recheck Seal before Git preparation")

git_ship_raw = (canonical / "skills/rite-ship/reference/git-ship.md").read_text()
git_ship = " ".join(git_ship_raw.split())
for phrase in (
    "pre-existing staged path outside the manifest",
    "git add --",
    "git diff --cached --name-status --no-renames -z",
    "`present` maps only to `A` or `M`; `deleted` maps only to `D`",
    "no index-to-worktree difference for any manifest path",
    "devrites-engine check candidate",
    "devrites-engine check seal",
    "exact candidate digest",
    "staged secret scan",
    "after the authorized commit and before any push or tag",
    "git diff-tree --root --no-commit-id --name-status --no-renames -r -z HEAD",
    "candidate paths still match `HEAD`",
    "Any mismatch stops; do not reinterpret it",
    "Push: pending",
    "git ls-remote",
    "HEAD equals the recorded",
    "fresh type-GO for only the remaining push/tag/PR commands",
):
    if phrase not in git_ship:
        raise SystemExit(f"Ship Git integrity contract missing {phrase!r}")

before_marker = "### Before type-GO"
after_marker = "### After type-GO"
before_start = git_ship_raw.index(before_marker)
after_start = git_ship_raw.index(after_marker)
prompt = git_ship_raw[:before_start]
before_go = git_ship_raw[before_start:after_start]
after_go = git_ship_raw[after_start:]
after_go_normalized = " ".join(after_go.split())
for phrase in (
    "Checkpoint collapse:",
    "Stage plan:",
):
    if phrase not in prompt:
        raise SystemExit(f"Ship literal approval prompt missing {phrase!r}")
if prompt.count("git reset --soft") != 1 or prompt.count("git add --") != 1:
    raise SystemExit("Ship literal approval prompt does not disclose exactly one collapse/stage command form")
for phrase in (
    "Pre-GO is read-only",
    "Analyze checkpoint history without mutating it",
    "Do not run the disclosed collapse or staging commands",
):
    if phrase not in before_go:
        raise SystemExit(f"Ship pre-GO contract missing {phrase!r}")
if "git reset --soft" in before_go or "git add --" in before_go:
    raise SystemExit("Ship pre-GO text contains an executable index/history mutation outside the literal disclosure prompt")
for phrase in (
    "Optional checkpoint collapse",
    "Exact staging",
    "Revalidate immediately before commit",
    "invalidates the one-use approval",
    "fresh type-GO prompt",
    "WIP(<slug>):",
):
    if phrase not in after_go_normalized:
        raise SystemExit(f"Ship post-GO contract missing {phrase!r}")
post_order = (
    "Optional checkpoint collapse",
    "Exact staging",
    "Compare staged scope",
    "Compare staged bytes",
    "Revalidate immediately before commit",
    "staged secret scan",
    "**Commit**",
)
positions = [after_go_normalized.index(phrase) for phrase in post_order]
if positions != sorted(positions):
    raise SystemExit("Ship mutates or commits before the required post-GO revalidation order")
resume_order = (
    "**Commit**",
    "Verify the commit",
    "**Record before push.**",
    "**Push**",
)
resume_positions = [after_go_normalized.index(phrase) for phrase in resume_order]
if resume_positions != sorted(resume_positions):
    raise SystemExit("Ship records push state outside the commit-verify-record-push order")

binding = "Candidate SHA-256: <64 lowercase hex>"
seal_template = (canonical / "skills/rite-seal/reference/seal-template.md").read_text()
if seal_template.count(binding) != 1:
    raise SystemExit("seal template must contain exactly one candidate digest binding")

spec_grammar = canonical / "skills/devrites-lib/reference/standards/spec-grammar.md"
spec_template = canonical / "skills/rite-spec/reference/spec-template.md"
ledger = canonical / "skills/rite-polish/reference/ledger.md"
for path, phrases in (
    (spec_grammar, (
        "Capability impact:",
        "Capability impact: none — <specific justification>",
        "new or materially revised feature spec",
    )),
    (ledger, (
        "complete current requirement block",
        "every existing scenario",
        "normative or source-grounded claim",
        "exact accepted `DEC-###`",
        "unexplained omission is a blocking conflict",
    )),
):
    normalized = " ".join(path.read_text().split())
    for phrase in phrases:
        if phrase not in normalized:
            raise SystemExit(f"semantic preservation contract missing {phrase!r} in {path}")
if len(re.findall(r"(?m)^Capability impact:", spec_template.read_text())) != 1:
    raise SystemExit("spec template must contain exactly one capability-impact declaration")
# spec.md is in every reviewer read-set: phase verdicts live in the state.md cursor,
# never in the product contract, or reviewers reject the spec as seeded.
spec_body = spec_template.read_text().split("```markdown", 1)[1].split("\n```", 1)[0]
verdicts = re.findall(r"(?m)^(Status:|Spec gate|## Readiness gate)|spec_gate", spec_body)
if verdicts:
    raise SystemExit(f"spec.md template carries phase verdicts {verdicts!r}; record them in state.md")
for name, phrase in (
    ("skills/rite-spec/SKILL.md", "`state.md` `spec_gate`"),
    ("skills/rite-spec/reference/state-workspace.md", "| spec_gate |"),
    ("skills/devrites-lib/reference/standards/agents.md", "status, readiness-checklist, or gate lines"),
):
    if phrase not in " ".join((canonical / name).read_text().split()):
        raise SystemExit(f"{name}: spec gate home contract missing {phrase!r}")

plan_template = canonical / "skills/rite-define/reference/plan-template.md"
plan_text = " ".join(plan_template.read_text().split())
for phrase in (
    "## Shared contract proof",
    "| Boundary | Canonical contract artifact | Provider-side asserting test | Consumer-side asserting test |",
    "Shared contract impact: none — <specific justification>",
    "consume the same artifact",
    "Reuse an existing canonical",
):
    if phrase not in plan_text:
        raise SystemExit(f"shared-contract plan grammar missing {phrase!r}")

shared_contract_owners = (
    canonical / "skills/rite-define/SKILL.md",
    canonical / "skills/rite-plan/SKILL.md",
    canonical / "skills/rite-plan/reference/task-breakdown.md",
    canonical / "skills/rite-vet/SKILL.md",
    canonical / "skills/rite-vet/reference/artifacts.md",
    canonical / "agents/devrites-plan-drafter.md",
    canonical / "agents/devrites-plan-reviewer.md",
)
for path in shared_contract_owners:
    if "Shared contract proof" not in path.read_text():
        raise SystemExit(f"{path} omits the canonical shared-contract plan section")
for path in (
    canonical / "skills/rite-vet/SKILL.md",
    canonical / "agents/devrites-plan-drafter.md",
    canonical / "agents/devrites-plan-reviewer.md",
):
    normalized = " ".join(path.read_text().split())
    for phrase in ("one-sided", "duplicated-contract", "vague, or non-consuming proof"):
        if phrase not in normalized:
            raise SystemExit(f"{path} does not fail closed on {phrase} shared-contract proof")

testing = canonical / "skills/devrites-lib/reference/standards/testing.md"
testing_text = " ".join(testing.read_text().split())
for phrase in (
    "positive, discriminating evidence",
    "Skipped, focused, filtered, or pending",
    "zero-test",
    "assertion-free",
    "success inferred only from exit status",
    "Build, compile, typecheck, and lint",
    "Explicit shell assertions and golden/text comparisons",
):
    if phrase not in testing_text:
        raise SystemExit(f"positive-proof standard missing {phrase!r}")
for path in (
    canonical / "skills/rite-vet/SKILL.md",
    canonical / "skills/rite-prove/SKILL.md",
    canonical / "skills/rite-prove/reference/acceptance-proof.md",
    canonical / "agents/devrites-test-analyst.md",
    canonical / "agents/devrites-proof-runner.md",
    canonical / "skills/rite-seal/reference/final-evidence.md",
):
    if "positive, discriminating" not in " ".join(path.read_text().split()):
        raise SystemExit(f"{path} does not apply the positive-proof rule")

schema_contract = " ".join(schema.read_text().split())
for phrase in (
    "`## Review trail` may cite concern-ordered `path:line` review stops",
    "cannot define or expand candidate scope",
):
    if phrase not in schema_contract:
        raise SystemExit(f"Review-trail scope contract missing {phrase!r}")
if "There is no second human-readable path list." in schema.read_text():
    raise SystemExit("workspace schema still forbids the permitted Review trail")

retired = (
    "snapshot", "readiness", "seal", "spec-validate", "check-acceptance", "evidence-fresh",
    "coverage", "doubt-coverage", "test-integrity", "review-integrity",
    "build-readiness", "readiness-digest", "analyze", "ledger", "resolve",
    "clarify-return", "tick-afk", "recovery", "close-out",
)
for path in [canonical / "settings.json", *(canonical / "skills").rglob("*.md"), *(canonical / "agents").glob("*.md")]:
    text = path.read_text()
    for command in (
        r"check\s+spec",
        r"state\s+clarify",
        r"state\s+tick-afk",
        r"state\s+recovery",
        r"state\s+resolve\s+next-qid",
        r"doctor",
    ):
        if re.search(rf"\bdevrites-engine\s+{command}(?:\s|`|$)", text):
            raise SystemExit(f"{path}: removed engine policy command survives: {command}")
    for command in retired:
        if re.search(rf"\bdevrites-engine\s+{re.escape(command)}(?:\s|`|$)", text):
            raise SystemExit(f"{path}: retired engine command survives: {command}")

spec = (canonical / "skills/devrites-lib/reference/standards/spec-grammar.md").read_text()
checkpoint = (canonical / "skills/rite-build/reference/checkpoint-protocol.md").read_text()
clarify = (canonical / "skills/rite-clarify/SKILL.md").read_text()
afk = (canonical / "skills/rite-build/reference/afk-discipline.md").read_text()
afk_contract = (canonical / "skills/devrites-lib/reference/standards/afk-hitl.md").read_text()
recovery = (canonical / "skills/devrites-debug-recovery/SKILL.md").read_text()
for text, required in (
    (spec, ("Native grammar re-read checklist", "No parser or replacement script")),
    (checkpoint, ("scan every question header", "re-read `questions.md` immediately before", "next unused", "under a bounded tool timeout", "append the exit code to `evidence.md`")),
    (clarify, ("return_phase", "return_next_action", "preserve unrelated Markdown", "/rite-plan repair")),
    (afk_contract, (
        "read-only config", "afk_slices_remaining", "released bullet", "pre-seed",
        "never increased or reinitialized", "max_agents", "max_minutes",
        "max_review_queue", "leftover `expires_at` is ignored", "do not add dispatch telemetry to `.devrites/`",
        "do not consult leftover sentinel `max_parallel`",
        "/rite-autocomplete --parallel N` writes",
        "Above it stop; at it run only reconciliation", "if declared but unobservable, stop",
        "per native activation and start fresh only", "remain durable/recomputed across wakes",
        "`/rite-autocomplete` deliberately ignores `max_slices`, `max_agents`, `max_minutes` and `max_review_queue`",
    )),
    (afk, ("afk-hitl.md", "dispatch, charging, and red-path behavior", "exactly once after each green built slice", "never below zero", "fails closed", "before dispatching another slice", "curl -fsS -m 10 -d")),
    (recovery, ("caller and recovery attempts", "three no-progress attempts", "Count an attempt only", "## Dead ends", "Next: none — technical recovery exhausted")),
    ((canonical / "skills/rite-build/reference/wright-dispatch.md").read_text(), (
        "Isolated writer-worktree pilot", "show-superproject-working-tree",
        "one writer", "transfer_commit", "preserve the worktree and commit",
        "actual `git rev-parse HEAD` equals supplied", "Mismatch returns a gap with no write",
        "Parallel isolated-writer worktrees", "remain forbidden until this serial pilot",
        "successful return, rejected result, stop, or a launch the host refused with no handle",
        "A timeout or unknown outcome keeps the claim",
        "](../../devrites-lib/reference/parallel-dispatch.md#cancellation-and-terminal-reconciliation)",
    )),
    ((canonical / "skills/rite-build/reference/parallel-batch.md").read_text(), (
        "stop for the human (never stash or commit their work); the batch is untouched",
    )),
):
    normalized = " ".join(text.split())
    for phrase in required:
        if phrase not in normalized:
            raise SystemExit(f"native policy contract missing {phrase!r}")

if "notify: 'curl " not in afk_contract or 'notify: "ntfy.sh' in afk_contract:
    raise SystemExit("afk-hitl.md sentinel notify example must be a runnable single-quoted shell command")

build = (canonical / "skills/rite-build/reference/phase-contract.md").read_text()
seal = (canonical / "skills/rite-seal/reference/phase-contract.md").read_text()
define = (canonical / "skills/rite-define/SKILL.md").read_text()
for text, required in (
    (build, ("devrites-engine check readiness", "devrites-test-analyst", "test hunks for deletion, skipping/focus, tautology, or weaker expectations")),
    (seal, ("devrites-proof-runner", "devrites-spec-reviewer", "by ID and meaning")),
    (define, ("Persist traceability natively", "traceability.md")),
):
    normalized = " ".join(text.split())
    for phrase in required:
        if phrase not in normalized:
            raise SystemExit(f"native semantic ownership missing {phrase!r}")

build_mode_contracts = {
    canonical / "skills/rite-build/SKILL.md": (
        "HITL stops; a later user invocation starts the next",
        "Explicit `.devrites/AFK` alone lets the controlling root chain",
        "Every wright returns after one slice",
    ),
    canonical / "skills/rite-build/reference/output.md": (
        "Only explicit `.devrites/AFK` lets the controlling root chain",
        "Every `devrites-slice-wright` returns after exactly one slice",
        "Build never enters `/rite-prove` automatically",
    ),
    canonical / "skills/rite-build/reference/one-slice-cycle.md": (
        "HITL stops; only explicit `.devrites/AFK` lets the controlling root chain",
        "Each wright returns after exactly one slice",
    ),
    canonical / "skills/rite-build/reference/anti-patterns.md": (
        "HITL needs a later user invocation",
        "Explicit `.devrites/AFK` may let the root chain",
        "each wright still returns after one",
    ),
    canonical / "skills/rite/SKILL.md": (
        "one slice/wright; HITL stops, `.devrites/AFK` may chain",
    ),
}
for path, required in build_mode_contracts.items():
    text = " ".join(path.read_text().split())
    for phrase in required:
        if phrase not in text:
            raise SystemExit(f"{path}: Build HITL/AFK contract missing {phrase!r}")

retired_build_phrases = {
    canonical / "skills/rite-build/SKILL.md": (
        "Not for multiple slices.",
        "Build and prove one slice, then **stop**",
        "**One slice at a time. DO NOT** start the next slice without the user asking.",
    ),
    canonical / "skills/rite-build/reference/output.md": (
        "**DO NOT continue to the next slice automatically**",
    ),
    canonical / "skills/rite-build/reference/one-slice-cycle.md": (
        "STOP → report + recommend next; do not start slice N+1",
        "## Why stop after one slice",
    ),
    canonical / "skills/rite-build/reference/anti-patterns.md": (
        "One slice, then stop. Full stop. The user asks for the next.",
        "About to start slice N+1 without the user asking.",
    ),
    canonical / "skills/rite/SKILL.md": (
        "implement exactly one verified vertical slice, then stop",
    ),
    canonical / "skills/rite-build/reference/parallel-batch.md": (
        "commit/stash and retry",
    ),
}
for path, retired in retired_build_phrases.items():
    text = " ".join(path.read_text().split())
    for phrase in retired:
        if phrase in text:
            raise SystemExit(f"{path}: retired unconditional Build wording returned: {phrase!r}")

# A failed read-only dispatch is a retryable transport gap, never a consumptive action.
agents_text = " ".join(agents_doc.read_text().split())
for phrase in (
    "**Transport failure is not a finding.**",
    "Re-dispatch one fresh child with the same brief and read-set — never a resume of the failed child.",
    "A second transport failure of the same role is a host defect: stop for HITL",
):
    if phrase not in agents_text:
        raise SystemExit(f"{agents_doc}: missing transport-failure retry rule: {phrase!r}")
# Verdicts recorded in role read-set artifacts are inputs, not packet seeding.
for phrase in (
    "are inputs, not seeding",
    "a verdict they record is a historical claim the reviewer re-verifies",
    "Never hand-edit installed read-sets",
):
    if phrase not in agents_text:
        raise SystemExit(f"{agents_doc}: missing read-set independence rule: {phrase!r}")
# A claim is released only on a host-confirmed terminal path; a timeout keeps it.
for phrase in (
    "host-confirmed terminal path after `add`",
    "](../parallel-dispatch.md#cancellation-and-terminal-reconciliation)",
):
    if phrase not in agents_text:
        raise SystemExit(f"{agents_doc}: missing claim-release terminal rule: {phrase!r}")
one_shot = canonical / "skills/devrites-lib/reference/standards/one-shot-actions.md"
one_shot_text = " ".join(one_shot.read_text().split())
if "needs fresh human authorization" in one_shot_text:
    raise SystemExit(f"{one_shot}: approval limits must not define a consumptive action")
if "Read-only `devrites-*` dispatches (scout, drafter, reviewers, proof runner) and local offline checks are never consumptive." not in one_shot_text:
    raise SystemExit(f"{one_shot}: missing read-only dispatch exclusion")
# Seeding means the dispatch instructions, not the engine-generated read-set.
admission = canonical / "agents/_shared/result-admission.md"
admission_text = " ".join(admission.read_text().split())
for phrase in (
    "in the dispatch instructions voids it",
    "Workspace artifacts in the engine-generated read-set are inputs, not seeding",
):
    if phrase not in admission_text:
        raise SystemExit(f"{admission}: missing read-set seeding boundary: {phrase!r}")
# rite-temper --mode tokens: the argument-hint must name only modes that the artifact
# template's `Mode:` grammar and the skill's own step 2 accept.
temper = canonical / "skills/rite-temper"
temper_skill = (temper / "SKILL.md").read_text()
hint_line = next(l for l in temper_skill.split("---", 2)[1].splitlines() if l.startswith("argument-hint:"))
hint_modes = set(re.search(r"--mode ([^\]\"]+)", hint_line).group(1).split("|"))
template_modes = set(re.search(r"^Mode: (.+)$", (temper / "reference/strategy-template.md").read_text(), re.M).group(1).split(" | "))
step_two = " ".join(temper_skill.split())
step_two_modes = set(re.search(r"scope mode \((.+?)\) with its", step_two).group(1).replace("`", "").replace(" opt-in", "").split(" · "))
if not hint_modes or not hint_modes <= template_modes or not hint_modes <= step_two_modes:
    raise SystemExit(
        f"rite-temper --mode hint {sorted(hint_modes)} is not a subset of strategy-template Mode: "
        f"{sorted(template_modes)} and SKILL.md step 2 {sorted(step_two_modes)}"
    )
menu_row = next(l for l in (canonical / "skills/rite/reference/menu.md").read_text().splitlines() if "`/rite-temper`" in l)
menu_modes = set(re.search(r"scope mode \(([^)]+)\)", menu_row).group(1).split("/"))
if not menu_modes <= template_modes:
    raise SystemExit(f"menu.md rite-temper modes {sorted(menu_modes)} are not strategy-template Mode: values {sorted(template_modes)}")
PY

echo "native orchestration contract: PASS"

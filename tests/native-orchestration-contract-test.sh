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

if len(documented) != 17:
    raise SystemExit(f"agent catalog contains {len(documented)} roles, want 17")
if claude != documented:
    raise SystemExit(f"Claude profiles differ from catalog: missing={documented-claude}, extra={claude-documented}")
if codex != documented:
    raise SystemExit(f"Codex profiles differ from catalog: missing={documented-codex}, extra={codex-documented}")

for role in sorted(documented):
    claude_text = (canonical / "agents" / f"{role}.md").read_text()
    codex_text = (root / "pack/generated/codex/agents" / f"{role}.toml").read_text()
    claude_mode = "acceptEdits" if role == "devrites-slice-wright" else "plan"
    codex_mode = ":workspace" if role == "devrites-slice-wright" else ":read-only"
    if f"name: {role}" not in claude_text or f"permissionMode: {claude_mode}" not in claude_text:
        raise SystemExit(f"Claude role {role} has the wrong identity or permission mode")
    if f'name = "{role}"' not in codex_text or f'default_permissions = "{codex_mode}"' not in codex_text:
        raise SystemExit(f"Codex role {role} has the wrong identity or permission mode")

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
        'argument-hint: "[idea] [--ship|--yolo] [--max-slices N] [--full] [--cross-model]"',
        "must occur once and be followed by a positive base-10 integer",
        "can never arm them",
        "profile: standard|full",
        "cross_model: yes|no",
        "no sentinel or workspace file has been written",
        "one-write AFK contract",
        "existing sentinel byte-for-byte",
        "mutable post-vet budget",
        "performed no Git action before fresh literal `GO` plus native approval",
    )),
    ((canonical / "skills/rite-customize/SKILL.md").read_text(), (
        "`--import-legacy` is active only when that exact standalone token occurs",
        "earlier context cannot activate it",
    )),
    ((canonical / "skills/rite-autocomplete/reference/stop-conditions.md").read_text(), (
        "any validated root-owned remaining value of zero stops before the next dispatch",
        "pre-existing remaining value, explicit flag, sentinel cap, or post-vet pending count",
        "Zero with no pending slices is normal completion",
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
    "OK",
    "WARN",
    "FAIL",
    "Remediation",
):
    if required not in doctor:
        raise SystemExit(f"native rite-doctor contract missing {required!r}")

settings = json.loads((canonical / "settings.json").read_text())
allowed = set(settings["permissions"]["allow"])
expected = {
    "Bash(devrites-engine check candidate *)",
    "Bash(devrites-engine check readiness *)",
    "Bash(devrites-engine check seal *)",
    "Bash(devrites-engine state resolve *)",
    "Bash(devrites-engine state close *)",
    "Bash(devrites-engine secret-scan --staged)",
    "Bash(devrites-engine secret-scan --stdin)",
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
    "Build maintains",
    "Prove binds",
    "Polish closes",
    "Review binds",
    "Seal binds",
    "Ship is candidate-read-only",
    "Candidate SHA-256: <64 lowercase hex>",
    "exactly once in `evidence.md`, `review.md`, and `seal.md`",
    "`browser-evidence.md` when that file exists",
    "refresh the manifest and rerun real proof",
    "Never synthesize a historical pass",
    "no legacy fallback",
):
    if phrase not in candidate_text:
        raise SystemExit(f"candidate lifecycle contract missing {phrase!r}")
if candidate_integrity.read_text().count("Candidate SHA-256: <64 lowercase hex>") != 1:
    raise SystemExit("candidate lifecycle must state the exact digest binding once")

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
    for phrase in ("one-sided", "duplicated-contract", "vague", "non-consuming"):
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
    "clarify-return", "tick-afk", "recovery", "close-out", "migrate",
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
    (checkpoint, ("scan every question header", "re-read `questions.md` immediately before", "next unused")),
    (clarify, ("return_phase", "return_next_action", "preserve unrelated Markdown", "/rite-plan repair")),
    (afk_contract, ("read-only config", "afk_slices_remaining", "released bullet", "pre-seed", "never increased or reinitialized")),
    (afk, ("afk-hitl.md", "dispatch, charging, and red-path behavior", "exactly once after each green built slice", "never below zero", "fails closed", "before dispatching another slice")),
    (recovery, ("caller and recovery attempts", "three total failed attempts", "## Dead ends", "never make a fourth")),
):
    for phrase in required:
        if phrase not in text:
            raise SystemExit(f"native policy contract missing {phrase!r}")

build = (canonical / "skills/rite-build/reference/phase-contract.md").read_text()
seal = (canonical / "skills/rite-seal/reference/phase-contract.md").read_text()
define = (canonical / "skills/rite-define/SKILL.md").read_text()
for text, required in (
    (build, ("devrites-engine check readiness", "devrites-test-analyst", "test hunks for deletion, skipping/focus, tautology, or weaker expectations")),
    (seal, ("devrites-proof-runner", "devrites-spec-reviewer", "by ID and meaning")),
    (define, ("Persist traceability natively", "traceability.md")),
):
    for phrase in required:
        if phrase not in text:
            raise SystemExit(f"native semantic ownership missing {phrase!r}")

build_mode_contracts = {
    canonical / "skills/rite-build/SKILL.md": (
        "HITL stops; a later user invocation starts the next",
        "Explicit `.devrites/AFK` alone lets the controlling root chain",
        "Every wright returns after it",
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
}
for path, retired in retired_build_phrases.items():
    text = " ".join(path.read_text().split())
    for phrase in retired:
        if phrase in text:
            raise SystemExit(f"{path}: retired unconditional Build wording returned: {phrase!r}")
PY

echo "native orchestration contract: PASS"

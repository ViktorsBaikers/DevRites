#!/usr/bin/env bash
set -euo pipefail
SCRIPT_PATH="$(cd "$(dirname "$0")" && pwd -P)/$(basename "$0")"
exec python3 - "$SCRIPT_PATH" "$@" <<'PY'
from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor
import contextlib
import errno
import fcntl
import hashlib
import json
import os
import re
import selectors
import shutil
import signal
import socket
import stat
import subprocess
import sys
import tempfile
import time
from pathlib import Path

SCRIPT = Path(sys.argv[1]).resolve()
ARGS = sys.argv[2:]
CANDIDATE_ROOT = SCRIPT.parent.parent


def delivery_execution_prefix() -> list[str]:
    """Prefer rtk proxy when available; CI runners may not install rtk."""
    return ["rtk", "proxy"] if shutil.which("rtk") else []


def with_delivery_execution_prefix(command: list[str]) -> list[str]:
    return [*delivery_execution_prefix(), *command]
AUTHORED = [
    "pack/.claude/skills/devrites-lib/reference/standards/workflow-artifacts.md",
    "pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md",
    "pack/.claude/skills/devrites-lib/reference/standards/one-shot-actions.md",
    "pack/.claude/skills/devrites-debug-recovery/SKILL.md",
    "pack/.claude/skills/rite-autocomplete/SKILL.md",
    "pack/.claude/skills/rite-autocomplete/reference/loop.md",
    "pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md",
    "pack/.claude/skills/rite-build/SKILL.md",
    "pack/.claude/skills/rite-build/reference/phase-contract.md",
    "pack/.claude/skills/rite-prove/SKILL.md",
    "pack/.claude/skills/rite-vet/SKILL.md",
    "evals/behavioral/workflow-artifact-identity.json",
    "tests/workflow-artifact-identity-test.sh",
    "tests/phase-gate-routing-test.sh",
    "tests/host-artifacts-test.sh",
    "tests/instruction-size-baseline.json",
]
GENERATED = [
    f"pack/generated/{host}/{path}"
    for host in ("claude", "codex")
    for path in (
        "skills/devrites-lib/reference/standards/workflow-artifacts.md",
        "skills/devrites-lib/reference/standards/afk-hitl.md",
        "skills/devrites-lib/reference/standards/one-shot-actions.md",
        "skills/devrites-debug-recovery/SKILL.md",
        "skills/rite-autocomplete/SKILL.md",
        "skills/rite-autocomplete/reference/loop.md",
        "skills/rite-autocomplete/reference/stop-conditions.md",
        "skills/rite-build/SKILL.md",
        "skills/rite-build/reference/phase-contract.md",
        "skills/rite-prove/SKILL.md",
        "skills/rite-vet/SKILL.md",
    )
]
ALL_DESTINATIONS = set(AUTHORED + GENERATED)
PROTECTED = [
    ".gitignore", ".devrites/ACTIVE",
    ".devrites/work/workspace-observation/touched-files.md",
]
WRITER_ALLOWLIST_SHA256 = "dc63a67ea39a0dc63b8292c43668efc2f21403ceae04e1c133ff9eb178f405c5"
MODULE_REL = AUTHORED[0]
CORPUS_REL = AUTHORED[11]
START = "<!-- devrites-workflow-artifact-journal:start -->"
END = "<!-- devrites-workflow-artifact-journal:end -->"
OUTSIDE_MANIFEST_CONTRACT = (
    "Delivery's one immutable transaction-private `outside-manifest.json` sidecar.\n"
    'Journal binds only exact relative name, SHA-256, encoded bytes, and row count; no\n'
    'generation duplicates payload. Descriptor-stable records: directory/file/symlink type/mode/uid/gid; file nlink/SHA-256,\n'
    'symlink target; fifo/socket same base; block/character add nonnegative integer\n'
    'non-bool `st_rdev`. Reject other types before acceptance. Protect\n'
    'ignored, nested-`.git`, and transaction-lookalike paths; exclude only root\n'
    '`.git` and the exact selected transaction subtree. Container/siblings protected. Limits: 200,000 rows,\n'
    '16,777,216 encoded bytes, one 600-second wall, and 1,048,576 journal bytes.\n'
    'Bootstrap sidecar/journal temps reconcile only before destination mutation.\n'
    'Sidecar is immutable evidence in `FAILED`/`CLEANED`; stage, backups, proof-cache,\n'
    'mutation artifacts clean exactly.'
)
OUTSIDE_MANIFEST_CONTRACT_CELLS = (
    "one immutable transaction-private `outside-manifest.json` sidecar",
    "only exact relative name, SHA-256, encoded bytes, and row count",
    "no\ngeneration duplicates payload",
    "Descriptor-stable records",
    "directory/file/symlink type/mode/uid/gid; file nlink/SHA-256",
    "symlink target; fifo/socket same base",
    "block/character add nonnegative integer\nnon-bool `st_rdev`",
    "Reject other types before acceptance",
    "Protect\nignored, nested-`.git`, and transaction-lookalike paths",
    "exclude only root\n`.git` and the exact selected transaction subtree",
    "Container/siblings protected",
    "200,000 rows",
    "16,777,216 encoded bytes",
    "one 600-second wall",
    "1,048,576 journal bytes",
    "Bootstrap sidecar/journal temps reconcile only before destination mutation",
    "Sidecar is immutable evidence in `FAILED`/`CLEANED`",
    "stage, backups, proof-cache,\nmutation artifacts clean exactly",
)
LIVE_PROTECTED_SHA256 = {
    ".gitignore": "24fc2f2ec652f10c946901863681711b541b018eda200292b51279819cec9484",
    ".devrites/ACTIVE": "fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267",
    ".devrites/work/workspace-observation/touched-files.md":
        "2dca74484895de119cd935db6c3692782df9173eef199c88a7d5a65898332ec9",
}
EXPECTED_NORMAL_GENERATED_DELTA = {
    "claude/skills/devrites-lib/reference/standards/workflow-artifacts.md",
    "codex/skills/devrites-lib/reference/standards/workflow-artifacts.md",
}
EXPECTED_NORMAL_GENERATED_SHA256 = {
    "claude/skills/devrites-lib/reference/standards/workflow-artifacts.md":
        "f6f12115c3a06b40ba1f499719c10b19cc41b119ba781e40e1de082c09e98e88",
    "codex/skills/devrites-lib/reference/standards/workflow-artifacts.md":
        "c603db8c1e9596beab6bb977792a61e4c2bf8621c85bded79c4ea0fc78907a41",
}
PRECHANGE_NORMAL_GENERATED_SHA256 = {
    "claude/skills/devrites-lib/reference/standards/workflow-artifacts.md":
        "03f75660d35c781986d14edac07dcebccd216bbd06793e596564b59145d40fd7",
    "codex/skills/devrites-lib/reference/standards/workflow-artifacts.md":
        "628e7a58eb958bbacde7e443683e1ff319b93ce4f601f3b149e8cd652a1b1ed5",
}
RESLICE_PRIOR_RECORDS = {
    'evals/behavioral/acceptance-preserving-reslice.json': (0o600, 'd171e50fd3e8d5d0a6370e7ace58c5b1552967dd4578da6eafe4ab428f69164e'),
    'evals/behavioral/fixtures/acceptance-preserving-reslice-packets.json': (0o600, '72eab43f907a7ee87e6518da551b6f34de8305ce0f27aa798d84f1209c06e9f5'),
    'pack/.claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md': (0o600, '844ddd36bf76b3674c7b54126747308dae8b9438d49490186d575254e15966bd'),
    'pack/.claude/skills/rite-autocomplete/reference/decision-policy.md': (0o600, '772a79298cf7d5613a3713fc54eff9167b66a972da201db0486d3f82db14a0aa'),
    'pack/.claude/skills/rite-plan/SKILL.md': (0o600, 'd7740e9359235bd5c1a825c8c8f46a2e1d371b274b502b5a508c813f1de649e4'),
    'pack/.claude/skills/rite-plan/reference/anti-patterns.md': (0o600, '280291f3fa0e64e82df61df75f7b453bc8f36b77e5b87f5281d6488beee786d4'),
    'pack/.claude/skills/rite-plan/reference/replan-and-repair.md': (0o600, 'af0bc98e9e3480d1be0a6fc99f9df003d6a0e5a425adf99c608665bede37e98c'),
    'pack/.claude/skills/rite-vet/reference/anti-patterns.md': (0o600, '0fdd02a3bb00914fe697b67d63f44305ed513bc83877ae4cb8b652eedb482697'),
    'pack/.claude/skills/rite-vet/reference/artifacts.md': (0o600, '2020cf2294c97e5a97c080bf74e812ad2806604d5e6eaa98b040a7855685256a'),
    'pack/.claude/skills/rite-vet/reference/depth.md': (0o600, '1e477745e062c5e1a1fb5533bd4983b8c225684d84bf2dbcd72c1ac4344baf0e'),
    'pack/.claude/skills/rite-vet/reference/review-axes.md': (0o600, '57cddb6cb49456554c528399761610d51819edd45891b55ea60e106f42e5f3ee'),
    'pack/generated/claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md': (0o600, '844ddd36bf76b3674c7b54126747308dae8b9438d49490186d575254e15966bd'),
    'pack/generated/claude/skills/rite-autocomplete/reference/decision-policy.md': (0o600, '772a79298cf7d5613a3713fc54eff9167b66a972da201db0486d3f82db14a0aa'),
    'pack/generated/claude/skills/rite-plan/SKILL.md': (0o600, 'd7740e9359235bd5c1a825c8c8f46a2e1d371b274b502b5a508c813f1de649e4'),
    'pack/generated/claude/skills/rite-plan/reference/anti-patterns.md': (0o600, '280291f3fa0e64e82df61df75f7b453bc8f36b77e5b87f5281d6488beee786d4'),
    'pack/generated/claude/skills/rite-plan/reference/replan-and-repair.md': (0o600, 'af0bc98e9e3480d1be0a6fc99f9df003d6a0e5a425adf99c608665bede37e98c'),
    'pack/generated/claude/skills/rite-vet/reference/anti-patterns.md': (0o600, '0fdd02a3bb00914fe697b67d63f44305ed513bc83877ae4cb8b652eedb482697'),
    'pack/generated/claude/skills/rite-vet/reference/artifacts.md': (0o600, '2020cf2294c97e5a97c080bf74e812ad2806604d5e6eaa98b040a7855685256a'),
    'pack/generated/claude/skills/rite-vet/reference/depth.md': (0o600, '1e477745e062c5e1a1fb5533bd4983b8c225684d84bf2dbcd72c1ac4344baf0e'),
    'pack/generated/claude/skills/rite-vet/reference/review-axes.md': (0o600, '57cddb6cb49456554c528399761610d51819edd45891b55ea60e106f42e5f3ee'),
    'pack/generated/codex/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md': (0o644, '844ddd36bf76b3674c7b54126747308dae8b9438d49490186d575254e15966bd'),
    'pack/generated/codex/skills/rite-autocomplete/reference/decision-policy.md': (0o644, '2cd420d990ee3fe0359e4f964b446d0476c48f2cf6703295c9e89ccf57e65228'),
    'pack/generated/codex/skills/rite-plan/SKILL.md': (0o644, '874961b16aec63e2b5de71e958845ace68240a8abc6438d8d1b9542c9620398a'),
    'pack/generated/codex/skills/rite-plan/reference/anti-patterns.md': (0o644, '80e4f70ecb83b6df8918a7acd6df6f7435e5271bea4b844ba7b1fa5ffda06f1f'),
    'pack/generated/codex/skills/rite-plan/reference/replan-and-repair.md': (0o644, '94accee995a3412a0a6f8f951b9696ce8388491442adb3ec10cd47f95bfa1df6'),
    'pack/generated/codex/skills/rite-vet/reference/anti-patterns.md': (0o644, '5afbb9ca60494e29b72a58eb43f35c326624a8bece94940066c6fdec41d47b43'),
    'pack/generated/codex/skills/rite-vet/reference/artifacts.md': (0o644, 'c31e88a64032fbc7a391b33e355fb0e8defa56e3ee7672d9717308ad365736b2'),
    'pack/generated/codex/skills/rite-vet/reference/depth.md': (0o644, '9269a6671618b845f0690d3450c08496daaad9727a49f40b3388b0f9e1edc4c6'),
    'pack/generated/codex/skills/rite-vet/reference/review-axes.md': (0o644, 'f792cc70698d52309e6c549fb3bb57a7439419c3d8ced686ccf642f51d9a9dc8'),
    'tests/acceptance-preserving-reslice-policy-test.sh': (0o600, '79febaeffcb39500f94e7cf87bdd40916fb56e06d7c7bec14559f1a5f7976439'),
}

RESLICE_WORKSPACE_RECORDS = {
    'ai-spec.md': (0o644, '42264bf6552f26bf5c7eb34ab8373341cc631416cee5b0f54b7254be9d7422d7'),
    'analysis.md': (0o644, 'c031e744d0940f162cc335791f21e1ba013a529c570c09992abda6f73059e1b5'),
    'architecture.md': (0o644, '35ac4fc3c1512157093d8657ebae8173d9c2423343876a3eaa0e7610cd938e48'),
    'assumptions.md': (0o644, '4b921c525d933395d2a2e3dfbec244cb849f9910a4486b03553427b72131a408'),
    'brief.md': (0o644, 'ab236cdc8f47a63b049038148ad917ec050d98e1406d8c1e0ba035ed0d3f799f'),
    'decision-coverage.md': (0o644, '9c69fbb325d458fe0799d879f34e8aab1539b796a3587cafe97ed6956c913daa'),
    'decisions.md': (0o644, '1018c02466b75e39b886f83db5981eae1f29012bcfe7fad96fdb315758f809b4'),
    'devex.md': (0o644, '089824f5b805c2357f4ceebd361c6677ac4bc3c20242c92ed3f87f1d6b1c76e5'),
    'eng-review.md': (0o644, 'e243ac0cadc64cab5ea9afe8ecf937c0ead714f06fe094434aef81b24fa3fcd1'),
    'evidence.md': (0o644, 'cc4cc3884ba8024f9ed34422eeab2b84ca3e9b8b7aede8f353c8fbc3ed45838e'),
    'plan.md': (0o644, 'aaffcee3357a5c649f8c3e514153ed05dc691032e18693f5e5e48835f5402917'),
    'polish-report.md': (0o644, '74cd67afc4d15776567d7b66812e14a1e593bd9503cb9df4c2f8741c14f843f7'),
    'questions.md': (0o644, '3d40ebff11e35f162ade3dbc3da46b9c7e7055140461022cbab3a0871ba796fa'),
    'review.md': (0o644, 'ea1406f536c1894eee87924bdcfa0213eea1af3c45db133b4b453c53f5bcf8f9'),
    'seal.md': (0o644, '314e868dc4b0c4967a0eeef3abfc4a7f32357d3f62f6c7c81d5639baf0ae5805'),
    'spec.md': (0o644, '7d18793c4098cb5e9472cd360df991d3337587855f30486b16a05cf547e48f86'),
    'state.md': (0o644, 'b53a7338ee6bb3410dfda1fc389a928c669b9d024b3ff2e260b5c10061a9cb99'),
    'strategy.md': (0o644, '39f4d1010e1a06e4ad9306e71c6abb819aa2d48c065b6c9de840d9fb4e9bcb06'),
    'tasks.md': (0o644, '786ab2e6b3427948e23823e5e7eeec8a045ef18f2667cbcf7915774028f22922'),
    'test-plan.md': (0o644, '611a8e0041c5c78244e6f8d290c681d7aa89c01a601dc63cc62dfa564ec1a917'),
    'touched-files.md': (0o644, 'd8a3a783efb649e78a95aa3799a78f8b705f5535ba3c8252886a40e3fe0da78c'),
    'traceability.md': (0o644, '0d30431c57275bf1ce32f94a0a878a586a2c02a97921da4c4ae1bdf91591b91c'),
}
EXPECTED_OPS = [
    "WA-OP-001-OWNER-ACQUIRE", "WA-OP-002-SOURCE-PROMOTE",
    "WA-OP-002A-STALE-SOURCE-GC", "WA-OP-003-JOURNAL-INIT",
    "WA-OP-004-STAGE-WRITE", "WA-OP-005-BACKUP-WRITE",
    "WA-OP-006-INSTALL", "WA-OP-007-PROVE", "WA-OP-008-ROLLBACK",
    "WA-OP-009-FAILURE-CLEANUP", "WA-OP-010-SUCCESS-CLEANUP",
    "WA-OP-011-RETRY-HANDOFF", "WA-OP-012-EXHAUSTION-GC",
    "WA-OP-013-EVIDENCE-UPDATE", "WA-OP-014-PRODUCT-SEPARATION",
    "WA-OP-015-VERIFY-EXISTING",
]
EXPECTED_SCENARIOS = [
    "WA-ADMISSION-SUCCESS", "WA-MISSING-IDENTITY", "WA-STALE-IDENTITY",
    "WA-STALE-WRITER-EXHAUSTION", "WA-FIRST-ROOT-FAILURE",
    "WA-REPLACEMENT-ROLLBACK", "WA-CLEANUP", "WA-IDENTITY-CONTINUITY",
    "WA-COMPLETED-HISTORICAL", "WA-IDEMPOTENT-RERUN",
]
EXPECTED_ROUTES = {
    "WA-ADMISSION-SUCCESS": "ROOT_TRANSACTION",
    "WA-MISSING-IDENTITY": "PLAN_VET_REPAIR",
    "WA-STALE-IDENTITY": "PLAN_VET_REPAIR",
    "WA-STALE-WRITER-EXHAUSTION": "PLAN_VET_REPAIR",
    "WA-FIRST-ROOT-FAILURE": "OFFLINE_RECOVERY",
    "WA-REPLACEMENT-ROLLBACK": "OFFLINE_RECOVERY",
    "WA-CLEANUP": "RESUME_CLEANUP",
    "WA-IDENTITY-CONTINUITY": "PROVE_AND_RETURN",
    "WA-COMPLETED-HISTORICAL": "NO_BACKFILL",
    "WA-IDEMPOTENT-RERUN": "VERIFY_EXISTING",
}
EXPECTED_RECOVERY_ROUTES = {
    "PLAN_VET_REPAIR": (
        "controlling root",
        "run /rite-plan repair <slug> then /rite-vet <slug> internally",
        "phase=plan, status=running, next_action=/rite-plan repair <slug> until Vet READY",
        "restore saved caller cursor; Autocomplete emits no intermediate reply",
    ),
    "OFFLINE_RECOVERY": (
        "controlling root",
        "run /devrites-debug-recovery <slug>, disposable re-preflight, then narrow /rite-vet <slug>",
        "status=running, next_action=/devrites-debug-recovery <slug>; retry only from durable FAILED and remaining cap",
        "preserve cursor and attempt history; no real action",
    ),
}
ADAPTERS = {
    "devrites-lib/reference/standards/afk-hitl.md": "pack/.claude/skills/devrites-lib/reference/standards/afk-hitl.md",
    "devrites-lib/reference/standards/one-shot-actions.md": "pack/.claude/skills/devrites-lib/reference/standards/one-shot-actions.md",
    "devrites-debug-recovery/SKILL.md": "pack/.claude/skills/devrites-debug-recovery/SKILL.md",
    "rite-autocomplete/SKILL.md": "pack/.claude/skills/rite-autocomplete/SKILL.md",
    "rite-autocomplete/reference/loop.md": "pack/.claude/skills/rite-autocomplete/reference/loop.md",
    "rite-autocomplete/reference/stop-conditions.md": "pack/.claude/skills/rite-autocomplete/reference/stop-conditions.md",
    "rite-build/SKILL.md": "pack/.claude/skills/rite-build/SKILL.md",
    "rite-build/reference/phase-contract.md": "pack/.claude/skills/rite-build/reference/phase-contract.md",
    "rite-prove/SKILL.md": "pack/.claude/skills/rite-prove/SKILL.md",
    "rite-vet/SKILL.md": "pack/.claude/skills/rite-vet/SKILL.md",
}


def fail(message: str) -> None:
    raise AssertionError(message)


def require(condition: bool, message: str) -> None:
    if not condition:
        fail(message)


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


DIRECTORY_FLAGS = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC
FILE_READ_FLAGS = os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC
DELIVERY_PROCESS_TIMEOUT_SECONDS = 600
DELIVERY_AGGREGATE_TIMEOUT_SECONDS = 3600
DELIVERY_OUTPUT_LIMIT_BYTES = 8_388_608
DELIVERY_TERMINATE_GRACE_SECONDS = 2
OUTSIDE_MANIFEST_NAME = "outside-manifest.json"
OUTSIDE_MANIFEST_TEMPORARY = ".outside-manifest.json.workflow-artifact.tmp"
JOURNAL_TEMPORARY = ".journal.json.workflow-artifact.tmp"
OUTSIDE_MANIFEST_MAX_ENTRIES = 200_000
OUTSIDE_MANIFEST_MAX_BYTES = 16_777_216
OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS = 600
DELIVERY_JOURNAL_MAX_BYTES = 1_048_576
DELIVERY_FIXTURE_ENV = (
    "DEVRITES_DELIVERY_FAST_FIXTURE",
    "DEVRITES_DELIVERY_TEST_MUTATION",
    "DEVRITES_DELIVERY_DEATH_BOUNDARY",
    "DEVRITES_DELIVERY_SKIP_GENERATED_MUTANT",
)
_DELIVERY_TEST = {
    "fast_fixture": False,
    "mutation": None,
    "death_boundary": None,
    "skip_generated": None,
}
HELD_GENERATOR_OUTPUT = ".held-out"
HELD_GENERATOR_ARTIFACTS = "artifacts"


def delivery_test_fast_fixture() -> bool:
    return _DELIVERY_TEST["fast_fixture"] is True


def delivery_test_mutation_name() -> str | None:
    return _DELIVERY_TEST["mutation"]


def delivery_test_death_boundary() -> str | None:
    return _DELIVERY_TEST["death_boundary"]


def delivery_test_skip_generated() -> str | None:
    return _DELIVERY_TEST["skip_generated"]


def delivery_parallel_workers() -> int:
    raw = os.environ.get("DEVRITES_WAI_PARALLEL_WORKERS", "4")
    try:
        workers = int(raw)
    except ValueError:
        workers = 4
    return max(1, min(8, workers))


def parse_boundary_shard_spec() -> tuple[int, int] | None:
    spec = os.environ.get("DEVRITES_WAI_BOUNDARY_SHARD", "").strip()
    if not spec:
        return None
    match = re.fullmatch(r"(\d+)/(\d+)", spec)
    require(match is not None, f"invalid DEVRITES_WAI_BOUNDARY_SHARD: {spec}")
    index = int(match.group(1))
    total = int(match.group(2))
    require(1 <= index <= total, f"invalid DEVRITES_WAI_BOUNDARY_SHARD: {spec}")
    return index, total


def filter_boundary_shard(boundaries: set[str]) -> set[str]:
    spec = parse_boundary_shard_spec()
    if spec is None:
        return boundaries
    index, total = spec
    selected = [
        boundary for offset, boundary in enumerate(sorted(boundaries))
        if offset % total == index - 1
    ]
    require(len(selected) > 0, f"empty workflow-artifact boundary shard: {index}/{total}")
    return set(selected)


def wai_skip_delivery_modes() -> bool:
    return os.environ.get("DEVRITES_WAI_SKIP_DELIVERY_MODES") == "1"


def wai_skip_delivery_model_matrix() -> bool:
    return os.environ.get("DEVRITES_WAI_SKIP_DELIVERY_MODEL_MATRIX") == "1"


def wai_boundary_only() -> bool:
    return os.environ.get("DEVRITES_WAI_BOUNDARY_ONLY") == "1"


def wai_delivery_model_only() -> bool:
    return os.environ.get("DEVRITES_WAI_DELIVERY_MODEL_ONLY") == "1"


def reject_delivery_fixture_environment() -> None:
    present = [name for name in DELIVERY_FIXTURE_ENV if name in os.environ]
    require(not present, "delivery modes reject fixture environment")


def reject_delivery_fixture_argv(config: dict) -> None:
    require(
        config["fast_fixture"] is not True
        and config["mutation"] is None
        and config["skip_generated"] is None,
        "delivery modes reject fixture argv",
    )


def production_delivery_argv(args: list[str]) -> bool:
    return (
        args == ["--delivery-prepare"]
        or (len(args) == 2 and args[0] in {"--delivery-install", "--delivery-recover"})
    )


def env_without_looping_bash(env: dict[str, str]) -> dict[str, str]:
    kept = []
    for directory in env.get("PATH", "").split(os.pathsep):
        if not directory:
            continue
        try:
            os.stat(os.path.join(directory, "bash"))
        except OSError as exc:
            if exc.errno == errno.ELOOP:
                continue
        kept.append(directory)
    env = dict(env)
    env["PATH"] = os.pathsep.join(kept)
    return env


def take_delivery_test_argv(args: list[str]) -> tuple[list[str], dict]:
    config = {
        "fast_fixture": False,
        "mutation": None,
        "death_boundary": None,
        "skip_generated": None,
    }
    rest: list[str] = []
    index = 0
    while index < len(args):
        item = args[index]
        if item == "--delivery-test-fast-fixture":
            config["fast_fixture"] = True
            index += 1
            continue
        if item == "--delivery-test-mutation" and index + 1 < len(args):
            config["mutation"] = args[index + 1]
            index += 2
            continue
        if item == "--delivery-test-death" and index + 1 < len(args):
            config["death_boundary"] = args[index + 1]
            index += 2
            continue
        if item == "--delivery-test-skip-generated" and index + 1 < len(args):
            config["skip_generated"] = args[index + 1]
            index += 2
            continue
        rest.append(item)
        index += 1
    return rest, config


class OutsideManifestWallTimeout(Exception):
    pass


@contextlib.contextmanager
def outside_manifest_wall_timer(seconds: float):
    if seconds <= 0:
        raise OutsideManifestWallTimeout
    started = time.monotonic()
    previous_timer = signal.setitimer(signal.ITIMER_REAL, 0)
    previous_handler = signal.getsignal(signal.SIGALRM)

    def timeout_handler(_signum, _frame):
        raise OutsideManifestWallTimeout

    signal.signal(signal.SIGALRM, timeout_handler)
    signal.setitimer(signal.ITIMER_REAL, seconds)
    try:
        yield
    finally:
        signal.setitimer(signal.ITIMER_REAL, 0)
        signal.signal(signal.SIGALRM, previous_handler)
        if previous_timer[0] > 0:
            remaining = max(previous_timer[0] - (time.monotonic() - started), 1e-6)
            signal.setitimer(signal.ITIMER_REAL, remaining, previous_timer[1])


def with_outside_manifest_deadline(operation, seconds: float | None = None):
    timeout = OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS if seconds is None else seconds
    deadline = time.monotonic() + max(timeout, 0)
    try:
        with outside_manifest_wall_timer(timeout):
            return operation(deadline)
    except OutsideManifestWallTimeout:
        fail("outside manifest scan-time bound")


def require_outside_manifest_deadline(deadline: float) -> None:
    require(time.monotonic() <= deadline, "outside manifest scan-time bound")


def fsync_dir(path: Path) -> None:
    fd = os.open(path, DIRECTORY_FLAGS)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def acquire_owner_lock(fd: int, primitive: str = "flock") -> None:
    require(primitive == "flock", "alternate owner lock primitive")
    fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)


def complete_write(fd: int, data: bytes, writer=os.write) -> None:
    view = memoryview(data)
    offset = 0
    while offset < len(view):
        progress = writer(fd, view[offset:])
        remaining = len(view) - offset
        if isinstance(progress, bool) or not isinstance(progress, int):
            raise OSError(errno.EIO, "invalid write progress")
        if progress < 1 or progress > remaining:
            raise OSError(errno.EIO, "invalid write progress")
        offset += progress


def relative_components(value: str) -> tuple[str, ...]:
    require(value != "" and not value.startswith("/") and "\\" not in value, "descriptor-relative path")
    parts = tuple(value.split("/"))
    require(all(part not in {"", ".", ".."} for part in parts), "descriptor-relative components")
    return parts


def validate_directory_fd(fd: int, mode: int | None = None) -> None:
    info = os.fstat(fd)
    require(stat.S_ISDIR(info.st_mode), "trusted directory descriptor")
    if mode is not None:
        require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == mode, "trusted directory mode")


def open_dir_components(root_fd: int, components: tuple[str, ...], create: bool = False,
                        mode: int = 0o700) -> int:
    current = os.dup(root_fd)
    try:
        validate_directory_fd(current)
        for component in components:
            try:
                following = os.open(component, DIRECTORY_FLAGS, dir_fd=current)
            except FileNotFoundError:
                if not create:
                    raise
                os.mkdir(component, mode, dir_fd=current)
                os.fsync(current)
                following = os.open(component, DIRECTORY_FLAGS, dir_fd=current)
            validate_directory_fd(following)
            os.close(current)
            current = following
        return current
    except BaseException:
        os.close(current)
        raise


def open_parent_fd(root_fd: int, relative: str, create: bool = False) -> tuple[int, str]:
    components = relative_components(relative)
    return open_dir_components(root_fd, components[:-1], create=create), components[-1]


def read_fd_bounded(fd: int, limit: int) -> bytes:
    chunks = []
    total = 0
    while True:
        chunk = os.read(fd, min(65536, limit + 1 - total))
        if not chunk:
            return b"".join(chunks)
        chunks.append(chunk)
        total += len(chunk)
        require(total <= limit, "bounded descriptor read")


def secure_file_info(info: os.stat_result, modes: set[int] | None = None) -> None:
    require(info.st_uid == os.getuid() and stat.S_ISREG(info.st_mode) and info.st_nlink == 1, "trusted regular descriptor")
    if modes is not None:
        require(stat.S_IMODE(info.st_mode) in modes, "trusted regular mode")


def read_file_at(root_fd: int, relative: str, limit: int = 16 * 1024 * 1024,
                 modes: set[int] | None = None) -> bytes:
    parent_fd, name = open_parent_fd(root_fd, relative)
    try:
        fd = os.open(name, FILE_READ_FLAGS, dir_fd=parent_fd)
        try:
            info = os.fstat(fd)
            secure_file_info(info, modes)
            require(info.st_size <= limit, "bounded file size")
            return read_fd_bounded(fd, limit)
        finally:
            os.close(fd)
    finally:
        os.close(parent_fd)


def entry_info_at(root_fd: int, relative: str) -> os.stat_result | None:
    parent_fd, name = open_parent_fd(root_fd, relative)
    try:
        try:
            return os.stat(name, dir_fd=parent_fd, follow_symlinks=False)
        except FileNotFoundError:
            return None
    finally:
        os.close(parent_fd)


def unlink_file_at(root_fd: int, relative: str, missing_ok: bool = False) -> None:
    parent_fd, name = open_parent_fd(root_fd, relative)
    try:
        try:
            info = os.stat(name, dir_fd=parent_fd, follow_symlinks=False)
        except FileNotFoundError:
            if missing_ok:
                return
            raise
        secure_file_info(info)
        os.unlink(name, dir_fd=parent_fd)
        os.fsync(parent_fd)
    finally:
        os.close(parent_fd)


def atomic_write_at(root_fd: int, relative: str, data: bytes, mode: int, lock_fd: int,
                    writer=os.write, death_boundary: str | None = None) -> None:
    validate_directory_fd(root_fd)
    lock_info = os.fstat(lock_fd)
    require(lock_info.st_uid == os.getuid() and stat.S_ISREG(lock_info.st_mode)
            and lock_info.st_nlink == 1 and stat.S_IMODE(lock_info.st_mode) == 0o600,
            "held owner lock descriptor")
    parent_fd, name = open_parent_fd(root_fd, relative, create=True)
    temp_name = f".{name}.workflow-artifact.tmp"
    flags = os.O_RDWR | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC
    fd = -1
    try:
        try:
            fd = os.open(temp_name, flags, 0o600, dir_fd=parent_fd)
            if death_boundary == "after-create":
                os.fsync(parent_fd)
                os._exit(86)
            prefix = b""
        except FileExistsError:
            fd = os.open(temp_name, os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=parent_fd)
            info = os.fstat(fd)
            secure_file_info(info, {0o600, mode})
            require(info.st_size <= len(data), "atomic temporary size")
            prefix = read_fd_bounded(fd, len(data))
            require(data.startswith(prefix), "atomic temporary prefix")
            os.lseek(fd, len(prefix), os.SEEK_SET)
        remaining = data[len(prefix):]
        if death_boundary == "after-partial" and remaining:
            partial_size = min(len(remaining), 257)
            require(partial_size < len(remaining), "atomic partial death requires remainder")
            complete_write(fd, remaining[:partial_size], writer)
            os.fsync(fd)
            os._exit(88)
        complete_write(fd, remaining, writer)
        os.fchmod(fd, mode)
        os.fsync(fd)
        if death_boundary == "after-sync":
            os._exit(87)
        temporary_info = os.fstat(fd)
        os.close(fd)
        fd = -1
        pathname_info = os.stat(temp_name, dir_fd=parent_fd, follow_symlinks=False)
        require((temporary_info.st_dev, temporary_info.st_ino)
                == (pathname_info.st_dev, pathname_info.st_ino),
                "atomic temporary pathname identity")
        if death_boundary == "before-rename":
            os._exit(89)
        src_fd = os.dup(parent_fd)
        dst_fd = os.dup(parent_fd)
        try:
            os.rename(temp_name, name, src_dir_fd=src_fd, dst_dir_fd=dst_fd)
            os.fsync(dst_fd)
        finally:
            os.close(src_fd)
            os.close(dst_fd)
    except BaseException:
        if fd >= 0:
            os.close(fd)
        raise
    finally:
        os.close(parent_fd)


def bootstrap_atomic_death(kind: str) -> str | None:
    configured = delivery_test_death_boundary()
    return {
        f"bootstrap-{kind}-after-create": "after-create",
        f"bootstrap-{kind}-after-partial": "after-partial",
        f"bootstrap-{kind}-after-sync": "after-sync",
        f"bootstrap-{kind}-before-rename": "before-rename",
    }.get(configured)


def absolute_descriptor(path: Path) -> tuple[int, str]:
    absolute = path.absolute()
    root_fd = os.open(absolute.anchor, DIRECTORY_FLAGS)
    return root_fd, absolute.relative_to(absolute.anchor).as_posix()


def open_absolute_directory(path: Path) -> int:
    anchor_fd, relative = absolute_descriptor(path)
    try:
        return open_dir_components(anchor_fd, relative_components(relative))
    finally:
        os.close(anchor_fd)


def hash_file_descriptor(fd: int, limit: int | None = None,
                         deadline: float | None = None) -> tuple[str, int]:
    digest = hashlib.sha256()
    total = 0
    while True:
        if deadline is not None:
            require_outside_manifest_deadline(deadline)
        read_size = 65536 if limit is None else min(65536, limit + 1 - total)
        require(read_size > 0, "bounded hash read")
        chunk = os.read(fd, read_size)
        if deadline is not None:
            require_outside_manifest_deadline(deadline)
        if not chunk:
            return digest.hexdigest(), total
        total += len(chunk)
        if limit is not None:
            require(total <= limit, "bounded hash read")
        digest.update(chunk)


def file_record_at(root_fd: int, relative: str) -> dict:
    initial = entry_info_at(root_fd, relative)
    if initial is None:
        return {"state": "absent"}
    secure_file_info(initial)
    require(initial.st_size <= OUTSIDE_MANIFEST_MAX_BYTES, "bounded file size")
    parent_fd, name = open_parent_fd(root_fd, relative)
    try:
        fd = os.open(name, FILE_READ_FLAGS, dir_fd=parent_fd)
        try:
            opened = os.fstat(fd)
            secure_file_info(opened)
            required = lambda info: (
                info.st_dev, info.st_ino, info.st_mode, info.st_uid,
                info.st_gid, info.st_nlink, info.st_size,
            )
            require(required(initial) == required(opened),
                    "file record pathname replacement")
            require(opened.st_size <= OUTSIDE_MANIFEST_MAX_BYTES, "bounded file size")
            digest, size = hash_file_descriptor(fd, OUTSIDE_MANIFEST_MAX_BYTES)
            require(size == opened.st_size, "file record descriptor mutation")
            try:
                final = os.stat(name, dir_fd=parent_fd, follow_symlinks=False)
            except FileNotFoundError:
                fail("file record pathname replacement")
            require(required(opened) == required(final),
                    "file record pathname replacement")
        finally:
            os.close(fd)
    finally:
        os.close(parent_fd)
    return {
        "state": "present", "mode": stat.S_IMODE(opened.st_mode),
        "sha256": digest, "size": opened.st_size,
    }


def compatible_tracked_modes(expected: int) -> set[int]:
    """Git tracks only the executable bit; checkout umask may yield 0644 or 0600."""
    if expected & 0o111:
        return {expected}
    return {0o600, 0o644}


def require_fixed_reslice_records(root_fd: int) -> None:
    require(len(RESLICE_PRIOR_RECORDS) == 30
            and sha(("\n".join(sorted(RESLICE_PRIOR_RECORDS)) + "\n").encode())
            == "1a43cc6e486c93cbd9548626df11fb20da988254626a0224c58c08ca30dd2e4c",
            "historical Reslice exact non-overlap path set")
    for relative, (mode, digest) in RESLICE_PRIOR_RECORDS.items():
        record = file_record_at(root_fd, relative)
        require(record["state"] == "present"
                and record["mode"] in compatible_tracked_modes(mode)
                and record["sha256"] == digest,
                f"historical Reslice record identity: {relative}")


def check_historical_reslice_identity() -> None:
    project = project_root_for_tests(canonical_root())
    root_fd = open_absolute_directory(project)
    try:
        require_fixed_reslice_records(root_fd)
        workspace_relative = ".devrites/work/acceptance-preserving-reslice-policy"
        workspace_info = entry_info_at(root_fd, workspace_relative)
        if workspace_info is None:
            print("reslice_workspace=not-applicable")
        else:
            require(workspace_info.st_uid == os.getuid()
                    and stat.S_ISDIR(workspace_info.st_mode)
                    and stat.S_IMODE(workspace_info.st_mode) == 0o755,
                    "historical Reslice workspace directory identity")
            workspace_fd = open_dir_components(
                root_fd, relative_components(workspace_relative),
            )
            try:
                require(set(os.listdir(workspace_fd)) == set(RESLICE_WORKSPACE_RECORDS),
                        "historical Reslice sealed workspace inventory")
                for relative, (mode, digest) in RESLICE_WORKSPACE_RECORDS.items():
                    record = file_record_at(workspace_fd, relative)
                    require(record["state"] == "present"
                            and record["mode"] == mode
                            and record["sha256"] == digest,
                            f"historical Reslice workspace record: {relative}")
            finally:
                os.close(workspace_fd)

        with tempfile.TemporaryDirectory() as tmp:
            fixture = Path(tmp).resolve() / "prior-reslice"
            fixture.mkdir()
            for relative, (mode, _digest) in RESLICE_PRIOR_RECORDS.items():
                data = read_file_at(
                    root_fd, relative, modes=compatible_tracked_modes(mode),
                )
                destination = fixture / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                atomic_write(destination, data, mode)
            fixture_fd = open_absolute_directory(fixture)
            try:
                require_fixed_reslice_records(fixture_fd)
                mutated = "pack/.claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md"
                mode = RESLICE_PRIOR_RECORDS[mutated][0]
                data = read_file_at(
                    fixture_fd, mutated, modes=compatible_tracked_modes(mode),
                )
                atomic_write(fixture / mutated, data + b"\n", mode)
                try:
                    require_fixed_reslice_records(fixture_fd)
                except AssertionError as error:
                    require(str(error) == f"historical Reslice record identity: {mutated}",
                            "historical Reslice newline mutation rejection")
                else:
                    fail("historical Reslice newline mutation survived")
            finally:
                os.close(fixture_fd)
    finally:
        os.close(root_fd)


def manifest_identity(info: os.stat_result) -> tuple[int, int]:
    return info.st_dev, info.st_ino


def require_manifest_path_identity(before: os.stat_result,
                                   after: os.stat_result) -> None:
    require(manifest_identity(before) == manifest_identity(after),
            "outside manifest pathname replacement")


def manifest_metadata(info: os.stat_result) -> dict:
    return {
        "mode": stat.S_IMODE(info.st_mode), "uid": info.st_uid, "gid": info.st_gid,
    }


def manifest_at(root_fd: int, excluded: set[str], excluded_prefix: str,
                deadline: float | None = None) -> dict:
    if deadline is None:
        return with_outside_manifest_deadline(
            lambda active_deadline: manifest_at(
                root_fd, excluded, excluded_prefix, active_deadline,
            ),
        )
    rows = {}

    def add(relative: str, record: dict) -> None:
        require_outside_manifest_deadline(deadline)
        require(len(rows) < OUTSIDE_MANIFEST_MAX_ENTRIES, "outside manifest entry bound")
        rows[relative] = record

    def path_info(directory_fd: int, name: str) -> os.stat_result:
        require_outside_manifest_deadline(deadline)
        info = os.stat(name, dir_fd=directory_fd, follow_symlinks=False)
        require_outside_manifest_deadline(deadline)
        return info

    def visit(directory_fd: int, prefix: str) -> None:
        require_outside_manifest_deadline(deadline)
        names = sorted(os.listdir(directory_fd))
        require_outside_manifest_deadline(deadline)
        for name in names:
            relative = f"{prefix}/{name}" if prefix else name
            if (relative == ".git" or relative in excluded
                    or relative == excluded_prefix or relative.startswith(excluded_prefix + "/")):
                continue
            before = path_info(directory_fd, name)
            if stat.S_ISDIR(before.st_mode):
                require_outside_manifest_deadline(deadline)
                child_fd = os.open(name, DIRECTORY_FLAGS, dir_fd=directory_fd)
                try:
                    opened = os.fstat(child_fd)
                    require(stat.S_ISDIR(opened.st_mode), "outside manifest directory type")
                    require_manifest_path_identity(before, opened)
                    add(relative, {"type": "directory", **manifest_metadata(opened)})
                    visit(child_fd, relative)
                    require_manifest_path_identity(opened, path_info(directory_fd, name))
                finally:
                    os.close(child_fd)
            elif stat.S_ISREG(before.st_mode):
                require_outside_manifest_deadline(deadline)
                fd = os.open(name, FILE_READ_FLAGS, dir_fd=directory_fd)
                try:
                    opened = os.fstat(fd)
                    require(stat.S_ISREG(opened.st_mode), "outside manifest file type")
                    require_manifest_path_identity(before, opened)
                    digest, _size = hash_file_descriptor(fd, deadline=deadline)
                    require_manifest_path_identity(opened, path_info(directory_fd, name))
                    add(relative, {
                        "type": "file", **manifest_metadata(opened),
                        "nlink": opened.st_nlink, "sha256": digest,
                    })
                finally:
                    os.close(fd)
            elif stat.S_ISLNK(before.st_mode):
                require_outside_manifest_deadline(deadline)
                target = os.readlink(name, dir_fd=directory_fd)
                require_outside_manifest_deadline(deadline)
                after = path_info(directory_fd, name)
                require_manifest_path_identity(before, after)
                # Linux may reuse the inode on unlink+symlink; re-read target.
                require(os.readlink(name, dir_fd=directory_fd) == target,
                        "outside manifest pathname replacement")
                add(relative, {
                    "type": "symlink", **manifest_metadata(before), "target": target,
                })
            elif stat.S_ISFIFO(before.st_mode):
                require_manifest_path_identity(before, path_info(directory_fd, name))
                add(relative, {"type": "fifo", **manifest_metadata(before)})
            elif stat.S_ISSOCK(before.st_mode):
                require_manifest_path_identity(before, path_info(directory_fd, name))
                add(relative, {"type": "socket", **manifest_metadata(before)})
            elif stat.S_ISBLK(before.st_mode):
                require_manifest_path_identity(before, path_info(directory_fd, name))
                add(relative, {
                    "type": "block", **manifest_metadata(before), "rdev": before.st_rdev,
                })
            elif stat.S_ISCHR(before.st_mode):
                require_manifest_path_identity(before, path_info(directory_fd, name))
                add(relative, {
                    "type": "character", **manifest_metadata(before), "rdev": before.st_rdev,
                })
            else:
                require_manifest_path_identity(before, path_info(directory_fd, name))
                fail("outside manifest unsupported object type")
    visit(root_fd, "")
    require_outside_manifest_deadline(deadline)
    return rows


def encode_outside_manifest(rows: dict, deadline: float | None = None) -> bytes:
    if deadline is None:
        return with_outside_manifest_deadline(
            lambda active_deadline: encode_outside_manifest(rows, active_deadline),
        )
    require_outside_manifest_deadline(deadline)
    require(len(rows) <= OUTSIDE_MANIFEST_MAX_ENTRIES, "outside manifest entry bound")
    encoded_rows = []
    for relative, record in sorted(rows.items()):
        require(record["type"] in {
                    "directory", "file", "symlink", "fifo", "socket", "block", "character",
                }, "outside manifest record type")
        row = [relative, record["type"], record["mode"], record["uid"], record["gid"]]
        if record["type"] == "file":
            row.extend((record["nlink"], record["sha256"]))
        elif record["type"] == "symlink":
            row.append(record["target"])
        elif record["type"] in {"block", "character"}:
            rdev = record["rdev"]
            require(isinstance(rdev, int) and not isinstance(rdev, bool) and rdev >= 0,
                    "outside manifest device identity")
            row.append(rdev)
        encoded_rows.append(row)
    encoded = (json.dumps(encoded_rows, separators=(",", ":")) + "\n").encode()
    require_outside_manifest_deadline(deadline)
    require(len(encoded) <= OUTSIDE_MANIFEST_MAX_BYTES, "outside manifest encoded-byte bound")
    return encoded


def capture_outside_manifest(root_fd: int, excluded: set[str],
                             excluded_prefix: str) -> tuple[dict, bytes]:
    def capture(deadline: float) -> tuple[dict, bytes]:
        rows = manifest_at(root_fd, excluded, excluded_prefix, deadline)
        return rows, encode_outside_manifest(rows, deadline)
    return with_outside_manifest_deadline(capture)


def outside_manifest_binding(rows: dict, encoded: bytes) -> dict:
    return {
        "relative": OUTSIDE_MANIFEST_NAME,
        "sha256": sha(encoded),
        "bytes": len(encoded),
        "rows": len(rows),
    }


def parse_outside_manifest(raw: bytes, binding: dict) -> dict:
    require(isinstance(binding, dict) and set(binding) == {"relative", "sha256", "bytes", "rows"},
            "outside manifest binding fields")
    require(binding["relative"] == OUTSIDE_MANIFEST_NAME, "outside manifest relative name")
    require(isinstance(binding["sha256"], str)
            and re.fullmatch(r"[0-9a-f]{64}", binding["sha256"]) is not None,
            "outside manifest binding hash")
    require(isinstance(binding["bytes"], int) and not isinstance(binding["bytes"], bool)
            and 0 < binding["bytes"] <= OUTSIDE_MANIFEST_MAX_BYTES,
            "outside manifest binding bytes")
    require(isinstance(binding["rows"], int) and not isinstance(binding["rows"], bool)
            and 0 <= binding["rows"] <= OUTSIDE_MANIFEST_MAX_ENTRIES,
            "outside manifest binding rows")
    require(len(raw) == binding["bytes"] and sha(raw) == binding["sha256"],
            "outside manifest binding identity")
    require(raw.endswith(b"\n") and raw.count(b"\n") == 1,
            "outside manifest exact line")

    encoded_rows = json.loads(raw.decode("utf-8"))
    require(isinstance(encoded_rows, list) and len(encoded_rows) == binding["rows"],
            "outside manifest row cardinality")
    rows = {}
    previous = None
    widths = {
        "directory": 5, "file": 7, "symlink": 6, "fifo": 5, "socket": 5,
        "block": 6, "character": 6,
    }
    for row in encoded_rows:
        require(isinstance(row, list) and len(row) >= 2,
                "outside manifest record width")
        relative, record_type = row[:2]
        require(isinstance(relative, str), "outside manifest row path")
        relative_components(relative)
        require(previous is None or previous < relative,
                "outside manifest row order")
        require(isinstance(record_type, str) and record_type in widths,
                "outside manifest record type")
        require(len(row) == widths[record_type], "outside manifest record width")
        mode, uid, gid = row[2:5]
        require(isinstance(mode, int) and not isinstance(mode, bool)
                and 0 <= mode <= 0o7777,
                "outside manifest record mode")
        require(isinstance(uid, int) and not isinstance(uid, bool) and uid >= 0,
                "outside manifest record uid")
        require(isinstance(gid, int) and not isinstance(gid, bool) and gid >= 0,
                "outside manifest record gid")
        record = {"type": record_type, "mode": mode, "uid": uid, "gid": gid}
        if record_type == "file":
            nlink, digest = row[5:7]
            require(isinstance(nlink, int) and not isinstance(nlink, bool) and nlink >= 1
                    and isinstance(digest, str)
                    and re.fullmatch(r"[0-9a-f]{64}", digest) is not None,
                    "outside manifest file identity")
            record.update({"nlink": nlink, "sha256": digest})
        elif record_type == "symlink":
            require(isinstance(row[5], str), "outside manifest symlink target")
            record["target"] = row[5]
        elif record_type in {"block", "character"}:
            rdev = row[5]
            require(isinstance(rdev, int) and not isinstance(rdev, bool) and rdev >= 0,
                    "outside manifest device identity")
            record["rdev"] = rdev
        rows[relative] = record
        previous = relative
    require(encode_outside_manifest(rows) == raw, "outside manifest canonical encoding")
    return rows


def read_outside_manifest(delivery_fd: int, binding: dict) -> dict:
    info = entry_info_at(delivery_fd, OUTSIDE_MANIFEST_NAME)
    require(info is not None and info.st_uid == os.getuid()
            and stat.S_ISREG(info.st_mode) and info.st_nlink == 1
            and stat.S_IMODE(info.st_mode) == 0o600,
            "outside manifest sidecar metadata")
    raw = read_file_at(
        delivery_fd, OUTSIDE_MANIFEST_NAME, OUTSIDE_MANIFEST_MAX_BYTES, {0o600},
    )
    return parse_outside_manifest(raw, binding)


def require_prejournal_private_inventory(delivery_fd: int) -> set[str]:
    entries = set(os.listdir(delivery_fd))
    allowed = {
        ".owner.lock", OUTSIDE_MANIFEST_NAME, OUTSIDE_MANIFEST_TEMPORARY,
        "journal.json", JOURNAL_TEMPORARY,
    }
    require(entries <= allowed,
            "unknown pre-journal delivery private entry")
    return entries


def create_or_reuse_outside_manifest(repo_fd: int, delivery_fd: int,
                                      lock_fd: int,
                                      expected_candidate_digest: str) -> dict:
    entries = require_prejournal_private_inventory(delivery_fd)
    require(not ({OUTSIDE_MANIFEST_NAME, OUTSIDE_MANIFEST_TEMPORARY} <= entries),
            "outside manifest final temporary ambiguity")
    require(not (OUTSIDE_MANIFEST_TEMPORARY in entries
                 and JOURNAL_TEMPORARY in entries),
            "bootstrap temporary ambiguity")
    require(not (JOURNAL_TEMPORARY in entries
                 and OUTSIDE_MANIFEST_NAME not in entries),
            "initial journal temporary requires outside manifest")
    rows, encoded = capture_outside_manifest(
        repo_fd, ALL_DESTINATIONS,
        f".devrites/work/workflow-artifact-identity/.generated-install/{expected_candidate_digest}",
    )
    binding = outside_manifest_binding(rows, encoded)
    if OUTSIDE_MANIFEST_NAME in entries:
        require(read_outside_manifest(delivery_fd, binding) == rows,
                "pre-journal outside manifest safe reuse")
        return binding
    atomic_write_at(
        delivery_fd, OUTSIDE_MANIFEST_NAME, encoded, 0o600, lock_fd,
        death_boundary=bootstrap_atomic_death("sidecar"),
    )
    return binding


def check_manifest_exclusions() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve()
        selected = root / ".generated-install/candidate/inside"
        sibling = root / ".generated-install/sibling/history"
        lookalike = root / ".generated-install-evil/observed"
        nested_git = root / "nested/.git/observed"
        root_git = root / ".git/ignored"
        for path in (selected, sibling, lookalike, nested_git, root_git):
            os.makedirs(path.parent, exist_ok=True)
            path.write_bytes(path.as_posix().encode())
        root_fd = open_absolute_directory(root)
        try:
            observed = manifest_at(root_fd, set(), ".generated-install/candidate")
            broad = manifest_at(root_fd, set(), ".generated-install")
        finally:
            os.close(root_fd)
        require(".generated-install/candidate" not in observed
                and ".generated-install/candidate/inside" not in observed
                and ".git/ignored" not in observed,
                "selected delivery and root Git exclusions")
        require(".generated-install" in observed
                and ".generated-install/sibling" in observed
                and ".generated-install/sibling/history" in observed,
                "transaction container and sibling history remain outside")
        require(".generated-install" not in broad
                and ".generated-install/sibling/history" not in broad,
                "broad transaction-parent exclusion mutant exposed")
        require(".generated-install-evil/observed" in observed,
                "lookalike delivery sibling remains outside")
        require("nested/.git/observed" in observed,
                "nested Git directory remains outside")
        directory = observed["nested/.git"]
        require(directory == {
                    "type": "directory", "mode": stat.S_IMODE(os.lstat(root / "nested/.git").st_mode),
                    "uid": os.getuid(), "gid": os.getgid(),
                }, "recursive outside directory metadata")


def check_manifest_descriptor_substitution() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); atomic_write(root / "file", b"old\n", 0o600)
        atomic_write(root / "replacement", b"new\n", 0o600)
        root_fd = open_absolute_directory(root)
        original_hash = hash_file_descriptor
        replaced = False
        try:
            def replace_during_hash(fd: int, limit: int | None = None,
                                    deadline: float | None = None):
                nonlocal replaced
                result = original_hash(fd, limit, deadline)
                if not replaced:
                    os.rename("file", "old-file", src_dir_fd=root_fd, dst_dir_fd=root_fd)
                    os.rename("replacement", "file", src_dir_fd=root_fd, dst_dir_fd=root_fd)
                    replaced = True
                return result
            globals()["hash_file_descriptor"] = replace_during_hash
            try:
                manifest_at(root_fd, set(), "__excluded__")
            except AssertionError as error:
                require(str(error) == "outside manifest pathname replacement",
                        "regular-file descriptor substitution diagnostic")
            else:
                fail("regular-file descriptor substitution mutant survived")
        finally:
            globals()["hash_file_descriptor"] = original_hash
            os.close(root_fd)

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); (root / "directory").mkdir(); (root / "replacement").mkdir()
        atomic_write(root / "directory/file", b"old\n", 0o600)
        atomic_write(root / "replacement/file", b"new\n", 0o600)
        root_fd = open_absolute_directory(root)
        target = os.lstat(root / "directory")
        original_listdir = os.listdir
        replaced = False
        try:
            def replace_during_list(fd: int):
                nonlocal replaced
                names = original_listdir(fd)
                opened = os.fstat(fd)
                if not replaced and manifest_identity(opened) == manifest_identity(target):
                    os.rename("directory", "old-directory", src_dir_fd=root_fd, dst_dir_fd=root_fd)
                    os.rename("replacement", "directory", src_dir_fd=root_fd, dst_dir_fd=root_fd)
                    replaced = True
                return names
            os.listdir = replace_during_list
            try:
                manifest_at(root_fd, set(), "__excluded__")
            except AssertionError as error:
                require(str(error) == "outside manifest pathname replacement",
                        "directory descriptor substitution diagnostic")
            else:
                fail("directory descriptor substitution mutant survived")
        finally:
            os.listdir = original_listdir
            os.close(root_fd)

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); os.symlink("old-target", root / "link")
        root_fd = open_absolute_directory(root)
        original_readlink = os.readlink
        replaced = False
        try:
            def replace_during_readlink(name, *, dir_fd=None):
                nonlocal replaced
                target = original_readlink(name, dir_fd=dir_fd)
                if not replaced:
                    os.unlink(name, dir_fd=dir_fd)
                    os.symlink("new-target", name, dir_fd=dir_fd)
                    replaced = True
                return target
            os.readlink = replace_during_readlink
            try:
                manifest_at(root_fd, set(), "__excluded__")
            except AssertionError as error:
                require(str(error) == "outside manifest pathname replacement",
                        "symlink descriptor substitution diagnostic")
            else:
                fail("symlink descriptor substitution mutant survived")
        finally:
            os.readlink = original_readlink
            os.close(root_fd)


def check_file_record_descriptor_identity_and_bound() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); atomic_write(root / "file", b"old\n", 0o600)
        atomic_write(root / "replacement", b"new\n", 0o600)
        root_fd = open_absolute_directory(root)
        original_hash = hash_file_descriptor
        try:
            def replace_during_hash(fd: int, limit: int | None = None,
                                    deadline: float | None = None):
                result = original_hash(fd, limit, deadline)
                os.rename("file", "old-file", src_dir_fd=root_fd, dst_dir_fd=root_fd)
                os.rename("replacement", "file", src_dir_fd=root_fd, dst_dir_fd=root_fd)
                return result
            globals()["hash_file_descriptor"] = replace_during_hash
            try:
                file_record_at(root_fd, "file")
            except AssertionError as error:
                require(str(error) == "file record pathname replacement",
                        "file record pathname replacement diagnostic")
            else:
                fail("file record pathname replacement mutant survived")
        finally:
            globals()["hash_file_descriptor"] = original_hash
            os.close(root_fd)

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); oversized = root / "oversized"
        with oversized.open("wb") as stream:
            stream.truncate(OUTSIDE_MANIFEST_MAX_BYTES + 1)
        oversized.chmod(0o600)
        root_fd = open_absolute_directory(root)
        original_read = os.read
        try:
            def reject_read(_fd: int, _size: int) -> bytes:
                fail("oversized file descriptor read")
            os.read = reject_read
            try:
                file_record_at(root_fd, "oversized")
            except AssertionError as error:
                require(str(error) == "bounded file size",
                        "file record byte bound before read")
            else:
                fail("oversized file record accepted")
        finally:
            os.read = original_read
            os.close(root_fd)


def check_generated_stage_manifest_security_bounds() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); atomic_write(root / "file", b"data\n", 0o600)
        root_fd = open_absolute_directory(root)
        try:
            try:
                generated_stage_manifest_at(root_fd, time.monotonic() - 1)
            except AssertionError as error:
                require(str(error) == "outside manifest scan-time bound",
                        "generated stage aggregate deadline diagnostic")
            else:
                fail("expired generated stage aggregate deadline accepted")
        finally:
            os.close(root_fd)

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); oversized = root / "oversized"
        with oversized.open("wb") as stream:
            stream.truncate(OUTSIDE_MANIFEST_MAX_BYTES + 1)
        oversized.chmod(0o600)
        root_fd = open_absolute_directory(root)
        original_read = os.read
        try:
            def reject_read(_fd: int, _size: int) -> bytes:
                fail("oversized generated stage descriptor read")
            os.read = reject_read
            try:
                generated_stage_manifest_at(root_fd, time.monotonic() + 30)
            except AssertionError as error:
                require(str(error) == "bounded file size",
                        "generated stage byte bound before read")
            else:
                fail("oversized generated stage file accepted")
        finally:
            os.read = original_read
            os.close(root_fd)

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); atomic_write(root / "file", b"old\n", 0o600)
        atomic_write(root / "replacement", b"new\n", 0o600)
        root_fd = open_absolute_directory(root)
        original_hash = hash_file_descriptor
        try:
            def replace_during_hash(fd: int, limit: int | None = None,
                                    deadline: float | None = None):
                result = original_hash(fd, limit, deadline)
                os.rename("file", "old-file", src_dir_fd=root_fd, dst_dir_fd=root_fd)
                os.rename("replacement", "file", src_dir_fd=root_fd, dst_dir_fd=root_fd)
                return result
            globals()["hash_file_descriptor"] = replace_during_hash
            try:
                generated_stage_manifest_at(root_fd, time.monotonic() + 30)
            except AssertionError as error:
                require(str(error) == "generated stage pathname replacement",
                        "generated stage pathname replacement diagnostic")
            else:
                fail("generated stage pathname replacement mutant survived")
        finally:
            globals()["hash_file_descriptor"] = original_hash
            os.close(root_fd)


def check_validate_delivery_stage_deadline() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve()
        delivery = root / "delivery"; (delivery / "stage").mkdir(parents=True)
        repo = root / "repo"; (repo / "pack/generated").mkdir(parents=True)
        delivery_fd = open_absolute_directory(delivery)
        repo_fd = open_absolute_directory(repo)
        original_manifest = generated_stage_manifest_at
        original_validate = validate_delivery_stage_records
        deadlines = []
        try:
            def capture_manifest(_fd: int, deadline: float) -> dict:
                require(isinstance(deadline, (int, float)) and not isinstance(deadline, bool)
                        and time.monotonic() < deadline < float("inf"),
                        "recovery generated stage finite deadline")
                deadlines.append(deadline)
                return {}

            def accept_empty(_journal: dict, staged: dict, current: dict) -> None:
                require(staged == {} and current == {},
                        "recovery generated stage captured manifests")

            globals()["generated_stage_manifest_at"] = capture_manifest
            globals()["validate_delivery_stage_records"] = accept_empty
            validate_delivery_stage_at({}, delivery_fd, repo_fd)
            require(len(deadlines) == 2 and deadlines[0] == deadlines[1],
                    "recovery generated stage shared deadline")
        finally:
            globals()["validate_delivery_stage_records"] = original_validate
            globals()["generated_stage_manifest_at"] = original_manifest
            os.close(repo_fd); os.close(delivery_fd)


def remove_tree_at(root_fd: int, relative: str, missing_ok: bool = False) -> None:
    parent_fd, name = open_parent_fd(root_fd, relative)
    try:
        try:
            directory_fd = os.open(name, DIRECTORY_FLAGS, dir_fd=parent_fd)
        except FileNotFoundError:
            if missing_ok:
                return
            raise
        try:
            validate_directory_fd(directory_fd)
            for entry in sorted(os.listdir(directory_fd)):
                info = os.stat(entry, dir_fd=directory_fd, follow_symlinks=False)
                if stat.S_ISDIR(info.st_mode):
                    remove_tree_at(directory_fd, entry)
                else:
                    secure_file_info(info)
                    os.unlink(entry, dir_fd=directory_fd)
                    os.fsync(directory_fd)
        finally:
            os.close(directory_fd)
        os.rmdir(name, dir_fd=parent_fd)
        os.fsync(parent_fd)
    finally:
        os.close(parent_fd)


def prune_empty_parents(root_fd: int, relative: str, stop_depth: int = 0) -> None:
    parts = list(relative_components(relative)[:-1])
    while len(parts) > stop_depth:
        parent_parts = tuple(parts[:-1])
        parent_fd = open_dir_components(root_fd, parent_parts)
        try:
            directory_fd = os.open(parts[-1], DIRECTORY_FLAGS, dir_fd=parent_fd)
            try:
                validate_directory_fd(directory_fd)
                if os.listdir(directory_fd):
                    return
            finally:
                os.close(directory_fd)
            os.rmdir(parts[-1], dir_fd=parent_fd)
            os.fsync(parent_fd)
        finally:
            os.close(parent_fd)
        parts.pop()


def fixture_lock() -> tuple[int, str]:
    fd, name = tempfile.mkstemp(prefix="workflow-artifact-fixture-lock.")
    os.fchmod(fd, 0o600)
    acquire_owner_lock(fd)
    return fd, name


def atomic_write(path: Path, data: bytes, mode: int, writer=os.write) -> None:
    fixture_path = Path(os.path.realpath(path.parent)) / path.name
    root_fd, relative = absolute_descriptor(fixture_path)
    lock_fd, lock_name = fixture_lock()
    try:
        atomic_write_at(root_fd, relative, data, mode, lock_fd, writer)
    finally:
        os.close(lock_fd)
        os.unlink(lock_name)
        os.close(root_fd)


def canonical_root() -> Path:
    if (CANDIDATE_ROOT / MODULE_REL).is_file():
        return CANDIDATE_ROOT
    fail("candidate root is incomplete")


def split_markdown_cells(row: str) -> list[str]:
    cells = []
    cell_start = 0
    backslashes = 0
    for index, character in enumerate(row):
        if character == "|":
            if backslashes % 2 == 0:
                cells.append(row[cell_start:index])
                cell_start = index + 1
            backslashes = 0
        elif character == "\\":
            backslashes += 1
        else:
            backslashes = 0
    cells.append(row[cell_start:])
    return cells


def check_markdown_backslash_parity() -> None:
    for count in range(5):
        row = "left" + "\\" * count + "|right"
        cells = split_markdown_cells(row)
        if count % 2 == 0:
            require(cells == ["left" + "\\" * count, "right"],
                    f"even backslash delimiter: {count}")
        else:
            require(len(cells) == 1
                    and cells[0].replace("\\|", "|") == "left" + "\\" * (count - 1) + "|right",
                    f"odd backslash escape: {count}")


def markdown_rows(text: str, prefix: str) -> list[list[str]]:
    rows = []
    for line in text.splitlines():
        if line.startswith(f"| `{prefix}"):
            cells = split_markdown_cells(line.strip().strip("|"))
            rows.append([cell.strip().replace("\\|", "|").replace("`", "") for cell in cells])
    return rows


def markdown_table(text: str, header: str) -> list[list[str]]:
    lines = text.splitlines()
    start = lines.index(header)
    require(start + 2 < len(lines) and lines[start + 1].startswith("| ---"), f"table separator: {header}")
    rows = []
    for line in lines[start + 2:]:
        if not line.startswith("|"):
            break
        cells = split_markdown_cells(line.strip().strip("|"))
        rows.append([cell.strip().replace("\\|", "|").replace("`", "") for cell in cells])
    return rows


def diagnostic_table_rows(module: str) -> list[list[str]]:
    return markdown_table(module, "|Reason ID|Boundary ID|Meaning|Next route|")


def check_golden_vector() -> None:
    slug = b"demo"
    binding = bytes(32)
    handle = hashlib.sha256(
        b"devrites.workflow-source.v1\0" + len(slug).to_bytes(4, "big") + slug + binding
    ).hexdigest()
    require(handle == "1557f28b7dbf713ae3828b0dc4e914702ba34063f65393d4f8b57d99bc6af3ad", "handle golden vector")
    content = b'print("ok")\n'
    content_hash = hashlib.sha256(content).digest()
    path = b"scripts/prove.py"
    identity = hashlib.sha256(
        b"devrites.workflow-identity.v1\0" + (1).to_bytes(4, "big")
        + len(path).to_bytes(4, "big") + path + (0o755).to_bytes(4, "big") + content_hash
    ).hexdigest()
    require(identity == "ce333944056552cf645c36cd03b5cd65774d167b5e920118639c6062e29f5c82", "identity golden vector")
    require(content_hash.hex() == "3a66aebdedbad3cf107d24e72a07d4b735819b1cf4020fdd922f63c064708172", "content golden vector")


def validate_limits(targets: int, per: int, aggregate: int, files: int, diagnostic: int,
                    lines: int, epochs: int, command: int, total: int, grace: int) -> None:
    values = (targets, per, aggregate, files, diagnostic, lines, epochs, command, total, grace)
    require(all(isinstance(v, int) and not isinstance(v, bool) and 0 <= v <= 2**63 - 1 for v in values), "checked integer")
    require(targets >= 1 and per >= 1 and aggregate >= 1, "positive target limits")
    require(files >= 3 * targets + 6, "transaction cardinality")
    require(diagnostic == 256, "diagnostic bound")
    require(epochs >= 3, "attempt bound")
    require(lines >= 30 + targets + epochs and lines <= 280, "journal headroom")
    require(command > grace > 0 and total >= command, "proof timing relation")


def check_limit_boundaries() -> None:
    validate_limits(1, 1, 1, 9, 256, 34, 3, 2, 2, 1)
    invalid = [
        (1, 1, 1, 8, 256, 34, 3, 2, 2, 1),
        (1, 1, 1, 9, 255, 34, 3, 2, 2, 1),
        (1, 1, 1, 9, 256, 33, 3, 2, 2, 1),
        (1, 1, 1, 9, 256, 34, 2, 2, 2, 1),
        (1, 1, 1, 9, 256, 34, 3, 1, 2, 1),
        (1, 1, 1, 9, 256, 34, 3, 3, 2, 1),
        (2**63, 1, 1, 9, 256, 34, 3, 2, 2, 1),
    ]
    for row in invalid:
        try:
            validate_limits(*row)
        except AssertionError:
            continue
        fail(f"invalid limits accepted: {row}")


def check_complete_writes() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        path = Path(tmp) / "out"
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        calls = []
        def partial(raw_fd, view):
            n = min(2, len(view))
            calls.append(n)
            return os.write(raw_fd, view[:n])
        complete_write(fd, b"abcdefg", partial)
        os.close(fd)
        require(path.read_bytes() == b"abcdefg" and len(calls) == 4, "partial positive writes")
        bad = [True, "1", 0, -1, 99]
        for value in bad:
            bad_path = Path(tmp) / f"bad-{len(str(value))}-{bad.index(value)}"
            fd = os.open(bad_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            try:
                try:
                    complete_write(fd, b"x", lambda _fd, _view, value=value: value)
                except OSError:
                    pass
                else:
                    fail(f"invalid write progress accepted: {value!r}")
            finally:
                os.close(fd)


EVIDENCE_FIELD_ORDER = (
    "transaction_id", "attempt_epoch", "attempt_id", "generation",
    "owned_section_preimage_sha256", "vet_readiness_binding", "source_handle",
    "identity_digest", "state", "boundary_id", "reason_id", "next_route",
    "product_candidate_digest", "product_readiness_binding", "built_slice_count",
    "caller_return_phase", "caller_return_next_action",
)
DEFAULT_IDENTITY = "2" * 64
DEFAULT_TARGETS = (
    ("00000000", "scripts/a.py", "0600", "6" * 64, "present", "0600", "7" * 64, "wbak:00000000", "PROVED"),
    ("00000001", "scripts/b.py", "0755", "8" * 64, "absent", "NONE", "NONE", "NONE", "PROVED"),
)
JOURNAL_STATE_PATTERN = re.compile(
    r"(?:PREPARING|PREPARED|INSTALLED|PROVING|PROVED|ROLLED_BACK|FAILED|CLEANED|EXHAUSTED"
    r"|(?:RETRY_PREPARING|INSTALLING|ROLLING_BACK|FAILURE_CLEANING|CLEANING)\([0-9]+\))"
)
JOURNAL_ROUTES = set(EXPECTED_ROUTES.values()) | {
    "OFFLINE_RECOVERY", "RESUME_CLEANUP", "PROVE_AND_RETURN", "WAIT_ACTIVE_OWNER",
    "BLOCKED_EXHAUSTED", "BLOCKED_GATE",
}


def parse_quoted_row(line: str, width: int, label: str, first_unquoted: bool = False) -> tuple[str, ...]:
    require(line.startswith("| ") and line.endswith(" |"), f"{label} row framing")
    raw_cells = split_markdown_cells(line.strip().strip("|"))
    require(len(raw_cells) == width, f"{label} row width")
    cells = []
    for index, raw in enumerate(raw_cells):
        cell = raw.strip()
        if first_unquoted and index == 0:
            require(re.fullmatch(r"[1-9][0-9]*", cell) is not None, f"{label} first cell grammar")
            cells.append(cell)
            continue
        require(re.fullmatch(r"`[^`\n]+`", cell) is not None, f"{label} cell grammar")
        cells.append(cell[1:-1].replace("\\|", "|"))
    return tuple(cells)


def valid_journal_state(value: str) -> bool:
    return JOURNAL_STATE_PATTERN.fullmatch(value) is not None


def validate_target_rows(targets: tuple[tuple[str, str, str, str, str, str, str, str, str], ...]) -> None:
    require(len(targets) > 0, "evidence target cardinality")
    for position, row in enumerate(targets):
        require(len(row) == 9, "evidence target width")
        index, logical_path, mode, content_hash, preimage, preimage_mode, preimage_hash, backup, result = row
        require(index == f"{position:08x}", "evidence target index")
        require(normalize_workflow_path(logical_path) == logical_path, "evidence target path")
        require(re.fullmatch(r"0[0-7]{3}", mode) is not None, "evidence target mode")
        require(re.fullmatch(r"[0-9a-f]{64}", content_hash) is not None, "evidence target hash")
        require(preimage in {"present", "absent"}, "evidence target preimage")
        if preimage == "present":
            require(re.fullmatch(r"0[0-7]{3}", preimage_mode) is not None, "evidence preimage mode")
            require(re.fullmatch(r"[0-9a-f]{64}", preimage_hash) is not None, "evidence preimage hash")
            require(re.fullmatch(r"wbak:[0-9a-f]{8}", backup) is not None, "evidence backup handle")
        else:
            require((preimage_mode, preimage_hash, backup) == ("NONE", "NONE", "NONE"), "absent evidence preimage")
        require(valid_journal_state(result), "evidence target result")


def validate_attempt_rows(attempts: tuple[tuple[int, str, str, str, str, str, str], ...], identity: str) -> None:
    for position, row in enumerate(attempts, 1):
        require(len(row) == 7, "evidence attempt width")
        epoch, attempt_id, fingerprint, reason, boundary, progress, result = row
        require(epoch == position, "evidence attempt epoch")
        require(attempt_id == f"wta:{identity}:{epoch:08x}", "evidence attempt ID")
        require(fingerprint == "NONE" or re.fullmatch(r"[0-9a-f]{64}", fingerprint) is not None, "evidence attempt fingerprint")
        require(reason == "NONE" or reason in EXPECTED_REASON_IDS, "evidence attempt reason")
        require(boundary in EXPECTED_BOUNDARY_IDS, "evidence attempt boundary")
        require(progress in {"resolved", "no-progress", "pending"}, "evidence attempt progress")
        require(valid_journal_state(result), "evidence attempt result")


def parse_prior_owned_section(prior: str) -> tuple[dict[str, str], tuple[tuple[str, ...], ...], tuple[tuple[int, str, str, str, str, str, str], ...]]:
    lines = prior.splitlines()
    require(len(lines) >= 31 and lines[:4] == [START, "## Workflow Artifact journal", "DevRites contract: devrites.workflow-artifact-journal.v1", ""], "prior evidence preamble")
    require(lines[4:6] == ["| Field | Value |", "| --- | --- |"], "prior evidence field header")
    cursor = 6
    fields = []
    while cursor < len(lines) and lines[cursor] != "":
        line = lines[cursor]
        require(line.startswith("| ") and line.endswith(" |"), "prior evidence field row framing")
        raw_cells = split_markdown_cells(line.strip().strip("|"))
        require(len(raw_cells) == 2, "prior evidence field row width")
        name, quoted_value = (cell.strip() for cell in raw_cells)
        require(re.fullmatch(r"[a-z0-9_]+", name) is not None,
                "prior evidence field name")
        require(re.fullmatch(r"`[^`\n]+`", quoted_value) is not None,
                "prior evidence field value")
        fields.append((name, quoted_value[1:-1].replace("\\|", "|")))
        cursor += 1
    record = dict(fields)
    expected_fields = list(EVIDENCE_FIELD_ORDER)
    if record.get("state") == "EXHAUSTED":
        expected_fields.insert(expected_fields.index("next_route") + 1, "exhaustion_cause")
    require([name for name, _value in fields] == expected_fields
            and len(record) == len(fields), "prior evidence field order")
    require(lines[cursor] == "", "prior evidence field terminator")
    cursor += 1
    target_header = "| Index | Path | Mode | Content SHA-256 | Preimage | Preimage mode | Preimage SHA-256 | Backup handle | Result |"
    require(lines[cursor:cursor + 2] == [target_header, "| --- | --- | --- | --- | --- | --- | --- | --- | --- |"], "prior evidence target header")
    cursor += 2
    targets = []
    while cursor < len(lines) and lines[cursor] != "":
        targets.append(parse_quoted_row(lines[cursor], 9, "prior target"))
        cursor += 1
    require(cursor < len(lines) and lines[cursor] == "", "prior evidence target terminator")
    cursor += 1
    attempt_header = "| Epoch | Attempt ID | Failure fingerprint | Reason | Boundary | Progress | Result |"
    require(lines[cursor:cursor + 2] == [attempt_header, "| --- | --- | --- | --- | --- | --- | --- |"], "prior evidence attempt header")
    cursor += 2
    attempts = []
    while cursor < len(lines) and lines[cursor] != END:
        cells = parse_quoted_row(lines[cursor], 7, "prior attempt", first_unquoted=True)
        require(re.fullmatch(r"[1-9][0-9]*", cells[0]) is not None, "prior attempt epoch grammar")
        attempts.append((int(cells[0]), *cells[1:]))
        cursor += 1
    require(cursor == len(lines) - 1 and lines[cursor] == END, "prior evidence complete consumption")
    identity = record["identity_digest"]
    require(re.fullmatch(r"[0-9a-f]{64}", identity) is not None, "prior identity digest")
    require(record["transaction_id"] == f"wtx:{identity}", "prior transaction ID")
    require(re.fullmatch(r"[1-9][0-9]*", record["attempt_epoch"]) is not None, "prior attempt epoch")
    epoch = int(record["attempt_epoch"])
    require(record["attempt_id"] == f"wta:{identity}:{epoch:08x}", "prior current attempt ID")
    require(re.fullmatch(r"0|[1-9][0-9]*", record["generation"]) is not None, "prior generation")
    require(record["owned_section_preimage_sha256"] == "ABSENT" or re.fullmatch(r"[0-9a-f]{64}", record["owned_section_preimage_sha256"]) is not None, "prior preimage binding")
    for name in ("vet_readiness_binding", "product_candidate_digest", "product_readiness_binding"):
        require(re.fullmatch(r"[0-9a-f]{64}", record[name]) is not None, f"prior digest field: {name}")
    require(re.fullmatch(r"wsrc:[0-9a-f]{64}", record["source_handle"]) is not None, "prior source handle")
    require(valid_journal_state(record["state"]), "prior journal state")
    require(record["boundary_id"] in EXPECTED_BOUNDARY_IDS, "prior boundary")
    require(record["reason_id"] == "NONE" or record["reason_id"] in EXPECTED_REASON_IDS, "prior reason")
    require(record["next_route"] in JOURNAL_ROUTES, "prior route")
    if record["state"] == "EXHAUSTED":
        require(record["exhaustion_cause"] in {"same-fingerprint-count", "total-epoch-limit"},
                "terminal exhaustion cause")
    else:
        require("exhaustion_cause" not in record, "nonterminal exhaustion cause absent")
    require(re.fullmatch(r"0|[1-9][0-9]*", record["built_slice_count"]) is not None, "prior built count")
    require(re.fullmatch(r"[a-z][a-z0-9-]*", record["caller_return_phase"]) is not None, "prior return phase")
    require(record["caller_return_next_action"].startswith("/") and len(record["caller_return_next_action"].encode()) <= 4096, "prior return action")
    target_tuple = tuple(targets)
    attempt_tuple = tuple(attempts)
    validate_target_rows(target_tuple)
    validate_attempt_rows(attempt_tuple, identity)
    require(epoch == max((row[0] for row in attempt_tuple), default=1), "prior active attempt epoch")
    if record["state"] == "FAILED":
        require(attempt_tuple and attempt_tuple[-1][6] == "FAILED", "failed prior attempt record")
    return record, target_tuple, attempt_tuple


def owned_section_bounds(text: str, starts: list[int], ends: list[int]) -> tuple[int, int]:
    require(starts[0] < ends[0], "marker ordering")
    end_line = text.find("\n", ends[0])
    require(end_line >= 0, "end marker newline")
    return starts[0], end_line + 1


def evidence_record(state: str, generation: int, preimage: str,
                    attempts: tuple[tuple[int, str, str, str, str, str, str], ...],
                    exhaustion_cause: str | None) -> dict[str, str]:
    epoch = max((row[0] for row in attempts), default=1)
    boundary = "WA-B013-SUCCESS-CLEANUP" if state == "CLEANED" else "WA-B010-PROVE" if state == "FAILED" else "WA-B005-JOURNAL"
    reason = "WA-R013-PROOF-FAILED" if state == "FAILED" else "NONE"
    route = "VERIFY_EXISTING" if state == "CLEANED" else "OFFLINE_RECOVERY" if state == "FAILED" else "ROOT_TRANSACTION"
    require((state == "EXHAUSTED") == (exhaustion_cause is not None),
            "terminal-owned exhaustion cause")
    if exhaustion_cause is not None:
        require(exhaustion_cause in {"same-fingerprint-count", "total-epoch-limit"},
                "exhaustion cause value")
    record = {
        "transaction_id": f"wtx:{DEFAULT_IDENTITY}",
        "attempt_epoch": str(epoch),
        "attempt_id": f"wta:{DEFAULT_IDENTITY}:{epoch:08x}",
        "generation": str(generation),
        "owned_section_preimage_sha256": preimage,
        "vet_readiness_binding": "3" * 64,
        "source_handle": "wsrc:" + "4" * 64,
        "identity_digest": DEFAULT_IDENTITY,
        "state": state,
        "boundary_id": boundary,
        "reason_id": reason,
        "next_route": route,
        "product_candidate_digest": "5" * 64,
        "product_readiness_binding": "9" * 64,
        "built_slice_count": "7",
        "caller_return_phase": "prove",
        "caller_return_next_action": "/rite-prove demo",
    }
    if exhaustion_cause is not None:
        fields = list(record.items())
        index = next(i for i, (name, _value) in enumerate(fields) if name == "next_route") + 1
        fields.insert(index, ("exhaustion_cause", exhaustion_cause))
        record = dict(fields)
    return record


def owned_section(existing: bytes, state: str, generation: int,
                  attempts: tuple[tuple[int, str, str, str, str, str, str], ...] = (),
                  targets: tuple[tuple[str, str, str, str, str, str, str, str, str], ...] = DEFAULT_TARGETS,
                  line_limit: int = 280, exhaustion_cause: str | None = None) -> bytes:
    text = existing.decode("utf-8")
    starts = [m.start() for m in re.finditer(rf"(?m)^{re.escape(START)}$", text)]
    ends = [m.start() for m in re.finditer(rf"(?m)^{re.escape(END)}$", text)]
    require(text.count(START) == len(starts) and text.count(END) == len(ends), "malformed evidence marker")
    require(len(starts) <= 1 and len(ends) <= 1 and len(starts) == len(ends), "marker cardinality")
    bindings = re.findall(r"(?m)^Candidate SHA-256: [0-9a-f]{64}$", text)
    require(len(bindings) == 1, "single candidate binding")
    preimage = "ABSENT"
    prefix = text
    suffix = ""
    validate_target_rows(targets)
    validate_attempt_rows(attempts, DEFAULT_IDENTITY)
    if starts:
        owned_start, owned_end = owned_section_bounds(text, starts, ends)
        prior_bytes = text[owned_start:owned_end].encode()
        preimage = sha(prior_bytes)
        prior_record, prior_targets, prior_attempts = parse_prior_owned_section(prior_bytes.decode())
        require(generation == int(prior_record["generation"]) + 1, "monotonic journal generation")
        require(prior_targets == targets, "immutable target rows")
        require(prior_attempts == attempts[:len(prior_attempts)], "immutable attempt rows")
        prefix, suffix = text[:owned_start], text[owned_end:]
    else:
        require(generation == 1, "initial journal generation")
    record = evidence_record(state, generation, preimage, attempts, exhaustion_cause)
    expected_fields = list(EVIDENCE_FIELD_ORDER)
    if state == "EXHAUSTED":
        expected_fields.insert(expected_fields.index("next_route") + 1, "exhaustion_cause")
    require(tuple(record) == tuple(expected_fields), "complete evidence field order")
    field_lines = "".join(f"| {name} | `{value}` |\n" for name, value in record.items())
    target_lines = "".join("| " + " | ".join(f"`{cell}`" for cell in row) + " |\n" for row in targets)
    attempt_lines = "".join(
        f"| {epoch} | `{attempt_id}` | `{fingerprint}` | `{reason}` | `{boundary}` | `{progress}` | `{result}` |\n"
        for epoch, attempt_id, fingerprint, reason, boundary, progress, result in attempts
    )
    section = (
        f"{START}\n## Workflow Artifact journal\n"
        "DevRites contract: devrites.workflow-artifact-journal.v1\n\n"
        "| Field | Value |\n| --- | --- |\n" + field_lines + "\n"
        "| Index | Path | Mode | Content SHA-256 | Preimage | Preimage mode | Preimage SHA-256 | Backup handle | Result |\n"
        "| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n" + target_lines + "\n"
        "| Epoch | Attempt ID | Failure fingerprint | Reason | Boundary | Progress | Result |\n"
        "| --- | --- | --- | --- | --- | --- | --- |\n" + attempt_lines +
        f"{END}\n"
    )
    if not starts:
        result = existing + (b"" if existing.endswith(b"\n") else b"\n") + b"\n" + section.encode()
    else:
        result = prefix.encode() + section.encode() + suffix.encode()
    require(len(result.decode().splitlines()) <= line_limit, "evidence line budget")
    return result


def check_evidence_ownership() -> None:
    prefix = b"# Evidence\nCandidate SHA-256: " + b"1" * 64 + b"\nEVID-777 keep\n"
    first = owned_section(prefix, "PREPARING", 1)
    first_start = first.index(START.encode())
    first_end = first.index((END + "\n").encode(), first_start) + len(END) + 1
    second = owned_section(first + b"suffix\n", "CLEANED", 2)
    require(second.startswith(prefix), "evidence prefix preservation")
    require(second.endswith(b"suffix\n"), "evidence suffix preservation")
    require(second.count(b"Candidate SHA-256:") == 1, "candidate binding duplication")
    require(f"| owned_section_preimage_sha256 | `{sha(first[first_start:first_end])}` |".encode() in second, "owned section preimage binding")
    for field in EVIDENCE_FIELD_ORDER:
        require(second.count(f"| {field} |".encode()) == 1, f"complete evidence field: {field}")
    require(b"| exhaustion_cause |" not in second, "nonterminal cause omitted")
    exhausted = owned_section(first, "EXHAUSTED", 2, exhaustion_cause="same-fingerprint-count")
    require(exhausted.count(b"| exhaustion_cause | `same-fingerprint-count` |") == 1,
            "terminal cause present exactly once")
    for state, cause in (("EXHAUSTED", None), ("FAILED", "same-fingerprint-count")):
        try:
            owned_section(prefix, state, 1, exhaustion_cause=cause)
        except AssertionError:
            pass
        else:
            fail(f"invalid exhaustion cause ownership accepted: {state}/{cause}")
    require(second.count(b"| `00000000` |") == 1 and second.count(b"| `00000001` |") == 1, "complete evidence target rows")
    for stale_generation in (1, 3):
        try:
            owned_section(first, "FAILED", stale_generation)
        except AssertionError:
            pass
        else:
            fail(f"stale journal generation accepted: {stale_generation}")
    for malformed in (
        prefix + f"{START}\n{START}\n{END}\n".encode(),
        prefix + f"{END}\n{START}\n".encode(),
        prefix + f"{START}\n{END}".encode(),
    ):
        try:
            owned_section(malformed, "FAILED", 3)
        except AssertionError:
            pass
        else:
            fail("malformed evidence markers accepted")


def check_flock() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp) / ".workflow-artifact-sources"
        old_umask = os.umask(0o077)
        try:
            root.mkdir(mode=0o700)
            lock = root / ".owner.lock"
            fd = os.open(lock, os.O_RDWR | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600)
        finally:
            os.umask(old_umask)
        acquire_owner_lock(fd)
        probe = subprocess.run(
            [sys.executable, "-c", "import fcntl,os,sys; f=os.open(sys.argv[1],os.O_RDWR|os.O_NOFOLLOW);\ntry: fcntl.flock(f,fcntl.LOCK_EX|fcntl.LOCK_NB)\nexcept BlockingIOError: raise SystemExit(7)", str(lock)],
            check=False,
        )
        require(probe.returncode == 7, "flock contention")
        for primitive in ("lockf", "F_SETLK"):
            try:
                acquire_owner_lock(fd, primitive)
            except AssertionError:
                pass
            else:
                fail(f"alternate lock primitive accepted: {primitive}")
        os.close(fd)
        reacquired = os.open(lock, os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC)
        acquire_owner_lock(reacquired)
        os.close(reacquired)
        require(stat.S_IMODE(root.stat().st_mode) == 0o700 and stat.S_IMODE(lock.stat().st_mode) == 0o600, "lock metadata")


def check_process_group_timeout() -> None:
    timeout = 3.0
    code = "import os,subprocess,sys,time; subprocess.Popen([sys.executable,'-c','import time; time.sleep(30)']); print('ready',flush=True); os.close(1); os.close(2); time.sleep(30)"
    ok, reason, output = run_proof_command([sys.executable, "-c", code], "ready", timeout, 0.5, 256)
    require(not ok and reason == "timeout" and output == b"ready\n", "proof group timeout and reap")
    with tempfile.TemporaryDirectory() as tmp:
        markers = Path(tmp) / "termination"
        atomic_write(markers, b"", 0o600)
        grace = 0.5
        barrier_read, barrier_write = os.pipe()
        fixture = "\n".join((
            "import os,signal,sys",
            "path=sys.argv[1]",
            "barrier=int(sys.argv[2])",
            "def mark(value):",
            " with open(path,'ab',buffering=0) as stream: stream.write(value)",
            "def terminate(_signal,_frame):",
            " mark(b'TERM\\n')",
            " mark(b'grace\\n')",
            " os.write(barrier,b'x')",
            "signal.signal(signal.SIGTERM,terminate)",
            "print('ready',flush=True)",
            "while True: signal.pause()",
        ))
        events = []
        def synchronize(event):
            events.append(event)
            if event == "TERM":
                require(os.read(barrier_read, 1) == b"x",
                        "TERM handler completion barrier")
        logical_clock = [0]
        ok, reason, output = run_proof_command(
            [sys.executable, "-c", fixture, str(markers), str(barrier_write)],
            "ready", timeout, grace, 256, synchronize,
            lambda: logical_clock[0], lambda: None,
            pass_fds=(barrier_write,),
        )
        require(not ok and reason == "timeout" and output == b"ready\n", "graceful escalation timeout route")
        require(markers.read_bytes() == b"TERM\ngrace\n", "ordered TERM and grace survival before KILL")
        require(events == ["TERM", "KILL", "REAP"], "TERM grace KILL reap synchronization")

        markers.write_bytes(b"")
        events.clear()
        try:
            run_proof_command(
                [sys.executable, "-c", fixture, str(markers), str(barrier_write)],
                "ready", timeout, grace, 256, synchronize,
                lambda: logical_clock[0], lambda: logical_clock.__setitem__(0, logical_clock[0] + 10**9),
                pass_fds=(barrier_write,),
            )
        except AssertionError as error:
            require(str(error) == "post-termination logical delay",
                    "post-termination delay mutant reason")
        else:
            fail("post-termination delay mutant survived")
        require(markers.read_bytes() == b"TERM\ngrace\n"
                and events == ["TERM", "KILL", "REAP"],
                "delay mutant preserves exact termination behavior")
        os.close(barrier_write); os.close(barrier_read)


def check_module_and_corpus(root: Path) -> None:
    module = (root / MODULE_REL).read_text()
    require("devrites.workflow-artifact-admission.v1" in module, "admission version")
    require("devrites.workflow-artifact-journal.v1" in module, "journal version")
    require("**transaction journal**" in module and "**evidence journal**" in module
            and "crash/recovery\nauthority" in module
            and "Neither aliases the other" in module,
            "canonical journal terminology separation")
    require("fifo/socket same base; block/character add" in module
            and "nonnegative integer\nnon-bool `st_rdev`" in module
            and "Reject other types before acceptance" in module,
            "canonical closed special-object identity")
    require("`<!-- ` + `devrites` +" in module and "`-workflow-artifact-journal:end -->`" in module, "exact marker fragments")
    require("only Python" in module and "fcntl.flock" in module, "canonical lock domain")
    require("both source and destination directory handles" in module, "relative replacement handles")
    require("Cold-resume migration" not in module and "sole materializer" not in module, "stale authority removed")
    require("There is no actor-history migration" in module, "actor-history migration removed")
    require("trusted Vet-approved admitted argv command and its descendants" in module
            and "in one fresh process group, where they remain until exit" in module
            and "adds no network or filesystem sandbox" in module
            and "makes no deliberate detached-session" in module
            and "Any surviving group member" in module, "admitted process-group contract")
    require("only a partial or empty" in module and "remaining suffix of that order" in module
            and "re-authenticate the intent before every remaining deletion" in module,
            "stale cleanup reachable suffix contract")
    require(module.index("stale directory is absent") < module.index("truncate the held lock intent")
            < module.index("read back\nexact empty content"), "stale intent clears after absent directory and parent sync")
    require("WA-FIX-P[A-Z0-9]" in module and "WA-FIX-F[A-Z0-9]" in module
            and "DevRites workflow reference:" in module
            and "WA-(BEH|IF|FIX|FAIL)" not in module, "current admission reference grammar")
    require("exhaustion_cause=<same-fingerprint-count|total-epoch-limit>" in module
            and "epoch exhaustion\nnever claims" in module
            and "<NONE|same-fingerprint-count|total-epoch-limit>" not in module,
            "conditional terminal exhaustion cause contract")
    require("next_action=none — technical recovery exhausted; requires new evidence or changed failure conditions" in module
            and "technical recovery exhausted for <causal fingerprint>" not in module,
            "frozen exhaustion action")
    require("WA-B016-PRODUCT-SEPARATION" not in module,
            "exact product-separation boundary")
    exact_manifest_contract = lambda text: text.count(OUTSIDE_MANIFEST_CONTRACT) == 1
    require(exact_manifest_contract(module), "exact outside-manifest contract")
    require(module.index("loss follows the same branches.")
            < module.index(OUTSIDE_MANIFEST_CONTRACT)
            < module.index("`PROVING` runs"),
            "outside-manifest contract locality")
    require(not exact_manifest_contract(
                module.replace(OUTSIDE_MANIFEST_CONTRACT, "", 1),
            ), "outside-manifest contract omission mutant")
    for index, cell in enumerate(OUTSIDE_MANIFEST_CONTRACT_CELLS):
        require(cell in OUTSIDE_MANIFEST_CONTRACT,
                f"outside-manifest contract cell fixture: {index}")
        mutant_contract = OUTSIDE_MANIFEST_CONTRACT.replace(
            cell, f"MUTANT-{index}", 1,
        )
        require(not exact_manifest_contract(
                    module.replace(OUTSIDE_MANIFEST_CONTRACT, mutant_contract, 1),
                ), f"outside-manifest contract cell mutant: {index}")
    operation_rows = markdown_rows(module, "WA-OP-")
    ops = [row[0] for row in operation_rows]
    require(ops == EXPECTED_OPS, f"operation table: {ops}")
    require(len({tuple(row) for row in operation_rows}) == 16, "operation row uniqueness")
    require(validate_atomic_ownership_contract(module) == 2, "atomic ownership operation count")
    require(reject_atomic_ownership_mutants(module) == 20, "atomic ownership mutation count")
    diagnostic_rows = diagnostic_table_rows(module)
    require(len(diagnostic_rows) == 22, "diagnostic row count")
    for row in diagnostic_rows:
        require(len(row) == 4, f"diagnostic width: {row}")
        line = f"WORKFLOW_ARTIFACT_FAILURE reason_id={row[0]} boundary_id={row[1]} next_route={row[3]}\n"
        require(line.isascii() and len(line.encode()) <= 256, f"diagnostic bound: {row[0]}")
    fallback = "WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R009-STATE-AMBIGUOUS boundary_id=WA-B005-JOURNAL next_route=OFFLINE_RECOVERY\n"
    require(len(fallback.encode()) <= 256, "fallback diagnostic bound")
    corpus = json.loads((root / CORPUS_REL).read_text())
    scenarios = corpus["scenarios"]
    require([row["id"] for row in scenarios] == EXPECTED_SCENARIOS, "scenario IDs/order")
    require({row["id"]: row["expected_route"] for row in scenarios} == EXPECTED_ROUTES, "scenario routes")
    require(all(set(row) == SCENARIO_FIELDS for row in scenarios), "exact scenario field maps")
    require(exact_map_digest(scenarios) == CORPUS_SCENARIOS_SHA256, "complete scenario corpus map")
    scenario_mutations = 0
    for row_index, row in enumerate(scenarios):
        for field in sorted(SCENARIO_FIELDS):
            mutant = json.loads(json.dumps(scenarios))
            if isinstance(row[field], list):
                mutant[row_index][field].append("MUTANT")
            else:
                mutant[row_index][field] += "-MUTANT"
            require(exact_map_digest(mutant) != CORPUS_SCENARIOS_SHA256,
                    f"scenario map cell mutant survived: {row['id']}:{field}")
            scenario_mutations += 1
    require(scenario_mutations == 140, "scenario map mutation count")
    for row in scenarios:
        require(row["durable_consequence"] and row["forbidden_actions"], f"scenario consequence: {row['id']}")
    adapter_rows = markdown_table(module, "|Canonical adapter|Entry trigger|Canonical action|Return cursor|")
    require(len(adapter_rows) == 10 and all(len(row) == 4 for row in adapter_rows), "canonical adapter table")
    require(exact_map_digest(adapter_rows) == ADAPTER_MAP_SHA256, "complete adapter map")
    require(reject_table_cell_mutants(adapter_rows, ADAPTER_MAP_SHA256, "adapter map") == 40,
            "adapter map mutation count")
    adapter_map = {row[0]: row[1:] for row in adapter_rows}
    forbidden = ("PREPARING → PREPARED", ".owner.lock", ".stale-cleanup", "attempt epoch begins")
    policy_restatements = (
        "Wait for the current owner to release its lock, then continue.",
        "Inspect journal state and choose the matching classifier route.",
        "Retry until the third matching failure, then declare exhaustion.",
        "Write transaction intent before replacing each destination.",
        "Restore target preimages during offline recovery.",
    )
    declaration_pattern = re.compile(r"^<!-- workflow-artifact-adapter: (\{.*\}) -->$", re.MULTILINE)
    def parse_adapter(text: str, path: str) -> dict[str, str]:
        declarations = declaration_pattern.findall(text)
        require(len(declarations) == 1, f"adapter declaration cardinality: {path}")
        value = json.loads(declarations[0])
        require(list(value) == ["module", "entry", "action", "return"]
                and all(isinstance(item, str) and item for item in value.values()),
                f"adapter declaration fields: {path}")
        return value
    def adapter_policy_free(text: str) -> bool:
        outside = declaration_pattern.sub("", text)
        return not any(token in outside for token in forbidden + policy_restatements)
    for canonical_path, path in ADAPTERS.items():
        text = (root / path).read_text()
        expected = {
            "module": "devrites-lib/reference/standards/workflow-artifacts.md",
            "entry": adapter_map[canonical_path][0],
            "action": adapter_map[canonical_path][1],
            "return": adapter_map[canonical_path][2],
        }
        require(parse_adapter(text, path) == expected, f"adapter declaration equality: {path}")
        require(adapter_policy_free(text), f"adapter duplicates authority: {path}")
        declaration = declaration_pattern.search(text).group(0)
        for field in expected:
            mutant_value = dict(expected); mutant_value[field] += "-MUTANT"
            mutant = text.replace(declaration, f"<!-- workflow-artifact-adapter: {json.dumps(mutant_value, separators=(',', ':'))} -->", 1)
            require(parse_adapter(mutant, path) != expected, f"adapter field mutant: {path}:{field}")
        try:
            parse_adapter(text + "\n" + declaration, path)
        except AssertionError:
            pass
        else:
            fail(f"duplicate adapter declaration accepted: {path}")
        for paraphrase in policy_restatements:
            mutant = text + "\n" + paraphrase + "\n"
            require(parse_adapter(mutant, path) == expected and paraphrase in declaration_pattern.sub("", mutant),
                    f"adapter paraphrase fixture malformed: {path}")
            require(not adapter_policy_free(mutant),
                    f"adapter paraphrase survived: {path}:{paraphrase}")
    if (root / "pack/generated").is_dir():
        with tempfile.TemporaryDirectory() as tmp:
            env = os.environ.copy()
            env["DEVRITES_HOST_ARTIFACT_DIR"] = tmp
            generated = subprocess.run(
                with_delivery_execution_prefix(
                    ["bash", str(root / "scripts/build-host-artifacts.sh")]
                ),
                cwd=root, env=env, text=True, stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT, check=False,
            )
            require(generated.returncode == 0, f"generated adapter fixture: {generated.stdout[-2000:]}")
            for host in ("claude", "codex"):
                for canonical_path, path in ADAPTERS.items():
                    mirror = Path(tmp) / host / "skills" / canonical_path
                    require(parse_adapter(mirror.read_text(), mirror.as_posix()) == {
                        "module": "devrites-lib/reference/standards/workflow-artifacts.md",
                        "entry": adapter_map[canonical_path][0], "action": adapter_map[canonical_path][1],
                        "return": adapter_map[canonical_path][2],
                    }, f"generated adapter declaration: {host}:{canonical_path}")


ADMISSION_FIELDS = [
    "active_slug", "readiness_binding_command", "return_phase", "return_next_action",
    "target_order", "target_count_limit", "per_target_bytes_limit", "aggregate_bytes_limit",
    "transaction_file_limit", "diagnostic_bytes_limit", "journal_line_limit",
    "attempt_epoch_limit", "proof_command_timeout_seconds",
    "proof_aggregate_timeout_seconds", "proof_terminate_grace_seconds",
]
OPERATION_TABLE_SHA256 = "e71302a719ebd9c2404f696a8d50d6e6e7e25cda153691fcb6d600a1562e1b90"
ROUTE_MAP_SHA256 = "0a1634b772ae61e3d0ef0c74b9c0e0f1715af8401a00549179454470c4f96cfa"
SCENARIO_MAP_SHA256 = "e94b71586407920931fbd012dc8c5f4fdcac5d88f2069029065d9862d26f063e"
ADAPTER_MAP_SHA256 = "bb97e86c2322bc46ffdd36ce4155e0fbaf48d5dfd1650293b33db89c5c13ca4a"
CORPUS_SCENARIOS_SHA256 = "4401c2757d9921cb039fe5ab7a459efc22b39a0f248711004f26b518686a6301"
SCENARIO_FIELDS = {
    "id", "prompt", "expected_output", "expectations", "trust_level", "fixtures",
    "rationalization", "pressure", "source", "expected_route", "durable_consequence",
    "forbidden_actions", "expected_resistance", "capitulation_markers",
}
EXPECTED_OPERATION_FACTS = {
    "WA-OP-001-OWNER-ACQUIRE": ("no local owner descriptor", "exclusive lock held; generation/owned hash observed", "WAIT_ACTIVE_OWNER or BLOCKED_GATE"),
    "WA-OP-002-SOURCE-PROMOTE": ("lock held; green retained bytes; no active target write", "trusted canonical ready bundle", "PLAN_VET_REPAIR"),
    "WA-OP-002A-STALE-SOURCE-GC": ("lock held; binding rollover; internally valid old canonical; no journal/temp/target write/unknown entry", "stale bundle absent; parent synced", "OFFLINE_RECOVERY"),
    "WA-OP-003-JOURNAL-INIT": ("lock held; trusted source; absent owned section; target set unmodified", "generation advanced; complete owned section in PREPARING", "OFFLINE_RECOVERY"),
    "WA-OP-004-STAGE-WRITE": ("PREPARING; source bytes retained; exact target parent", "exact private stage bytes/mode synced", "OFFLINE_RECOVERY"),
    "WA-OP-005-BACKUP-WRITE": ("PREPARING; target still unmodified", "exact private backup/preimage-absence record synced", "OFFLINE_RECOVERY"),
    "WA-OP-006-INSTALL": ("PREPARED/prior INSTALLING; stage/backups exact", "atomically move current destination into private claim; validate captured object against exact preimage/frozen expected-post pair; install desired bytes/absence no-replace; exact readback; advance/INSTALLED", "OFFLINE_RECOVERY"),
    "WA-OP-007-PROVE": ("INSTALLED; all targets read back exact", "next proof command or durable PROVED", "OFFLINE_RECOVERY"),
    "WA-OP-008-ROLLBACK": ("replacement occurred; before PROVED", "atomically move current destination into private claim; validate captured object against exact preimage/frozen expected-post pair; install desired bytes/absence no-replace; exact readback; advance/ROLLED_BACK", "BLOCKED_GATE if restore cannot complete"),
    "WA-OP-009-FAILURE-CLEANUP": ("zero replacements or durable ROLLED_BACK", "stages/backups/evidence temp removed; canonical source retained; FAILED", "OFFLINE_RECOVERY"),
    "WA-OP-010-SUCCESS-CLEANUP": ("durable PROVED; targets exact frozen identity", "stages/backups/source/temp removed; outside evidence preserved; CLEANED", "RESUME_CLEANUP"),
    "WA-OP-011-RETRY-HANDOFF": ("locked FAILED; accepted correction; green re-preflight; same-fingerprint count <3 and next epoch within admitted cap", "exact new epoch in PREPARING; prior rows unchanged", "OFFLINE_RECOVERY"),
    "WA-OP-012-EXHAUSTION-GC": ("locked FAILED; same-fingerprint count=3 or admitted epoch cap reached", "retained source and exact transaction files removed; EXHAUSTED with durable truthful exhaustion_cause", "BLOCKED_GATE if safe cleanup cannot complete"),
    "WA-OP-013-EVIDENCE-UPDATE": ("lock held; observed generation/hash match", "atomic synced marker-owned section; generation+1; all outside bytes exact", "state operation's route; never infer success"),
    "WA-OP-014-PRODUCT-SEPARATION": ("frozen pre-transaction product candidate/readiness/built count", "exact equality recorded in owned section", "BLOCKED_GATE"),
    "WA-OP-015-VERIFY-EXISTING": ("CLEANED; exact evidence and targets; source absent or already GC'd", "same bytes/state/counters", "route by finite diagnostic table"),
}
EXPECTED_OBSERVER_FACTS = {
    operation: f"WA-OBS-{index:03d}" for index, operation in enumerate(EXPECTED_OPS, 1)
}
ACTUAL_ENGINE_OUTPUT = b""
EXPECTED_REASON_IDS = {
    "WA-R001-OWNER-BUSY", "WA-R002-ADMISSION-INCOMPLETE", "WA-R003-IDENTITY-MISSING",
    "WA-R004-IDENTITY-STALE", "WA-R005-SOURCE-UNTRUSTED", "WA-R006-SOURCE-STALE-PREINSTALL",
    "WA-R007-SOURCE-STALE-ACTIVE", "WA-R008-SOURCE-STALE-POSTPROOF", "WA-R009-STATE-AMBIGUOUS",
    "WA-R010-WRITE-FAILED", "WA-R011-REPLACE-FAILED", "WA-R012-READBACK-MISMATCH",
    "WA-R013-PROOF-FAILED", "WA-R014-PROOF-TIMEOUT", "WA-R015-ROLLBACK-FAILED",
    "WA-R016-FAILURE-CLEANUP-FAILED", "WA-R017-SUCCESS-CLEANUP-FAILED",
    "WA-R018-PRODUCT-IDENTITY-CHANGED", "WA-R019-LIMIT-EXCEEDED", "WA-R020-RETRY-EXHAUSTED",
    "WA-R021-ACCESS-DENIED", "WA-R022-STALE-SOURCE-GC-FAILED",
}
EXPECTED_BOUNDARY_IDS = {
    "WA-B001-OWNER", "WA-B002-ADMISSION", "WA-B003-SOURCE-PROMOTE",
    "WA-B004-SOURCE-OPEN", "WA-B005-JOURNAL", "WA-B006-STAGE-WRITE",
    "WA-B007-BACKUP-WRITE", "WA-B008-INSTALL", "WA-B009-READBACK",
    "WA-B010-PROVE", "WA-B011-ROLLBACK", "WA-B012-FAILURE-CLEANUP",
    "WA-B013-SUCCESS-CLEANUP", "WA-B014-PRODUCT-SEPARATION",
    "WA-B015-RETRY", "WA-B016-STALE-SOURCE-GC",
}
EXPECTED_DIAGNOSTICS = {
    "WA-R001-OWNER-BUSY": (("WA-B001-OWNER",), "exclusive owner held elsewhere", "WAIT_ACTIVE_OWNER"),
    "WA-R002-ADMISSION-INCOMPLETE": (("WA-B002-ADMISSION",), "required admission absent or malformed", "PLAN_VET_REPAIR"),
    "WA-R003-IDENTITY-MISSING": (("WA-B004-SOURCE-OPEN",), "current frozen identity unavailable", "PLAN_VET_REPAIR"),
    "WA-R004-IDENTITY-STALE": (("WA-B004-SOURCE-OPEN",), "authority differs from frozen identity", "PLAN_VET_REPAIR"),
    "WA-R005-SOURCE-UNTRUSTED": (("WA-B003-SOURCE-PROMOTE",), "source lacks exact authority", "PLAN_VET_REPAIR"),
    "WA-R006-SOURCE-STALE-PREINSTALL": (("WA-B004-SOURCE-OPEN",), "stale before first replacement", "PLAN_VET_REPAIR"),
    "WA-R007-SOURCE-STALE-ACTIVE": (("WA-B004-SOURCE-OPEN",), "stale after replacement before proof", "OFFLINE_RECOVERY"),
    "WA-R008-SOURCE-STALE-POSTPROOF": (("WA-B013-SUCCESS-CLEANUP",), "stale during proved cleanup", "RESUME_CLEANUP"),
    "WA-R009-STATE-AMBIGUOUS": (("WA-B005-JOURNAL",), "relation not admitted", "OFFLINE_RECOVERY"),
    "WA-R010-WRITE-FAILED": (("WA-B006-STAGE-WRITE", "WA-B007-BACKUP-WRITE"), "bounded write failed before install", "OFFLINE_RECOVERY"),
    "WA-R011-REPLACE-FAILED": (("WA-B008-INSTALL",), "replacement failed", "OFFLINE_RECOVERY"),
    "WA-R012-READBACK-MISMATCH": (("WA-B009-READBACK",), "installed identity differs", "OFFLINE_RECOVERY"),
    "WA-R013-PROOF-FAILED": (("WA-B010-PROVE",), "nonzero or wrong proof signal", "OFFLINE_RECOVERY"),
    "WA-R014-PROOF-TIMEOUT": (("WA-B010-PROVE",), "proof group exceeded bound", "OFFLINE_RECOVERY"),
    "WA-R015-ROLLBACK-FAILED": (("WA-B011-ROLLBACK",), "preimages not restored", "BLOCKED_GATE"),
    "WA-R016-FAILURE-CLEANUP-FAILED": (("WA-B012-FAILURE-CLEANUP",), "failure files remain", "OFFLINE_RECOVERY"),
    "WA-R017-SUCCESS-CLEANUP-FAILED": (("WA-B013-SUCCESS-CLEANUP",), "proved cleanup incomplete", "RESUME_CLEANUP"),
    "WA-R018-PRODUCT-IDENTITY-CHANGED": (("WA-B014-PRODUCT-SEPARATION",), "product identity drifted", "BLOCKED_GATE"),
    "WA-R019-LIMIT-EXCEEDED": (("WA-B002-ADMISSION",), "byte/file/time/journal bound exceeded", "PLAN_VET_REPAIR"),
    "WA-R020-RETRY-EXHAUSTED": (("WA-B015-RETRY",), "same-fingerprint count or total attempt epoch reached its independent cap", "BLOCKED_EXHAUSTED"),
    "WA-R021-ACCESS-DENIED": (("WA-B001-OWNER",), "host access or canonical flock unavailable", "BLOCKED_GATE"),
    "WA-R022-STALE-SOURCE-GC-FAILED": (("WA-B016-STALE-SOURCE-GC",), "validated stale cleanup incomplete", "OFFLINE_RECOVERY"),
}


def normalize_workflow_path(value: str) -> str:
    require(value != "" and len(value.encode()) <= 4096, "workflow path length")
    require(not value.startswith("/") and "\\" not in value, "workflow path absolute/backslash")
    require(not any(ord(char) < 0x20 or char in "*?[]{}" for char in value), "workflow path control/glob")
    parts = value.split("/")
    require(all(part not in {"", ".", ".."} for part in parts), "workflow path components")
    require(parts[0] not in {".git", "node_modules", "vendor", "pack", "engine", "src"}, "workflow/product separation")
    require(value == Path(value).as_posix(), "workflow path normalization")
    return value


def admission_fixture() -> tuple[str, dict[str, bytes]]:
    fields = {
        "active_slug": "demo", "readiness_binding_command": "devrites-engine check readiness --emit-binding demo",
        "return_phase": "prove", "return_next_action": "/rite-prove demo",
        "target_order": "utf8-bytewise-path-ascending", "target_count_limit": "2",
        "per_target_bytes_limit": "32", "aggregate_bytes_limit": "64",
        "transaction_file_limit": "12", "diagnostic_bytes_limit": "256", "journal_line_limit": "35",
        "attempt_epoch_limit": "3", "proof_command_timeout_seconds": "30",
        "proof_aggregate_timeout_seconds": "60", "proof_terminate_grace_seconds": "2",
    }
    lines = ["## Workflow Artifact admission", "DevRites contract: devrites.workflow-artifact-admission.v1", "", "| Field | Value |", "| --- | --- |"]
    lines.extend(f"| {name} | `{fields[name]}` |" for name in ADMISSION_FIELDS)
    lines.extend([
        "", "| Index | Path | Mode | Behavior ref | Interface ref | Positive fixture | Failure fixtures | Proof command | Proof cwd | Proof signal | Rollback | Evidence fields |",
        "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |",
        "| `00000000` | `scripts/a.py` | `0600` | `WA-BEH-A` | `WA-IF-A` | `WA-FIX-PA` | `WA-FIX-FA` | `python3 scripts/a.py` | `active-workspace` | `A PASS` | `restore-preimage-or-absence` | `mode,sha256,proof_signal` |",
        "| `00000001` | `scripts/b.py` | `0755` | `WA-BEH-B` | `WA-IF-B` | `WA-FIX-PB` | `WA-FIX-FB` | `python3 scripts/b.py` | `repository-root` | `B PASS` | `restore-preimage-or-absence` | `mode,sha256,proof_signal` |", "",
    ])
    blocks = [
        ("WA-BEH-A", "behavior", (("success", "A command exits zero"), ("observable_effect", "A PASS is observed"))),
        ("WA-IF-A", "interface", (("inputs", "retained A bytes"), ("invariants", "target identity remains bound"), ("ordering", "install before proof"), ("errors", "nonzero routes recovery"), ("configuration", "active workspace cwd"), ("performance", "within admitted timeout"))),
        ("WA-FIX-PA", "positive-fixture", (("setup", "retained A source"), ("action", "run admitted A command"), ("expected", "A PASS"))),
        ("WA-FIX-FA", "failure-fixture", (("setup", "retained A source"), ("fault", "nonzero exit"), ("expected", "rollback then offline recovery"))),
        ("WA-BEH-B", "behavior", (("success", "B command exits zero"), ("observable_effect", "B PASS is observed"))),
        ("WA-IF-B", "interface", (("inputs", "retained B bytes"), ("invariants", "target identity remains bound"), ("ordering", "install before proof"), ("errors", "nonzero routes recovery"), ("configuration", "repository root cwd"), ("performance", "within admitted timeout"))),
        ("WA-FIX-PB", "positive-fixture", (("setup", "retained B source"), ("action", "run admitted B command"), ("expected", "B PASS"))),
        ("WA-FIX-FB", "failure-fixture", (("setup", "retained B source"), ("fault", "nonzero exit"), ("expected", "rollback then offline recovery"))),
    ]
    for reference, kind, body in blocks:
        lines.extend([f"## {reference}", f"DevRites workflow reference: {kind}", "| Field | Value |", "| --- | --- |"])
        lines.extend(f"| {name} | `{value}` |" for name, value in body)
        lines.append("")
    return "\n".join(lines), {"scripts/a.py": b"print('a')\n", "scripts/b.py": b"print('b')\n"}


def parse_admission(text: str, contents: dict[str, bytes]) -> dict:
    require(text.count("## Workflow Artifact admission") == 1, "admission heading cardinality")
    require(text.count("DevRites contract: devrites.workflow-artifact-admission.v1") == 1, "admission contract cardinality")
    lines = text.splitlines()
    heading = lines.index("## Workflow Artifact admission")
    require(lines[heading + 1] == "DevRites contract: devrites.workflow-artifact-admission.v1", "admission contract placement")
    field_start = lines.index("| Field | Value |", heading)
    require(lines[field_start + 1] == "| --- | --- |", "admission field separator")
    fields = []
    cursor = field_start + 2
    while cursor < len(lines) and lines[cursor].startswith("|"):
        cells = [cell.strip() for cell in split_markdown_cells(lines[cursor].strip("|"))]
        require(len(cells) == 2 and re.fullmatch(r"`[^`]+`", cells[1]) is not None, "admission field shape")
        fields.append((cells[0], cells[1][1:-1]))
        cursor += 1
    require([name for name, _ in fields] == ADMISSION_FIELDS, "admission field order")
    values = dict(fields)
    slug = values["active_slug"]
    require(re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", slug) is not None, "active slug")
    require(values["readiness_binding_command"] == f"devrites-engine check readiness --emit-binding {slug}", "readiness command")
    require(values["target_order"] == "utf8-bytewise-path-ascending", "target order")
    numeric_names = ADMISSION_FIELDS[5:]
    numbers = {}
    for name in numeric_names:
        raw = values[name]
        require(re.fullmatch(r"0|[1-9][0-9]*", raw) is not None, f"base-10 field: {name}")
        number = int(raw)
        require(number <= 2**63 - 1, f"checked field: {name}")
        numbers[name] = number
    target_header = "| Index | Path | Mode | Behavior ref | Interface ref | Positive fixture | Failure fixtures | Proof command | Proof cwd | Proof signal | Rollback | Evidence fields |"
    target_start = lines.index(target_header, cursor)
    require(lines[target_start + 1].count("---") == 12, "target separator")
    rows = []
    cursor = target_start + 2
    while cursor < len(lines) and lines[cursor].startswith("|"):
        raw_cells = [cell.strip() for cell in split_markdown_cells(lines[cursor].strip("|"))]
        require(len(raw_cells) == 12
                and all(re.fullmatch(r"`[^`\n]+`", cell) is not None for cell in raw_cells),
                "target row completeness")
        cells = [cell[1:-1] for cell in raw_cells]
        require(cells[0] == f"{len(rows):08x}", "target index")
        path = normalize_workflow_path(cells[1])
        require(re.fullmatch(r"0[0-7]{3}", cells[2]) is not None, "target mode")
        require(path in contents, "retained source content")
        require(not any(token in cells[7] for token in (";", "&&", "||", "\n")), "single proof command")
        require(len(split_markdown_cells(cells[7])) == 1, "proof command list separator")
        require(cells[8] in {"repository-root", "active-workspace"}, "logical proof cwd")
        require(cells[9].isascii() and 0 < len(cells[9].encode()) <= 128
                and all(0x20 <= ord(char) <= 0x7e for char in cells[9]), "fixed proof signal")
        require(cells[10] == "restore-preimage-or-absence", "rollback enum")
        evidence = cells[11].split(",")
        require(evidence and len(evidence) == len(set(evidence))
                and all(re.fullmatch(r"[a-z][a-z0-9_]*", field) is not None for field in evidence),
                "evidence field grammar")
        rows.append(cells)
        cursor += 1
    paths = [row[1] for row in rows]
    require(paths == sorted(paths, key=lambda path: path.encode()) and len(set(paths)) == len(paths), "target path order")
    require(1 <= len(rows) <= numbers["target_count_limit"], "target count")
    sizes = [len(contents[path]) for path in paths]
    require(all(size <= numbers["per_target_bytes_limit"] for size in sizes), "per-target bytes")
    require(sum(sizes) <= numbers["aggregate_bytes_limit"], "aggregate bytes")
    validate_limits(numbers["target_count_limit"], numbers["per_target_bytes_limit"], numbers["aggregate_bytes_limit"],
                    numbers["transaction_file_limit"], numbers["diagnostic_bytes_limit"],
                    numbers["journal_line_limit"], numbers["attempt_epoch_limit"],
                    numbers["proof_command_timeout_seconds"], numbers["proof_aggregate_timeout_seconds"],
                    numbers["proof_terminate_grace_seconds"])
    require(values["return_phase"] in {"define", "spec", "clarify", "plan", "vet", "build", "prove", "polish", "review", "seal", "ship"}, "return phase")
    require(re.fullmatch(rf"/rite-[a-z0-9-]+ {re.escape(slug)}", values["return_next_action"]) is not None, "return next action")
    reference_order = []
    schemas = {
        "behavior": (re.compile(r"WA-BEH-[A-Z0-9][A-Z0-9-]*"), ("success", "observable_effect")),
        "interface": (re.compile(r"WA-IF-[A-Z0-9][A-Z0-9-]*"), ("inputs", "invariants", "ordering", "errors", "configuration", "performance")),
        "positive-fixture": (re.compile(r"WA-FIX-P[A-Z0-9][A-Z0-9-]*"), ("setup", "action", "expected")),
        "failure-fixture": (re.compile(r"WA-FIX-F[A-Z0-9][A-Z0-9-]*"), ("setup", "fault", "expected")),
    }
    for row in rows:
        references = [
            (row[3], "behavior"), (row[4], "interface"), (row[5], "positive-fixture"),
            *[(item.strip(), "failure-fixture") for item in row[6].split(",")],
        ]
        for reference, kind in references:
            pattern, field_order = schemas[kind]
            require(pattern.fullmatch(reference) is not None and text.count(f"## {reference}") == 1,
                    f"reference ID/cardinality: {reference}")
            heading_index = lines.index(f"## {reference}")
            require(lines[heading_index + 1:heading_index + 4] == [
                f"DevRites workflow reference: {kind}", "| Field | Value |", "| --- | --- |",
            ], f"reference header: {reference}")
            body = []
            body_cursor = heading_index + 4
            while body_cursor < len(lines) and lines[body_cursor].startswith("|"):
                cells = [cell.strip() for cell in split_markdown_cells(lines[body_cursor].strip("|"))]
                require(len(cells) == 2 and re.fullmatch(r"`[^`\n]+`", cells[1]) is not None,
                        f"reference cell: {reference}")
                value = cells[1][1:-1].replace("\\|", "|")
                require(not re.search(r"(?i)^(?:tbd|todo|placeholder|<[^>]+>)$", value),
                        f"reference placeholder: {reference}")
                body.append((cells[0], value))
                body_cursor += 1
            require(tuple(name for name, _ in body) == field_order, f"reference field order: {reference}")
            reference_order.append(reference)
    headings = [line[3:] for line in lines if line.startswith("## WA-")]
    require(headings == reference_order and len(headings) == len(set(headings)), "reference body order/cardinality")
    return {"fields": values, "rows": rows, "references": tuple(reference_order)}


def check_admission_parser() -> None:
    text, contents = admission_fixture()
    parsed = parse_admission(text, contents)
    require(len(parsed["rows"]) == 2, "admission positive fixture")
    target_line = next(line for line in text.splitlines() if line.startswith("| `00000000`"))
    target_cells = split_markdown_cells(target_line.strip("|"))
    require(len(target_cells) == 12, "admission target wrapper fixture")
    for cell_index in range(12):
        for wrapper in ("missing", "extra"):
            mutant_cells = target_cells.copy()
            wrapped = mutant_cells[cell_index].strip()
            replacement = wrapped[1:-1] if wrapper == "missing" else f"`{wrapped}`"
            mutant_cells[cell_index] = f" {replacement} "
            mutant_line = "|" + "|".join(mutant_cells) + "|"
            try:
                parse_admission(text.replace(target_line, mutant_line, 1), contents)
            except (AssertionError, ValueError):
                pass
            else:
                fail(f"admission target {wrapper} wrapper accepted: {cell_index}")
    sparse_high_limit = (
        text.replace("| target_count_limit | `2` |", "| target_count_limit | `10` |", 1)
        .replace("| transaction_file_limit | `12` |", "| transaction_file_limit | `36` |", 1)
        .replace("| journal_line_limit | `35` |", "| journal_line_limit | `43` |", 1)
    )
    require(len(parse_admission(sparse_high_limit, contents)["rows"]) == 2,
            "sparse admission retains declared high-limit capacity")
    mutations = [
        text + "\n## Workflow Artifact admission\n", text.replace("| active_slug |", "| absent_slug |", 1),
        text.replace("| active_slug | `demo` |\n| readiness_binding_command", "| readiness_binding_command | `devrites-engine check readiness --emit-binding demo` |\n| active_slug", 1),
        text.replace("`scripts/a.py`", "`/scripts/a.py`", 1), text.replace("`scripts/a.py`", "`scripts/../a.py`", 1),
        text.replace("`scripts/b.py`", "`scripts/a.py`", 1), text.replace("`0600`", "`0688`", 1),
        text.replace("`00000001`", "`00000002`", 1), text.replace("## WA-FIX-FB", "## WA-FIX-FX", 1),
        text.replace("`python3 scripts/a.py`", "`python3 scripts/a.py && true`", 1),
        text.replace("| transaction_file_limit | `12` |", "| transaction_file_limit | `11` |", 1),
        text.replace("| journal_line_limit | `35` |", "| journal_line_limit | `34` |", 1),
        text.replace("| diagnostic_bytes_limit | `256` |", "| diagnostic_bytes_limit | `255` |", 1),
        text.replace("| attempt_epoch_limit | `3` |", "| attempt_epoch_limit | `2` |", 1),
        text.replace("| proof_command_timeout_seconds | `30` |", "| proof_command_timeout_seconds | `2` |", 1).replace("| proof_terminate_grace_seconds | `2` |", "| proof_terminate_grace_seconds | `2` |", 1),
        text.replace("| target_count_limit | `2` |", "| target_count_limit | `9223372036854775808` |", 1),
        text.replace("| target_count_limit | `2` |", "| target_count_limit | `10` |", 1),
        text.replace("| target_count_limit | `2` |", "| target_count_limit | `10` |", 1)
            .replace("| transaction_file_limit | `12` |", "| transaction_file_limit | `36` |", 1),
        text.replace("`python3 scripts/a.py`", "`python3 scripts/a.py | true`", 1),
        text.replace("`A PASS`", "`A\nPASS`", 1),
        text.replace("`restore-preimage-or-absence`", "`restore`", 1),
        text.replace("`mode,sha256,proof_signal`", "`mode,mode`", 1),
        text.replace("DevRites workflow reference: behavior", "DevRites workflow reference: interface", 1),
        text.replace("## WA-BEH-A", "## WA-BEH-X", 1),
        text.replace("WA-FIX-PA", "WA-FIX-A", 2),
        text.replace("DevRites workflow reference: behavior\n", "", 1),
        text.replace("| success | `A command exits zero` |\n| observable_effect |", "| observable_effect | `A PASS is observed` |\n| success |", 1),
        text.replace("`active-workspace`", "`.`", 1),
        text.replace("/rite-prove demo", "/rite-prove other", 1),
    ]
    for index, mutant in enumerate(mutations):
        try:
            parse_admission(mutant, contents)
        except (AssertionError, ValueError):
            pass
        else:
            fail(f"admission mutant accepted: {index}")
    for changed in ({**contents, "scripts/a.py": b"x" * 33}, {**contents, "scripts/a.py": b"x" * 32, "scripts/b.py": b"y" * 33}):
        try:
            parse_admission(text, changed)
        except AssertionError:
            pass
        else:
            fail("admission content limit mutant accepted")


def exact_map_digest(value) -> str:
    return sha(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode())


def reject_table_cell_mutants(rows: list[list[str]], expected_digest: str, label: str) -> int:
    rejected = 0
    for row_index, row in enumerate(rows):
        for cell_index in range(len(row)):
            mutant = [candidate.copy() for candidate in rows]
            mutant[row_index][cell_index] += "-MUTANT"
            require(exact_map_digest(mutant) != expected_digest,
                    f"{label} cell mutant survived: {row_index}:{cell_index}")
            rejected += 1
    return rejected


ATOMIC_OWNERSHIP_CLAUSES = (
    (2, "names exact intent-derived private claim/install artifacts before mutation"),
    (4, "atomically move current destination into private claim"),
    (4, "validate captured object against exact preimage/frozen expected-post pair"),
    (4, "install desired bytes/absence no-replace"),
    (5, "third state/concurrent occupant preserves claim and occupant when needed and blocks"),
)


def validate_atomic_ownership_contract(text: str) -> int:
    rows = {row[0]: row for row in markdown_rows(text, "WA-OP-")}
    operations = ("WA-OP-006-INSTALL", "WA-OP-008-ROLLBACK")
    for operation in operations:
        require(operation in rows and len(rows[operation]) == 8, f"atomic ownership row: {operation}")
        for column, clause in ATOMIC_OWNERSHIP_CLAUSES:
            require(clause in rows[operation][column], f"atomic ownership clause: {operation}:{column}:{clause}")
    return len(operations)


def reject_atomic_ownership_mutants(text: str) -> int:
    rejected = 0
    for operation in ("WA-OP-006-INSTALL", "WA-OP-008-ROLLBACK"):
        row = next(line for line in text.splitlines() if line.startswith(f"| `{operation}`|"))
        for _column, clause in ATOMIC_OWNERSHIP_CLAUSES:
            require(row.count(clause) == 1, f"atomic ownership fixture clause: {operation}:{clause}")
            for replacement in ("", clause.replace(" ", "-MUTANT ", 1)):
                mutant = text.replace(row, row.replace(clause, replacement, 1), 1)
                try:
                    validate_atomic_ownership_contract(mutant)
                except AssertionError:
                    rejected += 1
                else:
                    fail(f"atomic ownership mutant survived: {operation}:{clause}:{replacement!r}")
    return rejected


def operation_table_digest(rows: list[list[str]]) -> str:
    return sha("\n".join("|".join(row) for row in rows).encode())


def operation_trace_from_table(rows: list[list[str]]) -> tuple[tuple[str, str, str, str, str], ...]:
    require([row[0] for row in rows] == EXPECTED_OPS and all(len(row) == 8 for row in rows), "canonical operation table shape")
    return tuple((row[0], row[3], row[4], row[6], EXPECTED_OBSERVER_FACTS[row[0]]) for row in rows)


def expected_operation_trace() -> tuple[tuple[str, str, str, str, str], ...]:
    return tuple((operation, *EXPECTED_OPERATION_FACTS[operation], EXPECTED_OBSERVER_FACTS[operation]) for operation in EXPECTED_OPS)


def observe_regular_file(path: Path) -> tuple[bool, int, int, str]:
    program = (
        "import hashlib,json,os,stat,sys;"
        "f=os.open(sys.argv[1],os.O_RDONLY|os.O_NOFOLLOW|os.O_CLOEXEC);"
        "s=os.fstat(f);b=b'';"
        "\nwhile True:\n c=os.read(f,65536)\n if not c: break\n b+=c\n"
        "os.close(f);print(json.dumps([stat.S_ISREG(s.st_mode),stat.S_IMODE(s.st_mode),s.st_nlink,hashlib.sha256(b).hexdigest()]))"
    )
    observed = subprocess.run([sys.executable, "-c", program, str(path)], stdout=subprocess.PIPE,
                              stderr=subprocess.PIPE, text=True, check=False, timeout=5)
    require(observed.returncode == 0 and observed.stderr == "", "independent observer process")
    values = json.loads(observed.stdout)
    return bool(values[0]), int(values[1]), int(values[2]), str(values[3])


def read_barrier(fd: int, expected: bytes, timeout: float = 3.0) -> None:
    selector = selectors.DefaultSelector()
    selector.register(fd, selectors.EVENT_READ)
    try:
        require(bool(selector.select(timeout)), f"barrier timeout: {expected.decode()}")
        require(os.read(fd, len(expected)) == expected, f"barrier value: {expected.decode()}")
    finally:
        selector.close()


def prepare_operation_fixture(root: Path, operation: str) -> None:
    root.mkdir(parents=True, exist_ok=True)
    marker = root / ".fixture.ready"
    if marker.exists():
        return
    dimensions = {
        "candidate": sha(b"product-candidate\n"),
        "readiness": sha(b"product-readiness\n"),
        "built_count": 7,
    }
    initial = {
        "source": b"frozen-workflow\n", "target": b"target-preimage\n",
        "product.candidate": (dimensions["candidate"] + "\n").encode(),
        "product.readiness": (dimensions["readiness"] + "\n").encode(),
        "product.built-count": b"7\n",
        "product.frozen": (json.dumps(dimensions, sort_keys=True, separators=(",", ":")) + "\n").encode(),
        "engine.output": ACTUAL_ENGINE_OUTPUT, "stale.source": b"stale-workflow\n",
        "owner.lock": b"", "stage": b"partial-stage\n", "backup": b"target-preimage\n",
    }
    if operation in {"WA-OP-006-INSTALL", "WA-OP-007-PROVE", "WA-OP-008-ROLLBACK"}:
        initial["target"] = b"frozen-workflow\n"
    if operation in {"WA-OP-006-INSTALL", "WA-OP-007-PROVE"}:
        initial["backup"] = b"observed-active-preimage\n"
        initial["lifecycle.state"] = b"PROVING\n"
    if operation == "WA-OP-011-RETRY-HANDOFF":
        initial.update({
            "lifecycle.state": b"FAILED\n", "attempt.history": b"1:" + b"a" * 64 + b":no-progress\n",
            "retry.control": b'{"accepted":true,"epoch":1,"epoch_cap":3,"green":true,"same_count":1}\n',
        })
    if operation == "WA-OP-012-EXHAUSTION-GC":
        initial.update({
            "lifecycle.state": b"FAILED\n", "attempt.history": b"1:" + b"a" * 64 + b":no-progress\n2:" + b"a" * 64 + b":no-progress\n3:" + b"a" * 64 + b":no-progress\n",
            "retry.control": b'{"accepted":true,"epoch":3,"epoch_cap":4,"green":true,"same_count":3}\n',
        })
    if operation == "WA-OP-015-VERIFY-EXISTING":
        initial.update({"lifecycle.state": b"CLEANED\n", "evidence": b"observed evidence\n", "product.observer-pass": b"equal\n"})
    for name, payload in initial.items():
        path = root / name
        path.write_bytes(payload); path.chmod(0o600)
    marker.write_bytes(b"ready\n"); marker.chmod(0o600)


def operation_consumer(root: Path, row: list[str]) -> tuple[subprocess.Popen, int, int]:
    operation = row[0]
    prepare_operation_fixture(root, operation)
    ready_read, ready_write = os.pipe()
    gate_read, gate_write = os.pipe()
    program = r"""
import fcntl,hashlib,json,os,stat,subprocess,sys
root,operation,ready,gate=sys.argv[1],sys.argv[2],int(sys.argv[3]),int(sys.argv[4])
os.umask(0o077)
allowed={
'WA-OP-001-OWNER-ACQUIRE','WA-OP-002-SOURCE-PROMOTE','WA-OP-002A-STALE-SOURCE-GC','WA-OP-003-JOURNAL-INIT',
'WA-OP-004-STAGE-WRITE','WA-OP-005-BACKUP-WRITE','WA-OP-006-INSTALL','WA-OP-007-PROVE',
'WA-OP-008-ROLLBACK','WA-OP-009-FAILURE-CLEANUP','WA-OP-010-SUCCESS-CLEANUP','WA-OP-011-RETRY-HANDOFF',
'WA-OP-012-EXHAUSTION-GC','WA-OP-013-EVIDENCE-UPDATE','WA-OP-014-PRODUCT-SEPARATION','WA-OP-015-VERIFY-EXISTING'}
if operation not in allowed: raise SystemExit(64)
fd=os.open(root,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW|os.O_CLOEXEC)
def read(name):
 f=os.open(name,os.O_RDONLY|os.O_NOFOLLOW|os.O_CLOEXEC,dir_fd=fd); parts=[]
 while True:
  part=os.read(f,65536)
  if not part: break
  parts.append(part)
 os.close(f); return b''.join(parts)
def atomic(name,payload,mode=0o600):
 temporary='.'+name+'.tmp'
 try: os.unlink(temporary,dir_fd=fd)
 except FileNotFoundError: pass
 f=os.open(temporary,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,mode,dir_fd=fd)
 view=memoryview(payload)
 while view:
  written=os.write(f,view)
  if written<1: raise OSError('write made no progress')
  view=view[written:]
 os.fchmod(f,mode); os.fsync(f); os.close(f); os.rename(temporary,name,src_dir_fd=fd,dst_dir_fd=fd); os.fsync(fd)
def remove(name):
 try: os.unlink(name,dir_fd=fd); os.fsync(fd)
 except FileNotFoundError: pass
def exists(name):
 try: os.stat(name,dir_fd=fd,follow_symlinks=False); return True
 except FileNotFoundError: return False
def state(value): atomic('lifecycle.state',(value+'\n').encode())
def route(value): atomic('next.route',(value+'\n').encode())
retained_source=read('source') if operation=='WA-OP-004-STAGE-WRITE' and exists('source') else None
os.write(ready,b'R'); os.read(gate,1)
record={'operation_id':operation,'state':'INTENT'}
control=None
if operation in {'WA-OP-011-RETRY-HANDOFF','WA-OP-012-EXHAUSTION-GC'}:
 control=json.loads(read('retry.control'))
 history=read('attempt.history')
 record['prior_history_sha256']=hashlib.sha256(history).hexdigest()
 if operation=='WA-OP-011-RETRY-HANDOFF':
  if not control['accepted'] or not control['green'] or control['same_count']>=3 or control['epoch']>=control['epoch_cap']: raise RuntimeError('retry predicates')
  record['attempt_epoch']=control['epoch']+1
 else:
  if control['same_count']>=3: cause='same-fingerprint-count'
  elif control['epoch']>=control['epoch_cap']: cause='total-epoch-limit'
  else: raise RuntimeError('exhaustion predicate')
  record['attempt_epoch']=control['epoch']
try:
 prior=json.loads(read('journal.json'))
 if prior.get('operation_id')!=operation:
  if prior.get('state')!='COMPLETE': raise RuntimeError('foreign operation intent')
  atomic('journal.json',(json.dumps(record,sort_keys=True,separators=(',',':'))+'\n').encode())
 elif prior.get('state')=='INTENT': record=prior
except FileNotFoundError: atomic('journal.json',(json.dumps(record,sort_keys=True,separators=(',',':'))+'\n').encode())
os.write(ready,b'I'); os.read(gate,1)
if operation=='WA-OP-001-OWNER-ACQUIRE':
 try: os.mkdir('namespace',0o700,dir_fd=fd); os.fsync(fd)
 except FileExistsError: pass
 ns=os.open('namespace',os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW|os.O_CLOEXEC,dir_fd=fd)
 try:
  try: lock=os.open('owner.lock',os.O_RDWR|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,0o600,dir_fd=ns); os.fsync(lock); os.fsync(ns)
  except FileExistsError: lock=os.open('owner.lock',os.O_RDWR|os.O_NOFOLLOW|os.O_CLOEXEC,dir_fd=ns)
  info=os.fstat(lock)
  if not stat.S_ISREG(info.st_mode) or stat.S_IMODE(info.st_mode)!=0o600 or info.st_nlink!=1: raise RuntimeError('owner metadata')
  fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB); atomic('owner.observed',b'exclusive\n'); os.close(lock)
 finally: os.close(ns)
elif operation=='WA-OP-002-SOURCE-PROMOTE':
 try: stale=read('stale.authority')
 except FileNotFoundError: stale=None
 if stale is None: atomic('source.ready',b'devrites.workflow-source-ready.v1\ncount=1\nidentity='+b'a'*64+b'\n')
 else:
  atomic('diagnostic',b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR\n'); state('PLAN_VET_REPAIR')
elif operation=='WA-OP-002A-STALE-SOURCE-GC':
 remove('stale.source'); os.fsync(fd)
 lock=os.open('owner.lock',os.O_RDWR|os.O_NOFOLLOW|os.O_CLOEXEC,dir_fd=fd); os.ftruncate(lock,0); os.fsync(lock); os.lseek(lock,0,0)
 if os.read(lock,1)!=b'': raise RuntimeError('stale intent clear')
 os.close(lock); state('STALE_SOURCE_REMOVED')
elif operation=='WA-OP-003-JOURNAL-INIT': state('PREPARING')
elif operation=='WA-OP-004-STAGE-WRITE':
 if retained_source is None: remove('stage'); remove('backup'); state('SOURCE_LOSS_PREINSTALL'); route('PLAN_VET_REPAIR')
 else: atomic('stage',retained_source)
elif operation=='WA-OP-005-BACKUP-WRITE': atomic('backup',read('target'))
elif operation in {'WA-OP-006-INSTALL','WA-OP-007-PROVE'}:
 if not exists('source'):
  try: backup=read('backup')
  except FileNotFoundError: backup=None
  if backup!=b'observed-active-preimage\n':
   atomic('diagnostic',b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R015-ROLLBACK-FAILED boundary_id=WA-B011-ROLLBACK next_route=BLOCKED_GATE\n'); state('BLOCKED_GATE'); route('BLOCKED_GATE')
  else:
   atomic('target',backup); remove('stage'); remove('backup'); atomic('diagnostic',b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R007-SOURCE-STALE-ACTIVE boundary_id=WA-B004-SOURCE-OPEN next_route=OFFLINE_RECOVERY\n'); state('FAILED'); route('OFFLINE_RECOVERY')
 elif operation=='WA-OP-006-INSTALL': atomic('target',read('source'),0o755); state('INSTALLED')
 else:
  proof=subprocess.run([sys.executable,'-c',"print('WA-PROOF-001 PASS')"],stdout=subprocess.PIPE,check=True).stdout
  atomic('proof.output',proof); state('PROVED')
elif operation=='WA-OP-008-ROLLBACK':
 try: backup=read('backup')
 except FileNotFoundError: backup=None
 if backup!=b'target-preimage\n':
  atomic('diagnostic',b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R015-ROLLBACK-FAILED boundary_id=WA-B011-ROLLBACK next_route=BLOCKED_GATE\n'); state('BLOCKED_GATE'); route('BLOCKED_GATE')
 else: atomic('target',backup); state('ROLLED_BACK')
elif operation=='WA-OP-009-FAILURE-CLEANUP': remove('stage'); remove('backup'); state('FAILED')
elif operation=='WA-OP-010-SUCCESS-CLEANUP':
 if exists('lifecycle.state') and read('lifecycle.state')==b'BLOCKED_GATE\n': raise RuntimeError('product gate')
 source_lost=not exists('source'); remove('stage'); remove('backup'); remove('source'); remove('source.ready'); remove('stale.source'); state('CLEANED'); atomic('cursor',b'prove:/rite-prove demo\n')
 if source_lost: route('PLAN_VET_REPAIR')
elif operation=='WA-OP-011-RETRY-HANDOFF':
 pending=str(record['attempt_epoch']).encode()+b':'+control.get('fingerprint','a'*64).encode()+b':pending\n'
 if not history.endswith(pending): atomic('attempt.history',history+pending)
 state('PREPARING')
elif operation=='WA-OP-012-EXHAUSTION-GC':
 remove('source'); remove('source.ready'); remove('stage'); remove('backup'); record['state']='EXHAUSTED'; record['exhaustion_cause']=cause; atomic('journal.json',(json.dumps(record,sort_keys=True,separators=(',',':'))+'\n').encode()); atomic('exhaustion.cause',(cause+'\n').encode()); atomic('evidence',('state=EXHAUSTED\nexhaustion_cause='+cause+'\n').encode()); state('EXHAUSTED'); route('BLOCKED_EXHAUSTED')
elif operation=='WA-OP-013-EVIDENCE-UPDATE': atomic('evidence',b'prefix\n<!-- devrites-workflow-artifact-journal:start -->\nstate=CLEANED\n<!-- devrites-workflow-artifact-journal:end -->\nsuffix\n')
elif operation=='WA-OP-014-PRODUCT-SEPARATION':
 frozen=json.loads(read('product.frozen')); current={'candidate':read('product.candidate').decode().strip(),'readiness':read('product.readiness').decode().strip(),'built_count':int(read('product.built-count'))}
 atomic('product.comparison',(json.dumps({'current':current,'frozen':frozen},sort_keys=True,separators=(',',':'))+'\n').encode())
 if current!=frozen:
  atomic('diagnostic',b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R018-PRODUCT-IDENTITY-CHANGED boundary_id=WA-B014-PRODUCT-SEPARATION next_route=BLOCKED_GATE\n'); state('BLOCKED_GATE'); route('BLOCKED_GATE')
elif operation=='WA-OP-015-VERIFY-EXISTING':
 if read('lifecycle.state')!=b'CLEANED\n' or not read('evidence') or read('product.observer-pass')!=b'equal\n': raise RuntimeError('existing identity')
 atomic('verified',hashlib.sha256(read('target')).hexdigest().encode()+b'\n')
os.write(ready,b'E'); os.read(gate,1)
if operation!='WA-OP-012-EXHAUSTION-GC': record['state']='COMPLETE'; atomic('journal.json',(json.dumps(record,sort_keys=True,separators=(',',':'))+'\n').encode())
os.write(ready,b'C'); os.close(fd)
"""
    proc = subprocess.Popen(
        [sys.executable, "-c", program, str(root), operation, str(ready_write), str(gate_read)],
        pass_fds=(ready_write, gate_read), start_new_session=True,
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    os.close(ready_write); os.close(gate_read)
    return proc, ready_read, gate_write


def finish_operation_consumer(proc: subprocess.Popen, ready_fd: int, gate_fd: int) -> None:
    read_barrier(ready_fd, b"R")
    os.write(gate_fd, b"x")
    read_barrier(ready_fd, b"I")
    os.write(gate_fd, b"x")
    read_barrier(ready_fd, b"E")
    os.write(gate_fd, b"x")
    read_barrier(ready_fd, b"C")
    os.close(gate_fd)
    os.close(ready_fd)
    require(proc.wait(timeout=3) == 0 and not process_group_alive(proc.pid), "operation consumer completion")


def observe_operation(root: Path) -> dict:
    fixed = {
        operation: [operation, *EXPECTED_OPERATION_FACTS[operation], EXPECTED_OBSERVER_FACTS[operation]]
        for operation in EXPECTED_OPS
    }
    program = r"""
import hashlib,json,os,stat,sys
root=sys.argv[1]; fixed=json.loads(sys.argv[2])
def inspect(name):
 try: f=os.open(name,os.O_RDONLY|os.O_NOFOLLOW|os.O_CLOEXEC,dir_fd=directory)
 except FileNotFoundError: return None
 info=os.fstat(f); chunks=[]
 while True:
  chunk=os.read(f,65536)
  if not chunk: break
  chunks.append(chunk)
 os.close(f); return {'mode':stat.S_IMODE(info.st_mode),'links':info.st_nlink,'raw':b''.join(chunks)}
def raw(name):
 value=inspect(name); return None if value is None else value['raw']
def regular(name,mode=0o600):
 value=inspect(name); return value is not None and value['mode']==mode and value['links']==1
def absent(name): return inspect(name) is None
directory=os.open(root,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW|os.O_CLOEXEC)
journal=inspect('journal.json'); out={'journal':None,'result':None,'valid':False,'facts':{}}
if journal is not None:
 try: value=json.loads(journal['raw']); operation=value['operation_id']
 except Exception: value={}; operation=''
 out['journal']={'mode':journal['mode'],'links':journal['links'],'value':value}
 state=raw('lifecycle.state')
 product_frozen=json.loads(raw('product.frozen')) if raw('product.frozen') is not None else None
 product_current=None
 if all(raw(name) is not None for name in ('product.candidate','product.readiness','product.built-count')):
  product_current={'candidate':raw('product.candidate').decode().strip(),'readiness':raw('product.readiness').decode().strip(),'built_count':int(raw('product.built-count'))}
 product_equal=None if product_current is None or product_frozen is None else product_current==product_frozen
 checks={
 'WA-OP-001-OWNER-ACQUIRE': lambda: os.path.isdir(os.path.join(root,'namespace')) and stat.S_IMODE(os.stat('namespace',dir_fd=directory,follow_symlinks=False).st_mode)==0o700 and regular('owner.observed'),
 'WA-OP-002-SOURCE-PROMOTE': lambda: regular('source.ready') and raw('source.ready').startswith(b'devrites.workflow-source-ready.v1\n'),
 'WA-OP-002A-STALE-SOURCE-GC': lambda: absent('stale.source') and raw('owner.lock')==b'' and state==b'STALE_SOURCE_REMOVED\n',
 'WA-OP-003-JOURNAL-INIT': lambda: state==b'PREPARING\n',
 'WA-OP-004-STAGE-WRITE': lambda: (regular('stage') and raw('stage')==b'frozen-workflow\n') or (absent('source') and absent('stage') and absent('backup') and state==b'SOURCE_LOSS_PREINSTALL\n' and raw('next.route')==b'PLAN_VET_REPAIR\n'),
 'WA-OP-005-BACKUP-WRITE': lambda: regular('backup') and raw('backup')==raw('target'),
 'WA-OP-006-INSTALL': lambda: (regular('target',0o755) and raw('target')==raw('source') and state==b'INSTALLED\n') or (absent('source') and raw('target')==b'observed-active-preimage\n' and absent('stage') and absent('backup') and state==b'FAILED\n' and raw('next.route')==b'OFFLINE_RECOVERY\n' and raw('diagnostic')==b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R007-SOURCE-STALE-ACTIVE boundary_id=WA-B004-SOURCE-OPEN next_route=OFFLINE_RECOVERY\n'),
 'WA-OP-007-PROVE': lambda: (raw('proof.output')==b'WA-PROOF-001 PASS\n' and state==b'PROVED\n') or (absent('source') and raw('target')==b'observed-active-preimage\n' and absent('backup') and state==b'FAILED\n' and raw('next.route')==b'OFFLINE_RECOVERY\n' and raw('diagnostic')==b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R007-SOURCE-STALE-ACTIVE boundary_id=WA-B004-SOURCE-OPEN next_route=OFFLINE_RECOVERY\n'),
 'WA-OP-008-ROLLBACK': lambda: (raw('target')==b'target-preimage\n' and state==b'ROLLED_BACK\n') or (raw('target')==b'frozen-workflow\n' and state==b'BLOCKED_GATE\n' and raw('next.route')==b'BLOCKED_GATE\n' and raw('diagnostic')==b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R015-ROLLBACK-FAILED boundary_id=WA-B011-ROLLBACK next_route=BLOCKED_GATE\n'),
 'WA-OP-009-FAILURE-CLEANUP': lambda: absent('stage') and absent('backup') and state==b'FAILED\n',
 'WA-OP-010-SUCCESS-CLEANUP': lambda: all(absent(name) for name in ('stage','backup','source','source.ready','stale.source')) and state==b'CLEANED\n',
 'WA-OP-011-RETRY-HANDOFF': lambda: regular('source') and raw('attempt.history').startswith(b'1:'+b'a'*64+b':no-progress\n') and raw('attempt.history').endswith(str(value.get('attempt_epoch')).encode()+b':'+json.loads(raw('retry.control')).get('fingerprint','a'*64).encode()+b':pending\n') and state==b'PREPARING\n',
 'WA-OP-012-EXHAUSTION-GC': lambda: value.get('exhaustion_cause') in ('same-fingerprint-count','total-epoch-limit') and raw('exhaustion.cause')==(value['exhaustion_cause']+'\n').encode() and raw('evidence')==('state=EXHAUSTED\nexhaustion_cause='+value['exhaustion_cause']+'\n').encode() and all(absent(name) for name in ('source','source.ready','stage','backup')) and b'4:' not in raw('attempt.history') and state==b'EXHAUSTED\n',
 'WA-OP-013-EVIDENCE-UPDATE': lambda: raw('evidence')==b'prefix\n<!-- devrites-workflow-artifact-journal:start -->\nstate=CLEANED\n<!-- devrites-workflow-artifact-journal:end -->\nsuffix\n',
 'WA-OP-014-PRODUCT-SEPARATION': lambda: regular('product.comparison') and ((product_equal is True and state!=b'BLOCKED_GATE\n' and absent('diagnostic')) or (product_equal is False and state==b'BLOCKED_GATE\n' and raw('next.route')==b'BLOCKED_GATE\n' and raw('diagnostic')==b'WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R018-PRODUCT-IDENTITY-CHANGED boundary_id=WA-B014-PRODUCT-SEPARATION next_route=BLOCKED_GATE\n')),
 'WA-OP-015-VERIFY-EXISTING': lambda: state==b'CLEANED\n' and bool(raw('evidence')) and raw('product.observer-pass')==b'equal\n' and raw('verified')==hashlib.sha256(raw('target')).hexdigest().encode()+b'\n',
 }
 valid=operation in checks and checks[operation]()
 out['facts']={'state':None if state is None else state.decode().strip(),'route':None if raw('next.route') is None else raw('next.route').decode().strip(),'source_present':not absent('source'),'source_sha256':None if raw('source') is None else hashlib.sha256(raw('source')).hexdigest(),'stage_sha256':None if raw('stage') is None else hashlib.sha256(raw('stage')).hexdigest(),'target_sha256':None if raw('target') is None else hashlib.sha256(raw('target')).hexdigest(),'history':None if raw('attempt.history') is None else raw('attempt.history').decode(),'exhaustion_cause':value.get('exhaustion_cause'),'product_equal':product_equal}
 out['valid']=valid
 if valid: out['result']=fixed[operation]
os.close(directory); print(json.dumps(out,sort_keys=True,separators=(',',':')))
"""
    proc = subprocess.run(
        [sys.executable, "-c", program, str(root), json.dumps(fixed, sort_keys=True)], text=True,
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=3, check=False,
    )
    require(proc.returncode == 0 and proc.stderr == "", "operation observer process")
    return json.loads(proc.stdout)


def operation_observation_matches(observed: dict, row: list[str], state: str = "COMPLETE") -> bool:
    expected = [row[0], row[3], row[4], row[6], EXPECTED_OBSERVER_FACTS.get(row[0])]
    journal = observed.get("journal")
    if journal is None:
        return False
    value = journal["value"]
    expected_state = "EXHAUSTED" if row[0] == "WA-OP-012-EXHAUSTION-GC" and state == "COMPLETE" else state
    expected_journal = {"operation_id": row[0], "state": expected_state}
    if row[0] == "WA-OP-011-RETRY-HANDOFF":
        expected_journal.update({
            "attempt_epoch": 2,
            "prior_history_sha256": sha(b"1:" + b"a" * 64 + b":no-progress\n"),
        })
    elif row[0] == "WA-OP-012-EXHAUSTION-GC":
        expected_journal.update({
            "attempt_epoch": 3,
            "prior_history_sha256": sha(b"1:" + b"a" * 64 + b":no-progress\n2:" + b"a" * 64 + b":no-progress\n3:" + b"a" * 64 + b":no-progress\n"),
        })
        if expected_state == "EXHAUSTED":
            expected_journal["exhaustion_cause"] = "same-fingerprint-count"
    return (
        observed.get("valid") is True and observed.get("result") == expected
        and journal["mode"] == 0o600 and journal["links"] == 1
        and value == expected_journal
    )


def run_operation_death_matrix(rows: list[list[str]]) -> int:
    boundaries = 0
    for row in rows:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "before-intent"
            proc, ready_fd, gate_fd = operation_consumer(root, row)
            read_barrier(ready_fd, b"R")
            baseline = observe_operation(root)
            terminate_group(proc, 0.05)
            os.close(ready_fd)
            os.close(gate_fd)
            require(observe_operation(root) == baseline and baseline["journal"] is None,
                    f"pre-intent zero journal/target writes: {row[0]}")
            proc, ready_fd, gate_fd = operation_consumer(root, row)
            finish_operation_consumer(proc, ready_fd, gate_fd)
            require(operation_observation_matches(observe_operation(root), row),
                    f"pre-intent fresh resume: {row[0]}")
            boundaries += 1
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp) / "after-effect"
            proc, ready_fd, gate_fd = operation_consumer(root, row)
            read_barrier(ready_fd, b"R")
            os.write(gate_fd, b"x")
            read_barrier(ready_fd, b"I")
            os.write(gate_fd, b"x")
            read_barrier(ready_fd, b"E")
            terminate_group(proc, 0.05)
            os.close(ready_fd)
            os.close(gate_fd)
            partial = observe_operation(root)
            partial_state = "EXHAUSTED" if row[0] == "WA-OP-012-EXHAUSTION-GC" else "INTENT"
            require(operation_observation_matches(partial, row, partial_state),
                    f"post-effect durable partial: {row[0]}")
            proc, ready_fd, gate_fd = operation_consumer(root, row)
            finish_operation_consumer(proc, ready_fd, gate_fd)
            require(operation_observation_matches(observe_operation(root), row),
                    f"post-effect fresh resume: {row[0]}")
            boundaries += 1
    return boundaries


def reject_operation_mutants(rows: list[list[str]]) -> int:
    rejected = 0
    for row_index, row in enumerate(rows):
        for column in range(len(row)):
            mutant = row.copy(); mutant[column] += "-MUTANT"
            table_mutant = [candidate.copy() for candidate in rows]; table_mutant[row_index] = mutant
            require(operation_table_digest(table_mutant) != OPERATION_TABLE_SHA256,
                    f"operation table mutant survived: {row_index}:{column}")
            with tempfile.TemporaryDirectory() as tmp:
                fixture = Path(tmp)
                if column == 0:
                    proc, ready_fd, gate_fd = operation_consumer(fixture, mutant)
                    require(proc.wait(timeout=3) == 64, f"unknown operation dispatched: {row_index}")
                    os.close(ready_fd); os.close(gate_fd)
                else:
                    proc, ready_fd, gate_fd = operation_consumer(fixture, row)
                    finish_operation_consumer(proc, ready_fd, gate_fd)
                    observed = observe_operation(fixture)
                    require(operation_observation_matches(observed, row),
                            f"canonical operation failed under mutant probe: {row_index}:{column}")
                    if column in {3, 4, 6}:
                        require(not operation_observation_matches(observed, mutant),
                                f"observer fact mutant survived: {row_index}:{column}")
            rejected += 1
    return rejected


def check_owner_bootstrap_race() -> None:
    program = r"""
import fcntl,json,os,stat,sys
root,gate,ready,death=sys.argv[1],int(sys.argv[2]),int(sys.argv[3]),sys.argv[4]
os.umask(0o077)
rootfd=os.open(root,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW|os.O_CLOEXEC)
os.write(ready,b'R'); os.read(gate,1)
try: os.mkdir('.workflow-artifact-sources',0o700,dir_fd=rootfd); os.fsync(rootfd)
except FileExistsError: pass
if death=='after-directory': os._exit(86)
sourcefd=os.open('.workflow-artifact-sources',os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW|os.O_CLOEXEC,dir_fd=rootfd)
info=os.fstat(sourcefd)
if info.st_uid!=os.getuid() or stat.S_IMODE(info.st_mode)!=0o700: raise SystemExit(65)
try:
 lock=os.open('.owner.lock',os.O_RDWR|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,0o600,dir_fd=sourcefd); os.fsync(lock); os.fsync(sourcefd)
except FileExistsError: lock=os.open('.owner.lock',os.O_RDWR|os.O_NOFOLLOW|os.O_CLOEXEC,dir_fd=sourcefd)
info=os.fstat(lock)
if info.st_uid!=os.getuid() or not stat.S_ISREG(info.st_mode) or info.st_nlink!=1 or stat.S_IMODE(info.st_mode)!=0o600: raise SystemExit(66)
if death=='after-lock': os._exit(86)
os.write(ready,b'B'); os.read(gate,1)
try: fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
except BlockingIOError: os.write(ready,b'L'); raise SystemExit(7)
def write(name,payload):
 f=os.open(name,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW|os.O_CLOEXEC,0o600,dir_fd=rootfd); os.write(f,payload); os.fsync(f); os.close(f); os.fsync(rootfd)
pid=str(os.getpid()).encode(); write('journal.json',json.dumps({'owner_pid':os.getpid(),'state':'PREPARING'},sort_keys=True,separators=(',',':')).encode()+b'\n'); write('target',pid+b'\n')
os.write(ready,b'W'); os.read(gate,1); os.close(lock); os.close(sourcefd); os.close(rootfd)
"""
    def child(root: Path, death: str = "none", code: str = program) -> tuple[subprocess.Popen, int, int]:
        ready_read, ready_write = os.pipe(); gate_read, gate_write = os.pipe()
        proc = subprocess.Popen([sys.executable, "-c", code, str(root), str(gate_read), str(ready_write), death],
                                pass_fds=(gate_read, ready_write), start_new_session=True,
                                preexec_fn=lambda: os.umask(0o777),
                                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        os.close(gate_read); os.close(ready_write)
        return proc, ready_read, gate_write
    with tempfile.TemporaryDirectory() as tmp:
        death_root = Path(tmp) / "bootstrap-deaths"; death_root.mkdir()
        for boundary in ("after-directory", "after-lock"):
            proc, ready_fd, gate_fd = child(death_root, boundary)
            read_barrier(ready_fd, b"R"); os.write(gate_fd, b"x")
            require(proc.wait(timeout=3) == 86, f"owner bootstrap death: {boundary}")
            os.close(ready_fd); os.close(gate_fd)
            require(not (death_root / "journal.json").exists() and not (death_root / "target").exists(),
                    f"bootstrap death zero transaction writes: {boundary}")
        root = Path(tmp) / "race"; root.mkdir()
        children = [child(root), child(root)]
        require(not (root / ".workflow-artifact-sources").exists(), "owner race begins absent namespace")
        for _proc, ready_fd, _gate_fd in children: read_barrier(ready_fd, b"R")
        for _proc, _ready_fd, gate_fd in children: os.write(gate_fd, b"x")
        for _proc, ready_fd, _gate_fd in children: read_barrier(ready_fd, b"B")
        for _proc, _ready_fd, gate_fd in children: os.write(gate_fd, b"x")
        statuses = []
        for _proc, ready_fd, _gate_fd in children:
            selector = selectors.DefaultSelector(); selector.register(ready_fd, selectors.EVENT_READ)
            require(bool(selector.select(3)), "owner outcome barrier"); statuses.append(os.read(ready_fd, 1)); selector.close()
        require(sorted(statuses) == [b"L", b"W"], "owner race one winner")
        winner_index = statuses.index(b"W"); os.write(children[winner_index][2], b"x")
        results = []
        for proc, ready_fd, gate_fd in children:
            results.append(proc.wait(timeout=3)); os.close(ready_fd); os.close(gate_fd)
        require(sorted(results) == [0, 7], "owner race process outcomes")
        observer = subprocess.run([sys.executable, "-c", r"""
import json,os,stat,sys
root=sys.argv[1]; ns=os.lstat(root+'/.workflow-artifact-sources'); lock=os.lstat(root+'/.workflow-artifact-sources/.owner.lock')
j=json.loads(open(root+'/journal.json').read()); target=open(root+'/target').read(); print(json.dumps({'namespace_mode':stat.S_IMODE(ns.st_mode),'lock_mode':stat.S_IMODE(lock.st_mode),'lock_links':lock.st_nlink,'same_owner':target==str(j['owner_pid'])+'\n','journal_state':j['state']}))
""", str(root)], text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False, timeout=3)
        require(observer.returncode == 0 and observer.stderr == "" and json.loads(observer.stdout) == {
            "namespace_mode": 0o700, "lock_mode": 0o600, "lock_links": 1,
            "same_owner": True, "journal_state": "PREPARING",
        }, "independent owner race observer")
        mutant_root = Path(tmp) / "umask-mutant"; mutant_root.mkdir()
        mutant, ready_fd, gate_fd = child(mutant_root, code=program.replace("os.umask(0o077)\n", "", 1))
        read_barrier(ready_fd, b"R"); os.write(gate_fd, b"x")
        require(mutant.wait(timeout=3) != 0, "deleted bootstrap umask mutant rejected")
        os.close(ready_fd); os.close(gate_fd)
        require(stat.S_IMODE((mutant_root / ".workflow-artifact-sources").stat().st_mode) != 0o700,
                "hostile inherited umask independently observed")
    try:
        acquire_owner_lock(-1, "unsupported")
    except AssertionError as error:
        require(str(error) == "alternate owner lock primitive", "unsupported flock diagnostic")
    else:
        fail("unsupported flock accepted")


def load_actual_engine_output(root: Path) -> bytes:
    project = project_root_for_tests(root)
    with tempfile.TemporaryDirectory() as tmp:
        engine = Path(tmp) / "devrites-engine"
        built = subprocess.run(["go", "-C", str(project / "engine"), "build", "-o", str(engine), "."],
                               stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False)
        require(built.returncode == 0, "operation observer engine build")
        output = subprocess.run([str(engine), "version"], stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT, check=False, timeout=10)
        require(output.returncode == 0 and output.stdout.endswith(b"\n"), "actual engine observer output")
        return output.stdout


def check_operation_oracle(root: Path) -> None:
    global ACTUAL_ENGINE_OUTPUT
    ACTUAL_ENGINE_OUTPUT = load_actual_engine_output(root)
    rows = markdown_rows((root / MODULE_REL).read_text(), "WA-OP-")
    require(operation_table_digest(rows) == OPERATION_TABLE_SHA256, "canonical operation semantics")
    parsed_trace = operation_trace_from_table(rows)
    fixed_trace = expected_operation_trace()
    require(parsed_trace == fixed_trace, "fixed independent operation facts")
    for row_index, row in enumerate(rows):
        for column in range(len(row)):
            mutant = [candidate.copy() for candidate in rows]
            mutant[row_index][column] += "-MUTANT"
            require(operation_table_digest(mutant) != OPERATION_TABLE_SHA256, f"operation mutant {row_index}:{column}")
            if column in {0, 3, 4, 6}:
                try:
                    mutated_trace = operation_trace_from_table(mutant)
                except (AssertionError, KeyError):
                    continue
                require(mutated_trace != fixed_trace, f"operation fact mutant survived: {row_index}:{column}")
    for field in range(5):
        mutant = list(fixed_trace)
        changed = list(mutant[0]); changed[field] += "-MUTANT"; mutant[0] = tuple(changed)
        require(tuple(mutant) != parsed_trace, f"observer oracle mutant field {field}")
    with tempfile.TemporaryDirectory() as tmp:
        target = Path(tmp) / "target"
        atomic_write(target, b"frozen", 0o700)
        require(observe_regular_file(target) == (True, 0o700, 1, sha(b"frozen")), "independent target observer")
    require(run_operation_death_matrix(rows) == 32, "operation death boundary count")
    require(reject_operation_mutants(rows) == 128, "executable operation mutation count")
    check_owner_bootstrap_race()
    for operation_id, _pre, _post, route, _observer in parsed_trace:
        for boundary in ("before-intent", "after-operation"):
            if operation_id in {"WA-OP-010-SUCCESS-CLEANUP", "WA-OP-015-VERIFY-EXISTING"}:
                require(route in {"RESUME_CLEANUP", "route by finite diagnostic table"}, f"post-proof recovery: {boundary}")
            elif operation_id == "WA-OP-014-PRODUCT-SEPARATION":
                require(route == "BLOCKED_GATE", "product observer gate")


def secure_entry(path: Path, mode: int, directory: bool = False) -> None:
    info = path.lstat()
    require(info.st_uid == os.getuid() and (directory or info.st_nlink == 1), f"entry ownership/link: {path.name}")
    require(stat.S_IMODE(info.st_mode) == mode, f"entry mode: {path.name}")
    require(stat.S_ISDIR(info.st_mode) if directory else stat.S_ISREG(info.st_mode), f"entry type: {path.name}")


def fault_at(selected: str | None, boundary: str) -> None:
    if selected == boundary:
        raise RuntimeError(f"injected:{boundary}")


def source_handle_hex(slug: str, binding: str) -> str:
    slug_bytes = slug.encode()
    return sha(b"devrites.workflow-source.v1\0" + len(slug_bytes).to_bytes(4, "big") + slug_bytes + bytes.fromhex(binding))


def bootstrap_source_namespace(workspace_fd: int) -> tuple[int, int]:
    os.umask(0o077)
    try:
        os.mkdir(".workflow-artifact-sources", 0o700, dir_fd=workspace_fd)
        os.fsync(workspace_fd)
    except FileExistsError:
        pass
    namespace_fd = os.open(".workflow-artifact-sources", DIRECTORY_FLAGS, dir_fd=workspace_fd)
    validate_directory_fd(namespace_fd, 0o700)
    flags = os.O_RDWR | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC
    try:
        lock_fd = os.open(".owner.lock", flags, 0o600, dir_fd=namespace_fd)
        os.fsync(lock_fd)
        os.fsync(namespace_fd)
    except FileExistsError:
        lock_fd = os.open(".owner.lock", os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=namespace_fd)
    secure_file_info(os.fstat(lock_fd), {0o600})
    acquire_owner_lock(lock_fd)
    return namespace_fd, lock_fd


def ensure_source_file(namespace_fd: int, relative: str, data: bytes, lock_fd: int) -> None:
    info = entry_info_at(namespace_fd, relative)
    if info is None:
        atomic_write_at(namespace_fd, relative, data, 0o600, lock_fd)
        return
    secure_file_info(info, {0o600})
    require(read_file_at(namespace_fd, relative, len(data), {0o600}) == data, f"source resume bytes: {relative}")


def promote_source(namespace_fd: int, lock_fd: int, handle_hex: str, binding: str, identity: str,
                   contents: tuple[bytes, ...], fault: str | None = None) -> str:
    canonical = handle_hex
    preparing = f".{handle_hex}.preparing"
    namespace_entries = set(os.listdir(namespace_fd))
    require(namespace_entries <= {".owner.lock", canonical, preparing}, "unknown source namespace entry")
    require(not ({canonical, preparing} <= namespace_entries), "ambiguous source promotion state")
    if entry_info_at(namespace_fd, canonical) is not None:
        bundle_fd = os.open(canonical, DIRECTORY_FLAGS, dir_fd=namespace_fd)
        try:
            validate_directory_fd(bundle_fd, 0o700)
        finally:
            os.close(bundle_fd)
        return canonical
    try:
        os.mkdir(preparing, 0o700, dir_fd=namespace_fd)
        os.fsync(namespace_fd)
    except FileExistsError:
        pass
    preparing_fd = os.open(preparing, DIRECTORY_FLAGS, dir_fd=namespace_fd)
    try:
        validate_directory_fd(preparing_fd, 0o700)
    finally:
        os.close(preparing_fd)
    fault_at(fault, "mkdir")
    authority = (
        "devrites.workflow-source-authority.v1\n"
        f"handle=wsrc:{handle_hex}\nreadiness={binding}\n"
    ).encode()
    ensure_source_file(namespace_fd, f"{preparing}/.authority", authority, lock_fd)
    fault_at(fault, "authority")
    for index, content in enumerate(contents):
        ensure_source_file(namespace_fd, f"{preparing}/{index:08x}", content, lock_fd)
        fault_at(fault, f"source-{index}")
    ready = f"devrites.workflow-source-ready.v1\ncount={len(contents)}\nidentity={identity}\n".encode()
    ensure_source_file(namespace_fd, f"{preparing}/.ready", ready, lock_fd)
    fault_at(fault, "ready")
    preparing_fd = os.open(preparing, DIRECTORY_FLAGS, dir_fd=namespace_fd)
    try:
        os.fsync(preparing_fd)
    finally:
        os.close(preparing_fd)
    fault_at(fault, "temp-sync")
    src_fd = os.dup(namespace_fd); dst_fd = os.dup(namespace_fd)
    try:
        os.rename(preparing, canonical, src_dir_fd=src_fd, dst_dir_fd=dst_fd)
        fault_at(fault, "rename")
        os.fsync(dst_fd)
    finally:
        os.close(src_fd); os.close(dst_fd)
    fault_at(fault, "parent-sync")
    return canonical


def validate_source_bundle(namespace_fd: int, bundle: str, handle_hex: str, binding: str,
                           identity: str, count: int) -> tuple[bytes, ...]:
    bundle_fd = os.open(bundle, DIRECTORY_FLAGS, dir_fd=namespace_fd)
    try:
        validate_directory_fd(bundle_fd, 0o700)
        expected = {".authority", ".ready", *(f"{index:08x}" for index in range(count))}
        require(set(os.listdir(bundle_fd)) == expected, "source cardinality")
        authority = f"devrites.workflow-source-authority.v1\nhandle=wsrc:{handle_hex}\nreadiness={binding}\n".encode()
        ready = f"devrites.workflow-source-ready.v1\ncount={count}\nidentity={identity}\n".encode()
        require(read_file_at(bundle_fd, ".authority", len(authority), {0o600}) == authority, "source authority")
        require(read_file_at(bundle_fd, ".ready", len(ready), {0o600}) == ready, "source readiness")
        return tuple(read_file_at(bundle_fd, f"{index:08x}", 4096, {0o600}) for index in range(count))
    finally:
        os.close(bundle_fd)


def stale_marker_bytes(old_hex: str, current_binding: str, count: int) -> bytes:
    return (
        "devrites.workflow-source-stale-cleanup.v1\n"
        f"old_handle=wsrc:{old_hex}\ncurrent_readiness={current_binding}\ncount={count}\n"
    ).encode()


def stale_intent_bytes(old_hex: str, old_binding: str, current_binding: str,
                       identity: str, count: int) -> bytes:
    return (
        "devrites.workflow-source-stale-intent.v1\n"
        f"old_handle=wsrc:{old_hex}\nold_readiness={old_binding}\n"
        f"current_readiness={current_binding}\nidentity={identity}\ncount={count}\n"
    ).encode()


def read_lock_intent(lock_fd: int, limit: int) -> bytes:
    info = os.fstat(lock_fd)
    secure_file_info(info, {0o600})
    require(info.st_size <= limit, "stale intent bound")
    os.lseek(lock_fd, 0, os.SEEK_SET)
    return read_fd_bounded(lock_fd, limit)


def write_lock_intent(lock_fd: int, intent: bytes) -> None:
    current = read_lock_intent(lock_fd, len(intent))
    require(current in {b"", intent}, "stale cleanup intent authority")
    if current == intent:
        return
    os.ftruncate(lock_fd, 0)
    os.lseek(lock_fd, 0, os.SEEK_SET)
    complete_write(lock_fd, intent)
    os.fsync(lock_fd)
    require(read_lock_intent(lock_fd, len(intent)) == intent, "stale cleanup intent readback")


def clear_lock_intent(lock_fd: int, intent: bytes) -> None:
    require(read_lock_intent(lock_fd, len(intent)) == intent, "stale cleanup intent completion")
    os.ftruncate(lock_fd, 0)
    os.fsync(lock_fd)
    require(read_lock_intent(lock_fd, len(intent)) == b"", "stale cleanup intent cleared")


def validate_stale_bundle(bundle_fd: int, old_hex: str, old_binding: str, identity: str,
                          current_binding: str, count: int, marker_required: bool) -> set[str]:
    validate_directory_fd(bundle_fd, 0o700)
    indexed = {f"{index:08x}" for index in range(count)}
    allowed = {".authority", ".ready", ".stale-cleanup", *indexed}
    entries = set(os.listdir(bundle_fd))
    require(entries <= allowed, "unknown stale source entry")
    if ".authority" in entries:
        authority = f"devrites.workflow-source-authority.v1\nhandle=wsrc:{old_hex}\nreadiness={old_binding}\n".encode()
        require(read_file_at(bundle_fd, ".authority", len(authority), {0o600}) == authority, "stale authority")
    if ".ready" in entries:
        ready = f"devrites.workflow-source-ready.v1\ncount={count}\nidentity={identity}\n".encode()
        require(read_file_at(bundle_fd, ".ready", len(ready), {0o600}) == ready, "stale readiness")
    for name in entries & indexed:
        info = os.stat(name, dir_fd=bundle_fd, follow_symlinks=False)
        secure_file_info(info, {0o600})
    if marker_required or ".stale-cleanup" in entries:
        marker = stale_marker_bytes(old_hex, current_binding, count)
        require(read_file_at(bundle_fd, ".stale-cleanup", len(marker), {0o600}) == marker, "stale cleanup binding")
    return entries


def observe_stale_gc_preconditions(transaction_fd: int, target_preimages: tuple[tuple[str, dict], ...]) -> None:
    require(entry_info_at(transaction_fd, "evidence.md") is None, "stale GC journal absence")
    require(entry_info_at(transaction_fd, ".evidence.md.workflow-artifact.tmp") is None, "stale GC journal temporary absence")
    require(len(target_preimages) > 0, "stale GC target preimages")
    for relative, expected in target_preimages:
        relative_components(relative)
        require(file_record_at(transaction_fd, relative) == expected, f"stale GC target preimage: {relative}")


def stale_source_gc(namespace_fd: int, transaction_fd: int, lock_fd: int, old_hex: str,
                    slug: str, old_binding: str, current_binding: str, identity: str,
                    count: int, target_preimages: tuple[tuple[str, dict], ...],
                    fault: str | None = None) -> None:
    require(source_handle_hex(slug, old_binding) == old_hex, "stale basename authority")
    require(source_handle_hex(slug, current_binding) != old_hex, "stale current handle")
    observe_stale_gc_preconditions(transaction_fd, target_preimages)
    canonical = old_hex
    stale = f".{old_hex}.stale-cleaning"
    namespace_entries = set(os.listdir(namespace_fd))
    require(namespace_entries <= {".owner.lock", canonical, stale}, "unknown source namespace entry")
    require(not ({canonical, stale} <= namespace_entries), "ambiguous stale source state")
    marker = stale_marker_bytes(old_hex, current_binding, count)
    intent = stale_intent_bytes(old_hex, old_binding, current_binding, identity, count)
    existing_intent = read_lock_intent(lock_fd, len(intent))
    require(existing_intent in {b"", intent}, "stale cleanup intent authority")
    if canonical in namespace_entries:
        canonical_fd = os.open(canonical, DIRECTORY_FLAGS, dir_fd=namespace_fd)
        try:
            entries = validate_stale_bundle(canonical_fd, old_hex, old_binding, identity, current_binding, count, False)
            expected = {".authority", ".ready", *(f"{index:08x}" for index in range(count))}
            require(entries == expected or entries == expected | {".stale-cleanup"}, "stale exact indexed cardinality")
            if ".stale-cleanup" not in entries:
                atomic_write_at(canonical_fd, ".stale-cleanup", marker, 0o600, lock_fd)
            validate_stale_bundle(canonical_fd, old_hex, old_binding, identity, current_binding, count, True)
            fault_at(fault, "marker")
            os.fsync(canonical_fd)
        finally:
            os.close(canonical_fd)
        fault_at(fault, "bundle-sync")
        write_lock_intent(lock_fd, intent)
        fault_at(fault, "intent")
        src_fd = os.dup(namespace_fd); dst_fd = os.dup(namespace_fd)
        try:
            os.rename(canonical, stale, src_dir_fd=src_fd, dst_dir_fd=dst_fd)
            fault_at(fault, "stale-rename")
            os.fsync(dst_fd)
        finally:
            os.close(src_fd); os.close(dst_fd)
        fault_at(fault, "rename-sync")
    elif stale in namespace_entries:
        require(existing_intent == intent, "stale-cleaning durable intent")
    else:
        if existing_intent == intent:
            os.fsync(namespace_fd)
            clear_lock_intent(lock_fd, intent)
        return
    stale_fd = os.open(stale, DIRECTORY_FLAGS, dir_fd=namespace_fd)
    try:
        entries = validate_stale_bundle(stale_fd, old_hex, old_binding, identity, current_binding, count, False)
        ordered = [".authority", ".ready", *(f"{index:08x}" for index in range(count)), ".stale-cleanup"]
        require(any(entries == set(ordered[index:]) for index in range(len(ordered) + 1)),
                "stale cleanup deletion-order suffix")
        for index, name in enumerate(ordered):
            if name in entries:
                info = os.stat(name, dir_fd=stale_fd, follow_symlinks=False)
                secure_file_info(info, {0o600})
                os.unlink(name, dir_fd=stale_fd)
                os.fsync(stale_fd)
            fault_at(fault, f"unlink-{index}")
    finally:
        os.close(stale_fd)
    os.rmdir(stale, dir_fd=namespace_fd)
    fault_at(fault, "rmdir")
    os.fsync(namespace_fd)
    fault_at(fault, "gc-sync")
    clear_lock_intent(lock_fd, intent)
    fault_at(fault, "intent-clear")


def source_fixture() -> tuple[tempfile.TemporaryDirectory, int, int, int, str, str, str, str, str, tuple[bytes, ...], tuple[tuple[str, dict], ...]]:
    temporary = tempfile.TemporaryDirectory()
    workspace_fd = os.open(temporary.name, DIRECTORY_FLAGS)
    namespace_fd, lock_fd = bootstrap_source_namespace(workspace_fd)
    atomic_write_at(workspace_fd, "targets/present", b"frozen-preimage", 0o600, lock_fd)
    target_preimages = (
        ("targets/present", file_record_at(workspace_fd, "targets/present")),
        ("targets/absent", {"state": "absent"}),
    )
    slug = "demo"
    old_binding = "b" * 64
    current_binding = "d" * 64
    old_hex = source_handle_hex(slug, old_binding)
    identity = "c" * 64
    contents = (b"one", b"two")
    return temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages


def close_source_fixture(temporary: tempfile.TemporaryDirectory, workspace_fd: int,
                         namespace_fd: int, lock_fd: int) -> None:
    os.close(lock_fd); os.close(namespace_fd); os.close(workspace_fd); temporary.cleanup()


def check_source_lifecycle() -> None:
    promotion_boundaries = ["mkdir", "authority", "source-0", "source-1", "ready", "temp-sync", "rename", "parent-sync"]
    for boundary in promotion_boundaries:
        fixture = source_fixture()
        temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
        try:
            try:
                promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents, boundary)
            except RuntimeError:
                pass
            bundle = promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents)
            require(validate_source_bundle(namespace_fd, bundle, old_hex, old_binding, identity, 2) == contents, f"promotion resume: {boundary}")
        finally:
            close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    for canonical_exists in (False, True):
        fixture = source_fixture()
        temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
        try:
            if canonical_exists:
                promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents)
            os.mkdir("unrelated", 0o700, dir_fd=namespace_fd); os.fsync(namespace_fd)
            before = set(os.listdir(namespace_fd))
            try:
                promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents)
            except AssertionError:
                pass
            else:
                fail(f"promotion accepted unknown namespace sibling: canonical={canonical_exists}")
            require(set(os.listdir(namespace_fd)) == before,
                    f"promotion changed unknown namespace sibling state: canonical={canonical_exists}")
            if not canonical_exists:
                require(before == {".owner.lock", "unrelated"},
                        "unknown promotion fixture starts beside owner lock")
        finally:
            close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    gc_boundaries = ["marker", "bundle-sync", "intent", "stale-rename", "rename-sync", *(f"unlink-{index}" for index in range(5)), "rmdir", "gc-sync", "intent-clear"]
    for boundary in gc_boundaries:
        fixture = source_fixture()
        temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
        try:
            promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents)
            try:
                stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding, current_binding, identity, 2, target_preimages, fault=boundary)
            except RuntimeError:
                pass
            stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding, current_binding, identity, 2, target_preimages)
            require(entry_info_at(namespace_fd, old_hex) is None and entry_info_at(namespace_fd, f".{old_hex}.stale-cleaning") is None, f"stale GC resume: {boundary}")
            require(read_lock_intent(lock_fd, 512) == b"", f"stale GC intent cleanup: {boundary}")
        finally:
            close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    fixture = source_fixture()
    temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
    try:
        promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents)
        try:
            stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding,
                            current_binding, identity, 2, target_preimages, fault="stale-rename")
        except RuntimeError:
            pass
        stale = f".{old_hex}.stale-cleaning"
        stale_fd = os.open(stale, DIRECTORY_FLAGS, dir_fd=namespace_fd)
        try:
            for name in os.listdir(stale_fd):
                if name != ".authority":
                    os.unlink(name, dir_fd=stale_fd)
            os.fsync(stale_fd)
        finally:
            os.close(stale_fd)
        try:
            stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding,
                            current_binding, identity, 2, target_preimages)
        except AssertionError:
            pass
        else:
            fail("impossible stale-cleanup suffix accepted")
        require(entry_info_at(namespace_fd, stale) is not None,
                "impossible stale-cleanup state changed")
    finally:
        close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    for forged_name, forged_bytes in (
        (".authority", b"devrites.workflow-source-authority.v1\nhandle=wsrc:" + b"0" * 64 + b"\nreadiness=" + b"b" * 64 + b"\n"),
        (".ready", b"devrites.workflow-source-ready.v1\ncount=9\nidentity=" + b"c" * 64 + b"\n"),
        ("00000000", None),
        (".stale-cleanup", b"wrong binding\n"),
    ):
        fixture = source_fixture()
        temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
        try:
            bundle = promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents)
            bundle_fd = os.open(bundle, DIRECTORY_FLAGS, dir_fd=namespace_fd)
            try:
                if forged_name == "00000000":
                    os.unlink(forged_name, dir_fd=bundle_fd)
                    os.symlink("00000001", forged_name, dir_fd=bundle_fd)
                else:
                    unlink_file_at(bundle_fd, forged_name, missing_ok=True)
                    atomic_write_at(bundle_fd, forged_name, forged_bytes, 0o600, lock_fd)
            finally:
                os.close(bundle_fd)
            try:
                stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding, current_binding, identity, 2, target_preimages)
            except (AssertionError, OSError):
                pass
            else:
                fail(f"forged allowlisted stale source accepted: {forged_name}")
            require(entry_info_at(namespace_fd, old_hex) is not None, f"forged stale source deleted: {forged_name}")
        finally:
            close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    for condition in ("journal", "journal-temp", "target-present", "target-absent"):
        fixture = source_fixture()
        temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
        try:
            promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents)
            if condition == "journal":
                atomic_write_at(workspace_fd, "evidence.md", b"journal", 0o600, lock_fd)
            elif condition == "journal-temp":
                atomic_write_at(workspace_fd, ".evidence.md.workflow-artifact.tmp", b"journal-temp", 0o600, lock_fd)
            elif condition == "target-present":
                atomic_write_at(workspace_fd, "targets/present", b"drifted", 0o600, lock_fd)
            else:
                atomic_write_at(workspace_fd, "targets/absent", b"created", 0o600, lock_fd)
            try:
                stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding, current_binding, identity, 2, target_preimages)
            except AssertionError:
                pass
            else:
                fail(f"stale source deletion precondition accepted: {condition}")
            require(entry_info_at(namespace_fd, old_hex) is not None, f"stale source precondition deleted bundle: {condition}")
        finally:
            close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    fixture = source_fixture()
    temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
    try:
        stale = f".{old_hex}.stale-cleaning"
        os.mkdir(stale, 0o700, dir_fd=namespace_fd); os.fsync(namespace_fd)
        try:
            stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding, current_binding, identity, 2, target_preimages)
        except AssertionError:
            pass
        else:
            fail("forged empty stale-cleaning directory accepted")
        require(entry_info_at(namespace_fd, stale) is not None, "forged empty stale-cleaning directory deleted")
    finally:
        close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    canonical_intent_mutants = (
        stale_intent_bytes("0" * 64, "b" * 64, "d" * 64, "c" * 64, 2)[:-1],
        stale_intent_bytes("0" * 64, "b" * 64, "d" * 64, "c" * 64, 2) + b"extra\n",
        stale_intent_bytes("0" * 64, "b" * 64, "d" * 64, "e" * 64, 2),
    )
    for malformed_intent in canonical_intent_mutants:
        fixture = source_fixture()
        temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
        try:
            os.ftruncate(lock_fd, 0); os.lseek(lock_fd, 0, os.SEEK_SET)
            complete_write(lock_fd, malformed_intent); os.fsync(lock_fd)
            before = bytes(malformed_intent)
            try:
                stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding,
                                current_binding, identity, 2, target_preimages)
            except AssertionError:
                pass
            else:
                fail("malformed or orphan stale intent accepted")
            os.lseek(lock_fd, 0, os.SEEK_SET)
            require(read_fd_bounded(lock_fd, len(before) + 1) == before,
                    "malformed or orphan stale intent changed")
            require(set(os.listdir(namespace_fd)) == {".owner.lock"},
                    "orphan stale intent created relation")
        finally:
            close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    fixture = source_fixture()
    temporary, workspace_fd, namespace_fd, lock_fd, slug, old_binding, current_binding, old_hex, identity, contents, target_preimages = fixture
    try:
        promote_source(namespace_fd, lock_fd, old_hex, old_binding, identity, contents)
        os.mkdir("unrelated", 0o700, dir_fd=namespace_fd); os.fsync(namespace_fd)
        try:
            stale_source_gc(namespace_fd, workspace_fd, lock_fd, old_hex, slug, old_binding, current_binding, identity, 2, target_preimages)
        except AssertionError:
            pass
        else:
            fail("unrelated source namespace entry accepted")
        require(entry_info_at(namespace_fd, old_hex) is not None and entry_info_at(namespace_fd, "unrelated") is not None, "unrelated source namespace entry changed")
    finally:
        close_source_fixture(temporary, workspace_fd, namespace_fd, lock_fd)
    with tempfile.TemporaryDirectory() as tmp:
        path = Path(tmp) / "source"; replacement = Path(tmp) / "replacement"
        path.write_bytes(b"retained"); replacement.write_bytes(b"swapped")
        fd = os.open(path, FILE_READ_FLAGS)
        os.replace(replacement, path)
        try:
            retained = os.read(fd, 64)
        finally:
            os.close(fd)
        require(retained == b"retained" and sha(retained) == sha(b"retained"), "held descriptor source swap")


def check_filesystem_adversaries() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        regular = root / "regular"
        regular.write_bytes(b"x"); os.chmod(regular, 0o600)
        secure_entry(regular, 0o600)
        hard = root / "hard"; os.link(regular, hard)
        for path, mode, directory in ((regular, 0o600, False),):
            try:
                secure_entry(path, mode, directory)
            except AssertionError:
                pass
            else:
                fail("hard-linked regular accepted")
        hard.unlink(); regular.unlink()
        regular.write_bytes(b"x"); os.chmod(regular, 0o644)
        try:
            secure_entry(regular, 0o600)
        except AssertionError:
            pass
        else:
            fail("wrong mode accepted")
        symlink = root / "link"; symlink.symlink_to(regular)
        try:
            secure_entry(symlink, 0o600)
        except AssertionError:
            pass
        else:
            fail("symlink accepted")
        fifo = root / "fifo"; os.mkfifo(fifo, 0o600)
        try:
            secure_entry(fifo, 0o600)
        except AssertionError:
            pass
        else:
            fail("non-regular accepted")


def check_complete_write_matrix() -> None:
    failures = [
        lambda _fd, _view: None, lambda _fd, _view: 1.0,
        lambda _fd, _view: (_ for _ in ()).throw(OSError(errno.ENOSPC, "injected")),
        lambda _fd, _view: (_ for _ in ()).throw(OSError(errno.EIO, "injected")),
    ]
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        for index, writer in enumerate(failures):
            path = root / f"failure-{index}"
            try:
                atomic_write(path, b"payload", 0o600, writer)
            except OSError:
                pass
            else:
                fail(f"complete-write failure accepted: {index}")
            temporary = root / f".{path.name}.workflow-artifact.tmp"
            require(not path.exists() and temporary.exists(), "failed atomic write retained exact recoverable temporary")
            atomic_write(path, b"payload", 0o600)
            require(path.read_bytes() == b"payload" and not temporary.exists(), "failed atomic write recovery")
        for death_after in range(1, 8):
            path = root / f"partial-{death_after}"
            fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
            written = 0
            def die_after(raw_fd, view, limit=death_after):
                nonlocal written
                if written == limit:
                    raise InterruptedError("injected death")
                count = min(1, limit - written, len(view))
                written += count
                return os.write(raw_fd, view[:count])
            try:
                complete_write(fd, b"12345678", die_after)
            except InterruptedError:
                pass
            finally:
                os.close(fd)
            require(path.read_bytes() == b"12345678"[:death_after], f"partial write boundary: {death_after}")
            path.unlink()
    source = SCRIPT.read_text()
    require("src_dir_fd=src_fd" in source and "dst_dir_fd=dst_fd" in source, "relative replacement directory handles")
    require("path.parent.mkdir" + "(parents=True" not in source, "full-Path atomic traversal")


def atomic_death_child(root: str, boundary: str) -> None:
    root_fd = os.open(root, DIRECTORY_FLAGS)
    try:
        try:
            lock_fd = os.open(".owner.lock", os.O_RDWR | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600, dir_fd=root_fd)
            os.fsync(lock_fd); os.fsync(root_fd)
        except FileExistsError:
            lock_fd = os.open(".owner.lock", os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=root_fd)
        acquire_owner_lock(lock_fd)
        atomic_write_at(root_fd, "nested/target", b"crash-resumable", 0o600, lock_fd, death_boundary=boundary)
    finally:
        os.close(root_fd)


def check_atomic_write_death_recovery() -> None:
    for boundary, exit_code in (("after-create", 86), ("after-sync", 87)):
        with tempfile.TemporaryDirectory() as tmp:
            child = subprocess.run([str(SCRIPT), "--atomic-death-fixture", tmp, boundary], check=False,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=5)
            require(child.returncode == exit_code, f"atomic subprocess death boundary: {boundary}")
            root_fd = os.open(tmp, DIRECTORY_FLAGS)
            lock_fd = os.open(".owner.lock", os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=root_fd)
            try:
                acquire_owner_lock(lock_fd)
                atomic_write_at(root_fd, "nested/target", b"crash-resumable", 0o600, lock_fd)
                require(read_file_at(root_fd, "nested/target", 64, {0o600}) == b"crash-resumable", f"atomic death recovery: {boundary}")
                require(entry_info_at(root_fd, "nested/.target.workflow-artifact.tmp") is None, f"atomic temp reconciled: {boundary}")
            finally:
                os.close(lock_fd); os.close(root_fd)
    with tempfile.TemporaryDirectory() as tmp:
        root_fd = os.open(tmp, DIRECTORY_FLAGS)
        lock_fd, lock_name = fixture_lock()
        try:
            parent_fd = open_dir_components(root_fd, ("nested",), create=True)
            try:
                os.symlink("elsewhere", ".target.workflow-artifact.tmp", dir_fd=parent_fd)
            finally:
                os.close(parent_fd)
            try:
                atomic_write_at(root_fd, "nested/target", b"payload", 0o600, lock_fd)
            except OSError:
                pass
            else:
                fail("untrusted atomic temporary reconciled")
            require(entry_info_at(root_fd, "nested/.target.workflow-artifact.tmp") is not None, "untrusted atomic temporary preserved")
        finally:
            os.close(lock_fd); os.unlink(lock_name); os.close(root_fd)


def check_descriptor_ancestor_mutants() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        workspace = Path(tmp) / "workspace"; workspace.mkdir(mode=0o700)
        escape = Path(tmp) / "escape"; escape.mkdir(mode=0o700)
        root_fd = os.open(workspace, DIRECTORY_FLAGS)
        lock_fd = os.open(".owner.lock", os.O_RDWR | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600, dir_fd=root_fd)
        acquire_owner_lock(lock_fd)
        ancestor_fd = open_dir_components(root_fd, ("ancestor",), create=True)
        os.rename(workspace / "ancestor", workspace / "held-ancestor")
        (workspace / "ancestor").symlink_to(escape, target_is_directory=True)
        try:
            atomic_write_at(ancestor_fd, "target", b"held", 0o600, lock_fd)
            require((workspace / "held-ancestor/target").read_bytes() == b"held" and not (escape / "target").exists(), "ancestor swap held descriptor")
            try:
                atomic_write_at(root_fd, "ancestor/blocked", b"blocked", 0o600, lock_fd)
            except OSError:
                pass
            else:
                fail("symlink intermediate accepted")
            require(not (escape / "blocked").exists(), "symlink intermediate escape")
            atomic_write_at(root_fd, "prune/a/file", b"prune", 0o600, lock_fd)
            unlink_file_at(root_fd, "prune/a/file")
            prune_empty_parents(root_fd, "prune/a/file")
            require(entry_info_at(root_fd, "prune") is None, "descriptor-relative empty-parent prune")
        finally:
            os.close(ancestor_fd); os.close(lock_fd); os.close(root_fd)


def check_generator_descriptor_confinement() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(os.path.realpath(tmp))
        ancestor = base / "ancestor"
        repo = ancestor / "repo"
        (repo / "scripts").mkdir(parents=True)
        (repo / "pack/.claude/skills").mkdir(parents=True)
        (repo / "private/stage").mkdir(parents=True)
        (repo / "pack/.claude/skills/source").write_bytes(b"held")
        os.mkfifo(repo / "private/derived", 0o600)
        os.mkfifo(repo / "private/release", 0o600)
        (repo / "scripts/build-host-artifacts.sh").write_text(
            "#!/usr/bin/env bash\nset -euo pipefail\n"
            "OUT_ROOT=\"$DEVRITES_HOST_ARTIFACT_DIR\"\n"
            "exec 8<> ../../../derived\n"
            "exec 9<> ../../../release\n"
            "printf x >&8\n"
            "IFS= read -r -t 3 token <&9\n"
            "[ \"$token\" = go ]\n"
            "mkdir -p \"$OUT_ROOT\"\n"
            "cat pack/.claude/skills/source > \"$OUT_ROOT/held\"\n"
        )
        attacker = base / "attacker"
        (attacker / "repo/scripts").mkdir(parents=True)
        (attacker / "repo/pack/.claude/skills").mkdir(parents=True)
        (attacker / "repo/private/stage").mkdir(parents=True)
        (attacker / "repo/pack/.claude/skills/source").write_bytes(b"attacked")
        (attacker / "repo/scripts/build-host-artifacts.sh").write_text("replacement")
        attacker_fd = os.open(attacker / "repo", DIRECTORY_FLAGS)
        repo_fd = os.open(repo, DIRECTORY_FLAGS)
        before = manifest_at(attacker_fd, set(), "")
        actor_code = (
            "import os,sys;"
            "base=sys.argv[1];"
            "derived=os.path.join(base,'ancestor/repo/private/derived');"
            "fd=os.open(derived,os.O_RDONLY);token=os.read(fd,1);os.close(fd);"
            "os._exit(2) if token!=b'x' else None;"
            "os.rename(os.path.join(base,'ancestor'),os.path.join(base,'held-ancestor'));"
            "os.symlink(os.path.join(base,'attacker'),os.path.join(base,'ancestor'),target_is_directory=True);"
            "release=os.path.join(base,'held-ancestor/repo/private/release');"
            "fd=os.open(release,os.O_WRONLY);os.write(fd,b'go\\n');os.close(fd)"
        )
        actor = subprocess.Popen(
            [sys.executable, "-c", actor_code, str(base)],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        )
        try:
            generated_ok, generated_reason, _generated_output = run_private_generator(
                repo_fd, "private/stage",
                time.monotonic() + DELIVERY_AGGREGATE_TIMEOUT_SECONDS,
            )
            try:
                actor.wait(timeout=3)
            except subprocess.TimeoutExpired:
                actor.kill(); actor.wait()
                fail("generator swap actor timeout")
            require(actor.returncode == 0, "generator swap actor")
            require(generated_ok, f"descriptor-held generator fixture: {generated_reason}")
            require((base / "held-ancestor/repo/private/stage/held").read_bytes() == b"held",
                    "descriptor-held generator source and stage")
            require(manifest_at(attacker_fd, set(), "") == before,
                    "generator replacement tree untouched")
        finally:
            if actor.poll() is None:
                actor.kill(); actor.wait()
            if actor.stdout is not None:
                actor.stdout.close()
            os.close(repo_fd); os.close(attacker_fd)


def check_absolute_directory_acquisition() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(os.path.realpath(tmp))
        real = base / "real"; real.mkdir(mode=0o700)
        candidate = real / "candidate"; candidate.mkdir(mode=0o700)
        intermediate = base / "intermediate"; intermediate.symlink_to(real, target_is_directory=True)
        try:
            open_absolute_directory(intermediate / "candidate")
        except OSError:
            pass
        else:
            fail("absolute root intermediate symlink accepted")
        ancestor = base / "ancestor"; ancestor.mkdir(mode=0o700)
        held_candidate = ancestor / "candidate"; held_candidate.mkdir(mode=0o700)
        (held_candidate / "identity").write_bytes(b"held-root")
        os.chmod(held_candidate / "identity", 0o600)
        held_fd = open_absolute_directory(held_candidate)
        escape = base / "escape"; escape.mkdir(mode=0o700)
        (escape / "candidate").mkdir(mode=0o700)
        os.rename(ancestor, base / "held-ancestor")
        ancestor.symlink_to(escape, target_is_directory=True)
        try:
            require(read_file_at(held_fd, "identity", 64, {0o600}) == b"held-root", "held absolute root after ancestor swap")
            try:
                open_absolute_directory(held_candidate)
            except OSError:
                pass
            else:
                fail("swapped absolute root ancestor accepted")
            require(not (escape / "candidate/identity").exists(), "absolute root ancestor swap escaped")
        finally:
            os.close(held_fd)


def route_from_action(value: str) -> str:
    match = re.search(r"\b(?:ROOT_TRANSACTION|PLAN_VET_REPAIR|OFFLINE_RECOVERY|RESUME_CLEANUP|PROVE_AND_RETURN|VERIFY_EXISTING|NO_BACKFILL|WAIT_ACTIVE_OWNER|BLOCKED_EXHAUSTED|BLOCKED_GATE)\b", value)
    require(match is not None, "canonical route token")
    return match.group(0)


def parse_diagnostics(rows: list[list[str]]) -> dict[str, tuple[tuple[str, ...], str, str]]:
    require(all(len(row) == 4 for row in rows), "diagnostic table width")
    boundary_pattern = r"(WA-B[0-9]{3}-[A-Z]+(?:-[A-Z]+)*)(?: or (WA-B[0-9]{3}-[A-Z]+(?:-[A-Z]+)*))?"
    parsed = {}
    for reason, boundary_cell, meaning, route in rows:
        boundary_match = re.fullmatch(boundary_pattern, boundary_cell)
        require(boundary_match is not None and reason not in parsed, "diagnostic row identity")
        boundaries = tuple(value for value in boundary_match.groups() if value is not None)
        require(re.fullmatch(r"WA-R[0-9]{3}-[A-Z]+(?:-[A-Z]+)*", reason) is not None, "diagnostic reason grammar")
        require(re.fullmatch(r"[A-Z]+(?:_[A-Z]+)*", route) is not None, "diagnostic route grammar")
        parsed[reason] = (boundaries, meaning, route)
    return parsed


def validate_diagnostics(rows: list[list[str]]) -> dict[str, tuple[tuple[str, ...], str, str]]:
    parsed = parse_diagnostics(rows)
    require(set(parsed) == EXPECTED_REASON_IDS, "exact diagnostic reason IDs")
    boundaries = {boundary for values, _meaning, _route in parsed.values() for boundary in values}
    require(boundaries == EXPECTED_BOUNDARY_IDS, "exact diagnostic boundary IDs")
    require(list(parsed.items()) == list(EXPECTED_DIAGNOSTICS.items()), "exact ordered diagnostic rows")
    require(parsed["WA-R010-WRITE-FAILED"][0] == ("WA-B006-STAGE-WRITE", "WA-B007-BACKUP-WRITE"), "two-boundary write diagnostic")
    return parsed


def unknown_active_state_diagnostic(module: str, active_state: str) -> str:
    operation_rows = markdown_rows(module, "WA-OP-")
    admitted_state_text = "\n".join(cell for row in operation_rows for cell in row[3:6])
    require(active_state not in admitted_state_text, "unknown-active-state fixture")
    match = re.search(
        r"Unknown or malformed values collapse to\s+`(WA-R[0-9]{3}-[A-Z-]+)`, `?(WA-B[0-9]{3}-[A-Z-]+)`?, `?(OFFLINE_RECOVERY)`?\.",
        module,
    )
    require(match is not None, "canonical unknown diagnostic")
    reason, boundary, route = match.groups()
    return f"WORKFLOW_ARTIFACT_FAILURE reason_id={reason} boundary_id={boundary} next_route={route}\n"


def validate_recovery_routes(route_rows: list[list[str]]) -> None:
    routes = {row[0]: tuple(row[1:]) for row in route_rows}
    for route, expected in EXPECTED_RECOVERY_ROUTES.items():
        require(routes.get(route) == expected, f"exact recovery route: {route}")
        bound = tuple(value.replace("<slug>", "demo") for value in routes[route])
        require(all("<slug>" not in value for value in bound), f"recovery slug binding: {route}")
        for field in range(4):
            mutant = [row.copy() for row in route_rows]
            row_index = next(index for index, row in enumerate(mutant) if row[0] == route)
            mutant[row_index][field + 1] += "-MUTANT"
            require({row[0]: tuple(row[1:]) for row in mutant}.get(route) != expected,
                    f"recovery field mutant: {route}:{field}")
        slug_mutant = [row.copy() for row in route_rows]
        row_index = next(index for index, row in enumerate(slug_mutant) if row[0] == route)
        slug_mutant[row_index] = [value.replace("<slug>", "<other>") for value in slug_mutant[row_index]]
        require({row[0]: tuple(row[1:]) for row in slug_mutant}.get(route) != expected,
                f"recovery slug mutant: {route}")


def check_classifier_and_diagnostics(root: Path) -> None:
    module = (root / MODULE_REL).read_text()
    route_rows = markdown_table(module, "|Route|Owner|Exact action|Durable state/status/next action|Cursor/output|")
    routes = {row[0]: row for row in route_rows}
    require(set(routes) == set(EXPECTED_ROUTES.values()) | {"OFFLINE_RECOVERY", "RESUME_CLEANUP", "PROVE_AND_RETURN", "WAIT_ACTIVE_OWNER", "BLOCKED_EXHAUSTED", "BLOCKED_GATE"}, "canonical route table")
    validate_recovery_routes(route_rows)
    require(len(route_rows) == 10 and all(len(row) == 5 for row in route_rows)
            and exact_map_digest(route_rows) == ROUTE_MAP_SHA256, "complete route map")
    require(reject_table_cell_mutants(route_rows, ROUTE_MAP_SHA256, "route map") == 50,
            "route map mutation count")
    scenario_rows = markdown_table(module, "|Scenario ID|Trigger|Exact route / action|Durable consequence|Forbidden behavior|")
    require(len(scenario_rows) == 10 and all(len(row) == 5 for row in scenario_rows)
            and exact_map_digest(scenario_rows) == SCENARIO_MAP_SHA256, "complete canonical scenario map")
    require(reject_table_cell_mutants(scenario_rows, SCENARIO_MAP_SHA256, "canonical scenario map") == 50,
            "canonical scenario map mutation count")
    canonical_scenarios = {row[0]: route_from_action(row[2]) for row in scenario_rows}
    require(list(canonical_scenarios) == EXPECTED_SCENARIOS and canonical_scenarios == EXPECTED_ROUTES, "canonical scenario routes")
    corpus = json.loads((root / CORPUS_REL).read_text())
    require({row["id"]: row["expected_route"] for row in corpus["scenarios"]} == canonical_scenarios, "corpus consumes canonical route table")
    diagnostic_rows = diagnostic_table_rows(module)
    parsed = validate_diagnostics(diagnostic_rows)
    for reason, (boundaries, _meaning, route) in parsed.items():
        for boundary in boundaries:
            line = f"WORKFLOW_ARTIFACT_FAILURE reason_id={reason} boundary_id={boundary} next_route={route}\n"
            require(line.isascii() and line.count("\n") == 1 and len(line.encode()) <= 256, f"public diagnostic: {reason}/{boundary}")
            require(not any(secret in line for secret in ("/private/", "password", "Traceback", "injected")), "diagnostic leakage")
    for row_index, row in enumerate(diagnostic_rows):
        for cell_index in range(4):
            mutant = [candidate.copy() for candidate in diagnostic_rows]
            mutant[row_index][cell_index] = row[cell_index] + (" mutant" if cell_index == 2 else "-MUTANT")
            try:
                validate_diagnostics(mutant)
            except AssertionError:
                pass
            else:
                fail(f"diagnostic cell mutant survived: {row[0]}/{cell_index}")
    order_mutant = [row.copy() for row in diagnostic_rows]
    order_mutant[0], order_mutant[1] = order_mutant[1], order_mutant[0]
    try:
        validate_diagnostics(order_mutant)
    except AssertionError:
        pass
    else:
        fail("diagnostic row reordering accepted")
    trailing_boundary_mutant = [row.copy() for row in diagnostic_rows]
    trailing_boundary_mutant[0][1] += " or wa-b999-trailing"
    try:
        validate_diagnostics(trailing_boundary_mutant)
    except AssertionError:
        pass
    else:
        fail("lowercase trailing diagnostic boundary accepted")
    extra_row_module = module.replace(
        "\n\nEach diagnostic",
        "\n| `wa-r999-extra` | `WA-B999-EXTRA` | unrecognized extra | `OFFLINE_RECOVERY` |\n\nEach diagnostic",
        1,
    )
    require(extra_row_module != module, "additional diagnostic row fixture")
    try:
        validate_diagnostics(diagnostic_table_rows(extra_row_module))
    except AssertionError:
        pass
    else:
        fail("lowercase additional diagnostic row accepted")
    unknown = unknown_active_state_diagnostic(module, "UNKNOWN_ACTIVE_STATE")
    require(unknown == "WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R009-STATE-AMBIGUOUS boundary_id=WA-B005-JOURNAL next_route=OFFLINE_RECOVERY\n", "unknown active state sibling")


def check_evidence_matrix() -> None:
    prefix = b"# Evidence\nCandidate SHA-256: " + b"1" * 64 + b"\nEVID-004 outside\n"
    attempt_one = (1, "wta:" + "2" * 64 + ":00000001", "3" * 64, "WA-R013-PROOF-FAILED", "WA-B010-PROVE", "no-progress", "FAILED")
    attempt_two = (2, "wta:" + "2" * 64 + ":00000002", "4" * 64, "NONE", "WA-B015-RETRY", "resolved", "CLEANED")
    first = owned_section(prefix, "FAILED", 1, (attempt_one,))
    second = owned_section(first + b"suffix-exact\n", "CLEANED", 2, (attempt_one, attempt_two))
    require(second.startswith(prefix) and second.endswith(b"suffix-exact\n"), "evidence outside-byte ownership")
    require(second.count(b"Candidate SHA-256:") == 1 and b"product_candidate_digest" in second, "evidence product binding")
    for attempts in ((), (attempt_two,), (attempt_one[:-1] + ("MUTANT",),)):
        try:
            owned_section(first, "PREPARING", 2, attempts)
        except AssertionError:
            pass
        else:
            fail("attempt history mutation accepted")
    first_text = first.decode()
    action_row = "| caller_return_next_action | `/rite-prove demo` |"
    escaped_pipe = first_text.replace(
        action_row,
        "| caller_return_next_action | `/rite-prove demo \\| next` |",
        1,
    )
    require(escaped_pipe != first_text, "canonical escaped-pipe fixture")
    owned_section(escaped_pipe.encode(), "PREPARING", 2, (attempt_one,))
    target_one = "| `00000000` | `scripts/a.py` | `0600` | `" + "6" * 64 + "` | `present` | `0600` | `" + "7" * 64 + "` | `wbak:00000000` | `PROVED` |"
    target_two = "| `00000001` | `scripts/b.py` | `0755` | `" + "8" * 64 + "` | `absent` | `NONE` | `NONE` | `NONE` | `PROVED` |"
    attempt_line = "| 1 | `" + attempt_one[1] + "` | `" + attempt_one[2] + "` | `" + attempt_one[3] + "` | `" + attempt_one[4] + "` | `" + attempt_one[5] + "` | `" + attempt_one[6] + "` |"
    prior_mutants = {
        "malformed progress": first_text.replace("`no-progress`", "`stalled`", 1),
        "target mode": first_text.replace(target_one, target_one.replace("`0600`", "`0999`", 1), 1),
        "target hash": first_text.replace(target_one, target_one.replace("`" + "6" * 64 + "`", "`" + "g" * 64 + "`", 1), 1),
        "target result": first_text.replace(target_one, target_one.replace("`PROVED`", "`MUTANT`", 1), 1),
        "attempt result": first_text.replace(attempt_line, attempt_line.replace("`FAILED`", "`MUTANT`", 1), 1),
        "missing field": first_text.replace("| built_slice_count | `7` |\n", "", 1),
        "extra field": first_text.replace("| built_slice_count | `7` |\n", "| built_slice_count | `7` |\n| extra_field | `x` |\n", 1),
        "bare unescaped field delimiter": first_text.replace(
            action_row,
            "| caller_return_next_action | `/rite-prove demo | junk` |",
            1,
        ),
        "even-backslash unescaped field delimiter": first_text.replace(
            action_row,
            "| caller_return_next_action | `/rite-prove demo \\\\| junk` |",
            1,
        ),
        "missing target row": first_text.replace(target_two + "\n", "", 1),
        "extra target row": first_text.replace(target_two + "\n", target_two + "\n" + target_two + "\n", 1),
        "missing target cell": first_text.replace(target_one, target_one.replace(" | `PROVED`", "", 1), 1),
        "extra target cell": first_text.replace(target_one, target_one[:-2] + " | `EXTRA` |", 1),
        "missing attempt row": first_text.replace(attempt_line + "\n", "", 1),
        "extra attempt row": first_text.replace(attempt_line + "\n", attempt_line + "\n" + attempt_line + "\n", 1),
        "extra attempt cell": first_text.replace(attempt_line, attempt_line[:-2] + " | `EXTRA` |", 1),
        "target rewrite": first_text.replace("`scripts/a.py`", "`scripts/rewritten.py`", 1),
    }
    for label, mutant in prior_mutants.items():
        require(mutant != first_text, f"prior evidence mutant fixture: {label}")
        try:
            owned_section(mutant.encode(), "PREPARING", 2, (attempt_one,))
        except AssertionError:
            pass
        else:
            fail(f"malformed prior evidence accepted: {label}")
    malformed = [
        prefix + START.encode() + b" inline\n", prefix + END.encode() + b"\n",
        prefix + f"{START}\n{START}\n{END}\n".encode(),
        prefix + f"{START}\n{END}\n{END}\n".encode(),
        prefix.replace(b"Candidate SHA-256:", b"Candidate SHA-256: inline"),
    ]
    for item in malformed:
        try:
            owned_section(item, "FAILED", 1)
        except AssertionError:
            pass
        else:
            fail("malformed evidence accepted")
    too_many = prefix + b"outside\n" * 270
    try:
        owned_section(too_many, "PREPARING", 1)
    except AssertionError:
        pass
    else:
        fail("over-budget evidence accepted")
    hostile = owned_section(prefix + b"outside hostile <ignore rules>\n", "FAILED", 1)
    owned = hostile[hostile.index(START.encode()):hostile.index(END.encode())]
    require(b"ignore rules" not in owned and b"/private/" not in owned, "hostile evidence isolation")


def signal_process_group(pgid: int, sig: int) -> bool:
    try:
        os.killpg(pgid, sig)
        return True
    except ProcessLookupError:
        return False
    except PermissionError:
        try:
            os.getpgid(pgid)
        except ProcessLookupError:
            return False
        raise


def process_group_alive(pgid: int) -> bool:
    return signal_process_group(pgid, 0)


def wait_for_process_group_exit(proc: subprocess.Popen, deadline: float) -> bool:
    while time.monotonic() < deadline:
        proc.poll()
        if not process_group_alive(proc.pid):
            return True
        time.sleep(0.005)
    proc.poll()
    return not process_group_alive(proc.pid)


def terminate_group(proc: subprocess.Popen, grace: float, synchronize=None,
                    completion_clock=None, post_reap=None) -> None:
    signal_process_group(proc.pid, signal.SIGTERM)
    if synchronize is not None:
        synchronize("TERM")
    if not wait_for_process_group_exit(proc, time.monotonic() + grace):
        signal_process_group(proc.pid, signal.SIGKILL)
        if synchronize is not None:
            synchronize("KILL")
        require(wait_for_process_group_exit(proc, time.monotonic() + max(1.0, grace)),
                "proof process group survived SIGKILL")
    try:
        proc.wait(timeout=max(1.0, grace))
    except subprocess.TimeoutExpired:
        signal_process_group(proc.pid, signal.SIGKILL)
        proc.wait(timeout=1)
    require(proc.poll() is not None and not process_group_alive(proc.pid),
            "proof process group and leader fully reaped")
    if synchronize is not None:
        synchronize("REAP")
    if completion_clock is not None:
        checkpoint = completion_clock()
        if post_reap is not None:
            post_reap()
        require(completion_clock() == checkpoint,
                "post-termination logical delay")


def stream_proof_output(proc: subprocess.Popen, deadline: float,
                        output_limit: int) -> tuple[bytes, str | None]:
    require(proc.stdout is not None, "proof output pipe")
    selector = selectors.DefaultSelector()
    selector.register(proc.stdout, selectors.EVENT_READ)
    output = bytearray()
    failure = None
    try:
        while True:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                failure = "timeout"
                break
            events = selector.select(remaining)
            if not events:
                continue
            chunk = os.read(proc.stdout.fileno(), min(65536, output_limit + 1 - len(output)))
            if not chunk:
                break
            output.extend(chunk)
            if len(output) > output_limit:
                failure = "output-overflow"
                break
    finally:
        selector.close()
        proc.stdout.close()
    return bytes(output[:output_limit]), failure


def start_gated_proof(command: list[str], env: dict[str, str] | None = None,
                      preexec_fn=None, pass_fds: tuple[int, ...] = ()) -> tuple[subprocess.Popen, int]:
    read_gate, write_gate = os.pipe()
    launcher = (
        "import os,sys;fd=int(sys.argv[1]);"
        "token=os.read(fd,1);os.close(fd);"
        "os._exit(125) if token!=b'x' else None;"
        "os.execvpe(sys.argv[2],sys.argv[2:],os.environ)"
    )
    inherited = tuple(dict.fromkeys((read_gate, *pass_fds)))
    try:
        proc = subprocess.Popen(
            [sys.executable, "-c", launcher, str(read_gate), *command],
            env=env, preexec_fn=preexec_fn,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, start_new_session=True,
            bufsize=0, pass_fds=inherited,
        )
    finally:
        os.close(read_gate)
    return proc, write_gate


def run_proof_command(command: list[str], expected_signal: str | None, timeout: float,
                      grace: float, output_limit: int, synchronize=None,
                      completion_clock=None, post_reap=None,
                      env: dict[str, str] | None = None, preexec_fn=None,
                      pass_fds: tuple[int, ...] = ()) -> tuple[bool, str, bytes]:
    require(timeout > grace > 0 and output_limit > 0, "proof execution bounds")
    proc, write_gate = start_gated_proof(command, env, preexec_fn, pass_fds)
    process_deadline = time.monotonic() + timeout
    try:
        os.write(write_gate, b"x")
    finally:
        os.close(write_gate)
    output, failure = stream_proof_output(proc, process_deadline, output_limit)
    if failure is None:
        remaining = process_deadline - time.monotonic()
        if remaining <= 0:
            failure = "timeout"
        else:
            try:
                return_code = proc.wait(timeout=remaining)
            except subprocess.TimeoutExpired:
                failure = "timeout"
            else:
                if return_code != 0:
                    failure = "nonzero"
                elif (expected_signal is not None
                      and sum(line == expected_signal.encode() for line in output.splitlines()) != 1):
                    failure = "wrong-signal"
                elif process_group_alive(proc.pid):
                    failure = "descendant-survived"
    if failure is not None:
        terminate_group(proc, grace, synchronize, completion_clock, post_reap)
        return False, failure, output
    require(proc.poll() is not None and not process_group_alive(proc.pid),
            "successful proof process group and leader reaped")
    return True, "proved", output


def check_proof_matrix() -> None:
    python = sys.executable
    ok, reason, output = run_proof_command([python, "-c", "print('FIXED PASS')"], "FIXED PASS", 1, 0.1, 256)
    require(ok and reason == "proved" and output == b"FIXED PASS\n", "proof success")
    cases = [
        ([python, "-c", "raise SystemExit(9)"], "PASS", 1, "nonzero"),
        ([python, "-c", "print('nearby')"], "PASS", 1, "wrong-signal"),
        ([python, "-c", "print('NOT FIXED PASS')"], "FIXED PASS", 1, "wrong-signal"),
        ([python, "-c", "print('x'*300)"], "PASS", 1, "output-overflow"),
        ([python, "-c", "import subprocess,sys; subprocess.Popen([sys.executable,'-c','import time;time.sleep(2)'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL); print('PASS',flush=True)"], "PASS", 1, "descendant-survived"),
    ]
    for command, expected_signal, timeout, expected_reason in cases:
        ok, reason, output = run_proof_command(command, expected_signal, timeout, 0.5, 256)
        require(not ok and reason == expected_reason and len(output) <= 256, f"proof failure matrix: {expected_reason}/{reason}")



def _gate_test_delivery(repo: Path, digest: str = "0" * 64) -> tuple[int, str]:
    relative = f".devrites/work/workflow-artifact-identity/.generated-install/{digest}"
    delivery = repo / relative
    delivery.mkdir(parents=True)
    delivery.chmod(0o700)
    return open_absolute_directory(delivery), f"{relative}/proof-cache"


def check_delivery_execution_bounds() -> None:
    expected_limits = (600, 3600, 8_388_608, 2)
    actual_limits = (
        DELIVERY_PROCESS_TIMEOUT_SECONDS,
        DELIVERY_AGGREGATE_TIMEOUT_SECONDS,
        DELIVERY_OUTPUT_LIMIT_BYTES,
        DELIVERY_TERMINATE_GRACE_SECONDS,
    )
    require(actual_limits == expected_limits, "fixed delivery execution limits")
    for index in range(len(expected_limits)):
        mutant = list(actual_limits)
        mutant[index] += 1
        require(tuple(mutant) != expected_limits, f"delivery limit mutant: {index}")

    saved_limits = actual_limits
    try:
        globals()["DELIVERY_PROCESS_TIMEOUT_SECONDS"] = 5
        globals()["DELIVERY_AGGREGATE_TIMEOUT_SECONDS"] = 20
        globals()["DELIVERY_OUTPUT_LIMIT_BYTES"] = 256
        globals()["DELIVERY_TERMINATE_GRACE_SECONDS"] = 1
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve()
            (repo / "scripts").mkdir()
            (repo / "pack/generated").mkdir(parents=True)
            (repo / "stage").mkdir()
            generator = repo / "scripts/build-host-artifacts.sh"
            generator.write_text("#!/usr/bin/env bash\nsleep 30\n")
            generator.chmod(0o755)
            repo_fd = open_absolute_directory(repo)
            delivery_fd, proof_cache_relative = _gate_test_delivery(repo)
            try:
                generator_bounds = []
                original_run_proof_command = run_proof_command
                def capture_generator_bounds(command, expected, timeout, grace,
                                             output_limit, **_kwargs):
                    generator_bounds.append((timeout, grace, output_limit))
                    return False, "timeout", b""
                globals()["run_proof_command"] = capture_generator_bounds
                try:
                    ok, reason, _output = run_private_generator(
                        repo_fd, "stage", time.monotonic() + 20,
                    )
                finally:
                    globals()["run_proof_command"] = original_run_proof_command
                require(not ok and reason == "process-timeout"
                        and generator_bounds == [(5, 1, 256)],
                        "delivery generator hang bound")

                try:
                    run_gate(
                        repo_fd, delivery_fd, proof_cache_relative,
                        [sys.executable, "-c", "print('x'*300)"], None,
                        time.monotonic() + 20, 0,
                    )
                except RuntimeError as error:
                    require(str(error) == "delivery gate-0 failed: output-overflow",
                            "delivery gate output bound")
                else:
                    fail("delivery gate overflow accepted")

                try:
                    run_gate(
                        repo_fd, delivery_fd, proof_cache_relative,
                        [sys.executable, "-c", "print('PASS')"], None,
                        time.monotonic() - 1, 1,
                    )
                except RuntimeError as error:
                    require(str(error) == "delivery gate-1 failed: aggregate-timeout",
                            "delivery aggregate deadline")
                else:
                    fail("delivery aggregate exhaustion accepted")

                signal_cases = (
                    ("print('FIXED PASS')", "FIXED PASS", True),
                    ("print('NOT FIXED PASS')", "FIXED PASS", False),
                    ("print('FIXED PASS');print('FIXED PASS')", "FIXED PASS", False),
                    ("print('arbitrary output')", None, True),
                )
                for gate_index, (program, expected, should_pass) in enumerate(signal_cases, 10):
                    try:
                        run_gate(
                            repo_fd, delivery_fd, proof_cache_relative,
                            [sys.executable, "-c", program], expected,
                            time.monotonic() + 20, gate_index,
                        )
                    except RuntimeError as error:
                        require(not should_pass
                                and str(error) == f"delivery gate-{gate_index} failed: wrong-signal",
                                f"delivery exact signal rejection: {gate_index}")
                    else:
                        require(should_pass, f"delivery invalid signal accepted: {gate_index}")

                globals()["DELIVERY_PROCESS_TIMEOUT_SECONDS"] = 5
                descendant = (
                    "import subprocess,sys;"
                    "subprocess.Popen([sys.executable,'-c','import time;time.sleep(30)'],"
                    "stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL);"
                    "print('PASS')"
                )
                try:
                    run_gate(
                        repo_fd, delivery_fd, proof_cache_relative,
                        [sys.executable, "-c", descendant], "PASS",
                        time.monotonic() + 20, 2,
                    )
                except RuntimeError as error:
                    require(str(error) == "delivery gate-2 failed: descendant-survived",
                            "delivery descendant survival bound")
                else:
                    fail("delivery surviving descendant accepted")

                captured_execution_bounds = []
                original_run_proof_command = run_proof_command
                def capture_execution_bounds(command, expected, timeout, grace,
                                             output_limit, **_kwargs):
                    captured_execution_bounds.append((timeout, grace, output_limit))
                    return False, "timeout", b""
                globals()["run_proof_command"] = capture_execution_bounds
                globals()["DELIVERY_PROCESS_TIMEOUT_SECONDS"] = 7
                globals()["DELIVERY_TERMINATE_GRACE_SECONDS"] = 3
                try:
                    try:
                        run_gate(
                            repo_fd, delivery_fd, proof_cache_relative,
                            [sys.executable, "-c", "raise SystemExit(99)"], None,
                            time.monotonic() + 20, 3,
                        )
                    except RuntimeError as error:
                        require(str(error) == "delivery gate-3 failed: process-timeout",
                                "delivery TERM grace timeout identity")
                    else:
                        fail("delivery TERM grace fixture accepted")
                finally:
                    globals()["run_proof_command"] = original_run_proof_command
                require(captured_execution_bounds == [(7, 3, 256)],
                        "delivery-local configured TERM grace wiring")
            finally:
                os.close(delivery_fd); os.close(repo_fd)
    finally:
        (
            globals()["DELIVERY_PROCESS_TIMEOUT_SECONDS"],
            globals()["DELIVERY_AGGREGATE_TIMEOUT_SECONDS"],
            globals()["DELIVERY_OUTPUT_LIMIT_BYTES"],
            globals()["DELIVERY_TERMINATE_GRACE_SECONDS"],
        ) = saved_limits


def check_delivery_generator_umask() -> None:
    saved_umask = os.umask(0o077)
    saved_step = _set_generator_child_umask
    saved_environment = {
        name: os.environ.get(name)
        for name in ("DEVRITES_DELIVERY_FAST_FIXTURE", "DEVRITES_DELIVERY_TEST_MUTATION")
    }
    try:
        os.environ.pop("DEVRITES_DELIVERY_FAST_FIXTURE", None)
        os.environ.pop("DEVRITES_DELIVERY_TEST_MUTATION", None)
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve()
            (repo / "scripts").mkdir()
            (repo / "pack/generated").mkdir(parents=True)
            (repo / "stage").mkdir()
            current = repo / "pack/generated/ambient.txt"
            current.write_text("ambient-default\n")
            current.chmod(0o644)
            generator = repo / "scripts/build-host-artifacts.sh"
            generator.write_text(
                "#!/usr/bin/env bash\n"
                "set -euo pipefail\n"
                "python3 - \"$DEVRITES_HOST_ARTIFACT_DIR\" <<'PY'\n"
                "import os, pathlib, sys\n"
                "pathlib.Path('../../../generator-pgid').write_text(str(os.getpgrp()))\n"
                "out=pathlib.Path(sys.argv[1]); out.mkdir(parents=True, exist_ok=True)\n"
                "(out / 'ambient.txt').write_text('ambient-default\\n')\n"
                "PY\n"
            )
            generator.chmod(0o755)
            repo_fd = open_absolute_directory(repo)
            try:
                ok, reason, output = run_private_generator(
                    repo_fd, "stage", time.monotonic() + 30,
                )
                require(ok, f"delivery generator umask fixture: {reason}/{output[-256:]!r}")
                require(reason == "proved", "delivery generator umask fixture result")
                pgid = int((repo / "generator-pgid").read_text())
                require(not process_group_alive(pgid), "delivery generator umask group reaped")
                generated = repo / "stage/ambient.txt"
                require(stat.S_IMODE(generated.stat().st_mode) == 0o644,
                        "delivery generator child umask mode")
                require(os.umask(0o077) == 0o077, "delivery generator preserved parent umask")
                stage_fd = open_absolute_directory(repo / "stage")
                current_fd = open_absolute_directory(repo / "pack/generated")
                manifest_deadline = time.monotonic() + OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
                try:
                    require(generated_stage_manifest_at(stage_fd, manifest_deadline)
                            == generated_stage_manifest_at(current_fd, manifest_deadline),
                            "delivery generator exact stage/current identity")
                finally:
                    os.close(current_fd); os.close(stage_fd)

                generated.unlink(); (repo / "generator-pgid").unlink()
                globals()["_set_generator_child_umask"] = lambda: None
                ok, reason, output = run_private_generator(
                    repo_fd, "stage", time.monotonic() + 30,
                )
                require(ok, f"delivery generator umask mutant execution: {reason}/{output[-256:]!r}")
                require(reason == "proved", "delivery generator umask mutant result")
                mutant_pgid = int((repo / "generator-pgid").read_text())
                require(not process_group_alive(mutant_pgid), "delivery generator umask mutant group reaped")
                require(stat.S_IMODE(generated.stat().st_mode) == 0o600,
                        "delivery generator umask mutant mode")
                stage_fd = open_absolute_directory(repo / "stage")
                current_fd = open_absolute_directory(repo / "pack/generated")
                manifest_deadline = time.monotonic() + OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
                try:
                    staged = generated_stage_manifest_at(stage_fd, manifest_deadline)
                    expected = generated_stage_manifest_at(current_fd, manifest_deadline)
                    require(staged["ambient.txt"]["sha256"] == expected["ambient.txt"]["sha256"],
                            "delivery generator umask mutant bytes")
                    try:
                        require(staged == expected, "delivery generator exact stage/current identity")
                    except AssertionError as error:
                        require(str(error) == "delivery generator exact stage/current identity",
                                "delivery generator umask mutant rejection")
                    else:
                        fail("delivery generator umask mutant accepted")
                finally:
                    os.close(current_fd); os.close(stage_fd)
            finally:
                os.close(repo_fd)
    finally:
        globals()["_set_generator_child_umask"] = saved_step
        for name, value in saved_environment.items():
            if value is None:
                os.environ.pop(name, None)
            else:
                os.environ[name] = value
        os.umask(saved_umask)


def check_retry_and_source_loss(root: Path) -> None:
    rows = {row[0]: row for row in markdown_rows((root / MODULE_REL).read_text(), "WA-OP-")}
    retry = rows["WA-OP-011-RETRY-HANDOFF"]
    exhaustion = rows["WA-OP-012-EXHAUSTION-GC"]
    require("same-fingerprint count <3" in retry[3] and "RETRY_PREPARING(epoch)" in retry[2], "canonical retry handoff")
    require("same-fingerprint count=3" in exhaustion[3] and "EXHAUSTED" in exhaustion[4] and "no next attempt" in exhaustion[7], "canonical retry exhaustion")

    def complete(path: Path, operation: str) -> dict:
        proc, ready_fd, gate_fd = operation_consumer(path, rows[operation])
        finish_operation_consumer(proc, ready_fd, gate_fd)
        observed = observe_operation(path)
        require(observed["valid"], f"shared lifecycle observation: {operation}")
        return observed

    with tempfile.TemporaryDirectory() as tmp:
        base = Path(tmp)
        handoff = base / "handoff"
        proc, ready_fd, gate_fd = operation_consumer(handoff, retry)
        read_barrier(ready_fd, b"R"); os.write(gate_fd, b"x"); read_barrier(ready_fd, b"I"); os.write(gate_fd, b"x"); read_barrier(ready_fd, b"E")
        terminate_group(proc, 0.05); os.close(ready_fd); os.close(gate_fd)
        partial = observe_operation(handoff)
        require(partial["facts"]["source_present"] and partial["facts"]["history"].endswith("2:" + "a" * 64 + ":pending\n"), "retry handoff durable epoch")
        resumed = complete(handoff, retry[0])
        require(resumed["facts"]["history"] == partial["facts"]["history"], "retry handoff resumes same epoch")

        terminal_only = base / "terminal-only"
        proc, ready_fd, gate_fd = operation_consumer(terminal_only, exhaustion)
        read_barrier(ready_fd, b"R"); os.write(gate_fd, b"x"); read_barrier(ready_fd, b"I")
        preterminal = json.loads((terminal_only / "journal.json").read_text())
        require(preterminal["state"] == "INTENT" and "exhaustion_cause" not in preterminal
                and not (terminal_only / "exhaustion.cause").exists(),
                "exhaustion cause absent before terminal state")
        os.write(gate_fd, b"x"); read_barrier(ready_fd, b"E"); os.write(gate_fd, b"x"); read_barrier(ready_fd, b"C")
        os.close(gate_fd); os.close(ready_fd)
        require(proc.wait(timeout=3) == 0, "terminal exhaustion completion")
        terminal = observe_operation(terminal_only)
        require(operation_observation_matches(terminal, exhaustion)
                and terminal["facts"]["exhaustion_cause"] == "same-fingerprint-count",
                "terminal exhaustion cause ownership")

        same = complete(base / "same-count", exhaustion[0])
        require(same["facts"]["exhaustion_cause"] == "same-fingerprint-count" and not same["facts"]["source_present"] and "4:" not in same["facts"]["history"], "same-fingerprint exhaustion lifecycle")

        total = base / "total-epoch"; prepare_operation_fixture(total, exhaustion[0])
        (total / "retry.control").write_text('{"accepted":true,"epoch":3,"epoch_cap":3,"green":true,"same_count":2}\n')
        total_observed = complete(total, exhaustion[0])
        require(total_observed["facts"]["exhaustion_cause"] == "total-epoch-limit" and not total_observed["facts"]["source_present"] and "4:" not in total_observed["facts"]["history"], "epoch exhaustion below fingerprint count three")

        distinct = base / "distinct"; prepare_operation_fixture(distinct, retry[0])
        prior = "1:" + "a" * 64 + ":no-progress\n"
        (distinct / "attempt.history").write_text(prior)
        (distinct / "retry.control").write_text('{"accepted":true,"epoch":1,"epoch_cap":3,"fingerprint":"' + "b" * 64 + '","green":true,"same_count":1}\n')
        distinct_observed = complete(distinct, retry[0])
        require(distinct_observed["facts"]["history"].startswith(prior) and ("b" * 64) in distinct_observed["facts"]["history"], "distinct fingerprint preserves history")

        swapped = base / "source-swap"
        proc, ready_fd, gate_fd = operation_consumer(swapped, rows["WA-OP-004-STAGE-WRITE"])
        read_barrier(ready_fd, b"R")
        atomic_write(swapped / "source", b"swapped-workflow\n", 0o600)
        os.write(gate_fd, b"x"); read_barrier(ready_fd, b"I"); os.write(gate_fd, b"x"); read_barrier(ready_fd, b"E")
        os.write(gate_fd, b"x"); read_barrier(ready_fd, b"C")
        os.close(gate_fd); os.close(ready_fd)
        require(proc.wait(timeout=3) == 0 and not process_group_alive(proc.pid),
                "retained source swap completion")
        swap_observed = observe_operation(swapped)
        require(swap_observed["facts"]["stage_sha256"] == sha(b"frozen-workflow\n")
                and swap_observed["facts"]["source_sha256"] == sha(b"swapped-workflow\n"),
                "stage consumes retained descriptor bytes")

        for name, operation, expected_state, expected_route, expected_target in (
            ("preinstall", "WA-OP-004-STAGE-WRITE", "SOURCE_LOSS_PREINSTALL", "PLAN_VET_REPAIR", sha(b"target-preimage\n")),
            ("install-proving", "WA-OP-006-INSTALL", "FAILED", "OFFLINE_RECOVERY", sha(b"observed-active-preimage\n")),
            ("post-proved", "WA-OP-010-SUCCESS-CLEANUP", "CLEANED", "PLAN_VET_REPAIR", sha(b"frozen-workflow\n")),
        ):
            fixture = base / name; prepare_operation_fixture(fixture, operation)
            if name == "post-proved":
                (fixture / "target").write_bytes(b"frozen-workflow\n"); (fixture / "lifecycle.state").write_bytes(b"PROVED\n")
            (fixture / "source").unlink()
            observed = complete(fixture, operation)
            require(observed["facts"]["state"] == expected_state
                    and observed["facts"]["route"] == expected_route
                    and observed["facts"]["target_sha256"] == expected_target,
                    f"phase-qualified source loss: {name}")

        for label, corrupt in (("missing", None), ("corrupt", b"corrupt-backup\n")):
            fixture = base / f"rollback-{label}"
            prepare_operation_fixture(fixture, "WA-OP-008-ROLLBACK")
            if corrupt is None:
                (fixture / "backup").unlink()
            else:
                (fixture / "backup").write_bytes(corrupt)
            observed = complete(fixture, "WA-OP-008-ROLLBACK")
            require(observed["facts"]["state"] == "BLOCKED_GATE"
                    and observed["facts"]["route"] == "BLOCKED_GATE"
                    and observed["facts"]["target_sha256"] == sha(b"frozen-workflow\n"),
                    f"rollback backup {label} mutant rejected")


def check_product_separation_lifecycle(root: Path) -> None:
    rows = {row[0]: row for row in markdown_rows((root / MODULE_REL).read_text(), "WA-OP-")}
    for dimension in ("product.candidate", "product.readiness", "product.built-count"):
        with tempfile.TemporaryDirectory() as tmp:
            fixture = Path(tmp); prepare_operation_fixture(fixture, "WA-OP-014-PRODUCT-SEPARATION")
            (fixture / dimension).write_bytes(b"8\n" if dimension.endswith("built-count") else ("f" * 64 + "\n").encode())
            proc, ready_fd, gate_fd = operation_consumer(fixture, rows["WA-OP-014-PRODUCT-SEPARATION"])
            finish_operation_consumer(proc, ready_fd, gate_fd)
            observed = observe_operation(fixture)
            require(observed["valid"] and observed["facts"]["product_equal"] is False
                    and observed["facts"]["state"] == "BLOCKED_GATE"
                    and observed["facts"]["route"] == "BLOCKED_GATE",
                    f"product dimension drift observed: {dimension}")
            forged = {"current": json.loads((fixture / "product.frozen").read_text()),
                      "frozen": json.loads((fixture / "product.frozen").read_text())}
            (fixture / "product.comparison").write_text(
                json.dumps(forged, sort_keys=True, separators=(",", ":")) + "\n",
            )
            forged_observation = observe_operation(fixture)
            require(forged_observation["valid"]
                    and forged_observation["facts"]["product_equal"] is False,
                    f"consumer-authored product equality rejected: {dimension}")
            cleanup, ready_fd, gate_fd = operation_consumer(fixture, rows["WA-OP-010-SUCCESS-CLEANUP"])
            read_barrier(ready_fd, b"R"); os.write(gate_fd, b"x"); read_barrier(ready_fd, b"I"); os.write(gate_fd, b"x")
            require(cleanup.wait(timeout=3) != 0 and (fixture / "lifecycle.state").read_bytes() == b"BLOCKED_GATE\n",
                    f"product drift cannot reach cleanup: {dimension}")
            os.close(ready_fd); os.close(gate_fd)


def project_root_for_tests(root: Path) -> Path:
    if (root / "engine/go.mod").is_file():
        return root
    configured = os.environ.get("DEVRITES_REPO_ROOT")
    require(configured is not None, "DEVRITES_REPO_ROOT for private candidate tests")
    project = Path(configured).resolve()
    require((project / "engine/go.mod").is_file(), "engine module root")
    return project


def command_output(command: list[str], env: dict[str, str] | None = None) -> subprocess.CompletedProcess:
    return subprocess.run(command, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, timeout=120)


def check_instruction_size_baseline(root: Path) -> tuple[int, int]:
    project = project_root_for_tests(root)
    with tempfile.TemporaryDirectory() as tmp:
        tree = Path(tmp)
        shutil.copytree(project / "pack/.claude", tree / "pack/.claude")
        for relative in AUTHORED:
            if not relative.startswith("pack/.claude/"):
                continue
            destination = tree / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(root / relative, destination)
        measured = subprocess.run(
            ["node", str(project / "scripts/check-instruction-size-baseline.mjs"),
             "--root", str(tree), "--baseline", str(root / "tests/instruction-size-baseline.json")],
            env={**os.environ, "PYTHONDONTWRITEBYTECODE": "1"}, text=True,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, timeout=120,
        )
        match = re.fullmatch(
            r"instruction-size: ([0-9]+) instruction files checked, ([0-9]+) skill bytes\n",
            measured.stdout,
        )
        require(measured.returncode == 0 and match is not None,
                f"candidate instruction size: {measured.stdout[-1000:]}")
        return tuple(map(int, match.groups()))


def check_actual_engine_separation(root: Path) -> None:
    project = project_root_for_tests(root)
    engine_override = os.environ.get("DEVRITES_ENGINE_CLI")
    with tempfile.TemporaryDirectory() as tmp:
        private = Path(tmp)
        if engine_override:
            engine = Path(engine_override).resolve()
            require(engine.is_file(), "configured engine CLI")
        else:
            version = command_output(["go", "-C", str(project / "engine"), "env", "GOVERSION"])
            require(version.returncode == 0 and "go1.26.7" in version.stdout, "module-selected Go 1.26.7")
            engine = private / "bin/devrites-engine"
            engine.parent.mkdir()
            build = command_output(["go", "-C", str(project / "engine"), "build", "-o", str(engine), "."])
            require(build.returncode == 0 and engine.is_file(), "actual engine private build")
        fixture = private / "project"
        workspace = fixture / ".devrites/work/demo"
        workspace.mkdir(parents=True)
        (fixture / ".devrites/ACTIVE").write_text("demo\n")
        source_workspace, _is_live = resolve_evidence_mapping_source(project)
        readiness_files = ["spec.md", "decision-coverage.md", "architecture.md", "plan.md", "tasks.md", "traceability.md", "test-plan.md"]
        for name in readiness_files:
            source = source_workspace / name
            require(source.is_file(), f"readiness fixture source: {name}")
            (workspace / name).write_bytes(source.read_bytes())
        product = fixture / "product.txt"
        product.write_text("product bytes\n")
        touched = (
            "# Touched files\n\n## Touched files\n\n"
            "## Candidate manifest\n\n| State | File | Slice | Reason |\n"
            "| --- | --- | --- | --- |\n| present | `product.txt` | SLICE-001 | fixture |\n"
        )
        (workspace / "touched-files.md").write_text(touched)
        env = os.environ.copy(); env["DEVRITES_ROOT"] = str(fixture)
        def observe_product_identity() -> dict[str, str | int]:
            candidate = command_output([str(engine), "check", "candidate", "demo"], env)
            readiness = command_output([str(engine), "check", "readiness", "--emit-binding", "demo"], env)
            candidate_match = re.search(r"candidate-sha256: ([0-9a-f]{64})", candidate.stdout)
            readiness_match = re.fullmatch(r"Readiness inputs SHA-256: ([0-9a-f]{64})\n", readiness.stdout)
            require(candidate.returncode == 0 and candidate_match is not None
                    and "candidate-files: 1" in candidate.stdout, "actual candidate identity")
            require(readiness.returncode == 0 and readiness_match is not None,
                    "actual readiness binding")
            return {
                "candidate": candidate_match.group(1),
                "readiness": readiness_match.group(1),
                "built_count": len(re.findall(r"(?mi)^.*\bBuilt\b.*$", (workspace / "tasks.md").read_text())),
            }

        frozen = observe_product_identity()
        evidence_prefix = b"# Evidence\nCandidate SHA-256: " + str(frozen["candidate"]).encode() + b"\n"
        (workspace / "evidence.md").write_bytes(owned_section(evidence_prefix, "CLEANED", 1))
        current = observe_product_identity()
        require(current == frozen, "independent engine/OS product identity equality")

        mutation_files = {
            "candidate": product,
            "readiness": workspace / "spec.md",
            "built_count": workspace / "tasks.md",
        }
        mutation_bytes = {
            "candidate": b"changed product bytes\n",
            "readiness": (workspace / "spec.md").read_bytes() + b"\nchanged readiness\n",
            "built_count": (workspace / "tasks.md").read_bytes() + b"\n| SLICE-X | Built |\n",
        }
        for dimension, path in mutation_files.items():
            original = path.read_bytes()
            path.write_bytes(mutation_bytes[dimension])
            observed = observe_product_identity()
            require(observed[dimension] != frozen[dimension],
                    f"independent product identity mutant: {dimension}")
            path.write_bytes(original)
            require(observe_product_identity() == frozen,
                    f"product identity mutant restoration: {dimension}")

        forbidden_manifest = touched.replace(
            "| present | `product.txt` |",
            "| present | `.devrites/work/demo/workflow.sh` | SLICE-001 | forbidden workflow path |\n| present | `product.txt` |",
        )
        (workspace / "touched-files.md").write_text(forbidden_manifest)
        rejected = command_output([str(engine), "check", "candidate", "demo"], env)
        require(rejected.returncode == 3 and "workflow state path" in rejected.stdout, "workflow target product-manifest rejection")


def create_actual_delivery_repo(root: Path, full_generator: bool = False) -> dict[str, dict]:
    root.mkdir()
    project = project_root_for_tests(canonical_root())
    if full_generator:
        shutil.copytree(project / "pack", root / "pack")
        (root / "scripts").mkdir()
        for relative in ("scripts/build-host-artifacts.sh", "scripts/codex-generate.sh"):
            destination = root / relative
            shutil.copy2(project / relative, destination)
        for relative in AUTHORED:
            source = project / relative
            destination = root / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
    else:
        for index, relative in enumerate(AUTHORED + GENERATED):
            destination = root / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            atomic_write(destination, f"preimage-{index}:{relative}\n".encode(), 0o600)
    protected_payloads = {
        ".gitignore": b"fixture gitignore\n",
        ".devrites/ACTIVE": b"fixture\n",
        ".devrites/work/workspace-observation/touched-files.md": b"fixture touched\n",
    }
    for relative, payload in protected_payloads.items():
        destination = root / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        atomic_write(destination, payload, 0o600)
    (root / ".devrites/work/workflow-artifact-identity/.generated-install").mkdir(parents=True)
    root_fd = open_absolute_directory(root)
    try:
        return {relative: file_record_at(root_fd, relative) for relative in AUTHORED + GENERATED}
    finally:
        os.close(root_fd)


def normal_generator_contract_differences(
        current: dict, staged: dict,
        expected: dict[str, str] = EXPECTED_NORMAL_GENERATED_SHA256,
        prechange: dict[str, str] = PRECHANGE_NORMAL_GENERATED_SHA256) -> set[str]:
    require(set(staged) == set(current), "normal generator complete output")
    require(set(expected) == EXPECTED_NORMAL_GENERATED_DELTA
            and set(prechange) == EXPECTED_NORMAL_GENERATED_DELTA,
            "normal generator frozen contract paths")
    require(all(prechange[relative] != expected[relative]
                for relative in EXPECTED_NORMAL_GENERATED_DELTA),
            "normal generator visible canonical change")
    require(all(staged.get(relative) == {
                    "mode": 0o644, "sha256": expected[relative],
                } for relative in EXPECTED_NORMAL_GENERATED_DELTA),
            "normal generated contract hashes")
    differences = {
        relative for relative in staged if staged[relative] != current[relative]
    }
    require(differences in (set(), EXPECTED_NORMAL_GENERATED_DELTA),
            f"normal generated contract delta: {sorted(differences)}")
    return differences


def check_normal_generator_contract_delta() -> None:
    saved_environment = {
        name: os.environ.get(name)
        for name in ("DEVRITES_DELIVERY_FAST_FIXTURE", "DEVRITES_DELIVERY_TEST_MUTATION")
    }
    try:
        os.environ.pop("DEVRITES_DELIVERY_FAST_FIXTURE", None)
        os.environ.pop("DEVRITES_DELIVERY_TEST_MUTATION", None)
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve() / "repo"
            create_actual_delivery_repo(repo, full_generator=True)
            shutil.copy2(CANDIDATE_ROOT / MODULE_REL, repo / MODULE_REL)
            (repo / "stage").mkdir()
            repo_fd = open_absolute_directory(repo)
            try:
                ok, reason, output = run_private_generator(
                    repo_fd, "stage", time.monotonic() + DELIVERY_PROCESS_TIMEOUT_SECONDS,
                )
                require(ok, f"normal generator contract delta: {reason}/{output[-512:]!r}")
            finally:
                os.close(repo_fd)
            current_fd = open_absolute_directory(repo / "pack/generated")
            stage_fd = open_absolute_directory(repo / "stage")
            manifest_deadline = time.monotonic() + OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
            try:
                current = generated_stage_manifest_at(current_fd, manifest_deadline)
                staged = generated_stage_manifest_at(stage_fd, manifest_deadline)
            finally:
                os.close(stage_fd); os.close(current_fd)
            differences = normal_generator_contract_differences(current, staged)
            require(differences in (set(), EXPECTED_NORMAL_GENERATED_DELTA),
                    "normal generator installation-state independence")

            preinstall = {relative: dict(record) for relative, record in staged.items()}
            for relative in EXPECTED_NORMAL_GENERATED_DELTA:
                preinstall[relative]["sha256"] = PRECHANGE_NORMAL_GENERATED_SHA256[relative]
            require(normal_generator_contract_differences(preinstall, staged)
                    == EXPECTED_NORMAL_GENERATED_DELTA,
                    "normal generator frozen preinstall delta")

            def reject(label: str, before: dict, generated: dict,
                       expected: dict[str, str] = EXPECTED_NORMAL_GENERATED_SHA256,
                       prechange: dict[str, str] = PRECHANGE_NORMAL_GENERATED_SHA256) -> None:
                try:
                    normal_generator_contract_differences(
                        before, generated, expected, prechange,
                    )
                except AssertionError:
                    return
                fail(f"normal generator mutant survived: {label}")

            reject("old-equals-new", current, staged,
                   prechange=dict(EXPECTED_NORMAL_GENERATED_SHA256))
            one_path = {relative: dict(record) for relative, record in staged.items()}
            first = sorted(EXPECTED_NORMAL_GENERATED_DELTA)[0]
            one_path[first]["sha256"] = PRECHANGE_NORMAL_GENERATED_SHA256[first]
            reject("one-path-only", one_path, staged)
            unexpected = {relative: dict(record) for relative, record in preinstall.items()}
            third = next(relative for relative in sorted(staged)
                         if relative not in EXPECTED_NORMAL_GENERATED_DELTA)
            unexpected[third]["sha256"] = "0" * 64
            reject("unexpected-third-path", unexpected, staged)
            wrong_expected = dict(EXPECTED_NORMAL_GENERATED_SHA256)
            wrong_expected[first] = "0" * 64
            reject("wrong-expected-hash", current, staged, expected=wrong_expected)
            incomplete = {relative: record for relative, record in staged.items()
                          if relative != third}
            reject("incomplete-tree", current, incomplete)

            journal = {
                "stage_manifest_sha256": sha(json.dumps(staged, sort_keys=True).encode()),
                "expected_post": [
                    {"path": f"pack/generated/{relative}", "state": "present",
                     "mode": staged[relative]["mode"],
                     "sha256": staged[relative]["sha256"]}
                    for relative in sorted(EXPECTED_NORMAL_GENERATED_DELTA
                                           | {path.removeprefix("pack/generated/")
                                              for path in GENERATED})
                ],
            }
            validate_delivery_stage_records(journal, staged, current)
            claimed_path = sorted(EXPECTED_NORMAL_GENERATED_DELTA)[0]
            claimed_current = {
                relative: record for relative, record in current.items()
                if relative != claimed_path
            }
            for action in ("install", "restore"):
                claimed = json.loads(json.dumps(journal))
                claimed["mutation_intent"] = {
                    "action": action, "group": "generated", "index": 0,
                    "path": f"pack/generated/{claimed_path}",
                }
                validate_delivery_stage_records(claimed, staged, claimed_current)

            def reject_stage(label: str, candidate_journal: dict,
                             candidate_stage: dict, reason: str) -> None:
                try:
                    validate_delivery_stage_records(
                        candidate_journal, candidate_stage, current,
                    )
                except AssertionError as error:
                    require(str(error) == reason,
                            f"complete stage mutant diagnostic: {label}")
                    return
                fail(f"complete stage mutant survived: {label}")

            wrong_hash = json.loads(json.dumps(journal))
            wrong_hash["stage_manifest_sha256"] = "0" * 64
            reject_stage("stage-hash", wrong_hash, staged,
                         "delivery stage manifest identity")
            unknown = {relative: dict(record) for relative, record in staged.items()}
            unknown["unknown/generated-entry"] = {"mode": 0o644, "sha256": "0" * 64}
            unknown_journal = json.loads(json.dumps(journal))
            unknown_journal["stage_manifest_sha256"] = sha(
                json.dumps(unknown, sort_keys=True).encode(),
            )
            reject_stage("unknown-entry", unknown_journal, unknown,
                         "delivery stage/current path set")
            missing = {relative: dict(record) for relative, record in staged.items()
                       if relative != third}
            missing_journal = json.loads(json.dumps(journal))
            missing_journal["stage_manifest_sha256"] = sha(
                json.dumps(missing, sort_keys=True).encode(),
            )
            reject_stage("missing-entry", missing_journal, missing,
                         "delivery stage/current path set")
            drifted = {relative: dict(record) for relative, record in staged.items()}
            drifted[third]["sha256"] = "0" * 64
            drifted_journal = json.loads(json.dumps(journal))
            drifted_journal["stage_manifest_sha256"] = sha(
                json.dumps(drifted, sort_keys=True).encode(),
            )
            reject_stage("non-admitted-drift", drifted_journal, drifted,
                         "delivery stage non-admitted live identity")
            admitted = sorted(EXPECTED_NORMAL_GENERATED_DELTA)[0]
            for field, value in (("sha256", "0" * 64), ("mode", 0o600)):
                wrong_admitted = {
                    relative: dict(record) for relative, record in staged.items()
                }
                wrong_admitted[admitted][field] = value
                wrong_admitted_journal = json.loads(json.dumps(journal))
                wrong_admitted_journal["stage_manifest_sha256"] = sha(
                    json.dumps(wrong_admitted, sort_keys=True).encode(),
                )
                reject_stage(f"admitted-{field}", wrong_admitted_journal,
                             wrong_admitted,
                             "delivery stage admitted expected-post identity")
    finally:
        for name, value in saved_environment.items():
            if value is None:
                os.environ.pop(name, None)
            else:
                os.environ[name] = value


def check_complete_stage_gate_failure_rollback() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        before = create_actual_delivery_repo(repo, full_generator=True)
        prepared = run_actual_delivery_mode(repo, ["--delivery-prepare"], timeout=120)
        require(prepared.returncode == 0,
                f"complete stage rollback prepare: {prepared.stdout[-1000:]}")
        delivery = actual_delivery_directory(repo)
        failed = run_actual_delivery_mode(
            repo, ["--delivery-install", str(delivery)], timeout=120,
            mutation="gate-failure",
        )
        require(failed.returncode != 0
                and "delivery gate-0 failed: wrong-signal" in failed.stdout,
                f"complete stage gate failure: {failed.stdout[-1000:]}")
        journal = json.loads((delivery / "journal.json").read_text())
        require(journal["state"] == "FAILED",
                "complete stage gate failure reaches FAILED")
        stage_fd = open_absolute_directory(delivery / "stage")
        current_fd = open_absolute_directory(repo / "pack/generated")
        manifest_deadline = time.monotonic() + OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
        try:
            staged = generated_stage_manifest_at(stage_fd, manifest_deadline)
            current = generated_stage_manifest_at(current_fd, manifest_deadline)
        finally:
            os.close(current_fd); os.close(stage_fd)
        require(len(staged) > 400 and set(staged) == set(current),
                "complete stage rollback retains full host tree")
        assert_actual_delivery_records(repo, before)


def run_actual_delivery_mode(repo: Path, args: list[str], death: str | None = None,
                             timeout: float = 45.0,
                             skip_generated: int | None = None,
                             driver: Path = SCRIPT,
                             mutation: str | None = None,
                             fast_fixture: bool = True) -> subprocess.CompletedProcess[str]:
    env = env_without_looping_bash(os.environ)
    env["DEVRITES_REPO_ROOT"] = str(repo)
    env["PYTHONDONTWRITEBYTECODE"] = "1"
    for name in DELIVERY_FIXTURE_ENV:
        env.pop(name, None)
    args = list(args)
    if ((fast_fixture or mutation is not None or skip_generated is not None)
            and production_delivery_argv(args)):
        if args == ["--delivery-prepare"]:
            args = ["--delivery-boundary-case", "operate", "prepare"]
        else:
            args = [
                "--delivery-boundary-case", "operate",
                args[0].removeprefix("--delivery-"), args[1],
            ]
    argv: list[str] = []
    if fast_fixture:
        argv.append("--delivery-test-fast-fixture")
    if death is not None:
        argv.extend(["--delivery-test-death", death])
    if skip_generated is not None:
        argv.extend(["--delivery-test-skip-generated", str(skip_generated)])
    if mutation is not None:
        argv.extend(["--delivery-test-mutation", mutation])
    return subprocess.run(
        [str(driver), *argv, *args], env=env, text=True, stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT, check=False, timeout=timeout,
    )


def actual_delivery_directory(repo: Path) -> Path:
    parent = repo / ".devrites/work/workflow-artifact-identity/.generated-install"
    matches = [path for path in parent.iterdir() if path.is_dir() and re.fullmatch(r"[0-9a-f]{64}", path.name)]
    require(len(matches) == 1, "actual delivery directory cardinality")
    return matches[0]


def assert_actual_delivery_records(repo: Path, expected: dict[str, dict]) -> None:
    repo_fd = open_absolute_directory(repo)
    try:
        for relative, record in expected.items():
            require(file_record_at(repo_fd, relative) == record,
                    f"actual delivery observed identity: {relative}")
    finally:
        os.close(repo_fd)


def observe_delivery_destinations(repo: Path) -> list[dict]:
    program = r"""
import hashlib,json,os,stat,sys
root=sys.argv[1]; paths=json.loads(sys.argv[2]); out=[]
for rel in paths:
 path=os.path.join(root,rel)
 try:
  fd=os.open(path,os.O_RDONLY|os.O_NOFOLLOW|os.O_CLOEXEC); info=os.fstat(fd); chunks=[]
  while True:
   chunk=os.read(fd,65536)
   if not chunk: break
   chunks.append(chunk)
  os.close(fd); out.append({'path':rel,'state':'present','mode':stat.S_IMODE(info.st_mode),'sha256':hashlib.sha256(b''.join(chunks)).hexdigest()})
 except FileNotFoundError: out.append({'path':rel,'state':'absent'})
print(json.dumps(out,sort_keys=True,separators=(',',':')))
"""
    observed = subprocess.run([sys.executable, "-c", program, str(repo), json.dumps(AUTHORED + GENERATED)],
                              text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False, timeout=10)
    require(observed.returncode == 0 and observed.stderr == "", "independent delivery destination observer")
    return json.loads(observed.stdout)


def observe_delivery_destination_physical_identity(repo: Path) -> list[dict]:
    observed = []
    for relative in AUTHORED + GENERATED:
        try:
            info = os.lstat(repo / relative)
        except FileNotFoundError:
            observed.append({"path": relative, "state": "absent"})
            continue
        observed.append({
            "path": relative, "state": "present", "device": info.st_dev,
            "inode": info.st_ino, "ctime_ns": info.st_ctime_ns,
        })
    return observed


def valid_delivery_destination_observation(observed: list[dict], expected: list[dict]) -> bool:
    paths = [record.get("path") for record in observed]
    return (len(observed) == len(AUTHORED) + len(GENERATED) == len(expected)
            and len(paths) == len(set(paths)) and paths == AUTHORED + GENERATED
            and observed == expected)


def _run_delivery_boundary_child(boundary: str, operation, expected_exit: int) -> None:
    require(delivery_test_death_boundary() is None,
            "delivery boundary parent death selection")
    pid = os.fork()
    if pid == 0:
        try:
            _DELIVERY_TEST["death_boundary"] = boundary
            operation()
        except BaseException:
            sys.excepthook(*sys.exc_info())
            sys.stderr.flush()
            os._exit(125)
        os._exit(0)
    _, status = os.waitpid(pid, 0)
    require(not os.WIFSIGNALED(status),
            f"delivery boundary child signaled: {boundary}: {os.WTERMSIG(status)}")
    require(os.WIFEXITED(status), f"delivery boundary child termination: {boundary}")
    require(os.WEXITSTATUS(status) == expected_exit,
            f"delivery boundary child exit: {boundary}: {os.WEXITSTATUS(status)}")
    require(delivery_test_death_boundary() is None,
            "delivery boundary parent death selection cleared")


def delivery_boundary_case(kind: str, boundary: str) -> None:
    require(delivery_test_fast_fixture(),
            "delivery boundary case requires fast fixture")
    require(boundary in DELIVERY_BOUNDARIES, "delivery boundary case registry")
    prepare = (boundary == "journal-created" or boundary.startswith("bootstrap-")
               or boundary.startswith("snapshot-") or boundary.startswith("authored-"))
    rollback = boundary.startswith("rollback-")
    expected_kind = "prepare" if prepare else "rollback" if rollback else "install"
    require(kind == expected_kind, "delivery boundary case kind")
    repo = Path(os.environ["DEVRITES_REPO_ROOT"]).absolute()

    if kind == "prepare":
        expected_exit = (
            87 if boundary.endswith("after-sync")
            else 88 if boundary.endswith("after-partial")
            else 89 if boundary.endswith("before-rename")
            else 86
        )
        _run_delivery_boundary_child(boundary, delivery_prepare, expected_exit)
        delivery_prepare()
        delivery = actual_delivery_directory(repo)
        require(json.loads((delivery / "journal.json").read_text())["state"] == "SNAPSHOTTING",
                f"actual prepare resume: {boundary}")
        delivery_recover(str(delivery))
        require(json.loads((delivery / "journal.json").read_text())["state"] == "FAILED",
                f"actual prepare recovery: {boundary}")
        return

    delivery_prepare()
    delivery = actual_delivery_directory(repo)
    if kind == "install":
        expected_exit = 87 if boundary == "journal-replacement-after-sync" else 86
        _run_delivery_boundary_child(
            boundary, lambda: delivery_install(str(delivery)), expected_exit,
        )
        committed = boundary in {"commit-recorded", "cleaning-recorded", "cleanup-stage-removed",
                                 "cleanup-backups-removed", "cleaned-recorded"}
        if committed:
            journal = json.loads((delivery / "journal.json").read_text())
            expected_state_at_boundary = {
                "commit-recorded": "COMMITTED",
                "cleaning-recorded": "CLEANING",
                "cleanup-stage-removed": "CLEANING",
                "cleanup-backups-removed": "CLEANING",
                "cleaned-recorded": "CLEANED",
            }[boundary]
            require(journal["state"] == expected_state_at_boundary
                    and valid_delivery_destination_observation(
                        observe_delivery_destinations(repo), journal["expected_post"]),
                    f"exact post-commit death state: {boundary}")
        delivery_recover(str(delivery))
        expected = "CLEANED" if committed else "FAILED"
        require(json.loads((delivery / "journal.json").read_text())["state"] == expected,
                f"actual install recovery: {boundary}")
        return

    _run_delivery_boundary_child(
        "proving-recorded", lambda: delivery_install(str(delivery)), 86,
    )
    _run_delivery_boundary_child(
        boundary, lambda: delivery_recover(str(delivery)), 86,
    )
    delivery_recover(str(delivery))
    require(json.loads((delivery / "journal.json").read_text())["state"] == "FAILED",
            f"actual rollback recovery: {boundary}")


def check_actual_delivery_modes() -> None:
    executed: set[str] = set()
    full_prepare_boundaries = {
        boundary for boundary in DELIVERY_BOUNDARIES
        if boundary == "journal-created" or boundary.startswith("bootstrap-")
        or boundary.startswith("snapshot-") or boundary.startswith("authored-")
    }
    full_rollback_boundaries = {boundary for boundary in DELIVERY_BOUNDARIES if boundary.startswith("rollback-")}
    full_install_boundaries = set(DELIVERY_BOUNDARIES) - full_prepare_boundaries - full_rollback_boundaries
    if parse_boundary_shard_spec() is not None:
        allowed = filter_boundary_shard(set(DELIVERY_BOUNDARIES))
        prepare_boundaries = full_prepare_boundaries & allowed
        rollback_boundaries = full_rollback_boundaries & allowed
        install_boundaries = full_install_boundaries & allowed
        require(prepare_boundaries or rollback_boundaries or install_boundaries,
                "workflow-artifact boundary shard has no assigned boundaries")
    else:
        prepare_boundaries = full_prepare_boundaries
        rollback_boundaries = full_rollback_boundaries
        install_boundaries = full_install_boundaries
    expected_boundaries = (
        filter_boundary_shard(set(DELIVERY_BOUNDARIES))
        if parse_boundary_shard_spec() is not None else set(DELIVERY_BOUNDARIES)
    )

    def terminal(repo: Path, delivery: Path, before: dict[str, dict], expected_post: list[dict] | None,
                 expected_state: str) -> None:
        journal = json.loads((delivery / "journal.json").read_text())
        require(journal["state"] == expected_state and len(journal["authored"]) == len(AUTHORED)
                and len(journal["generated"]) == len(GENERATED),
                f"actual delivery terminal journal: {expected_state}")
        repo_fd = open_absolute_directory(repo); delivery_fd = open_absolute_directory(delivery)
        try:
            validate_delivery_journal(
                journal, delivery_fd, repo_fd, delivery.name,
                require_complete=True,
            )
        finally:
            os.close(delivery_fd); os.close(repo_fd)
        if expected_state == "FAILED":
            assert_actual_delivery_records(repo, before)
        else:
            require(expected_post is not None and valid_delivery_destination_observation(
                        observe_delivery_destinations(repo), expected_post),
                    f"independent 16/22 terminal observation: {expected_state}")
            require(not (delivery / "stage").exists() and not (delivery / "backups").exists(),
                    "actual CLEANED transaction trees absent")

    def prepare_case(boundary: str) -> tuple[tempfile.TemporaryDirectory, str, Path, Path, dict[str, dict]]:
        temporary = tempfile.TemporaryDirectory()
        try:
            repo = Path(temporary.name).resolve() / "repo"; before = create_actual_delivery_repo(repo)
            result = run_actual_delivery_mode(
                repo, ["--delivery-boundary-case", "prepare", boundary],
            )
            require(result.returncode == 0
                    and "delivery_state=SNAPSHOTTING" in result.stdout
                    and "delivery_state=FAILED" in result.stdout,
                    f"actual prepare boundary case: {boundary}: {result.stdout[-1000:]}")
            delivery = actual_delivery_directory(repo)
            return temporary, boundary, repo, delivery, before
        except BaseException:
            temporary.cleanup()
            raise

    prepare_results = []
    try:
        with ThreadPoolExecutor(max_workers=delivery_parallel_workers()) as executor:
            prepare_futures = [executor.submit(prepare_case, boundary)
                               for boundary in sorted(prepare_boundaries)]
            prepare_failure = None
            for future in prepare_futures:
                try:
                    prepare_results.append(future.result())
                except BaseException as error:
                    if prepare_failure is None:
                        prepare_failure = error
        if prepare_failure is not None:
            raise prepare_failure
        for temporary, boundary, repo, delivery, before in prepare_results:
            terminal(repo, delivery, before, None, "FAILED")
            executed.add(boundary)
    finally:
        for temporary, *_rest in prepare_results:
            temporary.cleanup()

    def install_case(boundary: str) -> tuple[
            tempfile.TemporaryDirectory, str, Path, Path, dict[str, dict], list[dict] | None, bool, str]:
        temporary = tempfile.TemporaryDirectory()
        try:
            repo = Path(temporary.name).resolve() / "repo"; before = create_actual_delivery_repo(repo)
            result = run_actual_delivery_mode(
                repo, ["--delivery-boundary-case", "install", boundary],
            )
            committed = boundary in {"commit-recorded", "cleaning-recorded", "cleanup-stage-removed",
                                     "cleanup-backups-removed", "cleaned-recorded"}
            expected = "CLEANED" if committed else "FAILED"
            require(result.returncode == 0 and f"delivery_state={expected}" in result.stdout,
                    f"actual install boundary case: {boundary}: {result.stdout[-1000:]}")
            delivery = actual_delivery_directory(repo)
            journal = json.loads((delivery / "journal.json").read_text())
            expected_post = journal["expected_post"] if committed else None
            return temporary, boundary, repo, delivery, before, expected_post, committed, expected
        except BaseException:
            temporary.cleanup()
            raise

    install_results = []
    try:
        with ThreadPoolExecutor(max_workers=delivery_parallel_workers()) as executor:
            install_futures = [executor.submit(install_case, boundary)
                               for boundary in sorted(install_boundaries)]
            install_failure = None
            for future in install_futures:
                try:
                    install_results.append(future.result())
                except BaseException as error:
                    if install_failure is None:
                        install_failure = error
        if install_failure is not None:
            raise install_failure
        for temporary, boundary, repo, delivery, before, expected_post, committed, expected in install_results:
            terminal(repo, delivery, before, expected_post if committed else None, expected)
            executed.add(boundary)
    finally:
        for temporary, *_rest in install_results:
            temporary.cleanup()

    def rollback_case(boundary: str) -> tuple[tempfile.TemporaryDirectory, str, Path, Path, dict[str, dict]]:
        temporary = tempfile.TemporaryDirectory()
        try:
            repo = Path(temporary.name).resolve() / "repo"; before = create_actual_delivery_repo(repo)
            result = run_actual_delivery_mode(
                repo, ["--delivery-boundary-case", "rollback", boundary],
            )
            require(result.returncode == 0 and "delivery_state=FAILED" in result.stdout,
                    f"actual rollback boundary case: {boundary}: {result.stdout[-1000:]}")
            delivery = actual_delivery_directory(repo)
            return temporary, boundary, repo, delivery, before
        except BaseException:
            temporary.cleanup()
            raise

    rollback_results = []
    try:
        with ThreadPoolExecutor(max_workers=delivery_parallel_workers()) as executor:
            rollback_futures = [executor.submit(rollback_case, boundary)
                                for boundary in sorted(rollback_boundaries)]
            rollback_failure = None
            for future in rollback_futures:
                try:
                    rollback_results.append(future.result())
                except BaseException as error:
                    if rollback_failure is None:
                        rollback_failure = error
        if rollback_failure is not None:
            raise rollback_failure
        for temporary, boundary, repo, delivery, before in rollback_results:
            terminal(repo, delivery, before, None, "FAILED")
            executed.add(boundary)
    finally:
        for temporary, *_rest in rollback_results:
            temporary.cleanup()

    if parse_boundary_shard_spec() is None or parse_boundary_shard_spec()[0] == 1:
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
            require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                    "post-commit mutant prepare")
            delivery = actual_delivery_directory(repo)
            committed = run_actual_delivery_mode(repo, ["--delivery-install", str(delivery)], "commit-recorded")
            committed_journal = json.loads((delivery / "journal.json").read_text())
            expected_post = committed_journal["expected_post"]
            require(committed.returncode == 86 and committed_journal["state"] == "COMMITTED",
                    "post-commit mutant terminal")
            target = repo / AUTHORED[0]; original = target.read_bytes(); mode = stat.S_IMODE(target.stat().st_mode)
            target.unlink()
            require(not valid_delivery_destination_observation(observe_delivery_destinations(repo), expected_post),
                    "post-commit removal mutant rejected")
            atomic_write(target, original, mode)
            atomic_write(target, b"corrupt\n", mode)
            require(not valid_delivery_destination_observation(observe_delivery_destinations(repo), expected_post),
                    "post-commit corruption mutant rejected")
            atomic_write(target, original, mode); target.chmod(0o600 if mode != 0o600 else 0o700)
            require(not valid_delivery_destination_observation(observe_delivery_destinations(repo), expected_post),
                    "post-commit mode mutant rejected")
            target.chmod(mode)
            observed = observe_delivery_destinations(repo)
            require(not valid_delivery_destination_observation(observed + [observed[0]], expected_post),
                    "post-commit cardinality mutant rejected")
            require(valid_delivery_destination_observation(observe_delivery_destinations(repo), expected_post),
                    "post-commit mutant restoration")
            require(run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)]).returncode == 0,
                    "post-commit mutant cleanup")

        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve() / "repo"; before = create_actual_delivery_repo(repo)
            require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                    "skipped replacement mutant prepare")
            delivery = actual_delivery_directory(repo)
            skipped = run_actual_delivery_mode(
                repo, ["--delivery-install", str(delivery)], skip_generated=0,
            )
            require(skipped.returncode != 0 and "generated replacement effect" in skipped.stdout,
                    "skipped changed generated replacement rejected")
            require(json.loads((delivery / "journal.json").read_text())["state"] == "FAILED",
                    "skipped replacement rolls back")
            assert_actual_delivery_records(repo, before)

        with tempfile.TemporaryDirectory() as tmp:
            base = Path(tmp).resolve()
            repo = base / "repo"; create_actual_delivery_repo(repo)
            candidate = base / "private-candidate"
            for relative in AUTHORED:
                source = canonical_root() / relative
                destination = candidate / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(source, destination)
            private_driver = candidate / "tests/workflow-artifact-identity-test.sh"
            prepared = run_actual_delivery_mode(repo, ["--delivery-prepare"], driver=private_driver)
            require(prepared.returncode == 0, f"private bootstrap prepare: {prepared.stdout[-1000:]}")
            delivery = actual_delivery_directory(repo)
            installed_driver = repo / "tests/workflow-artifact-identity-test.sh"
            installed = run_actual_delivery_mode(
                repo, ["--delivery-install", str(delivery)], driver=installed_driver,
            )
            require(installed.returncode == 0 and "delivery_state=CLEANED" in installed.stdout,
                    f"private bootstrap installed-driver handoff: {installed.stdout[-1000:]}")

    require(executed == expected_boundaries,
            f"delivery registry execution mismatch: {sorted(expected_boundaries - executed)}")


def check_delivery_third_state_guards() -> None:
    for relative, absent_preimage in ((AUTHORED[0], False), (AUTHORED[1], True)):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve() / "repo"
            create_actual_delivery_repo(repo)
            if absent_preimage:
                (repo / relative).unlink()
            prepared = run_actual_delivery_mode(repo, ["--delivery-prepare"])
            require(prepared.returncode == 0, f"third-state prepare: {relative}")
            delivery = actual_delivery_directory(repo)
            interrupted = run_actual_delivery_mode(
                repo, ["--delivery-install", str(delivery)], "proving-recorded",
            )
            require(interrupted.returncode == 86, f"third-state setup: {relative}")
            target = repo / relative
            atomic_write(target, b"concurrent-third-state\n", 0o600)
            before_recovery = observe_delivery_destinations(repo)
            recovered = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
            require(recovered.returncode != 0, f"third-state recovery accepted: {relative}")
            require(observe_delivery_destinations(repo) == before_recovery,
                    f"initial third-state sweep wrote destinations: {relative}")
            require(target.read_bytes() == b"concurrent-third-state\n",
                    f"third-state overwritten or unlinked: {relative}")

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "per-path guard prepare")
        delivery = actual_delivery_directory(repo)
        require(run_actual_delivery_mode(
                    repo, ["--delivery-install", str(delivery)], "proving-recorded",
                ).returncode == 86,
                "per-path guard setup")
        repo_fd = open_absolute_directory(repo)
        delivery_fd = open_absolute_directory(delivery)
        lock_fd = delivery_lock(delivery_fd)
        original_hook = delivery_test_mutation
        calls = 0
        concurrent_relative = GENERATED[-1]
        try:
            journal = read_journal(delivery_fd)
            before = observe_delivery_destinations(repo)
            def mutate_after_sweep(current_repo_fd: int, site: str,
                                   relative: str | None = None) -> None:
                nonlocal calls
                require(site == "before-rollback-claim" and relative is not None,
                        "per-path guard rollback hook")
                if calls == 0:
                    require(relative == concurrent_relative, "per-path guard rollback order")
                    atomic_write_at(current_repo_fd, relative, b"post-sweep-third-state\n", 0o600, lock_fd)
                calls += 1
                original_hook(current_repo_fd, site, relative)
            globals()["delivery_test_mutation"] = mutate_after_sweep
            try:
                restore_all(
                    repo_fd, delivery_fd, lock_fd, journal, delivery.name,
                )
            except AssertionError as error:
                require(str(error).startswith("delivery destination outside rollback pair:"),
                        "per-path third-state failure identity")
            else:
                fail("post-sweep destination change accepted")
            after = observe_delivery_destinations(repo)
            changed = [record["path"] for record, prior in zip(after, before) if record != prior]
            require(changed == [concurrent_relative]
                    and (repo / concurrent_relative).read_bytes() == b"post-sweep-third-state\n",
                    "atomic claim preserves post-sweep destination change")
        finally:
            globals()["delivery_test_mutation"] = original_hook
            os.close(lock_fd); os.close(delivery_fd); os.close(repo_fd)


def check_drift_016_delivery_mutants() -> None:
    for mutation, relative in (
        ("late-lookalike", ".generated-install-evil/deep/observed"),
        ("late-nested-git", "nested/.git/deep/observed"),
    ):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve() / "repo"; before = create_actual_delivery_repo(repo)
            require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                    f"late outside prepare: {mutation}")
            delivery = actual_delivery_directory(repo)
            rejected = run_actual_delivery_mode(
                repo, ["--delivery-install", str(delivery)], mutation=mutation,
            )
            journal = json.loads((delivery / "journal.json").read_text())
            require(rejected.returncode != 0 and journal["state"] == "RESTORED"
                    and "COMMITTED" not in rejected.stdout,
                    f"late outside drift rejected before commit: {mutation}")
            require((repo / relative).read_bytes() == (mutation + "\n").encode(),
                    f"late outside path preserved: {mutation}")
            assert_actual_delivery_records(repo, before)

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; before = create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "generator overflow prepare")
        delivery = actual_delivery_directory(repo)
        rejected = run_actual_delivery_mode(
            repo, ["--delivery-install", str(delivery)], mutation="generator-overflow",
        )
        journal = json.loads((delivery / "journal.json").read_text())
        pgid = int((delivery / "stage/generator-pgid").read_text())
        require(rejected.returncode != 0
                and "delivery generator failed: output-overflow captured=256" in rejected.stdout,
                "actual generator overflow diagnostic")
        require(journal["state"] == "FAILED" and "stage_manifest_sha256" not in journal
                and "expected_post" not in journal, "generator overflow before STAGED")
        require(not process_group_alive(pgid), "generator overflow group reaped")
        assert_actual_delivery_records(repo, before)

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "generated race prepare")
        delivery = actual_delivery_directory(repo)
        relative = GENERATED[0]
        rejected = run_actual_delivery_mode(
            repo, ["--delivery-install", str(delivery)], mutation="generated-race",
        )
        journal = json.loads((delivery / "journal.json").read_text())
        require(rejected.returncode != 0 and journal["state"].startswith("INSTALLING(")
                and journal["state"] != "COMMITTED", "generated race rejected before commit")
        require((repo / relative).read_bytes() == b"generated-race\n",
                "generated race preserves unowned bytes")

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "rollback race prepare")
        delivery = actual_delivery_directory(repo)
        interrupted = run_actual_delivery_mode(
            repo, ["--delivery-install", str(delivery)], "proving-recorded",
        )
        require(interrupted.returncode == 86, "rollback race install setup")
        relative = GENERATED[-1]
        rejected = run_actual_delivery_mode(
            repo, ["--delivery-recover", str(delivery)], mutation="rollback-race",
        )
        journal = json.loads((delivery / "journal.json").read_text())
        require(rejected.returncode != 0 and journal["state"].startswith("ROLLING_BACK(")
                and journal["state"] != "COMMITTED", "rollback race rejected")
        require((repo / relative).read_bytes() == b"rollback-race\n",
                "rollback race preserves unowned bytes")


def check_first_authored_claim_exception_rollback() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        before = create_actual_delivery_repo(repo)
        failed = run_actual_delivery_mode(
            repo, ["--delivery-prepare"],
            mutation="first-authored-claim-exception",
        )
        delivery = actual_delivery_directory(repo)
        journal = json.loads((delivery / "journal.json").read_text())
        require(failed.returncode != 0
                and "injected first authored claim exception" in failed.stdout,
                "first authored private-claim exception reached outer handler")
        require(journal["state"] == "FAILED"
                and "mutation_intent" not in journal
                and "installed_authored_intent" not in journal,
                "first authored intent reconciled to truthful FAILED")
        require(not any(name.startswith(".mutation-") for name in os.listdir(delivery)),
                "first authored rollback mutation artifacts absent")
        require(len(before) == len(AUTHORED) + len(GENERATED) == 38,
                "first authored rollback exact destination cardinality")
        assert_actual_delivery_records(repo, before)
        repo_fd = open_absolute_directory(repo)
        delivery_fd = open_absolute_directory(delivery)
        try:
            require(read_outside_manifest(delivery_fd, journal["outside_manifest"])
                    == manifest_at(
                        repo_fd, ALL_DESTINATIONS,
                        f".devrites/work/workflow-artifact-identity/.generated-install/{delivery.name}",
                    ), "first authored rollback outside identity")
        finally:
            os.close(delivery_fd); os.close(repo_fd)


def check_drift_017_prepare_outside_drift() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        before = create_actual_delivery_repo(repo)
        outside_relative = ".drift-017-prepare-outside/record"
        atomic_write(repo / outside_relative, b"frozen outside record\n", 0o600)
        repo_fd = open_absolute_directory(repo)
        try:
            protected_before = protected_records_at(repo_fd)
            frozen_outside = manifest_at(
                repo_fd, ALL_DESTINATIONS,
                ".devrites/work/workflow-artifact-identity/.generated-install",
            )
        finally:
            os.close(repo_fd)
        require(outside_relative in frozen_outside, "prepare outside fixture frozen")

        rejected = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], mutation="prepare-outside-drift",
        )
        delivery = actual_delivery_directory(repo)
        journal = json.loads((delivery / "journal.json").read_text())
        require(rejected.returncode != 0
                and "delivery outside manifest binding" in rejected.stdout,
                f"prepare outside drift rejected: {rejected.stdout[-1000:]}")
        require(journal["state"] == "SNAPSHOTTING"
                and len(journal["authored"]) == 0
                and len(journal["generated"]) == 0
                and journal["installed_authored"] == 0
                and journal["installed_generated"] == 0,
                "prepare outside drift zero-write journal")
        delivery_fd = open_absolute_directory(delivery)
        try:
            retained_outside = read_outside_manifest(
                delivery_fd, journal["outside_manifest"],
            )
        finally:
            os.close(delivery_fd)
        require(outside_relative in retained_outside
                and not (repo / outside_relative).exists(),
                "prepare outside drift changed frozen identity")
        require(not (delivery / "backups").exists()
                and not (delivery / "stage").exists(),
                "prepare outside drift before snapshot or stage")
        require(len(before) == len(AUTHORED) + len(GENERATED) == 38,
                "prepare outside drift destination cardinality")
        assert_actual_delivery_records(repo, before)
        repo_fd = open_absolute_directory(repo)
        try:
            require(protected_records_at(repo_fd) == protected_before,
                    "prepare outside drift protected identity")
        finally:
            os.close(repo_fd)


def journal_successor_fixture(journal: dict, state: str, **changes) -> dict:
    candidate = json.loads(json.dumps(journal))
    candidate["state"] = state
    candidate["updated_ns"] += 1
    candidate["journal_sequence"] += 1
    candidate.update(changes)
    return candidate


def write_journal_temporary_fixture(delivery: Path, journal: dict) -> None:
    encoded = (json.dumps(journal, sort_keys=True, separators=(",", ":")) + "\n").encode()
    require(len(encoded) <= DELIVERY_JOURNAL_MAX_BYTES,
            "temporary journal fixture bound")
    atomic_write(delivery / ".journal.json.workflow-artifact.tmp", encoded, 0o600)


def complete_gate_records_fixture() -> list[dict]:
    return [
        {
            "command": command, "execution_prefix": delivery_execution_prefix(),
            "sha256": "0" * 64, "signal": signal or "exit=0",
        }
        for command, signal in DELIVERY_GATES
    ]


def check_delivery_recovery_identity() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "replayed journal identity prepare")
        delivery = actual_delivery_directory(repo)
        for index, mode in enumerate(("--delivery-install", "--delivery-recover")):
            forged = delivery.parent / (("e" if index == 0 else "f") * 64)
            shutil.copytree(delivery, forged)
            forged.chmod(0o700)
            if mode == "--delivery-recover":
                current = json.loads((forged / "journal.json").read_text())
                successor = journal_successor_fixture(
                    current, "ROLLING_BACK(1)",
                    mutation_intent={
                        "action": "restore", "group": "generated",
                        "index": len(GENERATED) - 1, "path": GENERATED[-1],
                    },
                )
                write_journal_temporary_fixture(forged, successor)
            before = observe_delivery_destinations(repo)
            physical_before = observe_delivery_destination_physical_identity(repo)
            durable = (forged / "journal.json").read_bytes()
            rejected = run_actual_delivery_mode(repo, [mode, str(forged)])
            require(rejected.returncode != 0
                    and "delivery directory candidate identity" in rejected.stdout,
                    f"replayed journal directory identity mutant: {mode}")
            require(observe_delivery_destinations(repo) == before
                    and observe_delivery_destination_physical_identity(repo) == physical_before
                    and (forged / "journal.json").read_bytes() == durable,
                    f"replayed journal physical zero destination writes: {mode}")


def check_temporary_journal_successors() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; before = create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "legal rollback successor prepare")
        delivery = actual_delivery_directory(repo)
        current = json.loads((delivery / "journal.json").read_text())
        successor = journal_successor_fixture(
            current, "ROLLING_BACK(1)",
            mutation_intent={
                "action": "restore", "group": "generated",
                "index": len(GENERATED) - 1, "path": GENERATED[-1],
            },
        )
        write_journal_temporary_fixture(delivery, successor)
        recovered = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        require(recovered.returncode == 0 and "delivery_state=FAILED" in recovered.stdout
                and not (delivery / ".journal.json.workflow-artifact.tmp").exists(),
                "legal temporary journal successor promoted")
        assert_actual_delivery_records(repo, before)

    for fabricated_state in ("COMMITTED", "CLEANING", "CLEANED"):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
            require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                    f"fabricated successor prepare: {fabricated_state}")
            delivery = actual_delivery_directory(repo)
            staged = run_actual_delivery_mode(
                repo, ["--delivery-install", str(delivery)], "staged-recorded",
            )
            require(staged.returncode == 86, f"fabricated successor staged: {fabricated_state}")
            current = json.loads((delivery / "journal.json").read_text())
            candidate = journal_successor_fixture(
                current, fabricated_state, gates=complete_gate_records_fixture(),
            )
            write_journal_temporary_fixture(delivery, candidate)
            before = observe_delivery_destinations(repo)
            durable = (delivery / "journal.json").read_bytes()
            rejected = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
            require(rejected.returncode != 0
                    and (delivery / "journal.json").read_bytes() == durable
                    and observe_delivery_destinations(repo) == before,
                    f"STAGED fabricated terminal successor mutant: {fabricated_state}")

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "transition-only successor prepare")
        delivery = actual_delivery_directory(repo)
        require(run_actual_delivery_mode(
                    repo, ["--delivery-install", str(delivery)], "staged-recorded",
                ).returncode == 86, "transition-only successor staged")
        current = json.loads((delivery / "journal.json").read_text())
        for relative, expected in zip(GENERATED, current["expected_post"][len(AUTHORED):]):
            staged_path = delivery / "stage" / relative.removeprefix("pack/generated/")
            atomic_write(repo / relative, staged_path.read_bytes(), expected["mode"])
        candidate = journal_successor_fixture(
            current, "COMMITTED", gates=complete_gate_records_fixture(),
        )
        write_journal_temporary_fixture(delivery, candidate)
        before = observe_delivery_destinations(repo)
        durable = (delivery / "journal.json").read_bytes()
        rejected = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        require(rejected.returncode != 0
                and (delivery / "journal.json").read_bytes() == durable
                and observe_delivery_destinations(repo) == before,
                "legal-schema illegal transition mutant")

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "legal commit successor prepare")
        delivery = actual_delivery_directory(repo)
        require(run_actual_delivery_mode(
                    repo, ["--delivery-install", str(delivery)], "commit-before",
                ).returncode == 86, "legal commit successor proving")
        current = json.loads((delivery / "journal.json").read_text())
        incomplete = journal_successor_fixture(
            current, "COMMITTED", gates=current["gates"][:-1],
        )
        write_journal_temporary_fixture(delivery, incomplete)
        before = observe_delivery_destinations(repo)
        rejected = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        require(rejected.returncode != 0
                and observe_delivery_destinations(repo) == before,
                "committed temporary incomplete-gates mutant")
        (delivery / ".journal.json.workflow-artifact.tmp").unlink()
        write_journal_temporary_fixture(
            delivery, journal_successor_fixture(current, "COMMITTED"),
        )
        recovered = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        sidecar_info = os.lstat(delivery / OUTSIDE_MANIFEST_NAME)
        require(recovered.returncode == 0 and "delivery_state=CLEANED" in recovered.stdout
                and stat.S_ISREG(sidecar_info.st_mode) and sidecar_info.st_nlink == 1
                and stat.S_IMODE(sidecar_info.st_mode) == 0o600,
                "legal committed temporary successor promoted with sidecar evidence")

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "illegal legal-schema successor prepare")
        delivery = actual_delivery_directory(repo)
        require(run_actual_delivery_mode(
                    repo, ["--delivery-install", str(delivery)], "commit-before",
                ).returncode == 86, "illegal legal-schema successor proving")
        current = json.loads((delivery / "journal.json").read_text())
        write_journal_temporary_fixture(
            delivery, journal_successor_fixture(current, "CLEANING"),
        )
        durable = (delivery / "journal.json").read_bytes()
        before = observe_delivery_destinations(repo)
        rejected = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        require(rejected.returncode != 0
                and "journal temporary legal successor" in rejected.stdout
                and (delivery / "journal.json").read_bytes() == durable
                and observe_delivery_destinations(repo) == before,
                "standalone-valid illegal transition omission mutant")

    for boundary, successor_state in (("commit-before", "COMMITTED"),
                                      ("commit-recorded", "CLEANING")):
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
            require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                    f"post-state successor prepare: {boundary}")
            delivery = actual_delivery_directory(repo)
            require(run_actual_delivery_mode(
                        repo, ["--delivery-install", str(delivery)], boundary,
                    ).returncode == 86, f"post-state successor setup: {boundary}")
            current = json.loads((delivery / "journal.json").read_text())
            write_journal_temporary_fixture(
                delivery, journal_successor_fixture(current, successor_state),
            )
            atomic_write(repo / AUTHORED[0], b"post-state-mutant\n", 0o600)
            before = observe_delivery_destinations(repo)
            rejected = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
            require(rejected.returncode != 0
                    and observe_delivery_destinations(repo) == before,
                    f"temporary successor expected-post mutant: {boundary}")


def check_future_journal_clock_rollback() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        before = create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "future journal prepare")
        delivery = actual_delivery_directory(repo)
        current = json.loads((delivery / "journal.json").read_text())
        future_updated_ns = time.time_ns() + 10**15
        current["updated_ns"] = future_updated_ns
        atomic_write(
            delivery / "journal.json",
            (json.dumps(current, sort_keys=True, separators=(",", ":")) + "\n").encode(),
            0o600,
        )
        repo_fd = open_absolute_directory(repo)
        delivery_fd = open_absolute_directory(delivery)
        try:
            validate_delivery_journal(
                current, delivery_fd, repo_fd, delivery.name,
                require_complete=True,
            )
        finally:
            os.close(delivery_fd); os.close(repo_fd)
        successor = journal_successor_fixture(
            current, "ROLLING_BACK(1)",
            mutation_intent={
                "action": "restore", "group": "generated",
                "index": len(GENERATED) - 1, "path": GENERATED[-1],
            },
        )
        require(legal_journal_successor(current, successor),
                "future journal exact legal successor")
        write_journal_temporary_fixture(delivery, successor)
        recovered = run_actual_delivery_mode(
            repo, ["--delivery-recover", str(delivery)],
        )
        final = json.loads((delivery / "journal.json").read_text())
        require(recovered.returncode == 0 and final["state"] == "FAILED"
                and final["updated_ns"] > future_updated_ns
                and final["journal_sequence"] > current["journal_sequence"],
                "future journal recovery across wall-clock rollback")
        assert_actual_delivery_records(repo, before)


def check_recursive_outside_directory_protection() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; before = create_actual_delivery_repo(repo)
        outside = repo / ".outside-directory"
        outside.mkdir(mode=0o700)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "outside directory mode prepare")
        delivery = actual_delivery_directory(repo)
        rejected = run_actual_delivery_mode(
            repo, ["--delivery-install", str(delivery)],
            mutation="unrelated-directory-mode",
        )
        journal = json.loads((delivery / "journal.json").read_text())
        require(rejected.returncode != 0 and journal["state"] == "RESTORED"
                and stat.S_IMODE(outside.stat().st_mode) == 0o755,
                "unrelated directory mode mutant blocks commit")
        assert_actual_delivery_records(repo, before)


def check_terminal_recovery_idempotence() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; before = create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "FAILED idempotence prepare")
        delivery = actual_delivery_directory(repo)
        first = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        require(first.returncode == 0 and "delivery_state=FAILED" in first.stdout,
                "FAILED idempotence first recovery")
        journal_bytes = (delivery / "journal.json").read_bytes()
        sidecar_bytes = (delivery / OUTSIDE_MANIFEST_NAME).read_bytes()
        destinations = observe_delivery_destinations(repo)
        second = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        require(second.returncode == 0 and "delivery_state=FAILED" in second.stdout
                and (delivery / "journal.json").read_bytes() == journal_bytes
                and (delivery / OUTSIDE_MANIFEST_NAME).read_bytes() == sidecar_bytes
                and observe_delivery_destinations(repo) == destinations,
                "durable FAILED second recovery is read-only")
        assert_actual_delivery_records(repo, before)

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; before = create_actual_delivery_repo(repo)
        require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
                "RESTORED idempotence prepare")
        delivery = actual_delivery_directory(repo)
        require(run_actual_delivery_mode(
                    repo, ["--delivery-install", str(delivery)], "proving-recorded",
                ).returncode == 86, "RESTORED idempotence install setup")
        interrupted = run_actual_delivery_mode(
            repo, ["--delivery-recover", str(delivery)], "rollback-restored-recorded",
        )
        restored = json.loads((delivery / "journal.json").read_text())
        require(interrupted.returncode == 86 and restored["state"] == "RESTORED",
                "interrupted durable RESTORED setup")
        destinations = observe_delivery_destinations(repo)
        resumed = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        failed_bytes = (delivery / "journal.json").read_bytes()
        failed = json.loads(failed_bytes)
        require(resumed.returncode == 0 and failed["state"] == "FAILED"
                and failed["journal_sequence"] == restored["journal_sequence"] + 1
                and observe_delivery_destinations(repo) == destinations,
                "RESTORED advances once without rollback replay")
        repeated = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
        require(repeated.returncode == 0
                and (delivery / "journal.json").read_bytes() == failed_bytes,
                "interrupted RESTORED reaches idempotent FAILED")
        assert_actual_delivery_records(repo, before)


def check_outside_manifest_bounds() -> None:
    require(11_950_000 <= OUTSIDE_MANIFEST_MAX_BYTES <= 32 * 1024 * 1024,
            "outside manifest encoded-byte capacity")
    require(0 < OUTSIDE_MANIFEST_MAX_ENTRIES < 1_000_000
            and 0 < OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS <= DELIVERY_PROCESS_TIMEOUT_SECONDS,
            "outside manifest finite entry and scan-time bounds")
    def encode_fixture(value, *, canonical=True):
        separators = (",", ":") if canonical else None
        return (json.dumps(value, separators=separators) + "\n").encode()

    def fixture_binding(raw, rows):
        return {
            "relative": OUTSIDE_MANIFEST_NAME, "sha256": sha(raw),
            "bytes": len(raw), "rows": rows,
        }

    valid_rows = [
        ["block", "block", 0o600, os.getuid(), os.getgid(), 7],
        ["character", "character", 0o600, os.getuid(), os.getgid(), 8],
        ["directory", "directory", 0o700, os.getuid(), os.getgid()],
        ["fifo", "fifo", 0o600, os.getuid(), os.getgid()],
        ["file", "file", 0o600, os.getuid(), os.getgid(), 1, "0" * 64],
        ["link", "symlink", 0o777, os.getuid(), os.getgid(), "target"],
        ["socket", "socket", 0o600, os.getuid(), os.getgid()],
    ]
    valid_raw = encode_fixture(valid_rows)
    require(parse_outside_manifest(valid_raw, fixture_binding(valid_raw, 7)) == {
                "block": {"type": "block", "mode": 0o600,
                          "uid": os.getuid(), "gid": os.getgid(), "rdev": 7},
                "character": {"type": "character", "mode": 0o600,
                              "uid": os.getuid(), "gid": os.getgid(), "rdev": 8},
                "directory": {"type": "directory", "mode": 0o700,
                              "uid": os.getuid(), "gid": os.getgid()},
                "fifo": {"type": "fifo", "mode": 0o600,
                         "uid": os.getuid(), "gid": os.getgid()},
                "file": {"type": "file", "mode": 0o600,
                         "uid": os.getuid(), "gid": os.getgid(),
                         "nlink": 1, "sha256": "0" * 64},
                "link": {"type": "symlink", "mode": 0o777,
                         "uid": os.getuid(), "gid": os.getgid(), "target": "target"},
                "socket": {"type": "socket", "mode": 0o600,
                           "uid": os.getuid(), "gid": os.getgid()},
            }, "outside manifest tuple semantics")

    tuple_mutants = [
        ("outer", {}, 0, "outside manifest row cardinality"),
        ("directory-width", [["path", "directory", 0o700, 1]], 1,
         "outside manifest record width"),
        ("file-width", [["path", "file", 0o600, 1, 1, 1]], 1,
         "outside manifest record width"),
        ("symlink-width", [["path", "symlink", 0o777, 1, 1]], 1,
         "outside manifest record width"),
        ("block-width", [["path", "block", 0o600, 1, 1]], 1,
         "outside manifest record width"),
        ("character-width", [["path", "character", 0o600, 1, 1]], 1,
         "outside manifest record width"),
        ("block-rdev-bool", [["path", "block", 0o600, 1, 1, True]], 1,
         "outside manifest device identity"),
        ("block-rdev-negative", [["path", "block", 0o600, 1, 1, -1]], 1,
         "outside manifest device identity"),
        ("character-rdev-bool", [["path", "character", 0o600, 1, 1, True]], 1,
         "outside manifest device identity"),
        ("character-rdev-string", [["path", "character", 0o600, 1, 1, "8"]], 1,
         "outside manifest device identity"),
        ("former-other-type", [["path", "other", 0o600, 1, 1]], 1,
         "outside manifest record type"),
        ("path-type", [[1, "directory", 0o700, 1, 1]], 1,
         "outside manifest row path"),
        ("path-grammar", [["../path", "directory", 0o700, 1, 1]], 1,
         "descriptor-relative components"),
        ("record-type", [["path", "unknown", 0o600, 1, 1]], 1,
         "outside manifest record type"),
        ("mode-type", [["path", "directory", True, 1, 1]], 1,
         "outside manifest record mode"),
        ("uid-type", [["path", "directory", 0o700, True, 1]], 1,
         "outside manifest record uid"),
        ("gid-type", [["path", "directory", 0o700, 1, True]], 1,
         "outside manifest record gid"),
        ("nlink-type", [["path", "file", 0o600, 1, 1, True, "0" * 64]], 1,
         "outside manifest file identity"),
        ("target-type", [["path", "symlink", 0o777, 1, 1, 1]], 1,
         "outside manifest symlink target"),
        ("hash", [["path", "file", 0o600, 1, 1, 1, "bad"]], 1,
         "outside manifest file identity"),
        ("order", [["z", "directory", 0o700, 1, 1],
                   ["a", "directory", 0o700, 1, 1]], 2,
         "outside manifest row order"),
        ("duplicate", [["path", "directory", 0o700, 1, 1],
                       ["path", "directory", 0o700, 1, 1]], 2,
         "outside manifest row order"),
    ]
    for label, value, rows, diagnostic in tuple_mutants:
        raw = encode_fixture(value)
        try:
            parse_outside_manifest(raw, fixture_binding(raw, rows))
        except AssertionError as error:
            require(str(error) == diagnostic,
                    f"outside manifest tuple {label} diagnostic")
        else:
            fail(f"outside manifest tuple {label} mutant survived")

    wrong_binding = fixture_binding(valid_raw, 2)
    try:
        parse_outside_manifest(valid_raw, wrong_binding)
    except AssertionError as error:
        require(str(error) == "outside manifest row cardinality",
                "outside manifest tuple binding diagnostic")
    else:
        fail("outside manifest tuple binding mutant survived")

    noncanonical = encode_fixture(valid_rows, canonical=False)
    try:
        parse_outside_manifest(noncanonical, fixture_binding(noncanonical, 7))
    except AssertionError as error:
        require(str(error) == "outside manifest canonical encoding",
                "outside manifest tuple canonical-byte diagnostic")
    else:
        fail("outside manifest tuple canonical-byte mutant survived")

    try:
        encode_outside_manifest({
            "path": {"type": "other", "mode": 0o600,
                     "uid": os.getuid(), "gid": os.getgid()},
        })
    except AssertionError as error:
        require(str(error) == "outside manifest record type",
                "outside manifest encoder rejects former other type")
    else:
        fail("outside manifest encoder accepted former other type")

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve()
        fifo = root / "fifo"
        socket_path = root / "socket"
        os.mkfifo(fifo, 0o600)
        unix_socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        unix_socket.bind(str(socket_path))
        os.chmod(socket_path, 0o600)
        root_fd = open_absolute_directory(root)
        original_open = os.open
        special_opens = []
        replacement_socket = None
        def reject_special_open(name, *args, **kwargs):
            if name in {"fifo", "socket"}:
                special_opens.append(name)
                fail("outside manifest attempted to open special object")
            return original_open(name, *args, **kwargs)
        os.open = reject_special_open
        try:
            rows = manifest_at(root_fd, set(), "__excluded__")
            common = {"mode": 0o600, "uid": os.getuid(), "gid": os.getgid()}
            require(rows == {
                        "fifo": {"type": "fifo", **common},
                        "socket": {"type": "socket", **common},
                    }, "outside manifest FIFO and socket records")
            require(parse_outside_manifest(
                        encode_outside_manifest(rows),
                        outside_manifest_binding(rows, encode_outside_manifest(rows)),
                    ) == rows, "outside manifest FIFO and socket round trip")

            os.unlink(fifo)
            replacement_socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            replacement_socket.bind(str(fifo))
            os.chmod(fifo, 0o600)
            replaced_rows = manifest_at(root_fd, set(), "__excluded__")
            require({key: value for key, value in rows["fifo"].items() if key != "type"}
                    == {key: value for key, value in replaced_rows["fifo"].items()
                        if key != "type"}
                    and rows["fifo"]["type"] == "fifo"
                    and replaced_rows["fifo"]["type"] == "socket"
                    and rows != replaced_rows,
                    "outside manifest FIFO-to-socket exact identity")
            require(special_opens == [], "outside manifest special objects require no open")
        finally:
            os.open = original_open
            os.close(root_fd)
            unix_socket.close()
            if replacement_socket is not None:
                replacement_socket.close()

    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve()
        (root / "one").mkdir(); (root / "two").mkdir()
        root_fd = open_absolute_directory(root)
        saved_entries = OUTSIDE_MANIFEST_MAX_ENTRIES
        saved_bytes = OUTSIDE_MANIFEST_MAX_BYTES
        saved_timeout = OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
        try:
            globals()["OUTSIDE_MANIFEST_MAX_ENTRIES"] = 1
            try:
                manifest_at(root_fd, set(), "__excluded__")
            except AssertionError as error:
                require(str(error) == "outside manifest entry bound",
                        "outside manifest entry-bound diagnostic")
            else:
                fail("outside manifest entry-bound omission mutant survived")
            globals()["OUTSIDE_MANIFEST_MAX_ENTRIES"] = saved_entries
            globals()["OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS"] = -1
            try:
                manifest_at(root_fd, set(), "__excluded__")
            except AssertionError as error:
                require(str(error) == "outside manifest scan-time bound",
                        "outside manifest scan-time diagnostic")
            else:
                fail("outside manifest scan-time omission mutant survived")
            globals()["OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS"] = saved_timeout
            globals()["OUTSIDE_MANIFEST_MAX_BYTES"] = 1
            try:
                encode_outside_manifest({
                    "one": {"type": "directory", "mode": 0o700,
                            "uid": os.getuid(), "gid": os.getgid()},
                })
            except AssertionError as error:
                require(str(error) == "outside manifest encoded-byte bound",
                        "outside manifest encoded-byte diagnostic")
            else:
                fail("outside manifest encoded-byte omission mutant survived")
        finally:
            globals()["OUTSIDE_MANIFEST_MAX_ENTRIES"] = saved_entries
            globals()["OUTSIDE_MANIFEST_MAX_BYTES"] = saved_bytes
            globals()["OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS"] = saved_timeout
            os.close(root_fd)


def check_outside_manifest_hard_wall() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp).resolve(); atomic_write(root / "file", b"payload\n", 0o600)
        root_fd = open_absolute_directory(root)
        saved_timeout = OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
        original_listdir = os.listdir
        original_read = os.read
        original_dumps = json.dumps
        original_timer = outside_manifest_wall_timer
        try:
            globals()["OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS"] = 0.02
            for label, install, restore, operation in (
                ("list", lambda completed: setattr(
                    os, "listdir",
                    lambda fd: (time.sleep(0.2), completed.append("list"),
                                original_listdir(fd))[2],
                 ), lambda: setattr(os, "listdir", original_listdir),
                 lambda: manifest_at(root_fd, set(), "__excluded__")),
                ("hash", lambda completed: setattr(
                    os, "read",
                    lambda fd, size: (time.sleep(0.2), completed.append("hash"),
                                      original_read(fd, size))[2],
                 ), lambda: setattr(os, "read", original_read),
                 lambda: manifest_at(root_fd, set(), "__excluded__")),
                ("encode", lambda completed: setattr(
                    json, "dumps",
                    lambda *args, **kwargs: (
                        time.sleep(0.2), completed.append("encode"),
                        original_dumps(*args, **kwargs),
                    )[2],
                 ), lambda: setattr(json, "dumps", original_dumps),
                 lambda: encode_outside_manifest({})),
            ):
                completed = []
                install(completed)
                try:
                    operation()
                except AssertionError as error:
                    require(str(error) == "outside manifest scan-time bound",
                            f"outside manifest blocked {label} diagnostic")
                else:
                    fail(f"outside manifest blocked {label} accepted")
                finally:
                    restore()
                require(completed == [],
                        f"outside manifest blocked {label} operation interrupted")

            globals()["outside_manifest_wall_timer"] = lambda _seconds: contextlib.nullcontext()
            completed = []
            setattr(json, "dumps", lambda *args, **kwargs: (
                time.sleep(0.2), completed.append("encode"),
                original_dumps(*args, **kwargs),
            )[2])
            try:
                encode_outside_manifest({})
            except AssertionError as error:
                require(str(error) == "outside manifest scan-time bound",
                        "outside manifest wall omission diagnostic")
            else:
                fail("outside manifest hard-wall omission mutant survived")
            require(completed == ["encode"],
                    "disabled outside wall timer permits delayed operation completion")

            json.dumps = original_dumps
            globals()["outside_manifest_wall_timer"] = original_timer
            baseline_handler = signal.getsignal(signal.SIGALRM)
            def prior_handler(_signum, _frame):
                pass
            signal.signal(signal.SIGALRM, prior_handler)
            signal.setitimer(signal.ITIMER_REAL, 30)
            try:
                encode_outside_manifest({})
                remaining = signal.setitimer(signal.ITIMER_REAL, 0)[0]
                require(signal.getsignal(signal.SIGALRM) is prior_handler
                        and remaining > 0,
                        "outside manifest prior wall timer restoration")
            finally:
                signal.setitimer(signal.ITIMER_REAL, 0)
                signal.signal(signal.SIGALRM, baseline_handler)
        finally:
            globals()["OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS"] = saved_timeout
            globals()["outside_manifest_wall_timer"] = original_timer
            os.listdir = original_listdir; os.read = original_read; json.dumps = original_dumps
            os.close(root_fd)


def sidecar_adversary_fixture() -> tuple[tempfile.TemporaryDirectory, Path, Path, bytes, list[dict]]:
    temporary = tempfile.TemporaryDirectory()
    repo = Path(temporary.name).resolve() / "repo"; create_actual_delivery_repo(repo)
    atomic_write(repo / ".outside-manifest-marker", b"outside marker\n", 0o600)
    require(run_actual_delivery_mode(repo, ["--delivery-prepare"]).returncode == 0,
            "outside sidecar adversary prepare")
    delivery = actual_delivery_directory(repo)
    journal_raw = (delivery / "journal.json").read_bytes()
    sidecar_raw = (delivery / OUTSIDE_MANIFEST_NAME).read_bytes()
    journal = json.loads(journal_raw)
    require(set(journal["outside_manifest"]) == {"relative", "sha256", "bytes", "rows"}
            and len(journal_raw) <= DELIVERY_JOURNAL_MAX_BYTES
            and b".outside-manifest-marker" not in journal_raw
            and journal["outside_manifest"]["bytes"] == len(sidecar_raw)
            and journal["journal_sequence"] > len(AUTHORED) + len(GENERATED),
            "small journal binds one nonduplicated outside sidecar")
    return temporary, repo, delivery, sidecar_raw, observe_delivery_destinations(repo)


def reject_sidecar_adversary(repo: Path, delivery: Path,
                             destinations: list[dict], label: str) -> None:
    journal_path = delivery / "journal.json"
    durable = journal_path.read_bytes()
    rejected = run_actual_delivery_mode(repo, ["--delivery-recover", str(delivery)])
    require(rejected.returncode != 0 and journal_path.read_bytes() == durable
            and observe_delivery_destinations(repo) == destinations,
            f"outside sidecar adversary rejected without writes: {label}")


def check_outside_manifest_sidecar_metadata() -> None:
    temporary, repo, delivery, sidecar_raw, before = sidecar_adversary_fixture()
    try:
        sidecar = delivery / OUTSIDE_MANIFEST_NAME
        sidecar.unlink(); reject_sidecar_adversary(repo, delivery, before, "missing")
        atomic_write(sidecar, sidecar_raw, 0o600)
        sidecar.unlink(); os.symlink("journal.json", sidecar)
        reject_sidecar_adversary(repo, delivery, before, "symlink")
        sidecar.unlink(); atomic_write(sidecar, sidecar_raw, 0o600)
        sidecar.unlink(); sidecar.mkdir(mode=0o700)
        reject_sidecar_adversary(repo, delivery, before, "wrong-type")
        sidecar.rmdir(); atomic_write(sidecar, sidecar_raw, 0o600)
        sidecar.chmod(0o644)
        reject_sidecar_adversary(repo, delivery, before, "wrong-mode")
    finally:
        temporary.cleanup()


def check_outside_manifest_sidecar_integrity() -> None:
    temporary, repo, delivery, sidecar_raw, before = sidecar_adversary_fixture()
    try:
        sidecar = delivery / OUTSIDE_MANIFEST_NAME
        sidecar.write_bytes(sidecar_raw + b"x"); sidecar.chmod(0o600)
        reject_sidecar_adversary(repo, delivery, before, "wrong-hash")
        atomic_write(sidecar, sidecar_raw, 0o600)
        os.link(sidecar, delivery / "outside-manifest-link")
        reject_sidecar_adversary(repo, delivery, before, "nlink")
        (delivery / "outside-manifest-link").unlink()
        atomic_write(delivery / "unknown-private-sibling", b"unknown\n", 0o600)
        reject_sidecar_adversary(repo, delivery, before, "unknown-sibling")
    finally:
        temporary.cleanup()


def check_prejournal_sidecar_reuse() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; before = create_actual_delivery_repo(repo)
        killed = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], "bootstrap-sidecar-before-rename",
        )
        require(killed.returncode == 89, "pre-journal sidecar death setup")
        delivery = actual_delivery_directory(repo)
        require((delivery / OUTSIDE_MANIFEST_TEMPORARY).is_file()
                and not (delivery / OUTSIDE_MANIFEST_NAME).exists()
                and not (delivery / "journal.json").exists(),
                "pre-journal sidecar temporary retained")
        atomic_write(repo / ".post-sidecar-drift", b"drift\n", 0o600)
        rejected = run_actual_delivery_mode(repo, ["--delivery-prepare"])
        require(rejected.returncode != 0
                and "atomic temporary prefix" in rejected.stdout
                and not (delivery / "journal.json").exists(),
                "stale pre-journal sidecar reuse mutant rejected")
        assert_actual_delivery_records(repo, before)


def check_bootstrap_sidecar_temporary_adversaries() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        killed = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], "bootstrap-sidecar-after-create",
        )
        require(killed.returncode == 86, "bootstrap sidecar adversary setup")
        delivery = actual_delivery_directory(repo)
        temporary = delivery / OUTSIDE_MANIFEST_TEMPORARY
        physical = observe_delivery_destination_physical_identity(repo)

        def reject(label: str) -> None:
            rejected = run_actual_delivery_mode(repo, ["--delivery-prepare"])
            require(rejected.returncode != 0 and not (delivery / "journal.json").exists()
                    and observe_delivery_destination_physical_identity(repo) == physical,
                    f"bootstrap sidecar temporary rejected: {label}")

        temporary.chmod(0o644); reject("mode"); temporary.chmod(0o600)
        temporary.write_bytes(b"not-a-prefix"); temporary.chmod(0o600)
        reject("prefix"); atomic_write(temporary, b"", 0o600)
        temporary.unlink(); os.symlink(".owner.lock", temporary)
        reject("symlink"); temporary.unlink(); atomic_write(temporary, b"", 0o600)
        temporary.unlink(); temporary.mkdir(mode=0o700)
        reject("type"); temporary.rmdir(); atomic_write(temporary, b"", 0o600)
        with temporary.open("r+b") as stream:
            stream.truncate(OUTSIDE_MANIFEST_MAX_BYTES + 1)
        reject("size"); atomic_write(temporary, b"", 0o600)
        os.link(temporary, repo / ".bootstrap-sidecar-link")
        reject("nlink"); (repo / ".bootstrap-sidecar-link").unlink()
        lookalike = delivery / (OUTSIDE_MANIFEST_TEMPORARY + ".lookalike")
        atomic_write(lookalike, b"", 0o600); reject("lookalike"); lookalike.unlink()

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        killed = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], "bootstrap-journal-after-create",
        )
        require(killed.returncode == 86, "bootstrap sidecar ambiguity setup")
        delivery = actual_delivery_directory(repo)
        atomic_write(delivery / OUTSIDE_MANIFEST_TEMPORARY, b"", 0o600)
        rejected = run_actual_delivery_mode(repo, ["--delivery-prepare"])
        require(rejected.returncode != 0
                and "outside manifest final temporary ambiguity" in rejected.stdout,
                "bootstrap sidecar final temporary ambiguity")


def check_bootstrap_journal_temporary_adversaries() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        killed = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], "bootstrap-journal-after-create",
        )
        require(killed.returncode == 86, "bootstrap journal adversary setup")
        delivery = actual_delivery_directory(repo)
        temporary = delivery / JOURNAL_TEMPORARY
        physical = observe_delivery_destination_physical_identity(repo)

        def reject(label: str) -> None:
            rejected = run_actual_delivery_mode(repo, ["--delivery-prepare"])
            require(rejected.returncode != 0 and not (delivery / "journal.json").exists()
                    and observe_delivery_destination_physical_identity(repo) == physical,
                    f"bootstrap journal temporary rejected: {label}")

        temporary.chmod(0o644); reject("mode"); temporary.chmod(0o600)
        temporary.write_bytes(b"not-a-prefix"); temporary.chmod(0o600)
        reject("prefix"); atomic_write(temporary, b"", 0o600)
        temporary.unlink(); os.symlink(".owner.lock", temporary)
        reject("symlink"); temporary.unlink(); atomic_write(temporary, b"", 0o600)
        temporary.unlink(); temporary.mkdir(mode=0o700)
        reject("type"); temporary.rmdir(); atomic_write(temporary, b"", 0o600)
        with temporary.open("r+b") as stream:
            stream.truncate(DELIVERY_JOURNAL_MAX_BYTES + 1)
        reject("size"); atomic_write(temporary, b"", 0o600)
        os.link(temporary, repo / ".bootstrap-journal-link")
        reject("nlink"); (repo / ".bootstrap-journal-link").unlink()
        atomic_write(temporary, b"{}\n", 0o600); reject("complete-intent")
        atomic_write(temporary, b'{"unterminated":\n', 0o600); reject("malformed-complete")
        atomic_write(temporary, b'{"journal_sequence":1,"journal_sequence":1}\n', 0o600)
        reject("duplicate-complete")
        atomic_write(temporary, b"{}\ntrailing", 0o600); reject("trailing-bytes")
        atomic_write(temporary, b"", 0o600)
        lookalike = delivery / (JOURNAL_TEMPORARY + ".lookalike")
        atomic_write(lookalike, b"", 0o600); reject("lookalike")

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        killed = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], "bootstrap-sidecar-after-create",
        )
        require(killed.returncode == 86, "impossible bootstrap inventory setup")
        delivery = actual_delivery_directory(repo)
        os.rename(
            delivery / OUTSIDE_MANIFEST_TEMPORARY,
            delivery / JOURNAL_TEMPORARY,
        )
        physical = observe_delivery_destination_physical_identity(repo)
        rejected = run_actual_delivery_mode(repo, ["--delivery-prepare"])
        require(rejected.returncode != 0
                and "initial journal temporary requires outside manifest" in rejected.stdout
                and not (delivery / OUTSIDE_MANIFEST_NAME).exists()
                and observe_delivery_destination_physical_identity(repo) == physical,
                "impossible bootstrap inventory rejected before sidecar creation")


def check_initial_journal_strict_prefix_recovery() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"; create_actual_delivery_repo(repo)
        killed = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], "bootstrap-journal-before-rename",
        )
        require(killed.returncode == 89, "strict-prefix journal setup")
        delivery = actual_delivery_directory(repo)
        temporary = delivery / JOURNAL_TEMPORARY
        canonical = temporary.read_bytes()
        complete = parse_journal(canonical)
        expected = journal_body(complete)
        timestamp = str(complete["updated_ns"]).encode()
        timestamp_start = canonical.index(b'"updated_ns":') + len(b'"updated_ns":')
        timestamp_end = timestamp_start + len(timestamp)
        offsets = [
            ("empty", 0),
            ("first-byte", 1),
            ("before-timestamp", timestamp_start),
            ("inside-timestamp", timestamp_start + max(1, len(timestamp) // 2)),
            ("after-timestamp", timestamp_end),
            ("post-timestamp-body", timestamp_end + (len(canonical) - timestamp_end) // 2),
            ("final-byte-minus-one", len(canonical) - 1),
        ]
        physical = observe_delivery_destination_physical_identity(repo)
        repo_fd = open_absolute_directory(repo)
        delivery_fd = open_absolute_directory(delivery)
        lock_fd = delivery_lock(delivery_fd)
        try:
            for offset in range(len(canonical)):
                require(initial_journal_strict_prefix(canonical[:offset], expected),
                        f"initial journal exhaustive strict prefix: {offset}")
            require(not initial_journal_strict_prefix(canonical, expected),
                    "complete initial journal is not a strict prefix")
            future = dict(complete)
            future["updated_ns"] = time.time_ns() + 10**15
            future_raw = (json.dumps(
                future, sort_keys=True, separators=(",", ":"),
            ) + "\n").encode()
            future_timestamp_tag = b'\"updated_ns\":'
            future_timestamp_start = (
                future_raw.index(future_timestamp_tag) + len(future_timestamp_tag)
            )
            future_prefix = future_raw[:future_timestamp_start + 2]
            require(initial_journal_strict_prefix(future_prefix, expected),
                    "future initial journal strict prefix")
            atomic_write(temporary, future_prefix, 0o600)
            require(reconcile_initial_journal_temporary(
                        delivery_fd, repo_fd, delivery.name, expected,
                    ) is None and not temporary.exists(),
                    "future initial journal prefix reconciled after clock rollback")

            for label, offset in offsets:
                atomic_write(temporary, canonical[:offset], 0o600)
                recovered = reconcile_initial_journal_temporary(
                    delivery_fd, repo_fd, delivery.name, expected,
                )
                require(recovered is None and not temporary.exists(),
                        f"strict-prefix reconstruction requested: {label}")
                fresh = dict(expected)
                write_journal(delivery_fd, lock_fd, fresh, bootstrap=True)
                journal = validate_delivery_journal(
                    read_journal(delivery_fd), delivery_fd, repo_fd, delivery.name,
                )
                require(journal_body(journal) == expected
                        and journal["journal_sequence"] == 1
                        and journal["state"] == "SNAPSHOTTING"
                        and journal["authored"] == [] and journal["generated"] == []
                        and observe_delivery_destination_physical_identity(repo) == physical,
                        f"strict-prefix safe initial state: {label}")
                os.unlink("journal.json", dir_fd=delivery_fd)
                os.fsync(delivery_fd)

            for label, offset in offsets:
                prefix = canonical[:offset]
                if prefix:
                    replacement = b"!" if prefix[-1:] != b"!" else b"?"
                    mutant = prefix[:-1] + replacement
                else:
                    mutant = b"!"
                atomic_write(temporary, mutant, 0o600)
                try:
                    reconcile_initial_journal_temporary(
                        delivery_fd, repo_fd, delivery.name, expected,
                    )
                except AssertionError as error:
                    require(str(error) == "initial journal temporary partial prefix",
                            f"strict-prefix mismatch diagnostic: {label}")
                else:
                    fail(f"strict-prefix mismatch survived: {label}")
                require(not (delivery / "journal.json").exists()
                        and observe_delivery_destination_physical_identity(repo) == physical,
                        f"strict-prefix mismatch zero writes: {label}")
                temporary.unlink()
        finally:
            os.close(lock_fd); os.close(delivery_fd); os.close(repo_fd)


def check_delivery_journal_adversaries() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        expected_records = create_actual_delivery_repo(repo)
        prepared = run_actual_delivery_mode(repo, ["--delivery-prepare"])
        require(prepared.returncode == 0, f"journal adversary prepare: {prepared.stdout[-1000:]}")
        delivery = actual_delivery_directory(repo)
        repo_fd = open_absolute_directory(repo)
        delivery_fd = open_absolute_directory(delivery)
        lock_fd = delivery_lock(delivery_fd)
        try:
            journal = validate_delivery_journal(
                read_journal(delivery_fd), delivery_fd, repo_fd, delivery.name,
                require_complete=True,
            )
            mutants = []
            for field in sorted(DELIVERY_JOURNAL_FIELDS):
                mutant = json.loads(json.dumps(journal))
                del mutant[field]
                mutants.append((f"missing-{field}", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["unknown"] = "x"; mutants.append(("unknown-field", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["contract"] += ".forged"; mutants.append(("contract", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["state"] = "INSTALLING(23)"; mutants.append(("state-index", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["candidate_digest"] = "g" * 64; mutants.append(("digest", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["candidate_root"] = str(repo); mutants.append(("candidate-root", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["authored_allowlist"] = list(reversed(AUTHORED)); mutants.append(("allowlist", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["protected"] = {}; mutants.append(("protected", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["outside_manifest"] = {}; mutants.append(("outside", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["authored"][0]["path"] = GENERATED[0]; mutants.append(("snapshot-path", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["authored"][0]["index"] = 1; mutants.append(("snapshot-index", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["authored"][0]["backup"] = "backups/authored/ffffffff"; mutants.append(("snapshot-backup", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["authored"][0]["mode"] = True; mutants.append(("snapshot-mode", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["authored"][0]["sha256"] = "0" * 64; mutants.append(("snapshot-hash", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["installed_authored"] = True; mutants.append(("counter-type", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["installed_generated"] = 1; mutants.append(("counter-relation", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["state"] = "STAGED"; mutants.append(("missing-stage-manifest", mutant))
            mutant = json.loads(json.dumps(journal)); mutant["gates"] = [{"command": ["true"], "execution_prefix": ["rtk", "proxy"], "sha256": "0" * 64, "signal": "exit=0"}]; mutants.append(("gate-command", mutant))
            for label, mutant in mutants:
                try:
                    validate_delivery_journal(
                        mutant, delivery_fd, repo_fd, delivery.name,
                    )
                except (AssertionError, KeyError, TypeError):
                    pass
                else:
                    fail(f"forged delivery journal accepted: {label}")
            atomic_write_at(delivery_fd, "backups/authored/ffffffff", b"unknown", 0o600, lock_fd)
            try:
                validate_delivery_journal(
                    journal, delivery_fd, repo_fd, delivery.name,
                )
            except AssertionError:
                pass
            else:
                fail("unknown delivery backup accepted")
            unlink_file_at(delivery_fd, "backups/authored/ffffffff")
            backup = read_file_at(delivery_fd, "backups/authored/00000000", 16 * 1024 * 1024, {0o600})
            unlink_file_at(delivery_fd, "backups/authored/00000000")
            try:
                validate_delivery_journal(
                    journal, delivery_fd, repo_fd, delivery.name,
                )
            except (AssertionError, OSError):
                pass
            else:
                fail("missing delivery backup accepted")
            atomic_write_at(delivery_fd, "backups/authored/00000000", backup, 0o600, lock_fd)
            original = read_file_at(
                delivery_fd, "journal.json", DELIVERY_JOURNAL_MAX_BYTES, {0o600},
            )
            atomic_write_at(delivery_fd, "journal.json", b'{"contract":1,"contract":2}\n', 0o600, lock_fd)
            try:
                read_journal(delivery_fd)
            except AssertionError:
                pass
            else:
                fail("duplicate delivery journal field accepted")
            atomic_write_at(delivery_fd, "journal.json", original, 0o600, lock_fd)
            validate_delivery_journal(
                read_journal(delivery_fd), delivery_fd, repo_fd, delivery.name,
                require_complete=True,
            )
        finally:
            os.close(lock_fd); os.close(delivery_fd); os.close(repo_fd)
        candidate_fd = open_absolute_directory(canonical_root())
        try:
            installed_records = {relative: file_record_at(candidate_fd, relative) for relative in AUTHORED}
        finally:
            os.close(candidate_fd)
        installed_records.update({relative: expected_records[relative] for relative in GENERATED})
        assert_actual_delivery_records(repo, installed_records)


def delivery_fixture(root: Path) -> tuple[list[str], dict[str, bytes | None], dict[str, bytes]]:
    paths = [f"authored/{index:02d}" for index in range(16)] + [f"generated/{index:02d}" for index in range(22)]
    before = {path: (None if index % 5 == 0 else f"old-{index}".encode()) for index, path in enumerate(paths)}
    desired = {path: f"new-{index}".encode() for index, path in enumerate(paths)}
    for path, data in before.items():
        destination = root / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        if data is not None:
            atomic_write(destination, data, 0o600)
    return paths, before, desired


def assert_delivery_state(root: Path, values: dict[str, bytes | None]) -> None:
    for path, data in values.items():
        destination = root / path
        actual = destination.read_bytes() if destination.exists() else None
        require(actual == data, f"delivery identity: {path}")


def run_delivery_model(fault: str | None, owner: str = "wright", extra_stage: bool = False) -> tuple[str, list[str]]:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp) / "repo"; root.mkdir()
        paths, before, desired = delivery_fixture(root)
        journal = []
        require(owner == "wright", "root install rejected")
        snapshots = {}
        try:
            for index, path in enumerate(paths):
                journal.append(f"SNAPSHOTTING({index})")
                snapshots[path] = before[path]
                fault_at(fault, f"snapshot-{index}-before")
                fault_at(fault, f"snapshot-{index}-after")
            stage = Path(tmp) / "stage"; stage.mkdir()
            for path, data in desired.items():
                staged = stage / path; staged.parent.mkdir(parents=True, exist_ok=True); atomic_write(staged, data, 0o600)
            if extra_stage:
                atomic_write(stage / "generated/unknown", b"unknown", 0o600)
            staged_paths = {path.relative_to(stage).as_posix() for path in stage.rglob("*") if path.is_file()}
            require(staged_paths == set(paths), "delivery staged allowlist")
            journal.append("STAGED"); fault_at(fault, "stage-before"); fault_at(fault, "stage-after")
            for index, path in enumerate(paths[:16]):
                journal.append(f"INSTALLING-AUTHORED({index})"); fault_at(fault, f"authored-{index}-before")
                atomic_write(root / path, desired[path], 0o600); fault_at(fault, f"authored-{index}-after")
            for index, path in enumerate(paths[16:]):
                journal.append(f"INSTALLING({index + 1})"); fault_at(fault, f"generated-{index}-before")
                atomic_write(root / path, desired[path], 0o600); fault_at(fault, f"generated-{index}-after")
            journal.extend(["INSTALLED", "PROVING"]); fault_at(fault, "prove-before"); fault_at(fault, "prove-after")
            journal.append("COMMITTED"); fault_at(fault, "commit-after")
            journal.extend(["CLEANING", "CLEANED"]); fault_at(fault, "cleanup-after")
            assert_delivery_state(root, desired)
            return "CLEANED", journal
        except (RuntimeError, AssertionError):
            if "COMMITTED" in journal:
                assert_delivery_state(root, desired)
                return "CLEANED", journal + ["CLEANING", "CLEANED"]
            for position, path in enumerate(reversed(paths), 1):
                journal.append(f"ROLLING_BACK({position})")
                data = snapshots.get(path, before[path])
                destination = root / path
                if data is None:
                    destination.unlink(missing_ok=True)
                else:
                    atomic_write(destination, data, 0o600)
            journal.extend(["RESTORED", "FAILED"])
            assert_delivery_state(root, before)
            return "FAILED", journal


def check_delivery_model_matrix() -> None:
    faults = ["stage-before", "stage-after", "prove-before", "prove-after", "commit-after", "cleanup-after"]
    faults.extend(f"snapshot-{index}-{side}" for index in range(38) for side in ("before", "after"))
    faults.extend(f"authored-{index}-{side}" for index in range(16) for side in ("before", "after"))
    faults.extend(f"generated-{index}-{side}" for index in range(22) for side in ("before", "after"))

    def verify_fault(fault: str) -> None:
        state, journal = run_delivery_model(fault)
        if fault in {"commit-after", "cleanup-after"}:
            require(state == "CLEANED" and "RESTORED" not in journal, f"post-commit cleanup only: {fault}")
        else:
            require(state == "FAILED" and journal[-2:] == ["RESTORED", "FAILED"], f"pre-commit restore: {fault}")

    with ThreadPoolExecutor(max_workers=delivery_parallel_workers()) as executor:
        list(executor.map(verify_fault, faults))
    state, _ = run_delivery_model(None)
    require(state == "CLEANED", "delivery success")
    state, journal = run_delivery_model(None, extra_stage=True)
    require(state == "FAILED" and "COMMITTED" not in journal, "staged sibling rejection")
    try:
        run_delivery_model(None, owner="root")
    except AssertionError:
        pass
    else:
        fail("root delivery ownership accepted")


def check_journal_retry_stability() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        delivery_fd = os.open(tmp, DIRECTORY_FLAGS)
        lock_fd = delivery_lock(delivery_fd)
        try:
            journal = {"contract": "journal-retry-fixture", "state": "FIRST"}
            before = time.time_ns()
            write_journal(delivery_fd, lock_fd, journal)
            first_bytes = read_file_at(delivery_fd, "journal.json", 4096, {0o600})
            first = parse_journal(first_bytes)
            require(before <= first["updated_ns"] <= time.time_ns()
                    and first["journal_sequence"] == 1,
                    "journal first-write timestamp and sequence")

            journal["state"] = "SECOND"
            write_journal(delivery_fd, lock_fd, journal)
            second = read_journal(delivery_fd)
            require(second["updated_ns"] == first["updated_ns"] + 1
                    and second["journal_sequence"] == first["journal_sequence"] + 1,
                    "journal deterministic field advancement")

            retry = dict(second)
            retry["state"] = "THIRD"
            expected = dict(retry)
            expected["updated_ns"] += 1
            expected["journal_sequence"] += 1
            expected_bytes = (json.dumps(
                expected, sort_keys=True, separators=(",", ":"),
            ) + "\n").encode()
            child = os.fork()
            if child == 0:
                _DELIVERY_TEST["death_boundary"] = "journal-replacement-after-sync"
                write_journal(delivery_fd, lock_fd, retry)
                os._exit(99)
            _pid, status = os.waitpid(child, 0)
            require(os.waitstatus_to_exitcode(status) == 87,
                    "journal after-sync fixture death")
            temporary = JOURNAL_TEMPORARY
            require(read_file_at(delivery_fd, temporary, 4096, {0o600}) == expected_bytes,
                    "journal synced temporary intended bytes")

            retried = read_journal(delivery_fd)
            retried["state"] = "THIRD"
            write_journal(delivery_fd, lock_fd, retried)
            require(read_file_at(delivery_fd, "journal.json", 4096, {0o600}) == expected_bytes
                    and entry_info_at(delivery_fd, temporary) is None,
                    "journal after-sync retry exact bytes")
        finally:
            os.close(lock_fd); os.close(delivery_fd)


def check_cleanup_recovery() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        delivery_fd = os.open(tmp, DIRECTORY_FLAGS)
        lock_fd = delivery_lock(delivery_fd)
        journal = {"contract": "devrites.workflow-artifact-delivery.v1", "state": "COMMITTED"}
        try:
            write_journal(delivery_fd, lock_fd, journal)
            atomic_write_at(delivery_fd, "stage/blocked/file", b"stage", 0o600, lock_fd)
            atomic_write_at(delivery_fd, "backups/generated/00000000", b"backup", 0o600, lock_fd)
            blocked_fd = open_dir_components(delivery_fd, ("stage", "blocked"))
            try:
                os.fchmod(blocked_fd, 0o500)
            finally:
                os.close(blocked_fd)
            try:
                cleanup_delivery(delivery_fd, lock_fd, journal)
            except PermissionError:
                pass
            else:
                fail("real cleanup failure suppressed")
            failed = read_journal(delivery_fd)
            require(failed["state"] == "CLEANING" and entry_info_at(delivery_fd, "stage") is not None
                    and entry_info_at(delivery_fd, "backups") is not None, "failed cleanup remains resumable")
            blocked_fd = open_dir_components(delivery_fd, ("stage", "blocked"))
            try:
                os.fchmod(blocked_fd, 0o700)
            finally:
                os.close(blocked_fd)
            cleanup_delivery(delivery_fd, lock_fd, failed)
            cleaned = read_journal(delivery_fd)
            require(cleaned["state"] == "CLEANED" and entry_info_at(delivery_fd, "stage") is None
                    and entry_info_at(delivery_fd, "backups") is None, "cleanup recovery verifies absence before CLEANED")
        finally:
            os.close(lock_fd); os.close(delivery_fd)


def check_gate_descriptor_ancestor_replacement() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(tmp)
        lexical = base / "repo"
        held = base / "held"
        lexical.mkdir()
        repo_fd = open_absolute_directory(lexical.resolve())
        delivery_fd, proof_cache_relative = _gate_test_delivery(lexical.resolve())
        try:
            os.rename(lexical, held)
            lexical.mkdir()
            result = run_gate(
                repo_fd, delivery_fd, proof_cache_relative,
                [sys.executable, "-c", "from pathlib import Path;Path('gate-marker').write_text('held');print('GATE HELD PASS')"],
                "GATE HELD PASS", time.monotonic() + DELIVERY_AGGREGATE_TIMEOUT_SECONDS, 0,
            )
            require(result["signal"] == "GATE HELD PASS"
                    and (held / "gate-marker").read_text() == "held"
                    and not (lexical / "gate-marker").exists(),
                    "descriptor-held gate ancestor replacement")
        finally:
            os.close(delivery_fd); os.close(repo_fd)
        timeout_root = base / "timeout"
        timeout_root.mkdir()
        timeout_fd = open_absolute_directory(timeout_root.resolve())
        timeout_delivery_fd, timeout_proof_cache = _gate_test_delivery(timeout_root.resolve())
        saved_timeout = DELIVERY_PROCESS_TIMEOUT_SECONDS
        saved_grace = DELIVERY_TERMINATE_GRACE_SECONDS
        original_run_proof_command = run_proof_command
        captured_execution_bounds = []
        def capture_execution_bounds(command, expected, timeout, grace,
                                     output_limit, **_kwargs):
            captured_execution_bounds.append((timeout, grace, output_limit))
            return False, "timeout", b""
        try:
            globals()["DELIVERY_PROCESS_TIMEOUT_SECONDS"] = 9
            globals()["DELIVERY_TERMINATE_GRACE_SECONDS"] = 4
            globals()["run_proof_command"] = capture_execution_bounds
            try:
                run_gate(
                    timeout_fd, timeout_delivery_fd, timeout_proof_cache,
                    [sys.executable, "-c", "raise SystemExit(99)"], None,
                    time.monotonic() + 30, 1,
                )
            except RuntimeError as error:
                require(str(error) == "delivery gate-1 failed: process-timeout",
                        "bounded gate timeout diagnostic")
            else:
                fail("delivery gate timeout accepted")
            require(captured_execution_bounds == [(9, 4, DELIVERY_OUTPUT_LIMIT_BYTES)],
                    "bounded gate timeout wiring")
        finally:
            globals()["run_proof_command"] = original_run_proof_command
            globals()["DELIVERY_PROCESS_TIMEOUT_SECONDS"] = saved_timeout
            globals()["DELIVERY_TERMINATE_GRACE_SECONDS"] = saved_grace
            os.close(timeout_delivery_fd); os.close(timeout_fd)



def check_gate_bytecode_isolation() -> None:
    import importlib.util
    import py_compile

    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp)
        create_source = repo / "create_probe.py"
        update_source = repo / "update_probe.py"
        create_source.write_text("VALUE = 1\n")
        update_source.write_text("VALUE = 1\n")
        saved_prefix = sys.pycache_prefix
        sys.pycache_prefix = None
        try:
            create_cache = Path(importlib.util.cache_from_source(str(create_source)))
            update_cache = Path(importlib.util.cache_from_source(str(update_source)))
        finally:
            sys.pycache_prefix = saved_prefix
        update_cache.parent.mkdir()
        py_compile.compile(
            str(update_source), cfile=str(update_cache), doraise=True,
            invalidation_mode=py_compile.PycInvalidationMode.CHECKED_HASH,
        )
        update_source.write_text("VALUE = 2\n")
        repo_fd = open_absolute_directory(repo.resolve())
        delivery_fd, proof_cache_relative = _gate_test_delivery(repo.resolve())
        command = [
            "env", "-u", "PYTHONPYCACHEPREFIX", sys.executable, "-c",
            "import create_probe, update_probe",
        ]
        try:
            before = manifest_at(repo_fd, set(), "__excluded__")
            update_preimage = update_cache.read_bytes()
            run_gate(
                repo_fd, delivery_fd, proof_cache_relative, command, None,
                time.monotonic() + DELIVERY_AGGREGATE_TIMEOUT_SECONDS, 0,
            )
            require(manifest_at(repo_fd, set(), "__excluded__") == before
                    and not create_cache.exists() and update_cache.read_bytes() == update_preimage,
                    "delivery gate repository bytecode isolation")

            mutant_env = os.environ.copy()
            mutant_env.pop("DEVRITES_REPO_ROOT", None)
            mutant_env.pop("PYTHONDONTWRITEBYTECODE", None)
            mutant = subprocess.run(
                with_delivery_execution_prefix(command), env=mutant_env,
                preexec_fn=lambda: os.fchdir(repo_fd), pass_fds=(repo_fd,),
                text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False,
            )
            require(mutant.returncode == 0 and create_cache.is_file()
                    and update_cache.read_bytes() != update_preimage
                    and manifest_at(repo_fd, set(), "__excluded__") != before,
                    "bytecode environment omission mutant rejected")
        finally:
            os.close(delivery_fd); os.close(repo_fd)


def check_gate_proof_cache_isolation() -> None:
    import importlib.util
    import py_compile

    saved_prefix = os.environ.pop("PYTHONPYCACHEPREFIX", None)
    saved_runtime_prefix = sys.pycache_prefix
    sys.pycache_prefix = None
    saved_env_step = _set_gate_proof_cache_env
    saved_runner = run_delivery_process
    captured = []
    def capture_runner(*args, **kwargs):
        result = saved_runner(*args, **kwargs)
        captured.append(result)
        return result
    globals()["run_delivery_process"] = capture_runner
    try:
        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve()
            scripts = repo / "scripts"
            scripts.mkdir()
            source = scripts / "cache_probe.py"
            source.write_text("VALUE = 1\n")
            source.chmod(0o644)
            cache = Path(importlib.util.cache_from_source(str(source)))
            cache.parent.mkdir()
            py_compile.compile(
                str(source), cfile=str(cache), doraise=True,
                invalidation_mode=py_compile.PycInvalidationMode.CHECKED_HASH,
            )
            cache.chmod(0o600)
            repo_fd = open_absolute_directory(repo)
            delivery_fd, proof_cache_relative = _gate_test_delivery(repo)
            command = [
                sys.executable, "-c",
                "import os,pathlib,py_compile,sys;"
                "py_compile.compile(sys.argv[1],doraise=True,"
                "invalidation_mode=py_compile.PycInvalidationMode.CHECKED_HASH);"
                "prefix=os.environ.get('PYTHONPYCACHEPREFIX');"
                "created=bool(prefix and list(pathlib.Path(prefix).rglob('*.pyc')));"
                "print(f'CACHE prefix={int(bool(prefix))} created={int(created)} pgid={os.getpgrp()}')",
                str(source),
            ]
            try:
                before = manifest_at(
                    repo_fd, set(),
                    ".devrites/work/workflow-artifact-identity/.generated-install",
                )
                preimage = file_record_at(repo_fd, cache.relative_to(repo).as_posix())
                result = run_gate(
                    repo_fd, delivery_fd, proof_cache_relative,
                    command, None, time.monotonic() + 30, 0,
                )
                require(result["command"] == command, "proof cache gate command identity")
                normal = captured.pop()[2].decode()
                match = re.search(r"CACHE prefix=1 created=1 pgid=([0-9]+)", normal)
                require(match is not None and not process_group_alive(int(match.group(1))),
                        "proof cache normal group reaped")
                require(file_record_at(repo_fd, cache.relative_to(repo).as_posix()) == preimage,
                        "proof cache source pyc identity")
                require(entry_info_at(delivery_fd, "proof-cache") is None,
                        "proof cache normal cleanup")
                require(manifest_at(
                            repo_fd, set(),
                            ".devrites/work/workflow-artifact-identity/.generated-install",
                        ) == before,
                        "proof cache normal outside identity")

                globals()["_set_gate_proof_cache_env"] = lambda _env, _relative: None
                run_gate(
                    repo_fd, delivery_fd, proof_cache_relative,
                    command, None, time.monotonic() + 30, 1,
                )
                mutant = captured.pop()[2].decode()
                match = re.search(r"CACHE prefix=0 created=0 pgid=([0-9]+)", mutant)
                require(match is not None and not process_group_alive(int(match.group(1))),
                        "proof cache omission mutant group reaped")
                mutated = file_record_at(repo_fd, cache.relative_to(repo).as_posix())
                require(mutated["mode"] == 0o644
                        and mutated["sha256"] == preimage["sha256"],
                        "proof cache omission mutant mode-only drift")
                after = manifest_at(
                    repo_fd, set(),
                    ".devrites/work/workflow-artifact-identity/.generated-install",
                )
                changed = {relative for relative in before if before[relative] != after.get(relative)}
                require(changed == {cache.relative_to(repo).as_posix()} and after != before,
                        "proof cache omission mutant outside rejection")
                require(entry_info_at(delivery_fd, "proof-cache") is None,
                        "proof cache mutant cleanup")
            finally:
                cache.chmod(0o600)
                globals()["_set_gate_proof_cache_env"] = saved_env_step
                os.close(delivery_fd); os.close(repo_fd)

        with tempfile.TemporaryDirectory() as tmp:
            repo = Path(tmp).resolve()
            repo_fd = open_absolute_directory(repo)
            delivery_fd, proof_cache_relative = _gate_test_delivery(repo, "1" * 64)
            delivery = repo / proof_cache_relative.removesuffix("/proof-cache")
            try:
                _prepare_gate_proof_cache(delivery_fd, proof_cache_relative)
                (delivery / "proof-cache/nested").mkdir()
                (delivery / "proof-cache/nested/cache.pyc").write_bytes(b"private")
                sibling = delivery / "unknown-private"
                sibling.write_bytes(b"untouched")
                sibling.chmod(0o600)
                _reconcile_gate_proof_cache(delivery_fd)
                require(not (delivery / "proof-cache").exists()
                        and sibling.read_bytes() == b"untouched",
                        "orphan proof cache exact reconciliation")
                try:
                    validate_mutation_artifacts({}, delivery_fd, repo_fd)
                except AssertionError as error:
                    require(str(error) == "unknown delivery private entries: ['unknown-private']",
                            "orphan sibling private-entry rejection")
                else:
                    fail("unknown private sibling accepted after proof cache reconciliation")
            finally:
                os.close(delivery_fd); os.close(repo_fd)

        for kind in ("regular", "symlink"):
            with tempfile.TemporaryDirectory() as tmp:
                repo = Path(tmp).resolve()
                delivery_fd, proof_cache_relative = _gate_test_delivery(repo, "2" * 64)
                delivery = repo / proof_cache_relative.removesuffix("/proof-cache")
                target = repo / "target"
                target.mkdir()
                (target / "marker").write_bytes(b"safe")
                if kind == "regular":
                    (delivery / "proof-cache").write_bytes(b"wrong-type")
                else:
                    os.symlink(target, delivery / "proof-cache")
                try:
                    try:
                        _reconcile_gate_proof_cache(delivery_fd)
                    except AssertionError as error:
                        require(str(error) == "delivery proof cache metadata",
                                f"proof cache {kind} rejection")
                    else:
                        fail(f"proof cache {kind} accepted")
                    require((target / "marker").read_bytes() == b"safe",
                            f"proof cache {kind} target untouched")
                finally:
                    os.close(delivery_fd)
    finally:
        globals()["run_delivery_process"] = saved_runner
        globals()["_set_gate_proof_cache_env"] = saved_env_step
        sys.pycache_prefix = saved_runtime_prefix
        if saved_prefix is not None:
            os.environ["PYTHONPYCACHEPREFIX"] = saved_prefix


def check_delivery_mode_entry_guard() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(tmp).resolve()
        repo = base / "repo"
        before = create_actual_delivery_repo(repo)
        delivery = repo / ".devrites/work/workflow-artifact-identity/.generated-install" / ("0" * 64)
        env = os.environ.copy()
        env["DEVRITES_REPO_ROOT"] = str(repo)
        for name in DELIVERY_FIXTURE_ENV:
            env.pop(name, None)
        env.pop("PYTHONDONTWRITEBYTECODE", None)
        for args in (["--delivery-prepare"], ["--delivery-install", str(delivery)],
                     ["--delivery-recover", str(delivery)]):
            rejected = subprocess.run(
                [str(SCRIPT), *args], env=env, text=True, stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT, check=False, timeout=10,
            )
            require(rejected.returncode != 0
                    and "delivery modes require PYTHONDONTWRITEBYTECODE=1" in rejected.stdout,
                    f"delivery mode bytecode entry guard: {args[0]}")

        hostile_path = base / "hostile-path"
        hostile_path.mkdir()
        (hostile_path / "bash").symlink_to("bash")
        hostile_env = env.copy()
        hostile_env["PYTHONDONTWRITEBYTECODE"] = "1"
        hostile_env["PATH"] = f"{hostile_path}{os.pathsep}{hostile_env['PATH']}"
        reason = "delivery modes require executable bash PATH"
        # Invoke with absolute bash so Linux /usr/bin/env does not die on the
        # looping PATH entry before require_delivery_mode_environment runs.
        host_bash = shutil.which("bash")
        require(host_bash is not None, "host bash for delivery PATH guard")
        for args in (["--delivery-prepare"], ["--delivery-install", str(delivery)],
                     ["--delivery-recover", str(delivery)]):
            rejected = subprocess.run(
                [host_bash, str(SCRIPT), *args], env=hostile_env, text=True,
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, timeout=10,
            )
            require(rejected.returncode != 0
                    and rejected.stdout.splitlines()[-1:] == [f"AssertionError: {reason}"],
                    f"delivery mode bash PATH entry guard: {args[0]}")
            require(not any(delivery.parent.iterdir()),
                    f"delivery mode bash PATH artifacts absent: {args[0]}")
            repo_fd = open_absolute_directory(repo)
            try:
                require({relative: file_record_at(repo_fd, relative)
                         for relative in AUTHORED + GENERATED} == before,
                        f"delivery mode bash PATH destinations unchanged: {args[0]}")
            finally:
                os.close(repo_fd)

        mutant_root = base / "mutant-candidate"
        mutant_root.mkdir()
        for relative in AUTHORED:
            source_path = canonical_root() / relative
            secure_file_info(os.lstat(source_path))
            destination = mutant_root / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source_path, destination)
        inventory = sorted(
            path.relative_to(mutant_root).as_posix()
            for path in mutant_root.rglob("*")
            if stat.S_ISREG(os.lstat(path).st_mode)
        )
        require(inventory == sorted(AUTHORED), "delivery guard mutant exact authored inventory")
        mutant_driver = mutant_root / "tests/workflow-artifact-identity-test.sh"
        source = mutant_driver.read_text()
        guard_call = "        require_delivery_mode_environment()\n"
        require(source.count(guard_call) == 1, "delivery mode guard deletion site")
        mutant_driver.write_text(source.replace(guard_call, "        pass\n", 1))
        mutant_driver.chmod(0o755)
        accepted = subprocess.run(
            [str(mutant_driver), "--delivery-test-fast-fixture",
             "--delivery-boundary-case", "operate", "prepare"],
            env=env, text=True,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, timeout=45,
        )
        require(accepted.returncode == 0 and "delivery_state=SNAPSHOTTING" in accepted.stdout,
                f"delivery mode guard deletion mutant distinguished: {accepted.stdout[-1000:]}")


def check_walkthrough_outer_bound() -> None:
    env = os.environ.copy()
    env["DEVRITES_REPO_ROOT"] = os.environ.get("DEVRITES_REPO_ROOT", str(canonical_root()))
    proc = subprocess.run(
        [str(SCRIPT), "--prove-walkthrough"], env=env, text=True,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, timeout=90,
    )
    lines = proc.stdout.splitlines()
    require(proc.returncode == 0 and len(lines) == 6, f"walkthrough six lines: {lines}")
    require(lines[0] == "WA-PROOF-001 PASS"
            and re.fullmatch(r"tthw_ms=[0-9]+", lines[1]) is not None
            and lines[2] == "WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE boundary_id=WA-B004-SOURCE-OPEN next_route=PLAN_VET_REPAIR"
            and lines[3] == "cursor=prove:/rite-prove demo"
            and lines[4] == "product_identity=unchanged"
            and lines[5] == "WORKFLOW_ARTIFACT_WALKTHROUGH PASS",
            "walkthrough captured signals")


def check_drift_005_regressions() -> None:
    source = SCRIPT.read_text()
    require("def classify" + "_route(" not in source, "test-local classifier authority")
    require("def retry" + "_transition(" not in source, "test-local retry authority")
    require("ignore_errors" + "=True" not in source, "suppressed delivery cleanup error")
    for forbidden in (
        "import ctype" + "s", "def direct_child_" + "pids(", "def enable_child_" + "subreaper(",
        "def refresh_" + "descendants(", "def tracked_" + "survivors(", "def signal_" + "tracked(",
    ):
        require(forbidden not in source, f"detached-session tracking remains: {forbidden}")


def resolve_evidence_mapping_source(project: Path) -> tuple[Path, bool]:
    """Prefer live work, then archive, then committed fixture (CI has no .devrites)."""
    live = project / ".devrites/work/workflow-artifact-identity"
    if (live / "evidence.md").is_file() and (live / "traceability.md").is_file():
        return live, True
    archive = project / ".devrites/archive/workflow-artifact-identity"
    if (archive / "evidence.md").is_file() and (archive / "traceability.md").is_file():
        return archive, False
    fixture = SCRIPT.parent / "fixtures" / "workflow-artifact-identity-evidence-workspace"
    require((fixture / "evidence.md").is_file() and (fixture / "traceability.md").is_file(),
            "workflow-artifact evidence mapping fixture")
    return fixture, False


def check_workspace_evidence_mapping() -> None:
    project = project_root_for_tests(canonical_root())
    source, is_live = resolve_evidence_mapping_source(project)
    schema = ["python3", "scripts/validate-workspace-schema.py"]
    unmapped_008 = "evidence ID EVID-008 from evidence/browser proof is not mapped"
    unmapped_010 = "evidence ID EVID-010 from evidence/browser proof is not mapped"
    unmapped_013 = "evidence ID EVID-013 from evidence/browser proof is not mapped"
    source_evidence = (source / "evidence.md").read_bytes()
    source_trace = (source / "traceability.md").read_bytes()
    require(b"EVID-008" in source_evidence and b"EVID-010" in source_evidence
            and b"EVID-013" in source_evidence,
            "source evidence owns mapped IDs")
    require(b"EVID-008" in source_trace and b"EVID-010" in source_trace
            and b"EVID-013" in source_trace,
            "source traceability maps locked evidence IDs")
    with tempfile.TemporaryDirectory() as tmp:
        copy = Path(tmp) / "workflow-artifact-identity"
        shutil.copytree(source, copy, ignore=shutil.ignore_patterns(
            ".generated-install", ".root-proof*", ".prove-*", ".review-*",
            ".candidate-*", "correction-baseline*.json", "build-baseline.json",
        ))
        evidence = (copy / "evidence.md").read_text()
        require("EVID-008" in evidence and "EVID-010" in evidence
                and "EVID-013" in evidence,
                "unmapped fixture keeps evidence IDs")
        state_path = copy / "state.md"
        state_lines = state_path.read_text().splitlines(keepends=True)
        pinned: list[str] = []
        replaced_phase = False
        for line in state_lines:
            if not replaced_phase and line.startswith("| phase |"):
                pinned.append("| phase | prove |\n")
                replaced_phase = True
            else:
                pinned.append(line)
        require(replaced_phase, "disposable copy pins proofRequired phase")
        state_path.write_text("".join(pinned))
        trace_path = copy / "traceability.md"
        stripped = (trace_path.read_text()
                    .replace("EVID-008", "").replace("EVID-010", "").replace("EVID-013", ""))
        require("EVID-008" not in stripped and "EVID-010" not in stripped
                and "EVID-013" not in stripped,
                "unmapped fixture omits locked evidence IDs")
        trace_path.write_text(stripped)
        red = subprocess.run(
            schema + [str(copy)], cwd=project, text=True,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, timeout=120,
        )
        require(red.returncode != 0 and unmapped_008 in red.stdout
                and unmapped_010 in red.stdout and unmapped_013 in red.stdout,
                f"unmapped evidence IDs must fail schema: {red.stdout[-1000:]}")
        green_copy = Path(tmp) / "workflow-artifact-identity-green"
        shutil.copytree(source, green_copy, ignore=shutil.ignore_patterns(
            ".generated-install", ".root-proof*", ".prove-*", ".review-*",
            ".candidate-*", "correction-baseline*.json", "build-baseline.json",
        ))
        green = subprocess.run(
            schema + [str(green_copy)], cwd=project, text=True,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, check=False, timeout=120,
        )
        require(green.returncode == 0
                and "workspace-schema: OK: 1 workspace(s) validated" in green.stdout.splitlines(),
                f"source workspace schema: {green.stdout[-1000:]}")
    if is_live:
        require((source / "evidence.md").read_bytes() == source_evidence
                and (source / "traceability.md").read_bytes() == source_trace,
                "live workspace files unchanged by mapping fixture")


def default_tests(root: Path) -> None:
    check_drift_005_regressions()
    require(writer_allowlist_digest() == WRITER_ALLOWLIST_SHA256, "frozen writer allowlist")

    checks: list[tuple[str, object]] = [
        ("empty_generated_delta_install", check_empty_generated_delta_install),
        ("module_and_corpus", lambda: check_module_and_corpus(root)),
        ("historical_reslice_identity", check_historical_reslice_identity),
        ("workspace_evidence_mapping", check_workspace_evidence_mapping),
        ("normal_generator_contract_delta", check_normal_generator_contract_delta),
        ("complete_stage_gate_failure_rollback", check_complete_stage_gate_failure_rollback),
        ("instruction_size_baseline", lambda: (
            (lambda measured: require(
                measured[0] == 219 and measured[1] <= 860000 and 860000 - measured[1] > 14,
                "instruction size count/cap/headroom",
            ))(check_instruction_size_baseline(root))
        )),
        ("walkthrough_outer_bound", check_walkthrough_outer_bound),
        ("golden_vector", check_golden_vector),
        ("markdown_backslash_parity", check_markdown_backslash_parity),
        ("admission_parser", check_admission_parser),
        ("limit_boundaries", check_limit_boundaries),
        ("complete_writes", check_complete_writes),
        ("complete_write_matrix", check_complete_write_matrix),
        ("atomic_write_death_recovery", check_atomic_write_death_recovery),
        ("descriptor_ancestor_mutants", check_descriptor_ancestor_mutants),
        ("manifest_exclusions", check_manifest_exclusions),
        ("manifest_descriptor_substitution", check_manifest_descriptor_substitution),
        ("file_record_descriptor_identity_and_bound", check_file_record_descriptor_identity_and_bound),
        ("generated_stage_manifest_security_bounds", check_generated_stage_manifest_security_bounds),
        ("validate_delivery_stage_deadline", check_validate_delivery_stage_deadline),
        ("gate_descriptor_ancestor_replacement", check_gate_descriptor_ancestor_replacement),
        ("gate_bytecode_isolation", check_gate_bytecode_isolation),
        ("gate_proof_cache_isolation", check_gate_proof_cache_isolation),
        ("delivery_mode_entry_guard", check_delivery_mode_entry_guard),
        ("production_delivery_fixture_env", check_production_delivery_fixture_env),
        ("generator_descriptor_confinement", check_generator_descriptor_confinement),
        ("held_stage_generator_symlink_swap", check_held_stage_generator_symlink_swap),
        ("held_stage_generator_held_out_symlink_plant", check_held_stage_generator_held_out_symlink_plant),
        ("held_stage_generator_artifacts_symlink_plant", check_held_stage_generator_artifacts_symlink_plant),
        ("held_stage_generator_out_root_basename_plant", check_held_stage_generator_out_root_basename_plant),
        ("absolute_directory_acquisition", check_absolute_directory_acquisition),
        ("evidence_ownership", check_evidence_ownership),
        ("evidence_matrix", check_evidence_matrix),
        ("flock", check_flock),
        ("filesystem_adversaries", check_filesystem_adversaries),
        ("source_lifecycle", check_source_lifecycle),
        ("process_group_timeout", check_process_group_timeout),
        ("proof_matrix", check_proof_matrix),
        ("delivery_execution_bounds", check_delivery_execution_bounds),
        ("delivery_generator_umask", check_delivery_generator_umask),
        ("operation_oracle", lambda: check_operation_oracle(root)),
        ("classifier_and_diagnostics", lambda: check_classifier_and_diagnostics(root)),
        ("retry_and_source_loss", lambda: check_retry_and_source_loss(root)),
        ("product_separation_lifecycle", lambda: check_product_separation_lifecycle(root)),
        ("actual_engine_separation", lambda: check_actual_engine_separation(root)),
        ("delivery_third_state_guards", check_delivery_third_state_guards),
        ("drift_016_delivery_mutants", check_drift_016_delivery_mutants),
        ("first_authored_claim_exception_rollback", check_first_authored_claim_exception_rollback),
        ("drift_017_prepare_outside_drift", check_drift_017_prepare_outside_drift),
        ("delivery_recovery_identity", check_delivery_recovery_identity),
        ("temporary_journal_successors", check_temporary_journal_successors),
        ("future_journal_clock_rollback", check_future_journal_clock_rollback),
        ("recursive_outside_directory_protection", check_recursive_outside_directory_protection),
        ("terminal_recovery_idempotence", check_terminal_recovery_idempotence),
        ("outside_manifest_bounds", check_outside_manifest_bounds),
        ("outside_manifest_hard_wall", check_outside_manifest_hard_wall),
        ("outside_manifest_sidecar_metadata", check_outside_manifest_sidecar_metadata),
        ("outside_manifest_sidecar_integrity", check_outside_manifest_sidecar_integrity),
        ("prejournal_sidecar_reuse", check_prejournal_sidecar_reuse),
        ("bootstrap_sidecar_temporary_adversaries", check_bootstrap_sidecar_temporary_adversaries),
        ("bootstrap_journal_temporary_adversaries", check_bootstrap_journal_temporary_adversaries),
        ("initial_journal_strict_prefix_recovery", check_initial_journal_strict_prefix_recovery),
        ("delivery_journal_adversaries", check_delivery_journal_adversaries),
        ("journal_retry_stability", check_journal_retry_stability),
        ("cleanup_recovery", check_cleanup_recovery),
    ]
    if not wai_skip_delivery_modes():
        checks.append(("actual_delivery_modes", check_actual_delivery_modes))
    if not wai_skip_delivery_model_matrix():
        checks.append(("delivery_model_matrix", check_delivery_model_matrix))

    core_spec = os.environ.get("DEVRITES_WAI_CORE_SHARD", "").strip()
    if core_spec:
        match = re.fullmatch(r"(\d+)/(\d+)", core_spec)
        require(match is not None, f"invalid DEVRITES_WAI_CORE_SHARD: {core_spec}")
        index = int(match.group(1))
        total = int(match.group(2))
        require(1 <= index <= total, f"invalid DEVRITES_WAI_CORE_SHARD: {core_spec}")
        start = ((index - 1) * len(checks)) // total
        end = (index * len(checks)) // total
        checks = checks[start:end]
        require(len(checks) > 0, f"empty workflow-artifact core shard: {core_spec}")

    # Keep core checks on the main thread: several install signal handlers and
    # process-group helpers require signal.signal, which is main-thread only.
    for name, fn in checks:
        try:
            fn()
        except BaseException as error:
            raise AssertionError(f"workflow-artifact core check failed: {name}") from error


def writer_allowlist_digest() -> str:
    value = sha(("\n".join(AUTHORED + GENERATED) + "\n").encode())
    require(value == WRITER_ALLOWLIST_SHA256, "writer allowlist digest")
    return value


def parse_journal(raw: bytes) -> dict:
    def unique_object(pairs):
        value = {}
        for key, item in pairs:
            require(key not in value, f"duplicate delivery journal field: {key}")
            value[key] = item
        return value
    require(raw.endswith(b"\n") and raw.count(b"\n") == 1, "delivery journal exact line")
    return json.loads(raw.decode("utf-8"), object_pairs_hook=unique_object)


def read_journal(delivery_fd: int) -> dict:
    return parse_journal(read_file_at(
        delivery_fd, "journal.json", DELIVERY_JOURNAL_MAX_BYTES, {0o600},
    ))


DELIVERY_JOURNAL_FIELDS = {
    "contract", "state", "candidate_digest", "candidate_root", "candidate_root_identity", "driver_sha256",
    "authored_allowlist", "generated_allowlist", "writer_allowlist_sha256",
    "protected", "outside_manifest", "authored", "generated",
    "installed_authored", "installed_generated", "updated_ns", "journal_sequence",
}
DELIVERY_JOURNAL_OPTIONAL_FIELDS = {
    "installed_authored_intent", "stage_manifest_sha256", "expected_post", "gates",
    "mutation_intent",
}
DELIVERY_STATE_PATTERN = re.compile(
    r"(?:SNAPSHOTTING|STAGED|INSTALLED|PROVING|COMMITTED|CLEANING|CLEANED|RESTORED|FAILED"
    r"|INSTALLING\([1-9][0-9]*\)|ROLLING_BACK\([1-9][0-9]*\))"
)
DELIVERY_GATES = [
    (["bash", "-c", "bash --version && python3 --version && node --version && go -C engine env GOVERSION GOTOOLCHAIN"], "go1.26.7"),
    (["python3", "scripts/validate-workspace-schema.py", ".devrites/work/workflow-artifact-identity"], "workspace-schema: OK: 1 workspace(s) validated"),
    (["bash", "tests/workflow-artifact-identity-test.sh"], "workflow-artifact-identity: PASS"),
    (["bash", "tests/workflow-artifact-identity-test.sh", "--prove-walkthrough"], "WORKFLOW_ARTIFACT_WALKTHROUGH PASS"),
    (["bash", "scripts/run-behavioral-evals.sh"], "Validated 14 behavioral eval file(s); 82 scenario(s); 0 failed."),
    (["bash", "tests/phase-gate-routing-test.sh"], "phase-gate-routing-test: PASS"),
    (["bash", "tests/acceptance-preserving-reslice-policy-test.sh"], "acceptance-preserving-reslice-policy-test: PASS"),
    (["bash", "tests/host-artifacts-test.sh"], "host-artifacts-test: PASS"),
    (["node", "scripts/check-instruction-size-baseline.mjs"], None),
    (["bash", "scripts/validate.sh"], "VALIDATION PASSED"),
    (["python3", "scripts/check-cross-refs.py"], None),
    (["python3", "scripts/check-invocation-integrity.py"], None),
    (["python3", "scripts/scan-pack-security.py", "pack/.claude", "pack/generated"], None),
    (["go", "-C", "engine", "test", "./...", "-race", "-count=1"], None),
    (["node", "scripts/run-tests.mjs"], None),
    (["shasum", "-a", "256", ".gitignore", ".devrites/ACTIVE", ".devrites/work/workspace-observation/touched-files.md"], "24fc2f2ec652f10c946901863681711b541b018eda200292b51279819cec9484  .gitignore"),
]


def check_delivery_gate_signals() -> None:
    expected = [
        (["bash", "-c", "bash --version && python3 --version && node --version && go -C engine env GOVERSION GOTOOLCHAIN"], "go1.26.7"),
        (["python3", "scripts/validate-workspace-schema.py", ".devrites/work/workflow-artifact-identity"], "workspace-schema: OK: 1 workspace(s) validated"),
        (["bash", "tests/workflow-artifact-identity-test.sh"], "workflow-artifact-identity: PASS"),
        (["bash", "tests/workflow-artifact-identity-test.sh", "--prove-walkthrough"], "WORKFLOW_ARTIFACT_WALKTHROUGH PASS"),
        (["bash", "scripts/run-behavioral-evals.sh"], "Validated 14 behavioral eval file(s); 82 scenario(s); 0 failed."),
        (["bash", "tests/phase-gate-routing-test.sh"], "phase-gate-routing-test: PASS"),
        (["bash", "tests/acceptance-preserving-reslice-policy-test.sh"], "acceptance-preserving-reslice-policy-test: PASS"),
        (["bash", "tests/host-artifacts-test.sh"], "host-artifacts-test: PASS"),
        (["bash", "scripts/validate.sh"], "VALIDATION PASSED"),
        (["shasum", "-a", "256", ".gitignore", ".devrites/ACTIVE", ".devrites/work/workspace-observation/touched-files.md"], "24fc2f2ec652f10c946901863681711b541b018eda200292b51279819cec9484  .gitignore"),
    ]
    declared = [(command, signal) for command, signal in DELIVERY_GATES if signal is not None]
    require(declared == expected, "delivery gate exact expected-line registry")

    for _command, expected_line in expected:
        ok, reason, output = run_proof_command(
            [sys.executable, "-c", "import sys;print('before');print(sys.argv[1]);print('after')", expected_line],
            expected_line, 2, 0.1, 4096,
        )
        require(ok and reason == "proved"
                and output.splitlines() == [b"before", expected_line.encode(), b"after"],
                f"delivery gate multiline exact signal: {expected_line}")
        for fixture in (
            "import sys;print('before');print(sys.argv[1]+'!');print('after')",
            "import sys;print('before');print(sys.argv[1]);print(sys.argv[1]);print('after')",
        ):
            ok, reason, _output = run_proof_command(
                [sys.executable, "-c", fixture, expected_line],
                expected_line, 2, 0.1, 4096,
            )
            require(not ok and reason == "wrong-signal",
                    f"delivery gate rejects near-match or duplicate: {expected_line}")

    for command, expected_line in (expected[4], expected[-1]):
        ok, reason, output = run_proof_command(
            command, expected_line, 120, DELIVERY_TERMINATE_GRACE_SECONDS,
            DELIVERY_OUTPUT_LIMIT_BYTES,
        )
        require(ok and reason == "proved",
                f"delivery gate real signal: {command}: {reason}: {output[-1000:]!r}")


def validate_delivery_snapshot(record: dict, group: str, index: int, relative: str,
                               delivery_fd: int, verify_backup: bool) -> None:
    required = {"state", "path", "index"}
    require(isinstance(record, dict) and record.get("state") in {"present", "absent"}, "delivery snapshot state")
    if record["state"] == "present":
        required |= {"mode", "size", "sha256", "backup"}
    require(set(record) == required, f"delivery snapshot fields: {group}/{index}")
    require(record["path"] == relative and record["index"] == index, f"delivery snapshot identity: {group}/{index}")
    if record["state"] == "absent":
        return
    require(isinstance(record["mode"], int) and not isinstance(record["mode"], bool)
            and 0 <= record["mode"] <= 0o7777, "delivery snapshot mode")
    require(isinstance(record["size"], int) and not isinstance(record["size"], bool)
            and 0 <= record["size"] <= 16 * 1024 * 1024, "delivery snapshot size")
    require(isinstance(record["sha256"], str) and re.fullmatch(r"[0-9a-f]{64}", record["sha256"]) is not None, "delivery snapshot hash")
    backup = f"backups/{group}/{index:08x}"
    require(record["backup"] == backup, f"delivery snapshot backup: {group}/{index}")
    if verify_backup:
        data = read_file_at(delivery_fd, backup, record["size"], {0o600})
        require(len(data) == record["size"] and sha(data) == record["sha256"], f"delivery backup identity: {group}/{index}")


def validate_delivery_backup_tree(journal: dict, delivery_fd: int, repo_fd: int) -> None:
    backup_info = entry_info_at(delivery_fd, "backups")
    if journal["state"] == "CLEANED":
        require(backup_info is None, "cleaned delivery backup absence")
        return
    expected = {}
    optional = {}
    for group, compiled in (("authored", AUTHORED), ("generated", GENERATED)):
        for index, record in enumerate(journal[group]):
            if record["state"] == "present":
                expected[f"{group}/{index:08x}"] = {"mode": 0o600, "sha256": record["sha256"]}
        if journal["state"] == "SNAPSHOTTING" and len(journal[group]) < len(compiled):
            index = len(journal[group])
            current = file_record_at(repo_fd, compiled[index])
            if current["state"] == "present":
                optional[f"{group}/{index:08x}"] = {"mode": 0o600, "sha256": current["sha256"]}
    if backup_info is None:
        require(journal["state"] == "CLEANING"
                or not any(record["state"] == "present" for group in ("authored", "generated")
                           for record in journal[group]), "recorded delivery backups present")
        return
    backups_fd = os.open("backups", DIRECTORY_FLAGS, dir_fd=delivery_fd)
    manifest_deadline = time.monotonic() + OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
    try:
        observed = generated_stage_manifest_at(backups_fd, manifest_deadline)
    finally:
        os.close(backups_fd)
    if journal["state"] == "CLEANING":
        require(set(observed) <= set(expected)
                and all(observed[name] == expected[name] for name in observed),
                "cleaning delivery backup suffix")
        return
    extra = set(observed) - set(expected)
    require(all(observed.get(name) == value for name, value in expected.items())
            and len(observed) == len(expected) + len(extra)
            and len(extra) <= 1 and extra <= set(optional)
            and all(observed[name] == optional[name] for name in extra),
            "delivery backup exact tree")


def validate_delivery_journal(journal: dict, delivery_fd: int, repo_fd: int,
                              expected_candidate_digest: str,
                              require_complete: bool = False,
                              check_outside: bool = True,
                              allow_journal_temporary: bool = False) -> dict:
    require(isinstance(journal, dict), "delivery journal object")
    require(set(journal) >= DELIVERY_JOURNAL_FIELDS
            and set(journal) <= DELIVERY_JOURNAL_FIELDS | DELIVERY_JOURNAL_OPTIONAL_FIELDS,
            "delivery journal exact fields")
    require(journal["contract"] == "devrites.workflow-artifact-delivery.v1", "delivery journal contract")
    require(isinstance(journal["state"], str) and DELIVERY_STATE_PATTERN.fullmatch(journal["state"]) is not None, "delivery journal state")
    for name in ("candidate_digest", "driver_sha256", "writer_allowlist_sha256"):
        require(isinstance(journal[name], str) and re.fullmatch(r"[0-9a-f]{64}", journal[name]) is not None, f"delivery journal digest: {name}")
    require(re.fullmatch(r"[0-9a-f]{64}", expected_candidate_digest) is not None
            and journal["candidate_digest"] == expected_candidate_digest,
            "delivery directory candidate identity")
    require(journal["writer_allowlist_sha256"] == writer_allowlist_digest(), "delivery journal allowlist digest")
    require(journal["authored_allowlist"] == AUTHORED and journal["generated_allowlist"] == GENERATED, "delivery journal compiled allowlists")
    require(isinstance(journal["candidate_root"], str) and Path(journal["candidate_root"]).is_absolute()
            and len(journal["candidate_root"].encode()) <= 4096,
            "delivery candidate root grammar")
    candidate_fd = open_absolute_directory(Path(journal["candidate_root"]))
    try:
        root_info = os.fstat(candidate_fd)
        require(journal["candidate_root_identity"] == {
                    "device": root_info.st_dev, "inode": root_info.st_ino,
                }, "delivery candidate root identity")
        require(aggregate_at(candidate_fd, AUTHORED) == journal["candidate_digest"],
                "delivery candidate root aggregate")
        driver = file_record_at(candidate_fd, "tests/workflow-artifact-identity-test.sh")
        require(driver["state"] == "present" and driver["mode"] == 0o755
                and driver["sha256"] == journal["driver_sha256"],
                "delivery candidate root driver")
    finally:
        os.close(candidate_fd)
    require(isinstance(journal["updated_ns"], int) and not isinstance(journal["updated_ns"], bool)
            and journal["updated_ns"] >= 0, "delivery journal timestamp")
    require(isinstance(journal["journal_sequence"], int)
            and not isinstance(journal["journal_sequence"], bool)
            and journal["journal_sequence"] >= 1, "delivery journal sequence")
    require(journal["protected"] == protected_records_at(repo_fd), "delivery journal protected binding")
    outside_manifest = read_outside_manifest(delivery_fd, journal["outside_manifest"])
    for group, compiled in (("authored", AUTHORED), ("generated", GENERATED)):
        records = journal[group]
        require(isinstance(records, list) and len(records) <= len(compiled), f"delivery snapshot cardinality: {group}")
        if require_complete:
            require(len(records) == len(compiled), f"delivery complete snapshots: {group}")
        for index, record in enumerate(records):
            validate_delivery_snapshot(
                record, group, index, compiled[index], delivery_fd,
                journal["state"] not in {"CLEANING", "CLEANED"},
            )
    for name, maximum in (("installed_authored", len(AUTHORED)), ("installed_generated", len(GENERATED))):
        value = journal[name]
        require(isinstance(value, int) and not isinstance(value, bool) and 0 <= value <= maximum, f"delivery counter: {name}")
    require(journal["installed_authored"] <= len(journal["authored"])
            and journal["installed_generated"] <= len(journal["generated"]), "delivery counters within snapshots")
    state = journal["state"]
    complete = len(journal["authored"]) == len(AUTHORED) and len(journal["generated"]) == len(GENERATED)
    if state == "SNAPSHOTTING":
        require(journal["installed_generated"] == 0, "snapshotting generated counter")
    elif state == "STAGED":
        require(complete and journal["installed_authored"] == len(AUTHORED)
                and journal["installed_generated"] == 0, "staged counter relation")
    elif state.startswith("INSTALLING("):
        installing_index = int(state.removeprefix("INSTALLING(").removesuffix(")"))
        require(complete and journal["installed_authored"] == len(AUTHORED)
                and 1 <= installing_index <= len(GENERATED)
                and journal["installed_generated"] in {installing_index - 1, installing_index},
                "installing counter relation")
    elif state in {"INSTALLED", "PROVING", "COMMITTED", "CLEANING", "CLEANED"}:
        require(complete and journal["installed_authored"] == len(AUTHORED)
                and journal["installed_generated"] == len(GENERATED), "terminal install counters")
    else:
        require(complete, "recovery requires complete snapshots")
    if "installed_authored_intent" in journal:
        require(isinstance(journal["installed_authored_intent"], int)
                and journal["installed_authored"] <= journal["installed_authored_intent"] <= min(len(AUTHORED), journal["installed_authored"] + 1),
                "delivery authored intent counter")
    if "stage_manifest_sha256" in journal:
        require(isinstance(journal["stage_manifest_sha256"], str)
                and re.fullmatch(r"[0-9a-f]{64}", journal["stage_manifest_sha256"]) is not None,
                "delivery stage manifest hash")
    if "expected_post" in journal:
        expected_post = journal["expected_post"]
        require(isinstance(expected_post, list) and len(expected_post) == len(AUTHORED) + len(GENERATED),
                "delivery expected-post cardinality")
        require([record.get("path") for record in expected_post] == AUTHORED + GENERATED,
                "delivery expected-post order")
        for record in expected_post:
            require(set(record) == {"path", "state", "mode", "sha256"}
                    and record["state"] == "present"
                    and isinstance(record["mode"], int) and not isinstance(record["mode"], bool)
                    and re.fullmatch(r"[0-9a-f]{64}", record["sha256"]) is not None,
                    "delivery expected-post record")
    if state in {"STAGED", "INSTALLED", "PROVING", "COMMITTED", "CLEANING", "CLEANED"} or state.startswith("INSTALLING("):
        require("stage_manifest_sha256" in journal and "expected_post" in journal,
                "delivery staged identity required")
    if "gates" in journal:
        require(isinstance(journal["gates"], list) and len(journal["gates"]) <= len(DELIVERY_GATES),
                "delivery gate list")
        for index, gate in enumerate(journal["gates"]):
            expected_command, expected_signal = DELIVERY_GATES[index]
            require(isinstance(gate, dict) and set(gate) == {"command", "execution_prefix", "sha256", "signal"}
                    and gate["execution_prefix"] == delivery_execution_prefix()
                    and gate["command"] == expected_command
                    and re.fullmatch(r"[0-9a-f]{64}", gate["sha256"]) is not None
                    and gate["signal"] == (expected_signal or "exit=0"), "delivery gate record")
    if state in {"COMMITTED", "CLEANING", "CLEANED"}:
        require("gates" in journal and len(journal["gates"]) == len(DELIVERY_GATES),
                "committed delivery complete gates")
    validate_mutation_artifacts(
        journal, delivery_fd, repo_fd, allow_journal_temporary,
    )
    validate_delivery_state_effects(journal, delivery_fd, repo_fd)
    if check_outside:
        require(outside_manifest == manifest_at(
                    repo_fd, ALL_DESTINATIONS,
                    f".devrites/work/workflow-artifact-identity/.generated-install/{expected_candidate_digest}",
                ), "delivery outside manifest binding")
    validate_delivery_backup_tree(journal, delivery_fd, repo_fd)
    return journal


def write_journal(delivery_fd: int, lock_fd: int, data: dict,
                  bootstrap: bool = False) -> None:
    if "updated_ns" in data:
        data["updated_ns"] += 1
    else:
        data["updated_ns"] = time.time_ns()
    data["journal_sequence"] = data.get("journal_sequence", 0) + 1
    encoded = (json.dumps(data, sort_keys=True, separators=(",", ":")) + "\n").encode()
    require(len(encoded) <= DELIVERY_JOURNAL_MAX_BYTES, "delivery journal bound")
    death_boundary = bootstrap_atomic_death("journal") if bootstrap else None
    if (not bootstrap
            and delivery_test_death_boundary()
            == "journal-replacement-after-sync"):
        death_boundary = "after-sync"
    atomic_write_at(
        delivery_fd, "journal.json", encoded, 0o600, lock_fd,
        death_boundary=death_boundary,
    )


def journal_body(journal: dict) -> dict:
    value = json.loads(json.dumps(journal))
    value.pop("updated_ns", None)
    value.pop("journal_sequence", None)
    return value


def journal_changes_only(current: dict, candidate: dict, fields: set[str]) -> bool:
    current_body = journal_body(current)
    candidate_body = journal_body(candidate)
    for field in fields:
        current_body.pop(field, None)
        candidate_body.pop(field, None)
    return current_body == candidate_body


def legal_journal_successor(current: dict, candidate: dict) -> bool:
    if (candidate["journal_sequence"] != current["journal_sequence"] + 1
            or candidate["updated_ns"] != current["updated_ns"] + 1):
        return False
    current_state = current["state"]
    candidate_state = candidate["state"]

    if current_state == "SNAPSHOTTING" and candidate_state == "SNAPSHOTTING":
        for group in ("authored", "generated"):
            other = "generated" if group == "authored" else "authored"
            legal_group_order = (
                group == "authored"
                and len(current["authored"]) < len(AUTHORED)
                and len(current["generated"]) == 0
            ) or (
                group == "generated"
                and len(current["authored"]) == len(AUTHORED)
                and len(current["generated"]) < len(GENERATED)
            )
            if (legal_group_order and current["installed_authored"] == 0
                    and "mutation_intent" not in current
                    and candidate[group][:-1] == current[group]
                    and len(candidate[group]) == len(current[group]) + 1
                    and candidate[other] == current[other]
                    and journal_changes_only(current, candidate, {group})):
                return True
        if ("mutation_intent" not in current and "mutation_intent" in candidate
                and candidate.get("installed_authored_intent") == current["installed_authored"] + 1
                and journal_changes_only(
                    current, candidate,
                    {"installed_authored_intent", "mutation_intent"},
                )):
            return True
        if ("mutation_intent" in current and "mutation_intent" not in candidate
                and current["mutation_intent"].get("action") == "install"
                and current["mutation_intent"].get("group") == "authored"
                and candidate["installed_authored"] == current["installed_authored"] + 1
                and "installed_authored_intent" not in candidate
                and journal_changes_only(
                    current, candidate,
                    {"installed_authored", "installed_authored_intent", "mutation_intent"},
                )):
            return True
    if (current_state == "SNAPSHOTTING" and candidate_state == "STAGED"
            and journal_changes_only(
                current, candidate,
                {"state", "stage_manifest_sha256", "expected_post"},
            )):
        return True
    if (current_state == "STAGED" and candidate_state == "INSTALLING(1)"
            and "mutation_intent" not in current and "mutation_intent" in candidate
            and journal_changes_only(current, candidate, {"state", "mutation_intent"})):
        return True
    if current_state.startswith("INSTALLING("):
        index = int(current_state.removeprefix("INSTALLING(").removesuffix(")"))
        if (candidate_state == current_state and "mutation_intent" in current
                and "mutation_intent" not in candidate
                and candidate["installed_generated"] == index
                and journal_changes_only(
                    current, candidate, {"installed_generated", "mutation_intent"},
                )):
            return True
        if ("mutation_intent" not in current and current["installed_generated"] == index
                and index < len(GENERATED)
                and candidate_state == f"INSTALLING({index + 1})"
                and "mutation_intent" in candidate
                and journal_changes_only(current, candidate, {"state", "mutation_intent"})):
            return True
        if ("mutation_intent" not in current and index == len(GENERATED)
                and current["installed_generated"] == index
                and candidate_state == "INSTALLED"
                and journal_changes_only(current, candidate, {"state"})):
            return True
    if (current_state == "INSTALLED" and candidate_state == "PROVING"
            and candidate.get("gates") == []
            and journal_changes_only(current, candidate, {"state", "gates"})):
        return True
    if current_state == "PROVING" and candidate_state == "PROVING":
        current_gates = current.get("gates", [])
        candidate_gates = candidate.get("gates", [])
        if (candidate_gates[:-1] == current_gates
                and len(candidate_gates) == len(current_gates) + 1
                and journal_changes_only(current, candidate, {"gates"})):
            return True
    if (current_state == "PROVING" and candidate_state == "COMMITTED"
            and len(current.get("gates", [])) == len(DELIVERY_GATES)
            and candidate.get("gates") == current.get("gates")
            and journal_changes_only(current, candidate, {"state"})):
        return True
    if (current_state == "COMMITTED" and candidate_state == "CLEANING"
            and journal_changes_only(current, candidate, {"state"})):
        return True
    if (current_state == "CLEANING" and candidate_state == "CLEANED"
            and journal_changes_only(current, candidate, {"state"})):
        return True

    preterminal = (current_state in {"SNAPSHOTTING", "STAGED", "INSTALLED", "PROVING"}
                   or current_state.startswith("INSTALLING("))
    if (preterminal and candidate_state == "ROLLING_BACK(1)"
            and "mutation_intent" in candidate
            and journal_changes_only(current, candidate, {"state", "mutation_intent"})):
        return True
    if current_state.startswith("ROLLING_BACK("):
        position = int(current_state.removeprefix("ROLLING_BACK(").removesuffix(")"))
        if (candidate_state == current_state and "mutation_intent" in current
                and "mutation_intent" not in candidate
                and journal_changes_only(current, candidate, {"mutation_intent"})):
            return True
        if ("mutation_intent" not in current and position < len(AUTHORED) + len(GENERATED)
                and candidate_state == f"ROLLING_BACK({position + 1})"
                and "mutation_intent" in candidate
                and journal_changes_only(current, candidate, {"state", "mutation_intent"})):
            return True
        if ("mutation_intent" not in current
                and position == len(AUTHORED) + len(GENERATED)
                and candidate_state == "RESTORED"
                and journal_changes_only(current, candidate, {"state"})):
            return True
    return (current_state == "RESTORED" and candidate_state == "FAILED"
            and journal_changes_only(current, candidate, {"state"}))


def reconcile_journal_temporary(delivery_fd: int, repo_fd: int,
                                expected_candidate_digest: str) -> None:
    temporary = ".journal.json.workflow-artifact.tmp"
    if entry_info_at(delivery_fd, temporary) is None:
        return
    require(entry_info_at(delivery_fd, "journal.json") is not None,
            "journal temporary without durable predecessor")
    current = read_journal(delivery_fd)
    validate_delivery_journal(
        current, delivery_fd, repo_fd, expected_candidate_digest,
        check_outside=False, allow_journal_temporary=True,
    )
    candidate = parse_journal(read_file_at(
        delivery_fd, temporary, DELIVERY_JOURNAL_MAX_BYTES, {0o600},
    ))
    validate_delivery_journal(
        candidate, delivery_fd, repo_fd, expected_candidate_digest,
        check_outside=False, allow_journal_temporary=True,
    )
    require(legal_journal_successor(current, candidate),
            "journal temporary legal successor")
    os.rename(temporary, "journal.json", src_dir_fd=delivery_fd, dst_dir_fd=delivery_fd)
    os.fsync(delivery_fd)


def delivery_lock(delivery_fd: int) -> int:
    flags = os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_CLOEXEC
    fd = os.open(".owner.lock", flags, 0o600, dir_fd=delivery_fd)
    info = os.fstat(fd)
    require(info.st_uid == os.getuid() and stat.S_ISREG(info.st_mode) and info.st_nlink == 1
            and stat.S_IMODE(info.st_mode) == 0o600, "delivery lock metadata")
    acquire_owner_lock(fd)
    return fd


def protected_records_at(repo_fd: int) -> dict[str, dict]:
    return {rel: file_record_at(repo_fd, rel) for rel in PROTECTED}


PROTECTED_ACTIVE_BYTES = b"workflow-artifact-identity\n"
PROTECTED_OBSERVATION_FIXTURE = (
    SCRIPT.parent / "fixtures"
    / "workflow-artifact-protected-workspace-observation-touched-files.md"
)


def install_live_protected_fixtures(root: Path) -> list[tuple[Path, bytes | None]]:
    """Ensure live protected paths match LIVE_PROTECTED_SHA256 for CI/local.

    Returns restorations as (path, previous_bytes_or_None_if_created).
    """
    restorations: list[tuple[Path, bytes | None]] = []
    observation_bytes = PROTECTED_OBSERVATION_FIXTURE.read_bytes()
    require(
        hashlib.sha256(observation_bytes).hexdigest()
        == LIVE_PROTECTED_SHA256[".devrites/work/workspace-observation/touched-files.md"],
        "protected observation fixture digest",
    )
    wanted = {
        ".devrites/ACTIVE": PROTECTED_ACTIVE_BYTES,
        ".devrites/work/workspace-observation/touched-files.md": observation_bytes,
    }
    for relative, content in wanted.items():
        dest = root / relative
        previous: bytes | None
        if dest.is_file() and not dest.is_symlink():
            previous = dest.read_bytes()
            if hashlib.sha256(previous).hexdigest() == LIVE_PROTECTED_SHA256[relative]:
                continue
        else:
            previous = None
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_bytes(content)
        restorations.append((dest, previous))
    return restorations


def restore_live_protected_fixtures(restorations: list[tuple[Path, bytes | None]]) -> None:
    for path, previous in reversed(restorations):
        if previous is None:
            path.unlink(missing_ok=True)
            # Remove empty parents we likely created under .devrites/work/...
            parent = path.parent
            while parent.name and parent != parent.parent:
                try:
                    parent.rmdir()
                except OSError:
                    break
                parent = parent.parent
        else:
            path.write_bytes(previous)


def require_live_protected_identity() -> dict[str, dict]:
    repo_fd = open_absolute_directory(project_root_for_tests(canonical_root()))
    try:
        records = protected_records_at(repo_fd)
    finally:
        os.close(repo_fd)
    observed = {relative: record.get("sha256") for relative, record in records.items()}
    require(observed == LIVE_PROTECTED_SHA256,
            f"current protected byte identity: got {observed}")
    return records


def aggregate_at(root_fd: int, paths: list[str]) -> str:
    digest = hashlib.sha256()
    for rel in paths:
        record = file_record_at(root_fd, rel)
        require(record["state"] == "present", f"candidate absent: {rel}")
        digest.update(len(rel.encode()).to_bytes(4, "big")); digest.update(rel.encode())
        digest.update(record["mode"].to_bytes(4, "big")); digest.update(bytes.fromhex(record["sha256"]))
    return digest.hexdigest()


def snapshot_one(repo_fd: int, delivery_fd: int, lock_fd: int, rel: str, group: str, index: int) -> dict:
    record = file_record_at(repo_fd, rel)
    record.update({"path": rel, "index": index})
    if record["state"] == "present":
        backup = f"backups/{group}/{index:08x}"
        data = read_file_at(repo_fd, rel, record["size"], {record["mode"]})
        atomic_write_at(delivery_fd, backup, data, 0o600, lock_fd)
        record["backup"] = backup
    return record


def snapshot_destination_record(record: dict) -> dict:
    if record["state"] == "absent":
        return {"state": "absent"}
    return {name: record[name] for name in ("state", "mode", "sha256", "size")}


def delivery_pair_states(journal: dict) -> dict[str, tuple[dict, dict]]:
    preimages = {
        relative: snapshot_destination_record(journal[group][index])
        for group, compiled in (("authored", AUTHORED), ("generated", GENERATED))
        for index, relative in enumerate(compiled)
    }
    if "expected_post" in journal:
        expected = {
            record["path"]: {name: record[name] for name in ("state", "mode", "sha256")}
            for record in journal["expected_post"]
        }
    else:
        candidate_fd = open_absolute_directory(Path(journal["candidate_root"]))
        try:
            expected = {relative: file_record_at(candidate_fd, relative) for relative in AUTHORED}
        finally:
            os.close(candidate_fd)
        expected.update({relative: preimages[relative] for relative in GENERATED})
    return {relative: (preimages[relative], expected[relative]) for relative in AUTHORED + GENERATED}


def destination_matches_record(actual: dict, expected: dict) -> bool:
    if actual["state"] != expected["state"]:
        return False
    if actual["state"] == "absent":
        return True
    fields = ("mode", "sha256", *(('size',) if "size" in expected else ()))
    return all(actual[field] == expected[field] for field in fields)


def require_destination_pair_state(repo_fd: int, relative: str,
                                   pair_states: dict[str, tuple[dict, dict]]) -> None:
    actual = file_record_at(repo_fd, relative)
    require(any(destination_matches_record(actual, expected) for expected in pair_states[relative]),
            f"delivery destination outside rollback pair: {relative}")


def require_destination_record(repo_fd: int, relative: str, expected: dict,
                               label: str) -> None:
    require(destination_matches_record(file_record_at(repo_fd, relative), expected),
            f"{label}: {relative}")


def validate_delivery_stage_records(journal: dict, staged: dict,
                                    current: dict) -> None:
    require(sha(json.dumps(staged, sort_keys=True).encode())
            == journal["stage_manifest_sha256"],
            "delivery stage manifest identity")
    admitted = {relative.removeprefix("pack/generated/") for relative in GENERATED}
    allowed_missing = set()
    intent = journal.get("mutation_intent")
    if (isinstance(intent, dict) and intent.get("action") in {"install", "restore"}
            and intent.get("group") == "generated"
            and intent.get("path") in GENERATED):
        allowed_missing.add(intent["path"].removeprefix("pack/generated/"))
    require(not (set(current) - set(staged))
            and set(staged) - set(current) <= allowed_missing,
            "delivery stage/current path set")
    expected = {
        record["path"].removeprefix("pack/generated/"): {
            "mode": record["mode"], "sha256": record["sha256"],
        }
        for record in journal["expected_post"]
        if record["path"] in GENERATED
    }
    require(set(expected) == admitted, "delivery stage admitted expected-post set")
    require(all(staged.get(relative) == expected[relative]
                for relative in admitted),
            "delivery stage admitted expected-post identity")
    require(all(staged[relative] == current[relative]
                for relative in set(staged) - admitted),
            "delivery stage non-admitted live identity")


def validate_delivery_stage_at(journal: dict, delivery_fd: int,
                               repo_fd: int) -> None:
    stage_info = entry_info_at(delivery_fd, "stage")
    require(stage_info is not None and stat.S_ISDIR(stage_info.st_mode),
            "staged delivery tree present")
    stage_fd = os.open("stage", DIRECTORY_FLAGS, dir_fd=delivery_fd)
    current_fd = open_dir_components(repo_fd, ("pack", "generated"))
    manifest_deadline = time.monotonic() + OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
    try:
        staged = generated_stage_manifest_at(stage_fd, manifest_deadline)
        current = generated_stage_manifest_at(current_fd, manifest_deadline)
    finally:
        os.close(current_fd); os.close(stage_fd)
    validate_delivery_stage_records(journal, staged, current)


def validate_delivery_state_effects(journal: dict, delivery_fd: int,
                                    repo_fd: int) -> None:
    state = journal["state"]
    if state == "SNAPSHOTTING":
        return
    pairs = delivery_pair_states(journal)
    if state in {"RESTORED", "FAILED"}:
        for relative, (preimage, _post) in pairs.items():
            require_destination_record(repo_fd, relative, preimage, "restored destination identity")
        return
    if state.startswith("ROLLING_BACK("):
        if "stage_manifest_sha256" in journal:
            validate_delivery_stage_at(journal, delivery_fd, repo_fd)
        return
    require("expected_post" in journal, "state effects require expected post")
    if state in {"STAGED"} or state.startswith("INSTALLING("):
        completed_generated = journal["installed_generated"]
        active_generated = None
        if state.startswith("INSTALLING(") and "mutation_intent" in journal:
            active_generated = journal["mutation_intent"]["index"]
        for relative in AUTHORED:
            require_destination_record(
                repo_fd, relative, pairs[relative][1], "staged authored identity",
            )
        for index, relative in enumerate(GENERATED):
            if index == active_generated:
                continue
            expected = pairs[relative][1] if index < completed_generated else pairs[relative][0]
            require_destination_record(repo_fd, relative, expected, "staged generated progression")
    else:
        for relative, (_preimage, post) in pairs.items():
            require_destination_record(repo_fd, relative, post, "expected-post destination identity")
    if state in {"STAGED", "INSTALLED", "PROVING", "COMMITTED"} or state.startswith("INSTALLING("):
        validate_delivery_stage_at(journal, delivery_fd, repo_fd)
    if state == "CLEANED":
        require(entry_info_at(delivery_fd, "stage") is None
                and entry_info_at(delivery_fd, "backups") is None,
                "cleaned delivery transaction trees absent")


def linked_file_at(root_fd: int, relative: str, limit: int = 16 * 1024 * 1024
                   ) -> tuple[dict, bytes, os.stat_result | None]:
    info = entry_info_at(root_fd, relative)
    if info is None:
        return {"state": "absent"}, b"", None
    require(info.st_uid == os.getuid() and stat.S_ISREG(info.st_mode)
            and info.st_nlink in {1, 2}, "recoverable linked regular file")
    parent_fd, name = open_parent_fd(root_fd, relative)
    try:
        fd = os.open(name, FILE_READ_FLAGS, dir_fd=parent_fd)
        try:
            opened = os.fstat(fd)
            require((opened.st_dev, opened.st_ino) == (info.st_dev, info.st_ino),
                    "stable linked regular descriptor")
            require(opened.st_size <= limit, "bounded linked regular file")
            data = read_fd_bounded(fd, limit)
        finally:
            os.close(fd)
    finally:
        os.close(parent_fd)
    return {
        "state": "present", "mode": stat.S_IMODE(info.st_mode),
        "sha256": sha(data), "size": len(data), "nlink": info.st_nlink,
        "device": info.st_dev, "inode": info.st_ino,
    }, data, info


def mutation_identity(journal: dict) -> tuple[str, str, int, str, str, str]:
    intent = journal.get("mutation_intent")
    require(isinstance(intent, dict)
            and set(intent) == {"action", "group", "index", "path"},
            "delivery mutation intent fields")
    action = intent["action"]
    group = intent["group"]
    index = intent["index"]
    require(action in {"install", "restore"} and group in {"authored", "generated"}
            and isinstance(index, int) and not isinstance(index, bool),
            "delivery mutation intent grammar")
    compiled = AUTHORED if group == "authored" else GENERATED
    require(0 <= index < len(compiled) and intent["path"] == compiled[index],
            "delivery mutation intent destination")
    if action == "install" and group == "authored":
        require(journal["state"] == "SNAPSHOTTING"
                and journal.get("installed_authored_intent") == index + 1
                and journal["installed_authored"] == index,
                "authored mutation intent state")
    elif action == "install":
        require(journal["state"] == f"INSTALLING({index + 1})"
                and journal["installed_generated"] == index,
                "generated mutation intent state")
    else:
        compiled_reverse = [
            (current_group, current_index, relative)
            for current_group, paths in (("authored", AUTHORED), ("generated", GENERATED))
            for current_index, relative in enumerate(paths)
        ][::-1]
        position = int(journal["state"].removeprefix("ROLLING_BACK(").removesuffix(")"))
        require(compiled_reverse[position - 1] == (group, index, intent["path"]),
                "rollback mutation intent state")
    token = f"{action}-{group}-{index:08x}"
    return action, group, index, intent["path"], f".mutation-{token}.claim", f".mutation-{token}.install"


def mutation_desired_record(journal: dict, group: str, index: int, action: str) -> dict:
    relative = (AUTHORED if group == "authored" else GENERATED)[index]
    if action == "restore":
        return snapshot_destination_record(journal[group][index])
    if group == "generated":
        return {name: journal["expected_post"][len(AUTHORED) + index][name]
                for name in ("state", "mode", "sha256")}
    candidate_fd = open_absolute_directory(Path(journal["candidate_root"]))
    try:
        return file_record_at(candidate_fd, relative)
    finally:
        os.close(candidate_fd)


def mutation_desired_bytes(journal: dict, delivery_fd: int, group: str,
                           index: int, action: str) -> tuple[dict, bytes | None]:
    desired = mutation_desired_record(journal, group, index, action)
    if desired["state"] == "absent":
        return desired, None
    if action == "restore":
        data = read_file_at(delivery_fd, f"backups/{group}/{index:08x}", 16 * 1024 * 1024, {0o600})
    elif group == "generated":
        stage_fd = os.open("stage", DIRECTORY_FLAGS, dir_fd=delivery_fd)
        try:
            relative = GENERATED[index].removeprefix("pack/generated/")
            data = read_file_at(stage_fd, relative, 16 * 1024 * 1024, {desired["mode"]})
        finally:
            os.close(stage_fd)
    else:
        candidate_fd = open_absolute_directory(Path(journal["candidate_root"]))
        try:
            data = read_file_at(candidate_fd, AUTHORED[index], 16 * 1024 * 1024, {desired["mode"]})
        finally:
            os.close(candidate_fd)
    require(sha(data) == desired["sha256"], "delivery mutation desired bytes")
    return desired, data


def mutation_record_matches(record: dict, expected: dict) -> bool:
    return destination_matches_record(record, expected)


def validate_mutation_artifacts(journal: dict, delivery_fd: int, repo_fd: int,
                                allow_journal_temporary: bool = False,
                                check_destination: bool = True) -> None:
    entries = set(os.listdir(delivery_fd))
    mutation_entries = {name for name in entries if name.startswith(".mutation-")}
    allowed_root = {
        ".owner.lock", "journal.json", OUTSIDE_MANIFEST_NAME, "backups", "stage",
    } | mutation_entries
    if allow_journal_temporary:
        allowed_root.add(JOURNAL_TEMPORARY)
    require(entries <= allowed_root, f"unknown delivery private entries: {sorted(entries - allowed_root)}")
    if "mutation_intent" not in journal:
        require(not mutation_entries, "mutation artifacts without durable intent")
        return
    action, group, index, relative, claim, install = mutation_identity(journal)
    require(mutation_entries <= {claim, install}, "mutation artifacts outside durable intent")
    pair = delivery_pair_states(journal)[relative]
    desired, desired_bytes = mutation_desired_bytes(journal, delivery_fd, group, index, action)
    claim_record, _claim_data, _claim_info = linked_file_at(delivery_fd, claim)
    if claim_record["state"] == "present":
        require(any(mutation_record_matches(claim_record, expected) for expected in pair),
                "mutation claim outside destination pair")
    install_record, install_data, install_info = linked_file_at(delivery_fd, install)
    if install_record["state"] == "present":
        require(desired_bytes is not None and desired_bytes.startswith(install_data),
                "mutation install temporary prefix")
        require(install_record["mode"] in {0o600, desired["mode"]},
                "mutation install temporary mode")
        if install_record["nlink"] == 2:
            require(install_data == desired_bytes and install_record["mode"] == desired["mode"],
                    "linked mutation install identity")
    elif desired["state"] == "absent":
        require(install_info is None, "absent mutation install temporary")
    current, _current_data, current_info = linked_file_at(repo_fd, relative)
    if check_destination:
        require((current["state"] == "absent" and claim_record["state"] == "present")
                or any(mutation_record_matches(current, expected) for expected in (*pair, desired)),
                f"delivery destination outside active mutation pair: {relative}")
        linked_private = [record for record in (claim_record, install_record)
                          if record.get("nlink") == 2]
        if linked_private:
            require(len(linked_private) == 1 and current_info is not None
                    and (current_info.st_dev, current_info.st_ino)
                    == (linked_private[0]["device"], linked_private[0]["inode"]),
                    "active mutation hard-link identity")


def prepare_mutation_install(delivery_fd: int, install: str,
                             desired: dict, data: bytes | None) -> None:
    if data is None:
        require(entry_info_at(delivery_fd, install) is None, "absent desired install temporary")
        return
    flags = os.O_RDWR | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC
    try:
        fd = os.open(install, flags, 0o600, dir_fd=delivery_fd)
        prefix = b""
    except FileExistsError:
        record, prefix, _info = linked_file_at(delivery_fd, install, len(data))
        require(record["state"] == "present" and data.startswith(prefix),
                "mutation install temporary retry prefix")
        if record["nlink"] == 2:
            require(prefix == data and record["mode"] == desired["mode"],
                    "linked install temporary complete")
            return
        fd = os.open(install, os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=delivery_fd)
        os.lseek(fd, len(prefix), os.SEEK_SET)
    try:
        complete_write(fd, data[len(prefix):])
        os.fchmod(fd, desired["mode"])
        os.fsync(fd)
    finally:
        os.close(fd)
    os.fsync(delivery_fd)


def restore_claim_no_replace(repo_fd: int, relative: str,
                             delivery_fd: int, claim: str) -> bool:
    claim_record, _data, claim_info = linked_file_at(delivery_fd, claim)
    require(claim_info is not None, "mutation claim to restore")
    parent_fd, name = open_parent_fd(repo_fd, relative, create=True)
    try:
        current = entry_info_at(repo_fd, relative)
        if current is None:
            os.link(claim, name, src_dir_fd=delivery_fd, dst_dir_fd=parent_fd,
                    follow_symlinks=False)
            os.fsync(parent_fd)
        elif (current.st_dev, current.st_ino) != (claim_info.st_dev, claim_info.st_ino):
            return False
        os.unlink(claim, dir_fd=delivery_fd)
        os.fsync(delivery_fd)
        return True
    finally:
        os.close(parent_fd)


def atomic_destination_mutation(repo_fd: int, delivery_fd: int,
                                journal: dict) -> None:
    validate_mutation_artifacts(
        journal, delivery_fd, repo_fd, check_destination=False,
    )
    action, group, index, relative, claim, install = mutation_identity(journal)
    pair = delivery_pair_states(journal)[relative]
    desired, desired_bytes = mutation_desired_bytes(journal, delivery_fd, group, index, action)
    prepare_mutation_install(delivery_fd, install, desired, desired_bytes)
    claim_record, _claim_data, claim_info = linked_file_at(delivery_fd, claim)
    current, _current_data, current_info = linked_file_at(repo_fd, relative)
    if claim_info is None and mutation_record_matches(current, desired):
        if entry_info_at(delivery_fd, install) is not None:
            os.unlink(install, dir_fd=delivery_fd); os.fsync(delivery_fd)
        return
    if claim_info is None and current_info is not None:
        require(entry_info_at(delivery_fd, claim) is None, "vacant private mutation claim")
        destination_parent, destination_name = open_parent_fd(repo_fd, relative)
        try:
            os.rename(destination_name, claim, src_dir_fd=destination_parent,
                      dst_dir_fd=delivery_fd)
            os.fsync(destination_parent); os.fsync(delivery_fd)
        finally:
            os.close(destination_parent)
        if action == "install" and group == "authored" and index == 0:
            delivery_test_mutation(repo_fd, "after-first-authored-claim", relative)
        if action == "install" and group == "generated" and index == 0:
            delivery_death("generated-0-claimed")
        claim_record, _claim_data, claim_info = linked_file_at(delivery_fd, claim)
    if claim_info is not None and not any(
            mutation_record_matches(claim_record, expected) for expected in pair):
        restored = restore_claim_no_replace(repo_fd, relative, delivery_fd, claim)
        require(restored, f"unowned destination preserved in private claim: {relative}")
        fail(f"delivery destination outside rollback pair: {relative}")
    current, _current_data, current_info = linked_file_at(repo_fd, relative)
    if claim_info is not None and current_info is not None:
        if (current_info.st_dev, current_info.st_ino) == (claim_info.st_dev, claim_info.st_ino):
            destination_parent, destination_name = open_parent_fd(repo_fd, relative)
            try:
                os.unlink(destination_name, dir_fd=destination_parent); os.fsync(destination_parent)
            finally:
                os.close(destination_parent)
        elif not mutation_record_matches(current, desired):
            fail(f"concurrent destination occupant preserved: {relative}")
    if desired_bytes is not None:
        install_record, _install_data, install_info = linked_file_at(delivery_fd, install, len(desired_bytes))
        require(install_info is not None and install_record["sha256"] == desired["sha256"]
                and install_record["mode"] == desired["mode"], "complete mutation install temporary")
        current, _current_data, current_info = linked_file_at(repo_fd, relative)
        destination_parent, destination_name = open_parent_fd(repo_fd, relative, create=True)
        try:
            if current_info is None:
                os.link(install, destination_name, src_dir_fd=delivery_fd,
                        dst_dir_fd=destination_parent, follow_symlinks=False)
                os.fsync(destination_parent)
            elif (current_info.st_dev, current_info.st_ino) != (install_info.st_dev, install_info.st_ino):
                fail(f"concurrent destination occupant preserved: {relative}")
        finally:
            os.close(destination_parent)
        os.unlink(install, dir_fd=delivery_fd); os.fsync(delivery_fd)
        require(destination_matches_record(file_record_at(repo_fd, relative), desired),
                f"atomic destination install readback: {relative}")
    else:
        require(entry_info_at(repo_fd, relative) is None,
                f"concurrent destination occupant preserved: {relative}")
    if entry_info_at(delivery_fd, claim) is not None:
        os.unlink(claim, dir_fd=delivery_fd); os.fsync(delivery_fd)
    require(not {name for name in os.listdir(delivery_fd) if name.startswith(".mutation-")},
            "completed mutation private cleanup")


def complete_pending_mutation(repo_fd: int, delivery_fd: int,
                              lock_fd: int, journal: dict,
                              expected_candidate_digest: str) -> None:
    require(journal.get("candidate_digest") == expected_candidate_digest,
            "pending mutation candidate identity")
    if "mutation_intent" not in journal:
        return
    action, group, index, _relative, _claim, _install = mutation_identity(journal)
    atomic_destination_mutation(repo_fd, delivery_fd, journal)
    if action == "install" and group == "authored":
        journal["installed_authored"] = index + 1
        journal.pop("installed_authored_intent", None)
    elif action == "install":
        journal["installed_generated"] = index + 1
    journal.pop("mutation_intent")
    write_journal(delivery_fd, lock_fd, journal)


def restore_all(repo_fd: int, delivery_fd: int, lock_fd: int, journal: dict,
                expected_candidate_digest: str) -> None:
    validate_delivery_journal(
        journal, delivery_fd, repo_fd, expected_candidate_digest,
        require_complete=True, check_outside=False,
    )
    if journal["state"] == "FAILED":
        validate_delivery_journal(
            journal, delivery_fd, repo_fd, expected_candidate_digest,
            require_complete=True,
        )
        return
    if journal["state"] == "RESTORED":
        validate_delivery_journal(
            journal, delivery_fd, repo_fd, expected_candidate_digest,
            require_complete=True,
        )
        journal["state"] = "FAILED"
        write_journal(delivery_fd, lock_fd, journal)
        delivery_death("rollback-failed-recorded")
        return
    complete_pending_mutation(
        repo_fd, delivery_fd, lock_fd, journal, expected_candidate_digest,
    )
    validate_delivery_journal(
        journal, delivery_fd, repo_fd, expected_candidate_digest,
        require_complete=True, check_outside=False,
    )
    compiled = [("authored", index, relative) for index, relative in enumerate(AUTHORED)]
    compiled += [("generated", index, relative) for index, relative in enumerate(GENERATED)]
    pair_states = delivery_pair_states(journal)
    for _group, _index, relative in compiled:
        require_destination_pair_state(repo_fd, relative, pair_states)
    for position, (group, index, relative) in enumerate(reversed(compiled), 1):
        journal["state"] = f"ROLLING_BACK({position})"
        journal["mutation_intent"] = {
            "action": "restore", "group": group, "index": index, "path": relative,
        }
        write_journal(delivery_fd, lock_fd, journal)
        delivery_death(f"rollback-{group}-{index}-intent")
        delivery_test_mutation(repo_fd, "before-rollback-claim", relative)
        atomic_destination_mutation(repo_fd, delivery_fd, journal)
        delivery_death(f"rollback-{group}-{index}-effect")
        journal.pop("mutation_intent")
        write_journal(delivery_fd, lock_fd, journal)
    journal["state"] = "RESTORED"
    write_journal(delivery_fd, lock_fd, journal)
    delivery_death("rollback-restored-recorded")
    validate_delivery_journal(
        journal, delivery_fd, repo_fd, expected_candidate_digest,
        require_complete=True,
    )
    journal["state"] = "FAILED"
    write_journal(delivery_fd, lock_fd, journal)
    delivery_death("rollback-failed-recorded")


def delivery_boundary_registry() -> tuple[str, ...]:
    boundaries = [
        f"bootstrap-{kind}-{site}"
        for kind in ("sidecar", "journal")
        for site in ("after-create", "after-partial", "after-sync", "before-rename")
    ]
    boundaries += ["journal-created"]
    boundaries += [f"snapshot-{group}-{index}-{site}"
                   for group, count in (("authored", len(AUTHORED)), ("generated", len(GENERATED)))
                   for index in range(count) for site in ("before", "backup", "recorded")]
    boundaries += [f"authored-{index}-{site}" for index in range(len(AUTHORED))
                   for site in ("intent", "effect", "recorded")]
    boundaries += ["install-enter", "stage-before", "stage-generated", "staged-recorded"]
    boundaries += [f"generated-{index}-{site}" for index in range(len(GENERATED))
                   for site in ("intent", "effect", "recorded")]
    boundaries += ["generated-0-claimed", "journal-replacement-after-sync"]
    boundaries += ["installed-recorded", "proving-recorded"]
    boundaries += [f"gate-{index}-{site}" for index in range(len(DELIVERY_GATES))
                   for site in ("before", "recorded")]
    boundaries += ["commit-before", "commit-recorded", "cleaning-recorded",
                   "cleanup-stage-removed", "cleanup-backups-removed", "cleaned-recorded"]
    boundaries += [f"rollback-{group}-{index}-{site}"
                   for group, count in (("authored", len(AUTHORED)), ("generated", len(GENERATED)))
                   for index in range(count) for site in ("intent", "effect")]
    boundaries += ["rollback-restored-recorded", "rollback-failed-recorded"]
    require(len(boundaries) == len(set(boundaries)), "delivery boundary registry uniqueness")
    return tuple(boundaries)


DELIVERY_BOUNDARIES = delivery_boundary_registry()


def delivery_death(boundary: str) -> None:
    require(boundary in DELIVERY_BOUNDARIES, f"unregistered delivery death boundary: {boundary}")
    if delivery_test_death_boundary() == boundary:
        os._exit(86)


def snapshot_or_reconcile(repo_fd: int, delivery_fd: int, lock_fd: int,
                          journal: dict, group: str, compiled: list[str], index: int) -> dict:
    relative = compiled[index]
    record = file_record_at(repo_fd, relative)
    record.update({"path": relative, "index": index})
    if record["state"] == "present":
        backup = f"backups/{group}/{index:08x}"
        try:
            existing = entry_info_at(delivery_fd, backup)
        except FileNotFoundError:
            existing = None
        if existing is None:
            data = read_file_at(repo_fd, relative, record["size"], {record["mode"]})
            atomic_write_at(delivery_fd, backup, data, 0o600, lock_fd)
        else:
            secure_file_info(existing, {0o600})
            data = read_file_at(delivery_fd, backup, record["size"], {0o600})
            require(len(data) == record["size"] and sha(data) == record["sha256"],
                    f"orphan delivery backup identity: {group}/{index}")
        record["backup"] = backup
    return record


def resume_delivery_prepare(repo_fd: int, root_fd: int, delivery_fd: int,
                            lock_fd: int, journal: dict,
                            expected_candidate_digest: str) -> dict:
    validate_delivery_journal(
        journal, delivery_fd, repo_fd, expected_candidate_digest,
        check_outside=False,
    )
    complete_pending_mutation(
        repo_fd, delivery_fd, lock_fd, journal, expected_candidate_digest,
    )
    validate_delivery_journal(
        journal, delivery_fd, repo_fd, expected_candidate_digest,
    )
    require(journal["state"] == "SNAPSHOTTING", f"prepare state: {journal['state']}")
    require(aggregate_at(root_fd, AUTHORED) == journal["candidate_digest"], "prepare candidate aggregate")
    driver = file_record_at(root_fd, "tests/workflow-artifact-identity-test.sh")
    require(driver["state"] == "present" and driver["mode"] == 0o755
            and driver["sha256"] == journal["driver_sha256"], "prepare driver identity")
    for group, compiled in (("authored", AUTHORED), ("generated", GENERATED)):
        for index, record in enumerate(journal[group]):
            relative = compiled[index]
            actual = file_record_at(repo_fd, relative)
            snapshot = {name: record[name] for name in ("state", "mode", "sha256", "size") if name in record}
            if group == "authored" and index < journal["installed_authored"]:
                expected = file_record_at(root_fd, relative)
                require(actual == expected, f"installed authored prefix identity: {index}")
            elif (group == "authored" and index == journal["installed_authored"]
                  and journal.get("installed_authored_intent") == index + 1):
                source = file_record_at(root_fd, relative)
                require(actual == snapshot or actual == source,
                        f"authored intent/effect identity: {index}")
            else:
                require(actual == snapshot, f"snapshot destination identity: {group}/{index}")
    for group, compiled in (("authored", AUTHORED), ("generated", GENERATED)):
        while len(journal[group]) < len(compiled):
            index = len(journal[group])
            delivery_death(f"snapshot-{group}-{index}-before")
            record = snapshot_or_reconcile(repo_fd, delivery_fd, lock_fd, journal, group, compiled, index)
            delivery_death(f"snapshot-{group}-{index}-backup")
            journal[group].append(record)
            write_journal(delivery_fd, lock_fd, journal)
            delivery_death(f"snapshot-{group}-{index}-recorded")
    validate_delivery_journal(
        journal, delivery_fd, repo_fd, expected_candidate_digest,
        require_complete=True,
    )
    while journal["installed_authored"] < len(AUTHORED):
        index = journal["installed_authored"]
        relative = AUTHORED[index]
        source_record = file_record_at(root_fd, relative)
        source = read_file_at(root_fd, relative, source_record["size"], {source_record["mode"]})
        intent = index + 1
        if journal.get("installed_authored_intent") != intent:
            journal["installed_authored_intent"] = intent
            journal["mutation_intent"] = {
                "action": "install", "group": "authored", "index": index, "path": relative,
            }
            write_journal(delivery_fd, lock_fd, journal)
        delivery_death(f"authored-{index}-intent")
        atomic_destination_mutation(repo_fd, delivery_fd, journal)
        delivery_death(f"authored-{index}-effect")
        require(file_record_at(repo_fd, relative) == source_record, f"authored readback: {relative}")
        journal["installed_authored"] = intent
        journal.pop("installed_authored_intent", None)
        journal.pop("mutation_intent")
        write_journal(delivery_fd, lock_fd, journal)
        delivery_death(f"authored-{index}-recorded")
    return journal


def initial_delivery_journal(root_fd: int, repo_fd: int,
                             candidate_digest: str,
                             outside_binding: dict) -> dict:
    driver_record = file_record_at(root_fd, "tests/workflow-artifact-identity-test.sh")
    require(driver_record["state"] == "present" and driver_record["mode"] == 0o755,
            "bootstrap driver mode")
    return {
        "contract": "devrites.workflow-artifact-delivery.v1",
        "state": "SNAPSHOTTING",
        "candidate_digest": candidate_digest,
        "candidate_root": str(CANDIDATE_ROOT.absolute()),
        "candidate_root_identity": {
            "device": os.fstat(root_fd).st_dev,
            "inode": os.fstat(root_fd).st_ino,
        },
        "driver_sha256": driver_record["sha256"],
        "authored_allowlist": AUTHORED,
        "generated_allowlist": GENERATED,
        "writer_allowlist_sha256": writer_allowlist_digest(),
        "protected": protected_records_at(repo_fd),
        "outside_manifest": outside_binding,
        "authored": [], "generated": [],
        "installed_authored": 0, "installed_generated": 0,
    }


def initial_journal_strict_prefix(raw: bytes, expected: dict) -> bool:
    probe = dict(expected)
    probe["updated_ns"] = 0
    probe["journal_sequence"] = 1
    canonical = (json.dumps(
        probe, sort_keys=True, separators=(",", ":"),
    ) + "\n").encode()
    tag = b'"updated_ns":'
    timestamp_start = canonical.index(tag) + len(tag)
    stable_before = canonical[:timestamp_start]
    stable_after = canonical[timestamp_start + 1:]
    if len(raw) <= len(stable_before):
        return stable_before.startswith(raw)
    if not raw.startswith(stable_before):
        return False
    remainder = raw[len(stable_before):]
    digit_count = 0
    while digit_count < len(remainder) and 48 <= remainder[digit_count] <= 57:
        digit_count += 1
    digits = remainder[:digit_count]
    if not digits or digits[:1] == b"0":
        return False
    suffix = remainder[digit_count:]
    return len(suffix) < len(stable_after) and stable_after.startswith(suffix)


def reconcile_initial_journal_temporary(delivery_fd: int, repo_fd: int,
                                        expected_candidate_digest: str,
                                        expected: dict) -> dict | None:
    entries = require_prejournal_private_inventory(delivery_fd)
    require("journal.json" not in entries, "initial journal durable ambiguity")
    if JOURNAL_TEMPORARY not in entries:
        return None
    fd = os.open(JOURNAL_TEMPORARY, os.O_RDWR | os.O_NOFOLLOW | os.O_CLOEXEC,
                 dir_fd=delivery_fd)
    try:
        opened = os.fstat(fd)
        require(opened.st_uid == os.getuid() and stat.S_ISREG(opened.st_mode)
                and opened.st_nlink == 1 and stat.S_IMODE(opened.st_mode) == 0o600
                and opened.st_size <= DELIVERY_JOURNAL_MAX_BYTES,
                "initial journal temporary metadata")
        raw = read_fd_bounded(fd, DELIVERY_JOURNAL_MAX_BYTES)
        pathname = os.stat(JOURNAL_TEMPORARY, dir_fd=delivery_fd,
                           follow_symlinks=False)
        require((opened.st_dev, opened.st_ino) == (pathname.st_dev, pathname.st_ino),
                "initial journal temporary pathname identity")
        if raw.endswith(b"\n"):
            candidate = parse_journal(raw)
            canonical = (json.dumps(
                candidate, sort_keys=True, separators=(",", ":"),
            ) + "\n").encode()
            require(canonical == raw, "initial journal temporary canonical bytes")
            require(candidate.get("journal_sequence") == 1
                    and journal_body(candidate) == expected,
                    "initial journal temporary exact intent")
            validate_delivery_journal(
                candidate, delivery_fd, repo_fd, expected_candidate_digest,
                allow_journal_temporary=True,
            )
            os.rename(JOURNAL_TEMPORARY, "journal.json",
                      src_dir_fd=delivery_fd, dst_dir_fd=delivery_fd)
            os.fsync(delivery_fd)
            return candidate
        require(initial_journal_strict_prefix(raw, expected),
                "initial journal temporary partial prefix")
        pathname = os.stat(JOURNAL_TEMPORARY, dir_fd=delivery_fd,
                           follow_symlinks=False)
        require((opened.st_dev, opened.st_ino) == (pathname.st_dev, pathname.st_ino),
                "initial journal temporary unlink identity")
        os.unlink(JOURNAL_TEMPORARY, dir_fd=delivery_fd)
        os.fsync(delivery_fd)
        return None
    finally:
        os.close(fd)


def delivery_prepare() -> None:
    reject_delivery_fixture_environment()
    repo_env = os.environ.get("DEVRITES_REPO_ROOT")
    require(repo_env is not None, "DEVRITES_REPO_ROOT is required")
    repo = Path(repo_env).absolute()
    root = canonical_root().absolute()
    repo_fd = open_absolute_directory(repo)
    root_fd = open_absolute_directory(root)
    parent_relative = ".devrites/work/workflow-artifact-identity/.generated-install"
    parent_fd = open_dir_components(repo_fd, relative_components(parent_relative), create=True)
    os.fchmod(parent_fd, 0o700)
    candidate_digest = aggregate_at(root_fd, AUTHORED)
    try:
        try:
            os.mkdir(candidate_digest, 0o700, dir_fd=parent_fd)
            os.fsync(parent_fd)
        except FileExistsError:
            pass
        delivery_fd = os.open(candidate_digest, DIRECTORY_FLAGS, dir_fd=parent_fd)
        validate_directory_fd(delivery_fd, 0o700)
        delivery = repo / parent_relative / candidate_digest
        lock_fd = delivery_lock(delivery_fd)
        validated = False
        try:
            _reconcile_gate_proof_cache(delivery_fd)
            entries = set(os.listdir(delivery_fd))
            if "journal.json" in entries:
                require(OUTSIDE_MANIFEST_NAME in entries
                        and OUTSIDE_MANIFEST_TEMPORARY not in entries,
                        "durable journal outside manifest ambiguity")
                reconcile_journal_temporary(delivery_fd, repo_fd, candidate_digest)
                journal = validate_delivery_journal(
                    read_journal(delivery_fd), delivery_fd, repo_fd,
                    candidate_digest, check_outside=False,
                )
                validated = True
                if journal["state"] != "SNAPSHOTTING":
                    print(f"DEVRITES_DELIVERY_DIR={delivery}")
                    print(f"delivery_state={journal['state']}")
                    return
            else:
                outside_binding = create_or_reuse_outside_manifest(
                    repo_fd, delivery_fd, lock_fd, candidate_digest,
                )
                expected = initial_delivery_journal(
                    root_fd, repo_fd, candidate_digest, outside_binding,
                )
                journal = reconcile_initial_journal_temporary(
                    delivery_fd, repo_fd, candidate_digest, expected,
                )
                if journal is None:
                    journal = dict(expected)
                    write_journal(delivery_fd, lock_fd, journal, bootstrap=True)
                validate_delivery_journal(
                    journal, delivery_fd, repo_fd, candidate_digest,
                )
                validated = True
                delivery_test_mutation(repo_fd, "after-journal-created")
                delivery_death("journal-created")
            journal = resume_delivery_prepare(
                repo_fd, root_fd, delivery_fd, lock_fd, journal, candidate_digest,
            )
            require(file_record_at(repo_fd, "tests/workflow-artifact-identity-test.sh")["sha256"] == journal["driver_sha256"], "bootstrap driver hash")
            print(f"DEVRITES_DELIVERY_DIR={delivery}")
            print(f"delivery_state={journal['state']}")
            print(f"driver_sha256={journal['driver_sha256']}")
        except BaseException:
            if validated and entry_info_at(delivery_fd, "journal.json") is not None:
                current = read_journal(delivery_fd)
                if (len(current.get("authored", [])) == len(AUTHORED)
                        and len(current.get("generated", [])) == len(GENERATED)
                        and (current.get("installed_authored", 0) > 0
                             or "mutation_intent" in current)):
                    restore_all(
                        repo_fd, delivery_fd, lock_fd, current, candidate_digest,
                    )
            raise
        finally:
            os.close(lock_fd); os.close(delivery_fd)
    finally:
        os.close(parent_fd); os.close(root_fd); os.close(repo_fd)


def generated_stage_manifest_at(stage_fd: int, deadline: float) -> dict:
    rows = {}

    def check_deadline() -> None:
        require_outside_manifest_deadline(deadline)

    def path_info(directory_fd: int, name: str) -> os.stat_result:
        check_deadline()
        info = os.stat(name, dir_fd=directory_fd, follow_symlinks=False)
        check_deadline()
        return info

    def require_path_identity(before: os.stat_result, after: os.stat_result) -> None:
        require(manifest_identity(before) == manifest_identity(after),
                "generated stage pathname replacement")

    def visit(directory_fd: int, prefix: str) -> None:
        check_deadline()
        names = sorted(os.listdir(directory_fd))
        check_deadline()
        for name in names:
            check_deadline()
            relative = f"{prefix}/{name}" if prefix else name
            before = path_info(directory_fd, name)
            if stat.S_ISDIR(before.st_mode):
                check_deadline()
                child_fd = os.open(name, DIRECTORY_FLAGS, dir_fd=directory_fd)
                check_deadline()
                try:
                    opened = os.fstat(child_fd)
                    validate_directory_fd(child_fd)
                    require_path_identity(before, opened)
                    visit(child_fd, relative)
                    require_path_identity(opened, path_info(directory_fd, name))
                finally:
                    os.close(child_fd)
            else:
                secure_file_info(before)
                check_deadline()
                fd = os.open(name, FILE_READ_FLAGS, dir_fd=directory_fd)
                check_deadline()
                try:
                    opened = os.fstat(fd)
                    secure_file_info(opened)
                    require_path_identity(before, opened)
                    require(opened.st_size <= OUTSIDE_MANIFEST_MAX_BYTES,
                            "bounded file size")
                    digest, size = hash_file_descriptor(
                        fd, OUTSIDE_MANIFEST_MAX_BYTES, deadline,
                    )
                    require(size == opened.st_size,
                            "generated stage descriptor mutation")
                    require_path_identity(opened, path_info(directory_fd, name))
                finally:
                    os.close(fd)
                rows[relative] = {
                    "mode": stat.S_IMODE(opened.st_mode), "sha256": digest,
                }
        check_deadline()

    visit(stage_fd, "")
    check_deadline()
    return rows


def run_delivery_process(command: list[str], expected: str | None,
                         aggregate_deadline: float, env: dict[str, str],
                         preexec_fn, pass_fds: tuple[int, ...]) -> tuple[bool, str, bytes]:
    remaining = aggregate_deadline - time.monotonic()
    if remaining <= DELIVERY_TERMINATE_GRACE_SECONDS:
        return False, "aggregate-timeout", b""
    limited_by_aggregate = remaining < DELIVERY_PROCESS_TIMEOUT_SECONDS
    timeout = min(DELIVERY_PROCESS_TIMEOUT_SECONDS, remaining)
    output_limit = DELIVERY_OUTPUT_LIMIT_BYTES
    if (delivery_test_fast_fixture()
            and delivery_test_mutation_name() == "generator-overflow"):
        output_limit = 256
    ok, reason, output = run_proof_command(
        command, expected, timeout, DELIVERY_TERMINATE_GRACE_SECONDS,
        output_limit, env=env, preexec_fn=preexec_fn,
        pass_fds=pass_fds,
    )
    if reason == "timeout":
        reason = "aggregate-timeout" if limited_by_aggregate else "process-timeout"
    elif ok and time.monotonic() > aggregate_deadline:
        return False, "aggregate-timeout", output
    return ok, reason, output


def delivery_test_mutation(repo_fd: int, site: str, relative: str | None = None) -> None:
    mutation = delivery_test_mutation_name()
    if mutation is None:
        return
    require(delivery_test_fast_fixture(),
            "delivery test mutation requires fast fixture")
    allowed = {"late-lookalike", "late-nested-git", "generator-overflow",
               "generated-race", "rollback-race", "prepare-outside-drift",
               "unrelated-directory-mode", "gate-failure",
               "first-authored-claim-exception"}
    require(mutation in allowed, "known delivery test mutation")
    create_at = {
        "late-lookalike": ".generated-install-evil/deep/observed",
        "late-nested-git": "nested/.git/deep/observed",
    }
    if mutation in create_at and site == "gate-0":
        destination = create_at[mutation]
        parent_fd, name = open_parent_fd(repo_fd, destination, create=True)
        try:
            fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                         0o600, dir_fd=parent_fd)
            try:
                complete_write(fd, mutation.encode() + b"\n"); os.fsync(fd)
            finally:
                os.close(fd)
            os.fsync(parent_fd)
        finally:
            os.close(parent_fd)
    if mutation == "prepare-outside-drift" and site == "after-journal-created":
        remove_tree_at(repo_fd, ".drift-017-prepare-outside")
    if (mutation == "first-authored-claim-exception"
            and site == "after-first-authored-claim"):
        raise RuntimeError("injected first authored claim exception")
    if mutation == "unrelated-directory-mode" and site == "gate-0":
        directory_fd = open_dir_components(repo_fd, (".outside-directory",))
        try:
            os.fchmod(directory_fd, 0o755)
            os.fsync(directory_fd)
        finally:
            os.close(directory_fd)
    race = ((mutation == "generated-race" and site == "before-generated-claim")
            or (mutation == "rollback-race" and site == "before-rollback-claim"))
    if race:
        require(relative is not None, "delivery race destination")
        parent_fd, name = open_parent_fd(repo_fd, relative)
        temporary = f".{name}.drift-016-race"
        try:
            fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                         0o600, dir_fd=parent_fd)
            try:
                complete_write(fd, mutation.encode() + b"\n"); os.fsync(fd)
            finally:
                os.close(fd)
            os.rename(temporary, name, src_dir_fd=parent_fd, dst_dir_fd=parent_fd)
            os.fsync(parent_fd)
        finally:
            os.close(parent_fd)


def _set_generator_child_umask() -> None:
    os.umask(0o022)


def _copy_named_regular_file(src_dir_fd: int, dst_dir_fd: int, name: str) -> None:
    try:
        info = os.stat(name, dir_fd=src_dir_fd, follow_symlinks=False)
    except FileNotFoundError:
        return
    require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1, "held generator script")
    mode = stat.S_IMODE(info.st_mode)
    src = os.open(name, FILE_READ_FLAGS, dir_fd=src_dir_fd)
    try:
        data = read_fd_bounded(src, OUTSIDE_MANIFEST_MAX_BYTES)
    finally:
        os.close(src)
    dst = os.open(
        name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
        mode, dir_fd=dst_dir_fd,
    )
    try:
        complete_write(dst, data)
        os.fchmod(dst, mode)
        os.fsync(dst)
    finally:
        os.close(dst)


def _copy_directory_tree(src_dir_fd: int, dst_dir_fd: int) -> None:
    for name in os.listdir(src_dir_fd):
        info = os.stat(name, dir_fd=src_dir_fd, follow_symlinks=False)
        if stat.S_ISDIR(info.st_mode):
            mode = stat.S_IMODE(info.st_mode)
            os.mkdir(name, mode, dir_fd=dst_dir_fd)
            os.chmod(name, mode, dir_fd=dst_dir_fd, follow_symlinks=False)
            child_src = os.open(name, DIRECTORY_FLAGS, dir_fd=src_dir_fd)
            try:
                child_dst = os.open(name, DIRECTORY_FLAGS, dir_fd=dst_dir_fd)
                try:
                    _copy_directory_tree(child_src, child_dst)
                finally:
                    os.close(child_dst)
            finally:
                os.close(child_src)
        elif stat.S_ISREG(info.st_mode):
            _copy_named_regular_file(src_dir_fd, dst_dir_fd, name)
        else:
            fail("held generator source type")


def _prepare_held_generator_view(repo_fd: int, stage_fd: int, stage_relative: str) -> tuple[int, int]:
    relative_components(stage_relative)
    os.mkdir(HELD_GENERATOR_OUTPUT, 0o700, dir_fd=stage_fd)
    os.fsync(stage_fd)
    held_out_fd = os.open(HELD_GENERATOR_OUTPUT, DIRECTORY_FLAGS, dir_fd=stage_fd)
    output_fd = -1
    try:
        validate_directory_fd(held_out_fd, 0o700)
        os.mkdir(HELD_GENERATOR_ARTIFACTS, 0o700, dir_fd=held_out_fd)
        os.fsync(held_out_fd)
        output_fd = os.open(HELD_GENERATOR_ARTIFACTS, DIRECTORY_FLAGS, dir_fd=held_out_fd)
        validate_directory_fd(output_fd, 0o700)
        try:
            pack_src = os.open("pack", DIRECTORY_FLAGS, dir_fd=repo_fd)
        except FileNotFoundError:
            pack_src = -1
        try:
            if pack_src >= 0:
                try:
                    claude_src = os.open(".claude", DIRECTORY_FLAGS, dir_fd=pack_src)
                except FileNotFoundError:
                    claude_src = -1
                if claude_src >= 0:
                    try:
                        os.mkdir("pack", 0o700, dir_fd=output_fd)
                        pack_dst = os.open("pack", DIRECTORY_FLAGS, dir_fd=output_fd)
                        try:
                            os.mkdir(".claude", 0o700, dir_fd=pack_dst)
                            claude_dst = os.open(".claude", DIRECTORY_FLAGS, dir_fd=pack_dst)
                            try:
                                _copy_directory_tree(claude_src, claude_dst)
                            finally:
                                os.close(claude_dst)
                        finally:
                            os.close(pack_dst)
                    finally:
                        os.close(claude_src)
        finally:
            if pack_src >= 0:
                os.close(pack_src)
        os.mkdir("scripts", 0o700, dir_fd=output_fd)
        src_scripts = os.open("scripts", DIRECTORY_FLAGS, dir_fd=repo_fd)
        try:
            dst_scripts = os.open("scripts", DIRECTORY_FLAGS, dir_fd=output_fd)
            try:
                for name in ("build-host-artifacts.sh", "codex-generate.sh"):
                    _copy_named_regular_file(src_scripts, dst_scripts, name)
            finally:
                os.close(dst_scripts)
        finally:
            os.close(src_scripts)
        for name in ("claude", "codex"):
            os.mkdir(name, 0o700, dir_fd=output_fd)
            host_fd = os.open(name, DIRECTORY_FLAGS, dir_fd=output_fd)
            try:
                validate_directory_fd(host_fd, 0o700)
            finally:
                os.close(host_fd)
        os.fsync(output_fd)
        os.fsync(held_out_fd)
        return held_out_fd, output_fd
    except BaseException:
        if output_fd >= 0:
            os.close(output_fd)
        os.close(held_out_fd)
        raise


def _clear_held_generator_view(stage_fd: int, held_out_fd: int, output_fd: int, hoist: bool) -> None:
    if hoist:
        for name in os.listdir(output_fd):
            if name in {"pack", "scripts"}:
                continue
            os.rename(name, name, src_dir_fd=output_fd, dst_dir_fd=stage_fd)
    for name in os.listdir(held_out_fd):
        info = os.stat(name, dir_fd=held_out_fd, follow_symlinks=False)
        if stat.S_ISDIR(info.st_mode):
            remove_tree_at(held_out_fd, name)
        else:
            if not stat.S_ISLNK(info.st_mode):
                secure_file_info(info)
            os.unlink(name, dir_fd=held_out_fd)
    output_info = entry_info_at(stage_fd, HELD_GENERATOR_OUTPUT)
    if output_info is not None:
        if stat.S_ISLNK(output_info.st_mode):
            os.unlink(HELD_GENERATOR_OUTPUT, dir_fd=stage_fd)
        else:
            require(stat.S_ISDIR(output_info.st_mode), "held generator output")
            os.rmdir(HELD_GENERATOR_OUTPUT, dir_fd=stage_fd)
    if entry_info_at(stage_fd, "pack") is not None:
        remove_tree_at(stage_fd, "pack")
    if entry_info_at(stage_fd, "scripts") is not None:
        remove_tree_at(stage_fd, "scripts")
    os.fsync(stage_fd)


def run_private_generator(repo_fd: int, stage_relative: str,
                          aggregate_deadline: float) -> tuple[bool, str, bytes]:
    relative_components(stage_relative)
    env = os.environ.copy()
    env["PYTHONDONTWRITEBYTECODE"] = "1"
    if delivery_test_fast_fixture():
        def fixture_preexec() -> None:
            _set_generator_child_umask()
            os.fchdir(repo_fd)
        stage_paths = [relative.removeprefix("pack/generated/") for relative in GENERATED]
        if delivery_test_mutation_name() == "generator-overflow":
            program = (
                "import os,pathlib,sys,time;stage=pathlib.Path(sys.argv[1]);"
                "(stage/'generator-pgid').write_text(str(os.getpgrp()));"
                "os.write(1,b'x'*300);time.sleep(30)"
            )
        else:
            program = (
                "import json,pathlib,shutil,sys;"
                "stage=pathlib.Path(sys.argv[1]);"
                "shutil.copytree('pack/generated',stage,dirs_exist_ok=True);"
                "[(stage/rel).write_bytes((stage/rel).read_bytes()+b'changed-generated-stage\\n') "
                "for rel in json.loads(sys.argv[2])]"
            )
        return run_delivery_process(
            [sys.executable, "-c", program, stage_relative, json.dumps(stage_paths)],
            None, aggregate_deadline, env, fixture_preexec, (repo_fd,),
        )
    parent_fd, name = open_parent_fd(repo_fd, stage_relative)
    try:
        stage_fd = os.open(name, DIRECTORY_FLAGS, dir_fd=parent_fd)
    finally:
        os.close(parent_fd)
    generated_ok = False
    held_out_fd = -1
    output_fd = -1
    try:
        validate_directory_fd(stage_fd)
        held_out_fd, output_fd = _prepare_held_generator_view(repo_fd, stage_fd, stage_relative)
        def generator_preexec() -> None:
            _set_generator_child_umask()
            os.fchdir(output_fd)
        bash_env_read, bash_env_write = os.pipe()
        try:
            try:
                complete_write(
                    bash_env_write,
                    b'pwd() { [ "$#" -eq 1 ] && [ "$1" = -P ] && { printf ".\\n"; return; }; builtin pwd "$@"; }\n'
                    b'rm() {\n'
                    b'  if [ "$#" -eq 2 ] && { [ "$1" = -rf ] || [ "$1" = -fr ]; } && [ "$2" = . ]; then\n'
                    b'    for name in .* *; do\n'
                    b'      if [ "$name" = . ] || [ "$name" = .. ]; then\n'
                    b'        continue\n'
                    b'      fi\n'
                    b'      if { [ "$name" = pack ] || [ "$name" = scripts ] '
                    b'|| [ "$name" = claude ] || [ "$name" = codex ]; } '
                    b'&& [ ! -L "$name" ]; then\n'
                    b'        continue\n'
                    b'      fi\n'
                    b'      if [ -e "$name" ] || [ -L "$name" ]; then\n'
                    b'        command rm -rf -- "$name"\n'
                    b'      fi\n'
                    b'    done\n'
                    b'    return 0\n'
                    b'  fi\n'
                    b'  command rm "$@"\n'
                    b'}\n'
                    b'mkdir() {\n'
                    b'  for _root in claude codex; do\n'
                    b'    if [ -L "$_root" ]; then\n'
                    b'      command rm -f -- "$_root"\n'
                    b'    fi\n'
                    b'  done\n'
                    b'  command mkdir "$@" || return $?\n'
                    b'  for _root in claude codex; do\n'
                    b'    if [ -L "$_root" ]; then\n'
                    b'      echo "held-generator: refusing symlink plant at $_root" >&2\n'
                    b'      return 1\n'
                    b'    fi\n'
                    b'  done\n'
                    b'  return 0\n'
                    b'}\n'
                    b'cp() {\n'
                    b'  _dest=\n'
                    b'  for _a in "$@"; do\n'
                    b'    _dest="$_a"\n'
                    b'  done\n'
                    b'  case "$_dest" in\n'
                    b'    claude|codex|claude/*|codex/*|./claude|./codex|./claude/*|./codex/*)\n'
                    b'      _head="${_dest#./}"\n'
                    b'      _head="${_head%%/*}"\n'
                    b'      if [ -L "$_head" ]; then\n'
                    b'        echo "held-generator: refusing cp through symlink $_head" >&2\n'
                    b'        return 1\n'
                    b'      fi\n'
                    b'      ;;\n'
                    b'  esac\n'
                    b'  command cp "$@"\n'
                    b'}\n',
                )
            finally:
                os.close(bash_env_write)
                bash_env_write = -1
            env["BASH_ENV"] = f"/dev/fd/{bash_env_read}"
            env["DEVRITES_HOST_ARTIFACT_DIR"] = "."
            generated_ok, reason, output = run_delivery_process(
                with_delivery_execution_prefix(["bash", "scripts/build-host-artifacts.sh"]),
                None, aggregate_deadline, env, generator_preexec,
                (repo_fd, stage_fd, held_out_fd, output_fd, bash_env_read),
            )
            return generated_ok, reason, output
        finally:
            if bash_env_write >= 0:
                os.close(bash_env_write)
            os.close(bash_env_read)
    finally:
        try:
            if held_out_fd >= 0 and output_fd >= 0:
                _clear_held_generator_view(stage_fd, held_out_fd, output_fd, hoist=generated_ok)
        finally:
            if output_fd >= 0:
                os.close(output_fd)
            if held_out_fd >= 0:
                os.close(held_out_fd)
            os.close(stage_fd)


def _validate_gate_proof_cache_relative(relative: str) -> None:
    components = relative_components(relative)
    require(components[:4] == (".devrites", "work", "workflow-artifact-identity", ".generated-install")
            and len(components) == 6
            and re.fullmatch(r"[0-9a-f]{64}", components[4]) is not None
            and components[5] == "proof-cache",
            "delivery proof cache path")


def _set_gate_proof_cache_env(env: dict[str, str], relative: str) -> None:
    env["PYTHONPYCACHEPREFIX"] = relative


def _reconcile_gate_proof_cache(delivery_fd: int) -> None:
    info = entry_info_at(delivery_fd, "proof-cache")
    if info is None:
        return
    require(info.st_uid == os.getuid() and stat.S_ISDIR(info.st_mode)
            and stat.S_IMODE(info.st_mode) == 0o700,
            "delivery proof cache metadata")
    remove_tree_at(delivery_fd, "proof-cache")


def _prepare_gate_proof_cache(delivery_fd: int, relative: str) -> None:
    _validate_gate_proof_cache_relative(relative)
    require(entry_info_at(delivery_fd, "proof-cache") is None,
            "delivery proof cache starts absent")
    os.mkdir("proof-cache", 0o700, dir_fd=delivery_fd)
    os.fsync(delivery_fd)
    cache_fd = os.open("proof-cache", DIRECTORY_FLAGS, dir_fd=delivery_fd)
    try:
        validate_directory_fd(cache_fd, 0o700)
    finally:
        os.close(cache_fd)


def run_gate(repo_fd: int, delivery_fd: int, proof_cache_relative: str,
             command: list[str], expected: str | None,
             aggregate_deadline: float, gate_index: int) -> dict:
    delivery_test_mutation(repo_fd, f"gate-{gate_index}")
    execution_command = command
    if delivery_test_fast_fixture():
        execution_command = [
            sys.executable, "-c",
            "import sys;print(sys.argv[1]) if sys.argv[1] else None", expected or "",
        ]
    env = os.environ.copy()
    env.pop("DEVRITES_REPO_ROOT", None)
    env["PYTHONDONTWRITEBYTECODE"] = "1"
    _prepare_gate_proof_cache(delivery_fd, proof_cache_relative)
    try:
        _set_gate_proof_cache_env(env, proof_cache_relative)
        ok, reason, output = run_delivery_process(
            with_delivery_execution_prefix(list(execution_command)), expected, aggregate_deadline,
            env, lambda: os.fchdir(repo_fd), (repo_fd,),
        )
    finally:
        _reconcile_gate_proof_cache(delivery_fd)
    if (gate_index == 0
            and delivery_test_mutation_name() == "gate-failure"):
        ok, reason = False, "wrong-signal"
    if not ok:
        raise RuntimeError(f"delivery gate-{gate_index} failed: {reason}")
    return {
        "command": command, "execution_prefix": delivery_execution_prefix(),
        "sha256": sha(output), "signal": expected or "exit=0",
    }


def cleanup_delivery(delivery_fd: int, lock_fd: int, journal: dict) -> None:
    journal["state"] = "CLEANING"
    write_journal(delivery_fd, lock_fd, journal)
    delivery_death("cleaning-recorded")
    remove_tree_at(delivery_fd, "stage", missing_ok=True)
    delivery_death("cleanup-stage-removed")
    remove_tree_at(delivery_fd, "backups", missing_ok=True)
    delivery_death("cleanup-backups-removed")
    require(entry_info_at(delivery_fd, "stage") is None and entry_info_at(delivery_fd, "backups") is None, "delivery cleanup readback")
    journal["state"] = "CLEANED"
    write_journal(delivery_fd, lock_fd, journal)
    delivery_death("cleaned-recorded")


def open_delivery(repo_fd: int, repo: Path, delivery_arg: str) -> tuple[int, Path]:
    parent_relative = ".devrites/work/workflow-artifact-identity/.generated-install"
    supplied = Path(delivery_arg).absolute()
    require(supplied.parent == repo / parent_relative and re.fullmatch(r"[0-9a-f]{64}", supplied.name) is not None, "delivery directory authority")
    parent_fd = open_dir_components(repo_fd, relative_components(parent_relative))
    try:
        delivery_fd = os.open(supplied.name, DIRECTORY_FLAGS, dir_fd=parent_fd)
    finally:
        os.close(parent_fd)
    validate_directory_fd(delivery_fd, 0o700)
    return delivery_fd, supplied


def require_pre_generated_install_state(repo_fd: int, journal: dict) -> None:
    pairs = delivery_pair_states(journal)
    for relative in AUTHORED:
        actual = file_record_at(repo_fd, relative)
        require(destination_matches_record(actual, pairs[relative][1]),
                f"authored pre-generated identity: {relative}")
    for relative in GENERATED:
        actual = file_record_at(repo_fd, relative)
        require(destination_matches_record(actual, pairs[relative][0]),
                f"generated preinstall identity: {relative}")


def check_empty_generated_delta_install() -> None:
    require(
        LIVE_PROTECTED_SHA256[".devrites/ACTIVE"]
        == "fc0dd2b2c697c0701083bd82d3cf1db569478d474ab3755e1b65eb140c366267",
        "live ACTIVE pin",
    )
    empty_require = "        require(differences <= allowed_stage,\n"
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        create_actual_delivery_repo(repo, full_generator=True)
        repo_fd = open_absolute_directory(repo)
        try:
            protected_before = protected_records_at(repo_fd)
        finally:
            os.close(repo_fd)
        prepared = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], timeout=120, fast_fixture=False,
        )
        require(prepared.returncode == 0,
                f"empty-delta prepare: {prepared.stdout[-1000:]}")
        delivery = actual_delivery_directory(repo)
        staged = run_actual_delivery_mode(
            repo, ["--delivery-install", str(delivery)], "staged-recorded",
            timeout=180, fast_fixture=False,
        )
        require(staged.returncode == 86,
                f"empty-delta stage: {staged.stdout[-1000:]}")
        journal = json.loads((delivery / "journal.json").read_text())
        require(journal["state"] == "STAGED", "empty-delta STAGED")
        stage_fd = open_absolute_directory(delivery / "stage")
        current_fd = open_absolute_directory(repo / "pack/generated")
        live_fd = open_absolute_directory(canonical_root() / "pack/generated")
        deadline = time.monotonic() + OUTSIDE_MANIFEST_SCAN_TIMEOUT_SECONDS
        try:
            staged_manifest = generated_stage_manifest_at(stage_fd, deadline)
            current = generated_stage_manifest_at(current_fd, deadline)
            live = generated_stage_manifest_at(live_fd, deadline)
        finally:
            os.close(live_fd); os.close(current_fd); os.close(stage_fd)
        allowed_stage = {rel.removeprefix("pack/generated/") for rel in GENERATED}
        differences = {
            rel for rel in staged_manifest if staged_manifest[rel] != current[rel]
        }
        require(differences == set(), f"empty generated delta: {sorted(differences)}")
        require(allowed_stage <= set(staged_manifest), "complete admitted generated stage")
        post_by_path = {row["path"]: row for row in journal["expected_post"]}
        for rel in GENERATED:
            staged_rel = rel.removeprefix("pack/generated/")
            require(post_by_path[rel]["sha256"] == live[staged_rel]["sha256"],
                    f"expected-post live generated hash: {rel}")
        repo_fd = open_absolute_directory(repo)
        try:
            require(protected_records_at(repo_fd) == protected_before,
                    "empty-delta protected records")
            require(protected_records_at(repo_fd) == journal["protected"],
                    "empty-delta journal protected")
        finally:
            os.close(repo_fd)

    with tempfile.TemporaryDirectory() as tmp:
        base = Path(tmp).resolve()
        mutant_root = base / "mutant-candidate"
        for relative in AUTHORED:
            source = canonical_root() / relative
            destination = mutant_root / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
        mutant_driver = mutant_root / "tests/workflow-artifact-identity-test.sh"
        source = mutant_driver.read_text()
        require(source.count(empty_require) == 1, "empty-delta require site")
        mutant_driver.write_text(source.replace(
            empty_require,
            "        require(bool(differences) and differences <= allowed_stage,\n",
            1,
        ))
        mutant_driver.chmod(0o755)
        repo = base / "repo"
        create_actual_delivery_repo(repo, full_generator=True)
        prepared = run_actual_delivery_mode(
            repo, ["--delivery-prepare"], timeout=120, driver=mutant_driver,
            fast_fixture=False,
        )
        require(prepared.returncode == 0,
                f"empty-delta mutant prepare: {prepared.stdout[-1000:]}")
        delivery = actual_delivery_directory(repo)
        rejected = run_actual_delivery_mode(
            repo, ["--delivery-install", str(delivery)], "staged-recorded",
            timeout=180, driver=mutant_driver, fast_fixture=False,
        )
        require(rejected.returncode != 0
                and "generated delta outside allowlist" in rejected.stdout,
                f"empty-delta bool(differences) mutant: {rejected.stdout[-1000:]}")


def check_held_stage_generator_symlink_swap() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(os.path.realpath(tmp))
        repo = base / "repo"
        (repo / "scripts").mkdir(parents=True)
        (repo / "pack/.claude/skills").mkdir(parents=True)
        (repo / "stage").mkdir()
        (repo / "pack/.claude/skills/source").write_bytes(b"held")
        os.mkfifo(repo / "derived", 0o600)
        os.mkfifo(repo / "release", 0o600)
        (repo / "scripts/build-host-artifacts.sh").write_text(
            "#!/usr/bin/env bash\nset -euo pipefail\n"
            "OUT_ROOT=\"$DEVRITES_HOST_ARTIFACT_DIR\"\n"
            "exec 8<> ../../../derived\n"
            "exec 9<> ../../../release\n"
            "printf x >&8\n"
            "IFS= read -r -t 3 token <&9\n"
            "[ \"$token\" = go ]\n"
            "mkdir -p \"$OUT_ROOT\"\n"
            "cat pack/.claude/skills/source > \"$OUT_ROOT/held\"\n"
        )
        attacker = base / "attacker"
        (attacker / "stage").mkdir(parents=True)
        (attacker / "scripts").mkdir(parents=True)
        (attacker / "scripts/build-host-artifacts.sh").write_text("replacement")
        (attacker / "stage/marker").write_bytes(b"outside")
        attacker_fd = os.open(attacker, DIRECTORY_FLAGS)
        repo_fd = os.open(repo, DIRECTORY_FLAGS)
        before = manifest_at(attacker_fd, set(), "")
        actor_code = (
            "import os,sys;"
            "base=sys.argv[1];"
            "derived=os.path.join(base,'repo/derived');"
            "fd=os.open(derived,os.O_RDONLY);token=os.read(fd,1);os.close(fd);"
            "os._exit(2) if token!=b'x' else None;"
            "os.rename(os.path.join(base,'repo/stage'),os.path.join(base,'held-stage'));"
            "os.symlink(os.path.join(base,'attacker/stage'),os.path.join(base,'repo/stage'),"
            "target_is_directory=True);"
            "os.rename(os.path.join(base,'repo/scripts'),os.path.join(base,'held-scripts'));"
            "os.symlink(os.path.join(base,'attacker/scripts'),os.path.join(base,'repo/scripts'),"
            "target_is_directory=True);"
            "release=os.path.join(base,'repo/release');"
            "fd=os.open(release,os.O_WRONLY);os.write(fd,b'go\\n');os.close(fd)"
        )
        actor = subprocess.Popen(
            [sys.executable, "-c", actor_code, str(base)],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        )
        try:
            generated_ok, generated_reason, _generated_output = run_private_generator(
                repo_fd, "stage",
                time.monotonic() + DELIVERY_AGGREGATE_TIMEOUT_SECONDS,
            )
            try:
                actor.wait(timeout=3)
            except subprocess.TimeoutExpired:
                actor.kill(); actor.wait()
                fail("held-stage swap actor timeout")
            require(actor.returncode == 0, "held-stage swap actor")
            require(generated_ok, f"held-stage generator fixture: {generated_reason}")
            require((base / "held-stage/held").read_bytes() == b"held",
                    "held-stage inode received generator output")
            require((attacker / "stage/marker").read_bytes() == b"outside"
                    and not (attacker / "stage/held").exists(),
                    "held-stage symlink target unwritten")
            require(manifest_at(attacker_fd, set(), "") == before,
                    "held-stage attacker tree untouched")
        finally:
            if actor.poll() is None:
                actor.kill(); actor.wait()
            if actor.stdout is not None:
                actor.stdout.close()
            os.close(repo_fd); os.close(attacker_fd)


def check_held_stage_generator_held_out_symlink_plant() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(os.path.realpath(tmp))
        repo = base / "repo"
        (repo / "scripts").mkdir(parents=True)
        (repo / "pack/.claude/skills").mkdir(parents=True)
        (repo / "stage").mkdir()
        (repo / "pack/.claude/skills/source").write_bytes(b"held")
        os.mkfifo(repo / "derived", 0o600)
        os.mkfifo(repo / "release", 0o600)
        (repo / "scripts/build-host-artifacts.sh").write_text(
            "#!/usr/bin/env bash\nset -euo pipefail\n"
            "OUT_ROOT=\"$DEVRITES_HOST_ARTIFACT_DIR\"\n"
            "exec 8<> ../../../derived\n"
            "exec 9<> ../../../release\n"
            "printf x >&8\n"
            "IFS= read -r -t 3 token <&9\n"
            "[ \"$token\" = go ]\n"
            "mkdir -p \"$OUT_ROOT\"\n"
            "cat pack/.claude/skills/source > \"$OUT_ROOT/held\"\n"
        )
        outsider = base / "outsider"
        outsider.mkdir()
        (outsider / "marker").write_bytes(b"outside")
        outsider_fd = os.open(outsider, DIRECTORY_FLAGS)
        repo_fd = os.open(repo, DIRECTORY_FLAGS)
        before = manifest_at(outsider_fd, set(), "")
        actor_code = (
            "import os,sys;"
            "base=sys.argv[1];"
            "derived=os.path.join(base,'repo/derived');"
            "fd=os.open(derived,os.O_RDONLY);token=os.read(fd,1);os.close(fd);"
            "os._exit(2) if token!=b'x' else None;"
            "held=os.path.join(base,'repo/stage/.held-out');"
            "moved=os.path.join(base,'held-out-moved');"
            "os.rename(held,moved) if os.path.lexists(held) and not os.path.islink(held) else None;"
            "os.unlink(held) if os.path.lexists(held) else None;"
            "os.symlink(os.path.join(base,'outsider'),held,target_is_directory=True);"
            "release=os.path.join(base,'repo/release');"
            "fd=os.open(release,os.O_WRONLY);os.write(fd,b'go\\n');os.close(fd)"
        )
        actor = subprocess.Popen(
            [sys.executable, "-c", actor_code, str(base)],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        )
        try:
            generated_ok, generated_reason, _generated_output = run_private_generator(
                repo_fd, "stage",
                time.monotonic() + DELIVERY_AGGREGATE_TIMEOUT_SECONDS,
            )
            try:
                actor.wait(timeout=3)
            except subprocess.TimeoutExpired:
                actor.kill(); actor.wait()
                fail("held-out plant actor timeout")
            require(actor.returncode == 0, "held-out plant actor")
            require(generated_ok, f"held-out generator fixture: {generated_reason}")
            require((repo / "stage/held").is_file()
                    and (repo / "stage/held").read_bytes() == b"held",
                    "held-out inode hoisted to stage")
            require((outsider / "marker").read_bytes() == b"outside"
                    and not (outsider / "held").exists(),
                    "held-out symlink target unwritten")
            require(manifest_at(outsider_fd, set(), "") == before,
                    "held-out outsider tree untouched")
        finally:
            if actor.poll() is None:
                actor.kill(); actor.wait()
            if actor.stdout is not None:
                actor.stdout.close()
            os.close(repo_fd); os.close(outsider_fd)


def check_held_stage_generator_artifacts_symlink_plant() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(os.path.realpath(tmp))
        repo = base / "repo"
        (repo / "scripts").mkdir(parents=True)
        (repo / "pack/.claude/skills").mkdir(parents=True)
        (repo / "stage").mkdir()
        (repo / "pack/.claude/skills/source").write_bytes(b"held")
        os.mkfifo(repo / "derived", 0o600)
        os.mkfifo(repo / "release", 0o600)
        (repo / "scripts/build-host-artifacts.sh").write_text(
            "#!/usr/bin/env bash\nset -euo pipefail\n"
            "OUT_ROOT=\"$DEVRITES_HOST_ARTIFACT_DIR\"\n"
            "exec 8<> ../../../derived\n"
            "exec 9<> ../../../release\n"
            "printf x >&8\n"
            "IFS= read -r -t 3 token <&9\n"
            "[ \"$token\" = go ]\n"
            "mkdir -p \"$OUT_ROOT\"\n"
            "cat pack/.claude/skills/source > \"$OUT_ROOT/held\"\n"
        )
        outsider = base / "outsider"
        outsider.mkdir()
        (outsider / "marker").write_bytes(b"outside")
        outsider_fd = os.open(outsider, DIRECTORY_FLAGS)
        repo_fd = os.open(repo, DIRECTORY_FLAGS)
        before = manifest_at(outsider_fd, set(), "")
        actor_code = (
            "import os,sys;"
            "base=sys.argv[1];"
            "derived=os.path.join(base,'repo/derived');"
            "fd=os.open(derived,os.O_RDONLY);token=os.read(fd,1);os.close(fd);"
            "os._exit(2) if token!=b'x' else None;"
            "art=os.path.join(base,'repo/stage/.held-out/artifacts');"
            "moved=os.path.join(base,'artifacts-moved');"
            "os.rename(art,moved) if os.path.lexists(art) and not os.path.islink(art) else None;"
            "os.unlink(art) if os.path.lexists(art) else None;"
            "os.symlink(os.path.join(base,'outsider'),art,target_is_directory=True);"
            "release=os.path.join(base,'repo/release');"
            "fd=os.open(release,os.O_WRONLY);os.write(fd,b'go\\n');os.close(fd)"
        )
        actor = subprocess.Popen(
            [sys.executable, "-c", actor_code, str(base)],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        )
        try:
            generated_ok, generated_reason, _generated_output = run_private_generator(
                repo_fd, "stage",
                time.monotonic() + DELIVERY_AGGREGATE_TIMEOUT_SECONDS,
            )
            try:
                actor.wait(timeout=3)
            except subprocess.TimeoutExpired:
                actor.kill(); actor.wait()
                fail("artifacts plant actor timeout")
            require(actor.returncode == 0, "artifacts plant actor")
            require(generated_ok, f"artifacts generator fixture: {generated_reason}")
            require((repo / "stage/held").is_file()
                    and (repo / "stage/held").read_bytes() == b"held",
                    "artifacts inode hoisted to stage")
            require((outsider / "marker").read_bytes() == b"outside"
                    and not (outsider / "held").exists(),
                    "artifacts symlink target unwritten")
            require(manifest_at(outsider_fd, set(), "") == before,
                    "artifacts outsider tree untouched")
        finally:
            if actor.poll() is None:
                actor.kill(); actor.wait()
            if actor.stdout is not None:
                actor.stdout.close()
            os.close(repo_fd); os.close(outsider_fd)


def check_held_stage_generator_out_root_basename_plant() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(os.path.realpath(tmp))
        repo = base / "repo"
        (repo / "scripts").mkdir(parents=True)
        (repo / "pack/.claude/skills").mkdir(parents=True)
        (repo / "pack/.claude/agents").mkdir(parents=True)
        (repo / "pack/.claude/workflows").mkdir(parents=True)
        (repo / "stage").mkdir()
        (repo / "pack/.claude/skills/source").write_bytes(b"nested-held")
        (repo / "pack/.claude/settings.json").write_text("{}\n")
        os.mkfifo(repo / "derived", 0o600)
        os.mkfifo(repo / "release", 0o600)
        # Real build-host-artifacts.sh shape: rm OUT_ROOT, mkdir -p claude/codex, cp -R.
        (repo / "scripts/build-host-artifacts.sh").write_text(
            "#!/usr/bin/env bash\nset -euo pipefail\n"
            "OUT_ROOT=\"$DEVRITES_HOST_ARTIFACT_DIR\"\n"
            "exec 8<> ../../../derived\n"
            "exec 9<> ../../../release\n"
            "rm -rf \"$OUT_ROOT\"\n"
            "printf x >&8\n"
            "IFS= read -r -t 3 token <&9\n"
            "[ \"$token\" = go ]\n"
            "mkdir -p \"$OUT_ROOT/claude\" \"$OUT_ROOT/codex\"\n"
            "mkdir -p \"$OUT_ROOT/claude/skills\"\n"
            "cp -R pack/.claude/skills/. \"$OUT_ROOT/claude/skills/\"\n"
            "printf nested > \"$OUT_ROOT/codex/marker\"\n"
        )
        outsider = base / "outsider"
        outsider.mkdir()
        (outsider / "marker").write_bytes(b"outside")
        outsider_fd = os.open(outsider, DIRECTORY_FLAGS)
        repo_fd = os.open(repo, DIRECTORY_FLAGS)
        before = manifest_at(outsider_fd, set(), "")
        actor_code = (
            "import os,sys;"
            "base=sys.argv[1];"
            "derived=os.path.join(base,'repo/derived');"
            "fd=os.open(derived,os.O_RDONLY);token=os.read(fd,1);os.close(fd);"
            "os._exit(2) if token!=b'x' else None;"
            "art=os.path.join(base,'repo/stage/.held-out/artifacts');"
            "claude=os.path.join(art,'claude');"
            "codex=os.path.join(art,'codex');"
            "os.rename(claude,os.path.join(base,'claude-moved')) "
            "if os.path.lexists(claude) and not os.path.islink(claude) else None;"
            "os.unlink(claude) if os.path.lexists(claude) else None;"
            "os.symlink(os.path.join(base,'outsider'),claude,target_is_directory=True);"
            "os.rename(codex,os.path.join(base,'codex-moved')) "
            "if os.path.lexists(codex) and not os.path.islink(codex) else None;"
            "os.unlink(codex) if os.path.lexists(codex) else None;"
            "os.symlink(os.path.join(base,'outsider'),codex,target_is_directory=True);"
            "release=os.path.join(base,'repo/release');"
            "fd=os.open(release,os.O_WRONLY);os.write(fd,b'go\\n');os.close(fd)"
        )
        actor = subprocess.Popen(
            [sys.executable, "-c", actor_code, str(base)],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        )
        try:
            generated_ok, generated_reason, _generated_output = run_private_generator(
                repo_fd, "stage",
                time.monotonic() + DELIVERY_AGGREGATE_TIMEOUT_SECONDS,
            )
            try:
                actor.wait(timeout=3)
            except subprocess.TimeoutExpired:
                actor.kill(); actor.wait()
                fail("out-root basename plant actor timeout")
            require(actor.returncode == 0, "out-root basename plant actor")
            require(generated_ok, f"out-root basename generator fixture: {generated_reason}")
            require((repo / "stage/claude/skills/source").is_file()
                    and (repo / "stage/claude/skills/source").read_bytes() == b"nested-held",
                    "out-root basename claude hoisted to stage")
            require((repo / "stage/codex/marker").is_file()
                    and (repo / "stage/codex/marker").read_bytes() == b"nested",
                    "out-root basename codex hoisted to stage")
            require(not (repo / "stage/claude").is_symlink()
                    and not (repo / "stage/codex").is_symlink(),
                    "out-root basename hoist is real directories")
            require((outsider / "marker").read_bytes() == b"outside"
                    and not (outsider / "skills").exists()
                    and not any(path.name != "marker" for path in outsider.rglob("*")),
                    "out-root basename symlink target unwritten")
            require(manifest_at(outsider_fd, set(), "") == before,
                    "out-root basename outsider tree untouched")
        finally:
            if actor.poll() is None:
                actor.kill(); actor.wait()
            if actor.stdout is not None:
                actor.stdout.close()
            os.close(repo_fd); os.close(outsider_fd)


def check_production_delivery_fixture_env() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp).resolve() / "repo"
        before = create_actual_delivery_repo(repo)
        dummy = repo / ".devrites/work/workflow-artifact-identity/.generated-install" / ("0" * 64)
        parent = dummy.parent
        before_entries = sorted(path.name for path in parent.iterdir())
        env = env_without_looping_bash(os.environ)
        env["DEVRITES_REPO_ROOT"] = str(repo)
        env["PYTHONDONTWRITEBYTECODE"] = "1"
        for name in DELIVERY_FIXTURE_ENV:
            env.pop(name, None)
        modes = (
            ["--delivery-prepare"],
            ["--delivery-install", str(dummy)],
            ["--delivery-recover", str(dummy)],
        )
        for name in DELIVERY_FIXTURE_ENV:
            for args in modes:
                hostile = env.copy()
                hostile[name] = "1"
                rejected = subprocess.run(
                    [str(SCRIPT), *args], env=hostile, text=True,
                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                    check=False, timeout=10,
                )
                require(rejected.returncode != 0
                        and "delivery modes reject fixture environment" in rejected.stdout,
                        f"production rejects {name} on {args[0]}: {rejected.stdout[-400:]}")
                require(sorted(path.name for path in parent.iterdir()) == before_entries,
                        f"production fixture env artifacts absent: {name}/{args[0]}")
                repo_fd = open_absolute_directory(repo)
                try:
                    require({relative: file_record_at(repo_fd, relative)
                             for relative in AUTHORED + GENERATED} == before,
                            f"production fixture env destinations unchanged: {name}/{args[0]}")
                finally:
                    os.close(repo_fd)
        fixture_argv = (
            ["--delivery-test-fast-fixture"],
            ["--delivery-test-mutation", "gate-failure"],
            ["--delivery-test-skip-generated", "0"],
        )
        for extra in fixture_argv:
            for args in modes:
                rejected = subprocess.run(
                    [str(SCRIPT), *extra, *args], env=env, text=True,
                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                    check=False, timeout=10,
                )
                require(rejected.returncode != 0
                        and "delivery modes reject fixture argv" in rejected.stdout,
                        f"production rejects {' '.join(extra)} on {args[0]}: {rejected.stdout[-400:]}")
                require(sorted(path.name for path in parent.iterdir()) == before_entries,
                        f"production fixture argv artifacts absent: {extra[0]}/{args[0]}")
                repo_fd = open_absolute_directory(repo)
                try:
                    require({relative: file_record_at(repo_fd, relative)
                             for relative in AUTHORED + GENERATED} == before,
                            f"production fixture argv destinations unchanged: {extra[0]}/{args[0]}")
                finally:
                    os.close(repo_fd)
        hostile = env.copy()
        hostile["DEVRITES_DELIVERY_FAST_FIXTURE"] = "1"
        rejected = subprocess.run(
            [str(SCRIPT), "--delivery-test-fast-fixture", "--delivery-prepare"],
            env=hostile, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            check=False, timeout=10,
        )
        require(rejected.returncode != 0
                and "delivery modes reject fixture environment" in rejected.stdout,
                "production rejects fixture env even with test argv")
        require(sorted(path.name for path in parent.iterdir()) == before_entries,
                "test-argv fixture env artifacts absent")


def delivery_install(delivery_arg: str) -> None:
    reject_delivery_fixture_environment()
    repo_env = os.environ.get("DEVRITES_REPO_ROOT")
    require(repo_env is not None, "DEVRITES_REPO_ROOT is required")
    repo = Path(repo_env).absolute()
    repo_fd = open_absolute_directory(repo)
    delivery_fd, delivery = open_delivery(repo_fd, repo, delivery_arg)
    lock_fd = delivery_lock(delivery_fd)
    root_fd = -1
    stage_fd = -1
    validated = False
    try:
        _reconcile_gate_proof_cache(delivery_fd)
        reconcile_journal_temporary(delivery_fd, repo_fd, delivery.name)
        journal = validate_delivery_journal(
            read_journal(delivery_fd), delivery_fd, repo_fd, delivery.name,
            require_complete=True,
        )
        validated = True
        require(journal["state"] == "SNAPSHOTTING", f"install state: {journal['state']}")
        delivery_death("install-enter")
        require(journal["authored_allowlist"] == AUTHORED and journal["generated_allowlist"] == GENERATED, "delivery allowlists")
        require(journal["writer_allowlist_sha256"] == writer_allowlist_digest(), "delivery allowlist identity")
        require(journal["protected"] == protected_records_at(repo_fd), "protected preinstall identity")
        root = Path(journal["candidate_root"]).absolute()
        root_fd = open_absolute_directory(root)
        require(aggregate_at(root_fd, AUTHORED) == journal["candidate_digest"], "candidate aggregate")
        for rel in AUTHORED:
            require(file_record_at(repo_fd, rel) == file_record_at(root_fd, rel), f"authored candidate: {rel}")
        require(file_record_at(repo_fd, "tests/workflow-artifact-identity-test.sh")["sha256"] == journal["driver_sha256"], "installed driver hash")
        delivery_death("stage-before")
        if entry_info_at(delivery_fd, "stage") is not None:
            remove_tree_at(delivery_fd, "stage")
        os.mkdir("stage", 0o700, dir_fd=delivery_fd); os.fsync(delivery_fd)
        stage_relative = f".devrites/work/workflow-artifact-identity/.generated-install/{delivery.name}/stage"
        aggregate_deadline = time.monotonic() + DELIVERY_AGGREGATE_TIMEOUT_SECONDS
        generated_ok, generated_reason, generated_output = run_private_generator(
            repo_fd, stage_relative, aggregate_deadline,
        )
        if not generated_ok:
            raise RuntimeError(
                f"delivery generator failed: {generated_reason} captured={len(generated_output)}",
            )
        delivery_death("stage-generated")
        stage_fd = os.open("stage", DIRECTORY_FLAGS, dir_fd=delivery_fd)
        os.fchmod(stage_fd, 0o700)
        validate_directory_fd(stage_fd, 0o700)
        staged = generated_stage_manifest_at(stage_fd, aggregate_deadline)
        current_fd = open_dir_components(repo_fd, ("pack", "generated"))
        try:
            current = generated_stage_manifest_at(current_fd, aggregate_deadline)
        finally:
            os.close(current_fd)
        require(set(staged) == set(current), "complete generated stage")
        allowed_stage = {rel.removeprefix("pack/generated/") for rel in GENERATED}
        differences = {rel for rel in staged if staged[rel] != current[rel]}
        require(differences <= allowed_stage,
                f"generated delta outside allowlist: {sorted(differences - allowed_stage)}")
        require(allowed_stage <= set(staged), "complete admitted generated stage")
        if delivery_test_fast_fixture():
            require(differences == allowed_stage, "fast fixture changes every admitted generated stage")
        expected_post = []
        for rel in AUTHORED:
            record = file_record_at(root_fd, rel)
            expected_post.append({"path": rel, "state": "present", "mode": record["mode"], "sha256": record["sha256"]})
        for rel in GENERATED:
            staged_rel = rel.removeprefix("pack/generated/")
            record = file_record_at(stage_fd, staged_rel)
            expected_post.append({"path": rel, "state": "present", "mode": record["mode"], "sha256": record["sha256"]})
        journal["stage_manifest_sha256"] = sha(json.dumps(staged, sort_keys=True).encode())
        journal["expected_post"] = expected_post
        journal["state"] = "STAGED"
        write_journal(delivery_fd, lock_fd, journal)
        delivery_death("staged-recorded")
        require_pre_generated_install_state(repo_fd, journal)
        for index, rel in enumerate(GENERATED, 1):
            journal["state"] = f"INSTALLING({index})"
            journal["mutation_intent"] = {
                "action": "install", "group": "generated",
                "index": index - 1, "path": rel,
            }
            write_journal(delivery_fd, lock_fd, journal)
            delivery_death(f"generated-{index - 1}-intent")
            delivery_test_mutation(repo_fd, "before-generated-claim", rel)
            staged_rel = rel.removeprefix("pack/generated/")
            staged_record = file_record_at(stage_fd, staged_rel)
            if delivery_test_skip_generated() != str(index - 1):
                atomic_destination_mutation(repo_fd, delivery_fd, journal)
            delivery_death(f"generated-{index - 1}-effect")
            require(file_record_at(repo_fd, rel) == staged_record, f"generated replacement effect: {rel}")
            journal["installed_generated"] = index
            journal.pop("mutation_intent")
            write_journal(delivery_fd, lock_fd, journal)
            delivery_death(f"generated-{index - 1}-recorded")
        journal["state"] = "INSTALLED"
        write_journal(delivery_fd, lock_fd, journal)
        delivery_death("installed-recorded")
        journal["gates"] = []
        journal["state"] = "PROVING"
        write_journal(delivery_fd, lock_fd, journal)
        delivery_death("proving-recorded")
        gates = DELIVERY_GATES
        proof_cache_relative = f".devrites/work/workflow-artifact-identity/.generated-install/{delivery.name}/proof-cache"
        for gate_index, (command, expected) in enumerate(gates):
            delivery_death(f"gate-{gate_index}-before")
            journal["gates"].append(run_gate(
                repo_fd, delivery_fd, proof_cache_relative,
                command, expected, aggregate_deadline, gate_index,
            ))
            write_journal(delivery_fd, lock_fd, journal)
            delivery_death(f"gate-{gate_index}-recorded")
        require(aggregate_at(root_fd, AUTHORED) == journal["candidate_digest"],
                "candidate aggregate before commit")
        require(file_record_at(root_fd, "tests/workflow-artifact-identity-test.sh")["sha256"] == journal["driver_sha256"],
                "candidate driver before commit")
        current_outside = manifest_at(
            repo_fd, ALL_DESTINATIONS,
            f".devrites/work/workflow-artifact-identity/.generated-install/{journal['candidate_digest']}",
        )
        require(current_outside == read_outside_manifest(
                    delivery_fd, journal["outside_manifest"],
                ), "outside allowlist identity")
        require(protected_records_at(repo_fd) == journal["protected"], "protected final identity")
        require(valid_delivery_destination_observation(
                    observe_delivery_destinations(repo), journal["expected_post"]),
                "independent exact 16/22 replacement effects")
        delivery_death("commit-before")
        journal["state"] = "COMMITTED"
        write_journal(delivery_fd, lock_fd, journal)
        delivery_death("commit-recorded")
        cleanup_delivery(delivery_fd, lock_fd, journal)
        print("delivery_state=CLEANED")
        print(f"candidate_digest={journal['candidate_digest']}")
        print(f"authored_count={len(AUTHORED)}")
        print(f"generated_count={len(GENERATED)}")
    except BaseException:
        if validated and entry_info_at(delivery_fd, "journal.json") is not None:
            journal = read_journal(delivery_fd)
            if journal.get("state") not in {"COMMITTED", "CLEANING", "CLEANED", "FAILED"}:
                restore_all(
                    repo_fd, delivery_fd, lock_fd, journal, delivery.name,
                )
        raise
    finally:
        if stage_fd >= 0:
            os.close(stage_fd)
        if root_fd >= 0:
            os.close(root_fd)
        os.close(lock_fd); os.close(delivery_fd); os.close(repo_fd)


def delivery_recover(delivery_arg: str) -> None:
    reject_delivery_fixture_environment()
    repo_env = os.environ.get("DEVRITES_REPO_ROOT")
    require(repo_env is not None, "DEVRITES_REPO_ROOT is required")
    repo = Path(repo_env).absolute()
    repo_fd = open_absolute_directory(repo)
    delivery_fd, delivery = open_delivery(repo_fd, repo, delivery_arg)
    lock_fd = delivery_lock(delivery_fd)
    validated = False
    try:
        _reconcile_gate_proof_cache(delivery_fd)
        reconcile_journal_temporary(delivery_fd, repo_fd, delivery.name)
        journal = validate_delivery_journal(
            read_journal(delivery_fd), delivery_fd, repo_fd, delivery.name,
            require_complete=True, check_outside=False,
        )
        validated = True
        if journal["state"] in {"FAILED", "CLEANED"}:
            validate_delivery_journal(
                journal, delivery_fd, repo_fd, delivery.name,
                require_complete=True,
            )
            print(f"delivery_state={journal['state']}")
            return
        if journal["state"] == "RESTORED":
            restore_all(
                repo_fd, delivery_fd, lock_fd, journal, delivery.name,
            )
            print("delivery_state=FAILED")
            print(f"restored_authored={len(AUTHORED)}")
            print(f"restored_generated={len(GENERATED)}")
            return
        if journal["state"] in {"COMMITTED", "CLEANING"}:
            validate_delivery_journal(
                journal, delivery_fd, repo_fd, delivery.name,
                require_complete=True,
            )
            cleanup_delivery(delivery_fd, lock_fd, journal)
            print("delivery_state=CLEANED")
            return
        complete_pending_mutation(
            repo_fd, delivery_fd, lock_fd, journal, delivery.name,
        )
        validate_delivery_journal(
            journal, delivery_fd, repo_fd, delivery.name,
            require_complete=True,
        )
        restore_all(
            repo_fd, delivery_fd, lock_fd, journal, delivery.name,
        )
        print("delivery_state=FAILED")
        print(f"restored_authored={len(AUTHORED)}")
        print(f"restored_generated={len(GENERATED)}")
    except BaseException:
        if validated and entry_info_at(delivery_fd, "journal.json") is not None:
            current = read_journal(delivery_fd)
            state = current.get("state", "")
            if (state in {"SNAPSHOTTING", "STAGED", "INSTALLED", "PROVING"}
                    or state.startswith("INSTALLING(")
                    or state.startswith("ROLLING_BACK(")):
                restore_all(
                    repo_fd, delivery_fd, lock_fd, current, delivery.name,
                )
        raise
    finally:
        os.close(lock_fd); os.close(delivery_fd); os.close(repo_fd)


def walkthrough() -> None:
    global ACTUAL_ENGINE_OUTPUT
    start = time.monotonic_ns()
    root_path = canonical_root()
    ACTUAL_ENGINE_OUTPUT = load_actual_engine_output(root_path)
    rows = {row[0]: row for row in markdown_rows((root_path / MODULE_REL).read_text(), "WA-OP-")}
    admission, contents = admission_fixture(); parse_admission(admission, contents)
    def complete(workspace: Path, operation: str) -> dict:
        row = rows[operation]
        proc, ready_fd, gate_fd = operation_consumer(workspace, row)
        finish_operation_consumer(proc, ready_fd, gate_fd)
        observed = observe_operation(workspace)
        require(operation_observation_matches(observed, row), f"walkthrough lifecycle: {operation}")
        return observed
    with tempfile.TemporaryDirectory() as tmp:
        base = Path(tmp); success = base / "success"
        for operation in (
            "WA-OP-001-OWNER-ACQUIRE", "WA-OP-002-SOURCE-PROMOTE",
            "WA-OP-003-JOURNAL-INIT", "WA-OP-004-STAGE-WRITE",
            "WA-OP-005-BACKUP-WRITE", "WA-OP-006-INSTALL",
            "WA-OP-007-PROVE", "WA-OP-013-EVIDENCE-UPDATE",
        ):
            complete(success, operation)
        product_observed = complete(success, "WA-OP-014-PRODUCT-SEPARATION")
        require(product_observed["facts"]["product_equal"] is True, "walkthrough product dimensions equal")
        (success / "product.observer-pass").write_bytes(b"equal\n"); (success / "product.observer-pass").chmod(0o600)
        complete(success, "WA-OP-010-SUCCESS-CLEANUP")
        cleaned = observe_operation(success)
        require(cleaned["journal"]["value"]["operation_id"] == "WA-OP-010-SUCCESS-CLEANUP"
                and (success / "lifecycle.state").read_bytes() == b"CLEANED\n"
                and not (success / "source").exists(), "walkthrough observed CLEANED and source GC")
        cursor = (success / "cursor").read_text().strip()
        require(cursor == "prove:/rite-prove demo", "walkthrough cursor restoration")
        elapsed = max(0, (time.monotonic_ns() - start) // 1_000_000)

        interrupted = base / "interrupted"; install = rows["WA-OP-006-INSTALL"]
        proc, ready_fd, gate_fd = operation_consumer(interrupted, install)
        read_barrier(ready_fd, b"R"); os.write(gate_fd, b"x")
        read_barrier(ready_fd, b"I"); os.write(gate_fd, b"x"); read_barrier(ready_fd, b"E")
        terminate_group(proc, 0.05); os.close(ready_fd); os.close(gate_fd)
        require(operation_observation_matches(observe_operation(interrupted), install, "INTENT"),
                "walkthrough interrupted install")
        complete(interrupted, "WA-OP-006-INSTALL")

        stale = base / "stale"; prepare_operation_fixture(stale, "WA-OP-002-SOURCE-PROMOTE")
        (stale / "stale.authority").write_bytes(b"old-binding\n"); (stale / "stale.authority").chmod(0o600)
        proc, ready_fd, gate_fd = operation_consumer(stale, rows["WA-OP-002-SOURCE-PROMOTE"])
        finish_operation_consumer(proc, ready_fd, gate_fd)
        stale_line = (stale / "diagnostic").read_text().strip()
        require((stale / "lifecycle.state").read_bytes() == b"PLAN_VET_REPAIR\n"
                and stale_line.startswith("WORKFLOW_ARTIFACT_FAILURE reason_id=WA-R004-IDENTITY-STALE "),
                "walkthrough observed stale authority route")

        verified = complete(success, "WA-OP-015-VERIFY-EXISTING")
        require((success / "verified").is_file() and verified["facts"]["product_equal"] is True,
                "walkthrough idempotent verification")
        comparison = json.loads((success / "product.comparison").read_text())
        proof_line = (success / "proof.output").read_text().strip()
        product_line = "product_identity=unchanged" if comparison["current"] == comparison["frozen"] else ""
        pass_line = "WORKFLOW_ARTIFACT_WALKTHROUGH PASS" if verified["valid"] and product_line else ""
        require(proof_line == "WA-PROOF-001 PASS" and product_line and pass_line, "walkthrough observed outputs")
    print(proof_line)
    print(f"tthw_ms={elapsed}")
    print(stale_line)
    print(f"cursor={cursor}")
    print(product_line)
    print(pass_line)


def require_delivery_mode_environment() -> None:
    require(os.environ.get("PYTHONDONTWRITEBYTECODE") == "1",
            "delivery modes require PYTHONDONTWRITEBYTECODE=1")
    for directory in os.get_exec_path():
        try:
            info = os.stat(Path(directory) / "bash")
        except OSError as exc:
            if exc.errno == errno.ELOOP:
                fail("delivery modes require executable bash PATH")
            continue
        if stat.S_ISREG(info.st_mode) and info.st_mode & 0o111:
            break
    else:
        fail("delivery modes require executable bash PATH")


def main() -> None:
    args, test_config = take_delivery_test_argv(ARGS)
    production = production_delivery_argv(args)
    boundary = (
        len(args) == 3 and args[0] == "--delivery-boundary-case"
        and args[1] in {"prepare", "install", "rollback"}
        and args[2] in DELIVERY_BOUNDARIES
    )
    operate = (
        len(args) >= 3 and args[0] == "--delivery-boundary-case" and args[1] == "operate"
        and (
            args[2:] == ["prepare"]
            or (len(args) == 4 and args[2] in {"install", "recover"})
        )
    )
    usage = (
        "usage: workflow-artifact-identity-test.sh [--check-delivery-gate-signals | "
        "--prove-walkthrough | --delivery-prepare | --delivery-install DIR | "
        "--delivery-recover DIR | --delivery-boundary-case KIND BOUNDARY]"
    )
    if production or boundary or operate:
        require_delivery_mode_environment()
        if production:
            reject_delivery_fixture_environment()
            reject_delivery_fixture_argv(test_config)
        _DELIVERY_TEST.update(test_config)
    elif test_config != {
        "fast_fixture": False,
        "mutation": None,
        "death_boundary": None,
        "skip_generated": None,
    }:
        raise SystemExit(usage)
    if not args:
        if wai_boundary_only():
            check_actual_delivery_modes()
            print(
                "workflow-artifact-identity: PASS "
                f"(boundary shard {os.environ.get('DEVRITES_WAI_BOUNDARY_SHARD', '?')})"
            )
            return
        if wai_delivery_model_only():
            check_delivery_model_matrix()
            print("workflow-artifact-identity: PASS (delivery-model-matrix)")
            return
        root = project_root_for_tests(canonical_root())
        restorations = install_live_protected_fixtures(root)
        try:
            protected_before = require_live_protected_identity()
            default_tests(canonical_root())
            require(require_live_protected_identity() == protected_before,
                    "protected identity unchanged by private checks")
        finally:
            restore_live_protected_fixtures(restorations)
        print("workflow-artifact-identity: PASS")
        return
    if args == ["--check-delivery-gate-signals"]:
        check_delivery_gate_signals()
        print("delivery-gate-signals: PASS")
        return
    if args == ["--prove-walkthrough"]:
        walkthrough()
        return
    if boundary:
        delivery_boundary_case(args[1], args[2])
        return
    if operate:
        if args[2] == "prepare":
            delivery_prepare()
        elif args[2] == "install":
            delivery_install(args[3])
        else:
            delivery_recover(args[3])
        return
    if args == ["--delivery-prepare"]:
        delivery_prepare()
        return
    if len(args) == 2 and args[0] == "--delivery-install":
        delivery_install(args[1])
        return
    if len(args) == 2 and args[0] == "--delivery-recover":
        delivery_recover(args[1])
        return
    if len(args) == 3 and args[0] == "--atomic-death-fixture" and args[2] in {"after-create", "after-sync"}:
        atomic_death_child(args[1], args[2])
        return
    raise SystemExit(usage)


if __name__ == "__main__":
    main()
PY

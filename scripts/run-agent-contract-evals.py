#!/usr/bin/env python3
"""Run the hermetic Claude/Codex agent-contract matrix.

Fake mode is deterministic and network-free. Live mode is manual, paid, and
fails closed unless host pins, distinct scoped credentials, artifact input,
and both cost ceilings pass preflight.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import signal
import stat
import subprocess
import sys
import tempfile
import time
from collections import Counter
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation
from pathlib import Path, PurePosixPath
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
DEFAULT_MATRIX = ROOT / "evals" / "agent-contract" / "contract-matrix.json"
FAKE_HOST = ROOT / "scripts" / "live-hosts" / "fake-host.py"
LIVE_CONFIRMATION = "RUN-PAID-HOST-CONTRACTS"
SUMMARY_SCHEMA = "devrites-agent-contract-summary/v1"
PREFLIGHT_SCHEMA = "devrites-agent-contract-preflight/v1"
BUDGET_SCHEMA = "devrites-agent-contract-budget/v1"
LIVE_RUNTIME_COST_CAPS = {"claude": "cli-max-budget-usd"}
RUN_ID_RE = re.compile(r"^drv-run-v1:[0-9a-f]{32}$")
REASON_RE = re.compile(r'\bID\s*=\s*"([A-Z0-9-]+)"')
SAFE_TOKEN_RE = re.compile(r"^[a-z][a-z0-9-]{0,63}$")
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
SENSITIVE_ASSIGNMENT_RE = re.compile(
    r"(?i)(?:api[_-]?key|authorization|bearer|credential|password|secret|token)\s*[:=]\s*\S+"
)
SECRET_TOKEN_RE = re.compile(
    r"(?i)\b(?:sk|rk|ghp|github_pat|xox[baprs])[-_][a-z0-9_-]{8,}\b"
)
TEXT_PAYLOAD_RE = re.compile(r"(?i)(?:^|[\s,{])(prompt|raw|source|trace)\s*[:=]")
MANDATORY_SIGNALS = {
    "auth_preflight",
    "budget_bound",
    "cleanup_ready",
    "environment_allowlisted",
    "mcp_absent",
    "plugins_absent",
    "roots_isolated",
    "shell_startup_ignored",
    "version_pinned",
}
RESULT_KEYS = {
    "result_version",
    "run_id",
    "cell_id",
    "host",
    "role",
    "status",
    "baseline_id",
    "scope_id",
    "budget",
    "side_effects",
    "payload_type",
    "execution",
    "signals",
    "usage",
    "dispatch_reason_id",
    "result_reason_id",
    "outcome",
    "terminal_sentinel",
    "host_version",
    "model",
}
FORBIDDEN_RESULT_KEYS = {
    "absolute_path",
    "auth",
    "config",
    "home",
    "prompt",
    "raw",
    "scratch_root",
    "source",
    "trace",
}


class ContractError(Exception):
    def __init__(self, message: str, reason_id: str = "DRV-AGENT-RESULT-MALFORMED"):
        super().__init__(message)
        self.reason_id = reason_id


def canonical_json(value: Any) -> bytes:
    return json.dumps(
        value, sort_keys=True, separators=(",", ":"), ensure_ascii=True
    ).encode()


def sha256_bytes(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def load_json(path: Path) -> Any:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise ContractError("JSON-compatible YAML is required") from exc


def exact_keys(value: dict[str, Any], required: set[str], where: str) -> None:
    keys = set(value)
    if keys != required:
        raise ContractError(f"{where} keys do not match the frozen schema")


def safe_relative(value: Any, where: str) -> str:
    if not isinstance(value, str) or not value:
        raise ContractError(f"{where} must be a non-empty relative path")
    path = PurePosixPath(value)
    if path.is_absolute() or ".." in path.parts or str(path) != value or "\\" in value:
        raise ContractError(f"{where} is not a confined portable path")
    return value


def safe_id(value: Any, where: str) -> str:
    if (
        not isinstance(value, str)
        or not 1 <= len(value) <= 128
        or value.startswith("/")
        or any(ord(character) < 32 or ord(character) == 127 for character in value)
    ):
        raise ContractError(
            f"{where} is not a safe identifier", "DRV-AGENT-UNAVAILABLE"
        )
    return value


def positive_int(value: Any, where: str) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or value <= 0:
        raise ContractError(f"{where} must be a positive integer")
    return value


def money(value: Any, where: str, *, allow_zero: bool = False) -> Decimal:
    try:
        parsed = Decimal(str(value))
    except (InvalidOperation, ValueError) as exc:
        raise ContractError(f"{where} must be a decimal amount") from exc
    if not parsed.is_finite() or parsed < 0 or (not allow_zero and parsed == 0):
        raise ContractError(
            f"{where} must be {'non-negative' if allow_zero else 'positive'}"
        )
    return parsed


def private_string_fault(value: str) -> str | None:
    if (
        value.startswith(("/", "~", "file://"))
        or re.match(r"^[A-Za-z]:[\\/]", value)
        or "\x00" in value
    ):
        return "path"
    if len(value) > 256 or any(character in value for character in ("\n", "\r", "```")):
        return "text payload"
    if (
        SENSITIVE_ASSIGNMENT_RE.search(value)
        or SECRET_TOKEN_RE.search(value)
        or TEXT_PAYLOAD_RE.search(value)
    ):
        return "sensitive payload"
    return None


def frozen_reasons() -> set[str]:
    path = ROOT / "engine" / "internal" / "reason" / "reason.go"
    try:
        reasons = set(REASON_RE.findall(path.read_text(encoding="utf-8")))
    except OSError as exc:
        raise ContractError("frozen reason catalog is unavailable") from exc
    if not reasons:
        raise ContractError("frozen reason catalog is empty")
    return reasons


def validate_matrix(data: Any) -> dict[str, Any]:
    if not isinstance(data, dict):
        raise ContractError("matrix must be an object")
    exact_keys(
        data,
        {"schema", "packet_schema", "result_schema", "hosts", "budget", "scenarios"},
        "matrix",
    )
    if data["schema"] != "devrites-agent-contract-matrix/v1":
        raise ContractError("matrix schema is unsupported")
    if (
        data["packet_schema"] != "agent-packet/v1"
        or data["result_schema"] != "agent-result/v1"
    ):
        raise ContractError("transport schema is unsupported")

    reasons = frozen_reasons()
    hosts = data["hosts"]
    if not isinstance(hosts, list) or len(hosts) != 2:
        raise ContractError("matrix must define exactly two hosts")
    host_ids: set[str] = set()
    for host in hosts:
        if not isinstance(host, dict):
            raise ContractError("host row must be an object")
        exact_keys(host, {"id", "launcher", "fake_version"}, "host")
        host_id = host["id"]
        if host_id not in {"claude", "codex"} or host_id in host_ids:
            raise ContractError("host IDs must be unique claude and codex")
        host_ids.add(host_id)
        safe_relative(host["launcher"], "host.launcher")
        if not isinstance(host["fake_version"], str) or not host["fake_version"]:
            raise ContractError("host.fake_version must be non-empty")
        launcher = (ROOT / host["launcher"]).resolve()
        if ROOT.resolve() not in launcher.parents:
            raise ContractError("host launcher escapes the repository")

    budget = data["budget"]
    if not isinstance(budget, dict):
        raise ContractError("budget must be an object")
    exact_keys(
        budget,
        {
            "timeout_seconds",
            "estimated_max_cost_usd_per_cell",
            "max_files",
            "max_loaded_lines",
            "max_result_lines",
        },
        "budget",
    )
    for key in ("timeout_seconds", "max_files", "max_loaded_lines", "max_result_lines"):
        positive_int(budget[key], f"budget.{key}")
    money(
        budget["estimated_max_cost_usd_per_cell"],
        "budget.estimated_max_cost_usd_per_cell",
    )

    scenarios = data["scenarios"]
    if not isinstance(scenarios, list) or not scenarios:
        raise ContractError("matrix scenarios must be non-empty")
    scenario_ids: set[str] = set()
    for scenario in scenarios:
        if not isinstance(scenario, dict):
            raise ContractError("scenario row must be an object")
        exact_keys(
            scenario,
            {"id", "role", "role_class", "simulate", "allowed_repo_writes", "expect"},
            "scenario",
        )
        scenario_id = scenario["id"]
        if (
            not isinstance(scenario_id, str)
            or not SAFE_TOKEN_RE.fullmatch(scenario_id)
            or scenario_id in scenario_ids
        ):
            raise ContractError("scenario IDs must be unique safe tokens")
        scenario_ids.add(scenario_id)
        if not isinstance(scenario["role"], str) or not scenario["role"]:
            raise ContractError("scenario.role must be non-empty")
        if scenario["role_class"] not in {"reviewer", "wright"}:
            raise ContractError("scenario.role_class is unsupported")
        if scenario["simulate"] not in {
            "accepted",
            "generic-fallback",
            "inline-fallback",
            "interrupted",
            "malformed",
            "missing",
            "post-compaction",
            "reviewer-denied",
            "session-start",
            "stale",
            "wright-denied",
            "write-allowed",
        }:
            raise ContractError("scenario.simulate is unsupported")
        writes = scenario["allowed_repo_writes"]
        if not isinstance(writes, list) or len(writes) != len(set(writes)):
            raise ContractError("allowed writes must be a unique list")
        for index, path in enumerate(writes):
            safe_relative(path, f"allowed_repo_writes[{index}]")
        if scenario["role_class"] == "reviewer" and writes:
            raise ContractError("reviewers cannot receive repo writes")

        expect = scenario["expect"]
        if not isinstance(expect, dict):
            raise ContractError("scenario.expect must be an object")
        exact_keys(
            expect,
            {
                "execution_mode",
                "guard_strength",
                "independence",
                "dispatch_reason_id",
                "result_reason_id",
                "status",
                "outcome",
                "terminal_sentinel",
                "host_exit",
                "required_signals",
            },
            "scenario.expect",
        )
        if expect["execution_mode"] not in {"named", "generic", "inline"}:
            raise ContractError("execution mode is unsupported")
        if expect["guard_strength"] not in {
            "enforced",
            "observed",
            "unavailable",
            "bypassed",
        }:
            raise ContractError("guard strength is unsupported")
        if not isinstance(expect["independence"], bool):
            raise ContractError("independence must be boolean")
        for key in ("dispatch_reason_id", "result_reason_id"):
            if expect[key] not in reasons:
                raise ContractError(f"{key} is not in the frozen reason catalog")
        if expect["status"] not in {"complete", "partial", "blocked", "failed"}:
            raise ContractError("expected status is unsupported")
        if expect["outcome"] not in {"passed", "denied", "unavailable"}:
            raise ContractError("expected outcome is unsupported")
        if expect["terminal_sentinel"] not in {
            "complete",
            "interrupted",
            "malformed",
            "missing",
            "stale",
        }:
            raise ContractError("terminal sentinel is unsupported")
        if isinstance(expect["host_exit"], bool) or expect["host_exit"] not in {0, 130}:
            raise ContractError("expected host exit is unsupported")
        required_signals = expect["required_signals"]
        if (
            not isinstance(required_signals, list)
            or len(required_signals) != len(set(required_signals))
            or not all(
                isinstance(item, str)
                and SAFE_TOKEN_RE.fullmatch(item.replace("_", "-"))
                for item in required_signals
            )
        ):
            raise ContractError("required signals must be unique safe tokens")
    return data


def tree_snapshot(root: Path) -> dict[str, dict[str, Any]]:
    out: dict[str, dict[str, Any]] = {}
    if not root.exists():
        return out
    for path in sorted(root.rglob("*")):
        rel = path.relative_to(root).as_posix()
        mode = stat.S_IMODE(path.lstat().st_mode)
        if path.is_symlink():
            out[rel] = {
                "kind": "symlink",
                "mode": mode,
                "sha256": sha256_bytes(os.readlink(path).encode()),
            }
        elif path.is_file():
            out[rel] = {
                "kind": "file",
                "mode": mode,
                "sha256": sha256_bytes(path.read_bytes()),
            }
    return out


def tree_delta(
    before: dict[str, dict[str, Any]], after: dict[str, dict[str, Any]]
) -> list[dict[str, Any]]:
    delta: list[dict[str, Any]] = []
    for path in sorted(set(before) | set(after)):
        if before.get(path) == after.get(path):
            continue
        if path not in after:
            delta.append(
                {"path": path, "kind": "deleted", "sha256": sha256_bytes(b"deleted")}
            )
        else:
            delta.append(
                {
                    "path": path,
                    "kind": after[path]["kind"],
                    "sha256": after[path]["sha256"],
                }
            )
    return delta


def parse_assignments(values: list[str], hosts: set[str], label: str) -> dict[str, str]:
    out: dict[str, str] = {}
    for value in values:
        if "=" not in value:
            raise ContractError(f"{label} must use HOST=VALUE", "DRV-AGENT-UNAVAILABLE")
        host, item = value.split("=", 1)
        if host not in hosts or not item or host in out:
            raise ContractError(
                f"{label} has an invalid or duplicate host", "DRV-AGENT-UNAVAILABLE"
            )
        out[host] = item
    return out


def secure_auth_file(path_value: str) -> Path:
    path = Path(path_value)
    try:
        info = path.lstat()
    except OSError as exc:
        raise ContractError(
            "scoped auth file is unavailable", "DRV-AGENT-UNAVAILABLE"
        ) from exc
    if stat.S_ISLNK(info.st_mode) or not stat.S_ISREG(info.st_mode):
        raise ContractError(
            "scoped auth file must be a regular non-symlink", "DRV-AGENT-UNAVAILABLE"
        )
    if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o600:
        raise ContractError(
            "scoped auth file must be owner-only mode 0600", "DRV-AGENT-UNAVAILABLE"
        )
    if info.st_size <= 0 or info.st_size > 64 * 1024:
        raise ContractError("scoped auth file size is invalid", "DRV-AGENT-UNAVAILABLE")
    return path.resolve()


def launcher_version(
    host: dict[str, Any], fake: bool, fake_fault: str | None = None
) -> str:
    launcher = ROOT / host["launcher"]
    env = {
        "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
        "LANG": "C",
        "LC_ALL": "C",
    }
    if fake:
        env["DEVRITES_CONTRACT_FAKE_HOST"] = str(FAKE_HOST)
        env["DEVRITES_CONTRACT_FAKE_VERSION"] = host["fake_version"]
        env["DEVRITES_CONTRACT_HOST"] = host["id"]
        if fake_fault == "version-mismatch":
            env["DEVRITES_CONTRACT_FAKE_VERSION"] += "-mismatch"
    try:
        proc = subprocess.run(
            [str(launcher), "version"],
            env=env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            timeout=15,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise ContractError(
            "host version preflight failed", "DRV-AGENT-UNAVAILABLE"
        ) from exc
    version = proc.stdout.strip()
    if proc.returncode != 0 or not version or "\n" in version:
        raise ContractError("host version preflight failed", "DRV-AGENT-UNAVAILABLE")
    return version


def load_budget_ledger(path: Path) -> dict[str, Any]:
    data = load_json(path)
    if not isinstance(data, dict):
        raise ContractError("budget ledger must be an object", "DRV-AGENT-UNAVAILABLE")
    exact_keys(data, {"schema", "month", "spent_usd"}, "budget ledger")
    if data["schema"] != BUDGET_SCHEMA:
        raise ContractError(
            "budget ledger schema is unsupported", "DRV-AGENT-UNAVAILABLE"
        )
    if data["month"] != datetime.now(timezone.utc).strftime("%Y-%m"):
        raise ContractError(
            "budget ledger month is not current", "DRV-AGENT-UNAVAILABLE"
        )
    money(data["spent_usd"], "budget ledger spent_usd", allow_zero=True)
    return data


def live_preflight(
    args: argparse.Namespace, matrix: dict[str, Any]
) -> tuple[dict[str, dict[str, str]], Decimal, Decimal, Path]:
    hosts = {row["id"]: row for row in matrix["hosts"]}
    host_ids = set(hosts)
    if args.confirm_live != LIVE_CONFIRMATION:
        raise ContractError("live confirmation is missing", "DRV-AGENT-UNAVAILABLE")
    versions = parse_assignments(args.host_version, host_ids, "--host-version")
    models = parse_assignments(args.host_model, host_ids, "--host-model")
    versions = {
        host: safe_id(value, f"--host-version {host}")
        for host, value in versions.items()
    }
    models = {
        host: safe_id(value, f"--host-model {host}")
        for host, value in models.items()
    }
    auth_values = parse_assignments(args.auth_file, host_ids, "--auth-file")
    if (
        set(versions) != host_ids
        or set(models) != host_ids
        or set(auth_values) != host_ids
    ):
        raise ContractError(
            "every host needs a version, model, and auth file", "DRV-AGENT-UNAVAILABLE"
        )
    auth = {host: secure_auth_file(auth_values[host]) for host in host_ids}
    auth_identities = {
        (path.stat().st_dev, path.stat().st_ino) for path in auth.values()
    }
    if len(set(auth.values())) != len(auth) or len(auth_identities) != len(auth):
        raise ContractError(
            "host credentials must use separate scoped files", "DRV-AGENT-UNAVAILABLE"
        )
    actual_versions: dict[str, dict[str, str]] = {}
    for host_id, host in hosts.items():
        actual = launcher_version(host, False)
        if actual != versions[host_id]:
            raise ContractError(
                "host version does not match its exact pin", "DRV-AGENT-UNAVAILABLE"
            )
        actual_versions[host_id] = {
            "version": actual,
            "model": models[host_id],
            "auth_file": str(auth[host_id]),
        }

    if args.host_artifacts is None or not args.host_artifacts.is_dir():
        raise ContractError(
            "live mode requires a prepared host artifact directory",
            "DRV-AGENT-UNAVAILABLE",
        )
    per_run = money(args.max_cost_per_run_usd, "--max-cost-per-run-usd")
    monthly = money(args.max_monthly_cost_usd, "--max-monthly-cost-usd")
    if args.budget_ledger is None:
        raise ContractError(
            "live mode requires a monthly budget ledger", "DRV-AGENT-UNAVAILABLE"
        )
    ledger = load_budget_ledger(args.budget_ledger)
    spent = money(ledger["spent_usd"], "budget ledger spent_usd", allow_zero=True)
    projected = per_run * Decimal(len(matrix["hosts"]) * len(matrix["scenarios"]))
    if spent + projected > monthly:
        raise ContractError(
            "the full matrix exceeds the monthly ceiling", "DRV-AGENT-UNAVAILABLE"
        )
    uncapped = sorted(host_ids - LIVE_RUNTIME_COST_CAPS.keys())
    if uncapped:
        raise ContractError(
            "paid live host lacks an enforced runtime/provider cost cap: "
            + ", ".join(uncapped),
            "DRV-AGENT-UNAVAILABLE",
        )
    return actual_versions, per_run, monthly, args.budget_ledger


def seed_project(
    project: Path, live: bool, host_artifacts: Path | None, env: dict[str, str]
) -> None:
    project.mkdir(parents=True)
    (project / "README.md").write_text(
        "# Synthetic agent contract fixture\n", encoding="utf-8"
    )
    (project / "src").mkdir()
    if not live:
        return
    install_env = {
        "PATH": env["PATH"],
        "HOME": env["HOME"],
        "TMPDIR": env["TMPDIR"],
        "LANG": "C",
        "LC_ALL": "C",
        "DEVRITES_HOST_ARTIFACT_DIR": str(host_artifacts),
    }
    proc = subprocess.run(
        ["bash", str(ROOT / "install.sh"), "--target", str(project)],
        env=install_env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        timeout=180,
        check=False,
    )
    if proc.returncode != 0:
        raise ContractError(
            "isolated host-pack install failed", "DRV-AGENT-UNAVAILABLE"
        )
    fixture = ROOT / "evals" / "golden" / "shippable-feature"
    workspace = project / ".devrites" / "work" / "shippable-feature"
    workspace.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(fixture, workspace)
    (project / ".devrites" / "ACTIVE").write_text(
        "shippable-feature\n", encoding="utf-8"
    )


def make_packet(
    matrix: dict[str, Any],
    host: dict[str, Any],
    scenario: dict[str, Any],
    project: Path,
    scratch: Path,
) -> dict[str, Any]:
    baseline_tree = tree_snapshot(project)
    diff_hash = sha256_bytes(canonical_json(baseline_tree))
    touched_hash = sha256_bytes(canonical_json(sorted(baseline_tree)))
    run_id = "drv-run-v1:" + os.urandom(16).hex()
    baseline_id = f"no-git:{diff_hash}:{touched_hash}"
    budget = matrix["budget"]
    return {
        "packet_version": matrix["packet_schema"],
        "run_id": run_id,
        "cell_id": f"{host['id']}:{scenario['id']}",
        "host": host["id"],
        "role": scenario["role"],
        "role_class": scenario["role_class"],
        "phase": "eval",
        "attempt": 1,
        "baseline": {
            "id": baseline_id,
            "head": "no-git",
            "diff_sha256": diff_hash,
            "touched_sha256": touched_hash,
        },
        "objective_id": scenario["id"],
        "inputs": [{"path": "README.md", "purpose": "synthetic-contract-fixture"}],
        "scope": {
            "id": sha256_bytes(canonical_json(scenario["allowed_repo_writes"])),
            "in": ["README.md"],
            "out": ["repository-history", "external-systems"],
            "allowed_repo_writes": scenario["allowed_repo_writes"],
        },
        "budgets": {
            "max_files": budget["max_files"],
            "max_loaded_lines": budget["max_loaded_lines"],
            "max_result_lines": budget["max_result_lines"],
        },
        "scratch_root": str(scratch),
        "simulate": scenario["simulate"],
        "expected_execution": scenario["expect"]["execution_mode"],
        "expected_guard": scenario["expect"]["guard_strength"],
        "allowed_signals": sorted(
            MANDATORY_SIGNALS | set(scenario["expect"]["required_signals"])
        ),
        "result_schema": matrix["result_schema"],
    }


def write_transport(path: Path, value: Any) -> None:
    path.write_bytes(canonical_json(value) + b"\n")
    path.chmod(0o600)


def synthetic_reconcile(
    host: dict[str, Any],
    scenario: dict[str, Any],
    exit_code: int,
    reason_id: str,
    sentinel: str,
    host_version: str,
    model: str,
    *,
    live: bool,
    identity_matched: bool = True,
    baseline_matched: bool = True,
) -> dict[str, Any]:
    expect = scenario["expect"]
    return {
        "cell_id": f"{host['id']}:{scenario['id']}",
        "host": host["id"],
        "scenario": scenario["id"],
        "role": scenario["role"],
        "passed": False,
        "status": "partial" if sentinel == "interrupted" else "failed",
        "outcome": "unavailable",
        "execution_mode": expect["execution_mode"],
        "guard_strength": expect["guard_strength"],
        "independence": expect["independence"],
        "dispatch_reason_id": expect["dispatch_reason_id"],
        "result_reason_id": reason_id,
        "terminal_sentinel": sentinel,
        "host_exit": exit_code,
        "host_version": host_version,
        "model": model,
        "identity_matched": identity_matched,
        "baseline_matched": baseline_matched,
        "repo_writes": [],
        "signals": sorted(MANDATORY_SIGNALS if sentinel == "interrupted" else []),
        "usage": {
            "input_tokens": 0,
            "output_tokens": 0,
            "cost_usd": None if live else "0",
            "cost_provenance": "unavailable" if live else "synthetic",
        },
    }


def declared_writes(value: Any) -> list[dict[str, Any]]:
    if not isinstance(value, list):
        raise ContractError("repo_writes must be a list")
    out: list[dict[str, Any]] = []
    for item in value:
        if not isinstance(item, dict):
            raise ContractError("repo write must be an object")
        exact_keys(item, {"path", "kind", "sha256"}, "repo write")
        safe_relative(item["path"], "repo write path")
        if item["kind"] not in {
            "file",
            "symlink",
            "deleted",
        } or not SHA256_RE.fullmatch(str(item["sha256"])):
            raise ContractError("repo write metadata is invalid")
        out.append(item)
    if len({item["path"] for item in out}) != len(out):
        raise ContractError("repo writes contain duplicates")
    return sorted(out, key=lambda item: item["path"])


def validate_result(
    raw: Any,
    packet: dict[str, Any],
    host: dict[str, Any],
    scenario: dict[str, Any],
    exit_code: int,
    actual_delta: list[dict[str, Any]],
    host_version: str,
    model: str,
    reasons: set[str],
    live: bool,
) -> dict[str, Any]:
    if not isinstance(raw, dict) or set(raw) != RESULT_KEYS:
        raise ContractError("result object does not match the frozen schema")
    if any(key in FORBIDDEN_RESULT_KEYS for key in raw):
        raise ContractError("result contains a forbidden retention key")
    if raw["result_version"] != "agent-result/v1":
        raise ContractError("result version is unsupported")
    identity_fields = {
        "run_id": packet["run_id"],
        "cell_id": packet["cell_id"],
        "host": packet["host"],
        "role": packet["role"],
    }
    if any(raw.get(key) != value for key, value in identity_fields.items()):
        raise ContractError(
            "result identity mismatch", "DRV-AGENT-RESULT-IDENTITY-MISMATCH"
        )
    if not RUN_ID_RE.fullmatch(raw["run_id"]):
        raise ContractError("result run ID is malformed")
    if raw["baseline_id"] != packet["baseline"]["id"]:
        raise ContractError("result baseline is stale", "DRV-AGENT-RESULT-STALE")
    if raw["scope_id"] != packet["scope"]["id"]:
        raise ContractError(
            "result scope identity mismatch", "DRV-AGENT-RESULT-IDENTITY-MISMATCH"
        )
    if raw["host_version"] != host_version or raw["model"] != model:
        raise ContractError(
            "result host pin identity mismatch", "DRV-AGENT-RESULT-IDENTITY-MISMATCH"
        )

    if raw["status"] not in {"complete", "partial", "blocked", "failed"}:
        raise ContractError("result status is unsupported")
    if raw["outcome"] not in {"passed", "denied", "unavailable"}:
        raise ContractError("result outcome is unsupported")
    if raw["terminal_sentinel"] not in {"complete", "interrupted"}:
        raise ContractError("result terminal sentinel is unsupported")
    if raw["payload_type"] not in {"review-findings", "wright-report"}:
        raise ContractError("result payload type is unsupported")

    execution = raw["execution"]
    if not isinstance(execution, dict):
        raise ContractError("result execution must be an object")
    exact_keys(
        execution, {"mode", "guard_strength", "independence"}, "result execution"
    )
    if execution["mode"] not in {"named", "generic", "inline"}:
        raise ContractError("result execution mode is unsupported")
    if execution["guard_strength"] not in {
        "enforced",
        "observed",
        "unavailable",
        "bypassed",
    }:
        raise ContractError("result guard strength is unsupported")
    if not isinstance(execution["independence"], bool):
        raise ContractError("result independence must be boolean")

    for key in ("dispatch_reason_id", "result_reason_id"):
        if raw[key] not in reasons:
            raise ContractError("result reason is not frozen")
    budget = raw["budget"]
    if not isinstance(budget, dict):
        raise ContractError("result budget must be an object")
    exact_keys(
        budget,
        {"files_used", "loaded_lines_used", "result_lines_used"},
        "result budget",
    )
    packet_limits = packet["budgets"]
    for used_key, limit_key in (
        ("files_used", "max_files"),
        ("loaded_lines_used", "max_loaded_lines"),
        ("result_lines_used", "max_result_lines"),
    ):
        used = budget[used_key]
        if (
            isinstance(used, bool)
            or not isinstance(used, int)
            or used < 0
            or used > packet_limits[limit_key]
        ):
            raise ContractError("result exceeded its packet budget")

    side_effects = raw["side_effects"]
    if not isinstance(side_effects, dict):
        raise ContractError("result side_effects must be an object")
    exact_keys(
        side_effects, {"repo_writes", "scratch_write_count"}, "result side_effects"
    )
    writes = declared_writes(side_effects["repo_writes"])
    if (
        isinstance(side_effects["scratch_write_count"], bool)
        or not isinstance(side_effects["scratch_write_count"], int)
        or side_effects["scratch_write_count"] < 0
    ):
        raise ContractError("scratch write count is invalid")
    if writes != actual_delta:
        raise ContractError(
            "declared and observed repo effects differ", "DRV-AGENT-RESULT-OUT-OF-SCOPE"
        )
    allowed = set(packet["scope"]["allowed_repo_writes"])
    if any(item["path"] not in allowed for item in actual_delta):
        raise ContractError(
            "repo effect escaped the exact allowlist", "DRV-AGENT-RESULT-OUT-OF-SCOPE"
        )

    signals = raw["signals"]
    if (
        not isinstance(signals, list)
        or len(signals) != len(set(signals))
        or not all(isinstance(item, str) and item for item in signals)
    ):
        raise ContractError("result signals are invalid")
    if not MANDATORY_SIGNALS.issubset(signals):
        raise ContractError("result is missing hermetic preflight signals")
    allowed_signals = MANDATORY_SIGNALS | set(scenario["expect"]["required_signals"])
    if not set(signals).issubset(allowed_signals):
        raise ContractError("result contains a signal outside the scenario vocabulary")
    usage = raw["usage"]
    if not isinstance(usage, dict):
        raise ContractError("result usage must be an object")
    usage = dict(usage)
    if not live and set(usage) == {"input_tokens", "output_tokens", "cost_usd"}:
        usage["cost_provenance"] = "synthetic"
    exact_keys(
        usage,
        {"input_tokens", "output_tokens", "cost_usd", "cost_provenance"},
        "result usage",
    )
    for key in ("input_tokens", "output_tokens"):
        if (
            isinstance(usage[key], bool)
            or not isinstance(usage[key], int)
            or usage[key] < 0
        ):
            raise ContractError("result token count is invalid")
    provenance = usage["cost_provenance"]
    if provenance == "unavailable":
        if not live or usage["cost_usd"] is not None:
            raise ContractError("unavailable cost provenance is inconsistent")
    elif provenance == "provider-reported":
        if not live or usage["cost_usd"] is None:
            raise ContractError("provider cost provenance is inconsistent")
        usage["cost_usd"] = str(
            money(usage["cost_usd"], "result cost", allow_zero=True)
        )
    elif provenance == "synthetic":
        if live or usage["cost_usd"] is None:
            raise ContractError("synthetic cost provenance is inconsistent")
        usage["cost_usd"] = str(
            money(usage["cost_usd"], "result cost", allow_zero=True)
        )
    else:
        raise ContractError("result cost provenance is unsupported")

    return {
        "cell_id": packet["cell_id"],
        "host": host["id"],
        "scenario": scenario["id"],
        "role": scenario["role"],
        "passed": False,
        "status": raw["status"],
        "outcome": raw["outcome"],
        "execution_mode": execution["mode"],
        "guard_strength": execution["guard_strength"],
        "independence": execution["independence"],
        "dispatch_reason_id": raw["dispatch_reason_id"],
        "result_reason_id": raw["result_reason_id"],
        "terminal_sentinel": raw["terminal_sentinel"],
        "host_exit": exit_code,
        "host_version": raw["host_version"],
        "model": raw["model"],
        "identity_matched": True,
        "baseline_matched": True,
        "repo_writes": writes,
        "signals": sorted(signals),
        "usage": usage,
    }


def usage_cost(usage: dict[str, Any]) -> Decimal | None:
    if usage["cost_provenance"] == "unavailable":
        return None
    return money(usage["cost_usd"], "result cost", allow_zero=True)


def result_matches_expected(result: dict[str, Any], scenario: dict[str, Any]) -> bool:
    expect = scenario["expect"]
    pairs = {
        "status": expect["status"],
        "outcome": expect["outcome"],
        "execution_mode": expect["execution_mode"],
        "guard_strength": expect["guard_strength"],
        "independence": expect["independence"],
        "dispatch_reason_id": expect["dispatch_reason_id"],
        "result_reason_id": expect["result_reason_id"],
        "terminal_sentinel": expect["terminal_sentinel"],
        "host_exit": expect["host_exit"],
    }
    if any(result.get(key) != value for key, value in pairs.items()):
        return False
    if not set(expect["required_signals"]).issubset(result.get("signals", [])):
        return False
    if scenario["simulate"] == "write-allowed":
        if [item["path"] for item in result["repo_writes"]] != scenario[
            "allowed_repo_writes"
        ]:
            return False
    elif result["repo_writes"]:
        return False
    return True


def run_cell(
    matrix: dict[str, Any],
    host: dict[str, Any],
    scenario: dict[str, Any],
    *,
    live: bool,
    host_version: str,
    model: str,
    auth_file: Path | None,
    host_artifacts: Path | None,
    per_run_cost: Decimal,
    fake_fault: str | None,
) -> dict[str, Any]:
    reasons = frozen_reasons()
    run_root = Path(tempfile.mkdtemp(prefix="devrites-agent-contract-")).resolve()
    project = run_root / "project"
    config = run_root / "config"
    state = run_root / "state"
    home = run_root / "home"
    scratch = run_root / "scratch"
    tmp = run_root / "tmp"
    guard = run_root / "guard"
    for path in (config, state, home, scratch, tmp, guard):
        path.mkdir(mode=0o700)
    (guard / "sentinel").write_text("unchanged\n", encoding="utf-8")

    if auth_file is None:
        auth_file = state / f"{host['id']}.auth"
        auth_file.write_text("fake-scoped-credential\n", encoding="utf-8")
        auth_file.chmod(0o600)
    base_env = {
        "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
        "LANG": "C",
        "LC_ALL": "C",
        "HOME": str(home),
        "TMPDIR": str(tmp),
        "XDG_CONFIG_HOME": str(config),
        "XDG_STATE_HOME": str(state),
    }
    try:
        seed_project(project, live, host_artifacts, base_env)
        packet = make_packet(matrix, host, scenario, project, scratch)
        packet_path = scratch / "agent-packet.yaml"
        result_path = scratch / "agent-result.yaml"
        write_transport(packet_path, packet)

        before_project = tree_snapshot(project)
        before_guard = tree_snapshot(guard)
        env = {
            **base_env,
            "DEVRITES_ROOT": str(project),
            "DEVRITES_AGENT_SCRATCH": str(scratch),
            "DEVRITES_RUN_ID": packet["run_id"],
            "DEVRITES_CONTRACT_RUN_ROOT": str(run_root),
            "DEVRITES_CONTRACT_PACKET": str(packet_path),
            "DEVRITES_CONTRACT_RESULT": str(result_path),
            "DEVRITES_CONTRACT_HOST": host["id"],
            "DEVRITES_CONTRACT_HOST_VERSION_PIN": host_version,
            "DEVRITES_CONTRACT_MODEL": model,
            "DEVRITES_CONTRACT_AUTH_FILE": str(auth_file),
            "DEVRITES_CONTRACT_MAX_COST_USD": str(per_run_cost),
        }
        if not live:
            env["DEVRITES_CONTRACT_FAKE_HOST"] = str(FAKE_HOST)
            env["DEVRITES_CONTRACT_FAKE_VERSION"] = host["fake_version"]
            if fake_fault and fake_fault.startswith(packet["cell_id"] + ":"):
                env["DEVRITES_CONTRACT_FAKE_FAULT"] = fake_fault.rsplit(":", 1)[1]

        launcher = ROOT / host["launcher"]
        if live and scenario["simulate"] == "interrupted":
            proc_live = subprocess.Popen(
                [str(launcher), "run"],
                env=env,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                start_new_session=True,
            )
            time.sleep(2)
            if proc_live.poll() is None:
                os.killpg(proc_live.pid, signal.SIGINT)
            try:
                exit_code = proc_live.wait(timeout=15)
            except subprocess.TimeoutExpired:
                os.killpg(proc_live.pid, signal.SIGKILL)
                proc_live.wait()
                exit_code = 130
        else:
            try:
                proc = subprocess.run(
                    [str(launcher), "run"],
                    env=env,
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                    timeout=matrix["budget"]["timeout_seconds"],
                    check=False,
                    start_new_session=True,
                )
                exit_code = proc.returncode
            except subprocess.TimeoutExpired:
                exit_code = 130

        after_project = tree_snapshot(project)
        actual_delta = tree_delta(before_project, after_project)
        if before_guard != tree_snapshot(guard):
            result = synthetic_reconcile(
                host,
                scenario,
                exit_code,
                "DRV-AGENT-RESULT-OUT-OF-SCOPE",
                "complete",
                host_version,
                model,
                live=live,
            )
        elif not result_path.exists():
            result = synthetic_reconcile(
                host,
                scenario,
                exit_code,
                "DRV-AGENT-UNAVAILABLE",
                "missing",
                host_version,
                model,
                live=live,
            )
        else:
            try:
                raw = load_json(result_path)
                result = validate_result(
                    raw,
                    packet,
                    host,
                    scenario,
                    exit_code,
                    actual_delta,
                    host_version,
                    model,
                    reasons,
                    live,
                )
            except ContractError as exc:
                sentinel = {
                    "DRV-AGENT-RESULT-STALE": "stale",
                    "DRV-AGENT-RESULT-MALFORMED": "malformed",
                    "DRV-AGENT-RESULT-IDENTITY-MISMATCH": "malformed",
                    "DRV-AGENT-RESULT-OUT-OF-SCOPE": "malformed",
                }.get(exc.reason_id, "malformed")
                result = synthetic_reconcile(
                    host,
                    scenario,
                    exit_code,
                    exc.reason_id,
                    sentinel,
                    host_version,
                    model,
                    live=live,
                    identity_matched=exc.reason_id
                    != "DRV-AGENT-RESULT-IDENTITY-MISMATCH",
                    baseline_matched=exc.reason_id != "DRV-AGENT-RESULT-STALE",
                )
        if (
            scenario["simulate"] == "interrupted"
            and result["result_reason_id"] == "DRV-AGENT-UNAVAILABLE"
        ):
            result["status"] = "partial"
            result["terminal_sentinel"] = "interrupted"
            result["signals"] = sorted(
                MANDATORY_SIGNALS | {"progress_recorded", "partial_metadata"}
            )
        cost = usage_cost(result["usage"])
        if cost is not None and cost > per_run_cost:
            result["passed"] = False
            result["result_reason_id"] = "DRV-AGENT-UNAVAILABLE"
            return result
        result["passed"] = result_matches_expected(result, scenario)
        return result
    finally:
        shutil.rmtree(run_root, ignore_errors=True)


def assert_private_metadata(value: Any) -> None:
    def walk(item: Any) -> None:
        if isinstance(item, dict):
            if FORBIDDEN_RESULT_KEYS.intersection(item):
                raise ContractError("summary contains forbidden retention fields")
            for key, child in item.items():
                if isinstance(key, str):
                    if key.lower() in FORBIDDEN_RESULT_KEYS:
                        raise ContractError(
                            "summary contains a forbidden retention key"
                        )
                    if private_string_fault(key):
                        raise ContractError("summary contains a private metadata key")
                walk(child)
        elif isinstance(item, list):
            for child in item:
                walk(child)
        elif isinstance(item, str):
            fault = private_string_fault(item)
            if fault:
                raise ContractError(f"summary contains a private {fault}")

    walk(value)


def write_summary(results_dir: Path | None, summary: dict[str, Any]) -> None:
    assert_private_metadata(summary)
    encoded = json.dumps(summary, sort_keys=True, indent=2) + "\n"
    if results_dir is None:
        sys.stdout.write(encoded)
        return
    results_dir.mkdir(parents=True, exist_ok=True)
    target = results_dir / "summary.json"
    temp = results_dir / ".summary.json.tmp"
    temp.write_text(encoded, encoding="utf-8")
    temp.chmod(0o600)
    os.replace(temp, target)
    sys.stdout.write(encoded)


def write_ledger(path: Path, month: str, spent: Decimal) -> None:
    updated = {"schema": BUDGET_SCHEMA, "month": month, "spent_usd": str(spent)}
    temp = path.with_name(path.name + ".tmp")
    temp.write_bytes(canonical_json(updated) + b"\n")
    temp.chmod(0o600)
    os.replace(temp, path)


def locked_ledger(path: Path) -> tuple[int, Path, dict[str, Any]]:
    lock = path.with_name(path.name + ".lock")
    try:
        descriptor = os.open(lock, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    except FileExistsError as exc:
        raise ContractError(
            "budget ledger is locked by another run", "DRV-AGENT-UNAVAILABLE"
        ) from exc
    try:
        return descriptor, lock, load_budget_ledger(path)
    except Exception:
        os.close(descriptor)
        lock.unlink(missing_ok=True)
        raise


def reserve_budget(path: Path, monthly: Decimal, amount: Decimal) -> None:
    descriptor, lock, ledger = locked_ledger(path)
    try:
        spent = money(ledger["spent_usd"], "ledger spent", allow_zero=True)
        if spent + amount > monthly:
            raise ContractError(
                "the full matrix exceeds the monthly ceiling", "DRV-AGENT-UNAVAILABLE"
            )
        write_ledger(path, ledger["month"], spent + amount)
    finally:
        os.close(descriptor)
        lock.unlink(missing_ok=True)


def settle_budget(path: Path, reserved: Decimal, actual: Decimal | None) -> None:
    descriptor, lock, ledger = locked_ledger(path)
    try:
        spent = money(ledger["spent_usd"], "ledger spent", allow_zero=True)
        settled = reserved if actual is None else actual
        write_ledger(
            path, ledger["month"], max(Decimal("0"), spent - reserved + settled)
        )
    finally:
        os.close(descriptor)
        lock.unlink(missing_ok=True)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument(
        "--fake",
        action="store_true",
        help="run the deterministic fake-host matrix (default)",
    )
    mode.add_argument(
        "--live",
        action="store_true",
        help="run paid host CLIs after strict manual preflight",
    )
    parser.add_argument("--matrix", type=Path, default=DEFAULT_MATRIX)
    parser.add_argument("--results-dir", type=Path)
    parser.add_argument("--preflight-only", action="store_true")
    parser.add_argument("--confirm-live")
    parser.add_argument(
        "--host-version", action="append", default=[], metavar="HOST=VERSION"
    )
    parser.add_argument(
        "--host-model", action="append", default=[], metavar="HOST=MODEL"
    )
    parser.add_argument("--auth-file", action="append", default=[], metavar="HOST=PATH")
    parser.add_argument("--host-artifacts", type=Path)
    parser.add_argument("--max-cost-per-run-usd")
    parser.add_argument("--max-monthly-cost-usd")
    parser.add_argument("--budget-ledger", type=Path)
    parser.add_argument(
        "--fake-fault",
        help="fake-only fault injection as HOST:SCENARIO:identity-mismatch|escape-write|privacy-leak|cost-overrun",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        matrix = validate_matrix(load_json(args.matrix))
        matrix_digest = sha256_bytes(canonical_json(matrix))
        host_rows = {row["id"]: row for row in matrix["hosts"]}
        cells = len(matrix["hosts"]) * len(matrix["scenarios"])
        live = bool(args.live)
        live_info: dict[str, dict[str, str]] = {}
        ledger_path: Path | None = None
        monthly = Decimal("0")
        if live:
            if args.results_dir is None:
                raise ContractError(
                    "live mode requires --results-dir", "DRV-AGENT-UNAVAILABLE"
                )
            live_info, per_run, monthly, ledger_path = live_preflight(args, matrix)
        else:
            if any(
                [
                    args.confirm_live,
                    args.host_version,
                    args.host_model,
                    args.auth_file,
                    args.host_artifacts,
                    args.max_cost_per_run_usd,
                    args.max_monthly_cost_usd,
                    args.budget_ledger,
                ]
            ):
                raise ContractError("live-only options cannot be used in fake mode")
            per_run = money(
                matrix["budget"]["estimated_max_cost_usd_per_cell"],
                "matrix estimated cost",
            )
            for host_id, row in host_rows.items():
                actual = launcher_version(
                    row,
                    True,
                    "version-mismatch"
                    if args.fake_fault == f"{host_id}:version:version-mismatch"
                    else None,
                )
                if actual != row["fake_version"]:
                    raise ContractError(
                        "fake host version pin mismatch", "DRV-AGENT-UNAVAILABLE"
                    )
                live_info[host_id] = {
                    "version": actual,
                    "model": f"{host_id}-contract-fake",
                    "auth_file": "",
                }

        if args.preflight_only:
            summary = {
                "schema": PREFLIGHT_SCHEMA,
                "mode": "live" if live else "fake",
                "matrix_digest": matrix_digest,
                "cells": cells,
                "hosts": [
                    {
                        "host": host_id,
                        "version": live_info[host_id]["version"],
                        "model": live_info[host_id]["model"],
                    }
                    for host_id in sorted(host_rows)
                ],
                "max_live_cost_usd": str(per_run * Decimal(cells)),
                "monthly_ceiling_usd": str(monthly) if live else "0",
                "ready": True,
            }
            write_summary(args.results_dir, summary)
            return 0

        results: list[dict[str, Any]] = []
        reservation = per_run * Decimal(cells)
        if live and ledger_path is not None:
            reserve_budget(ledger_path, monthly, reservation)
        for host in matrix["hosts"]:
            info = live_info[host["id"]]
            auth = Path(info["auth_file"]) if live else None
            for scenario in matrix["scenarios"]:
                result = run_cell(
                    matrix,
                    host,
                    scenario,
                    live=live,
                    host_version=info["version"],
                    model=info["model"],
                    auth_file=auth,
                    host_artifacts=args.host_artifacts,
                    per_run_cost=per_run,
                    fake_fault=args.fake_fault,
                )
                results.append(result)

        observed_costs = [usage_cost(row["usage"]) for row in results]
        total_cost = (
            sum((cost for cost in observed_costs if cost is not None), Decimal("0"))
            if all(cost is not None for cost in observed_costs)
            else None
        )
        summary = {
            "schema": SUMMARY_SCHEMA,
            "mode": "live" if live else "fake",
            "matrix_digest": matrix_digest,
            "cells_total": len(results),
            "cells_passed": sum(1 for row in results if row["passed"]),
            "cells_failed": sum(1 for row in results if not row["passed"]),
            "cost_usd": str(total_cost) if total_cost is not None else None,
            "reason_counts": dict(
                sorted(Counter(row["result_reason_id"] for row in results).items())
            ),
            "results": results,
        }
        write_summary(args.results_dir, summary)
        if live and ledger_path is not None:
            settle_budget(ledger_path, reservation, total_cost)
        return 0 if summary["cells_failed"] == 0 else 1
    except ContractError as exc:
        sys.stderr.write(f"agent-contract preflight failed: {exc.reason_id}: {exc}\n")
        return 2


if __name__ == "__main__":
    sys.exit(main())

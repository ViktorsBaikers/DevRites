#!/usr/bin/env python3
"""Provide a deterministic fake for the agent-contract host wrappers."""
from __future__ import annotations

import hashlib
import json
import os
import stat
import sys
from pathlib import Path
from typing import Any


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
ALLOWED_ENV = {
    "DEVRITES_AGENT_SCRATCH",
    "DEVRITES_CONTRACT_AUTH_FILE",
    "DEVRITES_CONTRACT_FAKE_FAULT",
    "DEVRITES_CONTRACT_FAKE_VERSION",
    "DEVRITES_CONTRACT_HOST",
    "DEVRITES_CONTRACT_HOST_VERSION_PIN",
    "DEVRITES_CONTRACT_MAX_COST_USD",
    "DEVRITES_CONTRACT_MODEL",
    "DEVRITES_CONTRACT_PACKET",
    "DEVRITES_CONTRACT_RESULT",
    "DEVRITES_CONTRACT_RUN_ROOT",
    "DEVRITES_ROOT",
    "DEVRITES_RUN_ID",
    "HOME",
    "LANG",
    "LC_ALL",
    "PATH",
    "PYTHONDONTWRITEBYTECODE",
    "TMPDIR",
    "XDG_CONFIG_HOME",
    "XDG_STATE_HOME",
}


def canonical(value: Any) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode()


def digest(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def env_path(name: str) -> Path:
    value = os.environ.get(name, "")
    if not value:
        raise ValueError(f"missing {name}")
    return Path(value).resolve()


def contained(path: Path, root: Path) -> bool:
    return path == root or root in path.parents


def secure_auth(path: Path) -> bool:
    try:
        info = path.lstat()
        return (
            stat.S_ISREG(info.st_mode)
            and not stat.S_ISLNK(info.st_mode)
            and stat.S_IMODE(info.st_mode) == 0o600
            and 0 < info.st_size <= 64 * 1024
        )
    except OSError:
        return False


def hermetic_signals(packet: dict[str, Any]) -> set[str]:
    run_root = env_path("DEVRITES_CONTRACT_RUN_ROOT")
    roots = [
        env_path("DEVRITES_ROOT"),
        env_path("DEVRITES_AGENT_SCRATCH"),
        env_path("HOME"),
        env_path("TMPDIR"),
        env_path("XDG_CONFIG_HOME"),
        env_path("XDG_STATE_HOME"),
    ]
    signals: set[str] = set()
    visible_env = set(os.environ) - {"PWD", "OLDPWD", "SHLVL", "_", "__CF_USER_TEXT_ENCODING"}
    if visible_env <= ALLOWED_ENV:
        signals.add("environment_allowlisted")
    if len(set(roots)) == len(roots) and all(contained(path, run_root) for path in roots):
        signals.add("roots_isolated")
    if "BASH_ENV" not in os.environ and "ENV" not in os.environ:
        signals.add("shell_startup_ignored")
    config_roots = [env_path("HOME"), env_path("XDG_CONFIG_HOME")]
    names = [path.name.lower() for root in config_roots for path in root.rglob("*")]
    if not any("plugin" in name for name in names):
        signals.add("plugins_absent")
    if not any("mcp" in name for name in names):
        signals.add("mcp_absent")
    if secure_auth(env_path("DEVRITES_CONTRACT_AUTH_FILE")):
        signals.add("auth_preflight")
    budgets = packet.get("budgets", {})
    if all(isinstance(budgets.get(key), int) and budgets[key] > 0 for key in (
        "max_files",
        "max_loaded_lines",
        "max_result_lines",
    )):
        signals.add("budget_bound")
    if (
        os.environ.get("DEVRITES_CONTRACT_HOST_VERSION_PIN")
        == os.environ.get("DEVRITES_CONTRACT_FAKE_VERSION")
    ):
        signals.add("version_pinned")
    if contained(env_path("DEVRITES_CONTRACT_RESULT"), env_path("DEVRITES_AGENT_SCRATCH")):
        signals.add("cleanup_ready")
    return signals


def mode_and_reasons(simulate: str) -> tuple[str, str, bool, str, str, str]:
    mode = "named"
    guard = "enforced"
    independence = True
    dispatch = "DRV-AGENT-NAMED"
    result = "DRV-AGENT-RESULT-ACCEPTED"
    outcome = "passed"
    if simulate == "generic-fallback":
        mode = "generic"
        dispatch = "DRV-AGENT-GENERIC-FALLBACK"
    elif simulate in {"reviewer-denied", "wright-denied"}:
        result = "DRV-AGENT-RESULT-OUT-OF-SCOPE"
        outcome = "denied"
    elif simulate == "interrupted":
        result = "DRV-AGENT-UNAVAILABLE"
        outcome = "unavailable"
    return mode, guard, independence, dispatch, result, outcome


def run() -> int:
    packet_path = env_path("DEVRITES_CONTRACT_PACKET")
    result_path = env_path("DEVRITES_CONTRACT_RESULT")
    packet = json.loads(packet_path.read_text(encoding="utf-8"))
    simulate = packet["simulate"]
    fault = os.environ.get("DEVRITES_CONTRACT_FAKE_FAULT", "")
    if simulate in {"dispatch-unavailable", "missing"}:
        return 0
    if simulate == "malformed":
        result_path.write_text("not-json-compatible-yaml\n", encoding="utf-8")
        return 0

    writes: list[dict[str, str]] = []
    signals = hermetic_signals(packet)
    if simulate == "write-allowed":
        relative = packet["scope"]["allowed_repo_writes"][0]
        target = env_path("DEVRITES_ROOT") / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        content = b"synthetic contract write\n"
        target.write_bytes(content)
        writes.append({"path": relative, "kind": "file", "sha256": digest(content)})
        signals.add("declared_side_effects")
    elif simulate == "generic-fallback":
        signals.add("fresh_context")
    elif simulate == "reviewer-denied":
        signals.add("mutation_denied")
    elif simulate == "wright-denied":
        signals.add("scope_denied")
    elif simulate == "session-start":
        signals.add("session_start_oriented")
    elif simulate == "post-compaction":
        signals.add("resume_cursor_reinjected")
    elif simulate == "interrupted":
        signals.update({"progress_recorded", "partial_metadata"})

    mode, guard, independence, dispatch, result_reason, outcome = mode_and_reasons(simulate)
    status = "blocked" if outcome == "denied" else "complete"
    sentinel = "complete"
    if simulate == "interrupted":
        status = "partial"
        sentinel = "interrupted"
    run_id = packet["run_id"]
    if fault == "identity-mismatch":
        run_id = "drv-run-v1:" + "0" * 32
        if run_id == packet["run_id"]:
            run_id = "drv-run-v1:" + "1" * 32
    if fault == "escape-write":
        target = env_path("DEVRITES_ROOT") / "src" / "escape.txt"
        target.write_text("undeclared fake-host write\n", encoding="utf-8")
    if fault == "privacy-leak":
        signals.add("/operator/private/path")
    baseline_id = packet["baseline"]["id"]
    if simulate == "stale":
        baseline_id += ":stale"

    result = {
        "result_version": packet["result_schema"],
        "run_id": run_id,
        "cell_id": packet["cell_id"],
        "host": packet["host"],
        "role": packet["role"],
        "status": status,
        "baseline_id": baseline_id,
        "scope_id": packet["scope"]["id"],
        "budget": {
            "files_used": 1 if writes else 0,
            "loaded_lines_used": 1,
            "result_lines_used": 1,
        },
        "side_effects": {
            "repo_writes": writes,
            "scratch_write_count": 1,
        },
        "payload_type": "wright-report" if packet["role_class"] == "wright" else "review-findings",
        "execution": {
            "mode": mode,
            "guard_strength": guard,
            "independence": independence,
        },
        "signals": sorted(signals),
        "usage": {
            "input_tokens": 0,
            "output_tokens": 0,
            "cost_usd": "999" if fault == "cost-overrun" else "0",
        },
        "dispatch_reason_id": dispatch,
        "result_reason_id": result_reason,
        "outcome": outcome,
        "terminal_sentinel": sentinel,
        "host_version": os.environ["DEVRITES_CONTRACT_HOST_VERSION_PIN"],
        "model": os.environ["DEVRITES_CONTRACT_MODEL"],
    }
    result_path.write_bytes(canonical(result) + b"\n")
    result_path.chmod(0o600)
    return 130 if simulate == "interrupted" else 0


def main() -> int:
    if len(sys.argv) == 2 and sys.argv[1] == "version":
        version = os.environ.get("DEVRITES_CONTRACT_FAKE_VERSION", "")
        if not version:
            return 2
        print(version)
        return 0
    if len(sys.argv) == 2 and sys.argv[1] == "run":
        try:
            return run()
        except (KeyError, OSError, ValueError, json.JSONDecodeError):
            return 2
    return 2


if __name__ == "__main__":
    sys.exit(main())

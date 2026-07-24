#!/usr/bin/env python3
"""Run focused, network-free checks for the controlled behavioral runner."""

from __future__ import annotations

import json
import importlib.util
import os
import subprocess
import sys
import tempfile
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parent.parent
RUNNER = ROOT / "scripts" / "run-live-behavioral-evals.py"
FORBIDDEN_KEYS = {
    "absolute_path",
    "auth",
    "config",
    "fixture",
    "home",
    "operator",
    "path",
    "prompt",
    "raw",
    "scratch",
    "source",
    "task",
    "trace",
}
EVENT_KEYS = {"assistant_text", "tool_call", "tool_result", "usage", "other"}
TOOL_KEYS = {
    "read",
    "search",
    "write",
    "shell",
    "git_commit",
    "git_push",
    "git_tag",
    "mutation",
    "other",
}


def invoke(
    temporary: Path, *args: str, extra_env: dict[str, str] | None = None
) -> tuple[subprocess.CompletedProcess[str], Path]:
    results = temporary / ("results-" + str(len(list(temporary.glob("results-*")))))
    environment = {
        **os.environ,
        "TMPDIR": str(temporary / "tmp"),
        "BASH_ENV": str(temporary / "operator-startup"),
        "ENV": str(temporary / "operator-startup"),
        "CLAUDE_CONFIG_DIR": str(temporary / "operator-claude"),
        "CODEX_HOME": str(temporary / "operator-codex"),
        "ANTHROPIC_API_KEY": "DO-NOT-INHERIT",
        "OPENAI_API_KEY": "DO-NOT-INHERIT",
    }
    environment.update(extra_env or {})
    (temporary / "tmp").mkdir(exist_ok=True)
    (temporary / "operator-startup").write_text(
        "export DEVRITES_OPERATOR_STARTUP_LEAK=1\n", encoding="utf-8"
    )
    process = subprocess.run(
        [sys.executable, str(RUNNER), *args, "--results-dir", str(results)],
        cwd=ROOT,
        env=environment,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=90,
        check=False,
    )
    return process, results


def assert_private(value: Any) -> None:
    if isinstance(value, dict):
        assert not (FORBIDDEN_KEYS & set(value))
        for child in value.values():
            assert_private(child)
    elif isinstance(value, list):
        for child in value:
            assert_private(child)
    elif isinstance(value, str):
        assert not value.startswith("/")
        assert "\n" not in value
        assert "DO-NOT-INHERIT" not in value
        assert "operator-startup" not in value


def load_summary(path: Path) -> dict[str, Any]:
    return json.loads((path / "summary.json").read_text(encoding="utf-8"))


def main() -> int:
    spec = importlib.util.spec_from_file_location(
        "controlled_behavioral_runner", RUNNER
    )
    assert spec is not None and spec.loader is not None
    runner = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(runner)
    assert runner.git_actions("git commit -m ship") == ["commit"]
    assert runner.git_actions("git -C repo push && rtk proxy git tag v1") == [
        "push",
        "tag",
    ]
    assert runner.git_actions(
        "env MODE=test /usr/bin/git -c user.name=test commit -m ship"
    ) == ["commit"]
    assert runner.git_actions("git status; echo 'git commit'") == []

    with tempfile.TemporaryDirectory(prefix="devrites-live-behavioral-test-") as raw:
        temporary = Path(raw)

        dry, dry_dir = invoke(temporary, "--dry-run")
        assert dry.returncode == 0, dry.stderr
        dry_summary = load_summary(dry_dir)
        assert dry_summary["schema"] == "devrites-controlled-behavioral-preflight/v1"
        assert dry_summary["mode"] == "dry-run"
        assert dry_summary["sessions_total"] == 20
        assert dry_summary["ready"] is True

        fake, fake_dir = invoke(temporary, "--fake")
        assert fake.returncode == 0, fake.stderr
        summary = load_summary(fake_dir)
        assert summary["schema"] == "devrites-controlled-behavioral-summary/v1"
        assert summary["mode"] == "fake"
        assert summary["sessions_total"] == 20
        assert summary["sessions_passed"] == 20
        assert summary["sessions_failed"] == 0
        assert summary["usage_totals"] == {
            "input_tokens": 200,
            "output_tokens": 100,
            "cost_usd": "0",
            "cost_exposed": True,
        }
        rows = summary["results"]
        assert len({row["trial_id"] for row in rows}) == 20
        assert Counter(row["arm_id"] for row in rows) == {
            "control": 10,
            "candidate": 10,
        }
        assert sum(row["trial_id"].startswith("quick-be1-") for row in rows) == 10
        assert sum(row["trial_id"].startswith("ship-be1-") for row in rows) == 10
        assert all(set(row["event_counts"]) == EVENT_KEYS for row in rows)
        assert all(set(row["tool_counts"]) == TOOL_KEYS for row in rows)
        assert all(row["passed"] and all(row["predicates"].values()) for row in rows)
        assert len(summary["arm_statistics"]) == 4
        assert all(row["trials"] == 5 for row in summary["arm_statistics"])
        assert all(row["variance"] == 0 for row in summary["arm_statistics"])
        assert summary["decision"]["candidate"] in {"keep", "delete"}
        assert summary["decision"]["decision_reason_id"] in {
            "fake_evidence_only",
            "no_variant",
            "candidate_not_smaller",
        }
        assert summary["decision"]["candidate"] == "delete"
        assert_private(summary)
        assert "DO-NOT-INHERIT" not in fake.stdout

        failure, failure_dir = invoke(
            temporary,
            "--fake",
            "--fake-fault",
            "quick-be1-candidate-01:predicate-fail",
        )
        assert failure.returncode == 1, failure.stderr
        failed = load_summary(failure_dir)
        bad = [row for row in failed["results"] if not row["passed"]]
        assert len(bad) == 1
        assert bad[0]["failure_reason_id"] == "predicate_failed"
        candidate_quick = next(
            row
            for row in failed["arm_statistics"]
            if row["trial_group_id"] == "QUICK-BE1" and row["arm_id"] == "candidate"
        )
        assert candidate_quick["passed"] == 4
        assert candidate_quick["variance"] == 0.16
        assert 0 <= candidate_quick["confidence"]["low"]
        assert candidate_quick["confidence"]["high"] <= 1
        assert failed["decision"]["candidate"] == "delete"
        assert failed["decision"]["decision_reason_id"] == "candidate_regressed"

        interrupted, interrupted_dir = invoke(
            temporary,
            "--fake",
            "--fake-fault",
            "ship-be1-control-05:interrupted",
        )
        assert interrupted.returncode == 1, interrupted.stderr
        interrupted_summary = load_summary(interrupted_dir)
        interrupted_rows = [
            row
            for row in interrupted_summary["results"]
            if row["failure_reason_id"] == "interrupted"
        ]
        assert len(interrupted_rows) == 1

        budget, budget_dir = invoke(
            temporary,
            "--fake",
            "--fake-fault",
            "quick-be1-candidate-02:cost-overrun",
        )
        assert budget.returncode == 1, budget.stderr
        budget_summary = load_summary(budget_dir)
        exceeded = [
            row
            for row in budget_summary["results"]
            if row["failure_reason_id"] == "budget_exceeded"
        ]
        assert len(exceeded) == 1
        assert budget_summary["usage_totals"]["cost_usd"] == "2"

        privacy, privacy_dir = invoke(
            temporary,
            "--fake",
            "--fake-fault",
            "ship-be1-candidate-03:privacy-leak",
        )
        assert privacy.returncode == 2
        assert not (privacy_dir / "summary.json").exists()
        assert "forbidden retention key" in privacy.stderr
        assert "/operator/private/path" not in privacy.stderr

        live, live_dir = invoke(
            temporary,
            "--live",
            "--preflight-only",
            "--host",
            "claude",
        )
        assert live.returncode == 2
        assert not (live_dir / "summary.json").exists()
        assert "paid confirmation" in live.stderr

        binary = temporary / "bin"
        artifacts = temporary / "host-artifacts"
        binary.mkdir()
        artifacts.mkdir()
        fake_claude = binary / "claude"
        fake_claude.write_text(
            "#!/usr/bin/env sh\n"
            '[ "${1:-}" = "--version" ] && { echo claude-test/1; exit 0; }\n'
            "exit 99\n",
            encoding="utf-8",
        )
        fake_claude.chmod(0o755)
        credential = temporary / "claude.auth"
        credential.write_text("scoped-test-credential\n", encoding="utf-8")
        credential.chmod(0o600)
        ledger = temporary / "ledger.json"
        ledger.write_text(
            json.dumps(
                {
                    "schema": "devrites-agent-contract-budget/v1",
                    "month": datetime.now(timezone.utc).strftime("%Y-%m"),
                    "spent_usd": "0",
                }
            )
            + "\n",
            encoding="utf-8",
        )
        ledger.chmod(0o600)
        preflight, preflight_dir = invoke(
            temporary,
            "--live",
            "--preflight-only",
            "--host",
            "claude",
            "--confirm-live",
            "RUN-PAID-HOST-CONTRACTS",
            "--host-version",
            "claude=claude-test/1",
            "--host-model",
            "claude=claude-test-model",
            "--auth-file",
            f"claude={credential}",
            "--host-artifacts",
            str(artifacts),
            "--max-cost-per-run-usd",
            "0.50",
            "--max-monthly-cost-usd",
            "10.00",
            "--budget-ledger",
            str(ledger),
            extra_env={"PATH": f"{binary}:{os.environ.get('PATH', '')}"},
        )
        assert preflight.returncode == 0, preflight.stderr
        preflight_summary = load_summary(preflight_dir)
        assert preflight_summary["sessions_total"] == 20
        assert preflight_summary["max_live_cost_usd"] == "10.00"
        assert preflight_summary["monthly_ceiling_usd"] == "10.00"
        assert preflight_summary["ready"] is True
        assert_private(preflight_summary)

        leftovers = list((temporary / "tmp").glob("devrites-controlled-behavioral-*"))
        assert not leftovers

    print("live-behavioral-runner-test: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

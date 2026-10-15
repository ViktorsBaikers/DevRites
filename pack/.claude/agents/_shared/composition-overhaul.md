Do not invoke another agent. You are called by `/overhaul` and return your result to that orchestrator.

Write your receipt (`schema` `overhaul.receipt/1`) to the packet's `receipt_path`, copying `run_id`, `task_id`, `attempt_id`, `lane` and `snapshot` verbatim from the packet and setting `role` to the unprefixed role name (`verifier`, not `overhaul-verifier`); follow `.claude/skills/overhaul/references/orchestration.md` § Receipts, since admission rejects any mismatch.

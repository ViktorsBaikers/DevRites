# `--cross-model`: explicit outside integration

This optional pass runs only when the user supplies `--cross-model` and a
different-model plugin, MCP server, or other native integration is explicitly
connected. DevRites does not discover aliases, invent a fallback model, or wrap
availability.

Give the integration the same immutable `spec.md`, `plan.md`, and `tasks.md`
candidate, without author reasoning. Ask for evidence-backed findings on
boundaries, scope/reuse, proof design, performance, reversibility, and failure
modes. The integration must remain read-only.

If no explicit integration is connected, record `requested but not configured`
and continue with the native `devrites-plan-reviewer`; absence is neither a pass
nor a blocker.

Verify every returned finding against the project before accepting it. Record
only the integration used and accepted evidence in `eng-review.md`; do not
maintain availability aliases or outside-review telemetry.

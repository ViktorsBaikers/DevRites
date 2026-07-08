#!/usr/bin/env bash
# mcp-test.sh — verifies MCP tool calls invoke the engine binary, not deleted
# devrites-lib shell wrappers.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v node >/dev/null 2>&1 || { echo "mcp-test: SKIP (node not found)"; exit 0; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
PROJECT="$T/project"
FAKEBIN="$T/fakebin"
mkdir -p "$PROJECT" "$FAKEBIN" "$PROJECT/docs" "$PROJECT/.devrites/work/alpha" "$PROJECT/.devrites/work/docs"

cat > "$FAKEBIN/devrites-engine" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$DEVRITES_FAKE_ARGS"
case "$1" in
  build-readiness) printf 'readiness: OK %s\n' "${2:-}" ;;
  check-acceptance) printf 'check-acceptance: OK %s\n' "${2:-}" ;;
  review-integrity) printf 'review-integrity: OK %s\n' "${2:-}" ;;
  spec-validate) printf 'spec-validate: OK %s\n' "${2:-}" ;;
  *) printf 'unexpected command: %s\n' "$*" >&2; exit 9 ;;
esac
SH
chmod +x "$FAKEBIN/devrites-engine"

echo "== mcp-test (target: $PROJECT) =="

printf '%s\n%s\n%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"devrites_ready","arguments":{"slug":"alpha"}}}' \
  '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"devrites_acceptance","arguments":{"slug":"alpha"}}}' \
  '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"devrites_use","arguments":{"slug":"alpha"}}}' \
  '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"devrites_use","arguments":{"slug":"missing"}}}' \
  '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"devrites_use","arguments":{"slug":"../alpha"}}}' \
  '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"devrites_active","arguments":{}}}' \
  '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"devrites_acceptance","arguments":{"slug":"docs"}}}' \
  '{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"devrites_review_integrity","arguments":{"slug":"alpha"}}}' \
  '{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"devrites_spec_validate","arguments":{"path":".devrites/work/alpha/spec.md"}}}' \
  '{"jsonrpc":"2.0","id":11,"method":"tools/list","params":{}}' \
  | (cd "$PROJECT" && DEVRITES_CLI="$FAKEBIN/devrites-engine" DEVRITES_FAKE_ARGS="$T/args" node "$ROOT/mcp/devrites-mcp.mjs") > "$T/mcp.jsonl" 2> "$T/mcp.err"

if grep -q 'readiness: OK alpha' "$T/mcp.jsonl"; then
  ok "devrites_ready returns engine output"
else
  no "devrites_ready did not return engine output"
fi

if grep -q 'check-acceptance: OK .devrites/work/alpha' "$T/mcp.jsonl"; then
  ok "devrites_acceptance resolves slug to workspace dir"
else
  no "devrites_acceptance did not resolve slug to workspace dir"
fi

if grep -q '^build-readiness alpha$' "$T/args" && grep -q '^check-acceptance .devrites/work/alpha$' "$T/args"; then
  ok "MCP tools/call invokes devrites engine subcommands"
else
  no "MCP tools/call invoked unexpected commands"
  [ -f "$T/args" ] && sed -n '1,40p' "$T/args"
fi

if grep -q 'review-integrity: OK alpha' "$T/mcp.jsonl" && grep -q '^review-integrity alpha$' "$T/args"; then
  ok "devrites_review_integrity invokes the engine"
else
  no "devrites_review_integrity did not invoke the engine"
fi

if grep -q 'spec-validate: OK .devrites/work/alpha/spec.md' "$T/mcp.jsonl" && grep -q '^spec-validate .devrites/work/alpha/spec.md$' "$T/args"; then
  ok "devrites_spec_validate passes path arguments to the engine"
else
  no "devrites_spec_validate did not pass path arguments to the engine"
fi

if grep -q '"name":"devrites_test_integrity"' "$T/mcp.jsonl" && grep -q '"name":"devrites_package_existence"' "$T/mcp.jsonl"; then
  ok "tools/list exposes expanded engine gate surface"
else
  no "tools/list does not expose expanded engine gate surface"
fi

if grep -q '"id":4,.*"text":"alpha"' "$T/mcp.jsonl"; then
  ok "devrites_use accepts an existing workspace slug"
else
  no "devrites_use did not accept existing workspace slug"
fi

if grep -q '"id":5,.*"isError":true' "$T/mcp.jsonl" && grep -q 'unknown workspace slug: missing' "$T/mcp.jsonl"; then
  ok "devrites_use rejects nonexistent workspace slug"
else
  no "devrites_use did not reject nonexistent workspace slug"
fi

if grep -q '"id":6,.*"isError":true' "$T/mcp.jsonl" && grep -q 'invalid slug: ../alpha' "$T/mcp.jsonl"; then
  ok "devrites_use rejects path-like slug"
else
  no "devrites_use did not reject path-like slug"
fi

if grep -q '"id":7,.*"text":"alpha"' "$T/mcp.jsonl" && [ "$(cat "$PROJECT/.devrites/ACTIVE")" = "alpha" ]; then
  ok "devrites_use preserves active workspace after rejected slug"
else
  no "devrites_use did not preserve active workspace after rejected slug"
fi

if grep -q 'check-acceptance: OK .devrites/work/docs' "$T/mcp.jsonl" && grep -q '^check-acceptance .devrites/work/docs$' "$T/args" && ! grep -q '^check-acceptance docs$' "$T/args"; then
  ok "devrites_acceptance resolves colliding slug inside .devrites/work"
else
  no "devrites_acceptance used project-relative path for colliding slug"
fi

if grep -q 'devrites-lib/scripts/devrites.sh' "$T/args" "$T/mcp.jsonl" "$T/mcp.err" 2>/dev/null; then
  no "MCP still references deleted devrites.sh wrapper"
else
  ok "MCP no longer references deleted devrites.sh wrapper"
fi

[ "$fail" -eq 0 ] && echo "mcp-test: PASS" || echo "mcp-test: FAIL"
exit "$fail"

#!/usr/bin/env bash
# Validation for cim-nb5w codex-imagegen-mcp. Set CODEX_IMAGEGEN_E2E=1 to include
# the two real-codex generation steps (each ~1-2 min).
set -euo pipefail
cd "$(dirname "$0")/.."
fail() { echo "VALIDATION FAILED: $1" >&2; exit 1; }

echo "[1/6] gofmt + go vet"
unformatted=$(gofmt -l .)
[ -z "$unformatted" ] || fail "gofmt: unformatted files: $unformatted"
go vet ./... || fail "go vet"

echo "[2/6] build binary"
go build -o codex-imagegen-mcp . || fail "build"

echo "[3/6] unit tests (race detector)"
go test -race ./... || fail "unit tests"

echo "[5/6] stdio smoke: tools/list + check_codex"
smoke=$( { \
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"validate","version":"0"}}}'; sleep 0.4; \
  echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'; sleep 0.2; \
  echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'; sleep 0.3; \
  echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"check_codex","arguments":{}}}'; sleep 0.8; \
} | ./codex-imagegen-mcp 2>/dev/null )
for tool in generate_image check_codex list_recent; do
  echo "$smoke" | grep -q "\"$tool\"" || fail "tools/list missing $tool"
done
echo "$smoke" | grep -q '"codex_installed":true' || fail "check_codex: codex_installed not true"
echo "$smoke" | grep -q '"logged_in":true'       || fail "check_codex: logged_in not true"

if [ "${CODEX_IMAGEGEN_E2E:-}" = "1" ]; then
  echo "[4/6] real codex pipeline E2E (Run -> FindRollout -> ExtractPNG)"
  go test ./internal/codex/ -run TestRunRealImagegen -count=1 -v || fail "codex pipeline E2E"
  echo "[6/6] real generate_image -> list_recent E2E"
  go test ./internal/tools/ -run TestGenerateAndListRecentE2E -count=1 -v || fail "tools E2E"
else
  echo "[4/6 & 6/6] SKIPPED real-codex E2E (set CODEX_IMAGEGEN_E2E=1 to include)"
fi

echo "VALIDATION PASSED"

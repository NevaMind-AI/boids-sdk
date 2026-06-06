#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PYTHON="${PYTHON:-$(command -v python3 || command -v python || true)}"
NODE="${NODE:-$(command -v node || true)}"
GO="${GO:-$(command -v go || true)}"

if [ -z "$PYTHON" ]; then
  echo "Could not find python3/python. Set PYTHON to the executable path." >&2
  exit 1
fi
if [ -z "$NODE" ]; then
  echo "Could not find node. Set NODE to the executable path." >&2
  exit 1
fi

echo "== python chat/complete =="
"$PYTHON" "$SCRIPT_DIR/python_chat_complete.py"
echo "== python responses =="
"$PYTHON" "$SCRIPT_DIR/python_response.py"

echo "== js chat/complete =="
"$NODE" "$SCRIPT_DIR/js_chat_complete.mjs"
echo "== js responses =="
"$NODE" "$SCRIPT_DIR/js_response.mjs"

if [ -n "$GO" ]; then
  echo "== go chat/complete =="
  ( cd "$SCRIPT_DIR" && "$GO" run ./go_chat_complete.go )
  echo "== go responses =="
  ( cd "$SCRIPT_DIR" && "$GO" run ./go_response.go )
else
  echo "Skipping Go tests because Go was not found. Set GO to run them." >&2
fi

echo "All tests passed."

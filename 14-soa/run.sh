#!/usr/bin/env bash
# Starts the 5 processes. The ESB listens on BASE_PORT (8080 by default); the others on the next ports.
set -euo pipefail
cd "$(dirname "$0")"

BASE_PORT="${BASE_PORT:-8080}"
BIN_DIR="${BIN_DIR:-}"
TEMP_BIN=""
if [ -z "$BIN_DIR" ]; then
  BIN_DIR="$(mktemp -d)"
  TEMP_BIN="$BIN_DIR"
  ./build.sh "$BIN_DIR"
fi

export ESB_URL="http://localhost:$BASE_PORT"
export DATABASE_URL="http://localhost:$((BASE_PORT + 1))"
export CUSTOMER_URL="http://localhost:$((BASE_PORT + 2))"
export SALES_URL="http://localhost:$((BASE_PORT + 3))"
export BILLING_URL="http://localhost:$((BASE_PORT + 4))"

pids=()
start() { PORT="$2" "$BIN_DIR/$1" & pids+=("$!"); }

trap 'kill "${pids[@]}" 2>/dev/null || true; [ -z "$TEMP_BIN" ] || rm -rf "$TEMP_BIN"; exit 0' INT TERM

start database $((BASE_PORT + 1))
start customer $((BASE_PORT + 2))
start sales    $((BASE_PORT + 3))
start billing  $((BASE_PORT + 4))
start esb      "$BASE_PORT"

echo "ESB at http://localhost:$BASE_PORT. Ctrl+C stops all services."
wait

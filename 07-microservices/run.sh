#!/usr/bin/env bash
# Starts the 6 processes. The gateway listens on BASE_PORT (8080 by default); the others on the next ports.
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

export CATALOG_URL="http://localhost:$((BASE_PORT + 1))"
export CUSTOMERS_URL="http://localhost:$((BASE_PORT + 2))"
export ORDERS_URL="http://localhost:$((BASE_PORT + 3))"
export PAYMENTS_URL="http://localhost:$((BASE_PORT + 4))"
export NOTIFICATIONS_URL="http://localhost:$((BASE_PORT + 5))"

pids=()
start() { PORT="$2" "$BIN_DIR/$1" & pids+=("$!"); }

trap 'kill "${pids[@]}" 2>/dev/null || true; [ -z "$TEMP_BIN" ] || rm -rf "$TEMP_BIN"; exit 0' INT TERM

start catalog       $((BASE_PORT + 1))
start customers     $((BASE_PORT + 2))
start orders        $((BASE_PORT + 3))
start payments      $((BASE_PORT + 4))
start notifications $((BASE_PORT + 5))
start gateway       "$BASE_PORT"

echo "gateway at http://localhost:$BASE_PORT. Ctrl+C stops all services."
wait

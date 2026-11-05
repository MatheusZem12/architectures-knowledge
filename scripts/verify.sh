#!/usr/bin/env bash
# For each architecture: builds it, runs its unit tests, starts the binary and runs the contract
# suite. Usage: ./scripts/verify.sh [folder...]   (no arguments checks them all)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${VERIFY_PORT:-18080}"
BIN="$(mktemp -d)"
trap 'rm -rf "$BIN"' EXIT

if ! command -v go >/dev/null; then
  echo "go not found. Run with: mise exec -- ./scripts/verify.sh" >&2
  exit 1
fi

wait_health() {
  for _ in $(seq 1 100); do
    curl -fs "http://localhost:$PORT/health" >/dev/null 2>&1 && return 0
    sleep 0.05
  done
  return 1
}

# Runs the suite against a running process and stops it at the end. Arguments: label, pid.
run_contract() {
  local label="$1" pid="$2" ok=0
  if ! wait_health; then
    echo "  [FAILED] $label: did not answer on /health"; ok=1
  elif (cd "$ROOT/contract" && BASE_URL="http://localhost:$PORT" go test -count=1 ./... >"$BIN/out.txt" 2>&1); then
    echo "  [ok] contract: $label ($(grep -c '^func Test' "$ROOT/contract/contract_test.go") tests)"
  else
    echo "  [FAILED] contract: $label"; grep -E '^(--- FAIL|    contract_test)' "$BIN/out.txt" | head -30; ok=1
  fi
  kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
  return $ok
}

failures=0
targets=("$@")
if [ ${#targets[@]} -eq 0 ]; then
  mapfile -t targets < <(cd "$ROOT" && ls -d 0*/ | tr -d /)
fi

for dir in "${targets[@]}"; do
  echo "== $dir"
  cd "$ROOT/$dir" || { failures=$((failures+1)); continue; }

  if [ -x ./run.sh ]; then           # several processes (microservices)
    if ! ./build.sh "$BIN/$dir"; then echo "  [FAILED] build"; failures=$((failures+1)); continue; fi
    ./test.sh || failures=$((failures+1))
    BIN_DIR="$BIN/$dir" BASE_PORT="$PORT" ./run.sh >"$BIN/$dir.log" 2>&1 &
    run_contract "gateway + services" $! || failures=$((failures+1))
    continue
  fi

  if ! go build -o "$BIN/$dir" . ; then echo "  [FAILED] build"; failures=$((failures+1)); continue; fi
  if go vet ./... && go test -count=1 ./... >"$BIN/unit.txt" 2>&1; then
    echo "  [ok] own tests: $(grep -c '^ok' "$BIN/unit.txt") package(s) with tests"
  else
    echo "  [FAILED] own tests"; cat "$BIN/unit.txt"; failures=$((failures+1))
  fi
  PORT="$PORT" "$BIN/$dir" >"$BIN/$dir.log" 2>&1 &
  run_contract "in memory" $! || failures=$((failures+1))

  if [ "$dir" = "05-hexagonal" ]; then # the hexagonal promise: swap the adapter without touching the core
    STORE=json DATA_FILE="$BIN/data.json" PORT="$PORT" "$BIN/$dir" >"$BIN/$dir.log" 2>&1 &
    run_contract "JSON file adapter" $! || failures=$((failures+1))
  fi
done

echo
if [ "$failures" -eq 0 ]; then echo "ALL GREEN"; else echo "$failures FAILURE(S)"; exit 1; fi

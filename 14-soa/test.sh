#!/usr/bin/env bash
# Runs the tests. All services share one module, unlike folder 07.
set -uo pipefail
cd "$(dirname "$0")"
if go vet ./... && go test -count=1 ./... >/dev/null 2>&1; then
  echo "  [ok] own tests: the whole module"
else
  echo "  [FAILED] own tests"; exit 1
fi

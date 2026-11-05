#!/usr/bin/env bash
# Runs each service's tests: each one is an independent Go module.
set -uo pipefail
cd "$(dirname "$0")/services"
failed=0
for service in */; do
  if (cd "$service" && go vet ./... && go test -count=1 ./... >/dev/null 2>&1); then
    echo "  [ok] own tests: ${service%/}"
  else
    echo "  [FAILED] own tests: ${service%/}"; failed=1
  fi
done
exit $failed

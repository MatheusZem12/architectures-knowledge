#!/usr/bin/env bash
# Shows where a rule lives in each architecture. Usage: ./scripts/where-rule.sh R08
#   --check  verifies that all rules are tagged in all architectures
cd "$(dirname "$0")/.." || exit 1

if [ "${1:-}" = "--check" ]; then
  missing=0
  for dir in [0-9][0-9]-*/; do
    for n in $(seq -w 1 14); do
      grep -rqE --include='*.go' --exclude='*_test.go' "\bR$n\b" "$dir" || { echo "missing: R$n in $dir"; missing=1; }
    done
  done
  [ "$missing" -eq 0 ] && echo "all rules tagged in all architectures"
  exit "$missing"
fi

rule="${1:?name the rule, for example R08}"
for dir in [0-9][0-9]-*/; do
  echo "== ${dir%/}"
  grep -rnE --include='*.go' --exclude='*_test.go' "\b$rule\b" "$dir" | sed -E "s#^${dir}##; s/:[[:space:]]+/: /" | cut -c1-110
done

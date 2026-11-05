#!/usr/bin/env bash
# Builds each service separately. Usage: ./build.sh <destination folder>
set -euo pipefail
cd "$(dirname "$0")/services"
DEST="$1"
mkdir -p "$DEST"
for service in */; do
  service="${service%/}"
  (cd "$service" && go build -o "$DEST/$service" .)
done

#!/usr/bin/env bash
# Builds each service. Usage: ./build.sh <destination folder>
set -euo pipefail
cd "$(dirname "$0")"
DEST="$1"
mkdir -p "$DEST"
for service in services/*/; do
  name="$(basename "$service")"
  go build -o "$DEST/$name" "./services/$name"
done

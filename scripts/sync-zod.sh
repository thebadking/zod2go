#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
git submodule update --init --recursive vendor/zod
echo "vendor/zod pinned at: $(git -C vendor/zod rev-parse --short HEAD) ($(git -C vendor/zod describe --tags --always 2>/dev/null || echo main))"

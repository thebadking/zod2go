#!/usr/bin/env bash
# Verify npm zod version matches vendor/zod submodule package.json
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VENDOR_VER=$(node -p "require('$ROOT/vendor/zod/packages/zod/package.json').version")
NPM_VER=$(node -p "require('$ROOT/node_modules/zod/package.json').version")
if [[ "$VENDOR_VER" != "$NPM_VER" ]]; then
  echo "Version mismatch: vendor/zod=$VENDOR_VER npm zod=$NPM_VER" >&2
  echo "Run: npm install zod@$VENDOR_VER" >&2
  exit 1
fi
echo "Zod versions aligned at $VENDOR_VER"

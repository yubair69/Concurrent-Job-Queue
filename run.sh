#!/usr/bin/env bash
# PixelForge - build the frontend if needed, then start the server.
# Usage:  ./run.sh          (build if stale, then run)
#         ./run.sh --fresh  (force a frontend rebuild)
set -euo pipefail

cd "$(dirname "$0")"

FRESH=0
[ "${1:-}" = "--fresh" ] && FRESH=1

command -v go >/dev/null || { echo "Missing 'go'. Install from https://go.dev/dl/"; exit 1; }
command -v npm >/dev/null || { echo "Missing 'npm'. Install Node.js from https://nodejs.org/"; exit 1; }

if [ ! -d web/node_modules ]; then
  echo "==> Installing frontend dependencies..."
  (cd web && npm install)
fi

needs_build=$FRESH
if [ ! -f web/dist/index.html ]; then
  needs_build=1
elif [ -n "$(find web/src web/index.html web/tailwind.config.ts web/vite.config.ts \
             -newer web/dist/index.html 2>/dev/null | head -1)" ]; then
  needs_build=1
fi

if [ "$needs_build" -eq 1 ]; then
  echo "==> Building frontend..."
  (cd web && npm run build)
else
  echo "==> Frontend already up to date (use --fresh to force a rebuild)"
fi

export GOTASK_PORT="${GOTASK_PORT:-8080}"
export GOTASK_WORKER_COUNT="${GOTASK_WORKER_COUNT:-4}"

command -v ffmpeg >/dev/null || \
  echo "note: ffmpeg not found - video jobs will run but produce simulated output"

echo "==> PixelForge running at http://localhost:${GOTASK_PORT}  (Ctrl+C to stop)"
exec go run ./cmd/server

#!/bin/sh
# Lance le serveur MCP gofact depuis le plugin Claude Code, en compilant le
# binaire au premier usage (il n'est pas versionné). stdout reste vierge pour
# le JSON-RPC : toute sortie de build part sur stderr.
set -e
cd "$(dirname "$0")/.."
if [ ! -x ./gofact ]; then
  echo "gofact: compilation du binaire (premier usage)…" >&2
  go build -trimpath -ldflags="-s -w" -o gofact . 1>&2
fi
exec ./gofact mcp

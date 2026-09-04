#!/bin/sh
# Lance le serveur MCP gofact depuis le plugin Claude Code.
#
# Ce script est lancé par le client IA, pas par un humain : il doit aboutir sur
# la machine d'un indépendant qui n'a ni Go, ni compilateur, ni patience. Il
# résout donc le binaire par ordre de coût croissant — déjà installé, sinon
# téléchargé depuis la release, et compilé seulement en dernier recours, quand
# le plugin est en fait un dépôt cloné par un développeur.
#
# stdout appartient au JSON-RPC : toute sortie humaine part sur stderr.
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REPO="kOlapsis/gofact"

# 1 — un binaire déjà présent, du plus proche au plus lointain.
for candidate in "$ROOT/gofact" "${XDG_BIN_HOME:-$HOME/.local/bin}/gofact"; do
	if [ -x "$candidate" ]; then exec "$candidate" mcp "$@"; fi
done
if command -v gofact >/dev/null 2>&1; then
	exec "$(command -v gofact)" mcp "$@"
fi

# 2 — la release. C'est le chemin normal pour qui installe le plugin.
fetch_release() {
	OS=$(uname -s | tr '[:upper:]' '[:lower:]')
	ARCH=$(uname -m); case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; esac
	command -v curl >/dev/null 2>&1 || return 1
	TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null |
		sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
	[ -n "$TAG" ] || return 1
	ASSET="gofact_${TAG#v}_${OS}_${ARCH}.tar.gz"
	echo "gofact: téléchargement de $TAG pour $OS/$ARCH…" >&2
	TMP=$(mktemp -d) || return 1
	if curl -fsSL "https://github.com/$REPO/releases/download/$TAG/$ASSET" -o "$TMP/a.tar.gz" 2>/dev/null &&
		tar -xzf "$TMP/a.tar.gz" -C "$ROOT" gofact 2>/dev/null; then
		chmod +x "$ROOT/gofact"; rm -rf "$TMP"; return 0
	fi
	rm -rf "$TMP"; return 1
}
if fetch_release; then exec "$ROOT/gofact" mcp "$@"; fi

# 3 — compilation, si et seulement si les sources et Go sont là.
if [ -f "$ROOT/go.mod" ] && command -v go >/dev/null 2>&1; then
	echo "gofact: compilation depuis les sources (premier usage, une trentaine de secondes)…" >&2
	(cd "$ROOT" && go build -trimpath -ldflags="-s -w" -o gofact .) 1>&2
	exec "$ROOT/gofact" mcp "$@"
fi

echo "gofact: aucun binaire utilisable. Installez-le avec" >&2
echo "  curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | sh" >&2
exit 1

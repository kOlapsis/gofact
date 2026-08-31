#!/bin/sh
# Installe gofact : télécharge le binaire de la dernière release pour la
# plateforme, le place dans ~/.local/bin, puis lui laisse enregistrer les
# clients MCP du poste. Usage : sh install.sh
set -e
REPO="kolapsis/gofact"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m); case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; esac
TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
[ -n "$TAG" ] || { echo "release introuvable" >&2; exit 1; }
URL="https://github.com/$REPO/releases/download/$TAG/gofact_${TAG#v}_${OS}_${ARCH}.tar.gz"
DIR="${XDG_DATA_HOME:-$HOME/.local}/bin"
mkdir -p "$DIR"
echo "→ $URL"
curl -fsSL "$URL" | tar -xz -C "$DIR" gofact
chmod +x "$DIR/gofact"
echo "✓ gofact $TAG installé dans $DIR"
echo
"$DIR/gofact" install
echo "Pour terminer : $DIR/gofact install -yes"

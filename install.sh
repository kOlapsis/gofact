#!/bin/sh
# Installe gofact : télécharge le binaire de la dernière release pour la
# plateforme, vérifie sa somme de contrôle, le place dans ~/.local/bin, puis lui
# laisse enregistrer les clients MCP du poste. Usage : sh install.sh
set -e
REPO="kOlapsis/gofact"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m); case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; esac
TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
[ -n "$TAG" ] || { echo "release introuvable" >&2; exit 1; }

ASSET="gofact_${TAG#v}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$TAG"
# XDG_BIN_HOME, pas XDG_DATA_HOME : ce dernier désigne ~/.local/share, et y
# poser un binaire l'expédie dans ~/.local/share/bin, que rien ne lit.
DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"
mkdir -p "$DIR"

TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
echo "→ $BASE/$ASSET"
curl -fsSL "$BASE/$ASSET" -o "$TMP/$ASSET"

# La somme de contrôle est publiée avec la release : la vérifier ne coûte qu'un
# téléchargement de 600 octets, et c'est la moindre des choses pour un binaire
# qui manipulera des factures.
if curl -fsSL "$BASE/checksums.txt" -o "$TMP/checksums.txt" 2>/dev/null; then
	EXPECTED=$(sed -n "s/^\([0-9a-f]\{64\}\)  *$ASSET$/\1/p" "$TMP/checksums.txt")
	if [ -n "$EXPECTED" ]; then
		if command -v sha256sum >/dev/null 2>&1; then
			ACTUAL=$(sha256sum "$TMP/$ASSET" | cut -d' ' -f1)
		elif command -v shasum >/dev/null 2>&1; then
			ACTUAL=$(shasum -a 256 "$TMP/$ASSET" | cut -d' ' -f1)
		fi
		if [ -n "$ACTUAL" ] && [ "$ACTUAL" != "$EXPECTED" ]; then
			echo "somme de contrôle incorrecte pour $ASSET — installation interrompue" >&2
			exit 1
		fi
		[ -n "$ACTUAL" ] && echo "✓ somme de contrôle vérifiée"
	fi
fi

tar -xzf "$TMP/$ASSET" -C "$DIR" gofact
chmod +x "$DIR/gofact"
echo "✓ gofact $TAG installé dans $DIR"

# ~/.local/bin n'est pas dans le PATH de tout le monde. Sans cet avertissement,
# la commande suivante de la documentation — « gofact install » — échoue avec un
# « command not found » qui ressemble à une installation ratée.
case ":$PATH:" in
	*":$DIR:"*) IN_PATH=1 ;;
	*) IN_PATH=0 ;;
esac

if [ "$IN_PATH" = "0" ]; then
	case "${SHELL##*/}" in
		zsh)  RC="$HOME/.zshrc" ;;
		fish) RC="$HOME/.config/fish/config.fish" ;;
		*)    RC="$HOME/.bashrc" ;;
	esac
	echo
	echo "⚠ $DIR n'est pas dans votre PATH."
	if [ "${SHELL##*/}" = "fish" ]; then
		echo "  Ajoutez-le : fish_add_path $DIR"
	else
		echo "  Ajoutez-le :  echo 'export PATH=\"\$PATH:$DIR\"' >> $RC"
		echo "  Puis :        source $RC"
	fi
	echo "  En attendant, gofact reste utilisable par son chemin complet."
fi

echo
"$DIR/gofact" install
echo
if [ "$IN_PATH" = "1" ]; then
	echo "Pour terminer : gofact install -yes"
else
	echo "Pour terminer : $DIR/gofact install -yes"
fi

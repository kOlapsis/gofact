#!/usr/bin/env sh
# Fabrique les bundles MCP (.mcpb) à partir des binaires produits par GoReleaser.
#
# Un .mcpb est une archive zip contenant un manifest.json et le serveur. C'est le
# seul chemin de publication au registre officiel MCP pour un projet qui n'est
# dans aucun registre de paquets (npm, PyPI, cargo…) : le registre accepte le
# type `mcpb` hébergé sur GitHub Releases. Le binaire Go étant statique et sans
# dépendance, le bundle se réduit au binaire et à son manifeste.
#
# Usage : scripts/mcpb.sh <dist> <version-sans-v>
set -eu

DIST=${1:?dist manquant}
VERSION=${2:?version manquante}
OUT="$DIST/mcpb"
DESCRIPTION="Creates compliant French e-invoices locally: Factur-X PDF/A-3 with EN 16931 CII XML, legal numbering"

command -v jq >/dev/null || { echo "jq requis" >&2; exit 1; }
command -v zip >/dev/null || { echo "zip requis" >&2; exit 1; }

rm -rf "$OUT"
mkdir -p "$OUT"

built=0

# La liste passe par un fichier : un `read` alimenté par un tuyau tourne dans un
# sous-shell, où un `exit` ne ferait pas échouer le script.
binaries=$(mktemp)
jq -r '.[] | select(.type == "Binary") | [.path, .goos, .goarch] | @tsv' "$DIST/artifacts.json" > "$binaries"

while IFS="$(printf '\t')" read -r path goos goarch; do
	case "$goos" in
		windows) platform=win32; binary=gofact.exe ;;
		darwin)  platform=darwin; binary=gofact ;;
		linux)   platform=linux; binary=gofact ;;
		*) echo "plateforme inattendue : $goos" >&2; exit 1 ;;
	esac

	work="$OUT/work-$goos-$goarch"
	mkdir -p "$work/server"
	cp "$path" "$work/server/$binary"
	chmod +x "$work/server/$binary"

	# manifest_version 0.3 : le champ server.type "binary" n'exige aucun runtime.
	jq -n --arg v "$VERSION" --arg bin "server/$binary" --arg p "$platform" \
	      --arg desc "$DESCRIPTION" '{
		manifest_version: "0.3",
		name: "gofact",
		display_name: "gofact",
		version: $v,
		description: $desc,
		author: { name: "Benjamin Touchard", url: "https://github.com/kOlapsis" },
		homepage: "https://gofact.kolapsis.com/",
		documentation: "https://gofact.kolapsis.com/",
		repository: { type: "git", url: "https://github.com/kOlapsis/gofact" },
		license: "AGPL-3.0-or-later",
		keywords: ["factur-x", "en16931", "facturation", "france", "invoice"],
		server: {
			type: "binary",
			entry_point: $bin,
			mcp_config: { command: ("${__dirname}/" + $bin), args: ["mcp"], env: {} }
		},
		compatibility: { platforms: [$p] }
	}' > "$work/manifest.json"

	# L'URL doit pointer une release GitHub : c'est un hébergeur autorisé par le
	# registre officiel, et le nom porte « mcp » pour rester lisible côté client.
	name="gofact-mcp_${VERSION}_${goos}_${goarch}.mcpb"
	(cd "$work" && zip -qr "../$name" .)
	rm -rf "$work"

	echo "$name  $(sha256sum "$OUT/$name" | cut -d' ' -f1)"
	built=$((built + 1))
done < "$binaries"
rm -f "$binaries"

[ "$built" -gt 0 ] || { echo "aucun binaire dans $DIST/artifacts.json" >&2; exit 1; }

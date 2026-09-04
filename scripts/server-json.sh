#!/usr/bin/env sh
# Renseigne server.json des bundles .mcpb d'une release donnée : URL de
# téléchargement et empreinte SHA-256, telles que le registre officiel les
# vérifiera. Les empreintes sont calculées sur les fichiers réellement publiés,
# jamais sur une copie locale — c'est ce que le client MCP téléchargera.
#
# Usage : scripts/server-json.sh <tag> <dossier-des-bundles>
set -eu

TAG=${1:?tag manquant}
DIR=${2:?dossier manquant}
REPO=kOlapsis/gofact
VERSION=${TAG#v}

command -v jq >/dev/null || { echo "jq requis" >&2; exit 1; }

packages=$(mktemp)
echo '[]' > "$packages"

for bundle in "$DIR"/*.mcpb; do
	[ -f "$bundle" ] || { echo "aucun bundle dans $DIR" >&2; exit 1; }
	name=$(basename "$bundle")
	sha=$(sha256sum "$bundle" | cut -d' ' -f1)
	url="https://github.com/$REPO/releases/download/$TAG/$name"
	jq --arg id "$url" --arg v "$VERSION" --arg sha "$sha" \
	   '. + [{ registryType: "mcpb", identifier: $id, version: $v, fileSha256: $sha,
	           transport: { type: "stdio" } }]' "$packages" > "$packages.new"
	mv "$packages.new" "$packages"
done

jq --arg v "$VERSION" --slurpfile pkgs "$packages" \
   '.version = $v | .packages = $pkgs[0]' server.json > server.json.new
mv server.json.new server.json
rm -f "$packages"

echo "server.json : version $VERSION, $(jq '.packages | length' server.json) paquets"

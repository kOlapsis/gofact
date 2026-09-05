#!/usr/bin/env bash
# Régénère la cask Homebrew et le manifeste Scoop depuis la dernière release.
#
# Les deux recettes vivent dans ce dépôt plutôt que dans un tap et un bucket
# séparés : Scoop accepte n'importe quel dépôt git, et Homebrew aussi dès lors
# qu'on lui donne l'URL complète — la convention `homebrew-<nom>` ne sert qu'à
# la forme courte de `brew tap`. Un dépôt de moins à tenir vaut la commande en
# deux temps.
#
# Écrire ici ne demande aucun secret : le GITHUB_TOKEN du dépôt y suffit.
#
# Usage : scripts/packaging.sh
set -euo pipefail

REPO=kOlapsis/gofact
DESC="Compliant French e-invoicing: HTML to Factur-X (PDF/A-3 + EN 16931 CII XML), locally"
HOME_PAGE="https://kolapsis.github.io/gofact/"

command -v jq >/dev/null || { echo "jq requis" >&2; exit 1; }

TAG=$(gh api "repos/$REPO/releases/latest" --jq .tag_name)
VERSION=${TAG#v}

sums=$(mktemp)
trap 'rm -f "$sums"' EXIT
curl -fsSL "https://github.com/$REPO/releases/download/$TAG/checksums.txt" -o "$sums"

need() {
	local v; v=$(awk -v f="$1" '$2 == f { print $1 }' "$sums" | head -1)
	[ -n "$v" ] || { echo "somme absente pour $1" >&2; exit 1; }
	printf '%s' "$v"
}

mkdir -p Casks bucket

cat > Casks/gofact.rb <<RUBY
# Généré par scripts/packaging.sh depuis la release $TAG — ne pas modifier à la main.
cask "gofact" do
  version "$VERSION"

  on_macos do
    on_arm do
      sha256 "$(need "gofact_${VERSION}_darwin_arm64.tar.gz")"
      url "https://github.com/$REPO/releases/download/v#{version}/gofact_#{version}_darwin_arm64.tar.gz"
    end
    on_intel do
      sha256 "$(need "gofact_${VERSION}_darwin_amd64.tar.gz")"
      url "https://github.com/$REPO/releases/download/v#{version}/gofact_#{version}_darwin_amd64.tar.gz"
    end
  end
  on_linux do
    on_arm do
      sha256 "$(need "gofact_${VERSION}_linux_arm64.tar.gz")"
      url "https://github.com/$REPO/releases/download/v#{version}/gofact_#{version}_linux_arm64.tar.gz"
    end
    on_intel do
      sha256 "$(need "gofact_${VERSION}_linux_amd64.tar.gz")"
      url "https://github.com/$REPO/releases/download/v#{version}/gofact_#{version}_linux_amd64.tar.gz"
    end
  end

  name "gofact"
  desc "$DESC"
  homepage "$HOME_PAGE"

  livecheck do
    skip "Auto-generated on release."
  end

  binary "gofact"

  # Le binaire n'est ni signé ni notarisé : sans cela macOS le met en quarantaine
  # et annonce une application « endommagée ».
  postflight do
    if OS.mac?
      system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/gofact"]
    end
  end

  caveats <<~CAVEATS
    gofact a besoin d'un navigateur Chrome pour le rendu PDF. S'il n'est pas
    détecté, indiquez son chemin dans GOFACT_CHROME.

    Pour déclarer le serveur MCP à votre client IA : gofact install
  CAVEATS
end
RUBY

jq -n --arg v "$VERSION" --arg repo "$REPO" --arg desc "$DESC" --arg home "$HOME_PAGE" \
      --arg a64 "$(need "gofact_${VERSION}_windows_amd64.zip")" \
      --arg arm "$(need "gofact_${VERSION}_windows_arm64.zip")" '{
	version: $v,
	description: $desc,
	homepage: $home,
	license: "AGPL-3.0-or-later",
	architecture: {
		"64bit": {
			url: "https://github.com/\($repo)/releases/download/v\($v)/gofact_\($v)_windows_amd64.zip",
			bin: ["gofact.exe"],
			hash: $a64
		},
		arm64: {
			url: "https://github.com/\($repo)/releases/download/v\($v)/gofact_\($v)_windows_arm64.zip",
			bin: ["gofact.exe"],
			hash: $arm
		}
	},
	checkver: { github: "https://github.com/\($repo)" },
	autoupdate: {
		architecture: {
			"64bit": { url: "https://github.com/\($repo)/releases/download/v$version/gofact_$version_windows_amd64.zip" },
			arm64: { url: "https://github.com/\($repo)/releases/download/v$version/gofact_$version_windows_arm64.zip" }
		}
	}
}' > bucket/gofact.json

echo "cask et manifeste régénérés en $VERSION"

# Généré par scripts/packaging.sh depuis la release v1.1.0 — ne pas modifier à la main.
cask "gofact" do
  version "1.1.0"

  on_macos do
    on_arm do
      sha256 "1a55fe986321b413730b6938c6900692b1a80d7222c64d6bd796e2c66f341516"
      url "https://github.com/kOlapsis/gofact/releases/download/v#{version}/gofact_#{version}_darwin_arm64.tar.gz"
    end
    on_intel do
      sha256 "85973a9bcfdfc60b9a3ca709f737e1c206b1570dae9f9e037c3980d758810f12"
      url "https://github.com/kOlapsis/gofact/releases/download/v#{version}/gofact_#{version}_darwin_amd64.tar.gz"
    end
  end
  on_linux do
    on_arm do
      sha256 "8c15b1371a17583ca7ee0ab9c30afcf82872711a10bc3fa85222bee77678a605"
      url "https://github.com/kOlapsis/gofact/releases/download/v#{version}/gofact_#{version}_linux_arm64.tar.gz"
    end
    on_intel do
      sha256 "f8404e6c1b777e1fcccd947acf2aea44cf81756b9566c312a691d60e1d00a087"
      url "https://github.com/kOlapsis/gofact/releases/download/v#{version}/gofact_#{version}_linux_amd64.tar.gz"
    end
  end

  name "gofact"
  desc "Compliant French e-invoicing: HTML to Factur-X (PDF/A-3 + EN 16931 CII XML), locally"
  homepage "https://kolapsis.github.io/gofact/"

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

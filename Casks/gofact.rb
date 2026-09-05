# Généré par scripts/packaging.sh depuis la release v1.2.0 — ne pas modifier à la main.
cask "gofact" do
  version "1.2.0"

  on_macos do
    on_arm do
      sha256 "7e09431c2f69b08a9b5970ecb43450f03428b13323e6d0e7c3fedc59854badf8"
      url "https://github.com/kOlapsis/gofact/releases/download/v#{version}/gofact_#{version}_darwin_arm64.tar.gz"
    end
    on_intel do
      sha256 "b02aff66fafba0b270a0f2a447dd783364ae4697e56ceeed1ac4ba6b3ff8bef8"
      url "https://github.com/kOlapsis/gofact/releases/download/v#{version}/gofact_#{version}_darwin_amd64.tar.gz"
    end
  end
  on_linux do
    on_arm do
      sha256 "5419eeaded45bfdec9ecab4e2605552acf5f35ca4bdb1d44a80b429d9127e3e5"
      url "https://github.com/kOlapsis/gofact/releases/download/v#{version}/gofact_#{version}_linux_arm64.tar.gz"
    end
    on_intel do
      sha256 "4bb548331a7e597cf7e80ed67c6a5299f59be912f01a05e93e5c78726dc527f5"
      url "https://github.com/kOlapsis/gofact/releases/download/v#{version}/gofact_#{version}_linux_amd64.tar.gz"
    end
  end

  name "gofact"
  desc "Compliant French e-invoicing: HTML to Factur-X (PDF/A-3 + EN 16931 CII XML), locally"
  homepage "https://gofact.kolapsis.com/"

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

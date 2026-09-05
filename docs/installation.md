# Installation

Deux étapes : obtenir le binaire, puis le déclarer auprès de vos clients MCP. La seconde
est automatisée par `gofact install`.

## 1 — Le binaire

=== "Linux / macOS"

    ```sh
    curl -fsSL https://raw.githubusercontent.com/kOlapsis/gofact/main/install.sh | sh
    ```

    Le script télécharge le binaire de la dernière release pour votre plateforme dans
    `~/.local/bin`, puis affiche ce que `gofact install` propose de configurer.

=== "Windows"

    ```powershell
    irm https://raw.githubusercontent.com/kOlapsis/gofact/main/install.ps1 | iex
    ```

    Installe dans `%LOCALAPPDATA%\gofact`. Edge étant préinstallé, aucun autre
    prérequis.

=== "Homebrew (macOS, Linux)"

    ```sh
    brew install kOlapsis/tap/gofact
    ```

    Tap personnel : Homebrew core impose un seuil de notoriété que gofact n'atteint
    pas encore. `brew upgrade` suit les releases.

=== "Scoop (Windows)"

    ```powershell
    scoop bucket add kolapsis https://github.com/kOlapsis/scoop-bucket
    scoop install gofact
    ```

    Bucket personnel, pour la même raison. `scoop update gofact` suit les releases.

=== "Paquet Debian / RPM / Alpine"

    Chaque release publie un `.deb`, un `.rpm` et un `.apk` pour amd64 et arm64.

    ```sh
    # Debian, Ubuntu
    curl -fsSLO https://github.com/kOlapsis/gofact/releases/latest/download/gofact_1.1.0_linux_amd64.deb
    sudo dpkg -i gofact_1.1.0_linux_amd64.deb
    ```

    L'intérêt sur le script : une désinstallation propre (`apt remove gofact`).

=== "Depuis les sources"

    ```sh
    git clone https://github.com/kOlapsis/gofact && cd gofact
    go build -trimpath -ldflags="-s -w" -o gofact .
    ```

    Nécessite Go ≥ 1.24. C'est la seule méthode qui demande une toolchain.

**Prérequis unique** : un navigateur Chrome, Edge, Brave ou Chromium. gofact le détecte
seul sur les trois OS ; `GOFACT_CHROME=/chemin/vers/exécutable` force un choix.

!!! warning "Linux : éviter le chromium snap/flatpak"
    Les paquets confinés ne peuvent pas lire les fichiers temporaires que gofact leur
    soumet — ils sont automatiquement écartés de la détection. Installez le paquet
    classique de votre distribution ou Chrome.

## 2 — Déclarer le serveur MCP

```sh
gofact install        # montre ce qui serait configuré (rien n'est modifié)
gofact install -yes   # applique
```

`gofact install` détecte les clients présents et écrit leur configuration :

| Client | Méthode |
|---|---|
| **Claude Code** | via sa propre CLI (`claude mcp add --scope user`) |
| **Claude Desktop** | `claude_desktop_config.json` (emplacement selon l'OS) |
| **LM Studio** | `~/.lmstudio/mcp.json` |
| **Cursor** | `~/.cursor/mcp.json` |

Chaque fichier est **sauvegardé** avant modification (`.bak-` horodaté), l'écriture est
atomique, et une entrée `gofact` existante qui pointerait ailleurs n'est jamais écrasée
sans `-force`. Redémarrez ensuite le client pour qu'il découvre le serveur.

## Alternative : le plugin Claude Code

Pour les utilisateurs de Claude Code, le dépôt est aussi un plugin :

```
/plugin marketplace add kOlapsis/gofact
/plugin install gofact@gofact
```

Le plugin apporte le skill `creer-facture` **et** déclare le serveur MCP.

!!! note "Ce que le plugin n'installe pas"
    Le plugin compile le binaire au premier usage (**toolchain Go requise**) et ne le
    place pas dans votre `PATH` : hors de Claude Code, il n'y a pas de commande
    `gofact`. Il ne configure pas non plus Claude Desktop, LM Studio ni Cursor.
    Pour tout cela, la voie complète reste : binaire de release + `gofact install -yes`.

## Vérifier

```sh
gofact org list
```

Si aucun client MCP ni organisation n'existe encore, c'est normal : passez aux
[premiers pas](demarrage.md).

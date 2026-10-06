# Installer gofact en dix minutes, y compris sous Windows

**Réponse courte.** Installer gofact prend deux commandes et moins de dix
minutes, sur Linux, macOS ou Windows : une pour obtenir le binaire, une pour
le déclarer auprès de votre IA. Il n'y a rien d'autre à faire tourner — ni
Java, ni base de données, ni service à démarrer. Voici le déroulé exact, OS
par OS, et ce que chaque étape fait réellement.

---

## Ce qu'il faut avant de commencer

Une seule chose : un navigateur **Chrome, Edge, Brave ou Chromium** installé.
gofact l'utilise pour composer la facture en HTML avant de la transformer en
Factur-X — c'est la seule dépendance externe du binaire, et elle tient déjà
sur la plupart des machines.

Sous Windows, Edge est préinstallé : il n'y a donc, dans la grande majorité
des cas, strictement rien à installer avant l'étape 1.

!!! warning "Linux : éviter le Chromium confiné (snap, flatpak)"
    Les paquets Chromium distribués en snap ou en flatpak s'exécutent dans un
    bac à sable qui ne peut pas lire les fichiers temporaires que gofact lui
    soumet. gofact les écarte automatiquement de la détection. Le paquet
    Chromium classique de votre distribution, ou Chrome, fonctionnent sans
    cette limite.

## Étape 1 — Le binaire

=== "Windows"

    Ouvrez PowerShell et lancez :

    ```powershell
    irm https://raw.githubusercontent.com/kOlapsis/gofact/main/install.ps1 | iex
    ```

    Le script installe le binaire dans `%LOCALAPPDATA%\gofact`. Comme Edge est
    déjà là, c'est en général la seule commande à exécuter.

=== "Linux / macOS"

    Dans un terminal :

    ```sh
    curl -fsSL https://raw.githubusercontent.com/kOlapsis/gofact/main/install.sh | sh
    ```

    Le script télécharge le binaire de la dernière version pour votre
    plateforme dans `~/.local/bin`.

=== "Gestionnaire de paquets"

    Homebrew (macOS, Linux) :

    ```sh
    brew tap kOlapsis/gofact https://github.com/kOlapsis/gofact
    brew install kOlapsis/gofact/gofact
    ```

    Scoop (Windows) :

    ```powershell
    scoop bucket add gofact https://github.com/kOlapsis/gofact
    scoop install gofact
    ```

    Debian, Ubuntu (et équivalents `.rpm`, `.apk` pour Fedora et Alpine) :
    chaque version publie un paquet pour amd64 et arm64, téléchargeable
    directement depuis la page des versions du dépôt.

    Ces trois chemins ont un avantage que le script n'a pas : `brew upgrade`,
    `scoop update` ou `apt upgrade` suivent les nouvelles versions tout seuls.

=== "Depuis les sources"

    ```sh
    git clone https://github.com/kOlapsis/gofact && cd gofact
    go build -trimpath -ldflags="-s -w" -o gofact .
    ```

    La seule méthode qui demande une toolchain : Go 1.24 ou plus récent.

À la fin de cette étape, le binaire `gofact` est sur votre machine. Il ne
fait encore rien : il n'est connu de personne.

## Étape 2 — Le déclarer à votre IA

C'est l'étape qui change tout : gofact devient utilisable **en conversation**
une fois que votre client IA sait qu'il existe. Une seule commande s'en
charge :

```sh
gofact install        # montre ce qui serait configuré, sans rien modifier
gofact install -yes   # applique
```

Lancée sans `-yes`, la commande détecte les clients MCP présents sur votre
machine et affiche exactement ce qu'elle écrirait — sans toucher à rien. Vous
voyez avant d'agir.

`gofact install -yes` écrit alors la configuration propre à chaque client
trouvé :

| Client | Où gofact écrit |
| --- | --- |
| **Claude Code** | via sa propre CLI (`claude mcp add --scope user`) |
| **Claude Desktop** | son fichier de configuration, à l'emplacement standard de l'OS |
| **LM Studio** | `~/.lmstudio/mcp.json` |
| **Cursor** | `~/.cursor/mcp.json` |

Trois garanties tiennent sur cette étape, parce qu'une déclaration MCP ratée
casse un client entier au redémarrage :

- **Chaque fichier modifié est sauvegardé avant écriture** (horodaté,
  suffixe `.bak-...`) — vous pouvez toujours revenir en arrière à la main.
- **L'écriture est atomique** : le fichier de configuration n'est jamais
  laissé à moitié écrit, même si la commande est interrompue.
- **Une entrée `gofact` déjà présente, qui pointerait ailleurs, n'est jamais
  écrasée sans `-force`** — gofact ne remplace pas une configuration qui
  existe déjà sans qu'on le lui demande explicitement.

Redémarrez ensuite le client — il doit relire sa configuration pour
découvrir le nouveau serveur.

!!! note "Sans client MCP installé, ou pour l'automatiser"
    Chaque version publie aussi un bundle `.mcpb` par plateforme — une
    extension que Claude Desktop installe directement, sans passer par
    `gofact install`. Et le mode ligne de commande (`gofact cli ...`) fonctionne
    sans aucun client MCP, pour qui préfère scripter.

## Pourquoi deux étapes séparées, et pas une seule

On pourrait imaginer un seul script qui fait tout. gofact ne le fait pas,
volontairement : l'étape 1 ne touche qu'à un binaire dans un dossier qui vous
appartient ; l'étape 2 touche à la configuration d'un logiciel tiers (votre
client IA) que vous n'avez peut-être pas envie de modifier tout de suite, ou
pas de la même façon partout. Séparer les deux, c'est pouvoir obtenir le
binaire sans rien déclarer, inspecter ce que `gofact install` ferait avant de
l'appliquer, et refaire la déclaration plus tard sans retélécharger quoi que
ce soit.

## Vérifier que tout est en place

Deux commandes suffisent à confirmer que les deux étapes ont pris :

```sh
gofact version   # le binaire répond, et avec quelle version
gofact install   # sans -yes : réaffiche ce qui est détecté, sans rien changer
```

Si `gofact install` relance affiche encore votre client comme non configuré
après l'avoir exécuté avec `-yes`, c'est presque toujours le même oubli :
le client n'a pas été redémarré, donc il n'a pas relu sa configuration.

## Et ensuite

Une fois les deux étapes faites, tout se passe en conversation avec votre
IA — créer votre dossier d'organisation, puis votre première facture — décrit
dans [Premiers pas](../demarrage.md). gofact compose alors la facture, vérifie
sa conformité avant de l'écrire, et la tient prête à déposer. Le dépôt
lui-même passe par votre plateforme agréée : gofact produit le fichier,
ce n'est pas lui qui le transmet.

---

## Sources

- [`install.sh`](https://github.com/kOlapsis/gofact/blob/main/install.sh) et
  [`install.ps1`](https://github.com/kOlapsis/gofact/blob/main/install.ps1) —
  scripts d'installation, dépôt `kOlapsis/gofact`.
- [Installation — documentation gofact](../installation.md), page de référence
  dont cet article reprend et commente le déroulé.
- [Page des versions — GitHub](https://github.com/kOlapsis/gofact/releases),
  pour les paquets `.deb`, `.rpm`, `.apk` et les bundles `.mcpb`.

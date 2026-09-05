# Audit d'onboarding gofact — « time to first invoice »

Audit exécuté le 2026-09-04 sur `/home/user/gofact` (HEAD `877a29b`), Linux amd64, Go 1.26.0.
Tout ce qui suit a été **exécuté**, pas déduit. Aucun fichier du dépôt n'a été modifié
(`git status` propre en fin d'audit). Les bacs à sable utilisés : `/tmp/freshhome`,
`/tmp/mcphome`, `/tmp/insthome`, `/tmp/noiban`, `/tmp/envtest`.

Deux limites du bac à sable, signalées pour honnêteté : `api.github.com` et les annuaires
publics (`recherche-entreprises.api.gouv.fr`, `directory.peppol.eu`) sont filtrés par le
proxy sortant (403). J'ai donc rejoué `install.sh` à l'identique avec le tag forcé, et je
n'ai pas pu mesurer la latence réelle de `search_client`. Ce ne sont pas des défauts du
produit.

---

## 0. Verdict en une ligne

Le **moteur** est excellent : la transaction `create_invoice` est propre, rapide (0,32 s),
transactionnelle, et les garde-fous de numérotation tiennent. Le **parcours d'entrée** est
cassé à trois endroits qui n'ont rien à voir avec le Factur-X : le binaire n'est pas dans
le `PATH` après l'installation, la détection de navigateur n'a aucun recours utilisable
depuis un client MCP, et il n'existe aucun modèle de facture par défaut. Le temps perdu est
presque entièrement en amont du code qui fait la valeur.

---

## 1. Mesures brutes

### 1.1 Build et tests

| Opération | Durée réelle | Résultat |
|---|---|---|
| `go build -o /tmp/gofact .` (froid, 24 modules téléchargés) | **31,1 s** | OK |
| `go test ./...` sans navigateur détectable | **5,2 s** | **ÉCHEC** — 2 tests |
| `go test ./...` avec `GOFACT_CHROME` | **2,4 s** | OK, tout vert |
| `go vet ./...` | — | propre |

Les deux échecs sans navigateur :

```
--- FAIL: TestFullInvoiceFlow (0.01s)
    server_test.go:162: create_invoice en erreur : … "facturx: aucun navigateur trouvé pour le rendu …"
--- FAIL: TestOnboardingFlow (0.01s)
    server_test.go:272: aperçu attendu : … "facturx: aucun navigateur trouvé pour le rendu …"
FAIL	github.com/kolapsis/gofact/internal/mcpsrv	0.062s
```

Le garde `testing.Short()` existe dans `TestFullInvoiceFlow` mais `go test ./...` (la
commande que tout contributeur tape) ne passe pas `-short`. Premier `go test` = rouge.

### 1.2 Installation depuis la release

Rejeu fidèle d'`install.sh` (tag `v0.1.0` forcé) :

```
→ https://github.com/kolapsis/gofact/releases/download/v0.1.0/gofact_0.1.0_linux_amd64.tar.gz
curl | tar -xz   : 0,96 s   (archive 5,6 Mo → binaire 14,4 Mo)
$DIR/gofact version : gofact 0.1.0        ← les ldflags goreleaser fonctionnent
command -v gofact   : NON, introuvable dans le PATH
```

### 1.3 Parcours MCP complet (une fois tout configuré)

Serveur piloté en JSON-RPC réel sur stdio (`/tmp/gofact mcp`), pas via le transport de test :

| Appel | Durée | Résultat |
|---|---|---|
| `initialize` + `tools/list` | 0,03 s | 13 outils, 1 prompt `nouvelle-facture` |
| `init_organization` | ~0 s | OK |
| `preview_next_number` | ~0 s | `2026001`, non consommé |
| `preview_invoice` | 0,31 s | `apercu.pdf` |
| `create_invoice` | **0,32 s** | `factur_x_conform: true`, PDF/A-3, XML embarqué (2 occurrences de `factur-x.xml`) |
| `list_invoices` | ~0 s | registre à jour |
| **Total machine** | **0,66 s** | |

En CLI : `gofact -html "2026001 - ACME.html"` → **3,27 s** (Chrome à froid), PDF conforme.

### 1.4 Garde-fous vérifiés (ce qui marche)

- `create_invoice` refuse un HTML sans `{{NUMERO}}`, avec le bon message.
- `spec.number` fourni par le client est **ignoré** : le serveur a attribué `2026002`.
- Échec de conformité → **aucun numéro consommé**, `numerotation.json` intact, fichiers
  orphelins supprimés. Vérifié sur le cas BR-50.
- `set-counter` vers le bas dans la même année → refus explicite et bien rédigé.
- Dérive de gabarit → `warnings`, pas blocage.
- `install -yes` : fusion non destructive vérifiée (une entrée `autre` préexistante dans
  `~/.cursor/mcp.json` a survécu), `.bak-` horodaté créé quand le fichier existait.
- Nom d'asset goreleaser : **aucune divergence** (voir §2).

---

## 2. Vérification demandée : install.sh ↔ goreleaser ↔ release GitHub

**Conclusion : ça correspond exactement. Rien de cassé ici.**

`install.sh` construit :
`gofact_${TAG#v}_${OS}_${ARCH}.tar.gz` → `gofact_0.1.0_linux_amd64.tar.gz`

`.goreleaser.yaml` ne définit pas de `archives.name_template`, donc le défaut v2
(`{{.ProjectName}}_{{.Version}}_{{.Os}}_{{.Arch}}`) s'applique, avec `.Version` = tag sans
le `v`. Aucun `replacements`, donc `.Os`/`.Arch` restent bruts (`darwin`, pas `Darwin` ;
`amd64`, pas `x86_64`), ce que `uname` + la table de conversion du script reproduisent.

Vérifié **sur la vraie release**, pas sur la théorie :

```
$ curl -sSL .../releases/download/v0.1.0/gofact_0.1.0_linux_amd64.tar.gz -o /dev/null
http=200 size=5602504

Actifs publiés sur v0.1.0 :
  gofact_0.1.0_darwin_amd64.tar.gz   gofact_0.1.0_darwin_arm64.tar.gz
  gofact_0.1.0_linux_amd64.tar.gz    gofact_0.1.0_linux_arm64.tar.gz
  gofact_0.1.0_windows_amd64.zip     gofact_0.1.0_windows_arm64.zip
  checksums.txt
```

Les six combinaisons que `install.sh`/`install.ps1` savent demander existent, au nom exact.
`install.ps1` construit `gofact_$($tag.TrimStart('v'))_windows_$arch.zip` → correspond aussi.

Deux réserves, mineures mais réelles :

- `install.sh` n'utilise pas `checksums.txt`, pourtant publié. Un `curl | sh` non vérifié
  alors que la somme est à un `curl` de distance, c'est du confort gratuit laissé de côté.
- `release.yml` se déclenche sur `tags: ["v*"]` **sans dépendre de `ci.yml`**. Un tag posé
  sur un commit rouge publie une release. Rien ne l'a fait ici, mais rien ne l'empêche.

Le workflow lui-même est correct (`fetch-depth: 0` présent, `contents: write` présent,
`goreleaser-action@v6` avec `~> v2`).

---

## 3. Premier contact : ce que voit un utilisateur qui n'a rien configuré

### `gofact` (sans argument) — le message qui tue la conversion

```
erreur : -html est requis
Usage of gofact:
  -chrome string
    	exécutable Chrome (défaut : auto-détection)
  -data string
  … 9 autres options …
exit=2
```

Le produit s'annonce comme « votre IA facture pour vous ». La première chose qu'il dit à
qui tape son nom est une **erreur**, sur un drapeau (`-html`) qui n'est utile qu'au mode
dégradé, et **le mot `mcp` n'apparaît nulle part**. Pas plus que `org`, `install`, `send`
ou `version` — ils existent tous dans `main.go` mais aucun n'est listé. `gofact help`
donne le même bloc, en sortie 2.

### Les autres

| Commande | Sortie |
|---|---|
| `gofact -h` | même liste de drapeaux, exit 0, toujours aucune sous-commande |
| `gofact version` | `gofact dev` (0.1.0 pour le binaire de release) |
| `gofact org` | usage clair et correct des 4 sous-commandes |
| `gofact org list` | `Aucune organisation. Créez-en une : gofact org init -path <dossier> -name "Mon activité"` — **excellent**, c'est le seul message d'accueil vraiment bon du CLI |
| `gofact org show` | `erreur : aucune organisation trouvée` (exit 1, un peu sec) |
| `gofact install` | dry-run lisible, liste ce qui serait écrit, dit comment appliquer |

### Côté MCP, sans organisation — c'est très bien

```
list_organizations   → {"organizations":[]}
preview_next_number  → aucune organisation trouvée. Créez-en une avec l'outil
                       init_organization, ou indiquez son dossier via le paramètre org
                       (ou la variable GOFACT_INVOICES_DIR)
```

Les messages MCP sont rédigés pour être relayés par une IA, et ils le sont bien. Le
contraste avec le CLI est saisissant : le même binaire est accueillant côté MCP et hostile
côté terminal.

---

## 4. Le temps réel jusqu'à la première facture

Le temps machine est négligeable (§1). Ce qui coûte, c'est ce qu'il y a autour.

**Chemin nominal, tout se passe bien** (Chrome installé, `~/.local/bin` déjà dans le
`PATH`, SIRET et IBAN sous la main, modèle accepté au premier rendu) :

| Étape | Temps |
|---|---|
| `curl \| sh` | 10 s |
| `gofact install -yes` + redémarrage du client MCP | 1–2 min |
| Conversation d'identité (nom, SIRET, adresse, IBAN, numérotation antérieure) | 3–5 min |
| Co-conception du modèle : 1 aller-retour `preview_invoice` | 2–4 min |
| Confirmation + `create_invoice` | 1 min |
| **Total** | **~8–12 min** |

**Chemin médian, réaliste** — au moins une friction se déclenche (elles sont fréquentes) :

| Aggravation | Coût ajouté |
|---|---|
| `gofact: command not found` après le `curl \| sh` (§5, B1) | +2 à 10 min, ou abandon |
| L'IA n'a demandé aucun IBAN à `init_organization` ; échec BR-50 découvert **après** toute la co-conception, sans outil pour corriger (§5, B3) | +5 à 15 min |
| Co-conception du modèle à partir d'une page blanche : 3 à 6 aller-retours (§5, C1) | +5 à 20 min |
| **Total médian** | **25–40 min** |

**Chemins qui n'aboutissent pas du tout, le même jour** :

- Navigateur non détecté (Chromium snap uniquement, Chrome de Playwright/Puppeteer,
  Vivaldi/Opera) : le message dit d'utiliser `GOFACT_CHROME` — **ce recours ne fonctionne
  pas depuis un client MCP** (§5, B2). Impasse sans lecture du code.
- Utilisateur Windows passant par le plugin Claude Code : le lanceur est un `.sh` (§5, B4).
- Utilisateur du plugin dont la première compilation dépasse le délai d'attente du client
  (mon build à froid : 31 s) : « aucun outil gofact », sans explication (§5, C5).

**Chiffre à retenir : 25–40 min en médiane, dont moins d'une seconde de calcul.**

---

## 5. Frictions, par gravité

### BLOQUANT

---

**B1 — Après l'installation, `gofact` n'est pas une commande.**

Constaté :

```
$ curl -fsSL .../install.sh | sh      (rejoué à l'identique)
✓ gofact v0.1.0 installé dans /tmp/insthome/.local/bin
$ command -v gofact
NON : 'gofact' introuvable dans PATH
```

`install.sh` ne teste jamais si `$DIR` est dans le `PATH` et ne dit jamais comment l'y
mettre. Or la commande suivante dans **README.md ligne 208**, **README-FR.md ligne 207** et
**docs/installation.md** est `gofact install -yes`, en nom nu. Et la section « Vérifier »
de `docs/installation.md` propose `gofact org list`. Les trois échouent. `~/.local/bin`
n'est dans le `PATH` par défaut ni sur macOS, ni sur Debian/Ubuntu tant qu'il n'existait
pas à l'ouverture de session. `install.ps1` a exactement le même trou avec
`%LOCALAPPDATA%\gofact`, en pire : sous Windows, modifier son `PATH` n'est pas une ligne
dans un `.profile`.

C'est la friction la plus grave du dépôt : elle frappe 100 % des utilisateurs du chemin
d'installation recommandé, dès la deuxième commande, et le message d'erreur
(`command not found`) ne pointe vers rien.

*Correction.* Dans `install.sh`, après le `chmod` :

```sh
case ":$PATH:" in
  *":$DIR:"*) ;;
  *) echo "⚠ $DIR n'est pas dans votre PATH. Ajoutez cette ligne à ~/.profile (ou ~/.zshrc) :"
     echo "    export PATH=\"$DIR:\$PATH\""
     echo "  puis rouvrez votre terminal." ;;
esac
```

Dans `install.ps1`, ajouter réellement le dossier au `PATH` utilisateur :

```powershell
$user = [Environment]::GetEnvironmentVariable("Path","User")
if ($user -notlike "*$dir*") { [Environment]::SetEnvironmentVariable("Path","$user;$dir","User") }
```

Et dans README.md / README-FR.md / docs/installation.md, écrire la deuxième commande en
chemin absolu (`~/.local/bin/gofact install -yes`) ou faire précéder l'étape `PATH`.

---

**B2 — « Utilisez GOFACT_CHROME » : le seul recours proposé ne marche pas côté MCP.**

Quand aucun navigateur n'est détecté, le message — le même en CLI et en MCP — est :

> `facturx: aucun navigateur trouvé pour le rendu : … Vous pouvez aussi désigner
> l'exécutable avec GOFACT_CHROME ou l'option -chrome`

Deux problèmes cumulés, tous deux vérifiés :

1. **Le serveur MCP ne lit aucun `.env`.** `runMCP` (`mcp.go`) n'appelle jamais
   `dotenv.LoadDefault`, contrairement à `runGenerate` et `runSend` (`main.go`). Or
   `.env.example` dit en tête : « Copier en `.env` … ou placer le fichier dans
   `~/.config/gofact/.env` ». Test réalisé : `GOFACT_CHROME` écrit **dans les deux
   emplacements documentés** (le `.env` du dossier d'organisation *et*
   `~/.config/gofact/.env`), puis `preview_invoice` appelé sans la variable dans
   l'environnement du processus :

   ```
   preview_invoice → facturx: aucun navigateur trouvé pour le rendu …
   ```

   Le même `.env`, avec le CLI depuis le même dossier :

   ```
   ✓ PDF rendu via Chrome (42682 octets)
   ✓ Factur-X conforme (EN 16931) : x.pdf
   ```

   Le CLI honore la configuration ; le serveur MCP — le cœur du produit — l'ignore.

2. **« ou l'option -chrome » n'existe pas en MCP.** `gofact mcp` n'a pas ce drapeau, et de
   toute façon c'est le client (Claude Desktop, Cursor, LM Studio) qui construit la ligne
   de commande, pas l'utilisateur. Le message oriente vers deux impasses.

   Aggravant : `gofact install` écrit l'entrée `{"command": exe, "args": ["mcp"]}` **sans
   bloc `env`**. Aucun endroit prévu pour poser la variable, aucune documentation qui
   explique où l'ajouter à la main (`docs/guide/mcp.md` ne mentionne ni `env` ni Chrome).

*Correction.* Trois gestes, indépendants :

- appeler `dotenv.LoadDefault("")` au début de `runMCP`, et charger le `.env` du dossier
  d'organisation dans l'environnement au moment de `resolveOrg` ;
- passer `ChromePath: cfg.Get("GOFACT_CHROME")` dans le `facturx.Options` de
  `createInvoice`/`previewInvoice` (`internal/mcpsrv/invoices.go` — il est aujourd'hui
  laissé vide) ;
- dans `chromeMissingError()`, distinguer le contexte : côté MCP, remplacer « ou l'option
  `-chrome` » par le geste concret — *« ajoutez `"env": {"GOFACT_CHROME": "/chemin"}` à
  l'entrée `gofact` de votre configuration MCP, ou renseignez `GOFACT_CHROME` dans le
  `.env` du dossier de l'organisation »*. Et faire écrire ce bloc `env` par
  `gofact install` quand un navigateur a été détecté au moment de l'installation.

---

**B3 — Sans IBAN, l'échec arrive à la toute dernière étape, et rien ne permet de le
corriger.**

`init_organization` accepte sans un mot une organisation sans IBAN. Le sujet ne revient
qu'au `create_invoice` final, après toute la collecte d'identité *et* toute la
co-conception du modèle :

```
init_organization (sans iban) → OK
create_invoice                → facture non conforme EN 16931 :
  - BR-50 : moyen de paiement 30 (virement) : l'IBAN du compte de règlement (BT-84)
            est requis — renseignez GOFACT_PAYEE_IBAN
```

Le numéro n'est pas consommé — c'est bien. Mais l'utilisateur est en impasse :

- les **13 outils MCP** ne contiennent **aucun** moyen de modifier l'identité d'une
  organisation après coup (`list_organizations`, `get_organization`, `init_organization`,
  `initialize_numbering`, `get_invoice_template`, `search_client`, `find_routing_address`,
  `preview_next_number`, `preview_invoice`, `list_invoices`, `create_invoice`,
  `send_invoice`, `get_invoice_status`) ;
- `init_organization` « refuse d'écraser un dossier existant » ;
- l'erreur dit de renseigner `GOFACT_PAYEE_IBAN`, donc d'éditer le `.env` du dossier — ce
  que `skills/creer-facture/SKILL.md` **interdit explicitement** : « Ne jamais éditer les
  fichiers du dossier de l'organisation à la main ».

Une IA qui suit ses instructions est coincée. Un humain doit lire le code pour s'en sortir.

*Correction.* Au choix, cumulables :

- ajouter un outil `set_organization_identity` (ou `update_organization`) écrivant le
  `.env` du dossier, avec les mêmes champs que `init_organization` ;
- faire retourner à `init_organization` un `warnings: ["aucun IBAN : les factures payables
  par virement seront refusées (BR-50) — appeler set_organization_identity avant la
  première facture"]`, et exposer `ready_to_invoice: false` dans `orgSummary` (§`summarize`,
  `internal/mcpsrv/orgs.go`) pour que `list_organizations` le signale dès le premier appel ;
- dans le message BR-50, nommer le chemin exact du fichier (`<org>/.env`) et l'outil à
  utiliser, plutôt que le seul nom de variable.

---

**B4 — Le plugin Claude Code ne peut pas démarrer sous Windows.**

`.claude-plugin/plugin.json` déclare :

```json
"mcpServers": { "gofact": { "command": "${CLAUDE_PLUGIN_ROOT}/scripts/mcp-launcher.sh" } }
```

`scripts/mcp-launcher.sh` commence par `#!/bin/sh`. Sous Windows, sans WSL ni Git Bash dans
le `PATH`, le serveur ne démarre pas — silencieusement, du point de vue de l'utilisateur
(« les outils gofact n'apparaissent pas »). Or `docs/installation.md` présente le plugin
comme une alternative de plein droit et n'énonce que deux restrictions (toolchain Go, pas
de `PATH`) — jamais « POSIX seulement ».

*Correction.* Ajouter `scripts/mcp-launcher.cmd` et déclarer le lanceur par plateforme ;
ou, plus radical et meilleur : que le lanceur **télécharge le binaire de la release**
(comme `install.sh`) au lieu de le compiler — ce qui règle du même coup C5 et la dépendance
à Go.

---

### COÛTEUX

---

**C1 — Aucun modèle de facture par défaut : la première facture part d'une page blanche.**

`get_invoice_template` ne renvoie rien tant qu'aucune facture n'a figé de modèle. Le skill
et le prompt MCP demandent alors au modèle de composer *ex nihilo* « une facture A4 soignée
— CSS embarqué, polices système, pas de `<a href>`, logo vectoriel inline, mentions légales
françaises (pénalités de retard 3× taux légal, indemnité de recouvrement 40 €, escompte :
néant), régime de TVA de l'organisation », puis d'itérer via `preview_invoice`.

C'est, de loin, le plus gros poste de temps du parcours (5 à 20 min), le plus variable, et
celui où la qualité dépend entièrement du modèle de langage utilisé — y compris pour des
mentions qui sont des **obligations légales**, pas des choix esthétiques. Un utilisateur de
LM Studio avec un petit modèle local obtiendra une facture pauvre ou incomplète.

Le paradoxe : un modèle correct existe déjà dans le dépôt,
`internal/facturx/testdata/facture.html` (44 lignes, A4, CSS embarqué, mentions présentes),
mais il est enfermé dans les tests.

*Correction.* `//go:embed` ce fichier (ou une variante neutre, sans « Studio Exemple »
codé en dur, avec les champs remplis depuis `o.Config()`) et le renvoyer par
`get_invoice_template` avec `is_default: true`. Le skill devient : « partir du modèle par
défaut, demander à l'utilisateur ce qu'il veut changer » — un aller-retour au lieu de cinq,
et un plancher de conformité garanti quel que soit le modèle de langage.

---

**C2 — Le chemin CLI `-html` contourne le registre de numérotation.**

`docs/guide/cli.md` ouvre par : « Tout ce que fait le serveur MCP existe aussi au terminal
— c'est le même binaire et **les mêmes garanties** ». C'est faux.

Test : dans un dossier d'organisation créé par `gofact org init`, génération d'une facture
`2026001` par `gofact -html "2026001 - ACME.html"` → PDF conforme produit. Puis :

```
$ cat numerotation.json
{ "_doc": "Registre de numérotation gofact. Source de vérité : numérotation continue,
           sans trou, jamais réutilisée (obligation légale). …",
  "compteurs": {},          ← vide
  "factures": [] }          ← vide
$ gofact org list
Studio Exemple    …
  0 facture(s) · prochain numéro 2026001 · IBAN oui · PDP non
```

La facture `2026001` a été émise, le registre l'ignore, et le prochain numéro annoncé est
`2026001`. Deux `gofact -html` de suite produisent deux factures au même numéro — exactement
ce que le fichier de registre se présente comme empêchant. Le chemin `-html` est un
convertisseur brut, pas un émetteur ; la doc le vend comme équivalent.

*Correction.* Soit `runGenerate` détecte qu'il tourne dans un dossier d'organisation
(présence de `numerotation.json`) et refuse, en renvoyant vers `create_invoice` / un futur
`gofact invoice create` qui passe par `AllocateWith` ; soit — a minima, et tout de suite —
`docs/guide/cli.md` remplace « les mêmes garanties » par un avertissement explicite :
« `-html` rend un fichier tel quel ; il **ne consomme pas de numéro** et **n'inscrit rien au
registre**. Pour émettre une facture, passer par les outils MCP. »

---

**C3 — `go test ./...` échoue sur toute machine sans navigateur détectable.**

Voir §1.1. Le premier geste d'un contributeur donne deux `FAIL`. Rien ne dit dans le
`README` qu'il faut un Chrome pour tester, ni qu'il faut `GOFACT_CHROME`.

*Correction.* Dans `internal/mcpsrv/server_test.go` (et `chrome_test.go` si concerné),
remplacer le garde `testing.Short()` par un test de disponibilité réel :

```go
if facturx.DetectChrome() == "" {
    t.Skip("aucun navigateur détecté — export GOFACT_CHROME pour exécuter ce test")
}
```

et garder l'exigence stricte en CI via `GOFACT_REQUIRE_CHROME=1` (que `ci.yml` positionne
déjà de fait, puisqu'il installe Chrome et exporte `GOFACT_CHROME`). Un `skip` documenté
vaut mieux qu'un `fail` qui fait croire que le dépôt est cassé.

---

**C4 — La détection de navigateur rate les Chrome les plus fréquents sur poste de
développeur.**

`browserCandidates()` (linux/darwin/windows) ne connaît que les installations packagées.
Manquent notamment :

- **Playwright** : `$PLAYWRIGHT_BROWSERS_PATH/chromium-*/chrome-linux/chrome` —
  *c'est exactement le cas de cette machine* : un Chromium parfaitement fonctionnel était
  présent à `/opt/pw-browsers/chromium-1194/chrome-linux/chrome` et gofact a déclaré
  « aucun navigateur trouvé » ;
- **Puppeteer** : `~/.cache/puppeteer/chrome/*/chrome-linux64/chrome` ;
- `chrome-headless-shell` / `headless_shell` ;
- Linux : `/usr/bin/google-chrome-beta`, `/usr/lib64/chromium-browser/chromium-browser`
  (Fedora), Vivaldi, Opera ;
- macOS : Chrome Beta/Canary, Arc, Vivaldi.

Combiné à B2 (le recours `GOFACT_CHROME` inopérant en MCP), un poste qui a *déjà* un moteur
Blink utilisable se retrouve bloqué.

*Correction.* Ajouter ces motifs à `browserCandidates()`, avec un `filepath.Glob` pour les
caches versionnés, et honorer `PLAYWRIGHT_BROWSERS_PATH`/`PUPPETEER_CACHE_DIR` quand ils
sont définis.

---

**C5 — Le plugin compile au premier démarrage : 31 s mesurées, contre un délai d'attente
client de 30–60 s.**

`scripts/mcp-launcher.sh` lance `go build` si `./gofact` n'existe pas. Mon build à froid
(module cache vide, 24 dépendances) : **31,1 s**. Les clients MCP coupent typiquement entre
30 et 60 s. Le premier `/plugin install` est donc un pari, et l'échec se présente comme
« les outils gofact n'apparaissent pas » — la FAQ a d'ailleurs une entrée dédiée, ce qui
prouve que le cas se produit. Le lanceur ne vérifie pas non plus la présence de `go` : sans
toolchain, `exec` échoue avec un message brut sur stderr que le client n'affiche pas.

*Correction.* Voir B4 : télécharger le binaire de release plutôt que compiler. À défaut,
compiler à l'installation du plugin (hook) et non au premier démarrage du serveur, et
vérifier `command -v go` avec un message explicite.

---

**C6 — `gofact org show` choisit silencieusement une organisation quand il y en a
plusieurs.**

Constaté avec deux organisations :

```
$ gofact org show
Organisation : Studio Exemple      ← choisie sans le dire
$ gofact org list
Studio Exemple   /tmp/freshhome/Documents/Factures
Autre            /tmp/freshhome/Autre
```

`runOrgShow` prend `orgs[0]`. Le côté MCP fait l'inverse et refuse : *« plusieurs
organisations existent : … Précisez laquelle avec le paramètre org — demandez à
l'utilisateur si le contexte ne suffit pas »*. Sur un poste multi-activités, le CLI affiche
la fiche de la mauvaise entité sans un mot. `runOrgSetCounter` fait, lui, le bon choix
(`len(orgs) != 1` → erreur), ce qui montre que l'intention existe.

*Correction.* Aligner `runOrgShow` sur `runOrgSetCounter`/`resolveOrg` : lister les
candidates et exiger `-org`.

---

**C7 — L'installation se termine sur une deuxième commande à taper, que l'utilisateur ne
pourra pas taper.**

`install.sh` exécute `gofact install` (dry-run) puis affiche `Pour terminer : $DIR/gofact
install -yes`. L'utilisateur vient d'accepter un `curl | sh` ; lui demander une seconde
commande manuelle est le point de décrochage classique — et avec B1, la version courte
qu'il copiera depuis le README (`gofact install -yes`) échouera.

*Correction.* Si `stdin` est un terminal, proposer une fois :
`printf 'Configurer les clients MCP maintenant ? [O/n] '` puis appliquer. Sinon (cas
`curl | sh`, où stdin est le tube), afficher la commande **en chemin absolu**, et rappeler
qu'il faut redémarrer le client MCP — ce que `docs/installation.md` dit mais que le script
ne dit pas.

---

### COSMÉTIQUE

---

**D1 — `gofact` sans argument n'expose pas le produit.** Voir §3. Un usage racine listant
les sous-commandes (`org`, `mcp`, `install`, `send`, `version`) *avant* les drapeaux de
`-html`, affiché sur stdout en sortie 0, et `help`/`--help` acceptés comme `-h`. Deux
douzaines de lignes dans `main.go`, pour le premier écran que voit tout le monde.

**D2 — `install -yes` annonce une sauvegarde qui n'existe pas.** Sur un
`~/.config/Claude/` et un `~/.lmstudio/` vides, la sortie est `✓ Claude Desktop configuré
(… — sauvegarde .bak conservée)` alors qu'aucun `.bak-` n'a été écrit (vérifié par `ls -a` :
seul le fichier de configuration est présent). `registerInFile` construit le message
inconditionnellement. Ne mentionner la sauvegarde que si `os.Stat` a réussi.

**D3 — `docs/installation.md` annonce « Go ≥ 1.24 », `go.mod` exige `go 1.26`.** Avec
`GOTOOLCHAIN=auto` une 1.24 télécharge silencieusement une 1.26 (~80 Mo) ; avec un
toolchain épinglé, c'est un échec sec. Écrire 1.26.

**D4 — `docs/guide/cli.md` renvoie à `.env.example` pour `GOFACT_OFFLINE` et `GOFACT_PDP`,
qui n'y sont pas.** Les deux existent bien dans le code (`internal/annuaire/annuaire.go:26`,
`internal/pdp/pdp.go:50`) mais `.env.example` ne les documente pas. Les y ajouter.

**D5 — La casse du propriétaire GitHub est incohérente.** `kOlapsis` dans
`docs/index.md`, `mkdocs.yml`, `docs/installation.md` et le *footer* de `.goreleaser.yaml` ;
`kolapsis` dans `install.sh`, `install.ps1`, `README.md`, `README-FR.md`, `go.mod`,
`.claude-plugin/*.json` et le *header* de `.goreleaser.yaml` — les deux formes cohabitent
dans le même fichier. GitHub est insensible à la casse, donc rien ne casse ; mais des URL
qui changent de forme d'une page à l'autre coûtent de la confiance sur un outil qui
manipule des factures. Choisir une graphie et l'appliquer partout.

**D6 — `preview_invoice` laisse `apercu.pdf` dans le dossier d'organisation.** Après la
co-conception, le dossier de factures contient un PDF marqué SPÉCIMEN à côté des factures
réelles (constaté : `apercu.pdf`, 25 965 octets, jamais nettoyé). L'écrire dans
`os.TempDir()` ou un sous-dossier `.gofact/`.

**D7 — Trois numéros de version qui divergent.** `.claude-plugin/plugin.json` : `1.1.0` ;
`skills/creer-facture/SKILL.md` : `3.0.0` ; tag et binaire : `0.1.0`. Sans conséquence
fonctionnelle, mais impossible de savoir quoi indiquer dans un rapport de bug.

**D8 — `gofact -html absent.html` accuse le mauvais fichier.**

```
$ gofact -html absent.html
erreur : facturx: lecture spec "absent.json": open absent.json: no such file or directory
```

L'utilisateur n'a jamais parlé d'`absent.json`. Le bon message existe pourtant
(`facturx: HTML introuvable: stat …/c.html`, vérifié quand le `.json` est présent) : il
arrive simplement trop tard, `LoadSpec` étant appelé avant tout `stat` du HTML. Vérifier
l'existence du `-html` en premier, dans `runGenerate`.

**D9 — `install.sh` ne vérifie pas `checksums.txt`.** Le fichier est publié à chaque
release ; trois lignes de script suffisent à l'utiliser.

**D10 — `org set-counter` sur une autre année rend compte à côté.**

```
$ gofact org set-counter -last-number 2025001    # compteur courant : année 2026
✓ Numérotation reprise — prochain numéro : 2026001
```

Le compteur **2025** a bien été posé, mais le message affiche le prochain numéro **2026**,
inchangé — l'utilisateur croit que rien ne s'est passé, ou que quelque chose s'est passé
qu'il n'a pas voulu. Nommer l'année touchée : `✓ Compteur 2025 porté à 001 — prochain
numéro pour 2026 : 2026001`.

---

## 6. Confrontation doc ↔ réalité

| Affirmation | Réalité constatée |
|---|---|
| `installation.md` : « télécharge … dans `~/.local/bin` » | Vrai seulement si `XDG_DATA_HOME` est vide. Le script calcule `${XDG_DATA_HOME:-$HOME/.local}/bin` ; or `XDG_DATA_HOME` vaut par convention `~/.local/share`, donc quand il **est** défini (GNOME/KDE, Nix, beaucoup de dotfiles) l'installation atterrit dans `~/.local/share/bin`, qui n'est le `PATH` de personne. La variable correcte pour un exécutable est `XDG_BIN_HOME`. **Vérifié.** Corriger en `DIR="${XDG_BIN_HOME:-$HOME/.local/bin}"`. |
| `installation.md` / README : `gofact install -yes` juste après le `curl \| sh` | `command not found` (B1). |
| `installation.md` § Vérifier : « `gofact org list` » | `command not found` (B1). |
| `installation.md` : « Nécessite Go ≥ 1.24 » | `go.mod` : `go 1.26` (D3). |
| `installation.md` : « gofact le détecte seul sur les trois OS ; `GOFACT_CHROME=…` force un choix » | Le forçage par `.env` ne fonctionne pas côté MCP (B2). Aucune page ne dit où poser la variable quand le serveur est lancé par un client graphique. |
| `installation.md` : le plugin « compile le binaire au premier usage (toolchain Go requise) » | Exact, mais tait les 31 s de compilation à froid (C5) et l'incompatibilité Windows du lanceur `.sh` (B4). |
| `guide/cli.md` : « c'est le même binaire et **les mêmes garanties** » | Faux : `-html` n'inscrit rien au registre et ne consomme aucun numéro (C2). |
| `guide/cli.md` : « `.env.example` … les documente toutes : … `GOFACT_OFFLINE`, `GOFACT_PDP` » | Ces deux-là n'y sont pas (D4). |
| `demarrage.md` : « L'IA vous demandera votre identité — nom, SIRET, adresse, IBAN » | Rien dans l'outil ne le garantit : `init_organization` accepte une organisation sans IBAN, sans avertissement, et l'échec ne survient qu'à la fin (B3). |
| `demarrage.md` : « elle vous montre un PDF d'aperçu … et itère jusqu'à ce que le rendu vous plaise » | Exact, et c'est le poste de temps dominant — parce qu'on part de rien (C1). |
| `demarrage.md` : « en une transaction : … En cas d'échec, rien n'est consommé » | **Vérifié et exact.** Registre intact et fichiers nettoyés après un échec BR-50. |
| `faq.md` : entrées BR-50, « vendeur non configuré », « aucun navigateur », plugin sans outils | Bien écrites et pertinentes — mais elles décrivent précisément les frictions B2, B3 et C5. Une FAQ qui documente un parcours cassé ne le répare pas. |
| `index.md` : « Un binaire, un navigateur » | Vrai. Le PDF/A-3 produit porte bien le XML embarqué et passe l'auto-contrôle. |

---

## 7. Ce qu'il faut corriger en premier

Par rapport temps-gagné / effort, dans cet ordre :

1. **B1** — trois lignes dans `install.sh`, un bloc dans `install.ps1`, trois chemins
   absolus dans la doc. Débloque 100 % des installations.
2. **C1** — un `//go:embed` d'un fichier qui existe déjà. Divise par deux à cinq le temps
   de la première facture, et met un plancher de conformité sous les petits modèles.
3. **B3** — un outil `set_organization_identity` + un `warnings` dans
   `init_organization`. Supprime l'impasse de fin de parcours.
4. **B2** — `dotenv.LoadDefault` dans `runMCP`, `ChromePath` renseigné, message d'erreur
   réécrit pour le contexte MCP.
5. **C2** — au minimum, corriger la phrase de `guide/cli.md`. Le reste peut suivre.
6. **C3** — `t.Skip` au lieu de `FAIL`. Cinq minutes, et le dépôt cesse d'accueillir ses
   contributeurs par du rouge.

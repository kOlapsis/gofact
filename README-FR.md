# gofact

> 🇬🇧 [English version](README.md)

Petit binaire Go qui transforme une **facture HTML** (prête à imprimer) en
**Factur-X** : un PDF/A-3 conforme avec le XML CII EN 16931 embarqué, et le
dépose optionnellement sur une plateforme de dématérialisation partenaire (PDP).

Pensé pour la facturation d'un indépendant ou d'une petite structure française
(réforme de la facturation électronique), et livré aussi comme **plugin Claude
Code** avec le skill `creer-facture`.

## Chaîne

```
JSON  ──règles EN 16931──▶  refus si non conforme (avant toute production)
      ──gofact──────────▶  factur-x.xml (CII EN 16931)
HTML  ──Chrome headless─▶  PDF          (rendu identique au Ctrl+P, arrière-plans inclus)
PDF + XML ──Go pur──────▶  Factur-X     (PDF/A-3 + embarquement VERBATIM du XML)
Factur-X  ──auto-contrôle▶ relecture    (structure + intégrité du XML embarqué)
```

> **Embarquement verbatim — important.** Le XML est inséré **octet pour octet**.
> Un assembleur qui le re-sérialise via son propre modèle **perd les champs
> étendus**, en particulier les adresses électroniques de routage PDP
> (BT-34/BT-49). gofact ne retouche jamais le XML qu'il a produit.

## Dépendances système

**Un navigateur, et c'est tout.**

- **Chrome, Edge, Brave ou Chromium** — rendu HTML → PDF

Détecté automatiquement sur Linux, macOS et Windows (où **Edge** est
préinstallé, donc rien à faire). Les paquets confinés — snap, flatpak — sont
écartés : ils ne peuvent pas lire les fichiers temporaires que gofact leur
soumet. Pour désigner un exécutable précis : `GOFACT_CHROME` ou `-chrome`.

L'assemblage PDF/A-3 est fait en Go (pdfcpu) : ni Ghostscript, ni Java, ni
téléchargement au premier lancement. Le profil ICC sRGB est embarqué dans le
binaire.

## Conformité

Deux vérifications, à deux moments différents :

- **Règles métier EN 16931**, appliquées sur le modèle **avant** de produire quoi
  que ce soit. gofact refuse d'émettre une facture qu'il sait non conforme, avec
  l'identifiant de la règle (`BR-50`, `BR-CO-15`…) et le champ à corriger.
- **Auto-contrôle du PDF**, après écriture : relecture du fichier, vérification
  des structures Factur-X (`/AF`, `/OutputIntents`, XMP, `/EmbeddedFiles`) et
  comparaison octet pour octet du XML embarqué avec celui qui a été généré.

La conformité PDF/A-3b de fond est vérifiée par **veraPDF**, via Mustang, en
intégration continue — jamais chez l'utilisateur :

```sh
GOFACT_ORACLE_HTML=facture.html go test -tags=ci ./internal/facturx
```

## Build

```sh
go build -o gofact .
go test ./...
```

## Configuration

**Aucune identité ni aucun secret n'est codé en dur.** L'émetteur des factures,
l'IBAN de règlement et les identifiants PDP viennent de l'environnement.

```sh
cp .env.example .env    # puis renseigner ses propres valeurs
```

`gofact` charge, dans l'ordre et sans jamais écraser une variable déjà définie :
le fichier passé à `-env`, sinon `./.env`, puis `$XDG_CONFIG_HOME/gofact/.env`
(`~/.config/gofact/.env` par défaut). Le `.env` est ignoré par git — ne jamais le
committer.

| Variable | Rôle |
| --- | --- |
| `GOFACT_SELLER_NAME` | Nom du vendeur. **Requis** sauf si le JSON porte un bloc `seller`. |
| `GOFACT_SELLER_SIRET` / `GOFACT_SELLER_SIREN` | Identifiant légal FR (BT-30). Le SIREN est dérivé du SIRET. |
| `GOFACT_SELLER_VAT_NUMBER` | N° TVA intracom (BT-31). Vide en franchise 293 B ⇒ BT-32 = SIREN. |
| `GOFACT_SELLER_EMAIL` | E-mail du vendeur. |
| `GOFACT_SELLER_ADDRESS` / `_POSTAL_CODE` / `_CITY` / `_COUNTRY` | Adresse postale (défaut pays `FR`). |
| `GOFACT_SELLER_ELECTRONIC_ADDRESS` / `_SCHEME` | Routage Peppol BT-34. Vide ⇒ SIREN + scheme `0225`. |
| `GOFACT_PAYEE_IBAN` | IBAN de règlement (BT-84). Vide ⇒ omis de la facture. |
| `GOFACT_VAT_EXEMPTION_MENTION` / `_CODE` | Mention et code d'exonération par défaut (293 B). |
| `SUPERPDP_CLIENT_ID` / `SUPERPDP_CLIENT_SECRET` / `SUPERPDP_BASE` | Identifiants PDP (**secrets**), pour `-send` uniquement. |

Sans vendeur configuré, `gofact` **échoue explicitement** plutôt que d'émettre une
facture incomplète. Chaque valeur reste surchargeable facture par facture via le JSON.

## Serveur MCP — parler à son IA pour facturer

`gofact mcp` expose la facturation en **serveur MCP local (stdio)** : depuis
Claude Desktop, Claude Code, LM Studio ou tout client MCP, dire « fais-moi une
facture pour ACME, 2 jours à 600 € » suffit. L'IA est l'interface ; gofact
garantit la numérotation (séquence légale continue, attribution verrouillée et
transactionnelle), la conformité EN 16931 et l'archivage — en local.

```sh
# Enregistrer dans Claude Code, par exemple :
claude mcp add gofact -- /chemin/vers/gofact mcp
```

Outils : `list_organizations`, `get_organization`, `init_organization`,
`search_client`, `get_invoice_template`, `preview_next_number`, `list_invoices`,
`create_invoice`, `send_invoice` (seul outil destructif — dépôt PDP,
confirmation explicite exigée), `get_invoice_status`. Prompt : `nouvelle-facture`.

Une **organisation** (entité émettrice) = un dossier autonome : identité dans
son `.env`, registre `numerotation.json`, journal d'audit `journal.ndjson`,
modèle de facture figé à la première émission, et les factures elles-mêmes.
Gestion au CLI : `gofact org list | init | show`. Aucun secret ne sort jamais
d'une réponse d'outil.

## Usage (CLI)

```sh
# Le plus simple : -data et -out déduits du nom du HTML
gofact -html "2026011 - Client.html"
#   → lit "2026011 - Client.json", écrit "2026011 - Client.pdf"

# Chemins explicites + dump du XML pour debug
gofact -html f.html -data f.json -out f.pdf -xml f.xml

# Options
-env <path>       # fichier .env explicite
-validate=false   # n'exécute pas l'auto-contrôle du PDF produit
-chrome <path>    # force l'exécutable Chrome
-q                # silencieux (n'affiche que les erreurs)
```

Code de sortie `0` si le Factur-X est généré (et conforme si `-validate`).

## Envoi à une PDP (SuperPDP)

`gofact` dépose le PDF Factur-X sur SuperPDP (OAuth2 client credentials).

```sh
# générer puis déposer en une commande
gofact -html f.html -send -poll
# déposer un PDF déjà généré
gofact send -pdf f.pdf -poll
```

> **La PDP exige que le vendeur de la facture = la société du compte authentifié**
> (identifiant légal BT-30) et une **adresse électronique de routage** (BT-34/BT-49,
> scheme `0225`) pour l'émetteur comme le destinataire. Voir `electronic_address`
> ci-dessous. `-poll` affiche le cycle de vie (`fr:200` déposée → `fr:201` émise →
> `fr:202` reçue).

## Format JSON (sidecar)

Tous les montants sont en **centimes** (entiers). Le vendeur, l'IBAN, le régime de
TVA et les mentions légales FR ont des **valeurs par défaut** (environnement) — le
JSON ne porte que ce qui varie.

```json
{
  "number": "2026011",
  "type": "invoice",
  "issue_date": "2026-06-29",
  "buyer_reference": "D2026004",
  "buyer": {
    "name": "ACME SAS",
    "siret": "55208131700015",
    "email": "compta@acme.example",
    "address": "1 rue de la Paix",
    "postal_code": "75002",
    "city": "Paris",
    "country_code": "FR"
  },
  "lines": [
    { "name": "Développement", "unit": "day", "quantity": "2.00",
      "unit_price_ht_cents": 60000, "amount_ht_cents": 120000 }
  ]
}
```

Champs optionnels : `due_date` (défaut « à réception » = émission),
`delivery_date` (défaut = émission), `currency` (défaut EUR), `vat`
(défaut exonéré 293 B ; mettre `{"exempt": false, "rate_pct": "20.00"}` pour la
TVA standard), `seller`, `iban`, `notes`.

**Routage PDP** : `seller`/`buyer` acceptent `electronic_address` (BT-34/BT-49) et
`electronic_address_scheme` (ex. `"0225"`). Requis pour l'envoi à une PDP, qui route
sur ces adresses plutôt que sur l'adresse postale. Exemple (override vendeur) :

```json
"seller": {
  "name": "Studio Exemple", "siren": "123456789",
  "electronic_address": "123456789", "electronic_address_scheme": "0225"
}
```

## Installation

**Binaire précompilé** (release GitHub, aucune toolchain requise) :

```sh
curl -fsSL https://raw.githubusercontent.com/kolapsis/gofact/main/install.sh | sh
gofact install -yes    # enregistre le serveur MCP dans les clients détectés
```

`gofact install` détecte Claude Desktop, Claude Code, LM Studio et Cursor, montre ce
qu'il compte écrire (dry-run par défaut), sauvegarde chaque fichier avant modification
et n'écrase jamais une entrée divergente sans `-force`. Windows : `install.ps1`.

**Plugin Claude Code** — le dépôt est aussi un plugin : skill `creer-facture` + serveur
MCP déclaré (compilé au 1ᵉʳ usage, toolchain Go requise pour ce build initial) :

```
/plugin marketplace add kolapsis/gofact
/plugin install gofact@gofact
```

**Depuis les sources** : `go build -o gofact .`

Les organisations vivent dans des **dossiers locaux** (identité, registre, factures) —
jamais versionnés ici : ils contiennent des données réelles. `GOFACT_INVOICES_DIR`
reste honorée pour la compatibilité avec le skill historique.

## Limites connues

- Le rendu repose sur un navigateur Blink ; les paquets confinés (snap,
  flatpak) sont ignorés car ils ne lisent pas hors de `$HOME`.
- Mono-taux de TVA. Le multi-taux n'est pas géré.
- Devise ≠ EUR : la contre-valeur TVA en EUR (BT-111) doit être fournie en amont
  (champ `VATEur` du modèle) ; non exposé dans le JSON pour l'instant.
- Une seule PDP implémentée (SuperPDP).

## Licence

**GNU AGPL v3 ou ultérieure** — voir [LICENSE](LICENSE).

Copyright (C) 2026 Benjamin Touchard.

gofact est un logiciel libre : vous pouvez le redistribuer et le modifier selon les termes de
la GNU Affero General Public License telle que publiée par la Free Software Foundation, en
version 3 ou toute version ultérieure. Il est distribué dans l'espoir qu'il sera utile, mais
**SANS AUCUNE GARANTIE**, sans même la garantie implicite de qualité marchande ou d'adéquation
à un usage particulier.

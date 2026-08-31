---
name: creer-facture
version: 2.0.0
description: Crée une nouvelle facture à partir du template HTML et du registre de numérotation du dossier de facturation, puis génère le PDF Factur-X conforme (PDF/A-3 + XML CII EN 16931) via gofact. Gère factures standard, acomptes et soldes. Met à jour la numérotation automatiquement.
triggers:
  - créer une facture
  - nouvelle facture
  - facturer
  - facture d'acompte
  - facture de solde
  - générer une facture
allowed-tools:
  - Read
  - Write
  - Edit
  - Bash
  - AskUserQuestion
---

# Créer une facture

Génère une facture à partir du template maître du dossier de facturation, en
attribuant le bon numéro depuis le registre et en le mettant à jour, puis produit le
PDF **Factur-X** (PDF/A-3 + XML CII EN 16931) via le binaire `gofact` (livré avec ce plugin).

## Dossier de travail

Le dossier de facturation est **local à l'utilisateur** et n'est jamais versionné dans
ce dépôt. Son chemin vient de la variable d'environnement `GOFACT_INVOICES_DIR` :

```sh
INVOICES="${GOFACT_INVOICES_DIR:?définir GOFACT_INVOICES_DIR (dossier de facturation)}"
```

Si la variable n'est pas définie, **demander le chemin à l'utilisateur** (AskUserQuestion)
et lui suggérer de l'exporter durablement. Structure attendue :

```
$GOFACT_INVOICES_DIR/
├── template-factures.html   # template maître — NE JAMAIS éditer pour un client
├── numerotation.json        # registre + compteurs (source de vérité)
├── README.md                # mode opératoire local, conventions, mentions
└── logo.svg                 # logo (référencé en relatif par le HTML)
```

> Le `README.md` du dossier porte les **conventions propres à l'émetteur** (régime de TVA,
> coordonnées bancaires, format des numéros, cas particuliers). Le lire au premier doute :
> il fait autorité sur ce fichier.

## Procédure

### 1 — Charger le contexte
- Lire `$INVOICES/numerotation.json` (compteurs + historique clients).
- Lire `$INVOICES/template-factures.html` (structure + placeholders).
- Lire `$INVOICES/README.md` pour les conventions de l'émetteur.
- Date du jour disponible dans le contexte de session (`currentDate`).

### 2 — Récolter les informations (AskUserQuestion)
Demander en groupant, et **ne rien inventer**. Champs requis :

- **Type** : `standard` (temps passé / forfait) · `acompte` · `solde`
- **Client** : nom, contact, adresse L1 (rue), adresse L2 (CP + ville), SIRET (optionnel)
  - Si le client existe déjà dans `numerotation.json`, réutiliser ses coordonnées et confirmer.
  - **Adresse de routage Peppol** (nécessaire pour l'envoi PDP) : réutiliser celle d'un
    sidecar précédent du même client ; pour un **client inconnu**, la rechercher dans l'annuaire
    Peppol (procédure à l'étape 5 bis).
- **Objet** (`[PROJET_TITRE]`) + contexte optionnel (`[CONTEXTE_TEXTE]`)
- **Prestations** :
  - standard → pour chaque ligne : titre, détail, quantité (j), PU HT
  - acompte/solde → pourcentage, montant, devis de référence
- **Méta** : date d'émission (défaut = aujourd'hui, format `29 mai 2026`), échéance (défaut `À réception`)
- **Si rattaché à un devis** : numéro de devis, échéancier complet (jalons %, libellés, montants), jalon courant, solde restant

### 3 — Attribuer le numéro
- `YYYY` = année courante. Si absente de `compteurs`, partir de `0`.
- `nouveau = compteurs[YYYY] + 1`, formaté `{YYYY}{NNN}` (NNN sur 3 chiffres).
- Ex. compteur 2026 = 6 → `2026007`.
- ⚠️ Numérotation continue, sans trou, jamais réutilisée (obligation légale).

### 4 — Calculer les montants
- `TOTAL_LIGNE = QTE × PU` ; `TOTAL_HT = Σ TOTAL_LIGNE` (ou montant unique acompte/solde).
- TVA selon le régime de l'émetteur (cf. README du dossier). En franchise en base
  (art. 293 B), `TOTAL_NET = TOTAL_HT`.
- Acompte : `montant = total_devis × pourcentage`. `SOLDE_RESTANT = total_devis − déjà facturé`.
- Format : séparateur de milliers = espace fine, décimales seulement si non rondes (`840 €`, `1 120 €`, `1 250,50 €`).

### 5 — Générer le HTML
- Copier le template vers `{YYYY}{NNN} - {Client}[ - Suffixe].html` dans `$INVOICES`.
  - Suffixes : ` - Acompte`, ` - Solde`, ` - Audit`… selon l'objet. Conserver casse/apostrophes/accents du nom client.
- Remplacer **tous** les `[PLACEHOLDERS]`.
- Table : type `standard` → garder VARIANTE A, supprimer le bloc commenté VARIANTE B.
  Type `acompte`/`solde` → supprimer VARIANTE A, décommenter et remplir VARIANTE B.
- Bloc « Suivi du règlement » (`.schedule`) : garder seulement si rattaché à un devis échelonné
  (jalon courant en `is-current` + tag « Présente facture » ; ajuster `grid-template-columns`
  au nombre de jalons). Sinon supprimer le `<h2>` + `.schedule` + paragraphe de solde.
- Pas de contexte → supprimer le `<div class="context">`. Adapter/supprimer `[MENTION_DEVIS]`.
- **Supprimer** le `<div class="edit-banner no-print">`.

### 5 bis — Émettre le sidecar JSON (Factur-X)
Écrire un fichier JSON **du même nom que le HTML** (extension `.json`) à côté de lui. Il
porte les données structurées que `gofact` projette en XML CII.

> L'identité du vendeur, l'IBAN de règlement et le régime de TVA par défaut viennent de
> l'**environnement** (`GOFACT_SELLER_*`, `GOFACT_PAYEE_IBAN`, cf. `.env.example` du plugin).
> Ne **pas** les répéter dans le JSON : il ne porte que ce qui varie d'une facture à l'autre.

- **Montants en centimes** (entiers) : `montant_€ × 100` (ex. 1 250,50 € → `125050`).
- `type` : `invoice` (standard/acompte/solde) ou `credit_note` (avoir).
- Dates ISO `YYYY-MM-DD` : omettre `due_date` si « À réception », omettre `delivery_date`
  (défaut = date d'émission).
- Lignes : `unit` = `day` (TJM) ou `unit` (forfait/acompte) ; `quantity` chaîne décimale.
  - standard → une ligne par prestation (`name` = titre, `unit_price_ht_cents`, `amount_ht_cents`).
  - acompte/solde → une seule ligne (`unit` `unit`, `quantity` `"1.00"`, `amount_ht_cents` = montant).
- `buyer` : `name`, `siret` (ou `siren`), `email`, `address`, `postal_code`, `city`, et
  **`electronic_address` + `electronic_address_scheme`** (adresse de routage Peppol BT-49 —
  voir l'encadré ci-dessous ; requise pour l'envoi PDP).
- `buyer_reference` : n° de devis si rattaché (optionnel).

```json
{
  "number": "2026011",
  "type": "invoice",
  "issue_date": "2026-06-29",
  "buyer_reference": "D2026004",
  "buyer": { "name": "ACME SAS", "siret": "55208131700015", "email": "compta@acme.example",
             "electronic_address": "552081317", "electronic_address_scheme": "0225",
             "address": "1 rue de la Paix", "postal_code": "75002", "city": "Paris" },
  "lines": [
    { "name": "Développement backend", "unit": "day", "quantity": "2.00",
      "unit_price_ht_cents": 60000, "amount_ht_cents": 120000 }
  ]
}
```

> Émetteur assujetti à la TVA : ajouter `"vat": {"exempt": false, "rate_pct": "20.00"}`.

#### Adresse de routage Peppol (BT-49) — requise pour l'envoi PDP
La PDP lit l'adresse de routage de l'acheteur dans le XML (`electronic_address` +
`electronic_address_scheme`, BT-49) et **rejette le dépôt** si elle ne correspond à aucun
participant de l'annuaire (`pre-check: receiver address does not exist in peppol directory`).
Sans ces champs, gofact retombe sur l'e-mail (scheme `EM`), qui **n'est pas** une adresse de
routage Peppol → rejet systématique.

- **Client déjà connu** : réutiliser l'`electronic_address`/`electronic_address_scheme` d'un
  sidecar précédent du même client.
- **Client inconnu** : interroger l'annuaire Peppol public, **indexé par SIREN** (9 chiffres —
  le SIRET ne renvoie rien ; SIREN = 9 premiers chiffres du SIRET) :

  ```sh
  curl -s "https://directory.peppol.eu/search/1.0/json?q=<SIREN>" \
    | jq -r '.["total-result-count"], (.matches[]?.participantID.value)'
  ```
  - Résultat `<scheme>:<SIREN>` (ex. `0225:945226215`) → `electronic_address_scheme` = le scheme,
    `electronic_address` = la valeur (le SIREN). En France le participant est généralement
    indexé sous **`0225`** et `0002` (même SIREN) ; **préférer `0225`**, et si la PDP rejette
    encore, réessayer avec **`0002`**.
  - Ne retenir qu'un participant **actif** (`FR_ASSUJETTI_ACTIVE`).
- **Absent de l'annuaire** (`total-result-count` = 0) : le client n'est pas adressable sur le
  réseau → **ne pas tenter l'envoi PDP** ; livrer le PDF Factur-X par un autre canal (e-mail) et
  le signaler à l'utilisateur.

### 6 — Mettre à jour `numerotation.json`
- Incrémenter `compteurs[YYYY]`.
- Ajouter une entrée dans `factures` : `numero`, `date_emission` (ISO `YYYY-MM-DD`), `client`,
  `contact`, `projet`, `montant_ht`, `statut` (`émise` par défaut), `fichier`,
  et `devis_ref` si rattaché à un devis.

### 7 — Générer le PDF Factur-X (`gofact`)
Le binaire `gofact` (livré avec ce plugin) produit un **PDF/A-3 conforme Factur-X**
(XML CII EN 16931 embarqué) à partir du HTML + du sidecar JSON.

```sh
GOFACT="${CLAUDE_PLUGIN_ROOT:?plugin gofact introuvable}"
# Build au 1ᵉʳ usage seulement (binaire absent, non versionné) :
[ -x "$GOFACT/gofact" ] || (cd "$GOFACT" && go build -trimpath -ldflags="-s -w" -o gofact .)
# Génération (déduit le .json et le .pdf du nom du HTML) :
"$GOFACT/gofact" -html "$INVOICES/{YYYY}{NNN} - {Client}[ - Suffixe].html"
```

- Sortie : `{même nom}.pdf`. Affiche `✓ Factur-X conforme (EN 16931)`.
- Au 1ᵉʳ run, `gofact` télécharge Mustang dans `~/.cache/gofact/` (validation EN 16931 + PDF/A).
- Erreur « vendeur non configuré » ⇒ les variables `GOFACT_SELLER_*` ne sont pas en place :
  renseigner un `.env` (cf. `.env.example` du plugin) ou `~/.config/gofact/.env`.
- En cas de `✗ NON conforme`, lire le rapport affiché et corriger le `.json` (souvent un champ
  acheteur manquant : SIRET, adresse) ; ne pas livrer un PDF non conforme.
- Dépendances système : `google-chrome`, `gs` (Ghostscript ≥ 10), `java` (JRE 11+).

## Règles à ne pas transgresser
- Numérotation continue, sans trou, jamais réutilisée ni antidatée.
- Ne jamais écrire de secret, d'IBAN ou d'identité vendeur en dur dans un fichier de ce dépôt :
  tout passe par les variables d'environnement.
- Ne jamais éditer le template maître pour un client donné : toujours en partir par copie.
- Les mentions légales (pénalités de retard, indemnité de recouvrement, escompte) sont émises
  par `gofact` et présentes dans le template : ne pas les dupliquer dans le sidecar.

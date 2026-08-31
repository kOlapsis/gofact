# Le dossier d'organisation

Une entité émettrice = un dossier autonome, qui vous appartient et se sauvegarde comme
n'importe quel dossier. gofact n'a ni base de données ni état caché.

```
Factures/
├── .env                     # identité de l'émetteur + identifiants PDP (secrets)
├── numerotation.json        # registre : compteurs par année + factures émises
├── journal.ndjson           # piste d'audit, en ajout seul
├── modele-facture.html      # modèle de référence, figé à la première facture
├── 2026001 - ACME.html      # chaque facture : le HTML…
├── 2026001 - ACME.json      # …ses données structurées…
└── 2026001 - ACME.pdf       # …et le PDF Factur-X archivable
```

## La numérotation

C'est une obligation légale : séquence **continue, sans trou, jamais réutilisée**.
gofact la traite comme telle :

- l'attribution est **verrouillée** (fichier de verrou) : deux créations simultanées ne
  produisent jamais le même numéro ;
- elle est **transactionnelle** : le numéro n'existe qu'avec son entrée de registre et
  ses fichiers — un échec de génération ne consomme rien ;
- le registre est réécrit de façon **atomique**, et tout champ qu'il contient et que
  gofact ne gère pas est préservé ;
- chaque attribution et chaque dépôt PDP laissent une ligne dans `journal.ndjson`.

Format des numéros : `{année}{compteur sur 3 chiffres}` — `2026001`, `2026002`…

**Vous facturiez déjà avant gofact ?** Reprenez la séquence là où elle en est :

```sh
gofact org set-counter -last-number 2026011   # le prochain émis sera 2026012
```

(ou `-last-number` dès `org init`, ou l'outil MCP `initialize_numbering` en
conversation). L'opération est journalisée et le compteur ne peut que **monter** :
l'abaisser réutiliserait des numéros déjà émis, gofact le refuse.

!!! danger "Ne modifiez jamais les compteurs à la main"
    Éditer `numerotation.json` pour « rattraper » un numéro crée précisément
    l'irrégularité que tout ceci empêche. `set-counter` existe pour ça.

## Le modèle de facture

La hiérarchie est claire : la **numérotation est l'invariant** — la mise en page, elle,
peut évoluer librement. Le modèle n'existe que pour la cohérence au quotidien :

- il se **crée en conversation**, avec des aperçus PDF (`preview_invoice`, marqué
  SPÉCIMEN, sans rien consommer) jusqu'à ce que le rendu vous convienne ;
- la première facture le fige (`modele-facture.html`) et `get_invoice_template` le
  resert à l'IA pour les suivantes ;
- pour **changer de visuel** : demandez-le — l'IA itère en aperçu puis passe
  `update_template=true` à la facture suivante, et le nouveau modèle remplace l'ancien
  (changement journalisé) ;
- une mise en page qui dérive *sans* que ce soit demandé n'est qu'un avertissement,
  jamais un blocage.

## L'identité : le `.env` du dossier

```sh
GOFACT_SELLER_NAME="Studio Exemple"
GOFACT_SELLER_SIRET="12345678900014"
GOFACT_SELLER_EMAIL="contact@exemple.test"
GOFACT_SELLER_ADDRESS="1 rue de l'Exemple"
GOFACT_SELLER_POSTAL_CODE="33000"
GOFACT_SELLER_CITY="Bordeaux"
GOFACT_PAYEE_IBAN="FR76…"
# Assujetti à la TVA : GOFACT_SELLER_VAT_NUMBER="FR…"
# Envoi PDP : SUPERPDP_CLIENT_ID / SUPERPDP_CLIENT_SECRET
```

Chaque dossier a le sien : un même poste peut servir plusieurs entités sans qu'elles se
contaminent. Les valeurs manquantes retombent sur l'environnement du processus. Aucun
outil, MCP ou CLI, n'affiche jamais les secrets de ce fichier.

## La découverte

Comment gofact trouve vos organisations, par priorité :

1. le chemin donné explicitement (paramètre `org` d'un outil, flag `-org`) ;
2. le registre utilisateur `~/.config/gofact/orgs.json` (alimenté par `org init`) ;
3. `GOFACT_ORGS_DIRS` (liste de chemins) ;
4. `GOFACT_INVOICES_DIR` (compatibilité avec le skill historique) ;
5. le répertoire courant, s'il contient un `numerotation.json`.

Rien d'autre : gofact **n'explore jamais votre disque** à la recherche de factures.

## L'historique client

`search_client` reconstruit vos clients depuis les sidecars JSON du dossier : les
coordonnées de la facture la plus récente font foi, adresse de routage Peppol comprise.
Un client déjà facturé n'est jamais ressaisi.

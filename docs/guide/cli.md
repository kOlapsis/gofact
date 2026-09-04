# Ligne de commande

C'est le même binaire, mais pas le même contrat. Le terminal expose le **mode direct** :
un convertisseur HTML + JSON → Factur-X, utile pour scripter, déboguer, ou travailler sans
IA. Il applique les mêmes règles EN 16931 et le même auto-contrôle que le serveur MCP.

!!! warning "Le mode direct n'émet pas de facture"
    `-html` rend le fichier qu'on lui donne : il **n'attribue aucun numéro** et
    **n'inscrit rien au registre**. La numérotation légale — séquence continue, sans
    trou, jamais réutilisée — n'existe que par les outils MCP.

    Dans un dossier d'organisation, `gofact -html` refuse donc de générer un fichier
    inconnu du registre, et ne sert plus qu'à re-rendre une facture déjà émise. Pour en
    émettre une, demandez-la à votre IA (`create_invoice`).

## Générer une facture

```sh
# -data et -out déduits du nom du HTML :
gofact -html "2026001 - ACME.html"
#   → lit "2026001 - ACME.json", écrit "2026001 - ACME.pdf"

# Chemins explicites + XML CII dumpé pour inspection :
gofact -html f.html -data f.json -out f.pdf -xml f.xml
```

| Option | Effet |
|---|---|
| `-env <fichier>` | `.env` explicite (défaut : `./.env` puis `~/.config/gofact/.env`) |
| `-validate=false` | saute l'auto-contrôle du PDF produit |
| `-chrome <chemin>` | force l'exécutable du navigateur |
| `-q` | silencieux (erreurs seulement) |

Code de sortie `0` si le Factur-X est produit et conforme.

## Le sidecar JSON

Les données structurées de la facture. **Tous les montants en centimes** — jamais de
flottant. Le vendeur, l'IBAN et le régime de TVA viennent de la configuration de
l'organisation : le JSON ne porte que ce qui varie.

```json
{
  "number": "2026001",
  "issue_date": "2026-08-31",
  "buyer": {
    "name": "ACME SAS",
    "siret": "55208131700015",
    "email": "compta@acme.example",
    "electronic_address": "552081317",
    "electronic_address_scheme": "0225",
    "address": "1 rue de la Paix",
    "postal_code": "75002",
    "city": "Paris"
  },
  "lines": [
    { "name": "Développement backend", "unit": "day", "quantity": "2.00",
      "unit_price_ht_cents": 60000, "amount_ht_cents": 120000 }
  ]
}
```

Champs optionnels : `type` (`invoice` | `credit_note`), `due_date` (défaut : à
réception), `delivery_date` (défaut : émission), `currency` (défaut EUR), `vat`
(`{"exempt": false, "rate_pct": "20.00"}` pour la TVA standard), `buyer_reference`,
`seller`, `iban`, `notes`.

## Organisations

```sh
gofact org list    # organisations découvertes, compteurs, prochain numéro
gofact org init -path DIR -name "…" -siret … -iban …
gofact org show    # fiche détaillée — jamais de secret affiché
```

## Envoi PDP

```sh
gofact -html f.html -send -poll    # génère puis dépose
gofact send -pdf f.pdf -poll       # dépose un PDF existant
```

Identifiants lus dans l'environnement ou le `.env` (voir [Envoi PDP](../pdp.md)).

## Serveur et installation

```sh
gofact mcp                 # serveur MCP sur stdio
gofact install [-yes]      # configure les clients MCP du poste
```

## Variables d'environnement

Le fichier [`.env.example`](https://github.com/kOlapsis/gofact/blob/main/.env.example)
du dépôt les documente toutes : identité `GOFACT_SELLER_*`, `GOFACT_PAYEE_IBAN`,
mentions d'exonération `GOFACT_VAT_*`, `GOFACT_CHROME`, `GOFACT_OFFLINE`,
`GOFACT_PDP`, `SUPERPDP_*`.

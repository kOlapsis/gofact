# Ligne de commande

Tout ce que fait le serveur MCP existe aussi au terminal — c'est le même binaire et les
mêmes garanties. Utile pour scripter, déboguer, ou travailler sans IA.

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

# Parler à son IA — le serveur MCP

`gofact mcp` expose la facturation en serveur [MCP](https://modelcontextprotocol.io)
local (stdio). C'est le cœur du produit : votre IA orchestre, gofact exécute et
garantit.

## Les outils

### Lecture seule

| Outil | Rôle |
|---|---|
| `list_organizations` | Les entités émettrices configurées — le premier appel de toute session |
| `get_organization` | Fiche d'une organisation : identité publique, compteurs, modèle. Jamais de secret, jamais d'IBAN en clair |
| `get_invoice_template` | Le modèle HTML figé de l'organisation, à réutiliser pour chaque facture |
| `search_client` | Historique de facturation d'abord, annuaire SIRENE ensuite |
| `find_routing_address` | Adressabilité Peppol d'un SIREN — indispensable avant un dépôt PDP |
| `preview_next_number` | Le prochain numéro, **sans le consommer** — pour l'annoncer avant confirmation |
| `list_invoices` | Le registre, filtrable par année ou client |
| `get_invoice_status` | Cycle de vie PDP d'une facture déposée |

### Écriture

| Outil | Rôle |
|---|---|
| `init_organization` | Crée un dossier d'organisation ; refuse d'écraser l'existant |
| `create_invoice` | **La transaction** : numéro + fichiers + PDF Factur-X + registre, en un appel |
| `send_invoice` | Dépôt PDP — **seul outil destructif**, exige `confirm=true` après votre accord explicite |

## Trois garanties de conception

**Le numéro appartient au serveur.** L'IA écrit le jeton `{{NUMERO}}` dans son HTML ;
`create_invoice` attribue le numéro sous verrou et le substitue. Il n'existe pas d'outil
« réserver un numéro » qu'une conversation abandonnée laisserait consommé : un échec de
génération rend le numéro, et la séquence légale reste sans trou.

**Les secrets ne circulent pas.** Les identifiants PDP vivent dans le `.env` du dossier
d'organisation ; aucune sortie d'outil ne contient de secret (c'est testé), et l'IA est
guidée pour ne jamais vous les demander en conversation.

**Les erreurs sont faites pour être relayées.** « BR-50 : moyen de paiement 30
(virement) : l'IBAN du compte de règlement (BT-84) est requis — renseignez
GOFACT_PAYEE_IBAN » : l'IA peut vous transmettre le problème *et* l'action corrective,
sans jargon supplémentaire.

## Le prompt `nouvelle-facture`

Le serveur publie aussi un prompt MCP qui encode le déroulé complet (organisation →
client → composition → confirmation → création → envoi éventuel). Les clients qui
exposent les prompts peuvent le proposer tel quel.

## Enregistrement manuel

Si `gofact install` ne connaît pas votre client, la déclaration est standard :

```json
{
  "mcpServers": {
    "gofact": { "command": "/chemin/vers/gofact", "args": ["mcp"] }
  }
}
```

Le serveur n'écrit **que** du JSON-RPC sur stdout ; tout diagnostic part sur stderr.

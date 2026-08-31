# Envoi PDP

La réforme fait transiter les factures B2B par des **plateformes de dématérialisation
partenaires** (PDP). gofact dépose vos Factur-X sur la vôtre — c'est la seule opération
irréversible de l'outil, et elle est traitée comme telle.

## Prérequis

1. **Un compte chez une PDP.** gofact n'est pas une plateforme et ne s'intercale pas :
   le contrat, l'immatriculation et les identifiants sont les vôtres. Fournisseur
   implémenté : [SuperPDP](https://superpdp.tech) ; l'architecture accepte d'autres
   fournisseurs (`GOFACT_PDP`).
2. **Vos identifiants dans le `.env` du dossier** de l'organisation — jamais en
   conversation avec l'IA :

    ```sh
    SUPERPDP_CLIENT_ID="…"
    SUPERPDP_CLIENT_SECRET="…"
    # SUPERPDP_BASE="https://api.superpdp.tech"   # optionnel
    ```

3. **Un destinataire adressable.** La PDP route sur l'adresse électronique de
   l'acheteur (BT-49), pas sur son adresse postale. `find_routing_address` vérifie
   l'annuaire Peppol à partir du SIREN :

    - trouvé → `electronic_address` + `electronic_address_scheme` dans les données de
      la facture. Pour une PDP française, préférer le scheme `0225` ; si le dépôt est
      rejeté, réessayer en `0002` ;
    - introuvable → **ne pas tenter l'envoi** : livrez le PDF par un autre canal.

!!! note "L'émetteur aussi doit être adressable"
    La PDP exige que le vendeur de la facture corresponde à la société du compte
    authentifié (identifiant légal BT-30) et porte sa propre adresse de routage. Par
    défaut gofact route l'émetteur sur son SIREN en scheme `0225`.

## Déposer

En conversation, l'IA n'appellera `send_invoice` qu'après **votre confirmation
explicite** — l'outil est marqué destructif et refuse de s'exécuter sans elle. Au
terminal :

```sh
gofact send -pdf "2026001 - ACME.pdf" -poll
```

## Suivre

Le cycle de vie s'affiche au dépôt et se consulte ensuite (`get_invoice_status`,
`-poll`) :

| Statut | Sens |
|---|---|
| `fr:200` | déposée sur la plateforme |
| `fr:201` | émise vers le destinataire |
| `fr:202` | reçue par la plateforme du destinataire |

Chaque dépôt est tracé dans le `journal.ndjson` du dossier : quoi, quand, sous quelle
référence.

## Rejets courants

| Message | Cause | Correction |
|---|---|---|
| `receiver address does not exist in peppol directory` | BT-49 absente ou fausse | `find_routing_address`, scheme `0225` puis `0002` |
| Vendeur ≠ compte | l'identité de l'organisation ne correspond pas au compte PDP | vérifier SIREN/SIRET du `.env` |
| Identifiants refusés | secret expiré ou mal copié | régénérer sur la plateforme, mettre à jour le `.env` |
